package stresstest

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zUtil/zMap"
	"go.uber.org/zap"
)

type ActionFunc func(ctx *ScenarioContext) error

type Action struct {
	Name   string
	Weight int
	Fn     ActionFunc
}

type Scenario struct {
	Name       string
	Actions    []Action
	Setup      ActionFunc
	Teardown   ActionFunc
	Duration   time.Duration
	RatePerSec int
}

type ScenarioContext struct {
	ClientID  int
	Iteration int
	Data      map[string]interface{}
}

type MetricType int

const (
	MetricLatency MetricType = iota
	MetricSuccess
	MetricFailure
	MetricError
)

type Metric struct {
	Name      string
	Type      MetricType
	Value     int64
	Timestamp time.Time
}

type MetricsCollector struct {
	latencies *zMap.TypedMap[string, *latencyTracker]
	successes *zMap.TypedMap[string, *atomic.Uint64]
	failures  *zMap.TypedMap[string, *atomic.Uint64]
	errors    *zMap.TypedMap[string, *atomic.Uint64]
}

type latencyTracker struct {
	sum   atomic.Int64
	count atomic.Int64
	min   atomic.Int64
	max   atomic.Int64
}

func newLatencyTracker() *latencyTracker {
	return &latencyTracker{}
}

func (lt *latencyTracker) Record(d time.Duration) {
	nanos := d.Nanoseconds()
	lt.sum.Add(nanos)
	lt.count.Add(1)
	for {
		current := lt.min.Load()
		if current != 0 && nanos >= current {
			break
		}
		if lt.min.CompareAndSwap(current, nanos) {
			break
		}
	}
	for {
		current := lt.max.Load()
		if nanos <= current {
			break
		}
		if lt.max.CompareAndSwap(current, nanos) {
			break
		}
	}
}

func (lt *latencyTracker) Avg() time.Duration {
	count := lt.count.Load()
	if count == 0 {
		return 0
	}
	return time.Duration(lt.sum.Load() / count)
}

func (lt *latencyTracker) Min() time.Duration {
	return time.Duration(lt.min.Load())
}

func (lt *latencyTracker) Max() time.Duration {
	return time.Duration(lt.max.Load())
}

func (lt *latencyTracker) Count() int64 {
	return lt.count.Load()
}

func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		latencies: zMap.NewTypedMap[string, *latencyTracker](),
		successes: zMap.NewTypedMap[string, *atomic.Uint64](),
		failures:  zMap.NewTypedMap[string, *atomic.Uint64](),
		errors:    zMap.NewTypedMap[string, *atomic.Uint64](),
	}
}

func (mc *MetricsCollector) RecordLatency(action string, d time.Duration) {
	tracker, ok := mc.latencies.Load(action)
	if !ok {
		tracker = newLatencyTracker()
		mc.latencies.Store(action, tracker)
	}
	tracker.Record(d)
}

func (mc *MetricsCollector) RecordSuccess(action string) {
	counter, ok := mc.successes.Load(action)
	if !ok {
		counter = &atomic.Uint64{}
		mc.successes.Store(action, counter)
	}
	counter.Add(1)
}

func (mc *MetricsCollector) RecordFailure(action string) {
	counter, ok := mc.failures.Load(action)
	if !ok {
		counter = &atomic.Uint64{}
		mc.failures.Store(action, counter)
	}
	counter.Add(1)
}

func (mc *MetricsCollector) RecordError(action string) {
	counter, ok := mc.errors.Load(action)
	if !ok {
		counter = &atomic.Uint64{}
		mc.errors.Store(action, counter)
	}
	counter.Add(1)
}

type ActionStats struct {
	Name    string
	Count   int64
	Success uint64
	Failure uint64
	Errors  uint64
	AvgLat  time.Duration
	MinLat  time.Duration
	MaxLat  time.Duration
}

func (mc *MetricsCollector) Stats() []ActionStats {
	var stats []ActionStats

	mc.latencies.Range(func(name string, tracker *latencyTracker) bool {
		s := ActionStats{
			Name:   name,
			Count:  tracker.Count(),
			AvgLat: tracker.Avg(),
			MinLat: tracker.Min(),
			MaxLat: tracker.Max(),
		}
		if c, ok := mc.successes.Load(name); ok {
			s.Success = c.Load()
		}
		if c, ok := mc.failures.Load(name); ok {
			s.Failure = c.Load()
		}
		if c, ok := mc.errors.Load(name); ok {
			s.Errors = c.Load()
		}
		stats = append(stats, s)
		return true
	})

	return stats
}

type StressTestConfig struct {
	Concurrency int
	Duration    time.Duration
	RampUpTime  time.Duration
	Scenario    Scenario
}

type StressTestResult struct {
	ScenarioName  string
	Duration      time.Duration
	Concurrency   int
	TotalActions  int64
	SuccessRate   float64
	ActionsPerSec float64
	Stats         []ActionStats
}

type StressTester struct {
	config  StressTestConfig
	metrics *MetricsCollector
	running atomic.Bool
	wg      sync.WaitGroup
}

func NewStressTester(config StressTestConfig) *StressTester {
	if config.Concurrency <= 0 {
		config.Concurrency = 10
	}
	if config.Duration <= 0 {
		config.Duration = 30 * time.Second
	}
	return &StressTester{
		config:  config,
		metrics: NewMetricsCollector(),
	}
}

func (st *StressTester) Run() *StressTestResult {
	if !st.running.CompareAndSwap(false, true) {
		return nil
	}
	defer st.running.Store(false)

	start := time.Now()
	scenario := st.config.Scenario

	zLog.Info("Stress test starting",
		zap.String("scenario", scenario.Name),
		zap.Int("concurrency", st.config.Concurrency),
		zap.Duration("duration", st.config.Duration))

	totalActions := atomic.Int64{}

	rampUpPerClient := time.Duration(0)
	if st.config.RampUpTime > 0 && st.config.Concurrency > 1 {
		rampUpPerClient = st.config.RampUpTime / time.Duration(st.config.Concurrency)
	}

	for i := 0; i < st.config.Concurrency; i++ {
		st.wg.Add(1)
		go func(clientID int) {
			defer st.wg.Done()

			if rampUpPerClient > 0 {
				time.Sleep(time.Duration(clientID) * rampUpPerClient)
			}

			ctx := &ScenarioContext{
				ClientID:  clientID,
				Iteration: 0,
				Data:      make(map[string]interface{}),
			}

			if scenario.Setup != nil {
				if err := scenario.Setup(ctx); err != nil {
					zLog.Error("Scenario setup failed",
						zap.Int("client_id", clientID),
						zap.Error(err))
					return
				}
			}

			defer func() {
				if scenario.Teardown != nil {
					scenario.Teardown(ctx)
				}
			}()

			deadline := start.Add(st.config.Duration)
			for time.Now().Before(deadline) && st.running.Load() {
				ctx.Iteration++

				action := st.selectAction(scenario.Actions)
				if action == nil {
					time.Sleep(10 * time.Millisecond)
					continue
				}

				actionStart := time.Now()
				err := action.Fn(ctx)
				elapsed := time.Since(actionStart)

				totalActions.Add(1)

				if err != nil {
					st.metrics.RecordFailure(action.Name)
					st.metrics.RecordError(action.Name)
				} else {
					st.metrics.RecordSuccess(action.Name)
				}
				st.metrics.RecordLatency(action.Name, elapsed)
			}
		}(i)
	}

	st.wg.Wait()
	totalDuration := time.Since(start)

	stats := st.metrics.Stats()
	totalSuccess := uint64(0)
	totalFailure := uint64(0)
	for _, s := range stats {
		totalSuccess += s.Success
		totalFailure += s.Failure
	}

	total := totalSuccess + totalFailure
	successRate := float64(0)
	if total > 0 {
		successRate = float64(totalSuccess) / float64(total) * 100
	}

	result := &StressTestResult{
		ScenarioName:  scenario.Name,
		Duration:      totalDuration,
		Concurrency:   st.config.Concurrency,
		TotalActions:  totalActions.Load(),
		SuccessRate:   successRate,
		ActionsPerSec: float64(totalActions.Load()) / totalDuration.Seconds(),
		Stats:         stats,
	}

	st.printReport(result)

	return result
}

func (st *StressTester) selectAction(actions []Action) *Action {
	if len(actions) == 0 {
		return nil
	}

	totalWeight := 0
	for _, a := range actions {
		totalWeight += a.Weight
	}

	if totalWeight == 0 {
		return &actions[0]
	}

	target := int(time.Now().UnixNano()) % totalWeight
	current := 0
	for i := range actions {
		current += actions[i].Weight
		if target < current {
			return &actions[i]
		}
	}

	return &actions[0]
}

func (st *StressTester) Stop() {
	st.running.Store(false)
}

func (st *StressTester) printReport(result *StressTestResult) {
	zLog.Info("=== Stress Test Report ===",
		zap.String("scenario", result.ScenarioName),
		zap.Duration("duration", result.Duration),
		zap.Int("concurrency", result.Concurrency),
		zap.Int64("total_actions", result.TotalActions),
		zap.Float64("success_rate", result.SuccessRate),
		zap.Float64("actions_per_sec", result.ActionsPerSec))

	for _, s := range result.Stats {
		zLog.Info("Action stats",
			zap.String("action", s.Name),
			zap.Int64("count", s.Count),
			zap.Uint64("success", s.Success),
			zap.Uint64("failure", s.Failure),
			zap.Duration("avg_latency", s.AvgLat),
			zap.Duration("min_latency", s.MinLat),
			zap.Duration("max_latency", s.MaxLat))
	}
}

func FormatReport(result *StressTestResult) string {
	report := fmt.Sprintf("=== 压力测试报告 ===\n")
	report += fmt.Sprintf("场景: %s\n", result.ScenarioName)
	report += fmt.Sprintf("持续时间: %v\n", result.Duration)
	report += fmt.Sprintf("并发数: %d\n", result.Concurrency)
	report += fmt.Sprintf("总操作数: %d\n", result.TotalActions)
	report += fmt.Sprintf("成功率: %.2f%%\n", result.SuccessRate)
	report += fmt.Sprintf("吞吐量: %.2f ops/sec\n", result.ActionsPerSec)
	report += fmt.Sprintf("\n--- 操作详情 ---\n")

	for _, s := range result.Stats {
		report += fmt.Sprintf("[%s] 次数=%d 成功=%d 失败=%d 错误=%d 平均延迟=%v 最小=%v 最大=%v\n",
			s.Name, s.Count, s.Success, s.Failure, s.Errors, s.AvgLat, s.MinLat, s.MaxLat)
	}

	return report
}

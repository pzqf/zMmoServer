package connpool

import (
	"fmt"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zEngine/zNet"
	"go.uber.org/zap"
)

type ConnOption struct {
	Addr    string
	Timeout time.Duration
}

type PoolConfig struct {
	InitialCap int
	MaxCap     int
	Options    ConnOption
}

type ConnWrapper struct {
	client   *zNet.TcpClient
	id       int
	pool     *ConnectionPool
	lastUsed atomic.Int64
}

func (cw *ConnWrapper) Client() *zNet.TcpClient {
	return cw.client
}

func (cw *ConnWrapper) ID() int {
	return cw.id
}

func (cw *ConnWrapper) IsConnected() bool {
	return cw.client != nil && cw.client.IsConnected()
}

func (cw *ConnWrapper) LastUsed() time.Time {
	return time.UnixMilli(cw.lastUsed.Load())
}

func (cw *ConnWrapper) touch() {
	cw.lastUsed.Store(time.Now().UnixMilli())
}

type ConnectionPool struct {
	config     PoolConfig
	conns      []*ConnWrapper
	available  chan *ConnWrapper
	mu         sync.Mutex
	running    atomic.Bool
	factory    func(addr string, timeout time.Duration) (*zNet.TcpClient, error)
	roundRobin atomic.Uint64
}

func NewConnectionPool(config PoolConfig, factory func(addr string, timeout time.Duration) (*zNet.TcpClient, error)) (*ConnectionPool, error) {
	if config.InitialCap <= 0 {
		config.InitialCap = 1
	}
	if config.MaxCap <= 0 {
		config.MaxCap = 4
	}
	if config.InitialCap > config.MaxCap {
		config.InitialCap = config.MaxCap
	}

	pool := &ConnectionPool{
		config:    config,
		conns:     make([]*ConnWrapper, 0, config.MaxCap),
		available: make(chan *ConnWrapper, config.MaxCap),
		factory:   factory,
	}

	for i := 0; i < config.InitialCap; i++ {
		client, err := pool.factory(config.Options.Addr, config.Options.Timeout)
		if err != nil {
			pool.Close()
			return nil, fmt.Errorf("init conn %d: %w", i, err)
		}
		cw := &ConnWrapper{
			client: client,
			id:     i,
			pool:   pool,
		}
		cw.touch()
		pool.conns = append(pool.conns, cw)
		pool.available <- cw
	}

	pool.running.Store(true)
	zLog.Info("Connection pool created",
		zap.Int("initial", config.InitialCap),
		zap.Int("max", config.MaxCap),
		zap.String("addr", config.Options.Addr))

	return pool, nil
}

func (p *ConnectionPool) Get() (*ConnWrapper, error) {
	if !p.running.Load() {
		return nil, fmt.Errorf("pool is closed")
	}

	select {
	case cw := <-p.available:
		cw.touch()
		return cw, nil
	default:
	}

	p.mu.Lock()
	if len(p.conns) < p.config.MaxCap {
		client, err := p.factory(p.config.Options.Addr, p.config.Options.Timeout)
		if err != nil {
			p.mu.Unlock()
			return nil, fmt.Errorf("create conn: %w", err)
		}
		cw := &ConnWrapper{
			client: client,
			id:     len(p.conns),
			pool:   p,
		}
		cw.touch()
		p.conns = append(p.conns, cw)
		p.mu.Unlock()
		return cw, nil
	}
	p.mu.Unlock()

	select {
	case cw := <-p.available:
		cw.touch()
		return cw, nil
	case <-time.After(p.config.Options.Timeout):
		return nil, fmt.Errorf("get conn timeout")
	}
}

func (p *ConnectionPool) Put(cw *ConnWrapper) {
	if !p.running.Load() {
		return
	}
	cw.touch()
	p.available <- cw
}

func (p *ConnectionPool) RoundRobin() (*ConnWrapper, error) {
	if !p.running.Load() {
		return nil, fmt.Errorf("pool is closed")
	}

	p.mu.Lock()
	conns := make([]*ConnWrapper, 0, len(p.conns))
	for _, cw := range p.conns {
		if cw.IsConnected() {
			conns = append(conns, cw)
		}
	}
	p.mu.Unlock()

	if len(conns) == 0 {
		return nil, fmt.Errorf("no available connections")
	}

	idx := p.roundRobin.Add(1) % uint64(len(conns))
	cw := conns[idx]
	cw.touch()
	return cw, nil
}

func (p *ConnectionPool) Send(protoId zNet.ProtoIdType, data []byte) error {
	cw, err := p.RoundRobin()
	if err != nil {
		return err
	}
	return cw.client.Send(protoId, data)
}

func (p *ConnectionPool) ActiveCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	count := 0
	for _, cw := range p.conns {
		if cw.IsConnected() {
			count++
		}
	}
	return count
}

func (p *ConnectionPool) TotalCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.conns)
}

func (p *ConnectionPool) Close() {
	if !p.running.CompareAndSwap(true, false) {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	for len(p.available) > 0 {
		<-p.available
	}

	for _, cw := range p.conns {
		if cw.client != nil {
			cw.client.Close()
		}
	}
	p.conns = p.conns[:0]

	zLog.Info("Connection pool closed")
}

func (p *ConnectionPool) HealthCheck() map[int]bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	result := make(map[int]bool, len(p.conns))
	for _, cw := range p.conns {
		result[cw.id] = cw.IsConnected()
	}
	return result
}

func DefaultFactory(addr string, timeout time.Duration) (*zNet.TcpClient, error) {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf("invalid address format %s: %w", addr, err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, fmt.Errorf("invalid port %s: %w", portStr, err)
	}

	cfg := &zNet.TcpClientConfig{
		ServerAddr:        host,
		ServerPort:        port,
		ChanSize:          1024,
		HeartbeatDuration: 30,
		MaxPacketDataSize: 1024 * 1024,
		AutoReconnect:     true,
		ReconnectDelay:    5,
		DisableEncryption: true,
	}
	client := zNet.NewTcpClient(cfg)
	err = client.Connect()
	if err != nil {
		return nil, err
	}
	return client, nil
}

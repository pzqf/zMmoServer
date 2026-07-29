package security

import (
	"testing"

	"github.com/pzqf/zMmoServer/GatewayServer/config"
	"github.com/pzqf/zEngine/zNet"
)

// 反作弊统计维度的回归守护。演变过三次，两个方向都不能再回去：
//   - 按 sessionID：重连即换 ID、统计清零 → 检测被绕过（SEC-4 的原始教训）
//   - 按 clientIP：NAT/CGNAT 下大量玩家共用出口 IP，动作数叠加 → 一人超标、全楼连坐
//   - 现在按**账号**（未认证阶段才回退 IP）：重连稳定 + 玩家间独立，两个要求同时满足

func subjectTestConfig(maxActionsPerMinute, maxHighSeverity int) *config.Config {
	cfg := &config.Config{}
	cfg.AntiCheat = config.AntiCheatConfig{
		MaxActionsPerMinute:    maxActionsPerMinute,
		MaxErrorRatio:          1.0,
		MaxAbnormalActions:     1000000,
		MaxHighSeverityReports: maxHighSeverity,
		InactiveTimeoutMinutes: 30,
		CleanupIntervalMinutes: 10,
	}
	cfg.DDoS = zNet.DDoSConfig{}
	return cfg
}

// TestAntiCheat_SubjectsAreIndependent 同一出口 IP 后面的两个账号各记各的：
// 一个刷爆不影响另一个。这正是 NAT/CGNAT 场景。
func TestAntiCheat_SubjectsAreIndependent(t *testing.T) {
	// 每分钟 5 个动作、攒 3 次高危即拒绝。
	acm := NewAntiCheatManager(subjectTestConfig(5, 3))

	const noisy, quiet = "acct:1001", "acct:1002"

	// 刷爆 noisy：动作数远超上限，攒够高危次数。
	for i := 0; i < 50; i++ {
		acm.RecordClientAction(noisy, 64)
	}
	if allowed, _ := acm.CheckClientStatus(noisy); allowed {
		t.Fatalf("刷爆的账号应被判定为不可放行")
	}

	// 同一台机器/同一出口 IP 上的另一个账号必须照常可玩。
	for i := 0; i < 5; i++ {
		acm.RecordClientAction(quiet, 64)
	}
	if allowed, reason := acm.CheckClientStatus(quiet); !allowed {
		t.Fatalf("另一个账号被连坐了: %s", reason)
	}
}

// TestAntiCheat_UnauthenticatedFallsBackToIP 未认证阶段没有账号可用，
// 退回按 IP 统计仍要正常工作（这段流量只有 token 校验那几个包）。
func TestAntiCheat_UnauthenticatedFallsBackToIP(t *testing.T) {
	acm := NewAntiCheatManager(subjectTestConfig(5, 3))

	const subject = "ip:203.0.113.5"
	for i := 0; i < 50; i++ {
		acm.RecordClientAction(subject, 64)
	}
	if allowed, _ := acm.CheckClientStatus(subject); allowed {
		t.Fatalf("未认证主体刷爆后同样应被拦")
	}
}

// TestAntiCheat_UnknownSubjectAllowed 没有任何记录的主体默认放行（新玩家进来不能被误拦）。
func TestAntiCheat_UnknownSubjectAllowed(t *testing.T) {
	acm := NewAntiCheatManager(subjectTestConfig(5, 3))
	if allowed, reason := acm.CheckClientStatus("acct:9999"); !allowed {
		t.Fatalf("无记录的主体应放行, got %s", reason)
	}
}

// TestAntiCheat_NormalPlayNotFlagged 正常速率的玩家不应被判违规
//（阈值本身是否合理是另一回事，这里守护"没记录到超限就不判"）。
func TestAntiCheat_NormalPlayNotFlagged(t *testing.T) {
	acm := NewAntiCheatManager(subjectTestConfig(1000, 3))

	const subject = "acct:2001"
	for i := 0; i < 500; i++ {
		acm.RecordClientAction(subject, 64)
	}
	if allowed, reason := acm.CheckClientStatus(subject); !allowed {
		t.Fatalf("额度内的正常游玩不应被拦: %s", reason)
	}
}

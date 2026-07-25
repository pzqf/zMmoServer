package trade

// GoldAccount 交易金币交换所需的最小账户接口（*player.Player 实现之）。
// ReduceGold 为原子扣减，金币不足返回 false（底层 atomic CAS）。
type GoldAccount interface {
	ReduceGold(amount int64) bool
	AddGold(amount int64)
}

// ExecuteSwap 原子交换双方金币（带回滚）：a 付 goldA 给 b，b 付 goldB 给 a。
// 任一方金币不足即回滚并返回失败。金币本身是 atomic，故本函数在单 GameServer 内一致；
// 跨服/崩溃安全需上层持久化层，未做（最小闭环）。
func ExecuteSwap(a, b GoldAccount, goldA, goldB int64) (bool, string) {
	if goldA > 0 && !a.ReduceGold(goldA) {
		return false, "发起方金币不足"
	}
	if goldB > 0 && !b.ReduceGold(goldB) {
		if goldA > 0 {
			a.AddGold(goldA) // 回滚已扣的 A
		}
		return false, "对方金币不足"
	}
	if goldB > 0 {
		a.AddGold(goldB)
	}
	if goldA > 0 {
		b.AddGold(goldA)
	}
	return true, ""
}

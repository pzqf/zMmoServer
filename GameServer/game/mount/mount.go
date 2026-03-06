package mount

import (
	"github.com/pzqf/zMmoShared/common/id"
	"time"
)

// Mount 坐骑结构
type Mount struct {
	MountID     id.MountIdType   // 坐骑ID
	PlayerID    id.PlayerIdType  // 主人ID
	MountTypeID int              // 坐骑类型ID
	Name        string           // 坐骑名称
	Level       int              // 等级
	Exp         int64            // 经验值
	MaxExp      int64            // 最大经验值
	Speed       int              // 移动速度
	Stamina     int              // 耐力
	MaxStamina  int              // 最大耐力
	Loyalty     int              // 忠诚度
	IsRiding    bool             // 是否骑乘中
	CreateTime  time.Time        // 创建时间
	LastRideTime time.Time       // 最后骑乘时间
}

// NewMount 创建新坐骑
func NewMount(playerID id.PlayerIdType, mountTypeID int, name string) *Mount {
	mount := &Mount{
		MountID:     id.MountIdType(id.GenerateId()),
		PlayerID:    playerID,
		MountTypeID: mountTypeID,
		Name:        name,
		Level:       1,
		Exp:         0,
		MaxExp:      1000,
		Speed:       150, // 基础速度150%
		Stamina:     100,
		MaxStamina:  100,
		Loyalty:     100,
		IsRiding:    false,
		CreateTime:  time.Now(),
		LastRideTime: time.Now(),
	}

	return mount
}

// AddExp 添加经验值
func (m *Mount) AddExp(exp int64) bool {
	m.Exp += exp
	levelUp := false

	for m.Exp >= m.MaxExp {
		m.Exp -= m.MaxExp
		m.Level++
		m.MaxExp = int64(1000 + m.Level*200)
		m.Speed += 10 // 每级增加10%速度
		m.MaxStamina += 10
		m.Stamina = m.MaxStamina
		levelUp = true
	}

	return levelUp
}

// Ride 骑乘
func (m *Mount) Ride() bool {
	if m.IsRiding {
		return false
	}

	if m.Stamina <= 0 {
		return false
	}

	m.IsRiding = true
	m.LastRideTime = time.Now()
	return true
}

// Dismount 下马
func (m *Mount) Dismount() bool {
	if !m.IsRiding {
		return false
	}

	m.IsRiding = false
	return true
}

// ConsumeStamina 消耗耐力
func (m *Mount) ConsumeStamina(amount int) bool {
	if m.Stamina < amount {
		return false
	}

	m.Stamina -= amount
	return true
}

// RecoverStamina 恢复耐力
func (m *Mount) RecoverStamina(amount int) {
	m.Stamina = min(m.Stamina+amount, m.MaxStamina)
}

// Update 更新坐骑状态
func (m *Mount) Update() {
	// 非骑乘状态下恢复耐力
	if !m.IsRiding {
		m.RecoverStamina(5) // 每秒恢复5点耐力
	}

	// 骑乘状态下消耗耐力
	if m.IsRiding {
		m.ConsumeStamina(1) // 每秒消耗1点耐力
		
		// 耐力耗尽自动下马
		if m.Stamina <= 0 {
			m.Dismount()
		}
	}

	// 随时间减少忠诚度
	elapsed := time.Since(m.LastRideTime)
	if elapsed.Hours() > 24 {
		days := int(elapsed.Hours() / 24)
		m.Loyalty = max(m.Loyalty-days*5, 0)
	}
}

// GetSpeed 获取当前速度
func (m *Mount) GetSpeed() int {
	if !m.IsRiding {
		return 100 // 步行速度100%
	}
	return m.Speed
}

// GetCombatPower 获取战斗力加成
func (m *Mount) GetCombatPower() int {
	return m.Level * 5
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

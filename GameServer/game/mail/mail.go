package mail

import (
	"sync"
	"time"

	"github.com/pzqf/zMmoShared/common/id"
)

// MailType 邮件类型
type MailType int32

const (
	MailTypeSystem  MailType = 1 // 系统邮件
	MailTypePlayer  MailType = 2 // 玩家邮件
	MailTypeReward  MailType = 3 // 奖励邮件
	MailTypeGuild   MailType = 4 // 公会邮件
	MailTypeGM      MailType = 5 // GM邮件
)

// MailStatus 邮件状态
type MailStatus int32

const (
	MailStatusUnread   MailStatus = 0 // 未读
	MailStatusRead     MailStatus = 1 // 已读
	MailStatusClaimed  MailStatus = 2 // 已领取附件
	MailStatusDeleted  MailStatus = 3 // 已删除
)

// MailAttachment 邮件附件
type MailAttachment struct {
	Gold    int64
	Diamond int64
	Exp     int64
	Items   map[int32]int32 // itemConfigID -> count
}

// NewMailAttachment 创建邮件附件
func NewMailAttachment() *MailAttachment {
	return &MailAttachment{
		Items: make(map[int32]int32),
	}
}

// HasAttachment 检查是否有附件
func (a *MailAttachment) HasAttachment() bool {
	return a.Gold > 0 || a.Diamond > 0 || a.Exp > 0 || len(a.Items) > 0
}

// Mail 邮件结构
type Mail struct {
	mu           sync.RWMutex
	mailID       id.MailIdType
	mailType     MailType
	senderID     id.PlayerIdType
	senderName   string
	receiverID   id.PlayerIdType
	title        string
	content      string
	attachment   *MailAttachment
	status       MailStatus
	sendTime     int64
	readTime     int64
	claimTime    int64
	expireTime   int64
	isSystem     bool
}

// NewMail 创建新邮件
func NewMail(mailType MailType, receiverID id.PlayerIdType, title string, content string) *Mail {
	return &Mail{
		mailType:   mailType,
		receiverID: receiverID,
		title:      title,
		content:    content,
		attachment: NewMailAttachment(),
		status:     MailStatusUnread,
		sendTime:   time.Now().UnixMilli(),
		expireTime: time.Now().AddDate(0, 0, 7).UnixMilli(), // 默认7天过期
		isSystem:   mailType == MailTypeSystem || mailType == MailTypeGM,
	}
}

// NewSystemMail 创建系统邮件
func NewSystemMail(receiverID id.PlayerIdType, title string, content string) *Mail {
	mail := NewMail(MailTypeSystem, receiverID, title, content)
	mail.senderName = "系统"
	mail.isSystem = true
	return mail
}

// NewPlayerMail 创建玩家邮件
func NewPlayerMail(senderID id.PlayerIdType, senderName string, receiverID id.PlayerIdType, title string, content string) *Mail {
	mail := NewMail(MailTypePlayer, receiverID, title, content)
	mail.senderID = senderID
	mail.senderName = senderName
	mail.isSystem = false
	return mail
}

// NewRewardMail 创建奖励邮件
func NewRewardMail(receiverID id.PlayerIdType, title string, content string) *Mail {
	mail := NewMail(MailTypeReward, receiverID, title, content)
	mail.senderName = "系统奖励"
	mail.isSystem = true
	return mail
}

// GetMailID 获取邮件ID
func (m *Mail) GetMailID() id.MailIdType {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.mailID
}

// SetMailID 设置邮件ID
func (m *Mail) SetMailID(mailID id.MailIdType) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.mailID = mailID
}

// GetMailType 获取邮件类型
func (m *Mail) GetMailType() MailType {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.mailType
}

// GetSenderID 获取发件人ID
func (m *Mail) GetSenderID() id.PlayerIdType {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.senderID
}

// GetSenderName 获取发件人名称
func (m *Mail) GetSenderName() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.senderName
}

// GetReceiverID 获取收件人ID
func (m *Mail) GetReceiverID() id.PlayerIdType {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.receiverID
}

// GetTitle 获取邮件标题
func (m *Mail) GetTitle() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.title
}

// GetContent 获取邮件内容
func (m *Mail) GetContent() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.content
}

// GetAttachment 获取附件
func (m *Mail) GetAttachment() *MailAttachment {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.attachment
}

// SetAttachment 设置附件
func (m *Mail) SetAttachment(attachment *MailAttachment) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.attachment = attachment
}

// AddGold 添加金币附件
func (m *Mail) AddGold(gold int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.attachment == nil {
		m.attachment = NewMailAttachment()
	}
	m.attachment.Gold += gold
}

// AddDiamond 添加钻石附件
func (m *Mail) AddDiamond(diamond int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.attachment == nil {
		m.attachment = NewMailAttachment()
	}
	m.attachment.Diamond += diamond
}

// AddExp 添加经验附件
func (m *Mail) AddExp(exp int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.attachment == nil {
		m.attachment = NewMailAttachment()
	}
	m.attachment.Exp += exp
}

// AddItem 添加物品附件
func (m *Mail) AddItem(itemConfigID int32, count int32) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.attachment == nil {
		m.attachment = NewMailAttachment()
	}
	m.attachment.Items[itemConfigID] += count
}

// GetStatus 获取邮件状态
func (m *Mail) GetStatus() MailStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.status
}

// SetStatus 设置邮件状态
func (m *Mail) SetStatus(status MailStatus) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.status = status
}

// GetSendTime 获取发送时间
func (m *Mail) GetSendTime() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.sendTime
}

// GetReadTime 获取阅读时间
func (m *Mail) GetReadTime() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.readTime
}

// GetClaimTime 获取领取时间
func (m *Mail) GetClaimTime() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.claimTime
}

// GetExpireTime 获取过期时间
func (m *Mail) GetExpireTime() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.expireTime
}

// SetExpireTime 设置过期时间
func (m *Mail) SetExpireTime(expireTime int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.expireTime = expireTime
}

// IsSystem 检查是否是系统邮件
func (m *Mail) IsSystem() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.isSystem
}

// IsRead 检查是否已读
func (m *Mail) IsRead() bool {
	return m.GetStatus() != MailStatusUnread
}

// HasAttachment 检查是否有附件
func (m *Mail) HasAttachment() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.attachment != nil && m.attachment.HasAttachment()
}

// CanClaim 检查是否可以领取附件
func (m *Mail) CanClaim() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.status == MailStatusRead && m.attachment != nil && m.attachment.HasAttachment()
}

// IsExpired 检查是否已过期
func (m *Mail) IsExpired(currentTime int64) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return currentTime > m.expireTime
}

// IsDeleted 检查是否已删除
func (m *Mail) IsDeleted() bool {
	return m.GetStatus() == MailStatusDeleted
}

// Read 阅读邮件
func (m *Mail) Read() bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.status == MailStatusDeleted {
		return false
	}

	if m.status == MailStatusUnread {
		m.status = MailStatusRead
		m.readTime = time.Now().UnixMilli()
	}

	return true
}

// Claim 领取附件
func (m *Mail) Claim() (*MailAttachment, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.status == MailStatusDeleted {
		return nil, false
	}

	if m.status != MailStatusRead {
		return nil, false
	}

	if m.attachment == nil || !m.attachment.HasAttachment() {
		return nil, false
	}

	attachment := m.attachment
	m.attachment = NewMailAttachment()
	m.status = MailStatusClaimed
	m.claimTime = time.Now().UnixMilli()

	return attachment, true
}

// Delete 删除邮件
func (m *Mail) Delete() bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.status == MailStatusDeleted {
		return false
	}

	// 有附件未领取不能删除
	if m.status == MailStatusRead && m.attachment != nil && m.attachment.HasAttachment() {
		return false
	}

	m.status = MailStatusDeleted
	return true
}

// Clone 克隆邮件
func (m *Mail) Clone() *Mail {
	m.mu.RLock()
	defer m.mu.RUnlock()

	clone := &Mail{
		mailID:     m.mailID,
		mailType:   m.mailType,
		senderID:   m.senderID,
		senderName: m.senderName,
		receiverID: m.receiverID,
		title:      m.title,
		content:    m.content,
		attachment: &MailAttachment{
			Gold:    m.attachment.Gold,
			Diamond: m.attachment.Diamond,
			Exp:     m.attachment.Exp,
			Items:   make(map[int32]int32),
		},
		status:     m.status,
		sendTime:   m.sendTime,
		readTime:   m.readTime,
		claimTime:  m.claimTime,
		expireTime: m.expireTime,
		isSystem:   m.isSystem,
	}

	for k, v := range m.attachment.Items {
		clone.attachment.Items[k] = v
	}

	return clone
}

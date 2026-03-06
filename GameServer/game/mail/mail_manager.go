package mail

import (
	"errors"
	"sync"
	"time"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/GameServer/game/event"
	"github.com/pzqf/zMmoShared/common/id"
	"go.uber.org/zap"
)

// MailManager 邮件管理器
type MailManager struct {
	mu           sync.RWMutex
	playerID     id.PlayerIdType
	mails        map[id.MailIdType]*Mail
	mailCount    int32
	maxCount     int32
	unreadCount  int32
}

// NewMailManager 创建邮件管理器
func NewMailManager(playerID id.PlayerIdType, maxCount int32) *MailManager {
	return &MailManager{
		playerID:    playerID,
		mails:       make(map[id.MailIdType]*Mail),
		mailCount:   0,
		maxCount:    maxCount,
		unreadCount: 0,
	}
}

// GetPlayerID 获取玩家ID
func (mm *MailManager) GetPlayerID() id.PlayerIdType {
	return mm.playerID
}

// GetMailCount 获取邮件数量
func (mm *MailManager) GetMailCount() int32 {
	mm.mu.RLock()
	defer mm.mu.RUnlock()
	return mm.mailCount
}

// GetMaxCount 获取最大邮件数量
func (mm *MailManager) GetMaxCount() int32 {
	mm.mu.RLock()
	defer mm.mu.RUnlock()
	return mm.maxCount
}

// GetUnreadCount 获取未读邮件数量
func (mm *MailManager) GetUnreadCount() int32 {
	mm.mu.RLock()
	defer mm.mu.RUnlock()
	return mm.unreadCount
}

// IsFull 检查邮件箱是否已满
func (mm *MailManager) IsFull() bool {
	mm.mu.RLock()
	defer mm.mu.RUnlock()
	return mm.mailCount >= mm.maxCount
}

// AddMail 添加邮件
func (mm *MailManager) AddMail(mail *Mail) error {
	if mail == nil {
		return errors.New("mail is nil")
	}

	mm.mu.Lock()
	defer mm.mu.Unlock()

	if mm.mailCount >= mm.maxCount {
		return errors.New("mail box is full")
	}

	mailID := mail.GetMailID()
	if mailID == 0 {
		mm.mailCount++
		mail.SetMailID(id.MailIdType(mm.mailCount))
		mailID = mail.GetMailID()
	}

	mm.mails[mailID] = mail
	mm.unreadCount++

	mm.publishMailReceivedEvent(mail)

	zLog.Debug("Mail added",
		zap.Int64("player_id", int64(mm.playerID)),
		zap.Int64("mail_id", int64(mailID)),
		zap.String("title", mail.GetTitle()))

	return nil
}

// RemoveMail 移除邮件
func (mm *MailManager) RemoveMail(mailID id.MailIdType) (*Mail, error) {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	mail, exists := mm.mails[mailID]
	if !exists {
		return nil, errors.New("mail not found")
	}

	if mail.GetStatus() == MailStatusRead && mail.HasAttachment() {
		return nil, errors.New("cannot delete mail with unclaimed attachment")
	}

	delete(mm.mails, mailID)
	mm.mailCount--

	if !mail.IsRead() {
		mm.unreadCount--
	}

	zLog.Debug("Mail removed",
		zap.Int64("player_id", int64(mm.playerID)),
		zap.Int64("mail_id", int64(mailID)))

	return mail, nil
}

// GetMail 获取邮件
func (mm *MailManager) GetMail(mailID id.MailIdType) (*Mail, error) {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	mail, exists := mm.mails[mailID]
	if !exists {
		return nil, errors.New("mail not found")
	}
	return mail, nil
}

// GetAllMails 获取所有邮件
func (mm *MailManager) GetAllMails() []*Mail {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	mails := make([]*Mail, 0, len(mm.mails))
	for _, mail := range mm.mails {
		mails = append(mails, mail)
	}
	return mails
}

// GetUnreadMails 获取未读邮件
func (mm *MailManager) GetUnreadMails() []*Mail {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	mails := make([]*Mail, 0)
	for _, mail := range mm.mails {
		if !mail.IsRead() {
			mails = append(mails, mail)
		}
	}
	return mails
}

// GetMailsByType 获取指定类型的邮件
func (mm *MailManager) GetMailsByType(mailType MailType) []*Mail {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	mails := make([]*Mail, 0)
	for _, mail := range mm.mails {
		if mail.GetMailType() == mailType {
			mails = append(mails, mail)
		}
	}
	return mails
}

// GetMailsWithAttachment 获取有附件的邮件
func (mm *MailManager) GetMailsWithAttachment() []*Mail {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	mails := make([]*Mail, 0)
	for _, mail := range mm.mails {
		if mail.HasAttachment() {
			mails = append(mails, mail)
		}
	}
	return mails
}

// ReadMail 阅读邮件
func (mm *MailManager) ReadMail(mailID id.MailIdType) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	mail, exists := mm.mails[mailID]
	if !exists {
		return errors.New("mail not found")
	}

	if !mail.IsRead() {
		mm.unreadCount--
	}

	if !mail.Read() {
		return errors.New("read mail failed")
	}

	mm.publishMailReadEvent(mail)

	zLog.Debug("Mail read",
		zap.Int64("player_id", int64(mm.playerID)),
		zap.Int64("mail_id", int64(mailID)))

	return nil
}

// ClaimAttachment 领取附件
func (mm *MailManager) ClaimAttachment(mailID id.MailIdType) (*MailAttachment, error) {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	mail, exists := mm.mails[mailID]
	if !exists {
		return nil, errors.New("mail not found")
	}

	attachment, ok := mail.Claim()
	if !ok {
		return nil, errors.New("claim attachment failed")
	}

	mm.publishMailClaimedEvent(mail)

	zLog.Info("Mail attachment claimed",
		zap.Int64("player_id", int64(mm.playerID)),
		zap.Int64("mail_id", int64(mailID)),
		zap.Int64("gold", attachment.Gold),
		zap.Int64("diamond", attachment.Diamond))

	return attachment, nil
}

// ClaimAllAttachments 领取所有附件
func (mm *MailManager) ClaimAllAttachments() ([]*MailAttachment, int32) {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	attachments := make([]*MailAttachment, 0)
	claimedCount := int32(0)

	for _, mail := range mm.mails {
		if mail.CanClaim() {
			attachment, ok := mail.Claim()
			if ok {
				attachments = append(attachments, attachment)
				claimedCount++
				mm.publishMailClaimedEvent(mail)
			}
		}
	}

	if claimedCount > 0 {
		zLog.Info("All mail attachments claimed",
			zap.Int64("player_id", int64(mm.playerID)),
			zap.Int32("count", claimedCount))
	}

	return attachments, claimedCount
}

// DeleteMail 删除邮件
func (mm *MailManager) DeleteMail(mailID id.MailIdType) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	mail, exists := mm.mails[mailID]
	if !exists {
		return errors.New("mail not found")
	}

	if !mail.Delete() {
		return errors.New("delete mail failed")
	}

	delete(mm.mails, mailID)
	mm.mailCount--

	if !mail.IsRead() {
		mm.unreadCount--
	}

	zLog.Debug("Mail deleted",
		zap.Int64("player_id", int64(mm.playerID)),
		zap.Int64("mail_id", int64(mailID)))

	return nil
}

// DeleteReadMails 删除已读邮件
func (mm *MailManager) DeleteReadMails() int32 {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	deletedCount := int32(0)
	toDelete := make([]id.MailIdType, 0)

	for mailID, mail := range mm.mails {
		if mail.IsRead() && !mail.HasAttachment() {
			toDelete = append(toDelete, mailID)
		}
	}

	for _, mailID := range toDelete {
		delete(mm.mails, mailID)
		mm.mailCount--
		deletedCount++
	}

	if deletedCount > 0 {
		zLog.Info("Read mails deleted",
			zap.Int64("player_id", int64(mm.playerID)),
			zap.Int32("count", deletedCount))
	}

	return deletedCount
}

// CleanExpiredMails 清理过期邮件
func (mm *MailManager) CleanExpiredMails() int32 {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	currentTime := time.Now().UnixMilli()
	deletedCount := int32(0)
	toDelete := make([]id.MailIdType, 0)

	for mailID, mail := range mm.mails {
		if mail.IsExpired(currentTime) {
			toDelete = append(toDelete, mailID)
		}
	}

	for _, mailID := range toDelete {
		mail := mm.mails[mailID]
		delete(mm.mails, mailID)
		mm.mailCount--
		if !mail.IsRead() {
			mm.unreadCount--
		}
		deletedCount++
	}

	if deletedCount > 0 {
		zLog.Info("Expired mails cleaned",
			zap.Int64("player_id", int64(mm.playerID)),
			zap.Int32("count", deletedCount))
	}

	return deletedCount
}

// SendSystemMail 发送系统邮件
func (mm *MailManager) SendSystemMail(title string, content string, attachment *MailAttachment) error {
	mail := NewSystemMail(mm.playerID, title, content)
	if attachment != nil {
		mail.SetAttachment(attachment)
	}
	return mm.AddMail(mail)
}

// SendRewardMail 发送奖励邮件
func (mm *MailManager) SendRewardMail(title string, content string, attachment *MailAttachment) error {
	mail := NewRewardMail(mm.playerID, title, content)
	if attachment != nil {
		mail.SetAttachment(attachment)
	}
	return mm.AddMail(mail)
}

// Clear 清空邮件
func (mm *MailManager) Clear() {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	mm.mails = make(map[id.MailIdType]*Mail)
	mm.mailCount = 0
	mm.unreadCount = 0
}

// publishMailReceivedEvent 发布邮件接收事件
func (mm *MailManager) publishMailReceivedEvent(mail *Mail) {
	event.Publish(event.NewEvent(event.EventMailReceived, mm, &event.MailEventData{
		PlayerID: mm.playerID,
		MailID:   mail.GetMailID(),
	}))
}

// publishMailReadEvent 发布邮件阅读事件
func (mm *MailManager) publishMailReadEvent(mail *Mail) {
	event.Publish(event.NewEvent(event.EventMailRead, mm, &event.MailEventData{
		PlayerID: mm.playerID,
		MailID:   mail.GetMailID(),
	}))
}

// publishMailClaimedEvent 发布邮件领取事件
func (mm *MailManager) publishMailClaimedEvent(mail *Mail) {
	event.Publish(event.NewEvent(event.EventMailClaimed, mm, &event.MailEventData{
		PlayerID: mm.playerID,
		MailID:   mail.GetMailID(),
	}))
}

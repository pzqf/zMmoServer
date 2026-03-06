package object

import (
	"github.com/pzqf/zMmoServer/MapServer/common"
	"github.com/pzqf/zMmoShared/common/id"
)

// NPC NPC对象
type NPC struct {
	id        id.ObjectIdType
	npcID     int32
	name      string
	position  common.Vector3
	dialogue  string
	quests    []int32
	shopItems []int32
	state     NPCState
}

// NPCState NPC状态
type NPCState int

const (
	NPCStateIdle NPCState = iota
	NPCStateTalking
	NPCStateTrading
)

// NewNPC 创建新NPC
func NewNPC(objectID id.ObjectIdType, npcID int32, name string, pos common.Vector3, dialogue string) *NPC {
	return &NPC{
		id:        objectID,
		npcID:     npcID,
		name:      name,
		position:  pos,
		dialogue:  dialogue,
		quests:    make([]int32, 0),
		shopItems: make([]int32, 0),
		state:     NPCStateIdle,
	}
}

// GetID 获取对象ID
func (n *NPC) GetID() id.ObjectIdType {
	return n.id
}

// GetType 获取对象类型
func (n *NPC) GetType() common.GameObjectType {
	return common.GameObjectTypeNPC
}

// GetPosition 获取位置
func (n *NPC) GetPosition() common.Vector3 {
	return n.position
}

// SetPosition 设置位置
func (n *NPC) SetPosition(pos common.Vector3) {
	n.position = pos
}

// GetNPCID 获取NPC ID
func (n *NPC) GetNPCID() int32 {
	return n.npcID
}

// GetName 获取NPC名称
func (n *NPC) GetName() string {
	return n.name
}

// GetDialogue 获取对话内容
func (n *NPC) GetDialogue() string {
	return n.dialogue
}

// SetDialogue 设置对话内容
func (n *NPC) SetDialogue(dialogue string) {
	n.dialogue = dialogue
}

// AddQuest 添加任务
func (n *NPC) AddQuest(questID int32) {
	n.quests = append(n.quests, questID)
}

// RemoveQuest 移除任务
func (n *NPC) RemoveQuest(questID int32) {
	for i, id := range n.quests {
		if id == questID {
			n.quests = append(n.quests[:i], n.quests[i+1:]...)
			break
		}
	}
}

// GetQuests 获取任务列表
func (n *NPC) GetQuests() []int32 {
	return n.quests
}

// AddShopItem 添加商店物品
func (n *NPC) AddShopItem(itemID int32) {
	n.shopItems = append(n.shopItems, itemID)
}

// RemoveShopItem 移除商店物品
func (n *NPC) RemoveShopItem(itemID int32) {
	for i, id := range n.shopItems {
		if id == itemID {
			n.shopItems = append(n.shopItems[:i], n.shopItems[i+1:]...)
			break
		}
	}
}

// GetShopItems 获取商店物品列表
func (n *NPC) GetShopItems() []int32 {
	return n.shopItems
}

// GetState 获取NPC状态
func (n *NPC) GetState() NPCState {
	return n.state
}

// SetState 设置NPC状态
func (n *NPC) SetState(state NPCState) {
	n.state = state
}

// CanTalk 检查是否可以对话
func (n *NPC) CanTalk() bool {
	return n.state == NPCStateIdle
}

// CanTrade 检查是否可以交易
func (n *NPC) CanTrade() bool {
	return len(n.shopItems) > 0 && n.state == NPCStateIdle
}

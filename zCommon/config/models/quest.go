package models

// Quest 任务配置结构
type Quest struct {
	QuestID     int32  `json:"quest_id"`
	Name        string `json:"name"`
	Type        int32  `json:"type"`
	Level       int32  `json:"level"`
	Description string `json:"description"`
	Objectives  string `json:"objectives"` // JSON格式的任务目标
	Rewards     string `json:"rewards"`    // JSON格式的奖励
	NextQuestID int32  `json:"next_quest_id"`
	PreQuestID  int32  `json:"pre_quest_id"`
}

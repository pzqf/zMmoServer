package tables

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/pzqf/zCommon/config/models"
)

// TableManager 表管理器
// 管理所有配置表的加载和访问
type TableManager struct {
	itemLoader        *ItemTableLoader
	mapLoader         *MapTableLoader
	skillLoader       *SkillTableLoader
	questLoader       *QuestTableLoader
	playerLevelLoader *PlayerLevelTableLoader
	monsterLoader     *MonsterTableLoader
	buffLoader        *BuffTableLoader
	aiLoader          *AITableLoader
	spawnPointLoader  *SpawnPointTableLoader
	loaders           []TableLoaderInterface
	initialized       bool
}

// GlobalTableManager ȫ�ֱ������ù�����ʵ��
var (
	GlobalTableManager *TableManager
	tableOnce          sync.Once
)

// NewTableManager �����������ù�����
func NewTableManager() *TableManager {
	itemLoader := NewItemTableLoader()
	mapLoader := NewMapTableLoader()
	skillLoader := NewSkillTableLoader()
	questLoader := NewQuestTableLoader()
	playerLevelLoader := NewPlayerLevelTableLoader()
	monsterLoader := NewMonsterTableLoader()
	buffLoader := NewBuffTableLoader()
	aiLoader := NewAITableLoader()
	spawnPointLoader := NewSpawnPointTableLoader()

	return &TableManager{
		itemLoader:        itemLoader,
		mapLoader:         mapLoader,
		skillLoader:       skillLoader,
		questLoader:       questLoader,
		playerLevelLoader: playerLevelLoader,
		monsterLoader:     monsterLoader,
		buffLoader:        buffLoader,
		aiLoader:          aiLoader,
		spawnPointLoader:  spawnPointLoader,
		loaders: []TableLoaderInterface{
			itemLoader,
			mapLoader,
			skillLoader,
			questLoader,
			playerLevelLoader,
			monsterLoader,
			buffLoader,
			aiLoader,
			spawnPointLoader,
		},
		initialized: false,
	}
}

// GetTableManager ��ȡȫ�ֱ������ù�����ʵ��
// ʹ�õ���ģʽȷ��ȫ��ֻ��һ��ʵ��
func GetTableManager() *TableManager {
	if GlobalTableManager == nil {
		tableOnce.Do(func() {
			GlobalTableManager = NewTableManager()
		})
	}
	return GlobalTableManager
}

// LoadAllTables �����������ñ���
// ��resources/excel_tablesĿ¼��������Excel�����ļ�
// ����: ���ع����еĴ���
func (tm *TableManager) LoadAllTables() error {
	rootDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	tablesDir := filepath.Join(rootDir, "resources", "excel_tables")

	for _, loader := range tm.loaders {
		if err := loader.Load(tablesDir); err != nil {
			return fmt.Errorf("failed to load table %T: %w", loader, err)
		}
	}

	tm.initialized = true
	return nil
}

// GetItemLoader ��ȡ��Ʒ���������
func (tm *TableManager) GetItemLoader() *ItemTableLoader {
	return tm.itemLoader
}

// GetMapLoader ��ȡ��ͼ���������
func (tm *TableManager) GetMapLoader() *MapTableLoader {
	return tm.mapLoader
}

// GetSkillLoader ��ȡ���ܱ��������
func (tm *TableManager) GetSkillLoader() *SkillTableLoader {
	return tm.skillLoader
}

// GetQuestLoader ��ȡ������������
func (tm *TableManager) GetQuestLoader() *QuestTableLoader {
	return tm.questLoader
}

// GetPlayerLevelLoader ��ȡ����ȼ����������
func (tm *TableManager) GetPlayerLevelLoader() *PlayerLevelTableLoader {
	return tm.playerLevelLoader
}

// GetMonsterLoader ��ȡ������������
func (tm *TableManager) GetMonsterLoader() *MonsterTableLoader {
	return tm.monsterLoader
}



// GetBuffLoader ��ȡbuff���������
func (tm *TableManager) GetBuffLoader() *BuffTableLoader {
	return tm.buffLoader
}

// GetAILoader ��ȡAI���������
func (tm *TableManager) GetAILoader() *AITableLoader {
	return tm.aiLoader
}

// GetSpawnPointLoader ��ȡˢ�µ���������
func (tm *TableManager) GetSpawnPointLoader() *SpawnPointTableLoader {
	return tm.spawnPointLoader
}

// GetSpawnPointsByMap ��ȡָ����ͼ��ˢ�µ��б�
func (tm *TableManager) GetSpawnPointsByMap(mapID int32) []*models.SpawnPoint {
	return tm.spawnPointLoader.GetSpawnPointsByMap(mapID)
}

// IsInitialized �������Ƿ��Ѿ���ʼ��
func (tm *TableManager) IsInitialized() bool {
	return tm.initialized
}


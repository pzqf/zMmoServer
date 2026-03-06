package pet

import (
	"sync"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoShared/common/id"
	"go.uber.org/zap"
)

// PetManager 宠物管理器
type PetManager struct {
	pets       map[id.PetIdType]*Pet
	playerPets map[id.PlayerIdType][]id.PetIdType
	mutex      sync.RWMutex
}

// NewPetManager 创建宠物管理器
func NewPetManager() *PetManager {
	return &PetManager{
		pets:       make(map[id.PetIdType]*Pet),
		playerPets: make(map[id.PlayerIdType][]id.PetIdType),
	}
}

// CreatePet 创建宠物
func (pm *PetManager) CreatePet(playerID id.PlayerIdType, petTypeID int, name string) *Pet {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	pet := NewPet(playerID, petTypeID, name)
	pm.pets[pet.PetID] = pet
	pm.playerPets[playerID] = append(pm.playerPets[playerID], pet.PetID)

	zLog.Info("Pet created",
		zap.Uint64("pet_id", uint64(pet.PetID)),
		zap.Uint64("player_id", uint64(playerID)),
		zap.String("pet_name", name),
		zap.Int("pet_type", petTypeID))

	return pet
}

// GetPet 获取宠物
func (pm *PetManager) GetPet(petID id.PetIdType) *Pet {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	return pm.pets[petID]
}

// GetPlayerPets 获取玩家的所有宠物
func (pm *PetManager) GetPlayerPets(playerID id.PlayerIdType) []*Pet {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	petIDs := pm.playerPets[playerID]
	pets := make([]*Pet, 0, len(petIDs))

	for _, petID := range petIDs {
		if pet := pm.pets[petID]; pet != nil {
			pets = append(pets, pet)
		}
	}

	return pets
}

// SummonPet 召唤宠物
func (pm *PetManager) SummonPet(petID id.PetIdType) bool {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	pet := pm.pets[petID]
	if pet == nil {
		return false
	}

	// 先收回玩家的其他宠物
	playerPets := pm.playerPets[pet.PlayerID]
	for _, id := range playerPets {
		if p := pm.pets[id]; p != nil && p.IsSummoned {
			p.Dismiss()
		}
	}

	if pet.Summon() {
		zLog.Info("Pet summoned",
			zap.Uint64("pet_id", uint64(petID)),
			zap.Uint64("player_id", uint64(pet.PlayerID)),
			zap.String("pet_name", pet.Name))
		return true
	}

	return false
}

// DismissPet 收回宠物
func (pm *PetManager) DismissPet(petID id.PetIdType) bool {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	pet := pm.pets[petID]
	if pet == nil {
		return false
	}

	if pet.Dismiss() {
		zLog.Info("Pet dismissed",
			zap.Uint64("pet_id", uint64(petID)),
			zap.Uint64("player_id", uint64(pet.PlayerID)),
			zap.String("pet_name", pet.Name))
		return true
	}

	return false
}

// AddExpToPet 给宠物添加经验值
func (pm *PetManager) AddExpToPet(petID id.PetIdType, exp int64) bool {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	pet := pm.pets[petID]
	if pet == nil {
		return false
	}

	levelUp := pet.AddExp(exp)
	if levelUp {
		zLog.Info("Pet leveled up",
			zap.Uint64("pet_id", uint64(petID)),
			zap.Uint64("player_id", uint64(pet.PlayerID)),
			zap.String("pet_name", pet.Name),
			zap.Int("new_level", pet.Level))
	}

	return true
}

// FeedPet 喂食宠物
func (pm *PetManager) FeedPet(petID id.PetIdType, foodType int) bool {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	pet := pm.pets[petID]
	if pet == nil {
		return false
	}

	if pet.Feed(foodType) {
		zLog.Info("Pet fed",
			zap.Uint64("pet_id", uint64(petID)),
			zap.Uint64("player_id", uint64(pet.PlayerID)),
			zap.String("pet_name", pet.Name),
			zap.Int("food_type", foodType))
		return true
	}

	return false
}

// DeletePet 删除宠物
func (pm *PetManager) DeletePet(petID id.PetIdType) bool {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	pet := pm.pets[petID]
	if pet == nil {
		return false
	}

	// 从玩家宠物列表中移除
	playerID := pet.PlayerID
	petIDs := pm.playerPets[playerID]
	for i, id := range petIDs {
		if id == petID {
			pm.playerPets[playerID] = append(petIDs[:i], petIDs[i+1:]...)
			break
		}
	}

	// 删除宠物
	delete(pm.pets, petID)
	zLog.Info("Pet deleted",
		zap.Uint64("pet_id", uint64(petID)),
		zap.Uint64("player_id", uint64(playerID)),
		zap.String("pet_name", pet.Name))

	return true
}

// UpdatePetStats 更新宠物状态
func (pm *PetManager) UpdatePetStats() {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	for _, pet := range pm.pets {
		pet.UpdateStats()
	}
}

// GetPetCount 获取宠物总数
func (pm *PetManager) GetPetCount() int {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	return len(pm.pets)
}

// GetPlayerPetCount 获取玩家宠物数量
func (pm *PetManager) GetPlayerPetCount(playerID id.PlayerIdType) int {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	return len(pm.playerPets[playerID])
}

package player

import (
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/GameServer/game/pet"
	"github.com/pzqf/zMmoShared/common/id"
	"go.uber.org/zap"
)

// PlayerPet 玩家宠物组件
type PlayerPet struct {
	playerID    id.PlayerIdType
	pets        map[id.PetIdType]*pet.Pet
	activePetID id.PetIdType
	maxPetCount int
}

// NewPlayerPet 创建玩家宠物组件
func NewPlayerPet(playerID id.PlayerIdType) *PlayerPet {
	return &PlayerPet{
		playerID:    playerID,
		pets:        make(map[id.PetIdType]*pet.Pet),
		activePetID: 0,
		maxPetCount: 10, // 默认最多10只宠物
	}
}

// GetPets 获取所有宠物
func (pp *PlayerPet) GetPets() []*pet.Pet {
	result := make([]*pet.Pet, 0, len(pp.pets))
	for _, p := range pp.pets {
		result = append(result, p)
	}
	return result
}

// GetPet 获取指定宠物
func (pp *PlayerPet) GetPet(petID id.PetIdType) *pet.Pet {
	return pp.pets[petID]
}

// AddPet 添加宠物
func (pp *PlayerPet) AddPet(p *pet.Pet) bool {
	if len(pp.pets) >= pp.maxPetCount {
		zLog.Warn("Player pet count reached limit",
			zap.Uint64("player_id", uint64(pp.playerID)),
			zap.Int("max_count", pp.maxPetCount))
		return false
	}

	pp.pets[p.PetID] = p
	zLog.Info("Player added pet",
		zap.Uint64("player_id", uint64(pp.playerID)),
		zap.Uint64("pet_id", uint64(p.PetID)),
		zap.String("pet_name", p.Name))
	return true
}

// RemovePet 移除宠物
func (pp *PlayerPet) RemovePet(petID id.PetIdType) bool {
	if _, exists := pp.pets[petID]; !exists {
		return false
	}

	// 如果是激活的宠物，先收回
	if pp.activePetID == petID {
		pp.DismissPet()
	}

	delete(pp.pets, petID)
	zLog.Info("Player removed pet",
		zap.Uint64("player_id", uint64(pp.playerID)),
		zap.Uint64("pet_id", uint64(petID)))
	return true
}

// GetActivePet 获取当前激活的宠物
func (pp *PlayerPet) GetActivePet() *pet.Pet {
	if pp.activePetID == 0 {
		return nil
	}
	return pp.pets[pp.activePetID]
}

// GetActivePetID 获取当前激活的宠物ID
func (pp *PlayerPet) GetActivePetID() id.PetIdType {
	return pp.activePetID
}

// SummonPet 召唤宠物
func (pp *PlayerPet) SummonPet(petID id.PetIdType) bool {
	p, exists := pp.pets[petID]
	if !exists {
		return false
	}

	// 先收回当前激活的宠物
	if pp.activePetID > 0 && pp.activePetID != petID {
		pp.DismissPet()
	}

	if p.Summon() {
		pp.activePetID = petID
		zLog.Info("Player summoned pet",
			zap.Uint64("player_id", uint64(pp.playerID)),
			zap.Uint64("pet_id", uint64(petID)),
			zap.String("pet_name", p.Name))
		return true
	}

	return false
}

// DismissPet 收回宠物
func (pp *PlayerPet) DismissPet() bool {
	if pp.activePetID == 0 {
		return false
	}

	p := pp.pets[pp.activePetID]
	if p == nil {
		pp.activePetID = 0
		return false
	}

	if p.Dismiss() {
		zLog.Info("Player dismissed pet",
			zap.Uint64("player_id", uint64(pp.playerID)),
			zap.Uint64("pet_id", uint64(pp.activePetID)),
			zap.String("pet_name", p.Name))
		pp.activePetID = 0
		return true
	}

	return false
}

// GetPetCount 获取宠物数量
func (pp *PlayerPet) GetPetCount() int {
	return len(pp.pets)
}

// CanAddPet 是否可以添加更多宠物
func (pp *PlayerPet) CanAddPet() bool {
	return len(pp.pets) < pp.maxPetCount
}

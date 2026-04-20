package dungeon

import (
	"encoding/json"
	"fmt"

	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zEngine/zLog"
	"go.uber.org/zap"
)

type DungeonHandler struct {
	dm  *DungeonManager
	dlm *DungeonLifecycleManager
}

func NewDungeonHandler(dm *DungeonManager, dlm *DungeonLifecycleManager) *DungeonHandler {
	return &DungeonHandler{
		dm:  dm,
		dlm: dlm,
	}
}

func (dh *DungeonHandler) HandleMessage(msgID uint32, data []byte) ([]byte, error) {
	switch msgID {
	case MsgDungeonCreateRequest:
		return dh.handleCreate(data)
	case MsgDungeonEnterRequest:
		return dh.handleEnter(data)
	case MsgDungeonLeaveRequest:
		return dh.handleLeave(data)
	case MsgDungeonStartRequest:
		return dh.handleStart(data)
	case MsgDungeonDestroyRequest:
		return dh.handleDestroy(data)
	case MsgDungeonQueryRequest:
		return dh.handleQuery(data)
	case MsgDungeonMonsterKilled:
		return dh.handleMonsterKilled(data)
	default:
		return nil, fmt.Errorf("unknown dungeon message: %d", msgID)
	}
}

func (dh *DungeonHandler) handleCreate(data []byte) ([]byte, error) {
	var req DungeonCreateRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("unmarshal create request: %w", err)
	}

	canEnter, reason := dh.dm.CanEnterDungeon(req.LeaderID, req.DungeonID)
	if !canEnter {
		resp := DungeonCreateResponse{
			DungeonID: req.DungeonID,
			Success:   false,
			Error:     reason,
		}
		return json.Marshal(resp)
	}

	instance, err := dh.dm.CreateInstance(req.DungeonID)
	if err != nil {
		resp := DungeonCreateResponse{
			DungeonID: req.DungeonID,
			Success:   false,
			Error:     err.Error(),
		}
		return json.Marshal(resp)
	}

	for _, playerID := range req.Players {
		if err := dh.dm.EnterDungeon(playerID, instance.InstanceID); err != nil {
			zLog.Warn("Player failed to enter dungeon on create",
				zap.Int64("player_id", int64(playerID)),
				zap.Int64("instance_id", int64(instance.InstanceID)),
				zap.String("error", err.Error()))
		}
	}

	resp := DungeonCreateResponse{
		InstanceID: instance.InstanceID,
		DungeonID:  req.DungeonID,
		MapID:      instance.MapInstanceID,
		Success:    true,
	}

	zLog.Info("Dungeon created via handler",
		zap.Int64("instance_id", int64(instance.InstanceID)),
		zap.Int32("dungeon_id", int32(req.DungeonID)))

	return json.Marshal(resp)
}

func (dh *DungeonHandler) handleEnter(data []byte) ([]byte, error) {
	var req DungeonEnterRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("unmarshal enter request: %w", err)
	}

	err := dh.dm.EnterDungeon(req.PlayerID, req.InstanceID)
	if err != nil {
		resp := DungeonEnterResponse{
			InstanceID: req.InstanceID,
			PlayerID:   req.PlayerID,
			Success:    false,
			Error:      err.Error(),
		}
		return json.Marshal(resp)
	}

	instance, _ := dh.dm.GetInstance(req.InstanceID)

	resp := DungeonEnterResponse{
		InstanceID: req.InstanceID,
		PlayerID:   req.PlayerID,
		MapID:      instance.GetMapInstanceID(),
		Success:    true,
	}

	return json.Marshal(resp)
}

func (dh *DungeonHandler) handleLeave(data []byte) ([]byte, error) {
	var req DungeonLeaveRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("unmarshal leave request: %w", err)
	}

	err := dh.dm.LeaveDungeon(req.PlayerID, req.InstanceID)
	if err != nil {
		resp := DungeonLeaveResponse{
			InstanceID: req.InstanceID,
			PlayerID:   req.PlayerID,
			Success:    false,
			Error:      err.Error(),
		}
		return json.Marshal(resp)
	}

	resp := DungeonLeaveResponse{
		InstanceID: req.InstanceID,
		PlayerID:   req.PlayerID,
		Success:    true,
	}

	return json.Marshal(resp)
}

func (dh *DungeonHandler) handleStart(data []byte) ([]byte, error) {
	var req DungeonStartRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("unmarshal start request: %w", err)
	}

	err := dh.dm.StartDungeon(req.InstanceID)
	if err != nil {
		resp := DungeonStartResponse{
			InstanceID: req.InstanceID,
			Success:    false,
			Error:      err.Error(),
		}
		return json.Marshal(resp)
	}

	resp := DungeonStartResponse{
		InstanceID: req.InstanceID,
		Success:    true,
	}

	return json.Marshal(resp)
}

func (dh *DungeonHandler) handleDestroy(data []byte) ([]byte, error) {
	var req DungeonDestroyRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("unmarshal destroy request: %w", err)
	}

	if dh.dlm != nil {
		if err := dh.dlm.DestroyDungeon(req.InstanceID); err != nil {
			resp := DungeonDestroyResponse{
				InstanceID: req.InstanceID,
				Success:    false,
				Error:      err.Error(),
			}
			return json.Marshal(resp)
		}
	} else {
		if err := dh.dm.RemoveInstance(req.InstanceID); err != nil {
			resp := DungeonDestroyResponse{
				InstanceID: req.InstanceID,
				Success:    false,
				Error:      err.Error(),
			}
			return json.Marshal(resp)
		}
	}

	resp := DungeonDestroyResponse{
		InstanceID: req.InstanceID,
		Success:    true,
	}

	return json.Marshal(resp)
}

func (dh *DungeonHandler) handleQuery(data []byte) ([]byte, error) {
	var req DungeonQueryRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("unmarshal query request: %w", err)
	}

	records := dh.dm.GetPlayerRecords(req.PlayerID)

	resp := DungeonQueryResponse{
		PlayerID: req.PlayerID,
		Records:  records,
	}

	return json.Marshal(resp)
}

func (dh *DungeonHandler) handleMonsterKilled(data []byte) ([]byte, error) {
	var req DungeonMonsterKilledNotify
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("unmarshal monster killed: %w", err)
	}

	dh.dm.MonsterKilled(req.InstanceID, req.KillerID, 1)

	if dh.dlm != nil {
		if err := dh.dlm.OnMonsterKilled(req.InstanceID, req.KillerID); err != nil {
			zLog.Debug("Wave check after monster kill",
				zap.Int64("instance_id", int64(req.InstanceID)),
				zap.String("result", err.Error()))
		}
	}

	return nil, nil
}

func BuildDungeonCompleteNotify(instance *DungeonInstance) ([]byte, error) {
	if instance == nil {
		return nil, fmt.Errorf("instance is nil")
	}

	notify := DungeonCompleteNotify{
		InstanceID: instance.InstanceID,
		DungeonID:  instance.DungeonID,
		IsSuccess:  instance.IsSuccess,
		Exp:        instance.DungeonConfig.RewardExp,
		Gold:       instance.DungeonConfig.RewardGold,
	}

	return json.Marshal(notify)
}

func BuildDungeonWaveStartNotify(dm *DungeonManager, instance *DungeonInstance) ([]byte, error) {
	if instance == nil {
		return nil, fmt.Errorf("instance is nil")
	}

	monsterIDs, _ := dm.GetCurrentWaveMonsters(instance.InstanceID)

	notify := DungeonWaveStart{
		InstanceID: instance.InstanceID,
		WaveIndex:  instance.CurrentWave,
		MonsterIDs: monsterIDs,
	}

	if instance.Waves != nil && int(instance.CurrentWave) <= len(instance.Waves) {
		notify.IsBoss = instance.Waves[instance.CurrentWave-1].IsBoss
	}

	return json.Marshal(notify)
}

func GetCurrentWaveMonsters(dm *DungeonManager, instanceID id.InstanceIdType) ([]int32, error) {
	return dm.GetCurrentWaveMonsters(instanceID)
}

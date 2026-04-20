package ai

import (
	"math"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/MapServer/common"
	"github.com/pzqf/zMmoServer/MapServer/maps/object"
	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zCommon/config/models"
	"github.com/pzqf/zCommon/config/tables"
	"go.uber.org/zap"
)

type PlayerQuerier interface {
	GetPlayersInRange(position common.Vector3, radius float32) []common.IGameObject
}

type AIManager struct {
	mu                    sync.RWMutex
	monsterAI             map[id.ObjectIdType]*MonsterAI
	tableManager          *tables.TableManager
	defaultDetectionRange float32
	defaultAttackRange    float32
	defaultChaseRange     float32
}

var globalAIManager *AIManager
var aiOnce sync.Once

func NewAIManager() *AIManager {
	return &AIManager{
		monsterAI:             make(map[id.ObjectIdType]*MonsterAI),
		defaultDetectionRange: 15.0,
		defaultAttackRange:    2.0,
		defaultChaseRange:     20.0,
	}
}

func GetAIManager() *AIManager {
	if globalAIManager == nil {
		aiOnce.Do(func() {
			globalAIManager = NewAIManager()
		})
	}
	return globalAIManager
}

func (am *AIManager) SetTableManager(tm *tables.TableManager) {
	am.tableManager = tm
}

type MonsterAI struct {
	monster         *object.Monster
	mapRef          PlayerQuerier
	aiConfig        *models.AI
	state           AIState
	lastStateChange time.Time
	patrolIndex     int
	currentSkill    int
	skillCooldowns  map[int32]time.Time
}

type AIState int

const (
	AIStateIdle AIState = iota
	AIStatePatrolling
	AIStateChasing
	AIStateAttacking
	AIStateFleeing
	AIStateReturning
	AIStateDead
)

func (am *AIManager) CreateMonsterAI(monster *object.Monster, aiType string, mapRef PlayerQuerier) *MonsterAI {
	am.mu.Lock()
	defer am.mu.Unlock()

	ai := &MonsterAI{
		monster:        monster,
		mapRef:         mapRef,
		state:          AIStateIdle,
		skillCooldowns: make(map[int32]time.Time),
	}

	if am.tableManager != nil {
		if aiID, err := strconv.Atoi(aiType); err == nil {
			if aiConfig, ok := am.tableManager.GetAILoader().GetAI(int32(aiID)); ok {
				ai.aiConfig = aiConfig
				monster.SetAIType(aiConfig.Type)
				monster.SetBehaviorPattern(aiConfig.Behavior)
				if aiConfig.DetectionRange > 0 {
					monster.SetVisionRange(aiConfig.DetectionRange)
				}
				if aiConfig.AttackRange > 0 {
					monster.SetAttackRange(aiConfig.AttackRange)
				}
				if aiConfig.ChaseRange > 0 {
					monster.SetAggroRange(aiConfig.ChaseRange)
				}
			}
		}
	}

	am.monsterAI[monster.GetID()] = ai
	return ai
}

func (am *AIManager) RemoveMonsterAI(monsterID id.ObjectIdType) {
	am.mu.Lock()
	defer am.mu.Unlock()
	delete(am.monsterAI, monsterID)
}

func (am *AIManager) GetMonsterAI(monsterID id.ObjectIdType) (*MonsterAI, bool) {
	am.mu.RLock()
	defer am.mu.RUnlock()
	ai, ok := am.monsterAI[monsterID]
	return ai, ok
}

func (am *AIManager) Update(deltaTime time.Duration) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	for _, ai := range am.monsterAI {
		if ai.monster.IsDead() {
			ai.state = AIStateDead
			continue
		}
		ai.update(deltaTime)
	}
}

func (ai *MonsterAI) update(deltaTime time.Duration) {
	if ai.monster.IsDead() {
		ai.state = AIStateDead
		return
	}

	behavior := ai.monster.GetBehaviorPattern()
	switch behavior {
	case "passive":
		ai.updatePassiveBehavior()
	case "aggressive":
		ai.updateAggressiveBehavior()
	case "guard":
		ai.updateGuardBehavior()
	case "patrol":
		ai.updatePatrolBehavior()
	case "wander":
		ai.updateWanderBehavior()
	default:
		ai.updatePassiveBehavior()
	}
}

func (ai *MonsterAI) updatePassiveBehavior() {
	ai.state = AIStateIdle
}

func (ai *MonsterAI) updateAggressiveBehavior() {
	monster := ai.monster

	switch ai.state {
	case AIStateIdle, AIStatePatrolling:
		target := ai.findTarget()
		if target != nil {
			monster.SetTarget(target.GetID())
			ai.state = AIStateChasing
		}
	case AIStateChasing:
		targetID := monster.GetTarget()
		if targetID == 0 {
			ai.state = AIStateIdle
			return
		}
		distance := monster.GetPosition().DistanceTo(monster.GetTargetPosition())
		if distance <= monster.GetAttackRange() {
			ai.state = AIStateAttacking
		} else if distance > monster.GetAggroRange() {
			monster.SetTarget(0)
			ai.state = AIStateIdle
		} else {
			monster.Chase(monster.GetTargetPosition())
		}
	case AIStateAttacking:
		targetID := monster.GetTarget()
		if targetID == 0 {
			ai.state = AIStateIdle
			return
		}
		distance := monster.GetPosition().DistanceTo(monster.GetTargetPosition())
		if distance > monster.GetAttackRange() {
			ai.state = AIStateChasing
		} else {
			ai.tryAttack()
		}
	}
}

func (ai *MonsterAI) updateGuardBehavior() {
	monster := ai.monster

	switch ai.state {
	case AIStateIdle:
		target := ai.findTarget()
		if target != nil {
			monster.SetTarget(target.GetID())
			ai.state = AIStateChasing
		}
	case AIStateChasing:
		targetID := monster.GetTarget()
		if targetID == 0 {
			ai.state = AIStateIdle
			return
		}
		distance := monster.GetPosition().DistanceTo(monster.GetTargetPosition())
		if distance <= monster.GetAttackRange() {
			ai.state = AIStateAttacking
		} else if distance > monster.GetAggroRange()*1.5 {
			monster.SetTarget(0)
			ai.state = AIStateIdle
		} else {
			monster.Chase(monster.GetTargetPosition())
		}
	case AIStateAttacking:
		targetID := monster.GetTarget()
		if targetID == 0 {
			ai.state = AIStateIdle
			return
		}
		distance := monster.GetPosition().DistanceTo(monster.GetTargetPosition())
		if distance > monster.GetAttackRange() {
			ai.state = AIStateChasing
		} else {
			ai.tryAttack()
		}
	}
}

func (ai *MonsterAI) updatePatrolBehavior() {
	monster := ai.monster

	switch ai.state {
	case AIStateIdle:
		if len(monster.GetPatrolPath()) > 0 {
			ai.state = AIStatePatrolling
		} else {
			target := ai.findTarget()
			if target != nil {
				monster.SetTarget(target.GetID())
				ai.state = AIStateChasing
			}
		}
	case AIStatePatrolling:
		target := ai.findTarget()
		if target != nil {
			monster.SetTarget(target.GetID())
			ai.state = AIStateChasing
			return
		}

		nextPoint := monster.GetNextPatrolPoint()
		distance := monster.GetPosition().DistanceTo(nextPoint)
		if distance <= 1.0 {
			time.Sleep(2 * time.Second)
		} else {
			monster.Move(nextPoint)
		}
	case AIStateChasing:
		targetID := monster.GetTarget()
		if targetID == 0 {
			ai.state = AIStatePatrolling
			return
		}
		distance := monster.GetPosition().DistanceTo(monster.GetTargetPosition())
		if distance <= monster.GetAttackRange() {
			ai.state = AIStateAttacking
		} else if distance > monster.GetAggroRange()*2 {
			monster.SetTarget(0)
			ai.state = AIStatePatrolling
		} else {
			monster.Chase(monster.GetTargetPosition())
		}
	case AIStateAttacking:
		targetID := monster.GetTarget()
		if targetID == 0 {
			ai.state = AIStatePatrolling
			return
		}
		distance := monster.GetPosition().DistanceTo(monster.GetTargetPosition())
		if distance > monster.GetAttackRange() {
			ai.state = AIStateChasing
		} else {
			ai.tryAttack()
		}
	}
}

func (ai *MonsterAI) updateWanderBehavior() {
	monster := ai.monster

	switch ai.state {
	case AIStateIdle:
		randomOffset := common.Vector3{
			X: (rand.Float32() - 0.5) * 20,
			Y: (rand.Float32() - 0.5) * 20,
			Z: 0,
		}
		newPos := monster.GetPosition()
		newPos.X += randomOffset.X
		newPos.Y += randomOffset.Y
		monster.SetPosition(newPos)
		ai.state = AIStatePatrolling
	case AIStatePatrolling:
		target := ai.findTarget()
		if target != nil {
			monster.SetTarget(target.GetID())
			ai.state = AIStateChasing
			return
		}
		distance := monster.GetPosition().DistanceTo(monster.GetTargetPosition())
		if distance <= 1.0 {
			ai.state = AIStateIdle
		} else {
			monster.Move(monster.GetTargetPosition())
		}
	case AIStateChasing:
		targetID := monster.GetTarget()
		if targetID == 0 {
			ai.state = AIStateIdle
			return
		}
		distance := monster.GetPosition().DistanceTo(monster.GetTargetPosition())
		if distance <= monster.GetAttackRange() {
			ai.state = AIStateAttacking
		} else if distance > monster.GetAggroRange() {
			monster.SetTarget(0)
			ai.state = AIStateIdle
		} else {
			monster.Chase(monster.GetTargetPosition())
		}
	case AIStateAttacking:
		targetID := monster.GetTarget()
		if targetID == 0 {
			ai.state = AIStateIdle
			return
		}
		distance := monster.GetPosition().DistanceTo(monster.GetTargetPosition())
		if distance > monster.GetAttackRange() {
			ai.state = AIStateChasing
		} else {
			ai.tryAttack()
		}
	}
}

func (ai *MonsterAI) findTarget() common.IGameObject {
	monster := ai.monster
	detectionRange := monster.GetVisionRange()
	if detectionRange <= 0 {
		detectionRange = 15.0
	}

	return ai.findNearestPlayer(detectionRange)
}

func (ai *MonsterAI) findNearestPlayer(range_ float32) common.IGameObject {
	if ai.mapRef == nil {
		return nil
	}

	monster := ai.monster
	pos := monster.GetPosition()
	players := ai.mapRef.GetPlayersInRange(pos, range_)
	if len(players) == 0 {
		return nil
	}

	var nearest common.IGameObject
	minDist := float32(math.MaxFloat32)
	for _, p := range players {
		pPos := p.GetPosition()
		dx := pPos.X - pos.X
		dz := pPos.Z - pos.Z
		dist := float32(math.Sqrt(float64(dx*dx + dz*dz)))
		if dist < minDist {
			minDist = dist
			nearest = p
		}
	}

	return nearest
}

func (ai *MonsterAI) tryAttack() {
	monster := ai.monster
	if !monster.CanAttack() {
		return
	}

	if ai.useSkill() {
		return
	}

	monster.Attack(monster.GetTarget())
}

func (ai *MonsterAI) useSkill() bool {
	if ai.aiConfig == nil || ai.aiConfig.SkillIDs == "" {
		return false
	}

	skillIDs := parseSkillIDs(ai.aiConfig.SkillIDs)
	if len(skillIDs) == 0 {
		return false
	}

	for _, skillID := range skillIDs {
		if cooldown, ok := ai.skillCooldowns[skillID]; ok {
			if time.Since(cooldown) < 5*time.Second {
				continue
			}
		}

		ai.skillCooldowns[skillID] = time.Now()
		zLog.Debug("Monster used skill", zap.Int32("skill_id", skillID), zap.Int64("monster_id", int64(ai.monster.GetID())))
		return true
	}

	return false
}

func parseSkillIDs(skillStr string) []int32 {
	var skillIDs []int32
	if skillStr == "" {
		return skillIDs
	}

	var id int
	for _, c := range skillStr {
		if c >= '0' && c <= '9' {
			id = id*10 + int(c-'0')
		} else if c == ',' || c == ';' {
			if id > 0 {
				skillIDs = append(skillIDs, int32(id))
			}
			id = 0
		}
	}
	if id > 0 {
		skillIDs = append(skillIDs, int32(id))
	}

	return skillIDs
}

func (ai *MonsterAI) GetState() AIState {
	return ai.state
}

func (ai *MonsterAI) SetState(state AIState) {
	ai.state = state
	ai.lastStateChange = time.Now()
}

func (ai *MonsterAI) GetStateName() string {
	switch ai.state {
	case AIStateIdle:
		return "Idle"
	case AIStatePatrolling:
		return "Patrolling"
	case AIStateChasing:
		return "Chasing"
	case AIStateAttacking:
		return "Attacking"
	case AIStateFleeing:
		return "Fleeing"
	case AIStateReturning:
		return "Returning"
	case AIStateDead:
		return "Dead"
	default:
		return "Unknown"
	}
}

func calculateDistance(pos1, pos2 common.Vector3) float32 {
	dx := pos1.X - pos2.X
	dy := pos1.Y - pos2.Y
	dz := pos1.Z - pos2.Z
	return float32(math.Sqrt(float64(dx*dx + dy*dy + dz*dz)))
}


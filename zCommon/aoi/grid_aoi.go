package aoi

import (
	"sync"

	"github.com/pzqf/zEngine/zLog"
	"go.uber.org/zap"
)

type Coord struct {
	X float64
	Y float64
}

type AOIEventType int

const (
	AOIEventEnter AOIEventType = iota
	AOIEventLeave
	AOIEventMove
)

type AOIEvent struct {
	Type     AOIEventType
	Watcher  int64
	Target   int64
	OldCoord Coord
	NewCoord Coord
}

type AOIListener func(event AOIEvent)

type Grid struct {
	mu       sync.RWMutex
	entityID int64
	entities map[int64]Coord
}

func NewGrid(entityID int64) *Grid {
	return &Grid{
		entityID: entityID,
		entities: make(map[int64]Coord),
	}
}

func (g *Grid) Add(entityID int64, coord Coord) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.entities[entityID] = coord
}

func (g *Grid) Remove(entityID int64) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.entities, entityID)
}

func (g *Grid) Update(entityID int64, coord Coord) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.entities[entityID] = coord
}

func (g *Grid) GetAll() map[int64]Coord {
	g.mu.RLock()
	defer g.mu.RUnlock()
	result := make(map[int64]Coord, len(g.entities))
	for k, v := range g.entities {
		result[k] = v
	}
	return result
}

func (g *Grid) Count() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.entities)
}

type GridManager struct {
	mu         sync.RWMutex
	gridWidth  float64
	gridHeight float64
	mapWidth   float64
	mapHeight  float64
	grids      map[int64]*Grid
	listener   AOIListener
}

func NewGridManager(mapWidth, mapHeight, gridWidth, gridHeight float64) *GridManager {
	return &GridManager{
		gridWidth:  gridWidth,
		gridHeight: gridHeight,
		mapWidth:   mapWidth,
		mapHeight:  mapHeight,
		grids:      make(map[int64]*Grid),
	}
}

func (gm *GridManager) SetListener(listener AOIListener) {
	gm.listener = listener
}

func (gm *GridManager) getGridID(coord Coord) int64 {
	gx := int(coord.X / gm.gridWidth)
	gy := int(coord.Y / gm.gridHeight)
	if gx < 0 {
		gx = 0
	}
	if gy < 0 {
		gy = 0
	}
	maxGX := int(gm.mapWidth / gm.gridWidth)
	maxGY := int(gm.mapHeight / gm.gridHeight)
	if gx >= maxGX {
		gx = maxGX - 1
	}
	if gy >= maxGY {
		gy = maxGY - 1
	}
	return int64(gy*maxGX + gx)
}

func (gm *GridManager) getSurroundingGridIDs(gridID int64) []int64 {
	maxGX := int(gm.mapWidth / gm.gridWidth)
	maxGY := int(gm.mapHeight / gm.gridHeight)
	gx := int(gridID) % maxGX
	gy := int(gridID) / maxGX

	var ids []int64
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			nx := gx + dx
			ny := gy + dy
			if nx >= 0 && nx < maxGX && ny >= 0 && ny < maxGY {
				ids = append(ids, int64(ny*maxGX+nx))
			}
		}
	}
	return ids
}

func (gm *GridManager) getOrCreateGrid(gridID int64) *Grid {
	gm.mu.Lock()
	defer gm.mu.Unlock()
	if grid, ok := gm.grids[gridID]; ok {
		return grid
	}
	grid := NewGrid(gridID)
	gm.grids[gridID] = grid
	return grid
}

func (gm *GridManager) AddEntity(entityID int64, coord Coord) {
	gridID := gm.getGridID(coord)
	grid := gm.getOrCreateGrid(gridID)
	grid.Add(entityID, coord)

	surroundingGrids := gm.getSurroundingGridIDs(gridID)
	for _, sid := range surroundingGrids {
		sgrid := gm.getOrCreateGrid(sid)
		for watcherID := range sgrid.GetAll() {
			if watcherID != entityID && gm.listener != nil {
				gm.listener(AOIEvent{
					Type:    AOIEventEnter,
					Watcher: watcherID,
					Target:  entityID,
					NewCoord: coord,
				})
			}
		}
	}

	zLog.Debug("Entity added to AOI",
		zap.Int64("entity_id", entityID),
		zap.Int64("grid_id", gridID))
}

func (gm *GridManager) RemoveEntity(entityID int64, coord Coord) {
	oldGridID := gm.getGridID(coord)

	surroundingGrids := gm.getSurroundingGridIDs(oldGridID)
	for _, sid := range surroundingGrids {
		sgrid := gm.getOrCreateGrid(sid)
		for watcherID := range sgrid.GetAll() {
			if watcherID != entityID && gm.listener != nil {
				gm.listener(AOIEvent{
					Type:     AOIEventLeave,
					Watcher:  watcherID,
					Target:   entityID,
					OldCoord: coord,
				})
			}
		}
	}

	gm.mu.RLock()
	if grid, ok := gm.grids[oldGridID]; ok {
		grid.Remove(entityID)
	}
	gm.mu.RUnlock()

	zLog.Debug("Entity removed from AOI",
		zap.Int64("entity_id", entityID),
		zap.Int64("grid_id", oldGridID))
}

func (gm *GridManager) MoveEntity(entityID int64, oldCoord, newCoord Coord) {
	oldGridID := gm.getGridID(oldCoord)
	newGridID := gm.getGridID(newCoord)

	if oldGridID == newGridID {
		gm.mu.RLock()
		if grid, ok := gm.grids[oldGridID]; ok {
			grid.Update(entityID, newCoord)
		}
		gm.mu.RUnlock()

		if gm.listener != nil {
			gm.listener(AOIEvent{
				Type:     AOIEventMove,
				Target:   entityID,
				OldCoord: oldCoord,
				NewCoord: newCoord,
			})
		}
		return
	}

	gm.RemoveEntity(entityID, oldCoord)
	gm.AddEntity(entityID, newCoord)
}

func (gm *GridManager) GetSurroundingEntities(coord Coord) map[int64]Coord {
	gridID := gm.getGridID(coord)
	surroundingGrids := gm.getSurroundingGridIDs(gridID)

	result := make(map[int64]Coord)
	for _, sid := range surroundingGrids {
		gm.mu.RLock()
		if grid, ok := gm.grids[sid]; ok {
			for id, c := range grid.GetAll() {
				result[id] = c
			}
		}
		gm.mu.RUnlock()
	}
	return result
}

func (gm *GridManager) GetEntitiesInGrid(coord Coord) map[int64]Coord {
	gridID := gm.getGridID(coord)
	gm.mu.RLock()
	defer gm.mu.RUnlock()
	if grid, ok := gm.grids[gridID]; ok {
		return grid.GetAll()
	}
	return make(map[int64]Coord)
}

func (gm *GridManager) GetGridID(coord Coord) int64 {
	return gm.getGridID(coord)
}

package game

import "math"

// The game's rules, as the Godot version plays them: the player walks a cell at a time and pushes
// boxes and keys; a key pushed into a door opens it and is spent; a teleport sends what walks into
// it out of the other end, the same way, unless that cell is taken; buttons do what their level
// says; a laser's beam resets the level when the player stands in it; the heart ends the level.
// The logic happens at once; how things move is only drawn over the next fraction of a second.

// Direction is a step on the grid.
type Direction struct{ X, Y int }

var (
	Up    = Direction{0, -1}
	Down  = Direction{0, 1}
	Left  = Direction{-1, 0}
	Right = Direction{1, 0}
)

// How long a move is drawn, in seconds: the player a little slower than what it pushes, as in the
// original (tweens of 1/3.5 and 1/4 of a second).
const (
	playerMoveTime = 1 / 3.5
	pushedMoveTime = 1 / 4.0
	bumpTime       = 0.18
)

// Entity is a thing of a level while it is played.
type Entity struct {
	Thing
	Gone     bool // opened, spent or broken
	Blocked  bool // a box left on a button: it moves no more
	Active   bool // a button something stands on
	Disabled bool // a button used up, a laser turned off
	pair     *Entity

	// how its last move is drawn: from a cell to another (a teleport's, when it was sent on),
	// from a moment, for a while
	fromX, fromY, viaX, viaY int
	moveStart, moveTime      float64
}

// Event is something the program may want to play a sound for.
type Event int

const (
	EventStep Event = iota
	EventPush
	EventBump
	EventTeleport
	EventDoorOpen
	EventLock
	EventReset
	EventComplete
)

// World is a level being played.
type World struct {
	Level    *Level
	Entities []*Entity
	Player   *Entity
	Steps    int
	Clock    float64 // seconds since the level began
	Complete bool    // the player reached the heart

	walls     map[[2]int]int // the tile map's walls: their drawing
	byName    map[string]*Entity
	resetAt   float64 // when a laser caught the player: the level starts again then
	bumpStart float64
	bumpDir   Direction
	events    []Event
}

// NewWorld starts a level.
func NewWorld(level *Level) *World {
	world := &World{Level: level, walls: map[[2]int]int{}, byName: map[string]*Entity{}, resetAt: -1, bumpStart: -10}
	for _, tile := range level.Tiles {
		world.walls[[2]int{tile.X, tile.Y}] = tile.Variant
	}
	for _, thing := range level.Things {
		entity := &Entity{Thing: thing}
		entity.fromX, entity.fromY, entity.viaX, entity.viaY = thing.X, thing.Y, thing.X, thing.Y
		world.Entities = append(world.Entities, entity)
		world.byName[thing.Name] = entity
		if thing.Kind == KindPlayer {
			world.Player = entity
		}
	}
	for _, entity := range world.Entities {
		if entity.Kind == KindTeleport {
			entity.pair = world.byName[entity.Pair]
		}
	}
	return world
}

// Events are what happened since the last call, for sounds; reading them empties the list.
func (world *World) Events() []Event {
	events := world.events
	world.events = world.events[:0]
	return events
}

// Busy tells whether something is still moving: the player waits for it, as in the original.
func (world *World) Busy() bool {
	for _, entity := range world.Entities {
		if !entity.Gone && world.Clock < entity.moveStart+entity.moveTime {
			return true
		}
	}
	return world.resetAt >= 0
}

// Update moves the clock on; a level a laser caught the player in starts again.
func (world *World) Update(seconds float64) {
	world.Clock += seconds
	if world.resetAt >= 0 && world.Clock >= world.resetAt {
		world.Reset()
		return
	}
	if world.resetAt < 0 && !world.Busy() && world.laserHitsPlayer() {
		world.resetAt = world.Clock + 0.25
		world.events = append(world.events, EventReset)
	}
}

// Reset starts the level again.
func (world *World) Reset() {
	events := world.events
	*world = *NewWorld(world.Level)
	world.events = events
}

// Move walks the player a cell, pushing what it can.
func (world *World) Move(direction Direction) {
	if world.Busy() || world.Complete || world.Player == nil {
		return
	}
	player := world.Player
	if blocker := world.solidAt(player.X+direction.X, player.Y+direction.Y); blocker != nil {
		if !blocker.movable() || !world.push(blocker, direction) {
			world.bump(direction)
			return
		}
		world.events = append(world.events, EventPush)
	}
	world.Steps++
	world.events = append(world.events, EventStep)
	world.moveEntity(player, direction, playerMoveTime)
	world.arrive(player, direction, 0)
}

// push moves a box or a key a cell, or spends a key on a door; it answers whether the way is free.
func (world *World) push(entity *Entity, direction Direction) bool {
	if entity.Blocked {
		return false
	}
	x, y := entity.X+direction.X, entity.Y+direction.Y
	blocker := world.solidAt(x, y)
	if blocker == nil {
		world.moveEntity(entity, direction, pushedMoveTime)
		world.arrive(entity, direction, 0)
		return true
	}
	if entity.Kind == KindKey && blocker.Kind == KindDoor {
		blocker.Gone, entity.Gone = true, true
		world.events = append(world.events, EventDoorOpen)
		return true
	}
	return false
}

func (world *World) moveEntity(entity *Entity, direction Direction, seconds float64) {
	entity.fromX, entity.fromY = entity.X, entity.Y
	entity.X, entity.Y = entity.X+direction.X, entity.Y+direction.Y
	entity.viaX, entity.viaY = entity.X, entity.Y
	entity.moveStart, entity.moveTime = world.Clock, seconds
}

// arrive is what happens where something has just stepped: buttons let go and are pressed, a
// teleport sends it on, the heart ends the level.
func (world *World) arrive(entity *Entity, direction Direction, depth int) {
	world.releaseButtons()
	for _, other := range world.Entities {
		if other.Gone || other == entity || other.X != entity.X || other.Y != entity.Y {
			continue
		}
		switch other.Kind {
		case KindTeleport:
			if depth < 4 {
				world.teleport(entity, other, direction, depth)
				return
			}
		case KindButton:
			if !other.Disabled && !other.Active {
				other.Active = true
				world.buttonPressed(other, entity)
			}
		case KindHeart:
			if entity == world.Player {
				world.Complete = true
				world.events = append(world.events, EventComplete)
			}
		}
	}
}

// teleport sends what stepped into a teleport out of its pair, the way it was going; if that cell
// is taken (and the player cannot push what is there), it stays on the teleport.
func (world *World) teleport(entity, teleport *Entity, direction Direction, depth int) {
	pair := teleport.pair
	if pair == nil {
		return
	}
	x, y := pair.X+direction.X, pair.Y+direction.Y
	if blocker := world.solidAt(x, y); blocker != nil {
		if entity != world.Player || !blocker.movable() || !world.push(blocker, direction) {
			world.bump(direction)
			return
		}
		world.Steps++
	}
	entity.X, entity.Y = x, y // drawn walking into the teleport, then out of the pair
	world.events = append(world.events, EventTeleport)
	world.arrive(entity, direction, depth+1)
}

// releaseButtons lets go of the buttons nothing stands on any more.
func (world *World) releaseButtons() {
	for _, button := range world.Entities {
		if button.Kind != KindButton || button.Disabled || !button.Active {
			continue
		}
		if world.weightAt(button.X, button.Y) == nil {
			button.Active = false
		}
	}
}

func (world *World) bump(direction Direction) {
	world.bumpStart, world.bumpDir = world.Clock, direction
	world.events = append(world.events, EventBump)
}

// solidAt is what stops a step into a cell: a wall, a box, a key, a door or the player.
func (world *World) solidAt(x, y int) *Entity {
	if _, wall := world.walls[[2]int{x, y}]; wall {
		return &Entity{Thing: Thing{Kind: KindWall, X: x, Y: y}}
	}
	return world.weightAt(x, y)
}

// weightAt is the body in a cell, if any, other than the tile map's walls.
func (world *World) weightAt(x, y int) *Entity {
	for _, entity := range world.Entities {
		if !entity.Gone && entity.X == x && entity.Y == y && entity.solid() {
			return entity
		}
	}
	return nil
}

func (entity *Entity) solid() bool {
	switch entity.Kind {
	case KindWall, KindBox, KindKey, KindDoor, KindPlayer:
		return true
	}
	return false
}

func (entity *Entity) movable() bool { return entity.Kind == KindBox || entity.Kind == KindKey }

// Beam is a laser's: the cells it crosses, from the laser's own, and where it stops.
type Beam struct {
	Laser      *Entity
	Cells      int  // how many cells it crosses before what stops it
	HitsPlayer bool // what stops it is the player
}

// maxBeamCells is how far a beam goes with nothing in its way (2000 pixels in the original).
const maxBeamCells = 125

// Beams are the lasers' beams now.
func (world *World) Beams() []Beam {
	var beams []Beam
	for _, laser := range world.Entities {
		if laser.Kind != KindLaser || laser.Gone || laser.Disabled {
			continue
		}
		direction := laser.direction()
		beam := Beam{Laser: laser, Cells: maxBeamCells}
		for step := 0; step < maxBeamCells; step++ {
			blocker := world.solidAt(laser.X+direction.X*step, laser.Y+direction.Y*step)
			if blocker != nil {
				beam.Cells, beam.HitsPlayer = step, blocker == world.Player
				break
			}
		}
		beams = append(beams, beam)
	}
	return beams
}

func (world *World) laserHitsPlayer() bool {
	for _, beam := range world.Beams() {
		if beam.HitsPlayer {
			return true
		}
	}
	return false
}

func (entity *Entity) direction() Direction {
	switch entity.Direction {
	case "right":
		return Right
	case "down":
		return Down
	case "left":
		return Left
	}
	return Up
}

// DrawnAt is where an entity is drawn now, in pixels: walking from its last cell (into a teleport,
// then out of its pair at once), eased as the original's sine tweens.
func (world *World) DrawnAt(entity *Entity) (x, y float64) {
	progress := 1.0
	if entity.moveTime > 0 {
		progress = (world.Clock - entity.moveStart) / entity.moveTime
	}
	if progress >= 1 {
		return float64(entity.X * TileSize), float64(entity.Y * TileSize)
	}
	eased := (1 - math.Cos(math.Pi*max(progress, 0))) / 2
	return (float64(entity.fromX) + float64(entity.viaX-entity.fromX)*eased) * TileSize,
		(float64(entity.fromY) + float64(entity.viaY-entity.fromY)*eased) * TileSize
}

// Shake is how far the camera is pushed now by the last bump, in pixels.
func (world *World) Shake() (x, y float64) {
	elapsed := world.Clock - world.bumpStart
	if elapsed < 0 || elapsed > bumpTime {
		return 0, 0
	}
	amount := 2 * math.Sin(elapsed/bumpTime*math.Pi*4) * (1 - elapsed/bumpTime)
	return amount * float64(world.bumpDir.X), amount * float64(world.bumpDir.Y)
}

// Wall tells whether the tile map has a wall in a cell, and which of its drawings.
func (world *World) Wall(x, y int) (variant int, found bool) {
	variant, found = world.walls[[2]int{x, y}]
	return variant, found
}

// Named is the level's entity of a name (its path in the scene).
func (world *World) Named(name string) *Entity { return world.byName[name] }

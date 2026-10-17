package game

import (
	"testing"

	"github.com/akrck02/linus-tiny-adventure/3ds/assets"
)

// level makes a level of walls around a room, with things in it.
func level(number int, things ...Thing) *Level {
	result := &Level{Number: number, Things: things}
	for x := -1; x <= 8; x++ {
		result.Tiles = append(result.Tiles, Tile{X: x, Y: -1}, Tile{X: x, Y: 5})
	}
	for y := 0; y <= 4; y++ {
		result.Tiles = append(result.Tiles, Tile{X: -1, Y: y}, Tile{X: 8, Y: y})
	}
	return result
}

// settle lets every move finish.
func settle(world *World) {
	for step := 0; step < 100 && world.Busy(); step++ {
		world.Update(0.05)
	}
}

func move(world *World, directions ...Direction) {
	for _, direction := range directions {
		world.Move(direction)
		settle(world)
		world.Update(0) // a laser looks once all is still
		settle(world)
	}
}

func TestThePlayerPushesABoxUntilAWallStopsIt(t *testing.T) {
	world := NewWorld(level(2, Thing{Kind: KindPlayer, Name: "Player", X: 5, Y: 2}, Thing{Kind: KindBox, Name: "Box", X: 6, Y: 2}))
	move(world, Right)
	if box := world.Named("Box"); box.X != 7 || world.Player.X != 6 || world.Steps != 1 {
		t.Fatalf("box at %d, player at %d, %d steps", box.X, world.Player.X, world.Steps)
	}
	move(world, Right) // the box is against the wall now
	if world.Player.X != 6 || world.Steps != 1 {
		t.Fatalf("pushing a box into a wall moved the player to %d (%d steps)", world.Player.X, world.Steps)
	}
}

func TestAKeyPushedIntoADoorOpensItAndIsSpent(t *testing.T) {
	world := NewWorld(level(2, Thing{Kind: KindPlayer, Name: "Player", X: 1, Y: 2}, Thing{Kind: KindKey, Name: "Key", X: 2, Y: 2},
		Thing{Kind: KindDoor, Name: "Door", X: 3, Y: 2}))
	move(world, Right)
	if !world.Named("Door").Gone || !world.Named("Key").Gone || world.Player.X != 2 {
		t.Fatalf("door gone %v, key gone %v, player at %d", world.Named("Door").Gone, world.Named("Key").Gone, world.Player.X)
	}
}

func TestATeleportSendsThePlayerOutOfItsPairTheSameWay(t *testing.T) {
	world := NewWorld(level(2, Thing{Kind: KindPlayer, Name: "Player", X: 0, Y: 1},
		Thing{Kind: KindTeleport, Name: "A", X: 1, Y: 1, Pair: "B"}, Thing{Kind: KindTeleport, Name: "B", X: 5, Y: 3, Pair: "A"}))
	move(world, Right)
	if world.Player.X != 6 || world.Player.Y != 3 {
		t.Fatalf("the player came out at %d,%d, want 6,3", world.Player.X, world.Player.Y)
	}
}

func TestATeleportWhoseWayOutIsTakenKeepsWhatWalkedIn(t *testing.T) {
	world := NewWorld(level(2, Thing{Kind: KindPlayer, Name: "Player", X: 0, Y: 1},
		Thing{Kind: KindTeleport, Name: "A", X: 1, Y: 1, Pair: "B"}, Thing{Kind: KindTeleport, Name: "B", X: 7, Y: 3, Pair: "A"}))
	move(world, Right) // out of B to the right is the wall
	if world.Player.X != 1 || world.Player.Y != 1 {
		t.Fatalf("the player is at %d,%d, want on the teleport 1,1", world.Player.X, world.Player.Y)
	}
}

func TestALaserResetsTheLevelWhenThePlayerStepsIntoIt(t *testing.T) {
	world := NewWorld(level(2, Thing{Kind: KindPlayer, Name: "Player", X: 2, Y: 0}, Thing{Kind: KindLaser, Name: "Laser", X: 3, Y: 4, Direction: "up"}))
	move(world, Right)
	world.Update(1)
	if world.Player.X != 2 || world.Steps != 0 {
		t.Fatalf("after the laser the player is at %d with %d steps: the level did not start again", world.Player.X, world.Steps)
	}
}

func TestBothBoxesOnTheFirstLevelsButtonsBreakTheWall(t *testing.T) {
	world := NewWorld(level(1,
		Thing{Kind: KindPlayer, Name: "Player", X: 1, Y: 1}, Thing{Kind: KindBox, Name: "Box", X: 2, Y: 1}, Thing{Kind: KindButton, Name: "Button", X: 3, Y: 1},
		Thing{Kind: KindBox, Name: "Box2", X: 2, Y: 3}, Thing{Kind: KindButton, Name: "Button2", X: 3, Y: 3},
		Thing{Kind: KindWall, Name: "BreakeableWall", X: 6, Y: 2}))
	move(world, Right)
	if world.Named("BreakeableWall").Gone || !world.Named("Box").Blocked {
		t.Fatal("one button broke the wall, or its box can still move")
	}
	move(world, Left, Down, Down, Right)
	if !world.Named("BreakeableWall").Gone {
		t.Fatal("both buttons pressed by boxes did not break the wall")
	}
}

func TestTheHeartEndsTheLevel(t *testing.T) {
	world := NewWorld(level(2, Thing{Kind: KindPlayer, Name: "Player", X: 1, Y: 1}, Thing{Kind: KindHeart, Name: "Heart", X: 2, Y: 1}))
	move(world, Right)
	if !world.Complete {
		t.Fatal("stepping on the heart did not end the level")
	}
}

func TestTheConvertedLevelsHaveAPlayerAndAHeartEach(t *testing.T) {
	data, err := assets.Files.ReadFile("levels.json")
	if err != nil {
		t.Fatal(err)
	}
	levels, err := LoadLevels(data)
	if err != nil || len(levels) != 3 {
		t.Fatalf("%d levels, %v", len(levels), err)
	}
	for _, level := range levels {
		world := NewWorld(&level)
		players, hearts := 0, 0
		for _, entity := range world.Entities {
			switch entity.Kind {
			case KindPlayer:
				players++
			case KindHeart:
				hearts++
			}
		}
		if players != 1 || hearts != 1 {
			t.Errorf("level %d: %d players, %d hearts", level.Number, players, hearts)
		}
		if _, onWall := world.Wall(world.Player.X, world.Player.Y); onWall {
			t.Errorf("level %d: the player starts in a wall", level.Number)
		}
	}
}

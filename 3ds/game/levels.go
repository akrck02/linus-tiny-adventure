package game

import "encoding/json"

// A level as the converter writes it from the Godot scene (tools/convert) and the game reads it:
// cells are 16 pixels, counted from the scene's origin, and may be negative.

// TileSize is a cell's side in pixels, as in the original game.
const TileSize = 16

// Level is one of the game's levels.
type Level struct {
	Number int     `json:"number"`
	Name   string  `json:"name"`
	Tiles  []Tile  `json:"tiles"`  // the walls of the tile map
	Things []Thing `json:"things"` // the player, boxes, keys, doors, buttons, teleports, lasers…
	Labels []Label `json:"labels"` // texts written in the world
}

// Tile is a wall of the tile map: its cell and which of the three wall drawings it shows.
type Tile struct {
	X       int `json:"x"`
	Y       int `json:"y"`
	Variant int `json:"variant"`
}

// The kinds of things.
const (
	KindPlayer   = "player"
	KindBox      = "box"
	KindKey      = "key"
	KindDoor     = "door"
	KindWall     = "wall"
	KindButton   = "button"
	KindTeleport = "teleport"
	KindLaser    = "laser"
	KindHeart    = "heart"
	KindRadio    = "radio"
)

// Thing is something placed in a level.
type Thing struct {
	Kind      string `json:"kind"`
	Name      string `json:"name"` // its path in the scene: the level's rules refer to it
	X         int    `json:"x"`
	Y         int    `json:"y"`
	Pair      string `json:"pair,omitempty"`      // a teleport's other end
	Direction string `json:"direction,omitempty"` // a laser's: up, right, down or left
	Colour    string `json:"colour,omitempty"`    // a button's, as rrggbb
	Scale     int    `json:"scale,omitempty"`     // a heart drawn larger (the last one)
	Song      string `json:"song,omitempty"`      // a radio's: the music it plays (assets/sounds)
}

// Label is a text in the world, in pixels from the scene's origin.
type Label struct {
	Text   string  `json:"text"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
	Size   float64 `json:"size"`
	Centre bool    `json:"centre"`
}

// LoadLevels reads the levels the converter wrote.
func LoadLevels(data []byte) ([]Level, error) {
	var levels []Level
	if err := json.Unmarshal(data, &levels); err != nil {
		return nil, err
	}
	return levels, nil
}

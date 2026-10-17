// Package assets holds what tools/convert wrote from the Godot project: the levels, the sprites as
// PNG strips, the sounds and the game's font. Run the converter again after changing the original.
package assets

import "embed"

//go:embed levels.json sprites/*.png sounds/* early-gameboy.ttf
var Files embed.FS

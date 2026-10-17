// Command convert turns the Godot project's levels and sprites into what the 3DS port embeds:
// assets/levels.json and the sprites as PNG strips. It runs on a computer, from the 3ds folder:
//
//	go run ./tools/convert
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/png"
	"io"
	"log"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/akrck02/linus-tiny-adventure/3ds/game"
)

func main() {
	project := flag.String("project", "../LinusTinyAdventure", "the Godot project")
	out := flag.String("out", "assets", "where to write")
	flag.Parse()
	if err := convert(*project, *out); err != nil {
		log.Fatal(err)
	}
}

func convert(project, out string) error {
	if err := os.MkdirAll(filepath.Join(out, "sprites"), 0o755); err != nil {
		return err
	}
	levels, err := convertLevels(filepath.Join(project, "scenes", "levels"))
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(levels, "", "\t")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(out, "levels.json"), data, 0o644); err != nil {
		return err
	}
	if err := convertSprites(project, filepath.Join(out, "sprites")); err != nil {
		return err
	}
	return copyFile(filepath.Join(project, "fonts", "Early GameBoy.ttf"), filepath.Join(out, "early-gameboy.ttf"))
}

// convertLevels reads every "N-Name.tscn" in order.
func convertLevels(folder string) ([]game.Level, error) {
	entries, err := os.ReadDir(folder)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".tscn") {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	var levels []game.Level
	for index, name := range names {
		level, err := convertLevel(filepath.Join(folder, name))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		level.Number = index + 1
		level.Name = strings.TrimSuffix(name[strings.Index(name, "-")+1:], ".tscn")
		levels = append(levels, level)
	}
	return levels, nil
}

// instanced are the scenes a level places, by what they are in the port.
var instanced = map[string]string{
	"player.tscn": game.KindPlayer, "box.tscn": game.KindBox, "key.tscn": game.KindKey, "door.tscn": game.KindDoor,
	"wall.tscn": game.KindWall, "button.tscn": game.KindButton, "teleport.tscn": game.KindTeleport,
	"laser.tscn": game.KindLaser, "heart.tscn": game.KindHeart, "radio.tscn": game.KindRadio,
}

func convertLevel(path string) (game.Level, error) {
	var level game.Level
	source, err := readScene(path)
	if err != nil {
		return level, err
	}
	for _, item := range source.nodes {
		x, y := source.position(item)
		switch {
		case item.kind == "TileMap":
			tiles, err := tilesOf(item, x, y)
			if err != nil {
				return level, err
			}
			level.Tiles = append(level.Tiles, tiles...)
		case item.kind == "RichTextLabel":
			level.Labels = append(level.Labels, labelOf(item, x, y))
		case instanced[filepath.Base(item.instance)] != "":
			level.Things = append(level.Things, thingOf(item, instanced[filepath.Base(item.instance)], x, y))
		}
	}
	return level, nil
}

// cell is the cell a pixel position falls in.
func cell(value float64) int { return int(math.Floor(value/game.TileSize + 1e-6)) }

func thingOf(item *node, kind string, x, y float64) game.Thing {
	thing := game.Thing{Kind: kind, Name: item.path()}
	switch kind {
	case game.KindHeart:
		// its sprite is centred on it, 16 pixels wide times its scale
		scale := 1.0
		if sx, _, found := item.vector("scale"); found {
			scale = sx
		}
		thing.Scale = int(scale + 0.5)
		x, y = x-8*scale, y-8*scale
	case game.KindLaser:
		// a laser fires up; turned, it fires the way it was turned, its sprite turned around its
		// top-left corner
		thing.Direction = "up"
		switch int(math.Round(item.number("rotation") / (math.Pi / 2))) {
		case 1, -3:
			thing.Direction, x = "right", x-game.TileSize
		case 2, -2:
			thing.Direction, x, y = "down", x-game.TileSize, y-game.TileSize
		case 3, -1:
			thing.Direction, y = "left", y-game.TileSize
		}
	case game.KindTeleport:
		if pair := item.properties["pair"]; pair != "" {
			relative := strings.TrimSuffix(strings.TrimPrefix(pair, `NodePath("`), `")`)
			thing.Pair = resolve(item, relative)
		}
	case game.KindButton:
		thing.Colour = strings.Trim(item.properties["color"], `"`)
	}
	thing.X, thing.Y = cell(x), cell(y)
	return thing
}

// tilesOf reads a TileMap's first layer: three integers a cell, its coordinates (x in the low 16
// bits, y in the high ones), its source with the atlas x in the high bits, and the atlas y.
func tilesOf(item *node, x, y float64) ([]game.Tile, error) {
	data := item.properties["layer_0/tile_data"]
	data = strings.TrimSuffix(strings.TrimPrefix(data, "PackedInt32Array("), ")")
	if strings.TrimSpace(data) == "" {
		return nil, nil
	}
	fields := strings.Split(data, ",")
	var tiles []game.Tile
	for index := 0; index+2 < len(fields); index += 3 {
		packed, err := strconv.ParseInt(strings.TrimSpace(fields[index]), 10, 64)
		if err != nil {
			return nil, err
		}
		source, _ := strconv.ParseInt(strings.TrimSpace(fields[index+1]), 10, 64)
		column, row := int(int16(uint32(packed)&0xffff)), int(int16(uint32(packed)>>16))
		tiles = append(tiles, game.Tile{X: cell(x) + column, Y: cell(y) + row, Variant: int(source >> 16)})
	}
	return tiles, nil
}

func labelOf(item *node, x, y float64) game.Label {
	text := item.text("text")
	centre := strings.Contains(text, "[center]")
	for _, tag := range []string{"[center]", "[/center]"} {
		text = strings.ReplaceAll(text, tag, "")
	}
	left, top := item.number("offset_left"), item.number("offset_top")
	size := item.number("theme_override_font_sizes/normal_font_size")
	if size == 0 {
		size = 16
	}
	return game.Label{Text: strings.TrimSpace(text), X: x + left, Y: y + top,
		Width: item.number("offset_right") - left, Height: item.number("offset_bottom") - top, Size: size, Centre: centre}
}

// convertSprites writes every sprite the port draws as a PNG strip.
func convertSprites(project, out string) error {
	items, err := filepath.Glob(filepath.Join(project, "sprites", "items", "*.svg"))
	if err != nil {
		return err
	}
	for _, path := range items {
		picture, err := rasteriseSVG(path)
		if err != nil {
			return err
		}
		name := strings.ToLower(strings.TrimSuffix(filepath.Base(path), ".svg"))
		if err := writePNG(filepath.Join(out, name+".png"), picture); err != nil {
			return err
		}
	}
	for _, name := range []string{"characters/player.png", "characters/shadow.png", "background/logo.png", "background/press_start.png"} {
		target := strings.ReplaceAll(name, "/", "-")
		if err := copyFile(filepath.Join(project, "sprites", name), filepath.Join(out, target)); err != nil {
			return err
		}
	}
	return nil
}

func writePNG(path string, picture image.Image) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return png.Encode(file, picture)
}

func copyFile(from, to string) error {
	source, err := os.Open(from)
	if err != nil {
		return err
	}
	defer source.Close()
	target, err := os.Create(to)
	if err != nil {
		return err
	}
	defer target.Close()
	_, err = io.Copy(target, source)
	return err
}

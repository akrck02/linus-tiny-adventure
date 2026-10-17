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
	"os/exec"
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
	if err := convertSounds(project, filepath.Join(out, "sounds")); err != nil {
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

// songs are the level being converted's resources, for its radio's song.
var songs map[string]string

func convertLevel(path string) (game.Level, error) {
	var level game.Level
	source, err := readScene(path)
	if err != nil {
		return level, err
	}
	songs = source.resources
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
	case game.KindRadio:
		if song := resourcePattern.FindStringSubmatch(item.properties["current_song"]); song != nil {
			thing.Song = soundName(songs[song[1]])
		}
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
	if err := convertInterface(project, out); err != nil {
		return err
	}
	for _, name := range []string{"characters/player.png", "characters/shadow.png", "background/logo.png", "background/press_start.png",
		"ui/lock.png", "ui/option_exit.png", "ui/slider_grab.png", "ui/slider_grab_selected.png", "ui/music.png"} {
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

// soundName is what a sound is called in the port: its file's name, lower case, words joined by
// dashes ("res://sounds/effects/box move.mp3" is "box-move").
func soundName(path string) string {
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	return strings.ReplaceAll(strings.ToLower(name), " ", "-")
}

// convertSounds turns the MP3s into what go-sound reads, with ffmpeg: the music into Ogg Vorbis
// (four minutes would be ten megabytes as WAV), the effects into WAV (short, played often:
// decoding would cost more than the bytes). Mono at 22050 frames a second: the console's speakers
// and the original's sounds lose nothing that would be heard.
func convertSounds(project, out string) error {
	if err := os.MkdirAll(out, 0o755); err != nil {
		return err
	}
	music, err := filepath.Glob(filepath.Join(project, "sounds", "OST", "*.mp3"))
	if err != nil {
		return err
	}
	for _, path := range music {
		if err := ffmpeg(path, filepath.Join(out, soundName(path)+".ogg"), "-c:a", "libvorbis", "-q:a", "2"); err != nil {
			return err
		}
	}
	effects, err := filepath.Glob(filepath.Join(project, "sounds", "effects", "*.mp3"))
	if err != nil {
		return err
	}
	for _, path := range effects {
		if err := ffmpeg(path, filepath.Join(out, soundName(path)+".wav"), "-c:a", "pcm_s16le"); err != nil {
			return err
		}
	}
	return nil
}

func ffmpeg(from, to string, codec ...string) error {
	arguments := append([]string{"-v", "error", "-y", "-i", from, "-ac", "1", "-ar", "22050"}, codec...)
	command := exec.Command("ffmpeg", append(arguments, to)...)
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("ffmpeg %s: %w", from, err)
	}
	return nil
}

// convertInterface writes the interface's icons: the menu's four (pixel art on a grid of ten,
// drawn at a tenth), the camera filters' and the controls'.
func convertInterface(project, out string) error {
	icons := filepath.Join(project, "sprites", "ui")
	for _, name := range []string{"levels", "controls", "graphics", "sound"} {
		picture, err := rasteriseSVGScaled(filepath.Join(icons, name+".svg"), 0.1)
		if err != nil {
			return err
		}
		if err := writePNG(filepath.Join(out, "ui-"+name+".png"), picture); err != nil {
			return err
		}
	}
	for _, folder := range []string{"graphics", "controls/gamepad", "controls/keyboard"} {
		paths, err := filepath.Glob(filepath.Join(icons, folder, "*.svg"))
		if err != nil {
			return err
		}
		for _, path := range paths {
			picture, err := rasteriseSVG(path)
			if err != nil {
				return err
			}
			name := "ui-" + strings.ReplaceAll(folder, "/", "-") + "-" + strings.TrimSuffix(filepath.Base(path), ".svg")
			if err := writePNG(filepath.Join(out, name+".png"), picture); err != nil {
				return err
			}
		}
	}
	return nil
}

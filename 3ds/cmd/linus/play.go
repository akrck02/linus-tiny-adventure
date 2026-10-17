//go:build nintendo3ds

package main

import (
	"strconv"
	"strings"

	"github.com/akrck02/go-renderer/ctr"
	"github.com/akrck02/go-renderer/draw2d"
	ui "github.com/akrck02/go-ui"

	"github.com/akrck02/linus-tiny-adventure/3ds/assets"
	"github.com/akrck02/linus-tiny-adventure/3ds/game"
)

// The screens of the game.
const (
	screenTitle = iota
	screenLevel
	screenLevels
)

// play is the game's state: the levels, the one played, how far the player has got.
type play struct {
	levels   []game.Level
	sprites  *game.Sprites
	world    *game.World
	camera   game.Camera
	screen   int
	reached  int     // the highest level opened this time the game runs
	finished float64 // when the last level was finished, to say so for a while; 0 when not
	clock    float64
	audio    *audio
}

func newPlay() (*play, error) {
	data, err := assets.Files.ReadFile("levels.json")
	if err != nil {
		return nil, err
	}
	levels, err := game.LoadLevels(data)
	if err != nil {
		return nil, err
	}
	sprites, err := game.LoadSprites(assets.Files)
	if err != nil {
		return nil, err
	}
	reached := 1
	if count, err := strconv.Atoi(unlocked); err == nil {
		reached = max(1, min(count, len(levels)))
	}
	return &play{levels: levels, sprites: sprites, reached: reached, audio: newAudio()}, nil
}

// unlocked, set at build time, opens levels from the start, to try them:
//
//	tinygo build -target=nintendo3ds -ldflags="-X main.unlocked=3" -o linus.3dsx ./cmd/linus
var unlocked string

// open starts a level, by its number.
func (play *play) open(number int) {
	if number < 1 || number > len(play.levels) {
		return
	}
	play.world = game.NewWorld(&play.levels[number-1])
	play.camera.Jump()
	play.reached = max(play.reached, number)
	play.screen = screenLevel
}

// walkThreshold is how far the circle pad leans before it walks.
const walkThreshold = 0.5

// step moves the game on: the player walks while a direction is held, as in the original.
func (play *play) step(reading ctr.Reading, pressed ctr.Buttons, seconds float64) {
	play.clock += seconds
	if play.world == nil {
		play.audio.follow(nil)
	}
	switch play.screen {
	case screenTitle:
		if pressed&ctr.ButtonA != 0 {
			play.open(play.reached)
		}
		return
	case screenLevels:
		if pressed&ctr.ButtonB != 0 {
			play.screen = screenLevel
			if play.world == nil {
				play.screen = screenTitle
			}
		}
		return
	}
	world := play.world
	world.Update(seconds)
	if pressed&ctr.ButtonSelect != 0 {
		world.Reset()
	}
	if pressed&ctr.ButtonY != 0 {
		play.screen = screenLevels
	}
	if direction, held := heldDirection(reading); held {
		world.Move(direction)
	}
	play.audio.play(world.Events())
	play.audio.follow(world)
	if world.Complete && !world.Busy() {
		if world.Level.Number < len(play.levels) {
			play.open(world.Level.Number + 1)
		} else {
			play.finished = play.clock
			play.world = nil
			play.screen = screenTitle
		}
		return
	}
	play.camera.Follow(world, ctr.ScreenWidth, ctr.ScreenHeight, seconds)
}

func heldDirection(reading ctr.Reading) (game.Direction, bool) {
	circleX, circleY := reading.Circle()
	switch {
	case reading.Held&ctr.ButtonRight != 0 || circleX > walkThreshold:
		return game.Right, true
	case reading.Held&ctr.ButtonLeft != 0 || circleX < -walkThreshold:
		return game.Left, true
	case reading.Held&ctr.ButtonUp != 0 || circleY > walkThreshold:
		return game.Up, true
	case reading.Held&ctr.ButtonDown != 0 || circleY < -walkThreshold:
		return game.Down, true
	}
	return game.Direction{}, false
}

// The palette of the original: its background, white ink and a darker mint for keys.
var (
	mint      = game.Background
	mintDark  = draw2d.Color{0.6, 0.74, 0.68, 1}
	mintDeep  = draw2d.Color{0.46, 0.6, 0.54, 1}
	ink       = draw2d.Hex("#ffffff")
	inkShadow = draw2d.Color{0, 0, 0, 0.16}
)

func theme() ui.Theme {
	theme := ui.LightTheme()
	theme.Background, theme.Ink, theme.InkSoft, theme.InkQuiet = mint, ink, ink, mintDeep
	theme.TextSize, theme.SmallTextSize, theme.HeadingSize = 8, 8, 16
	theme.ControlRadius, theme.PanelRadius = 0, 0
	return theme
}

// drawTop draws the top screen: the title, or the level.
func (play *play) drawTop(list *draw2d.List) {
	if play.screen == screenTitle || play.world == nil {
		list.Rectangle(draw2d.Rect{Width: ctr.ScreenWidth, Height: ctr.ScreenHeight}, 0, mint, draw2d.Color{}, 0)
		title := "Linu's tiny adventure"
		size := float32(16)
		x := float32(ctr.ScreenWidth)/2 - float32(len(title))*size/2
		shadowed(list, title, x, 112, size)
		if play.finished > 0 {
			shadowed(list, "you beat the game", float32(ctr.ScreenWidth)/2-17*4, 140, 8)
		}
		if int(play.clock*2)%2 == 0 {
			shadowed(list, "press a", float32(ctr.ScreenWidth)/2-7*4, 168, 8)
		}
		return
	}
	play.sprites.Draw(list, play.world, &play.camera, ctr.ScreenWidth, ctr.ScreenHeight)
}

func shadowed(list *draw2d.List, text string, x, baseline, size float32) {
	list.Text(text, x+1, baseline+1, size, draw2d.Regular, inkShadow)
	list.Text(text, x, baseline, size, draw2d.Regular, ink)
}

// buildBottom builds the bottom screen: the title's start, the level's figures and keys, or the
// levels to choose.
func (play *play) buildBottom(context *ui.Context) {
	context.Box(ui.Layout{Direction: ui.Vertical, Width: ui.Grow(), Height: ui.Grow(), Padding: ui.PadXY(16, 14), Gap: 10}, func() {
		switch play.screen {
		case screenTitle:
			play.titleScreen(context)
		case screenLevels:
			play.levelsScreen(context)
		default:
			play.levelScreen(context)
		}
	})
}

func (play *play) titleScreen(context *ui.Context) {
	words(context, "LINU'S TINY", 16)
	words(context, "ADVENTURE", 16)
	context.Box(ui.Layout{Height: ui.Grow()}, nil)
	if key(context, "play", "PLAY - A", true) {
		play.open(play.reached)
	}
	if key(context, "levels", "LEVELS", true) {
		play.screen = screenLevels
	}
	if play.audio.silent != "" {
		words(context, play.audio.silent, 8)
	}
	words(context, "START: HOME MENU", 8)
}

func (play *play) levelScreen(context *ui.Context) {
	world := play.world
	words(context, "LEVEL "+strconv.Itoa(world.Level.Number), 8)
	words(context, strings.ToUpper(world.Level.Name), 16)
	context.Box(ui.Layout{Height: ui.Grow()}, nil)
	words(context, "STEPS: "+strconv.Itoa(world.Steps), 16)
	context.Box(ui.Layout{Height: ui.Grow()}, nil)
	context.Box(ui.Layout{Width: ui.Grow(), Gap: 8}, func() {
		if key(context, "reset", "RESET - SELECT", true) {
			world.Reset()
		}
		if key(context, "levels", "LEVELS - Y", true) {
			play.screen = screenLevels
		}
	})
	words(context, "WALK: D-PAD OR CIRCLE PAD", 8)
}

func (play *play) levelsScreen(context *ui.Context) {
	words(context, "LEVEL SELECTION", 16)
	for index := range play.levels {
		level := &play.levels[index]
		open := level.Number <= play.reached
		label := strconv.Itoa(level.Number) + "  " + strings.ToUpper(level.Name)
		if !open {
			label = strconv.Itoa(level.Number) + "  ???"
		}
		if key(context, "level "+strconv.Itoa(level.Number), label, open) && open {
			play.open(level.Number)
		}
	}
	context.Box(ui.Layout{Height: ui.Grow()}, nil)
	words(context, "B: BACK", 8)
}

// words is white text in the game's font with its soft shadow.
func words(context *ui.Context, text string, size float32) {
	context.Text(text, ui.TextStyle{Size: size, Color: ink, Shadow: inkShadow})
}

// key is a button in the game's look: a darker mint block with a white edge that darkens when
// pressed; a closed one is quiet.
func key(context *ui.Context, name, label string, open bool) bool {
	normal := ui.Look{Fill: mintDark, Border: ink, BorderWidth: 2}
	if !open {
		normal = ui.Look{Fill: mint, Border: mintDark, BorderWidth: 2}
	}
	looks := ui.Looks{Normal: normal, Hovered: normal, Pressed: ui.Look{Fill: mintDeep, Border: ink, BorderWidth: 2}}
	return context.Clickable(name, ui.Layout{Width: ui.Grow(), Height: ui.Fixed(32), Padding: ui.PadXY(10, 0), AlignY: ui.Center}, looks, func() {
		words(context, label, 8)
	})
}

//go:build nintendo3ds

package main

import (
	"math"
	"strconv"
	"strings"

	"github.com/akrck02/go-renderer/ctr"
	"github.com/akrck02/go-renderer/draw2d"
)

// The top screen. The loading screen is the original's, its 1280 × 720 laid on the console's 400 ×
// 240 at five sixteenths: the title above, "Loading..." below, Linu walking in from the left, and a
// darker panel on the right with the tips and the heart. The title screen starts from it, as the
// original's start does: Linu waits on the left and "Press start" takes the place of "Loading".

// The loading screen's colours: its background, the panel's darker shade over it, and the tip's
// faded white.
var (
	loaderBackground = draw2d.Color{0.639046, 0.768718, 0.711277, 1}
	loaderPanel      = draw2d.Color{0.00784314, 0.12549, 0.152941, 0.0705882}
	white            = draw2d.Color{1, 1, 1, 1}
	textShadow       = draw2d.Color{0, 0, 0, 0.12}
)

// The loading screen's walk: Linu's place at the original's moments (in its pixels), a hop of a
// cell at a time, and then down off the screen.
var loaderWalk = []struct {
	time float64
	x, y float64
}{
	{0.9, -48, 304}, {1.3, 64, 304}, {1.7, 64, 304}, {2.1, 192, 304}, {3, 192, 304}, {3.5, 320, 304}, {3.8, 320, 304}, {4.3, 320, 448},
}

// loaderScale lays the original's 1280 × 720 on the top screen.
const loaderScale = 5.0 / 16

func (app *app) drawTop(list *draw2d.List) {
	switch app.state {
	case stateLevel:
		app.drawLevel(list)
	case stateLoading:
		app.drawLoader(list, app.clock-app.loadingStart, "Loading...")
	case stateSelector:
		app.drawLoader(list, -1, "Level selection")
	default:
		app.drawLoader(list, -1, "Press start")
	}
	app.sprites.Filter().Overlay(list, ctr.ScreenWidth, ctr.ScreenHeight, app.grain)
}

// pulse is the original's texts' breathing: full, then a little more than a third, every second.
func (app *app) pulse() float32 {
	return float32(0.684 + 0.316*math.Cos(app.clock*math.Pi))
}

// colour is a colour through the camera filter.
func (app *app) colour(colour draw2d.Color) draw2d.Color { return app.sprites.Filter().Colour(colour) }

// drawLoader draws the loading screen at a moment of its walk (below zero: Linu waits where he
// starts the title), with its bottom line saying what it says.
func (app *app) drawLoader(list *draw2d.List, elapsed float64, bottom string) {
	list.Rectangle(draw2d.Rect{Width: ctr.ScreenWidth, Height: ctr.ScreenHeight}, 0, app.colour(loaderBackground), draw2d.Color{}, 0)
	// the panel is as wide as its title and the heart need
	panel := ctr.ScreenWidth - max(app.fonts.Width("TIP'S FOR PLAYERS", 8)+40, 130)
	list.Rectangle(draw2d.Rect{X: panel, Width: ctr.ScreenWidth - panel, Height: ctr.ScreenHeight}, 0, app.colour(loaderPanel), draw2d.Color{}, 0)
	breathing := app.pulse()
	faded := func(alpha float32) draw2d.Color { colour := app.colour(white); colour[3] = alpha; return colour }
	app.text(list, "LINU'S TINY ADVENTURE", 10, 18, 8, faded(breathing))
	app.text(list, upper(bottom), 10, 226, 8, faded(breathing))
	// the panel: its title, a dashed line under it, the heart, and the tip
	app.text(list, "TIP'S FOR PLAYERS", panel+8, 20, 8, faded(breathing))
	for dash := 0; dash < 8; dash++ {
		list.Rectangle(draw2d.Rect{X: panel + 9 + float32(dash)*10, Y: 26, Width: 5, Height: 1}, 0, faded(0.6), draw2d.Color{}, 0)
	}
	if heart := app.sprites.Frame("heart", int(app.clock/0.75)); heart != nil {
		list.Image(heart, draw2d.Rect{X: 378, Y: 8, Width: 16, Height: 16}, 0, app.colour(draw2d.Color{1, 0.286275, 0.286275, 1}))
	}
	app.text(list, upper(tipFor(app.state)), panel+9, 40, 8, faded(0.58*breathing))
	// Linu, twice his size, on his shadow
	x, y := 64.0, 304.0 // where he waits on the title
	if elapsed >= 0 {
		x, y = loaderPlace(elapsed)
	}
	screenX, screenY := float32(math.Round(x*loaderScale)), float32(math.Round(y*loaderScale))+8
	if shadow := app.sprites.Frame("characters-shadow", 0); shadow != nil {
		list.Image(shadow, draw2d.Rect{X: screenX - 2, Y: screenY + 18, Width: 32, Height: 16}, 0, white)
	}
	if player := app.bigPlayer[int(app.clock/0.75)%len(app.bigPlayer)]; player != nil {
		list.Image(player, draw2d.Rect{X: screenX, Y: screenY, Width: 28, Height: 28}, 0, white)
	}
}

// tipFor is the tip the panel gives: the original's loading screen says one word.
func tipFor(state int) string {
	if state == stateSelector {
		return "Choose a level."
	}
	return "Walk."
}

// loaderPlace is where the walk has Linu at a moment, moving straight between its places.
func loaderPlace(elapsed float64) (x, y float64) {
	first := loaderWalk[0]
	if elapsed <= first.time {
		return first.x, first.y
	}
	for index := 1; index < len(loaderWalk); index++ {
		from, to := loaderWalk[index-1], loaderWalk[index]
		if elapsed <= to.time {
			share := (elapsed - from.time) / (to.time - from.time)
			return from.x + (to.x-from.x)*share, from.y + (to.y-from.y)*share
		}
	}
	last := loaderWalk[len(loaderWalk)-1]
	return last.x, last.y
}

// drawLevel draws the level and, as the original over it, the steps and the reset.
func (app *app) drawLevel(list *draw2d.List) {
	if app.world == nil {
		return
	}
	app.sprites.Draw(list, app.world, &app.camera, ctr.ScreenWidth, ctr.ScreenHeight)
	app.text(list, "STEPS: "+strconv.Itoa(app.world.Steps), 10, 16, 8, app.colour(white))
	label, key := "RESET LEVEL", "SELECT"
	keyWidth := app.fonts.Width(key, 8) + 8
	app.text(list, label, ctr.ScreenWidth-10-keyWidth-6-app.fonts.Width(label, 8), 16, 8, app.colour(white))
	keyBox(list, app, key, ctr.ScreenWidth-10-keyWidth, 5, keyWidth, 14)
}

// text writes in the game's font with the original's soft shadow; baseline is the text's foot.
func (app *app) text(list *draw2d.List, text string, x, baseline, size float32, colour draw2d.Color) {
	shadow := textShadow
	shadow[3] *= colour[3]
	list.Text(text, x+1, baseline+1, size, draw2d.Regular, shadow)
	list.Text(text, x, baseline, size, draw2d.Regular, colour)
}

// keyBox is a key as the original's control icons draw one: an outline of two pixels with its
// name inside.
func keyBox(list *draw2d.List, app *app, name string, x, y, width, height float32) {
	colour := app.colour(white)
	list.Rectangle(draw2d.Rect{X: x, Y: y, Width: width, Height: height}, 0, draw2d.Color{}, colour, 1.5)
	textWidth := app.fonts.Width(name, 8)
	list.Text(name, x+(width-textWidth)/2, y+height-3, 8, draw2d.Regular, colour)
}

func upper(text string) string { return strings.ToUpper(text) }

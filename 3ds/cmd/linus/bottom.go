//go:build nintendo3ds

package main

import (
	"math"
	"strconv"

	"github.com/akrck02/go-renderer/ctr"
	"github.com/akrck02/go-renderer/draw2d"

	"github.com/akrck02/linus-tiny-adventure/3ds/game"
)

// The bottom screen, touched or moved through with the cross: the level selection's polaroids,
// the loading screen's line, and while a level plays its game menu and the options, as the
// original draws them over a blurred level. A chosen polaroid stands a little higher.

const (
	bottomWidth  = float32(ctr.BottomWidth)
	bottomHeight = float32(ctr.BottomHeight)
	raised       = 10 // how much higher the chosen polaroid stands
)

func (app *app) drawBottom(list *draw2d.List) {
	app.regions = app.regions[:0]
	list.Rectangle(draw2d.Rect{Width: bottomWidth, Height: bottomHeight}, 0, app.colour(game.Background), draw2d.Color{}, 0)
	switch app.state {
	case stateTitle:
		app.drawTitleBottom(list)
	case stateSelector:
		app.drawSelector(list)
	case stateLoading:
		app.text(list, "LOADING...", 10, bottomHeight-14, 8, app.faded(app.pulse()))
	case stateLevel:
		app.drawMenu(list)
	}
	if app.audio.silent != "" && app.state == stateTitle {
		app.text(list, app.audio.silent, 10, bottomHeight-14, 8, app.faded(0.7))
	}
	app.sprites.Filter().Overlay(list, bottomWidth, bottomHeight, app.grain)
}

func (app *app) faded(alpha float32) draw2d.Color {
	colour := app.colour(white)
	colour[3] = alpha
	return colour
}

func (app *app) centred(list *draw2d.List, text string, baseline, size float32, colour draw2d.Color) {
	app.text(list, text, (bottomWidth-app.fonts.Width(text, float64(size)))/2, baseline, size, colour)
}

// drawTitleBottom invites to start: the whole screen is a button.
func (app *app) drawTitleBottom(list *draw2d.List) {
	app.centred(list, "PRESS START", 128, 16, app.faded(app.pulse()))
	app.centred(list, "OR TOUCH THE SCREEN", 152, 8, app.faded(0.7))
	app.regions = append(app.regions, region{area: draw2d.Rect{Width: bottomWidth, Height: bottomHeight}, action: app.openSelector})
}

// drawSelector draws the levels' polaroids in a row, the chosen one in the middle, with its name
// above as the original's "Level selection" screen.
func (app *app) drawSelector(list *draw2d.List) {
	app.text(list, "LEVEL SELECTION", 10, 20, 8, app.colour(white))
	name := upper(app.levels[app.selected].Name)
	if app.selected+1 > app.reached {
		name = "???"
	}
	app.text(list, name, bottomWidth-10-app.fonts.Width(name, 8), 20, 8, app.colour(white))
	const spacing = 104
	for index, card := range app.levelCards {
		centreX := bottomWidth/2 + float32(index-app.selected)*spacing
		top := float32(84)
		if index == app.selected {
			top -= raised
		}
		area := app.putCard(list, card, centreX, top)
		level := index
		app.regions = append(app.regions, region{area: area, action: func() {
			if app.selected == level && level+1 <= app.reached {
				app.openLevel(level + 1)
				return
			}
			app.selected = level
		}})
	}
	app.centred(list, "A: PLAY   B: BACK", bottomHeight-14, 8, app.faded(0.7))
}

// putCard draws a polaroid centred on a column, its top at a height, and gives back where it is.
func (app *app) putCard(list *draw2d.List, card *draw2d.Image, centreX, top float32) draw2d.Rect {
	width, height := card.Size()
	area := draw2d.Rect{X: float32(math.Round(float64(centreX) - float64(width)/2)), Y: top, Width: float32(width), Height: float32(height)}
	list.Image(card, area, 0, white)
	return area
}

// drawMenu draws the game menu, or the option chosen in it.
func (app *app) drawMenu(list *draw2d.List) {
	switch app.menu {
	case menuControls:
		app.drawControls(list)
	case menuGraphics:
		app.drawGraphics(list)
	case menuSound:
		app.drawSound(list)
	default:
		app.drawMainMenu(list)
	}
}

// chosen tells whether a menu's entry is the one the cross is on (only while the menu is used).
func (app *app) chosen(entry int) bool { return app.menuFocused && app.focus == entry }

func (app *app) drawMainMenu(list *draw2d.List) {
	app.centred(list, "GAME MENU", 34, 16, app.colour(white))
	for index, card := range app.menuCards {
		top := float32(70)
		if app.chosen(index) {
			top -= raised
		}
		area := app.putCard(list, card, 41+float32(index)*79, top)
		entry := index
		app.regions = append(app.regions, region{area: area, action: func() { app.menuFocused = true; app.choose(entry) }})
	}
	app.drawExit(list, len(menuEntries))
	if !app.menuFocused {
		app.centred(list, "START: MENU   SELECT: RESET", bottomHeight-8, 8, app.faded(0.6))
	}
}

// drawExit draws the menu's exit, the original's arrow to a bar, under the polaroids.
func (app *app) drawExit(list *draw2d.List, entry int) {
	area := draw2d.Rect{X: bottomWidth/2 - 12, Y: 186, Width: 24, Height: 22}
	border := app.faded(0.5)
	if app.chosen(entry) {
		border = app.colour(white)
	}
	list.Rectangle(area, 0, draw2d.Color{1, 1, 1, 0.15}, border, 1.5)
	if app.exitIcon != nil {
		width, height := app.exitIcon.Size()
		list.Image(app.exitIcon, draw2d.Rect{X: area.X + (area.Width-float32(width))/2, Y: area.Y + (area.Height-float32(height))/2,
			Width: float32(width), Height: float32(height)}, 0, app.colour(white))
	}
	app.regions = append(app.regions, region{area: area, action: func() { app.menuFocused = true; app.choose(entry) }})
}

// drawControls shows the console's controls as the original shows the keyboard's: each named
// above its key.
func (app *app) drawControls(list *draw2d.List) {
	app.centred(list, "CONTROLS", 34, 16, app.colour(white))
	controls := []struct {
		name, key string
		icon      string
	}{
		{"UP", "", "ui-controls-keyboard-up"}, {"DOWN", "", "ui-controls-keyboard-down"},
		{"LEFT", "", "ui-controls-keyboard-left"}, {"RIGHT", "", "ui-controls-keyboard-right"},
		{"RESET LEVEL", "SELECT", ""}, {"GAME MENU", "START", ""}, {"CHOOSE", "A", ""}, {"BACK", "B", ""},
	}
	for index, control := range controls {
		// the arrows four in a row, the named keys two in a row: their names are longer
		column, row := index%4, 0
		centreX := 40 + float32(column)*80
		if index >= 4 {
			column, row = (index-4)%2, 1+(index-4)/2
			centreX = 80 + float32(column)*160
		}
		top := 56 + float32(row)*46
		app.text(list, control.name, centreX-app.fonts.Width(control.name, 8)/2, top, 8, app.colour(white))
		if control.icon != "" {
			if icon := app.sprites.Frame(control.icon, 0); icon != nil {
				list.Image(icon, draw2d.Rect{X: centreX - 16, Y: top + 8, Width: 32, Height: 32}, 0, app.colour(white))
			}
			continue
		}
		width := max(app.fonts.Width(control.key, 8)+12, 26)
		keyBox(list, app, control.key, centreX-width/2, top+12, width, 22)
	}
	app.drawExit(list, 0)
}

// drawGraphics shows the camera filters as polaroids, four and two, as the original does.
func (app *app) drawGraphics(list *draw2d.List) {
	app.centred(list, "GRAPHIC OPTIONS", 26, 16, app.colour(white))
	app.text(list, "CAMERA EFFECT:", 14, 44, 8, app.colour(white))
	for index, card := range app.filterCards {
		column, row := index%4, index/4
		top := 52 + float32(row)*68
		if app.chosen(index) {
			top -= raised / 2
		}
		area := app.putCard(list, card, 41+float32(column)*79, top)
		if game.Filters[index].Filter == app.sprites.Filter() {
			// the filter in use: a mark under its polaroid
			list.Rectangle(draw2d.Rect{X: area.X + area.Width/2 - 6, Y: area.Y + area.Height + 2, Width: 12, Height: 2}, 0, app.colour(white), draw2d.Color{}, 0)
		}
		entry := index
		app.regions = append(app.regions, region{area: area, action: func() { app.menuFocused = true; app.focus = entry; app.choose(entry) }})
	}
	app.drawExit(list, len(game.Filters))
}

// drawSound shows the music's and the effects' sliders, with their figures, as the original.
func (app *app) drawSound(list *draw2d.List) {
	app.centred(list, "SOUND OPTIONS", 40, 16, app.colour(white))
	for slider, name := range []string{"MUSIC", "EFFECTS"} {
		top := 76 + float32(slider)*48
		left, width := float32(60), float32(176)
		app.text(list, name, left, top, 8, app.colour(white))
		icon := "ui-music"
		if slider == 1 {
			icon = "ui-sound"
		}
		if picture := app.sprites.Frame(icon, 0); picture != nil {
			pictureWidth, pictureHeight := picture.Size()
			list.Image(picture, draw2d.Rect{X: left + width - float32(pictureWidth), Y: top - float32(pictureHeight) + 2, Width: float32(pictureWidth), Height: float32(pictureHeight)}, 0, app.colour(white))
		}
		volume := volumeOf(app, slider)
		line := draw2d.Rect{X: left, Y: top + 12, Width: width, Height: 2}
		list.Rectangle(line, 0, app.faded(0.6), draw2d.Color{}, 0)
		grab := "ui-slider_grab"
		if app.chosen(slider) {
			grab = "ui-slider_grab_selected"
		}
		if picture := app.sprites.Frame(grab, 0); picture != nil {
			pictureWidth, pictureHeight := picture.Size()
			x := float32(math.Round(float64(left + width*float32(volume) - float32(pictureWidth)/2)))
			list.Image(picture, draw2d.Rect{X: x, Y: line.Y + 1 - float32(pictureHeight)/2, Width: float32(pictureWidth), Height: float32(pictureHeight)}, 0, white)
		}
		app.text(list, strconv.Itoa(int(volume*100+0.5)), left+width+10, top+16, 8, app.colour(white))
		index := slider
		app.regions = append(app.regions, region{area: draw2d.Rect{X: left - 8, Y: top + 2, Width: width + 16, Height: 22}, action: func() {
			app.menuFocused, app.focus = true, index
			x, _ := app.touch.x, app.touch.y
			app.setVolume(index, float64(min(max((x-left)/width, 0), 1)))
		}})
	}
	app.drawExit(list, 2)
}

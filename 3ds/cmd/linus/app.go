//go:build nintendo3ds

package main

import (
	"image"
	"strconv"

	"github.com/akrck02/go-renderer/ctr"
	"github.com/akrck02/go-renderer/draw2d"

	"github.com/akrck02/linus-tiny-adventure/3ds/assets"
	"github.com/akrck02/linus-tiny-adventure/3ds/game"
)

// The game's screens, as the original goes through them: the title, the level selection, the
// loading screen and a level. A level's game menu and its options (controls, graphics, sound) live
// on the bottom screen, as a DS game keeps its menus there.
const (
	stateTitle = iota
	stateSelector
	stateLoading
	stateLevel
)

// The bottom screen's menus while a level plays.
const (
	menuMain = iota // the game menu's four polaroids
	menuControls
	menuGraphics
	menuSound
)

// The game menu's polaroids, in order.
var menuEntries = []struct{ name, icon string }{
	{"Levels", "ui-levels"}, {"Controls", "ui-controls"}, {"Graphics", "ui-graphics"}, {"Sound", "ui-sound"},
}

// loadingTime is how long the loading screen shows: the original's walk across it.
const loadingTime = 4.6

type app struct {
	levels  []game.Level
	sprites *game.Sprites
	fonts   *game.Fonts
	audio   *audio
	grain   *draw2d.Image

	state   int
	clock   float64
	reached int // the highest level opened this run

	selected     int     // the level the selector is on, from 0
	loadingLevel int     // the level the loading screen leads to
	loadingStart float64 // when it began

	world  *game.World
	camera game.Camera

	menu        int
	menuFocused bool // the player is in the menu (the level waits), not walking
	focus       int  // the menu's chosen entry; the last one is the exit

	music, effects float64 // 0 to 1, as the sound options set them

	// pictures made once: the levels' polaroids, the menu's and the filters'
	levelCards  []*draw2d.Image
	menuCards   []*draw2d.Image
	filterCards []*draw2d.Image
	bigPlayer   []*draw2d.Image // the loading screen's Linu, twice the size
	exitIcon    *draw2d.Image
	lock        *draw2d.Image

	regions []region // what a touch on the bottom screen does, as last drawn
	touch   touchState
}

// region is a part of the bottom screen a touch acts on.
type region struct {
	area   draw2d.Rect
	action func()
}

func newApp() (*app, error) {
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
	fontData, err := assets.Files.ReadFile("early-gameboy.ttf")
	if err != nil {
		return nil, err
	}
	fonts, err := game.NewFonts(fontData)
	if err != nil {
		return nil, err
	}
	reached := 1
	if count, err := strconv.Atoi(unlocked); err == nil {
		reached = max(1, min(count, len(levels)))
	}
	application := &app{levels: levels, sprites: sprites, fonts: fonts, audio: newAudio(), grain: game.NewGrain(400, 240),
		reached: reached, music: 0.5, effects: 0.5}
	application.sprites.SetFilter(game.DefaultFilter)
	application.makePictures()
	application.audio.setVolumes(application.music, application.effects)
	return application, nil
}

// unlocked, set at build time, opens levels from the start, to try them:
//
//	tinygo build -target=nintendo3ds -ldflags="-X main.unlocked=3" -o linus.3dsx ./cmd/linus
var unlocked string

// makePictures makes the polaroids and the large Linu, through the camera filter in use.
func (app *app) makePictures() {
	filter := app.sprites.Filter()
	turn := func(picture *draw2d.Image) *draw2d.Image {
		if !filter.TurnsColours() || picture == nil {
			return picture
		}
		return game.TurnPicture(filter, picture)
	}
	icon := func(name string) *image.NRGBA { return app.sprites.Picture(name) }
	app.levelCards = app.levelCards[:0]
	for index := range app.levels {
		number := index + 1
		style := game.PolaroidStyle{Width: 84, Height: 72, Label: strconv.Itoa(number), LabelSize: 16}
		if number > app.reached {
			style = game.PolaroidStyle{Width: 84, Height: 72, Locked: true, Icon: icon("ui-lock"), IconScale: 1}
		} else {
			// the original turns each opened polaroid a little, by chance: here by the level, so it
			// stays the same each time
			style.Degrees = float64(int(game.Hash(number, 17)*21) - 10)
		}
		app.levelCards = append(app.levelCards, turn(app.fonts.Polaroid(style)))
	}
	app.menuCards = app.menuCards[:0]
	for _, entry := range menuEntries {
		app.menuCards = append(app.menuCards, turn(app.fonts.Polaroid(game.PolaroidStyle{Width: 74, Height: 76,
			Icon: icon(entry.icon), IconScale: 2, Label: upper(entry.name), LabelSize: 8})))
	}
	app.filterCards = app.filterCards[:0]
	for _, entry := range game.Filters {
		app.filterCards = append(app.filterCards, turn(app.fonts.Polaroid(game.PolaroidStyle{Width: 74, Height: 64,
			Icon: icon(entry.Icon), IconScale: 2, Label: upper(entry.Name), LabelSize: 8})))
	}
	app.bigPlayer = app.bigPlayer[:0]
	for index := 0; index < 3; index++ {
		app.bigPlayer = append(app.bigPlayer, game.Doubled(app.sprites.Frame("characters-player", index)))
	}
	app.exitIcon = app.sprites.Frame("ui-option_exit", 0)
	app.lock = app.sprites.Frame("ui-lock", 0)
}

// setFilter chooses a camera filter, as the graphic options do.
func (app *app) setFilter(filter game.Filter) {
	app.sprites.SetFilter(filter)
	app.makePictures()
}

// openLevel shows the loading screen, then the level.
func (app *app) openLevel(number int) {
	if number < 1 || number > len(app.levels) {
		return
	}
	app.state, app.loadingLevel, app.loadingStart = stateLoading, number, app.clock
	app.world = nil
	app.audio.follow(nil)
}

func (app *app) startLevel(number int) {
	app.world = game.NewWorld(&app.levels[number-1])
	app.camera.Jump()
	if number > app.reached {
		app.reached = number
		app.makePictures() // a polaroid unlocked
	}
	app.state, app.menu, app.menuFocused, app.focus = stateLevel, menuMain, false, 0
}

func (app *app) openSelector() {
	app.state, app.world = stateSelector, nil
	app.selected = 0 // the original's selection starts on the first level
	app.audio.follow(nil)
}

// step moves the game on a frame: the buttons pressed, the cross held, a touch let go.
func (app *app) step(reading ctr.Reading, pressed ctr.Buttons, seconds float64) {
	app.clock += seconds
	if x, y, tapped := app.touch.update(reading); tapped {
		app.tap(x, y)
	}
	switch app.state {
	case stateTitle:
		if pressed&(ctr.ButtonStart|ctr.ButtonA) != 0 {
			app.openSelector()
		}
	case stateSelector:
		app.stepSelector(pressed)
	case stateLoading:
		if app.clock-app.loadingStart >= loadingTime || pressed&ctr.ButtonA != 0 {
			app.startLevel(app.loadingLevel)
		}
	case stateLevel:
		app.stepLevel(reading, pressed, seconds)
	}
}

func (app *app) stepSelector(pressed ctr.Buttons) {
	switch {
	case pressed&ctr.ButtonRight != 0:
		app.selected = min(app.selected+1, len(app.levels)-1)
	case pressed&ctr.ButtonLeft != 0:
		app.selected = max(app.selected-1, 0)
	case pressed&ctr.ButtonA != 0:
		if app.selected+1 <= app.reached {
			app.openLevel(app.selected + 1)
		} else {
			app.audio.play([]game.Event{game.EventBump})
		}
	case pressed&ctr.ButtonB != 0:
		app.state = stateTitle
	}
}

func (app *app) stepLevel(reading ctr.Reading, pressed ctr.Buttons, seconds float64) {
	world := app.world
	if pressed&ctr.ButtonStart != 0 {
		// START opens the game menu, as in the original, and closes it
		if app.menuFocused {
			app.closeMenu()
		} else {
			app.menuFocused, app.menu, app.focus = true, menuMain, 0
		}
	}
	if app.menuFocused {
		app.stepMenu(pressed)
	} else {
		world.Update(seconds)
		if pressed&ctr.ButtonSelect != 0 {
			world.Reset()
		}
		if direction, held := heldDirection(reading); held {
			world.Move(direction)
		}
	}
	app.audio.play(world.Events())
	app.audio.follow(world)
	if world.Complete && !world.Busy() {
		if world.Level.Number < len(app.levels) {
			app.openLevel(world.Level.Number + 1)
		} else {
			app.state, app.world = stateTitle, nil
			app.audio.follow(nil)
		}
		return
	}
	app.camera.Follow(world, ctr.ScreenWidth, ctr.ScreenHeight, seconds)
}

func (app *app) closeMenu() {
	app.menuFocused, app.menu, app.focus = false, menuMain, 0
}

// menuSize is how many entries the menu in use has, its exit counted.
func (app *app) menuSize() int {
	switch app.menu {
	case menuMain:
		return len(menuEntries) + 1
	case menuGraphics:
		return len(game.Filters) + 1
	case menuSound:
		return 3
	}
	return 1
}

// stepMenu moves through the bottom screen's menu with the cross, A choosing and B going back.
func (app *app) stepMenu(pressed ctr.Buttons) {
	size := app.menuSize()
	switch {
	case pressed&(ctr.ButtonRight) != 0 && app.menu != menuSound:
		app.focus = (app.focus + 1) % size
	case pressed&(ctr.ButtonLeft) != 0 && app.menu != menuSound:
		app.focus = (app.focus + size - 1) % size
	case pressed&ctr.ButtonDown != 0:
		if app.menu == menuSound {
			app.focus = min(app.focus+1, size-1)
		} else {
			app.focus = size - 1 // down to the exit
		}
	case pressed&ctr.ButtonUp != 0:
		if app.menu == menuSound {
			app.focus = max(app.focus-1, 0)
		} else {
			app.focus = 0
		}
	case pressed&ctr.ButtonA != 0:
		app.choose(app.focus)
	case pressed&ctr.ButtonB != 0:
		app.back()
	}
	if app.menu == menuSound && app.focus < 2 {
		change := 0.0
		if pressed&ctr.ButtonRight != 0 {
			change = 0.1
		}
		if pressed&ctr.ButtonLeft != 0 {
			change = -0.1
		}
		if change != 0 {
			app.setVolume(app.focus, min(max(volumeOf(app, app.focus)+change, 0), 1))
		}
	}
}

func volumeOf(app *app, slider int) float64 {
	if slider == 0 {
		return app.music
	}
	return app.effects
}

func (app *app) setVolume(slider int, volume float64) {
	volume = float64(int(volume*100+0.5)) / 100
	if slider == 0 {
		app.music = volume
	} else {
		app.effects = volume
	}
	app.audio.setVolumes(app.music, app.effects)
}

// choose does what a menu's entry does.
func (app *app) choose(entry int) {
	size := app.menuSize()
	if entry == size-1 {
		app.back()
		return
	}
	switch app.menu {
	case menuMain:
		switch entry {
		case 0:
			app.openSelector()
		case 1:
			app.menu, app.focus = menuControls, 0
		case 2:
			app.menu, app.focus = menuGraphics, 0
		case 3:
			app.menu, app.focus = menuSound, 0
		}
		app.menuFocused = app.state == stateLevel
	case menuGraphics:
		app.setFilter(game.Filters[entry].Filter)
	}
}

// back leaves an option's screen for the game menu, and the game menu for the level.
func (app *app) back() {
	if app.menu != menuMain {
		app.menu, app.focus = menuMain, 0
		return
	}
	app.closeMenu()
}

// tap acts on a touch let go on the bottom screen.
func (app *app) tap(x, y float32) {
	for _, region := range app.regions {
		area := region.area
		if x >= area.X && x < area.X+area.Width && y >= area.Y && y < area.Y+area.Height {
			region.action()
			return
		}
	}
}

// touchState follows the finger: a tap is a touch let go, where it last was.
type touchState struct {
	touching bool
	x, y     float32
}

func (touch *touchState) update(reading ctr.Reading) (x, y float32, tapped bool) {
	if reading.Held&ctr.ButtonTouch != 0 {
		touch.touching, touch.x, touch.y = true, float32(reading.TouchX), float32(reading.TouchY)
		return 0, 0, false
	}
	if touch.touching {
		touch.touching = false
		return touch.x, touch.y, true
	}
	return 0, 0, false
}

// walkThreshold is how far the circle pad leans before it walks.
const walkThreshold = 0.5

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

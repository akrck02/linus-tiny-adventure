//go:build nintendo3ds

// Linu's tiny adventure on the Nintendo 3DS, as the original: its title, its level selection of
// polaroids, its loading screen, its levels pixel for pixel, its game menu and options (controls,
// camera filters, sound), with its particles, sounds and music. The top screen is the game; the
// bottom one, touched or moved through with the cross, holds the polaroids. The cross or the circle
// pad walk, SELECT starts the level again, START opens the game menu, the HOME button leaves.
//
//	tinygo build -target=nintendo3ds -o linus.3dsx ./cmd/linus
package main

import (
	"runtime"
	"time"

	"github.com/akrck02/go-renderer/ctr"
	"github.com/akrck02/go-renderer/draw2d"
	"github.com/akrck02/go-renderer/input"
	"github.com/akrck02/go-renderer/opengl/overlay"

	"github.com/akrck02/linus-tiny-adventure/3ds/assets"
	"github.com/akrck02/linus-tiny-adventure/3ds/game"
)

func main() {
	font, err := assets.Files.ReadFile("early-gameboy.ttf")
	if err != nil {
		println(err.Error())
		return
	}
	overlay.UseFonts(overlay.Fonts{Regular: font, Medium: font, Bold: font, Mono: font})
	top, err := ctr.Open()
	if err != nil {
		println(err.Error())
		return
	}
	defer top.Close()
	bottom, err := ctr.OpenBottomScreen()
	if err != nil {
		println(err.Error())
		return
	}
	flat, err := ctr.NewFlatRenderer(bottom)
	if err != nil {
		println(err.Error())
		return
	}
	application, err := newApp()
	if err != nil {
		println(err.Error())
		return
	}

	var controls ctr.Controls
	state := input.NewState() // kept so the controls tell presses from holds
	var topList, bottomList draw2d.List
	last := time.Now()
	for ctr.Running() {
		reading := ctr.ReadControls()
		pressed := controls.Pressed(reading)
		controls.Apply(state, reading)
		state.EndFrame()
		now := time.Now()
		seconds := min(now.Sub(last).Seconds(), 0.1)
		last = now

		application.step(reading, pressed, seconds)
		topList.Reset()
		application.drawTop(&topList)
		bottomList.Reset()
		application.drawBottom(&bottomList)

		ctr.BeginFrame()
		flat.PaintOn(top, &topList, game.Background)
		flat.Paint(&bottomList, game.Background)
		ctr.EndFrame()
		// TinyGo's scheduler on the console is cooperative, and waiting for the screen happens in
		// C: yield, so the sound's goroutine fills its buffers
		runtime.Gosched()
	}
}

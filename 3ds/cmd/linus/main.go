//go:build nintendo3ds

// Linu's tiny adventure on the Nintendo 3DS. The top screen is the level, pixel for pixel as the
// original draws it; the bottom one, touched, holds the level's name, the steps, the reset and the
// levels to choose. The cross or the circle pad walk, SELECT starts the level again, START goes
// back to the HOME menu.
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
	ui "github.com/akrck02/go-ui"

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
	context, err := ui.New()
	if err != nil {
		println(err.Error())
		return
	}
	context.Theme = theme()
	play, err := newPlay()
	if err != nil {
		println(err.Error())
		return
	}

	state := input.NewState()
	state.FramebufferWidth, state.FramebufferHeight = ctr.BottomWidth, ctr.BottomHeight
	state.WindowWidth, state.WindowHeight = ctr.BottomWidth, ctr.BottomHeight
	var controls ctr.Controls
	var topList draw2d.List
	last := time.Now()
	for ctr.Running() {
		reading := ctr.ReadControls()
		pressed := controls.Pressed(reading)
		if pressed&ctr.ButtonStart != 0 {
			return
		}
		controls.Apply(state, reading)
		now := time.Now()
		seconds := min(now.Sub(last).Seconds(), 0.1)
		last = now

		play.step(reading, pressed, seconds)
		topList.Reset()
		play.drawTop(&topList)

		ctr.BeginFrame()
		flat.PaintOn(top, &topList, game.Background)
		context.Begin(state)
		play.buildBottom(context)
		context.End()
		flat.Paint(context.Drawing(), context.Theme.Background)
		ctr.EndFrame()
		state.EndFrame()
		// TinyGo's scheduler on the console is cooperative, and waiting for the screen happens in
		// C: yield, so the sound's goroutine fills its buffers
		runtime.Gosched()
	}
}

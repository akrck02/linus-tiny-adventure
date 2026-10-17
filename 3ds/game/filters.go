package game

import (
	"image"
	"image/color"
	"math"

	"github.com/akrck02/go-renderer/draw2d"
)

// The camera filters of the original's graphic options. Godot draws each as a shader over the
// screen; the console draws the same looks its own way: a layer over everything (grain, scan lines,
// the cozy tint) or every colour and picture turned before it is drawn (grayscale, midnight).

// Filter is a camera filter.
type Filter int

const (
	FilterNone Filter = iota
	FilterScanlines
	FilterGrainy
	FilterGrayscale
	FilterMidnight
	FilterCozy
	filterCount
)

// DefaultFilter is the original's: grain.
const DefaultFilter = FilterGrainy

// Filters are the filters in the order the graphic options show them, with their names and icons.
var Filters = []struct {
	Filter Filter
	Name   string
	Icon   string
}{
	{FilterNone, "None", "ui-graphics-camera_none"},
	{FilterScanlines, "Scanlines", "ui-graphics-camera_scanlines"},
	{FilterGrainy, "Grainy", "ui-graphics-camera_grainy"},
	{FilterGrayscale, "Grayscale", "ui-graphics-camera_grayscale"},
	{FilterMidnight, "Midnight", "ui-graphics-camera_midnight"},
	{FilterCozy, "Cozy", "ui-graphics-camera_cozy"},
}

// Colour turns a colour as the filter's shader would: grayscale averages it (the shader divides
// the sum by 2.8, a little brighter than a third), midnight keeps its blue over a dark red and
// green. The other filters draw over the picture and leave colours alone.
func (filter Filter) Colour(colour draw2d.Color) draw2d.Color {
	switch filter {
	case FilterGrayscale:
		brightness := min((colour[0]+colour[1]+colour[2])/2.8, 1)
		return draw2d.Color{brightness, brightness, brightness, colour[3]}
	case FilterMidnight:
		return draw2d.Color{0.1, 0.1, colour[2], colour[3]}
	}
	return colour
}

// TurnsColours tells whether the filter changes colours (and so pictures) rather than drawing a
// layer over them.
func (filter Filter) TurnsColours() bool {
	return filter == FilterGrayscale || filter == FilterMidnight
}

// turnPicture is a picture with every pixel turned by the filter.
func (filter Filter) turnPicture(picture *image.NRGBA) *image.NRGBA {
	turned := image.NewNRGBA(picture.Rect)
	for index := 0; index+3 < len(picture.Pix); index += 4 {
		colour := filter.Colour(draw2d.Color{float32(picture.Pix[index]) / 255, float32(picture.Pix[index+1]) / 255,
			float32(picture.Pix[index+2]) / 255, float32(picture.Pix[index+3]) / 255})
		for channel := 0; channel < 4; channel++ {
			turned.Pix[index+channel] = uint8(colour[channel]*255 + 0.5)
		}
	}
	return turned
}

// Overlay draws the filter's layer over a screen of a size: the grain's speckles (the shader's
// noise at its 0.025 of opacity), the scan lines (a dark line every other row at a tenth), or the
// cozy blue at a tenth. grain is a picture of speckles the size of the largest screen.
func (filter Filter) Overlay(list *draw2d.List, width, height float32, grain *draw2d.Image) {
	switch filter {
	case FilterGrainy:
		if grain != nil {
			list.Image(grain, draw2d.Rect{Width: 400, Height: 240}, 0, draw2d.Color{1, 1, 1, 1})
		}
	case FilterScanlines:
		for y := float32(1); y < height; y += 3 {
			list.Rectangle(draw2d.Rect{Y: y, Width: width, Height: 1}, 0, draw2d.Color{0, 0, 0, 0.1}, draw2d.Color{}, 0)
		}
	case FilterCozy:
		list.Rectangle(draw2d.Rect{Width: width, Height: height}, 0, draw2d.Color{0, 0, 1, 0.1}, draw2d.Color{}, 0)
	}
}

// NewGrain makes the grain's speckles as the original's grainy shader draws them: each pixel's
// colour moved by sin(y + x²) in red, sin(x + y²) in green and tan(x + y) in blue, kept between
// black and white, and laid over the screen at 0.025 of opacity. The shader moves the colour under
// each pixel; a picture laid over everything cannot know it, so the grain is made over the game's
// background, which is most of the screen.
func NewGrain(width, height int) *draw2d.Image {
	picture := image.NewNRGBA(image.Rect(0, 0, width, height))
	channel := func(base, change float32) uint8 {
		return uint8(min(max(base+change, 0), 1)*255 + 0.5)
	}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			fx, fy := float32(x), float32(y)
			picture.SetNRGBA(x, y, color.NRGBA{
				R: channel(Background[0], float32(math.Sin(float64(fy+fx*fx)))),
				G: channel(Background[1], float32(math.Sin(float64(fx+fy*fy)))),
				B: channel(Background[2], float32(math.Tan(float64(fy+fx)))),
				A: grainOpacity,
			})
		}
	}
	return draw2d.NewImage(picture)
}

// grainOpacity is the shader's 0.025, in a byte.
const grainOpacity = 6

// TurnPicture is a picture as a filter turns its colours.
func TurnPicture(filter Filter, picture *draw2d.Image) *draw2d.Image {
	return draw2d.NewImage(filter.turnPicture(picture.Pixels))
}

// Doubled is a picture twice as large, pixel for pixel.
func Doubled(picture *draw2d.Image) *draw2d.Image {
	if picture == nil {
		return nil
	}
	return draw2d.NewImage(scale(picture.Pixels, 2))
}

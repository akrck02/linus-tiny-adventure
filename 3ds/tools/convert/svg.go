package main

import (
	"encoding/xml"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"os"
	"strconv"
	"strings"
)

// The game's sprites are pixel art drawn as SVG rectangles on whole pixels, in strips of frames,
// some groups cut by a clip path (one frame each). That is all rasteriseSVG reads: rectangles,
// groups, clip paths of rectangles, a fill colour and its opacity.

type svgElement struct {
	XMLName  xml.Name
	Attrs    []xml.Attr   `xml:",any,attr"`
	Children []svgElement `xml:",any"`
}

func (element *svgElement) attribute(name string) string {
	for _, attribute := range element.Attrs {
		if attribute.Name.Local == name {
			return attribute.Value
		}
	}
	return ""
}

func (element *svgElement) number(name string) int {
	value, _ := strconv.ParseFloat(element.attribute(name), 64)
	return int(value + 0.5)
}

// rasteriseSVG draws an SVG of rectangles into an image of its own size.
func rasteriseSVG(path string) (*image.NRGBA, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var root svgElement
	if err := xml.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	picture := image.NewNRGBA(image.Rect(0, 0, root.number("width"), root.number("height")))
	clips := map[string]image.Rectangle{}
	collectClips(&root, clips)
	paint(&root, picture, picture.Bounds(), clips)
	return picture, nil
}

// collectClips finds every clipPath and the rectangle it keeps.
func collectClips(element *svgElement, clips map[string]image.Rectangle) {
	if element.XMLName.Local == "clipPath" {
		for _, child := range element.Children {
			if child.XMLName.Local == "rect" {
				clips[element.attribute("id")] = rectangleOf(&child)
			}
		}
		return
	}
	for index := range element.Children {
		collectClips(&element.Children[index], clips)
	}
}

// rectangleOf is a rect's area, after its transform (a matrix or a translation: the drawings
// mirror some frames, turning nothing).
func rectangleOf(element *svgElement) image.Rectangle {
	x, y := element.number("x"), element.number("y")
	area := [4]float64{float64(x), float64(y), float64(x + element.number("width")), float64(y + element.number("height"))}
	matrix := transformOf(element.attribute("transform"))
	left, top := matrix[0]*area[0]+matrix[2]*area[1]+matrix[4], matrix[1]*area[0]+matrix[3]*area[1]+matrix[5]
	right, bottom := matrix[0]*area[2]+matrix[2]*area[3]+matrix[4], matrix[1]*area[2]+matrix[3]*area[3]+matrix[5]
	return image.Rect(int(math.Round(left)), int(math.Round(top)), int(math.Round(right)), int(math.Round(bottom))).Canon()
}

// transformOf reads matrix(a b c d e f) or translate(x y); none is the identity.
func transformOf(transform string) [6]float64 {
	identity := [6]float64{1, 0, 0, 1, 0, 0}
	name, arguments, found := strings.Cut(strings.TrimSpace(transform), "(")
	if !found {
		return identity
	}
	var values []float64
	for _, field := range strings.FieldsFunc(strings.TrimSuffix(arguments, ")"), func(r rune) bool { return r == ' ' || r == ',' }) {
		value, err := strconv.ParseFloat(field, 64)
		if err != nil {
			return identity
		}
		values = append(values, value)
	}
	switch {
	case name == "matrix" && len(values) == 6:
		return [6]float64(values)
	case name == "translate" && len(values) >= 1:
		translation := identity
		translation[4] = values[0]
		if len(values) > 1 {
			translation[5] = values[1]
		}
		return translation
	}
	return identity
}

// paint draws an element's rectangles, cut to the clip in force, and its groups' within theirs.
func paint(element *svgElement, picture *image.NRGBA, clip image.Rectangle, clips map[string]image.Rectangle) {
	switch element.XMLName.Local {
	case "defs", "clipPath":
		return
	case "rect":
		fill := colourOf(element.attribute("fill"), element.attribute("fill-opacity"))
		if fill.A == 0 {
			return
		}
		area := rectangleOf(element).Intersect(clip)
		draw.Draw(picture, area, &image.Uniform{fill}, image.Point{}, draw.Over)
		return
	}
	if reference := element.attribute("clip-path"); reference != "" {
		id := strings.TrimSuffix(strings.TrimPrefix(reference, "url(#"), ")")
		if area, found := clips[id]; found {
			clip = clip.Intersect(area)
		}
	}
	for index := range element.Children {
		paint(&element.Children[index], picture, clip, clips)
	}
}

// colourOf reads a fill: #rrggbb, a few names, and an opacity from 0 to 1.
func colourOf(fill, opacity string) color.NRGBA {
	alpha := 1.0
	if opacity != "" {
		alpha, _ = strconv.ParseFloat(opacity, 64)
	}
	named := map[string]string{"white": "#ffffff", "black": "#000000"}
	if hex, found := named[fill]; found {
		fill = hex
	}
	if len(fill) != 7 || fill[0] != '#' {
		return color.NRGBA{}
	}
	value, err := strconv.ParseUint(fill[1:], 16, 32)
	if err != nil {
		return color.NRGBA{}
	}
	return color.NRGBA{R: uint8(value >> 16), G: uint8(value >> 8), B: uint8(value), A: uint8(alpha*255 + 0.5)}
}

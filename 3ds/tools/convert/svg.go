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
func rasteriseSVG(path string) (*image.NRGBA, error) { return rasteriseSVGScaled(path, 1) }

// rasteriseSVGScaled draws it at a fraction of its size: the interface's large icons are pixel art
// on a grid of ten, drawn at a tenth.
func rasteriseSVGScaled(path string, scale float64) (*image.NRGBA, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var root svgElement
	if err := xml.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	width, height := int(math.Round(float64(root.number("width"))*scale)), int(math.Round(float64(root.number("height"))*scale))
	picture := image.NewNRGBA(image.Rect(0, 0, width, height))
	clips := map[string]image.Rectangle{}
	collectClips(&root, clips)
	paint(&root, picture, picture.Bounds(), clips, scale)
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
func paint(element *svgElement, picture *image.NRGBA, clip image.Rectangle, clips map[string]image.Rectangle, scale float64) {
	switch element.XMLName.Local {
	case "defs", "clipPath":
		return
	case "rect":
		fill := colourOf(element.attribute("fill"), element.attribute("fill-opacity"))
		if fill.A == 0 {
			return
		}
		area := scaled(rectangleOf(element), scale).Intersect(clip)
		draw.Draw(picture, area, &image.Uniform{fill}, image.Point{}, draw.Over)
		return
	case "path":
		fill := colourOf(element.attribute("fill"), element.attribute("fill-opacity"))
		if fill.A == 0 {
			return
		}
		for _, area := range pathRectangles(element.attribute("d")) {
			draw.Draw(picture, scaled(area, scale).Intersect(clip), &image.Uniform{fill}, image.Point{}, draw.Over)
		}
		return
	}
	if reference := element.attribute("clip-path"); reference != "" {
		id := strings.TrimSuffix(strings.TrimPrefix(reference, "url(#"), ")")
		if area, found := clips[id]; found {
			clip = clip.Intersect(area)
		}
	}
	for index := range element.Children {
		paint(&element.Children[index], picture, clip, clips, scale)
	}
}

func scaled(area image.Rectangle, scale float64) image.Rectangle {
	if scale == 1 {
		return area
	}
	at := func(value int) int { return int(math.Round(float64(value) * scale)) }
	return image.Rect(at(area.Min.X), at(area.Min.Y), at(area.Max.X), at(area.Max.Y))
}

// pathRectangles reads a path of axis-aligned subpaths (M, H, V, L and Z, absolute, as pixel art
// drawn in an editor comes out) as the rectangles that bound each one.
func pathRectangles(data string) []image.Rectangle {
	var areas []image.Rectangle
	var x, y float64
	var bounds [4]float64
	open := false
	finish := func() {
		if open {
			areas = append(areas, image.Rect(int(math.Round(bounds[0])), int(math.Round(bounds[1])), int(math.Round(bounds[2])), int(math.Round(bounds[3]))))
		}
		open = false
	}
	add := func() {
		if !open {
			bounds, open = [4]float64{x, y, x, y}, true
			return
		}
		bounds = [4]float64{min(bounds[0], x), min(bounds[1], y), max(bounds[2], x), max(bounds[3], y)}
	}
	fields := strings.FieldsFunc(data, func(r rune) bool { return r == ' ' || r == ',' })
	var tokens []string
	for _, field := range fields {
		// commands stick to their numbers: "M80", "40H90"
		start := 0
		for index, character := range field {
			if strings.ContainsRune("MHVLZmhvlz", character) {
				if index > start {
					tokens = append(tokens, field[start:index])
				}
				tokens = append(tokens, string(character))
				start = index + 1
			}
		}
		if start < len(field) {
			tokens = append(tokens, field[start:])
		}
	}
	number := func(index int) float64 {
		value, _ := strconv.ParseFloat(tokens[index], 64)
		return value
	}
	for index := 0; index < len(tokens); index++ {
		switch tokens[index] {
		case "M":
			finish()
			x, y = number(index+1), number(index+2)
			index += 2
			add()
		case "L":
			x, y = number(index+1), number(index+2)
			index += 2
			add()
		case "H":
			x = number(index + 1)
			index++
			add()
		case "V":
			y = number(index + 1)
			index++
			add()
		case "Z", "z":
			finish()
		}
	}
	finish()
	return areas
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

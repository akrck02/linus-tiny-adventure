package game

import (
	"image"
	"image/color"
	"image/draw"
	"math"

	"github.com/akrck02/go-renderer/draw2d"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// The original's menus are polaroids: a white card, a photo inside it a little darker than the
// background (or dark with a padlock, for a level not reached), a wider white band under the
// photo, and the chosen one tilted or raised. The console cannot turn a picture as it draws it,
// so each polaroid is made once as a picture, turned already, pixel for pixel.

var (
	cardWhite   = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	photoMint   = color.NRGBA{R: 172, G: 202, B: 189, A: 255}
	photoLocked = color.NRGBA{R: 51, G: 51, B: 51, A: 255}
)

// PolaroidStyle is what a polaroid shows.
type PolaroidStyle struct {
	Width, Height int          // the card's
	Locked        bool         // dark, with the padlock
	Icon          *image.NRGBA // drawn in the photo, above the label
	IconScale     int
	Label         string // under the icon, or alone and large (a level's number)
	LabelSize     float64
	Degrees       float64 // turned clockwise
}

// Fonts draws text into pictures, in the game's font.
type Fonts struct {
	font  *opentype.Font
	faces map[float64]font.Face
}

// NewFonts reads the game's font for drawing into pictures.
func NewFonts(data []byte) (*Fonts, error) {
	parsed, err := opentype.Parse(data)
	if err != nil {
		return nil, err
	}
	return &Fonts{font: parsed, faces: map[float64]font.Face{}}, nil
}

func (fonts *Fonts) face(size float64) font.Face {
	if face, found := fonts.faces[size]; found {
		return face
	}
	face, err := opentype.NewFace(fonts.font, &opentype.FaceOptions{Size: size, DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		return nil
	}
	fonts.faces[size] = face
	return face
}

// write draws a text centred on a point's column, its baseline at y, in a colour.
func (fonts *Fonts) write(picture *image.NRGBA, text string, centreX, baseline int, size float64, colour color.NRGBA) {
	face := fonts.face(size)
	if face == nil {
		return
	}
	width := font.MeasureString(face, text).Round()
	drawer := font.Drawer{Dst: picture, Src: image.NewUniform(colour), Face: face, Dot: fixed.P(centreX-width/2, baseline)}
	drawer.DrawString(text)
}

// Polaroid makes a polaroid's picture.
func (fonts *Fonts) Polaroid(style PolaroidStyle) *draw2d.Image {
	card := image.NewNRGBA(image.Rect(0, 0, style.Width, style.Height))
	draw.Draw(card, card.Rect, image.NewUniform(cardWhite), image.Point{}, draw.Src)
	border, band := max(style.Width/20, 2), max(style.Height/6, 6)
	photo := image.Rect(border, border, style.Width-border, style.Height-band)
	fill := photoMint
	if style.Locked {
		fill = photoLocked
	}
	draw.Draw(card, photo, image.NewUniform(fill), image.Point{}, draw.Src)
	centreX, centreY := (photo.Min.X+photo.Max.X)/2, (photo.Min.Y+photo.Max.Y)/2
	icon := style.Icon
	if icon != nil {
		scale := max(style.IconScale, 1)
		width, height := icon.Rect.Dx()*scale, icon.Rect.Dy()*scale
		top := centreY - height/2
		if style.Label != "" {
			top = centreY - height/2 - int(style.LabelSize)/2 - 2
		}
		area := image.Rect(centreX-width/2, top, centreX-width/2+width, top+height)
		draw.Draw(card, area, scaleNearest(icon, scale), image.Point{}, draw.Over)
		if style.Label != "" {
			fonts.write(card, style.Label, centreX, area.Max.Y+4+int(style.LabelSize), style.LabelSize, cardWhite)
		}
	} else if style.Label != "" {
		fonts.write(card, style.Label, centreX, centreY+int(style.LabelSize)/2, style.LabelSize, cardWhite)
	}
	return draw2d.NewImage(rotateNearest(card, style.Degrees))
}

func scaleNearest(picture *image.NRGBA, times int) *image.NRGBA {
	if times == 1 {
		return picture
	}
	return scale(picture, times)
}

// rotateNearest turns a picture clockwise by some degrees, pixel for pixel: each pixel of the
// result takes the source's nearest one, as a pixel-art game turns its sprites.
func rotateNearest(picture *image.NRGBA, degrees float64) *image.NRGBA {
	if degrees == 0 {
		return picture
	}
	angle := degrees * math.Pi / 180
	cosine, sine := math.Cos(angle), math.Sin(angle)
	width, height := float64(picture.Rect.Dx()), float64(picture.Rect.Dy())
	turnedWidth := int(math.Ceil(math.Abs(width*cosine) + math.Abs(height*sine)))
	turnedHeight := int(math.Ceil(math.Abs(width*sine) + math.Abs(height*cosine)))
	turned := image.NewNRGBA(image.Rect(0, 0, turnedWidth, turnedHeight))
	for y := 0; y < turnedHeight; y++ {
		for x := 0; x < turnedWidth; x++ {
			// back from the result's pixel to the source's, around both centres
			dx, dy := float64(x)+0.5-float64(turnedWidth)/2, float64(y)+0.5-float64(turnedHeight)/2
			sourceX := dx*cosine + dy*sine + width/2
			sourceY := -dx*sine + dy*cosine + height/2
			if sourceX < 0 || sourceY < 0 || sourceX >= width || sourceY >= height {
				continue
			}
			turned.SetNRGBA(x, y, picture.NRGBAAt(int(sourceX), int(sourceY)))
		}
	}
	return turned
}

// Picture is a sprite's first frame as a picture, for making polaroids of icons.
func (sprites *Sprites) Picture(name string) *image.NRGBA {
	frames := sprites.base[name]
	if len(frames) == 0 {
		return nil
	}
	return frames[0].Pixels
}

// Width is how wide a text is in the game's font at a size, in pixels.
func (fonts *Fonts) Width(text string, size float64) float32 {
	face := fonts.face(size)
	if face == nil {
		return 0
	}
	return float32(font.MeasureString(face, text).Round())
}

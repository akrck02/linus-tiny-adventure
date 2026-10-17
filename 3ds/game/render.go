package game

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"io/fs"
	"math"
	"strings"

	"github.com/akrck02/go-renderer/draw2d"
)

// The colours of the original: the background (the project's clear colour), the default button,
// a used button and the laser.
var (
	Background   = draw2d.Color{0.709804, 0.823529, 0.772549, 1}
	buttonColour = draw2d.Hex("#6cbcf6")
	usedColour   = draw2d.Hex("#818181")
	laserColour  = draw2d.Color{0.713726, 0.133333, 0.109804, 1}
	labelColour  = draw2d.Hex("#ffffff")
	labelShadow  = draw2d.Color{0, 0, 0, 0.18}

	// the tints the item scenes give their sprites (modulate)
	heartTint     = draw2d.Color{1, 0.286275, 0.286275, 1}
	keyTint       = draw2d.Color{0.882353, 0.690196, 0, 1}
	doorTint      = draw2d.Color{1, 0.6, 0.266667, 1}
	lockTint      = draw2d.Color{0.964706, 0.803922, 0, 1}
	teleportTint  = draw2d.Color{0.803922, 0.647059, 1, 1}
	frameDuration = 0.75 // seconds a frame of an idle animation shows
)

// Sprites are the game's pictures, each strip cut into its frames, as made and as the camera
// filter in use turns them (made the first time a filter is chosen).
type Sprites struct {
	base     map[string][]*draw2d.Image
	frames   map[string][]*draw2d.Image // the filter's
	filter   Filter
	variants map[Filter]map[string][]*draw2d.Image
}

// SetFilter chooses the camera filter the sprites are drawn through.
func (sprites *Sprites) SetFilter(filter Filter) {
	sprites.filter = filter
	if !filter.TurnsColours() {
		sprites.frames = sprites.base
		return
	}
	if variant, found := sprites.variants[filter]; found {
		sprites.frames = variant
		return
	}
	variant := map[string][]*draw2d.Image{}
	for name, frames := range sprites.base {
		for _, frame := range frames {
			variant[name] = append(variant[name], draw2d.NewImage(filter.turnPicture(frame.Pixels)))
		}
	}
	sprites.variants[filter] = variant
	sprites.frames = variant
}

// Filter is the camera filter the sprites are drawn through.
func (sprites *Sprites) Filter() Filter { return sprites.filter }

// Frame is a sprite's picture, as the filter draws it.
func (sprites *Sprites) Frame(name string, index int) *draw2d.Image {
	return sprites.frame(name, index)
}

// LoadSprites reads the PNG strips the converter wrote. Lasers get their frames turned for each
// way they fire, and the heart a copy three times larger, pixel for pixel: the screen draws
// pictures as they are, so the turning and the scaling are done here.
func LoadSprites(files fs.FS) (*Sprites, error) {
	sprites := &Sprites{base: map[string][]*draw2d.Image{}, variants: map[Filter]map[string][]*draw2d.Image{}}
	sprites.frames = sprites.base
	entries, err := fs.Glob(files, "sprites/*.png")
	if err != nil {
		return nil, err
	}
	for _, path := range entries {
		data, err := fs.ReadFile(files, path)
		if err != nil {
			return nil, err
		}
		picture, err := png.Decode(bytes.NewReader(data))
		if err != nil {
			return nil, err
		}
		name := strings.TrimSuffix(strings.TrimPrefix(path, "sprites/"), ".png")
		frames := cutFrames(picture)
		if strings.HasPrefix(name, "ui-") {
			frames = []*image.NRGBA{toNRGBA(picture)} // the interface's icons are single pictures
		}
		for _, frame := range frames {
			sprites.base[name] = append(sprites.base[name], draw2d.NewImage(frame))
		}
		switch name {
		case "laser":
			for turns, way := range []string{"right", "down", "left"} {
				for _, frame := range frames {
					sprites.base["laser-"+way] = append(sprites.base["laser-"+way], draw2d.NewImage(turn(frame, turns+1)))
				}
			}
		case "heart":
			for _, frame := range frames {
				sprites.base["heart-3"] = append(sprites.base["heart-3"], draw2d.NewImage(scale(frame, 3)))
			}
		case "characters-player":
			// the player's scene draws it at seven eighths, 14 pixels
			for index, frame := range frames {
				sprites.base[name][index] = draw2d.NewImage(resize(frame, 14, 14, false))
			}
		case "ui-lock", "ui-option_exit":
			// drawn five and three times larger than their pixels: back to one
			times := 5
			if name == "ui-option_exit" {
				times = 3
			}
			bounds := frames[0].Rect
			sprites.base[name][0] = draw2d.NewImage(resize(frames[0], bounds.Dx()/times, bounds.Dy()/times, false))
		case "characters-shadow":
			// a soft picture of 151 × 150 drawn at a tenth: 16 pixels, averaged
			sprites.base[name][0] = draw2d.NewImage(resize(frames[0], 16, 16, true))
		}
	}
	return sprites, nil
}

// cutFrames cuts a strip of square frames (48 × 16: three); other pictures stay whole.
func cutFrames(picture image.Image) []*image.NRGBA {
	bounds := picture.Bounds()
	side := bounds.Dy()
	count := 1
	if side > 0 && bounds.Dx()%side == 0 && bounds.Dx()/side <= 4 {
		count = bounds.Dx() / side
	}
	width := bounds.Dx() / count
	frames := make([]*image.NRGBA, count)
	for index := range frames {
		frame := image.NewNRGBA(image.Rect(0, 0, width, bounds.Dy()))
		for y := 0; y < bounds.Dy(); y++ {
			for x := 0; x < width; x++ {
				frame.Set(x, y, picture.At(bounds.Min.X+index*width+x, bounds.Min.Y+y))
			}
		}
		frames[index] = frame
	}
	return frames
}

func toNRGBA(picture image.Image) *image.NRGBA {
	bounds := picture.Bounds()
	result := image.NewNRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	for y := 0; y < bounds.Dy(); y++ {
		for x := 0; x < bounds.Dx(); x++ {
			result.Set(x, y, picture.At(bounds.Min.X+x, bounds.Min.Y+y))
		}
	}
	return result
}

// turn turns a square picture a quarter clockwise, times times.
func turn(picture *image.NRGBA, times int) *image.NRGBA {
	result := picture
	for ; times > 0; times-- {
		side := result.Rect.Dx()
		turned := image.NewNRGBA(image.Rect(0, 0, side, side))
		for y := 0; y < side; y++ {
			for x := 0; x < side; x++ {
				turned.Set(side-1-y, x, result.At(x, y))
			}
		}
		result = turned
	}
	return result
}

// resize makes a picture of another size: each pixel the nearest one's, or the average of those it
// covers (for a soft picture made much smaller).
func resize(picture *image.NRGBA, width, height int, average bool) *image.NRGBA {
	bounds := picture.Rect
	resized := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			fromX, toX := x*bounds.Dx()/width, max((x+1)*bounds.Dx()/width, x*bounds.Dx()/width+1)
			fromY, toY := y*bounds.Dy()/height, max((y+1)*bounds.Dy()/height, y*bounds.Dy()/height+1)
			if !average {
				resized.SetNRGBA(x, y, picture.NRGBAAt(fromX, fromY))
				continue
			}
			var red, green, blue, alpha, count int
			for sourceY := fromY; sourceY < toY; sourceY++ {
				for sourceX := fromX; sourceX < toX; sourceX++ {
					colour := picture.NRGBAAt(sourceX, sourceY)
					red, green, blue, alpha, count = red+int(colour.R), green+int(colour.G), blue+int(colour.B), alpha+int(colour.A), count+1
				}
			}
			resized.SetNRGBA(x, y, color.NRGBA{R: uint8(red / count), G: uint8(green / count), B: uint8(blue / count), A: uint8(alpha / count)})
		}
	}
	return resized
}

func scale(picture *image.NRGBA, times int) *image.NRGBA {
	bounds := picture.Rect
	scaled := image.NewNRGBA(image.Rect(0, 0, bounds.Dx()*times, bounds.Dy()*times))
	for y := 0; y < scaled.Rect.Dy(); y++ {
		for x := 0; x < scaled.Rect.Dx(); x++ {
			scaled.Set(x, y, picture.At(x/times, y/times))
		}
	}
	return scaled
}

// frame is a sprite's picture now: its idle animation's frame, or a fixed one.
func (sprites *Sprites) frame(name string, index int) *draw2d.Image {
	frames := sprites.frames[name]
	if len(frames) == 0 {
		return nil
	}
	return frames[index%len(frames)]
}

// Camera is what the top screen shows of a level: its top-left corner, in pixels.
type Camera struct {
	X, Y   float64
	placed bool
}

// Follow places the camera: the whole level in the middle when it fits the screen, otherwise on
// the player, eased and kept inside the level.
func (camera *Camera) Follow(world *World, width, height float64, seconds float64) {
	left, top, right, bottom := world.bounds()
	targetX, targetY := (left+right-width)/2, (top+bottom-height)/2
	if right-left > width || bottom-top > height {
		playerX, playerY := world.DrawnAt(world.Player)
		if right-left > width {
			targetX = min(max(playerX+TileSize/2-width/2, left), right-width)
		}
		if bottom-top > height {
			targetY = min(max(playerY+TileSize/2-height/2, top), bottom-height)
		}
	}
	if !camera.placed {
		camera.X, camera.Y, camera.placed = targetX, targetY, true
		return
	}
	ease := 1 - math.Exp(-seconds*6)
	camera.X += (targetX - camera.X) * ease
	camera.Y += (targetY - camera.Y) * ease
}

// Jump puts the camera where it should be at once, for a level just begun.
func (camera *Camera) Jump() { camera.placed = false }

// bounds are the level's walls' extent in pixels.
func (world *World) bounds() (left, top, right, bottom float64) {
	first := true
	add := func(x, y int) {
		pixelX, pixelY := float64(x*TileSize), float64(y*TileSize)
		if first {
			left, top, right, bottom, first = pixelX, pixelY, pixelX+TileSize, pixelY+TileSize, false
			return
		}
		left, top = min(left, pixelX), min(top, pixelY)
		right, bottom = max(right, pixelX+TileSize), max(bottom, pixelY+TileSize)
	}
	for _, tile := range world.Level.Tiles {
		add(tile.X, tile.Y)
	}
	for _, thing := range world.Level.Things {
		if thing.Kind == KindWall {
			add(thing.X, thing.Y)
		}
	}
	return left, top, right, bottom
}

// Draw draws the level as the camera sees it, on a screen of a size.
func (sprites *Sprites) Draw(list *draw2d.List, world *World, camera *Camera, width, height float64) {
	list.Rectangle(draw2d.Rect{Width: float32(width), Height: float32(height)}, 0, sprites.filter.Colour(Background), draw2d.Color{}, 0)
	shakeX, shakeY := world.Shake()
	originX, originY := math.Round(-camera.X+shakeX), math.Round(-camera.Y+shakeY)
	at := func(x, y float64) (float32, float32) {
		return float32(math.Round(x + originX)), float32(math.Round(y + originY))
	}
	animation := int(world.Clock / frameDuration)
	put := func(picture *draw2d.Image, x, y float64, tint draw2d.Color) {
		if picture == nil {
			return
		}
		screenX, screenY := at(x, y)
		pictureWidth, pictureHeight := picture.Size()
		if screenX > float32(width) || screenY > float32(height) || screenX+float32(pictureWidth) < 0 || screenY+float32(pictureHeight) < 0 {
			return
		}
		list.Image(picture, draw2d.Rect{X: screenX, Y: screenY, Width: float32(pictureWidth), Height: float32(pictureHeight)}, 0, sprites.filter.Colour(tint))
	}
	white := draw2d.Color{1, 1, 1, 1}
	for _, tile := range world.Level.Tiles {
		put(sprites.frame("wall", tile.Variant), float64(tile.X*TileSize), float64(tile.Y*TileSize), white)
	}
	for _, label := range world.Level.Labels {
		sprites.drawLabel(list, label, originX, originY)
	}
	// what lies on the floor first, then what stands, the player last
	for index, entity := range world.Entities {
		if entity.Gone {
			continue
		}
		x, y := world.DrawnAt(entity)
		switch entity.Kind {
		case KindTeleport:
			put(sprites.frame("teleport", animation), x, y, teleportTint)
			put(sprites.frame("teleport_frame", animation), x, y, white)
			screenX, screenY := at(x, y)
			sprites.drawParticles(list, teleportSparks, float64(screenX), float64(screenY), index, world.Clock)
		case KindButton:
			frame, tint := animation, buttonColour
			if entity.Colour != "" {
				tint = draw2d.Hex("#" + entity.Colour)
			}
			if entity.Disabled {
				frame, tint = 0, usedColour
			}
			put(sprites.frame("button_base", frame), x, y, white)
			clicker := frame
			if entity.Disabled {
				clicker = 2
			}
			put(sprites.frame("button_clicker", clicker), x, y, tint)
			put(sprites.frame("button_sign", frame), x, y, white)
		case KindHeart:
			if entity.Scale > 1 {
				put(sprites.frame("heart-3", animation), x, y, heartTint)
			} else {
				put(sprites.frame("heart", animation), x, y, heartTint)
			}
		case KindRadio:
			put(sprites.frame("radio", animation), x, y, white)
		case KindLaser:
			name := "laser"
			if entity.Direction != "up" && entity.Direction != "" {
				name = "laser-" + entity.Direction
			}
			put(sprites.frame(name, animation), x, y, white)
		}
	}
	for _, beam := range world.Beams() {
		sprites.drawBeam(list, beam, originX, originY)
	}
	for index, entity := range world.Entities {
		if entity.Gone {
			continue
		}
		x, y := world.DrawnAt(entity)
		switch entity.Kind {
		case KindBox:
			put(sprites.frame("box", animation), x, y, white)
		case KindKey:
			put(sprites.frame("key", animation), x, y, keyTint)
			screenX, screenY := at(x, y)
			sprites.drawParticles(list, keySparks, float64(screenX), float64(screenY), index, world.Clock)
		case KindDoor:
			put(sprites.frame("door", animation), x, y, doorTint)
			put(sprites.frame("door_lock", animation), x, y, lockTint)
		case KindWall:
			put(sprites.frame("wall", 0), x, y, white)
		}
	}
	if player := world.Player; player != nil {
		x, y := world.DrawnAt(player)
		if shadow := sprites.frame("characters-shadow", 0); shadow != nil {
			shadowWidth, shadowHeight := shadow.Size()
			put(shadow, x+8-float64(shadowWidth)/2, y+11-float64(shadowHeight)/2, white)
		}
		put(sprites.frame("characters-player", animation), x+1, y, white)
	}
}

// drawBeam draws a laser's beam, a pixel wide, from its muzzle to what stops it.
func (sprites *Sprites) drawBeam(list *draw2d.List, beam Beam, originX, originY float64) {
	laser := beam.Laser
	cellX, cellY := float64(laser.X*TileSize), float64(laser.Y*TileSize)
	reach := float64(beam.Cells * TileSize)
	var area draw2d.Rect
	switch laser.direction() {
	case Up:
		end := cellY - reach + TileSize
		area = draw2d.Rect{X: float32(cellX + 8), Y: float32(end), Width: 1, Height: float32(cellY + 13 - end)}
	case Down:
		end := cellY + reach
		area = draw2d.Rect{X: float32(cellX + 7), Y: float32(cellY + 3), Width: 1, Height: float32(end - cellY - 3)}
	case Right:
		end := cellX + reach
		area = draw2d.Rect{X: float32(cellX + 3), Y: float32(cellY + 8), Width: float32(end - cellX - 3), Height: 1}
	case Left:
		end := cellX - reach + TileSize
		area = draw2d.Rect{X: float32(end), Y: float32(cellY + 7), Width: float32(cellX + 13 - end), Height: 1}
	}
	if area.Width <= 0 || area.Height <= 0 {
		return
	}
	area.X, area.Y = area.X+float32(originX), area.Y+float32(originY)
	list.Rectangle(area, 0, sprites.filter.Colour(laserColour), draw2d.Color{}, 0)
}

// drawLabel writes a text of the world in the game's font, white with a soft shadow.
func (sprites *Sprites) drawLabel(list *draw2d.List, label Label, originX, originY float64) {
	size := float32(min(label.Size, 16))
	x := float32(math.Round(label.X + originX))
	if label.Centre {
		x += float32(label.Width)/2 - float32(len([]rune(label.Text)))*size/2
	}
	baseline := float32(math.Round(label.Y+originY)) + size
	list.Text(label.Text, x+1, baseline+1, size, draw2d.Regular, labelShadow)
	list.Text(label.Text, x, baseline, size, draw2d.Regular, sprites.filter.Colour(labelColour))
}

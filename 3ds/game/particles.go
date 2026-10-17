package game

import (
	"math"

	"github.com/akrck02/go-renderer/draw2d"
)

// The original's particles: the key gives off golden sparks that rise, the teleport purple ones
// that hang around it. Each is drawn from the clock alone (when it was born, where, how old it is
// now), so nothing is kept between frames and nothing is made each frame.

// emitter is a GPUParticles2D of the original, as its scene sets it.
type emitter struct {
	offsetX, offsetY float64 // where it is on its thing's cell
	radius           float64 // the sphere its particles are born in
	gravity          float64 // pixels a second squared, downwards
	lifetime         float64 // seconds
	randomness       float64 // of the lifetime
	colour           draw2d.Color
}

// particleAmount is Godot's default: eight particles alive at a time.
const particleAmount = 8

var (
	keySparks = emitter{offsetX: 8, offsetY: 4, radius: 10.23, gravity: -98, lifetime: 0.6, randomness: 0.03,
		colour: draw2d.Color{0.878431, 0.705882, 0, 1}}
	teleportSparks = emitter{offsetX: 8, offsetY: 7, radius: 14, lifetime: 1, randomness: 0.49,
		colour: draw2d.Color{0.486275, 0, 0.486275, 0.294118}}
)

// drawParticles draws an emitter's particles for a thing drawn at x, y (its cell's top-left on
// the screen); seed tells things apart, so two keys do not sparkle alike.
func (sprites *Sprites) drawParticles(list *draw2d.List, source emitter, x, y float64, seed int, clock float64) {
	colour := sprites.filter.Colour(source.colour)
	for index := 0; index < particleAmount; index++ {
		// each particle is reborn every lifetime, a share of a lifetime after the one before
		phase := clock + float64(index)*source.lifetime/particleAmount
		round := math.Floor(phase / source.lifetime)
		lifetime := source.lifetime * (1 - source.randomness*hash(seed, index, int(round), 3))
		age := phase - round*source.lifetime
		if age > lifetime {
			continue
		}
		// born somewhere in the sphere, seen from the front: a point of the disc
		angle := 2 * math.Pi * hash(seed, index, int(round), 1)
		distance := source.radius * math.Sqrt(hash(seed, index, int(round), 2))
		particleX := x + source.offsetX + distance*math.Cos(angle)
		particleY := y + source.offsetY + distance*math.Sin(angle) + source.gravity*age*age/2
		list.Rectangle(draw2d.Rect{X: float32(math.Round(particleX)), Y: float32(math.Round(particleY)), Width: 1, Height: 1}, 0, colour, draw2d.Color{}, 0)
	}
}

// Hash is a number from 0 to 1 that stays the same for the same arguments: chance that does not
// change from frame to frame or run to run.
func Hash(values ...int) float64 { return hash(values...) }

func hash(values ...int) float64 {
	state := uint64(14695981039346656037)
	for _, value := range values {
		state = (state ^ uint64(value)) * 1099511628211
		state ^= state >> 29
	}
	return float64(state%10007) / 10007
}

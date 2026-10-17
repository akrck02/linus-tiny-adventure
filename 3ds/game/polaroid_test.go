package game

import (
	"testing"

	"github.com/akrck02/go-renderer/draw2d"

	"github.com/akrck02/linus-tiny-adventure/3ds/assets"
)

func TestATurnedPolaroidIsLargerAndKeepsItsCard(t *testing.T) {
	data, err := assets.Files.ReadFile("early-gameboy.ttf")
	if err != nil {
		t.Fatal(err)
	}
	fonts, err := NewFonts(data)
	if err != nil {
		t.Fatal(err)
	}
	straight := fonts.Polaroid(PolaroidStyle{Width: 84, Height: 72, Label: "1", LabelSize: 16})
	turned := fonts.Polaroid(PolaroidStyle{Width: 84, Height: 72, Label: "1", LabelSize: 16, Degrees: 10})
	if width, height := straight.Size(); width != 84 || height != 72 {
		t.Fatalf("a straight polaroid is %d × %d", width, height)
	}
	width, height := turned.Size()
	if width <= 84 || height <= 72 {
		t.Fatalf("a turned polaroid is %d × %d: no larger than the card", width, height)
	}
	if centre := turned.Pixels.NRGBAAt(width/2, height/2); centre.A == 0 {
		t.Fatal("the middle of a turned polaroid is empty")
	}
}

func TestParticlesAreTheSameForTheSameMoment(t *testing.T) {
	sprites := &Sprites{}
	first, second := drawn(sprites, 1.25), drawn(sprites, 1.25)
	if len(first) == 0 || len(first) != len(second) {
		t.Fatalf("%d and %d particles", len(first), len(second))
	}
	for index := range first {
		if first[index] != second[index] {
			t.Fatal("the same moment drew particles in other places")
		}
	}
}

func drawn(sprites *Sprites, clock float64) [][2]float32 {
	var list drawList
	sprites.drawParticles(&list.List, keySparks, 100, 100, 3, clock)
	var places [][2]float32
	for _, command := range list.Commands {
		places = append(places, [2]float32{command.Area.X, command.Area.Y})
	}
	return places
}

type drawList struct{ draw2d.List }

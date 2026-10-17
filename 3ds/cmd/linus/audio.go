//go:build nintendo3ds

package main

import (
	"bytes"
	"math"
	"strings"

	sound "github.com/akrck02/go-sound"
	"github.com/akrck02/go-sound/vorbis"
	"github.com/akrck02/go-sound/wav"

	"github.com/akrck02/linus-tiny-adventure/3ds/assets"
	"github.com/akrck02/linus-tiny-adventure/3ds/game"
)

// audio plays the game's sounds with go-sound: an effect for what happens, and the music of the
// level's radio, which sounds fainter the farther the player is from it, as the original's
// positional player does.
type audio struct {
	context *sound.Context
	effects map[game.Event]*sound.Player
	music   *sound.Player
	song    string // the song music plays
	silent  string // why there is no sound, to say on the title screen; empty when there is
}

// sampleRate is the console's: the sounds are converted to it, so nothing is resampled.
const sampleRate = 22050

// The radio's player in the original: heard up to 500 pixels away, fading with this attenuation.
const (
	radioReach       = 500.0
	radioAttenuation = 3.4822
)

// effectSounds are the effect each event plays, and how loud (the teleport's player is at -10 dB).
var effectSounds = map[game.Event]struct {
	name   string
	volume float64
}{
	game.EventPush:     {"box-move", 1},
	game.EventBump:     {"impossible", 1},
	game.EventTeleport: {"teleport", 0.316},
	game.EventDoorOpen: {"door-open", 1},
	game.EventLock:     {"lock", 1},
}

// newAudio opens the sound; a console whose DSP cannot start plays the game silent.
func newAudio() *audio {
	context, err := sound.NewContext(sound.Options{SampleRate: sampleRate, Name: "Linu's tiny adventure"})
	if err == nil {
		err = context.Err()
	}
	if err != nil {
		println("linus: no sound:", err.Error())
		return &audio{silent: "NO SOUND: /3DS/DSPFIRM.CDC IS MISSING"}
	}
	result := &audio{context: context, effects: map[game.Event]*sound.Player{}}
	for event, effect := range effectSounds {
		data, err := assets.Files.ReadFile("sounds/" + effect.name + ".wav")
		if err != nil {
			continue
		}
		stream, err := wav.Decode(bytes.NewReader(data))
		if err != nil {
			continue
		}
		player, err := context.NewPlayer(stream)
		if err != nil {
			continue
		}
		player.SetVolume(effect.volume)
		result.effects[event] = player
	}
	return result
}

// play plays the effects of what happened; an effect still sounding is not started again, as the
// original's player does not overlap its own sounds.
func (audio *audio) play(events []game.Event) {
	for _, event := range events {
		if player := audio.effects[event]; player != nil && !player.IsPlaying() {
			player.Play()
		}
	}
}

// follow plays the music of the level's radio, at the volume the player's distance to it gives;
// with no level, or a radio without a song, the music stops.
func (audio *audio) follow(world *game.World) {
	if audio.context == nil {
		return
	}
	radio := radioOf(world)
	if radio == nil || radio.Song == "" {
		if audio.music != nil {
			audio.music.Pause()
		}
		return
	}
	if radio.Song != audio.song {
		audio.start(radio.Song)
	}
	if audio.music == nil {
		return
	}
	playerX, playerY := world.DrawnAt(world.Player)
	radioX, radioY := world.DrawnAt(radio)
	distance := math.Hypot(playerX-radioX, playerY-radioY)
	audio.music.SetVolume(math.Pow(max(1-distance/radioReach, 0), radioAttenuation))
	if !audio.music.IsPlaying() {
		audio.music.Play()
	}
}

// start plays a song from its beginning, over and over.
func (audio *audio) start(song string) {
	if audio.music != nil {
		audio.music.Close()
		audio.music = nil
	}
	audio.song = song
	data, err := assets.Files.ReadFile("sounds/" + strings.ToLower(song) + ".ogg")
	if err != nil {
		return
	}
	stream, err := vorbis.Decode(bytes.NewReader(data))
	if err != nil {
		return
	}
	player, err := audio.context.NewPlayer(stream)
	if err != nil {
		return
	}
	player.SetLoop(true)
	audio.music = player
}

func radioOf(world *game.World) *game.Entity {
	if world == nil {
		return nil
	}
	for _, entity := range world.Entities {
		if entity.Kind == game.KindRadio && !entity.Gone {
			return entity
		}
	}
	return nil
}

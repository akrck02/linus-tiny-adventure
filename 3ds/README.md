# Linu's tiny adventure on the Nintendo 3DS

A port of the game to the 3DS as homebrew, written in Go and built with TinyGo
([tinygo-3ds](https://github.com/akrck02/tinygo-3ds)), drawn with go-renderer's `ctr` (citro3d)
and go-ui for the bottom screen.

- **Top screen**: the level, pixel for pixel as the original draws it (16-pixel cells; a level of
  20 × 11 cells fits the 400 × 240 screen whole, the last one scrolls with the player).
- **Bottom screen**, touched: the level's name, the steps, reset, and the levels to choose.
- **Sound**, with [go-sound](https://github.com/akrck02/go-sound): an effect for each push, bump,
  teleport, door and lock, and the radio's music, fainter the farther the player is from it, as
  the original's positional player. The console needs the DSP's firmware, `/3ds/dspfirm.cdc` on the
  SD card; without it the title screen says so and the game plays silent.
- **Controls**: the cross or the circle pad walk; SELECT starts the level again; Y opens the
  levels; START goes back to the HOME menu.

## How it is made

The levels and sprites are not drawn again: `tools/convert` reads the Godot project and writes
what the port embeds.

| | From | To |
|---|---|---|
| Levels | `scenes/levels/*.tscn`: the tile map, the placed scenes, the texts | `assets/levels.json` |
| Sprites | `sprites/items/*.svg` (pixel art in rectangles, strips of three frames) and the PNGs | `assets/sprites/*.png` |
| Font | `fonts/Early GameBoy.ttf` | `assets/early-gameboy.ttf` |
| Sounds | `sounds/OST/*.mp3`, `sounds/effects/*.mp3` | `assets/sounds/`: the music as Ogg Vorbis, the effects as WAV, mono at 22050 Hz (with ffmpeg) |

```bash
go run ./tools/convert                                   # after changing the Godot project
go test ./...                                            # the rules, on any computer
tinygo build -target=nintendo3ds -o linus.3dsx ./cmd/linus
```

`game` holds the rules as the GDScript plays them (pushing, keys and doors, teleports, lasers,
buttons and each level's script) and draws a level into a `draw2d` list; `cmd/linus` is the
console's loop. To try every level without playing the ones before:
`-ldflags="-X main.unlocked=3"`.

## Not yet

- Progress is not saved between runs (TinyGo has no file system on the 3DS yet).
- The options screen (graphics filter, sound, controls) and the loading screen.

## AI disclosure

This port (everything in `3ds/`) was written with Claude, an AI assistant, and is pending human
review. The game itself, its art, music and levels are the original's; the converter reads them
as they are.

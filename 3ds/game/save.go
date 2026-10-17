package game

import (
	"encoding/json"
	"strings"
)

// The original keeps two files, both JSON: the save (the level last played and the highest
// reached) and the options (the camera filter and the volumes). The port keeps the same two, with
// the same fields; the filter is kept by its name rather than by the Godot shader's path.

// Progress is the save file's: the level last played and the highest reached, from 1.
type Progress struct {
	CurrentLevel int `json:"current_level"`
	MaxLevel     int `json:"max_level"`
}

// NewProgress is the original's new game: the first level.
func NewProgress() Progress { return Progress{CurrentLevel: 1, MaxLevel: 1} }

// ReadProgress reads a save file; a missing or broken one is a new game, as the original starts one.
func ReadProgress(data []byte) Progress {
	progress := NewProgress()
	if json.Unmarshal(data, &progress) != nil || progress.MaxLevel < 1 {
		return NewProgress()
	}
	progress.CurrentLevel = max(progress.CurrentLevel, 1)
	return progress
}

// Options are the options file's: the camera filter and the music's and effects' volumes, 0 to 1.
type Options struct {
	Filter         Filter
	Music, Effects float64
}

// DefaultOptions are the original's: grain, and both volumes at half.
func DefaultOptions() Options { return Options{Filter: DefaultFilter, Music: 0.5, Effects: 0.5} }

// optionsFile is the options file as the original lays it out.
type optionsFile struct {
	Graphics struct {
		Filter string `json:"filter"`
	} `json:"graphics"`
	Audio struct {
		Music   *float64 `json:"music"`
		Effects *float64 `json:"effects"`
	} `json:"audio"`
}

// ReadOptions reads an options file; what is missing or broken keeps its default.
func ReadOptions(data []byte) Options {
	options := DefaultOptions()
	var file optionsFile
	if json.Unmarshal(data, &file) != nil {
		return options
	}
	for _, entry := range Filters {
		if strings.EqualFold(entry.Name, file.Graphics.Filter) {
			options.Filter = entry.Filter
		}
	}
	volume := func(value *float64, otherwise float64) float64 {
		if value == nil {
			return otherwise
		}
		return min(max(*value, 0), 1)
	}
	options.Music = volume(file.Audio.Music, options.Music)
	options.Effects = volume(file.Audio.Effects, options.Effects)
	return options
}

// Marshal writes the progress as the save file holds it.
func (progress Progress) Marshal() []byte {
	data, _ := json.Marshal(progress)
	return data
}

// Marshal writes the options as the options file holds them.
func (options Options) Marshal() []byte {
	var file optionsFile
	for _, entry := range Filters {
		if entry.Filter == options.Filter {
			file.Graphics.Filter = strings.ToLower(entry.Name)
		}
	}
	file.Audio.Music, file.Audio.Effects = &options.Music, &options.Effects
	data, _ := json.Marshal(file)
	return data
}

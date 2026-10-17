module github.com/akrck02/linus-tiny-adventure/3ds

go 1.25.6

require github.com/akrck02/go-renderer v0.0.0

require (
	github.com/ebitengine/oto/v3 v3.5.1 // indirect
	github.com/ebitengine/purego v0.11.0 // indirect
	github.com/jfreymuth/oggvorbis v1.0.5 // indirect
	github.com/jfreymuth/pulse v0.1.3 // indirect
	github.com/jfreymuth/vorbis v1.0.2 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.41.0 // indirect
)

// go-renderer's feature/3ds, which has ctr.
replace github.com/akrck02/go-renderer => github.com/akrck02/go-renderer v0.0.0-20261017114320-1fc015dd62b2

require (
	github.com/akrck02/go-sound v0.0.0-20261017125245-5537ae6b3ed1
	golang.org/x/image v0.45.0
)

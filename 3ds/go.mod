module github.com/akrck02/linus-tiny-adventure/3ds

go 1.25.6

require (
	github.com/akrck02/go-renderer v0.0.0
	github.com/akrck02/go-ui v0.0.0-20261017114322-990d96da72d6
)

require (
	golang.org/x/image v0.45.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.41.0 // indirect
)

// go-ui asks for go-renderer v0.0.0 (it builds against a local copy); this is go-renderer's
// feature/3ds, which has ctr.
replace github.com/akrck02/go-renderer => github.com/akrck02/go-renderer v0.0.0-20261017114320-1fc015dd62b2

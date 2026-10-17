module github.com/akrck02/linus-tiny-adventure/3ds

go 1.25.6

require (
	github.com/akrck02/go-renderer v0.0.0
	github.com/akrck02/go-ui v0.0.0
)

require (
	golang.org/x/image v0.45.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.41.0 // indirect
)

// While the port is made: the local copies of go-renderer's and go-ui's feature/3ds.
replace (
	github.com/akrck02/go-renderer => /home/claude/projects/go-3ds-fase0/go-renderer-main
	github.com/akrck02/go-ui => /home/claude/projects/smart-factory-sim/go-ui
)

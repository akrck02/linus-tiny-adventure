package main

import (
	"bufio"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// scene is a Godot text scene (.tscn): its external resources by id and its nodes in file order.
type scene struct {
	resources map[string]string // id → res:// path
	nodes     []*node
}

// node is a node of a scene: its name, type or instanced scene, parent path and properties, as
// written (Vector2(…), "text", 1.5708…).
type node struct {
	name, kind, parent, instance string
	properties                   map[string]string
}

var (
	sectionPattern   = regexp.MustCompile(`^\[(\w+)(.*)\]$`)
	attributePattern = regexp.MustCompile(`(\w+)=("[^"]*"|ExtResource\("[^"]*"\)|\S+)`)
	vectorPattern    = regexp.MustCompile(`Vector2\(([-0-9.e]+), ([-0-9.e]+)\)`)
	resourcePattern  = regexp.MustCompile(`ExtResource\("([^"]*)"\)`)
)

// readScene reads a .tscn file. Only what the levels use is kept: resources and nodes.
func readScene(path string) (*scene, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	result := &scene{resources: map[string]string{}}
	var current *node
	lines := bufio.NewScanner(file)
	lines.Buffer(make([]byte, 1<<20), 1<<24)
	for lines.Scan() {
		line := lines.Text()
		if match := sectionPattern.FindStringSubmatch(line); match != nil {
			current = nil
			attributes := map[string]string{}
			for _, pair := range attributePattern.FindAllStringSubmatch(match[2], -1) {
				attributes[pair[1]] = strings.Trim(pair[2], `"`)
			}
			switch match[1] {
			case "ext_resource":
				result.resources[attributes["id"]] = attributes["path"]
			case "node":
				current = &node{name: attributes["name"], kind: attributes["type"], parent: attributes["parent"], properties: map[string]string{}}
				if resource := resourcePattern.FindStringSubmatch(attributes["instance"]); resource != nil {
					current.instance = result.resources[resource[1]]
				}
				result.nodes = append(result.nodes, current)
			}
			continue
		}
		if current == nil {
			continue
		}
		if key, value, found := strings.Cut(line, " = "); found {
			current.properties[key] = value
		}
	}
	return result, lines.Err()
}

// path is a node's path from the scene's root: "" for the root, "Name" or "Parent/Name".
func (item *node) path() string {
	switch item.parent {
	case "":
		return ""
	case ".":
		return item.name
	}
	return item.parent + "/" + item.name
}

// vector reads a Vector2 property; ok is false when the node does not set it.
func (item *node) vector(key string) (x, y float64, ok bool) {
	match := vectorPattern.FindStringSubmatch(item.properties[key])
	if match == nil {
		return 0, 0, false
	}
	x, _ = strconv.ParseFloat(match[1], 64)
	y, _ = strconv.ParseFloat(match[2], 64)
	return x, y, true
}

func (item *node) number(key string) float64 {
	value, _ := strconv.ParseFloat(item.properties[key], 64)
	return value
}

func (item *node) text(key string) string {
	value, err := strconv.Unquote(item.properties[key])
	if err != nil {
		return strings.Trim(item.properties[key], `"`)
	}
	return value
}

// position is where a node is from the scene's root, adding its parents' positions. A plain Node
// parent (not a Node2D) breaks the chain, as in Godot: its children are placed on the canvas, so
// the root's own position is taken away from them.
func (result *scene) position(item *node) (x, y float64) {
	byPath := map[string]*node{}
	var root *node
	for _, other := range result.nodes {
		byPath[other.path()] = other
		if other.parent == "" {
			root = other
		}
	}
	for at := item; at != nil && at.parent != ""; {
		dx, dy, _ := at.vector("position")
		x, y = x+dx, y+dy
		if at.parent == "." {
			return x, y
		}
		parent := byPath[at.parent]
		if parent != nil && parent.kind == "Node" {
			if root != nil {
				rootX, rootY, _ := root.vector("position")
				x, y = x-rootX, y-rootY
			}
			return x, y
		}
		at = parent
	}
	return x, y
}

// resolve turns a NodePath written relative to a node ("../Teleport2") into a path from the root.
func resolve(from *node, relative string) string {
	parts := strings.Split(from.path(), "/")
	for _, part := range strings.Split(relative, "/") {
		switch part {
		case "..":
			if len(parts) > 0 {
				parts = parts[:len(parts)-1]
			}
		case ".", "":
		default:
			parts = append(parts, part)
		}
	}
	return strings.Join(parts, "/")
}

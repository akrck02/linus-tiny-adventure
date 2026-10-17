package game

// buttonPressed does what a level's buttons do, as its script in the original did (level_1.gd,
// level_3.gd). The second level has none.
func (world *World) buttonPressed(button, by *Entity) {
	switch world.Level.Number {
	case 1:
		// a box on a button stays there and the button locks; when both are pressed the
		// breakable wall goes
		if by.Kind != KindBox {
			return
		}
		by.Blocked = true
		world.disable(button)
		if world.allPressed("Button", "Button2") {
			if wall := world.Named("BreakeableWall"); wall != nil {
				wall.Gone = true
			}
		}
	case 3:
		// each button turns its laser off, whatever presses it
		lasers := map[string]string{"Buttons/StartLaserButton": "Lasers/StartLaser", "Buttons/EndLaserButton": "Lasers/EndLaser"}
		if name, found := lasers[button.Name]; found {
			if laser := world.Named(name); laser != nil {
				laser.Disabled = true
			}
			world.disable(button)
		}
	}
}

func (world *World) disable(button *Entity) {
	button.Disabled = true
	world.events = append(world.events, EventLock)
}

func (world *World) allPressed(names ...string) bool {
	for _, name := range names {
		if button := world.Named(name); button == nil || !button.Active {
			return false
		}
	}
	return true
}

extends Node

# Engine signals
signal scene_change_requested(scene : String) 
signal level_change_requested(level : int, transition : bool)
signal level_changed(level : int)
signal world_stopped
signal world_restarted
signal camera_shake(direction : Direction)
signal fullscreen_requested

# Savestate signals
signal new_game_created
signal save_game_requested
signal game_savestate_loaded()

# Option signals
signal music_volume_changed(value : float)
signal effects_volume_changed(value : float)

# Player signals
signal player_coins_gathered(coins : int)
signal player_death
signal player_stop
signal player_restart
signal player_step(current_steps : int)

# Mechanic signals
signal button_pressed(button : PuzzleButton, action_starter : Node2D)
signal teleport_requested(node : Node2D, position : Vector2, detector : RayCast2D, direction : int)

# Level selector
signal button_level_focus(button : LevelButton)

# UI signals
signal ui_show_settings
signal ui_camera_filter_changed(filter : String)

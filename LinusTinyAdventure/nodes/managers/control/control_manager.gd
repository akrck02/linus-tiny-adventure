extends Node

const controls = [];
var current_controller;

# Called when the node enters the scene tree for the first time.
func _ready():
	Input.joy_connection_changed.connect(check_controls)
	

func _input(_event):
	
	if Input.is_action_just_pressed(Controls.FULLSCREEN):
		SignalDatabase.fullscreen_requested.emit();

func check_controls(_device,_connected):
	var pads = Input.get_connected_joypads();
	print(pads)
	
	for pad in pads:
		pass

func is_gamepad_connected():
	var gamepad = Input.get_joy_name(0)
	return gamepad != ""

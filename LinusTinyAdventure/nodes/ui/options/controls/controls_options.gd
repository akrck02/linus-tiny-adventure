extends Control
class_name ControlsOptionsUI

signal hide_requested

@onready var gamepad_controls = $Gamepad
@onready var keyboard_controls = $Keyboard
@onready var exit_button = $ExitButton

@onready var keyboard_up : Sprite2D = $Keyboard/UpSprite
@onready var keyboard_down : Sprite2D = $Keyboard/DownSprite
@onready var keyboard_left : Sprite2D = $Keyboard/LeftSprite
@onready var keyboard_right : Sprite2D = $Keyboard/RightSprite
@onready var keyboard_reset : Sprite2D = $Keyboard/ResetSprite
@onready var keyboard_interact : Sprite2D = $Keyboard/InteractSprite

@onready var gamepad_up : Sprite2D = $Gamepad/Controls/Unselected/UpSprite
@onready var gamepad_down : Sprite2D = $Gamepad/Controls/Unselected/DownSprite
@onready var gamepad_left : Sprite2D = $Gamepad/Controls/Unselected/LeftSprite
@onready var gamepad_right : Sprite2D = $Gamepad/Controls/Unselected/RightSprite
@onready var gamepad_reset : Sprite2D = $Gamepad/ResetLevel/Unselected/ResetSprite
@onready var gamepad_interact : Sprite2D = $Gamepad/Interaction/Unselected/InteractionRightSprite

@onready var gamepad_up_selected : Sprite2D = $Gamepad/Controls/Selected/UpSprite
@onready var gamepad_down_selected : Sprite2D = $Gamepad/Controls/Selected/DownSprite
@onready var gamepad_left_selected : Sprite2D = $Gamepad/Controls/Selected/LeftSprite
@onready var gamepad_right_selected : Sprite2D = $Gamepad/Controls/Selected/RightSprite
@onready var gamepad_reset_selected : Sprite2D = $Gamepad/ResetLevel/Selected/ResetSprite
@onready var gamepad_interact_selected : Sprite2D = $Gamepad/Interaction/Selected/InteractionRightSprite

var gamepad;

func show_now():
	exit_button.grab_focus()

func hide_now():
	hide_requested.emit();
	
func _ready():
	exit_button.pressed.connect(exit)

func _process(_delta):
	
	if Input.get_joy_name(0) != gamepad:
		set_input_icons()
		

func _input(_event):
	
	if Input.is_action_just_pressed(Controls.UP):
		gamepad_up_selected.show()
		gamepad_up.hide()
	elif Input.is_action_just_released(Controls.UP):
		gamepad_up_selected.hide()
		gamepad_up.show()

	if Input.is_action_just_pressed(Controls.DOWN):
		gamepad_down_selected.show()
		gamepad_down.hide()
	elif Input.is_action_just_released(Controls.DOWN):
		gamepad_down_selected.hide()
		gamepad_down.show()

	if Input.is_action_just_pressed(Controls.LEFT):
		gamepad_left_selected.show()
		gamepad_left.hide()
	elif Input.is_action_just_released(Controls.LEFT):
		gamepad_left_selected.hide()
		gamepad_left.show()
		
	if Input.is_action_just_pressed(Controls.RIGHT):
		gamepad_right_selected.show()
		gamepad_right.hide()
	elif Input.is_action_just_released(Controls.RIGHT):
		gamepad_right_selected.hide()
		gamepad_right.show()
		
	if Input.is_action_just_pressed(Controls.RESET_LEVEL):
		gamepad_reset_selected.show()
		gamepad_reset.hide()
	elif Input.is_action_just_released(Controls.RESET_LEVEL):
		gamepad_reset_selected.hide()
		gamepad_reset.show()
		

func set_input_icons():
	gamepad = Input.get_joy_name(0)
		
	if gamepad == "":
		gamepad_controls.hide()
		keyboard_controls.show()
	else: 
		keyboard_controls.hide()
		gamepad_controls.show()


func exit():
	hide_now()


extends Node2D

const levelButtonPrototype = preload(Constants.LEVEL_BUTTON);
@onready var ButtonPanel = $ButtonPanel
@onready var LevelNameText = $ui/LevelNameText
@onready var camera = $Camera2D
var LevelButtons = []

# Called when the node enters the scene tree for the first time.
func _ready():
	
	var levels = LevelManager.levels
	var i = 0
	for level in levels:
		
		var button : LevelButton = levelButtonPrototype.instantiate();
		button.text = String("%d" % level.number);
		button.set_meta("level_name",level.name)
		button.set_meta("level_number",level.number)
		
		button.position.y = 80
		button.position.x = (30 + i*1.25) * 100 
		
		ButtonPanel.add_child(button)
		LevelButtons.append(button)
		
		if i < LevelManager.max_level.number:
			button.unblock()
			button.rotate(randi_range(-10,10))
		else:
			button.block()
		
		i += 1
	
	SignalDatabase.button_level_focus.connect(focus_on_camera)
	LevelButtons[0].grab_focus()
	
func focus_on_camera(node : LevelButton):
	camera.position.x = node.position.x - 110
	LevelNameText.text = node.get_meta("level_name")
	
		

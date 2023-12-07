extends CanvasLayer

@onready var StepCounter : RichTextLabel = $StepsLevelText
@onready var ResetLevelIcon : Sprite2D = $ResetLevelIcon

var gamepad;

# Called when the node enters the scene tree for the first time.
func _ready():
	set_input_icons()
	set_steps(0)
	SignalDatabase.player_step.connect(set_steps)

func set_steps(steps = 0):
	StepCounter.text = String("Steps: %d" % steps);

func _process(_delta):
	
	if Input.get_joy_name(0) != gamepad:
		set_input_icons()
			

func set_input_icons():
	gamepad = Input.get_joy_name(0)
		
	if gamepad == "":
		ResetLevelIcon.texture = load(Constants.UI_SPRITE_PATH + "option_keyboard_f5.png")
	else: 
		ResetLevelIcon.texture = load(Constants.UI_SPRITE_PATH + "option_gamepad_r1.png")

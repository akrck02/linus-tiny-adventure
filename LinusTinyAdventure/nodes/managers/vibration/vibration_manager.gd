extends Node

const DEFAULT_DEVICE : int = 0
const DEFAULT_DURATION : float = 0.25 

const SHORT_WEAK : float = 0.1
const SHORT_STRONG : float = 0.2

func vibrate(weak : float = SHORT_WEAK, strong : float = SHORT_STRONG, duration : float = DEFAULT_DURATION):
	Input.start_joy_vibration(DEFAULT_DEVICE, weak, strong, duration)

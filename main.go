package main

import (
	"fmt"
	"log"
	"math"
	"os/exec"

	"github.com/go-vgo/robotgo"
	"github.com/veandco/go-sdl2/sdl"
)

const (
	analogDeadzone   = 8000 // Threshold for analog stick movement
	triggerThreshold = 16383

	leftTriggerAxis  = 4
	rightTriggerAxis = 5

	DIR_NONE  = 0
	DIR_UP    = 1 << 0
	DIR_RIGHT = 1 << 1
	DIR_DOWN  = 1 << 2
	DIR_LEFT  = 1 << 3
)

// Simple mapping of controller buttons to keyboard keys
var buttonKeyMap = map[int]string{
	0:  "shift", // Cross
	1:  "space", // Circle
	2:  "lctrl", // Square
	3:  "x",     // Triangle
	4:  "tab",   // Share
	5:  "k",     // PS Button
	6:  "esc",   // Options
	7:  "3",     // L3
	8:  "2",     // R3
	9:  "q",     // L1
	10: "w",     // R1
}

var triggerKeyMap = map[int]string{
	leftTriggerAxis:  "a",
	rightTriggerAxis: "s",
}

type ControllerMapper struct {
	running          bool
	pressedKeys      map[string]bool
	controller       *sdl.GameController
	currentDirection int
}

func showAlert(message string) {
	cmd := exec.Command("osascript", "-e", fmt.Sprintf(`display alert "Controller Error" message "%s" as critical`, message))
	cmd.Run()
}

func NewControllerMapper() (*ControllerMapper, error) {
	if err := sdl.Init(sdl.INIT_GAMECONTROLLER); err != nil {
		showAlert(fmt.Sprintf("Failed to initialize SDL: %v", err))
		return nil, fmt.Errorf("failed to init SDL: %v", err)
	}

	if sdl.NumJoysticks() < 1 {
		showAlert("No game controller detected. Please connect a controller and try again.")
		return nil, fmt.Errorf("no controller detected")
	}

	controller := sdl.GameControllerOpen(0)
	if controller == nil {
		showAlert("Failed to open the game controller. Please reconnect the controller and try again.")
		return nil, fmt.Errorf("failed to open controller")
	}

	fmt.Printf("Controller detected: %s\n", controller.Name())
	return &ControllerMapper{
		running:          true,
		pressedKeys:      make(map[string]bool),
		controller:       controller,
		currentDirection: DIR_NONE,
	}, nil
}

func (cm *ControllerMapper) pressKey(key string) {
	if !cm.pressedKeys[key] {
		robotgo.KeyDown(key)
		cm.pressedKeys[key] = true
	}
}

func (cm *ControllerMapper) releaseKey(key string) {
	if cm.pressedKeys[key] {
		robotgo.KeyUp(key)
		delete(cm.pressedKeys, key)
	}
}

func (cm *ControllerMapper) handleDirectionalInputs() {
	direction := DIR_NONE

	// Handle D-pad
	if cm.controller.Button(sdl.CONTROLLER_BUTTON_DPAD_UP) == 1 {
		direction |= DIR_UP
	}
	if cm.controller.Button(sdl.CONTROLLER_BUTTON_DPAD_DOWN) == 1 {
		direction |= DIR_DOWN
	}
	if cm.controller.Button(sdl.CONTROLLER_BUTTON_DPAD_LEFT) == 1 {
		direction |= DIR_LEFT
	}
	if cm.controller.Button(sdl.CONTROLLER_BUTTON_DPAD_RIGHT) == 1 {
		direction |= DIR_RIGHT
	}

	// Handle analog stick
	x := cm.controller.Axis(sdl.CONTROLLER_AXIS_LEFTX)
	y := cm.controller.Axis(sdl.CONTROLLER_AXIS_LEFTY)

	if math.Abs(float64(x)) > float64(analogDeadzone) || math.Abs(float64(y)) > float64(analogDeadzone) {
		if x > analogDeadzone {
			direction |= DIR_RIGHT
		} else if x < -analogDeadzone {
			direction |= DIR_LEFT
		}
		if y > analogDeadzone {
			direction |= DIR_DOWN
		} else if y < -analogDeadzone {
			direction |= DIR_UP
		}
	}

	// Update direction keys
	if direction&DIR_UP != 0 {
		cm.pressKey("up")
	} else {
		cm.releaseKey("up")
	}
	if direction&DIR_DOWN != 0 {
		cm.pressKey("down")
	} else {
		cm.releaseKey("down")
	}
	if direction&DIR_LEFT != 0 {
		cm.pressKey("left")
	} else {
		cm.releaseKey("left")
	}
	if direction&DIR_RIGHT != 0 {
		cm.pressKey("right")
	} else {
		cm.releaseKey("right")
	}

	cm.currentDirection = direction
}

func (cm *ControllerMapper) handleButtons() {
	for button, key := range buttonKeyMap {
		if cm.controller.Button(sdl.GameControllerButton(button)) == 1 {
			cm.pressKey(key)
		} else {
			cm.releaseKey(key)
		}
	}
}

func (cm *ControllerMapper) handleTriggers() {
	for axis, key := range triggerKeyMap {
		if cm.controller.Axis(sdl.GameControllerAxis(axis)) > triggerThreshold {
			cm.pressKey(key)
		} else {
			cm.releaseKey(key)
		}
	}
}

func (cm *ControllerMapper) Run() {
	defer func() {
		for key := range cm.pressedKeys {
			cm.releaseKey(key)
		}
		cm.controller.Close()
		sdl.Quit()
	}()

	for cm.running {
		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			if _, ok := event.(*sdl.QuitEvent); ok {
				cm.running = false
				return
			}
		}

		cm.handleDirectionalInputs()
		cm.handleButtons()
		cm.handleTriggers()

		sdl.Delay(16) // ~60Hz polling rate
	}
}

func main() {
	mapper, err := NewControllerMapper()
	if err != nil {
		showAlert(fmt.Sprintf("Failed to initialize controller mapper: %v", err))
		log.Fatalf("Failed to initialize controller mapper: %v", err)
	}

	fmt.Println("MGS controller mapper started.")
	mapper.Run()
}

package main

import (
	"fmt"
	"log"
	"os/exec"
	"time"

	"github.com/go-vgo/robotgo"
	"github.com/veandco/go-sdl2/sdl"
)

const (
	L2_AXIS           = 4         // Left trigger
	R2_AXIS           = 5         // Right trigger
	TRIGGER_THRESHOLD = 32767 / 2 // Threshold for considering trigger pressed
)

type ControllerMapper struct {
	running         bool
	pressedKeys     map[string]bool
	controller      *sdl.GameController
	lastPressTime   time.Time
	initialDelay    time.Duration
	repeatDelay     time.Duration
	analogState     map[string]bool
	lastAnalogPress map[string]time.Time
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
		running:         true,
		pressedKeys:     make(map[string]bool),
		controller:      controller,
		initialDelay:    500 * time.Millisecond,
		repeatDelay:     50 * time.Millisecond,
		analogState:     make(map[string]bool),
		lastAnalogPress: make(map[string]time.Time),
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

func (cm *ControllerMapper) handleAnalogDirection(direction string, isActive bool) {
	keyMap := map[string]string{
		"up":    "up",
		"down":  "down",
		"left":  "left",
		"right": "right",
	}

	key := keyMap[direction]
	now := time.Now()

	if isActive {
		if !cm.analogState[direction] {
			cm.pressKey(key)
			cm.analogState[direction] = true
			cm.lastAnalogPress[direction] = now
		} else {
			timeSinceLast := now.Sub(cm.lastAnalogPress[direction])
			if timeSinceLast >= cm.initialDelay {
				cm.releaseKey(key)
				cm.pressKey(key)
				cm.lastAnalogPress[direction] = now.Add(-cm.initialDelay + cm.repeatDelay)
			}
		}
	} else {
		if cm.analogState[direction] {
			cm.releaseKey(key)
			cm.analogState[direction] = false
			delete(cm.lastAnalogPress, direction)
		}
	}
}

func (cm *ControllerMapper) handleAnalogStick(x, y int16) {
	const deadzone = 8000 // SDL joystick values range from -32768 to 32767

	cm.handleAnalogDirection("right", x > deadzone)
	cm.handleAnalogDirection("left", x < -deadzone)
	cm.handleAnalogDirection("down", y > deadzone)
	cm.handleAnalogDirection("up", y < -deadzone)
}

// To read the triggers as analog values:
func getTriggerValues(gameController *sdl.GameController) (l2, r2 int16) {
	l2 = gameController.Axis(L2_AXIS)
	r2 = gameController.Axis(R2_AXIS)
	return
}

// Or to treat them as digital buttons:
func areTriggersPressed(gameController *sdl.GameController) (l2Pressed, r2Pressed bool) {
	l2 := gameController.Axis(L2_AXIS)
	r2 := gameController.Axis(R2_AXIS)

	l2Pressed = l2 > TRIGGER_THRESHOLD
	r2Pressed = r2 > TRIGGER_THRESHOLD
	return
}

func (cm *ControllerMapper) Run() {
	defer func() {
		// Release all pressed keys
		for key := range cm.pressedKeys {
			cm.releaseKey(key)
		}
		cm.controller.Close()
		sdl.Quit()
	}()

	buttonMap := map[int]string{
		0:  "shift", // Cross
		1:  "space", // Circle
		2:  "LCTRL", // Square
		3:  "x",     // Triangle
		4:  "tab",   // Share
		5:  "k",     // ??
		6:  "esc",   // Options
		7:  "3",     // L3
		8:  "2",     // R3
		9:  "q",     // L1
		10: "w",     // R1
		11: "up",    // dpad up
		12: "down",  // dpad left
		13: "left",  // dpad down
		14: "right", // dpad right
	}

	for cm.running {
		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			switch event.(type) {
			case *sdl.QuitEvent:
				cm.running = false
				return
			}
		}

		// Handle regular buttons
		for button, key := range buttonMap {
			state := cm.controller.Button(sdl.GameControllerButton(button))
			if state == 1 {
				cm.pressKey(key)
			} else {
				cm.releaseKey(key)
			}
		}

		// Handle triggers (L2 and R2)
		l2Pressed, r2Pressed := areTriggersPressed(cm.controller)
		if l2Pressed {
			cm.pressKey("a")
		} else {
			cm.releaseKey("a")
		}
		if r2Pressed {
			cm.pressKey("s")
		} else {
			cm.releaseKey("s")
		}

		// Handle left analog stick
		x := cm.controller.Axis(sdl.CONTROLLER_AXIS_LEFTX)
		y := cm.controller.Axis(sdl.CONTROLLER_AXIS_LEFTY)
		cm.handleAnalogStick(x, y)

		time.Sleep(10 * time.Millisecond)
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

package servo

import (
	"fmt"
	"log"
	"os/exec"
)

const gpioPin = 18

// Servo — управление сервоприводом через gpiozero (software PWM).
type Servo struct {
	pin int
}

// New создаёт контроллер сервопривода.
func New() *Servo {
	return &Servo{pin: gpioPin}
}

// Init проверяет доступность python3 и gpiozero.
func (s *Servo) Init() error {
	out, err := exec.Command("python3", "-c", "from gpiozero import Servo; print('ok')").CombinedOutput()
	if err != nil {
		return fmt.Errorf("gpiozero недоступен: %s: %w", string(out), err)
	}
	log.Printf("[slave/servo] инициализирован (GPIO%d, gpiozero)", s.pin)
	return nil
}

// MoveUp устанавливает серво в позицию 90° (вверх).
func (s *Servo) MoveUp() error {
	log.Println("[slave/servo] движение ВВЕРХ (90°)")
	return s.run("max")
}

// MoveDown устанавливает серво в позицию 0° (вниз).
func (s *Servo) MoveDown() error {
	log.Println("[slave/servo] движение ВНИЗ (0°)")
	return s.run("min")
}

// Close — ничего не делает, ресурсы освобождаются при завершении python.
func (s *Servo) Close() error {
	return nil
}

// run выполняет команду серво через python3 + gpiozero.
func (s *Servo) run(action string) error {
	script := fmt.Sprintf(`
from gpiozero import Servo
from time import sleep
s = Servo(%d)
s.%s()
sleep(0.5)
s.close()
`, s.pin, action)

	out, err := exec.Command("python3", "-c", script).CombinedOutput()
	if err != nil {
		return fmt.Errorf("servo %s: %s: %w", action, string(out), err)
	}
	log.Printf("[slave/servo] команда %s выполнена", action)
	return nil
}

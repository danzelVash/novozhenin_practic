package servo

import (
	"fmt"
	"log"
	"os"
	"time"
)

const (
	pwmChip    = "0"
	pwmChannel = "0"
	periodNs   = 20000000 // 20мс (50 Гц)
	dutyMinNs  = 500000   // ~0° (0.5мс)
	dutyMaxNs  = 1500000  // ~90° (1.5мс)
)

// Servo — управление сервоприводом через sysfs PWM.
type Servo struct {
	basePath string
}

// New создаёт контроллер сервопривода.
func New() *Servo {
	return &Servo{
		basePath: fmt.Sprintf("/sys/class/pwm/pwmchip%s/pwm%s", pwmChip, pwmChannel),
	}
}

// Init инициализирует PWM-канал.
func (s *Servo) Init() error {
	// Экспорт PWM-канала
	exportPath := fmt.Sprintf("/sys/class/pwm/pwmchip%s/export", pwmChip)
	if err := os.WriteFile(exportPath, []byte(pwmChannel), 0644); err != nil {
		log.Printf("[servo] export: %v (возможно уже экспортирован)", err)
	}

	time.Sleep(100 * time.Millisecond)

	// Установка периода
	if err := s.write("period", fmt.Sprintf("%d", periodNs)); err != nil {
		return fmt.Errorf("установка периода: %w", err)
	}

	// Начальная позиция — вниз (0°)
	if err := s.write("duty_cycle", fmt.Sprintf("%d", dutyMinNs)); err != nil {
		return fmt.Errorf("установка duty_cycle: %w", err)
	}

	// Включение PWM
	if err := s.write("enable", "1"); err != nil {
		return fmt.Errorf("включение PWM: %w", err)
	}

	log.Println("[servo] инициализирован (GPIO18, PWM0)")
	return nil
}

// MoveUp устанавливает серво в позицию 90° (вверх).
func (s *Servo) MoveUp() error {
	log.Println("[servo] движение ВВЕРХ (90°)")
	return s.write("duty_cycle", fmt.Sprintf("%d", dutyMaxNs))
}

// MoveDown устанавливает серво в позицию 0° (вниз).
func (s *Servo) MoveDown() error {
	log.Println("[servo] движение ВНИЗ (0°)")
	return s.write("duty_cycle", fmt.Sprintf("%d", dutyMinNs))
}

// Close отключает PWM.
func (s *Servo) Close() error {
	return s.write("enable", "0")
}

func (s *Servo) write(file, value string) error {
	path := fmt.Sprintf("%s/%s", s.basePath, file)
	return os.WriteFile(path, []byte(value), 0644)
}

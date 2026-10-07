package audit

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

// наблюдатель, который пишет события в файл
type FileObserver struct {
	filePath string
	mu       sync.Mutex
}

// создаем наблюдатель для записи в файл
func NewFileObserver(filePath string) (*FileObserver, error) {
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("Ошибка открытия файла аудита: %w", err)
	}
	file.Close()

	return &FileObserver{
		filePath: filePath,
	}, nil
}

// записываем событие в файл
func (o *FileObserver) Send(event AuditEvent) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	file, err := os.OpenFile(o.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("Ошибка открытия файла аудита: %w", err)
	}
	defer file.Close()

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("Ошибка сериализации события аудита: %w", err)
	}

	if _, err := file.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("Ошибка записи события аудита: %w", err)
	}

	return nil
}

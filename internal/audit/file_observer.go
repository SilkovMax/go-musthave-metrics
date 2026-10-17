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
	file     *os.File
	mu       sync.Mutex
}

// создаем наблюдатель для записи в файл
func NewFileObserver(filePath string) (*FileObserver, error) {
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("ошибка открытия файла аудита: %w", err)
	}

	return &FileObserver{
		filePath: filePath,
		file:     file,
	}, nil
}

// записываем событие в файл
func (o *FileObserver) Send(event AuditEvent) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	data, err := json.Marshal(event)

	if err != nil {
		return fmt.Errorf("ошибка открытия файла аудита: %w", err)
	}
	// Пишем прямо в открытый файл + перенос строки
	if _, err := o.file.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write audit event: %w", err)
	}

	return nil
}

func (o *FileObserver) Close() error {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.file.Close()
}

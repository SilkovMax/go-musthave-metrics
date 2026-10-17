package audit

import (
	"log"
	"sync/atomic"
)

// Принимает события от хендлеров и рассылает их наблюдателям
type Manager struct {
	observers []Observer
	eventCh   chan AuditEvent
	stopCh    chan struct{}
	stopped   atomic.Bool
}

func NewManager() *Manager {
	m := &Manager{
		observers: make([]Observer, 0),
		eventCh:   make(chan AuditEvent, 100), // буфер на 100 событий
		stopCh:    make(chan struct{}),
	}

	// Запускаем горутину для Асинхронной отправки событий
	go m.processEvents()

	return m
}

func (m *Manager) AddObserver(observer Observer) {
	m.observers = append(m.observers, observer)
}

// Отправляем событие всем наблюдателям
func (m *Manager) Notify(event AuditEvent) {
	if m.stopped.Load() {
		return
	}

	select {
	case m.eventCh <- event:
	default:
		log.Printf("AuditManager: очередь событий переполнена, событие пропущено")
	}
}

func (m *Manager) Stop() {
	// Сначала сигнализируем об остановке
	m.stopped.Store(true)
	close(m.stopCh)
}

// горутина, которая читает события из канала и рассылает наблюдателям\
func (m *Manager) processEvents() {
	defer close(m.eventCh)
	for {
		select {
		case event, ok := <-m.eventCh:
			if !ok {
				return // канал закрыт
			}

			// Отправляем событие всем наблюдателям
			for _, observer := range m.observers {
				if err := observer.Send(event); err != nil {
					log.Printf("AuditManager: ошибка отправки события: %v", err)
				}
			}

		case <-m.stopCh:
			return
		}
	}
}

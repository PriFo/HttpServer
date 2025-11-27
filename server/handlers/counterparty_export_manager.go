package handlers

import (
	"context"
	"errors"
	"time"
)

// CounterpartyExportManager контролирует количество одновременных тяжелых выгрузок.
const (
	defaultExportParallel       = 2
	defaultExportAcquireTimeout = 45 * time.Second
)

type CounterpartyExportManager struct {
	slots          chan struct{}
	acquireTimeout time.Duration
}

// ErrExportQueueBusy возвращается, когда очередь занята и запрос нельзя обслужить.
var ErrExportQueueBusy = errors.New("export queue is busy")

// NewCounterpartyExportManager создает менеджер с ограничением по параллелизму.
func NewCounterpartyExportManager(maxParallel int, acquireTimeout time.Duration) *CounterpartyExportManager {
	if maxParallel <= 0 {
		maxParallel = defaultExportParallel
	}
	if acquireTimeout <= 0 {
		acquireTimeout = defaultExportAcquireTimeout
	}
	return &CounterpartyExportManager{
		slots:          make(chan struct{}, maxParallel),
		acquireTimeout: acquireTimeout,
	}
}

// Acquire пытается занять слот очереди.
func (m *CounterpartyExportManager) Acquire(ctx context.Context) error {
	if m == nil {
		return nil
	}

	select {
	case m.slots <- struct{}{}:
		return nil
	default:
	}

	timer := time.NewTimer(m.acquireTimeout)
	defer timer.Stop()

	select {
	case m.slots <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return ErrExportQueueBusy
	}
}

// Release освобождает занятый слот.
func (m *CounterpartyExportManager) Release() {
	if m == nil {
		return
	}
	select {
	case <-m.slots:
	default:
	}
}

// NewDefaultCounterpartyExportManager упрощает создание менеджера с настройками по умолчанию.
func NewDefaultCounterpartyExportManager() *CounterpartyExportManager {
	return NewCounterpartyExportManager(defaultExportParallel, defaultExportAcquireTimeout)
}

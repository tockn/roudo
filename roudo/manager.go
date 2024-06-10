package roudo

import (
	"fmt"
	"log/slog"
	"roudo/roudo_event"
	"time"
)

type RoudoManager struct {
	reporter        RoudoReporter
	eventWatchers   []roudo_event.Watcher
	logger          *slog.Logger
	exitCh          chan error
	pollingInterval time.Duration
}

func NewRoudoManager(reporter RoudoReporter, eventWatchers []roudo_event.Watcher, logger *slog.Logger, pollingInterval time.Duration) *RoudoManager {
	return &RoudoManager{
		reporter:        reporter,
		eventWatchers:   eventWatchers,
		logger:          logger,
		exitCh:          make(chan error),
		pollingInterval: pollingInterval,
	}
}

func (m *RoudoManager) Kansi() error {
	for _, watcher := range m.eventWatchers {
		watcher := watcher
		go func() {
			m.logger.Debug("start watching: ", watcher.Name())
			if err := watcher.Watch(func() {
				if err := m.reporter.HandleRoudoEvent(); err != nil {
					m.logger.Error("handle event. event: %s, err: %s\n", watcher.Name(), err)
				}
			}); err != nil {
				m.exitCh <- fmt.Errorf("failed to start watch. event: %s, err: %s\n", watcher.Name(), err)
			}
		}()
	}
	m.logger.Debug("start polling")
	for {
		select {
		case <-time.After(m.pollingInterval):
			if err := m.reporter.Kansi(); err != nil {
				return err
			}
		case err := <-m.exitCh:
			return err
		}
	}
}

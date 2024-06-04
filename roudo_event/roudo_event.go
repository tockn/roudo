package roudo_event

import "log/slog"

type Watcher interface {
	Name() string
	Watch(onEvent func()) error
}

func NewAllWatchers(logger *slog.Logger) []Watcher {
	return []Watcher{
		&KeyboardEventWatcher{},
		&MouseEventWatcher{
			logger: logger,
		},
	}
}

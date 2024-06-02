package main

import (
	hook "github.com/robotn/gohook"
)

type IEventWatcher interface {
	Name() string
	Watch(onEvent func()) error
}

type KeyEventWatcher struct {
}

func (w *KeyEventWatcher) Name() string {
	return "KeyEventWatcher"
}

func (w *KeyEventWatcher) Watch(onEvent func()) error {
	hook.Register(hook.KeyDown, hook.AnyKeyCmd, func(e hook.Event) {
		onEvent()
	})

	s := hook.Start()
	<-hook.Process(s)
	return nil
}

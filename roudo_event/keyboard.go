package roudo_event

import (
	hook "github.com/robotn/gohook"
)

type KeyboardEventWatcher struct {
}

func (w *KeyboardEventWatcher) Name() string {
	return "KeyboardEventWatcher"
}

func (w *KeyboardEventWatcher) Watch(onEvent func()) error {
	hook.Register(hook.KeyDown, hook.AnyKeyCmd, func(e hook.Event) {
		onEvent()
	})

	s := hook.Start()
	<-hook.Process(s)
	return nil
}

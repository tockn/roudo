package main

import (
	"bytes"
	"fmt"
	"os/exec"
)

type Notificator interface {
	Notify(title, message string) error
}

type MacNotificator struct{}

func (no *MacNotificator) Notify(title string, message string) error {
	var errOut bytes.Buffer
	cmd := exec.Command("osascript", "-e", `display notification "`+message+`" with title "roudo" subtitle "`+title+`" sound name "Blow"`)
	cmd.Stderr = &errOut
	if err := cmd.Run(); err != nil {
		return fmt.Errorf(errOut.String())
	}
	return nil
}

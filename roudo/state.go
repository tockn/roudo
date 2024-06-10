package roudo

type RoudoState string

const (
	RoudoStateOff      = RoudoState("off")
	RoudoStateBreaking = RoudoState("breaking")
	RoudoStateWorking  = RoudoState("working")
)

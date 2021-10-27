package process

import (
	"time"
)

const (
	eventKindErr  kind = "error"
	eventKindData kind = "info"
)

type kind string

type event struct {
	time    time.Time
	kind    kind
	message string
}

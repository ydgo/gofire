package component

import (
	"context"
	"gofire/event"
)

type Receiver interface {
	ReadMessage(ctx context.Context) (*event.Event, error)
	Shutdown() error
}

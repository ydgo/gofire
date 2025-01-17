package component

import "context"

type Receiver interface {
	ReadMessage(ctx context.Context) (map[string]interface{}, error)
	Shutdown() error
}

package fswatch

import "context"

type KeyType int

// Register is register value of context.
func Register(ctx context.Context, key KeyType, value interface{}) context.Context {
	return context.WithValue(ctx, key, value)
}

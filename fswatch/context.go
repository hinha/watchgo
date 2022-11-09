package fswatch

import "context"

type KeyType int

const Log KeyType = iota

// Register is register value of context
func Register(ctx context.Context, key KeyType, value interface{}) context.Context {
	return context.WithValue(ctx, key, value)
}

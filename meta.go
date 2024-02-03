package pubsub

import (
	"context"
)

type Meta struct {
	ID        string
	EventName string
	Project   string
	Timestamp int64
}

func (this Meta) withContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, "metadata", this)
}

func fromContext(ctx context.Context) Meta {
	mt, ok := ctx.Value("metadata").(Meta)
	if !ok {
		return Meta{}
	}
	return mt
}

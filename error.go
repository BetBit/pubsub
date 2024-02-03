package pubsub

import (
	"context"
	"fmt"
)

type errMsg struct {
	Message string
}

func (this errMsg) withContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, "error", this)
}

func hasError(ctx context.Context) error {
	mt, ok := ctx.Value("error").(errMsg)
	if !ok {
		return nil
	}
	return fmt.Errorf(mt.Message)
}

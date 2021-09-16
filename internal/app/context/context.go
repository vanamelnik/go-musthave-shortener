package context

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

type privateKey string

const (
	idKey privateKey = "uuid"
)

// WithId добавляет в передаваемый контекст поле id.
func WithId(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, idKey, id)
}

// Id извлекает из передаваемого контекста поле id типа uuid.
func Id(ctx context.Context) (uuid.UUID, error) {
	id, ok := ctx.Value(idKey).(uuid.UUID)
	if !ok {
		return uuid.Nil, fmt.Errorf("context: Wrong uuid: %v", id)
	}

	return id, nil
}

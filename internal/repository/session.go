// Package repository определяет абстрактные интерфейсы для работы с данными.
package repository

import (
	"context"

	"github.com/antonlearn/go-shop-backend/internal/model"
)

// SessionRepository описывает контракт для управления сессиями (Refresh-токенами).
type SessionRepository interface {
	Create(ctx context.Context, session *model.Session) error
	GetByToken(ctx context.Context, token string) (*model.Session, error)
	DeleteByToken(ctx context.Context, token string) error
}

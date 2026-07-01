// Package repository определяет абстрактные интерфейсы для работы с данными.
package repository

import (
	"context"

	"github.com/antonlearn/go-shop-backend/internal/model"
)

// UserRepository описывает контракт для работы с таблицей пользователей.
type UserRepository interface {
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	GetByID(ctx context.Context, id int) (*model.User, error)
	Create(ctx context.Context, user *model.User) error
	Delete(ctx context.Context, id int) error
}

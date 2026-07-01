// Package repository определяет абстрактные интерфейсы для работы с данными.
package repository

import (
	"context"

	"github.com/antonlearn/go-shop-backend/internal/model"
)

// ProductRepository задает контракт взаимодействия бизнес-логики со складом товаров.
type ProductRepository interface {
	Create(ctx context.Context, product *model.Product) error
	GetByID(ctx context.Context, id int) (*model.Product, error)
	GetAll(ctx context.Context) ([]*model.Product, error)
	Update(ctx context.Context, product *model.Product) error
	Delete(ctx context.Context, id int) error
}

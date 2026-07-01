// Package repository определяет абстрактные интерфейсы для работы с данными.
// Этот пакет является точкой входа для бизнес-логики, обеспечивая независимость
// от конкретных технологий хранения.
package repository

import (
	"context"

	"github.com/antonlearn/go-shop-backend/internal/model"
)

// OrderRepository описывает контракт для работы с заказами.
// Бизнес-логика (service layer) должна зависеть от этого интерфейса, а не от SQLite.
type OrderRepository interface {
	// Create оформляет заказ внутри транзакции.
	Create(ctx context.Context, userID int) (*model.Order, error)
	// GetByID возвращает детальную информацию о заказе.
	GetByID(ctx context.Context, orderID, userID int) (*model.OrderOutput, error)
	// GetByUserID выгружает список заказов пользователя.
	GetByUserID(ctx context.Context, userID int) ([]*model.Order, error)
}

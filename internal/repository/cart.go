// Package repository определяет абстрактные интерфейсы для работы с данными.
package repository

import (
	"context"

	"github.com/antonlearn/go-shop-backend/internal/model"
)

// CartRepository описывает контракт для управления корзиной пользователя.
type CartRepository interface {
	// Add добавляет товар или обновляет количество (Smart Upsert).
	Add(ctx context.Context, item *model.CartItem) error
	// GetByUserID возвращает все элементы корзины с данными о товарах.
	GetByUserID(ctx context.Context, userID int) ([]*model.CartOutputItem, error)
	// Delete удаляет товар из корзины.
	Delete(ctx context.Context, userID, productID int) error
}

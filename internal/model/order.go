// Package model содержит доменные модели и DTO (Data Transfer Objects) приложения.
package model

import "time"

// Константы для статусов заказа
const (
	StatusCreated   = "created"   // Заказ создан, ожидается оплата
	StatusPaid      = "paid"      // Оплачен
	StatusDelivered = "delivered" // Доставлен
	StatusCancelled = "cancelled" // Отменен
)

// Order отражает общую информацию о заказе в базе данных.
type Order struct {
	ID         int       `json:"id" db:"id"`
	UserID     int       `json:"user_id" db:"user_id"`
	TotalPrice int       `json:"total_price" db:"total_price"` // Итоговая сумма в копейках
	Status     string    `json:"status" db:"status"`           // См. константы выше
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}

// OrderItem хранит информацию о конкретном выкупленном товаре внутри заказа.
type OrderItem struct {
	ID        int `json:"id" db:"id"`
	OrderID   int `json:"order_id" db:"order_id"`
	ProductID int `json:"product_id" db:"product_id"`
	Quantity  int `json:"quantity" db:"quantity"`
	Price     int `json:"price" db:"price"` // Цена товара НА МОМЕНТ ПОКУПКИ
}

// OrderOutput — развернутая структура заказа со списком всех его товаров для ответа API.
type OrderOutput struct {
	ID         int                `json:"id"`
	TotalPrice int                `json:"total_price"`
	Status     string             `json:"status"`
	CreatedAt  time.Time          `json:"created_at"`
	Items      []*OrderItemDetail `json:"items"`
}

// OrderItemDetail обогащает позицию заказа базовой информацией о товаре.
type OrderItemDetail struct {
	ProductID   int    `json:"product_id" db:"product_id"`
	ProductName string `json:"product_name" db:"product_name"`
	Quantity    int    `json:"quantity" db:"quantity"`
	Price       int    `json:"price" db:"price"` // Цена на момент покупки
	TotalPrice  int    `json:"total_price" db:"total_price"`
}

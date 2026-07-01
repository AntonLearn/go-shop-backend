// Package model содержит доменные модели и DTO (Data Transfer Objects) приложения.
package model

import "time"

// CartItem описывает структуру связи в базе данных.
type CartItem struct {
	ID        int       `json:"id" db:"id"`
	UserID    int       `json:"user_id" db:"user_id"`
	ProductID int       `json:"product_id" db:"product_id"`
	Quantity  int       `json:"quantity" db:"quantity"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// CartOutputItem — развернутая структура для ответа API (выгрузка корзины).
// Включает в себя агрегированные данные из таблицы товаров, чтобы фронтенд мог сразу всё отрисовать.
type CartOutputItem struct {
	CartItemID  int    `json:"cart_item_id" db:"cart_item_id"`
	ProductID   int    `json:"product_id" db:"product_id"`
	ProductName string `json:"product_name" db:"product_name"`
	Price       int    `json:"price" db:"price"` // Цена за 1 шт в копейках
	Quantity    int    `json:"quantity" db:"quantity"`
	TotalPrice  int    `json:"total_price" db:"total_price"` // Рассчитывается как Price * Quantity
}

// AddToCartInput — DTO для добавления товара в корзину или обновления его количества.
type AddToCartInput struct {
	ProductID int `json:"product_id" validate:"required,gt=0"`
	Quantity  int `json:"quantity" validate:"required,gt=0"` // Запрещено передавать 0 или отрицательные числа
}

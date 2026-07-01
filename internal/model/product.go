// Package model содержит доменные модели и DTO (Data Transfer Objects) приложения.
package model

import "time"

// Product описывает сущность товара в каталоге интернет-магазина.
type Product struct {
	ID          int       `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	Price       int       `json:"price" db:"price"` // Цена хранится в копейках/центах
	Stock       int       `json:"stock" db:"stock"` // Складской остаток
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// CreateProductInput — DTO для валидации данных при добавлении нового товара (доступно только Admin).
type CreateProductInput struct {
	Name        string `json:"name" validate:"required,min=3,max=150"`
	Description string `json:"description" validate:"required,max=1000"`
	Price       int    `json:"price" validate:"required,gt=0"`  // Цена не может быть нулевой или отрицательной
	Stock       int    `json:"stock" validate:"required,gte=0"` // Остаток может быть 0, но не меньше
}

// UpdateProductInput — DTO для частичного или полного обновления товара.
// Используем указатели, чтобы отличать "не передали поле" (nil) от "передали пустое значение".
type UpdateProductInput struct {
	Name        *string `json:"name" validate:"omitempty,min=3,max=150"`
	Description *string `json:"description" validate:"omitempty,max=1000"`
	Price       *int    `json:"price" validate:"omitempty,gt=0"`
	Stock       *int    `json:"stock" validate:"omitempty,gte=0"`
}

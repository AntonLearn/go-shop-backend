// Package model определяет базовые структуры данных, доменные сущности,
// DTO и глобальные ошибки приложения.
package model

import "errors"

var (
	// Ошибки сущности User и аутентификации
	ErrUserNotFound       = errors.New("user not found")
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrInvalidCredentials = errors.New("invalid email or password")

	// Ошибки сессий и токенов
	ErrSessionNotFound = errors.New("session not found")
	ErrInvalidToken    = errors.New("invalid or expired refresh token")

	// Ошибки сущности Product
	ErrProductNotFound = errors.New("product not found")

	// Ошибки сущности Cart
	ErrCartItemNotFound = errors.New("cart item not found")
	ErrCartIsEmpty      = errors.New("cart is empty")

	// Ошибки сущности Order
	ErrOrderNotFound = errors.New("order not found")
)

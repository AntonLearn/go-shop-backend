// Package model определяет базовые структуры данных, доменные сущности
// и модели ввода-вывода (DTO), используемые во всех слоях приложения.
package model

import "time"

// User описывает доменную сущность пользователя в системе.
type User struct {
	ID           int       `json:"id" db:"id"`
	Email        string    `json:"email" db:"email"`
	Name         string    `json:"name" db:"name"`       // Добавили имя пользователя
	PasswordHash string    `json:"-" db:"password_hash"` // Хеш пароля скрываем из JSON-ответов
	Role         string    `json:"role" db:"role"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// SignInInput описывает структуру входящих данных для формы авторизации.
type SignInInput struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// SignUpInput описывает структуру входящих данных для формы регистрации.
type SignUpInput struct {
	Name            string `json:"name" validate:"required,min=2,max=50"` // Добавили имя на этапе регистрации (минимум 2 символа)
	Email           string `json:"email" validate:"required,email"`
	Password        string `json:"password" validate:"required,min=8"`
	PasswordConfirm string `json:"password_confirm" validate:"required,eqfield=Password"`
}

// AuthResponse описывает структуру успешного ответа с парой токенов.
type AuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

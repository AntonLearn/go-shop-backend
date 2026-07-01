// Package model определяет базовые структуры данных, доменные сущности
// и модели ввода-вывода (DTO), используемые во всех слоях приложения.
package model

import "time"

type Session struct {
	ID           int       `db:"id"`
	UserID       int       `db:"user_id"`
	RefreshToken string    `db:"token"` // Синхронизировано с колонкой 'token' в схеме БД
	ExpiresAt    time.Time `db:"expires_at"`
	CreatedAt    time.Time `db:"created_at"`
}

// Package handler реализует транспортный слой (HTTP) приложения,
// обрабатывает входящие запросы, валидирует DTO и вызывает слой бизнес-логики.
package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/antonlearn/go-shop-backend/pkg/logger"
)

// Объявляем приватный тип для ключей контекста внутри пакета, чтобы избежать коллизий
type contextKey string

const (
	userIDKey    contextKey = "user_id"
	userRoleKey  contextKey = "user_role"
	userEmailKey contextKey = "user_email"
)

// --- Хелперы для работы с контекстом пользователей ---

// ContextWithUserID добавляет ID пользователя в контекст.
func ContextWithUserID(ctx context.Context, userID int) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// ContextWithUserRole добавляет роль пользователя в контекст.
func ContextWithUserRole(ctx context.Context, role string) context.Context {
	return context.WithValue(ctx, userRoleKey, role)
}

// ContextWithUserEmail добавляет Email пользователя в контекст.
func ContextWithUserEmail(ctx context.Context, email string) context.Context {
	return context.WithValue(ctx, userEmailKey, email)
}

// GetUserID безопасно извлекает ID пользователя из контекста.
func GetUserID(ctx context.Context) (int, bool) {
	id, ok := ctx.Value(userIDKey).(int)
	return id, ok
}

// GetUserRole безопасно извлекает роль пользователя из контекста.
func GetUserRole(ctx context.Context) (string, bool) {
	role, ok := ctx.Value(userRoleKey).(string)
	return role, ok
}

// GetUserEmail безопасно извлекает email пользователя из контекста.
func GetUserEmail(ctx context.Context) (string, bool) {
	email, ok := ctx.Value(userEmailKey).(string)
	return email, ok
}

// --- Единые хелперы ответа ---

// respondWithJSON — единый пакетный хелпер для отправки успешных JSON-ответов.
func respondWithJSON(w http.ResponseWriter, ctx context.Context, log *logger.Logger, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.ErrorContext(ctx, "критическая ошибка сериализации JSON ответа", "error", err)
	}
}

// respondWithError — единый пакетный хелпер для отправки стандартизированных JSON-ошибок.
func respondWithError(w http.ResponseWriter, ctx context.Context, log *logger.Logger, status int, message string) {
	respondWithJSON(w, ctx, log, status, map[string]string{"error": message})
}

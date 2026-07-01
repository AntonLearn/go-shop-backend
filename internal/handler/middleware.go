// Package handler реализует транспортный слой (HTTP) приложения,
// обрабатывает входящие запросы, валидирует DTO и вызывает слой бизнес-логики.
package handler

import (
	"net/http"
	"strings"

	"github.com/antonlearn/go-shop-backend/pkg/jwt"
	"github.com/antonlearn/go-shop-backend/pkg/logger"
)

// TokenParserInterface описывает контракт для валидации JWT токенов,
// позволяя подменять парсер на мок-объект в тестах роутера и middleware.
type TokenParserInterface interface {
	ValidateToken(token string) (*jwt.CustomClaims, error)
}

// TokenParserWrapper — стандартная обертка для вызова функций из pkg/jwt.
type TokenParserWrapper struct{}

// ValidateToken вызывает валидацию токена из пакета pkg/jwt.
func (w TokenParserWrapper) ValidateToken(token string) (*jwt.CustomClaims, error) {
	return jwt.ValidateToken(token)
}

// AuthMiddleware перехватывает запрос, проверяет Access Token (JWT)
// и обогащает контекст запроса типизированными данными пользователя.
func AuthMiddleware(tokenParser TokenParserInterface, log *logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				respondWithError(w, r.Context(), log, http.StatusUnauthorized, "отсутствует заголовок Authorization")
				return
			}

			// Ожидаем формат: "Bearer <token>"
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				respondWithError(w, r.Context(), log, http.StatusUnauthorized, "неверный формат заголовка Authorization (ожидается Bearer)")
				return
			}

			tokenString := parts[1]

			// Валидируем токен через интерфейс парсера
			claims, err := tokenParser.ValidateToken(tokenString)
			if err != nil {
				log.WarnContext(r.Context(), "попытка доступа с невалидным JWT токеном", "error", err)
				respondWithError(w, r.Context(), log, http.StatusUnauthorized, "неверный или протухший токен доступа")
				return
			}

			// Записываем данные в контекст с помощью безопасных функций пакета
			ctx := ContextWithUserID(r.Context(), claims.UserID)
			ctx = ContextWithUserRole(ctx, claims.Role)
			ctx = ContextWithUserEmail(ctx, claims.Email)

			// Передаем управление дальше с обновленным контекстом
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole проверяет, обладает ли текущий пользователь нужной ролью (RBAC).
// Должен вызываться СТРОГО после AuthMiddleware.
func RequireRole(allowedRole string, log *logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Достаем роль из контекста через безопасный хелпер
			role, ok := GetUserRole(r.Context())
			if !ok {
				respondWithError(w, r.Context(), log, http.StatusUnauthorized, "пользователь не аутентифицирован")
				return
			}

			if role != allowedRole {
				log.WarnContext(r.Context(), "отклонен доступ по роли", "expected", allowedRole, "actual", role)
				respondWithError(w, r.Context(), log, http.StatusForbidden, "недостаточно прав для выполнения данной операции")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

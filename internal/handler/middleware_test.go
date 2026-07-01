// Package handler реализует транспортный слой приложения (HTTP).
// Файл middleware_test.go содержит модульные тесты для защитных механизмов:
// аутентификации (проверка JWT) и авторизации (разграничение прав по ролям RBAC).
package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/antonlearn/go-shop-backend/pkg/jwt"
	"github.com/antonlearn/go-shop-backend/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockTokenParser реализует контракт TokenParserInterface.
// Используется для изоляции логики мидлварей от реального процесса декодирования JWT.
type MockTokenParser struct {
	mock.Mock
}

// ValidateToken имитирует валидацию строки токена и возврат кастомных клеймов.
func (m *MockTokenParser) ValidateToken(token string) (*jwt.CustomClaims, error) {
	args := m.Called(token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*jwt.CustomClaims), args.Error(1)
}

// TestAuthMiddleware проверяет цепочку извлечения токена, его валидацию
// и последующее безопасное обогащение контекста запроса.
func TestAuthMiddleware(t *testing.T) {
	// Создаем легковесный логгер для захвата предупреждений в тестах
	log, _ := logger.New("Console", "INFO")

	// Тестовый хендлер-заглушка (next). Должен вызываться только при успешной аутентификации.
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Внутри проверяем, что мидлварь корректно распаковала и записала данные в контекст
		userID, okID := GetUserID(r.Context())
		role, okRole := GetUserRole(r.Context())
		email, okEmail := GetUserEmail(r.Context())

		assert.True(t, okID, "ID пользователя должен присутствовать в контексте")
		assert.True(t, okRole, "Роль пользователя должна присутствовать в контексте")
		assert.True(t, okEmail, "Email пользователя должен присутствовать в контексте")

		assert.Equal(t, 123, userID)
		assert.Equal(t, "client", role)
		assert.Equal(t, "test@test.com", email)

		w.WriteHeader(http.StatusOK)
	})

	t.Run("Success Valid Token", func(t *testing.T) {
		mockParser := new(MockTokenParser)
		middleware := AuthMiddleware(mockParser, log)

		// Подготавливаем ожидаемый результат парсинга токена
		claims := &jwt.CustomClaims{
			UserID: 123,
			Role:   "client",
			Email:  "test@test.com",
		}
		mockParser.On("ValidateToken", "valid_jwt_string").Return(claims, nil)

		// Конструируем HTTP-запрос с правильным заголовком Authorization
		req := httptest.NewRequest(http.MethodGet, "/api/v1/cart", nil)
		req.Header.Set("Authorization", "Bearer valid_jwt_string")
		rr := httptest.NewRecorder()

		// Запускаем цепочку выполнения
		middleware(nextHandler).ServeHTTP(rr, req)

		// Проверяем, что запрос успешно дошел до внутреннего хендлера
		assert.Equal(t, http.StatusOK, rr.Code)
		mockParser.AssertExpectations(t)
	})

	t.Run("Missing Authorization Header", func(t *testing.T) {
		mockParser := new(MockTokenParser)
		middleware := AuthMiddleware(mockParser, log)

		// Отправляем запрос без заголовков
		req := httptest.NewRequest(http.MethodGet, "/api/v1/cart", nil)
		rr := httptest.NewRecorder()

		middleware(nextHandler).ServeHTTP(rr, req)

		// Ожидаем прерывание запроса со статусом 401 Unauthorized
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		assert.Contains(t, rr.Body.String(), "отсутствует заголовок Authorization")
	})

	t.Run("Invalid Header Format - Bad Type", func(t *testing.T) {
		mockParser := new(MockTokenParser)
		middleware := AuthMiddleware(mockParser, log)

		// Отправляем заголовок в неверном формате (например, Basic вместо Bearer)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/cart", nil)
		req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
		rr := httptest.NewRecorder()

		middleware(nextHandler).ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		assert.Contains(t, rr.Body.String(), "неверный формат заголовка Authorization (ожидается Bearer)")
	})

	t.Run("Invalid Header Format - Single Word", func(t *testing.T) {
		mockParser := new(MockTokenParser)
		middleware := AuthMiddleware(mockParser, log)

		// Отправляем заголовок, состоящий только из одного слова (без самого токена)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/cart", nil)
		req.Header.Set("Authorization", "Bearer")
		rr := httptest.NewRecorder()

		middleware(nextHandler).ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		assert.Contains(t, rr.Body.String(), "неверный формат заголовка Authorization (ожидается Bearer)")
	})

	t.Run("Token Verification Error", func(t *testing.T) {
		mockParser := new(MockTokenParser)
		middleware := AuthMiddleware(mockParser, log)

		// Имитируем ситуацию, когда токен просрочен или поврежден
		mockParser.On("ValidateToken", "expired_token").Return(nil, errors.New("token is expired"))

		req := httptest.NewRequest(http.MethodGet, "/api/v1/cart", nil)
		req.Header.Set("Authorization", "Bearer expired_token")
		rr := httptest.NewRecorder()

		middleware(nextHandler).ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		assert.Contains(t, rr.Body.String(), "неверный или протухший токен доступа")
	})
}

// TestRequireRole проверяет корректность разграничения доступа на основе ролей (RBAC).
func TestRequireRole(t *testing.T) {
	log, _ := logger.New("Console", "INFO")
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	t.Run("Success Role Matches", func(t *testing.T) {
		// Настраиваем ограничение: доступ разрешен только для "admin"
		middleware := RequireRole("admin", log)

		// Имитируем успешное прохождение AuthMiddleware через ручную сборку контекста с ролью "admin"
		ctx := ContextWithUserRole(context.Background(), "admin")
		req := httptest.NewRequest(http.MethodPost, "/api/v1/products", nil).WithContext(ctx)
		rr := httptest.NewRecorder()

		middleware(nextHandler).ServeHTTP(rr, req)

		// Ожидаем успешный доступ к эндпоинту
		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("Forbidden Role Mismatch", func(t *testing.T) {
		middleware := RequireRole("admin", log)

		// Передаем контекст обычного клиента ("client"), пытающегося зайти в админку
		ctx := ContextWithUserRole(context.Background(), "client")
		req := httptest.NewRequest(http.MethodPost, "/api/v1/products", nil).WithContext(ctx)
		rr := httptest.NewRecorder()

		middleware(nextHandler).ServeHTTP(rr, req)

		// Ожидаем блокировку доступа: 403 Forbidden
		assert.Equal(t, http.StatusForbidden, rr.Code)
		assert.Contains(t, rr.Body.String(), "недостаточно прав для выполнения данной операции")
	})

	t.Run("Unauthorized Missing Role In Context", func(t *testing.T) {
		middleware := RequireRole("admin", log)

		// Отправляем запрос с пустым контекстом (имитация вызова RequireRole в обход AuthMiddleware)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/products", nil)
		rr := httptest.NewRecorder()

		middleware(nextHandler).ServeHTTP(rr, req)

		// Система должна безопасно отклонить запрос со статусом 401 Unauthorized вместо паники
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		assert.Contains(t, rr.Body.String(), "пользователь не аутентифицирован")
	})
}

// Package handler реализует транспортный слой приложения (HTTP).
// Файл router_test.go проверяет корректность сборки дерева маршрутов Chi,
// маппинг путей (URL) и правильность наложения middleware на разные группы API.
package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/antonlearn/go-shop-backend/pkg/jwt"
	"github.com/antonlearn/go-shop-backend/pkg/logger"
	"github.com/stretchr/testify/assert"
)

// TestInitRouter_MiddlewareIntegration проверяет, что роутер правильно
// распределяет запросы и защищает приватные зоны с помощью мидлварей.
func TestInitRouter_MiddlewareIntegration(t *testing.T) {
	log, _ := logger.New("Console", "INFO")

	// Инициализируем пустые структуры хендлеров.
	// Нам не нужно наполнять их сервисами, так как мы проверяем перехват запросов
	// на уровне мидлварей ДО того, как они дойдут до логики самих хендлеров.
	authH := &AuthHandler{}
	productH := &ProductHandler{}
	cartH := &CartHandler{}

	// --- ГРУППА 1: ПУБЛИЧНЫЕ ЭНДПОИНТЫ АУТЕНТИФИКАЦИИ ---
	t.Run("Public Auth Routes Access", func(t *testing.T) {
		mockParser := new(MockTokenParser) // Используем мок из middleware_test.go
		router := InitRouter(authH, productH, cartH, mockParser, log)

		routes := []struct {
			method string
			path   string
		}{
			{http.MethodPost, "/api/v1/auth/sign-up"},
			{http.MethodPost, "/api/v1/auth/sign-in"},
			{http.MethodPost, "/api/v1/auth/refresh"},
			{http.MethodPost, "/api/v1/auth/logout"},
		}

		for _, tc := range routes {
			t.Run(tc.method+" "+tc.path, func(t *testing.T) {
				req := httptest.NewRequest(tc.method, tc.path, nil)
				rr := httptest.NewRecorder()

				router.ServeHTTP(rr, req)

				// Так как хендлер пустой, успешный проход через мидлвари вызовет панику на уровне
				// бизнес-логики, которую поймает Recoverer и вернет 500 вместо 401/403.
				assert.Equal(t, http.StatusInternalServerError, rr.Code,
					"Эндпоинт аутентификации должен быть публичным и пропускать запрос к хендлеру")
			})
		}
	})

	// --- ГРУППА 2: КАТАЛОГ ТОВАРОВ (ПУБЛИЧНЫЙ ДОСТУП) ---
	t.Run("Public Product Routes Access", func(t *testing.T) {
		mockParser := new(MockTokenParser)
		router := InitRouter(authH, productH, cartH, mockParser, log)

		routes := []struct {
			method string
			path   string
		}{
			{http.MethodGet, "/api/v1/products"},
			{http.MethodGet, "/api/v1/products/10"},
		}

		for _, tc := range routes {
			t.Run(tc.method+" "+tc.path, func(t *testing.T) {
				req := httptest.NewRequest(tc.method, tc.path, nil)
				rr := httptest.NewRecorder()

				router.ServeHTTP(rr, req)

				assert.Equal(t, http.StatusInternalServerError, rr.Code,
					"Просмотр товаров должен быть доступен без авторизации")
			})
		}
	})

	// --- ГРУППА 3: КАТАЛОГ ТОВАРОВ (УПРАВЛЕНИЕ АДМИНИСТРАТОРОМ) ---
	t.Run("Admin Product Routes - RBAC Verification", func(t *testing.T) {
		routes := []struct {
			method string
			path   string
		}{
			{http.MethodPost, "/api/v1/products"},
			{http.MethodPut, "/api/v1/products/10"},
			{http.MethodDelete, "/api/v1/products/10"},
		}

		for _, tc := range routes {
			// Сценарий А: Запрос вообще без токена -> 401 Unauthorized
			t.Run(tc.method+" "+tc.path+" - Missing Token", func(t *testing.T) {
				mockParser := new(MockTokenParser)
				router := InitRouter(authH, productH, cartH, mockParser, log)

				req := httptest.NewRequest(tc.method, tc.path, nil)
				rr := httptest.NewRecorder()

				router.ServeHTTP(rr, req)

				assert.Equal(t, http.StatusUnauthorized, rr.Code)
				assert.Contains(t, rr.Body.String(), "отсутствует заголовок Authorization")
			})

			// Сценарий Б: Невалидный или протухший токен -> 401 Unauthorized
			t.Run(tc.method+" "+tc.path+" - Invalid Token", func(t *testing.T) {
				mockParser := new(MockTokenParser)
				router := InitRouter(authH, productH, cartH, mockParser, log)

				mockParser.On("ValidateToken", "expired-token").Return((*jwt.CustomClaims)(nil), errors.New("token expired"))

				req := httptest.NewRequest(tc.method, tc.path, nil)
				req.Header.Set("Authorization", "Bearer expired-token")
				rr := httptest.NewRecorder()

				router.ServeHTTP(rr, req)

				assert.Equal(t, http.StatusUnauthorized, rr.Code)
				assert.Contains(t, rr.Body.String(), "неверный или протухший токен доступа")
				mockParser.AssertExpectations(t)
			})

			// Сценарий В: Токен валидный, но роль обычного пользователя -> 403 Forbidden
			t.Run(tc.method+" "+tc.path+" - Insufficient Rights (Role: user)", func(t *testing.T) {
				mockParser := new(MockTokenParser)
				router := InitRouter(authH, productH, cartH, mockParser, log)

				userClaims := &jwt.CustomClaims{
					UserID: 42,
					Role:   "user", // Обычный пользователь, у которого нет прав админа
					Email:  "user@shop.com",
				}
				mockParser.On("ValidateToken", "user-token").Return(userClaims, nil)

				req := httptest.NewRequest(tc.method, tc.path, nil)
				req.Header.Set("Authorization", "Bearer user-token")
				rr := httptest.NewRecorder()

				router.ServeHTTP(rr, req)

				assert.Equal(t, http.StatusForbidden, rr.Code, "Должен вернуться 403 Forbidden для не-админа")
				assert.Contains(t, rr.Body.String(), "недостаточно прав для выполнения данной операции")
				mockParser.AssertExpectations(t)
			})

			// Сценарий Г: Честный проход Администратора -> Мидлвари пройдены, падает в пустой хендлер (500)
			t.Run(tc.method+" "+tc.path+" - Access Granted (Role: admin)", func(t *testing.T) {
				mockParser := new(MockTokenParser)
				router := InitRouter(authH, productH, cartH, mockParser, log)

				adminClaims := &jwt.CustomClaims{
					UserID: 1,
					Role:   "admin", // Идеальное совпадение с RequireRole("admin")
					Email:  "admin@shop.com",
				}
				mockParser.On("ValidateToken", "admin-token").Return(adminClaims, nil)

				req := httptest.NewRequest(tc.method, tc.path, nil)
				req.Header.Set("Authorization", "Bearer admin-token")
				rr := httptest.NewRecorder()

				router.ServeHTTP(rr, req)

				assert.Equal(t, http.StatusInternalServerError, rr.Code,
					"Администратор должен успешно проходить проверку ролей и вызывать панику в пустом хендлере")
				mockParser.AssertExpectations(t)
			})
		}
	})

	// --- ГРУППА 4: КОРЗИНА (ТРЕБУЕТ ТОЛЬКО АВТОРИЗАЦИЮ) ---
	t.Run("Cart Routes - Authentication Verification", func(t *testing.T) {
		routes := []struct {
			method string
			path   string
		}{
			{http.MethodPost, "/api/v1/cart"},
			{http.MethodGet, "/api/v1/cart"},
			{http.MethodDelete, "/api/v1/cart/5"},
		}

		for _, tc := range routes {
			t.Run(tc.method+" "+tc.path+" - Unauthorized", func(t *testing.T) {
				mockParser := new(MockTokenParser)
				router := InitRouter(authH, productH, cartH, mockParser, log)

				req := httptest.NewRequest(tc.method, tc.path, nil)
				rr := httptest.NewRecorder()

				router.ServeHTTP(rr, req)

				assert.Equal(t, http.StatusUnauthorized, rr.Code, "Доступ без токена заблокирован")
			})

			t.Run(tc.method+" "+tc.path+" - Authorized Pass", func(t *testing.T) {
				mockParser := new(MockTokenParser)
				router := InitRouter(authH, productH, cartH, mockParser, log)

				userClaims := &jwt.CustomClaims{
					UserID: 100,
					Role:   "user", // Для корзины роли "user" абсолютно достаточно
					Email:  "buyer@shop.com",
				}
				mockParser.On("ValidateToken", "any-valid-token").Return(userClaims, nil)

				req := httptest.NewRequest(tc.method, tc.path, nil)
				req.Header.Set("Authorization", "Bearer any-valid-token")
				rr := httptest.NewRecorder()

				router.ServeHTTP(rr, req)

				assert.Equal(t, http.StatusInternalServerError, rr.Code,
					"Авторизованный пользователь должен беспрепятственно доходить до хендлеров корзины")
				mockParser.AssertExpectations(t)
			})
		}
	})

	// --- ГРУППА 5: ОБРАБОТКА НЕИЗВЕСТНЫХ МАРШРУТОВ ---
	t.Run("Non Existent Route 404", func(t *testing.T) {
		mockParser := new(MockTokenParser)
		router := InitRouter(authH, productH, cartH, mockParser, log)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/unknown-endpoint", nil)
		rr := httptest.NewRecorder()

		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusNotFound, rr.Code, "Роутер должен возвращать 404 для неизвестных путей")
	})
}

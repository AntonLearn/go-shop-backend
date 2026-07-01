// Package handler_test предоставляет модульные тесты для проверки функциональности
// транспортного слоя управления корзиной (CartHandler). Тестирование спроектировано
// по принципу "черного ящика" (black-box), изолируя логику HTTP-обработчиков
// от уровня базы данных и внешних доменных служб при помощи моков.
package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/antonlearn/go-shop-backend/internal/handler"
	"github.com/antonlearn/go-shop-backend/internal/model"
	"github.com/antonlearn/go-shop-backend/pkg/logger"
)

// mockCartService является реализацией CartServiceInterface для изоляции транспортного слоя.
type mockCartService struct {
	mock.Mock
}

func (m *mockCartService) Add(ctx context.Context, userID int, input model.AddToCartInput) error {
	args := m.Called(ctx, userID, input)
	return args.Error(0)
}

func (m *mockCartService) GetByUserID(ctx context.Context, userID int) ([]*model.CartOutputItem, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.CartOutputItem), args.Error(1)
}

func (m *mockCartService) Delete(ctx context.Context, userID int, productID int) error {
	args := m.Called(ctx, userID, productID)
	return args.Error(0)
}

// cartErrorResponse отображает структуру стандартной ошибки транспортного слоя.
type cartErrorResponse struct {
	Error string `json:"error"`
}

// cartSuccessResponse отображает дефолтную структуру успешного текстового ответа.
type cartSuccessResponse struct {
	Message string `json:"message"`
}

// setupCartTestDeps настраивает тестовые зависимости для обработки запросов корзины.
func setupCartTestDeps(t *testing.T) (*handler.CartHandler, *mockCartService) {
	t.Helper()

	log, err := logger.New("local", "Stdout")
	require.NoError(t, err, "Не удалось инициализировать тестовый логгер")

	mockService := new(mockCartService)
	cartHandler := handler.NewCartHandler(mockService, log)

	return cartHandler, mockService
}

// TestCart_Unauthorized проверяет единое бизнес-требование безопасности:
// все эндпоинты корзины должны возвращать 401 Unauthorized, если в контексте отсутствует ID пользователя.
func TestCart_Unauthorized(t *testing.T) {
	h, _ := setupCartTestDeps(t)

	tests := []struct {
		name    string
		method  string
		url     string
		handler http.HandlerFunc
		body    any
	}{
		{
			name:    "Добавление в корзину без авторизации",
			method:  http.MethodPost,
			url:     "/api/v1/cart",
			handler: h.Add,
			body:    model.AddToCartInput{ProductID: 1, Quantity: 2},
		},
		{
			name:    "Получение корзины без авторизации",
			method:  http.MethodGet,
			url:     "/api/v1/cart",
			handler: h.GetByID,
			body:    nil,
		},
		{
			name:    "Удаление из корзины без авторизации",
			method:  http.MethodDelete,
			url:     "/api/v1/cart/1",
			handler: h.Delete,
			body:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			var buf bytes.Buffer
			if tt.body != nil {
				err := json.NewEncoder(&buf).Encode(tt.body)
				require.NoError(t, err)
			}

			// Намеренно НЕ добавляем userID в контекст запроса
			req := httptest.NewRequest(tt.method, tt.url, &buf)
			rr := httptest.NewRecorder()

			// Act
			tt.handler(rr, req)

			// Assert
			assert.Equal(t, http.StatusUnauthorized, rr.Code)

			var resp cartErrorResponse
			err := json.Unmarshal(rr.Body.Bytes(), &resp)
			require.NoError(t, err)
			assert.Equal(t, "пользователь не аутентифицирован", resp.Error)
		})
	}
}

// TestCartAdd_Success проверяет успешное добавление товара в корзину.
func TestCartAdd_Success(t *testing.T) {
	// Arrange
	h, s := setupCartTestDeps(t)
	userID := 42
	input := model.AddToCartInput{
		ProductID: 101,
		Quantity:  3,
	}

	s.On("Add", mock.Anything, userID, input).Return(nil)

	body, err := json.Marshal(input)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/cart", bytes.NewReader(body))
	req = req.WithContext(handler.ContextWithUserID(req.Context(), userID))
	rr := httptest.NewRecorder()

	// Act
	h.Add(rr, req)

	// Assert
	assert.Equal(t, http.StatusOK, rr.Code)

	var resp cartSuccessResponse
	err = json.Unmarshal(rr.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "товар успешно добавлен в корзину", resp.Message)
	s.AssertExpectations(t)
}

// TestCartAdd_ValidationErrors верифицирует работу тегов валидации (gt=0) структуры AddToCartInput.
func TestCartAdd_ValidationErrors(t *testing.T) {
	tests := []struct {
		name  string
		input any
		msg   string
	}{
		{
			name:  "Некорректный JSON",
			input: "{invalid-json}",
			msg:   "некорректное тело запроса",
		},
		{
			name:  "ProductID равен нулю",
			input: model.AddToCartInput{ProductID: 0, Quantity: 5},
			msg:   "ошибка валидации данных",
		},
		{
			name:  "Количество товара отрицательное",
			input: model.AddToCartInput{ProductID: 10, Quantity: -1},
			msg:   "ошибка валидации данных",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			h, _ := setupCartTestDeps(t)
			userID := 42

			var body []byte
			var err error
			if str, ok := tt.input.(string); ok {
				body = []byte(str)
			} else {
				body, err = json.Marshal(tt.input)
				require.NoError(t, err)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/v1/cart", bytes.NewReader(body))
			req = req.WithContext(handler.ContextWithUserID(req.Context(), userID))
			rr := httptest.NewRecorder()

			// Act
			h.Add(rr, req)

			// Assert
			assert.Equal(t, http.StatusBadRequest, rr.Code)

			var resp cartErrorResponse
			err = json.Unmarshal(rr.Body.Bytes(), &resp)
			require.NoError(t, err)
			assert.Equal(t, tt.msg, resp.Error)
		})
	}
}

// TestCartAdd_ProductNotFound проверяет обработку ситуации, когда добавляемого товара нет в каталоге.
func TestCartAdd_ProductNotFound(t *testing.T) {
	// Arrange
	h, s := setupCartTestDeps(t)
	userID := 42
	input := model.AddToCartInput{ProductID: 999, Quantity: 1}

	s.On("Add", mock.Anything, userID, input).Return(model.ErrProductNotFound)

	body, err := json.Marshal(input)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/cart", bytes.NewReader(body))
	req = req.WithContext(handler.ContextWithUserID(req.Context(), userID))
	rr := httptest.NewRecorder()

	// Act
	h.Add(rr, req)

	// Assert
	assert.Equal(t, http.StatusNotFound, rr.Code)

	var resp cartErrorResponse
	err = json.Unmarshal(rr.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "товар не найден в каталоге", resp.Error)
}

// TestCartGet_Success проверяет успешное чтение развернутого содержимого корзины текущего пользователя.
func TestCartGet_Success(t *testing.T) {
	// Arrange
	h, s := setupCartTestDeps(t)
	userID := 42

	expectedItems := []*model.CartOutputItem{
		{
			CartItemID:  1,
			ProductID:   10,
			ProductName: "Кофеварка",
			Price:       500000,
			Quantity:    1,
			TotalPrice:  500000,
		},
		{
			CartItemID:  2,
			ProductID:   20,
			ProductName: "Кофейные зерна 1кг",
			Price:       120000,
			Quantity:    2,
			TotalPrice:  240000,
		},
	}

	s.On("GetByUserID", mock.Anything, userID).Return(expectedItems, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cart", nil)
	req = req.WithContext(handler.ContextWithUserID(req.Context(), userID))
	rr := httptest.NewRecorder()

	// Act
	h.GetByID(rr, req)

	// Assert
	assert.Equal(t, http.StatusOK, rr.Code)

	var actualItems []*model.CartOutputItem
	err := json.Unmarshal(rr.Body.Bytes(), &actualItems)
	require.NoError(t, err)
	require.Len(t, actualItems, 2)
	assert.Equal(t, expectedItems[0].ProductName, actualItems[0].ProductName)
	assert.Equal(t, expectedItems[1].TotalPrice, actualItems[1].TotalPrice)
}

// TestCartDelete_Success проверяет штатное удаление товарной позиции из корзины по ID.
func TestCartDelete_Success(t *testing.T) {
	// Arrange
	h, s := setupCartTestDeps(t)
	userID := 42
	productID := 10

	s.On("Delete", mock.Anything, userID, productID).Return(nil)

	// Используем chi.Router для симуляции извлечения параметров из URL-пути {id}
	router := chi.NewRouter()
	router.Delete("/api/v1/cart/{id}", h.Delete)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/cart/10", nil)
	req = req.WithContext(handler.ContextWithUserID(req.Context(), userID))
	rr := httptest.NewRecorder()

	// Act
	router.ServeHTTP(rr, req)

	// Assert
	assert.Equal(t, http.StatusOK, rr.Code)

	var resp cartSuccessResponse
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "товар успешно удален из корзины", resp.Message)
	s.AssertExpectations(t)
}

// TestCartDelete_InvalidID проверяет защиту эндпоинта от передачи невалидных или отрицательных ID в URL.
func TestCartDelete_InvalidID(t *testing.T) {
	tests := []struct {
		name      string
		productID string
	}{
		{name: "Передача строки вместо числа", productID: "abc"},
		{name: "Передача отрицательного ID", productID: "-5"},
		{name: "Передача нулевого ID", productID: "0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			h, _ := setupCartTestDeps(t)
			userID := 42

			router := chi.NewRouter()
			router.Delete("/api/v1/cart/{id}", h.Delete)

			req := httptest.NewRequest(http.MethodDelete, "/api/v1/cart/"+tt.productID, nil)
			req = req.WithContext(handler.ContextWithUserID(req.Context(), userID))
			rr := httptest.NewRecorder()

			// Act
			router.ServeHTTP(rr, req)

			// Assert
			assert.Equal(t, http.StatusBadRequest, rr.Code)

			var resp cartErrorResponse
			err := json.Unmarshal(rr.Body.Bytes(), &resp)
			require.NoError(t, err)
			assert.Equal(t, "некорректный ID товара", resp.Error)
		})
	}
}

// TestCartDelete_ItemNotFound проверяет обработку ошибки, когда удаляемого товара не было в корзине пользователя.
func TestCartDelete_ItemNotFound(t *testing.T) {
	// Arrange
	h, s := setupCartTestDeps(t)
	userID := 42
	productID := 55

	s.On("Delete", mock.Anything, userID, productID).Return(model.ErrCartItemNotFound)

	router := chi.NewRouter()
	router.Delete("/api/v1/cart/{id}", h.Delete)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/cart/55", nil)
	req = req.WithContext(handler.ContextWithUserID(req.Context(), userID))
	rr := httptest.NewRecorder()

	// Act
	router.ServeHTTP(rr, req)

	// Assert
	assert.Equal(t, http.StatusNotFound, rr.Code)

	var resp cartErrorResponse
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "товар в корзине не найден", resp.Error)
}

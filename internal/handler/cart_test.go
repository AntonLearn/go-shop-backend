// Package handler_test содержит модульные тесты для транспортного слоя (HTTP-хендлеров).
// Тесты используют моки для изоляции от слоя бизнес-логики и проверяют корректность
// обработки входящих запросов, валидацию DTO и формирование HTTP-ответов.
package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/antonlearn/go-shop-backend/internal/handler"
	"github.com/antonlearn/go-shop-backend/internal/model"
	"github.com/antonlearn/go-shop-backend/pkg/logger"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// --- MOCK ---

// MockCartService — заглушка (mock) для интерфейса CartServiceInterface.
// Используется для имитации поведения бизнес-логики корзины пользователей в тестах.
type MockCartService struct {
	mock.Mock
}

// Add имитирует добавление или обеспечение количества товара в корзине пользователя.
func (m *MockCartService) Add(ctx context.Context, userID int, input model.AddToCartInput) error {
	return m.Called(mock.Anything, userID, input).Error(0)
}

// GetByUserID имитирует выгрузку содержимого корзины с развернутыми агрегированными данными.
func (m *MockCartService) GetByUserID(ctx context.Context, userID int) ([]*model.CartOutputItem, error) {
	args := m.Called(mock.Anything, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.CartOutputItem), args.Error(1)
}

// Delete имитирует удаление конкретной позиции товара из корзины.
func (m *MockCartService) Delete(ctx context.Context, userID int, productID int) error {
	return m.Called(mock.Anything, userID, productID).Error(0)
}

// --- SUITE ---

type CartHandlerTestSuite struct {
	suite.Suite
	mockSvc *MockCartService
	h       *handler.CartHandler
}

func (s *CartHandlerTestSuite) SetupTest() {
	log, _ := logger.New("Console", "DEBUG")
	s.mockSvc = new(MockCartService)
	s.h = handler.NewCartHandler(s.mockSvc, log)
}

// withAuthenticatedUser — вспомогательный хелпер для тестов.
// Инжектирует идентификатор пользователя в контекст запроса.
func (s *CartHandlerTestSuite) withAuthenticatedUser(r *http.Request, userID int) *http.Request {
	type contextKey string
	return r.WithContext(context.WithValue(r.Context(), contextKey("user_id"), userID))
}

// withChiParam — хелпер для внедрения параметров пути Chi в контекст запроса.
func (s *CartHandlerTestSuite) withChiParam(r *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

// ==========================================
// ТЕСТЫ: Add (Добавление товара в корзину)
// ==========================================

func (s *CartHandlerTestSuite) TestAdd() {
	userID := 10
	input := model.AddToCartInput{ProductID: 1, Quantity: 2}

	s.Run("Success", func() {
		s.mockSvc.On("Add", mock.Anything, userID, input).Return(nil).Once()

		body, _ := json.Marshal(input)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/cart", bytes.NewBuffer(body))
		req = s.withAuthenticatedUser(req, userID)
		w := httptest.NewRecorder()

		s.h.Add(w, req)

		s.Equal(http.StatusOK, w.Code)
		s.mockSvc.AssertExpectations(s.T())
	})

	s.Run("Unauthorized", func() {
		body, _ := json.Marshal(input)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/cart", bytes.NewBuffer(body))
		// ID пользователя намеренно не передается в контекст
		w := httptest.NewRecorder()

		s.h.Add(w, req)

		s.Equal(http.StatusUnauthorized, w.Code)
	})

	s.Run("Invalid JSON", func() {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/cart", bytes.NewBufferString("{bad-json"))
		req = s.withAuthenticatedUser(req, userID)
		w := httptest.NewRecorder()

		s.h.Add(w, req)

		s.Equal(http.StatusBadRequest, w.Code)
	})

	s.Run("Validation Error", func() {
		// Передаем пустую структуру, которая завалит теги валидатора (например, если ProductID/Quantity обязательны)
		invalidInput := model.AddToCartInput{}

		body, _ := json.Marshal(invalidInput)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/cart", bytes.NewBuffer(body))
		req = s.withAuthenticatedUser(req, userID)
		w := httptest.NewRecorder()

		s.h.Add(w, req)

		s.Equal(http.StatusBadRequest, w.Code)
	})

	s.Run("Product Not Found", func() {
		s.mockSvc.On("Add", mock.Anything, userID, input).Return(model.ErrProductNotFound).Once()

		body, _ := json.Marshal(input)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/cart", bytes.NewBuffer(body))
		req = s.withAuthenticatedUser(req, userID)
		w := httptest.NewRecorder()

		s.h.Add(w, req)

		s.Equal(http.StatusNotFound, w.Code)
	})

	s.Run("Internal Server Error", func() {
		s.mockSvc.On("Add", mock.Anything, userID, input).Return(errors.New("db failure")).Once()

		body, _ := json.Marshal(input)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/cart", bytes.NewBuffer(body))
		req = s.withAuthenticatedUser(req, userID)
		w := httptest.NewRecorder()

		s.h.Add(w, req)

		s.Equal(http.StatusInternalServerError, w.Code)
	})
}

// ==========================================
// ТЕСТЫ: GetByID (Получение содержимого корзины)
// ==========================================

func (s *CartHandlerTestSuite) TestGetContent() {
	userID := 10
	expectedItems := []*model.CartOutputItem{
		{CartItemID: 1, ProductID: 5, ProductName: "Кофеварка", Price: 500000, Quantity: 1, TotalPrice: 500000},
	}

	s.Run("Success", func() {
		s.mockSvc.On("GetByUserID", mock.Anything, userID).Return(expectedItems, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/cart", nil)
		req = s.withAuthenticatedUser(req, userID)
		w := httptest.NewRecorder()

		s.h.GetByID(w, req)

		s.Equal(http.StatusOK, w.Code)
		s.mockSvc.AssertExpectations(s.T())
	})

	s.Run("Unauthorized", func() {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/cart", nil)
		w := httptest.NewRecorder()

		s.h.GetByID(w, req)

		s.Equal(http.StatusUnauthorized, w.Code)
	})

	s.Run("Internal Server Error", func() {
		s.mockSvc.On("GetByUserID", mock.Anything, userID).Return(nil, errors.New("cache down")).Once()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/cart", nil)
		req = s.withAuthenticatedUser(req, userID)
		w := httptest.NewRecorder()

		s.h.GetByID(w, req)

		s.Equal(http.StatusInternalServerError, w.Code)
	})
}

// ==========================================
// ТЕСТЫ: Delete (Удаление товара из корзины)
// ==========================================

func (s *CartHandlerTestSuite) TestDelete() {
	userID := 10
	productID := 5

	s.Run("Success", func() {
		s.mockSvc.On("Delete", mock.Anything, userID, productID).Return(nil).Once()

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/cart/5", nil)
		req = s.withAuthenticatedUser(req, userID)
		req = s.withChiParam(req, "id", "5")
		w := httptest.NewRecorder()

		s.h.Delete(w, req)

		s.Equal(http.StatusOK, w.Code)
		s.mockSvc.AssertExpectations(s.T())
	})

	s.Run("Unauthorized", func() {
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/cart/5", nil)
		req = s.withChiParam(req, "id", "5")
		w := httptest.NewRecorder()

		s.h.Delete(w, req)

		s.Equal(http.StatusUnauthorized, w.Code)
	})

	s.Run("Invalid ID - Non-numeric", func() {
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/cart/abc", nil)
		req = s.withAuthenticatedUser(req, userID)
		req = s.withChiParam(req, "id", "abc")
		w := httptest.NewRecorder()

		s.h.Delete(w, req)

		s.Equal(http.StatusBadRequest, w.Code)
	})

	s.Run("Invalid ID - Negative Value", func() {
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/cart/-1", nil)
		req = s.withAuthenticatedUser(req, userID)
		req = s.withChiParam(req, "id", "-1")
		w := httptest.NewRecorder()

		s.h.Delete(w, req)

		s.Equal(http.StatusBadRequest, w.Code)
	})

	s.Run("Cart Item Not Found", func() {
		s.mockSvc.On("Delete", mock.Anything, userID, productID).Return(model.ErrCartItemNotFound).Once()

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/cart/5", nil)
		req = s.withAuthenticatedUser(req, userID)
		req = s.withChiParam(req, "id", "5")
		w := httptest.NewRecorder()

		s.h.Delete(w, req)

		s.Equal(http.StatusNotFound, w.Code)
	})

	s.Run("Internal Server Error", func() {
		s.mockSvc.On("Delete", mock.Anything, userID, productID).Return(errors.New("transaction error")).Once()

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/cart/5", nil)
		req = s.withAuthenticatedUser(req, userID)
		req = s.withChiParam(req, "id", "5")
		w := httptest.NewRecorder()

		s.h.Delete(w, req)

		s.Equal(http.StatusInternalServerError, w.Code)
	})
}

func TestCartHandlerSuite(t *testing.T) {
	suite.Run(t, new(CartHandlerTestSuite))
}

// Package handler реализует транспортный слой приложения (HTTP).
// Файл order_handler_test.go содержит модульные тесты для обработчиков заказов,
// изолируя бизнес-логику с помощью моков и проверяя все ветки ответов.
package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/antonlearn/go-shop-backend/internal/model"
	"github.com/antonlearn/go-shop-backend/pkg/logger"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// --- MOCK ---

// MockOrderService — реализация мока для интерфейса OrderServiceInterface.
type MockOrderService struct {
	mock.Mock
}

func (m *MockOrderService) Create(ctx context.Context, userID int) (*model.Order, error) {
	args := m.Called(mock.Anything, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Order), args.Error(1)
}

func (m *MockOrderService) GetByID(ctx context.Context, userID, orderID int) (*model.OrderOutput, error) {
	args := m.Called(mock.Anything, userID, orderID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.OrderOutput), args.Error(1)
}

func (m *MockOrderService) GetByUserID(ctx context.Context, userID int) ([]*model.Order, error) {
	args := m.Called(mock.Anything, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Order), args.Error(1)
}

// --- SUITE ---

type OrderHandlerTestSuite struct {
	suite.Suite
	mockSvc *MockOrderService
	h       *OrderHandler
}

func (s *OrderHandlerTestSuite) SetupTest() {
	log, _ := logger.New("Console", "DEBUG")
	s.mockSvc = new(MockOrderService)
	s.h = NewOrderHandler(s.mockSvc, log)
}

// withAuthenticatedUser инжектирует ID пользователя в контекст запроса.
func (s *OrderHandlerTestSuite) withAuthenticatedUser(r *http.Request, userID int) *http.Request {
	return r.WithContext(ContextWithUserID(r.Context(), userID))
}

// withChiParam внедряет параметры пути Chi роутера в контекст запроса.
func (s *OrderHandlerTestSuite) withChiParam(r *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

// ==========================================
// ТЕСТЫ: Create (Оформление заказа)
// ==========================================

func (s *OrderHandlerTestSuite) TestCreate() {
	userID := 1

	s.Run("Success", func() {
		s.mockSvc.On("Create", mock.Anything, userID).Return(&model.Order{ID: 100}, nil).Once()

		req := httptest.NewRequest(http.MethodPost, "/api/v1/orders", nil)
		req = s.withAuthenticatedUser(req, userID)
		rr := httptest.NewRecorder()

		s.h.Create(rr, req)

		s.Equal(http.StatusCreated, rr.Code)
		s.mockSvc.AssertExpectations(s.T())
	})

	s.Run("Unauthorized", func() {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/orders", nil)
		rr := httptest.NewRecorder()

		s.h.Create(rr, req)

		s.Equal(http.StatusUnauthorized, rr.Code)
	})

	s.Run("Cart Is Empty", func() {
		s.mockSvc.On("Create", mock.Anything, userID).Return(nil, model.ErrCartIsEmpty).Once()

		req := httptest.NewRequest(http.MethodPost, "/api/v1/orders", nil)
		req = s.withAuthenticatedUser(req, userID)
		rr := httptest.NewRecorder()

		s.h.Create(rr, req)

		s.Equal(http.StatusBadRequest, rr.Code)
		s.Contains(rr.Body.String(), "корзина пуста")
	})

	s.Run("Internal Server Error", func() {
		s.mockSvc.On("Create", mock.Anything, userID).Return(nil, errors.New("db crash")).Once()

		req := httptest.NewRequest(http.MethodPost, "/api/v1/orders", nil)
		req = s.withAuthenticatedUser(req, userID)
		rr := httptest.NewRecorder()

		s.h.Create(rr, req)

		s.Equal(http.StatusInternalServerError, rr.Code)
		s.Contains(rr.Body.String(), "не удалось оформить заказ")
	})
}

// ==========================================
// ТЕСТЫ: GetByID (Детализация заказа)
// ==========================================

func (s *OrderHandlerTestSuite) TestGetByID() {
	userID := 1
	orderID := 10

	s.Run("Success", func() {
		expectedOrder := &model.OrderOutput{ID: orderID, TotalPrice: 500}
		s.mockSvc.On("GetByID", mock.Anything, userID, orderID).Return(expectedOrder, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/orders/10", nil)
		req = s.withAuthenticatedUser(req, userID)
		req = s.withChiParam(req, "id", "10")
		rr := httptest.NewRecorder()

		s.h.GetByID(rr, req)

		s.Equal(http.StatusOK, rr.Code)
		s.mockSvc.AssertExpectations(s.T())
	})

	s.Run("Unauthorized", func() {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/orders/10", nil)
		req = s.withChiParam(req, "id", "10")
		rr := httptest.NewRecorder()

		s.h.GetByID(rr, req)

		s.Equal(http.StatusUnauthorized, rr.Code)
	})

	s.Run("Invalid ID - Non-numeric", func() {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/orders/abc", nil)
		req = s.withAuthenticatedUser(req, userID)
		req = s.withChiParam(req, "id", "abc")
		rr := httptest.NewRecorder()

		s.h.GetByID(rr, req)

		s.Equal(http.StatusBadRequest, rr.Code)
		s.Contains(rr.Body.String(), "некорректный ID заказа")
	})

	s.Run("Invalid ID - Zero Value", func() {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/orders/0", nil)
		req = s.withAuthenticatedUser(req, userID)
		req = s.withChiParam(req, "id", "0")
		rr := httptest.NewRecorder()

		s.h.GetByID(rr, req)

		s.Equal(http.StatusBadRequest, rr.Code)
	})

	s.Run("Order Not Found", func() {
		s.mockSvc.On("GetByID", mock.Anything, userID, 999).Return(nil, model.ErrOrderNotFound).Once()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/orders/999", nil)
		req = s.withAuthenticatedUser(req, userID)
		req = s.withChiParam(req, "id", "999")
		rr := httptest.NewRecorder()

		s.h.GetByID(rr, req)

		s.Equal(http.StatusNotFound, rr.Code)
		s.Contains(rr.Body.String(), "заказ не найден")
	})

	s.Run("Internal Server Error", func() {
		s.mockSvc.On("GetByID", mock.Anything, userID, orderID).Return(nil, errors.New("network error")).Once()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/orders/10", nil)
		req = s.withAuthenticatedUser(req, userID)
		req = s.withChiParam(req, "id", "10")
		rr := httptest.NewRecorder()

		s.h.GetByID(rr, req)

		s.Equal(http.StatusInternalServerError, rr.Code)
	})
}

// ==========================================
// ТЕСТЫ: GetByUserID (История заказов)
// ==========================================

func (s *OrderHandlerTestSuite) TestGetByUserID() {
	userID := 1

	s.Run("Success - With Orders", func() {
		s.mockSvc.On("GetByUserID", mock.Anything, userID).Return([]*model.Order{{ID: 1}}, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/orders", nil)
		req = s.withAuthenticatedUser(req, userID)
		rr := httptest.NewRecorder()

		s.h.GetByUserID(rr, req)

		s.Equal(http.StatusOK, rr.Code)
		s.mockSvc.AssertExpectations(s.T())
	})

	s.Run("Success - Empty History (Returns empty array instead of null)", func() {
		// Имитируем ситуацию, когда сервис возвращает nil-слайс
		s.mockSvc.On("GetByUserID", mock.Anything, userID).Return(nil, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/orders", nil)
		req = s.withAuthenticatedUser(req, userID)
		rr := httptest.NewRecorder()

		s.h.GetByUserID(rr, req)

		s.Equal(http.StatusOK, rr.Code)
		// Проверяем, что в JSON ушел инициализированный пустой массив `[]`
		s.JSONEq("[]", rr.Body.String())
		s.mockSvc.AssertExpectations(s.T())
	})

	s.Run("Unauthorized", func() {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/orders", nil)
		rr := httptest.NewRecorder()

		s.h.GetByUserID(rr, req)

		s.Equal(http.StatusUnauthorized, rr.Code)
	})

	s.Run("Internal Server Error", func() {
		s.mockSvc.On("GetByUserID", mock.Anything, userID).Return(nil, errors.New("read error")).Once()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/orders", nil)
		req = s.withAuthenticatedUser(req, userID)
		rr := httptest.NewRecorder()

		s.h.GetByUserID(rr, req)

		s.Equal(http.StatusInternalServerError, rr.Code)
	})
}

func TestOrderHandlerSuite(t *testing.T) {
	suite.Run(t, new(OrderHandlerTestSuite))
}

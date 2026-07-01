// Package service_test содержит unit-тесты для сервиса заказов.
// Тесты покрывают логику оформления заказов, получения деталей заказа и истории покупок,
// включая обработку бизнес-ошибок и сбоев репозитория.
package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/antonlearn/go-shop-backend/internal/model"
	"github.com/antonlearn/go-shop-backend/internal/service"
	"github.com/antonlearn/go-shop-backend/pkg/logger"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// --- MOCKS (Тестовые двойники зависимостей) ---

// MockOrderRepository имитирует слой хранения заказов.
type MockOrderRepository struct{ mock.Mock }

func (m *MockOrderRepository) Create(ctx context.Context, userID int) (*model.Order, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Order), args.Error(1)
}

func (m *MockOrderRepository) GetByID(ctx context.Context, orderID, userID int) (*model.OrderOutput, error) {
	args := m.Called(ctx, orderID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.OrderOutput), args.Error(1)
}

func (m *MockOrderRepository) GetByUserID(ctx context.Context, userID int) ([]*model.Order, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Order), args.Error(1)
}

// --- SUITE ---

// OrderServiceTestSuite организует набор тестов для OrderService.
type OrderServiceTestSuite struct {
	suite.Suite
	service *service.OrderService
	repo    *MockOrderRepository
	ctx     context.Context
}

// SetupTest инициализирует сервис и моки перед каждым запуском теста.
func (s *OrderServiceTestSuite) SetupTest() {
	s.repo = new(MockOrderRepository)
	log, _ := logger.New("Console", "INFO")

	s.service = service.NewOrderService(s.repo, log)
	s.ctx = context.Background()
}

// --- CREATE TESTS ---

// TestCreate_Success проверяет штатное оформление заказа.
func (s *OrderServiceTestSuite) TestCreate_Success() {
	expectedOrder := &model.Order{ID: 777, UserID: 1, TotalPrice: 3500, Status: "created"}
	s.repo.On("Create", s.ctx, 1).Return(expectedOrder, nil)

	order, err := s.service.Create(s.ctx, 1)
	s.NoError(err)
	s.NotNil(order)
	s.Equal(777, order.ID)
}

// TestCreate_InvalidUserID проверяет валидацию идентификатора пользователя.
func (s *OrderServiceTestSuite) TestCreate_InvalidUserID() {
	_, err := s.service.Create(s.ctx, 0)
	s.Error(err)
	s.Contains(err.Error(), "некорректный идентификатор пользователя")
}

// TestCreate_EmptyCart проверяет специфическую бизнес-ошибку при пустой корзине.
func (s *OrderServiceTestSuite) TestCreate_EmptyCart() {
	s.repo.On("Create", s.ctx, 1).Return(nil, errors.New("корзина пуста"))

	_, err := s.service.Create(s.ctx, 1)
	s.Error(err)
	s.Contains(err.Error(), "невозможно оформить заказ: корзина пуста")
}

// TestCreate_RepositoryError проверяет ошибку при сбое транзакции в базе данных.
func (s *OrderServiceTestSuite) TestCreate_RepositoryError() {
	s.repo.On("Create", s.ctx, 1).Return(nil, errors.New("db connection lost"))

	_, err := s.service.Create(s.ctx, 1)
	s.Error(err)
	s.Contains(err.Error(), "failed to place order")
}

// --- GET BY ID TESTS ---

// TestGetByID_Success проверяет успешное получение деталей заказа.
func (s *OrderServiceTestSuite) TestGetByID_Success() {
	expectedOutput := &model.OrderOutput{
		ID:         777,
		TotalPrice: 1200,
		Items:      []*model.OrderItemDetail{{ProductID: 5, ProductName: "Коврик", Quantity: 1, Price: 1200}},
	}
	s.repo.On("GetByID", s.ctx, 777, 1).Return(expectedOutput, nil)

	output, err := s.service.GetByID(s.ctx, 777, 1)
	s.NoError(err)
	s.Equal(777, output.ID)
}

// TestGetByID_NotFound проверяет ошибку, когда заказ не найден.
func (s *OrderServiceTestSuite) TestGetByID_NotFound() {
	s.repo.On("GetByID", s.ctx, 999, 1).Return(nil, model.ErrOrderNotFound)

	_, err := s.service.GetByID(s.ctx, 999, 1)
	s.ErrorIs(err, model.ErrOrderNotFound)
}

// TestGetByID_RepositoryError проверяет ошибку при сбое базы данных.
func (s *OrderServiceTestSuite) TestGetByID_RepositoryError() {
	s.repo.On("GetByID", s.ctx, 777, 1).Return(nil, errors.New("db error"))

	_, err := s.service.GetByID(s.ctx, 777, 1)
	s.Error(err)
	s.Contains(err.Error(), "failed to retrieve order")
}

// --- GET BY USER ID TESTS ---

// TestGetByUserID_Success проверяет успешное получение истории заказов.
func (s *OrderServiceTestSuite) TestGetByUserID_Success() {
	expectedOrders := []*model.Order{
		{ID: 10, UserID: 1, TotalPrice: 500},
		{ID: 11, UserID: 1, TotalPrice: 1500},
	}
	s.repo.On("GetByUserID", s.ctx, 1).Return(expectedOrders, nil)

	orders, err := s.service.GetByUserID(s.ctx, 1)
	s.NoError(err)
	s.Len(orders, 2)
}

// TestGetByUserID_RepositoryError проверяет ошибку при сбое базы данных.
func (s *OrderServiceTestSuite) TestGetByUserID_RepositoryError() {
	s.repo.On("GetByUserID", s.ctx, 1).Return(nil, errors.New("db error"))

	orders, err := s.service.GetByUserID(s.ctx, 1)
	s.Error(err)
	s.Nil(orders)
	s.Contains(err.Error(), "failed to retrieve user order history")
}

// TestOrderServiceSuite запускает весь набор тестов для OrderService.
func TestOrderServiceSuite(t *testing.T) {
	suite.Run(t, new(OrderServiceTestSuite))
}

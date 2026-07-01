// Package service_test содержит unit-тесты для сервиса корзины.
// Тесты проверяют бизнес-логику добавления, получения и удаления товаров,
// а также корректную обработку ошибок репозитория и валидацию данных.
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

// MockCartRepository имитирует слой хранения корзины.
type MockCartRepository struct{ mock.Mock }

func (m *MockCartRepository) Add(ctx context.Context, item *model.CartItem) error {
	return m.Called(ctx, item).Error(0)
}
func (m *MockCartRepository) GetByUserID(ctx context.Context, userID int) ([]*model.CartOutputItem, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.CartOutputItem), args.Error(1)
}
func (m *MockCartRepository) Delete(ctx context.Context, userID, productID int) error {
	return m.Called(ctx, userID, productID).Error(0)
}

// MockCartProductRepository имитирует слой получения данных о товаре.
type MockCartProductRepository struct{ mock.Mock }

func (m *MockCartProductRepository) GetByID(ctx context.Context, id int) (*model.Product, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Product), args.Error(1)
}

// --- SUITE ---

// CartServiceTestSuite организует набор тестов для CartService.
type CartServiceTestSuite struct {
	suite.Suite
	service  *service.CartService
	cartRepo *MockCartRepository
	prodRepo *MockCartProductRepository
	ctx      context.Context
}

// SetupTest инициализирует сервисы и моки перед каждым запуском теста.
func (s *CartServiceTestSuite) SetupTest() {
	s.cartRepo = new(MockCartRepository)
	s.prodRepo = new(MockCartProductRepository)
	log, _ := logger.New("Console", "INFO")

	s.service = service.NewCartService(s.cartRepo, s.prodRepo, log)
	s.ctx = context.Background()
}

// --- ADD TESTS ---

// TestAdd_Success проверяет штатный сценарий добавления товара.
func (s *CartServiceTestSuite) TestAdd_Success() {
	input := model.AddToCartInput{ProductID: 10, Quantity: 2}
	product := &model.Product{ID: 10, Name: "Товар", Stock: 5}

	s.prodRepo.On("GetByID", s.ctx, 10).Return(product, nil)
	s.cartRepo.On("Add", s.ctx, &model.CartItem{UserID: 1, ProductID: 10, Quantity: 2}).Return(nil)

	err := s.service.Add(s.ctx, 1, input)
	s.NoError(err)
}

// TestAdd_InvalidQuantity проверяет валидацию: количество товара должно быть > 0.
func (s *CartServiceTestSuite) TestAdd_InvalidQuantity() {
	input := model.AddToCartInput{ProductID: 10, Quantity: 0}
	err := s.service.Add(s.ctx, 1, input)
	s.Error(err)
	s.Contains(err.Error(), "количество товара должно быть строго больше нуля")
}

// TestAdd_ProductNotFound проверяет ошибку, если товара не существует в БД.
func (s *CartServiceTestSuite) TestAdd_ProductNotFound() {
	input := model.AddToCartInput{ProductID: 404, Quantity: 1}
	s.prodRepo.On("GetByID", s.ctx, 404).Return(nil, model.ErrProductNotFound)

	err := s.service.Add(s.ctx, 1, input)
	s.ErrorIs(err, model.ErrProductNotFound)
}

// TestAdd_InsufficientStock проверяет ошибку при попытке добавить товара больше, чем есть на складе.
func (s *CartServiceTestSuite) TestAdd_InsufficientStock() {
	input := model.AddToCartInput{ProductID: 10, Quantity: 10}
	product := &model.Product{ID: 10, Name: "Товар", Stock: 3}

	s.prodRepo.On("GetByID", s.ctx, 10).Return(product, nil)

	err := s.service.Add(s.ctx, 1, input)
	s.Error(err)
	s.Contains(err.Error(), "недостаточно товара на складе")
}

// TestAdd_RepositoryError проверяет ошибку при сбое базы данных во время добавления.
func (s *CartServiceTestSuite) TestAdd_RepositoryError() {
	input := model.AddToCartInput{ProductID: 10, Quantity: 2}
	product := &model.Product{ID: 10, Stock: 5}

	s.prodRepo.On("GetByID", s.ctx, 10).Return(product, nil)
	s.cartRepo.On("Add", s.ctx, mock.Anything).Return(errors.New("db error"))

	err := s.service.Add(s.ctx, 1, input)
	s.Error(err)
	s.Contains(err.Error(), "failed to add item to cart")
}

// --- GET BY USER ID TESTS ---

// TestGetByUserID_Success проверяет успешное получение списка товаров корзины.
func (s *CartServiceTestSuite) TestGetByUserID_Success() {
	expectedItems := []*model.CartOutputItem{
		{ProductID: 1, ProductName: "Товар 1", Quantity: 2, Price: 500},
	}
	s.cartRepo.On("GetByUserID", s.ctx, 1).Return(expectedItems, nil)

	items, err := s.service.GetByUserID(s.ctx, 1)
	s.NoError(err)
	s.Len(items, 1)
	s.Equal("Товар 1", items[0].ProductName)
}

// TestGetByUserID_RepositoryError проверяет ошибку при сбое базы данных.
func (s *CartServiceTestSuite) TestGetByUserID_RepositoryError() {
	s.cartRepo.On("GetByUserID", s.ctx, 1).Return(nil, errors.New("db error"))

	items, err := s.service.GetByUserID(s.ctx, 1)
	s.Error(err)
	s.Nil(items)
}

// --- DELETE TESTS ---

// TestDelete_Success проверяет успешное удаление товара.
func (s *CartServiceTestSuite) TestDelete_Success() {
	s.cartRepo.On("Delete", s.ctx, 1, 10).Return(nil)
	s.NoError(s.service.Delete(s.ctx, 1, 10))
}

// TestDelete_ItemNotFound проверяет ошибку, если удаляемый товар не найден в корзине.
func (s *CartServiceTestSuite) TestDelete_ItemNotFound() {
	s.cartRepo.On("Delete", s.ctx, 1, 99).Return(model.ErrCartItemNotFound)
	s.ErrorIs(s.service.Delete(s.ctx, 1, 99), model.ErrCartItemNotFound)
}

// TestDelete_RepositoryError проверяет ошибку при сбое базы данных.
func (s *CartServiceTestSuite) TestDelete_RepositoryError() {
	s.cartRepo.On("Delete", s.ctx, 1, 10).Return(errors.New("db error"))
	s.Error(s.service.Delete(s.ctx, 1, 10))
}

// TestCartServiceSuite запускает весь набор тестов для CartService.
func TestCartServiceSuite(t *testing.T) {
	suite.Run(t, new(CartServiceTestSuite))
}

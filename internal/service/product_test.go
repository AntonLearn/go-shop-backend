// Package service_test содержит unit-тесты для сервиса каталога товаров.
// Тесты проверяют бизнес-логику CRUD-операций, валидацию данных и обработку
// ошибок репозитория.
package service_test

import (
	"context"
	"testing"

	"github.com/antonlearn/go-shop-backend/internal/model"
	"github.com/antonlearn/go-shop-backend/internal/service"
	"github.com/antonlearn/go-shop-backend/pkg/logger"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// --- MOCKS (Тестовые двойники зависимостей) ---

// MockProductRepository имитирует слой хранения товаров.
type MockProductRepository struct{ mock.Mock }

func (m *MockProductRepository) Create(ctx context.Context, product *model.Product) error {
	product.ID = 100 // имитируем автоинкремент БД
	return m.Called(ctx, product).Error(0)
}
func (m *MockProductRepository) GetByID(ctx context.Context, id int) (*model.Product, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Product), args.Error(1)
}
func (m *MockProductRepository) GetAll(ctx context.Context) ([]*model.Product, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Product), args.Error(1)
}
func (m *MockProductRepository) Update(ctx context.Context, product *model.Product) error {
	return m.Called(ctx, product).Error(0)
}
func (m *MockProductRepository) Delete(ctx context.Context, id int) error {
	return m.Called(ctx, id).Error(0)
}

// --- SUITE ---

// ProductServiceTestSuite организует набор тестов для ProductService.
type ProductServiceTestSuite struct {
	suite.Suite
	service *service.ProductService
	repo    *MockProductRepository
	ctx     context.Context
}

// SetupTest инициализирует сервис и моки перед каждым запуском теста.
func (s *ProductServiceTestSuite) SetupTest() {
	s.repo = new(MockProductRepository)
	log, _ := logger.New("Console", "INFO")

	s.service = service.NewProductService(s.repo, log)
	s.ctx = context.Background()
}

// --- CREATE TESTS ---

// TestCreate_Success проверяет успешное создание товара.
func (s *ProductServiceTestSuite) TestCreate_Success() {
	input := model.CreateProductInput{Name: "Клавиатура", Price: 4500, Stock: 15}
	s.repo.On("Create", s.ctx, mock.Anything).Return(nil)

	prod, err := s.service.Create(s.ctx, input)
	s.NoError(err)
	s.Equal(100, prod.ID)
	s.Equal("Клавиатура", prod.Name)
}

// TestCreate_InvalidPrice проверяет валидацию цены (должна быть > 0).
func (s *ProductServiceTestSuite) TestCreate_InvalidPrice() {
	input := model.CreateProductInput{Name: "Товар", Price: -10, Stock: 5}
	_, err := s.service.Create(s.ctx, input)
	s.Error(err)
	s.Contains(err.Error(), "цена товара должна быть строго больше нуля")
}

// TestCreate_InvalidStock проверяет валидацию остатков (не должны быть < 0).
func (s *ProductServiceTestSuite) TestCreate_InvalidStock() {
	input := model.CreateProductInput{Name: "Товар", Price: 100, Stock: -5}
	_, err := s.service.Create(s.ctx, input)
	s.Error(err)
	s.Contains(err.Error(), "остаток товара на складе не может быть отрицательным")
}

// --- GET BY ID TESTS ---

// TestGetByID_Success проверяет успешное получение товара по ID.
func (s *ProductServiceTestSuite) TestGetByID_Success() {
	expectedProduct := &model.Product{ID: 1, Name: "Мышь"}
	s.repo.On("GetByID", s.ctx, 1).Return(expectedProduct, nil)

	prod, err := s.service.GetByID(s.ctx, 1)
	s.NoError(err)
	s.Equal("Мышь", prod.Name)
}

// TestGetByID_NotFound проверяет ошибку, если товар отсутствует.
func (s *ProductServiceTestSuite) TestGetByID_NotFound() {
	s.repo.On("GetByID", s.ctx, 404).Return(nil, model.ErrProductNotFound)

	_, err := s.service.GetByID(s.ctx, 404)
	s.ErrorIs(err, model.ErrProductNotFound)
}

// --- GET ALL TESTS ---

// TestGetAll_Success проверяет получение всех товаров.
func (s *ProductServiceTestSuite) TestGetAll_Success() {
	expected := []*model.Product{{ID: 1, Name: "A"}, {ID: 2, Name: "B"}}
	s.repo.On("GetAll", s.ctx).Return(expected, nil)

	list, err := s.service.GetAll(s.ctx)
	s.NoError(err)
	s.Len(list, 2)
}

// --- UPDATE TESTS ---

// TestUpdate_Success проверяет частичное обновление товара (patch).
func (s *ProductServiceTestSuite) TestUpdate_Success() {
	existing := &model.Product{ID: 5, Name: "Старое", Description: "Описание", Price: 1000, Stock: 10}
	s.repo.On("GetByID", s.ctx, 5).Return(existing, nil)

	newName := "Новое"
	newPrice := 1500
	input := model.UpdateProductInput{Name: &newName, Price: &newPrice}

	s.repo.On("Update", s.ctx, mock.MatchedBy(func(p *model.Product) bool {
		return p.Name == "Новое" && p.Price == 1500 && p.Description == "Описание"
	})).Return(nil)

	updated, err := s.service.Update(s.ctx, 5, input)
	s.NoError(err)
	s.Equal("Новое", updated.Name)
}

// TestUpdate_InvalidPrice проверяет валидацию цены при обновлении.
func (s *ProductServiceTestSuite) TestUpdate_InvalidPrice() {
	s.repo.On("GetByID", s.ctx, 5).Return(&model.Product{ID: 5, Price: 1000}, nil)
	invalidPrice := -50
	input := model.UpdateProductInput{Price: &invalidPrice}

	_, err := s.service.Update(s.ctx, 5, input)
	s.Error(err)
	s.Contains(err.Error(), "цена товара должна быть строго больше нуля")
}

// --- DELETE TESTS ---

// TestDelete_Success проверяет удаление товара.
func (s *ProductServiceTestSuite) TestDelete_Success() {
	s.repo.On("Delete", s.ctx, 10).Return(nil)
	s.NoError(s.service.Delete(s.ctx, 10))
}

// TestDelete_NotFound проверяет ошибку при удалении несуществующего товара.
func (s *ProductServiceTestSuite) TestDelete_NotFound() {
	s.repo.On("Delete", s.ctx, 99).Return(model.ErrProductNotFound)
	s.ErrorIs(s.service.Delete(s.ctx, 99), model.ErrProductNotFound)
}

// TestProductServiceSuite запускает весь набор тестов для ProductService.
func TestProductServiceSuite(t *testing.T) {
	suite.Run(t, new(ProductServiceTestSuite))
}

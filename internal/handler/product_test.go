// Package handler_test содержит модульные тесты для транспортного слоя (HTTP-хендлеров).
// Тесты используют моки для изоляции от слоя бизнес-логики и проверяют корректность
// обработки входящих запросов, валидицию DTO и формирование HTTP-ответов.
package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/antonlearn/go-shop-backend/internal/handler"
	"github.com/antonlearn/go-shop-backend/internal/model"
	"github.com/antonlearn/go-shop-backend/pkg/logger"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// --- MOCK ---

type MockProductService struct {
	mock.Mock
}

func (m *MockProductService) Create(ctx context.Context, input model.CreateProductInput) (*model.Product, error) {
	args := m.Called(mock.Anything, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Product), args.Error(1)
}

func (m *MockProductService) GetByID(ctx context.Context, id int) (*model.Product, error) {
	args := m.Called(mock.Anything, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Product), args.Error(1)
}

func (m *MockProductService) GetAll(ctx context.Context) ([]*model.Product, error) {
	args := m.Called(mock.Anything)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Product), args.Error(1)
}

func (m *MockProductService) Update(ctx context.Context, id int, input model.UpdateProductInput) (*model.Product, error) {
	args := m.Called(mock.Anything, id, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Product), args.Error(1)
}

func (m *MockProductService) Delete(ctx context.Context, id int) error {
	return m.Called(mock.Anything, id).Error(0)
}

// --- SUITE ---

type ProductHandlerTestSuite struct {
	suite.Suite
	mockSvc *MockProductService
	h       *handler.ProductHandler
}

func (s *ProductHandlerTestSuite) SetupTest() {
	log, _ := logger.New("Console", "DEBUG")
	s.mockSvc = new(MockProductService)
	s.h = handler.NewProductHandler(s.mockSvc, log)
}

// withChiURLParam имитирует разбор параметров пути роутером go-chi.
func (s *ProductHandlerTestSuite) withChiURLParam(r *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

// Вспомогательные хелперы для создания указателей из литералов «на лету»
func strPtr(v string) *string { return &v }
func intPtr(v int) *int       { return &v }

// ==========================================
// ТЕСТЫ: Create (POST /api/v1/products)
// ==========================================

func (s *ProductHandlerTestSuite) TestCreate() {
	input := model.CreateProductInput{
		Name:        "Клавиатура Механическая",
		Description: "RGB подсветка",
		Price:       4500,
		Stock:       15,
	}

	s.Run("Success", func() {
		expected := &model.Product{ID: 1, Name: input.Name, Price: input.Price, CreatedAt: time.Now()}
		s.mockSvc.On("Create", mock.Anything, input).Return(expected, nil).Once()

		body, _ := json.Marshal(input)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/products", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		s.h.Create(w, req)

		s.Equal(http.StatusCreated, w.Code)
		s.mockSvc.AssertExpectations(s.T())
	})

	s.Run("Invalid JSON", func() {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/products", bytes.NewBufferString("{bad json"))
		w := httptest.NewRecorder()

		s.h.Create(w, req)

		s.Equal(http.StatusBadRequest, w.Code)
		s.Contains(w.Body.String(), "невалидный формат JSON")
	})

	s.Run("Validation Error", func() {
		invalidInput := model.CreateProductInput{Name: "Ab", Price: -10} // Слишком короткое имя, плохая цена
		body, _ := json.Marshal(invalidInput)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/products", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		s.h.Create(w, req)

		s.Equal(http.StatusBadRequest, w.Code)
		s.Contains(w.Body.String(), "ошибка валидации полей")
	})

	s.Run("Internal Server Error", func() {
		s.mockSvc.On("Create", mock.Anything, input).Return(nil, errors.New("db error")).Once()

		body, _ := json.Marshal(input)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/products", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		s.h.Create(w, req)

		s.Equal(http.StatusInternalServerError, w.Code)
	})
}

// ==========================================
// ТЕСТЫ: GetByID (GET /api/v1/products/{id})
// ==========================================

func (s *ProductHandlerTestSuite) TestGetByID() {
	productID := 42

	s.Run("Success", func() {
		expected := &model.Product{ID: productID, Name: "Мышь", Price: 2000}
		s.mockSvc.On("GetByID", mock.Anything, productID).Return(expected, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/products/42", nil)
		req = s.withChiURLParam(req, "id", strconv.Itoa(productID))
		w := httptest.NewRecorder()

		s.h.GetByID(w, req)

		s.Equal(http.StatusOK, w.Code)
		s.mockSvc.AssertExpectations(s.T())
	})

	s.Run("Invalid ID - Non-numeric", func() {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/products/abc", nil)
		req = s.withChiURLParam(req, "id", "abc")
		w := httptest.NewRecorder()

		s.h.GetByID(w, req)

		s.Equal(http.StatusBadRequest, w.Code)
		s.Contains(w.Body.String(), "неверный идентификатор товара")
	})

	s.Run("Invalid ID - Zero or Negative", func() {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/products/0", nil)
		req = s.withChiURLParam(req, "id", "0")
		w := httptest.NewRecorder()

		s.h.GetByID(w, req)

		s.Equal(http.StatusBadRequest, w.Code)
	})

	s.Run("Not Found", func() {
		s.mockSvc.On("GetByID", mock.Anything, productID).Return(nil, model.ErrProductNotFound).Once()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/products/42", nil)
		req = s.withChiURLParam(req, "id", strconv.Itoa(productID))
		w := httptest.NewRecorder()

		s.h.GetByID(w, req)

		s.Equal(http.StatusNotFound, w.Code)
		s.Contains(w.Body.String(), "товар не найден")
	})

	s.Run("Internal Server Error", func() {
		s.mockSvc.On("GetByID", mock.Anything, productID).Return(nil, errors.New("cluster down")).Once()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/products/42", nil)
		req = s.withChiURLParam(req, "id", strconv.Itoa(productID))
		w := httptest.NewRecorder()

		s.h.GetByID(w, req)

		s.Equal(http.StatusInternalServerError, w.Code)
	})
}

// ==========================================
// ТЕСТЫ: GetAll (GET /api/v1/products)
// ==========================================

func (s *ProductHandlerTestSuite) TestGetAll() {
	s.Run("Success - With Elements", func() {
		expected := []*model.Product{{ID: 1, Name: "Товар 1"}, {ID: 2, Name: "Товар 2"}}
		s.mockSvc.On("GetAll", mock.Anything).Return(expected, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/products", nil)
		w := httptest.NewRecorder()

		s.h.GetAll(w, req)

		s.Equal(http.StatusOK, w.Code)
		s.mockSvc.AssertExpectations(s.T())
	})

	s.Run("Success - Empty Returns Array", func() {
		s.mockSvc.On("GetAll", mock.Anything).Return(([]*model.Product)(nil), nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/products", nil)
		w := httptest.NewRecorder()

		s.h.GetAll(w, req)

		s.Equal(http.StatusOK, w.Code)
		s.JSONEq("[]", w.Body.String())
	})

	s.Run("Internal Server Error", func() {
		s.mockSvc.On("GetAll", mock.Anything).Return(nil, errors.New("read error")).Once()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/products", nil)
		w := httptest.NewRecorder()

		s.h.GetAll(w, req)

		s.Equal(http.StatusInternalServerError, w.Code)
	})
}

// ==========================================
// ТЕСТЫ: Update (PUT /api/v1/products/{id})
// ==========================================

func (s *ProductHandlerTestSuite) TestUpdate() {
	productID := 5
	input := model.UpdateProductInput{
		Name:  strPtr("Обновленное имя"),
		Price: intPtr(9990),
	}

	s.Run("Success", func() {
		expected := &model.Product{ID: productID, Name: *input.Name, Price: 9990}
		s.mockSvc.On("Update", mock.Anything, productID, input).Return(expected, nil).Once()

		body, _ := json.Marshal(input)
		req := httptest.NewRequest(http.MethodPut, "/api/v1/products/5", bytes.NewBuffer(body))
		req = s.withChiURLParam(req, "id", strconv.Itoa(productID))
		w := httptest.NewRecorder()

		s.h.Update(w, req)

		s.Equal(http.StatusOK, w.Code)
		s.mockSvc.AssertExpectations(s.T())
	})

	s.Run("Invalid ID", func() {
		req := httptest.NewRequest(http.MethodPut, "/api/v1/products/0", nil)
		req = s.withChiURLParam(req, "id", "0")
		w := httptest.NewRecorder()

		s.h.Update(w, req)

		s.Equal(http.StatusBadRequest, w.Code)
	})

	s.Run("Invalid JSON", func() {
		req := httptest.NewRequest(http.MethodPut, "/api/v1/products/5", bytes.NewBufferString("{bad json"))
		req = s.withChiURLParam(req, "id", strconv.Itoa(productID))
		w := httptest.NewRecorder()

		s.h.Update(w, req)

		s.Equal(http.StatusBadRequest, w.Code)
	})

	s.Run("Not Found", func() {
		s.mockSvc.On("Update", mock.Anything, productID, input).Return(nil, model.ErrProductNotFound).Once()

		body, _ := json.Marshal(input)
		req := httptest.NewRequest(http.MethodPut, "/api/v1/products/5", bytes.NewBuffer(body))
		req = s.withChiURLParam(req, "id", strconv.Itoa(productID))
		w := httptest.NewRecorder()

		s.h.Update(w, req)

		s.Equal(http.StatusNotFound, w.Code)
		s.Contains(w.Body.String(), "товар для обновления не найден")
	})

	s.Run("Internal Server Error", func() {
		s.mockSvc.On("Update", mock.Anything, productID, input).Return(nil, errors.New("failed update")).Once()

		body, _ := json.Marshal(input)
		req := httptest.NewRequest(http.MethodPut, "/api/v1/products/5", bytes.NewBuffer(body))
		req = s.withChiURLParam(req, "id", strconv.Itoa(productID))
		w := httptest.NewRecorder()

		s.h.Update(w, req)

		s.Equal(http.StatusInternalServerError, w.Code)
	})
}

// ==========================================
// ТЕСТЫ: Delete (DELETE /api/v1/products/{id})
// ==========================================

func (s *ProductHandlerTestSuite) TestDelete() {
	productID := 7

	s.Run("Success", func() {
		s.mockSvc.On("Delete", mock.Anything, productID).Return(nil).Once()

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/products/7", nil)
		req = s.withChiURLParam(req, "id", strconv.Itoa(productID))
		w := httptest.NewRecorder()

		s.h.Delete(w, req)

		s.Equal(http.StatusOK, w.Code)
		s.Contains(w.Body.String(), `"status":"deleted"`)
		s.mockSvc.AssertExpectations(s.T())
	})

	s.Run("Invalid ID", func() {
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/products/abc", nil)
		req = s.withChiURLParam(req, "id", "abc")
		w := httptest.NewRecorder()

		s.h.Delete(w, req)

		s.Equal(http.StatusBadRequest, w.Code)
	})

	s.Run("Not Found", func() {
		s.mockSvc.On("Delete", mock.Anything, productID).Return(model.ErrProductNotFound).Once()

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/products/7", nil)
		req = s.withChiURLParam(req, "id", strconv.Itoa(productID))
		w := httptest.NewRecorder()

		s.h.Delete(w, req)

		s.Equal(http.StatusNotFound, w.Code)
		s.Contains(w.Body.String(), "товар для удаления не найден")
	})

	s.Run("Internal Server Error", func() {
		s.mockSvc.On("Delete", mock.Anything, productID).Return(errors.New("delete failed")).Once()

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/products/7", nil)
		req = s.withChiURLParam(req, "id", strconv.Itoa(productID))
		w := httptest.NewRecorder()

		s.h.Delete(w, req)

		s.Equal(http.StatusInternalServerError, w.Code)
	})
}

func TestProductHandlerSuite(t *testing.T) {
	suite.Run(t, new(ProductHandlerTestSuite))
}

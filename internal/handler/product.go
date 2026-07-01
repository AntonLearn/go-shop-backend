// Package handler реализует транспортный слой (HTTP) приложения,
// обрабатывает входящие запросы, валидирует DTO и вызывает слой бизнес-логики.
package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/antonlearn/go-shop-backend/internal/model"
	"github.com/antonlearn/go-shop-backend/pkg/logger"
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
)

// ProductService описывает интерфейс взаимодействия хендлера с бизнес-логикой каталога товаров.
type ProductService interface {
	Create(ctx context.Context, input model.CreateProductInput) (*model.Product, error)
	GetByID(ctx context.Context, id int) (*model.Product, error)
	GetAll(ctx context.Context) ([]*model.Product, error)
	Update(ctx context.Context, id int, input model.UpdateProductInput) (*model.Product, error)
	Delete(ctx context.Context, id int) error
}

// ProductHandler инкапсулирует логику HTTP-обработчиков для управления каталогом товаров.
type ProductHandler struct {
	service   ProductService
	validator *validator.Validate
	log       *logger.Logger
}

// NewProductHandler конструирует новый HTTP-контроллер для управления каталогом.
func NewProductHandler(service ProductService, log *logger.Logger) *ProductHandler {
	return &ProductHandler{
		service:   service,
		validator: validator.New(),
		log:       log,
	}
}

// Create обрабатывает POST /api/v1/products (Доступно: Admin).
func (h *ProductHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input model.CreateProductInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondWithError(w, r.Context(), h.log, http.StatusBadRequest, "невалидный формат JSON")
		return
	}

	if err := h.validator.Struct(input); err != nil {
		respondWithError(w, r.Context(), h.log, http.StatusBadRequest, "ошибка валидации полей: "+err.Error())
		return
	}

	product, err := h.service.Create(r.Context(), input)
	if err != nil {
		h.log.ErrorContext(r.Context(), "ошибка при обработке создания товара", "error", err)
		respondWithError(w, r.Context(), h.log, http.StatusInternalServerError, "внутренняя ошибка сервера")
		return
	}

	respondWithJSON(w, r.Context(), h.log, http.StatusCreated, product)
}

// GetByID обрабатывает GET /api/v1/products/{id} (Доступно: Все).
func (h *ProductHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idParam)
	if err != nil || id <= 0 {
		respondWithError(w, r.Context(), h.log, http.StatusBadRequest, "неверный идентификатор товара")
		return
	}

	product, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, model.ErrProductNotFound) {
			respondWithError(w, r.Context(), h.log, http.StatusNotFound, "товар не найден")
			return
		}
		h.log.ErrorContext(r.Context(), "ошибка получения товара по ID", "product_id", id, "error", err)
		respondWithError(w, r.Context(), h.log, http.StatusInternalServerError, "внутренняя ошибка сервера")
		return
	}

	respondWithJSON(w, r.Context(), h.log, http.StatusOK, product)
}

// GetAll обрабатывает GET /api/v1/products (Доступно: Все).
func (h *ProductHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	products, err := h.service.GetAll(r.Context())
	if err != nil {
		h.log.ErrorContext(r.Context(), "ошибка получения каталога товаров", "error", err)
		respondWithError(w, r.Context(), h.log, http.StatusInternalServerError, "внутренняя ошибка сервера")
		return
	}

	// Если товаров в базе нет вообще, возвращаем пустой слайс `[]` вместо `null` в JSON
	if products == nil {
		products = make([]*model.Product, 0)
	}

	respondWithJSON(w, r.Context(), h.log, http.StatusOK, products)
}

// Update обрабатывает PUT /api/v1/products/{id} (Доступно: Admin).
func (h *ProductHandler) Update(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idParam)
	if err != nil || id <= 0 {
		respondWithError(w, r.Context(), h.log, http.StatusBadRequest, "неверный идентификатор товара")
		return
	}

	var input model.UpdateProductInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondWithError(w, r.Context(), h.log, http.StatusBadRequest, "невалидный формат JSON")
		return
	}

	if err := h.validator.Struct(input); err != nil {
		respondWithError(w, r.Context(), h.log, http.StatusBadRequest, "ошибка валидации полей: "+err.Error())
		return
	}

	product, err := h.service.Update(r.Context(), id, input)
	if err != nil {
		if errors.Is(err, model.ErrProductNotFound) {
			respondWithError(w, r.Context(), h.log, http.StatusNotFound, "товар для обновления не найден")
			return
		}
		h.log.ErrorContext(r.Context(), "ошибка обновления товара", "product_id", id, "error", err)
		respondWithError(w, r.Context(), h.log, http.StatusInternalServerError, "внутренняя ошибка сервера")
		return
	}

	respondWithJSON(w, r.Context(), h.log, http.StatusOK, product)
}

// Delete обрабатывает DELETE /api/v1/products/{id} (Доступно: Admin).
func (h *ProductHandler) Delete(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idParam)
	if err != nil || id <= 0 {
		respondWithError(w, r.Context(), h.log, http.StatusBadRequest, "неверный идентификатор товара")
		return
	}

	err = h.service.Delete(r.Context(), id)
	if err != nil {
		if errors.Is(err, model.ErrProductNotFound) {
			respondWithError(w, r.Context(), h.log, http.StatusNotFound, "товар для удаления не найден")
			return
		}
		h.log.ErrorContext(r.Context(), "ошибка удаления товара", "product_id", id, "error", err)
		respondWithError(w, r.Context(), h.log, http.StatusInternalServerError, "внутренняя ошибка сервера")
		return
	}

	respondWithJSON(w, r.Context(), h.log, http.StatusOK, map[string]string{"status": "deleted"})
}

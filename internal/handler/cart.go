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

// CartServiceInterface описывает контракт бизнес-логики управления корзиной.
type CartServiceInterface interface {
	Add(ctx context.Context, userID int, input model.AddToCartInput) error
	GetByUserID(ctx context.Context, userID int) ([]*model.CartOutputItem, error)
	Delete(ctx context.Context, userID int, productID int) error
}

// CartHandler инкапсулирует логику HTTP-обработчиков для корзины.
type CartHandler struct {
	services  CartServiceInterface
	validator *validator.Validate
	log       *logger.Logger
}

// NewCartHandler конструирует новый хендлер для управления корзиной.
func NewCartHandler(services CartServiceInterface, log *logger.Logger) *CartHandler {
	return &CartHandler{
		services:  services,
		validator: validator.New(),
		log:       log,
	}
}

// Add обрабатывает запрос на добавление товара в корзину (POST /api/v1/cart).
func (h *CartHandler) Add(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r.Context())
	if !ok {
		h.log.WarnContext(r.Context(), "попытка доступа к корзине неаутентифицированным пользователем")
		respondWithError(w, r.Context(), h.log, http.StatusUnauthorized, "пользователь не аутентифицирован")
		return
	}

	var input model.AddToCartInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		h.log.ErrorContext(r.Context(), "не удалось декодировать тело запроса добавления в корзину", "error", err)
		respondWithError(w, r.Context(), h.log, http.StatusBadRequest, "некорректное тело запроса")
		return
	}

	if err := h.validator.Struct(input); err != nil {
		h.log.WarnContext(r.Context(), "ошибка валидации данных добавления в корзину", "error", err)
		respondWithError(w, r.Context(), h.log, http.StatusBadRequest, "ошибка валидации данных")
		return
	}

	if err := h.services.Add(r.Context(), userID, input); err != nil {
		h.log.ErrorContext(r.Context(), "сбой при добавлении товара в корзину через сервис", "userID", userID, "productID", input.ProductID, "error", err)
		if errors.Is(err, model.ErrProductNotFound) {
			respondWithError(w, r.Context(), h.log, http.StatusNotFound, "товар не найден в каталоге")
			return
		}
		respondWithError(w, r.Context(), h.log, http.StatusInternalServerError, "не удалось добавить товар в корзину")
		return
	}

	respondWithJSON(w, r.Context(), h.log, http.StatusOK, map[string]string{"message": "товар успешно добавлен в корзину"})
}

// GetByID обрабатывает запрос на получение всех товаров в корзине (GET /api/v1/cart).
func (h *CartHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r.Context())
	if !ok {
		h.log.WarnContext(r.Context(), "попытка получения содержимого корзины неаутентифицированным пользователем")
		respondWithError(w, r.Context(), h.log, http.StatusUnauthorized, "пользователь не аутентифицирован")
		return
	}

	items, err := h.services.GetByUserID(r.Context(), userID)
	if err != nil {
		h.log.ErrorContext(r.Context(), "сбой при получении корзины из сервиса", "userID", userID, "error", err)
		respondWithError(w, r.Context(), h.log, http.StatusInternalServerError, "не удалось получить содержимое корзины")
		return
	}

	respondWithJSON(w, r.Context(), h.log, http.StatusOK, items)
}

// Delete обрабатывает запрос на удаление товара из корзины (DELETE /api/v1/cart/{id}).
func (h *CartHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r.Context())
	if !ok {
		h.log.WarnContext(r.Context(), "попытка удаления товара из корзины неаутентифицированным пользователем")
		respondWithError(w, r.Context(), h.log, http.StatusUnauthorized, "пользователь не аутентифицирован")
		return
	}

	productID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil || productID <= 0 {
		h.log.WarnContext(r.Context(), "передан некорректный ID товара для удаления из корзины", "rawID", chi.URLParam(r, "id"))
		respondWithError(w, r.Context(), h.log, http.StatusBadRequest, "некорректный ID товара")
		return
	}

	if err := h.services.Delete(r.Context(), userID, productID); err != nil {
		h.log.ErrorContext(r.Context(), "сбой при удалении товара из корзины через сервис", "userID", userID, "productID", productID, "error", err)
		if errors.Is(err, model.ErrCartItemNotFound) {
			respondWithError(w, r.Context(), h.log, http.StatusNotFound, "товар в корзине не найден")
			return
		}
		respondWithError(w, r.Context(), h.log, http.StatusInternalServerError, "не удалось удалить товар из корзины")
		return
	}

	respondWithJSON(w, r.Context(), h.log, http.StatusOK, map[string]string{"message": "товар успешно удален из корзины"})
}

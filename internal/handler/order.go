// Package handler реализует транспортный слой (HTTP) приложения,
// обрабатывает входящие запросы и вызывает слой бизнес-логики.
package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/antonlearn/go-shop-backend/internal/model"
	"github.com/antonlearn/go-shop-backend/pkg/logger"
	"github.com/go-chi/chi/v5"
)

// OrderServiceInterface описывает контракт бизнес-логики для заказов.
// Хэндлер зависит от этого интерфейса, а не от конкретной реализации (OrderService).
type OrderServiceInterface interface {
	Create(ctx context.Context, userID int) (*model.Order, error)
	GetByID(ctx context.Context, userID, orderID int) (*model.OrderOutput, error)
	GetByUserID(ctx context.Context, userID int) ([]*model.Order, error)
}

// OrderHandler инкапсулирует логику HTTP-обработчиков для заказов.
type OrderHandler struct {
	service OrderServiceInterface
	log     *logger.Logger
}

// NewOrderHandler конструирует новый хэндлер для работы с заказами.
func NewOrderHandler(service OrderServiceInterface, log *logger.Logger) *OrderHandler {
	return &OrderHandler{
		service: service,
		log:     log,
	}
}

// Create обрабатывает POST /api/v1/orders (Создание заказа из корзины).
func (h *OrderHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r.Context())
	if !ok {
		respondWithError(w, r.Context(), h.log, http.StatusUnauthorized, "пользователь не аутентифицирован")
		return
	}

	order, err := h.service.Create(r.Context(), userID)
	if err != nil {
		h.log.ErrorContext(r.Context(), "ошибка при оформлении заказа", "userID", userID, "error", err)

		// Здесь можно добавить проверку на специфические ошибки, например, если корзина пуста
		if errors.Is(err, model.ErrCartIsEmpty) {
			respondWithError(w, r.Context(), h.log, http.StatusBadRequest, "корзина пуста")
			return
		}

		respondWithError(w, r.Context(), h.log, http.StatusInternalServerError, "не удалось оформить заказ")
		return
	}

	respondWithJSON(w, r.Context(), h.log, http.StatusCreated, order)
}

// GetByID обрабатывает GET /api/v1/orders/{id} (Получение детальной информации о заказе).
func (h *OrderHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r.Context())
	if !ok {
		respondWithError(w, r.Context(), h.log, http.StatusUnauthorized, "пользователь не аутентифицирован")
		return
	}

	orderID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil || orderID <= 0 {
		respondWithError(w, r.Context(), h.log, http.StatusBadRequest, "некорректный ID заказа")
		return
	}

	order, err := h.service.GetByID(r.Context(), userID, orderID)
	if err != nil {
		h.log.ErrorContext(r.Context(), "ошибка получения заказа", "userID", userID, "orderID", orderID, "error", err)

		if errors.Is(err, model.ErrOrderNotFound) {
			respondWithError(w, r.Context(), h.log, http.StatusNotFound, "заказ не найден")
			return
		}

		respondWithError(w, r.Context(), h.log, http.StatusInternalServerError, "не удалось получить информацию о заказе")
		return
	}

	respondWithJSON(w, r.Context(), h.log, http.StatusOK, order)
}

// GetByUserID обрабатывает GET /api/v1/orders (История заказов пользователя).
func (h *OrderHandler) GetByUserID(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r.Context())
	if !ok {
		respondWithError(w, r.Context(), h.log, http.StatusUnauthorized, "пользователь не аутентифицирован")
		return
	}

	orders, err := h.service.GetByUserID(r.Context(), userID)
	if err != nil {
		h.log.ErrorContext(r.Context(), "ошибка получения истории заказов", "userID", userID, "error", err)
		respondWithError(w, r.Context(), h.log, http.StatusInternalServerError, "не удалось получить историю заказов")
		return
	}

	// Возвращаем пустой массив, если заказов нет
	if orders == nil {
		orders = make([]*model.Order, 0)
	}

	respondWithJSON(w, r.Context(), h.log, http.StatusOK, orders)
}

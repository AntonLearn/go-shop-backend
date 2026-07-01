// Package service реализует слой бизнес-логики приложения (Application Service Layer).
package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/antonlearn/go-shop-backend/internal/model"
	"github.com/antonlearn/go-shop-backend/pkg/logger"
)

// OrderRepository описывает требования сервиса к слою хранения для заказов.
type OrderRepository interface {
	// Create выполняет транзакционную процедуру оформления: фиксирует снимок цен,
	// создает запись заказа, списывает остатки товаров на складе и очищает корзину.
	Create(ctx context.Context, userID int) (*model.Order, error)
	GetByID(ctx context.Context, orderID, userID int) (*model.OrderOutput, error)
	GetByUserID(ctx context.Context, userID int) ([]*model.Order, error)
}

// OrderService координирует процессы транзакционного оформления заказов
// и извлечения истории покупок пользователей.
type OrderService struct {
	repo OrderRepository
	log  *logger.Logger
}

// NewOrderService конструирует новый сервис для управления заказами.
func NewOrderService(repo OrderRepository, log *logger.Logger) *OrderService {
	return &OrderService{
		repo: repo,
		log:  log,
	}
}

// Create координирует транзакционное превращение корзины в оформленный заказ.
func (s *OrderService) Create(ctx context.Context, userID int) (*model.Order, error) {
	if userID <= 0 {
		return nil, errors.New("некорректный идентификатор пользователя")
	}

	// Вся тяжелая транзакционная логика (создание заказа, моментальный снимок цен товаров,
	// уменьшение остатков на складах и очистка корзины покупателя) инкапсулирована
	// на уровне репозитория для обеспечения строгой атомарности (ACID) средствами СУБД.
	order, err := s.repo.Create(ctx, userID)
	if err != nil {
		if errors.Is(err, errors.New("корзина пуста")) { // Пример возможной бизнес-ошибки
			return nil, errors.New("невозможно оформить заказ: корзина пуста")
		}
		s.log.ErrorContext(ctx, "ошибка при транзакционном оформлении заказа", "user_id", userID, "error", err)
		return nil, fmt.Errorf("failed to place order: %w", err)
	}

	s.log.InfoContext(ctx, "бизнес-логика: успешно оформлен новый заказ", "order_id", order.ID, "user_id", userID)
	return order, nil
}

// GetByID возвращает расширенные данные по конкретному заказу (включая все его позиции).
func (s *OrderService) GetByID(ctx context.Context, orderID, userID int) (*model.OrderOutput, error) {
	orderOutput, err := s.repo.GetByID(ctx, orderID, userID)
	if err != nil {
		if errors.Is(err, model.ErrOrderNotFound) {
			return nil, model.ErrOrderNotFound
		}
		s.log.ErrorContext(ctx, "ошибка извлечения деталей заказа", "order_id", orderID, "user_id", userID, "error", err)
		return nil, fmt.Errorf("failed to retrieve order: %w", err)
	}

	return orderOutput, nil
}

// GetByUserID возвращает краткий список всех прошлых заказов пользователя (историю покупок).
func (s *OrderService) GetByUserID(ctx context.Context, userID int) ([]*model.Order, error) {
	orders, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		s.log.ErrorContext(ctx, "ошибка получения истории заказов пользователя", "user_id", userID, "error", err)
		return nil, fmt.Errorf("failed to retrieve user order history: %w", err)
	}

	return orders, nil
}

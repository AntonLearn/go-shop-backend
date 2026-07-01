// Package service реализует слой бизнес-логики приложения (Application Service Layer).
package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/antonlearn/go-shop-backend/internal/model"
	"github.com/antonlearn/go-shop-backend/pkg/logger"
)

// CartRepository описывает методы слоя хранения для корзины покупателя.
type CartRepository interface {
	Add(ctx context.Context, item *model.CartItem) error
	GetByUserID(ctx context.Context, userID int) ([]*model.CartOutputItem, error)
	Delete(ctx context.Context, userID, productID int) error
}

// CartProductRepository описывает метод проверки существования и состояния товара.
type CartProductRepository interface {
	GetByID(ctx context.Context, id int) (*model.Product, error)
}

// CartService координирует операции добавления, просмотра и очистки элементов корзины,
// гарантируя консистентность данных перед отправкой в репозиторий.
type CartService struct {
	cartRepo    CartRepository
	productRepo CartProductRepository
	log         *logger.Logger
}

// NewCartService конструирует новый сервис бизнес-логики корзины.
func NewCartService(cartRepo CartRepository, productRepo CartProductRepository, log *logger.Logger) *CartService {
	return &CartService{
		cartRepo:    cartRepo,
		productRepo: productRepo,
		log:         log,
	}
}

// Add проверяет существование товара, его доступность на складе и добавляет в корзину пользователя.
func (s *CartService) Add(ctx context.Context, userID int, input model.AddToCartInput) error {
	// 1. Защитная бизнес-валидация входных данных
	if input.Quantity <= 0 {
		s.log.WarnContext(ctx, "попытка добавить невалидное количество товара", "user_id", userID, "quantity", input.Quantity)
		return errors.New("количество товара должно быть строго больше нуля")
	}

	// 2. Бизнес-проверка: проверяем, существует ли вообще такой товар в каталоге магазина
	product, err := s.productRepo.GetByID(ctx, input.ProductID)
	if err != nil {
		if errors.Is(err, model.ErrProductNotFound) {
			s.log.WarnContext(ctx, "попытка добавления в корзину несуществующего товара", "product_id", input.ProductID)
			return model.ErrProductNotFound
		}
		s.log.ErrorContext(ctx, "ошибка проверки товара при добавлении в корзину", "product_id", input.ProductID, "error", err)
		return fmt.Errorf("failed to verify product existence: %w", err)
	}

	// 3. Бизнес-проверка: достаточно ли остатка на складе под запрашиваемое количество
	if product.Stock < input.Quantity {
		s.log.WarnContext(ctx, "недостаточно товара на складе для корзины",
			"product_id", input.ProductID, "requested", input.Quantity, "available", product.Stock)
		return errors.New("недостаточно товара на складе")
	}

	item := &model.CartItem{
		UserID:    userID,
		ProductID: input.ProductID,
		Quantity:  input.Quantity,
	}

	// 4. Сохранение элемента в репозитории
	if err := s.cartRepo.Add(ctx, item); err != nil {
		s.log.ErrorContext(ctx, "ошибка сохранения элемента корзины в репозитории", "user_id", userID, "error", err)
		return fmt.Errorf("failed to add item to cart: %w", err)
	}

	s.log.InfoContext(ctx, "бизнес-логика: товар успешно добавлен в корзину", "user_id", userID, "product_id", input.ProductID)
	return nil
}

// GetByUserID возвращает все товары в корзине конкретного пользователя.
func (s *CartService) GetByUserID(ctx context.Context, userID int) ([]*model.CartOutputItem, error) {
	items, err := s.cartRepo.GetByUserID(ctx, userID)
	if err != nil {
		s.log.ErrorContext(ctx, "ошибка получения корзины из репозитория", "user_id", userID, "error", err)
		return nil, fmt.Errorf("failed to get cart items: %w", err)
	}

	return items, nil
}

// Delete удаляет товар из корзины пользователя.
func (s *CartService) Delete(ctx context.Context, userID, productID int) error {
	if err := s.cartRepo.Delete(ctx, userID, productID); err != nil {
		if errors.Is(err, model.ErrCartItemNotFound) {
			return model.ErrCartItemNotFound
		}
		s.log.ErrorContext(ctx, "ошибка удаления товара из корзины в репозитории", "user_id", userID, "product_id", productID, "error", err)
		return fmt.Errorf("failed to delete item from cart: %w", err)
	}

	s.log.InfoContext(ctx, "бизнес-логика: товар удален из корзины покупателя", "user_id", userID, "product_id", productID)
	return nil
}

// Package service реализует слой бизнес-логики приложения (Application Service Layer).
package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/antonlearn/go-shop-backend/internal/model"
	"github.com/antonlearn/go-shop-backend/pkg/logger"
)

// ProductRepository контракт, который мы уже реализовали в слое repository.
type ProductRepository interface {
	Create(ctx context.Context, product *model.Product) error
	GetByID(ctx context.Context, id int) (*model.Product, error)
	GetAll(ctx context.Context) ([]*model.Product, error)
	Update(ctx context.Context, product *model.Product) error
	Delete(ctx context.Context, id int) error
}

// ProductService управляет полным жизненным циклом номенклатурных позиций (товаров)
// в каталоге интернет-магазина, выполняя защитную бизнес-валидацию данных.
type ProductService struct {
	repo ProductRepository
	log  *logger.Logger
}

// NewProductService конструирует новый сервис для работы с каталогом товаров.
func NewProductService(repo ProductRepository, log *logger.Logger) *ProductService {
	return &ProductService{
		repo: repo,
		log:  log,
	}
}

// Create принимает валидированную DTO, перекладывает данные в доменную модель и сохраняет её.
func (s *ProductService) Create(ctx context.Context, input model.CreateProductInput) (*model.Product, error) {
	// Дополнительная доменная бизнес-валидация параметров
	if input.Price <= 0 {
		s.log.WarnContext(ctx, "попытка создать товар с некорректной ценой", "price", input.Price)
		return nil, errors.New("цена товара должна быть строго больше нуля")
	}
	if input.Stock < 0 {
		s.log.WarnContext(ctx, "попытка создать товар с отрицательным остатком", "stock", input.Stock)
		return nil, errors.New("остаток товара на складе не может быть отрицательным")
	}

	product := &model.Product{
		Name:        input.Name,
		Description: input.Description,
		Price:       input.Price,
		Stock:       input.Stock,
	}

	if err := s.repo.Create(ctx, product); err != nil {
		return nil, fmt.Errorf("failed to create product in service: %w", err)
	}

	s.log.InfoContext(ctx, "бизнес-логика: успешно создан новый товар", "product_id", product.ID)
	return product, nil
}

// GetByID возвращает товар по его идентификатору.
func (s *ProductService) GetByID(ctx context.Context, id int) (*model.Product, error) {
	product, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, model.ErrProductNotFound) {
			return nil, model.ErrProductNotFound
		}
		return nil, fmt.Errorf("failed to get product by id %d: %w", id, err)
	}
	return product, nil
}

// GetAll возвращает полный список товаров для витрины.
func (s *ProductService) GetAll(ctx context.Context) ([]*model.Product, error) {
	products, err := s.repo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all products: %w", err)
	}
	return products, nil
}

// Update реализует логику умного патча (частичного обновления) товара с доменной валидацией.
func (s *ProductService) Update(ctx context.Context, id int, input model.UpdateProductInput) (*model.Product, error) {
	// 1. Сначала получаем текущее состояние товара из базы
	product, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, model.ErrProductNotFound) {
			return nil, model.ErrProductNotFound
		}
		return nil, fmt.Errorf("failed to find product for update: %w", err)
	}

	// 2. Если поле в DTO передано (не nil), валидируем бизнес-правила и обновляем его в модели
	if input.Name != nil {
		if *input.Name == "" {
			return nil, errors.New("название товара не может быть пустым")
		}
		product.Name = *input.Name
	}
	if input.Description != nil {
		product.Description = *input.Description
	}
	if input.Price != nil {
		if *input.Price <= 0 {
			s.log.WarnContext(ctx, "попытка обновления цены на некорректную", "product_id", id, "price", *input.Price)
			return nil, errors.New("цена товара должна быть строго больше нуля")
		}
		product.Price = *input.Price
	}
	if input.Stock != nil {
		if *input.Stock < 0 {
			s.log.WarnContext(ctx, "попытка обновления остатков в минус", "product_id", id, "stock", *input.Stock)
			return nil, errors.New("остаток товара на складе не может быть отрицательным")
		}
		product.Stock = *input.Stock
	}

	// 3. Отправляем обновленную модель обратно в базу данных
	if err := s.repo.Update(ctx, product); err != nil {
		return nil, fmt.Errorf("failed to update product fields: %w", err)
	}

	s.log.InfoContext(ctx, "бизнес-логика: товар успешно обновлен", "product_id", id)
	return product, nil
}

// Delete удаляет товар из каталога.
func (s *ProductService) Delete(ctx context.Context, id int) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		if errors.Is(err, model.ErrProductNotFound) {
			return model.ErrProductNotFound
		}
		return fmt.Errorf("failed to delete product in service: %w", err)
	}
	return nil
}

// Package sqlite реализует слой доступа к данным для СУБД SQLite.
package sqlite

import (
	"context"

	"github.com/antonlearn/go-shop-backend/internal/model"
	"github.com/antonlearn/go-shop-backend/internal/repository"
	"github.com/antonlearn/go-shop-backend/pkg/logger"
	"github.com/jmoiron/sqlx"
)

// Гарантия компиляции: проверяем, что SQLiteProductRepository реализует repository.ProductRepository.
var _ repository.ProductRepository = (*SQLiteProductRepository)(nil)

// SQLiteProductRepository реализует контракт ProductRepository для работы с СУБД SQLite.
type SQLiteProductRepository struct {
	db  *sqlx.DB
	log *logger.Logger
}

// dbProduct — внутренняя структура для маппинга данных товара из БД.
type dbProduct struct {
	ID          int    `db:"id"`
	Name        string `db:"name"`
	Description string `db:"description"`
	Price       int    `db:"price"`
	Stock       int    `db:"stock"`
	CreatedAt   string `db:"created_at"`
}

// NewSQLiteProductRepository конструирует репозиторий товаров.
func NewSQLiteProductRepository(db *sqlx.DB, log *logger.Logger) *SQLiteProductRepository {
	return &SQLiteProductRepository{
		db:  db,
		log: log,
	}
}

// Create добавляет новый товар в каталог.
func (r *SQLiteProductRepository) Create(ctx context.Context, product *model.Product) error {
	query := `INSERT INTO products (name, description, price, stock) 
              VALUES (:name, :description, :price, :stock)`

	result, err := r.db.NamedExecContext(ctx, query, product)
	if err != nil {
		r.log.ErrorContext(ctx, "ошибка выполнения SQL при добавлении товара", "name", product.Name, "error", err)
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	product.ID = int(id)

	return nil
}

// GetByID ищет один конкретный товар по его ID.
func (r *SQLiteProductRepository) GetByID(ctx context.Context, id int) (*model.Product, error) {
	query := `SELECT id, name, description, price, stock, created_at FROM products WHERE id = :id LIMIT 1`

	rows, err := r.db.NamedQueryContext(ctx, query, map[string]any{"id": id})
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, model.ErrProductNotFound
	}

	var p dbProduct
	if err := rows.StructScan(&p); err != nil {
		return nil, err
	}

	parsedTime, err := parseSQLiteTime(p.CreatedAt)
	if err != nil {
		return nil, err
	}

	return &model.Product{
		ID:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		Price:       p.Price,
		Stock:       p.Stock,
		CreatedAt:   parsedTime,
	}, nil
}

// GetAll выгружает весь перечень товаров.
func (r *SQLiteProductRepository) GetAll(ctx context.Context) ([]*model.Product, error) {
	query := `SELECT id, name, description, price, stock, created_at FROM products ORDER BY id DESC`

	rows, err := r.db.QueryxContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []*model.Product
	for rows.Next() {
		var p dbProduct
		if err := rows.StructScan(&p); err != nil {
			return nil, err
		}

		parsedTime, err := parseSQLiteTime(p.CreatedAt)
		if err != nil {
			return nil, err
		}

		products = append(products, &model.Product{
			ID:          p.ID,
			Name:        p.Name,
			Description: p.Description,
			Price:       p.Price,
			Stock:       p.Stock,
			CreatedAt:   parsedTime,
		})
	}
	return products, nil
}

// Update обновляет поля товара.
func (r *SQLiteProductRepository) Update(ctx context.Context, product *model.Product) error {
	query := `UPDATE products 
              SET name = :name, description = :description, price = :price, stock = :stock 
              WHERE id = :id`

	result, err := r.db.NamedExecContext(ctx, query, product)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return model.ErrProductNotFound
	}
	return nil
}

// Delete удаляет товар из каталога.
func (r *SQLiteProductRepository) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM products WHERE id = :id`

	result, err := r.db.NamedExecContext(ctx, query, map[string]any{"id": id})
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return model.ErrProductNotFound
	}
	return nil
}

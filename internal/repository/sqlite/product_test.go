// Package sqlite_test содержит интеграционные тесты для слоя доступа к данным (DAL) на базе SQLite.
// Файл product_test.go тестирует функциональность каталога товаров.
package sqlite_test

import (
	"context"
	"testing"

	"github.com/antonlearn/go-shop-backend/internal/model"
	"github.com/antonlearn/go-shop-backend/internal/repository/sqlite"
	"github.com/stretchr/testify/assert"
)

func TestSQLiteProductRepository(t *testing.T) {
	db, log := setupTestDB(t)
	defer db.Close()

	repo := sqlite.NewSQLiteProductRepository(db, log)
	ctx := context.Background()

	product := &model.Product{
		Name:        "Клавиатура",
		Description: "Механическая RGB клавиатура",
		Price:       5000,
		Stock:       10,
	}

	t.Run("Create Success", func(t *testing.T) {
		err := repo.Create(ctx, product)
		assert.NoError(t, err)
		assert.NotZero(t, product.ID)
	})

	t.Run("GetByID Success", func(t *testing.T) {
		p, err := repo.GetByID(ctx, product.ID)
		assert.NoError(t, err)
		assert.Equal(t, product.Name, p.Name)
		assert.Equal(t, product.Price, p.Price)
	})

	t.Run("GetAll Success", func(t *testing.T) {
		products, err := repo.GetAll(ctx)
		assert.NoError(t, err)
		assert.Len(t, products, 1)
	})

	t.Run("Update Success", func(t *testing.T) {
		product.Price = 5500
		product.Stock = 8

		// Исправлено: вызываем метод напрямую у репозитория
		err := repo.Update(ctx, product)
		assert.NoError(t, err)

		p, err := repo.GetByID(ctx, product.ID)
		assert.NoError(t, err)
		assert.Equal(t, 5500, p.Price)
		assert.Equal(t, 8, p.Stock)
	})

	t.Run("Delete Success", func(t *testing.T) {
		err := repo.Delete(ctx, product.ID)
		assert.NoError(t, err)

		_, err = repo.GetByID(ctx, product.ID)
		assert.ErrorIs(t, err, model.ErrProductNotFound)
	})
}

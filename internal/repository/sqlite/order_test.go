// Package sqlite_test содержит интеграционные тесты для слоя доступа к данным (DAL) на базе SQLite.
// Файл order_test.go тестирует жизненный цикл заказов, включая транзакции.
package sqlite_test

import (
	"testing"

	"github.com/antonlearn/go-shop-backend/internal/model"
	"github.com/antonlearn/go-shop-backend/internal/repository/sqlite"
	"github.com/stretchr/testify/assert"
)

// TestSQLiteOrderRepository проверяет логику работы с заказами в полной изоляции.
func TestSQLiteOrderRepository(t *testing.T) {
	// Инициализируем изолированную in-memory БД
	db, log := setupTestDB(t)

	orderRepo := sqlite.NewSQLiteOrderRepository(db, log)
	ctx := t.Context()

	// Фикстурные данные
	const (
		testUserID1   = 1
		testUserID2   = 2
		testProductID = 1
		testPrice     = 1500.0
	)

	// 1. ПОДГОТОВКА ДАННЫХ
	seedUser(t, db, testUserID1, "user1@example.com")
	seedUser(t, db, testUserID2, "user2@example.com") // Второй пользователь для тестов безопасности
	seedProduct(t, db, testProductID, "Gaming Mouse", testPrice)
	seedCartItem(t, db, testUserID1, testProductID, 2)

	var createdOrder *model.Order

	// 2. СЦЕНАРИИ
	t.Run("Create Success", func(t *testing.T) {
		order, err := orderRepo.Create(ctx, testUserID1)

		assert.NoError(t, err)
		assert.NotNil(t, order)
		assert.Equal(t, testUserID1, order.UserID)
		assert.Equal(t, 3000, order.TotalPrice)
		assert.Equal(t, model.StatusCreated, order.Status)

		createdOrder = order
	})

	t.Run("Cart Cleared After Create", func(t *testing.T) {
		var count int
		err := db.Get(&count, "SELECT COUNT(*) FROM cart_items WHERE user_id = ?", testUserID1)
		assert.NoError(t, err)
		assert.Equal(t, 0, count, "корзина должна быть очищена после создания заказа")
	})

	t.Run("GetByID Success", func(t *testing.T) {
		order, err := orderRepo.GetByID(ctx, createdOrder.ID, testUserID1)

		assert.NoError(t, err)
		assert.NotNil(t, order)
		assert.Equal(t, createdOrder.ID, order.ID)
		assert.Len(t, order.Items, 1)
		assert.Equal(t, "Gaming Mouse", order.Items[0].ProductName)
		assert.Equal(t, 1500, order.Items[0].Price)
	})

	t.Run("GetByID Unauthorized Access", func(t *testing.T) {
		// Попытка получить чужой заказ
		_, err := orderRepo.GetByID(ctx, createdOrder.ID, testUserID2)
		assert.ErrorIs(t, err, model.ErrOrderNotFound, "чужой заказ не должен быть доступен")
	})

	t.Run("GetByID Not Found", func(t *testing.T) {
		_, err := orderRepo.GetByID(ctx, 99999, testUserID1)
		assert.ErrorIs(t, err, model.ErrOrderNotFound)
	})

	t.Run("GetByUserID Success", func(t *testing.T) {
		orders, err := orderRepo.GetByUserID(ctx, testUserID1)

		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(orders), 1)
		assert.Equal(t, createdOrder.ID, orders[0].ID)
	})

	t.Run("GetByUserID Empty History", func(t *testing.T) {
		orders, err := orderRepo.GetByUserID(ctx, testUserID2) // У юзера 2 нет заказов

		assert.NoError(t, err)
		assert.Empty(t, orders, "список заказов должен быть пустым, а не nil/ошибка")
	})

	t.Run("Create Fails With Empty Cart", func(t *testing.T) {
		_, err := orderRepo.Create(ctx, testUserID1)
		assert.ErrorIs(t, err, model.ErrCartIsEmpty)
	})
}

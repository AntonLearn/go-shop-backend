// Package sqlite_test содержит интеграционные тесты для слоя доступа к данным (DAL) на базе SQLite.
// В данном файле тестируется логика управления корзиной (добавление, изменение объемов, удаление).
// Изоляция данных обеспечивается техникой прямого SQL-занесения фикстур (Raw SQL Seeding).
package sqlite_test

import (
	"testing"

	"github.com/antonlearn/go-shop-backend/internal/model"
	"github.com/antonlearn/go-shop-backend/internal/repository/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSQLiteCartRepository агрегирует тест-кейсы проверки логики работы корзины в изолированной in-memory БД.
func TestSQLiteCartRepository(t *testing.T) {
	// Инициализируем полностью изолированную инстанс-БД в памяти (RAM) для текущего теста.
	db, log := setupTestDB(t)

	// Создаем тестируемый репозиторий. Сторонние репозитории игнорируются для соблюдения изоляции.
	cartRepo := sqlite.NewSQLiteCartRepository(db, log)
	ctx := t.Context()

	// Объявляем константные фикстурные ID для точности связей данных в рамках тест-сьюта
	const (
		testUserID    = 42
		testProductID = 7
		testPrice     = 3000.00
	)

	// Наполняем базу созависимыми сущностями напрямую через независимые SQL-хелперы (Raw SQL Seeding)
	seedUser(t, db, testUserID, "cart_owner@test.com")
	seedProduct(t, db, testProductID, "Игровая Мышь", testPrice)

	// === ПОЗИТИВНЫЕ СЦЕНАРИИ ===

	t.Run("Add Item and Upsert Check", func(t *testing.T) {
		item := &model.CartItem{
			UserID:    testUserID,
			ProductID: testProductID,
			Quantity:  2,
		}

		// Сценарий 1.1: Первичное добавление товара в корзину пользователя
		err := cartRepo.Add(ctx, item)
		assert.NoError(t, err)

		// Сценарий 1.2: Повторное добавление (проверка Smart Upsert / ON CONFLICT на уровне СУБД)
		err = cartRepo.Add(ctx, item)
		assert.NoError(t, err)

		// Валидация: Должна остаться одна запись, количество просуммировано (2+2=4), цена вычислена базой
		items, err := cartRepo.GetByUserID(ctx, testUserID)
		assert.NoError(t, err)
		require.Len(t, items, 1)

		assert.Equal(t, testProductID, items[0].ProductID)
		assert.Equal(t, "Игровая Мышь", items[0].ProductName)
		assert.Equal(t, 4, items[0].Quantity)
		assert.Equal(t, testPrice, items[0].Price)
		assert.Equal(t, 12000.00, items[0].TotalPrice) // Ожидаем расчет формулы: 3000 * 4
	})

	t.Run("Delete Item Success", func(t *testing.T) {
		// Сценарий 2: Успешное удаление существующей позиции из корзины
		err := cartRepo.Delete(ctx, testUserID, testProductID)
		assert.NoError(t, err)

		// Валидация: Метод выборки обязан вернуть пустой слайс без записей
		items, err := cartRepo.GetByUserID(ctx, testUserID)
		assert.NoError(t, err)
		assert.Empty(t, items)
	})

	// === НЕГАТИВНЫЕ СЦЕНАРИИ ===

	t.Run("GetByUserID Empty Cart", func(t *testing.T) {
		// Сценарий 3: Запрос корзины для пользователя, у которого нет добавленных позиций
		items, err := cartRepo.GetByUserID(ctx, 99999)

		// Валидация: Ошибки отсутствия строк быть не должно, отдаем пустой слайс
		assert.NoError(t, err)
		assert.Len(t, items, 0)
	})

	t.Run("Delete Item Not Found", func(t *testing.T) {
		// Сценарий 4: Попытка удалить из корзины товар, которого там заведомо нет (уже удален на шаге 2)
		err := cartRepo.Delete(ctx, testUserID, testProductID)

		// Валидация: Проверяем обработку условия RowsAffected == 0 и маппинг в доменную ошибку "не найдено"
		assert.ErrorIs(t, err, model.ErrCartItemNotFound)
	})
}

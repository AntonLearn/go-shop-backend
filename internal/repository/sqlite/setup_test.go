// Package sqlite_test инкапсулирует конфигурацию, фикстуры и вспомогательные утилиты для тестирования DAL.
// Содержит общие методы развертывания изолированных in-memory инстансов базы данных
// и методы наполнения СУБД созависимыми данными напрямую через низкоуровневые SQL-запросы.
package sqlite_test

import (
	"fmt"
	"testing"

	"github.com/antonlearn/go-shop-backend/internal/db/migrations"
	"github.com/antonlearn/go-shop-backend/internal/repository/sqlite"
	"github.com/antonlearn/go-shop-backend/pkg/logger"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

// setupTestDB разворачивает изолированную базу данных прямо в оперативной памяти (RAM) индивидуально для каждого теста.
func setupTestDB(t *testing.T) (*sqlx.DB, *logger.Logger) {
	t.Helper()

	log, err := logger.New("dev", "Stdout")
	require.NoError(t, err)

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := sqlite.InitDB(dsn, log)
	require.NoError(t, err)

	err = sqlite.RunMigrations(t.Context(), db.DB, log, migrations.FS)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = db.Close()
	})

	return db, log
}

// seedUser осуществляет жесткую вставку тестового пользователя напрямую через SQL.
func seedUser(t *testing.T, db *sqlx.DB, id int, email string) {
	t.Helper()
	query := `INSERT INTO users (id, email, name, password_hash, role) VALUES (?, ?, ?, ?, ?)`
	_, err := db.Exec(query, id, email, "Test User", "hash_123", "client")
	require.NoError(t, err, "не удалось подготовить тестового пользователя")
}

// seedProduct осуществляет жесткую вставку тестового товара напрямую через SQL.
func seedProduct(t *testing.T, db *sqlx.DB, id int, name string, price float64) {
	t.Helper()
	query := `INSERT INTO products (id, name, description, price, stock) VALUES (?, ?, ?, ?, ?)`
	_, err := db.Exec(query, id, name, "Игровой девайс", price, 10)
	require.NoError(t, err, "не удалось подготовить тестовый товар")
}

// seedCartItem осуществляет прямую вставку записи в корзину для проверки процесса оформления заказа.
func seedCartItem(t *testing.T, db *sqlx.DB, userID, productID, quantity int) {
	t.Helper()
	query := `INSERT INTO cart_items (user_id, product_id, quantity) VALUES (?, ?, ?)`
	_, err := db.Exec(query, userID, productID, quantity)
	require.NoError(t, err, "не удалось подготовить запись в корзине")
}

// seedOrder осуществляет жесткую вставку заголовка заказа напрямую через SQL.
func seedOrder(t *testing.T, db *sqlx.DB, id, userID, totalPrice int, status string) {
	t.Helper()
	query := `INSERT INTO orders (id, user_id, total_price, status, created_at, updated_at) VALUES (?, ?, ?, ?, datetime('now'), datetime('now'))`
	_, err := db.Exec(query, id, userID, totalPrice, status)
	require.NoError(t, err, "не удалось подготовить заголовок заказа")
}

// seedOrderItem осуществляет жесткую вставку позиции состава заказа напрямую через SQL.
func seedOrderItem(t *testing.T, db *sqlx.DB, orderID, productID, quantity, price int) {
	t.Helper()
	query := `INSERT INTO order_items (order_id, product_id, quantity, price) VALUES (?, ?, ?, ?)`
	_, err := db.Exec(query, orderID, productID, quantity, price)
	require.NoError(t, err, "не удалось подготовить позицию заказа")
}

// Package sqlite реализует слой доступа к данным для СУБД SQLite.
// Все методы соответствуют интерфейсам, определенным в пакете repository.
package sqlite

import (
	"context"

	"github.com/antonlearn/go-shop-backend/internal/model"
	"github.com/antonlearn/go-shop-backend/internal/repository"
	"github.com/antonlearn/go-shop-backend/pkg/logger"
	"github.com/jmoiron/sqlx"
)

// Гарантия компиляции: проверяем, что SQLiteOrderRepository реализует repository.OrderRepository.
var _ repository.OrderRepository = (*SQLiteOrderRepository)(nil)

// SQLiteOrderRepository отвечает за низкоуровневое взаимодействие с СУБД SQLite через sqlx.
type SQLiteOrderRepository struct {
	db  *sqlx.DB
	log *logger.Logger
}

// NewSQLiteOrderRepository принимает готовый пул *sqlx.DB и логгер.
func NewSQLiteOrderRepository(db *sqlx.DB, log *logger.Logger) *SQLiteOrderRepository {
	return &SQLiteOrderRepository{
		db:  db,
		log: log,
	}
}

// Create оформляет заказ внутри ACID-транзакции.
func (r *SQLiteOrderRepository) Create(ctx context.Context, userID int) (*model.Order, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		r.log.ErrorContext(ctx, "не удалось открыть транзакцию для создания заказа", "user_id", userID, "error", err)
		return nil, err
	}
	defer tx.Rollback()

	// 1. Извлекаем текущее содержимое корзины пользователя
	cartQuery := `
		SELECT c.product_id, p.price, c.quantity
		FROM cart_items c
		JOIN products p ON c.product_id = p.id
		WHERE c.user_id = :user_id
	`

	namedQuery, args, err := sqlx.Named(cartQuery, map[string]any{"user_id": userID})
	if err != nil {
		r.log.ErrorContext(ctx, "ошибка парсинга именованного запроса корзины", "user_id", userID, "error", err)
		return nil, err
	}

	rows, err := tx.QueryxContext(ctx, tx.Rebind(namedQuery), args...)
	if err != nil {
		r.log.ErrorContext(ctx, "ошибка получения товаров из корзины внутри транзакции заказа", "user_id", userID, "error", err)
		return nil, err
	}
	defer rows.Close()

	type cartItemSnapshot struct {
		ProductID int `db:"product_id"`
		Price     int `db:"price"`
		Quantity  int `db:"quantity"`
	}

	var cartItems []cartItemSnapshot
	for rows.Next() {
		var item cartItemSnapshot
		if err := rows.StructScan(&item); err != nil {
			r.log.ErrorContext(ctx, "ошибка маппинга товара из корзины внутри транзакции заказа", "user_id", userID, "error", err)
			return nil, err
		}
		cartItems = append(cartItems, item)
	}

	if err := rows.Err(); err != nil {
		r.log.ErrorContext(ctx, "ошибка при итерации по товарам корзины в транзакции заказа", "user_id", userID, "error", err)
		return nil, err
	}

	if len(cartItems) == 0 {
		r.log.DebugContext(ctx, "попытка оформить заказ с пустой корзиной", "user_id", userID)
		return nil, model.ErrCartIsEmpty
	}

	// 2. Рассчитываем общую стоимость
	totalPrice := 0
	for _, item := range cartItems {
		totalPrice += item.Price * item.Quantity
	}

	// 3. Создаем головную запись заказа
	orderQuery := `
		INSERT INTO orders (user_id, total_price, status)
		VALUES (:user_id, :total_price, :status)
	`
	orderParams := map[string]any{
		"user_id":     userID,
		"total_price": totalPrice,
		"status":      model.StatusCreated,
	}

	res, err := tx.NamedExecContext(ctx, orderQuery, orderParams)
	if err != nil {
		r.log.ErrorContext(ctx, "ошибка вставки заголовка заказа", "user_id", userID, "error", err)
		return nil, err
	}

	orderID64, err := res.LastInsertId()
	if err != nil {
		r.log.ErrorContext(ctx, "не удалось получить ID созданной записи заказа", "user_id", userID, "error", err)
		return nil, err
	}
	orderID := int(orderID64)

	// 4. Переносим позиции в order_items
	itemQuery := `
		INSERT INTO order_items (order_id, product_id, quantity, price)
		VALUES (:order_id, :product_id, :quantity, :price)
	`
	for _, item := range cartItems {
		itemParams := map[string]any{
			"order_id":   orderID,
			"product_id": item.ProductID,
			"quantity":   item.Quantity,
			"price":      item.Price,
		}
		if _, err := tx.NamedExecContext(ctx, itemQuery, itemParams); err != nil {
			r.log.ErrorContext(ctx, "ошибка добавления позиции в заказ", "order_id", orderID, "product_id", item.ProductID, "error", err)
			return nil, err
		}
	}

	// 5. Очищаем корзину
	clearQuery := `DELETE FROM cart_items WHERE user_id = :user_id`
	if _, err := tx.NamedExecContext(ctx, clearQuery, map[string]any{"user_id": userID}); err != nil {
		r.log.ErrorContext(ctx, "ошибка очистки корзины после успешного создания заказа", "user_id", userID, "order_id", orderID, "error", err)
		return nil, err
	}

	// 6. Коммит
	if err := tx.Commit(); err != nil {
		r.log.ErrorContext(ctx, "ошибка коммита транзакции создания заказа", "order_id", orderID, "error", err)
		return nil, err
	}

	r.log.InfoContext(ctx, "заказ успешно оформлен", "user_id", userID, "order_id", orderID, "total_price", totalPrice)

	return &model.Order{
		ID:         orderID,
		UserID:     userID,
		TotalPrice: totalPrice,
		Status:     model.StatusCreated,
	}, nil
}

// GetByID возвращает детальную информацию о конкретном заказе.
func (r *SQLiteOrderRepository) GetByID(ctx context.Context, orderID, userID int) (*model.OrderOutput, error) {
	orderQuery := `SELECT id, total_price, status, created_at FROM orders WHERE id = :id AND user_id = :user_id LIMIT 1`

	rows, err := r.db.NamedQueryContext(ctx, orderQuery, map[string]any{"id": orderID, "user_id": userID})
	if err != nil {
		r.log.ErrorContext(ctx, "ошибка выполнения SQL запроса при поиске заказа по id", "id", orderID, "user_id", userID, "error", err)
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			r.log.ErrorContext(ctx, "ошибка при итерации по строкам заголовка заказа", "id", orderID, "error", err)
			return nil, err
		}
		return nil, model.ErrOrderNotFound
	}

	var dbOrder struct {
		ID         int    `db:"id"`
		TotalPrice int    `db:"total_price"`
		Status     string `db:"status"`
		CreatedAt  string `db:"created_at"`
	}

	if err := rows.StructScan(&dbOrder); err != nil {
		return nil, err
	}

	parsedTime, err := parseSQLiteTime(dbOrder.CreatedAt)
	if err != nil {
		return nil, err
	}

	// Выгружаем позиции
	itemsQuery := `
		SELECT 
			oi.product_id,
			p.name AS product_name,
			oi.quantity,
			oi.price,
			(oi.price * oi.quantity) AS total_price
		FROM order_items oi
		JOIN products p ON oi.product_id = p.id
		WHERE oi.order_id = :order_id
	`
	itemRows, err := r.db.NamedQueryContext(ctx, itemsQuery, map[string]any{"order_id": dbOrder.ID})
	if err != nil {
		return nil, err
	}
	defer itemRows.Close()

	var items []*model.OrderItemDetail
	for itemRows.Next() {
		var item model.OrderItemDetail
		if err := itemRows.StructScan(&item); err != nil {
			return nil, err
		}
		items = append(items, &item)
	}

	return &model.OrderOutput{
		ID:         dbOrder.ID,
		TotalPrice: dbOrder.TotalPrice,
		Status:     dbOrder.Status,
		CreatedAt:  parsedTime,
		Items:      items,
	}, nil
}

// GetByUserID выгружает список всех заказов конкретного пользователя.
func (r *SQLiteOrderRepository) GetByUserID(ctx context.Context, userID int) ([]*model.Order, error) {
	query := `SELECT id, user_id, total_price, status, created_at, updated_at FROM orders WHERE user_id = :user_id ORDER BY created_at DESC`

	rows, err := r.db.NamedQueryContext(ctx, query, map[string]any{"user_id": userID})
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []*model.Order
	for rows.Next() {
		var dbOrder struct {
			ID         int    `db:"id"`
			UserID     int    `db:"user_id"`
			TotalPrice int    `db:"total_price"`
			Status     string `db:"status"`
			CreatedAt  string `db:"created_at"`
			UpdatedAt  string `db:"updated_at"`
		}

		if err := rows.StructScan(&dbOrder); err != nil {
			return nil, err
		}

		parsedCreatedAt, _ := parseSQLiteTime(dbOrder.CreatedAt)
		parsedUpdatedAt, _ := parseSQLiteTime(dbOrder.UpdatedAt)

		orders = append(orders, &model.Order{
			ID:         dbOrder.ID,
			UserID:     dbOrder.UserID,
			TotalPrice: dbOrder.TotalPrice,
			Status:     dbOrder.Status,
			CreatedAt:  parsedCreatedAt,
			UpdatedAt:  parsedUpdatedAt,
		})
	}
	return orders, nil
}

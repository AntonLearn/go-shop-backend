// Package sqlite реализует слой доступа к данным (Data Access Layer) для СУБД SQLite.
// В данном файле сосредоточена реализация методов работы с корзиной покупателя:
// добавление позиций с автосуммированием количества, удаление и выборка с агрегацией цен.
package sqlite

import (
	"context"

	"github.com/antonlearn/go-shop-backend/internal/model"
	"github.com/antonlearn/go-shop-backend/internal/repository"
	"github.com/antonlearn/go-shop-backend/pkg/logger"
	"github.com/jmoiron/sqlx"
)

// Гарантия компиляции: статическая проверка соответствия структуры интерфейсу репозитория.
var _ repository.CartRepository = (*SQLiteCartRepository)(nil)

// SQLiteCartRepository предоставляет методы работы с корзиной покупателя в терминах СУБД SQLite.
type SQLiteCartRepository struct {
	db  *sqlx.DB
	log *logger.Logger
}

// NewSQLiteCartRepository конструирует и возвращает экземпляр SQLite-репозитория корзины.
func NewSQLiteCartRepository(db *sqlx.DB, log *logger.Logger) *SQLiteCartRepository {
	return &SQLiteCartRepository{
		db:  db,
		log: log,
	}
}

// Add добавляет выбранный товар в корзину. Если товар данного типа уже присутствует у пользователя,
// срабатывает механика Smart Upsert (ON CONFLICT), суммирующая текущее количество с переданным.
func (r *SQLiteCartRepository) Add(ctx context.Context, item *model.CartItem) error {
	query := `
		INSERT INTO cart_items (user_id, product_id, quantity) 
		VALUES (:user_id, :product_id, :quantity)
		ON CONFLICT(user_id, product_id) 
		DO UPDATE SET quantity = quantity + excluded.quantity;
	`

	params := map[string]any{
		"user_id":    item.UserID,
		"product_id": item.ProductID,
		"quantity":   item.Quantity,
	}

	_, err := r.db.NamedExecContext(ctx, query, params)
	if err != nil {
		r.log.ErrorContext(ctx, "ошибка добавления или обновления товара в корзине", "user_id", item.UserID, "product_id", item.ProductID, "error", err)
		return err
	}

	r.log.DebugContext(ctx, "товар успешно добавлен или обновлен в корзине", "user_id", item.UserID, "product_id", item.ProductID, "quantity", item.Quantity)
	return nil
}

// GetByUserID извлекает структуру корзины пользователя с расчетом полной стоимости позиций
// на стороне базы данных посредством JOIN с таблицей продуктов.
func (r *SQLiteCartRepository) GetByUserID(ctx context.Context, userID int) ([]*model.CartOutputItem, error) {
	query := `
		SELECT 
			c.id AS cart_item_id,
			c.product_id AS product_id,
			p.name AS product_name,
			p.price AS price,
			c.quantity AS quantity,
			(p.price * c.quantity) AS total_price
		FROM cart_items c
		JOIN products p ON c.product_id = p.id
		WHERE c.user_id = :user_id
	`

	rows, err := r.db.NamedQueryContext(ctx, query, map[string]any{"user_id": userID})
	if err != nil {
		r.log.ErrorContext(ctx, "ошибка выполнения SQL запроса содержимого корзины", "user_id", userID, "error", err)
		return nil, err
	}
	defer rows.Close()

	var items []*model.CartOutputItem
	for rows.Next() {
		var item model.CartOutputItem
		if err := rows.StructScan(&item); err != nil {
			r.log.ErrorContext(ctx, "ошибка маппинга строки бд в структуру элемента корзины", "user_id", userID, "error", err)
			return nil, err
		}
		items = append(items, &item)
	}

	// Обязательная проверка на ошибки, возникшие в процессе итерации курсора
	if err := rows.Err(); err != nil {
		r.log.ErrorContext(ctx, "ошибка при итерации по строкам элементов корзины", "user_id", userID, "error", err)
		return nil, err
	}

	return items, nil
}

// Delete безвозвратно удаляет позицию товара из корзины конкретного пользователя.
// Если позиция отсутствовала, возвращает доменную ошибку модели model.ErrCartItemNotFound.
func (r *SQLiteCartRepository) Delete(ctx context.Context, userID, productID int) error {
	query := `DELETE FROM cart_items WHERE user_id = :user_id AND product_id = :product_id`

	params := map[string]any{
		"user_id":    userID,
		"product_id": productID,
	}

	res, err := r.db.NamedExecContext(ctx, query, params)
	if err != nil {
		r.log.ErrorContext(ctx, "ошибка удаления товара из корзины", "user_id", userID, "product_id", productID, "error", err)
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}

	// Если СУБД не затронула ни одной строки, значит товара в корзине не было
	if rowsAffected == 0 {
		return model.ErrCartItemNotFound
	}

	return nil
}

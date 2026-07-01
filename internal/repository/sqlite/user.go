// Package sqlite реализует слой доступа к данным для СУБД SQLite.
package sqlite

import (
	"context"

	"github.com/antonlearn/go-shop-backend/internal/model"
	"github.com/antonlearn/go-shop-backend/internal/repository"
	"github.com/antonlearn/go-shop-backend/pkg/logger"
	"github.com/jmoiron/sqlx"
)

// Гарантия компиляции: проверяем, что SQLiteUserRepository реализует repository.UserRepository.
var _ repository.UserRepository = (*SQLiteUserRepository)(nil)

// SQLiteUserRepository отвечает за низкоуровневое взаимодействие с СУБД SQLite.
type SQLiteUserRepository struct {
	db  *sqlx.DB
	log *logger.Logger
}

// dbUser — внутренняя структура для маппинга данных из БД.
type dbUser struct {
	ID           int    `db:"id"`
	Email        string `db:"email"`
	Name         string `db:"name"`
	PasswordHash string `db:"password_hash"`
	Role         string `db:"role"`
	CreatedAt    string `db:"created_at"`
}

// NewSQLiteUserRepository конструирует репозиторий пользователей.
func NewSQLiteUserRepository(db *sqlx.DB, log *logger.Logger) *SQLiteUserRepository {
	return &SQLiteUserRepository{
		db:  db,
		log: log,
	}
}

// GetByEmail осуществляет поиск пользователя по email.
func (r *SQLiteUserRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	query := `SELECT id, email, name, password_hash, role, created_at FROM users WHERE email = :email LIMIT 1`

	rows, err := r.db.NamedQueryContext(ctx, query, map[string]any{"email": email})
	if err != nil {
		r.log.ErrorContext(ctx, "ошибка SQL при поиске по email", "email", email, "error", err)
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, model.ErrUserNotFound
	}

	var u dbUser
	if err := rows.StructScan(&u); err != nil {
		return nil, err
	}

	parsedTime, err := parseSQLiteTime(u.CreatedAt)
	if err != nil {
		return nil, err
	}

	return &model.User{
		ID:           u.ID,
		Email:        u.Email,
		Name:         u.Name,
		PasswordHash: u.PasswordHash,
		Role:         u.Role,
		CreatedAt:    parsedTime,
	}, nil
}

// GetByID осуществляет поиск пользователя по ID.
func (r *SQLiteUserRepository) GetByID(ctx context.Context, id int) (*model.User, error) {
	query := `SELECT id, email, name, password_hash, role, created_at FROM users WHERE id = :id LIMIT 1`

	rows, err := r.db.NamedQueryContext(ctx, query, map[string]any{"id": id})
	if err != nil {
		r.log.ErrorContext(ctx, "ошибка SQL при поиске по id", "id", id, "error", err)
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, model.ErrUserNotFound
	}

	var u dbUser
	if err := rows.StructScan(&u); err != nil {
		return nil, err
	}

	parsedTime, err := parseSQLiteTime(u.CreatedAt)
	if err != nil {
		return nil, err
	}

	return &model.User{
		ID:           u.ID,
		Email:        u.Email,
		Name:         u.Name,
		PasswordHash: u.PasswordHash,
		Role:         u.Role,
		CreatedAt:    parsedTime,
	}, nil
}

// Create добавляет новую запись пользователя.
func (r *SQLiteUserRepository) Create(ctx context.Context, user *model.User) error {
	query := `INSERT INTO users (email, name, password_hash, role) VALUES (:email, :name, :password_hash, :role)`

	result, err := r.db.NamedExecContext(ctx, query, user)
	if err != nil {
		r.log.ErrorContext(ctx, "ошибка при создании пользователя", "email", user.Email, "error", err)
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	user.ID = int(id)

	return nil
}

// Delete удаляет пользователя по ID.
func (r *SQLiteUserRepository) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM users WHERE id = :id`

	result, err := r.db.NamedExecContext(ctx, query, map[string]any{"id": id})
	if err != nil {
		r.log.ErrorContext(ctx, "ошибка удаления пользователя", "id", id, "error", err)
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return model.ErrUserNotFound
	}

	return nil
}

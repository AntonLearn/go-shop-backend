// Package sqlite реализует слой доступа к данным для СУБД SQLite.
package sqlite

import (
	"context"

	"github.com/antonlearn/go-shop-backend/internal/model"
	"github.com/antonlearn/go-shop-backend/internal/repository"
	"github.com/antonlearn/go-shop-backend/pkg/logger"
	"github.com/jmoiron/sqlx"
)

// Гарантия компиляции: проверяем, что SQLiteSessionRepository реализует repository.SessionRepository.
var _ repository.SessionRepository = (*SQLiteSessionRepository)(nil)

// SQLiteSessionRepository отвечает за взаимодействие с СУБД SQLite.
type SQLiteSessionRepository struct {
	db  *sqlx.DB
	log *logger.Logger
}

// NewSQLiteSessionRepository конструирует репозиторий сессий.
func NewSQLiteSessionRepository(db *sqlx.DB, log *logger.Logger) *SQLiteSessionRepository {
	return &SQLiteSessionRepository{
		db:  db,
		log: log,
	}
}

// Create сохраняет новую сессию в таблицу.
func (r *SQLiteSessionRepository) Create(ctx context.Context, session *model.Session) error {
	query := `INSERT INTO sessions (user_id, token, expires_at) 
              VALUES (:user_id, :token, :expires_at)`

	params := map[string]any{
		"user_id":    session.UserID,
		"token":      session.RefreshToken,
		"expires_at": session.ExpiresAt,
	}

	_, err := r.db.NamedExecContext(ctx, query, params)
	if err != nil {
		r.log.ErrorContext(ctx, "ошибка при создании записи сессии", "user_id", session.UserID, "error", err)
		return err
	}

	return nil
}

// GetByToken осуществляет поиск активной сессии.
func (r *SQLiteSessionRepository) GetByToken(ctx context.Context, token string) (*model.Session, error) {
	query := `SELECT user_id, token, expires_at FROM sessions WHERE token = :token LIMIT 1`

	rows, err := r.db.NamedQueryContext(ctx, query, map[string]any{"token": token})
	if err != nil {
		r.log.ErrorContext(ctx, "ошибка SQL при поиске сессии", "error", err)
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, model.ErrSessionNotFound
	}

	var dbSession struct {
		UserID    int    `db:"user_id"`
		Token     string `db:"token"`
		ExpiresAt string `db:"expires_at"`
	}

	if err := rows.StructScan(&dbSession); err != nil {
		return nil, err
	}

	parsedTime, err := parseSQLiteTime(dbSession.ExpiresAt)
	if err != nil {
		return nil, err
	}

	return &model.Session{
		UserID:       dbSession.UserID,
		RefreshToken: dbSession.Token,
		ExpiresAt:    parsedTime,
	}, nil
}

// DeleteByToken удаляет сессию из базы данных.
func (r *SQLiteSessionRepository) DeleteByToken(ctx context.Context, token string) error {
	query := `DELETE FROM sessions WHERE token = :token`

	result, err := r.db.NamedExecContext(ctx, query, map[string]any{"token": token})
	if err != nil {
		r.log.ErrorContext(ctx, "ошибка при удалении сессии", "error", err)
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return model.ErrSessionNotFound
	}

	return nil
}

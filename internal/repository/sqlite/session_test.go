// Package sqlite_test содержит интеграционные тесты для слоя доступа к данным (DAL) на базе SQLite.
// Файл session_test.go тестирует жизненный цикл сессий пользователей.
package sqlite_test

import (
	"context"
	"testing"
	"time"

	"github.com/antonlearn/go-shop-backend/internal/model"
	"github.com/antonlearn/go-shop-backend/internal/repository/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSQLiteSessionRepository(t *testing.T) {
	db, log := setupTestDB(t)
	defer db.Close()

	userRepo := sqlite.NewSQLiteUserRepository(db, log)
	sessionRepo := sqlite.NewSQLiteSessionRepository(db, log)
	ctx := context.Background()

	// Подготовка: создаем пользователя для привязки сессии
	testUser := &model.User{
		Email:        "session.user@example.com",
		Name:         "Session Owner",
		PasswordHash: "hash",
		Role:         "client",
	}
	err := userRepo.Create(ctx, testUser)
	require.NoError(t, err)

	t.Run("Create Session Success", func(t *testing.T) {
		session := &model.Session{
			UserID:       testUser.ID,
			RefreshToken: "sample_refresh_token_string",
			// Округляем до секунды, так как SQLite часто не хранит миллисекунды
			ExpiresAt: time.Now().Add(24 * time.Hour).Round(time.Second),
		}

		err := sessionRepo.Create(ctx, session)
		assert.NoError(t, err)
	})

	t.Run("GetByToken Success", func(t *testing.T) {
		token := "sample_refresh_token_string"
		session, err := sessionRepo.GetByToken(ctx, token)

		assert.NoError(t, err)
		assert.NotNil(t, session)
		assert.Equal(t, token, session.RefreshToken)
		assert.Equal(t, testUser.ID, session.UserID)
	})

	t.Run("DeleteByToken Success", func(t *testing.T) {
		token := "sample_refresh_token_string"

		err := sessionRepo.DeleteByToken(ctx, token)
		assert.NoError(t, err)

		// Проверяем, что сессия удалена
		_, err = sessionRepo.GetByToken(ctx, token)
		assert.ErrorIs(t, err, model.ErrSessionNotFound)
	})

	t.Run("DeleteByToken Not Found", func(t *testing.T) {
		err := sessionRepo.DeleteByToken(ctx, "non-existent-token")
		assert.ErrorIs(t, err, model.ErrSessionNotFound)
	})
}

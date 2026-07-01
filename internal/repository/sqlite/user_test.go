// Package sqlite_test содержит интеграционные тесты для слоя доступа к данным (DAL) на базе SQLite.
// Файл user_test.go изолированно тестирует CRUD-операции репозитория пользователей.
package sqlite_test

import (
	"context"
	"testing"

	"github.com/antonlearn/go-shop-backend/internal/model"
	"github.com/antonlearn/go-shop-backend/internal/repository/sqlite" // Подключаем обновленный пакет реализации
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSQLiteUserRepository(t *testing.T) {
	db, log := setupTestDB(t)
	defer db.Close()

	repo := sqlite.NewSQLiteUserRepository(db, log)
	ctx := context.Background()

	t.Run("Create Success", func(t *testing.T) {
		user := &model.User{
			Email:        "test@example.com",
			Name:         "John Doe",
			PasswordHash: "hashed_string_here",
			Role:         "client",
		}

		err := repo.Create(ctx, user)
		assert.NoError(t, err)
		assert.NotZero(t, user.ID)
	})

	t.Run("GetByEmail Success", func(t *testing.T) {
		email := "test@example.com"
		user, err := repo.GetByEmail(ctx, email)

		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, email, user.Email)
	})

	t.Run("GetByEmail Not Found", func(t *testing.T) {
		user, err := repo.GetByEmail(ctx, "non-existent@example.com")
		assert.ErrorIs(t, err, model.ErrUserNotFound)
		assert.Nil(t, user)
	})

	t.Run("GetByID Success", func(t *testing.T) {
		user, err := repo.GetByID(ctx, 1)

		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, 1, user.ID)
	})

	t.Run("Delete Success", func(t *testing.T) {
		tmpUser := &model.User{
			Email:        "delete-me@example.com",
			Name:         "To Be Deleted",
			PasswordHash: "hash",
			Role:         "client",
		}
		err := repo.Create(ctx, tmpUser)
		require.NoError(t, err)

		err = repo.Delete(ctx, tmpUser.ID)
		assert.NoError(t, err)

		_, err = repo.GetByID(ctx, tmpUser.ID)
		assert.ErrorIs(t, err, model.ErrUserNotFound)
	})
}

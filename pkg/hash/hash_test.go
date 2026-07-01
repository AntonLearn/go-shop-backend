// Package hash_test содержит модульные тесты для проверки корректности
// работы криптографического компонента хеширования Bcrypt.
package hash_test

import (
	"testing"

	"github.com/antonlearn/go-shop-backend/pkg/hash"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestBcryptHasher(t *testing.T) {
	hasher := hash.NewBcryptHasher()
	password := "my_super_secret_password_123"

	t.Run("GenerateHash Success", func(t *testing.T) {
		hashedPassword, err := hasher.GenerateHash(password)

		assert.NoError(t, err)
		assert.NotEmpty(t, hashedPassword)

		// Bcrypt-хеши традиционно начинаются с префикса $2a$, $2b$ или $2y$ (в зависимости от версии)
		assert.Contains(t, hashedPassword, "$2a$", "результат должен быть валидным bcrypt-хешем")
	})

	t.Run("Compare Success", func(t *testing.T) {
		hashedPassword, err := hasher.GenerateHash(password)
		require.NoError(t, err)

		// Сравниваем правильный пароль с хешем
		err = hasher.Compare(hashedPassword, password)
		assert.NoError(t, err, "метод Compare должен подтвердить совпадение правильного пароля")
	})

	t.Run("Compare Failure Invalid Password", func(t *testing.T) {
		hashedPassword, err := hasher.GenerateHash(password)
		require.NoError(t, err)

		// Передаем заведомо неверный пароль
		err = hasher.Compare(hashedPassword, "wrong_password")

		// Проверяем, что возвращается именно оригинальная ошибка библиотеки bcrypt
		assert.ErrorIs(t, err, bcrypt.ErrMismatchedHashAndPassword, "метод должен вернуть ошибку несоответствия пароля")
	})

	t.Run("Empty Password Handle", func(t *testing.T) {
		// Проверяем, что библиотека корректно обрабатывает пустые строки
		hashedPassword, err := hasher.GenerateHash("")
		assert.NoError(t, err)
		assert.NotEmpty(t, hashedPassword)

		err = hasher.Compare(hashedPassword, "")
		assert.NoError(t, err)
	})
}

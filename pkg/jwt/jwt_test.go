// Package jwt содержит модульные тесты для проверки корректности генерации,
// валидации и обработки жизненного цикла криптографических токенов доступа.
package jwt

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitJWT(t *testing.T) {
	// Сбрасываем секрет перед тестом для чистоты проверки
	jwtSecret = nil

	t.Run("Empty Secret Error", func(t *testing.T) {
		err := InitJWT("")
		assert.ErrorIs(t, err, ErrEmptySecret)
		assert.Nil(t, jwtSecret)
	})

	t.Run("Valid Secret Success", func(t *testing.T) {
		secretKey := "super-secure-app-secret-key"
		err := InitJWT(secretKey)

		assert.NoError(t, err)
		assert.Equal(t, []byte(secretKey), jwtSecret, "секрет внутри пакета должен совпадать с переданным")
	})
}

func TestGenerateAndValidateToken_Success(t *testing.T) {
	err := InitJWT("my-awesome-test-secret-key-12345")
	require.NoError(t, err)

	userID := 42
	email := "developer@example.com"
	role := "admin"
	ttl := time.Hour

	// 1. Тестируем генерацию токена
	token, err := GenerateToken(userID, email, role, ttl)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// 2. Тестируем валидацию этого же токена
	claims, err := ValidateToken(token)
	assert.NoError(t, err)
	assert.NotNil(t, claims)

	// 3. Проверяем точность сохранения полезной нагрузки (Claims)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, email, claims.Email)
	assert.Equal(t, role, claims.Role)
}

func TestToken_ErrorsAndEdgeCases(t *testing.T) {
	t.Run("Secret Not Initialized", func(t *testing.T) {
		// Принудительно очищаем секрет пакета
		jwtSecret = nil

		token, err := GenerateToken(1, "user@test.com", "client", time.Hour)
		assert.ErrorIs(t, err, ErrSecretNotInitialized)
		assert.Empty(t, token)

		claims, err := ValidateToken("any-token-string")
		assert.ErrorIs(t, err, ErrSecretNotInitialized)
		assert.Nil(t, claims)
	})

	t.Run("Invalid Token Format", func(t *testing.T) {
		err := InitJWT("active-secret")
		require.NoError(t, err)

		// Передаем строку, не являющуюся валидным JWT структурой
		claims, err := ValidateToken("not.a.valid.jwt.token.string")
		assert.ErrorIs(t, err, ErrInvalidToken)
		assert.Nil(t, claims)
	})

	t.Run("Expired Token Handling", func(t *testing.T) {
		err := InitJWT("active-secret")
		require.NoError(t, err)

		// Генерируем токен с отрицательным TTL (он протух 1 час назад)
		token, err := GenerateToken(1, "old@session.com", "client", -time.Hour)
		require.NoError(t, err)

		// Валидация должна вернуть ошибку инвалидного/истекшего токена
		claims, err := ValidateToken(token)
		assert.ErrorIs(t, err, ErrInvalidToken)
		assert.Nil(t, claims)
	})
}

func TestTokenManager_InterfaceImplementation(t *testing.T) {
	err := InitJWT("token-manager-secret-key")
	require.NoError(t, err)

	// Проверяем инициализацию структуры структуры-менеджера
	manager := NewTokenManager()
	require.NotNil(t, manager)

	userID := 100
	email := "manager@shop.com"
	role := "manager"

	// Генерируем через метод структуры (для DI сервисов)
	token, err := manager.GenerateToken(userID, email, role, time.Minute)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// Проверяем, что сгенерированный токен успешно парсится стандартным валидатором
	claims, err := ValidateToken(token)
	assert.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, email, claims.Email)
	assert.Equal(t, role, claims.Role)
}

// Package handler_test содержит модульные тесты для проверки утилит работы
// с контекстом пользователя и единых хелперов ответа транспортного слоя.
package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/antonlearn/go-shop-backend/internal/handler"
	"github.com/antonlearn/go-shop-backend/pkg/logger"
)

// TestContextHelpers верифицирует корректность добавления и безопасного
// извлечения доменных данных пользователя (ID, роль, Email) из context.Context.
func TestContextHelpers(t *testing.T) {
	t.Run("Успешная запись и чтение UserID", func(t *testing.T) {
		// Arrange
		expectedID := 123
		ctx := context.Background()

		// Act
		ctx = handler.ContextWithUserID(ctx, expectedID)
		actualID, ok := handler.GetUserID(ctx)

		// Assert
		assert.True(t, ok)
		assert.Equal(t, expectedID, actualID)
	})

	t.Run("UserID отсутствует в контексте", func(t *testing.T) {
		// Arrange & Act
		actualID, ok := handler.GetUserID(context.Background())

		// Assert
		assert.False(t, ok)
		assert.Zero(t, actualID)
	})

	t.Run("Успешная запись и чтение UserRole", func(t *testing.T) {
		// Arrange
		expectedRole := "admin"
		ctx := context.Background()

		// Act
		ctx = handler.ContextWithUserRole(ctx, expectedRole)
		actualRole, ok := handler.GetUserRole(ctx)

		// Assert
		assert.True(t, ok)
		assert.Equal(t, expectedRole, actualRole)
	})

	t.Run("UserRole отсутствует в контексте", func(t *testing.T) {
		// Arrange & Act
		actualRole, ok := handler.GetUserRole(context.Background())

		// Assert
		assert.False(t, ok)
		assert.Empty(t, actualRole)
	})

	t.Run("Успешная запись и чтение UserEmail", func(t *testing.T) {
		// Arrange
		expectedEmail := "test@example.com"
		ctx := context.Background()

		// Act
		ctx = handler.ContextWithUserEmail(ctx, expectedEmail)
		actualEmail, ok := handler.GetUserEmail(ctx)

		// Assert
		assert.True(t, ok)
		assert.Equal(t, expectedEmail, actualEmail)
	})

	t.Run("UserEmail отсутствует в контексте", func(t *testing.T) {
		// Arrange & Act
		actualEmail, ok := handler.GetUserEmail(context.Background())

		// Assert
		assert.False(t, ok)
		assert.Empty(t, actualEmail)
	})
}

// TestRespondWithJSON_Success проверяет штатное поведение хелпера ответов:
// установку заголовков, HTTP-статуса и корректную маршализацию переданных данных.
func TestRespondWithJSON_Success(t *testing.T) {
	// Arrange
	log, err := logger.New("local", "Stdout")
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	ctx := context.Background()
	status := http.StatusAccepted

	type testData struct {
		Foo string `json:"foo"`
		Bar int    `json:"bar"`
	}
	payload := testData{Foo: "baz", Bar: 42}

	// Act
	handler.RespondWithJSON(rr, ctx, log, status, payload)

	// Assert
	assert.Equal(t, status, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	var actualPayload testData
	err = json.Unmarshal(rr.Body.Bytes(), &actualPayload)
	require.NoError(t, err)
	assert.Equal(t, payload, actualPayload)
}

// TestRespondWithError проверяет стандартизированный формат тела JSON-ответа
// в случае возникновения ошибок на транспортном уровне.
func TestRespondWithError(t *testing.T) {
	// Arrange
	log, err := logger.New("local", "Stdout")
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	ctx := context.Background()
	status := http.StatusBadRequest
	errorMessage := "некорректный формат входных данных"

	// Act
	handler.RespondWithError(rr, ctx, log, status, errorMessage)

	// Assert
	assert.Equal(t, status, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	var actualResponse map[string]string
	err = json.Unmarshal(rr.Body.Bytes(), &actualResponse)
	require.NoError(t, err)

	assert.Contains(t, actualResponse, "error")
	assert.Equal(t, errorMessage, actualResponse["error"])
}

// TestRespondWithJSON_SerializationError проверяет устойчивость функции хелпера
// к критическим сбоям сериализации JSON (например, при передаче несериализуемых типов данных).
func TestRespondWithJSON_SerializationError(t *testing.T) {
	// Arrange
	log, err := logger.New("local", "Stdout")
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	ctx := context.Background()

	// Передаем канал (chan), который принципиально невозможно упаковать в JSON
	unsupportedPayload := make(chan int)

	// Act
	handler.RespondWithJSON(rr, ctx, log, http.StatusOK, unsupportedPayload)

	// Assert
	// Хелпер успевает вызвать WriteHeader до ошибки кодирования, поэтому статус проставится
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
	// Проверяем, что тело ответа осталось пустым или неполным, а приложение не упало в panic
	assert.Empty(t, rr.Body.String())
}

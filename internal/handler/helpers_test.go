// Package handler реализует транспортный слой приложения (HTTP).
// Файл helpers_test.go тестирует низкоуровневые системные утилиты пакета:
// безопасную потокозащищенную работу с контекстом и унифицированный вывод ответов.
package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/antonlearn/go-shop-backend/pkg/logger"
	"github.com/stretchr/testify/assert"
)

// TestContextHelpers проверяет атомарные операции записи и чтения
// контекста для предотвращения коллизий типов ключей.
func TestContextHelpers(t *testing.T) {
	baseCtx := context.Background()

	t.Run("User ID Context Operations", func(t *testing.T) {
		// 1. Записываем ID пользователя в контекст через хелпер
		ctx := ContextWithUserID(baseCtx, 42)

		// 2. Извлекаем данные обратно
		id, ok := GetUserID(ctx)

		// 3. Проверяем результат
		assert.True(t, ok, "ID должен успешно извлекаться")
		assert.Equal(t, 42, id)
	})

	t.Run("User Role Context Operations", func(t *testing.T) {
		// 1. Записываем роль в контекст
		ctx := ContextWithUserRole(baseCtx, "admin")

		// 2. Извлекаем роль обратно
		role, ok := GetUserRole(ctx)

		// 3. Проверяем результат
		assert.True(t, ok, "Роль должна успешно извлекаться")
		assert.Equal(t, "admin", role)
	})

	t.Run("User Email Context Operations", func(t *testing.T) {
		// 1. Записываем email в контекст
		ctx := ContextWithUserEmail(baseCtx, "developer@shop.com")

		// 2. Извлекаем email обратно
		email, ok := GetUserEmail(ctx)

		// 3. Проверяем результат
		assert.True(t, ok, "Email должен успешно извлекаться")
		assert.Equal(t, "developer@shop.com", email)
	})

	t.Run("Missing Values In Context", func(t *testing.T) {
		// Проверяем защитное поведение хелперов при отсутствии данных
		emptyCtx := context.Background()

		id, okID := GetUserID(emptyCtx)
		role, okRole := GetUserRole(emptyCtx)
		email, okEmail := GetUserEmail(emptyCtx)

		// Проверяем, что хелперы не падают в nil-pointer panic, а возвращают дефолты
		assert.False(t, okID)
		assert.Zero(t, id)

		assert.False(t, okRole)
		assert.Empty(t, role)

		assert.False(t, okEmail)
		assert.Empty(t, email)
	})
}

// TestResponseHelpers проверяет корректность работы функций форматирования вывода в HTTP-транспорт.
func TestResponseHelpers(t *testing.T) {
	log, _ := logger.New("Console", "INFO")
	ctx := context.Background()

	t.Run("Success respondWithJSON", func(t *testing.T) {
		// Инициализируем инструмент записи ответа (Recorder)
		rr := httptest.NewRecorder()
		testData := map[string]int{"product_id": 99, "quantity": 5}

		// Вызываем пакетный JSON-хелпер
		respondWithJSON(rr, ctx, log, http.StatusAccepted, testData)

		// Проверяем статус, заголовки и валидность сериализованного JSON
		assert.Equal(t, http.StatusAccepted, rr.Code)
		assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

		var actualBody map[string]int
		err := json.Unmarshal(rr.Body.Bytes(), &actualBody)

		assert.NoError(t, err, "Тело ответа должно быть валидным JSON")
		assert.Equal(t, 99, actualBody["product_id"])
		assert.Equal(t, 5, actualBody["quantity"])
	})

	t.Run("Error respondWithError", func(t *testing.T) {
		rr := httptest.NewRecorder()
		expectedErrorMessage := "неверный формат входных данных DTO"

		// Вызываем хелпер стандартизированной ошибки
		respondWithError(rr, ctx, log, http.StatusBadRequest, expectedErrorMessage)

		// Проверяем соответствие контракту ошибок {"error": "сообщение"}
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

		var errorResponse map[string]string
		err := json.Unmarshal(rr.Body.Bytes(), &errorResponse)

		assert.NoError(t, err)
		assert.Contains(t, errorResponse, "error")
		assert.Equal(t, expectedErrorMessage, errorResponse["error"])
	})

	t.Run("JSON Serialization Failure", func(t *testing.T) {
		rr := httptest.NewRecorder()

		// Каналы (chan) невозможно сериализовать в JSON.
		// Передача такого объекта гарантированно вызовет ошибку внутри json.NewEncoder.Encode()
		unserializableData := make(chan int)

		// Проверяем, что хелпер обрабатывает ошибку внутри и не падает в панику
		assert.NotPanics(t, func() {
			respondWithJSON(rr, ctx, log, http.StatusOK, unserializableData)
		})
	})
}

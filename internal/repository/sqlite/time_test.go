// Package sqlite предоставляет реализацию слоя доступа к данным (DAL) для СУБД SQLite.
// Файл time_test.go содержит white-box unit-тесты для внутренних утилит пакета,
// такие как парсинг специфичных форматов времени SQLite.
package sqlite

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestParseSQLiteTime проверяет корректность преобразования строковых представлений
// даты и времени из SQLite в стандартный тип time.Time языка Go.
func TestParseSQLiteTime(t *testing.T) {
	// Табличное тестирование (Table-Driven Tests) позволяет легко расширять
	// список поддерживаемых форматов времени.
	tests := []struct {
		name     string
		input    string
		expected time.Time
	}{
		{
			name:     "Standard SQLite format",
			input:    "2026-06-26 17:50:10",
			expected: time.Date(2026, 6, 26, 17, 50, 10, 0, time.UTC),
		},
		{
			name:     "RFC3339 format",
			input:    "2026-06-26T17:50:10Z",
			expected: time.Date(2026, 6, 26, 17, 50, 10, 0, time.UTC),
		},
		{
			name:     "With monotonic clock removal",
			input:    "2026-06-26 17:50:10 m=+0.000000001",
			expected: time.Date(2026, 6, 26, 17, 50, 10, 0, time.UTC),
		},
		{
			name:     "Full format with nanoseconds",
			input:    "2026-06-26 17:50:10.123456789",
			expected: time.Date(2026, 6, 26, 17, 50, 10, 123456789, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseSQLiteTime(tt.input)

			assert.NoError(t, err, "Парсинг должен быть успешным для формата: %s", tt.input)

			// Используем .Equal() для сравнения, так как он корректно обрабатывает
			// возможные различия в Location (UTC/Local).
			assert.True(t, tt.expected.Equal(got), "Ожидалось %v, но получено %v", tt.expected, got)
		})
	}

	// Тест на негативный сценарий: некорректная строка
	t.Run("Invalid Format", func(t *testing.T) {
		_, err := parseSQLiteTime("invalid-date-string")
		assert.Error(t, err, "Функция должна возвращать ошибку при передаче некорректного формата")
	})
}

// Package logger содержит модульные тесты для проверки инициализации,
// выбора правильных обработчиков (текст/JSON) и управления жизненным циклом файлов логов.
package logger

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew_StdoutMode(t *testing.T) {
	t.Run("Development TextHandler", func(t *testing.T) {
		log, err := New("development", "Stdout")
		require.NoError(t, err)
		require.NotNil(t, log)

		// Проверяем, что в режиме Stdout файл не создается
		assert.Nil(t, log.file, "в режиме Stdout дескриптор файла должен быть nil")

		// Проверяем, что метод Close отрабатывает без ошибок
		err = log.Close()
		assert.NoError(t, err)
	})

	t.Run("Production JSONHandler", func(t *testing.T) {
		log, err := New("production", "Stdout")
		require.NoError(t, err)
		require.NotNil(t, log)
		assert.Nil(t, log.file)

		// Просто проверяем, что вызовы логов не паникуют
		log.Info("тестовое сообщение для проверки JSON формата в stdout")

		err = log.Close()
		assert.NoError(t, err)
	})
}

func TestNew_FileModes(t *testing.T) {
	t.Run("File Only Mode", func(t *testing.T) {
		log, err := New("development", "File")
		require.NoError(t, err)
		require.NotNil(t, log)
		require.NotNil(t, log.file, "в режиме File дескриптор файла обязан инициализироваться")

		// Запоминаем имя файла лога для последующей проверки и очистки
		logPath := log.file.Name()

		log.Info("запись в файл")

		// Закрываем логгер, чтобы сбросить буферы на диск и освободить файл
		err = log.Close()
		require.NoError(t, err)

		// Проверяем, что файл действительно создался на диске и не пустой
		fileInfo, err := os.Stat(logPath)
		assert.NoError(t, err, "файл лога должен физически существовать")
		assert.Greater(t, fileInfo.Size(), int64(0), "размер файла лога должен быть больше нуля")

		// Очищаем диск после теста
		err = os.Remove(logPath)
		assert.NoError(t, err, "не удалось удалить временный файл лога")
	})

	t.Run("Both Mode (File + Stdout)", func(t *testing.T) {
		log, err := New("production", "Both")
		require.NoError(t, err)
		require.NotNil(t, log)
		require.NotNil(t, log.file)

		logPath := log.file.Name()
		log.Info("запись в оба потока")

		err = log.Close()
		require.NoError(t, err)

		// Проверяем наличие файла
		assert.FileExists(t, logPath)

		// Читаем содержимое, чтобы убедиться, что там JSON (так как appMode = production)
		content, err := os.ReadFile(logPath)
		require.NoError(t, err)

		// Любая JSON-строка от slog начинается с открытия фигурной скобки
		assert.True(t, strings.HasPrefix(string(content), "{"), "в режиме production логи должны записываться в формате JSON")

		err = os.Remove(logPath)
		assert.NoError(t, err)
	})
}

func TestGenerateLocalFileName(t *testing.T) {
	ext := ".log"
	fileName := generateLocalFileName(ext)

	assert.True(t, strings.HasPrefix(fileName, "app_"), "имя файла должно начинаться префиксом app_")
	assert.Equal(t, ext, filepath.Ext(fileName), "расширение файла должно соответствовать переданному")
	assert.Len(t, fileName, 27, "длина сгенерированного имени файла должна быть фиксированной (app_YYYY-MM-DD_HH-MM-SS.log)")
}

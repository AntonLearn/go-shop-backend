// Package config_test предоставляет модульные тесты для пакета конфигурации приложения.
// Тестирование выполняется методом черного ящика (black-box testing) исключительно через
// публичный интерфейс пакета. Для тестирования неэкспортируемой функции dbAbsPath в
// изолированных сценариях используется экспортированный на этапе компиляции мост
// в файле export_test.go, если требуется прямая проверка детерминированного поведения.
package config_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/antonlearn/go-shop-backend/internal/config"
)

// setupEnvHelper очищает и подготавливает переменные окружения для изоляции тестов.
// Возвращает функцию восстановления исходного состояния окружения.
func setupEnvHelper(t *testing.T, envs map[string]string) func() {
	t.Helper()

	// Сохраняем старые значения, чтобы не нарушать параллельный запуск и состояние ОС
	oldEnvs := make(map[string]string)
	for key := range envs {
		if val, exists := os.LookupEnv(key); exists {
			oldEnvs[key] = val
		}
		// Временно очищаем перед установкой тестовых значений
		err := os.Unsetenv(key)
		require.NoError(t, err)
	}

	// Устанавливаем новые тестовые переменные
	for key, val := range envs {
		if val != "" {
			err := os.Setenv(key, val)
			require.NoError(t, err)
		}
	}

	// Функция восстановления среды (Teardown)
	return func() {
		for key := range envs {
			err := os.Unsetenv(key)
			require.NoError(t, err)
		}
		for key, val := range oldEnvs {
			err := os.Setenv(key, val)
			require.NoError(t, err)
		}
	}
}

// TestLoadConfig_Success проверяет успешные сценарии загрузки конфигурации
// для различных режимов работы приложения (local, production).
func TestLoadConfig_Success(t *testing.T) {
	// Для изоляции тестов от реального файла .env подменим рабочую директорию
	// или временно переименуем переменные, но так как godotenv игнорирует отсутствие .env,
	// тест стабилен при использовании системного окружения.

	t.Run("LocalMode_Defaults", func(t *testing.T) {
		// Arrange: задаем минимальный набор для локального режима
		cleanup := setupEnvHelper(t, map[string]string{
			"APP_MODE": "local",
			"DB_FILE":  "test_shop.db",
		})
		defer cleanup()

		// Act
		cfg, err := config.LoadConfig()

		// Assert
		require.NoError(t, err)
		require.NotNil(t, cfg)
		assert.Equal(t, "local", cfg.AppMode)
		assert.Equal(t, "8080", cfg.HTTP.Port) // Дефолт из default.go
		assert.Contains(t, cfg.DB.DBFile, "test_shop.db")
		assert.Equal(t, 5*time.Second, cfg.HTTP.ShutdownTimeout)
	})

	t.Run("ProductionMode_WithSecret", func(t *testing.T) {
		// Arrange: в продакшене обязательно проверяется JWT_SECRET
		cleanup := setupEnvHelper(t, map[string]string{
			"APP_MODE":             "production",
			"JWT_SECRET":           "super-secure-high-entropy-secret-key",
			"APP_PORT":             "9090",
			"APP_SHUTDOWN_TIMEOUT": "10s",
		})
		defer cleanup()

		// Act
		cfg, err := config.LoadConfig()

		// Assert
		require.NoError(t, err)
		require.NotNil(t, cfg)
		assert.Equal(t, "production", cfg.AppMode)
		assert.Equal(t, "super-secure-high-entropy-secret-key", cfg.JWTSecret)
		assert.Equal(t, "9090", cfg.HTTP.Port)
		assert.Equal(t, 10*time.Second, cfg.HTTP.ShutdownTimeout)
	})

	t.Run("Absolute_DB_Path", func(t *testing.T) {
		// Arrange: если передан абсолютный путь, он не должен склеиваться с os.Executable()
		absPath := filepath.FromSlash("/tmp/absolute_test_shop.db")
		if filepath.Separator == '\\' {
			absPath = `C:\absolute_test_shop.db`
		}

		cleanup := setupEnvHelper(t, map[string]string{
			"APP_MODE": "local",
			"DB_FILE":  absPath,
		})
		defer cleanup()

		// Act
		cfg, err := config.LoadConfig()

		// Assert
		require.NoError(t, err)
		assert.Equal(t, absPath, cfg.DB.DBFile)
	})
}

// TestLoadConfig_Errors проверяет валидацию конфигурации и обработку
// некорректных параметров (отсутствие секретов, невалидные режимы).
func TestLoadConfig_Errors(t *testing.T) {
	tests := []struct {
		name     string
		envs     map[string]string
		contains string
	}{
		{
			name: "Production_Missing_JWT_Secret",
			envs: map[string]string{
				"APP_MODE":   "production",
				"JWT_SECRET": "", // Пустой секрет вызовет панику/ошибку валидации
			},
			contains: "требуется уникальный и надежный JWT_SECRET",
		},
		{
			name: "Unknown_App_Mode",
			envs: map[string]string{
				"APP_MODE": "staging",
			},
			contains: "неизвестный режим staging",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			cleanup := setupEnvHelper(t, tt.envs)
			defer cleanup()

			// Act
			cfg, err := config.LoadConfig()

			// Assert
			assert.Error(t, err)
			assert.Nil(t, cfg)
			assert.Contains(t, err.Error(), tt.contains)
		})
	}
}

// TestLoadConfig_FallbackTimeout проверяет устойчивость парсинга к поврежденным
// или некорректным строкам тайм-аутов, гарантируя падение до значений по умолчанию.
func TestLoadConfig_FallbackTimeout(t *testing.T) {
	// Arrange: передаем заведомо некорректный формат длительности времени
	cleanup := setupEnvHelper(t, map[string]string{
		"APP_MODE":             "local",
		"APP_SHUTDOWN_TIMEOUT": "invalid-duration-string",
	})
	defer cleanup()

	// Act
	cfg, err := config.LoadConfig()

	// Assert
	require.NoError(t, err)
	require.NotNil(t, cfg)
	// Должно примениться дефолтное значение defaultShutdownTimeout из default.go
	assert.Equal(t, 5*time.Second, cfg.HTTP.ShutdownTimeout)
}

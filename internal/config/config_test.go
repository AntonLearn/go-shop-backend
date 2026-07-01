// Package config_test предоставляет комплексный набор модульных тестов для проверки
// механизмов инициализации, парсинга и валидации конфигурации приложения.
// Тестирование организовано по принципу "черного ящика" (black-box testing), обеспечивая
// проверку исключительно публичного контракта пакета без привязки к деталям реализации.
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

// setupEnvHelper изолирует окружение каждого теста, предотвращая взаимное влияние
// параллельных или последовательных запусков через глобальные переменные ОС.
func setupEnvHelper(t *testing.T, envs map[string]string) {
	t.Helper()

	// Список всех ключевых переменных, используемых пакетом конфигурации
	targetKeys := []string{
		"APP_MODE",
		"JWT_SECRET",
		"LOG_MODE",
		"DB_FILE",
		"APP_PORT",
		"APP_SHUTDOWN_TIMEOUT",
	}

	// Сохраняем текущее состояние операционной системы для последующего восстановления
	backup := make(map[string]string)
	for _, key := range targetKeys {
		if val, exists := os.LookupEnv(key); exists {
			backup[key] = val
			_ = os.Unsetenv(key)
		}
	}

	// Гарантируем очистку текущих и восстановление исходных переменных после завершения теста
	t.Cleanup(func() {
		for _, key := range targetKeys {
			_ = os.Unsetenv(key)
		}
		for key, val := range backup {
			_ = os.Setenv(key, val)
		}
	})

	// Устанавливаем новые тестовые значения
	for key, val := range envs {
		err := os.Setenv(key, val)
		require.NoError(t, err, "Не удалось настроить тестовое окружение для переменной: %s", key)
	}
}

// TestLoadConfig_OkLocal проверяет успешную инициализацию конфигурации
// со значениями по умолчанию в локальном режиме работы приложения.
func TestLoadConfig_OkLocal(t *testing.T) {
	// Arrange
	setupEnvHelper(t, map[string]string{
		"APP_MODE": "local",
	})

	// Act
	cfg, err := config.LoadConfig()

	// Assert
	require.NoError(t, err, "Загрузка конфигурации в режиме 'local' не должна вызывать ошибок")
	require.NotNil(t, cfg, "Конфигурация не должна быть nil при успешном разборе")

	assert.Equal(t, "local", cfg.AppMode)
	assert.Equal(t, "8080", cfg.HTTP.Port)
	assert.Equal(t, 5*time.Second, cfg.HTTP.ShutdownTimeout)
	assert.Equal(t, "Both", cfg.LogMode)
	assert.True(t, filepath.IsAbs(cfg.DB.DBFile), "Путь к базе данных должен быть приведен к абсолютному виду")
	assert.Contains(t, cfg.DB.DBFile, "shop.db")
}

// TestLoadConfig_OkProd проверяет корректную сборку конфигурации для production-среды
// при условии, что передан валидный и заполненный секретный ключ JWT.
func TestLoadConfig_OkProd(t *testing.T) {
	// Arrange
	setupEnvHelper(t, map[string]string{
		"APP_MODE":   "production",
		"JWT_SECRET": "super_secure_and_very_long_production_secret_key_2026",
		"APP_PORT":   "443",
	})

	// Act
	cfg, err := config.LoadConfig()

	// Assert
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, "production", cfg.AppMode)
	assert.Equal(t, "super_secure_and_very_long_production_secret_key_2026", cfg.JWTSecret)
	assert.Equal(t, "443", cfg.HTTP.Port)
}

// TestLoadConfig_ErrProdNoSecret верифицирует критическое требование безопасности:
// отказ запуска в режиме production, если не задана секретная строка подписи токенов.
func TestLoadConfig_ErrProdNoSecret(t *testing.T) {
	// Arrange
	setupEnvHelper(t, map[string]string{
		"APP_MODE": "production",
		// JWT_SECRET намеренно не задается или передается пустым
		"JWT_SECRET": "",
	})

	// Act
	cfg, err := config.LoadConfig()

	// Assert
	assert.Error(t, err, "Ожидалась ошибка валидации безопасности для production среды")
	assert.Nil(t, cfg, "Объект конфигурации должен быть nil в случае ошибки валидации")
	assert.Contains(t, err.Error(), "требуется уникальный и надежный JWT_SECRET")
}

// TestLoadConfig_ErrInvalidMode проверяет защиту бизнес-логики от некорректных
// или неподдерживаемых режимов окружения, переданных извне.
func TestLoadConfig_ErrInvalidMode(t *testing.T) {
	// Arrange
	setupEnvHelper(t, map[string]string{
		"APP_MODE": "staging",
	})

	// Act
	cfg, err := config.LoadConfig()

	// Assert
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "неизвестный режим staging")
}

// TestLoadConfig_TimeoutFallback проверяет отказоустойчивость парсинга временных
// интервалов: при некорректном формате таймаута система должна применить дефолтное значение.
func TestLoadConfig_TimeoutFallback(t *testing.T) {
	// Arrange
	setupEnvHelper(t, map[string]string{
		"APP_MODE":             "local",
		"APP_SHUTDOWN_TIMEOUT": "invalid_duration_string",
	})

	// Act
	cfg, err := config.LoadConfig()

	// Assert
	require.NoError(t, err)
	require.NotNil(t, cfg)
	// Должно восстановиться дефолтное значение (5 секунд), несмотря на ошибку парсинга
	assert.Equal(t, 5*time.Second, cfg.HTTP.ShutdownTimeout, "Не сработал fallback к значению по умолчанию для таймаута")
}

// TestLoadConfig_AbsoluteDBPath контролирует логику вычисления путей к инфраструктурным файлам:
// если передан изначально абсолютный путь к БД, он должен остаться неизменным.
func TestLoadConfig_AbsoluteDBPath(t *testing.T) {
	// Arrange
	targetAbsPath := filepath.Clean(os.TempDir() + string(filepath.Separator) + "custom_production_shop.db")

	setupEnvHelper(t, map[string]string{
		"APP_MODE": "local",
		"DB_FILE":  targetAbsPath,
	})

	// Act
	cfg, err := config.LoadConfig()

	// Assert
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, targetAbsPath, cfg.DB.DBFile, "Абсолютный путь к базе данных был искажен внутренней логикой")
}

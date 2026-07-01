// Package config_test реализует модульное тестирование подсистемы конфигурации.
// Используется подход Table-Driven Testing, который является стандартом для Go
// при тестировании функций с множеством входных параметров и валидацией.
package config_test

import (
	"strings"
	"testing"
	"time"

	"github.com/antonlearn/go-shop-backend/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	// Мы не используем ручную очистку через os.Unsetenv, так как t.Setenv
	// гарантирует автоматический откат изменений после завершения теста.

	type testCase struct {
		name        string
		setupEnv    func(t *testing.T)
		wantErr     bool
		errContains string
		validateCfg func(t *testing.T, cfg *config.Config)
	}

	tests := []testCase{
		{
			name: "Успешная загрузка с дефолтными значениями (режим local)",
			setupEnv: func(t *testing.T) {
				t.Setenv("APP_MODE", "local")
			},
			wantErr: false,
			validateCfg: func(t *testing.T, cfg *config.Config) {
				assert.Equal(t, "local", cfg.AppMode)
				assert.Equal(t, "Both", cfg.LogMode)
				assert.Equal(t, "8080", cfg.HTTP.Port)
				assert.Equal(t, 5*time.Second, cfg.HTTP.ShutdownTimeout)
				assert.True(t, strings.HasSuffix(cfg.DB.DBFile, "shop.db"))
			},
		},
		{
			name: "Успешная загрузка в режиме production с валидным JWT_SECRET",
			setupEnv: func(t *testing.T) {
				t.Setenv("APP_MODE", "production")
				t.Setenv("JWT_SECRET", "super-safe-and-long-secret-key-12345")
				t.Setenv("APP_PORT", "9090")
				t.Setenv("LOG_MODE", "Console")
			},
			wantErr: false,
			validateCfg: func(t *testing.T, cfg *config.Config) {
				assert.Equal(t, "production", cfg.AppMode)
				assert.Equal(t, "super-safe-and-long-secret-key-12345", cfg.JWTSecret)
				assert.Equal(t, "9090", cfg.HTTP.Port)
				assert.Equal(t, "Console", cfg.LogMode)
			},
		},
		{
			name: "Ошибка: режим production без указания JWT_SECRET",
			setupEnv: func(t *testing.T) {
				t.Setenv("APP_MODE", "production")
				t.Setenv("JWT_SECRET", "")
			},
			wantErr:     true,
			errContains: "требуется уникальный и надежный JWT_SECRET",
		},
		{
			name: "Ошибка: указан неизвестный режим работы приложения (AppMode)",
			setupEnv: func(t *testing.T) {
				t.Setenv("APP_MODE", "staging")
			},
			wantErr:     true,
			errContains: "неизвестный режим staging",
		},
		{
			name: "Успешный парсинг кастомного таймаута плавного завершения (ShutdownTimeout)",
			setupEnv: func(t *testing.T) {
				t.Setenv("APP_MODE", "local")
				t.Setenv("APP_SHUTDOWN_TIMEOUT", "15s")
			},
			wantErr: false,
			validateCfg: func(t *testing.T, cfg *config.Config) {
				assert.Equal(t, 15*time.Second, cfg.HTTP.ShutdownTimeout)
			},
		},
		{
			name: "Неверный формат таймаута завершения — откат на дефолт",
			setupEnv: func(t *testing.T) {
				t.Setenv("APP_MODE", "local")
				t.Setenv("APP_SHUTDOWN_TIMEOUT", "invalid_duration_string")
			},
			wantErr: false,
			validateCfg: func(t *testing.T, cfg *config.Config) {
				assert.Equal(t, 5*time.Second, cfg.HTTP.ShutdownTimeout)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Настраиваем окружение
			tt.setupEnv(t)

			// Выполняем тест
			cfg, err := config.LoadConfig()

			// Проверяем результат
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, cfg)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, cfg)
				if tt.validateCfg != nil {
					tt.validateCfg(t, cfg)
				}
			}
		})
	}
}

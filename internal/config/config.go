// Package config предоставляет централизованные механизмы для управления конфигурацией приложения.
// Данный файл отвечает за исполнительную логику пакета: определение строго типизированных
// структур конфигурации, чтение переменных из операционной системы и .env-файлов, безопасное
// приведение типов (включая парсинг временных интервалов time.Duration), динамический расчет
// абсолютных путей к файлам инфраструктуры (SQLite) и валидацию критически важных параметров
// безопасности в зависимости от режима окружения (production/local).
package config

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/joho/godotenv"
)

// DBConfig объединяет параметры подключения и состояние пула соединений базы данных.
type DBConfig struct {
	DBFile  string `yaml:"dbfile" env:"DB_FILE" env-default:"shop.db"`
	Connect *sql.DB
}

// HTTPConfig инкапсулирует сетевые параметры веб-сервера и настройки его жизненного цикла.
type HTTPConfig struct {
	Port            string
	ShutdownTimeout time.Duration
}

// Config является корневым объектом конфигурации (Composition Root Config) всего приложения.
type Config struct {
	AppMode   string
	JWTSecret string
	LogMode   string
	HTTP      HTTPConfig // Сгруппированные сетевые настройки сервера
	DB        DBConfig
}

// LoadConfig выполняет поэтапную сборку конфигурации, парсинг данных и валидацию режимов работы.
func LoadConfig() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		log.Printf("файл .env не найден или не прочитан. Используются системные переменные окружения")
	} else {
		log.Println("файл .env успешно обнаружен и загружен")
	}

	dbPath, err := dbAbsPath(getEnv("DB_FILE", defaultDBFile))
	if err != nil {
		return nil, fmt.Errorf("ошибка при определении пути к базе данных: %v", err)
	}

	// Безопасное чтение и парсинг таймаута плавного завершения работы
	shutdownTimeout := defaultShutdownTimeout
	if timeoutStr := getEnv("APP_SHUTDOWN_TIMEOUT", ""); timeoutStr != "" {
		if parsedDuration, err := time.ParseDuration(timeoutStr); err == nil {
			shutdownTimeout = parsedDuration
		} else {
			log.Printf("неверный формат APP_SHUTDOWN_TIMEOUT: %v. Применено значение по умолчанию", err)
		}
	}

	cfg := &Config{
		AppMode:   getEnv("APP_MODE", defaultAppMode),
		JWTSecret: getEnv("JWT_SECRET", ""),
		LogMode:   getEnv("LOG_MODE", defaultLogMode),
		HTTP: HTTPConfig{
			Port:            getEnv("APP_PORT", defaultServerPort),
			ShutdownTimeout: shutdownTimeout,
		},
		DB: DBConfig{
			DBFile: dbPath,
		},
	}

	switch cfg.AppMode {
	case "production":
		if cfg.JWTSecret == "" {
			return nil, fmt.Errorf("для режима %s требуется уникальный и надежный JWT_SECRET", cfg.AppMode)
		}
		return cfg, nil
	case "local":
		return cfg, nil
	default:
		return nil, fmt.Errorf("неизвестный режим %s", cfg.AppMode)
	}
}

// getEnv проверяет наличие переменной в ОС.
// Если переменная пустая или отсутствует, возвращает fallback.
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		return value
	}
	return fallback
}

// dbAbsPath возвращает абсолютный путь к файлу базы данных.
func dbAbsPath(dbFile string) (string, error) {
	// Если dbFile уже является абсолютным путем, возвращаем его без изменений
	if filepath.IsAbs(dbFile) {
		return dbFile, nil
	}

	// Определяем корневой каталог приложения относительно текущего исполняемого файла
	rootPath := "."
	exePath, err := os.Executable()
	if err != nil {
		return "", err
	}
	rootPath = filepath.Dir(exePath)

	// Вычисляем абсолютный путь к базе данных и возвращаем его
	return filepath.Join(rootPath, dbFile), nil
}

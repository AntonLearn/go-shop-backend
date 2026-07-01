// Package logger предоставляет структурированную реализацию уровневого логирования
// на базе slog.
package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"
)

// Logger оборачивает стандартный slog.Logger и хранит дескриптор файла.
type Logger struct {
	*slog.Logger
	file *os.File
}

// New инициализирует и возвращает настроенный экземпляр Logger на базе slog.
// Автоматически переключает формат (JSON в продакшене, Text в разработке).
func New(appMode, logMode string) (*Logger, error) {
	var file *os.File
	var err error
	var writer io.Writer

	// 1. Подготавливаем локальный файл логов, если требуется
	if logMode == "File" || logMode == "Both" || logMode == "" {
		logFileName := generateLocalFileName(".log")
		file, err = os.Create(logFileName)
		if err != nil {
			return nil, fmt.Errorf("failed to create log file %s: %w", logFileName, err)
		}
	}

	// 2. Настраиваем целевые потоки вывода
	switch logMode {
	case "File":
		writer = file
	case "Stdout":
		writer = os.Stdout
	default: // Режим "Both" или по умолчанию
		if file != nil {
			writer = io.MultiWriter(file, os.Stdout)
		} else {
			writer = os.Stdout
		}
	}

	// 3. Настраиваем опции slog
	opts := &slog.HandlerOptions{
		AddSource: true,           // Выводить файл и строку кода, где вызван лог
		Level:     slog.LevelInfo, // Минимальный уровень логов
	}

	// 4. Выбораем обработчик в зависимости от режима запуска приложения
	var handler slog.Handler
	if appMode == "production" {
		handler = slog.NewJSONHandler(writer, opts)
	} else {
		handler = slog.NewTextHandler(writer, opts)
	}

	return &Logger{
		Logger: slog.New(handler),
		file:   file,
	}, nil
}

// Close штатно закрывает дескриптор файла логов.
func (l *Logger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// generateLocalFileName создает уникальное имя файла, включая в него
// текущую UTC-метку времени для исключения риска перезаписи логов.
func generateLocalFileName(ext string) string {
	timeStamp := time.Now().UTC().Format("2006-01-02_15-04-05")
	return "app_" + timeStamp + ext
}

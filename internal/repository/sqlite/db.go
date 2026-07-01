// Package sqlite реализует слой доступа к данным (Data Access Layer) для СУБД SQLite.
// Предоставляет потокобезопасную инициализацию пула соединений с БД, конфигурацию
// специфичных для SQLite параметров работы, а также механизм наката встроенных SQL-миграций.
package sqlite

import (
	"context"
	"database/sql"
	"embed"
	"errors"

	"github.com/antonlearn/go-shop-backend/pkg/logger"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

// InitDB настраивает, инициализирует и возвращает расширенный пул соединений sqlx.DB.
// Принимает DSN строку. Автоматически выставляет лимит на 1 открытое соединение,
// что является критически важным для SQLite во избежание взаимных блокировок (database is locked) при записи.
func InitDB(dsn string, log *logger.Logger) (*sqlx.DB, error) {
	db, err := sqlx.Open("sqlite3", dsn+"?_parse_time=true")
	if err != nil {
		return nil, err
	}

	// Ограничиваем пул до 1 соединения ради потокобезопасности SQLite при операциях записи
	db.SetMaxOpenConns(1)

	// Проверяем фактическую доступность базы данных
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}

	return db, nil
}

// RunMigrations выполняет накат встроенных SQL-миграций на переданное активное соединение СУБД.
// Использует инструмент golang-migrate и драйвер iofs для чтения файлов из бинарника Go.
func RunMigrations(ctx context.Context, db *sql.DB, log *logger.Logger, embedFS embed.FS) error {
	log.InfoContext(ctx, "инициализация встроенных миграций СУБД SQLite")

	// Инициализируем источник миграций из встроенной файловой системы Go (embed.FS)
	sourceDriver, err := iofs.New(embedFS, ".")
	if err != nil {
		return err
	}

	// Подключаем драйвер миграций к уже открытому инстансу соединения, сохраняя изоляцию в памяти (:memory:)
	dbDriver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithInstance("iofs", sourceDriver, "sqlite3", dbDriver)
	if err != nil {
		return err
	}
	defer m.Close()

	// Запускаем процесс обновления схемы данных до актуальной версии
	if err := m.Up(); err != nil {
		// Ошибка migrate.ErrNoChange означает, что база уже обновлена, это штатная ситуация
		if errors.Is(err, migrate.ErrNoChange) {
			log.InfoContext(ctx, "схема базы данных актуальна")
			return nil
		}
		return err
	}

	log.InfoContext(ctx, "миграции успешно применены")
	return nil
}

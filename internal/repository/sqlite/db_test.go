// Package sqlite_test содержит интеграционные тесты для слоя доступа к данным (DAL) на базе SQLite.
// Данный файл проверяет работоспособность низкоуровневого драйвера подключения,
// валидацию параметров DSN и корректность жизненного цикла встроенного мигратора.
package sqlite_test

import (
	"embed"
	"fmt"
	"testing"

	"github.com/antonlearn/go-shop-backend/internal/repository/sqlite"
	"github.com/antonlearn/go-shop-backend/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Используем go:embed для вшивания тестовой схемы во встроенную FS драйвера тестов.
//
//go:embed 000001_test_init.up.sql
var testMigrationFS embed.FS

// TestInitDB_Success проверяет штатное создание пула соединений in-memory и успешный Ping.
func TestInitDB_Success(t *testing.T) {
	log, _ := logger.New("Console", "INFO")
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())

	db, err := sqlite.InitDB(dsn, log)
	require.NoError(t, err)
	defer db.Close()

	assert.NotNil(t, db)
	err = db.Ping()
	assert.NoError(t, err)
}

// TestInitDB_InvalidPath проверяет реакцию драйвера на передачу некорректных параметров конфигурации SQLite URI.
func TestInitDB_InvalidPath(t *testing.T) {
	log, _ := logger.New("Console", "INFO")
	_, err := sqlite.InitDB("file:invalid_db?mode=unknown", log)
	assert.Error(t, err)
}

// TestRunMigrations_Success тестирует накат миграций, а также повторный вызов (проверка обработки состояния "без изменений").
func TestRunMigrations_Success(t *testing.T) {
	log, _ := logger.New("Console", "INFO")
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())

	db, err := sqlite.InitDB(dsn, log)
	require.NoError(t, err)
	defer db.Close()

	// Первичный накат схемы таблиц
	err = sqlite.RunMigrations(t.Context(), db.DB, log, testMigrationFS)
	assert.NoError(t, err)

	// Повторный накат на уже готовую схему (должен обработать migrate.ErrNoChange Barnes и вернуть nil)
	err = sqlite.RunMigrations(t.Context(), db.DB, log, testMigrationFS)
	assert.NoError(t, err)
}

// TestRunMigrations_InvalidFS проверяет поведение мигратора при передаче пустой или поврежденной файловой системы iofs.
func TestRunMigrations_InvalidFS(t *testing.T) {
	log, _ := logger.New("Console", "INFO")
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())

	db, err := sqlite.InitDB(dsn, log)
	require.NoError(t, err)
	defer db.Close()

	var emptyFS embed.FS
	err = sqlite.RunMigrations(t.Context(), db.DB, log, emptyFS)
	assert.Error(t, err)
}

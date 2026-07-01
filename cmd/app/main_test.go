// Package main предоставляет интеграционные тесты для сквозной верификации
// инициализации, сборки зависимостей и корректности работы механизма Graceful Shutdown.
package main

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRun_SuccessLifecycle(t *testing.T) {
	// 1. Создаем временный файл для базы данных SQLite.
	// Мы не используем ":memory:", так как метод RunMigrations() открывает и сразу закрывает
	// свое соединение. В SQLite база данных в памяти уничтожается сразу после закрытия соединения,
	// и последующий InitDB() получил бы пустую БД. Файл решает эту проблему.
	tmpDB, err := os.CreateTemp("", "shop_main_test_*.db")
	require.NoError(t, err)

	// Закрываем дескриптор, чтобы им могли пользоваться компоненты приложения,
	// но путь к файлу сохраняем. Очистку диска гарантируем через defer.
	tmpDB.Close()
	defer os.Remove(tmpDB.Name())

	// 2. Настраиваем переменные окружения для конфигуратора приложения.
	t.Setenv("APP_MODE", "local")  // Используем корректный режим из вашего конфига
	t.Setenv("LOG_MODE", "Stdout") // Выводим в консоль, чтобы не захламлять диск файлами логов
	t.Setenv("JWT_SECRET", "super-secret-key-specifically-for-integration-testing-123")
	t.Setenv("DB_FILE", tmpDB.Name())
	t.Setenv("HTTP_PORT", "0") // ОС сама выделит случайный свободный порт, исключая конфликты
	t.Setenv("HTTP_SHUTDOWN_TIMEOUT", "2s")

	// Канал для перехвата результата работы функции run()
	runErrChan := make(chan error, 1)

	// 3. Запускаем все приложение в отдельной горутине, так как run() блокирует поток
	go func() {
		runErrChan <- run()
	}()

	// Даем приложению достаточно времени (1 секунду), чтобы раскатать миграции,
	// инициализировать все слои и встать в режим ожидания сигналов.
	time.Sleep(1 * time.Second)

	// 4. Имитируем системный сигнал остановки (Graceful Shutdown)
	// Находим текущий процесс приложения и посылаем ему сигнал прерывания Interrupt (Ctrl+C)
	currentProcess, err := os.FindProcess(os.Getpid())
	require.NoError(t, err)

	err = currentProcess.Signal(os.Interrupt)
	require.NoError(t, err, "ошибка при отправке сигнала остановки приложению")

	// 5. Проверяем, как приложение отреагировало на сигнал
	select {
	case runErr := <-runErrChan:
		// Метод run() при успешном получении сигнала и корректном закрытии всех дескрипторов
		// обязан вернуть nil.
		assert.NoError(t, runErr, "приложение должно завершить работу без критических ошибок")
	case <-time.After(5 * time.Second):
		t.Fatal("таймаут: приложение проигнорировало системный сигнал и зависло при попытке закрытия")
	}
}

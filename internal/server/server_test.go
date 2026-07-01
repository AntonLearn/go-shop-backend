// Package server_test содержит интеграционные тесты для проверки жизненного цикла HTTP-сервера.
package server_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/antonlearn/go-shop-backend/internal/server"
	"github.com/stretchr/testify/assert"
)

func TestServer_Lifecycle(t *testing.T) {
	// Создаем минимальный пустой обработчик (mock handler)
	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Передаем порт "0", чтобы ОС автоматически выделила первый попавшийся свободный порт.
	// Это гарантирует изолированность теста и защищает от конфликтов портов.
	srv := server.New("0", dummyHandler)

	// Канал для перехвата ошибки, которую вернет метод Start() после закрытия
	serverErrChan := make(chan error, 1)

	// Запускаем сервер в отдельной горутине, так как метод Start() блокирует поток
	go func() {
		err := srv.Start()
		serverErrChan <- err
	}()

	// Небольшая пауза, чтобы горутина успела вызвать ListenAndServe
	time.Sleep(50 * time.Millisecond)

	// Создаем контекст с таймаутом для плавного завершения (Graceful Shutdown)
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Инициируем остановку сервера
	shutdownErr := srv.Shutdown(shutdownCtx)
	assert.NoError(t, shutdownErr, "метод Shutdown() не должен возвращать ошибок при нормальном закрытии")

	// Проверяем, как завершился метод Start()
	select {
	case startErr := <-serverErrChan:
		// Жизненный цикл HTTP-сервера спроектирован так, что при успешном плавном закрытии
		// метод ListenAndServe() всегда возвращает именно http.ErrServerClosed.
		assert.ErrorIs(t, startErr, http.ErrServerClosed, "после вызова Shutdown метод Start обязан вернуть http.ErrServerClosed")
	case <-time.After(1 * time.Second):
		t.Fatal("таймаут: сервер завис и не завершил работу после вызова Shutdown")
	}
}

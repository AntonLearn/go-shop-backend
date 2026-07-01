// Package main является точкой входа в приложение и выполняет роль Composition Root.
// Он оркестрирует запуск всего приложения: считывает конфигурационные данные,
// инициализирует сквозные компоненты (логгер, криптографические утилиты), настраивает
// инфраструктуру базы данных (миграции, пул соединений) и связывает слои архитектуры
// (Repository -> Service -> Handler) посредством ручного внедрения зависимостей (Dependency Injection).
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/antonlearn/go-shop-backend/internal/config"
	"github.com/antonlearn/go-shop-backend/internal/db/migrations"
	"github.com/antonlearn/go-shop-backend/internal/handler"
	"github.com/antonlearn/go-shop-backend/internal/repository"
	"github.com/antonlearn/go-shop-backend/internal/server"
	"github.com/antonlearn/go-shop-backend/internal/service"
	"github.com/antonlearn/go-shop-backend/pkg/hash"
	"github.com/antonlearn/go-shop-backend/pkg/jwt"
	"github.com/antonlearn/go-shop-backend/pkg/logger"
)

const failExitCode = 1

func main() {
	if err := run(); err != nil {
		os.Exit(failExitCode)
	}
}

func run() (err error) {
	log.Println("начало инициализации приложения")

	// 1. Инициализация компонента конфигурации
	log.Println("начало инициализации компонента конфигурации")
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Println("инициализация компонента конфигурации прервана", "ошибка", err)
		return err
	}
	log.Println("инициализация компонента конфигурации успешно выполнена")

	// 2. Инициализация компонента системного логгера
	log.Println("начало инициализации компонента системного логгера")
	appLogger, err := logger.New(cfg.AppMode, cfg.LogMode)
	if err != nil {
		log.Println("инициализация компонента системного логгера прервана", "ошибка", err)
		return err
	}

	// Создаем корневой контекст для этапа инициализации и работы приложения
	ctx := context.Background()
	appLogger.InfoContext(ctx, "инициализация компонента системного логгера успешно выполнена")

	defer func() {
		// Используем стандартный лог, так как сам компонент логгера закрывается
		log.Println("начало завершения работы компонента системного логгера")
		if closeErr := appLogger.Close(); closeErr != nil {
			log.Println("завершение работы компонента системного логгера прервано", "ошибка", closeErr)
			if err == nil {
				err = closeErr
			}
		} else {
			log.Println("завершение работы компонента системного логгера успешно выполнено")
		}
	}()

	// 3. Инициализация криптографического компонента JWT
	appLogger.InfoContext(ctx, "начало инициализации криптографического компонента JWT")
	if err := jwt.InitJWT(cfg.JWTSecret); err != nil {
		appLogger.ErrorContext(ctx, "инициализация криптографического компонента JWT прервана", "ошибка", err)
		return err
	}
	appLogger.InfoContext(ctx, "инициализация криптографического компонента JWT успешно выполнена")

	// 4. Инициализация криптографического компонента Bcrypt
	appLogger.InfoContext(ctx, "начало инициализации криптографического компонента Bcrypt")
	hasher := hash.NewBcryptHasher()
	appLogger.InfoContext(ctx, "инициализация криптографического компонента Bcrypt успешно выполнена")

	// Инициализация генератора токенов (Адаптер для сервиса)
	tokenGenerator := jwt.NewTokenManager()

	// 5. Инициализация инфраструктуры базы данных SQLite

	// 5.1. Инициализация компонента миграций базы данных SQLite
	appLogger.InfoContext(ctx, "начало инициализации компонента миграций базы данных SQLite")
	if err := repository.RunMigrations(cfg.DB.DBFile, appLogger, migrations.FS); err != nil {
		appLogger.ErrorContext(ctx, "инициализация компонента миграций базы данных SQLite прервана", "ошибка", err)
		return err
	}
	appLogger.InfoContext(ctx, "инициализация компонента миграций базы данных SQLite успешно выполнена")

	// 5.2. Инициализация компонента пула соединений базы данных SQLite
	appLogger.InfoContext(ctx, "начало инициализации компонента пула соединений базы данных SQLite")
	db, err := repository.InitDB(cfg.DB.DBFile, appLogger)
	if err != nil {
		appLogger.ErrorContext(ctx, "инициализация компонента пула соединений базы данных SQLite прервана", "ошибка", err)
		return err
	}
	appLogger.InfoContext(ctx, "инициализация компонента пула соединений базы данных SQLite успешно выполнена")

	defer func() {
		// Используем независимый контекст для гарантированного закрытия ресурсов
		cleanupCtx := context.Background()
		appLogger.InfoContext(cleanupCtx, "начало завершения работы компонента пула соединений базы данных SQLite")
		if closeErr := db.Close(); closeErr != nil {
			appLogger.ErrorContext(cleanupCtx, "завершение работы компонента пула соединений базы данных SQLite прервано", "ошибка", closeErr)
			if err == nil {
				err = closeErr
			}
		} else {
			appLogger.InfoContext(cleanupCtx, "завершение работы компонента пула соединений базы данных SQLite успешно выполнено")
		}
	}()

	// 5.3. Инициализация репозиториев
	appLogger.InfoContext(ctx, "начало инициализации репозиториев")
	userRepo := repository.NewSQLiteUserRepository(db, appLogger)
	sessionRepo := repository.NewSQLiteSessionRepository(db, appLogger)
	productRepo := repository.NewSQLiteProductRepository(db, appLogger)
	cartRepo := repository.NewSQLiteCartRepository(db, appLogger)
	orderRepo := repository.NewSQLiteOrderRepository(db, appLogger)
	appLogger.InfoContext(ctx, "инициализация всех репозиториев успешно выполнена")

	// 6. Инициализация компонентов сервисов
	appLogger.InfoContext(ctx, "начало инициализации компонентов сервисов")
	authService := service.NewAuthService(userRepo, sessionRepo, hasher, tokenGenerator, appLogger)
	productService := service.NewProductService(productRepo, appLogger)
	cartService := service.NewCartService(cartRepo, productRepo, appLogger)
	orderService := service.NewOrderService(orderRepo, cartRepo, appLogger)
	appLogger.InfoContext(ctx, "инициализация всех сервисов успешно выполнена")

	// 7. Инициализация компонентов обработчиков
	appLogger.InfoContext(ctx, "начало инициализации компонентов обработчиков")
	authHandler := handler.NewAuthHandler(authService, appLogger)
	productHandler := handler.NewProductHandler(productService, appLogger)
	cartHandler := handler.NewCartHandler(cartService, appLogger)
	orderHandler := handler.NewOrderHandler(orderService, appLogger)
	appLogger.InfoContext(ctx, "инициализация всех обработчиков успешно выполнена")

	// 8. Инициализация компонента маршрутизатора HTTP-сервера
	appLogger.InfoContext(ctx, "начало инициализации компонента маршрутизатора HTTP-сервера")
	router := handler.InitRouter(
		authHandler,
		productHandler,
		cartHandler,
		orderHandler,
		tokenGenerator,
		appLogger,
	)
	appLogger.InfoContext(ctx, "инициализация компонента маршрутизатора HTTP-сервера успешно выполнена")

	// 9. Инициализация и запуск компонента HTTP-сервера
	appLogger.InfoContext(ctx, "начало инициализации компонента HTTP-сервера")
	srv := server.New(cfg.HTTP.Port, router)
	appLogger.InfoContext(ctx, "инициализация компонента HTTP-сервера успешно выполнена")

	serverErrors := make(chan error, 1)

	appLogger.InfoContext(ctx, "начало запуска компонента HTTP-сервера", "порт", cfg.HTTP.Port)
	go func() {
		if listenErr := srv.Start(); listenErr != nil && !errors.Is(listenErr, http.ErrServerClosed) {
			serverErrors <- listenErr
		}
	}()
	appLogger.InfoContext(ctx, "компонент HTTP-сервер успешно запущен")

	appLogger.InfoContext(ctx, "инициализация приложения успешно выполнена")

	// 10. Ожидание сигналов закрытия (Graceful Shutdown)
	shutdownSignal := make(chan os.Signal, 1)
	signal.Notify(shutdownSignal, os.Interrupt, syscall.SIGTERM)

	select {
	case listenErr := <-serverErrors:
		appLogger.ErrorContext(ctx, "критическая ошибка при запуске компонента HTTP-сервера, экстренное завершение", "ошибка", listenErr)
		return listenErr

	case sig := <-shutdownSignal:
		appLogger.InfoContext(ctx, "начало завершения работы приложения", "сигнал", sig.String())

		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
		defer cancel()

		if shutdownErr := srv.Shutdown(shutdownCtx); shutdownErr != nil {
			appLogger.ErrorContext(shutdownCtx, "завершение работы компонента HTTP-сервера прервано", "ошибка", shutdownErr)
			return shutdownErr
		}
	}

	appLogger.InfoContext(context.Background(), "завершение работы приложения успешно выполнено")
	return nil
}

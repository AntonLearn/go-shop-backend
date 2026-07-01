// Package handler реализует транспортный слой приложения (HTTP).
// Файл router.go отвечает за конфигурацию путей, группировку эндпоинтов
// и наложение глобальных или специфичных для модулей middleware.
package handler

import (
	"net/http"

	"github.com/antonlearn/go-shop-backend/pkg/logger"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// InitRouter собирает все хендлеры приложения в единое дерево маршрутов Сhi.
func InitRouter(
	authHandler *AuthHandler,
	productHandler *ProductHandler,
	cartHandler *CartHandler,
	tokenParser TokenParserInterface,
	log *logger.Logger,
) http.Handler {
	r := chi.NewRouter()

	// 1. Подключение базовых системных middleware для надежности
	// ВНИМАНИЕ: middleware.RealIP удален из-за уязвимости к IP-спуфингу (GHSA-3fxj-6jh8-hvhx).
	// Безопасное извлечение IP должно происходить на уровне Reverse Proxy (Nginx/Cloudflare).
	r.Use(middleware.Recoverer) // Перехват паник (panic) внутри хендлеров во избежание падения сервера

	// 2. Группировка API по версиям
	r.Route("/api/v1", func(r chi.Router) {

		// Модуль аутентификации и сессий
		r.Route("/auth", func(r chi.Router) {
			r.Post("/sign-up", authHandler.SignUp)
			r.Post("/sign-in", authHandler.SignIn)
			r.Post("/refresh", authHandler.Refresh)
			r.Post("/logout", authHandler.Logout)
		})

		// Модуль каталога товаров
		r.Route("/products", func(r chi.Router) {
			// Публичные эндпоинты (доступны всем посетителям без авторизации)
			r.Get("/", productHandler.GetAll)
			r.Get("/{id}", productHandler.GetByID)

			// Защищенная зона управления каталогом (модификация доступна ТОЛЬКО Администратору)
			r.Group(func(r chi.Router) {
				r.Use(AuthMiddleware(tokenParser, log))
				r.Use(RequireRole("admin", log)) // Передаем логгер для консистентности

				r.Post("/", productHandler.Create)
				r.Put("/{id}", productHandler.Update)
				r.Delete("/{id}", productHandler.Delete)
			})
		})

		// Модуль корзины товаров
		r.Route("/cart", func(r chi.Router) {
			// Абсолютно все операции с корзиной требуют обязательной авторизации,
			// так как нам жизненно необходим user_id из JWT-токена
			r.Use(AuthMiddleware(tokenParser, log))

			r.Post("/", cartHandler.Add)          // Добавить товар или увеличить кол-во
			r.Get("/", cartHandler.GetByID)       // Посмотреть содержимое своей корзины
			r.Delete("/{id}", cartHandler.Delete) // Удалить конкретный товар из корзины
		})
	})

	return r
}

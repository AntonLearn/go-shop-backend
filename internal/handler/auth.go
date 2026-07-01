// Package handler реализует транспортный слой (HTTP) приложения,
// обрабатывает входящие запросы, валидирует DTO и вызывает слой бизнес-логики.
package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/antonlearn/go-shop-backend/internal/model"
	"github.com/antonlearn/go-shop-backend/pkg/logger"
	"github.com/go-playground/validator/v10"
)

// AuthServiceInterface описывает контракт бизнес-логики аутентификации,
// необходимый для изоляции и тестирования транспортного слоя.
type AuthServiceInterface interface {
	SignUp(ctx context.Context, input model.SignUpInput) (model.AuthResponse, error)
	SignIn(ctx context.Context, input model.SignInInput) (model.AuthResponse, error)
	Refresh(ctx context.Context, refreshToken string) (model.AuthResponse, error)
	Logout(ctx context.Context, refreshToken string) error
}

// AuthHandler инкапсулирует бизнес-логику аутентификации, логгер и валидатор полей.
type AuthHandler struct {
	authService AuthServiceInterface
	log         *logger.Logger
	validate    *validator.Validate
}

// NewAuthHandler конструирует новый HTTP-обработчик аутентификации.
func NewAuthHandler(authService AuthServiceInterface, log *logger.Logger) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		log:         log,
		validate:    validator.New(),
	}
}

// SignUp обрабатывает POST-запрос на регистрацию нового пользователя.
func (h *AuthHandler) SignUp(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondWithError(w, r.Context(), h.log, http.StatusMethodNotAllowed, "метод не поддерживается")
		return
	}

	var input model.SignUpInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		h.log.ErrorContext(r.Context(), "не удалось декодировать тело запроса регистрации", "error", err)
		respondWithError(w, r.Context(), h.log, http.StatusBadRequest, "некорректный формат JSON")
		return
	}

	// Полноценная валидация структуры на основе тегов в модели (min=8, eqfield и т.д.)
	if err := h.validate.Struct(input); err != nil {
		h.log.WarnContext(r.Context(), "ошибка валидации данных регистрации", "error", err)
		respondWithError(w, r.Context(), h.log, http.StatusUnprocessableEntity, "ошибка валидации полей запроса")
		return
	}

	// Вызов слоя бизнес-логики
	tokens, err := h.authService.SignUp(r.Context(), input)
	if err != nil {
		h.log.ErrorContext(r.Context(), "сбой при регистрации пользователя в сервисе", "error", err)

		// Пример обработки доменной ошибки (если пользователь уже существует)
		if errors.Is(err, model.ErrUserAlreadyExists) {
			respondWithError(w, r.Context(), h.log, http.StatusConflict, "пользователь с таким email уже зарегистрирован")
			return
		}

		respondWithError(w, r.Context(), h.log, http.StatusInternalServerError, "внутренняя ошибка сервера")
		return
	}

	respondWithJSON(w, r.Context(), h.log, http.StatusCreated, tokens)
}

// SignIn обрабатывает POST-запрос для авторизации существующего пользователя.
func (h *AuthHandler) SignIn(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondWithError(w, r.Context(), h.log, http.StatusMethodNotAllowed, "метод не поддерживается")
		return
	}

	var input model.SignInInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		h.log.ErrorContext(r.Context(), "не удалось декодировать тело запроса авторизации", "error", err)
		respondWithError(w, r.Context(), h.log, http.StatusBadRequest, "некорректный формат JSON")
		return
	}

	if err := h.validate.Struct(input); err != nil {
		h.log.WarnContext(r.Context(), "ошибка валидации данных авторизации", "error", err)
		respondWithError(w, r.Context(), h.log, http.StatusUnprocessableEntity, "необходимо заполнить все поля корректно")
		return
	}

	tokens, err := h.authService.SignIn(r.Context(), input)
	if err != nil {
		h.log.WarnContext(r.Context(), "неудачная попытка входа", "email", input.Email, "error", err)

		if errors.Is(err, model.ErrInvalidCredentials) {
			respondWithError(w, r.Context(), h.log, http.StatusUnauthorized, "неверный email или пароль")
			return
		}

		respondWithError(w, r.Context(), h.log, http.StatusInternalServerError, "внутренняя ошибка сервера")
		return
	}

	respondWithJSON(w, r.Context(), h.log, http.StatusOK, tokens)
}

// Refresh обновляет пару токенов доступа (Access/Refresh) по предоставленному Refresh-токену.
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondWithError(w, r.Context(), h.log, http.StatusMethodNotAllowed, "метод не поддерживается")
		return
	}

	// Локальное DTO для извлечения токена из JSON
	var input struct {
		RefreshToken string `json:"refresh_token" validate:"required"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		h.log.ErrorContext(r.Context(), "не удалось декодировать токен обновления", "error", err)
		respondWithError(w, r.Context(), h.log, http.StatusBadRequest, "некорректный формат JSON")
		return
	}

	if err := h.validate.Struct(&input); err != nil {
		respondWithError(w, r.Context(), h.log, http.StatusBadRequest, "отсутствует refresh_token")
		return
	}

	tokens, err := h.authService.Refresh(r.Context(), input.RefreshToken)
	if err != nil {
		h.log.WarnContext(r.Context(), "ошибка обновления токенов сессии", "error", err)

		if errors.Is(err, model.ErrSessionNotFound) {
			respondWithError(w, r.Context(), h.log, http.StatusUnauthorized, "сессия не найдена или устарела")
			return
		}

		respondWithError(w, r.Context(), h.log, http.StatusInternalServerError, "внутренняя ошибка сервера")
		return
	}

	respondWithJSON(w, r.Context(), h.log, http.StatusOK, tokens)
}

// Logout завершает сессию пользователя и отзывает Refresh-токен.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondWithError(w, r.Context(), h.log, http.StatusMethodNotAllowed, "метод не поддерживается")
		return
	}

	var input struct {
		RefreshToken string `json:"refresh_token" validate:"required"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		h.log.ErrorContext(r.Context(), "не удалось декодировать токен при выходе", "error", err)
		respondWithError(w, r.Context(), h.log, http.StatusBadRequest, "некорректный формат JSON")
		return
	}

	if err := h.validate.Struct(&input); err != nil {
		respondWithError(w, r.Context(), h.log, http.StatusBadRequest, "отсутствует refresh_token")
		return
	}

	if err := h.authService.Logout(r.Context(), input.RefreshToken); err != nil {
		h.log.ErrorContext(r.Context(), "ошибка при удалении сессии из базы данных", "error", err)

		if errors.Is(err, model.ErrSessionNotFound) {
			respondWithError(w, r.Context(), h.log, http.StatusNotFound, "активная сессия не найдена")
			return
		}

		respondWithError(w, r.Context(), h.log, http.StatusInternalServerError, "внутренняя ошибка сервера")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

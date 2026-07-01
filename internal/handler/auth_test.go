// Package handler_test содержит модульные тесты для транспортного слоя (HTTP-хендлеров).
// Тесты используют моки для изоляции от слоя бизнес-логики и проверяют корректность
// обработки входящих запросов, валидацию DTO и формирование HTTP-ответов.
package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/antonlearn/go-shop-backend/internal/handler"
	"github.com/antonlearn/go-shop-backend/internal/model"
	"github.com/antonlearn/go-shop-backend/pkg/logger"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// --- MOCK ---

// MockAuthService — заглушка (mock) для интерфейса AuthServiceInterface.
// Позволяет контролировать и проверять вызовы к слою бизнес-логики аутентификации.
type MockAuthService struct {
	mock.Mock
}

// SignUp имитирует вызов метода регистрации пользователя.
func (m *MockAuthService) SignUp(ctx context.Context, input model.SignUpInput) (model.AuthResponse, error) {
	args := m.Called(mock.Anything, input)
	return args.Get(0).(model.AuthResponse), args.Error(1)
}

// SignIn имитирует вызов метода авторизации пользователя.
func (m *MockAuthService) SignIn(ctx context.Context, input model.SignInInput) (model.AuthResponse, error) {
	args := m.Called(mock.Anything, input)
	return args.Get(0).(model.AuthResponse), args.Error(1)
}

// Refresh имитирует вызов метода обновления пары JWT-токенов.
func (m *MockAuthService) Refresh(ctx context.Context, token string) (model.AuthResponse, error) {
	args := m.Called(mock.Anything, token)
	return args.Get(0).(model.AuthResponse), args.Error(1)
}

// Logout имитирует вызов метода завершения сессии и отзыва токена.
func (m *MockAuthService) Logout(ctx context.Context, token string) error {
	return m.Called(mock.Anything, token).Error(0)
}

// --- TEST SUITE ---

type AuthHandlerTestSuite struct {
	suite.Suite
	mockSvc *MockAuthService
	h       *handler.AuthHandler
}

func (s *AuthHandlerTestSuite) SetupTest() {
	log, _ := logger.New("Console", "DEBUG")
	s.mockSvc = new(MockAuthService)
	s.h = handler.NewAuthHandler(s.mockSvc, log)
}

// ==========================================
// ТЕСТЫ: SignUp (Регистрация)
// ==========================================

func (s *AuthHandlerTestSuite) TestSignUp() {
	input := model.SignUpInput{
		Name:            "Anton",
		Email:           "test@shop.com",
		Password:        "password123",
		PasswordConfirm: "password123",
	}
	resp := model.AuthResponse{AccessToken: "at", RefreshToken: "rt"}

	s.Run("Success", func() {
		s.mockSvc.On("SignUp", mock.Anything, input).Return(resp, nil).Once()

		body, _ := json.Marshal(input)
		req := httptest.NewRequest(http.MethodPost, "/signup", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		s.h.SignUp(w, req)

		s.Equal(http.StatusCreated, w.Code)
		s.mockSvc.AssertExpectations(s.T())
	})

	s.Run("Method Not Allowed", func() {
		req := httptest.NewRequest(http.MethodGet, "/signup", nil)
		w := httptest.NewRecorder()

		s.h.SignUp(w, req)

		s.Equal(http.StatusMethodNotAllowed, w.Code)
	})

	s.Run("Invalid JSON", func() {
		req := httptest.NewRequest(http.MethodPost, "/signup", bytes.NewBufferString("{invalid-json"))
		w := httptest.NewRecorder()

		s.h.SignUp(w, req)

		s.Equal(http.StatusBadRequest, w.Code)
	})

	s.Run("Validation Error", func() {
		invalidInput := input
		invalidInput.Password = "123" // Слишком короткий пароль

		body, _ := json.Marshal(invalidInput)
		req := httptest.NewRequest(http.MethodPost, "/signup", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		s.h.SignUp(w, req)

		s.Equal(http.StatusUnprocessableEntity, w.Code)
	})

	s.Run("Conflict - User Already Exists", func() {
		s.mockSvc.On("SignUp", mock.Anything, input).Return(model.AuthResponse{}, model.ErrUserAlreadyExists).Once()

		body, _ := json.Marshal(input)
		req := httptest.NewRequest(http.MethodPost, "/signup", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		s.h.SignUp(w, req)

		s.Equal(http.StatusConflict, w.Code)
	})

	s.Run("Internal Server Error", func() {
		s.mockSvc.On("SignUp", mock.Anything, input).Return(model.AuthResponse{}, errors.New("db crash")).Once()

		body, _ := json.Marshal(input)
		req := httptest.NewRequest(http.MethodPost, "/signup", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		s.h.SignUp(w, req)

		s.Equal(http.StatusInternalServerError, w.Code)
	})
}

// ==========================================
// ТЕСТЫ: SignIn (Аутентификация)
// ==========================================

func (s *AuthHandlerTestSuite) TestSignIn() {
	input := model.SignInInput{Email: "user@shop.com", Password: "password123"}
	resp := model.AuthResponse{AccessToken: "at", RefreshToken: "rt"}

	s.Run("Success", func() {
		s.mockSvc.On("SignIn", mock.Anything, input).Return(resp, nil).Once()

		body, _ := json.Marshal(input)
		req := httptest.NewRequest(http.MethodPost, "/signin", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		s.h.SignIn(w, req)

		s.Equal(http.StatusOK, w.Code)
		s.mockSvc.AssertExpectations(s.T())
	})

	s.Run("Method Not Allowed", func() {
		req := httptest.NewRequest(http.MethodDelete, "/signin", nil)
		w := httptest.NewRecorder()

		s.h.SignIn(w, req)

		s.Equal(http.StatusMethodNotAllowed, w.Code)
	})

	s.Run("Invalid JSON", func() {
		req := httptest.NewRequest(http.MethodPost, "/signin", bytes.NewBufferString("bad-data"))
		w := httptest.NewRecorder()

		s.h.SignIn(w, req)

		s.Equal(http.StatusBadRequest, w.Code)
	})

	s.Run("Validation Error", func() {
		invalidInput := model.SignInInput{Email: "", Password: ""}

		body, _ := json.Marshal(invalidInput)
		req := httptest.NewRequest(http.MethodPost, "/signin", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		s.h.SignIn(w, req)

		s.Equal(http.StatusUnprocessableEntity, w.Code)
	})

	s.Run("Unauthorized - Invalid Credentials", func() {
		s.mockSvc.On("SignIn", mock.Anything, input).Return(model.AuthResponse{}, model.ErrInvalidCredentials).Once()

		body, _ := json.Marshal(input)
		req := httptest.NewRequest(http.MethodPost, "/signin", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		s.h.SignIn(w, req)

		s.Equal(http.StatusUnauthorized, w.Code)
	})

	s.Run("Internal Server Error", func() {
		s.mockSvc.On("SignIn", mock.Anything, input).Return(model.AuthResponse{}, errors.New("redis offline")).Once()

		body, _ := json.Marshal(input)
		req := httptest.NewRequest(http.MethodPost, "/signin", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		s.h.SignIn(w, req)

		s.Equal(http.StatusInternalServerError, w.Code)
	})
}

// ==========================================
// ТЕСТЫ: Refresh (Обновление токенов)
// ==========================================

func (s *AuthHandlerTestSuite) TestRefresh() {
	validToken := "valid_token"
	inputBody := map[string]string{"refresh_token": validToken}
	resp := model.AuthResponse{AccessToken: "new_at", RefreshToken: "valid_token"}

	s.Run("Success", func() {
		s.mockSvc.On("Refresh", mock.Anything, validToken).Return(resp, nil).Once()

		body, _ := json.Marshal(inputBody)
		req := httptest.NewRequest(http.MethodPost, "/refresh", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		s.h.Refresh(w, req)

		s.Equal(http.StatusOK, w.Code)
		s.mockSvc.AssertExpectations(s.T())
	})

	s.Run("Method Not Allowed", func() {
		req := httptest.NewRequest(http.MethodPut, "/refresh", nil)
		w := httptest.NewRecorder()

		s.h.Refresh(w, req)

		s.Equal(http.StatusMethodNotAllowed, w.Code)
	})

	s.Run("Invalid JSON", func() {
		req := httptest.NewRequest(http.MethodPost, "/refresh", bytes.NewBufferString("{bad}"))
		w := httptest.NewRecorder()

		s.h.Refresh(w, req)

		s.Equal(http.StatusBadRequest, w.Code)
	})

	s.Run("Missing Refresh Token", func() {
		invalidBody := map[string]string{"refresh_token": ""} // Пустое поле, заваленный validate:"required"

		body, _ := json.Marshal(invalidBody)
		req := httptest.NewRequest(http.MethodPost, "/refresh", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		s.h.Refresh(w, req)

		s.Equal(http.StatusBadRequest, w.Code)
	})

	s.Run("Unauthorized - Session Not Found", func() {
		s.mockSvc.On("Refresh", mock.Anything, validToken).Return(model.AuthResponse{}, model.ErrSessionNotFound).Once()

		body, _ := json.Marshal(inputBody)
		req := httptest.NewRequest(http.MethodPost, "/refresh", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		s.h.Refresh(w, req)

		s.Equal(http.StatusUnauthorized, w.Code)
	})

	s.Run("Internal Server Error", func() {
		s.mockSvc.On("Refresh", mock.Anything, validToken).Return(model.AuthResponse{}, errors.New("crypto crash")).Once()

		body, _ := json.Marshal(inputBody)
		req := httptest.NewRequest(http.MethodPost, "/refresh", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		s.h.Refresh(w, req)

		s.Equal(http.StatusInternalServerError, w.Code)
	})
}

// ==========================================
// ТЕСТЫ: Logout (Выход из системы)
// ==========================================

func (s *AuthHandlerTestSuite) TestLogout() {
	targetToken := "token_to_revoke"
	inputBody := map[string]string{"refresh_token": targetToken}

	s.Run("Success", func() {
		s.mockSvc.On("Logout", mock.Anything, targetToken).Return(nil).Once()

		body, _ := json.Marshal(inputBody)
		req := httptest.NewRequest(http.MethodPost, "/logout", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		s.h.Logout(w, req)

		s.Equal(http.StatusNoContent, w.Code)
		s.mockSvc.AssertExpectations(s.T())
	})

	s.Run("Method Not Allowed", func() {
		req := httptest.NewRequest(http.MethodGet, "/logout", nil)
		w := httptest.NewRecorder()

		s.h.Logout(w, req)

		s.Equal(http.StatusMethodNotAllowed, w.Code)
	})

	s.Run("Invalid JSON", func() {
		req := httptest.NewRequest(http.MethodPost, "/logout", bytes.NewBufferString("not-a-json"))
		w := httptest.NewRecorder()

		s.h.Logout(w, req)

		s.Equal(http.StatusBadRequest, w.Code)
	})

	s.Run("Missing Refresh Token", func() {
		invalidBody := map[string]string{"refresh_token": ""}

		body, _ := json.Marshal(invalidBody)
		req := httptest.NewRequest(http.MethodPost, "/logout", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		s.h.Logout(w, req)

		s.Equal(http.StatusBadRequest, w.Code)
	})

	s.Run("Not Found - Session Missing", func() {
		s.mockSvc.On("Logout", mock.Anything, targetToken).Return(model.ErrSessionNotFound).Once()

		body, _ := json.Marshal(inputBody)
		req := httptest.NewRequest(http.MethodPost, "/logout", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		s.h.Logout(w, req)

		s.Equal(http.StatusNotFound, w.Code)
	})

	s.Run("Internal Server Error", func() {
		s.mockSvc.On("Logout", mock.Anything, targetToken).Return(errors.New("db disconnect")).Once()

		body, _ := json.Marshal(inputBody)
		req := httptest.NewRequest(http.MethodPost, "/logout", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		s.h.Logout(w, req)

		s.Equal(http.StatusInternalServerError, w.Code)
	})
}

func TestAuthHandlerSuite(t *testing.T) {
	suite.Run(t, new(AuthHandlerTestSuite))
}

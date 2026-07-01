// Package handler_test содержит модульные тесты для проверки транспортного слоя
// аутентификации (AuthHandler). Все тесты используют принцип black-box.
package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/antonlearn/go-shop-backend/internal/handler"
	"github.com/antonlearn/go-shop-backend/internal/model"
	"github.com/antonlearn/go-shop-backend/pkg/logger"
)

// =============================================================================
// Мок-объект для изоляции AuthServiceInterface
// =============================================================================

type mockAuthService struct {
	mock.Mock
}

func (m *mockAuthService) SignUp(ctx context.Context, input model.SignUpInput) (model.AuthResponse, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(model.AuthResponse), args.Error(1)
}

func (m *mockAuthService) SignIn(ctx context.Context, input model.SignInInput) (model.AuthResponse, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(model.AuthResponse), args.Error(1)
}

func (m *mockAuthService) Refresh(ctx context.Context, refreshToken string) (model.AuthResponse, error) {
	args := m.Called(ctx, refreshToken)
	return args.Get(0).(model.AuthResponse), args.Error(1)
}

func (m *mockAuthService) Logout(ctx context.Context, refreshToken string) error {
	args := m.Called(ctx, refreshToken)
	return args.Error(0)
}

type authErrorResponse struct {
	Error string `json:"error"`
}

func setupAuthTestDeps(t *testing.T) (*handler.AuthHandler, *mockAuthService) {
	t.Helper()
	log, err := logger.New("local", "Stdout")
	require.NoError(t, err, "Не удалось инициализировать тестовый логгер")

	mockService := new(mockAuthService)
	authHandler := handler.NewAuthHandler(mockService, log)

	return authHandler, mockService
}

// =============================================================================
// Тесты для метода SignUp (Регистрация)
// =============================================================================

func TestAuthSignUp_Success(t *testing.T) {
	h, s := setupAuthTestDeps(t)
	input := model.SignUpInput{
		Email:           "test@example.com",
		Password:        "password123",
		ConfirmPassword: "password123",
	}
	expectedResponse := model.AuthResponse{
		AccessToken:  "access_token_mock",
		RefreshToken: "refresh_token_mock",
	}

	s.On("SignUp", mock.Anything, input).Return(expectedResponse, nil)

	body, err := json.Marshal(input)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/sign-up", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	h.SignUp(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)
	var actualResponse model.AuthResponse
	err = json.Unmarshal(rr.Body.Bytes(), &actualResponse)
	require.NoError(t, err)
	assert.Equal(t, expectedResponse, actualResponse)
	s.AssertExpectations(t)
}

func TestAuthSignUp_Errors(t *testing.T) {
	tests := []struct {
		name        string
		method      string
		body        any
		mockSetup   func(s *mockAuthService)
		expectedSt  int
		expectedMsg string
	}{
		{
			name:        "Неподдерживаемый метод GET",
			method:      http.MethodGet,
			body:        nil,
			mockSetup:   func(s *mockAuthService) {},
			expectedSt:  http.StatusMethodNotAllowed,
			expectedMsg: "метод не поддерживается",
		},
		{
			name:        "Некорректный JSON",
			method:      http.MethodPost,
			body:        "{bad-json}",
			mockSetup:   func(s *mockAuthService) {},
			expectedSt:  http.StatusBadRequest,
			expectedMsg: "некорректный формат JSON",
		},
		{
			name:        "Ошибка валидации структуры",
			method:      http.MethodPost,
			body:        model.SignUpInput{Email: ""}, // Пустые поля провалят "required"
			mockSetup:   func(s *mockAuthService) {},
			expectedSt:  http.StatusUnprocessableEntity,
			expectedMsg: "ошибка валидации полей запроса",
		},
		{
			name:   "Пользователь уже существует (Conflict)",
			method: http.MethodPost,
			body: model.SignUpInput{
				Email:           "existing@example.com",
				Password:        "password123",
				ConfirmPassword: "password123",
			},
			mockSetup: func(s *mockAuthService) {
				s.On("SignUp", mock.Anything, mock.Anything).Return(model.AuthResponse{}, model.ErrUserAlreadyExists)
			},
			expectedSt:  http.StatusConflict,
			expectedMsg: "пользователь с таким email уже зарегистрирован",
		},
		{
			name:   "Внутренняя ошибка сервиса",
			method: http.MethodPost,
			body: model.SignUpInput{
				Email:           "error@example.com",
				Password:        "password123",
				ConfirmPassword: "password123",
			},
			mockSetup: func(s *mockAuthService) {
				s.On("SignUp", mock.Anything, mock.Anything).Return(model.AuthResponse{}, assert.AnError)
			},
			expectedSt:  http.StatusInternalServerError,
			expectedMsg: "внутренняя ошибка сервера",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, s := setupAuthTestDeps(t)
			tt.mockSetup(s)

			var buf bytes.Buffer
			if str, ok := tt.body.(string); ok {
				buf.WriteString(str)
			} else if tt.body != nil {
				err := json.NewEncoder(&buf).Encode(tt.body)
				require.NoError(t, err)
			}

			req := httptest.NewRequest(tt.method, "/api/v1/auth/sign-up", &buf)
			rr := httptest.NewRecorder()

			h.SignUp(rr, req)

			assert.Equal(t, tt.expectedSt, rr.Code)
			var resp authErrorResponse
			err := json.Unmarshal(rr.Body.Bytes(), &resp)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedMsg, resp.Error)
		})
	}
}

// =============================================================================
// Тесты для метода SignIn (Авторизация)
// =============================================================================

func TestAuthSignIn_Success(t *testing.T) {
	h, s := setupAuthTestDeps(t)
	input := model.SignInInput{
		Email:    "test@example.com",
		Password: "password123",
	}
	expectedResponse := model.AuthResponse{
		AccessToken:  "access_token_mock",
		RefreshToken: "refresh_token_mock",
	}

	s.On("SignIn", mock.Anything, input).Return(expectedResponse, nil)

	body, err := json.Marshal(input)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/sign-in", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	h.SignIn(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var actualResponse model.AuthResponse
	err = json.Unmarshal(rr.Body.Bytes(), &actualResponse)
	require.NoError(t, err)
	assert.Equal(t, expectedResponse, actualResponse)
}

func TestAuthSignIn_InvalidCredentials(t *testing.T) {
	h, s := setupAuthTestDeps(t)
	input := model.SignInInput{
		Email:    "wrong@example.com",
		Password: "wrongpassword",
	}

	s.On("SignIn", mock.Anything, input).Return(model.AuthResponse{}, model.ErrInvalidCredentials)

	body, err := json.Marshal(input)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/sign-in", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	h.SignIn(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	var resp authErrorResponse
	err = json.Unmarshal(rr.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "неверный email или пароль", resp.Error)
}

func TestAuthSignIn_ValidationErrors(t *testing.T) {
	h, _ := setupAuthTestDeps(t)
	input := model.SignInInput{Email: ""} // Пустой email завалит валидацию

	body, err := json.Marshal(input)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/sign-in", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	h.SignIn(rr, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rr.Code)
	var resp authErrorResponse
	err = json.Unmarshal(rr.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "необходимо заполнить все поля корректно", resp.Error)
}

// =============================================================================
// Тесты для метода Refresh (Обновление токенов)
// =============================================================================

func TestAuthRefresh_Success(t *testing.T) {
	h, s := setupAuthTestDeps(t)
	tokenInput := map[string]string{"refresh_token": "valid_refresh_token"}
	expectedResponse := model.AuthResponse{
		AccessToken:  "new_access",
		RefreshToken: "new_refresh",
	}

	s.On("Refresh", mock.Anything, "valid_refresh_token").Return(expectedResponse, nil)

	body, err := json.Marshal(tokenInput)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	h.Refresh(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var actualResponse model.AuthResponse
	err = json.Unmarshal(rr.Body.Bytes(), &actualResponse)
	require.NoError(t, err)
	assert.Equal(t, expectedResponse, actualResponse)
}

func TestAuthRefresh_MissingToken(t *testing.T) {
	h, _ := setupAuthTestDeps(t)
	tokenInput := map[string]string{"refresh_token": ""} // Пустой токен

	body, err := json.Marshal(tokenInput)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	h.Refresh(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp authErrorResponse
	err = json.Unmarshal(rr.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "отсутствует refresh_token", resp.Error)
}

func TestAuthRefresh_SessionNotFound(t *testing.T) {
	h, s := setupAuthTestDeps(t)
	tokenInput := map[string]string{"refresh_token": "expired_token"}

	s.On("Refresh", mock.Anything, "expired_token").Return(model.AuthResponse{}, model.ErrSessionNotFound)

	body, err := json.Marshal(tokenInput)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	h.Refresh(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	var resp authErrorResponse
	err = json.Unmarshal(rr.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "сессия не найдена или устарела", resp.Error)
}

// =============================================================================
// Тесты для метода Logout (Выход из системы)
// =============================================================================

func TestAuthLogout_Success(t *testing.T) {
	h, s := setupAuthTestDeps(t)
	tokenInput := map[string]string{"refresh_token": "token_to_destroy"}

	s.On("Logout", mock.Anything, "token_to_destroy").Return(nil)

	body, err := json.Marshal(tokenInput)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	h.Logout(rr, req)

	assert.Equal(t, http.StatusNoContent, rr.Code)
	assert.Empty(t, rr.Body.String())
	s.AssertExpectations(t)
}

func TestAuthLogout_SessionNotFound(t *testing.T) {
	h, s := setupAuthTestDeps(t)
	tokenInput := map[string]string{"refresh_token": "unknown_token"}

	s.On("Logout", mock.Anything, "unknown_token").Return(model.ErrSessionNotFound)

	body, err := json.Marshal(tokenInput)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	h.Logout(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	var resp authErrorResponse
	err = json.Unmarshal(rr.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "активная сессия не найдена", resp.Error)
}

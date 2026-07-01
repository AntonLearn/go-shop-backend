// Package service_test содержит unit-тесты для слоя бизнес-логики.
// Здесь тестируются правила аутентификации, генерации токенов и управления сессиями.
// Все тесты написаны с использованием testify/suite, что позволяет изолировать
// зависимости и четко структурировать позитивные и негативные сценарии.
package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/antonlearn/go-shop-backend/internal/model"
	"github.com/antonlearn/go-shop-backend/internal/service"
	"github.com/antonlearn/go-shop-backend/pkg/logger"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// --- MOCKS (Тестовые двойники зависимостей) ---

// MockUserRepo имитирует работу с хранилищем пользователей.
type MockUserRepo struct{ mock.Mock }

func (m *MockUserRepo) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}
func (m *MockUserRepo) Create(ctx context.Context, u *model.User) error {
	return m.Called(ctx, u).Error(0)
}
func (m *MockUserRepo) GetByID(ctx context.Context, id int) (*model.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

// MockSessionRepo имитирует хранилище сессий.
type MockSessionRepo struct{ mock.Mock }

func (m *MockSessionRepo) Create(ctx context.Context, s *model.Session) error {
	return m.Called(ctx, s).Error(0)
}
func (m *MockSessionRepo) GetByToken(ctx context.Context, t string) (*model.Session, error) {
	args := m.Called(ctx, t)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Session), args.Error(1)
}
func (m *MockSessionRepo) DeleteByToken(ctx context.Context, t string) error {
	return m.Called(ctx, t).Error(0)
}

// MockHasher имитирует работу с хэшированием паролей.
type MockHasher struct{ mock.Mock }

func (m *MockHasher) GenerateHash(p string) (string, error) {
	args := m.Called(p)
	return args.String(0), args.Error(1)
}
func (m *MockHasher) Compare(h, p string) error { return m.Called(h, p).Error(0) }

// MockTokenGenerator имитирует генератор JWT.
type MockTokenGenerator struct{ mock.Mock }

func (m *MockTokenGenerator) GenerateToken(uid int, email, role string, ttl time.Duration) (string, error) {
	args := m.Called(uid, email, role, ttl)
	return args.String(0), args.Error(1)
}

// --- SUITE ---

// AuthServiceTestSuite организует набор тестов для AuthService.
type AuthServiceTestSuite struct {
	suite.Suite
	service  *service.AuthService
	uRepo    *MockUserRepo
	sRepo    *MockSessionRepo
	hasher   *MockHasher
	tokenGen *MockTokenGenerator
	ctx      context.Context
}

// SetupTest инициализирует сервисы и моки перед каждым запуском теста.
func (s *AuthServiceTestSuite) SetupTest() {
	s.uRepo = new(MockUserRepo)
	s.sRepo = new(MockSessionRepo)
	s.hasher = new(MockHasher)
	s.tokenGen = new(MockTokenGenerator)
	log, _ := logger.New("Console", "INFO")

	s.service = service.NewAuthService(s.uRepo, s.sRepo, s.hasher, s.tokenGen, log)
	s.ctx = context.Background()
}

// --- SIGN UP TESTS ---

// TestSignUp_Success проверяет штатный сценарий регистрации.
func (s *AuthServiceTestSuite) TestSignUp_Success() {
	input := model.SignUpInput{Email: "test@test.com", Password: "pwd"}
	s.uRepo.On("GetByEmail", s.ctx, input.Email).Return(nil, model.ErrUserNotFound)
	s.hasher.On("GenerateHash", input.Password).Return("hash", nil)
	s.uRepo.On("Create", s.ctx, mock.Anything).Return(nil)
	s.tokenGen.On("GenerateToken", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("access", nil)
	s.sRepo.On("Create", s.ctx, mock.Anything).Return(nil)

	res, err := s.service.SignUp(s.ctx, input)
	s.NoError(err)
	s.Equal("access", res.AccessToken)
}

// TestSignUp_DuplicateEmail проверяет поведение при попытке регистрации существующего email.
func (s *AuthServiceTestSuite) TestSignUp_DuplicateEmail() {
	input := model.SignUpInput{Email: "exists@test.com"}
	s.uRepo.On("GetByEmail", s.ctx, input.Email).Return(&model.User{ID: 1}, nil)

	_, err := s.service.SignUp(s.ctx, input)
	s.ErrorIs(err, model.ErrUserAlreadyExists)
}

// TestSignUp_HasherError проверяет ошибку при сбое хэшировщика.
func (s *AuthServiceTestSuite) TestSignUp_HasherError() {
	input := model.SignUpInput{Email: "test@test.com", Password: "pwd"}
	s.uRepo.On("GetByEmail", s.ctx, input.Email).Return(nil, model.ErrUserNotFound)
	s.hasher.On("GenerateHash", input.Password).Return("", errors.New("hasher error"))

	_, err := s.service.SignUp(s.ctx, input)
	s.Error(err)
}

// TestSignUp_CreateDBError проверяет ошибку при сбое базы данных во время создания пользователя.
func (s *AuthServiceTestSuite) TestSignUp_CreateDBError() {
	input := model.SignUpInput{Email: "test@test.com", Password: "pwd"}
	s.uRepo.On("GetByEmail", s.ctx, input.Email).Return(nil, model.ErrUserNotFound)
	s.hasher.On("GenerateHash", input.Password).Return("hashed", nil)
	s.uRepo.On("Create", s.ctx, mock.Anything).Return(errors.New("db error"))

	_, err := s.service.SignUp(s.ctx, input)
	s.Error(err)
}

// --- SIGN IN TESTS ---

// TestSignIn_Success проверяет успешную аутентификацию.
func (s *AuthServiceTestSuite) TestSignIn_Success() {
	input := model.SignInInput{Email: "test@test.com", Password: "pwd"}
	user := &model.User{ID: 1, Email: input.Email, PasswordHash: "hash"}
	s.uRepo.On("GetByEmail", s.ctx, input.Email).Return(user, nil)
	s.hasher.On("Compare", "hash", "pwd").Return(nil)
	s.tokenGen.On("GenerateToken", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("access", nil)
	s.sRepo.On("Create", s.ctx, mock.Anything).Return(nil)

	res, err := s.service.SignIn(s.ctx, input)
	s.NoError(err)
	s.Equal("access", res.AccessToken)
}

// TestSignIn_InvalidCredentials проверяет ошибку при неверных данных пользователя.
func (s *AuthServiceTestSuite) TestSignIn_InvalidCredentials() {
	input := model.SignInInput{Email: "test@test.com", Password: "pwd"}
	s.uRepo.On("GetByEmail", s.ctx, input.Email).Return(nil, model.ErrUserNotFound)

	_, err := s.service.SignIn(s.ctx, input)
	s.ErrorIs(err, model.ErrInvalidCredentials)
}

// --- REFRESH TESTS ---

// TestRefresh_Success проверяет успешное обновление пары токенов (Rotation).
func (s *AuthServiceTestSuite) TestRefresh_Success() {
	oldToken := "old"
	s.sRepo.On("GetByToken", s.ctx, oldToken).Return(&model.Session{UserID: 1, ExpiresAt: time.Now().Add(time.Hour)}, nil)
	s.uRepo.On("GetByID", s.ctx, 1).Return(&model.User{ID: 1}, nil)
	s.sRepo.On("DeleteByToken", s.ctx, oldToken).Return(nil)
	s.tokenGen.On("GenerateToken", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("new", nil)
	s.sRepo.On("Create", s.ctx, mock.Anything).Return(nil)

	_, err := s.service.Refresh(s.ctx, oldToken)
	s.NoError(err)
}

// TestRefresh_Expired проверяет ошибку, если Refresh-токен просрочен.
func (s *AuthServiceTestSuite) TestRefresh_Expired() {
	oldToken := "old"
	s.sRepo.On("GetByToken", s.ctx, oldToken).Return(&model.Session{ExpiresAt: time.Now().Add(-time.Hour)}, nil)
	s.sRepo.On("DeleteByToken", s.ctx, oldToken).Return(nil)

	_, err := s.service.Refresh(s.ctx, oldToken)
	s.ErrorIs(err, model.ErrInvalidToken)
}

// TestRefresh_UserNotFound проверяет случай, когда сессия есть, а пользователя в системе больше нет.
func (s *AuthServiceTestSuite) TestRefresh_UserNotFound() {
	oldToken := "old"
	s.sRepo.On("GetByToken", s.ctx, oldToken).Return(&model.Session{UserID: 99}, nil)
	s.uRepo.On("GetByID", s.ctx, 99).Return(nil, model.ErrUserNotFound)

	_, err := s.service.Refresh(s.ctx, oldToken)
	s.ErrorIs(err, model.ErrInvalidToken)
}

// --- LOGOUT TESTS ---

// TestLogout_Success проверяет успешное удаление сессии при логауте.
func (s *AuthServiceTestSuite) TestLogout_Success() {
	s.sRepo.On("DeleteByToken", s.ctx, "tok").Return(nil)
	s.NoError(s.service.Logout(s.ctx, "tok"))
}

// TestLogout_DBError проверяет поведение при ошибке базы данных во время логаута.
func (s *AuthServiceTestSuite) TestLogout_DBError() {
	s.sRepo.On("DeleteByToken", s.ctx, "tok").Return(errors.New("db error"))
	s.Error(s.service.Logout(s.ctx, "tok"))
}

// TestAuthServiceSuite запускает весь набор тестов для AuthService.
func TestAuthServiceSuite(t *testing.T) {
	suite.Run(t, new(AuthServiceTestSuite))
}

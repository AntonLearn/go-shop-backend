// Package service реализует слой бизнес-логики приложения (Application Service Layer).
// Он координирует выполнение доменных правил, управляет транзакциями и изолирует
// внутренние процессы от транспортного уровня (HTTP/gRPC) и деталей хранения данных.
package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/antonlearn/go-shop-backend/internal/model"
	"github.com/antonlearn/go-shop-backend/pkg/logger"
)

// Константы конфигурации безопасности
const (
	accessTokenTTL  = 15 * time.Minute    // Короткоживущий токен доступа
	refreshTokenTTL = 30 * 24 * time.Hour // Долгоживущий токен обновления (30 дней)
)

// TokenGenerator описывает контракт для генерации JWT токенов доступа.
// Это позволяет изолировать бизнес-логику от конкретной реализации JWT-библиотеки.
type TokenGenerator interface {
	GenerateToken(userID int, email, role string, ttl time.Duration) (string, error)
}

// UserRepository описывает требования сервиса к хранилищу пользователей.
type UserRepository interface {
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	Create(ctx context.Context, user *model.User) error
	// Используется при ротации токенов (Refresh) для проверки актуального статуса пользователя
	GetByID(ctx context.Context, id int) (*model.User, error)
}

// SessionRepository описывает требования к хранилищу сессий (Refresh-токенов).
type SessionRepository interface {
	Create(ctx context.Context, session *model.Session) error
	GetByToken(ctx context.Context, token string) (*model.Session, error)
	DeleteByToken(ctx context.Context, token string) error
}

// PasswordHasher адаптирует наш pkg/hash под нужды бизнес-логики.
type PasswordHasher interface {
	GenerateHash(password string) (string, error)
	Compare(hash, password string) error
}

// AuthService оркестрирует процессы аутентификации, регистрации пользователей,
// а также жизненный цикл сессий по паттерну Rolling Sessions (Token Rotation).
type AuthService struct {
	userRepo       UserRepository
	sessionRepo    SessionRepository
	hasher         PasswordHasher
	tokenGenerator TokenGenerator
	log            *logger.Logger
}

// NewAuthService конструирует новый сервис аутентификации.
func NewAuthService(
	uRepo UserRepository,
	sRepo SessionRepository,
	hasher PasswordHasher,
	tokenGen TokenGenerator,
	log *logger.Logger,
) *AuthService {
	return &AuthService{
		userRepo:       uRepo,
		sessionRepo:    sRepo,
		hasher:         hasher,
		tokenGenerator: tokenGen,
		log:            log,
	}
}

// SignUp принимает валидированную DTO-структуру SignUpInput, регистрирует пользователя и возвращает токены.
func (s *AuthService) SignUp(ctx context.Context, input model.SignUpInput) (model.AuthResponse, error) {
	// 1. Проверяем, свободен ли email
	existingUser, err := s.userRepo.GetByEmail(ctx, input.Email)
	if err != nil && !errors.Is(err, model.ErrUserNotFound) {
		s.log.ErrorContext(ctx, "ошибка проверки уникальности email", "email", input.Email, "error", err)
		return model.AuthResponse{}, fmt.Errorf("signup check failed: %w", err)
	}
	if existingUser != nil {
		s.log.WarnContext(ctx, "попытка регистрации дубликата email", "email", input.Email)
		return model.AuthResponse{}, model.ErrUserAlreadyExists
	}

	// 2. Хэшируем пароль
	passwordHash, err := s.hasher.GenerateHash(input.Password)
	if err != nil {
		s.log.ErrorContext(ctx, "критический сбой хэширования пароля", "error", err)
		return model.AuthResponse{}, fmt.Errorf("password hashing failed: %w", err)
	}

	// 3. Собираем доменную модель User
	newUser := &model.User{
		Email:        input.Email,
		Name:         input.Name,
		PasswordHash: passwordHash,
		Role:         "client",
	}

	// 4. Пишем в базу данных
	if err := s.userRepo.Create(ctx, newUser); err != nil {
		return model.AuthResponse{}, fmt.Errorf("user storage failed: %w", err)
	}
	s.log.InfoContext(ctx, "бизнес-логика: успешно создан новый пользователь", "user_id", newUser.ID)

	// 5. Сразу авторизуем пользователя после успешной регистрации
	return s.generateTokensPair(ctx, newUser)
}

// SignIn аутентифицирует пользователя и возвращает структуру ответов с токенами.
func (s *AuthService) SignIn(ctx context.Context, input model.SignInInput) (model.AuthResponse, error) {
	// 1. Ищем пользователя по email
	user, err := s.userRepo.GetByEmail(ctx, input.Email)
	if err != nil {
		if errors.Is(err, model.ErrUserNotFound) {
			return model.AuthResponse{}, model.ErrInvalidCredentials
		}
		return model.AuthResponse{}, fmt.Errorf("login lookup failed: %w", err)
	}

	// 2. Сверяем хэши паролей
	if err := s.hasher.Compare(user.PasswordHash, input.Password); err != nil {
		s.log.WarnContext(ctx, "неверный пароль для учетной записи", "user_id", user.ID)
		return model.AuthResponse{}, model.ErrInvalidCredentials
	}

	// 3. Выпускаем сессионный пакет
	return s.generateTokensPair(ctx, user)
}

// Refresh реализует Token Rotation (безопасное обновление протухшего Access-токена)
func (s *AuthService) Refresh(ctx context.Context, oldRefreshToken string) (model.AuthResponse, error) {
	// 1. Извлекаем сессию из базы
	session, err := s.sessionRepo.GetByToken(ctx, oldRefreshToken)
	if err != nil {
		if errors.Is(err, model.ErrSessionNotFound) {
			return model.AuthResponse{}, model.ErrInvalidToken
		}
		return model.AuthResponse{}, fmt.Errorf("refresh session look up failed: %w", err)
	}

	// 2. Проверяем валидность по времени
	if time.Now().After(session.ExpiresAt) {
		s.log.WarnContext(ctx, "рефреш токен устарел, удаление сессии", "user_id", session.UserID)
		_ = s.sessionRepo.DeleteByToken(ctx, oldRefreshToken)
		return model.AuthResponse{}, model.ErrInvalidToken
	}

	// 3. Получаем свежие данные пользователя (проверка блокировок или смены роли)
	user, err := s.userRepo.GetByID(ctx, session.UserID)
	if err != nil {
		if errors.Is(err, model.ErrUserNotFound) {
			return model.AuthResponse{}, model.ErrInvalidToken
		}
		return model.AuthResponse{}, fmt.Errorf("user validation during refresh failed: %w", err)
	}

	// 4. Защита: удаляем старый использованный Refresh-токен
	if err := s.sessionRepo.DeleteByToken(ctx, oldRefreshToken); err != nil {
		s.log.ErrorContext(ctx, "не удалось удалить отработанный refresh токен", "error", err)
		return model.AuthResponse{}, err
	}

	// 5. Генерируем совершенно новую пару
	return s.generateTokensPair(ctx, user)
}

// Logout аннулирует текущую сессию пользователя на сервере.
func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	if err := s.sessionRepo.DeleteByToken(ctx, refreshToken); err != nil {
		s.log.ErrorContext(ctx, "ошибка удаления сессии при выходе", "error", err)
		return fmt.Errorf("logout operation failed: %w", err)
	}
	return nil
}

// Приватный хелпер инкапсулирует совместную сборку JWT и СУБД-сессии.
func (s *AuthService) generateTokensPair(ctx context.Context, user *model.User) (model.AuthResponse, error) {
	// Генерация JWT Access Token
	accessToken, err := s.tokenGenerator.GenerateToken(user.ID, user.Email, user.Role, accessTokenTTL)
	if err != nil {
		s.log.ErrorContext(ctx, "критическая ошибка генерации JWT подписи", "user_id", user.ID, "error", err)
		return model.AuthResponse{}, fmt.Errorf("access token generation aborted: %w", err)
	}

	// Генерация случайного безопасного Refresh токена
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return model.AuthResponse{}, fmt.Errorf("crypto randomness failure: %w", err)
	}
	refreshToken := hex.EncodeToString(bytes)

	// Сохранение сессии в БД
	session := &model.Session{
		UserID:       user.ID,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(refreshTokenTTL),
	}

	if err := s.sessionRepo.Create(ctx, session); err != nil {
		return model.AuthResponse{}, fmt.Errorf("session persistence failed: %w", err)
	}

	return model.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

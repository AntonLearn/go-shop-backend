// Package jwt инкапсулирует логику работы с криптографическими токенами доступа (Access Tokens).
// Предоставляет потокобезопасные механизмы для инициализации секретных ключей,
// генерации токенов с полезной нагрузкой (Claims) и сквозной валидации подписей.
package jwt

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Ошибки пакета для стандартизации ответов на уровне API и логирования.
var (
	ErrSecretNotInitialized = errors.New("jwt secret key is not initialized")
	ErrInvalidToken         = errors.New("invalid or expired token")
	ErrEmptySecret          = errors.New("jwt secret key cannot be empty")
)

// jwtSecret хранит приватный ключ в виде слайса байт для подписи и проверки токенов.
// Переменная является приватной для пакета, что исключает её случайную модификацию извне.
// Безопасна для конкурентного чтения после инициализации in main.go.
var jwtSecret []byte

// CustomClaims описывает внутреннюю структуру (полезную нагрузку) нашего JWT-токена.
type CustomClaims struct {
	UserID int    `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// InitJWT безопасно инициализирует секретный ключ токенов на этапе запуска приложения.
func InitJWT(secret string) error {
	if secret == "" {
		return ErrEmptySecret
	}
	jwtSecret = []byte(secret)
	return nil
}

// GenerateToken генерирует новый подписанный Access-токен с заданным сроком жизни (ttl).
// Алгоритм подписи: HMAC-SHA256 (HS256).
func GenerateToken(userID int, email, role string, ttl time.Duration) (string, error) {
	if len(jwtSecret) == 0 {
		return "", ErrSecretNotInitialized
	}

	claims := CustomClaims{
		UserID: userID,
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)), // Срок действия теперь динамический
			IssuedAt:  jwt.NewNumericDate(time.Now()),          // Время выпуска
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Подписываем токен нашим секретным ключом
	signedToken, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return signedToken, nil
}

// ValidateToken принимает зашифрованную строку токена, проверяет его целостность,
// актуальность срока действия (exp) и валидность криптографической подписи.
func ValidateToken(tokenString string) (*CustomClaims, error) {
	if len(jwtSecret) == 0 {
		return nil, ErrSecretNotInitialized
	}

	// Парсим токен и сразу валидируем алгоритм подписи
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	// Проверяем успешность парсинга и флаг валидности от библиотеки jwt
	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, ErrInvalidToken
}

// --- Адаптер для Dependency Injection ---

// TokenManager реализует интерфейсы service.TokenGenerator и handler.TokenParserInterface.
// Он предоставляет обертку над функциями пакета для внедрения зависимостей.
type TokenManager struct{}

// NewTokenManager создает новый экземпляр TokenManager.
func NewTokenManager() *TokenManager {
	return &TokenManager{}
}

// GenerateToken генерирует JWT токен, используя логику пакета.
// Этот метод позволяет TokenManager соответствовать интерфейсу service.TokenGenerator.
func (tm *TokenManager) GenerateToken(userID int, email, role string, ttl time.Duration) (string, error) {
	return GenerateToken(userID, email, role, ttl)
}

// ValidateToken валидирует JWT токен, используя логику пакета.
// Этот метод позволяет TokenManager напрямую соответствовать интерфейсу handler.TokenParserInterface без костылей.
func (tm *TokenManager) ValidateToken(tokenString string) (*CustomClaims, error) {
	return ValidateToken(tokenString)
}

// Package hash предоставляет инструменты для безопасного шифрования данных.
// Он инкапсулирует алгоритм криптографического хеширования паролей Bcrypt,
// реализуя абстрактный интерфейс для генерации и сравнения защищенных хешей.
package hash

import (
	"golang.org/x/crypto/bcrypt"
)

type BcryptHasher struct{}

func NewBcryptHasher() *BcryptHasher {
	return &BcryptHasher{}
}

// Compare сравнивает чистый пароль с его криптографическим хешем.
func (h *BcryptHasher) Compare(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

// GenerateHash создает безопасный Bcrypt-хеш из строки чистого пароля.
// Вместо ручного проброса числового уровня стоимости (cost), метод инкапсулирует
// внутри себя стандартную константу bcrypt.DefaultCost (равную 10).
func (h *BcryptHasher) GenerateHash(password string) (string, error) {
	// Используем официальную стандартную константу пакета bcrypt
	hashpassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashpassword), nil
}

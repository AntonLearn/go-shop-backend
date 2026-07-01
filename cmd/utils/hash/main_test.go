// Package main предоставляет модульные тесты для исполнительной логики CLI-утилиты.
// Тесты используют виртуальные буферы байтов bytes.Buffer для эмуляции пользовательского
// ввода и перехвата консольного вывода, гарантируя изоляцию от операционной системы.
package main

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRun_WithFlag(t *testing.T) {
	// Имитируем запуск утилиты с флагом: ./hasher -pass my_secret_pass
	args := []string{"cmd", "-pass", "my_secret_pass"}
	var stdin bytes.Buffer
	var stdout bytes.Buffer
	ctx := context.Background()

	err := run(ctx, args, &stdin, &stdout)

	// Проверяем, что утилита отработала без ошибок и вывела корректные данные
	assert.NoError(t, err)
	output := stdout.String()
	assert.Contains(t, output, "Чистый пароль: my_secret_pass")
	assert.Contains(t, output, "Хэш для БД:    $2a$") // Накапливаемый префикс bcrypt-хэшей
}

func TestRun_WithInteractiveStdin(t *testing.T) {
	// Имитируем запуск без флагов: ./hasher
	args := []string{"cmd"}
	var stdin bytes.Buffer
	// Записываем в буфер ввода пароль и эмулируем нажатие Enter (\n)
	stdin.WriteString("stdin_password\n")
	var stdout bytes.Buffer
	ctx := context.Background()

	err := run(ctx, args, &stdin, &stdout)

	assert.NoError(t, err)
	output := stdout.String()
	// Проверяем, что утилита честно попросила пользователя ввести данные
	assert.Contains(t, output, "Введите пароль для хеширования:")
	assert.Contains(t, output, "Чистый пароль: stdin_password")
	assert.Contains(t, output, "Хэш для БД:")
}

func TestRun_EmptyInputError(t *testing.T) {
	// Имитируем запуск без флагов и пустой ввод от пользователя (просто нажали Enter)
	args := []string{"cmd"}
	var stdin bytes.Buffer
	stdin.WriteString("\n")
	var stdout bytes.Buffer
	ctx := context.Background()

	err := run(ctx, args, &stdin, &stdout)

	// Проверяем, что логика выбросила ошибку валидации
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "пароль не может быть пустым")

	// Проверяем, что разметка с хэшем не вывелась на экран
	assert.NotContains(t, stdout.String(), "Хэш для БД:")
}

func TestRun_ContextCancelled(t *testing.T) {
	// Имитируем запуск без флагов, но с уже отмененным контекстом
	args := []string{"cmd"}
	var stdin bytes.Buffer
	var stdout bytes.Buffer

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Отменяем контекст до выполнения

	err := run(ctx, args, &stdin, &stdout)

	// Проверяем, что выполнение прервалось по ошибке контекста
	assert.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
	assert.NotContains(t, stdout.String(), "Хэш для БД:")
}

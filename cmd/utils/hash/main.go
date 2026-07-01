// Package main реализует автономную утилиту командной строки (CLI) для безопасного
// хэширования паролей по алгоритму bcrypt. Эта утилита используется разработчиками
// и администраторами для генерации криптографических хэшей «на лету» с целью их
// последующей ручной вставки в базу данных SQLite (например, при сидировании первичных
// данных пользователей или создании учетной записи суперадминистратора).
//
// Поддерживает два режима работы: прямую передачу целевого значения через флаг -pass
// и интерактивный ввод из стандартного потока (stdin) при запуске без флагов.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"

	"golang.org/x/crypto/bcrypt"
)

const failExitCode = 1

func main() {
	ctx := context.Background()
	// Вызываем исполнительную логику утилиты, передавая реальные системные потоки и аргументы.
	if err := run(ctx, os.Args, os.Stdin, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "\n[ERROR] %v\n", err)
		os.Exit(failExitCode)
	}
}

// run инкапсулирует логику работы CLI-утилиты. Выделение потоков io.Reader и io.Writer
// в параметры позволяет полностью изолировать логику и протестировать её в unit-тестах
// без блокировок консоли и вызовов жесткого завершения процесса через os.Exit.
func run(ctx context.Context, args []string, stdin io.Reader, stdout io.Writer) error {
	// Используем локальный FlagSet вместо глобального flag.CommandLine,
	// чтобы тесты могли безопасно выполняться параллельно и не конфликтовать.
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	flags.SetOutput(stdout)

	passFlag := flags.String("pass", "", "Пароль для хеширования")

	// Парсим переданные аргументы (пропуская нулевой элемент — имя бинаря)
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}

	password := *passFlag

	// Если флаг не передан, переключаемся в интерактивный режим чтения из потока stdin
	if password == "" {
		fmt.Fprint(stdout, "Введите пароль для хеширования: ")

		if err := ctx.Err(); err != nil {
			return err
		}

		_, err := fmt.Fscanln(stdin, &password)
		if err != nil || password == "" {
			return fmt.Errorf("пароль не может быть пустым")
		}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("не удалось сгенерировать хэш: %w", err)
	}

	fmt.Fprintln(stdout, "\n========================================================")
	fmt.Fprintf(stdout, "Чистый пароль: %s\n", password)
	fmt.Fprintf(stdout, "Хэш для БД:    %s\n", string(hash))
	fmt.Fprintln(stdout, "========================================================")

	return nil
}

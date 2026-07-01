// Package handler реализует транспортный слой (HTTP) приложения.
// Данный файл (export_test.go) экспортирует приватные хелперы ответа
// исключительно для целей тестирования пакетом handler_test.
package handler

var (
	RespondWithJSON  = respondWithJSON
	RespondWithError = respondWithError
)

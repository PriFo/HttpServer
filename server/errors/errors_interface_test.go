//go:build ignore

package errors

import (
	"httpserver/server/middleware"
)

// Проверка, что AppError реализует интерфейс middleware.HTTPError
// Этот файл не компилируется в обычной сборке (build tag ignore)
// но может быть использован для проверки интерфейса вручную
var _ middleware.HTTPError = (*AppError)(nil)


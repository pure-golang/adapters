// Package encoders предоставляет кодировщики для сообщений очередей.
//
// Реализует интерфейс [queue.Encoder] для преобразования данных в []byte.
// Доступные кодировщики:
//   - [JSON] — JSON кодирование (application/json)
//   - [Text] — текстовое кодирование (text/plain)
//
// Использование (JSON):
//
//	enc := encoders.JSON{}
//	data, err := enc.Encode(myStruct)
//	contentType := enc.ContentType() // "application/json"
//
// Использование (Text):
//
//	enc := encoders.Text{}
//	data, err := enc.Encode("string or []byte")
//	contentType := enc.ContentType() // "text/plain"
//
// Особенности:
//   - JSON использует стандартный encoding/json
//   - Text принимает только string или []byte
//   - ContentType() возвращает MIME-тип для заголовков
package encoders

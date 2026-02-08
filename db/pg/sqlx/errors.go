package sqlx

import (
	"errors"
	"strings"

	"github.com/lib/pq"
)

// Коды ошибок PostgreSQL
const (
	UniqueViolationCode     = pq.ErrorCode("23505")
	ForeignKeyViolationCode = pq.ErrorCode("23503")
	CheckViolationCode      = pq.ErrorCode("23514")
	NotNullViolationCode    = pq.ErrorCode("23502")
)

// IsUniqueViolation проверяет, является ли ошибка нарушением ограничения уникальности
func IsUniqueViolation(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == UniqueViolationCode
}

// IsForeignKeyViolation проверяет, является ли ошибка нарушением внешнего ключа
func IsForeignKeyViolation(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == ForeignKeyViolationCode
}

// IsCheckViolation проверяет, является ли ошибка нарушением ограничения CHECK
func IsCheckViolation(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == CheckViolationCode
}

// IsNotNullViolation проверяет, является ли ошибка нарушением ограничения NOT NULL
func IsNotNullViolation(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == NotNullViolationCode
}

// IsConstraintViolation проверяет, является ли ошибка нарушением любого ограничения
func IsConstraintViolation(err error) bool {
	var pqErr *pq.Error
	if !errors.As(err, &pqErr) {
		return false
	}

	return pqErr.Code == UniqueViolationCode ||
		pqErr.Code == ForeignKeyViolationCode ||
		pqErr.Code == CheckViolationCode ||
		pqErr.Code == NotNullViolationCode
}

// GetConstraintName извлекает имя нарушенного ограничения из ошибки
func GetConstraintName(err error) string {
	var pqErr *pq.Error
	if !errors.As(err, &pqErr) {
		return ""
	}

	// Формат сообщения: ошибка нарушения ограничения уникальности "constraint_name"
	parts := strings.Split(pqErr.Message, "\"")
	if len(parts) >= 3 {
		return parts[1]
	}
	return ""
}

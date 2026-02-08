package pgx

import (
	"github.com/jackc/pgconn"
	"github.com/pkg/errors"
)

// ErrorCode from https://www.postgresql.org/docs/current/errcodes-appendix.html
type ErrorCode string

const (
	UniqueViolation     ErrorCode = "23505"
	ForeignKeyViolation ErrorCode = "23503"
	CheckViolation      ErrorCode = "23514"
)

func (e ErrorCode) String() string {
	return string(e)
}

// ErrorIs checks if error is *pgconn.PgError and compares codes
func ErrorIs(err error, code ErrorCode) (*pgconn.PgError, bool) {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return nil, false
	}
	if pgErr.Code == string(code) {
		return pgErr, true
	}
	return nil, false
}

// FromError converts error to *pgconn.PgError if it's possible
func FromError(err error) (*pgconn.PgError, bool) {
	if err == nil {
		return nil, false
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr, true
	}
	return nil, false
}

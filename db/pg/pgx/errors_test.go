package pgx

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErrorIs_MatchingCode(t *testing.T) {
	// Create a pgconn.PgError with UniqueViolation code
	pgErr := &pgconn.PgError{
		Code:    string(UniqueViolation),
		Message: "duplicate key value violates unique constraint",
	}

	// Test ErrorIs with matching code
	result, isMatch := ErrorIs(pgErr, UniqueViolation)

	require.True(t, isMatch)
	require.NotNil(t, result)
	assert.Equal(t, string(UniqueViolation), result.Code)
	assert.Equal(t, "duplicate key value violates unique constraint", result.Message)
}

func TestErrorIs_NonMatchingCode(t *testing.T) {
	// Create a pgconn.PgError with UniqueViolation code
	pgErr := &pgconn.PgError{
		Code:    string(UniqueViolation),
		Message: "duplicate key value violates unique constraint",
	}

	// Test ErrorIs with different code (ForeignKeyViolation)
	result, isMatch := ErrorIs(pgErr, ForeignKeyViolation)

	require.False(t, isMatch)
	require.Nil(t, result)
}

func TestErrorIs_NonPgError(t *testing.T) {
	// Test with a standard Go error
	stdErr := errors.New("standard error")

	result, isMatch := ErrorIs(stdErr, UniqueViolation)

	require.False(t, isMatch)
	require.Nil(t, result)
}

func TestErrorIs_NilError(t *testing.T) {
	result, isMatch := ErrorIs(nil, UniqueViolation)

	require.False(t, isMatch)
	require.Nil(t, result)
}

func TestErrorIs_WrappedPgError(t *testing.T) {
	// Create a pgconn.PgError wrapped in another error
	pgErr := &pgconn.PgError{
		Code:    string(ForeignKeyViolation),
		Message: "insert or update on table violates foreign key constraint",
	}
	wrappedErr := fmt.Errorf("database operation failed: %w", pgErr)

	// Test ErrorIs with wrapped error
	result, isMatch := ErrorIs(wrappedErr, ForeignKeyViolation)

	require.True(t, isMatch)
	require.NotNil(t, result)
	assert.Equal(t, string(ForeignKeyViolation), result.Code)
}

func TestErrorIs_AllErrorCodeCombinations(t *testing.T) {
	// Test all defined error codes
	codes := []ErrorCode{
		UniqueViolation,
		ForeignKeyViolation,
		CheckViolation,
	}

	for _, code := range codes {
		t.Run(code.String(), func(t *testing.T) {
			pgErr := &pgconn.PgError{
				Code:    string(code),
				Message: "test error message",
			}

			// Should match with same code
			result, isMatch := ErrorIs(pgErr, code)
			require.True(t, isMatch)
			require.NotNil(t, result)
			assert.Equal(t, string(code), result.Code)

			// Should not match with different code
			for _, otherCode := range codes {
				if otherCode != code {
					result, isMatch := ErrorIs(pgErr, otherCode)
					require.False(t, isMatch)
					require.Nil(t, result)
				}
			}
		})
	}
}

func TestFromError_WithPgError(t *testing.T) {
	pgErr := &pgconn.PgError{
		Code:           string(CheckViolation),
		Message:        "new row violates check constraint",
		Severity:       "ERROR",
		ConstraintName: "check_positive_balance",
		TableName:      "accounts",
	}

	result, ok := FromError(pgErr)

	require.True(t, ok)
	require.NotNil(t, result)
	assert.Equal(t, string(CheckViolation), result.Code)
	assert.Equal(t, "new row violates check constraint", result.Message)
	assert.Equal(t, "ERROR", result.Severity)
	assert.Equal(t, "check_positive_balance", result.ConstraintName)
	assert.Equal(t, "accounts", result.TableName)
}

func TestFromError_WithNilError(t *testing.T) {
	result, ok := FromError(nil)

	require.False(t, ok)
	require.Nil(t, result)
}

func TestFromError_WithNonPgError(t *testing.T) {
	stdErr := errors.New("standard error")

	result, ok := FromError(stdErr)

	require.False(t, ok)
	require.Nil(t, result)
}

func TestFromError_WithWrappedPgError(t *testing.T) {
	pgErr := &pgconn.PgError{
		Code:    string(UniqueViolation),
		Message: "duplicate key",
	}
	wrappedErr := fmt.Errorf("operation failed: %w", pgErr)

	result, ok := FromError(wrappedErr)

	require.True(t, ok)
	require.NotNil(t, result)
	assert.Equal(t, string(UniqueViolation), result.Code)
}

func TestErrorCode_StringMethod(t *testing.T) {
	tests := []struct {
		name string
		code ErrorCode
		want string
	}{
		{
			name: "UniqueViolation",
			code: UniqueViolation,
			want: "23505",
		},
		{
			name: "ForeignKeyViolation",
			code: ForeignKeyViolation,
			want: "23503",
		},
		{
			name: "CheckViolation",
			code: CheckViolation,
			want: "23514",
		},
		{
			name: "CustomErrorCode",
			code: ErrorCode("42000"),
			want: "42000",
		},
		{
			name: "EmptyErrorCode",
			code: ErrorCode(""),
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.code.String())
		})
	}
}

func TestErrorIs_CheckConstraint(t *testing.T) {
	pgErr := &pgconn.PgError{
		Code:           string(CheckViolation),
		Message:        "violates check constraint",
		ConstraintName: "positive_balance",
	}

	result, isMatch := ErrorIs(pgErr, CheckViolation)

	require.True(t, isMatch)
	require.NotNil(t, result)
	assert.Equal(t, string(CheckViolation), result.Code)
	assert.Equal(t, "positive_balance", result.ConstraintName)
}

func TestErrorIs_DoubleWrappedPgError(t *testing.T) {
	// Create a double-wrapped pgconn.PgError
	pgErr := &pgconn.PgError{
		Code:    string(UniqueViolation),
		Message: "duplicate key",
	}
	firstWrap := fmt.Errorf("first wrapper: %w", pgErr)
	secondWrap := fmt.Errorf("second wrapper: %w", firstWrap)

	// Test ErrorIs with double-wrapped error
	result, isMatch := ErrorIs(secondWrap, UniqueViolation)

	require.True(t, isMatch)
	require.NotNil(t, result)
	assert.Equal(t, string(UniqueViolation), result.Code)
}

func TestFromError_MultipleWrappingLevels(t *testing.T) {
	pgErr := &pgconn.PgError{
		Code:    string(ForeignKeyViolation),
		Message: "foreign key violation",
	}
	firstWrap := fmt.Errorf("level 1: %w", pgErr)
	secondWrap := fmt.Errorf("level 2: %w", firstWrap)
	thirdWrap := fmt.Errorf("level 3: %w", secondWrap)

	result, ok := FromError(thirdWrap)

	require.True(t, ok)
	require.NotNil(t, result)
	assert.Equal(t, string(ForeignKeyViolation), result.Code)
}

func TestErrorCode_Constants(t *testing.T) {
	// Verify all error code constants are set correctly
	assert.Equal(t, ErrorCode("23505"), UniqueViolation)
	assert.Equal(t, ErrorCode("23503"), ForeignKeyViolation)
	assert.Equal(t, ErrorCode("23514"), CheckViolation)
}

func TestErrorIs_WithMultipleWrappers(t *testing.T) {
	pgErr := &pgconn.PgError{
		Code:    "23505",
		Message: "duplicate key",
	}

	// Wrap multiple times
	err := fmt.Errorf("wrap1: %w", pgErr)
	err = fmt.Errorf("wrap2: %w", err)
	err = fmt.Errorf("wrap3: %w", err)

	result, isMatch := ErrorIs(err, UniqueViolation)
	require.True(t, isMatch)
	assert.NotNil(t, result)
	assert.Equal(t, "23505", result.Code)
}

func TestFromError_WithRegularErrorChain(t *testing.T) {
	// Create a chain of regular errors
	baseErr := errors.New("base error")
	wrappedErr := fmt.Errorf("wrapped: %w", baseErr)

	result, ok := FromError(wrappedErr)
	require.False(t, ok)
	require.Nil(t, result)
}

package handlers

import (
	"errors"
	"testing"

	validation "github.com/go-ozzo/ozzo-validation/v4"
)

func TestExtractValidationError(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		if err := extractValidationError(nil); err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})

	t.Run("non validation error", func(t *testing.T) {
		original := errors.New("some random error")
		if result := extractValidationError(original); result != original {
			t.Errorf("expected original error back, got %v", result)
		}
	})

	t.Run("ozzo validation errors", func(t *testing.T) {
		ve := validation.Errors{
			"Username": errors.New("cannot be blank"),
			"Email":    errors.New("must be a valid email"),
		}
		result := extractValidationError(ve)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		apiErr, ok := errors.AsType[apiError](result)
		if !ok {
			t.Fatalf("expected apiError, got %T", result)
		}
		if apiErr.Kind != "InvalidData" {
			t.Errorf("expected Kind InvalidData, got %s", apiErr.Kind)
		}
		if _, ok := apiErr.Details.(validation.Errors); !ok {
			t.Fatalf("expected validation.Errors details, got %T", apiErr.Details)
		}
	})

	t.Run("lowercases first letter", func(t *testing.T) {
		ve := validation.Errors{"Username": errors.New("cannot be blank")}
		result := extractValidationError(ve)
		apiErr, _ := errors.AsType[apiError](result)
		details := apiErr.Details.(validation.Errors)
		if _, exists := details["username"]; !exists {
			t.Errorf("expected lowercase 'username' key, got %v", details)
		}
	})
}

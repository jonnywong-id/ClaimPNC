package apierror

import "claim-pnc/internal/platform/validation"

// FieldError adalah satu pelanggaran pada jawaban galat, berkunci `field`.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"pesan"`
}

// ColumnError adalah satu pelanggaran pada jawaban galat, berkunci `kolom`.
type ColumnError struct {
	Field   string `json:"kolom"`
	Message string `json:"pesan"`
}

// InputError adalah satu pelanggaran pada jawaban galat, berkunci `isian`.
type InputError struct {
	Field   string `json:"isian"`
	Message string `json:"pesan"`
}

// FieldErrors menyalin seluruh pelanggaran ke bentuk berkunci `field`.
func FieldErrors(list []validation.Violation) []FieldError {
	return Details(list, func(field, message string) FieldError { return FieldError{Field: field, Message: message} })
}

// ColumnErrors menyalin seluruh pelanggaran ke bentuk berkunci `kolom`.
func ColumnErrors(list []validation.Violation) []ColumnError {
	return Details(list, func(field, message string) ColumnError { return ColumnError{Field: field, Message: message} })
}

// InputErrors menyalin seluruh pelanggaran ke bentuk berkunci `isian`.
func InputErrors(list []validation.Violation) []InputError {
	return Details(list, func(field, message string) InputError { return InputError{Field: field, Message: message} })
}

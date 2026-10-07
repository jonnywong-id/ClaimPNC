package httpquery

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// NonNegative membaca bilangan bulat tak negatif dari parameter kueri; nilai yang tidak terbaca
// atau negatif menjadi 0.
func NonNegative(raw string) int {
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return 0
	}
	return value
}

// NonNegativeTrimmed seperti NonNegative, tetapi spasi di kedua ujung diabaikan lebih dulu.
func NonNegativeTrimmed(raw string) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value < 0 {
		return 0
	}
	return value
}

// NonNegativeOr membaca bilangan bulat tak negatif; teks kosong menghasilkan fallback, teks
// yang tidak terbaca atau negatif menghasilkan galat.
func NonNegativeOr(raw string, fallback int) (int, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(trimmed)
	if err != nil || value < 0 {
		return 0, fmt.Errorf("bukan bilangan bulat tak negatif")
	}
	return value, nil
}

// Date membaca tanggal berformat 2006-01-02 pada zona loc (UTC bila nil); teks kosong
// menghasilkan nil tanpa galat.
func Date(raw string, loc *time.Location) (*time.Time, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}
	if loc == nil {
		loc = time.UTC
	}
	parsed, err := time.ParseInLocation("2006-01-02", trimmed, loc)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

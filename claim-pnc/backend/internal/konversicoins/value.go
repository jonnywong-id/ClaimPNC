package konversicoins

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Column adalah satu kolom tabel tujuan menurut kamus data basis data TEST.
type Column struct {
	Name     string
	DataType string // VARCHAR2, NUMBER, …
	// MaxBytes adalah batas panjang kolom teks dalam byte (DATA_LENGTH); 0 = tidak dibatasi.
	MaxBytes int
	// MaxChars adalah batas panjang dalam karakter bila kolomnya ber-CHAR semantics.
	MaxChars int
}

// Coerce mengubah satu nilai JSON menjadi nilai yang dapat diikat ke kolom tujuan.
//
// T_COINSLIST hanya berisi kolom NUMBER dan VARCHAR2. Nilai kosong menjadi NULL. Angka yang
// bukan angka menjadi NULL dan peringatannya dikembalikan supaya dilaporkan — baris tetap
// ditulis, sebab satu isian rusak tidak boleh menggagalkan seluruh polis.
func Coerce(value any, column Column) (any, string) {
	if isBlank(value) {
		return nil, ""
	}
	switch column.DataType {
	case "NUMBER", "FLOAT", "BINARY_DOUBLE", "BINARY_FLOAT", "INTEGER":
		text := strings.ReplaceAll(strings.TrimSpace(asText(value)), ",", "")
		number, err := strconv.ParseFloat(text, 64)
		if err != nil {
			return nil, fmt.Sprintf("%s: %q bukan angka, ditulis NULL", column.Name, shorten(text))
		}
		return number, ""
	}
	text := asText(value)
	cut := truncate(text, column)
	if cut != text {
		return cut, fmt.Sprintf("%s: dipotong dari %d ke %d karakter", column.Name,
			utf8.RuneCountInString(text), utf8.RuneCountInString(cut))
	}
	return text, ""
}

// asText mengubah nilai JSON apa pun menjadi teksnya.
func asText(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case json.Number:
		return string(v)
	case bool:
		return strconv.FormatBool(v)
	case nil:
		return ""
	}
	return fmt.Sprint(value)
}

// truncate memotong teks ke batas kolom tanpa memecah satu karakter UTF-8.
func truncate(text string, column Column) string {
	if column.MaxChars > 0 && utf8.RuneCountInString(text) > column.MaxChars {
		text = string([]rune(text)[:column.MaxChars])
	}
	if column.MaxBytes > 0 && len(text) > column.MaxBytes {
		cut := column.MaxBytes
		for cut > 0 && !utf8.RuneStart(text[cut]) {
			cut--
		}
		text = text[:cut]
	}
	return text
}

func shorten(text string) string {
	if utf8.RuneCountInString(text) <= 40 {
		return text
	}
	return string([]rune(text)[:40]) + "…"
}

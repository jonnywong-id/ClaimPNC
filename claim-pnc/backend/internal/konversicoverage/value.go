package konversicoverage

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// Column adalah satu kolom tabel tujuan menurut kamus data basis data TEST.
type Column struct {
	Name     string
	DataType string // VARCHAR2, NUMBER, DATE, BLOB, CLOB, …
	// MaxBytes adalah batas panjang kolom teks dalam byte (DATA_LENGTH); 0 = tidak dibatasi.
	MaxBytes int
	// MaxChars adalah batas panjang dalam karakter bila kolomnya ber-CHAR semantics.
	MaxChars int
}

// jakarta dipakai mengubah waktu GMT pada dokumen Pega menjadi tanggal bisnis.
var jakarta = loadJakarta()

func loadJakarta() *time.Location {
	if loc, err := time.LoadLocation("Asia/Jakarta"); err == nil {
		return loc
	}
	return time.FixedZone("WIB", 7*60*60)
}

// Coerce mengubah satu nilai JSON menjadi nilai yang dapat diikat ke kolom tujuan.
//
// Nilai kosong menjadi NULL. Bila nilai tidak dapat diubah (angka yang bukan angka,
// tanggal yang formatnya tidak dikenal), hasilnya NULL dan peringatannya dikembalikan
// supaya dilaporkan — baris tetap ditulis, sebab satu isian rusak tidak boleh
// menggagalkan seluruh polis.
func Coerce(value any, column Column) (any, string) {
	if isBlank(value) {
		return nil, ""
	}
	switch {
	case column.DataType == "NUMBER" || column.DataType == "FLOAT" ||
		column.DataType == "BINARY_DOUBLE" || column.DataType == "BINARY_FLOAT" ||
		column.DataType == "INTEGER":
		text := strings.ReplaceAll(strings.TrimSpace(asText(value)), ",", "")
		switch strings.ToLower(text) {
		case "true":
			return float64(1), ""
		case "false":
			return float64(0), ""
		}
		number, err := strconv.ParseFloat(text, 64)
		if err != nil {
			return nil, fmt.Sprintf("%s: %q bukan angka, ditulis NULL", column.Name, shorten(text))
		}
		return number, ""

	case column.DataType == "DATE" || strings.HasPrefix(column.DataType, "TIMESTAMP"):
		text := strings.TrimSpace(asText(value))
		parsed, ok := parseDate(text)
		if !ok {
			return nil, fmt.Sprintf("%s: tanggal %q tidak dikenali, ditulis NULL", column.Name, shorten(text))
		}
		return parsed, ""

	case column.DataType == "BLOB" || column.DataType == "RAW" || column.DataType == "LONG RAW":
		return []byte(asText(value)), ""

	case column.DataType == "CLOB" || column.DataType == "NCLOB":
		return asText(value), ""
	}

	// Teks: VARCHAR2, CHAR, NVARCHAR2, dan tipe lain yang tidak dikenali.
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
	case json.RawMessage:
		return string(v)
	case nil:
		return ""
	}
	return fmt.Sprint(value)
}

// truncate memotong teks ke batas kolom tanpa memecah satu karakter UTF-8.
func truncate(text string, column Column) string {
	if column.MaxChars > 0 && utf8.RuneCountInString(text) > column.MaxChars {
		runes := []rune(text)
		text = string(runes[:column.MaxChars])
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

// Format tanggal yang dikenali.
//
// Pega menyimpan DateTime sebagai `20240115T170000.000 GMT` — waktu GMT. Yang ditulis
// adalah TANGGAL-JAM WIB-nya, karena itulah tanggal bisnis yang dilihat pengguna;
// `20240115T170000.000 GMT` adalah 16 Januari 00:00 WIB, bukan 15 Januari.
var dateLayouts = []struct {
	layout string
	gmt    bool
}{
	{"20060102T150405.000 MST", true},
	{"20060102T150405 MST", true},
	{"20060102T150405.000", true},
	{"20060102", false},
	{"2006-01-02T15:04:05Z07:00", true},
	{"2006-01-02T15:04:05.000Z07:00", true},
	{"2006-01-02 15:04:05", false},
	{"2006-01-02", false},
	{"02/01/2006 15:04:05", false},
	{"02/01/2006", false},
	{"02-Jan-2006", false},
}

func parseDate(text string) (time.Time, bool) {
	for _, l := range dateLayouts {
		parsed, err := time.Parse(l.layout, text)
		if err != nil {
			continue
		}
		if l.gmt {
			wall := parsed.In(jakarta)
			return time.Date(wall.Year(), wall.Month(), wall.Day(), wall.Hour(), wall.Minute(),
				wall.Second(), 0, time.UTC), true
		}
		return parsed, true
	}
	return time.Time{}, false
}

func shorten(text string) string {
	if utf8.RuneCountInString(text) <= 40 {
		return text
	}
	return string([]rune(text)[:40]) + "…"
}

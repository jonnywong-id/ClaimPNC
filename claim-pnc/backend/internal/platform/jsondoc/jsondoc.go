// Package jsondoc membaca dokumen JSON klaim (`POOLDATA.JSON_KLAIM.DATA_JSONBLOB`) sebagai peta
// bebas: menelusuri jalur bertitik, menuliskan nilai sebagai teks, dan menyusun baris grid.
//
// Angka dibaca sebagai teks apa adanya, bukan float64: hampir setiap angka di dokumen itu
// adalah nilai uang atau persentase share, dan float64 mengubah `1234567890123.45` menjadi
// nilai yang dibulatkan tanpa satu pun galat (`I-12`).
package jsondoc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Document adalah dokumen JSON yang sudah diurai.
type Document map[string]any

// Parse mengurai dokumen; teks kosong menjadi dokumen kosong, bukan galat.
func Parse(raw string) (Document, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return Document{}, nil
	}
	decoder := json.NewDecoder(bytes.NewReader([]byte(trimmed)))
	decoder.UseNumber()
	var parsed Document
	if err := decoder.Decode(&parsed); err != nil {
		return nil, fmt.Errorf("mengurai dokumen klaim: %w", err)
	}
	return parsed, nil
}

// Lookup menelusuri jalur bertitik, misalnya "PolicyData.PolicyNo". Nilai kedua false bila
// jalurnya kosong atau salah satu langkahnya tidak ada.
func (d Document) Lookup(path string) (any, bool) {
	if path == "" {
		return nil, false
	}
	var current any = map[string]any(d)
	for _, step := range strings.Split(path, ".") {
		object, isObject := current.(map[string]any)
		if !isObject {
			return nil, false
		}
		next, exists := object[step]
		if !exists {
			return nil, false
		}
		current = next
	}
	return current, true
}

// Text menuliskan nilai tunggal sebagai teks. Boolean menjadi "true"/"false" — penerjemahannya
// keputusan tampilan. Objek dan larik menjadi teks kosong: nilai berstruktur di tempat nilai
// tunggal adalah tanda jalurnya salah, dan menumpahkannya ke sel tabel hanya memindahkan
// kebingungannya ke pengguna.
func Text(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case json.Number:
		return typed.String()
	case bool:
		return strconv.FormatBool(typed)
	default:
		return ""
	}
}

// Column memetakan satu kolom grid: Key di baris hasil, Path di objek baris dokumen. Path
// kosong berarti kolomnya tidak punya sumber.
type Column struct {
	Key  string
	Path string
}

// Rows menyusun baris grid dari nilai larik — atau objek tunggal, yang diperlakukan sebagai
// satu baris, karena dokumen Pega kadang menyimpan larik berisi satu baris sebagai objek.
// Butir yang bukan objek dilewati. Nilai lain menghasilkan nil.
func Rows(value any, columns []Column) []map[string]string {
	switch typed := value.(type) {
	case []any:
		result := make([]map[string]string, 0, len(typed))
		for _, item := range typed {
			if row, ok := rowOf(item, columns); ok {
				result = append(result, row)
			}
		}
		return result
	case map[string]any:
		if row, ok := rowOf(typed, columns); ok {
			return []map[string]string{row}
		}
		return nil
	default:
		return nil
	}
}

func rowOf(item any, columns []Column) (map[string]string, bool) {
	object, isObject := item.(map[string]any)
	if !isObject {
		return nil, false
	}
	row := make(map[string]string, len(columns))
	for _, column := range columns {
		if column.Path == "" {
			continue
		}
		row[column.Key] = Text(object[column.Path])
	}
	return row, true
}

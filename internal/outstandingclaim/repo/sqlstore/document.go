package sqlstore

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"claim-pnc/internal/outstandingclaim"
)

// document adalah dokumen klaim yang sudah diurai — isi kolom
// `POOLDATA.JSON_KLAIM.DATA_JSONBLOB`.
//
// # Kenapa peta bebas, bukan struct berisi 97 field
//
// Karena bentuk dokumen ini BELUM PERNAH DIPERIKSA: DDL tabelnya tidak tersedia (`R-08`) dan
// isinya belum pernah dilihat. Struct akan menyatakan bentuk yang belum terbukti, dan isian
// yang namanya ternyata berbeda akan hilang tanpa satu pun tanda.
//
// Peta bebas membuat ketidakcocokan TERLIHAT: isian yang jalurnya tidak ada dapat dibedakan
// dari isian yang ada tetapi kosong, dan perbedaan itu yang dilaporkan sebagai temuan
// alih-alih diam-diam menjadi sel kosong di layar.
type document map[string]any

// parseDocument mengurai dokumen klaim.
//
// Dokumen KOSONG bukan galat: gabungan ke JSON_KLAIM adalah LEFT JOIN, sehingga klaim yang
// belum punya baris di sana mengembalikan NULL — dan klaim seperti itu tetap dapat dibuka,
// hanya isinya yang kosong.
//
// Dokumen yang ADA tetapi tidak dapat diurai JUSTRU galat, dan galat yang menyebut sebabnya.
// Menelannya menjadi rincian kosong berarti kerusakan data tersaji kepada pengguna sebagai
// "klaim ini memang belum diisi".
func parseDocument(raw string) (document, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return document{}, nil
	}

	decoder := json.NewDecoder(bytes.NewReader([]byte(trimmed)))

	// Angka dibaca sebagai teks apa adanya, bukan sebagai float64. Di layar ini hampir
	// setiap angka adalah NILAI UANG atau persentase share reasuransi, dan float64 mengubah
	// `1234567890123.45` menjadi nilai yang dibulatkan tanpa satu pun galat (`I-12`).
	decoder.UseNumber()

	var parsed document
	if err := decoder.Decode(&parsed); err != nil {
		return nil, fmt.Errorf("mengurai dokumen klaim: %w", err)
	}
	return parsed, nil
}

// lookup menelusuri satu jalur bertitik di dalam dokumen.
//
// Mengembalikan nilai yang ditemukan dan penanda ADA-nya. Penandanya dipisah dari nilainya
// karena "tidak ada" dan "ada tetapi kosong" adalah dua keadaan yang berbeda — yang pertama
// menunjuk jalur yang salah, yang kedua menunjuk data yang belum diisi.
func (d document) lookup(path string) (any, bool) {
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

// text mengubah satu nilai JSON menjadi teks yang siap digambar.
//
// # Kenapa boolean menjadi "true"/"false", bukan "Ya"/"Tidak"
//
// Karena penerjemahannya adalah keputusan tampilan, dan tempatnya di layar — bukan di
// pembaca penyimpanan. Menerjemahkannya di sini berarti nilai `false` dan teks `"false"`
// yang kebetulan tersimpan sebagai teks menjadi tidak dapat dibedakan lagi.
//
// Objek dan senarai menjadi teks KOSONG, bukan bentuk JSON-nya. Isian skalar yang ternyata
// berisi struktur adalah tanda jalurnya salah, dan menumpahkan JSON mentah ke sel tabel
// hanya memindahkan kebingungannya ke pengguna. Ketidakcocokan seperti itu terlihat pada
// penghitung di Stats, bukan di layar.
func text(value any) string {
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

// rows mengubah satu nilai JSON menjadi daftar baris grid.
//
// Nilai yang BUKAN senarai menghasilkan nol baris, bukan galat: Pega menulis page list yang
// berisi satu baris kadang sebagai objek tunggal, dan bentuk dokumen ini belum dapat
// diperiksa (`R-08`). Objek tunggal karena itu diperlakukan sebagai satu baris — bukan
// dibuang, karena membuang satu-satunya baris membuat grid tampak kosong padahal berisi.
func rows(value any, columns []outstandingclaim.GridColumn) []outstandingclaim.GridRow {
	switch typed := value.(type) {
	case []any:
		result := make([]outstandingclaim.GridRow, 0, len(typed))
		for _, item := range typed {
			if row, ok := rowOf(item, columns); ok {
				result = append(result, row)
			}
		}
		return result

	case map[string]any:
		if row, ok := rowOf(typed, columns); ok {
			return []outstandingclaim.GridRow{row}
		}
		return nil

	default:
		return nil
	}
}

// rowOf mengubah satu elemen senarai menjadi satu baris grid.
func rowOf(item any, columns []outstandingclaim.GridColumn) (outstandingclaim.GridRow, bool) {
	object, isObject := item.(map[string]any)
	if !isObject {
		return nil, false
	}

	row := make(outstandingclaim.GridRow, len(columns))
	for _, column := range columns {
		if column.Path == "" {
			continue
		}
		row[column.Key] = text(object[column.Path])
	}
	return row, true
}

// Stats menghitung berapa isian dan grid yang jalurnya TIDAK ditemukan di dokumen.
//
// Ia bukan hiasan. Bentuk dokumen ini belum pernah diperiksa, dan satu-satunya cara jalur
// yang salah terlihat adalah menghitungnya: layar yang seluruh isiannya kosong terbaca sama
// persis, entah karena klaimnya memang belum diisi atau karena 97 jalurnya salah semua.
//
// Angkanya dicatat di log oleh lapisan usecase, bukan ditampilkan ke pengguna.
type Stats struct {
	// FieldsFound adalah jumlah isian yang jalurnya ADA di dokumen.
	FieldsFound int

	// FieldsMissing adalah jumlah isian yang jalurnya TIDAK ada.
	//
	// Isian yang Blocked tidak dihitung di sini: ia memang tidak punya jalur.
	FieldsMissing int

	// GridsFound dan GridsMissing menghitung hal yang sama untuk senarai grid.
	GridsFound   int
	GridsMissing int
}

// readDocument memetik seluruh isian dan grid dari satu dokumen, mengikuti susunan yang
// ditetapkan section.go.
//
// Isian yang jalurnya tidak ada TIDAK dimasukkan ke hasil. Bedanya dengan isian bernilai
// kosong terbawa sampai ke layar: yang pertama tidak punya kunci sama sekali, yang kedua
// punya kunci berisi teks kosong.
func readDocument(doc document) (map[string]string, map[string][]outstandingclaim.GridRow, Stats) {
	values := map[string]string{}
	stats := Stats{}

	for _, field := range outstandingclaim.Fields() {
		if field.Blocked {
			continue
		}
		value, exists := doc.lookup(field.Path)
		if !exists {
			stats.FieldsMissing++
			continue
		}
		stats.FieldsFound++
		values[field.Key] = text(value)
	}

	gridRows := map[string][]outstandingclaim.GridRow{}
	for _, grid := range outstandingclaim.GridList() {
		if grid.Blocked {
			continue
		}
		value, exists := doc.lookup(grid.Path)
		if !exists {
			stats.GridsMissing++
			continue
		}
		stats.GridsFound++
		gridRows[grid.Code] = rows(value, grid.Columns)
	}

	return values, gridRows, stats
}

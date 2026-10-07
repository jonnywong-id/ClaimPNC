package sqlstore

import (
	"claim-pnc/internal/inputacceptation"
	"claim-pnc/internal/platform/jsondoc"
)

// document adalah dokumen klaim yang sudah diurai — isi kolom
// `POOLDATA.JSON_KLAIM.DATA_JSONBLOB`.
//
// Kolomnya `DATA_JSONBLOB`, sama dengan modul Outstanding Claim — BUKAN `DATA_JSON` yang
// dibaca keempat kueri inbox non-prop. Itu diukur, bukan disimpulkan: pada seluruh 14 klaim
// `CLMNP-%` yang punya baris di JSON_KLAIM, `DATA_JSON` berpanjang NOL sementara
// `DATA_JSONBLOB` berisi 5–10 KB. Lihat catatan di kepala inputacceptation.sql.
//
// # Kenapa peta bebas, bukan struct berisi ~50 field
//
// Karena bentuk dokumen ini BELUM PERNAH DIPERIKSA: DDL tabelnya tidak tersedia (`R-08`) dan
// isinya belum pernah dilihat. Struct akan menyatakan bentuk yang belum terbukti, dan isian
// yang namanya ternyata berbeda akan hilang tanpa satu pun tanda.
//
// Peta bebas membuat ketidakcocokan TERLIHAT: isian yang jalurnya tidak ada dapat dibedakan
// dari isian yang ada tetapi kosong, dan perbedaan itu yang dilaporkan sebagai temuan
// alih-alih diam-diam menjadi sel kosong di layar.
type document = jsondoc.Document

// parseDocument mengurai dokumen klaim.
//
// Dokumen KOSONG bukan galat: gabungan ke JSON_KLAIM adalah LEFT JOIN, sehingga klaim yang
// belum punya baris di sana mengembalikan NULL — dan klaim seperti itu tetap dapat dibuka,
// hanya isinya yang kosong.
//
// Dokumen yang ADA tetapi tidak dapat diurai JUSTRU galat, dan galat yang menyebut sebabnya.
// Menelannya menjadi rincian kosong berarti kerusakan data tersaji kepada pengguna sebagai
// "klaim ini memang belum diisi".
func parseDocument(raw string) (document, error) { return jsondoc.Parse(raw) }

// text mengubah satu nilai JSON menjadi teks yang siap digambar.
//
// Boolean menjadi "true"/"false", bukan "Ya"/"Tidak": penerjemahannya keputusan tampilan, dan
// tempatnya di layar. Menerjemahkannya di sini membuat nilai `false` dan teks `"false"` yang
// kebetulan tersimpan sebagai teks tidak dapat dibedakan lagi.
//
// Objek dan senarai menjadi teks KOSONG, bukan bentuk JSON-nya. Isian skalar yang ternyata
// berisi struktur adalah tanda jalurnya salah, dan menumpahkan JSON mentah ke sel tabel hanya
// memindahkan kebingungannya ke pengguna. Ketidakcocokan seperti itu terlihat pada penghitung
// di Stats, bukan di layar.
func text(value any) string { return jsondoc.Text(value) }

// rows mengubah satu nilai JSON menjadi daftar baris grid.
//
// Nilai yang BUKAN senarai menghasilkan nol baris, bukan galat: Pega menulis page list yang
// berisi satu baris kadang sebagai objek tunggal, dan bentuk dokumen ini belum dapat diperiksa
// (`R-08`). Objek tunggal karena itu diperlakukan sebagai satu baris — bukan dibuang, karena
// membuang satu-satunya baris membuat grid tampak kosong padahal berisi.
func rows(value any, columns []inputacceptation.GridColumn) []inputacceptation.GridRow {
	paths := make([]jsondoc.Column, len(columns))
	for i, column := range columns {
		paths[i] = jsondoc.Column{Key: column.Key, Path: column.Path}
	}
	found := jsondoc.Rows(value, paths)
	if found == nil {
		return nil
	}
	result := make([]inputacceptation.GridRow, len(found))
	for i, row := range found {
		result[i] = inputacceptation.GridRow(row)
	}
	return result
}

// Stats menghitung berapa isian dan grid yang jalurnya TIDAK ditemukan di dokumen.
//
// Ia bukan hiasan. Bentuk dokumen ini belum pernah diperiksa, dan satu-satunya cara jalur yang
// salah terlihat adalah menghitungnya: layar yang seluruh isiannya kosong terbaca sama persis,
// entah karena klaimnya memang belum diisi atau karena seluruh jalurnya salah.
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
// kosong terbawa sampai ke layar: yang pertama tidak punya kunci sama sekali, yang kedua punya
// kunci berisi teks kosong.
func readDocument(doc document) (
	map[string]string, map[string][]inputacceptation.GridRow, Stats,
) {
	values := map[string]string{}
	stats := Stats{}

	for _, field := range inputacceptation.Fields() {
		if field.Blocked {
			continue
		}
		value, exists := doc.Lookup(field.Path)
		if !exists {
			stats.FieldsMissing++
			continue
		}
		stats.FieldsFound++
		values[field.Key] = text(value)
	}

	gridRows := map[string][]inputacceptation.GridRow{}
	for _, grid := range inputacceptation.GridList() {
		if grid.Blocked {
			continue
		}
		value, exists := doc.Lookup(grid.Path)
		if !exists {
			stats.GridsMissing++
			continue
		}
		stats.GridsFound++
		gridRows[grid.Code] = rows(value, grid.Columns)
	}

	return values, gridRows, stats
}

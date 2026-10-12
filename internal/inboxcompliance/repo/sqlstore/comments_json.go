package sqlstore

import (
	"encoding/json"
	"fmt"
	"time"

	"claim-pnc/internal/inboxcompliance"
)

// Grid komentar Compliance disimpan sebagai SATU kolom JSON pada tabel keputusan —
// `KOMENTAR_JSON` pada `POOLDATA.CPNC_KEPUTUSAN_COMPLIANCE`.
//
// # Kenapa bukan tabel kedua
//
// Rancangan pertama memakai `CPNC_KOMENTAR_COMPLIANCE` tersendiri. Work Owner meminta satu
// tabel saja (2026-10-07), dan itu ternyata bukan sekadar lebih ringkas — ia MENGHAPUS satu
// kelas kegagalan:
//
// Dengan dua tabel, penyimpanan menempuh dua perintah yang bukan satu transaksi, karena
// seam Repo modul ini belum punya kepemilikan transaksi (`08-TECHNICAL-STRATEGY.md` §4.5).
// Bila yang kedua gagal, keputusan tersimpan tanpa komentarnya. Dengan satu kolom, form
// yang menyimpan keduanya dalam satu tombol kini juga menyimpannya dalam satu pernyataan.
//
// # Kenapa Go yang mengurainya, bukan JSON_TABLE
//
// Supaya kuerinya tetap SQL biasa dan portabel apa adanya ke PostgreSQL (`D-20`). Basis
// data hanya menyimpan dan memvalidasi bentuknya lewat `CHECK (KOMENTAR_JSON IS JSON)`;
// penguraian ada di Go, tempat aturan bisnis memang tinggal.

// commentJSON adalah bentuk satu komentar di dalam kolom.
//
// Nama fieldnya BERBAHASA INDONESIA, dan itu disengaja: isi kolom basis data adalah
// kontrak penyimpanan, sekelas dengan nama kolom — bukan nama internal (`D-80`). Nama yang
// sama persis dengan DTO layar juga membuat keduanya mudah dibandingkan saat menelusuri.
type commentJSON struct {
	Index int       `json:"urutan"`
	Date  time.Time `json:"tanggal"`
	Text  string    `json:"komentar"`
}

// encodeComments merakit grid menjadi satu teks JSON.
//
// Grid kosong menghasilkan **NULL**, bukan `"[]"`. Keduanya berbeda bagi pembaca SQL: NULL
// berarti "tidak ada komentar", sedangkan `"[]"` adalah senarai kosong yang tetap menempati
// ruang dan tetap harus diurai. Kolomnya memang boleh NULL.
func encodeComments(comments []inboxcompliance.Comment) (any, error) {
	if len(comments) == 0 {
		return nil, nil
	}

	rows := make([]commentJSON, 0, len(comments))
	for _, c := range comments {
		rows = append(rows, commentJSON{Index: c.Index, Date: c.Date, Text: c.Text})
	}

	encoded, err := json.Marshal(rows)
	if err != nil {
		// Praktis tidak mungkin terjadi pada tipe ini, tetapi TIDAK diabaikan: galat yang
		// ditelan di sini akan menyimpan kolom kosong pada keputusan yang punya komentar,
		// dan kehilangannya baru terlihat saat form dibuka kembali.
		return nil, fmt.Errorf("merakit grid komentar menjadi JSON: %w", err)
	}
	return string(encoded), nil
}

// decodeComments mengurai kolom JSON menjadi grid.
//
// Kolom NULL atau kosong menghasilkan grid kosong tanpa galat — itu keadaan normal setiap
// klaim yang diputuskan tanpa komentar, bukan kerusakan.
func decodeComments(raw string) ([]inboxcompliance.Comment, error) {
	if raw == "" {
		return nil, nil
	}

	var rows []commentJSON
	if err := json.Unmarshal([]byte(raw), &rows); err != nil {
		// Teks rusak DILAPORKAN, bukan dibulatkan menjadi grid kosong.
		//
		// Membulatkannya akan menyembunyikan kehilangan data: petugas membuka form,
		// melihat grid kosong, mengira belum pernah berkomentar, lalu menulis ulang —
		// dan komentar lamanya tertimpa tanpa seorang pun tahu ia pernah ada.
		return nil, fmt.Errorf("mengurai kolom KOMENTAR_JSON: %w", err)
	}

	comments := make([]inboxcompliance.Comment, 0, len(rows))
	for _, r := range rows {
		comments = append(comments, inboxcompliance.Comment{
			Index: r.Index,
			Date:  r.Date,
			Text:  r.Text,
		})
	}
	return comments, nil
}

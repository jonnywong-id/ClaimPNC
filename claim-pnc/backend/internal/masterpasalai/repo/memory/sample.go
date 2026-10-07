package memory

import (
	_ "embed"

	"claim-pnc/internal/masterpasalai"
	"claim-pnc/internal/platform/sampledata"
)

// sampleJSON memuat seluruh data contoh paket ini; lihat platform/sampledata.
//
//go:embed sample.json
var sampleJSON []byte

// SampleClause adalah baris contoh untuk pengembangan dan pengujian.
//
// # Seluruhnya KARANGAN
//
// Tidak satu pun berasal dari basis data mana pun. Isi `POOLDATA.MST_PASAL_AI` belum pernah
// dilihat siapa pun di tim ini; yang diterima adalah kuerinya, bukan datanya.
//
// `WP_ID` di sini dibuat berurutan dan berpadding tiga digit. **Bentuk aslinya tidak
// diketahui** — DDL-nya belum ada (`R-08`). Padding dipakai supaya urutan teks pada contoh
// ini sama dengan urutan angkanya; pada data sungguhan, `ORDER BY WP_ID` adalah pengurutan
// teks dan "10" akan mendahului "9".
//
// Bentuknya menyerupai wording polis asuransi umum supaya layarnya dapat dinilai dengan
// teks sepanjang yang wajar, terutama kolom **Kejadian** yang digambar `pxTextArea` dan
// paling lebar (283 px berbanding 48 dan 47).
//
// # Kenapa 28 baris, bukan segenggam
//
// Ukuran halamannya **25** (`masterpasalai.PageSize`), dan paginasinya dikerjakan di sisi
// server. Dengan kurang dari 26 baris, halaman kedua tidak pernah terbentuk — dan cacat
// paginasi adalah yang paling mudah lolos justru karena datanya terlalu sedikit untuk
// memunculkannya.
//
// 28 memberi halaman pertama yang penuh dan halaman kedua yang berisi tiga baris, sehingga
// kedua keadaan itu terlihat sekaligus.
func SampleClause() []masterpasalai.Clause {
	return sampledata.Must[[]masterpasalai.Clause](sampleJSON, "SampleClause")
}

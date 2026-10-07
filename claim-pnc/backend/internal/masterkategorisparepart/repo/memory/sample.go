package memory

import (
	_ "embed"

	"claim-pnc/internal/masterkategorisparepart"
	"claim-pnc/internal/platform/sampledata"
)

// sampleJSON memuat seluruh data contoh paket ini; lihat platform/sampledata.
//
//go:embed sample.json
var sampleJSON []byte

// SampleList adalah isi contoh untuk pengembangan tanpa Oracle.
//
// # Yang diwakilinya, dan kenapa persis ini
//
// KETIGA status hadir, supaya seluruh alur layar dapat dicoba tanpa basis data — termasuk
// dua keadaan yang paling mudah terlupa diuji:
//
//   - tab **Waiting Approval** berisi lebih dari satu baris, sehingga keputusan borongan
//     benar-benar teruji sebagai borongan dan bukan sebagai satu baris yang kebetulan
//     berhasil;
//   - tab **Reject** berisi baris yang namanya masih memblokir pemakaian nama itu, sehingga
//     perilaku yang paling mengejutkan pada modul ini — lihat
//     masterkategorisparepart.ErrNameTaken — dapat dicoba langsung.
//
// # ID-nya angka berurut, dan itu bukan kelalaian
//
// Kunci di sini "1".."6", bukan "KAT01".."KAT03" seperti pada
// `mastersparepart/repo/memory/sample.go`. Bentuk angka inilah yang benar:
// `RDB List/InsertMasterSparepartCategory_sql-SQL.xml` menerbitkannya dengan
// `nvl(max(PART_CATEGORY_ID),0)+1`, yang mustahil bekerja atas kunci berbentuk "KAT01".
//
// Contoh pada modul Master Sparepart TIDAK diubah menyesuaikan ini. Modul itu sudah selesai
// dan berada di bawah Isolasi Protektif, dan bentuk kunci contohnya tidak memengaruhi apa
// pun di produksi — ia hanya dipakai saat aplikasi berjalan tanpa Oracle. Selisihnya
// dicatat di docs/keputusan-implementasi.md supaya ia tidak terbaca sebagai kelalaian.
//
// Akibat yang harus disadari saat mencoba tanpa Oracle: kedua modul punya penyimpanan
// memori SENDIRI-SENDIRI, sehingga kategori yang ditambahkan di layar ini TIDAK muncul di
// dropdown layar Master Sparepart. Terhadap Oracle keduanya membaca tabel yang sama dan
// tautannya bekerja. Itu keterbatasan modus memori, bukan cacat modul.
//
// # Namanya mengikuti penggolongan alat berat yang nyata
//
// ENGINE, HYDRAULIC, dan UNDERCARRIAGE adalah tiga yang sudah dipakai contoh Master
// Sparepart, dan ketiganya dipertahankan supaya kedua layar bercerita tentang dunia yang
// sama. Tiga sisanya melengkapi keempat status yang perlu diwakili.
//
// Catatan pada isinya, yang kini tersimpan di sample.json:
//
// Baris ditolak yang namanya TETAP memblokir pemakaian nama itu. Ia yang membuat
// perilaku paling mengejutkan pada modul ini dapat dicoba tanpa menyiapkan data
// sendiri.
func SampleList() []masterkategorisparepart.PartCategory {
	return sampledata.Must[[]masterkategorisparepart.PartCategory](sampleJSON, "SampleList")
}

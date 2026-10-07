package memory

import (
	_ "embed"

	"claim-pnc/internal/mastertipesparepart"
	"claim-pnc/internal/platform/sampledata"
)

// sampleJSON memuat seluruh data contoh paket ini; lihat platform/sampledata.
//
//go:embed sample.json
var sampleJSON []byte

// SampleCategoryList adalah acuan kategori contoh untuk pengembangan tanpa Oracle.
//
// # Keempatnya dianggap SUDAH DISETUJUI
//
// Hanya yang disetujui yang disimpan, meniru `type_category_list` yang menyaring
// APPROVAL = '1' di sisi basis data. Kategori yang menunggu maupun ditolak tidak diwakili
// di sini karena keduanya memang tidak pernah sampai ke dropdown layar ini.
//
// # ID dan namanya SENGAJA sama dengan contoh Master Kategori Sparepart
//
// "1" ENGINE, "2" HYDRAULIC, "3" UNDERCARRIAGE adalah tiga baris disetujui pada
// `masterkategorisparepart/repo/memory/sample.go`, dan ketiganya diulang di sini apa
// adanya. Dengan begitu kedua layar bercerita tentang dunia yang sama saat dicoba tanpa
// Oracle, dan ID yang terlihat di satu layar berarti hal yang sama di layar lain.
//
// Baris keempat — "9" tanpa padanan di modul kategori — sengaja TIDAK ditambahkan. Yang
// ditambahkan justru kebalikannya; lihat baris ber-CategoryID "99" pada SampleList.
//
// # Keterbatasan modus memori yang harus disadari
//
// Kedua modul punya penyimpanan memori SENDIRI-SENDIRI. Kategori yang ditambahkan di layar
// Master Kategori Sparepart TIDAK akan muncul di dropdown layar ini selama aplikasi
// berjalan tanpa Oracle. Terhadap Oracle keduanya membaca tabel yang sama dan tautannya
// bekerja — itu keterbatasan modus memori, bukan cacat modul.
func SampleCategoryList() []mastertipesparepart.Category {
	return sampledata.Must[[]mastertipesparepart.Category](sampleJSON, "SampleCategoryList")
}

// SampleList adalah isi contoh untuk pengembangan tanpa Oracle.
//
// # Yang diwakilinya, dan kenapa persis ini
//
// KETIGA status hadir, supaya seluruh alur layar dapat dicoba tanpa basis data — termasuk
// tiga keadaan yang paling mudah terlupa diuji:
//
//   - tab **Waiting Approval** berisi lebih dari satu baris, sehingga keputusan borongan
//     benar-benar teruji sebagai borongan dan bukan sebagai satu baris yang kebetulan
//     berhasil;
//   - tab **Reject** berisi baris yang namanya masih memblokir pemakaian nama itu, sehingga
//     perilaku yang paling mengejutkan pada modul ini — lihat
//     mastertipesparepart.ErrNameTaken — dapat dicoba langsung;
//   - satu baris **tanpa kategori yang sah**, yang di sistem lama akan HILANG dari layar
//     karena inner join-nya. Lihat baris ber-CategoryID "99" di bawah.
//
// # Dua nama tipe yang sama di kategori berbeda TIDAK diwakili, dan itu disengaja
//
// Keadaan seperti itu tidak dapat dibuat lewat layar: `ValidationSparepartType` menolak
// nama ganda di SELURUH tabel, bukan per kategori (`P-5`, keputusan Work Owner
// 2026-09-21). Menaruhnya di contoh akan menggambarkan keadaan yang mustahil dicapai
// pengguna, dan membuat pembaca menyangka keunikannya per kategori.
//
// # ID-nya angka berurut
//
// Kunci di sini "1".."6", meniru `nvl(max(PART_SECTION_ID),0)+1` pada
// `RDB List/InsertMasterSparepartType_sql-SQL.xml` — yang mustahil bekerja atas kunci
// berbentuk teks seperti "TIP01".
//
// # Namanya mengikuti suku cadang alat berat yang nyata
//
// Setiap tipe ditaruh di bawah kategori yang masuk akal baginya, supaya kolom Kategori pada
// grid bercerita alih-alih menampilkan pasangan acak.
//
// Catatan pada isinya, yang kini tersimpan di sample.json:
//
// Baris ditolak yang namanya TETAP memblokir pemakaian nama itu. Ia yang membuat
// perilaku paling mengejutkan pada modul ini dapat dicoba tanpa menyiapkan data
// sendiri: mencoba menambah tipe bernama "CONTROL VALVE" akan ditolak, dan barisnya
// tidak terlihat dari tab Approve maupun Waiting Approval.
// Baris YATIM: kategorinya "99" tidak ada di SampleCategoryList.
//
// Di sistem lama baris seperti ini HILANG dari layar — inner join-nya membuangnya —
// sehingga ia tidak dapat dilihat maupun diperbaiki siapa pun. Di modul ini ia
// tetap muncul dengan kolom Kategori kosong, dan hanya dapat disimpan ulang setelah
// petugas memilih kategori yang sah.
//
// Ia ada di contoh supaya selisih perilaku itu dapat DILIHAT saat mencoba tanpa
// Oracle, bukan hanya dibaca di komentar berkas .sql.
func SampleList() []mastertipesparepart.PartType {
	return sampledata.Must[[]mastertipesparepart.PartType](sampleJSON, "SampleList")
}

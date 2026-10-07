package memory

import (
	_ "embed"

	"claim-pnc/internal/masterreas"
	"claim-pnc/internal/platform/sampledata"
)

// sampleJSON memuat seluruh data contoh paket ini; lihat platform/sampledata.
//
//go:embed sample.json
var sampleJSON []byte

// SampleList adalah isi contoh untuk pengembangan tanpa Oracle.
//
// # Seluruh nama perusahaan, kode, login, dan surel di sini KARANGAN
//
// Bukan mitra reasuransi nyata, bukan kode nyata, dan bukan alamat nyata. Surelnya memakai
// domain `contoh.invalid` — ranah tingkat atas yang RFC 2606 cadangkan supaya tidak pernah
// dapat diselesaikan DNS, sehingga contoh yang tidak sengaja terkirim tidak akan sampai ke
// siapa pun.
//
// Aturannya sendiri berasal dari `D-69`: alamat surel **selalu disamarkan** di artefak yang
// di-commit. Di sini bahkan tidak ada yang disamarkan — tidak satu pun alamat nyata pernah
// masuk ke berkas ini.
//
// # Yang diwakilinya, dan kenapa persis ini
//
// Tujuh baris, dipilih supaya setiap keadaan yang menentukan perilaku layar dapat dicoba
// tanpa menyiapkan data sendiri:
//
//   - **Satu perusahaan dengan TIGA baris TYPE berbeda** (`RE-001`). Inilah bentuk yang
//     paling mudah disalahpahami di layar ini: satu nama muncul tiga kali, dengan surel yang
//     berbeda-beda, dan yang membedakannya hanya kolom Tipe. Tanpa contoh seperti ini,
//     kolom Tipe tampak seperti keterangan yang tidak berguna.
//   - **Satu perusahaan TANPA baris cadangan** (`RE-004`, hanya TYPE `2`). Ia keadaan yang
//     dicacah `reas_count_without_fallback`: dokumen berjenis lain tidak akan menemukan
//     surel tujuannya sama sekali.
//   - **Dua baris berbagi LOGIN yang sama** (`RE-002` dan `RE-005`). Keadaan yang sistem
//     lama sendiri akui mungkin terjadi — `GetPNCList_PLA1` memilih sembarang satu dengan
//     `order by reinsurerid desc fetch next 1 row only` — dan yang dicacah
//     `reas_count_shared_login`. Ia dibiarkan ada di contoh supaya layar diuji dengan data
//     yang bentuknya seperti produksi, bukan yang sudah dirapikan.
//   - **Satu baris COUNTRY kosong** (`RE-003`). Ia pasti terjadi pada baris yang lahir dari
//     `GetListDataLoginReas`, yang menyisipkan tanpa kolom COUNTRY dan TYPE sama sekali.
//
// # TYPE-nya satu karakter, dan `1` berarti cadangan
//
// Lihat masterreas.FallbackType. Nilai selain `1` di sini — `2` dan `3` — dipilih sekadar
// sebagai karakter yang berbeda; **artinya dalam bahasa bisnis tidak diketahui** (`R-16`),
// dan contoh ini tidak berpura-pura mengetahuinya.
//
// Catatan pada isinya, yang kini tersimpan di sample.json:
//
// Satu perusahaan, tiga jenis dokumen, tiga surel berbeda.
// Berbagi LOGIN dengan RE-005; lihat catatan di atas.
// COUNTRY kosong — bentuk baris yang lahir dari GetListDataLoginReas.
// Tanpa baris cadangan: hanya TYPE "2".
// Berbagi LOGIN dengan RE-002.
func SampleList() []masterreas.Member {
	return sampledata.Must[[]masterreas.Member](sampleJSON, "SampleList")
}

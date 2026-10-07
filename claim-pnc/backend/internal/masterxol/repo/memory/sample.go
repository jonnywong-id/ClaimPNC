package memory

import (
	_ "embed"

	"claim-pnc/internal/masterxol"
	"claim-pnc/internal/platform/sampledata"
)

// sampleJSON memuat seluruh data contoh paket ini; lihat platform/sampledata.
//
//go:embed sample.json
var sampleJSON []byte

// SampleMaster adalah contoh isi Master XOL yang MENIRU BENTUK data produksi.
//
// # Kenapa bentuknya ditiru, bukan dikarang rapi
//
// Penyimpanan memori dipakai mengembangkan dan menguji layar. Bila contohnya rapi
// sementara produksinya tidak, layar akan tampak benar saat dikembangkan lalu pecah saat
// menyentuh data sebenarnya. Keempat keanehan berikut karena itu SENGAJA ikut, dan
// keempatnya nyata — diperiksa langsung ke portal ASM pada 2026-09-20:
//
//  1. **Nomor induk berlubang.** Produksi memuat 10001, 10002, lalu 10004 — 10003 pernah
//     dihapus. Layar tidak boleh mengandaikan nomornya berurutan.
//  2. **TYPEXOL kosong.** Dua dari delapan induk menyimpan NULL. Dropdown-nya harus dapat
//     menampilkan keadaan "belum dipilih" tanpa memaksanya menjadi salah satu kode.
//  3. **Lapisan tanpa isi.** Lapisan `10017` milik induk `10008` bernama kosong, limit
//     dan excess-nya NULL. Grid harus tetap terbaca.
//  4. **Total share bukan 100%.** Lapisan `10004` milik induk `10002` tidak punya satu
//     pun reasuradur, sehingga totalnya 0 — dan ia tetap tersimpan. Inilah bukti bahwa
//     aturan 100% adalah peringatan, bukan penolakan (lihat masterxol.ShareWarning).
//
// # Yang TIDAK disalin
//
// Tidak ada data nasabah di tabel ini — isinya struktur treaty dan nama perusahaan
// reasuransi, bukan nomor polis maupun nama tertanggung (`D-69`). Nama induk pun berupa
// label teknis seperti "Section 1".
//
// Catatan pada isinya, yang kini tersimpan di sample.json:
//
// TYPEXOL kosong, dan lapisannya tanpa satu pun reasuradur — keanehan 2 dan 4.
// Nomor 10003 sengaja dilewati — keanehan 1.
// Lapisan tanpa nama, limit, dan excess — keanehan 3.
func SampleMaster() []masterxol.Master {
	return sampledata.Must[[]masterxol.Master](sampleJSON, "SampleMaster")
}

// SampleYear meniru isi POOLDATA.M_TREATYYEAR yang dipakai dropdown Tahun.
//
// Produksi memuat 36 tahun, 1995 sampai 2030. Contoh ini memuat rentang yang jauh lebih
// pendek tetapi mencakup seluruh tahun yang benar-benar dipakai SampleMaster, supaya
// dropdown di layar pengembangan tidak pernah menampilkan tahun yang tidak ada
// pilihannya.
func SampleYear() []string { return sampledata.Must[[]string](sampleJSON, "SampleYear") }

// sampleBusinessGroup meniru pilihan grup bisnis beserta nama grup treaty induknya.
//
// Kolom treatyTag adalah POOLDATA.PROPORTIONALARRG.TREATYGROUPNAME yang menentukan sebuah
// grup muncul pada Type XOL yang mana. Nilainya disalin dari hasil penelusuran ke portal
// ASM pada 2026-09-20, termasuk akibatnya yang janggal:
//
//   - Type 2 (`%PA%` / `%GA%`) hanya mengembalikan **AVIATION HULL**, dan sebabnya
//     kebetulan belaka: grup treaty induknya bernama "AVIATION & AEROSPACE", dan kata
//     AERO**SPA**CE memuat potongan "PA". PA dan GA (OTHERS) — dua grup yang justru
//     dimaksud penyaring ini — tidak muncul, karena keduanya bernaung di bawah
//     "GENERAL ACCIDENT" yang tidak memuat "PA" maupun "GA" secara berurutan.
//   - HEAVY EQUIPMENT muncul pada Type **1**, bukan Type 3, karena grup treaty induknya
//     ENGINEERING.
//
// Keduanya ditiru apa adanya (keputusan Work Owner 2026-09-20). Contoh ini dibuat supaya
// uji dapat membuktikan kejanggalan itu memang direproduksi, bukan diam-diam diperbaiki.
func sampleBusinessGroup() []businessGroupRow {
	return []businessGroupRow{
		{masterxol.Business{ID: "10004", Name: "MOTOR VEHICLE"}, "MOTOR VEHICLE"},
		{masterxol.Business{ID: "10012", Name: "MOTOR CYCLE"}, "MOTOR VEHICLE"},
		{masterxol.Business{ID: "10013", Name: "FIRE"}, "PROPERTY"},
		{masterxol.Business{ID: "10030", Name: "ASURANSI SIMAS SATELIT"}, "PROPERTY"},
		{masterxol.Business{ID: "10009", Name: "ENGINEERING"}, "ENGINEERING"},
		{masterxol.Business{ID: "10014", Name: "HEAVY EQUIPMENT"}, "ENGINEERING"},
		{masterxol.Business{ID: "10033", Name: "AVIATION HULL"}, "AVIATION & AEROSPACE"},
		{masterxol.Business{ID: "10002", Name: "MARINE CARGO"}, "MARINE CARGO"},
		{masterxol.Business{ID: "10003", Name: "MARINE HULL"}, "MARINE HULL"},
		{masterxol.Business{ID: "10006", Name: "PA"}, "GENERAL ACCIDENT"},
		{masterxol.Business{ID: "10005", Name: "GA (OTHERS)"}, "GENERAL ACCIDENT"},
		{masterxol.Business{ID: "10007", Name: "HEALTH"}, "HOSPITAL"},
		{masterxol.Business{ID: "10015", Name: "SURETY BOND"}, "SURETY BOND"},
	}
}

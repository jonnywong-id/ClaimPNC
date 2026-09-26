package memory

import (
	"time"

	"claim-pnc/internal/inboxreceivetka"
)

// SampleTasks mengembalikan daftar contoh untuk pengembangan tanpa Oracle.
//
// # Seluruh isinya KARANGAN, dan itu wajib
//
// Tidak ada satu pun nomor polis, nama tertanggung, nama peserta, maupun nomor klaim nyata
// di sini. `D-69` melarang data nasabah ditulis ke berkas yang di-commit, dan larangan itu
// berlaku pada data contoh persis seperti pada dokumen.
//
// **Bentuknya** ditiru dari data nyata yang diperiksa pada 2026-09-24 — nomor kasus
// `PNC-xxxx` empat digit, nomor polis 14 digit, dan tanggal registrasi yang berjarak
// tahunan dari hari ini — supaya layar diuji menghadapi rupa yang benar-benar akan
// ditemuinya. **Isinya** tidak.
//
// # Yang sengaja diwakili
//
// Keenam baris di bawah dipilih supaya setiap keadaan yang dapat dihadapi layar muncul
// sekurang-kurangnya sekali:
//
//	baris 1   lengkap, menunggu paling lama            -> tampil paling atas
//	baris 2   lengkap, menunggu beberapa bulan
//	baris 3   TANPA Nama Peserta                       -> polisnya tidak ditemukan
//	baris 4   TANPA Date Of Loss                       -> kolom tanggal yang dapat kosong
//	baris 5   TANPA tanggal registrasi                 -> jatuh ke AKHIR daftar
//	baris 6   ClaimKey KOSONG                          -> klaim yatim; Submit DITOLAK
//
// Dua baris terakhir yang paling perlu ada.
//
// Baris 5 membuktikan baris bertanggal kosong jatuh di akhir, sejajar dengan NULL pada
// `ORDER BY ... ASC` — dan bahwa kolom Aging-nya ditulis sebagai tanda hubung, bukan
// "0 hari" yang akan terbaca seolah klaimnya baru masuk hari ini.
//
// Baris 6 membuktikan jalur yang paling mudah terlewat: gabungan ke `T_CLAIM_PNC` sengaja
// LEFT, sehingga pekerjaan yang klaimnya tidak ditemukan TETAP TAMPIL — persis seperti di
// Pega — tetapi Submit atasnya ditolak dengan ErrClaimMissing.
//
// # Urutan penulisannya sengaja TIDAK terurut
//
// Baris-barisnya ditulis acak supaya pengurutan benar-benar diuji. Bila daftar contoh sudah
// terurut sejak awal, fungsi pengurutan yang rusak pun akan tampak benar.
//
// # Seluruhnya SUDAH merupakan klaim TKA yang belum selesai
//
// Tidak ada baris yang tanggal kelengkapan dokumennya terisi dan tidak ada yang berstatus
// selesai, karena kedua penyaring itu hidup di dalam kueri SQL — bukan di dalam Filter.
func SampleTasks() []inboxreceivetka.Task {
	return []inboxreceivetka.Task{
		{
			Reference:       "ASM-FW-GCNMFW-WORK PNC-1729",
			ClaimKey:        "ASM-FW-GCNMFW-WORK PNC-1729",
			ClaimNumber:     "PNC-1729",
			PolicyNumber:    "12200007000001",
			InsuredName:     "PT Contoh Karya Mandiri",
			ParticipantName: "PT Contoh Karya Mandiri",
			DateOfLoss:      at("2024-03-19T00:00:00Z"),
			RegisteredOn:    at("2024-03-19T00:00:00Z"),
		},
		{
			Reference:       "ASM-FW-GCNMFW-WORK PNC-1546",
			ClaimKey:        "ASM-FW-GCNMFW-WORK PNC-1546",
			ClaimNumber:     "PNC-1546",
			PolicyNumber:    "12000000000002",
			InsuredName:     "PT Contoh Sejahtera Abadi",
			ParticipantName: "Peserta Contoh Satu",
			DateOfLoss:      at("2021-08-16T00:00:00Z"),
			RegisteredOn:    at("2023-05-10T00:00:00Z"),
		},
		{
			// Polisnya tidak ditemukan di tabel polis, sehingga nama pesertanya kosong.
			// Barisnya tetap dapat dikerjakan — yang hilang hanya satu sel.
			Reference:       "ASM-FW-GCNMFW-WORK PNC-1811",
			ClaimKey:        "ASM-FW-GCNMFW-WORK PNC-1811",
			ClaimNumber:     "PNC-1811",
			PolicyNumber:    "12200007000003",
			InsuredName:     "PT Contoh Bahari Nusantara",
			ParticipantName: "",
			DateOfLoss:      at("2025-02-11T00:00:00Z"),
			RegisteredOn:    at("2025-02-20T00:00:00Z"),
		},
		{
			// Date Of Loss kosong.
			Reference:       "ASM-FW-GCNMFW-WORK PNC-1902",
			ClaimKey:        "ASM-FW-GCNMFW-WORK PNC-1902",
			ClaimNumber:     "PNC-1902",
			PolicyNumber:    "12200007000004",
			InsuredName:     "PT Contoh Rekayasa Utama",
			ParticipantName: "Peserta Contoh Dua",
			DateOfLoss:      nil,
			RegisteredOn:    at("2026-06-01T00:00:00Z"),
		},
		{
			// Tanggal registrasi kosong — kolom sumbernya VARCHAR2, dan bentuk yang tidak
			// dikenali diurai menjadi nil alih-alih tanggal karangan.
			Reference:       "ASM-FW-GCNMFW-WORK PNC-1955",
			ClaimKey:        "ASM-FW-GCNMFW-WORK PNC-1955",
			ClaimNumber:     "PNC-1955",
			PolicyNumber:    "12200007000005",
			InsuredName:     "PT Contoh Adikarya Persada",
			ParticipantName: "Peserta Contoh Tiga",
			DateOfLoss:      at("2026-07-04T00:00:00Z"),
			RegisteredOn:    nil,
		},
		{
			// Klaim YATIM: pekerjaannya ada di tabel kerja Pega, klaimnya tidak ada di
			// tabel bisnis. Ia TAMPIL, dan Submit atasnya ditolak.
			Reference:       "ASM-FW-GCNMFW-WORK PNC-1977",
			ClaimKey:        "",
			ClaimNumber:     "PNC-1977",
			PolicyNumber:    "12200007000006",
			InsuredName:     "PT Contoh Lintas Benua",
			ParticipantName: "",
			DateOfLoss:      at("2026-08-30T00:00:00Z"),
			RegisteredOn:    at("2026-09-01T00:00:00Z"),
		},
	}
}

// at mengurai waktu contoh, dan panik bila penulisannya salah.
//
// Panik di sini aman: nilainya konstanta di dalam berkas ini, bukan masukan pengguna,
// sehingga kesalahan penulisannya adalah cacat yang harus terlihat pada uji pertama.
func at(value string) *time.Time {
	moment, err := time.Parse(time.RFC3339, value)
	if err != nil {
		panic("inboxreceivetka/memory: waktu contoh salah tulis: " + value)
	}
	return &moment
}

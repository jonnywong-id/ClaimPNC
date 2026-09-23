package memory

import (
	"time"

	"claim-pnc/internal/inboxanalystdoctor"
)

// SampleOperator adalah login petugas pemilik antrean contoh.
//
// # Kenapa BUKAN `DRRATNA`
//
// Karena `DRRATNA` adalah Operator ID sungguhan yang tertanam di
// `Flow/Register_Flow.xml` (`Assignment13`, `pyOperator`), dan ia salah satu dari 24
// hardcode yang `D-15` perintahkan dihapus. Menuliskannya sebagai data contoh akan
// memindahkan hardcode itu ke sistem baru lewat pintu belakang — tepat hal yang sedang
// dihapus.
//
// `D-69` memang mengizinkan nama Operator ID ditulis di dokumen supaya tiket dapat menunjuk
// hardcode mana yang dibuang. Izin itu untuk DOKUMEN, bukan untuk data yang dijalankan.
const SampleOperator = "ADMINKLAIM"

// SampleOtherOperator adalah petugas LAIN, dipakai membuktikan batas kewenangan bekerja.
//
// Tanpa baris milik orang lain di dalam contoh, penyaring `pxAssignedOperatorID` yang hilang
// tidak akan terlihat sama sekali: seluruh baris tetap muncul, dan semuanya kebetulan benar.
const SampleOtherOperator = "ADMINLAIN"

// at membentuk waktu UTC, supaya contoh terbaca dan urutannya mudah diperiksa dengan mata.
func at(year int, month time.Month, date, hour int) time.Time {
	return time.Date(year, month, date, hour, 0, 0, 0, time.UTC)
}

// SampleTasks adalah antrean contoh untuk pengembangan lokal dan pengujian.
//
// # Kenapa nomor polis dan nama tertanggungnya karangan
//
// Karena data nasabah TIDAK PERNAH ditulis ke berkas yang di-commit (`D-69`). Nomor polis,
// nama tertanggung, dan nomor klaim di sini seluruhnya karangan yang bentuknya saja
// menyerupai aslinya. Pada layar ini aturan itu lebih mengikat daripada di modul lain:
// antreannya berisi klaim Personal Accident, dan `FR-R2` memperlakukan data medis secara
// khusus.
//
// # Apa yang sengaja dibuat pada contohnya
//
// Ia tidak sekadar "beberapa baris": setiap penyaring memperoleh baris yang membuatnya dapat
// dibuktikan bekerja, dan setiap baris seperti itu diberi keterangan di tempatnya.
//
//   - Satu baris ber-`TransferFlag "1"` — milik antrean COMPLIANCE. Bila penyaring penanda
//     antrean hilang, baris itu bocor ke layar ini.
//   - Satu baris milik operator LAIN. Bila batas kewenangan hilang, ia ikut tampil.
//   - Satu baris ber-`Resolved-Completed`. Bila penyaring status terbalik menjadi `=`,
//     hanya baris itu yang tersisa.
//   - Satu baris ber-`Resolved-Rejected` yang HARUS TETAP MUNCUL — Report Definition hanya
//     mengecualikan `Resolved-Completed`, dan menyamakan keduanya adalah kesalahan yang
//     paling mudah dibuat di modul ini.
//   - Dua baris berwaktu daftar SAMA PERSIS, sehingga pemutus seri `pzInsKey` menurun dapat
//     diperiksa; tanpanya urutan keduanya tidak tetap.
//   - Satu baris dengan komentar PIC Teknis terisi, sehingga kolom yang di Oracle masih
//     kosong tetap terlihat bentuknya saat dikembangkan.
var SampleTasks = []Record{
	{
		TransferFlag: inboxanalystdoctor.TransferAnalystDoctor,
		Task: inboxanalystdoctor.AnalystDoctorTask{
			ClaimID:          "ASM-FW-GCNMFW-WORK PNCN.26.0311",
			ClaimNumber:      "PNCN.26.0311",
			PolicyNumber:     "26.002.2026.00311",
			InsuredName:      "Bayu Pratama Contoh",
			BranchName:       "JAKARTA PUSAT",
			AdminName:        "ADMINREG1",
			TechnicalPIC:     "PICTEKNIKPA1",
			TechnicalPICNote: "Mohon dinilai kelayakan biaya rawat inap hari ke-4 sampai ke-9.",
			RegisteredAt:     at(2026, time.September, 18, 9),
			ProcessStatus:    "Open",
			AssignedOperator: SampleOperator,
		},
	},
	{
		TransferFlag: inboxanalystdoctor.TransferAnalystDoctor,
		Task: inboxanalystdoctor.AnalystDoctorTask{
			ClaimID:          "ASM-FW-GCNMFW-WORK PNCN.26.0298",
			ClaimNumber:      "PNCN.26.0298",
			PolicyNumber:     "26.002.2026.00298",
			InsuredName:      "Sari Melati Contoh",
			BranchName:       "SURABAYA",
			AdminName:        "ADMINREG2",
			TechnicalPIC:     "PICTEKNIKPA1",
			TechnicalPICNote: "",
			RegisteredAt:     at(2026, time.September, 15, 14),
			ProcessStatus:    "Pending-AnalystDoctor",
			AssignedOperator: SampleOperator,
		},
	},
	{
		// Berwaktu daftar SAMA PERSIS dengan baris di bawahnya. Keduanya ada supaya pemutus
		// seri `pzInsKey` menurun dapat diperiksa — yang ber-`…0290` harus mendahului
		// yang ber-`…0289`.
		TransferFlag: inboxanalystdoctor.TransferAnalystDoctor,
		Task: inboxanalystdoctor.AnalystDoctorTask{
			ClaimID:          "ASM-FW-GCNMFW-WORK PNCN.26.0290",
			ClaimNumber:      "PNCN.26.0290",
			PolicyNumber:     "26.002.2026.00290",
			InsuredName:      "Rahmat Hidayat Contoh",
			BranchName:       "BANDUNG",
			AdminName:        "ADMINREG1",
			TechnicalPIC:     "PICTEKNIKPA2",
			RegisteredAt:     at(2026, time.September, 12, 8),
			ProcessStatus:    "Open",
			AssignedOperator: SampleOperator,
		},
	},
	{
		TransferFlag: inboxanalystdoctor.TransferAnalystDoctor,
		Task: inboxanalystdoctor.AnalystDoctorTask{
			ClaimID:          "ASM-FW-GCNMFW-WORK PNCN.26.0289",
			ClaimNumber:      "PNCN.26.0289",
			PolicyNumber:     "26.005.2026.00289",
			InsuredName:      "Dewi Anggraini Contoh",
			BranchName:       "MEDAN",
			AdminName:        "ADMINREG3",
			TechnicalPIC:     "PICTEKNIKTRV",
			RegisteredAt:     at(2026, time.September, 12, 8),
			ProcessStatus:    "Open",
			AssignedOperator: SampleOperator,
		},
	},
	{
		// HARUS TETAP MUNCUL. Report Definition mengecualikan `Resolved-Completed` saja,
		// sehingga klaim yang DITOLAK tetap berada di antrean ini.
		TransferFlag: inboxanalystdoctor.TransferAnalystDoctor,
		Task: inboxanalystdoctor.AnalystDoctorTask{
			ClaimID:          "ASM-FW-GCNMFW-WORK PNCN.26.0255",
			ClaimNumber:      "PNCN.26.0255",
			PolicyNumber:     "26.002.2026.00255",
			InsuredName:      "Joko Susilo Contoh",
			BranchName:       "SEMARANG",
			AdminName:        "ADMINREG2",
			TechnicalPIC:     "PICTEKNIKPA2",
			RegisteredAt:     at(2026, time.September, 5, 11),
			ProcessStatus:    "Resolved-Rejected",
			AssignedOperator: SampleOperator,
		},
	},
	{
		// TIDAK BOLEH MUNCUL — tugasnya sudah tuntas.
		TransferFlag: inboxanalystdoctor.TransferAnalystDoctor,
		Task: inboxanalystdoctor.AnalystDoctorTask{
			ClaimID:          "ASM-FW-GCNMFW-WORK PNCN.26.0240",
			ClaimNumber:      "PNCN.26.0240",
			PolicyNumber:     "26.002.2026.00240",
			InsuredName:      "Lestari Wulandari Contoh",
			BranchName:       "JAKARTA PUSAT",
			AdminName:        "ADMINREG1",
			TechnicalPIC:     "PICTEKNIKPA1",
			RegisteredAt:     at(2026, time.August, 30, 10),
			ProcessStatus:    inboxanalystdoctor.StatusKerjaSelesai,
			AssignedOperator: SampleOperator,
		},
	},
	{
		// TIDAK BOLEH MUNCUL — milik antrean Compliance, bukan antrean ini.
		TransferFlag: inboxanalystdoctor.TransferCompliance,
		Task: inboxanalystdoctor.AnalystDoctorTask{
			ClaimID:          "ASM-FW-GCNMFW-WORK PNCN.26.0231",
			ClaimNumber:      "PNCN.26.0231",
			PolicyNumber:     "26.003.2026.00231",
			InsuredName:      "PT Sumber Contoh",
			BranchName:       "JAKARTA PUSAT",
			AdminName:        "ADMINREG3",
			TechnicalPIC:     "PICTEKNIKANK",
			RegisteredAt:     at(2026, time.September, 20, 9),
			ProcessStatus:    "Open",
			AssignedOperator: SampleOperator,
		},
	},
	{
		// TIDAK BOLEH MUNCUL bagi SampleOperator — antrean ini milik satu orang.
		TransferFlag: inboxanalystdoctor.TransferAnalystDoctor,
		Task: inboxanalystdoctor.AnalystDoctorTask{
			ClaimID:          "ASM-FW-GCNMFW-WORK PNCN.26.0222",
			ClaimNumber:      "PNCN.26.0222",
			PolicyNumber:     "26.002.2026.00222",
			InsuredName:      "Hendra Kusuma Contoh",
			BranchName:       "DENPASAR",
			AdminName:        "ADMINREG2",
			TechnicalPIC:     "PICTEKNIKPA2",
			RegisteredAt:     at(2026, time.September, 19, 16),
			ProcessStatus:    "Open",
			AssignedOperator: SampleOtherOperator,
		},
	},
}

// NewSampleStore membentuk pembaca berisi antrean contoh.
//
// Salinan dibuat supaya dua portal yang sama-sama memakai penyimpanan memori tidak berbagi
// senarai yang sama. Modul ini memang tidak menulis, sehingga hari ini tidak ada yang dapat
// saling menimpa — tetapi berbagi senarai antarportal adalah bentuk kebocoran yang persis
// dilarang `R-20`, dan mencegahnya sejak awal jauh lebih murah daripada menemukannya kelak.
func NewSampleStore() *Store {
	records := make([]Record, len(SampleTasks))
	copy(records, SampleTasks)
	return NewStore(records)
}

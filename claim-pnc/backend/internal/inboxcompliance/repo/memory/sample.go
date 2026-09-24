package memory

import (
	"time"

	"claim-pnc/internal/inboxcompliance"
)

// NewSampleStore membentuk antrean contoh untuk menjalankan aplikasi tanpa basis data.
//
// # Aturan isi data contoh
//
// Tidak ada satu pun data nasabah nyata di sini. Nomor polis, nama tertanggung, dan nomor
// klaim seluruhnya dikarang, dan itu keharusan: `D-69` melarang nilai seperti itu ditulis
// di berkas yang di-commit.
//
// Nama admin memakai bentuk yang jelas-jelas contoh, bukan Operator ID nyata dari export.
// `D-69` memang mengizinkan Operator ID ditulis lengkap di DOKUMEN — supaya tiket dapat
// menunjuk hardcode mana yang dihapus — tetapi menyalinnya ke data contoh yang tampil di
// layar pengembangan adalah hal yang berbeda, dan tidak ada gunanya.
//
// # Kenapa tanggalnya relatif terhadap sekarang
//
// Supaya kolom Aging benar-benar terlihat bekerja saat aplikasi dijalankan tanpa basis
// data. Tanggal tetap akan membuat Aging tumbuh tak terbatas seiring waktu, dan setelah
// beberapa bulan layar contohnya menampilkan "412 days ago" pada setiap baris — yang
// membuat kolom itu tampak rusak.
func NewSampleStore() *Store {
	now := time.Now().UTC()

	// Selisih dipilih supaya ketiga cabang pemformatan Aging terlihat sekaligus: di bawah
	// 24 jam, di atas 24 jam, dan melewati akhir pekan.
	return NewStore(
		Row{
			Workbasket: inboxcompliance.WorkbasketCompliance,
			CreatedAt:  now.Add(-3 * time.Hour),
			Item: inboxcompliance.WorkItem{
				CaseID:             "PNC-900101",
				Reference:          "ASM-FW-GCNMFW-WORK PNC-900101",
				PolicyNumber:       "00.000.0000.00001",
				InsuredName:        "Tertanggung Contoh Satu",
				BusinessName:       "Property All Risk",
				BranchName:         "Jakarta Pusat",
				AdminName:          "ADMINCONTOH1",
				ComplianceSentDate: timePtr(now.Add(-3 * time.Hour)),
			},
		},
		Row{
			Workbasket: inboxcompliance.WorkbasketCompliance,
			CreatedAt:  now.Add(-50 * time.Hour),
			Item: inboxcompliance.WorkItem{
				CaseID:             "PNC-900102",
				Reference:          "ASM-FW-GCNMFW-WORK PNC-900102",
				PolicyNumber:       "00.000.0000.00002",
				InsuredName:        "Tertanggung Contoh Dua",
				BusinessName:       "Marine Cargo",
				BranchName:         "Surabaya",
				AdminName:          "ADMINCONTOH2",
				ComplianceSentDate: timePtr(now.Add(-50 * time.Hour)),
			},
		},
		Row{
			Workbasket: inboxcompliance.WorkbasketCompliance,
			CreatedAt:  now.Add(-14 * 24 * time.Hour),
			Item: inboxcompliance.WorkItem{
				CaseID:             "PNC-900103",
				Reference:          "ASM-FW-GCNMFW-WORK PNC-900103",
				PolicyNumber:       "00.000.0000.00003",
				InsuredName:        "Tertanggung Contoh Tiga",
				BusinessName:       "Personal Accident",
				BranchName:         "Medan",
				AdminName:          "ADMINCONTOH3",
				ComplianceSentDate: timePtr(now.Add(-14 * 24 * time.Hour)),
			},
		},

		// Baris tanpa Tanggal Kirim Compliance. Ia ADA di data nyata — tabel datar diisi
		// prosedur konversi yang berjalan terpisah, sehingga klaim yang baru masuk antrean
		// dapat belum punya barisnya. Kolom Aging-nya harus tampil KOSONG, bukan
		// "0 hours ago" (`P-5` butir 13).
		Row{
			Workbasket: inboxcompliance.WorkbasketCompliance,
			CreatedAt:  now.Add(-90 * time.Minute),
			Item: inboxcompliance.WorkItem{
				CaseID:       "PNC-900104",
				Reference:    "ASM-FW-GCNMFW-WORK PNC-900104",
				PolicyNumber: "00.000.0000.00004",
				InsuredName:  "Tertanggung Contoh Empat",
				BusinessName: "Contractor All Risk",
				BranchName:   "Bandung",
				AdminName:    "ADMINCONTOH1",
			},
		},

		// Klaim yang sudah selesai. Ia TIDAK boleh muncul di antrean, dan ia ada di sini
		// supaya penyaring yang lupa dipasang langsung terlihat saat pengembangan lokal.
		Row{
			Workbasket: inboxcompliance.WorkbasketCompliance,
			CreatedAt:  now.Add(-20 * 24 * time.Hour),
			Resolved:   true,
			Item: inboxcompliance.WorkItem{
				CaseID:             "PNC-900105",
				Reference:          "ASM-FW-GCNMFW-WORK PNC-900105",
				PolicyNumber:       "00.000.0000.00005",
				InsuredName:        "Tertanggung Contoh Lima",
				BusinessName:       "Travel",
				BranchName:         "Denpasar",
				AdminName:          "ADMINCONTOH2",
				ComplianceSentDate: timePtr(now.Add(-20 * 24 * time.Hour)),
			},
		},

		// Baris di antrean LAIN. Ia juga tidak boleh muncul, dan alasannya berbeda dari
		// baris di atas — yang satu tersaring status, yang ini tersaring nama antrean.
		Row{
			Workbasket: "RCLPUCL",
			CreatedAt:  now.Add(-time.Hour),
			Item: inboxcompliance.WorkItem{
				CaseID:             "PNC-900106",
				Reference:          "ASM-FW-GCNMFW-WORK PNC-900106",
				PolicyNumber:       "00.000.0000.00006",
				InsuredName:        "Tertanggung Contoh Enam",
				BusinessName:       "Fire",
				BranchName:         "Semarang",
				AdminName:          "ADMINCONTOH3",
				ComplianceSentDate: timePtr(now.Add(-time.Hour)),
			},
		},

		// ── Tab Post Audit ──────────────────────────────────────────────────────────
		//
		// Barisnya TIDAK punya workbasket dan TIDAK punya status, karena
		// `POOLDATA.T_CLAIM_COMPLIANCE_H` memang tidak menyimpan keduanya. Ia juga tidak
		// punya Tanggal Kirim Compliance, sehingga kolom Aging tab ini selalu kosong —
		// dan itu benar, karena tab ini memang tidak menggambar kolom Aging.
		Row{
			Tab: inboxcompliance.TabPostAudit,
			Item: inboxcompliance.WorkItem{
				CaseID:            "CPL-19",
				ClaimNumber:       "ASM-FW-GCNMFW-WORK PNC-900201",
				Reference:         "ASM-FW-GCNMFW-WORK PNC-900201",
				PolicyNumber:      "00.000.0000.00011",
				InsuredName:       "Tertanggung Contoh Sebelas",
				PostAuditSentDate: timePtr(now.Add(-2 * 24 * time.Hour)),
				ComplianceRemarks: "Dokumen pendukung sudah lengkap, diteruskan ke Post Audit.",
			},
		},
		Row{
			Tab: inboxcompliance.TabPostAudit,
			Item: inboxcompliance.WorkItem{
				CaseID:            "CPL-17",
				ClaimNumber:       "ASM-FW-GCNMFW-WORK PNC-900202",
				Reference:         "ASM-FW-GCNMFW-WORK PNC-900202",
				PolicyNumber:      "00.000.0000.00012",
				InsuredName:       "Tertanggung Contoh Dua Belas",
				PostAuditSentDate: timePtr(now.Add(-9 * 24 * time.Hour)),
				ComplianceRemarks: "Menunggu konfirmasi cabang atas nilai kerugian.",
			},
		},

		// Baris tanpa Tanggal Kirim Post Audit. SELURUH kolom tabel itu boleh NULL — tidak
		// ada satu pun `NOT NULL` di DDL-nya — sehingga keadaan ini memang mungkin, dan
		// urutannya harus menempatkannya paling belakang (`NULLS LAST`).
		Row{
			Tab: inboxcompliance.TabPostAudit,
			Item: inboxcompliance.WorkItem{
				CaseID:       "CPL-15",
				ClaimNumber:  "ASM-FW-GCNMFW-WORK PNC-900203",
				Reference:    "ASM-FW-GCNMFW-WORK PNC-900203",
				PolicyNumber: "00.000.0000.00013",
				InsuredName:  "Tertanggung Contoh Tiga Belas",
			},
		},
	)
}

// timePtr menyalin sebuah waktu menjadi pointer.
func timePtr(at time.Time) *time.Time { return &at }

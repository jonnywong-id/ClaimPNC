package memory

import (
	"time"

	"claim-pnc/internal/pelaporanklaim"
)

// SampleReports mengembalikan laporan contoh untuk pengembangan tanpa basis data.
//
// # Seluruh isinya KARANGAN
//
// Nama orang, nomor polis, alamat surel, dan nomor telepon di bawah tidak merujuk siapa
// pun. `D-69` melarang data nasabah ditulis di berkas yang di-commit, dan larangan itu
// berlaku untuk data contoh sama seperti untuk dokumen.
//
// # Kenapa contohnya mencakup kelima tahap
//
// Supaya layar daftar dapat dicoba pada keadaan yang sebenarnya — termasuk dua tahap yang
// hari ini BELUM DAPAT TERJADI di jalur nyata karena modul klaim belum ada
// (`StageAccepted` dan `StageRejected`, lihat komentar Stage pada paket pelaporanklaim).
// Tanpa contoh untuk keduanya, tab-nya selalu kosong dan tidak ada yang tahu apakah ia
// kosong karena benar atau karena rusak.
func SampleReports() []pelaporanklaim.ClaimReport {
	// Waktu tetap, bukan time.Now(): daftar contoh yang berubah setiap kali proses
	// dijalankan membuat tangkapan layar dan uji manual tidak dapat dibandingkan.
	base := time.Date(2026, time.September, 15, 2, 0, 0, 0, time.UTC)
	daysAgo := func(n int) *time.Time {
		t := base.AddDate(0, 0, -n)
		return &t
	}

	return []pelaporanklaim.ClaimReport{
		{
			Number:               "LPK.00.0001",
			ReporterName:         "Bagas Prasetya",
			SenderEmail:          "bagas.prasetya@contoh.example",
			SenderPhone:          "021-5550101",
			CourierName:          "JNE",
			EmailSubject:         "Laporan kerugian kebakaran gudang",
			PolicyNumber:         "CONTOH-PL-000117",
			InsuredName:          "PT Harapan Sentosa",
			InsuredEmail:         "klaim@harapansentosa.example",
			BusinessCode:         "006",
			GroupPanel:           "006",
			ReferenceNumber:      "REF-2026-0117",
			LossDate:             daysAgo(9),
			LossLocation:         "Gudang Cakung, Jakarta Timur",
			Chronology:           "Api muncul dari panel listrik lantai satu pada malam hari saat gudang kosong.",
			DamageDetails:        "Rak penyimpanan, atap seng, dan sebagian barang jadi.",
			EstimatedValue:       "450000000",
			ClaimType:            "Fire",
			DocumentCount:        6,
			DocumentReceivedDate: daysAgo(6),
			BranchCode:           "100081",
			CreatedBy:            "petugas.penerimaan",
			CreatedAt:            base.AddDate(0, 0, -8),
			UpdatedAt:            base.AddDate(0, 0, -8),
		},
		{
			Number:               "LPK.00.0002",
			ReporterName:         "Nadia Rahmawati",
			SenderEmail:          "nadia.r@contoh.example",
			SenderPhone:          "0811-555-0102",
			CourierName:          "Kurir internal cabang",
			EmailSubject:         "Klaim perjalanan — keterlambatan bagasi",
			PolicyNumber:         "CONTOH-PL-000204",
			InsuredName:          "Ibu Saraswati",
			InsuredEmail:         "saraswati@contoh.example",
			BusinessCode:         "005",
			GroupPanel:           "005",
			LossDate:             daysAgo(14),
			LossLocation:         "Bandara Changi, Singapura",
			Chronology:           "Bagasi tidak sampai bersama penerbangan dan baru diterima tiga hari kemudian.",
			DamageDetails:        "Barang pribadi dan biaya penggantian pakaian.",
			EstimatedValue:       "12500000",
			ClaimType:            "Travel",
			DocumentCount:        4,
			DocumentReceivedDate: daysAgo(10),
			Transferred:          true,
			TransferredAt:        daysAgo(9),
			BranchCode:           "100081",
			CreatedBy:            "petugas.penerimaan",
			CreatedAt:            base.AddDate(0, 0, -12),
			UpdatedAt:            base.AddDate(0, 0, -9),
		},
		{
			Number:               "LPK.00.0003",
			ReporterName:         "Rizal Abdurrahman",
			SenderEmail:          "rizal.a@contoh.example",
			SenderPhone:          "0812-555-0103",
			CourierName:          "TIKI",
			PolicyNumber:         "CONTOH-PL-000318",
			InsuredName:          "CV Bina Karya",
			BusinessCode:         "004",
			GroupPanel:           "004",
			ReferenceNumber:      "REF-2026-0318",
			LossDate:             daysAgo(25),
			LossLocation:         "Pelabuhan Tanjung Priok",
			Chronology:           "Kemasan kargo rusak saat bongkar muat dan sebagian isinya basah.",
			DamageDetails:        "Dua peti kemas berisi komponen mesin.",
			EstimatedValue:       "87500000",
			ClaimType:            "Marine Cargo",
			DocumentCount:        9,
			DocumentReceivedDate: daysAgo(21),
			Transferred:          true,
			TransferredAt:        daysAgo(20),
			NotRegisteredNote:    "Menunggu surat keterangan dari pihak pelabuhan.",
			BranchCode:           "100081",
			CreatedBy:            "petugas.penerimaan",
			CreatedAt:            base.AddDate(0, 0, -24),
			UpdatedAt:            base.AddDate(0, 0, -20),
		},
		{
			Number:               "LPK.00.0004",
			ReporterName:         "Yulianti Kusuma",
			SenderEmail:          "yulianti.k@contoh.example",
			SenderPhone:          "021-5550104",
			CourierName:          "Auto Service",
			PolicyNumber:         "CONTOH-PL-000422",
			InsuredName:          "Bapak Hendarto",
			BusinessCode:         "002",
			GroupPanel:           "002",
			LossDate:             daysAgo(40),
			LossLocation:         "Jalan Raya Bogor KM 21",
			Chronology:           "Tertanggung terjatuh saat menuruni tangga dan menjalani perawatan inap.",
			DamageDetails:        "Biaya perawatan rumah sakit.",
			DriverLicense:        "—",
			EstimatedValue:       "35000000",
			ClaimType:            "Personal Accident",
			DocumentCount:        11,
			DocumentReceivedDate: daysAgo(36),
			Transferred:          true,
			TransferredAt:        daysAgo(35),
			ClaimNumber:          "PNCN.26.0148",
			RegisteredAt:         daysAgo(34),
			BranchCode:           "100081",
			CreatedBy:            "petugas.penerimaan",
			CreatedAt:            base.AddDate(0, 0, -39),
			UpdatedAt:            base.AddDate(0, 0, -34),
		},
		{
			Number:               "LPK.00.0005",
			ReporterName:         "Teguh Wicaksono",
			SenderEmail:          "teguh.w@contoh.example",
			SenderPhone:          "0813-555-0105",
			CourierName:          "SiCepat",
			PolicyNumber:         "CONTOH-PL-000509",
			InsuredName:          "PT Sumber Makmur",
			BusinessCode:         "003",
			GroupPanel:           "003",
			LossDate:             daysAgo(70),
			LossLocation:         "Pabrik Karawang",
			Chronology:           "Mesin produksi berhenti setelah lonjakan arus listrik.",
			DamageDetails:        "Satu unit mesin injeksi.",
			EstimatedValue:       "260000000",
			ClaimType:            "Aneka",
			DocumentCount:        8,
			DocumentReceivedDate: daysAgo(66),
			Transferred:          true,
			TransferredAt:        daysAgo(65),
			ClaimNumber:          "PNCN.26.0092",
			RegisteredAt:         daysAgo(64),
			Outcome:              pelaporanklaim.OutcomeAccepted,
			BranchCode:           "100081",
			CreatedBy:            "petugas.penerimaan",
			CreatedAt:            base.AddDate(0, 0, -69),
			UpdatedAt:            base.AddDate(0, 0, -50),
		},
		{
			Number:               "LPK.00.0006",
			ReporterName:         "Sekar Ayu Ningrum",
			SenderEmail:          "sekar.a@contoh.example",
			SenderPhone:          "0821-555-0106",
			CourierName:          "Kurir internal cabang",
			PolicyNumber:         "CONTOH-PL-000611",
			InsuredName:          "Bapak Mulyadi",
			BusinessCode:         "002",
			GroupPanel:           "002",
			LossDate:             daysAgo(95),
			LossLocation:         "Perumahan Bintaro Sektor 9",
			Chronology:           "Laporan masuk jauh setelah batas waktu pelaporan polis.",
			DamageDetails:        "Kerusakan perabot.",
			EstimatedValue:       "18000000",
			ClaimType:            "Personal Accident",
			DocumentCount:        3,
			DocumentReceivedDate: daysAgo(90),
			Transferred:          true,
			TransferredAt:        daysAgo(89),
			ClaimNumber:          "PNCN.26.0071",
			RegisteredAt:         daysAgo(88),
			Outcome:              pelaporanklaim.OutcomeRejected,
			BranchCode:           "100081",
			CreatedBy:            "petugas.penerimaan",
			CreatedAt:            base.AddDate(0, 0, -94),
			UpdatedAt:            base.AddDate(0, 0, -80),
		},
	}
}

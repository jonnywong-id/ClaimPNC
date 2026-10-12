package memory

import (
	"time"

	"claim-pnc/internal/archivedokumenklaim"
)

// Data contoh untuk mode pengembangan tanpa basis data.
//
// # Kenapa isinya karangan, dan kenapa itu justru yang benar
//
// `D-64` menetapkan staging memakai salinan data produksi apa adanya — dan aturan kerja
// proyek melarang menuliskan data nasabah ke berkas yang di-commit (`D-69`). Data di sini
// karena itu SELURUHNYA karangan: nomor polis, nama tertanggung, dan nomor klaimnya tidak
// menunjuk siapa pun.
//
// Yang ditiru bukan isinya melainkan BENTUKNYA — termasuk empat keadaan yang paling
// mudah luput saat menggambar layar:
//
//	berkas yang sudah dikirim ke layanan Arsip     (CABANGSTATUS '1', ada tanggal kirim)
//	berkas yang belum dikirim                      (CABANGSTATUS '0', tanggal kirim kosong)
//	berkas lini PA dan Travel                      (menguji saringan jabatan)
//	berkas yang kode tipe dokumennya tidak ada di master (nama kosong, LEFT JOIN)

// SampleFiles adalah isi awal grid ARCHIVE FILE KLAIM.
func SampleFiles() []archivedokumenklaim.ArchiveFile {
	return []archivedokumenklaim.ArchiveFile{
		{
			ID:                   1,
			ClaimNumber:          "PNC-100001",
			PolicyNumber:         "POL-2024-000001",
			InsuredName:          "CONTOH TERTANGGUNG SATU",
			LossDate:             day(2024, 3, 11),
			TechnicalPIC:         "PICTEKNIK1",
			DocumentReceivedDate: day(2024, 3, 18),
			InputDate:            day(2024, 3, 20),
			SheetCount:           24,
			DocumentTypeCode:     "0001",
			DocumentKindCode:     "000101",
			BoxName:              "BOX-A-01",
			FillingCode:          "FIL-2024-001",
			InputUser:            "USERARSIP1",
			GroupPanel:           "006",
			BranchStatus:         archivedokumenklaim.BranchStatusPending,
		},
		{
			ID:                   2,
			ClaimNumber:          "PNC-100002",
			PolicyNumber:         "POL-2024-000002",
			InsuredName:          "CONTOH TERTANGGUNG DUA",
			LossDate:             day(2024, 4, 2),
			TechnicalPIC:         "PICTEKNIK2",
			DocumentReceivedDate: day(2024, 4, 9),
			InputDate:            day(2024, 4, 10),
			SheetCount:           8,
			DocumentTypeCode:     "0001",
			DocumentKindCode:     "000102",
			BoxName:              "BOX-A-01",
			FillingCode:          "FIL-2024-001",
			InputUser:            "USERARSIP1",
			SentDate:             day(2024, 4, 12),
			GroupPanel:           "004",
			BranchStatus:         archivedokumenklaim.BranchStatusSent,
			ServiceCode:          "200",
			ServiceNote:          "Berhasil diterima sistem Arsip",
		},
		{
			// Lini Personal Accident — ia HILANG dari daftar kirim ke cabang bagi
			// petugas berjabatan PA maupun NONMBU. Lihat archivedokumenklaim.BranchScope.
			ID:                   3,
			ClaimNumber:          "PNC-100003",
			PolicyNumber:         "POL-2024-000003",
			InsuredName:          "CONTOH TERTANGGUNG TIGA",
			LossDate:             day(2024, 5, 5),
			TechnicalPIC:         "PICTEKNIK1",
			DocumentReceivedDate: day(2024, 5, 14),
			InputDate:            day(2024, 5, 15),
			SheetCount:           112,
			DocumentTypeCode:     "0002",
			DocumentKindCode:     "000201",
			BoxName:              "BOX-B-07",
			FillingCode:          "FIL-2024-014",
			InputUser:            "USERARSIP2",
			GroupPanel:           archivedokumenklaim.GroupPanelPersonalAccident,
			BranchStatus:         archivedokumenklaim.BranchStatusPending,
		},
		{
			// Lini Travel — hilang bagi petugas berjabatan TRAVEL maupun NONMBU.
			ID:                   4,
			ClaimNumber:          "PNC-100004",
			PolicyNumber:         "POL-2024-000004",
			InsuredName:          "CONTOH TERTANGGUNG EMPAT",
			LossDate:             day(2024, 6, 1),
			TechnicalPIC:         "PICTEKNIK3",
			DocumentReceivedDate: day(2024, 6, 6),
			InputDate:            day(2024, 6, 7),
			SheetCount:           5,
			DocumentTypeCode:     "0002",
			DocumentKindCode:     "000202",
			BoxName:              "BOX-B-08",
			FillingCode:          "FIL-2024-015",
			InputUser:            "USERARSIP2",
			GroupPanel:           archivedokumenklaim.GroupPanelTravel,
			BranchStatus:         archivedokumenklaim.BranchStatusPending,
		},
		{
			// Kode tipe dokumennya TIDAK ADA di master, sehingga kedua nama dokumennya
			// kosong. Baris seperti ini nyata di data warisan, dan layar harus tetap
			// menggambarnya alih-alih menyembunyikannya.
			ID:                   5,
			ClaimNumber:          "PNCN.26.0001",
			PolicyNumber:         "POL-2026-000001",
			InsuredName:          "CONTOH TERTANGGUNG LIMA",
			LossDate:             day(2026, 1, 9),
			TechnicalPIC:         "PICTEKNIK3",
			DocumentReceivedDate: day(2026, 1, 15),
			InputDate:            day(2026, 1, 16),
			SheetCount:           40,
			DocumentTypeCode:     "9999",
			DocumentKindCode:     "999901",
			BoxName:              "BOX-C-02",
			FillingCode:          "FIL-2026-003",
			InputUser:            "USERARSIP1",
			GroupPanel:           "003",
			BranchStatus:         archivedokumenklaim.BranchStatusPending,
		},
	}
}

// SampleClaims adalah calon klaim pada bagian Input Data Archive.
func SampleClaims() []archivedokumenklaim.ClaimCandidate {
	return []archivedokumenklaim.ClaimCandidate{
		{
			Number:        "PNC-100006",
			PolicyNumber:  "POL-2024-000006",
			InsuredName:   "CONTOH TERTANGGUNG ENAM",
			LossDate:      day(2024, 7, 21),
			BusinessName:  "FIRE",
			BranchName:    "CABANG CONTOH",
			WorkStatus:    "Resolved-Completed",
			ClaimPosition: "Close Claim",
			CloseDate:     day(2024, 8, 30),
			CloseNote:     "Selesai dibayarkan",
			TechnicalPIC:  "PICTEKNIK1",
			GroupPanel:    "006",
		},
		{
			Number:        "PNC-100007",
			PolicyNumber:  "POL-2024-000006",
			InsuredName:   "CONTOH TERTANGGUNG ENAM",
			LossDate:      day(2024, 9, 2),
			BusinessName:  "MARINE CARGO",
			BranchName:    "CABANG CONTOH",
			WorkStatus:    "Resolved-Completed",
			ClaimPosition: "Paid",
			CloseDate:     day(2024, 10, 11),
			CloseNote:     "",
			TechnicalPIC:  "PICTEKNIK2",
			GroupPanel:    "004",
		},
		{
			// Klaim yang BELUM tutup. Sistem lama tidak melarang mengarsipkan berkasnya,
			// dan larangan itu tidak ditambahkan di sini.
			Number:        "PNCN.26.0002",
			PolicyNumber:  "POL-2026-000002",
			InsuredName:   "CONTOH TERTANGGUNG TUJUH",
			LossDate:      day(2026, 2, 14),
			BusinessName:  "PERSONAL ACCIDENT",
			BranchName:    "CABANG CONTOH",
			WorkStatus:    "Open",
			ClaimPosition: "Register",
			TechnicalPIC:  "PICTEKNIK3",
			GroupPanel:    archivedokumenklaim.GroupPanelPersonalAccident,
		},
	}
}

// SampleDocumentTypes adalah pilihan Tipe Dokumen.
func SampleDocumentTypes() []archivedokumenklaim.DocumentTypeOption {
	return []archivedokumenklaim.DocumentTypeOption{
		{Code: "0001", Name: "Dokumen Klaim"},
		{Code: "0002", Name: "Dokumen Survey"},
		{Code: "0003", Name: "Dokumen Pendukung"},
	}
}

// SampleDocumentKinds adalah pilihan Jenis Dokumen beserta tipe pemiliknya.
func SampleDocumentKinds() []archivedokumenklaim.DocumentKindOption {
	return []archivedokumenklaim.DocumentKindOption{
		{Code: "000101", Name: "Laporan Kerugian", TypeCode: "0001"},
		{Code: "000102", Name: "Polis dan Endorsement", TypeCode: "0001"},
		{Code: "000201", Name: "Berita Acara Survey", TypeCode: "0002"},
		{Code: "000202", Name: "Foto Lokasi", TypeCode: "0002"},
		{Code: "000301", Name: "Korespondensi", TypeCode: "0003"},
	}
}

// NewSampleRepo membentuk repo memori berisi seluruh data contoh.
func NewSampleRepo() *Repo {
	return NewRepo(Options{
		Files:  SampleFiles(),
		Claims: SampleClaims(),
		Types:  SampleDocumentTypes(),
		Kinds:  SampleDocumentKinds(),
	})
}

// day menyusun satu tanggal kalender UTC.
func day(year int, month time.Month, dayOfMonth int) *time.Time {
	moment := time.Date(year, month, dayOfMonth, 0, 0, 0, 0, time.UTC)
	return &moment
}

package memory

import (
	"time"

	"claim-pnc/internal/inboxlaporanklaim"
)

// # Seluruh isi berkas ini KARANGAN
//
// Nomor polis, nama tertanggung, nama cabang, dan isi percakapan di bawah tidak merujuk
// satu pun data nasabah nyata. `D-69` melarang data nasabah ditulis di artefak yang
// di-commit, dan larangan itu berlaku pula untuk data contoh — data contoh yang disalin
// dari produksi tetap data produksi.
//
// Yang ditiru dari kenyataan hanyalah BENTUK dan SEBARANNYA: setiap tab punya isi,
// jumlah barisnya melewati satu halaman supaya paginasi benar-benar teruji, dan ketiga
// Position hadir semuanya.

// businessMeta menitipkan dua kode yang di basis data nyata tinggal di tabel lain.
//
// POOLDATA.BUSINESS dan BUSINESSGROUP tidak ikut dikirim bersama export, dan menirunya
// sebagai dua tabel penuh di memori akan menambah banyak tanpa menambah apa pun yang
// diuji. Yang dibutuhkan penyaring "Bisnis" hanyalah dua kode ini per baris.
type businessMeta struct {
	groupPanel    string
	businessGroup string
}

var businessByID = map[string]businessMeta{
	"RCV-0001": {groupPanel: "002", businessGroup: "10001"}, // Personal Accident
	"RCV-0002": {groupPanel: "005", businessGroup: "10002"}, // Travel
	"RCV-0003": {groupPanel: "003", businessGroup: "10003"}, // Aneka
	"RCV-0004": {groupPanel: "006", businessGroup: "10004"}, // Fire
	"RCV-0005": {groupPanel: "004", businessGroup: "10005"}, // Marine Cargo
	"RCV-0006": {groupPanel: "003", businessGroup: "10008"}, // kelompok bisnis khusus
	"RCV-0007": {groupPanel: "002", businessGroup: "10001"},
	"RCV-0008": {groupPanel: "005", businessGroup: "10002"},
	"RCV-0009": {groupPanel: "003", businessGroup: "10015"}, // kelompok bisnis khusus
	"RCV-0010": {groupPanel: "006", businessGroup: "10004"},
	"RCV-0011": {groupPanel: "004", businessGroup: "10005"},
	"RCV-0012": {groupPanel: "002", businessGroup: "10001"},
	"RCV-0013": {groupPanel: "003", businessGroup: "10003"},
	"RCV-0014": {groupPanel: "005", businessGroup: "10002"},
}

func businessOf(id string) businessMeta { return businessByID[id] }

// Ketiga penanda di bawah menggantikan kolom yang penyimpanan memori tidak simpan:
// PYSTATUSWORK pada tabel Pega dan keberadaan NOAKSEPTASI di T_CLAIM_ADJUSTMENT.
var (
	// completedID: PYSTATUSWORK = 'Resolved-Completed'.
	completedID = map[string]bool{"RCV-0013": true}

	// rejectedID: klaimnya berstatus 'Resolved-Rejected' di T_CLAIM_PNC.
	rejectedID = map[string]bool{"RCV-0011": true, "RCV-0014": true}

	// acceptedID: klaimnya sudah punya NOAKSEPTASI.
	acceptedID = map[string]bool{"RCV-0004": true, "RCV-0010": true}
)

// SampleRegions adalah isi dropdown "Pilih Kanwil".
func SampleRegions() []inboxlaporanklaim.Region {
	return []inboxlaporanklaim.Region{
		{Code: "01", Name: "Kanwil Jakarta"},
		{Code: "02", Name: "Kanwil Jawa Barat"},
		{Code: "03", Name: "Kanwil Jawa Timur"},
	}
}

// SampleBranches memetakan kode cabang ke kanwilnya — pengganti POOLDATA.BRANCH.
func SampleBranches() map[string]string {
	return map[string]string{
		"1001": "01",
		"1002": "01",
		"2001": "02",
		"3001": "03",
	}
}

// SampleOptions merakit seluruh bahan contoh sekaligus.
//
// Ia satu fungsi, bukan empat yang harus dipanggil berurutan, supaya pemanggil tidak
// dapat merakit setengahnya — repo tanpa peta cabang akan menyaring kanwil menjadi
// kosong tanpa satu pun galat.
func SampleOptions(clock inboxlaporanklaim.Clock) Options {
	now := clock.Now()
	return Options{
		Rows:    SampleList(now),
		Region:  SampleRegions(),
		Branch:  SampleBranches(),
		Message: SampleMessages(now),
		Clock:   clock,
	}
}

// SampleList adalah empat belas berkas laporan contoh.
//
// Sebarannya disengaja: lebih dari satu halaman bawaan (10 baris) supaya paginasi
// teruji, ketiga Position hadir, dan setiap penyaring punya baris yang cocok maupun yang
// tidak. Tanggal aging dibuat mundur dari waktu acuan supaya kolom umur berkas
// menampilkan angka yang masuk akal berapa pun aplikasi dijalankan.
func SampleList(now time.Time) []inboxlaporanklaim.ClaimReport {
	day := func(n int) time.Time { return now.AddDate(0, 0, -n) }

	rows := []inboxlaporanklaim.ClaimReport{
		{
			ID: "RCV-0001", ClaimNumber: "PNC-1801", Transferred: true,
			AssignmentRef: "ASM-FW-GCNMFW-WORK RCV-0001",
			PolicyNumber:  "POL-PA-0001", InsuredName: "Tertanggung Contoh A",
			BusinessName: "Personal Accident", ReferenceNumber: "REF-0001",
			DateOfLoss: day(12), CreatedAt: day(10), CreatedBy: "adminpnc",
			BranchCode: "1001", BranchName: "Cabang Contoh Jakarta 1", AgingAt: day(10),
			EmailSubject: "Laporan kecelakaan diri",
		},
		{
			ID: "RCV-0002", ClaimNumber: "", Transferred: true,
			AssignmentRef: "ASM-FW-GCNMFW-WORK RCV-0002",
			PolicyNumber:  "POL-TRV-0002", InsuredName: "Tertanggung Contoh B",
			BusinessName: "Travel", ReferenceNumber: "REF-0002",
			DateOfLoss: day(9), CreatedAt: day(8), CreatedBy: "adminpnc",
			BranchCode: "1001", BranchName: "Cabang Contoh Jakarta 1", AgingAt: day(8),
			Reason: "Dokumen pendukung belum lengkap", EmailSubject: "Klaim perjalanan",
		},
		{
			ID: "RCV-0003", ClaimNumber: "", Transferred: false,
			PolicyNumber: "POL-ANK-0003", InsuredName: "Tertanggung Contoh C",
			BusinessName: "Aneka", ReferenceNumber: "REF-0003",
			DateOfLoss: day(7), CreatedAt: day(7), CreatedBy: "adminpnc",
			BranchCode: "1001", BranchName: "Cabang Contoh Jakarta 1", AgingAt: day(7),
			EmailSubject: "Laporan kerugian aneka",
		},
		{
			ID: "RCV-0004", ClaimNumber: "PNC-1804", Transferred: true,
			AssignmentRef: "ASM-FW-GCNMFW-WORK RCV-0004",
			PolicyNumber:  "POL-FIR-0004", InsuredName: "Tertanggung Contoh D",
			BusinessName: "Fire", ReferenceNumber: "REF-0004",
			DateOfLoss: day(30), CreatedAt: day(28), CreatedBy: "pictekniks",
			BranchCode: "1002", BranchName: "Cabang Contoh Jakarta 2", AgingAt: day(28),
			EmailSubject: "Laporan kebakaran gudang",
		},
		{
			ID: "RCV-0005", ClaimNumber: "", Transferred: false,
			PolicyNumber: "POL-MRN-0005", InsuredName: "Tertanggung Contoh E",
			BusinessName: "Marine Cargo", ReferenceNumber: "REF-0005",
			DateOfLoss: day(5), CreatedAt: day(5), CreatedBy: "pictekniks",
			BranchCode: "1002", BranchName: "Cabang Contoh Jakarta 2", AgingAt: day(5),
		},
		{
			ID: "RCV-0006", ClaimNumber: "", Transferred: true,
			AssignmentRef: "ASM-FW-GCNMFW-WORK RCV-0006",
			PolicyNumber:  "POL-KHS-0006", InsuredName: "Tertanggung Contoh F",
			BusinessName: "Kelompok khusus", ReferenceNumber: "REF-0006",
			DateOfLoss: day(4), CreatedAt: day(4), CreatedBy: "adminpnc",
			BranchCode: "2001", BranchName: "Cabang Contoh Bandung", AgingAt: day(4),
			Reason: "Menunggu konfirmasi cabang",
		},
		{
			ID: "RCV-0007", ClaimNumber: "PNC-1807", Transferred: true,
			AssignmentRef: "ASM-FW-GCNMFW-WORK RCV-0007",
			PolicyNumber:  "POL-PA-0007", InsuredName: "Tertanggung Contoh G",
			BusinessName: "Personal Accident", ReferenceNumber: "REF-0007",
			DateOfLoss: day(20), CreatedAt: day(18), CreatedBy: "adminpnc",
			BranchCode: "2001", BranchName: "Cabang Contoh Bandung", AgingAt: day(18),
		},
		{
			ID: "RCV-0008", ClaimNumber: "", Transferred: false,
			PolicyNumber: "POL-TRV-0008", InsuredName: "Tertanggung Contoh H",
			BusinessName: "Travel", ReferenceNumber: "REF-0008",
			DateOfLoss: day(3), CreatedAt: day(3), CreatedBy: "adminpnc",
			BranchCode: "3001", BranchName: "Cabang Contoh Surabaya", AgingAt: day(3),
		},
		{
			ID: "RCV-0009", ClaimNumber: "", Transferred: true,
			AssignmentRef: "ASM-FW-GCNMFW-WORK RCV-0009",
			PolicyNumber:  "POL-KHS-0009", InsuredName: "Tertanggung Contoh I",
			BusinessName: "Kelompok khusus", ReferenceNumber: "REF-0009",
			DateOfLoss: day(15), CreatedAt: day(14), CreatedBy: "pictekniks",
			BranchCode: "3001", BranchName: "Cabang Contoh Surabaya", AgingAt: day(14),
			Reason: "Polis belum ditemukan",
		},
		{
			ID: "RCV-0010", ClaimNumber: "PNC-1810", Transferred: true,
			AssignmentRef: "ASM-FW-GCNMFW-WORK RCV-0010",
			PolicyNumber:  "POL-FIR-0010", InsuredName: "Tertanggung Contoh J",
			BusinessName: "Fire", ReferenceNumber: "REF-0010",
			DateOfLoss: day(45), CreatedAt: day(40), CreatedBy: "adminpnc",
			BranchCode: "1001", BranchName: "Cabang Contoh Jakarta 1", AgingAt: day(40),
		},
		{
			ID: "RCV-0011", ClaimNumber: "PNC-1811", Transferred: true,
			AssignmentRef: "ASM-FW-GCNMFW-WORK RCV-0011",
			PolicyNumber:  "POL-MRN-0011", InsuredName: "Tertanggung Contoh K",
			BusinessName: "Marine Cargo", ReferenceNumber: "REF-0011",
			DateOfLoss: day(60), CreatedAt: day(55), CreatedBy: "adminpnc",
			BranchCode: "1001", BranchName: "Cabang Contoh Jakarta 1", AgingAt: day(55),
			Reason: "Kerugian tidak dijamin polis",
		},
		{
			ID: "RCV-0012", ClaimNumber: "", Transferred: false,
			PolicyNumber: "POL-PA-0012", InsuredName: "Tertanggung Contoh L",
			BusinessName: "Personal Accident", ReferenceNumber: "REF-0012",
			DateOfLoss: day(2), CreatedAt: day(2), CreatedBy: "adminpnc",
			BranchCode: "1001", BranchName: "Cabang Contoh Jakarta 1", AgingAt: day(2),
		},
		{
			ID: "RCV-0013", ClaimNumber: "PNC-1813", Transferred: true,
			AssignmentRef: "ASM-FW-GCNMFW-WORK RCV-0013",
			PolicyNumber:  "POL-ANK-0013", InsuredName: "Tertanggung Contoh M",
			BusinessName: "Aneka", ReferenceNumber: "REF-0013",
			DateOfLoss: day(90), CreatedAt: day(85), CreatedBy: "adminpnc",
			BranchCode: "1001", BranchName: "Cabang Contoh Jakarta 1", AgingAt: day(85),
		},
		{
			ID: "RCV-0014", ClaimNumber: "PNC-1814", Transferred: true,
			AssignmentRef: "ASM-FW-GCNMFW-WORK RCV-0014",
			PolicyNumber:  "POL-TRV-0014", InsuredName: "Tertanggung Contoh N",
			BusinessName: "Travel", ReferenceNumber: "REF-0014",
			DateOfLoss: day(70), CreatedAt: day(65), CreatedBy: "pictekniks",
			BranchCode: "2001", BranchName: "Cabang Contoh Bandung", AgingAt: day(65),
			Reason: "Laporan melewati batas waktu",
		},
	}

	// Position diturunkan, tidak diketik: aturannya satu dan hidup di domain, sehingga
	// berkas contoh tidak dapat memuat kombinasi yang mustahil.
	for i := range rows {
		rows[i].Origin = inboxlaporanklaim.OriginLegacy
		rows[i].Position = inboxlaporanklaim.DerivePosition(rows[i].Registered(), rows[i].Transferred)
	}
	return rows
}

// SampleMessages adalah percakapan contoh untuk ketiga tab komunikasi.
//
// Ia hanya menyentuh berkas milik `adminpnc`, karena ketiga tab itu memang menyaring
// berkas yang dibuat pemanggil sendiri.
func SampleMessages(now time.Time) map[string][]messageRow {
	day := func(n int) time.Time { return now.AddDate(0, 0, -n) }

	return map[string][]messageRow{
		"RCV-0002": {
			{Status: "0", Sender: "adminpnc", Text: "Mohon dibantu cek kelengkapan berkas.", SentAt: day(6)},
		},
		"RCV-0003": {
			{Status: "0", Sender: "kantorpusat", Text: "Dokumen kronologi belum kami terima.", SentAt: day(5)},
		},
		"RCV-0006": {
			{Status: "1", Sender: "kantorpusat", Text: "Sudah kami proses, silakan lanjutkan.", SentAt: day(3)},
		},
		"RCV-0012": {
			{Status: "0", Sender: "adminpnc", Text: "Berkas menunggu konfirmasi polis.", SentAt: day(1)},
			{Status: "0", Sender: "kantorpusat", Text: "Polis sedang kami telusuri.", SentAt: day(1)},
		},
	}
}

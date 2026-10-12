package memory

import "time"

// SampleRows adalah baris contoh Laporan Hasil AI untuk pengembangan lokal dan pengujian.
//
// # Nilainya KARANGAN, dan itu disengaja
//
// Tidak satu pun nomor klaim, nama tertanggung, maupun nomor polis nyata boleh masuk ke
// berkas yang di-commit (`D-69`). Yang ditiru di sini adalah BENTUK dan KEANEHAN datanya,
// bukan isinya.
//
// # Apa yang sengaja diwakili
//
// Setiap baris ada untuk membuktikan satu hal, dan tanpa salah satunya ada penyaring atau
// aturan tampilan yang tidak akan pernah ketahuan salah:
//
//	AI setuju dengan komite          jalur paling umum
//	AI berbeda dengan komite         justru inilah yang dicari laporan ini
//	komite MENUNGGU                  membuktikan kolom "Menunggu" pada ringkasan
//	AI belum menilai (kosong)        membuktikan Total != jumlah baris
//	jenjang kedua pada klaim sama    membuktikan No Klaim dikosongkan
//	dua objek pada satu jenjang      membuktikan satu baris = satu penilaian AI
//	TANGGALKOMITE kosong             WAJIB HILANG dari hasil
//	case belum Resolved-Completed    WAJIB HILANG dari hasil
//	di luar rentang tanggal          WAJIB HILANG pada penyaring yang wajar
//
// Dua yang terakhir sebelum baris penutup adalah yang paling penting: keduanya harus
// TIDAK MUNCUL. Data contoh yang seluruhnya lolos penyaring tidak menguji penyaring apa
// pun.
func SampleRows() []Row {
	// Tanggal disusun sebagai tanggal kalender polos di UTC, sama seperti yang dihasilkan
	// Filter.Clean. Tidak ada penambahan tujuh jam di mana pun.
	day := func(year int, month time.Month, date int) time.Time {
		return time.Date(year, month, date, 0, 0, 0, 0, time.UTC)
	}

	return []Row{
		// Satu klaim, satu jenjang, dua objek pertanggungan. Keduanya muncul sebagai dua
		// baris; hanya baris pertama yang bernomor klaim karena keduanya jenjang "1" —
		// dan itu memang yang terjadi di Pega: aturannya melihat JENJANG, bukan urutan
		// baris. Akibatnya nomor klaim yang sama tergambar dua kali.
		{
			CommitteeID:   "KMT-000101",
			CommitteeStep: "1",
			ObjectID:      "OBJ-01",
			CoverageID:    "CVG-01",
			ClaimNumber:   "PNCN.26.0001",
			ApproveCode:   "1",
			CommitteeDate: day(2026, time.September, 3),
			AIResult:      "DITERIMA",
			AIDate:        day(2026, time.September, 1),
			CaseResolved:  true,
		},
		{
			CommitteeID:   "KMT-000101",
			CommitteeStep: "1",
			ObjectID:      "OBJ-02",
			CoverageID:    "CVG-01",
			ClaimNumber:   "PNCN.26.0001",
			ApproveCode:   "1",
			CommitteeDate: day(2026, time.September, 3),
			AIResult:      "DITOLAK",
			AIDate:        day(2026, time.September, 1),
			CaseResolved:  true,
		},

		// Jenjang KEDUA pada klaim yang sama: nomor klaimnya dikosongkan.
		{
			CommitteeID:   "KMT-000101",
			CommitteeStep: "2",
			ObjectID:      "OBJ-01",
			CoverageID:    "CVG-01",
			ClaimNumber:   "PNCN.26.0001",
			ApproveCode:   "1",
			CommitteeDate: day(2026, time.September, 5),
			AIResult:      "DITERIMA",
			AIDate:        day(2026, time.September, 1),
			CaseResolved:  true,
		},

		// AI menerima, komite MENOLAK — selisih pendapat, yang justru dicari laporan ini.
		{
			CommitteeID:   "KMT-000102",
			CommitteeStep: "1",
			ObjectID:      "OBJ-01",
			CoverageID:    "CVG-02",
			ClaimNumber:   "PNCN.26.0002",
			ApproveCode:   "2",
			CommitteeDate: day(2026, time.September, 8),
			AIResult:      "DITERIMA",
			AIDate:        day(2026, time.September, 6),
			CaseResolved:  true,
		},

		// Komite belum memutuskan. Ia masuk kolom "Menunggu" dan TIDAK masuk Total.
		{
			CommitteeID:   "KMT-000103",
			CommitteeStep: "1",
			ObjectID:      "OBJ-01",
			CoverageID:    "CVG-03",
			ClaimNumber:   "PNCN.26.0003",
			ApproveCode:   "0",
			CommitteeDate: day(2026, time.September, 10),
			AIResult:      "DITOLAK",
			AIDate:        day(2026, time.September, 9),
			CaseResolved:  true,
		},

		// AI belum menilai sama sekali. Barisnya TETAP muncul — komite memutuskan tanpa
		// penilaian AI, dan itu keadaan yang justru perlu terlihat.
		{
			CommitteeID:   "KMT-000104",
			CommitteeStep: "1",
			ObjectID:      "OBJ-01",
			CoverageID:    "CVG-01",
			ClaimNumber:   "PNCN.26.0004",
			ApproveCode:   "1",
			CommitteeDate: day(2026, time.September, 12),
			AIResult:      "",
			AIDate:        time.Time{},
			CaseResolved:  true,
		},

		// HILANG — TANGGALKOMITE kosong.
		{
			CommitteeID:   "KMT-000105",
			CommitteeStep: "1",
			ObjectID:      "OBJ-01",
			CoverageID:    "CVG-01",
			ClaimNumber:   "PNCN.26.0005",
			ApproveCode:   "1",
			CommitteeDate: time.Time{},
			AIResult:      "DITERIMA",
			AIDate:        day(2026, time.September, 11),
			CaseResolved:  true,
		},

		// HILANG — belum ada baris komite ber-STATUSCASE Resolved-Completed.
		{
			CommitteeID:   "KMT-000106",
			CommitteeStep: "1",
			ObjectID:      "OBJ-01",
			CoverageID:    "CVG-01",
			ClaimNumber:   "PNCN.26.0006",
			ApproveCode:   "0",
			CommitteeDate: day(2026, time.September, 13),
			AIResult:      "DITERIMA",
			AIDate:        day(2026, time.September, 12),
			CaseResolved:  false,
		},

		// Jauh di luar rentang yang wajar dipilih pengguna.
		{
			CommitteeID:   "KMT-000107",
			CommitteeStep: "1",
			ObjectID:      "OBJ-01",
			CoverageID:    "CVG-01",
			ClaimNumber:   "PNCN.25.0099",
			ApproveCode:   "1",
			CommitteeDate: day(2025, time.March, 4),
			AIResult:      "DITERIMA",
			AIDate:        day(2025, time.March, 1),
			CaseResolved:  true,
		},
	}
}

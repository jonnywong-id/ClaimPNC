package memory

import "claim-pnc/internal/inboxautoclaim"

// SampleMaster adalah isi contoh POOLDATA.M_AUTO_CLAIM_PNC.
//
// # Dari mana nama-nama ini berasal
//
// TIDAK dari data nyata. Isi tabelnya tidak pernah diterima — hanya kolomnya yang terbaca
// dari RDB List/BrowseAutoKlaim-SQL.xml dan DDL-nya. Nama di bawah KARANGAN, dan sengaja
// terdengar seperti nama lembaga pembiayaan supaya layarnya terasa nyata tanpa menyalin
// satu pun data nasabah ke dalam repository (D-69).
//
// `SRVY` sengaja ada tetapi tidak punya batch satu pun: ia yang membuktikan penyaring
// perusahaan membaca MASTER, bukan tabel batch — dengan sumber yang salah, ia tidak akan
// muncul di dropdown.
func SampleMaster() map[string]string {
	return map[string]string{
		"MFIN": "Mitra Finansial Nusantara",
		"BPRC": "Bank Perkreditan Cahaya",
		"KRDU": "Kredit Utama Sejahtera",
		"SRVY": "Sarana Vista Pembiayaan",
	}
}

// SamplePolicy meniru hasil pencarian polis pada rantai unggah.
//
// Di basis data ia dua kueri berbeda — `GetReceiverClaimAsuransiKredit` atas T_GENERAL dan
// `BrowsePolisAso` atas JSON_POLIS. Di sini keduanya disatukan karena yang ditiru adalah
// HASILNYA.
//
// Keempat baris memperlihatkan empat keadaan yang masing-masing menghasilkan perilaku
// berbeda, dan ketiga yang terakhir mudah terlewat saat menguji dengan tangan:
//
//	polis lengkap                    -> baris lolos
//	perusahaan kosong                -> baris DITOLAK, tidak disimpan sama sekali
//	prodke kosong                    -> baris disimpan bertanda "No Polis tidak di temukan"
//	polis tidak terdaftar sama sekali -> sama dengan perusahaan kosong
func SamplePolicy() map[string]PolicyRow {
	return map[string]PolicyRow{
		"0100120260500": {CompanyCode: "MFIN", ProductSeq: "1"},
		"0100120260501": {CompanyCode: "MFIN", ProductSeq: "2"},
		"0200120260500": {CompanyCode: "BPRC", ProductSeq: "1"},

		// Polis ada di T_GENERAL tetapi sumber bisnisnya tidak menunjuk perusahaan
		// rekanan yang aktif.
		"0900120260500": {CompanyCode: "", ProductSeq: "1"},

		// Polis dikenali perusahaan tetapi tidak ada di JSON_POLIS.
		"0100120260999": {CompanyCode: "MFIN", ProductSeq: ""},
	}
}

// SampleLines adalah isi contoh POOLDATA.TMP_BATCH_AUTO_CLAIM.
//
// Empat batch yang sengaja memperlihatkan empat keadaan berbeda:
//
//	MFIN batch 2  seluruhnya selesai, ada yang berhasil dan ada yang gagal
//	MFIN batch 1  belum diproses sama sekali  -> kedua tombol ekspor menghasilkan berkas kosong
//	BPRC batch 1  baru diproses sebagian      -> "di upload" dan "telah diproses" berselisih
//	ZZZZ batch 1  kode perusahaan TIDAK ADA di master -> nama perusahaannya kosong
//
// Batch terakhir ada dengan sengaja. Ia yang membuktikan LEFT JOIN benar-benar berlaku —
// dengan INNER JOIN seperti kueri Pega aslinya, barisnya hilang dari layar, dan hilangnya
// tidak akan terlihat kecuali ada data seperti ini saat mencoba.
//
// CURRENCY diisi ID beserta kodenya, mengikuti bentuk aslinya: kolomnya menyimpan id, dan
// kode yang dibaca pengguna adalah hasil lookup ke POOLDATA.CURRENCY.
//
// Nomor polis, nilai, dan tanggalnya karangan. Tanggalnya berformat dd/mm/yyyy mengikuti
// bentuk yang dibaca Activity/CreateCasePNC_AutoClaim-Act.xml.
func SampleLines() []inboxautoclaim.Line {
	const sukses = inboxautoclaim.MessageSuccess

	return []inboxautoclaim.Line{
		// MFIN batch 2 — selesai seluruhnya.
		{
			CompanyCode: "MFIN", BatchNumber: "2", ProcessedDate: "12/01/2026",
			PolicyNo: "0100120260001", ProductSeq: "1",
			ClaimID: "PNCN.26.101", AcceptanceNo: "AKS-26-0001",
			Currency: "1", CurrencyCode: "IDR", ClaimAmount: "12500000.00",
			CauseOfLoss: "12002", DateOfLoss: "03/01/2026", ReportDate: "05/01/2026",
			Note: "Klaim meninggal dunia", Keyword: "REF-MFIN-0001",
			Message: sukses, UploadedBy: "ADMINPNC",
		},
		{
			CompanyCode: "MFIN", BatchNumber: "2", ProcessedDate: "12/01/2026",
			PolicyNo: "0100120260002", ProductSeq: "1",
			ClaimID: "PNCN.26.102", AcceptanceNo: "AKS-26-0002",
			Currency: "1", CurrencyCode: "IDR", ClaimAmount: "7800000.50",
			CauseOfLoss: "12002", DateOfLoss: "04/01/2026", ReportDate: "06/01/2026",
			Keyword: "REF-MFIN-0002",
			Message: sukses, UploadedBy: "ADMINPNC",
		},
		{
			// Baris gagal: ketiga kolom penanda membawa teks galat yang sama, persis
			// seperti yang dilakukan InsertKlaimToTable_Other.
			CompanyCode: "MFIN", BatchNumber: "2", ProcessedDate: "12/01/2026",
			PolicyNo: "0100120260003", ProductSeq: "1",
			ClaimID:      "Penyebab kerugian tidak ditemukan",
			AcceptanceNo: "Penyebab kerugian tidak ditemukan",
			Currency:     "1", CurrencyCode: "IDR", ClaimAmount: "4300000.00",
			DateOfLoss: "05/01/2026", ReportDate: "07/01/2026",
			Keyword: "REF-MFIN-0003",
			Message: "Penyebab kerugian tidak ditemukan", UploadedBy: "ADMINPNC",
		},
		{
			CompanyCode: "MFIN", BatchNumber: "2", ProcessedDate: "12/01/2026",
			PolicyNo: "0100120260004", ProductSeq: "2",
			ClaimID:      inboxautoclaim.MessagePolicyNotFound,
			AcceptanceNo: inboxautoclaim.MessagePolicyNotFound,
			Currency:     "1", CurrencyCode: "IDR", ClaimAmount: "15000000.00",
			CauseOfLoss: "12002", DateOfLoss: "06/01/2026", ReportDate: "09/01/2026",
			Keyword: "REF-MFIN-0004",
			Message: inboxautoclaim.MessagePolicyNotFound, UploadedBy: "ADMINPNC",
		},

		// MFIN batch 1 — belum diproses sama sekali.
		{
			CompanyCode: "MFIN", BatchNumber: "1", ProcessedDate: "05/01/2026",
			PolicyNo: "0100120260010", ProductSeq: "1",
			Currency: "1", CurrencyCode: "IDR", ClaimAmount: "9250000.00",
			CauseOfLoss: "12002", DateOfLoss: "12/02/2026", ReportDate: "13/02/2026",
			Keyword: "REF-MFIN-0010", UploadedBy: "ADMINPNC",
		},
		{
			CompanyCode: "MFIN", BatchNumber: "1", ProcessedDate: "05/01/2026",
			PolicyNo: "0100120260011", ProductSeq: "1",
			Currency: "1", CurrencyCode: "IDR", ClaimAmount: "3100000.00",
			CauseOfLoss: "12002", DateOfLoss: "12/02/2026", ReportDate: "14/02/2026",
			Keyword: "REF-MFIN-0011", UploadedBy: "ADMINPNC",
		},

		// BPRC batch 1 — baru sebagian diproses.
		{
			CompanyCode: "BPRC", BatchNumber: "1", ProcessedDate: "20/01/2026",
			PolicyNo: "0200120260001", ProductSeq: "1",
			ClaimID: "PNCN.26.201", AcceptanceNo: "AKS-26-0201",
			Currency: "1", CurrencyCode: "IDR", ClaimAmount: "22000000.00",
			CauseOfLoss: "12002", DateOfLoss: "20/01/2026", ReportDate: "22/01/2026",
			Keyword: "REF-BPRC-0001",
			Message: sukses, UploadedBy: "PICTEKNIKS",
		},
		{
			CompanyCode: "BPRC", BatchNumber: "1", ProcessedDate: "20/01/2026",
			PolicyNo: "0200120260002", ProductSeq: "1",
			Currency: "1", CurrencyCode: "IDR", ClaimAmount: "5400000.00",
			CauseOfLoss: "12002", DateOfLoss: "21/01/2026", ReportDate: "23/01/2026",
			Keyword: "REF-BPRC-0002", UploadedBy: "PICTEKNIKS",
		},
		{
			CompanyCode: "BPRC", BatchNumber: "1", ProcessedDate: "20/01/2026",
			PolicyNo: "0200120260003", ProductSeq: "1",
			Currency: "2", CurrencyCode: "USD", ClaimAmount: "1500.75",
			CauseOfLoss: "12002", DateOfLoss: "22/01/2026", ReportDate: "24/01/2026",
			Keyword: "REF-BPRC-0003", UploadedBy: "PICTEKNIKS",
		},

		// ZZZZ batch 1 — perusahaan yang TIDAK ADA di master.
		{
			CompanyCode: "ZZZZ", BatchNumber: "1", ProcessedDate: "02/03/2026",
			PolicyNo: "0900120260001", ProductSeq: "1",
			ClaimID:      "Penerima klaim tidak ditemukan",
			AcceptanceNo: "Penerima klaim tidak ditemukan",
			Currency:     "1", CurrencyCode: "IDR", ClaimAmount: "1000000.00",
			CauseOfLoss: "12002", DateOfLoss: "02/03/2026", ReportDate: "03/03/2026",
			Keyword: "REF-ZZZZ-0001",
			Message: "Penerima klaim tidak ditemukan", UploadedBy: "ADMINPNC",
		},
	}
}

// SampleKreditLines adalah isi contoh POOLDATA.TMP_BATCH_CLAIM_KREDIT — tab Asuransi
// Kredit.
//
// Isinya SENGAJA berbeda dari tab ANEKA, bukan salinannya. Dengan isi yang sama, layar
// yang lupa mengirim `?sumber=` akan tampak benar di kedua tab — persis cacat yang paling
// mungkin terjadi dan paling sulit terlihat, karena tabel tetap terisi dan angkanya tetap
// masuk akal.
func SampleKreditLines() []inboxautoclaim.Line {
	return []inboxautoclaim.Line{
		{
			CompanyCode: "KRDU", BatchNumber: "7", ProcessedDate: "11/02/2026",
			PolicyNo: "0300120260101", ProductSeq: "1",
			ClaimID: "PNCN.26.501", AcceptanceNo: "AKS-26-0501",
			Currency: "1", CurrencyCode: "IDR", ClaimAmount: "24000000.00",
			CauseOfLoss: "12002", DateOfLoss: "02/02/2026", ReportDate: "04/02/2026",
			Keyword: "REF-KRDU-0001",
			Message: "Sukses Klaim", UploadedBy: "ADMINPNC",
		},
		{
			CompanyCode: "KRDU", BatchNumber: "7", ProcessedDate: "11/02/2026",
			PolicyNo: "0300120260102", ProductSeq: "1",
			Currency: "1", CurrencyCode: "IDR", ClaimAmount: "8750000.00",
			CauseOfLoss: "", DateOfLoss: "03/02/2026", ReportDate: "06/02/2026",
			Keyword: "REF-KRDU-0002",
			Message: "Penyebab kerugian tidak ditemukan", UploadedBy: "ADMINPNC",
		},
		{
			CompanyCode: "BPRC", BatchNumber: "3", ProcessedDate: "18/02/2026",
			PolicyNo: "0200120260110", ProductSeq: "1",
			Currency: "1", CurrencyCode: "IDR", ClaimAmount: "3200000.00",
			CauseOfLoss: "12002", DateOfLoss: "10/02/2026", ReportDate: "12/02/2026",
			Keyword: "REF-BPRC-0100", UploadedBy: "PICTEKNIKS",
		},
	}
}

// SampleTravelLines adalah isi contoh POOLDATA.TMP_BATCH_AUTO_TRAVEL — tab Travel.
func SampleTravelLines() []inboxautoclaim.Line {
	return []inboxautoclaim.Line{
		{
			CompanyCode: "SRVY", BatchNumber: "2", ProcessedDate: "05/03/2026",
			PolicyNo: "0400120260201", ProductSeq: "1",
			ClaimID: "PNCN.26.611", AcceptanceNo: "AKS-26-0611",
			// Travel banyak bervaluta asing; baris ini yang memperlihatkan kolom mata
			// uang benar-benar dipakai, bukan selalu IDR.
			Currency: "2", CurrencyCode: "USD", ClaimAmount: "820.50",
			CauseOfLoss: "12002", DateOfLoss: "20/02/2026", ReportDate: "22/02/2026",
			Keyword: "REF-SRVY-0001",
			Message: "Sukses Klaim", UploadedBy: "ADMINPNC",
		},
		{
			CompanyCode: "SRVY", BatchNumber: "2", ProcessedDate: "05/03/2026",
			PolicyNo: "0400120260202", ProductSeq: "1",
			Currency: "2", CurrencyCode: "USD", ClaimAmount: "410.00",
			CauseOfLoss: "12002", DateOfLoss: "21/02/2026", ReportDate: "23/02/2026",
			Keyword: "REF-SRVY-0002", UploadedBy: "ADMINPNC",
		},
	}
}

// NewSampleRepo membentuk penyimpanan memori beserta isi contoh KETIGA tab.
//
// Ketiganya diisi, bukan hanya tab bawaan: tab yang kosong di lingkungan pengembangan
// tidak dapat dibedakan dari tab yang gagal memuat, dan keduanya tampak sama di layar.
func NewSampleRepo() *Repo {
	repo := NewRepo(SampleMaster(), SamplePolicy(), SampleLines()...)
	repo.Seed(inboxautoclaim.SourceKredit, SampleKreditLines()...)
	repo.Seed(inboxautoclaim.SourceTravel, SampleTravelLines()...)
	return repo
}

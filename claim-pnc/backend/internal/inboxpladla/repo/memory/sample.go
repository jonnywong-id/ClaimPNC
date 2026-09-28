package memory

import "time"

// pegaWorkKeyPrefix adalah awalan kunci objek kerja Pega pada `T_CLAIM_PNC.CLAIMID`.
//
// Ia ditiru di data contoh karena pencarian di layar ini mencocokkan KUNCI itu, bukan
// nomor klaim. Perhatikan SPASI di ujungnya — ia bagian dari nilainya.
const pegaWorkKeyPrefix = "ASM-FW-GCNMFW-WORK "

func workKey(claimNo string) string { return pegaWorkKeyPrefix + claimNo }

func day(year int, month time.Month, date int) time.Time {
	return time.Date(year, month, date, 0, 0, 0, 0, time.UTC)
}

// SampleReinsurerLogin adalah login reasuradur pada data contoh.
//
// Ia diekspor supaya perakitan di cmd dapat menyebutkannya saat menjelaskan mengapa layar
// ini kosong bagi login lain — dan supaya uji tidak perlu menebaknya.
const SampleReinsurerLogin = "REASCONTOH"

// SampleSecondLogin adalah login reasuradur KEDUA pada data contoh.
//
// Ia punya DUA kode reasuradur, dan itu satu-satunya cara menguji perbedaan antara daftar
// yang memakai seluruh kode dan daftar yang hanya memakai kode tertinggi.
const SampleSecondLogin = "REASGANDA"

// NewSampleStore membentuk penyimpanan berisi data contoh.
//
// # Janji yang dipegang data di bawah
//
// SETIAP penyaring punya baris yang cocok MAUPUN yang tidak. Yang membuktikan sebuah
// penyaring bekerja bukan baris yang muncul, melainkan baris yang seharusnya TIDAK muncul
// dan memang tidak muncul.
//
// Empat baris punya tugas tambahan, dan keempatnya menjaga perilaku Pega yang mudah
// "diperbaiki" menjadi salah:
//
//	PNC-2004  DLA ditandai terkirim TANPA tanggal kirim -> TIDAK dihitung terkirim
//	PNC-2005  Resolved-Completed + menunggu tutup       -> muncul di DLA, status 1139
//	PNC-2006  Resolved-Rejected                         -> TIDAK muncul di daftar mana pun
//	PNC-2007  tanpa baris tabel kerja Pega              -> muncul di PLA, TIDAK di DLA
//	PNC-2009  lini Personal Accident TANPA pemberitahuan -> HANYA di daftar komunikasi
//
// Baris terakhir itu satu-satunya yang membuktikan DUA perbedaan sekaligus antara daftar
// komunikasi dan daftar pemberitahuan: ia lini `002` yang ketiga daftar pemberitahuan
// kecualikan, dan ia tidak punya satu pun PLA maupun DLA.
func NewSampleStore() *Store {
	store := NewStore()
	store.Seed(
		sampleClaims(), sampleAdvices(),
		sampleReinsurers(), sampleXOL(), sampleLabels(),
	)
	store.SeedMessages(sampleMessages())
	store.SeedDocuments(sampleDocuments())
	return store
}

// sampleReinsurers memetakan dua login ke tiga kode.
//
// `REASGANDA` sengaja punya dua kode — `R900` dan `R901` — supaya perbedaan antara
// `reinscode IN (…)` pada daftar Close dan `reinscode = (… FETCH 1)` pada dua daftar lain
// benar-benar dapat diuji. Tanpa login bergkode ganda, kedua bentuk itu menghasilkan
// jawaban yang sama dan perbedaannya tidak akan pernah terlihat.
func sampleReinsurers() []Reinsurer {
	return []Reinsurer{
		{Code: "R100", Login: SampleReinsurerLogin},
		{Code: "R900", Login: SampleSecondLogin},
		{Code: "R901", Login: SampleSecondLogin},
	}
}

// sampleLabels adalah sebagian isi `POOLDATA.M_STS_CLAIM`.
//
// Kode `1139` ikut, karena itulah kode yang menggantikan kode asli pada klaim yang
// menunggu penutupan — dan tanpa artinya, reasuradur hanya membaca angka.
func sampleLabels() []StatusLabel {
	return []StatusLabel{
		{Code: "1139", Label: "Pending Close Claim"},
		{Code: "1147", Label: "Register"},
		{Code: "1149", Label: "Claim Committee"},
		{Code: "1163", Label: "Paid"},
	}
}

// sampleClaims adalah delapan baris klaim contoh.
//
// Nama tertanggung dan nomor polis di sini KARANGAN — data nasabah tidak pernah disalin
// ke berkas yang di-commit (`D-69`).
func sampleClaims() []Claim {
	return []Claim{
		{
			// PLA terkirim, DLA belum -> daftar PLA.
			Key: workKey("PNC-2001"), No: "PNC-2001",
			PolicyNo: "POL-2026-2001", Insured: "PT Contoh Satu",
			BusinessName: "FIRE", GroupPanel: "006",
			RegisterDate: day(2026, time.January, 5),
			LossDate:     day(2026, time.January, 2),
			PICTeknik:    "BUDI",
			WorkStatus:   "Open", StatusCode: "1147", HasWorkRow: true,
		},
		{
			// PLA dan DLA keduanya terkirim -> daftar DLA saja, BUKAN daftar PLA.
			Key: workKey("PNC-2002"), No: "PNC-2002",
			PolicyNo: "POL-2026-2002", Insured: "PT Contoh Dua",
			BusinessName: "MARINE CARGO", GroupPanel: "004",
			RegisterDate: day(2026, time.January, 6),
			LossDate:     day(2026, time.January, 3),
			PICTeknik:    "SITI",
			WorkStatus:   "Open", StatusCode: "1149", HasWorkRow: true,
		},
		{
			// Sudah selesai dan tidak menunggu penutupan -> daftar Close.
			Key: workKey("PNC-2003"), No: "PNC-2003",
			PolicyNo: "POL-2026-2003", Insured: "PT Contoh Tiga",
			BusinessName: "ANEKA", GroupPanel: "003",
			RegisterDate: day(2026, time.January, 7),
			LossDate:     day(2026, time.January, 4),
			PICTeknik:    "AGUS", CloseNote: "Selesai dibayar",
			WorkStatus: "Resolved-Completed", StatusCode: "1163", HasWorkRow: true,
		},
		{
			// PLA-nya ditandai terkirim TANPA tanggal kirim -> tidak masuk daftar mana
			// pun. Kueri lama menuntut ketiga syaratnya sekaligus.
			Key: workKey("PNC-2004"), No: "PNC-2004",
			PolicyNo: "POL-2026-2004", Insured: "PT Contoh Empat",
			BusinessName: "FIRE", GroupPanel: "006",
			RegisterDate: day(2026, time.January, 8),
			LossDate:     day(2026, time.January, 5),
			PICTeknik:    "BUDI",
			WorkStatus:   "Open", StatusCode: "1147", HasWorkRow: true,
		},
		{
			// Sudah `Resolved-Completed` TETAPI masih menunggu penutupan.
			//
			// Ia muncul di daftar DLA dengan kode status DIGANTI `1139`, dan TIDAK
			// muncul di daftar Close.
			Key: workKey("PNC-2005"), No: "PNC-2005",
			PolicyNo: "POL-2026-2005", Insured: "PT Contoh Lima",
			BusinessName: "ANEKA", GroupPanel: "009",
			RegisterDate: day(2026, time.January, 9),
			LossDate:     day(2026, time.January, 6),
			PICTeknik:    "RINA",
			WorkStatus:   "Resolved-Completed", StatusCode: "1163",
			PendingClose: true, HasWorkRow: true,
		},
		{
			// DITOLAK. Ia punya PLA terkirim, tetapi tidak muncul di daftar mana pun —
			// termasuk daftar Close, yang hanya menerima `Resolved-Completed`.
			//
			// Akibatnya reasuradur kehilangan jejak klaim yang pernah diberitahukan
			// kepadanya, tanpa satu pun pemberitahuan. Itu perilaku Pega (`P-5`).
			Key: workKey("PNC-2006"), No: "PNC-2006",
			PolicyNo: "POL-2026-2006", Insured: "PT Contoh Enam",
			BusinessName: "ANEKA", GroupPanel: "003",
			RegisterDate: day(2026, time.January, 10),
			LossDate:     day(2026, time.January, 7),
			PICTeknik:    "RINA",
			WorkStatus:   "Resolved-Rejected", StatusCode: "1142", HasWorkRow: true,
		},
		{
			// TANPA baris tabel kerja Pega.
			//
			// Ia muncul di daftar PLA — kueri lamanya memakai sub-kueri, setara LEFT
			// JOIN — tetapi TIDAK di daftar DLA, yang menggabungkannya secara INNER.
			Key: workKey("PNC-2007"), No: "PNC-2007",
			PolicyNo: "POL-2026-2007", Insured: "PT Contoh Tujuh",
			BusinessName: "FIRE", GroupPanel: "006",
			RegisterDate: day(2026, time.January, 11),
			LossDate:     day(2026, time.January, 8),
			PICTeknik:    "AGUS",
			WorkStatus:   "Open", StatusCode: "", HasWorkRow: false,
		},
		{
			// Milik login KEDUA, dan PLA-nya dikirim ke kode reasuradur yang LEBIH
			// RENDAH (`R900`, bukan `R901`).
			//
			// Ia karena itu muncul di daftar Close — yang mencocokkan seluruh kode —
			// tetapi TIDAK di daftar PLA, yang hanya mencocokkan kode tertinggi.
			// Inilah satu-satunya baris yang membuktikan perbedaan `IN` versus `=`.
			Key: workKey("PNC-2008"), No: "PNC-2008",
			PolicyNo: "POL-2026-2008", Insured: "PT Contoh Delapan",
			BusinessName: "ANEKA", GroupPanel: "003",
			RegisterDate: day(2026, time.January, 12),
			LossDate:     day(2026, time.January, 9),
			PICTeknik:    "SITI", CloseNote: "Ditutup",
			WorkStatus: "Resolved-Completed", StatusCode: "1163", HasWorkRow: true,
		},
		{
			// Lini PERSONAL ACCIDENT (`002`), TANPA satu pun pemberitahuan.
			//
			// Ia tidak akan pernah muncul di ketiga daftar pemberitahuan — keduanya
			// karena lini bisnisnya dikecualikan DAN karena tidak ada PLA maupun DLA
			// yang dikirimkan. Ia muncul HANYA di daftar komunikasi.
			//
			// Tanpa baris ini, kedua perbedaan itu tidak dapat dibuktikan: menambahkan
			// penyaring `grouppanel` ke kueri komunikasi akan lolos setiap uji.
			Key: workKey("PNC-2009"), No: "PNC-2009",
			PolicyNo: "POL-2026-2009", Insured: "PT Contoh Sembilan",
			BusinessName: "PERSONAL ACCIDENT", GroupPanel: "002",
			RegisterDate: day(2026, time.January, 13),
			LossDate:     day(2026, time.January, 10),
			PICTeknik:    "BUDI",
			WorkStatus:   "Open", StatusCode: "1147", HasWorkRow: true,
		},
	}
}

// sampleAdvices adalah dokumen PLA dan DLA contoh.
func sampleAdvices() []Advice {
	sentOn := func(d time.Time) (string, time.Time, string) {
		return "1", d, "reas@contoh.example"
	}

	sent1, date1, mail1 := sentOn(day(2026, time.January, 15))

	return []Advice{
		// PNC-2001 — hanya PLA.
		{
			ClaimKey: workKey("PNC-2001"), Kind: "pla", No: "PLA/2026/2001",
			ReinsCode: "R100", Revision: 0,
			Sent: sent1, SentDate: date1, Email: mail1,
			Type: "Treaty", Amount: "125000000.00",
			AdviceDate: day(2026, time.January, 14),
		},
		{
			// Revisi lebih tinggi -> nomor INILAH yang digambar kolom "No PLA".
			ClaimKey: workKey("PNC-2001"), Kind: "pla", No: "PLA/2026/2001-R1",
			ReinsCode: "R100", Revision: 1,
			Sent: sent1, SentDate: date1, Email: mail1,
			Type: "Treaty", Amount: "140000000.00",
			AdviceDate: day(2026, time.January, 15),
		},
		{
			// Milik reasuradur LAIN pada klaim yang SAMA.
			//
			// Ia tidak boleh terlihat di grid rincian PNC-2001. Di Pega ia justru
			// terlihat: gridnya dimuat dari objek kerja klaim, yang memuat seluruh
			// mitra beserta nilai masing-masing. Tanpa baris ini, selisih itu tidak
			// dapat dibuktikan.
			ClaimKey: workKey("PNC-2001"), Kind: "pla", No: "PLA/2026/2001-LAIN",
			ReinsCode: "R900", Revision: 0,
			Sent: sent1, SentDate: date1, Email: mail1,
			Type: "Fac Out", Amount: "99000000.00",
			AdviceDate: day(2026, time.January, 14),
		},

		// PNC-2002 — PLA dan DLA keduanya terkirim.
		{
			ClaimKey: workKey("PNC-2002"), Kind: "pla", No: "PLA/2026/2002",
			ReinsCode: "R100", Revision: 0,
			Sent: sent1, SentDate: date1, Email: mail1,
			Type: "Treaty", Amount: "75000000.00",
			AdviceDate: day(2026, time.January, 14),
		},
		{
			ClaimKey: workKey("PNC-2002"), Kind: "dla", No: "DLA/2026/2002",
			ReinsCode: "R100", Revision: 0,
			Sent: sent1, SentDate: date1, Email: mail1,
			Type: "Treaty", Amount: "70000000.00",
			AcceptanceNo: "AKS/2026/2002",
			AdviceDate:   day(2026, time.January, 16),
		},
		{
			// BELUM terkirim — ia tidak boleh muncul di grid rincian.
			//
			// Grid Pega memuatnya; di sini tidak. Lihat PlannedDifferences.
			ClaimKey: workKey("PNC-2002"), Kind: "dla", No: "DLA/2026/2002-DRAF",
			ReinsCode: "R100", Revision: 0,
			Sent: "0",
			Type: "Treaty", Amount: "70000000.00",
			AdviceDate: day(2026, time.January, 17),
		},

		// PNC-2003 — PLA terkirim; klaimnya sudah selesai.
		{
			ClaimKey: workKey("PNC-2003"), Kind: "pla", No: "PLA/2026/2003",
			ReinsCode: "R100", Revision: 0,
			Sent: sent1, SentDate: date1, Email: mail1,
		},

		// PNC-2004 — ditandai terkirim TANPA tanggal kirim dan tanpa alamat surel.
		{
			ClaimKey: workKey("PNC-2004"), Kind: "pla", No: "PLA/2026/2004",
			ReinsCode: "R100", Revision: 0,
			Sent: "1",
		},

		// PNC-2005 — DLA terkirim; klaimnya menunggu penutupan.
		{
			ClaimKey: workKey("PNC-2005"), Kind: "dla", No: "DLA/2026/2005",
			ReinsCode: "R100", Revision: 0,
			Sent: sent1, SentDate: date1, Email: mail1,
		},

		// PNC-2006 — PLA terkirim, tetapi klaimnya ditolak.
		{
			ClaimKey: workKey("PNC-2006"), Kind: "pla", No: "PLA/2026/2006",
			ReinsCode: "R100", Revision: 0,
			Sent: sent1, SentDate: date1, Email: mail1,
		},

		// PNC-2007 — PLA terkirim; klaimnya tanpa baris tabel kerja.
		{
			ClaimKey: workKey("PNC-2007"), Kind: "pla", No: "PLA/2026/2007",
			ReinsCode: "R100", Revision: 0,
			Sent: sent1, SentDate: date1, Email: mail1,
		},

		// PNC-2008 — dikirim ke kode yang LEBIH RENDAH milik login bergkode ganda.
		{
			ClaimKey: workKey("PNC-2008"), Kind: "pla", No: "PLA/2026/2008",
			ReinsCode: "R900", Revision: 0,
			Sent: sent1, SentDate: date1, Email: mail1,
		},
	}
}

// sampleXOL adalah ringkasan XOL contoh.
//
// Satu baris milik reasuradur LAIN, dan satu baris BELUM terkirim. Keduanya ada supaya
// penyaringnya benar-benar teruji — termasuk penyaring reasuradur, yang di Pega justru
// tidak mengikuti pemanggil sama sekali.
func sampleXOL() []XOL {
	return []XOL{
		{
			Kind: "PLA", ReinsCode: "R100", Year: "2026",
			CauseOfLoss: "Kebakaran", Sent: true,
			InsertDate: day(2026, time.February, 1),
		},
		{
			Kind: "PLA", ReinsCode: "R100", Year: "2026",
			CauseOfLoss: "Kebakaran", Sent: true,
			InsertDate: day(2026, time.February, 10),
		},
		{
			Kind: "DLA", ReinsCode: "R100", Year: "2025",
			CauseOfLoss: "Banjir", Sent: true,
			InsertDate: day(2025, time.December, 20),
		},
		{
			// Milik reasuradur lain — tidak boleh terlihat.
			Kind: "PLA", ReinsCode: "R900", Year: "2026",
			CauseOfLoss: "Gempa", Sent: true,
			InsertDate: day(2026, time.March, 1),
		},
		{
			// Belum terkirim — `SENDDATE IS NULL`.
			Kind: "DLA", ReinsCode: "R100", Year: "2026",
			CauseOfLoss: "Pencurian", Sent: false,
			InsertDate: day(2026, time.March, 5),
		},
	}
}

// sampleMessages adalah percakapan contoh pada `POOLDATA.M_KOMUNIKASI_PNC`.
//
// # Setiap baris menjawab satu penyaring
//
//	KOM-01  masuk, belum dijawab      -> tab "Komunikasi Masuk"
//	KOM-02  terkirim, belum dijawab   -> tab "Terkirim — Belum Dijawab"
//	KOM-03  terkirim, sudah dijawab   -> tab "Terkirim — Sudah Dijawab"
//	KOM-04  milik mitra LAIN          -> TIDAK terlihat di tab mana pun
//	KOM-05  pada klaim PA tanpa PLA   -> membuktikan daftar komunikasi tidak mengecualikan
//	                                     lini bisnis dan tidak menuntut pemberitahuan
//
// Isi pesannya KARANGAN. Percakapan nyata memuat tulisan manusia yang dapat menyebut nomor
// polis dan nama tertanggung, dan keduanya tidak pernah disalin ke berkas yang di-commit
// (`D-69`).
func sampleMessages() []Message {
	return []Message{
		{
			ID: "KOM-01", ClaimKey: workKey("PNC-2001"),
			SenderLogin: "BUDI", SenderName: "Budi Santoso",
			RecipientCode: SampleReinsurerLogin,
			Message:       "Mohon konfirmasi nilai estimasi pada PLA terlampir.",
			CreatedAt:     day(2026, time.February, 2),
			Status:        "0",
		},
		{
			ID: "KOM-02", ClaimKey: workKey("PNC-2002"),
			SenderLogin: SampleReinsurerLogin, SenderName: "Mitra Contoh",
			RecipientCode: "BUDI",
			Message:       "Kami meminta rincian perhitungan share pada DLA ini.",
			CreatedAt:     day(2026, time.February, 3),
			Status:        "0",
		},
		{
			ID: "KOM-03", ClaimKey: workKey("PNC-2002"),
			SenderLogin: SampleReinsurerLogin, SenderName: "Mitra Contoh",
			RecipientCode: "SITI",
			Message:       "Apakah dokumen pendukung sudah lengkap?",
			CreatedAt:     day(2026, time.February, 4),
			Reply:         "Sudah lengkap, terima kasih.",
			ReplierLogin:  "SITI", ReplierName: "Siti Aminah",
			RepliedAt: day(2026, time.February, 5),
			Status:    "1",
		},
		{
			// Antara petugas dan mitra LAIN. Ia tidak menyangkut pemanggil sama sekali,
			// dan karena itu tidak boleh terlihat — termasuk di layar rincian.
			ID: "KOM-04", ClaimKey: workKey("PNC-2001"),
			SenderLogin: "BUDI", SenderName: "Budi Santoso",
			RecipientCode: SampleSecondLogin,
			Message:       "Percakapan ini bukan untuk mitra contoh.",
			CreatedAt:     day(2026, time.February, 6),
			Status:        "0",
		},
		{
			// Klaim lini Personal Accident yang TIDAK punya satu pun pemberitahuan.
			ID: "KOM-05", ClaimKey: workKey("PNC-2009"),
			SenderLogin: "BUDI", SenderName: "Budi Santoso",
			RecipientCode: SampleReinsurerLogin,
			Message:       "Mohon tanggapan atas klaim kecelakaan diri ini.",
			CreatedAt:     day(2026, time.February, 7),
			Status:        "0",
		},
	}
}

// sampleDocuments adalah dokumen reasuransi contoh.
//
// # Setiap baris menjawab satu penyaring pula
//
//	DOK-01  milik pemanggil, PLA terkirim   -> terlihat
//	DOK-02  milik pemanggil, DLA terkirim   -> terlihat pada jenis DLA
//	DOK-03  `LOGIN` milik mitra LAIN        -> TIDAK terlihat  (penyaring `D-73`/Work Owner)
//	DOK-04  nomornya BELUM terkirim         -> TIDAK terlihat
//
// DOK-03 yang paling penting: ia satu-satunya yang membuktikan penyaring `T_DOC_REAS.LOGIN`
// benar-benar bekerja. `GetDokumenReas` tidak memakainya, dan penambahannya diputuskan
// Work Owner pada 2026-09-28.
func sampleDocuments() []Document {
	return []Document{
		{
			ID: "DOK-01", ClaimKey: workKey("PNC-2001"),
			AdviceNo: "PLA/2026/2001-R1", Kind: "PLA",
			Login:    SampleReinsurerLogin,
			Category: "Dokumen Klaim", SubCategory: "Laporan Kerugian",
			Name: "laporan-kerugian.pdf", MimeType: "application/pdf",
			Content: []byte("%PDF-1.4 contoh laporan kerugian"),
		},
		{
			ID: "DOK-02", ClaimKey: workKey("PNC-2002"),
			AdviceNo: "DLA/2026/2002", Kind: "DLA",
			Login:    SampleReinsurerLogin,
			Category: "Dokumen Klaim", SubCategory: "Perhitungan Akseptasi",
			Name: "perhitungan-akseptasi.pdf", MimeType: "application/pdf",
			Content: []byte("%PDF-1.4 contoh perhitungan akseptasi"),
		},
		{
			// Tercatat atas nama mitra LAIN pada nomor pemberitahuan yang sama.
			ID: "DOK-03", ClaimKey: workKey("PNC-2001"),
			AdviceNo: "PLA/2026/2001-R1", Kind: "PLA",
			Login:    SampleSecondLogin,
			Category: "Dokumen Klaim", SubCategory: "Laporan Kerugian",
			Name: "milik-mitra-lain.pdf", MimeType: "application/pdf",
			Content: []byte("%PDF-1.4 tidak boleh terlihat"),
		},
		{
			// Nomornya ada, tetapi pemberitahuannya BELUM terkirim.
			ID: "DOK-04", ClaimKey: workKey("PNC-2002"),
			AdviceNo: "DLA/2026/2002-DRAF", Kind: "DLA",
			Login:    SampleReinsurerLogin,
			Category: "Dokumen Klaim", SubCategory: "Draf",
			Name: "draf.pdf", MimeType: "application/pdf",
			Content: []byte("%PDF-1.4 draf"),
		},
	}
}

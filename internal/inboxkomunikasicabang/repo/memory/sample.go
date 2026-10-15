package memory

import "claim-pnc/internal/inboxkomunikasicabang"

// SampleRows adalah baris contoh untuk pengembangan lokal dan pengujian.
//
// # Janji yang dipegang daftar ini
//
// SETIAP penyaring punya baris yang cocok MAUPUN yang tidak. Baris yang hanya cocok
// membuktikan penyaringnya meloloskan yang benar; baris yang tidak cocok membuktikan ia
// menolak yang salah — dan penyaring yang hilang hanya ketahuan lewat yang kedua.
//
// Penyaring yang punya saksi penolak, satu per satu:
//
//	CASEID = 'CABANG'         -> KOM-0007 ber-CABANG SELESAI
//	SENDER IS NOT NULL        -> KOM-0008 tanpa pengirim
//	MESSAGE IS NOT NULL       -> KOM-0009 tanpa pesan
//	batas cabang              -> KOM-0010 milik cabang 1003
//	REPLYMESSAGE (tab)        -> tersebar di seluruh daftar
//	REPLYFROM (pencacah)      -> KOM-0006, dibalas TANPA penjawab tercatat
//
// # Tidak ada data nasabah di sini
//
// Nama tertanggung, nomor polis, dan nomor klaim TIDAK ada di tabel ini sama sekali —
// isinya percakapan, bukan klaim. Nama operator yang dipakai adalah login contoh yang sama
// dengan provider identitas tiruan, bukan nama orang sungguhan (`D-69`).
//
// # Kode cabang yang dipakai
//
//	1        kantor pusat  (inboxkomunikasicabang.HeadOfficeCode)
//	1001     cabang petugas `adminpnc`
//	1002     cabang petugas `pictekniks`
//	1003     cabang yang TIDAK dimiliki satu pun login contoh — saksi penolak
//
// Ketiganya sejalan dengan SampleBranchOfLogin di branch.go, dan itu bukan kebetulan:
// tanpa kesejajaran itu, tidak satu pun login contoh dapat melihat satu pun baris.
func SampleRows() []Row {
	return []Row{
		// ── Terlihat oleh KANTOR PUSAT (kode "1") ──────────────────────────────────

		{
			ID:              "KOM-0001",
			CaseID:          inboxkomunikasicabang.CaseOpen,
			CommunicateFrom: "1001",
			CommunicateTo:   inboxkomunikasicabang.HeadOfficeCode,
			Sender:          "pictekniks",

			// Nama pengirim TERISI, dan justru itu yang diuji: ia tidak boleh muncul di
			// layar. Kolom "Pengirim(Dari)" menampilkan kode asal, bukan nama ini.
			SenderName: "PIC Teknik Surabaya",

			Message:   "Mohon konfirmasi kelengkapan dokumen survei untuk objek kedua.",
			Status:    "0",
			CreatedAt: "2026-09-01 08:15",
		},
		{
			ID:              "KOM-0002",
			CaseID:          inboxkomunikasicabang.CaseOpen,
			CommunicateFrom: inboxkomunikasicabang.HeadOfficeCode,
			CommunicateTo:   "1001",
			Sender:          "adminpnc",
			SenderName:      "Admin PNC Pusat",
			Message:         "Dokumen sudah kami terima, mohon tunggu proses akseptasi.",
			ReplyMessage:    "Baik, kami tunggu kabarnya.",
			ReplyFrom:       "pictekniks",
			ReplyFromName:   "PIC Teknik Surabaya",
			Status:          "1",
			CreatedAt:       "2026-09-02 09:30",
			RepliedAt:       "2026-09-03 10:05",

			Attachments: []inboxkomunikasicabang.Attachment{
				{
					DocumentID: "DOC-11001",
					TypeName:   "DOKUMEN KLAIM",
					DetailName: "Laporan Survei",
					Note:       "Revisi kedua",
					UploadedAt: "2026-09-03 10:07",
				},
				{
					// Lampiran yang BELUM diunggah — tanggalnya kosong.
					//
					// Ia saksi bahwa Attachment.Uploaded benar-benar membedakan keduanya.
					// Tanpa baris ini, penanda yang selalu bernilai sama tetap lulus uji.
					DocumentID: "DOC-11002",
					TypeName:   "DOKUMEN KLAIM",
					DetailName: "Foto Objek",
					Note:       "Menunggu kiriman cabang",
				},
			},
		},
		{
			// Percakapan yang tanggal pesannya PALING LAMA di antara yang belum dijawab.
			//
			// Ia yang harus berada di baris TERATAS tab "Belum Dijawab" — dan itulah yang
			// membuktikan urutan menaiknya benar-benar berlaku.
			ID:              "KOM-0003",
			CaseID:          inboxkomunikasicabang.CaseOpen,
			CommunicateFrom: "1002",
			CommunicateTo:   inboxkomunikasicabang.HeadOfficeCode,
			Sender:          "pictekniks",
			SenderName:      "PIC Teknik Bandung",
			Message:         "Apakah nilai estimasi sudah dapat dinaikkan ke komite?",
			Status:          "0",
			CreatedAt:       "2026-08-28 14:40",
		},

		// ── Terlihat oleh CABANG 1001 ──────────────────────────────────────────────

		{
			// Balasan TERBARU di antara yang sudah dijawab pada cabang 1001.
			//
			// Ia yang harus berada di baris TERATAS tab "Sudah Dijawab" — pasangan dari
			// KOM-0003, dan bersamanya ia membuktikan kedua arah urutan memang berlawanan.
			ID:              "KOM-0004",
			CaseID:          inboxkomunikasicabang.CaseOpen,
			CommunicateFrom: inboxkomunikasicabang.HeadOfficeCode,
			CommunicateTo:   "1001",
			Sender:          "adminpnc",
			SenderName:      "Admin PNC Pusat",
			Message:         "Mohon lengkapi berita acara kerugian.",
			ReplyMessage:    "Berita acara sudah diunggah hari ini.",
			ReplyFrom:       "pictekniks",
			ReplyFromName:   "PIC Teknik Surabaya",
			Status:          "1",
			CreatedAt:       "2026-09-05 11:00",
			RepliedAt:       "2026-09-12 16:20",
		},
		{
			ID:              "KOM-0005",
			CaseID:          inboxkomunikasicabang.CaseOpen,
			CommunicateFrom: "1001",
			CommunicateTo:   inboxkomunikasicabang.HeadOfficeCode,
			Sender:          "pictekniks",
			SenderName:      "PIC Teknik Surabaya",
			Message:         "Tertanggung menanyakan perkiraan tanggal pembayaran.",
			Status:          "0",
			CreatedAt:       "2026-09-10 07:50",
		},
		{
			// DIBALAS, tetapi penjawabnya TIDAK tercatat.
			//
			// Ia saksi selisih satu kolom antara grid dan pencacah: baris ini MUNCUL di tab
			// "Sudah Dijawab" (penyaringnya hanya `REPLYMESSAGE`) tetapi TIDAK terhitung di
			// pencacah mana pun — bukan di "Answered" karena `REPLYFROM` kosong, bukan pula
			// di "Not Answered" karena `REPLYMESSAGE` terisi.
			//
			// Tanpa baris ini, selisih itu tidak dapat dibuktikan ada.
			ID:              "KOM-0006",
			CaseID:          inboxkomunikasicabang.CaseOpen,
			CommunicateFrom: inboxkomunikasicabang.HeadOfficeCode,
			CommunicateTo:   "1001",
			Sender:          "adminpnc",
			SenderName:      "Admin PNC Pusat",
			Message:         "Mohon kirim ulang rincian biaya perbaikan.",
			ReplyMessage:    "Rincian menyusul.",
			Status:          "1",
			CreatedAt:       "2026-09-06 13:15",
			RepliedAt:       "2026-09-07 08:00",
		},

		// ── Saksi PENOLAK — tidak boleh muncul di layar mana pun ───────────────────

		{
			// Percakapan yang SUDAH DITUTUP lewat tombol "Selesai Komunikasi".
			//
			// Nilainya berawalan `CABANG`, dan justru itulah gunanya: penyaring yang
			// ditulis sebagai awalan alih-alih perbandingan persis akan meloloskannya.
			ID:              "KOM-0007",
			CaseID:          inboxkomunikasicabang.CaseClosed,
			CommunicateFrom: "1001",
			CommunicateTo:   inboxkomunikasicabang.HeadOfficeCode,
			Sender:          "pictekniks",
			SenderName:      "PIC Teknik Surabaya",
			Message:         "Sudah selesai, terima kasih.",
			Status:          "2",
			CreatedAt:       "2026-08-20 10:00",
		},
		{
			// Tanpa PENGIRIM — tertolak `SENDER IS NOT NULL`.
			ID:              "KOM-0008",
			CaseID:          inboxkomunikasicabang.CaseOpen,
			CommunicateFrom: "1001",
			CommunicateTo:   inboxkomunikasicabang.HeadOfficeCode,
			Message:         "Pesan tanpa pengirim tercatat.",
			Status:          "0",
			CreatedAt:       "2026-09-04 09:00",
		},
		{
			// Tanpa PESAN — tertolak `MESSAGE IS NOT NULL`.
			ID:              "KOM-0009",
			CaseID:          inboxkomunikasicabang.CaseOpen,
			CommunicateFrom: "1001",
			CommunicateTo:   inboxkomunikasicabang.HeadOfficeCode,
			Sender:          "pictekniks",
			SenderName:      "PIC Teknik Surabaya",
			Status:          "0",
			CreatedAt:       "2026-09-04 09:30",
		},
		{
			// Milik cabang 1003, yang TIDAK dimiliki satu pun login contoh.
			//
			// Ia saksi batas cabang: petugas cabang 1001 maupun 1002 tidak boleh melihatnya,
			// dan kantor pusat pun tidak — karena asal maupun tujuannya bukan `1`.
			ID:              "KOM-0010",
			CaseID:          inboxkomunikasicabang.CaseOpen,
			CommunicateFrom: "1003",
			CommunicateTo:   "1004",
			Sender:          "petugaslain",
			SenderName:      "Petugas Cabang Lain",
			Message:         "Percakapan milik cabang lain.",
			Status:          "0",
			CreatedAt:       "2026-09-08 08:00",
		},
	}
}

// SampleHistory adalah baris contoh `POOLDATA.M_KOMUNIKASI_CABANG` — UTAS percakapan.
//
// # Kenapa ia ada sejak 2026-09-24
//
// Karena layar detail membaca tabel ini, bukan tabel percakapan. Penyimpanan tanpa riwayat
// akan menampilkan layar detail yang KOSONG untuk setiap percakapan contoh — dan uji yang
// memakainya gagal karena alasan yang tidak ada hubungannya dengan yang diujinya.
//
// # Kesejajaran yang dijaga
//
// Setiap baris di sini menunjuk percakapan yang BENAR-BENAR ada di SampleRows, dan isinya
// sejalan: percakapan yang sudah dijawab punya DUA ucapan — pesan lalu balasan — sementara
// yang belum dijawab punya satu.
//
// Tanggalnya pula dijaga sejalan: ucapan pertama bertanggal sama dengan `CreatedAt` barisnya,
// ucapan kedua dengan `RepliedAt`. Riwayat yang tanggalnya menyimpang dari kepalanya akan
// membuat utas terbaca dengan urutan yang tidak masuk akal.
//
// # SATU percakapan sengaja TIDAK punya riwayat
//
// KOM-0003 punya kepala tetapi tidak satu pun baris di sini. Ia saksi keadaan yang nyata di
// produksi: percakapan yang dibuat lewat jalur lain, atau data warisan sebelum tabel riwayat
// dipakai, punya kepala tanpa utas.
//
// Tanpa saksi itu, "utas kosong" dan "percakapan tidak ada" tidak dapat dibuktikan berbeda —
// dan menyamakan keduanya akan menjawab "tidak ditemukan" untuk percakapan yang nyata.
func SampleHistory() []ReplyHistory {
	return []ReplyHistory{
		{
			ConversationID: "KOM-0001",
			Sender:         "pictekniks",
			Message:        "Mohon konfirmasi kelengkapan dokumen survei untuk objek kedua.",
			Channel:        "1",
			CreatedAt:      "2026-09-01 08:15",
		},

		// KOM-0002 — DUA ucapan: pesannya, lalu balasannya.
		{
			ConversationID: "KOM-0002",
			Sender:         "adminpnc",
			Message:        "Dokumen sudah kami terima, mohon tunggu proses akseptasi.",
			Channel:        "1001",
			CreatedAt:      "2026-09-02 09:30",
		},
		{
			ConversationID: "KOM-0002",
			Sender:         "pictekniks",
			Message:        "Baik, kami tunggu kabarnya.",
			Channel:        inboxkomunikasicabang.CaseOpen,
			CreatedAt:      "2026-09-03 10:05",
		},

		// KOM-0004 — dua ucapan pula.
		{
			ConversationID: "KOM-0004",
			Sender:         "adminpnc",
			Message:        "Mohon lengkapi berita acara kerugian.",
			Channel:        "1001",
			CreatedAt:      "2026-09-05 11:00",
		},
		{
			ConversationID: "KOM-0004",
			Sender:         "pictekniks",
			Message:        "Berita acara sudah diunggah hari ini.",
			Channel:        inboxkomunikasicabang.CaseOpen,
			CreatedAt:      "2026-09-12 16:20",
		},

		{
			ConversationID: "KOM-0005",
			Sender:         "pictekniks",
			Message:        "Tertanggung menanyakan perkiraan tanggal pembayaran.",
			Channel:        "1",
			CreatedAt:      "2026-09-10 07:50",
		},

		{
			ConversationID: "KOM-0006",
			Sender:         "adminpnc",
			Message:        "Mohon kirim ulang rincian biaya perbaikan.",
			Channel:        "1001",
			CreatedAt:      "2026-09-06 13:15",
		},
		{
			ConversationID: "KOM-0006",
			Sender:         "adminpnc",
			Message:        "Rincian menyusul.",
			Channel:        inboxkomunikasicabang.CaseOpen,
			CreatedAt:      "2026-09-07 08:00",
		},
	}
}

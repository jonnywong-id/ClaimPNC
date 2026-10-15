package memory

import "claim-pnc/internal/daftartipedokumenbisnis"

// # PERINGATAN — INI BUKAN DATA PRODUKSI
//
// Seluruh baris di berkas ini adalah SUSUNAN SENDIRI. Isi
// POOLDATA.LST_TYPE_DOC_BUSINESS yang sebenarnya tidak ada di export Pega dan belum
// pernah diminta ke DBA.
//
// Akibatnya mengikat: berkas ini TIDAK BOLEH dipakai sebagai dasar uji kesetaraan gerbang
// 1 (`D-42`). Ia hanya menghidupkan layar saat pengembangan tanpa Oracle, dan satu-satunya
// hal yang dibuktikannya adalah bahwa layarnya berfungsi — bukan bahwa hasilnya sama
// dengan Pega.
//
// Yang DAPAT dipercaya dari susunan ini hanyalah BENTUKNYA, dan bentuk itu memang dibaca
// dari export:
//
//   - keenam nilai tahap dokumen, dari keenam kueri unggah
//   - ID berawalan kode situs diikuti empat digit, dari `PEGA_LST_DET_TYPE_DOC_BUSINESS.prc:22`
//   - STS_WAJIB berupa teks '1'/'0', dari pembandingan berkutip di keenam kueri itu

// SampleList mengembalikan contoh aturan dokumen untuk pengembangan lokal.
//
// Empat keadaan sengaja diwakili, dan masing-masing pernah menjadi sebab cacat di modul
// lain:
//
//   - baris wajib YANG PUNYA jaminan — satu-satunya bentuk yang benar-benar wajib di klaim
//   - baris wajib TANPA jaminan — tampak wajib di layar ini, tetapi tidak pernah wajib di
//     klaim; lihat kepala paket domain
//   - baris ber-DETAIL_DOKUMEN "-" — tersembunyi dari seluruh layar unggah
//   - baris tanpa objek dokumen — kolomnya memang boleh kosong
func SampleList() []daftartipedokumenbisnis.DocumentRule {
	return []daftartipedokumenbisnis.DocumentRule{
		{
			ID:               "10001",
			BusinessID:       "001",
			BusinessName:     "Fire",
			DocumentTypeID:   "20001",
			DocumentTypeName: "REGISTER",
			ObjectDocID:      "30001",
			ObjectDocName:    "Bangunan",
			DetailTypeDocID:  "40001",
			DetailDocument:   "Laporan Kerugian",
			Mandatory:        true,
			MinDocument:      1,
			Coverages: []daftartipedokumenbisnis.Coverage{
				{ID: "10009"},
			},
		},
		{
			ID:               "10002",
			BusinessID:       "001",
			BusinessName:     "Fire",
			DocumentTypeID:   "20001",
			DocumentTypeName: "REGISTER",
			DetailTypeDocID:  "40002",
			DetailDocument:   "Foto Lokasi Kejadian",
			Mandatory:        true,
			MinDocument:      3,
		},
		{
			ID:               "10003",
			BusinessID:       "001",
			BusinessName:     "Fire",
			DocumentTypeID:   "20003",
			DocumentTypeName: "SURVEY",
			DetailTypeDocID:  "40003",
			DetailDocument:   "-",
			Mandatory:        false,
			MinDocument:      0,
		},
		{
			ID:               "10004",
			BusinessID:       "004",
			BusinessName:     "Marine Cargo",
			DocumentTypeID:   "20002",
			DocumentTypeName: "COMMITEE",
			ObjectDocID:      "30002",
			ObjectDocName:    "Kargo",
			DetailTypeDocID:  "40004",
			DetailDocument:   "Bill of Lading",
			Mandatory:        true,
			MinDocument:      1,
		},
	}
}

// SampleBusinessList mengembalikan contoh lini bisnis.
//
// Memuat satu bisnis yang BELUM punya aturan dokumen sama sekali ("005", Travel). Ia ada
// dengan sengaja: bisnis seperti itulah yang paling sering dituju layar Tambah, dan tanpa
// satu pun di contoh, cacat "daftar pilihan hanya memuat bisnis yang sudah terisi" tidak
// akan pernah terlihat saat pengembangan.
// Satu kode lini MBU — `10028`, Personal Accident — sengaja ikut. Ia salah satu dari lima
// yang dilewati tombol "Pilih semua" (`Activity/SetAllBusiness-Act.xml:984`), dan tanpa
// satu pun di contoh, pengecualiannya tidak pernah teruji saat pengembangan.
//
// Perhatikan `ExcludedFromBulkSelect` TIDAK diisi di sini: penandanya dipasang lapisan
// usecase dari konfigurasi, bukan disimpan sebagai isi tabel.
func SampleBusinessList() []daftartipedokumenbisnis.Business {
	return []daftartipedokumenbisnis.Business{
		{ID: "001", Name: "Fire"},
		{ID: "003", Name: "Aneka"},
		{ID: "004", Name: "Marine Cargo"},
		{ID: "005", Name: "Travel"},
		{ID: "10028", Name: "Personal Accident"},
	}
}

// SampleDocumentTypeList mengembalikan keenam tahap dokumen.
//
// Keenamnya BUKAN karangan — masing-masing dibaca dari kueri unggah yang memakainya:
//
//	REGISTER             RDB List/BrowseRegister_upload-SQL.xml
//	SURVEY               RDB List/BrowseSurvey_upload-SQL.xml
//	COMMITEE             RDB List/BrowseCommitee_upload-SQL.xml
//	PAYMENT              RDB List/BrowsePayment_upload-SQL.xml
//	SALVAGE              RDB List/BrowseSalvage_upload-SQL.xml
//	COLLECTING DOCUMENT  RDB List/BrowseCollectingDoc_upload-SQL.xml
//
// Ejaan "COMMITEE" dengan satu T dipertahankan. Ia salah ketik yang ada DI DALAM DATA, dan
// memperbaikinya di sini akan membuat contoh tidak lagi cocok dengan kueri yang mencarinya
// — kueri itu mencari teks persis. Perbaikannya, bila dikehendaki, menempuh `D-63`.
func SampleDocumentTypeList() []daftartipedokumenbisnis.Reference {
	return []daftartipedokumenbisnis.Reference{
		{ID: "20001", Name: "REGISTER"},
		{ID: "20002", Name: "COMMITEE"},
		{ID: "20003", Name: "SURVEY"},
		{ID: "20004", Name: "PAYMENT"},
		{ID: "20005", Name: "SALVAGE"},
		{ID: "20006", Name: "COLLECTING DOCUMENT"},
	}
}

// SampleObjectDocList mengembalikan contoh objek dokumen.
func SampleObjectDocList() []daftartipedokumenbisnis.Reference {
	return []daftartipedokumenbisnis.Reference{
		{ID: "30001", Name: "Bangunan"},
		{ID: "30002", Name: "Kargo"},
		{ID: "30003", Name: "Orang"},
	}
}

// SampleDetailTypeDocList mengembalikan contoh rincian dokumen.
//
// Masternya dimiliki modul Daftar Detail Tipe Dokumen, MENU_ID 41. Contoh di sini
// TERSENDIRI dan tidak diambil dari modul itu — setiap modul mendeklarasikan seam-nya
// sendiri, sehingga contohnya pun tidak saling mengimpor.
//
// ParentID SENGAJA terisi pada setiap baris, dan tersebar ke beberapa tahap. Ia yang
// membuat penyempitan pilihan Detail Dokumen benar-benar teruji saat pengembangan: bila
// seluruh contoh bernaung pada satu tahap, daftar yang tidak menyempit dan daftar yang
// menyempit akan tampak sama persis di layar.
func SampleDetailTypeDocList() []daftartipedokumenbisnis.Reference {
	return []daftartipedokumenbisnis.Reference{
		{ID: "40001", Name: "Laporan Kerugian", ParentID: "20001"},
		{ID: "40002", Name: "Foto Lokasi Kejadian", ParentID: "20001"},
		{ID: "40003", Name: "Berita Acara Survei", ParentID: "20003"},
		{ID: "40004", Name: "Bill of Lading", ParentID: "20002"},
		{ID: "40005", Name: "Kuitansi Pembayaran", ParentID: "20004"},
	}
}

// Package memory memenuhi seam inboxrclpucl.Repo dengan penyimpanan di memori.
//
// # Untuk apa ia ada
//
// Dua hal, dan keduanya nyata:
//
//   - Pengujian aturan modul TANPA basis data, sehingga uji aturan bisnis berjalan cepat
//     dan tidak menuntut Oracle (`14-TESTING-STRATEGY.md` §3).
//   - Pengembangan lokal saat variabel `PENYIMPANAN` tidak menunjuk basis data mana pun.
//
// # Kenapa penyaringnya ditiru, bukan disederhanakan
//
// Karena kalau tidak, uji yang lulus di sini tidak menyatakan apa pun tentang yang berjalan
// di Oracle. Keenam penyaring ditiru sedekat-dekatnya dengan predikat SQL-nya: kelas objek
// kerja, akun antrean bersama, status kerja, tanggal cetak surat, penanda persetujuan, dan
// penanda jalur MSIG.
//
// # Satu penyaring yang ditiru justru karena ia TERASA seperti cacat
//
// `PUCLAPPROVE_1 <> '1'` tidak menangkap nilai kosong — `NULL <> '1'` menghasilkan UNKNOWN
// di Oracle maupun PostgreSQL, bukan TRUE. Penyimpanan ini meniru perilaku yang sama persis
// (lihat matchesApproval), dan baris contohnya memuat satu klaim yang TERTOLAK karenanya.
//
// Menyederhanakannya menjadi "yang penting bukan '1'" akan membuat uji lulus untuk perilaku
// yang TIDAK terjadi di Oracle — dan justru penyaring inilah yang paling mungkin
// menyembunyikan baris di produksi.
package memory

import (
	"context"
	"sort"
	"strings"
	"time"

	"claim-pnc/internal/inboxrclpucl"
)

// Row adalah satu baris contoh beserta kolom yang TIDAK ditampilkan tetapi menyaring,
// mengurutkan, atau dipakai laporan.
//
// Isian tambahan tidak masuk inboxrclpucl.WorkItem dengan sengaja: tidak satu pun sampai ke
// layar, dan menaruhnya di tipe domain akan membuat orang menduga semuanya bagian dari
// kontrak.
type Row struct {
	Item inboxrclpucl.WorkItem

	// TrackCode adalah kode jalur MENTAH — `RCL_PUCL_1`, berisi "1" atau "2".
	//
	// Ia disimpan MENTAH, bukan sudah diterjemahkan, justru supaya penerjemahnya ikut
	// teruji. Item.Track diisi dari sini lewat inboxrclpucl.TrackOf saat baris diambil —
	// menyimpannya sudah jadi akan membuat penerjemah yang salah tetap lulus uji.
	TrackCode string

	// WorkClass adalah kelas objek kerja — `PXOBJCLASS`.
	//
	// Satu tabel Pega menampung beberapa kelas sekaligus, dan tanpa penyaring ini
	// penyimpanan memori tidak dapat membuktikan penyaringnya benar-benar dipakai.
	WorkClass string

	// AssignedOperator adalah pemegang penugasan — `ASSIGNED_OPERATOR_ID`.
	//
	// Di Pega ia nama AKUN antrean bersama (`RCLPUCL`). Di `TC_PNC_PUCL` yang berjalan ia
	// berisi NAMA ORANG, dan apa yang SEHARUSNYA diisi masih pertanyaan terbuka. Ia tidak
	// lagi menyaring apa pun; dipertahankan karena ia bagian dari bentuk barisnya.
	AssignedOperator string

	// OutsideFlatTable menandai baris yang TIDAK akan ada di `POOLDATA.TC_PNC_PUCL`.
	//
	// # Kenapa penanda, bukan disimpulkan dari AssignedOperator
	//
	// Karena keanggotaan tabel datar ditentukan PROSES PENGISI, bukan oleh nilai kolom mana
	// pun. Menyimpulkannya dari nama antrean berarti menghidupkan kembali penyaring yang
	// baru saja dihapus — dan penyaring itu dihapus justru karena kolomnya tidak menyatakan
	// antrean.
	//
	// Dua baris contoh memakainya, dan keduanya tetap berguna: klaim di antrean LAIN, dan
	// klaim Personal Accident di luar antrean. Keduanya tidak muncul di tab mana pun karena
	// memang tidak ada di tabelnya — tetapi yang kedua TETAP ikut laporan harian, yang
	// membaca tabel Pega dan bukan tabel datar.
	OutsideFlatTable bool

	// WorkStatus adalah status kerja — `PYSTATUSWORK`.
	WorkStatus string

	// ExpiryCaseStatus adalah `STATUSCASE_1` — penyaring tab Cetak Surat.
	//
	// Ia BUKAN Item.ExpiryStatus, yang digambar layar sebagai "Status Kadaluarsa" dan
	// berasal dari `STATUSKLAIM_1`. Keduanya sengaja dipisah di sini supaya perangkap
	// penamaan itu ikut teruji: kolom yang MENYARING dan kolom yang DIGAMBAR memang
	// berbeda.
	ExpiryCaseStatus string

	// PUCLApprove adalah `PUCLAPPROVE_1` — penyaring tab Kelengkapan Dokumen dan Klaim
	// MSIG. Kosong berarti kolomnya NULL di basis data.
	PUCLApprove string

	// MSIG adalah `MSIG_1` — penanda jalur MSIG. Kosong berarti NULL.
	//
	// Di produksi kolom ini tampaknya TIDAK PERNAH terisi; lihat catatan di kepala paket
	// inboxrclpucl. Di sini ia sengaja diisi pada satu baris contoh, supaya tab Klaim MSIG
	// punya sesuatu untuk dibuktikan.
	MSIG string

	// GroupPanel adalah `GROUPPANEL_1` — dipakai cabang kedua laporan harian, DAN sejak
	// 2026-10-01 menentukan tombol "Kirim Ke Analyst" (PA) versus "Kirim ke PIC Teknik"
	// (Travel) di layar kerja.
	GroupPanel string

	// ClaimStatus adalah `STATUSCLAIM_1` — Status Klaim ber-33 kode `1134`–`1166`
	// (`R-06`), digambar HANYA di laporan harian.
	//
	// # Kenapa ia isian tersendiri dan bukan Item.ExpiryStatus
	//
	// Karena keduanya kolom yang BERBEDA pada tabel yang sama, dan namanya hanya berbeda
	// satu huruf: `STATUSCLAIM_1` di sini, `STATUSKLAIM_1` pada isian yang digambar grid
	// sebagai "Status Kadaluarsa". Menukarnya tidak menghasilkan satu pun galat, dan
	// memisahkannya di sini membuat tertukarnya dapat tertangkap uji.
	ClaimStatus string

	// CreatedAt adalah waktu objek kerja dibuat — `PXCREATEDATETIME`.
	//
	// Ia dasar pengurutan ketiga kueri daftar, dan TERPISAH dari `Item.InboxEntryAt` yang
	// digambar layar. Pemisahan itu bukan kehalusan: keduanya memang kolom yang berbeda,
	// dan justru karena urutannya memakai kolom yang TIDAK terlihat, tabel dapat terbaca
	// tidak urut. Menyamakan keduanya di sini akan menyembunyikan hal itu dari uji.
	CreatedAt time.Time

	// LossDate adalah `DATEOFLOSS_1` — Tanggal Kejadian, digambar layar kerja.
	//
	// Ia tidak ada di grid mana pun, sehingga tidak masuk WorkItem.
	LossDate string

	// FirstObjectName meniru `ObjectList(1).ObjectName` —
	// `POOLDATA.T_CLAIM_OBJECTLIST.OBJECTNAME` pada objek ber-`OBJECTID` terkecil.
	//
	// Ia mengisi DUA isian layar kerja sekaligus, "Nama Peserta" DAN "UP", karena
	// `SetDataLampiranSuratRCLPUCL_Act` memang menunjuk ekspresi yang sama untuk keduanya.
	FirstObjectName string

	// FirstProposeValue meniru `ObjectList(1).ObjectCoverageList(1).AdjustmentList(1)
	// .ProposeValue` — `POOLDATA.T_CLAIM_ADJUSTMENT.PROPOSE_VALUE`.
	FirstProposeValue string

	// Subject, OpeningNote, BodyNote, ClosingNote meniru `PERIHAL` dan ketiga `KETERANGAN`
	// pada `TC_PNC_PUCL`. Keempatnya dulu bertanda "di clipboard Pega"; kolomnya ditemukan
	// ADA pada 2026-10-01.
	Subject     string
	OpeningNote string
	BodyNote    string
	ClosingNote string

	// PUCLNote adalah `KOMENTARPUCL_1` — satu-satunya isian bagian "Penerimaan Dokumen"
	// yang punya kolom terverifikasi.
	PUCLNote string

	// SentAt adalah `TANGGALKIRIMPUCL_1` sebagai waktu — dasar penyaring dan pengurutan
	// LAPORAN HARIAN.
	//
	// Ia terpisah dari `Item.InboxEntryAt`, yang menyimpan kolom yang sama sebagai TEKS
	// apa adanya untuk digambar. Bentuk teks kolomnya tidak diketahui (`R-08`), sehingga
	// menyaring di atas teks akan menguji hal yang berbeda dari yang dilakukan Oracle.
	SentAt time.Time
}

// Store adalah penyimpanan antrean di memori.
//
// Ia tidak dilindungi mutex karena tidak pernah berubah setelah dibentuk: modul ini hanya
// membaca, dan tidak ada satu pun operasi yang menulis.
type Store struct {
	rows []Row
}

// NewStore membentuk penyimpanan berisi baris yang diberikan.
func NewStore(rows ...Row) *Store {
	return &Store{rows: rows}
}

// NewSampleStore membentuk penyimpanan berisi contoh bawaan.
func NewSampleStore() *Store {
	return NewStore(SampleRows()...)
}

// List mengembalikan satu halaman baris yang lolos penyaring beserta jumlah seluruhnya.
func (s *Store) List(
	_ context.Context,
	q inboxrclpucl.Query,
	page inboxrclpucl.Pagination,
) (inboxrclpucl.Page, error) {
	type ordered struct {
		item inboxrclpucl.WorkItem
		at   time.Time
		id   string
	}

	matched := []ordered{}

	for _, candidate := range s.rows {
		if !matchesWorkClass(candidate) {
			continue
		}
		if candidate.OutsideFlatTable {
			// Bukan di tabel datar, sehingga ketiga tab tidak dapat melihatnya. Ini
			// menggantikan penyaring antrean yang dulu mengeluarkannya.
			continue
		}
		if !matchesWorkStatus(candidate) {
			continue
		}
		if !matchesLetterPrinted(candidate, q) {
			continue
		}
		if !matchesExpiryCaseStatus(candidate, q) {
			continue
		}
		if !matchesApproval(candidate, q) {
			continue
		}
		if !matchesMSIG(candidate, q) {
			continue
		}

		matched = append(matched, ordered{
			item: decorate(candidate),
			at:   candidate.CreatedAt,
			id:   candidate.Item.CaseID,
		})
	}

	// Urutan ditetapkan supaya paginasi di atasnya stabil: yang terbaru masuk lebih dulu,
	// persis seperti `ORDER BY w.PXCREATEDATETIME DESC, w.PYID DESC` pada ketiga kueri.
	// Perhatikan pemutus serinya pun MENURUN, mengikuti `PYID DESC` — bukan menaik.
	sort.SliceStable(matched, func(i, j int) bool {
		left, right := matched[i].at, matched[j].at
		if !left.Equal(right) {
			return left.After(right)
		}
		return matched[i].id > matched[j].id
	})

	items := make([]inboxrclpucl.WorkItem, 0, len(matched))
	for _, row := range matched {
		items = append(items, row.item)
	}

	return inboxrclpucl.Slice(items, page), nil
}

// DailyReport mengembalikan satu halaman laporan harian pada rentang tanggal.
//
// # Kenapa penyaringnya JAUH lebih longgar daripada List
//
// Karena kuerinya memang begitu. `GetDataPUCLRCLForDailyReport-SQL.xml` tidak menyaring
// status kerja, tidak menyaring tanggal cetak surat, dan tidak menyaring penanda
// persetujuan — dan cabang keduanya bahkan tidak menggabung tabel antrean bersama sama
// sekali. Menyaringnya seperti List akan membuat uji lulus untuk laporan yang isinya
// berbeda dari yang dihasilkan Oracle.
func (s *Store) DailyReport(
	_ context.Context,
	rng inboxrclpucl.DateRange,
	page inboxrclpucl.Pagination,
) ([]inboxrclpucl.DailyReportRow, int, error) {
	from, to, ok := parseRange(rng)
	if !ok {
		// Rentang yang tidak terbaca seharusnya sudah ditolak NewReportRequest. Sampai di
		// sini, mengembalikan kosong lebih jujur daripada mengembalikan seluruh baris.
		return []inboxrclpucl.DailyReportRow{}, 0, nil
	}

	type ordered struct {
		row inboxrclpucl.DailyReportRow
		at  time.Time
		id  string
	}

	// `UNION`, bukan `UNION ALL`: baris yang memenuhi KEDUA cabang muncul sekali saja.
	// Klaim Personal Accident yang berada di antrean RCL/PUCL memenuhi keduanya, dan
	// inilah satu-satunya tempat perbedaan itu terlihat.
	seen := map[string]bool{}
	matched := []ordered{}

	for _, candidate := range s.rows {
		if candidate.WorkClass != inboxrclpucl.WorkClassClaim {
			continue
		}
		if !withinRange(candidate.SentAt, from, to) {
			continue
		}

		// Cabang pertama: berada di antrean bersama RCL/PUCL.
		// Cabang kedua: ber-Group Panel Personal Accident, tanpa melihat antreannya.
		inWorkbasket := strings.EqualFold(
			candidate.AssignedOperator, inboxrclpucl.RCLPUCLWorkbasket)
		isPA := strings.TrimSpace(candidate.GroupPanel) == inboxrclpucl.GroupPanelPA

		if !inWorkbasket && !isPA {
			continue
		}
		if seen[candidate.Item.Reference] {
			continue
		}
		seen[candidate.Item.Reference] = true

		matched = append(matched, ordered{
			row: reportRowOf(candidate),
			at:  candidate.SentAt,
			id:  candidate.Item.CaseID,
		})
	}

	// `ORDER BY r.SENT_AT DESC, r.CASE_ID DESC` pada kueri.
	sort.SliceStable(matched, func(i, j int) bool {
		left, right := matched[i].at, matched[j].at
		if !left.Equal(right) {
			return left.After(right)
		}
		return matched[i].id > matched[j].id
	})

	clean := page.Normalize()
	total := len(matched)

	offset := clean.Offset()
	if offset >= total {
		return []inboxrclpucl.DailyReportRow{}, total, nil
	}
	end := offset + clean.Size
	if end > total {
		end = total
	}

	rows := make([]inboxrclpucl.DailyReportRow, 0, end-offset)
	for _, row := range matched[offset:end] {
		rows = append(rows, row.row)
	}
	return rows, total, nil
}

// Detail mengembalikan isi layar kerja RCL/PUCL untuk satu klaim.
//
// # Kenapa ia meniru penyaring kelas objek kerja pula
//
// Karena kueri SQL-nya begitu, dan penyaring itu menutup kelas kekeliruan yang tidak
// menghasilkan galat: kunci milik kelas lain mengembalikan baris berkolom PUCL kosong yang
// terbaca seperti klaim yang belum diisi.
//
// # Kenapa ia TIDAK menyaring antrean maupun status kerja
//
// Karena kuerinya juga tidak. Layar kerja dibuka dengan KUNCI, bukan lewat antrean — dan
// klaim yang sudah berpindah antrean sejak daftarnya dimuat tetap harus dapat dibuka.
func (s *Store) Detail(
	_ context.Context,
	reference string,
) (inboxrclpucl.ClaimDetail, error) {
	wanted := strings.TrimSpace(reference)

	for _, candidate := range s.rows {
		if candidate.WorkClass != inboxrclpucl.WorkClassClaim {
			continue
		}
		if candidate.Item.Reference != wanted {
			continue
		}
		return detailOf(candidate), nil
	}

	return inboxrclpucl.ClaimDetail{}, inboxrclpucl.ErrClaimNotFound
}

// sampleDocuments adalah dokumen contoh untuk SETIAP klaim yang dikenal penyimpanan memori.
//
// Isinya teks pendek, bukan PDF palsu. Yang diuji di atas memori adalah ALUR-nya — daftar
// tergambar, kepemilikan ditegakkan, berkasnya terserah dengan nama dan jenis yang benar —
// dan berkas biner palsu tidak menambah satu pun jawaban atas pertanyaan itu.
var sampleDocuments = []inboxrclpucl.Document{
	{
		ID:          "DOC-0001",
		Name:        "Surat Keterangan.pdf",
		MimeType:    "application/pdf",
		Category:    "Dokumen Klaim",
		SubCategory: "Surat Keterangan",
		UploadedAt:  "2026-09-02 09:15:00",
		UploadedBy:  "PETUGASCONTOH",
	},
	{
		ID:          "DOC-0002",
		Name:        "Kuitansi.jpg",
		MimeType:    "image/jpeg",
		Category:    "Dokumen Klaim",
		SubCategory: "Kuitansi",
		UploadedAt:  "2026-09-03 14:40:00",
		UploadedBy:  "PETUGASCONTOH",
	},
}

// Documents mengembalikan dokumen contoh milik satu klaim.
//
// Klaim yang TIDAK dikenal mengembalikan ErrClaimNotFound, bukan senarai kosong. Keduanya
// berbeda: yang pertama kekeliruan pemanggil, yang kedua klaim tanpa dokumen — dan hanya yang
// pertama yang perlu diperbaiki.
func (s *Store) Documents(
	_ context.Context,
	caseNumber string,
) ([]inboxrclpucl.Document, error) {
	wanted := strings.TrimSpace(caseNumber)

	for _, candidate := range s.rows {
		if candidate.WorkClass != inboxrclpucl.WorkClassClaim {
			continue
		}
		if candidate.Item.CaseID != wanted && candidate.Item.Reference != wanted {
			continue
		}
		return append([]inboxrclpucl.Document{}, sampleDocuments...), nil
	}
	return nil, inboxrclpucl.ErrClaimNotFound
}

// DocumentContent mengembalikan isi satu dokumen contoh.
//
// Kepemilikannya ditegakkan DI SINI pula, bukan hanya di sisi SQL. Uji yang berjalan di atas
// memori adalah tempat paling murah untuk menangkap pemanggil yang meminta dokumen milik
// klaim lain, dan penegakan yang hanya ada di satu sisi membuat uji itu lulus untuk perilaku
// yang tidak benar.
func (s *Store) DocumentContent(
	ctx context.Context,
	caseNumber, documentID string,
) (inboxrclpucl.DocumentContent, error) {
	documents, err := s.Documents(ctx, caseNumber)
	if err != nil {
		return inboxrclpucl.DocumentContent{}, err
	}

	wanted := strings.TrimSpace(documentID)
	for _, document := range documents {
		if document.ID != wanted {
			continue
		}
		return inboxrclpucl.DocumentContent{
			Name:     document.Name,
			MimeType: document.MimeType,
			Content:  []byte("isi contoh " + document.ID),
		}, nil
	}
	return inboxrclpucl.DocumentContent{}, inboxrclpucl.ErrDocumentNotFound
}

// detailOf menyusun isi layar kerja dari satu baris contoh.
//
// Ketiga isian TURUNAN dihitung di sini dengan cara yang sama seperti kueri SQL — termasuk
// "UP" yang mengambil sumber yang SAMA dengan "Nama Peserta". Menyimpangkannya akan membuat
// uji yang berjalan di atas memori menyatakan hal yang tidak benar tentang Oracle, dan justru
// pada isian yang paling mudah disangka cacat.
func detailOf(candidate Row) inboxrclpucl.ClaimDetail {
	return inboxrclpucl.ClaimDetail{
		Reference:   candidate.Item.Reference,
		ClaimNumber: candidate.Item.CaseID,

		// Keduanya memilih TOMBOL, bukan mengisi isian — sama seperti di sisi Oracle.
		// Melewatkannya di sini akan membuat uji yang berjalan di atas memori menggambar
		// susunan tombol yang berbeda dari layar yang sebenarnya.
		MSIG:       candidate.MSIG,
		GroupPanel: candidate.GroupPanel,

		Letter: inboxrclpucl.LetterDraft{
			Track:        inboxrclpucl.TrackOf(candidate.TrackCode),
			TrackCode:    candidate.TrackCode,
			AnalystNote:  candidate.Item.AnalystNote,
			PolicyNumber: candidate.Item.PolicyNumber,
			LossDate:     inboxrclpucl.DisplayTimeText(candidate.LossDate),

			InsuredName: candidate.FirstObjectName,
			SumInsured:  candidate.FirstObjectName,

			BillAmount: candidate.FirstProposeValue,

			Subject:     candidate.Subject,
			OpeningNote: candidate.OpeningNote,
			BodyNote:    candidate.BodyNote,
			ClosingNote: candidate.ClosingNote,
		},

		DocumentReceipt: inboxrclpucl.DocumentReceipt{
			// Satu baris, seperti di Oracle: hanya baris pertama page list yang terbaca.
			ReceivedDates: inboxrclpucl.ReceivedDatesOf("2026-09-04 10:05:00", "Dokumen awal"),
			PUCLNote:      candidate.PUCLNote,
		},
	}
}

// decorate mengisi isian yang di sistem baru DITURUNKAN, bukan disimpan.
//
// Dua: jalur penanganan dan keempat isian tanggal. Keduanya diturunkan lewat penggambar
// milik domain — `TrackOf` dan `DisplayTimeText` — dan memakai penggambar yang SAMA dengan
// penyimpanan SQL adalah syarat agar uji yang berjalan di atas memori menyatakan sesuatu
// tentang yang berjalan di Oracle.
func decorate(candidate Row) inboxrclpucl.WorkItem {
	item := candidate.Item
	item.Track = inboxrclpucl.TrackOf(candidate.TrackCode)

	// Ketiga isian tanggal baris contoh dilewatkan penggambar yang sama dengan penyimpanan
	// SQL, bukan dipakai apa adanya.
	//
	// Baris contoh menuliskannya sudah dalam bentuk tampilan, sehingga ini nyaris selalu
	// tanpa akibat — dan justru itulah gunanya: baris contoh yang kelak ditulis dalam
	// bentuk lain akan tergambar sama seperti di Oracle, alih-alih meloloskan uji yang
	// tidak akan lolos di produksi.
	item.InboxEntryAt = inboxrclpucl.DisplayTimeText(item.InboxEntryAt)
	item.LetterPrintedAt = inboxrclpucl.DisplayTimeText(item.LetterPrintedAt)
	item.ClaimAge = inboxrclpucl.DisplayTimeText(item.ClaimAge)

	// `CreatedAt` DITURUNKAN dari kolom pengurut, bukan diisi sendiri pada tiap baris
	// contoh.
	//
	// Ia harus mustahil menyimpang dari `Row.CreatedAt`: kolom yang digambar layar dan
	// kolom yang mengurutkannya adalah kolom yang SAMA, dan baris contoh yang keduanya
	// diisi terpisah dapat menyatakan urutan yang tidak sesuai dengan angka yang tampil.
	//
	// Waktu NOL menghasilkan teks kosong, bukan "0001-01-01": baris contoh yang tidak
	// menyebut waktunya berarti waktunya tidak diketahui, dan tanggal tahun 1 di layar
	// terbaca sebagai data, bukan sebagai ketiadaan data.
	item.CreatedAt = inboxrclpucl.DisplayTime(candidate.CreatedAt)

	return item
}

// reportRowOf menyusun satu baris laporan dari baris contoh.
//
// Perhatikan `ClaimStatus` diambil dari isian tersendiri, bukan dari `Item.ExpiryStatus`.
// Keduanya kolom yang berbeda — `STATUSCLAIM_1` dan `STATUSKLAIM_1` — dan namanya hanya
// berbeda satu huruf. Menyamakannya di sini akan menyembunyikan tertukarnya keduanya.
func reportRowOf(candidate Row) inboxrclpucl.DailyReportRow {
	return inboxrclpucl.DailyReportRow{
		Reference:       candidate.Item.Reference,
		CaseID:          candidate.Item.CaseID,
		PolicyNumber:    candidate.Item.PolicyNumber,
		InsuredName:     candidate.Item.InsuredName,
		SentAt:          inboxrclpucl.DisplayTimeText(candidate.Item.InboxEntryAt),
		AnalystNote:     candidate.Item.AnalystNote,
		LetterPrintedAt: inboxrclpucl.DisplayTimeText(candidate.Item.LetterPrintedAt),
		Track:           inboxrclpucl.TrackOf(candidate.TrackCode),
		ClaimStatus:     candidate.ClaimStatus,
	}
}

// matchesWorkClass meniru penyaring `PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'`.
//
// Ia penyaring PERTAMA karena ia pula yang paling mahal bila terlewat: satu tabel Pega
// menampung beberapa kelas objek kerja, dan semuanya punya `PYID`, `POLICYNO`, serta
// `QQNAME` — sehingga baris yang salah lolos tanpa terlihat keliru.
func matchesWorkClass(candidate Row) bool {
	return candidate.WorkClass == inboxrclpucl.WorkClassClaim
}

// Penyaring antrean bersama DIHAPUS 2026-10-01, mengikuti penyimpanan SQL.
//
// `TC_PNC_PUCL` adalah tabel khusus RCL/PUCL, sehingga penyaring itu tidak lagi menyeleksi
// apa pun — dan datanya menyimpan NAMA ORANG di kolom itu, bukan nama antrean, sehingga
// menyaringnya dengan `'RCLPUCL'` mengosongkan ketiga tab.
//
// `Row.AssignedOperator` sengaja DIPERTAHANKAN meski tidak lagi menyaring: ia bagian dari
// bentuk barisnya, dan baris contoh yang kehilangan isian itu akan menyembunyikan bahwa
// kolomnya ada beserta pertanyaan terbuka tentang apa yang seharusnya diisi.

// matchesWorkStatus meniru `PYSTATUSWORK <> 'Resolved-Completed'`.
//
// Perhatikan ia hanya mengeluarkan yang SELESAI. Klaim `Resolved-Rejected` TETAP lolos, dan
// itu memang benar: klaim yang ditolak justru pekerjaan utama antrean RCL.
func matchesWorkStatus(candidate Row) bool {
	return candidate.WorkStatus != inboxrclpucl.WorkStatusCompleted
}

// matchesLetterPrinted meniru penyaring `TANGGALCETAKDOKUMENPUCL_1`.
//
//	tab Cetak Surat                     IS NULL
//	tab Kelengkapan Dokumen, Klaim MSIG IS NOT NULL
//
// Inilah penyaring yang memindahkan satu klaim dari tab pertama ke tab kedua begitu
// suratnya dicetak.
func matchesLetterPrinted(candidate Row, q inboxrclpucl.Query) bool {
	printed := strings.TrimSpace(candidate.Item.LetterPrintedAt) != ""
	return printed == q.Tab.LetterPrinted
}

// matchesExpiryCaseStatus meniru `STATUSCASE_1 = '0'` tab Cetak Surat.
//
// Ia HANYA berlaku di sana — kedua tab lain tidak menyaring kolom ini sama sekali.
//
// Perhatikan kolom yang disaring di sini BUKAN kolom yang digambar layar sebagai "Status
// Kadaluarsa"; yang digambar adalah `STATUSKLAIM_1`. Lihat Row.ExpiryCaseStatus.
func matchesExpiryCaseStatus(candidate Row, q inboxrclpucl.Query) bool {
	if q.Tab.Code != inboxrclpucl.TabCetakSurat {
		return true
	}
	return candidate.ExpiryCaseStatus == inboxrclpucl.ExpiryStatusActive
}

// matchesApproval meniru `PUCLAPPROVE_1 <> '1'` tab Kelengkapan Dokumen dan Klaim MSIG.
//
// # Nilai KOSONG sengaja TIDAK lolos
//
// `NULL <> '1'` menghasilkan UNKNOWN di Oracle maupun PostgreSQL, bukan TRUE, sehingga
// baris yang kolomnya belum pernah diisi TIDAK muncul di kedua tab itu. Itu perilaku sistem
// lama, dan ditiru di sini tanpa diperbaiki — lihat catatan di kepala paket dan pada berkas
// .sql.
//
// Menuliskannya sebagai `candidate.PUCLApprove != PUCLReturnedToAnalyst` akan MELOLOSKAN nilai
// kosong, dan uji yang lulus karenanya tidak menyatakan apa pun tentang Oracle.
func matchesApproval(candidate Row, q inboxrclpucl.Query) bool {
	if q.Tab.Code == inboxrclpucl.TabCetakSurat {
		return true
	}
	if strings.TrimSpace(candidate.PUCLApprove) == "" {
		return false
	}
	return candidate.PUCLApprove != inboxrclpucl.PUCLReturnedToAnalyst
}

// matchesMSIG meniru penyaring `MSIG_1` tab Kelengkapan Dokumen dan Klaim MSIG.
//
//	tab Kelengkapan Dokumen  IS NULL
//	tab Klaim MSIG           = 'MSIG'
//
// Tab Cetak Surat tidak menyaringnya sama sekali — Report Definition-nya memang tidak punya
// penyaring itu, sehingga klaim jalur MSIG yang suratnya belum dicetak TETAP muncul di
// sana. Itu perilaku sistem lama apa adanya.
func matchesMSIG(candidate Row, q inboxrclpucl.Query) bool {
	if q.Tab.Code == inboxrclpucl.TabCetakSurat {
		return true
	}

	marked := strings.TrimSpace(candidate.MSIG)
	if q.Tab.MSIG {
		return marked == inboxrclpucl.MSIGMarker
	}
	return marked == ""
}

// parseRange membaca kedua batas rentang laporan.
func parseRange(rng inboxrclpucl.DateRange) (time.Time, time.Time, bool) {
	const layout = "2006-01-02"

	from, err := time.Parse(layout, strings.TrimSpace(rng.From))
	if err != nil {
		return time.Time{}, time.Time{}, false
	}
	to, err := time.Parse(layout, strings.TrimSpace(rng.To))
	if err != nil {
		return time.Time{}, time.Time{}, false
	}
	return from, to, true
}

// withinRange meniru `>= awal AND < akhir + 1 hari`.
//
// Batas atasnya SETENGAH TERBUKA, persis seperti kuerinya, sehingga klaim yang dikirim pada
// jam berapa pun di tanggal akhir tetap ikut. Menuliskannya sebagai `<= akhir` akan
// membuang seluruh baris yang jamnya bukan tengah malam — kesalahan yang hanya terlihat
// pada data yang punya komponen jam.
func withinRange(at, from, to time.Time) bool {
	if at.IsZero() {
		return false
	}
	if at.Before(from) {
		return false
	}
	return at.Before(to.AddDate(0, 0, 1))
}

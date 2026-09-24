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

	// AssignedOperator adalah pemegang penugasan — `PXASSIGNEDOPERATORID`.
	//
	// Pada antrean bersama, isinya nama AKUN antrean (`RCLPUCL`), bukan nama orang.
	AssignedOperator string

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

	// GroupPanel adalah `GROUPPANEL_1` — dipakai HANYA cabang kedua laporan harian.
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
		if !matchesQueue(candidate) {
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

// decorate mengisi isian yang di sistem baru DITURUNKAN, bukan disimpan.
//
// Hanya satu: jalur penanganan, yang diturunkan dari kode mentah lewat penerjemah milik
// domain. Memakai penerjemah yang sama dengan penyimpanan SQL adalah syarat agar uji yang
// berjalan di atas memori menyatakan sesuatu tentang yang berjalan di Oracle.
func decorate(candidate Row) inboxrclpucl.WorkItem {
	item := candidate.Item
	item.Track = inboxrclpucl.TrackOf(candidate.TrackCode)
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
		SentAt:          candidate.Item.InboxEntryAt,
		AnalystNote:     candidate.Item.AnalystNote,
		LetterPrintedAt: candidate.Item.LetterPrintedAt,
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

// matchesQueue meniru gabungan ke antrean bersama beserta penyaring akunnya.
//
// Ketiga tab memakai akun yang SAMA (`RCLPUCL`), sehingga penyaring ini tidak bergantung
// tab sama sekali.
func matchesQueue(candidate Row) bool {
	return strings.EqualFold(candidate.AssignedOperator, inboxrclpucl.RCLPUCLWorkbasket)
}

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
// Menuliskannya sebagai `candidate.PUCLApprove != PUCLApproved` akan MELOLOSKAN nilai
// kosong, dan uji yang lulus karenanya tidak menyatakan apa pun tentang Oracle.
func matchesApproval(candidate Row, q inboxrclpucl.Query) bool {
	if q.Tab.Code == inboxrclpucl.TabCetakSurat {
		return true
	}
	if strings.TrimSpace(candidate.PUCLApprove) == "" {
		return false
	}
	return candidate.PUCLApprove != inboxrclpucl.PUCLApproved
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

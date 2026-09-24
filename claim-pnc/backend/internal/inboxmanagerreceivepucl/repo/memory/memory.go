// Package memory memenuhi seam inboxmanagerreceivepucl.Repo dengan penyimpanan di memori.
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
// di Oracle. Ketiga penyaring ditiru sedekat-dekatnya dengan predikat SQL-nya: kelas objek
// kerja, tabel penugasan yang dipakai, dan Group Panel.
//
// # Satu isian yang DIHITUNG di sini, bukan disimpan
//
// `ClaimType` tidak ada di Row, dan itu disengaja: di sistem lama ia bukan kolom melainkan
// properti di dalam blob, dan di sistem baru ia DITURUNKAN dari Group Panel. Menyimpannya
// sebagai data contoh akan membuat penyaring yang salah tetap lulus uji, karena nilainya
// datang dari tempat yang salah.
package memory

import (
	"context"
	"sort"
	"strings"
	"time"

	"claim-pnc/internal/inboxmanagerreceivepucl"
)

// Row adalah satu baris contoh beserta kolom yang TIDAK ditampilkan tetapi menyaring,
// mengurutkan, atau menghitung.
//
// Keempat isian tambahan tidak masuk inboxmanagerreceivepucl.WorkItem dengan sengaja:
// tidak satu pun sampai ke layar, dan menaruhnya di tipe domain akan membuat orang menduga
// keempatnya bagian dari kontrak.
type Row struct {
	Item inboxmanagerreceivepucl.WorkItem

	// WorkClass adalah kelas objek kerja — `PXOBJCLASS`.
	//
	// Di sistem lama ia yang memisahkan berkas penerimaan dokumen dari klaim, dan keduanya
	// hidup di tabel yang sama. Ia ditiru di sini karena tanpanya penyimpanan memori tidak
	// dapat membuktikan penyaring itu benar-benar dipakai.
	WorkClass string

	// FromWorkbasket menyatakan baris ini berada di PC_ASSIGN_WORKBASKET, bukan di
	// PC_ASSIGN_WORKLIST.
	//
	// Di sistem lama ia bukan kolom melainkan TABEL yang berbeda — kedua kueri Receive
	// membaca penugasan per orang, kueri RCL/PUCL membaca antrean bersama.
	FromWorkbasket bool

	// AssignedOperator adalah pemegang penugasan — `PXASSIGNEDOPERATORID`.
	//
	// Pada antrean bersama, isinya nama AKUN antrean (`RCLPUCL`), bukan nama orang.
	AssignedOperator string

	// GroupPanel adalah segmentasi lini bisnis — `GROUPPANEL_1`.
	//
	// Ia yang menggantikan `.ReceiveDocument.TypeOfClaim`, dan dari sinilah
	// `WorkItem.ClaimType` diturunkan — bukan disimpan.
	GroupPanel string

	// WorkStatus adalah status kerja — `PYSTATUSWORK`.
	//
	// Hanya tab RCL/PUCL yang menyaringnya, dan hanya untuk mengeluarkan yang SELESAI.
	WorkStatus string

	// CreatedAt adalah waktu objek kerja dibuat — `PXCREATEDATETIME`.
	//
	// Ia dasar pengurutan ketiga kueri. Ia TERPISAH dari `Item.InboxEntryAt`, yang teksnya
	// digambar di layar: yang satu dipakai mengurutkan, yang lain dibaca pengguna, dan
	// keduanya dapat berbeda bentuk karena bentuk teks kolomnya tidak diketahui (`R-08`).
	CreatedAt time.Time
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
	q inboxmanagerreceivepucl.Query,
	page inboxmanagerreceivepucl.Pagination,
) (inboxmanagerreceivepucl.Page, error) {
	type ordered struct {
		item inboxmanagerreceivepucl.WorkItem
		at   time.Time
	}

	matched := []ordered{}

	for _, candidate := range s.rows {
		if !matchesWorkClass(candidate, q) {
			continue
		}
		if !matchesQueue(candidate, q) {
			continue
		}
		if !matchesClaimType(candidate, q) {
			continue
		}
		if !matchesWorkStatus(candidate, q) {
			continue
		}

		matched = append(matched, ordered{
			item: decorate(candidate, q),
			at:   candidate.CreatedAt,
		})
	}

	// Urutan ditetapkan supaya paginasi di atasnya stabil: yang terbaru masuk lebih dulu,
	// persis seperti `ORDER BY w.PXCREATEDATETIME DESC, w.PYID` pada ketiga kueri. Baris
	// yang waktunya sama diurutkan menurut nomor case supaya tetap deterministik.
	sort.SliceStable(matched, func(i, j int) bool {
		left, right := matched[i].at, matched[j].at
		if !left.Equal(right) {
			return left.After(right)
		}
		return matched[i].item.CaseID < matched[j].item.CaseID
	})

	items := make([]inboxmanagerreceivepucl.WorkItem, 0, len(matched))
	for _, row := range matched {
		items = append(items, row.item)
	}

	return inboxmanagerreceivepucl.Slice(items, page), nil
}

// decorate mengisi isian yang di sistem baru DITURUNKAN, bukan disimpan.
//
// Hanya satu: Jenis Klaim, yang diturunkan dari Group Panel lewat penerjemah milik domain.
// Memakai penerjemah yang sama dengan penyimpanan SQL adalah syarat agar uji yang berjalan
// di atas memori menyatakan sesuatu tentang yang berjalan di Oracle.
//
// Baris tab RCL/PUCL tidak punya Group Panel — kolomnya `CAST(NULL …)` di SQL — sehingga
// isian ini dibiarkan kosong, bukan diisi "NONMBU" yang tidak berarti apa pun bagi klaim.
func decorate(
	candidate Row,
	q inboxmanagerreceivepucl.Query,
) inboxmanagerreceivepucl.WorkItem {
	item := candidate.Item

	if q.Tab.ClaimType != "" {
		item.ClaimType = inboxmanagerreceivepucl.ClaimTypeOf(candidate.GroupPanel)
	}

	return item
}

// matchesWorkClass meniru penyaring `PXOBJCLASS` yang membedakan kedua kelas objek kerja.
//
// Ia penyaring PERTAMA di sini karena ia pula yang paling mahal bila terlewat: kedua kelas
// hidup di satu tabel dan bentuk barisnya cukup mirip untuk lolos tanpa terlihat.
func matchesWorkClass(candidate Row, q inboxmanagerreceivepucl.Query) bool {
	wanted := inboxmanagerreceivepucl.WorkClassReceiveDocument
	if q.Tab.Code == inboxmanagerreceivepucl.TabRCLPUCL {
		wanted = inboxmanagerreceivepucl.WorkClassClaim
	}
	return candidate.WorkClass == wanted
}

// matchesQueue meniru pemilihan TABEL PENUGASAN ketiga kueri.
//
//	tab Receive    PC_ASSIGN_WORKLIST, tanpa penyaring pemegangnya
//	tab RCL/PUCL   PC_ASSIGN_WORKBASKET, PXASSIGNEDOPERATORID = RCLPUCL
//
// Perhatikan kedua tab Receive TIDAK menyaring menurut pemegang penugasan. Itu memang
// perilaku Report Definition-nya: satu-satunya penyaring penugasan di sana adalah unit
// organisasi, dan parameternya tidak pernah diisi.
func matchesQueue(candidate Row, q inboxmanagerreceivepucl.Query) bool {
	if !q.Tab.FromWorkbasket {
		return !candidate.FromWorkbasket
	}

	return candidate.FromWorkbasket &&
		strings.EqualFold(
			candidate.AssignedOperator,
			inboxmanagerreceivepucl.RCLPUCLWorkbasket,
		)
}

// matchesClaimType meniru penyaring Group Panel kedua tab Receive.
//
// Baris yang Group Panel-nya kosong tidak lolos SATU PUN dari keduanya, dan itu bukan
// kekeliruan: `GROUPPANEL_1 <> '002'` tidak menangkap NULL di Oracle maupun PostgreSQL,
// sehingga penyimpanan ini meniru perilaku yang sama persis.
func matchesClaimType(candidate Row, q inboxmanagerreceivepucl.Query) bool {
	if q.Tab.ClaimType == "" {
		return true
	}

	panel := strings.TrimSpace(candidate.GroupPanel)
	if panel == "" {
		return false
	}

	return inboxmanagerreceivepucl.ClaimTypeOf(panel) == q.Tab.ClaimType
}

// matchesWorkStatus meniru penyaring `PYSTATUSWORK <> 'Resolved-Completed'` tab RCL/PUCL.
//
// Ia HANYA berlaku di sana. Kedua tab Receive tidak menyaring status kerja sama sekali —
// Report Definition-nya memang tidak punya penyaring itu, dan menambahkannya akan
// menyembunyikan berkas yang di layar lama terlihat.
func matchesWorkStatus(candidate Row, q inboxmanagerreceivepucl.Query) bool {
	if q.Tab.Code != inboxmanagerreceivepucl.TabRCLPUCL {
		return true
	}
	return candidate.WorkStatus != inboxmanagerreceivepucl.WorkStatusCompleted
}

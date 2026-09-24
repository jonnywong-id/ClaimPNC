// Package memory memenuhi seam inboxclaimtreatyprop.Repo dengan penyimpanan di memori.
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
// di Oracle. Ketiga penyaring ditiru sedekat-dekatnya dengan predikat SQL-nya: penanda
// `%CLMP%` pada kunci objek kerja, kepemilikan penugasan, dan antrean teknik.
package memory

import (
	"context"
	"sort"
	"strings"
	"time"

	"claim-pnc/internal/inboxclaimtreatyprop"
)

// Row adalah satu baris contoh beserta kolom yang TIDAK ditampilkan tetapi menyaring atau
// mengurutkan.
//
// Kedua kolom tambahan tidak masuk inboxclaimtreatyprop.WorkItem dengan sengaja: keduanya
// tidak pernah sampai ke layar, dan menaruhnya di tipe domain akan membuat orang menduga
// keduanya bagian dari kontrak.
type Row struct {
	Item inboxclaimtreatyprop.WorkItem

	// FromWorkbasket menyatakan baris ini berada di PC_ASSIGN_WORKBASKET, bukan di
	// PC_ASSIGN_WORKLIST.
	//
	// Pembedaan ini memisahkan tab Teknik dari kedua tab worklist. Di sistem lama ia
	// bukan kolom melainkan TABEL yang berbeda — `GetClaimTreatyTeknik_SQL` membaca
	// workbasket, dua kueri lainnya membaca worklist.
	FromWorkbasket bool

	// AssignedAt adalah waktu penugasan dibuat — `PXCREATEDATETIME`, dasar pengurutan.
	AssignedAt time.Time
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

// claimKeyMarker adalah penanda yang membedakan objek kerja klaim treaty dari objek kerja
// lain di tabel penugasan yang sama.
//
// Ketiga kueri memakainya: `WHERE a.PXREFOBJECTKEY LIKE '%CLMP%'`. Ia ditiru di sini supaya
// baris contoh yang tidak bertanda itu ikut tersaring, persis seperti di Oracle.
const claimKeyMarker = "CLMP"

// List mengembalikan satu halaman baris yang lolos penyaring beserta jumlah seluruhnya.
func (s *Store) List(
	_ context.Context,
	q inboxclaimtreatyprop.Query,
	page inboxclaimtreatyprop.Pagination,
) (inboxclaimtreatyprop.Page, error) {
	matched := []inboxclaimtreatyprop.WorkItem{}
	order := map[string]time.Time{}

	for _, candidate := range s.rows {
		if !strings.Contains(strings.ToUpper(candidate.Item.WorkKey), claimKeyMarker) {
			continue
		}
		if !matchesQueue(candidate, q) {
			continue
		}
		matched = append(matched, candidate.Item)
		order[rowKey(candidate.Item)] = candidate.AssignedAt
	}

	// Urutan ditetapkan supaya paginasi di atasnya stabil: penugasan terbaru lebih dulu,
	// mengikuti `ORDER BY A.PXCREATEDATETIME DESC` pada GetClaimTreaty_SQL. Baris yang
	// waktunya sama diurutkan menurut Claim ID supaya tetap deterministik.
	sort.SliceStable(matched, func(i, j int) bool {
		left, right := order[rowKey(matched[i])], order[rowKey(matched[j])]
		if !left.Equal(right) {
			return left.After(right)
		}
		return matched[i].ClaimID < matched[j].ClaimID
	})

	return inboxclaimtreatyprop.Slice(matched, page), nil
}

// rowKey menyusun kunci pengurutan satu baris.
//
// Reference dipakai lebih dulu karena ia kunci teknis yang unik; Claim ID menjadi cadangan
// bagi baris contoh yang sengaja tidak memilikinya.
func rowKey(item inboxclaimtreatyprop.WorkItem) string {
	if item.Reference != "" {
		return item.Reference
	}
	return item.ClaimID
}

// matchesQueue meniru pemilihan antrean ketiga kueri.
//
//	tab Teknik                      workbasket, PXASSIGNEDOPERATORID = TreatyinPNCTeknik
//	tab Work List tanpa "See All"   worklist, PXASSIGNEDOPERATORID = pemanggil
//	tab Work List dengan "See All"  worklist, tanpa penyaring operator
func matchesQueue(candidate Row, q inboxclaimtreatyprop.Query) bool {
	if q.Tab.Code == inboxclaimtreatyprop.TabTechnical {
		return candidate.FromWorkbasket &&
			strings.EqualFold(
				candidate.Item.AssignedOperator,
				inboxclaimtreatyprop.TechnicalWorkbasket,
			)
	}

	if candidate.FromWorkbasket {
		return false
	}
	if !q.ScopedToCaller() {
		return true
	}
	return strings.EqualFold(candidate.Item.AssignedOperator, q.Caller.Login)
}

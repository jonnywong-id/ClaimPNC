// Package memory memenuhi seam inboxclaimtreatynonprop.Repo dengan penyimpanan di memori.
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
// di Oracle. Keempat penyaring ditiru sedekat-dekatnya dengan predikat SQL-nya: awalan
// `CLMNP-` pada nomor klaim, kepemilikan penugasan, antrean teknik, dan polis kosong.
//
// # Dua isian yang DIHITUNG di sini, bukan disimpan
//
// `Status` dan `AgingDays` tidak ada di Row, dan itu disengaja: di sistem lama keduanya
// bukan kolom melainkan hasil kuerinya — teks tetap dan pengurangan dua tanggal. Menyimpan
// keduanya sebagai data contoh akan membuat penyaring yang salah tetap lulus uji, karena
// nilainya datang dari tempat yang salah.
package memory

import (
	"context"
	"sort"
	"strings"
	"time"

	"claim-pnc/internal/inboxclaimtreatynonprop"
)

// Row adalah satu baris contoh beserta kolom yang TIDAK ditampilkan tetapi menyaring,
// mengurutkan, atau menghitung.
//
// Ketiga kolom tambahan tidak masuk inboxclaimtreatynonprop.WorkItem dengan sengaja:
// ketiganya tidak pernah sampai ke layar, dan menaruhnya di tipe domain akan membuat orang
// menduga ketiganya bagian dari kontrak.
type Row struct {
	Item inboxclaimtreatynonprop.WorkItem

	// FromWorkbasket menyatakan baris ini berada di PC_ASSIGN_WORKBASKET, bukan di
	// PC_ASSIGN_WORKLIST.
	//
	// Pembedaan ini memisahkan tab Teknik dari tab Admin. Di sistem lama ia bukan kolom
	// melainkan TABEL yang berbeda — `GetInboxListCNP_SQL` membaca workbasket, ketiga
	// kueri Admin membaca worklist.
	FromWorkbasket bool

	// WorkCreatedAt adalah waktu OBJEK KERJA dibuat — `b.PXCREATEDATETIME`.
	//
	// Ia dasar dua hal sekaligus: pengurutan seluruh kueri, dan perhitungan Aging.
	// Perhatikan ia milik objek kerja, BUKAN milik baris penugasan — kueri lama
	// mengurutkan dan menghitung umur dari tabel `b`, bukan `a`.
	WorkCreatedAt time.Time
}

// Store adalah penyimpanan antrean di memori.
//
// Ia tidak dilindungi mutex karena tidak pernah berubah setelah dibentuk: modul ini hanya
// membaca, dan tidak ada satu pun operasi yang menulis.
type Store struct {
	rows []Row

	// now adalah sumber waktu untuk menghitung Aging.
	//
	// Ia dapat diganti supaya uji tidak bergantung pada tanggal hari ini — umur yang
	// dihitung terhadap `time.Now()` berubah setiap hari, dan uji yang memeriksanya akan
	// gagal esok hari tanpa ada yang menyentuh kode.
	now func() time.Time
}

// NewStore membentuk penyimpanan berisi baris yang diberikan.
func NewStore(rows ...Row) *Store {
	return &Store{rows: rows, now: time.Now}
}

// NewStoreAt membentuk penyimpanan dengan sumber waktu tetap.
//
// Dipakai uji yang memeriksa Aging. Ia tidak dipakai di jalur produksi mana pun.
func NewStoreAt(clock func() time.Time, rows ...Row) *Store {
	return &Store{rows: rows, now: clock}
}

// NewSampleStore membentuk penyimpanan berisi contoh bawaan.
func NewSampleStore() *Store {
	return NewStore(SampleRows()...)
}

// List mengembalikan satu halaman baris yang lolos penyaring beserta jumlah seluruhnya.
func (s *Store) List(
	_ context.Context,
	q inboxclaimtreatynonprop.Query,
	page inboxclaimtreatynonprop.Pagination,
) (inboxclaimtreatynonprop.Page, error) {
	type ordered struct {
		item inboxclaimtreatynonprop.WorkItem
		at   time.Time
	}

	matched := []ordered{}

	for _, candidate := range s.rows {
		// Penyaring awalan nomor klaim, meniru `PXREFOBJECTINSNAME LIKE 'CLMNP-%'`.
		//
		// Ia AWALAN, bukan potongan di tengah: nomor yang memuat "CLMNP-" di tengahnya
		// tidak lolos, persis seperti jangkar depan pada kuerinya.
		if !strings.HasPrefix(
			strings.ToUpper(candidate.Item.ClaimID),
			inboxclaimtreatynonprop.ClaimPrefix,
		) {
			continue
		}
		if !matchesQueue(candidate, q) {
			continue
		}
		if !matchesTBA(candidate, q) {
			continue
		}

		matched = append(matched, ordered{
			item: s.decorate(candidate, q),
			at:   candidate.WorkCreatedAt,
		})
	}

	// Urutan ditetapkan supaya paginasi di atasnya stabil, dan ARAHNYA BERBEDA per tab —
	// persis seperti kuerinya. Tab Admin mendahulukan yang terbaru
	// (`ORDER BY b.PXCREATEDATETIME DESC`); tab Teknik mendahulukan yang paling lama
	// menunggu (`ORDER BY CARI21` menaik). Baris yang waktunya sama diurutkan menurut
	// nomor klaim supaya tetap deterministik.
	ascending := q.Tab.Code == inboxclaimtreatynonprop.TabTechnical
	sort.SliceStable(matched, func(i, j int) bool {
		left, right := matched[i].at, matched[j].at
		if !left.Equal(right) {
			if ascending {
				return left.Before(right)
			}
			return left.After(right)
		}
		return matched[i].item.ClaimID < matched[j].item.ClaimID
	})

	items := make([]inboxclaimtreatynonprop.WorkItem, 0, len(matched))
	for _, row := range matched {
		items = append(items, row.item)
	}

	return inboxclaimtreatynonprop.Slice(items, page), nil
}

// decorate mengisi kedua isian yang di sistem lama dihasilkan KUERI, bukan disimpan.
//
//	Status     teks tetap yang berbeda per antrean
//	AgingDays  selisih hari kalender antara hari ini dan pembuatan objek kerja
//
// Keduanya dihitung di sini supaya penyimpanan memori menghasilkan baris yang bentuknya
// sama dengan yang datang dari Oracle — termasuk sifatnya yang berubah sendiri tiap hari.
func (s *Store) decorate(
	candidate Row,
	q inboxclaimtreatynonprop.Query,
) inboxclaimtreatynonprop.WorkItem {
	item := candidate.Item

	if q.Tab.Code == inboxclaimtreatynonprop.TabTechnical {
		item.Status = inboxclaimtreatynonprop.StatusAcceptation
	} else {
		item.Status = inboxclaimtreatynonprop.StatusEstimation
	}

	item.AgingDays = agingDays(s.now(), candidate.WorkCreatedAt)
	return item
}

// agingDays meniru `TRUNC(SYSDATE) - TRUNC(b.PXCREATEDATETIME)`.
//
// Kedua sisi dipangkas ke tanggal lebih dulu, lalu selisihnya dihitung dalam hari penuh.
// Memangkas keduanya penting: selisih dua timestamp mentah akan menghasilkan angka yang
// berubah menurut JAM, sehingga pekerjaan yang dibuat kemarin sore terhitung nol hari
// sampai lewat 24 jam.
//
// Baris tanpa pasangan objek kerja tidak punya waktu pembuatan; umurnya nol, sama dengan
// yang dihasilkan pemindai SQL saat kolomnya NULL.
func agingDays(now, created time.Time) int {
	if created.IsZero() {
		return 0
	}

	truncate := func(t time.Time) time.Time {
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	}

	days := int(truncate(now).Sub(truncate(created.In(now.Location()))).Hours() / 24)
	if days < 0 {
		// Objek kerja bertanggal masa depan. Oracle akan menghasilkan angka negatif;
		// di sini ia dijepit ke nol, karena umur negatif tidak punya arti bagi pembaca
		// grid dan hanya akan terbaca sebagai kerusakan.
		return 0
	}
	return days
}

// matchesQueue meniru pemilihan antrean kelima kueri.
//
//	tab Teknik                 workbasket, PXASSIGNEDOPERATORID = TreatyinPNCTeknik
//	tab Admin tanpa "See All"  worklist, PXASSIGNEDOPERATORID = pemanggil
//	tab Admin dengan "See All" worklist, tanpa penyaring operator
func matchesQueue(candidate Row, q inboxclaimtreatynonprop.Query) bool {
	if q.Tab.Code == inboxclaimtreatynonprop.TabTechnical {
		return candidate.FromWorkbasket &&
			strings.EqualFold(
				candidate.Item.AssignedOperator,
				inboxclaimtreatynonprop.TechnicalWorkbasket,
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

// matchesTBA meniru penyaring `c.NOPOLIS IS NULL` pada kueri TBA.
//
// Nomor polis kosong di sini berarti NULL di sana: kedua kueri TBA mencari klaim treaty
// yang sudah masuk tetapi polisnya belum terbit.
func matchesTBA(candidate Row, q inboxclaimtreatynonprop.Query) bool {
	if !q.TBAOnly {
		return true
	}
	return strings.TrimSpace(candidate.Item.PolicyNumber) == ""
}

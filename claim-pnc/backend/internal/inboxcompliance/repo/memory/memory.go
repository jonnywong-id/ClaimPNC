// Package memory memenuhi seam inboxcompliance.Repo dengan penyimpanan di memori.
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
// di Oracle. Penyaring workbasket dan urutan barisnya ditiru sedekat-dekatnya dengan
// predikat SQL-nya — termasuk urutan `PXCREATEDATETIME DESC, PYID DESC` yang menentukan
// baris mana masuk halaman pertama.
package memory

import (
	"context"
	"sort"
	"strconv"
	"sync"
	"time"

	"claim-pnc/internal/inboxcompliance"
)

// Row adalah satu baris contoh beserta kolom yang TIDAK ditampilkan tetapi menyaring atau
// mengurutkan.
//
// Kedua kolom tambahan itu tidak masuk inboxcompliance.WorkItem dengan sengaja: keduanya
// tidak pernah sampai ke layar, dan menaruhnya di tipe domain akan membuat orang menduga
// ia bagian dari kontrak.
type Row struct {
	Item inboxcompliance.WorkItem

	// Tab adalah kode tab tempat baris ini muncul.
	//
	// Ia ada karena kedua tab membaca TABEL YANG BERBEDA, bukan satu tabel dengan
	// penyaring berbeda — dan penyimpanan di memori harus meniru pemisahan itu. Tanpanya,
	// baris Post Audit akan bocor ke tab Compliance saat pengembangan lokal, lalu
	// perbedaannya baru ketahuan di Oracle.
	//
	// Kosong berarti tab Compliance, supaya baris contoh yang sudah ada tidak perlu
	// diubah.
	Tab string

	// Workbasket adalah antrean tempat baris ini menunggu —
	// `PC_ASSIGN_WORKBASKET.PXASSIGNEDOPERATORID`. Hanya berlaku pada tab Compliance;
	// tab Post Audit tidak membaca workbasket sama sekali.
	Workbasket string

	// CreatedAt — `PXCREATEDATETIME`, kunci pengurutan utama.
	CreatedAt time.Time

	// Resolved menandai klaimnya sudah selesai (`PYSTATUSWORK = 'Resolved-Completed'`),
	// sehingga barisnya HILANG dari antrean.
	//
	// Baris seperti ini sengaja ada di data contoh: tanpanya, penyaring yang lupa
	// dipasang tidak akan pernah ketahuan saat pengembangan lokal.
	Resolved bool
}

// tab mengembalikan kode tab baris ini, dengan tab Compliance sebagai bawaan.
func (r Row) tab() string {
	if r.Tab == "" {
		return inboxcompliance.TabCompliance
	}
	return r.Tab
}

// sortKey mengembalikan waktu yang dipakai mengurutkan baris ini pada sebuah tab.
//
// Tab Compliance memakai waktu buat barisnya; tab Post Audit memakai Tanggal Kirim Post
// Audit, karena tabel datarnya memang tidak punya kolom waktu buat.
func (r Row) sortKey(tabCode string) time.Time {
	if tabCode == inboxcompliance.TabPostAudit {
		if r.Item.PostAuditSentDate == nil {
			// Tanggal kosong diurutkan paling belakang, meniru `NULLS LAST` pada
			// kuerinya.
			return time.Time{}
		}
		return *r.Item.PostAuditSentDate
	}
	return r.CreatedAt
}

// Store adalah penyimpanan antrean di memori.
//
// Ia dilindungi mutex karena CreatePostAudit MENAMBAH baris. Operasi bacanya sendiri tidak
// mengubah apa pun, tetapi keduanya menyentuh senarai yang sama.
type Store struct {
	mu   sync.Mutex
	rows []Row

	// sequence meniru POOLDATA.CPNC_POST_AUDIT_SEQ, termasuk titik mulainya.
	sequence int64
}

// NewStore membentuk penyimpanan berisi baris yang diberikan.
func NewStore(rows ...Row) *Store {
	// 100000, bukan 100001: pencacah dinaikkan LEBIH DULU saat dipakai, sehingga nomor
	// pertama yang terbit tetap CPL-100001 — sama dengan START WITH sequence-nya.
	return &Store{rows: rows, sequence: 100000}
}

// List mengambil satu halaman antrean, meniru predikat dan urutan kueri Oracle.
func (s *Store) List(
	_ context.Context,
	q inboxcompliance.Query,
	page inboxcompliance.Pagination,
) (inboxcompliance.Page, error) {
	clean := page.Normalize()

	matched := make([]Row, 0, len(s.rows))
	for _, row := range s.rows {
		if row.tab() != q.Tab.Code {
			continue
		}

		// Kedua penyaring berikut HANYA berlaku pada tab Compliance, karena hanya tab itu
		// yang membaca workbasket dan status kerja. Tab Post Audit membaca tabel datar
		// yang tidak punya kolom status sama sekali — lihat catatan di
		// repo/sqlstore/inboxcompliance.sql.
		if q.Tab.Code == inboxcompliance.TabCompliance {
			if row.Resolved || row.Workbasket != q.Workbasket {
				continue
			}
		}

		matched = append(matched, row)
	}

	// Kedua tab diurutkan BERBEDA, dan keduanya meniru kuerinya masing-masing:
	//
	//	Compliance  ORDER BY A.PXCREATEDATETIME DESC, A.PYID DESC
	//	Post Audit  ORDER BY CASEID DESC, TGL_KIRIM_POST_AUDIT DESC
	//
	// Perhatikan tab Post Audit: nomor case yang menjadi kunci PERTAMA, dan ia diurutkan
	// sebagai TEKS. Itu bukan penyederhanaan — layar Pega menampilkan
	// `CPL-3, CPL-2, CPL-19, CPL-17, …`, yang hanya masuk akal bila teksnya yang
	// dibandingkan. Mengurutkan angkanya akan menaruh `CPL-19` di atas `CPL-3`.
	sort.SliceStable(matched, func(i, j int) bool {
		if q.Tab.Code == inboxcompliance.TabPostAudit {
			if matched[i].Item.CaseID != matched[j].Item.CaseID {
				return matched[i].Item.CaseID > matched[j].Item.CaseID
			}
			return matched[i].sortKey(q.Tab.Code).After(matched[j].sortKey(q.Tab.Code))
		}

		left, right := matched[i].sortKey(q.Tab.Code), matched[j].sortKey(q.Tab.Code)
		if !left.Equal(right) {
			return left.After(right)
		}
		return matched[i].Item.CaseID > matched[j].Item.CaseID
	})

	result := inboxcompliance.Page{
		Items:      []inboxcompliance.WorkItem{},
		Total:      len(matched),
		Pagination: clean,
	}

	offset := clean.Offset()
	if offset >= len(matched) {
		return result, nil
	}

	end := offset + clean.Size
	if end > len(matched) {
		end = len(matched)
	}

	for _, row := range matched[offset:end] {
		result.Items = append(result.Items, row.Item)
	}

	return result, nil
}

// FindInQueue mencari satu klaim yang sedang menunggu di antrean Compliance.
//
// Penyaringnya ditiru dari kueri `find_compliance_claim`, termasuk kenyataan bahwa klaim
// yang sudah selesai maupun yang berada di antrean lain TIDAK ditemukan.
func (s *Store) FindInQueue(
	_ context.Context, q inboxcompliance.Query, reference string,
) (inboxcompliance.WorkItem, bool, error) {
	for _, row := range s.rows {
		if row.tab() != inboxcompliance.TabCompliance {
			continue
		}
		if row.Resolved || row.Workbasket != q.Workbasket {
			continue
		}
		if row.Item.Reference != reference {
			continue
		}
		return row.Item, true, nil
	}

	return inboxcompliance.WorkItem{}, false, nil
}

// CreatePostAudit menulis satu baris Post Audit ke dalam memori.
//
// Nomornya dibentuk dari pencacah di memori, meniru sequence basis data — termasuk titik
// mulainya, 100.001, supaya nomor contoh berbentuk sama dengan yang terbit di Oracle.
//
// Berbeda dari operasi baca, yang ini MENGUBAH keadaan, sehingga penyimpanan ini kini
// dilindungi mutex. Tanpa itu, dua permintaan bersamaan saat pengembangan lokal dapat
// menerbitkan nomor yang sama — cacat yang justru tidak akan terjadi di Oracle, sehingga ia
// hanya akan membingungkan.
func (s *Store) CreatePostAudit(
	_ context.Context, entry inboxcompliance.PostAuditEntry,
) (inboxcompliance.PostAuditEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sequence++

	saved := entry
	saved.CaseID = "CPL-" + strconv.FormatInt(s.sequence, 10)

	s.rows = append(s.rows, Row{
		Tab: inboxcompliance.TabPostAudit,
		Item: inboxcompliance.WorkItem{
			CaseID:            saved.CaseID,
			ClaimNumber:       saved.ClaimNumber,
			Reference:         saved.ClaimNumber,
			InsuredName:       saved.InsuredName,
			PolicyNumber:      saved.PolicyNumber,
			ComplianceRemarks: saved.Remarks,
			PostAuditSentDate: &saved.SentAt,
		},
	})

	return saved, nil
}

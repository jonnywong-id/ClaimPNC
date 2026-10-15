// Package memory memenuhi seam inboxmanageradmin.Repo dengan penyimpanan di memori.
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
// di Oracle. Ketiga penyaring kueri ditiru sedekat-dekatnya dengan predikat SQL-nya: kelas
// objek kerja, unit organisasi penugasan, dan status kerja.
//
// Yang ditiru paling hati-hati adalah unit organisasi. Ia satu-satunya yang membedakan
// ketiga tab, dan satu-satunya kolom di modul ini yang keberadaannya di basis data belum
// terbukti (lihat catatan di inboxmanageradmin.sql). Justru karena itu penyimpanan memori
// harus membuktikan penyaringnya benar-benar dipakai — bukan mengembalikan seluruh baris
// dan kebetulan lulus.
package memory

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"claim-pnc/internal/inboxmanageradmin"
)

// WorkClassPNC adalah kelas objek kerja klaim PNC — `PXOBJCLASS`.
//
// Ia ditiru di sini karena kuerinya menyaring dengannya: tabel
// POOLDATA.T_CLAIMLIST_ADMIN memuat lebih dari satu jenis objek kerja, dan tanpa penyaring
// ini berkas penerimaan dokumen ikut bocor ke ketiga tab.
//
// Itu bukan dugaan: `inboxlaporanklaim` menggabung tabel yang sama pada kelas
// `ASM-FW-GCNMFW-Work-ReceiveDocument`, dan README mencatat 142 baris RCV di dalamnya.
// Karena RCV dan klaim sama-sama punya PYID, kebocorannya tidak akan tampak salah.
const WorkClassPNC = "ASM-FW-GCNMFW-Work-PNC"

// Status kerja yang membuat sebuah baris HILANG dari antrean.
//
// Keduanya dibawa apa adanya dari filter `B` dan `C` Report Definition.
const (
	StatusCompleted = "Resolved-Completed"
	StatusRejected  = "Resolved-Rejected"
)

// Row adalah satu baris contoh beserta kolom yang TIDAK ditampilkan tetapi menyaring.
//
// Ketiga isian tambahan tidak masuk inboxmanageradmin.WorkItem dengan sengaja: tidak satu
// pun sampai ke layar, dan menaruhnya di tipe domain akan membuat orang menduga ketiganya
// bagian dari kontrak.
type Row struct {
	Item inboxmanageradmin.WorkItem

	// WorkClass adalah kelas objek kerja — `PXOBJCLASS`.
	WorkClass string

	// OrgUnit adalah unit organisasi penugasan — `PXASSIGNEDORGUNIT` pada
	// POOLDATA.T_CLAIMLIST_ADMIN.
	//
	// Ia yang memisahkan ketiga tab satu sama lain. Perhatikan ia milik baris PENUGASAN,
	// bukan milik klaim: klaim yang sama dapat berpindah unit organisasi bila
	// penugasannya berpindah.
	OrgUnit string

	// WorkStatus adalah status kerja — `PYSTATUSWORK`.
	WorkStatus string
}

// Store adalah penyimpanan antrean di memori.
//
// Aman dipakai bersamaan: ia dibagi seluruh permintaan pada satu portal, dan uji
// menjalankan beberapa permintaan sekaligus.
type Store struct {
	mu   sync.RWMutex
	rows []Row

	// lines memetakan login petugas ke lini bisnisnya — tiruan
	// `M_LOGIN_PNC.LINE_BUSINESS`.
	//
	// Kuncinya SELALU huruf besar tanpa spasi tepi, sama seperti kueri sungguhannya
	// membandingkan `UPPER(TRIM(LOGIN_ID))`. Menyimpannya apa adanya akan membuat uji
	// lulus pada ejaan yang gagal di Oracle.
	lines map[string]string

	// defaultLine adalah lini bisnis bagi login yang tidak terdaftar di atas.
	//
	// # Kenapa ia ada, dan kenapa HANYA pada penyimpanan contoh
	//
	// Karena tanpa isian ini `PENYIMPANAN=memori` menjadi tidak dapat dipakai sama sekali:
	// login datang dari HCQ sungguhan dan tidak dapat diketahui di muka, sehingga tidak ada
	// satu pun kunci yang akan cocok — dan setiap pengembang melihat layar "tidak ada
	// antrean yang menjadi hak lini bisnis Anda", selamanya.
	//
	// Itu bukan tiruan produksi melainkan kelumpuhan. Cacat itu benar-benar terjadi pada
	// 2026-09-27, beberapa jam setelah lini bisnis dipindahkan ke seam ini.
	//
	// NewStore membiarkannya KOSONG supaya uji tetap ketat — uji yang memperoleh tab dari
	// nilai bawaan tidak membuktikan apa pun. Hanya NewSampleStore, yang memang fixture
	// pengembangan, mengisinya.
	defaultLine string
}

// NewStore membentuk penyimpanan berisi baris yang diberikan.
func NewStore(rows ...Row) *Store {
	stored := make([]Row, len(rows))
	copy(stored, rows)
	return &Store{rows: stored, lines: map[string]string{}}
}

// SetLineBusiness menetapkan lini bisnis seorang petugas.
//
// Petugas yang tidak pernah ditetapkan mengembalikan teks kosong — keadaan yang bukan
// kekecualian melainkan yang PALING UMUM di produksi hari ini, karena kolomnya baru terisi
// pada sebagian petugas (`migrations/0004_DICABUT.md`).
func (s *Store) SetLineBusiness(loginID, line string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.lines == nil {
		s.lines = map[string]string{}
	}
	s.lines[strings.ToUpper(strings.TrimSpace(loginID))] = strings.TrimSpace(line)
}

// LineBusinessFor membaca lini bisnis seorang petugas.
//
// Tidak pernah menghasilkan galat, sama seperti pengisi SQL-nya: petugas tanpa baris di
// `M_LOGIN_PNC` sekadar tidak membuka tab mana pun, persis seperti `pyPosition` yang tidak
// cocok satu pun di Pega.
func (s *Store) LineBusinessFor(_ context.Context, loginID string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if line, listed := s.lines[strings.ToUpper(strings.TrimSpace(loginID))]; listed {
		return line, nil
	}
	return s.defaultLine, nil
}

// NewSampleStore membentuk penyimpanan berisi baris contoh, siap dipakai pengembangan lokal.
//
// Berbeda dari NewStore, ia memberi lini bisnis bawaan bagi SIAPA PUN yang belum terdaftar
// — lihat Store.defaultLine. Tanpa itu `PENYIMPANAN=memori` tidak menampilkan satu tab pun
// kepada siapa pun, karena login datang dari HCQ sungguhan dan tidak dapat diketahui di muka.
//
// Non-MBU dipilih karena ia tab pertama di layar, dan karena baris contohnya paling banyak
// di sana.
func NewSampleStore() *Store {
	store := NewStore(SampleRows()...)
	store.defaultLine = inboxmanageradmin.LineNonMBU
	return store
}

// NewStrictStore membentuk penyimpanan berisi baris contoh TANPA lini bisnis bawaan.
//
// Dipakai uji yang memang ingin menguji keadaan "petugas tanpa lini bisnis" — keadaan yang
// paling umum di produksi, dan yang nilai bawaan NewSampleStore justru sembunyikan.
func NewStrictStore() *Store {
	return NewStore(SampleRows()...)
}

// List mengembalikan SELURUH baris yang cocok, belum dipaginasi.
//
// Urutannya sama dengan `ORDER BY w.PXCREATEDATETIME DESC, w.PYID` pada kueri: terbaru
// lebih dulu, dan `CaseID` sebagai pemutus seri. Tanpa pemutus seri itu, dua baris bertanggal
// sama dapat bertukar tempat antar pemanggilan — dan halaman yang dipotong dari urutan yang
// berubah-ubah membuat satu baris muncul dua kali.
func (s *Store) List(
	_ context.Context,
	q inboxmanageradmin.Query,
) ([]inboxmanageradmin.WorkItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	matched := []inboxmanageradmin.WorkItem{}
	for _, row := range s.rows {
		if !row.matches(q.Tab.OrgUnit) {
			continue
		}

		// Status Klaim DITURUNKAN di sini pula, dari isian yang sama dengan kueri
		// sungguhannya — `WorkStatus`, tiruan `PYSTATUSWORK`.
		//
		// Menyimpannya sebagai teks siap pakai di data contoh akan membuat fixture ini
		// menyimpang dari SQL tanpa satu pun uji gagal: isinya dikarang dari kode yang
		// sedang diuji. Kekeliruan persis itu sudah pernah terjadi di modul Inbox
		// Outstanding, dan yang membongkarnya satu baris data produksi.
		item := row.Item
		item.ClaimStatus = inboxmanageradmin.DisplayStatusFor(row.WorkStatus)

		matched = append(matched, item)
	}

	sort.SliceStable(matched, func(i, j int) bool {
		left, right := matched[i], matched[j]
		if !sameTime(left.RegisteredAt, right.RegisteredAt) {
			return laterThan(left.RegisteredAt, right.RegisteredAt)
		}
		return left.CaseID < right.CaseID
	})

	return matched, nil
}

// matches meniru ketiga penyaring kueri, berurutan seperti di sana.
func (r Row) matches(orgUnit string) bool {
	if r.WorkClass != WorkClassPNC {
		return false
	}
	// Perbandingan PERSIS, tidak memandang spasi maupun huruf besar-kecil — sama seperti
	// `=` pada Oracle. Melonggarkannya di sini akan membuat uji lulus untuk nilai yang
	// justru ditolak basis data.
	if r.OrgUnit != orgUnit {
		return false
	}
	if r.WorkStatus == StatusCompleted || r.WorkStatus == StatusRejected {
		return false
	}
	return true
}

// sameTime membandingkan dua tanggal yang boleh kosong.
func sameTime(left, right *time.Time) bool {
	switch {
	case left == nil && right == nil:
		return true
	case left == nil || right == nil:
		return false
	default:
		return left.Equal(*right)
	}
}

// laterThan menyatakan tanggal kiri lebih baru daripada kanan.
//
// Tanggal kosong dianggap PALING LAMA, sehingga baris tanpa tanggal pendaftaran tenggelam
// ke bawah alih-alih memimpin daftar yang diurutkan menurun.
func laterThan(left, right *time.Time) bool {
	switch {
	case left == nil:
		return false
	case right == nil:
		return true
	default:
		return left.After(*right)
	}
}

// OrgUnits menyebut unit organisasi yang benar-benar ada di penyimpanan ini.
//
// Dipakai uji untuk memastikan data contohnya mencakup ketiga tab. Hasilnya diurutkan
// supaya dapat dibandingkan langsung.
func (s *Store) OrgUnits() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	seen := map[string]struct{}{}
	for _, row := range s.rows {
		if unit := strings.TrimSpace(row.OrgUnit); unit != "" {
			seen[unit] = struct{}{}
		}
	}

	result := make([]string, 0, len(seen))
	for unit := range seen {
		result = append(result, unit)
	}
	sort.Strings(result)
	return result
}

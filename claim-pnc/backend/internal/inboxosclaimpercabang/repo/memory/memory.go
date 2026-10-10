// Package memory memenuhi seam inboxosclaimpercabang.Repo tanpa basis data.
//
// Ia dipakai pengembangan lokal dan pengujian. Keberadaannya bukan kenyamanan: seam Repository
// baru nyata bila punya dua pengisi, dan pengisi kedua inilah yang membuat aturan modul dapat
// diuji tanpa Oracle sama sekali (`04-FUTURE-ARCHITECTURE.md` §3.1).
//
// Penyaring dan urutannya ditiru dari berkas .sql, bukan dikarang: baris disaring menurut kode
// cabang dan diurutkan menurut tanggal registrasi menaik — urutan yang sama dengan umur
// menurun.
package memory

import (
	"context"
	"sort"
	"strings"
	"sync"

	"claim-pnc/internal/inboxosclaimpercabang"
)

// Store adalah penyimpanan di memori untuk satu portal.
type Store struct {
	mutex sync.RWMutex

	// rows adalah seluruh baris, termasuk yang tidak outstanding.
	//
	// Baris yang tidak outstanding sengaja ikut disimpan: penyaringnya harus dapat DIUJI,
	// dan penyimpanan yang hanya memuat baris yang lolos tidak pernah membuktikan
	// penyaringnya bekerja.
	rows []row

	// branches memetakan kode cabang RINCI ke cabangnya, meniru POOLDATA.BRANCH.
	//
	// Kuncinya `OLDID` — kode 3 digit yang dikirim HCQ sebagai `DetailBranchCode` — dan
	// nilainya membawa `ID` sekaligus `BRANCHNAME`. Bentuknya sengaja sama dengan yang
	// dikembalikan kueri `branch_of`: penyimpanan uji yang kuncinya berbeda dari penyimpanan
	// sungguhan akan meloloskan cacat yang justru paling ingin ditangkap di sini.
	branches map[string]inboxosclaimpercabang.Branch

	// factors memetakan kunci internal klaim ke faktor dominannya, sudah berurut.
	factors map[string][]string
}

// row adalah satu baris tersimpan beserta hal-hal yang menentukan ia tampil atau tidak.
type row struct {
	item inboxosclaimpercabang.ExportRow

	// outstanding meniru penyaring
	// `pystatuswork NOT IN ('Resolved-Rejected', 'Resolved-Completed')`.
	outstanding bool

	// detail memuat yang HANYA tampil di popup — objek, riwayat progres, dan komunikasi.
	//
	// Ia melekat pada baris, bukan disimpan sebagai peta tersendiri, supaya baris yang tidak
	// lolos penyaring cabang tidak punya isi popup yang dapat diambil tanpa melewatinya.
	detail inboxosclaimpercabang.Detail
}

// NewStore membentuk penyimpanan kosong.
func NewStore() *Store {
	return &Store{
		branches: map[string]inboxosclaimpercabang.Branch{},
		factors:  map[string][]string{},
	}
}

// BranchOf menerjemahkan kode cabang rinci menjadi cabang, atau false bila tidak dikenal.
func (s *Store) BranchOf(
	_ context.Context,
	detailBranchCode string,
) (inboxosclaimpercabang.Branch, bool, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	clean := strings.TrimSpace(detailBranchCode)
	if clean == "" {
		return inboxosclaimpercabang.Branch{}, false, nil
	}

	branch, known := s.branches[clean]
	return branch, known, nil
}

// List mengembalikan satu halaman baris grid.
func (s *Store) List(
	_ context.Context,
	query inboxosclaimpercabang.Query,
	page inboxosclaimpercabang.Pagination,
) (inboxosclaimpercabang.Page, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// Pencarian disaring DI SINI, bukan di `matching`. `matching` dipakai bersama
	// ListForExport, dan berkas ekspor sengaja TIDAK mengikuti kotak cari — menaruhnya di
	// sana akan diam-diam mengubah isi berkas.
	items := make([]inboxosclaimpercabang.WorkItem, 0, len(s.rows))
	for _, r := range s.matching(query) {
		if !cocok(r.item.WorkItem, query.Search) {
			continue
		}
		items = append(items, r.item.WorkItem)
	}

	return inboxosclaimpercabang.Slice(items, page), nil
}

// cocok memutuskan apakah satu baris lolos kotak cari.
//
// Dicocokkan ke NOMOR KLAIM dan NOMOR POLIS, tanpa membedakan besar kecil huruf — sama
// dengan `UPPER(...) LIKE '%...%'` pada pengisi SQL. Kecocokan sebagian, bukan sama persis:
// penyelia mengetik beberapa angka terakhir nomor polis, bukan seluruhnya.
func cocok(item inboxosclaimpercabang.WorkItem, search string) bool {
	needle := strings.ToUpper(strings.TrimSpace(search))
	if needle == "" {
		return true
	}
	return strings.Contains(strings.ToUpper(item.ClaimNumber), needle) ||
		strings.Contains(strings.ToUpper(item.PolicyNumber), needle)
}

// ListForExport mengembalikan satu halaman baris berkas ekspor.
func (s *Store) ListForExport(
	_ context.Context,
	query inboxosclaimpercabang.Query,
	page inboxosclaimpercabang.Pagination,
) (inboxosclaimpercabang.ExportPage, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	matched := s.matching(query)
	clean := page.Normalize()

	result := inboxosclaimpercabang.ExportPage{
		Items:      []inboxosclaimpercabang.ExportRow{},
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

	for _, r := range matched[offset:end] {
		result.Items = append(result.Items, r.item)
	}
	return result, nil
}

// DominantFactors mengembalikan faktor dominan klaim outstanding satu cabang.
//
// Ia menyaring menurut cabang yang sama dengan daftarnya, bukan mengembalikan seluruh isi
// peta. Pengisi SQL pun begitu, dan penyimpanan yang lebih longgar akan membuat uji lolos
// terhadap perilaku yang tidak dimiliki pengisi sebenarnya.
func (s *Store) DominantFactors(
	_ context.Context,
	query inboxosclaimpercabang.Query,
) (map[string][]string, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	result := map[string][]string{}
	for _, r := range s.matching(query) {
		if names, exists := s.factors[r.item.ClaimKey]; exists {
			result[r.item.ClaimKey] = append([]string{}, names...)
		}
	}
	return result, nil
}

// matching memilih baris yang cocok lalu mengurutkannya, meniru WHERE dan ORDER BY di .sql.
//
// Pemanggil WAJIB sudah memegang kunci baca.
func (s *Store) matching(query inboxosclaimpercabang.Query) []row {
	branch := strings.TrimSpace(query.Branch.Code)

	matched := make([]row, 0, len(s.rows))
	for _, r := range s.rows {
		switch {
		case !r.outstanding:
			continue
		case r.item.RegisterDate == nil:
			// Meniru `WHERE c.registerdate IS NOT NULL`.
			continue
		case r.item.BranchCode != branch:
			continue
		}
		matched = append(matched, r)
	}

	// `ORDER BY c.registerdate ASC, c.claimno ASC`. Pemisah kedua ada di kedua tempat
	// supaya dua baris bertanggal sama selalu keluar dengan urutan yang sama — tanpa itu,
	// halaman kedua dapat memuat baris yang sudah tampil di halaman pertama.
	sort.SliceStable(matched, func(i, j int) bool {
		left, right := matched[i].item, matched[j].item
		if !left.RegisterDate.Equal(*right.RegisterDate) {
			return left.RegisterDate.Before(*right.RegisterDate)
		}
		return left.ClaimNumber < right.ClaimNumber
	})

	return matched
}

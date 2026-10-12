// Package memory memenuhi seam inboxprogressclaim.Repo dengan penyimpanan di memori.
//
// # Untuk apa ia ada
//
// Dua hal, dan keduanya nyata:
//
//   - Pengujian aturan modul TANPA basis data, sehingga uji aturan bisnis berjalan cepat
//     dan tidak menuntut Oracle (`14-TESTING-STRATEGY.md` §3).
//   - Pengembangan lokal saat variabel `PENYIMPANAN` tidak menunjuk basis data mana pun.
//
// # Kenapa penyaring dan urutannya ditiru, bukan disederhanakan
//
// Karena kalau tidak, uji yang lulus di sini tidak menyatakan apa pun tentang yang berjalan
// di Oracle. Yang ditiru: kata kunci yang menyentuh tiga kolom sekaligus, saringan jatuh
// tempo pada region Next Follow Up, urutan `TGL_PROSES` menaik dengan nomor klaim sebagai
// pemutus seri, dan paginasi yang memotong SETELAH seluruh baris terurut.
//
// # Yang TIDAK dapat ditiru: pencacah rekap per PIC
//
// Kelima pencacahnya adalah hasil agregasi lima subkueri terhadap riwayat progres. Menirunya
// di memori berarti menulis ulang aturan yang justru sedang diuji, dan hasilnya akan
// meyakinkan tanpa membuktikan apa pun. Di sini pencacahnya DISIMPAN apa adanya pada baris
// contoh; yang benar-benar diuji hanyalah penyaringnya — lini bisnis, pemilik, dan rentang
// tanggal.
package memory

import (
	"context"
	"sort"
	"strings"
	"time"

	"claim-pnc/internal/inboxprogressclaim"
)

// ClaimRecord adalah satu baris klaim contoh beserta kolom yang TIDAK ditampilkan tetapi
// menyaring.
//
// Kolom tambahan itu tidak masuk inboxprogressclaim.ClaimRow dengan sengaja: ia tidak
// pernah sampai ke layar, dan menaruhnya di tipe domain akan membuat orang menduga ia
// bagian dari kontrak.
type ClaimRecord struct {
	Item inboxprogressclaim.ClaimRow

	// LastFollowUpByPIC adalah tindak lanjut TERAKHIR yang dicatat PIC klaim itu sendiri.
	//
	// Ia padanan subkueri
	// `SELECT MAX(NEXT_FOLLOWUP) ... WHERE PNCCASEID = a.NOKLAIM AND USER_INPUT = a.PIC`,
	// dan hanya ia yang menentukan sebuah baris muncul di region Next Follow Up. Baris
	// yang tindak lanjut terakhirnya dicatat orang lain TIDAK muncul di sana — perilaku
	// sistem lama yang dibawa apa adanya.
	//
	// Nil berarti belum pernah ada catatan, dan baris seperti itu tidak pernah muncul di
	// Next Follow Up: pada SQL, subkuerinya menghasilkan NULL dan perbandingan
	// `NULL <= :today` bernilai UNKNOWN.
	LastFollowUpByPIC *time.Time
}

// PICRecord adalah satu baris rekap contoh beserta kolom penyaringnya.
type PICRecord struct {
	Item inboxprogressclaim.PICSummary

	// Business adalah lini bisnis tempat petugas ini terdaftar — padanan
	// `MST_USER_TEKNIK.TYPE_BUSINESS`.
	Business inboxprogressclaim.BusinessLine

	// RegisteredAt adalah tanggal registrasi klaim yang diwakili baris ini, dipakai
	// saringan rentang tanggal.
	//
	// Pada SQL, rentang itu menyaring SETIAP klaim sebelum dihitung, sehingga rentang yang
	// sempit mengecilkan pencacahnya. Di sini ia menyaring seluruh barisnya sekaligus —
	// penyederhanaan yang disengaja, dan alasannya ada di kepala paket.
	RegisteredAt *time.Time
}

// Store adalah pembaca progres klaim di memori.
type Store struct {
	claims []ClaimRecord
	pics   []PICRecord
}

// NewStore membentuk pembaca dari baris yang diberikan.
func NewStore(claims []ClaimRecord, pics []PICRecord) *Store {
	return &Store{claims: claims, pics: pics}
}

// ListClaims mengambil satu halaman baris klaim beserta jumlah seluruh baris yang cocok.
func (s *Store) ListClaims(
	_ context.Context,
	q inboxprogressclaim.ClaimQuery,
	page inboxprogressclaim.Pagination,
) (inboxprogressclaim.ClaimPage, error) {
	matched := []inboxprogressclaim.ClaimRow{}

	for _, record := range s.claims {
		if !matchesKeyword(record.Item, q.Keyword) {
			continue
		}
		if q.View == inboxprogressclaim.ViewNextFollowUp && !isDue(record, q.Today) {
			continue
		}
		matched = append(matched, record.Item)
	}

	sortClaims(matched)

	clean := page.Normalize()
	result := inboxprogressclaim.ClaimPage{
		Items:      []inboxprogressclaim.ClaimRow{},
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

	result.Items = matched[offset:end]
	return result, nil
}

// ListPICSummary mengambil rekap yang cocok dengan penyaringnya.
func (s *Store) ListPICSummary(
	_ context.Context,
	q inboxprogressclaim.PICQuery,
) ([]inboxprogressclaim.PICSummary, error) {
	result := []inboxprogressclaim.PICSummary{}

	for _, record := range s.pics {
		if record.Business != q.Business {
			continue
		}
		// Rekapnya selalu terikat pemanggil, persis seperti langkah "Progress Claim per
		// User" yang berjalan tanpa prakondisi di sistem lama.
		if !strings.EqualFold(record.Item.PIC, q.Caller.Login) {
			continue
		}
		if !withinRange(record.RegisteredAt, q.From, q.To) {
			continue
		}
		result = append(result, record.Item)
	}

	sort.SliceStable(result, func(i, j int) bool {
		return result[i].PIC < result[j].PIC
	})

	return result, nil
}

// matchesKeyword meniru saringan kata kunci: ia menyentuh nomor klaim, nomor polis, DAN
// nama PIC sekaligus.
//
// Kolom ketiga itu mudah terlewat saat membaca kuerinya, dan justru ia yang membuat
// mengetik nama seorang petugas memunculkan seluruh klaim yang ia tangani.
func matchesKeyword(item inboxprogressclaim.ClaimRow, keyword string) bool {
	needle := strings.ToUpper(strings.TrimSpace(keyword))
	if needle == "" {
		return true
	}

	for _, field := range []string{item.ClaimNumber, item.PolicyNumber, item.TechnicalPIC} {
		if strings.Contains(strings.ToUpper(field), needle) {
			return true
		}
	}
	return false
}

// isDue menyatakan tindak lanjut terakhir baris ini sudah jatuh tempo pada `today`.
func isDue(record ClaimRecord, today time.Time) bool {
	if record.LastFollowUpByPIC == nil {
		return false
	}

	last := truncate(*record.LastFollowUpByPIC)
	return !last.After(truncate(today))
}

// withinRange menyatakan sebuah tanggal berada di dalam rentang yang diminta.
//
// Batas yang tidak diisi berarti tanpa batas pada sisi itu. Baris tanpa tanggal ikut
// tersaring keluar begitu ada batas — pada SQL, perbandingan terhadap NULL bernilai UNKNOWN
// dan barisnya tidak lolos.
func withinRange(at, from, to *time.Time) bool {
	if from == nil && to == nil {
		return true
	}
	if at == nil {
		return false
	}

	value := truncate(*at)
	if from != nil && value.Before(truncate(*from)) {
		return false
	}
	if to != nil && value.After(truncate(*to)) {
		return false
	}
	return true
}

// truncate memangkas sebuah waktu menjadi tanggalnya saja.
func truncate(at time.Time) time.Time {
	return time.Date(at.Year(), at.Month(), at.Day(), 0, 0, 0, 0, time.UTC)
}

// sortClaims mengurutkan seperti kuerinya: TGL_PROSES menaik, nomor klaim sebagai pemutus
// seri.
//
// Baris tanpa tanggal proses ditempatkan di BELAKANG. Pada Oracle, `ORDER BY ... ASC`
// menempatkan NULL di belakang secara bawaan, dan menirunya di sini membuat halaman
// pertama berisi baris yang sama pada kedua penyimpanan.
func sortClaims(items []inboxprogressclaim.ClaimRow) {
	sort.SliceStable(items, func(i, j int) bool {
		left, right := items[i].ProcessDate, items[j].ProcessDate

		switch {
		case left == nil && right == nil:
			return items[i].ClaimNumber < items[j].ClaimNumber
		case left == nil:
			return false
		case right == nil:
			return true
		case left.Equal(*right):
			return items[i].ClaimNumber < items[j].ClaimNumber
		default:
			return left.Before(*right)
		}
	})
}

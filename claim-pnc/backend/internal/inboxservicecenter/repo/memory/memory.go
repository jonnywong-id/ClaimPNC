// Package memory memenuhi seam inboxservicecenter.Repo dengan penyimpanan di memori.
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
// di Oracle. Keempat penyaingnya ditiru sedekat-dekatnya dengan predikat SQL-nya — termasuk
// dua kenyataan yang mudah tergoda untuk "dirapikan":
//
//   - Pencarian adalah IRISAN dua kelompok, bukan gabungan. Mencari dengan ID saja tidak
//     menghasilkan baris, persis seperti di Pega.
//   - Paginasi mati saat mencari.
package memory

import (
	"context"
	"sort"
	"strings"

	"claim-pnc/internal/inboxservicecenter"
)

// Store adalah penyimpanan klaim portal rekanan di memori.
//
// Ia tidak dilindungi mutex karena tidak pernah berubah setelah dibentuk: modul ini hanya
// membaca, dan tidak ada satu pun operasi yang menulis.
type Store struct {
	claims   []inboxservicecenter.ServiceClaim
	details  []inboxservicecenter.ClaimDetail
	progress []inboxservicecenter.ProgressNote
}

// NewStore membentuk penyimpanan berisi baris daftar yang diberikan.
//
// Rincian dan riwayat progresnya kosong. Itu disengaja: uji yang hanya menguji daftar tidak
// perlu menyiapkan 83 isian rincian, dan rincian yang tidak disiapkan dijawab ErrNotFound —
// jawaban yang benar, bukan baris kosong yang menyamar sebagai data.
func NewStore(claims ...inboxservicecenter.ServiceClaim) *Store {
	return &Store{claims: claims}
}

// WithDetails menyerahkan salinan penyimpanan yang rincian dan riwayat progresnya terisi.
func (s *Store) WithDetails(
	details []inboxservicecenter.ClaimDetail,
	progress []inboxservicecenter.ProgressNote,
) *Store {
	clone := *s
	clone.details = details
	clone.progress = progress
	return &clone
}

// NewSampleStore membentuk penyimpanan berisi baris contoh, lengkap dengan rincian dan
// riwayat progresnya.
func NewSampleStore() *Store {
	return NewStore(SampleClaims()...).WithDetails(SampleDetails(), SampleProgress())
}

// List menyaring, mengurutkan, lalu memotong satu halaman.
//
// Urutan langkahnya sengaja sama dengan kuerinya: saring, urut, potong. Menukar urut dan
// potong akan menghasilkan halaman yang isinya benar tetapi urutannya bukan urutan yang
// dilihat pengguna.
func (s *Store) List(
	_ context.Context,
	q inboxservicecenter.Query,
	page inboxservicecenter.Pagination,
) (inboxservicecenter.Page, error) {
	matched := []inboxservicecenter.ServiceClaim{}
	for _, claim := range s.claims {
		if matches(claim, q) {
			matched = append(matched, claim)
		}
	}

	sortClaims(matched)

	clean := page.Normalize()
	result := inboxservicecenter.Page{
		Items:      []inboxservicecenter.ServiceClaim{},
		Total:      len(matched),
		Pagination: clean,
		Paginated:  q.Paginated(),
	}

	if !result.Paginated {
		// Seluruh baris yang cocok — paginasi memang mati saat mencari.
		result.Items = matched
		return result, nil
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

// matches menirukan seluruh klausa WHERE kueri, berurutan seperti di sana.
func matches(claim inboxservicecenter.ServiceClaim, q inboxservicecenter.Query) bool {
	if !matchesApproval(claim, q.Approval) {
		return false
	}

	// Penyaring PIC — berlaku pada keempat tab, dan inilah yang membuatnya "inbox saya".
	if !strings.EqualFold(strings.TrimSpace(claim.TechnicalPIC), strings.TrimSpace(q.Caller.Login)) {
		return false
	}

	if q.Keyword == "" {
		return true
	}

	keyword := strings.ToUpper(q.Keyword)

	// Kelompok pertama — Activity/DataServiceCenter-Act.xml langkah 17.
	first := contains(claim.ID, keyword) ||
		contains(claim.PolicyNumber, keyword) ||
		contains(claim.CustomerName, keyword) ||
		contains(claim.IMEI, keyword)

	// Kelompok kedua — langkah 18.
	second := contains(claim.ClaimNumber, keyword) ||
		contains(claim.PolicyNumber, keyword) ||
		contains(claim.CustomerName, keyword) ||
		contains(claim.IMEI, keyword)

	// AND, bukan OR. Keduanya potongan SQL terpisah yang sama-sama terpasang, sehingga yang
	// benar-benar dapat dicari hanyalah irisannya. Ini cacat sistem lama yang DIREPLIKASI
	// (`P-5`) — lihat inboxservicecenter.Limitations.
	return first && second
}

// matchesApproval menirukan penyaring `STS_APPROVAL` yang berbeda bentuk antar tab.
func matchesApproval(
	claim inboxservicecenter.ServiceClaim,
	filter inboxservicecenter.ApprovalFilter,
) bool {
	status := strings.TrimSpace(claim.ApprovalStatus)

	if filter.MatchNull {
		return status == ""
	}

	for _, code := range filter.Codes {
		if status == code {
			return true
		}
	}
	return false
}

// contains memeriksa satu isian memuat kata kunci, tanpa membedakan huruf besar-kecil.
func contains(value, upperKeyword string) bool {
	return strings.Contains(strings.ToUpper(value), upperKeyword)
}

// sortClaims mengurutkan sesuai `ORDER BY k.INPUTDATE DESC, k.ID` pada kuerinya.
//
// Baris tanpa tanggal input ditaruh PALING BAWAH, bukan paling atas: pada urutan menurun,
// nilai kosong yang diperlakukan sebagai tahun 1 akan berkumpul di ujung — dan di sinilah
// ujung itu harus berada supaya sama dengan Oracle, yang menaruh NULL terakhir pada `DESC`
// secara bawaan.
func sortClaims(claims []inboxservicecenter.ServiceClaim) {
	sort.SliceStable(claims, func(i, j int) bool {
		left, right := claims[i], claims[j]

		switch {
		case left.InputDate == nil && right.InputDate == nil:
			return left.ID < right.ID
		case left.InputDate == nil:
			return false
		case right.InputDate == nil:
			return true
		case left.InputDate.Equal(*right.InputDate):
			return left.ID < right.ID
		default:
			return left.InputDate.After(*right.InputDate)
		}
	})
}

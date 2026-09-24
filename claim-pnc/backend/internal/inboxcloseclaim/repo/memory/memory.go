// Package memory memenuhi seam inboxcloseclaim.Repo dan RequestRepo di dalam memori.
//
// # Untuk apa ia ada
//
// Dua hal, dan keduanya nyata:
//
//   - PENGUJIAN tanpa basis data. Seluruh aturan modul ini — penyaring, paginasi,
//     pencegahan permintaan ganda — dapat diuji tanpa Oracle, sehingga ujinya berjalan
//     setiap kali berkas disimpan.
//   - MENJALANKAN APLIKASI tanpa Oracle, supaya layarnya dapat dilihat dan dinilai Work
//     Owner sebelum DBA menjalankan migrasi `0006`.
//
// # Aturan yang mengikat berkas ini
//
// Predikat di sini WAJIB sedekat-dekatnya dengan predikat SQL-nya. Bila keduanya menyimpang,
// uji yang lulus di atas memori tidak menyatakan apa pun tentang perilaku terhadap Oracle —
// dan itulah kegagalan yang paling mahal, karena ia terlihat seperti keberhasilan.
//
// Dua tempat yang paling mudah menyimpang, dan keduanya ditiru tegas:
//
//   - penyaring BONDING memakai `IN`, bukan `NOT IN` seperti modul Inbox Outstanding;
//   - penyaring "BELUM LUNAS" memakai `<>` terhadap `STATUSCLAIM_1`, yang di Oracle
//     MEMBUANG baris ber-NULL. Lihat unpaidMatches.
package memory

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"sort"
	"strings"
	"sync"
	"time"

	"claim-pnc/internal/inboxcloseclaim"
)

// Store menyimpan klaim tutup beserta permintaan atasnya.
type Store struct {
	mu       sync.RWMutex
	claims   []inboxcloseclaim.ClosedClaim
	requests []inboxcloseclaim.ClaimRequest
}

// NewStore membentuk penyimpanan kosong.
func NewStore() *Store { return &Store{} }

// NewStoreWithSamples membentuk penyimpanan berisi data contoh.
func NewStoreWithSamples() *Store {
	store := NewStore()
	store.claims = SampleClaims()
	return store
}

// List menyaring, mengurutkan, dan memotong halaman — meniru kedua kueri SQL.
func (s *Store) List(
	_ context.Context,
	f inboxcloseclaim.Filter,
) (inboxcloseclaim.Page, error) {
	f = f.Normalize()

	s.mu.RLock()
	defer s.mu.RUnlock()

	matched := make([]inboxcloseclaim.ClosedClaim, 0, len(s.claims))
	for _, claim := range s.claims {
		if matches(claim, f) {
			matched = append(matched, claim)
		}
	}

	// Urutan mengikuti `ORDER BY A.PXCREATEDATETIME ASC, A.PZINSKEY` — termasuk kunci
	// keduanya. Tanpa kunci kedua, dua klaim berwaktu sama dapat bertukar tempat antar
	// permintaan, dan satu di antaranya hilang dari paginasi sementara yang lain muncul
	// dua kali.
	sort.SliceStable(matched, func(i, j int) bool {
		if !matched[i].RegisteredAt.Equal(matched[j].RegisteredAt) {
			return matched[i].RegisteredAt.Before(matched[j].RegisteredAt)
		}
		return matched[i].ClaimID < matched[j].ClaimID
	})

	total := len(matched)
	if f.Offset >= total {
		return inboxcloseclaim.Page{Claims: []inboxcloseclaim.ClosedClaim{}, Total: total}, nil
	}
	end := f.Offset + f.Limit
	if end > total {
		end = total
	}

	page := make([]inboxcloseclaim.ClosedClaim, end-f.Offset)
	copy(page, matched[f.Offset:end])
	return inboxcloseclaim.Page{Claims: page, Total: total}, nil
}

// ClosedClaimNumber meniru kueri close_claim_exists.
func (s *Store) ClosedClaimNumber(_ context.Context, claimID string) (string, error) {
	key := strings.TrimSpace(claimID)

	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, claim := range s.claims {
		if claim.ClaimID == key && isClosed(claim) {
			return claim.ClaimNumber, nil
		}
	}
	return "", inboxcloseclaim.ErrClaimNotFound
}

// Record menyimpan satu permintaan, menolak yang kedua selama yang pertama menunggu.
//
// Penolakan itu meniru constraint unik pada migrasi `0006`, bukan sekadar pemeriksaan
// tambahan: bila keduanya berbeda, uji di atas memori akan meloloskan keadaan yang Oracle
// tolak — atau sebaliknya.
func (s *Store) Record(_ context.Context, request inboxcloseclaim.ClaimRequest) error {
	request = request.Normalize()

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, existing := range s.requests {
		if existing.Status != inboxcloseclaim.RequestPending {
			continue
		}
		if existing.Kind == request.Kind && existing.ClaimID == request.ClaimID {
			return inboxcloseclaim.ErrRequestPending
		}
	}

	s.requests = append(s.requests, request)
	return nil
}

// PendingFor mengembalikan permintaan yang masih menunggu, dikunci ClaimID.
func (s *Store) PendingFor(
	_ context.Context,
	claimIDs []string,
) (map[string][]inboxcloseclaim.ClaimRequest, error) {
	wanted := map[string]bool{}
	for _, id := range claimIDs {
		if clean := strings.TrimSpace(id); clean != "" {
			wanted[clean] = true
		}
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	result := map[string][]inboxcloseclaim.ClaimRequest{}
	for _, request := range s.requests {
		if request.Status != inboxcloseclaim.RequestPending || !wanted[request.ClaimID] {
			continue
		}
		result[request.ClaimID] = append(result[request.ClaimID], request)
	}
	return result, nil
}

// Requests mengembalikan seluruh permintaan yang tersimpan — dipakai uji.
func (s *Store) Requests() []inboxcloseclaim.ClaimRequest {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]inboxcloseclaim.ClaimRequest, len(s.requests))
	copy(result, s.requests)
	return result
}

// AddClaim menambahkan satu klaim — dipakai uji.
func (s *Store) AddClaim(claim inboxcloseclaim.ClosedClaim) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.claims = append(s.claims, claim)
}

// isClosed meniru `PYSTATUSWORK IN ('Resolved-Completed','Resolved-Rejected')`.
//
// Penyimpanan ini dapat berisi klaim berstatus apa pun — uji memang memasukkannya — dan
// penyaring inilah yang menjaga layar tetap berisi klaim tutup saja.
func isClosed(claim inboxcloseclaim.ClosedClaim) bool {
	switch strings.TrimSpace(claim.ProcessStatus) {
	case inboxcloseclaim.StatusKerjaSelesai, inboxcloseclaim.StatusKerjaDitolak:
		return true
	default:
		return false
	}
}

func matches(claim inboxcloseclaim.ClosedClaim, f inboxcloseclaim.Filter) bool {
	if !isClosed(claim) {
		return false
	}
	// `A.BRANCHNAME <> 'ASNET' OR A.BRANCHNAME IS NULL` — cabang ASNET dibuang, tetapi
	// cabang yang KOSONG tetap tampil. Tanpa cabang kedua itu, setiap klaim yang cabangnya
	// belum terisi akan hilang dari layar karena aritmetika tiga-nilai SQL, bukan karena
	// dikecualikan.
	if strings.EqualFold(strings.TrimSpace(claim.BranchName), "ASNET") {
		return false
	}

	if f.Search != "" &&
		!containsFold(claim.PolicyNumber, f.Search) &&
		!containsFold(claim.ClaimNumber, f.Search) {
		return false
	}
	if f.PolicyNumber != "" && !containsFold(claim.PolicyNumber, f.PolicyNumber) {
		return false
	}
	if f.ClaimNumber != "" && !containsFold(claim.ClaimNumber, f.ClaimNumber) {
		return false
	}
	if f.TechnicalPIC != "" && !containsFold(claim.TechnicalPIC, f.TechnicalPIC) {
		return false
	}
	if !businessMatches(claim, f.Business) {
		return false
	}

	switch f.Transfer {
	case inboxcloseclaim.TransferDone:
		if !claim.TransferredToCashier {
			return false
		}
	case inboxcloseclaim.TransferNone:
		if claim.TransferredToCashier {
			return false
		}
	}

	switch f.Payment {
	case inboxcloseclaim.PaymentPaid:
		if strings.TrimSpace(claim.ClaimStatusCode) != inboxcloseclaim.KodeStatusLunas {
			return false
		}
	case inboxcloseclaim.PaymentUnpaid:
		if !unpaidMatches(claim) {
			return false
		}
	}
	return true
}

// unpaidMatches meniru `A.STATUSCLAIM_1 <> '1163'` APA ADANYA, termasuk perilaku NULL-nya.
//
// # Kenapa baris tanpa kode status DIBUANG
//
// Di Oracle, `NULL <> '1163'` bernilai UNKNOWN, dan baris ber-UNKNOWN dibuang WHERE. Jadi
// klaim yang `STATUSCLAIM_1`-nya kosong TIDAK muncul saat penyaring "BELUM LUNAS" dipakai —
// meski secara bisnis ia jelas belum lunas.
//
// Itu perilaku sistem lama, dan `P-5` menetapkan perilaku dipertahankan lebih dulu.
//
// # DIKONFIRMASI Work Owner 2026-09-23: perilaku ini BENAR, bukan cacat
//
// Ditanyakan tegas — apakah klaim tanpa kode status seharusnya ikut terbaca sebagai belum
// lunas — dan jawabannya **tidak**. Klaim yang kode statusnya kosong memang BUKAN "belum
// lunas"; ia klaim yang keadaan bayarnya belum diketahui sama sekali, dan menampilkannya di
// bawah penyaring "Belum Lunas" akan menyatakan sesuatu yang tidak diketahui sebagai fakta.
//
// Dengan begitu pertanyaan terbukanya TERTUTUP: yang berlaku adalah perilaku Pega, dan ia
// berlaku karena benar — bukan sekadar karena `P-5` menuntut peniruan.
//
// Ditiru tegas di sini supaya keadaan itu menjadi keputusan yang tercatat dan diuji, bukan
// kebetulan yang kelak "diperbaiki" seseorang tanpa menyadari ia mengubah hasil.
func unpaidMatches(claim inboxcloseclaim.ClosedClaim) bool {
	code := strings.TrimSpace(claim.ClaimStatusCode)
	if code == "" {
		return false
	}
	return code != inboxcloseclaim.KodeStatusLunas
}

// businessMatches meniru penyaring lini bisnis, termasuk kedua perbedaannya terhadap modul
// lain — lihat `inboxcloseclaim.BusinessLine`.
func businessMatches(claim inboxcloseclaim.ClosedClaim, line inboxcloseclaim.BusinessLine) bool {
	panel := strings.TrimSpace(claim.GroupPanel)
	group := strings.TrimSpace(claim.BusinessGroupID)

	switch line {
	case inboxcloseclaim.BusinessAll, "":
		return true
	case inboxcloseclaim.BusinessNonMBU:
		return contains([]string{"003", "004", "006"}, panel) &&
			!contains(bondingGroups, group)
	case inboxcloseclaim.BusinessBonding:
		// `IN`, bukan `NOT IN`. Modul Inbox Outstanding memakai arah yang BERLAWANAN pada
		// pilihan yang bernama sama, dan menyeragamkan keduanya akan menampilkan lini yang
		// salah tanpa satu pun galat.
		return contains(bondingGroups, group)
	case inboxcloseclaim.BusinessPA:
		return panel == "002"
	case inboxcloseclaim.BusinessTravel:
		return panel == "005"
	default:
		return false
	}
}

// bondingGroups adalah kelompok bisnis Bonding.
//
// Keempat nilainya disalin apa adanya dari `Activity/GCNMGetManagerReopenCase_Act-Act.xml`.
// Artinya tidak diketahui — ia kode di master `businessgroup` yang tidak ada di export —
// sehingga nilainya dipertahankan tanpa ditafsirkan.
var bondingGroups = []string{"10008", "10010", "10015", "10023"}

func contains(values []string, value string) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}

// containsFold meniru `UPPER(kolom) LIKE '%…%'`.
func containsFold(haystack, needle string) bool {
	return strings.Contains(
		strings.ToUpper(strings.TrimSpace(haystack)),
		strings.ToUpper(strings.TrimSpace(needle)),
	)
}

// IDGenerator membangkitkan pengenal permintaan 128 bit dalam heksadesimal.
//
// Ia dipakai PRODUKSI, bukan hanya uji — sama seperti `komite/repo/memory.IDGenerator`.
// Letaknya di paket memory karena ia tidak menyentuh basis data sama sekali, bukan karena
// ia tiruan.
//
// Acak, bukan berurut: pengenal permintaan tidak punya makna bisnis dan tidak boleh
// membocorkan berapa banyak permintaan yang sudah tercatat.
type IDGenerator struct{}

// New mengembalikan pengenal berikutnya.
func (IDGenerator) New() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// crypto/rand tidak gagal pada sistem yang sehat. Bila ia gagal, melanjutkan dengan
		// pengenal yang dapat ditebak lebih berbahaya daripada berhenti.
		panic("inboxcloseclaim/memory: sumber acak tidak tersedia: " + err.Error())
	}
	return hex.EncodeToString(b[:])
}

// FixedClock adalah jam yang tidak bergerak — dipakai uji.
type FixedClock struct{ At time.Time }

// Now mengembalikan waktu tetap.
func (c FixedClock) Now() time.Time { return c.At }

var (
	_ inboxcloseclaim.Repo        = (*Store)(nil)
	_ inboxcloseclaim.RequestRepo = (*Store)(nil)
	_ inboxcloseclaim.IDGenerator = IDGenerator{}
	_ inboxcloseclaim.Clock       = FixedClock{}
)

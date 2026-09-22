// Package memory memenuhi seam inboxoutstanding.Repo dan LineBusinessRepo di dalam
// proses, tanpa basis data.
//
// Ia dipakai dua keadaan: pengembangan lokal (`PENYIMPANAN=memori`) dan seluruh pengujian.
// Keduanya berjalan tanpa Oracle dan tanpa jaringan.
//
// # Kenapa penyaringannya ditulis ulang di sini, bukan disederhanakan
//
// Godaannya besar untuk membuat adapter memori "asal jalan" — mengembalikan semua baris
// dan membiarkan layar menyaring. Itu akan membuat pengujian membuktikan hal yang salah:
// yang diuji menjadi penyaring di layar, bukan penyaring yang sesungguhnya berjalan di
// produksi.
//
// Karena itu aturan penyaringan di sini menirukan sqlstore sedekat mungkin, dan setiap
// perbedaan yang tidak terhindarkan disebut di komentarnya.
package memory

import (
	"context"
	"sort"
	"strings"

	"claim-pnc/internal/inboxoutstanding"
)

// Repo menyimpan klaim di memori.
type Repo struct {
	claims []inboxoutstanding.OutstandingClaim

	// lines memetakan LOGIN_ID ke nilai LINEBUSINESS.
	//
	// Kuncinya disimpan dalam huruf besar supaya pencocokannya tidak peka besar-kecil —
	// sama dengan sqlstore, yang memakai UPPER(TRIM(...)) karena kolom LOGIN_ID di sistem
	// lama tidak diseragamkan.
	lines map[string]string
}

// NewRepo membentuk repo kosong.
func NewRepo() *Repo {
	return &Repo{lines: map[string]string{}}
}

// NewRepoWithSamples membentuk repo berisi klaim contoh.
//
// Data contohnya KARANGAN — bukan data nasabah nyata. `D-69` melarang nomor polis, nama
// tertanggung, dan nomor klaim sungguhan ditulis di berkas yang di-commit.
func NewRepoWithSamples() *Repo {
	r := NewRepo()
	r.claims = sampleClaims()
	r.lines = sampleLines()
	return r
}

// SetLineBusiness menetapkan lini bisnis seorang pengguna. Dipakai pengujian.
func (r *Repo) SetLineBusiness(loginID, lineBusiness string) {
	r.lines[normalize(loginID)] = lineBusiness
}

// Add menambahkan klaim. Dipakai pengujian.
func (r *Repo) Add(claims ...inboxoutstanding.OutstandingClaim) {
	r.claims = append(r.claims, claims...)
}

// LineBusinessFor memenuhi inboxoutstanding.LineBusinessRepo.
//
// Pengguna yang tidak terdaftar mengembalikan string kosong TANPA galat — itu keadaan
// biasa, bukan kegagalan. Lihat komentar seam-nya.
func (r *Repo) LineBusinessFor(_ context.Context, loginID string) (string, error) {
	return r.lines[normalize(loginID)], nil
}

// List memenuhi inboxoutstanding.Repo.
func (r *Repo) List(_ context.Context, f inboxoutstanding.Filter) (inboxoutstanding.Page, error) {
	f = f.Normalize()

	matched := make([]inboxoutstanding.OutstandingClaim, 0, len(r.claims))
	for _, c := range r.claims {
		if matches(c, f) {
			matched = append(matched, c)
		}
	}

	// Urutan mengikuti sistem lama: `ORDER BY a.pxCreateDateTime DESC` —
	// `RDB List/BrowseInboxOutstanding1-SQL.xml:130`. Yang terbaru di atas.
	//
	// ClaimID menjadi pemutus seri supaya urutannya STABIL. Tanpa itu, dua klaim dengan
	// tanggal pendaftaran identik dapat bertukar tempat antar permintaan, dan barisnya
	// tampak melompat saat pengguna berpindah halaman.
	sort.SliceStable(matched, func(i, j int) bool {
		if matched[i].RegisteredAt.Equal(matched[j].RegisteredAt) {
			return matched[i].ClaimID < matched[j].ClaimID
		}
		return matched[i].RegisteredAt.After(matched[j].RegisteredAt)
	})

	total := len(matched)

	// Paginasi dilakukan SETELAH menghitung total, supaya angka yang dilaporkan adalah
	// jumlah seluruh yang cocok — bukan jumlah pada halaman ini.
	if f.Offset >= total {
		return inboxoutstanding.Page{Claims: []inboxoutstanding.OutstandingClaim{}, Total: total}, nil
	}
	end := f.Offset + f.Limit
	if end > total {
		end = total
	}

	page := make([]inboxoutstanding.OutstandingClaim, end-f.Offset)
	copy(page, matched[f.Offset:end])

	return inboxoutstanding.Page{Claims: page, Total: total}, nil
}

// matches menerapkan seluruh penyaring pada satu klaim.
func matches(c inboxoutstanding.OutstandingClaim, f inboxoutstanding.Filter) bool {
	if !withinScope(c, f.Scope) {
		return false
	}
	if f.Stage != "" && !strings.EqualFold(c.CurrentStage, f.Stage) {
		return false
	}
	if f.BranchCode != "" && !strings.EqualFold(c.BranchName, f.BranchCode) {
		return false
	}
	if f.Search != "" && !matchesSearch(c, f.Search) {
		return false
	}
	return true
}

// withinScope memeriksa batas data.
//
// Inilah aturan yang paling berbahaya bila salah, karena kesalahannya tidak menghasilkan
// galat — hanya baris yang seharusnya tidak terlihat.
func withinScope(c inboxoutstanding.OutstandingClaim, scope inboxoutstanding.LineScope) bool {
	if scope.Unrestricted {
		return true
	}
	if len(scope.GroupPanels) == 0 && len(scope.ExcludedBusinessGroups) == 0 {
		// Batas yang tidak menyebut apa pun, dan tidak pula menyatakan dirinya tanpa
		// batas, tidak meloloskan apa pun. Gagal TERTUTUP: sebuah scope kosong hampir
		// pasti cacat pemrograman, dan meloloskan semuanya akan mengubah cacat itu
		// menjadi kebocoran data yang senyap.
		return false
	}

	// Penyaring lini hanya berlaku bila scope menyebutkannya. BONDING sengaja tidak
	// menyebut satu pun Group Panel — seluruh aturannya ada pada pengecualian di bawah.
	if len(scope.GroupPanels) > 0 {
		cocok := false
		for _, panel := range scope.GroupPanels {
			if c.GroupPanel == panel {
				cocok = true
				break
			}
		}
		if !cocok {
			return false
		}
	}

	// Kelompok bisnis yang KOSONG tidak pernah dikecualikan — sama dengan sisi SQL, yang
	// memeriksa `BUSINESSGROUPID IS NULL` lebih dulu supaya baris tanpa kelompok tidak
	// hilang oleh aritmetika tiga-nilai.
	if c.BusinessGroupID != "" {
		for _, group := range scope.ExcludedBusinessGroups {
			if c.BusinessGroupID == group {
				return false
			}
		}
	}
	return true
}

// matchesSearch mencari pada tiga field sekaligus, meniru kotak tunggal layar lama yang
// berlabel "No Klaim / No Polis / PIC".
func matchesSearch(c inboxoutstanding.OutstandingClaim, search string) bool {
	needle := strings.ToLower(search)
	for _, field := range []string{c.ClaimNumber, c.PolicyNumber, c.TechnicalPIC} {
		if strings.Contains(strings.ToLower(field), needle) {
			return true
		}
	}
	return false
}

func normalize(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

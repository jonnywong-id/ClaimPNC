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

	// lines memetakan LOGIN_ID ke lini bisnisnya — padanan `M_LOGIN_PNC.LINE_BUSINESS`.
	//
	// Ia HANYA dipakai export. Daftar tidak mengenal lini bisnis sama sekali.
	lines map[string]inboxoutstanding.LineBusiness

	// legacy memetakan identitas login ke identitas LAMA orang yang sama — padanan
	// `T_ACCESS_GROUP_PNC.OPERATOR_ID` → `OLD_OPERATOR_ID`.
	legacy map[string]string
}

// NewRepo membentuk repo kosong.
func NewRepo() *Repo {
	return &Repo{
		lines:  map[string]inboxoutstanding.LineBusiness{},
		legacy: map[string]string{},
	}
}

// NewRepoWithSamples membentuk repo berisi klaim contoh.
//
// Data contohnya KARANGAN — bukan data nasabah nyata. `D-69` melarang nomor polis, nama
// tertanggung, dan nomor klaim sungguhan ditulis di berkas yang di-commit.
func NewRepoWithSamples() *Repo {
	r := NewRepo()
	r.claims = sampleClaims()
	return r
}

// Add menambahkan klaim. Dipakai pengujian.
func (r *Repo) Add(claims ...inboxoutstanding.OutstandingClaim) {
	r.claims = append(r.claims, claims...)
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

// SetLineBusiness menetapkan lini bisnis seorang petugas. Dipakai pengujian.
func (r *Repo) SetLineBusiness(loginID string, line inboxoutstanding.LineBusiness) {
	if r.lines == nil {
		r.lines = map[string]inboxoutstanding.LineBusiness{}
	}
	r.lines[normalize(loginID)] = line
}

// SummarizeDocumentStatus memenuhi inboxoutstanding.Repo.
//
// Penyaring status sengaja DIKOSONGKAN sebelum menghitung — ringkasan harus memuat seluruh
// status, bukan hanya yang sedang dipilih. Lihat catatan pada kueri produksinya.
func (r *Repo) SummarizeDocumentStatus(_ context.Context, f inboxoutstanding.Filter) (inboxoutstanding.Summary, error) {
	f = f.Normalize()
	if f.AssignedTo == "" {
		return inboxoutstanding.Summary{}, inboxoutstanding.ErrAssigneeRequired
	}
	f.DocumentStatus = ""

	var lengkap, belum int
	for _, c := range r.claims {
		if !matches(c, f) {
			continue
		}
		if documentStatusOf(c) == inboxoutstanding.StatusComplete {
			lengkap++
			continue
		}
		belum++
	}

	return inboxoutstanding.BuildSummary(map[inboxoutstanding.DocumentStatus]int{
		inboxoutstanding.StatusComplete:   lengkap,
		inboxoutstanding.StatusIncomplete: belum,
		inboxoutstanding.StatusAll:        lengkap + belum,
	}, lengkap+belum), nil
}

// documentStatusOf menirukan `DOKUMENLENGKAP_1 = '1'` versus `'0' atau NULL`.
//
// OutstandingClaim tidak memuat kolom itu — ia memuat apa yang ditampilkan. Yang dipakai
// di sini adalah DocumentComplete pada klaim contoh, yang sengaja ditambahkan supaya
// adapter memori dapat membedakan keduanya tanpa membocorkan nama kolom ke domain.
func documentStatusOf(c inboxoutstanding.OutstandingClaim) inboxoutstanding.DocumentStatus {
	if c.DocumentComplete {
		return inboxoutstanding.StatusComplete
	}
	return inboxoutstanding.StatusIncomplete
}

// SetLegacyOperator menetapkan identitas lama seorang petugas. Dipakai pengujian.
func (r *Repo) SetLegacyOperator(loginID, legacy string) {
	if r.legacy == nil {
		r.legacy = map[string]string{}
	}
	r.legacy[normalize(loginID)] = normalize(legacy)
}

// LegacyOperatorFor memenuhi inboxoutstanding.Repo.
func (r *Repo) LegacyOperatorFor(_ context.Context, loginID string) (string, error) {
	return r.legacy[normalize(loginID)], nil
}

// LineBusinessFor memenuhi inboxoutstanding.Repo.
//
// Petugas yang tidak terdaftar mengembalikan LineUnknown tanpa galat — sama seperti
// sqlstore, dan sama seperti Pega memperlakukan pyPosition yang tidak cocok satu pun.
func (r *Repo) LineBusinessFor(_ context.Context, loginID string) (inboxoutstanding.LineBusiness, error) {
	return r.lines[normalize(loginID)], nil
}

// Export memenuhi inboxoutstanding.Repo — TANPA menyaring pemilik pekerjaan.
//
// # Dua penyaring produksi yang TIDAK dapat ditirukan di sini
//
// `my_inbox_export` juga menyaring `PXFLOWNAME`, `ISPENDINGCLOSE`, dan `BUSINESSGROUPID`.
// Ketiga kolom itu TIDAK ada pada OutstandingClaim — ia memuat apa yang ditampilkan, bukan
// salinan utuh baris. Menambahkannya semata demi adapter memori berarti membocorkan detail
// penyimpanan ke lapisan domain.
//
// Akibatnya disebut terang-terangan, bukan disembunyikan: cakupan **BONDING** dan kedua
// penyaring itu hanya terbukti terhadap Oracle, bukan di sini. Pengujian yang
// menyangkutnya harus dijalankan terhadap basis data sungguhan.
func (r *Repo) Export(_ context.Context, f inboxoutstanding.ExportFilter) (inboxoutstanding.Page, error) {
	f = f.Normalize()

	matched := make([]inboxoutstanding.OutstandingClaim, 0, len(r.claims))
	for _, c := range r.claims {
		if matchesExport(c, f) {
			matched = append(matched, c)
		}
	}

	sort.SliceStable(matched, func(i, j int) bool {
		if matched[i].RegisteredAt.Equal(matched[j].RegisteredAt) {
			return matched[i].ClaimID < matched[j].ClaimID
		}
		return matched[i].RegisteredAt.After(matched[j].RegisteredAt)
	})

	total := len(matched)
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

// matchesExport menerapkan cakupan lini bisnis dan rentang tanggal.
func matchesExport(c inboxoutstanding.OutstandingClaim, f inboxoutstanding.ExportFilter) bool {
	// Status berjalan disaring kueri produksi di dalam WHERE; di sini diperiksa eksplisit
	// supaya data contoh yang memuat klaim tertutup tidak ikut terbawa unduhan.
	switch strings.TrimSpace(c.ProcessStatus) {
	case "Resolved-Completed", "Resolved-Rejected":
		return false
	}

	if !inScope(c, f.LineBusiness) {
		return false
	}
	if f.From != nil && c.RegisteredAt.Before(*f.From) {
		return false
	}
	// Batas atas EKSKLUSIF, sama seperti `PXCREATEDATETIME < :9` pada SQL.
	if f.To != nil && !c.RegisteredAt.Before(*f.To) {
		return false
	}
	return true
}

// inScope menerapkan cakupan lini bisnis seperti pada my_inbox_export.
func inScope(c inboxoutstanding.OutstandingClaim, line inboxoutstanding.LineBusiness) bool {
	panel := normalize(c.GroupPanel)

	switch line {
	case inboxoutstanding.LinePA:
		return panel == "002"
	case inboxoutstanding.LineTravel:
		return panel == "005"
	case inboxoutstanding.LineBonding:
		// Tidak dapat ditirukan: BUSINESSGROUPID tidak ada pada OutstandingClaim.
		// Mengembalikan false akan MEMBOHONGI pengujian dengan berkas kosong yang tampak
		// sah, jadi seluruh baris diloloskan dan keterbatasannya disebut di atas.
		return true
	case inboxoutstanding.LineNonMBU:
		switch panel {
		case "003", "004", "006":
		default:
			return false
		}
		if strings.EqualFold(strings.TrimSpace(c.BranchName), "ASNET") {
			return false
		}
		return strings.TrimSpace(c.TechnicalPIC) != ""
	default:
		// LineUnknown: tanpa cakupan, persis seperti fragmen kosong di Pega.
		return true
	}
}

// matches menerapkan seluruh penyaring pada satu klaim.
func matches(c inboxoutstanding.OutstandingClaim, f inboxoutstanding.Filter) bool {
	// Dua identitas untuk satu orang — lihat Filter.AssignedToLegacy.
	if f.AssignedTo != "" {
		cocok := strings.EqualFold(c.CurrentHolder, f.AssignedTo)
		if !cocok && f.AssignedToLegacy != "" {
			cocok = strings.EqualFold(c.CurrentHolder, f.AssignedToLegacy)
		}
		if !cocok {
			return false
		}
	}
	if f.GroupPanel != "" && !strings.EqualFold(c.GroupPanel, f.GroupPanel) {
		return false
	}
	if f.RCVID != "" && !strings.EqualFold(c.RCVID, f.RCVID) {
		return false
	}
	if f.Stage != "" && !strings.EqualFold(c.CurrentStage, f.Stage) {
		return false
	}
	if f.BranchCode != "" && !strings.EqualFold(c.BranchName, f.BranchCode) {
		return false
	}
	if f.DocumentStatus != "" && documentStatusOf(c) != f.DocumentStatus {
		return false
	}
	if f.Search != "" && !matchesSearch(c, f.Search) {
		return false
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

// Claims mengembalikan seluruh klaim yang tersimpan, TANPA penyaring apa pun.
//
// Dipakai pengujian yang memeriksa ISI data contoh — bukan hasil List. Keduanya berbeda
// sejak daftar terikat pada pemiliknya: sebagian keadaan yang sengaja diwakili data contoh,
// seperti tugas yang belum bertuan, memang tidak boleh muncul di My Inbox.
func (r *Repo) Claims() []inboxoutstanding.OutstandingClaim {
	out := make([]inboxoutstanding.OutstandingClaim, len(r.claims))
	copy(out, r.claims)
	return out
}

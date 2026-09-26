// Package memory memenuhi seam inboxpladlapredla.Repo tanpa basis data.
//
// # Untuk apa ia ada
//
// Dua hal, dan keduanya nyata:
//
//   - Pengembangan lokal tanpa Oracle. Kredensial basis data pengembangan tidak dibagikan
//     ke setiap orang yang menyentuh layar ini.
//   - Uji aturan bisnis TANPA infrastruktur (`14-TESTING-STRATEGY.md` §3). Aturan yang
//     diuji di sini — penyaring tiap tab, arti pencarian, cara tanggal advice dipilih —
//     adalah aturan yang sama yang ditegakkan penyimpanan SQL.
//
// # Yang ia JANJIKAN sama dengan penyimpanan SQL, dan yang tidak
//
// Sama: baris mana yang lolos penyaring tab, baris mana yang lolos pencarian dan rentang
// tanggal, cara tanggal advice dipilih, urutan baris, dan bentuk halaman.
//
// TIDAK sama: apa pun yang menyangkut tipe data Oracle. Kolom `ISKIRIM` di sini teks
// biasa; di Oracle ia kolom yang dapat NULL, dan `NULL` beserta `'0'` sama-sama berarti
// "belum dikirim". Keduanya diwakili di sini — lihat sample.go.
package memory

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"claim-pnc/internal/inboxpladlapredla"
)

// Claim adalah satu baris `POOLDATA.T_CLAIM_PNC` sejauh yang dibaca layar ini.
//
// Hanya sepuluh kolomnya yang disimpan — sisanya tidak pernah dibaca modul ini, dan
// menyalin seluruh tabel ke sini akan membuat data contoh tampak lebih berwenang daripada
// sebenarnya.
type Claim struct {
	// Key adalah `CLAIMID`, kunci objek kerja Pega.
	Key string

	No       string
	PolicyNo string
	Insured  string

	RegisterDate time.Time
	LossDate     time.Time
	PICTeknik    string

	// GroupPanel adalah `GROUPPANEL`. `002` Personal Accident dan `005` Travel
	// dikecualikan ketiga tab.
	GroupPanel string

	// BranchName adalah `BRANCHNAME`. Hanya tab DLA menyaringnya, dan ia mengecualikan
	// `ASNET`.
	//
	// Kosong di sini berarti `NULL` di Oracle — dan itu TIDAK sama dengan "bukan ASNET".
	// Lihat claimPassesBranch.
	BranchName string

	// BusinessGroupID adalah `BUSINESSGROUPID` pada `POOLDATA.BUSINESS` milik
	// `BUSINESSCODE` klaim ini. Hanya tab DLA menyaringnya, mengecualikan `10008`.
	//
	// Ia disimpan MENDATAR di sini, bukan lewat tabel kedua: yang diuji adalah
	// penyaringnya, bukan cara gabungannya disusun.
	BusinessGroupID string
}

// Advice adalah satu baris `T_PLALIST`, `T_DLALIST`, atau `T_PREDLALIST`.
//
// Ketiganya diwakili satu tipe karena kolom yang dibaca layar ini nyaris sama. Yang
// membedakan tabelnya adalah isian Kind.
type Advice struct {
	ClaimKey string

	// Kind menunjuk tabel asalnya.
	Kind inboxpladlapredla.AdviceKind

	No        string
	Reinsurer string
	Type      string

	// Revision adalah `REVISI`. PLA saja; `GetDLAList` tidak mengambilnya.
	Revision string

	Date time.Time

	// Sent adalah `ISKIRIM`. Kosong berarti `NULL`.
	//
	// Ketiga nilai yang benar-benar muncul di data diwakili di sample.go: kosong, `"0"`,
	// dan `"1"`. Dua yang pertama sama-sama berarti belum terkirim bagi penyaring
	// keanggotaan — tetapi TIDAK sama bagi kolom tanggal advice, dan selisih itulah yang
	// diuji.
	Sent string

	SentDate     time.Time
	ReceivedDate time.Time
	Notes        string
	Email        string

	// AcceptanceNo adalah `NOAKSEP`. Pada DLA ia kolom yang digambar; pada Pre-DLA ia
	// justru PENYARING keanggotaan daftar.
	AcceptanceNo string

	// ReinsCode adalah `REINSCODE`. Hanya tab PLA menyaringnya — PLA yang belum menunjuk
	// reasuradur belum dapat dikirim ke siapa pun.
	ReinsCode string
}

// Store adalah penyimpanan di memori, aman dipakai beberapa goroutine.
type Store struct {
	mu      sync.RWMutex
	claims  []Claim
	advices []Advice
}

// NewStore membentuk penyimpanan kosong.
func NewStore() *Store {
	return &Store{claims: []Claim{}, advices: []Advice{}}
}

// Seed mengisi penyimpanan dengan data contoh.
func (s *Store) Seed(claims []Claim, advices []Advice) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.claims = make([]Claim, len(claims))
	copy(s.claims, claims)

	s.advices = make([]Advice, len(advices))
	copy(s.advices, advices)
}

// List mengembalikan satu halaman baris yang cocok beserta jumlah seluruhnya.
func (s *Store) List(
	_ context.Context,
	query inboxpladlapredla.Query,
	page inboxpladlapredla.Pagination,
) (inboxpladlapredla.Page, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	matched := []inboxpladlapredla.Row{}

	for _, claim := range s.claims {
		if !claimPassesTab(claim, query.Tab) {
			continue
		}
		if !s.hasQualifyingAdvice(claim.Key, query) {
			continue
		}

		row := inboxpladlapredla.Row{
			ClaimKey:     claim.Key,
			ClaimNo:      claim.No,
			PolicyNo:     claim.PolicyNo,
			Insured:      claim.Insured,
			RegisterDate: dateText(claim.RegisterDate),
			LossDate:     dateText(claim.LossDate),
			PICTeknik:    claim.PICTeknik,
			AdviceDate:   s.latestUnsentDate(claim.Key, query.Tab.Kind),
		}

		// Pencarian diperiksa TERHADAP BARIS, memakai aturan domain — bukan aturan yang
		// ditulis ulang di sini. Itulah yang membuat penyimpanan ini dan penyimpanan SQL
		// tidak dapat memahami pencarian secara berbeda tanpa ada uji yang gagal.
		if !query.Matches(row) {
			continue
		}

		matched = append(matched, row)
	}

	// Urutan mengikuti kueri lama: tanggal registrasi menaik, nomor klaim sebagai
	// pemutus seri.
	//
	// Pemutus serinya DITAMBAHKAN di sini dan di SQL. `ORDER BY b.registerdate ASC`
	// sendirian tidak menentukan urutan klaim yang registrasinya pada tanggal yang sama —
	// dan pada layar berpaginasi, urutan yang tidak ditentukan berarti satu baris dapat
	// muncul di dua halaman sekaligus sementara baris lain tidak muncul sama sekali.
	sort.SliceStable(matched, func(i, j int) bool {
		if matched[i].RegisterDate != matched[j].RegisterDate {
			return matched[i].RegisterDate < matched[j].RegisterDate
		}
		return matched[i].ClaimNo < matched[j].ClaimNo
	})

	return inboxpladlapredla.Slice(matched, page), nil
}

// claimPassesTab memeriksa penyaring yang berlaku pada BARIS KLAIM.
//
// Penyaring yang berlaku pada baris DOKUMEN ada di hasQualifyingAdvice. Keduanya dipisah
// karena memang dua hal yang berbeda di kuerinya: yang satu di klausa WHERE utama, yang
// lain di dalam EXISTS.
func claimPassesTab(claim Claim, tab inboxpladlapredla.Tab) bool {
	if claim.GroupPanel == "002" || claim.GroupPanel == "005" {
		return false
	}
	if tab.ExcludeASNET && !claimPassesBranch(claim) {
		return false
	}
	if tab.ExcludeBusinessGroup != "" &&
		claim.BusinessGroupID == tab.ExcludeBusinessGroup {
		return false
	}
	return true
}

// claimPassesBranch meniru `c.BRANCHNAME <> 'ASNET'` APA ADANYA, termasuk perlakuannya
// terhadap kolom yang kosong.
//
// Di Oracle, `NULL <> 'ASNET'` menghasilkan UNKNOWN — bukan TRUE — sehingga klaim yang
// cabangnya kosong ikut TERSARING KELUAR. Kueri lama tidak menulis `OR BRANCHNAME IS
// NULL`, dan perilakunya dibawa apa adanya (`P-5`).
//
// Menuliskannya sebagai fungsi tersendiri, bukan sebagai satu baris di dalam
// claimPassesTab, supaya kejanggalan ini punya tempat untuk dijelaskan — dan supaya uji
// dapat menunjuknya langsung.
func claimPassesBranch(claim Claim) bool {
	return strings.TrimSpace(claim.BranchName) != "" &&
		claim.BranchName != "ASNET"
}

// hasQualifyingAdvice memeriksa apakah klaim ini punya dokumen yang membuatnya masuk
// antrean.
//
// Penyaringnya BERBEDA per tab, dan perbedaannya bukan kelalaian — lihat catatan di
// kepala inboxpladlapredla.go.
func (s *Store) hasQualifyingAdvice(
	claimKey string,
	query inboxpladlapredla.Query,
) bool {
	for _, advice := range s.advices {
		if advice.ClaimKey != claimKey || advice.Kind != query.Tab.Kind {
			continue
		}
		if query.Tab.RequiresReinsurer && strings.TrimSpace(advice.ReinsCode) == "" {
			continue
		}
		if query.Tab.SentFilterApplies && !isUnsent(advice.Sent) {
			continue
		}
		// Tab Pre DLA: yang mengeluarkan baris dari antrean adalah terbitnya Nomor
		// Akseptasi, bukan terkirimnya surat.
		if query.Tab.RequiresNoAcceptance &&
			strings.TrimSpace(advice.AcceptanceNo) != "" {
			continue
		}
		if !advice.Date.IsZero() && !query.WithinRange(advice.Date) {
			continue
		}
		return true
	}
	return false
}

// isUnsent menyatakan sebuah dokumen belum dikirim.
//
// `NULL` (kosong di sini) dan `'0'` sama-sama berarti belum. Keduanya benar-benar ada di
// data, dan menyamakan salah satunya saja akan menghilangkan sebagian antrean.
func isUnsent(sent string) bool {
	clean := strings.TrimSpace(sent)
	return clean == "" || clean == "0"
}

// latestUnsentDate mengambil tanggal dokumen TERBARU yang `ISKIRIM`-nya kosong.
//
// # Ia menyaring LEBIH SEMPIT daripada penyaring keanggotaan, dan itu disengaja
//
// Penyaring keanggotaan menerima `ISKIRIM IS NULL OR ISKIRIM = '0'`; sub-kueri tanggalnya
// di Pega hanya menerima `ISKIRIM IS NULL`. Klaim yang SELURUH dokumennya ber-`ISKIRIM =
// '0'` karena itu muncul di daftar dengan kolom tanggal KOSONG.
//
// Itu perilaku Pega (`P-5`), dan ia ditiru di sini persis supaya uji kesetaraan tidak
// melaporkannya sebagai selisih.
func (s *Store) latestUnsentDate(
	claimKey string,
	kind inboxpladlapredla.AdviceKind,
) string {
	var latest time.Time
	for _, advice := range s.advices {
		if advice.ClaimKey != claimKey || advice.Kind != kind {
			continue
		}
		if strings.TrimSpace(advice.Sent) != "" {
			continue
		}
		if advice.Date.After(latest) {
			latest = advice.Date
		}
	}
	return dateText(latest)
}

// Documents mengembalikan isi grid rincian satu klaim.
//
// Ia mengembalikan SELURUH dokumen milik klaim itu, terkirim maupun belum — kueri lama pun
// tidak menyaring `ISKIRIM`. Kolom "Terkirim" dan "Tanggal Kirim" justru ada supaya
// perbedaannya terlihat.
func (s *Store) Documents(
	_ context.Context,
	tab inboxpladlapredla.Tab,
	claimKey string,
) ([]inboxpladlapredla.Document, error) {
	key := strings.TrimSpace(claimKey)
	if key == "" {
		return nil, inboxpladlapredla.ErrRowNotFound
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.claimExistsLocked(key) {
		return nil, inboxpladlapredla.ErrRowNotFound
	}

	items := []inboxpladlapredla.Document{}
	for _, advice := range s.advices {
		if advice.ClaimKey != key || advice.Kind != tab.Kind {
			continue
		}

		item := inboxpladlapredla.Document{
			AdviceNo:     advice.No,
			Reinsurer:    advice.Reinsurer,
			AdviceType:   advice.Type,
			AdviceDate:   dateText(advice.Date),
			Sent:         advice.Sent,
			SentDate:     dateText(advice.SentDate),
			ReceivedDate: dateText(advice.ReceivedDate),
			Notes:        advice.Notes,
			Email:        advice.Email,
		}

		// Dua kolom yang hanya ada di salah satu tabel dikosongkan pada tabel yang lain
		// — persis seperti kuerinya, yang mengisinya `NULL`. Membawanya apa adanya akan
		// menampilkan nilai pada kolom yang di Pega memang tidak pernah terisi.
		if tab.Kind == inboxpladlapredla.KindPLA {
			item.Revision = advice.Revision
		}
		if tab.Kind == inboxpladlapredla.KindDLA {
			item.AcceptanceNo = advice.AcceptanceNo
		}

		items = append(items, item)
	}

	sort.SliceStable(items, func(i, j int) bool {
		if items[i].AdviceDate != items[j].AdviceDate {
			return items[i].AdviceDate < items[j].AdviceDate
		}
		return items[i].AdviceNo < items[j].AdviceNo
	})

	return items, nil
}

// claimExistsLocked memeriksa keberadaan klaim. Pemanggil sudah memegang kunci baca.
func (s *Store) claimExistsLocked(key string) bool {
	for _, claim := range s.claims {
		if claim.Key == key {
			return true
		}
	}
	return false
}

// dateText menuliskan tanggal sebagai `YYYY-MM-DD`, dan tanggal kosong sebagai teks
// kosong.
//
// Nilai nol menghasilkan teks KOSONG, bukan `0001-01-01`. Perbedaannya nyata di layar ini:
// kolom tanggal advice memang kadang kosong, dan `0001-01-01` akan terbaca sebagai tanggal
// yang benar-benar tercatat.
func dateText(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format(inboxpladlapredla.DateLayout)
}

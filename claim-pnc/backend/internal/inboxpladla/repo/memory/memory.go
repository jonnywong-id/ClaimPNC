// Package memory memenuhi seam inboxpladla.Repo tanpa basis data.
//
// # Untuk apa ia ada
//
// Dua hal, dan keduanya nyata:
//
//   - Pengembangan lokal tanpa Oracle.
//   - Uji aturan bisnis TANPA infrastruktur (`14-TESTING-STRATEGY.md` §3). Aturan yang
//     diuji di sini — penyaring tiap tab, cara kode reasuradur dipilih, penggantian kode
//     status menjadi `1139` — adalah aturan yang sama yang ditegakkan penyimpanan SQL.
//
// # Satu aturan diuji di sini yang TIDAK dapat diuji di tempat lain
//
// Dua dari tiga daftar memakai kode reasuradur TERTINGGI saja, satu memakai semuanya.
// Perbedaan itu ada di dalam teks SQL — sebagai `=` versus `IN` — sehingga tidak ada uji
// yang dapat membuktikannya tanpa Oracle, kecuali di sini.
package memory

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"claim-pnc/internal/inboxpladla"
)

// Claim adalah satu baris `POOLDATA.T_CLAIM_PNC` sejauh yang dibaca layar ini.
//
// SUMBER BARU (2026-10-08): kode status dan penanda tunggu-tutup dulu dibaca dari tabel
// kerja Pega (`PZINSKEY = CLAIMID`); kini keduanya kolom `T_CLAIM_PNC` sendiri.
type Claim struct {
	// Key adalah `CLAIMID`, kunci objek kerja Pega.
	Key string

	No           string
	PolicyNo     string
	Insured      string
	BusinessName string
	RegisterDate time.Time
	LossDate     time.Time
	PICTeknik    string
	CloseNote    string

	// GroupPanel adalah `GROUPPANEL`. `002` dan `005` dikecualikan ketiga daftar.
	GroupPanel string

	// WorkStatus adalah `T_CLAIM_PNC.STATUSWORK`.
	WorkStatus string

	// StatusCode adalah `T_CLAIM_PNC.STATUSCLAIM`.
	//
	// SUMBER BARU (2026-10-08): dulu `PC_ASM_FW_GCNMFW_WORK.STATUSCLAIM_1`; tabel kerja
	// Pega sudah tidak dipakai.
	StatusCode string

	// PendingClose adalah `T_CLAIM_PNC.ISPENDINGCLOSE = 'true'` (dulu kolom yang sama pada
	// tabel kerja Pega).
	//
	// Ia menentukan dua hal sekaligus pada tab DLA: apakah klaimnya ikut daftar meski
	// sudah `Resolved-Completed`, dan apakah kode statusnya diganti `1139`.
	PendingClose bool
}

// Advice adalah satu baris `T_PLALIST` atau `T_DLALIST` sejauh yang dibaca layar ini.
type Advice struct {
	ClaimKey string

	// Kind bernilai `"pla"` atau `"dla"`, sejalan dengan Tab.AdviceKindSent.
	Kind string

	No string

	// ReinsCode adalah `REINSCODE` — kode reasuradur tujuan dokumen ini.
	ReinsCode string

	// Revision adalah `REVISI`. Yang tertinggi menentukan nomor yang digambar kolom
	// "No PLA".
	Revision int

	// Sent, SentDate, dan Email bersama-sama menyatakan dokumen ini SUDAH terkirim.
	//
	// Ketiganya diperiksa, bukan hanya Sent: kueri lama menuntut
	// `iskirim='1' AND tglkirim IS NOT NULL AND emailpla IS NOT NULL`. Dokumen yang
	// ditandai terkirim tetapi tanpa tanggal maupun alamat TIDAK dihitung, dan keadaan
	// itu benar-benar ada di data.
	Sent     string
	SentDate time.Time
	Email    string

	// Ketiga isian di bawah HANYA dipakai layar rincian — grid "PLA" dan "DLA".
	//
	//	Type          TIPEPLA / TIPEDLA
	//	Amount        NILAIPLA / NILAIDLA   dibawa sebagai TEKS; lihat AdviceRow.Amount
	//	AcceptanceNo  NOAKSEP               hanya ada pada DLA
	Type         string
	Amount       string
	AcceptanceNo string

	// AdviceDate adalah `TGLPLA` / `TGLDLA` — yang MENGURUTKAN grid rincian.
	AdviceDate time.Time
}

// Reinsurer adalah satu baris `POOLDATA.T_REINSURER` sejauh yang dibaca layar ini.
type Reinsurer struct {
	Code  string
	Login string
}

// XOL adalah satu baris `T_PLA_XOL` atau `T_DLA_XOL`.
type XOL struct {
	Kind        string
	ReinsCode   string
	Year        string
	CauseOfLoss string

	// Sent menyatakan `SENDDATE IS NOT NULL` — hanya yang sudah terkirim yang dihitung.
	Sent       bool
	InsertDate time.Time
}

// StatusLabel adalah satu baris `POOLDATA.M_STS_CLAIM`.
type StatusLabel struct {
	Code  string
	Label string
}

// Store adalah penyimpanan di memori, aman dipakai beberapa goroutine.
type Store struct {
	mu         sync.RWMutex
	claims     []Claim
	advices    []Advice
	reinsurers []Reinsurer
	xol        []XOL
	labels     []StatusLabel

	// messages dan documents melayani layar RINCIAN — lihat detail.go.
	//
	// Keduanya diisi lewat SeedMessages dan SeedDocuments, bukan lewat Seed: menambahkan
	// keduanya sebagai parameter Seed akan memaksa setiap uji yang sudah ada menyebut dua
	// senarai kosong tanpa alasan.
	messages  []Message
	documents []Document
}

// NewStore membentuk penyimpanan kosong.
func NewStore() *Store {
	return &Store{
		claims:     []Claim{},
		advices:    []Advice{},
		reinsurers: []Reinsurer{},
		xol:        []XOL{},
		labels:     []StatusLabel{},
		messages:   []Message{},
		documents:  []Document{},
	}
}

// Seed mengisi penyimpanan dengan data contoh.
func (s *Store) Seed(
	claims []Claim,
	advices []Advice,
	reinsurers []Reinsurer,
	xol []XOL,
	labels []StatusLabel,
) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.claims = append([]Claim{}, claims...)
	s.advices = append([]Advice{}, advices...)
	s.reinsurers = append([]Reinsurer{}, reinsurers...)
	s.xol = append([]XOL{}, xol...)
	s.labels = append([]StatusLabel{}, labels...)
}

// ReinsurerCodes mengembalikan kode reasuradur milik satu login, terurut MENURUN.
func (s *Store) ReinsurerCodes(
	_ context.Context,
	login string,
) ([]string, error) {
	clean := strings.TrimSpace(login)
	if clean == "" {
		return nil, nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.reinsurerCodesLocked(clean), nil
}

// reinsurerCodesLocked mencari kode reasuradur milik satu login. Pemanggil sudah memegang
// kunci baca.
//
// Ia dipisahkan dari ReinsurerCodes supaya XOL tidak mengambil kunci baca DUA KALI.
// `sync.RWMutex` tidak menjamin pengambilan kunci baca bersarang aman: satu penulis yang
// menunggu di antara keduanya membuat keduanya saling menunggu.
func (s *Store) reinsurerCodesLocked(login string) []string {
	codes := []string{}
	for _, reinsurer := range s.reinsurers {
		if strings.EqualFold(strings.TrimSpace(reinsurer.Login), login) {
			codes = append(codes, reinsurer.Code)
		}
	}

	// Menurun, sama dengan `ORDER BY REINSURERID DESC` di SQL. Yang PERTAMA di senarai
	// inilah yang dipakai kedua daftar yang hanya memakai satu kode.
	sort.Sort(sort.Reverse(sort.StringSlice(codes)))
	return codes
}

// List mengembalikan satu halaman baris yang cocok beserta jumlah seluruhnya.
func (s *Store) List(
	_ context.Context,
	query inboxpladla.Query,
	page inboxpladla.Pagination,
) (inboxpladla.Page, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	matched := s.rowsFor(query)
	return inboxpladla.Slice(matched, page), nil
}

// Counts mengembalikan tabel ringkas "Status / Jumlah".
//
// Ia dihitung dari populasi yang SAMA dengan List — bukan dari halaman yang sedang tampil,
// dan bukan dari penyaring yang disalin ulang. Keduanya memanggil rowsFor.
func (s *Store) Counts(
	_ context.Context,
	query inboxpladla.Query,
) ([]inboxpladla.StatusCount, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	totals := map[string]int{}
	order := []string{}

	for _, row := range s.rowsFor(query) {
		if _, seen := totals[row.StatusCode]; !seen {
			order = append(order, row.StatusCode)
		}
		totals[row.StatusCode]++
	}

	sort.Strings(order)

	counts := []inboxpladla.StatusCount{}
	for _, code := range order {
		counts = append(counts, inboxpladla.StatusCount{
			Code:  code,
			Label: s.labelOfLocked(code),
			Total: totals[code],
		})
	}
	return counts, nil
}

// XOL mengembalikan isi grid "DATA PLA DLA XOL KLAIM".
func (s *Store) XOL(
	_ context.Context,
	login string,
) ([]inboxpladla.XOLRow, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	codes := s.reinsurerCodesLocked(strings.TrimSpace(login))

	type key struct{ kind, year, cause string }
	latest := map[key]time.Time{}

	for _, item := range s.xol {
		if !item.Sent || !containsValue(codes, item.ReinsCode) {
			continue
		}
		k := key{item.Kind, item.Year, item.CauseOfLoss}
		if item.InsertDate.After(latest[k]) {
			latest[k] = item.InsertDate
		}
	}

	rows := []inboxpladla.XOLRow{}
	for k, moment := range latest {
		rows = append(rows, inboxpladla.XOLRow{
			Year:           k.year,
			CauseOfLoss:    k.cause,
			Kind:           k.kind,
			LastInsertDate: dateText(moment),
		})
	}

	// `ORDER BY "Type","City" desc` pada kueri lama: jenis menaik, tahun menurun.
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].Kind != rows[j].Kind {
			return rows[i].Kind < rows[j].Kind
		}
		if rows[i].Year != rows[j].Year {
			return rows[i].Year > rows[j].Year
		}
		return rows[i].CauseOfLoss < rows[j].CauseOfLoss
	})

	return rows, nil
}

// rowsFor menyusun SELURUH baris yang cocok, belum dipotong halaman.
//
// Pemanggil sudah memegang kunci baca.
func (s *Store) rowsFor(query inboxpladla.Query) []inboxpladla.Row {
	codes := query.EffectiveReinsurerCodes()
	matched := []inboxpladla.Row{}

	for _, claim := range s.claims {
		if !claimPassesTab(claim, query.Tab) {
			continue
		}

		if query.Tab.Source == inboxpladla.SourceCommunication {
			// Daftar komunikasi TIDAK memeriksa dokumen pemberitahuan sama sekali —
			// `BrowseCommunicationReas` tidak memuat satu pun syarat itu. Yang
			// memasukkan sebuah klaim ke sini adalah adanya PERCAKAPAN.
			if !s.hasConversation(claim.Key, query.Tab, query.Caller.Login) {
				continue
			}
		} else {
			if !s.hasSentAdvice(claim.Key, query.Tab.AdviceKindSent, codes) {
				continue
			}
			if query.Tab.ExcludeWhenDLASent && s.hasSentAdvice(claim.Key, "dla", codes) {
				continue
			}
		}

		row := inboxpladla.Row{
			ClaimKey:     claim.Key,
			ClaimNo:      claim.No,
			PolicyNo:     claim.PolicyNo,
			Insured:      claim.Insured,
			BusinessName: claim.BusinessName,
			RegisterDate: dateText(claim.RegisterDate),
			LossDate:     dateText(claim.LossDate),
			PICTeknik:    claim.PICTeknik,
			StatusCode:   statusCodeOf(claim, query.Tab),
			CloseNote:    claim.CloseNote,
			AdviceNo:     s.latestAdviceNo(claim.Key, topCodeOf(query)),
		}
		row.StatusLabel = s.labelOfLocked(row.StatusCode)

		if !query.Matches(row) {
			continue
		}

		matched = append(matched, row)
	}

	// `ORDER BY b.claimno asc` pada ketiga kueri lama.
	sort.SliceStable(matched, func(i, j int) bool {
		return matched[i].ClaimNo < matched[j].ClaimNo
	})

	return matched
}

// claimPassesTab memeriksa penyaring yang berlaku pada BARIS KLAIM.
func claimPassesTab(claim Claim, tab inboxpladla.Tab) bool {
	// Pengecualian lini bisnis dibaca dari TAB-nya, bukan ditulis tetap di sini.
	//
	// Ketiga daftar pemberitahuan mengecualikan `002` dan `005`; ketiga daftar komunikasi
	// tidak mengecualikan apa pun. Menuliskannya tetap di sini akan membuat klaim
	// Personal Accident hilang dari daftar komunikasi — dan hilangnya tidak akan terlihat
	// sampai seseorang membandingkannya dengan Pega.
	if tab.ExcludesGroupPanel(claim.GroupPanel) {
		return false
	}

	switch tab.WorkStatus {
	case inboxpladla.WorkOpen:
		return !containsValue(inboxpladla.ClosedWorkStatuses, claim.WorkStatus)

	case inboxpladla.WorkOpenOrPendingClose:
		// Gabungan INNER ke tabel kerja Pega sudah tidak ada (SUMBER BARU 2026-10-08):
		// klaim tanpa baris kerja Pega kini ikut, sama seperti pada dua tab lain.
		if !containsValue(inboxpladla.ClosedWorkStatuses, claim.WorkStatus) {
			return true
		}
		return claim.WorkStatus == inboxpladla.CompletedWorkStatus && claim.PendingClose

	case inboxpladla.WorkClosed:
		return claim.WorkStatus == inboxpladla.CompletedWorkStatus && !claim.PendingClose

	default:
		return false
	}
}

// statusCodeOf mengembalikan kode status yang DIGAMBAR, bukan kode yang tersimpan.
//
// Hanya tab DLA yang menggantinya menjadi `1139`, dan hanya ketika klaimnya menunggu
// penutupan.
func statusCodeOf(claim Claim, tab inboxpladla.Tab) string {
	if tab.PendingCloseBecomes1139 && claim.PendingClose {
		return inboxpladla.PendingCloseStatusCode
	}
	return claim.StatusCode
}

// hasSentAdvice menyatakan klaim ini punya dokumen jenis tertentu yang SUDAH terkirim ke
// salah satu kode reasuradur yang diberikan.
//
// Ketiga syarat kueri lama diperiksa seluruhnya — `iskirim = '1'`, tanggal kirim terisi,
// dan alamat surel terisi. Memeriksa yang pertama saja akan memasukkan dokumen yang
// ditandai terkirim tanpa pernah benar-benar dikirim.
func (s *Store) hasSentAdvice(claimKey, kind string, codes []string) bool {
	for _, advice := range s.advices {
		if advice.ClaimKey != claimKey || advice.Kind != kind {
			continue
		}
		if advice.Sent != "1" ||
			advice.SentDate.IsZero() ||
			strings.TrimSpace(advice.Email) == "" {
			continue
		}
		if containsValue(codes, advice.ReinsCode) {
			return true
		}
	}
	return false
}

// latestAdviceNo mengambil nomor PLA berrevisi tertinggi milik satu kode reasuradur.
//
// Kode yang dipakai SELALU yang tertinggi, bahkan pada tab Close yang penyaring
// daftarnya memakai seluruh kode. Itu perilaku kueri lama: satu kueri, dua aturan
// berbeda.
func (s *Store) latestAdviceNo(claimKey, reinsurerCode string) string {
	if reinsurerCode == "" {
		return ""
	}

	best := ""
	highest := -1
	for _, advice := range s.advices {
		if advice.ClaimKey != claimKey ||
			advice.Kind != "pla" ||
			advice.ReinsCode != reinsurerCode {
			continue
		}
		if advice.Revision > highest {
			highest = advice.Revision
			best = advice.No
		}
	}
	return best
}

// topCodeOf mengembalikan kode reasuradur TERTINGGI milik pemanggil.
func topCodeOf(query inboxpladla.Query) string {
	if len(query.ReinsurerCodes) == 0 {
		return ""
	}
	return query.ReinsurerCodes[0]
}

// labelOfLocked mencari arti sebuah kode status. Pemanggil sudah memegang kunci baca.
func (s *Store) labelOfLocked(code string) string {
	for _, label := range s.labels {
		if label.Code == code {
			return label.Label
		}
	}
	return ""
}

// containsValue menyatakan sebuah nilai ada di dalam senarai.
func containsValue(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

// dateText menuliskan tanggal sebagai `YYYY-MM-DD`, dan tanggal kosong sebagai teks
// kosong.
func dateText(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format(inboxpladla.DateLayout)
}

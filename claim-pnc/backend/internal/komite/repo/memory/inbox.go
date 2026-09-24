package memory

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"sort"
	"sync"

	"claim-pnc/internal/komite"
)

// InboxStore adalah pengisi seam komite.InboxRepo dan komite.DecisionRepo di dalam
// memori.
//
// Ia adalah adapter KEDUA yang membuat kedua seam itu nyata, bukan hipotetis
// (`docs/Steering/04-FUTURE-ARCHITECTURE.md` §3) — dan karenanya seluruh aturan Inbox
// Komite dapat diuji tanpa Oracle sama sekali.
//
// Ia juga yang dipakai saat aplikasi berjalan tanpa basis data, sehingga layar Inbox
// Komite dapat dibuka dan dicoba lengkap dengan alur keputusannya sebelum akun aplikasi
// diberi hak baca ke `DATAPEGA.PC_ASM_FW_GCNMFW_WORK`.
//
// # Ia MEMAKAI aturan domain, bukan menirunya
//
// Penyaringan kotak, pencarian, dan rentang tanggal seluruhnya memanggil metode pada
// komite.CommitteeCase. Menuliskan ulang aturannya di sini akan membuat uji yang berjalan
// di atas adapter ini menguji salinan, bukan aturan yang sebenarnya berlaku.
type InboxStore struct {
	mu        sync.RWMutex
	cases     []komite.CommitteeCase
	decisions []komite.Decision
	err       error
}

// NewInboxStore membentuk penyimpanan berisi kasus yang diberikan.
//
// Dipanggil tanpa argumen ia KOSONG — bukan otomatis berisi contoh. Pengujian yang ingin
// menguji inbox kosong tidak perlu melawan nilai bawaan yang tidak diminta.
func NewInboxStore(cases ...komite.CommitteeCase) *InboxStore {
	copied := make([]komite.CommitteeCase, 0, len(cases))
	for _, c := range cases {
		copied = append(copied, c.Normalized())
	}
	return &InboxStore{cases: copied}
}

// NewSampleInboxStore membentuk penyimpanan berisi kasus contoh.
func NewSampleInboxStore() *InboxStore { return NewInboxStore(SampleCases()...) }

// SetError membuat penyimpanan menjawab dengan galat, untuk menguji jalur gagal.
func (s *InboxStore) SetError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.err = err
}

// ListCases mengembalikan satu halaman kasus beserta jumlah seluruh yang cocok.
func (s *InboxStore) ListCases(_ context.Context, f komite.InboxFilter) (komite.InboxPage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.err != nil {
		return komite.InboxPage{}, s.err
	}

	matched := s.matching(f)
	total := len(matched)

	// Halaman dipotong SETELAH seluruh yang cocok dihitung, sehingga Total melaporkan
	// keseluruhan — bukan isi halaman ini. Inilah yang membedakannya dari
	// `pyMaxRecords=500` sistem lama, yang memotong tanpa pernah menyebut ada yang
	// terpotong (`T-12`).
	if f.Offset >= total {
		return komite.InboxPage{Cases: []komite.CommitteeCase{}, Total: total}, nil
	}
	end := f.Offset + f.Limit
	if end > total {
		end = total
	}
	return komite.InboxPage{Cases: matched[f.Offset:end], Total: total}, nil
}

// Summarize menghitung isi ketiga kotak untuk operator pada penyaring.
//
// Pencarian dan rentang tanggal IKUT diterapkan, kotaknya tidak. Dengan begitu lencana
// pada tab menjawab pertanyaan yang benar: "berapa yang cocok dengan pencarian saya di
// kotak lain", bukan "berapa isi kotak lain seluruhnya" — yang akan membuat pengguna
// berpindah tab lalu menemukan tabel kosong.
func (s *InboxStore) Summarize(_ context.Context, f komite.InboxFilter) (komite.InboxSummary, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.err != nil {
		return komite.InboxSummary{}, s.err
	}

	var summary komite.InboxSummary
	summary.Outstanding = len(s.matching(withKind(f, komite.InboxOutstanding)))
	summary.Accepted = len(s.matching(withKind(f, komite.InboxAccepted)))
	summary.Rejected = len(s.matching(withKind(f, komite.InboxRejected)))
	return summary, nil
}

// FindCase mengambil satu kasus tanpa memandang kotak maupun pemiliknya.
//
// Pemeriksaan kepemilikan sengaja TIDAK dikerjakan di sini — ia milik lapisan usecase,
// supaya "tidak ada" dan "bukan milik Anda" dapat dibedakan di log meski disamakan di
// peramban.
func (s *InboxStore) FindCase(_ context.Context, caseID string) (komite.CommitteeCase, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.err != nil {
		return komite.CommitteeCase{}, s.err
	}
	for _, c := range s.cases {
		if c.CaseID == caseID {
			return s.withDecisions(c), nil
		}
	}
	return komite.CommitteeCase{}, komite.ErrCaseNotFound
}

// ListForCases mengembalikan keputusan seluruh kasus yang disebut.
func (s *InboxStore) ListForCases(
	_ context.Context,
	caseIDs []string,
) (map[string][]komite.Decision, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.err != nil {
		return nil, s.err
	}

	wanted := make(map[string]bool, len(caseIDs))
	for _, id := range caseIDs {
		wanted[id] = true
	}

	result := map[string][]komite.Decision{}
	for _, d := range s.decisions {
		if wanted[d.CaseID] {
			result[d.CaseID] = append(result[d.CaseID], d)
		}
	}
	return result, nil
}

// Record menyimpan satu keputusan. Ia tidak pernah menimpa apa pun.
func (s *InboxStore) Record(_ context.Context, d komite.Decision) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.err != nil {
		return s.err
	}
	s.decisions = append(s.decisions, d)
	return nil
}

// matching menyaring seluruh kasus memakai aturan domain, lalu mengurutkannya.
//
// Pemanggil WAJIB sudah memegang kunci baca.
func (s *InboxStore) matching(f komite.InboxFilter) []komite.CommitteeCase {
	// Operator kosong berarti NOL BARIS, bukan semua baris. Kegagalan yang aman pada
	// sebuah inbox adalah menampilkan terlalu sedikit — lihat catatan pada
	// InboxFilter.Normalize.
	if f.Operator == "" {
		return nil
	}

	var result []komite.CommitteeCase
	for _, c := range s.cases {
		enriched := s.withDecisions(c)
		if !enriched.InBox(f.Kind, f.Operator) {
			continue
		}
		if !enriched.MatchesSearch(f.Search) {
			continue
		}
		if !enriched.WithinDateRange(f.DateFrom, f.DateTo) {
			continue
		}
		result = append(result, enriched)
	}

	// Yang paling lama menunggu di atas, persis `ORDER BY "AgingKomite" DESC` pada
	// `GetKomitePAOutstanding`. CaseID menjadi pemecah seri supaya urutannya PASTI —
	// dua kasus bertanggal sama tidak boleh berpindah tempat antar permintaan.
	sort.SliceStable(result, func(i, j int) bool {
		a, b := result[i], result[j]
		if !a.CommitteeDate.Equal(b.CommitteeDate) {
			return a.CommitteeDate.Before(b.CommitteeDate)
		}
		return a.CaseID < b.CaseID
	})
	return result
}

// withDecisions menempelkan keputusan yang tercatat ke sebuah kasus.
//
// Pemanggil WAJIB sudah memegang kunci baca.
func (s *InboxStore) withDecisions(c komite.CommitteeCase) komite.CommitteeCase {
	var own []komite.Decision
	for _, d := range s.decisions {
		if d.CaseID == c.CaseID {
			own = append(own, d)
		}
	}
	c.Progress = komite.Evaluate(own, c.TierCount)
	return c
}

func withKind(f komite.InboxFilter, kind komite.InboxKind) komite.InboxFilter {
	f.Kind = kind
	return f
}

// IDGenerator membangkitkan pengenal keputusan.
//
// Ia acak, bukan berurut: pengenal keputusan tidak boleh membocorkan berapa banyak
// keputusan yang sudah tercatat, dan tidak boleh dapat ditebak dari pengenal lain.
type IDGenerator struct{}

// New mengembalikan pengenal 128 bit dalam heksadesimal.
func (IDGenerator) New() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// crypto/rand tidak gagal pada sistem yang sehat. Bila ia gagal, melanjutkan
		// dengan pengenal yang dapat ditebak lebih berbahaya daripada berhenti.
		panic("komite/memory: sumber acak tidak tersedia: " + err.Error())
	}
	return hex.EncodeToString(b[:])
}

var (
	_ komite.InboxRepo    = (*InboxStore)(nil)
	_ komite.DecisionRepo = (*InboxStore)(nil)
	_ komite.IDGenerator  = IDGenerator{}
)

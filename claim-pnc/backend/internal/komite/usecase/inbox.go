package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"claim-pnc/internal/komite"
)

// Orkestrasi Inbox Komite — `TKT-B07-002`.
//
// Lapisan ini menggabungkan DUA SUMBER yang kepemilikannya berbeda, dan itulah seluruh
// alasan ia ada:
//
//	kasus komite   dibaca dari tabel WARISAN, tidak pernah ditulis   (InboxRepo)
//	keputusan      ditulis ke tabel MILIK APLIKASI INI               (DecisionRepo)
//
// Pembelahan itu bukan selera. `P-1` menetapkan satu tabel hanya boleh ditulis satu
// sistem, dan `POOLDATA.T_CLAIM_KOMITE_LIST` masih ditulis Pega. Menulis keputusan ke
// sana akan membuat dua sistem menulis satu tabel dengan aturan validasi yang berbeda —
// kelas konflik data yang `docs/Steering/07-MIGRATION-STRATEGY.md` sebut hampir mustahil
// dilacak.

// InboxService melayani layar Inbox Komite.
//
// Ia TERPISAH dari Service yang melayani master ambang dan penjenjangan, meski keduanya
// milik modul `B-7` yang sama. Alasannya kepemilikan data, bukan ukuran berkas: Service
// tidak menulis apa pun dan tidak pernah membutuhkan jam maupun pembangkit pengenal,
// sedangkan InboxService menulis dan membutuhkan keduanya. Menyatukannya akan memaksa
// seluruh pemakai master ambang ikut menyediakan bahan yang tidak mereka pakai.
type InboxService struct {
	cases     komite.InboxRepo
	decisions komite.DecisionRepo
	ids       komite.IDGenerator
	now       func() time.Time
}

// InboxOptions adalah bahan pembentuk InboxService.
type InboxOptions struct {
	Cases     komite.InboxRepo
	Decisions komite.DecisionRepo
	IDs       komite.IDGenerator

	// Clock boleh dikosongkan; bila kosong dipakai jam sistem dalam UTC.
	//
	// Ia dipasok dari luar supaya aturan yang bergantung waktu — Aging dan waktu
	// keputusan — dapat diuji secara deterministik (`F-5`).
	Clock interface{ Now() time.Time }
}

// NewInboxService membentuk InboxService dan menolak bahan yang tidak lengkap.
//
// Kegagalannya terjadi saat start, bukan saat anggota komite pertama membuka layar.
func NewInboxService(o InboxOptions) (*InboxService, error) {
	if o.Cases == nil {
		return nil, errors.New("komite/usecase: seam kasus komite wajib diisi")
	}
	if o.Decisions == nil {
		return nil, errors.New("komite/usecase: seam keputusan komite wajib diisi")
	}
	if o.IDs == nil {
		return nil, errors.New("komite/usecase: pembangkit pengenal wajib diisi")
	}

	now := time.Now
	if o.Clock != nil {
		now = o.Clock.Now
	}
	return &InboxService{cases: o.Cases, decisions: o.Decisions, ids: o.IDs, now: now}, nil
}

// InboxResult adalah satu halaman inbox beserta bahan yang dibutuhkan layar.
type InboxResult struct {
	Cases   []komite.CommitteeCase
	Total   int
	Summary komite.InboxSummary

	// Now adalah jam yang dipakai menghitung Aging.
	//
	// Ia dikirim ke layar, bukan dibiarkan layar memakai jam peramban. Jam peramban
	// dapat berbeda dari jam server — dan Aging yang dihitung dua kali dengan dua jam
	// yang berbeda menghasilkan dua angka untuk satu kenyataan.
	Now time.Time
}

// Inbox membaca satu halaman inbox milik seorang anggota komite.
//
// # Kenapa keputusan dibaca sekaligus untuk seluruh halaman
//
// Membacanya per baris adalah kueri di dalam perulangan — satu perjalanan ke basis data
// per baris tabel. `docs/Steering/15-NFR-PERFORMANCE-SCALABILITY.md` §3.2 menyebutnya
// sebagai hambatan peringkat ketiga, dan pada layar yang dibuka setiap hari oleh orang
// yang sama, biayanya berulang setiap kali.
func (s *InboxService) Inbox(ctx context.Context, f komite.InboxFilter) (InboxResult, error) {
	f = f.Normalize()
	if err := f.Validate(); err != nil {
		return InboxResult{}, err
	}

	page, err := s.cases.ListCases(ctx, f)
	if err != nil {
		return InboxResult{}, fmt.Errorf("komite/usecase: membaca inbox komite: %w", err)
	}

	summary, err := s.cases.Summarize(ctx, f)
	if err != nil {
		return InboxResult{}, fmt.Errorf("komite/usecase: menghitung isi kotak: %w", err)
	}

	cases, err := s.withProgress(ctx, page.Cases)
	if err != nil {
		return InboxResult{}, err
	}

	return InboxResult{
		Cases:   cases,
		Total:   page.Total,
		Summary: summary,
		Now:     s.now(),
	}, nil
}

// Case mengambil satu kasus beserta seluruh keputusannya.
//
// # Kenapa kepemilikan diperiksa di sini, bukan di dalam WHERE
//
// Menaruhnya di kueri akan membuat "tidak ada" dan "bukan milik Anda" menjadi satu
// keadaan yang tidak dapat dibedakan — termasuk oleh log, yang justru perlu
// membedakannya saat seseorang melaporkan pekerjaannya hilang.
//
// Yang disamakan adalah JAWABAN KE PERAMBAN, bukan pengetahuan server: lapisan transport
// memetakan keduanya ke 404 yang sama, supaya endpoint ini tidak dapat dipakai menebak
// nomor case (lihat catatan pada komite.ErrNotAssigned).
func (s *InboxService) Case(
	ctx context.Context,
	caseID string,
	operator string,
) (komite.CommitteeCase, error) {
	found, err := s.cases.FindCase(ctx, caseID)
	if err != nil {
		return komite.CommitteeCase{}, err
	}
	if !found.BelongsTo(operator) {
		return komite.CommitteeCase{}, komite.ErrNotAssigned
	}

	enriched, err := s.withProgress(ctx, []komite.CommitteeCase{found})
	if err != nil {
		return komite.CommitteeCase{}, err
	}
	return enriched[0], nil
}

// Actor adalah orang yang memberi keputusan.
//
// Login dan nama dibawa BERSAMA keputusannya, tidak dirujuk ke tabel pengguna: jejak yang
// namanya diambil lewat join akan berubah ketika orangnya berganti nama, dan jejak yang
// dapat berubah bukan jejak.
type Actor struct {
	Login string
	Name  string
}

// Decide mencatat satu keputusan komite.
//
// # Urutan pemeriksaannya disengaja
//
//  1. Bentuk perintahnya sah          → 422, kesalahan pengguna
//  2. Kasusnya ada dan milik pemanggil → 404, disamakan dengan tidak ada
//  3. Komitenya masih terbuka          → 409, konflik keadaan
//  4. Baru kemudian dicatat
//
// Butir 3 menjaga sesuatu yang benar-benar terjadi: dua anggota membuka layar bersamaan,
// keduanya menekan tombol. Tanpa pemeriksaan itu, keputusan kedua tercatat sebagai
// jenjang yang sama dua kali dan penjenjangan berhenti dapat dipercaya.
//
// # Yang TIDAK dikerjakan di sini, dan itu bukan kelalaian
//
// Keputusan ini TIDAK menyentuh Pega. `PYSTATUSWORK` di sana tidak berubah dan penugasan
// worklist-nya tidak dicabut — `P-1` melarangnya. Akibatnya kasus yang sudah diputuskan
// di sini tetap terbuka di Pega selama masa paralel. Lihat catatan kepala decision.go.
func (s *InboxService) Decide(
	ctx context.Context,
	cmd komite.DecisionCommand,
	actor Actor,
) (komite.CommitteeCase, error) {
	cmd = cmd.Normalize()
	if err := cmd.Validate(); err != nil {
		return komite.CommitteeCase{}, err
	}

	current, err := s.Case(ctx, cmd.CaseID, actor.Login)
	if err != nil {
		return komite.CommitteeCase{}, err
	}

	if current.Progress.Closed() {
		return komite.CommitteeCase{}, komite.ErrDecisionClosed
	}
	if current.Progress.DecidedBy(actor.Login) {
		return komite.CommitteeCase{}, komite.ErrDecisionClosed
	}

	decision := komite.Decision{
		ID:          s.ids.New(),
		CaseID:      current.CaseID,
		ClaimNumber: current.ClaimNumber,

		// Jenjangnya ditetapkan SERVER dari keadaan kasus, tidak pernah dikirim klien.
		// Klien yang boleh menyebut jenjangnya sendiri dapat menyetujui jenjang yang
		// bukan gilirannya.
		Tier: current.Progress.CurrentTier,

		Kind:       cmd.Kind,
		Note:       cmd.Note,
		ActorLogin: komite.OperatorKey(actor.Login),
		ActorName:  actor.Name,
		DecidedAt:  s.now(),
	}

	if err := s.decisions.Record(ctx, decision); err != nil {
		return komite.CommitteeCase{}, fmt.Errorf("komite/usecase: mencatat keputusan komite: %w", err)
	}

	// Keadaan sesudahnya dihitung ulang dari daftar keputusan, bukan ditebak dengan
	// menambahkan satu ke keadaan sebelumnya. Keduanya hampir selalu sama — dan "hampir"
	// itulah yang membuat perbedaannya tidak terlihat sampai seseorang membandingkannya.
	current.Progress = komite.Evaluate(
		append(append([]komite.Decision(nil), current.Progress.Decisions...), decision),
		current.TierCount,
	)
	return current, nil
}

// withProgress menumpangkan keputusan milik aplikasi ini di atas kasus warisan.
//
// Inilah tempat kedua sumber bertemu, dan satu-satunya. Kasusnya datang dari tabel Pega
// yang tidak tahu apa pun tentang keputusan kita; keputusannya datang dari tabel kita
// yang tidak memuat satu pun kolom kasus.
func (s *InboxService) withProgress(
	ctx context.Context,
	cases []komite.CommitteeCase,
) ([]komite.CommitteeCase, error) {
	if len(cases) == 0 {
		return cases, nil
	}

	ids := make([]string, 0, len(cases))
	for _, c := range cases {
		ids = append(ids, c.CaseID)
	}

	byCase, err := s.decisions.ListForCases(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("komite/usecase: membaca keputusan komite: %w", err)
	}

	result := make([]komite.CommitteeCase, 0, len(cases))
	for _, c := range cases {
		c.Progress = komite.Evaluate(byCase[c.CaseID], c.TierCount)
		result = append(result, c)
	}
	return result, nil
}

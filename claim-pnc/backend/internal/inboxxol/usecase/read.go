// Package usecase mengorkestrasi modul Inbox XOL.
//
// Enam operasi, seluruhnya membaca:
//
//	ListMasters      daftar perjanjian XOL — grid "PILIH MASTER XOL"
//	SummarizeClaims  akumulasi klaim satu perjanjian — grid utama tab 1
//	Breakdown        rincian di balik satu baris — Sec_Detail_claim_XOL
//	SearchAdvice     PLA/DLA yang sudah terbit — "Generated" dan "Cari Data"
//	ListApprovals    antrean persetujuan — tab Komite
//	ListCauseOfLoss  isi dropdown Penyebab Kerugian
//
// # Di mana pembagian kurs terjadi, dan kenapa di sini
//
// Grid tab 1 berjudul "OS Value (USD)" sedangkan tabelnya menyimpan rupiah. Sistem lama
// membaginya di lapisan aktivitas, bukan di SQL
// (`Activity/GetClaimXOL-Act.xml`: `.Currency := @toDecimal(.Currency)/local.kurs`).
//
// Tempatnya dipertahankan di lapisan yang setara — usecase — dan itu bukan sekadar
// kesetiaan: pembagian di SQL akan mengulang kurs yang sama di empat kueri berbeda, dan
// satu di antaranya yang lupa tidak akan menghasilkan galat apa pun, hanya angka yang
// salah. Di sini kursnya dibaca SEKALI dari master yang sedang dipilih.
package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"claim-pnc/internal/inboxxol"
)

// Service melayani modul Inbox XOL.
type Service struct {
	repoSelector inboxxol.RepoSelector
}

// Options adalah bahan pembentuk Service.
type Options struct {
	RepoSelector inboxxol.RepoSelector
}

// NewService membentuk layanan modul Inbox XOL.
func NewService(o Options) (*Service, error) {
	if o.RepoSelector == nil {
		return nil, errors.New("inboxxol/usecase: RepoSelector wajib diisi")
	}
	return &Service{repoSelector: o.RepoSelector}, nil
}

// repoFor memilih penyimpanan milik satu portal.
//
// Galat pemilihan portal diteruskan APA ADANYA, tidak dibungkus galat modul ini:
// transport mengenali `portal.ErrNotReady` untuk menjawab dengan kode yang sudah dikenal
// frontend, dan membungkusnya akan memutus pengenalan itu.
func (s *Service) repoFor(portalAlias string) (inboxxol.Repo, error) {
	return s.repoSelector(portalAlias)
}

// ListMasters mengembalikan seluruh perjanjian XOL.
//
// Dipakai dua tempat sekaligus: grid "PILIH MASTER XOL" di modal tab 1, dan penyedia
// kurs bagi perhitungan tab 1. Keduanya membutuhkan daftar yang sama.
func (s *Service) ListMasters(ctx context.Context, portalAlias string, caller inboxxol.Caller) ([]inboxxol.MasterXOL, error) {
	if strings.TrimSpace(caller.Login) == "" {
		return nil, inboxxol.ErrCallerUnknown
	}
	repo, err := s.repoFor(portalAlias)
	if err != nil {
		return nil, err
	}
	return repo.ListMasterXOL(ctx)
}

// ClaimOverview adalah isi grid utama tab 1 beserta perjanjian yang menjadi dasarnya.
//
// Master ikut dikembalikan, bukan hanya barisnya, karena grid rincian menampilkan Type
// Master, ID Master, Tahun, dan Kurs IDR — dan layar tidak boleh menebaknya dari daftar
// master yang mungkin sudah berubah sejak dimuat.
type ClaimOverview struct {
	Master inboxxol.MasterXOL
	Rows   []inboxxol.ClaimSummary
}

// SummarizeClaims merakit grid "DATA XOL BASED ON DOL AND COL" untuk satu perjanjian.
//
// Urutannya mengikuti `GetClaimXOL`: daftar master lebih dulu, lalu akumulasi klaim
// perjanjian yang dipilih, lalu nilainya dibagi kurs perjanjian itu.
func (s *Service) SummarizeClaims(
	ctx context.Context,
	portalAlias string,
	caller inboxxol.Caller,
	masterID string,
) (ClaimOverview, error) {
	if strings.TrimSpace(caller.Login) == "" {
		return ClaimOverview{}, inboxxol.ErrCallerUnknown
	}
	if violation := requireValue(inboxxol.FieldMasterID, masterID, "Perjanjian XOL wajib dipilih."); violation != nil {
		return ClaimOverview{}, violation
	}

	repo, err := s.repoFor(portalAlias)
	if err != nil {
		return ClaimOverview{}, err
	}

	master, err := findMaster(ctx, repo, masterID)
	if err != nil {
		return ClaimOverview{}, err
	}

	filter := inboxxol.ClaimFilter{
		Year:             master.Year,
		BusinessGroupIDs: master.BusinessGroupIDs(),
	}

	// Perjanjian tanpa group business tidak punya klaim yang dapat dijumlahkan. Ia
	// dijawab daftar kosong, BUKAN galat: master seperti itu memang ada — ia baru
	// dibuat dan belum diisi — dan menolaknya akan membuat layar tampak rusak.
	if filter.Empty() {
		return ClaimOverview{Master: master}, nil
	}

	rows, err := repo.SummarizeClaims(ctx, filter)
	if err != nil {
		return ClaimOverview{}, fmt.Errorf("inboxxol/usecase: akumulasi klaim: %w", err)
	}

	businessGroupNames := master.BusinessGroupNames()
	for i := range rows {
		// Nama group business TIDAK datang dari kueri akumulasi — sistem lama mengisinya
		// dari master yang sedang dipilih, sehingga seluruh baris bernilai sama.
		rows[i].BusinessGroup = businessGroupNames
		rows[i].OutstandingValue = convert(rows[i].OutstandingValue, master.ExchangeRate)
		rows[i].AcceptedValue = convert(rows[i].AcceptedValue, master.ExchangeRate)
	}

	return ClaimOverview{Master: master, Rows: rows}, nil
}

// Breakdown mengembalikan rincian di balik satu baris grid utama.
//
// Dua sumber disatukan menjadi satu daftar, persis seperti `GetClaimXOL` menyusun
// `DLAList`: klaim milik sendiri per group business, lalu satu baris treaty inward.
// Urutannya dipertahankan — treaty inward selalu paling bawah.
func (s *Service) Breakdown(
	ctx context.Context,
	portalAlias string,
	caller inboxxol.Caller,
	masterID string,
	lossDate string,
	causeOfLoss string,
) ([]inboxxol.BusinessBreakdown, error) {
	if strings.TrimSpace(caller.Login) == "" {
		return nil, inboxxol.ErrCallerUnknown
	}

	violations := make([]inboxxol.Violation, 0, 3)
	violations = appendRequired(violations, inboxxol.FieldMasterID, masterID, "Perjanjian XOL wajib dipilih.")
	violations = appendRequired(violations, inboxxol.FieldLossDate, lossDate, "Tanggal Kejadian wajib diisi.")
	violations = appendRequired(violations, inboxxol.FieldCauseOfLoss, causeOfLoss, "Penyebab Kerugian wajib diisi.")
	if err := inboxxol.NewValidationError(violations); err != nil {
		return nil, err
	}

	repo, err := s.repoFor(portalAlias)
	if err != nil {
		return nil, err
	}

	master, err := findMaster(ctx, repo, masterID)
	if err != nil {
		return nil, err
	}

	filter := inboxxol.BreakdownFilter{
		LossDate:         strings.TrimSpace(lossDate),
		CauseOfLoss:      strings.TrimSpace(causeOfLoss),
		BusinessGroupIDs: master.BusinessGroupIDs(),
	}

	result := make([]inboxxol.BusinessBreakdown, 0, len(master.BusinessGroups)+1)

	if !filter.Empty() {
		own, err := repo.BreakdownByBusiness(ctx, filter)
		if err != nil {
			return nil, fmt.Errorf("inboxxol/usecase: rincian per group business: %w", err)
		}
		for i := range own {
			// Hanya klaim milik sendiri yang dibagi kurs. Baris treaty inward sudah
			// dikonversi di kuerinya sendiri — membaginya lagi akan mengecilkan
			// nilainya sebesar kurs untuk kedua kalinya.
			own[i].OutstandingValue = convert(own[i].OutstandingValue, master.ExchangeRate)
			own[i].AcceptedValue = convert(own[i].AcceptedValue, master.ExchangeRate)
			own[i].Source = inboxxol.SourceOwnBusiness
		}
		result = append(result, own...)
	}

	if !filter.TreatyEmpty() {
		inward, err := repo.BreakdownTreatyInward(ctx, filter)
		if err != nil {
			return nil, fmt.Errorf("inboxxol/usecase: rincian treaty inward: %w", err)
		}
		for i := range inward {
			inward[i].Source = inboxxol.SourceTreatyInward
		}
		result = append(result, inward...)
	}

	return result, nil
}

// SearchAdvice mengembalikan pemberitahuan PLA atau DLA yang sudah diterbitkan.
//
// Melayani dua tombol sekaligus — "Generated DLA PLA XOL" dan "Cari Data DLA PLA XOL" —
// karena keduanya menjalankan aktivitas yang sama dengan parameter yang sama
// (`Activity/BrowseDataXOLPLADLAGenerated-Act.xml`). Yang membedakannya di sistem lama
// hanyalah dari mana nilai penyaringnya diambil.
func (s *Service) SearchAdvice(
	ctx context.Context,
	portalAlias string,
	caller inboxxol.Caller,
	filter inboxxol.AdviceFilter,
) ([]inboxxol.Advice, error) {
	if strings.TrimSpace(caller.Login) == "" {
		return nil, inboxxol.ErrCallerUnknown
	}

	violations := make([]inboxxol.Violation, 0, 3)
	violations = appendRequired(violations, inboxxol.FieldYear, filter.Year, "Tahun XOL wajib diisi.")
	violations = appendRequired(violations, inboxxol.FieldCauseOfLoss, filter.CauseOfLoss, "Penyebab Kerugian wajib diisi.")
	if !filter.Type.Valid() {
		violations = append(violations, inboxxol.Violation{
			Field:   inboxxol.FieldAdviceType,
			Message: "Tipe pemberitahuan harus PLA atau DLA.",
		})
	}
	if err := inboxxol.NewValidationError(violations); err != nil {
		return nil, err
	}

	repo, err := s.repoFor(portalAlias)
	if err != nil {
		return nil, err
	}

	clean := inboxxol.AdviceFilter{
		Year:        strings.TrimSpace(filter.Year),
		CauseOfLoss: strings.TrimSpace(filter.CauseOfLoss),
		Type:        filter.Type,
	}
	rows, err := repo.SearchAdvice(ctx, clean)
	if err != nil {
		return nil, fmt.Errorf("inboxxol/usecase: pencarian PLA/DLA: %w", err)
	}
	return rows, nil
}

// ApprovalQueue adalah isi tab "Inbox XOL Komite": dua antrean yang berdiri sendiri.
//
// Keduanya dikembalikan bersama karena tabnya memuat keduanya sekaligus, dan dua
// permintaan terpisah akan membuat kedua grid dapat berasal dari saat yang berbeda.
type ApprovalQueue struct {
	// Advices adalah antrean persetujuan pemberitahuan PLA/DLA.
	Advices []inboxxol.ApprovalItem

	// Masters adalah antrean persetujuan perjanjian XOL yang baru diajukan.
	Masters []inboxxol.MasterXOL
}

// ListApprovals merakit kedua antrean pada tab Komite.
func (s *Service) ListApprovals(ctx context.Context, portalAlias string, caller inboxxol.Caller) (ApprovalQueue, error) {
	if strings.TrimSpace(caller.Login) == "" {
		return ApprovalQueue{}, inboxxol.ErrCallerUnknown
	}
	repo, err := s.repoFor(portalAlias)
	if err != nil {
		return ApprovalQueue{}, err
	}

	advices, err := repo.ListPendingAdviceApproval(ctx)
	if err != nil {
		return ApprovalQueue{}, fmt.Errorf("inboxxol/usecase: antrean persetujuan PLA/DLA: %w", err)
	}
	masters, err := repo.ListPendingMasterApproval(ctx)
	if err != nil {
		return ApprovalQueue{}, fmt.Errorf("inboxxol/usecase: antrean persetujuan master XOL: %w", err)
	}

	return ApprovalQueue{Advices: advices, Masters: masters}, nil
}

// ListCauseOfLoss mengembalikan isi dropdown Penyebab Kerugian pada modal.
func (s *Service) ListCauseOfLoss(ctx context.Context, portalAlias string, caller inboxxol.Caller) ([]inboxxol.CauseOfLoss, error) {
	if strings.TrimSpace(caller.Login) == "" {
		return nil, inboxxol.ErrCallerUnknown
	}
	repo, err := s.repoFor(portalAlias)
	if err != nil {
		return nil, err
	}
	return repo.ListCauseOfLoss(ctx)
}

// findMaster mencari satu perjanjian XOL dari daftar seluruh perjanjian.
//
// # Kenapa seluruh daftar dibaca alih-alih satu baris dicari langsung
//
// Karena satu perjanjian bukan satu baris: group business-nya ada di tabel lain, dan
// sistem lama merakitnya lewat fungsi basis data yang `D-02` larang dipanggil. Membaca
// keduanya sekaligus adalah satu-satunya cara memperoleh perjanjian yang utuh dengan dua
// kueri, bukan dua kueri per perjanjian.
//
// Jumlah perjanjian XOL adalah jumlah TAHUN perjanjian, bukan jumlah klaim — puluhan
// baris, bukan puluhan juta. Membacanya seluruhnya karena itu bukan persoalan volume.
func findMaster(ctx context.Context, repo inboxxol.Repo, masterID string) (inboxxol.MasterXOL, error) {
	masters, err := repo.ListMasterXOL(ctx)
	if err != nil {
		return inboxxol.MasterXOL{}, fmt.Errorf("inboxxol/usecase: daftar perjanjian XOL: %w", err)
	}
	wanted := strings.TrimSpace(masterID)
	for _, master := range masters {
		if strings.TrimSpace(master.ID) == wanted {
			return master, nil
		}
	}
	return inboxxol.MasterXOL{}, inboxxol.ErrMasterNotFound
}

// convert mengubah nilai rupiah menjadi mata uang perjanjian.
//
// Kurs nol atau negatif mengembalikan NOL, bukan hasil pembagian.
//
// Ini bukan kehati-hatian berlebihan. `KURSVALUE` tidak punya constraint yang dapat
// diperiksa (`R-08`), dan sistem lama membaginya tanpa memeriksa apa pun — pembagian
// dengan nol pada `@toDecimal(.Currency)/local.kurs` menghasilkan nilai tak hingga yang
// kemudian ditampilkan sebagai angka. Nol lebih jujur daripada tak hingga, dan lebih
// mudah terlihat daripada angka yang tampak masuk akal.
func convert(amount, rate float64) float64 {
	if rate <= 0 {
		return 0
	}
	return amount / rate
}

// requireValue mengembalikan galat validasi bila satu isian wajib kosong.
func requireValue(field, value, message string) error {
	return inboxxol.NewValidationError(appendRequired(nil, field, value, message))
}

// appendRequired menambahkan pelanggaran bila isian wajib kosong.
func appendRequired(violations []inboxxol.Violation, field, value, message string) []inboxxol.Violation {
	if strings.TrimSpace(value) == "" {
		return append(violations, inboxxol.Violation{Field: field, Message: message})
	}
	return violations
}

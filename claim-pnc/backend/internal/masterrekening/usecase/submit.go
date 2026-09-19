// Package usecase mengorkestrasi alur Master Rekening.
//
// Ia memegang urutan langkah dan aturan yang melibatkan lebih dari satu seam; aturan
// yang hanya menyangkut satu rekening tetap tinggal di paket domain.
//
// Lapisan Aplikasi — boleh mengimpor paket domain, dilarang mengimpor HTTP dan SQL.
package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"claim-pnc/internal/masterrekening"
	"claim-pnc/internal/platform/clock"
)

// Submitter adalah orang yang mengajukan rekening.
//
// Ia dipisahkan dari isian formulir supaya siapa yang menginput tidak pernah dapat
// dikirim dari peramban — nilainya selalu berasal dari sesi.
type Submitter struct {
	// Identity adalah NIK karyawan atau LOGIN_ID non-karyawan.
	Identity string
	Name     string
	Email    string
}

// Submission adalah isian formulir master rekening.
//
// Ia sengaja bukan masterrekening.Account: field yang dimiliki sistem — status
// persetujuan, komite, waktu keputusan, dan seluruh jejak Kasir — tidak boleh dapat
// diisi dari luar.
type Submission struct {
	Number      string
	OwnerName   string
	BankName    string
	BankBranch  string
	BankAddress string
	BankCode    string
	AccountType string
	Email       string
	Phone       string
	NIK         string
	DocumentID  string
	Note        string
	Active      bool

	// Tiga field berikut hanya terisi bila pengajuan ini menggantikan rekening lama
	// pada klaim yang sedang berjalan.
	PreviousBankCode  string
	PreviousNumber    string
	PreviousOwnerName string
}

// Service menjalankan alur Master Rekening.
type Service struct {
	repo     masterrekening.Repo
	bank     masterrekening.BankRepo
	cashier  masterrekening.Cashier
	notifier masterrekening.Notifier
	clock    clock.Clock

	// portalAlias adalah alias portal yang sedang dilayani. Ia menentukan apakah
	// rekening yang disetujui didaftarkan ke Kasir — lihat Decide.
	portalAlias string

	// komiteBaku adalah identitas komite yang menerima pengajuan bila tidak ada yang
	// ditunjuk. Lihat catatan di Submit.
	defaultCommittee string
}

// Options adalah bahan pembentuk Service.
type Options struct {
	Repo     masterrekening.Repo
	Bank     masterrekening.BankRepo
	Cashier  masterrekening.Cashier
	Notifier masterrekening.Notifier
	Clock    clock.Clock

	PortalAlias      string
	DefaultCommittee string
}

// NewService membentuk layanan; Repo dan Jam wajib terisi.
func NewService(o Options) *Service {
	return &Service{
		repo:             o.Repo,
		bank:             o.Bank,
		cashier:          o.Cashier,
		notifier:         o.Notifier,
		clock:            o.Clock,
		portalAlias:      strings.ToUpper(strings.TrimSpace(o.PortalAlias)),
		defaultCommittee: o.DefaultCommittee,
	}
}

// Submit mendaftarkan rekening baru dengan status menunggu keputusan komite.
//
// Urutannya mengikuti CNMUpdateMasterRekening_act dan tidak boleh dibalik:
//
//  1. Periksa kelengkapan isian.
//  2. Cari nomor rekening yang sama. Bila ada dan BUKAN bekas penolakan komite,
//     pengajuan ditolak.
//  3. Bila ada dan bekas penolakan, baris lama dibuang lalu pengajuan disisipkan.
//  4. Rekening baru selalu lahir dengan status menunggu — tidak ada jalan bagi
//     submitter untuk menerbitkan rekening yang langsung disetujui.
func (l *Service) Submit(ctx context.Context, p Submission, oleh Submitter) (masterrekening.Account, error) {
	sekarang := l.clock.Now()

	r := masterrekening.Account{
		Number:            tidy(p.Number),
		OwnerName:         tidy(p.OwnerName),
		BankName:          tidy(p.BankName),
		BankBranch:        tidy(p.BankBranch),
		BankAddress:       tidy(p.BankAddress),
		BankCode:          tidy(p.BankCode),
		AccountType:       tidy(p.AccountType),
		Email:             tidy(p.Email),
		Phone:             tidy(p.Phone),
		NIK:               tidy(p.NIK),
		DocumentID:        tidy(p.DocumentID),
		Note:              strings.TrimSpace(p.Note),
		Active:            p.Active,
		PreviousBankCode:  tidy(p.PreviousBankCode),
		PreviousNumber:    tidy(p.PreviousNumber),
		PreviousOwnerName: tidy(p.PreviousOwnerName),

		// Ditetapkan sistem, bukan dikirim peramban.
		Status:            masterrekening.StatusPending,
		CommitteeApproval: l.defaultCommittee,
		CreatedBy:         oleh.Identity,
		CreatedAt:         sekarang,
		UpdatedBy:         oleh.Identity,
		SubmitterEmail:    tidy(oleh.Email),
	}

	if err := r.Check(); err != nil {
		return masterrekening.Account{}, err
	}

	serupa, err := l.repo.FindByNumber(ctx, r.Number)
	if err != nil {
		return masterrekening.Account{}, fmt.Errorf("masterrekening/usecase: memeriksa duplikasi: %w", err)
	}
	if !masterrekening.CanBeResubmitted(serupa) {
		return masterrekening.Account{}, masterrekening.ErrAlreadyExists
	}

	// Baris bekas penolakan dibuang lebih dulu. Perilaku sistem lama dipertahankan
	// apa adanya (P-5); utang tekniknya dicatat di masterrekening.CanBeResubmitted.
	for _, lama := range serupa {
		if lama.Status != masterrekening.StatusRejected {
			continue
		}
		if err := l.repo.ClearRejected(ctx, lama.KeyOf()); err != nil {
			return masterrekening.Account{}, fmt.Errorf("masterrekening/usecase: membuang pengajuan yang ditolak: %w", err)
		}
	}

	if err := l.repo.Save(ctx, r); err != nil {
		return masterrekening.Account{}, fmt.Errorf("masterrekening/usecase: menyimpan rekening: %w", err)
	}
	return r, nil
}

// Update memperbarui rekening yang sudah ada.
//
// Account yang sudah decided komite tidak dapat diubah lewat jalur ini: mengubah
// nomor rekening yang sudah disetujui berarti uang klaim berpindah tujuan tanpa
// seorang pun menyetujuinya. Submission baru adalah jalannya.
func (l *Service) Update(ctx context.Context, k masterrekening.Key, p Submission, oleh Submitter) (masterrekening.Account, error) {
	existing, err := l.repo.Get(ctx, k)
	if err != nil {
		return masterrekening.Account{}, err
	}
	if !existing.AwaitingDecision() {
		return masterrekening.Account{}, masterrekening.ErrAlreadyDecided
	}

	existing.OwnerName = tidy(p.OwnerName)
	existing.BankName = tidy(p.BankName)
	existing.BankBranch = tidy(p.BankBranch)
	existing.BankAddress = tidy(p.BankAddress)
	existing.AccountType = tidy(p.AccountType)
	existing.Email = tidy(p.Email)
	existing.Phone = tidy(p.Phone)
	existing.NIK = tidy(p.NIK)
	existing.Active = p.Active
	existing.UpdatedBy = oleh.Identity
	if d := tidy(p.DocumentID); d != "" {
		existing.DocumentID = d
	}
	if c := strings.TrimSpace(p.Note); c != "" {
		existing.Note = c
	}

	if err := existing.Check(); err != nil {
		return masterrekening.Account{}, err
	}
	if err := l.repo.Update(ctx, existing); err != nil {
		return masterrekening.Account{}, fmt.Errorf("masterrekening/usecase: memperbarui rekening: %w", err)
	}
	return existing, nil
}

// List membaca rekening yang cocok dengan filter.
func (l *Service) List(ctx context.Context, f masterrekening.Filter) ([]masterrekening.Account, int, error) {
	rows, total, err := l.repo.List(ctx, f)
	if err != nil {
		return nil, 0, fmt.Errorf("masterrekening/usecase: membaca daftar rekening: %w", err)
	}
	return rows, total, nil
}

// Get membaca satu rekening.
func (l *Service) Get(ctx context.Context, k masterrekening.Key) (masterrekening.Account, error) {
	return l.repo.Get(ctx, k)
}

// ListBanks membaca daftar bank dari GENERAL.LST_BANK_GROUP.
func (l *Service) ListBanks(ctx context.Context) ([]masterrekening.Bank, error) {
	if l.bank == nil {
		return nil, errors.New("masterrekening/usecase: sumber daftar bank belum dipasang")
	}
	list, err := l.bank.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("masterrekening/usecase: membaca daftar bank: %w", err)
	}
	return list, nil
}

func tidy(s string) string { return strings.TrimSpace(s) }

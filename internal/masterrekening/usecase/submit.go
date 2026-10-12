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

	// SubmitterEmail adalah "Email Inputor" pada layar Pega — alamat yang dipakai
	// Kasir mengirim pemberitahuan approval kembali ke pengaju.
	//
	// Ia DIISI PENGGUNA, bukan diambil dari sesi. Layar lama menandainya wajib dan
	// memberinya keterangan "Wajib masukan email Anda untuk Notif Approval dari
	// Kasir" — menandakan alamatnya tidak selalu sama dengan surel akun yang sedang
	// masuk. Mengambilnya diam-diam dari sesi akan mengirim pemberitahuan ke alamat
	// yang bukan dimaksud pengaju, dan ia tidak punya cara memperbaikinya.
	SubmitterEmail string

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

		// Email Inputor berasal dari isian, bukan dari sesi. Surel sesi dipakai hanya
		// sebagai nilai awal bila isiannya kosong — supaya pengaju yang memang memakai
		// alamat akunnya sendiri tidak perlu mengetik ulang, tanpa menghilangkan
		// kemampuan menggantinya.
		SubmitterEmail: pilihTerisi(p.SubmitterEmail, oleh.Email),
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
// # Rekening yang SUDAH diputuskan komite boleh diubah
//
// Sebelumnya jalur ini menolaknya dengan ErrAlreadyDecided, dan itu **terlalu keras**.
// Layar lama menyediakan tombol `Ubah` pada setiap baris — termasuk pada tab `Approve` —
// dan `RDB List/UpdateMasterRekening-SQL.xml` memang menulis ulang `approval`,
// `OLDBANID`, dan `OLDACCOUNT_NO` sekaligus. Mengubah rekening yang sudah disetujui
// adalah alur yang memang ada, bukan yang dilarang.
//
// # Tetapi ia kembali menunggu persetujuan
//
// Kekhawatiran yang mendasari penolakan lama tetap sah: mengubah nomor rekening yang
// sudah disetujui berarti uang klaim berpindah tujuan. Jawabannya bukan melarang
// perubahannya, melainkan **mencabut persetujuannya** — rekening kembali ke status
// menunggu, dan komite memutuskan ulang atas data yang baru.
//
// Itu pula yang dilakukan sistem lama: activity `CNMUpdateMasterRekening_act` memanggil
// `GetKomiteApproval` saat menyimpan, dan menugaskan komite hanya masuk akal bila
// rekeningnya memang kembali menunggu keputusan.
//
// Keadaan sebelum diubah disimpan di `PreviousBankCode`, `PreviousNumber`, dan
// `PreviousOwnerName` — padanan `OLDBANID`, `OLDACCOUNT_NO`, dan `ACCOUNTNAMEOLD`.
// Tanpa itu, tidak ada yang dapat menjawab "rekening ini dulunya apa" setelah
// perubahannya disetujui.
func (l *Service) Update(ctx context.Context, k masterrekening.Key, p Submission, oleh Submitter) (masterrekening.Account, error) {
	existing, err := l.repo.Get(ctx, k)
	if err != nil {
		return masterrekening.Account{}, err
	}

	// Dicatat SEBELUM satu field pun ditimpa.
	sudahDiputuskan := !existing.AwaitingDecision()
	sebelumnya := existing

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
	existing.SubmitterEmail = pilihTerisi(p.SubmitterEmail, existing.SubmitterEmail)
	if d := tidy(p.DocumentID); d != "" {
		existing.DocumentID = d
	}
	if c := strings.TrimSpace(p.Note); c != "" {
		existing.Note = c
	}

	if sudahDiputuskan {
		// Persetujuan lama dicabut: datanya sudah bukan data yang disetujui komite.
		existing.Status = masterrekening.StatusPending
		existing.DecidedAt = nil
		existing.CommitteeApproval = l.defaultCommittee
		existing.ChangeFlag = flagDiubah

		existing.PreviousBankCode = sebelumnya.BankCode
		existing.PreviousNumber = sebelumnya.Number
		existing.PreviousOwnerName = sebelumnya.OwnerName

		// Jejak Kasir SENGAJA tidak dihapus.
		//
		// `UpdateMasterRekening-SQL.xml` pun tidak menyentuh STS_SERVICE, ID_REKASIR,
		// maupun RESPONSE_KASIR (P-5). Lebih dari itu, pendaftaran yang lama adalah
		// fakta yang benar-benar terjadi — menghapusnya membuat rekening tampak belum
		// pernah didaftarkan padahal Kasir masih memegang catatannya. Ia ditimpa dengan
		// sendirinya saat komite menyetujui ulang dan pendaftaran dijalankan lagi.
	}

	if err := existing.Check(); err != nil {
		return masterrekening.Account{}, err
	}
	if err := l.repo.Update(ctx, existing); err != nil {
		return masterrekening.Account{}, fmt.Errorf("masterrekening/usecase: memperbarui rekening: %w", err)
	}
	return existing, nil
}

// flagDiubah adalah isi kolom FLAGUPDATE untuk rekening yang pernah diubah setelah
// diputuskan komite.
//
// Nilainya "U" — huruf itu TIDAK terbaca dari export (`FLAGUPDATE` tidak pernah muncul
// di satu pun rule yang menulisnya), sehingga ia ditetapkan di sini, bukan disalin.
// Yang penting bukan hurufnya melainkan keberadaannya: tanpa penanda, baris yang pernah
// diubah tidak dapat dibedakan dari yang sejak awal begitu.
const flagDiubah = "U"

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

// pilihTerisi mengembalikan nilai pertama yang tidak kosong setelah dirapikan.
//
// Dipakai pada Email Inputor: isian pengguna menang, dan surel sesi hanya menjadi
// nilai awal bila isiannya memang kosong.
func pilihTerisi(utama, cadangan string) string {
	if v := tidy(utama); v != "" {
		return v
	}
	return tidy(cadangan)
}

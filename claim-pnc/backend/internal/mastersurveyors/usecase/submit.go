// Package usecase mengorkestrasi alur Master Surveyors: melihat daftar, membuka satu
// baris, mengajukan, mengubah, dan memutuskan sebagai komite.
//
// Ia memegang urutan langkah dan aturan yang melibatkan lebih dari satu seam; aturan yang
// hanya menyangkut satu surveyor tetap tinggal di paket domain.
//
// # Satu aksi yang sengaja tidak ada
//
// Tidak ada Hapus. Itu bukan kelalaian:
//
//   - `Section/GridDetailSurveyors-Section.xml` hanya memuat tombol **Tambah** dan
//     **Refresh** — tidak ada tombol hapus.
//   - `Database/PEGA_D_SURVEYORS.prc` hanya mengenal INSERT dan UPDATE.
//   - Menghapus seorang surveyor akan memutus setiap hasil survei yang merujuknya, dan
//     hasil survei adalah dasar nilai klaim yang sudah dibayarkan.
//
// Persis alasan `ADR-0012` menetapkan master tidak dihapus permanen, dan sejalan `D-66`
// yang melarang penghapusan fisik data bernilai bisnis.
//
// Lapisan Aplikasi — boleh mengimpor paket domain, dilarang mengimpor HTTP dan SQL.
package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"claim-pnc/internal/mastersurveyors"
	"claim-pnc/internal/platform/clock"
)

// Submitter adalah orang yang mengajukan atau mengubah surveyor.
//
// Ia dipisahkan dari isian formulir supaya siapa yang menginput tidak pernah dapat
// dikirim dari peramban — nilainya selalu berasal dari sesi.
type Submitter struct {
	// Identity adalah NIK karyawan atau LOGIN_ID non-karyawan.
	Identity string
	Name     string
}

// Submission adalah isian formulir master surveyor.
//
// Ia sengaja BUKAN mastersurveyors.Surveyor: field yang dimiliki sistem — status
// persetujuan, komite yang ditunjuk, waktu keputusan, dan catatan komite — tidak boleh
// dapat diisi dari luar. Mengirimkannya sebagai satu struct yang sama akan membuat
// peramban dapat menyetujui surveyornya sendiri.
//
// Kedua belas field di bawah adalah PERSIS yang disalin
// `Activity/SetDetailSurveryorsValue_act-Act.xml` ke halaman TempDetailSurveyors, dikurangi
// yang dimiliki sistem.
type Submission struct {
	TypeCode     string
	Name         string
	Address      string
	PostalCode   string
	State        string
	Phone        string
	Fax          string
	Email        string
	OtherContact string
	BranchCode   string
	BranchName   string
	AppLogin     string
	DocumentID   string
}

// Service menjalankan alur Master Surveyors di atas seam penyimpanan per portal.
type Service struct {
	repoSelector mastersurveyors.RepoSelector
	committee    mastersurveyors.CommitteeResolver
	accounts     mastersurveyors.AccountRegistrar
	clock        clock.Clock
}

// Options adalah bahan pembentuk Service.
type Options struct {
	// RepoSelector memilih penyimpanan milik satu portal entitas. Wajib.
	RepoSelector mastersurveyors.RepoSelector

	// Committee menetapkan komite yang berwenang memutuskan. Wajib.
	Committee mastersurveyors.CommitteeResolver

	// Accounts membuat akun aplikasi surveyor. Wajib.
	Accounts mastersurveyors.AccountRegistrar

	// Clock adalah sumber waktu. Boleh nil; bila nil dipakai jam sistem.
	Clock clock.Clock
}

// NewService membentuk Service dan menolak bahan yang tidak lengkap.
//
// Penolakannya terjadi saat perakitan di cmd, bukan saat permintaan pertama datang:
// rakitan yang setengah jadi harus gagal saat start, bukan saat pengguna sedang bekerja.
func NewService(o Options) (*Service, error) {
	if o.RepoSelector == nil {
		return nil, errors.New("mastersurveyors/usecase: RepoSelector wajib diisi")
	}
	if o.Committee == nil {
		return nil, errors.New("mastersurveyors/usecase: Committee wajib diisi")
	}
	if o.Accounts == nil {
		return nil, errors.New("mastersurveyors/usecase: Accounts wajib diisi")
	}
	jam := o.Clock
	if jam == nil {
		jam = clock.System{}
	}
	return &Service{
		repoSelector: o.RepoSelector,
		committee:    o.Committee,
		accounts:     o.Accounts,
		clock:        jam,
	}, nil
}

// List mengembalikan surveyor yang cocok dengan filter beserta jumlah seluruhnya.
func (s *Service) List(ctx context.Context, portalAlias string, f mastersurveyors.Filter) ([]mastersurveyors.Surveyor, int, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return nil, 0, err
	}

	rows, total, err := repo.List(ctx, f)
	if err != nil {
		return nil, 0, fmt.Errorf("mastersurveyors/usecase: membaca daftar surveyor: %w", err)
	}
	return rows, total, nil
}

// Get mengembalikan satu surveyor.
//
// Ia menggantikan `SetDetailSurveryorsValue_act(dsurveyid)`, yang menjalankan Report
// Definition `SelectVDSurveyors_RD` lalu menyalin hasilnya ke halaman
// TempDetailSurveyors.
func (s *Service) Get(ctx context.Context, portalAlias, id string) (mastersurveyors.Surveyor, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return mastersurveyors.Surveyor{}, err
	}

	surveyor, err := repo.Get(ctx, strings.TrimSpace(id))
	if err != nil {
		if errors.Is(err, mastersurveyors.ErrNotFound) {
			return mastersurveyors.Surveyor{}, err
		}
		return mastersurveyors.Surveyor{}, fmt.Errorf("mastersurveyors/usecase: membaca surveyor %q: %w", id, err)
	}
	return surveyor, nil
}

// Submit mengajukan surveyor baru.
//
// # Urutan langkahnya, dan kenapa urutannya begitu
//
// Mengikuti `CNMInsertDetailSurveyors_act` sejauh langkah-langkahnya dapat dibawa:
//
//  1. periksa kelengkapan isian                     (langkah 8 dan 9 di Pega)
//  2. periksa nama ganda                            (langkah 9, ValidasiMasterSurveyor)
//  3. periksa bentrok login di dalam master ini     (langkah 10–14)
//  4. tetapkan komite                               (langkah 6–7, GetKomiteApproval)
//  5. simpan                                        (langkah 15–17)
//  6. mintakan akun aplikasi                        (langkah 18, GCNMCreateOperator)
//
// Langkah 6 SENGAJA berada SESUDAH penyimpanan, sama seperti di Pega. Alasannya sama
// dengan yang dicatat `masterrekening.Decide` tentang Kasir: pembuatan akun menyentuh
// sistem di luar basis data ini, dan kegagalannya tidak boleh membatalkan surveyor yang
// sudah sah diajukan.
//
// Perbedaannya dari Pega — dan ia disengaja: langkah 1 sampai 3 di sini dijalankan SEBELUM
// apa pun ditulis. Pega menyimpan lampiran pada langkah 2, yaitu sebelum validasi apa pun,
// sehingga isian yang ditolak tetap meninggalkan berkas menggantung tanpa pemilik.
func (s *Service) Submit(ctx context.Context, portalAlias string, in Submission, by Submitter) (mastersurveyors.Surveyor, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return mastersurveyors.Surveyor{}, err
	}

	surveyor := surveyorFrom(in)
	surveyor.Status = mastersurveyors.StatusPending
	surveyor.CreatedBy = strings.TrimSpace(by.Identity)
	surveyor.CreatedAt = s.clock.Now()

	if err := surveyor.Check(); err != nil {
		return mastersurveyors.Surveyor{}, err
	}
	if err := ensureNameFree(ctx, repo, surveyor.Name, ""); err != nil {
		return mastersurveyors.Surveyor{}, err
	}
	if err := ensureLoginFree(ctx, repo, surveyor.AppLogin, ""); err != nil {
		return mastersurveyors.Surveyor{}, err
	}

	// Komite ditetapkan SEKALI, di sini. Prasyarat aslinya `TempDetailSurveyors.KOMITE==""`
	// pada langkah 6 — yang berarti penyuntingan berikutnya tidak menetapkannya ulang.
	// Kueri surveyor_update karena itu tidak menyentuh kolom KOMITE sama sekali.
	committee, err := s.committee.Resolve(ctx, portalAlias, surveyor)
	if err != nil {
		return mastersurveyors.Surveyor{}, fmt.Errorf("mastersurveyors/usecase: menetapkan komite: %w", err)
	}
	// Komite kosong TIDAK menggagalkan pengajuan. Itu bukan kelonggaran: sistem lama pun
	// mengambil `pxResults(1)` tanpa memeriksa apakah ada hasilnya, sehingga surveyor
	// berkomite kosong adalah keadaan yang memang dapat terjadi di sana. Akibatnya ia
	// tidak muncul di antrean komite mana pun sampai seseorang menetapkannya — dan itu
	// terlihat di tab "Menunggu Approval", bukan hilang.
	surveyor.Committee = strings.TrimSpace(committee)

	saved, err := repo.Insert(ctx, surveyor)
	if err != nil {
		return mastersurveyors.Surveyor{}, translateRepoError(err, "menyimpan surveyor")
	}

	if err := s.requestAccount(ctx, portalAlias, saved); err != nil {
		return mastersurveyors.Surveyor{}, err
	}
	return saved, nil
}

// Update mengubah surveyor yang sudah ada.
//
// # Menyunting MENGEMBALIKAN surveyor ke antrean komite
//
// Ini perilaku Pega, bukan tambahan — dan ia sempat TIDAK dibawa, lalu dibetulkan pada
// 2026-09-20 setelah langkah 4 `CNMInsertDetailSurveyors_act` dibaca sampai prasyaratnya:
//
//	pyStepsPreCondition: true          (tanpa syarat, selalu jalan)
//	TempDetailSurveyors.APPROVAL := Param.approval
//
// Tombol Simpan pada tab "Approve" dan "Reject" mengirim `approval = "0"`. Artinya
// menyunting surveyor yang sudah disetujui — atau sudah ditolak — MENGEMBALIKANNYA ke
// posisi menunggu.
//
// Itu kontrol yang nyata, bukan kejanggalan: tanpa itu, nama login dan alamat surveyor
// dapat diubah setelah komite menyetujuinya, dan komite tidak pernah melihat perubahannya.
// Penguat lainnya, `SetDetailSurveryorsValue_act` yang mengisi formulir Ubah **tidak
// menyalin APPROVAL sama sekali** — nilainya memang hanya datang dari parameter.
//
// # Yang TIDAK dapat diubah lewat jalur ini
//
// Komite yang ditunjuk. Ia ditetapkan sekali saat pengajuan pertama dan dibawa apa adanya
// — sejalan dengan kueri `surveyor_update` yang tidak menyentuh kolom KOMITE.
//
// Status persetujuan juga tidak dapat DIKIRIM pemanggil: ia selalu dipaksa menjadi
// menunggu. Tanpa itu, siapa pun yang boleh menyunting surveyor dapat menyetujuinya
// sendiri dengan menaruh status di badan permintaan — dan `D-59` sudah menetapkan tidak
// ada pemisahan tugas formal, sehingga bentuk kontraknya adalah penjagaan yang tersisa.
func (s *Service) Update(ctx context.Context, portalAlias, id string, in Submission, by Submitter) (mastersurveyors.Surveyor, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return mastersurveyors.Surveyor{}, err
	}

	existing, err := repo.Get(ctx, strings.TrimSpace(id))
	if err != nil {
		if errors.Is(err, mastersurveyors.ErrNotFound) {
			return mastersurveyors.Surveyor{}, err
		}
		return mastersurveyors.Surveyor{}, fmt.Errorf("mastersurveyors/usecase: membaca surveyor %q: %w", id, err)
	}

	changed := surveyorFrom(in)
	changed.ID = existing.ID
	changed.LegacyID = existing.LegacyID

	// Kembali ke antrean komite — meniru `approval = "0"` yang dikirim tombol Simpan.
	changed.Status = mastersurveyors.StatusPending

	// Jejak keputusan SEBELUMNYA dibuang bersamaan, dan ini bagian yang TIDAK ada
	// padanannya di Pega: kedua kolomnya ditambahkan modul ini (migrasi 0004).
	//
	// Membiarkannya akan menghasilkan baris yang statusnya "menunggu" tetapi memuat
	// tanggal dan catatan keputusan — keadaan yang tidak dapat dibaca siapa pun, dan yang
	// justru merusak jejak audit alih-alih menjaganya.
	changed.DecidedAt = nil
	changed.Note = ""

	// Komite yang ditunjuk dibawa apa adanya: ia ditetapkan sekali saat pengajuan pertama,
	// dan kueri surveyor_update memang tidak menyentuh kolomnya.
	changed.Committee = existing.Committee
	changed.CommitteeTransferred = existing.CommitteeTransferred
	changed.CreatedBy = existing.CreatedBy
	changed.CreatedAt = existing.CreatedAt
	changed.UpdatedBy = strings.TrimSpace(by.Identity)

	if err := changed.Check(); err != nil {
		return mastersurveyors.Surveyor{}, err
	}
	if err := ensureNameFree(ctx, repo, changed.Name, changed.ID); err != nil {
		return mastersurveyors.Surveyor{}, err
	}
	if err := ensureLoginFree(ctx, repo, changed.AppLogin, changed.ID); err != nil {
		return mastersurveyors.Surveyor{}, err
	}

	if err := repo.Update(ctx, changed); err != nil {
		return mastersurveyors.Surveyor{}, translateRepoError(err, "mengubah surveyor")
	}

	// Akun diminta hanya bila nama loginnya BARU terisi atau BERGANTI. Memintanya pada
	// setiap penyuntingan akan menerbitkan sandi sementara baru setiap kali alamat
	// surveyor diperbaiki — dan pemiliknya tidak akan tahu sandinya berubah.
	if changed.AppLogin != "" && !strings.EqualFold(changed.AppLogin, existing.AppLogin) {
		if err := s.requestAccount(ctx, portalAlias, changed); err != nil {
			return mastersurveyors.Surveyor{}, err
		}
	}
	return changed, nil
}

// EnsurePortalReady menyatakan portal ini punya penyimpanan yang dapat dipakai.
//
// Dipakai transport untuk menjawab lebih awal dengan pesan yang dapat ditindaklanjuti,
// alih-alih membiarkan kegagalan muncul di tengah pembacaan daftar.
func (s *Service) EnsurePortalReady(portalAlias string) error {
	_, err := s.repoSelector(portalAlias)
	return err
}

// requestAccount meminta pembuatan akun aplikasi bila surveyor ini memang memakainya.
//
// Surveyor tanpa nama login dilewati tanpa galat: surveyor eksternal memang tidak masuk
// ke aplikasi, dan meminta akun tanpa nama pengguna akan gagal dengan alasan yang
// menyesatkan.
func (s *Service) requestAccount(ctx context.Context, portalAlias string, surveyor mastersurveyors.Surveyor) error {
	if strings.TrimSpace(surveyor.AppLogin) == "" {
		return nil
	}
	req := mastersurveyors.AccountRequestFor(surveyor)
	if err := s.accounts.Register(ctx, portalAlias, req); err != nil {
		if errors.Is(err, mastersurveyors.ErrLoginTaken) {
			return err
		}
		return fmt.Errorf("mastersurveyors/usecase: meminta akun aplikasi untuk %q: %w", surveyor.AppLogin, err)
	}
	return nil
}

// surveyorFrom menyalin isian formulir menjadi nilai domain.
func surveyorFrom(in Submission) mastersurveyors.Surveyor {
	return mastersurveyors.Surveyor{
		TypeCode:     in.TypeCode,
		Name:         in.Name,
		Address:      in.Address,
		PostalCode:   in.PostalCode,
		State:        in.State,
		Phone:        in.Phone,
		Fax:          in.Fax,
		Email:        in.Email,
		OtherContact: in.OtherContact,
		BranchCode:   in.BranchCode,
		BranchName:   in.BranchName,
		AppLogin:     in.AppLogin,
		DocumentID:   in.DocumentID,
	}.Clean()
}

// ensureNameFree menolak nama yang sudah dipakai surveyor lain.
//
// exceptID dikecualikan supaya menyunting surveyor tanpa mengubah namanya tidak ditolak
// oleh namanya sendiri.
func ensureNameFree(ctx context.Context, repo mastersurveyors.Repo, name, exceptID string) error {
	rows, err := repo.FindByNameKey(ctx, name)
	if err != nil {
		return fmt.Errorf("mastersurveyors/usecase: memeriksa nama ganda: %w", err)
	}
	for _, row := range rows {
		if row.ID != exceptID {
			return mastersurveyors.ErrNameTaken
		}
	}
	return nil
}

// ensureLoginFree menolak nama login yang sudah dipakai surveyor lain.
func ensureLoginFree(ctx context.Context, repo mastersurveyors.Repo, login, exceptID string) error {
	if strings.TrimSpace(login) == "" {
		return nil
	}
	rows, err := repo.FindByAppLogin(ctx, login)
	if err != nil {
		return fmt.Errorf("mastersurveyors/usecase: memeriksa login ganda: %w", err)
	}
	for _, row := range rows {
		if row.ID != exceptID {
			return mastersurveyors.ErrLoginTaken
		}
	}
	return nil
}

// translateRepoError meneruskan galat domain apa adanya dan membungkus sisanya.
//
// Galat domain diteruskan UTUH supaya transport dapat memetakannya ke kode HTTP yang
// benar; membungkusnya dengan fmt.Errorf tanpa %w akan membuat bentrok nama muncul sebagai
// 500.
func translateRepoError(err error, activity string) error {
	switch {
	case errors.Is(err, mastersurveyors.ErrNameTaken),
		errors.Is(err, mastersurveyors.ErrLoginTaken),
		errors.Is(err, mastersurveyors.ErrNotFound),
		errors.Is(err, mastersurveyors.ErrNoSite):
		return err
	default:
		return fmt.Errorf("mastersurveyors/usecase: %s: %w", activity, err)
	}
}

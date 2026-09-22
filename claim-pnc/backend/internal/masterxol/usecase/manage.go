// Package usecase mengorkestrasi pengelolaan Master XOL: melihat daftar, membuka satu
// induk, menyimpan, menghapus, dan menyediakan bekal isian layar.
//
// Lapisan orkestrasi: ia memanggil seam yang dideklarasikan paket masterxol dan tidak
// tahu apa pun soal HTTP maupun SQL.
package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"claim-pnc/internal/masterxol"
	"claim-pnc/internal/platform/logging"
)

// Service mengelola Master XOL di atas seam penyimpanan PER PORTAL.
type Service struct {
	repoSelector masterxol.RepoSelector
	notifier     masterxol.Notifier
	logger       *slog.Logger
}

// Options adalah bahan pembentuk Service.
type Options struct {
	// RepoSelector memilih penyimpanan milik satu portal entitas. Wajib.
	RepoSelector masterxol.RepoSelector

	// Notifier mengirim pemberitahuan pengajuan ke komite. Wajib — bila pemberitahuan
	// memang tidak dikehendaki, yang dipasang adalah tiruan yang merekam, bukan nil.
	// Dengan begitu tidak ada percabangan nil di jalur simpan.
	Notifier masterxol.Notifier

	Logger *slog.Logger
}

// NewService membentuk Service dan menolak bahan yang tidak lengkap — kegagalannya
// terjadi saat start, bukan saat pengguna pertama membuka layar.
func NewService(o Options) (*Service, error) {
	if o.RepoSelector == nil {
		return nil, errors.New("masterxol/usecase: RepoSelector wajib diisi")
	}
	if o.Notifier == nil {
		return nil, errors.New("masterxol/usecase: Notifier wajib diisi")
	}
	return &Service{repoSelector: o.RepoSelector, notifier: o.Notifier, logger: o.Logger}, nil
}

// EnsurePortalReady memeriksa portal dapat dilayani tanpa menyentuh satu baris pun.
func (s *Service) EnsurePortalReady(portalAlias string) error {
	if _, err := s.repoSelector(portalAlias); err != nil {
		return fmt.Errorf("masterxol/usecase: portal %q tidak dapat dilayani: %w", portalAlias, err)
	}
	return nil
}

// List mengembalikan seluruh induk XOL tanpa anaknya.
//
// Tidak dipaginasi. Kueri lamanya pun tidak — `RDB List/GetDataMasterXOL-SQL.xml` tidak
// punya satu pun pembatas baris — dan isinya tumbuh beberapa baris per tahun treaty:
// delapan baris pada 2026-09-20, setelah sebelas tahun dipakai.
func (s *Service) List(ctx context.Context, portalAlias string) ([]masterxol.Master, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return nil, err
	}

	list, err := repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("masterxol/usecase: membaca daftar master XOL: %w", err)
	}
	return list, nil
}

// Get mengembalikan satu induk lengkap dengan bisnis, lapisan, dan reas-nya.
//
// Menggantikan `Activity/UpdateMasterXOL-Act.xml`, yang memuat keempat tingkat sekaligus
// sebelum form Ubah dibuka.
func (s *Service) Get(ctx context.Context, portalAlias, id string) (masterxol.Master, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return masterxol.Master{}, err
	}

	master, err := repo.Get(ctx, strings.TrimSpace(id))
	if err != nil {
		if errors.Is(err, masterxol.ErrNotFound) {
			return masterxol.Master{}, err
		}
		return masterxol.Master{}, fmt.Errorf("masterxol/usecase: membaca master XOL %q: %w", id, err)
	}
	return master, nil
}

// SaveResult adalah hasil penyimpanan: induk yang tersimpan beserta peringatan yang
// menyertainya.
//
// Peringatan dibedakan tegas dari galat, dan itu inti perilaku modul ini: total share
// yang belum 100% TIDAK menggagalkan penyimpanan — lihat masterxol.ShareWarning. Bila
// keduanya disatukan, layar tidak punya cara membedakan "tersimpan, tetapi perhatikan
// ini" dari "tidak tersimpan".
type SaveResult struct {
	Master  masterxol.Master
	Warning []string
}

// Save menyimpan satu induk beserta anaknya, lalu mengajukannya ke komite.
//
// # Kenapa Simpan sekaligus MENGAJUKAN KE KOMITE
//
// Karena itulah yang dilakukan layar lama, dan ia melakukannya tanpa syarat.
// `Activity/InsertUpdateMasterXOL-Act.xml` menutup rangkaiannya dengan dua pemanggilan
// berturut-turut:
//
//	pySteps(10)  Call UpdateStatusMasterKomitexol  → PIC, STSKOMITE='0', REMARKPIC
//	pySteps(11)  Call SendDataMasterXOLToKomites   → pemberitahuan, FlagKomites=1
//
// Keduanya tanpa prasyarat. Artinya menyimpan induk yang SUDAH disetujui komite akan
// mengembalikannya ke keadaan menunggu — dan itu masuk akal: strukturnya berubah,
// sehingga persetujuan atas bentuk sebelumnya tidak lagi berlaku.
//
// Perilaku ini ditiru apa adanya (keputusan Work Owner 2026-09-20, `P-5`).
//
// # Kenapa kegagalan pemberitahuan TIDAK menggagalkan penyimpanan
//
// Datanya sudah tersimpan dan pengajuannya sudah tercatat sebelum pemberitahuan dikirim.
// Menggagalkan permintaan pada titik itu akan menampilkan pesan gagal atas pekerjaan yang
// sebenarnya berhasil, dan pengguna akan menyimpan ulang — mengajukan dua kali. Sistem
// lama pun tidak memeriksa hasil pengirimannya.
//
// Kegagalannya dicatat ke log supaya tetap dapat ditelusuri, bukan hilang diam-diam.
func (s *Service) Save(ctx context.Context, portalAlias string, master masterxol.Master, caller string) (SaveResult, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return SaveResult{}, err
	}

	master = master.Clean()
	if err := masterxol.NewValidationError(masterxol.CheckMaster(master)); err != nil {
		return SaveResult{}, err
	}

	saved, err := repo.Save(ctx, master)
	if err != nil {
		switch {
		case errors.Is(err, masterxol.ErrNotFound), errors.Is(err, masterxol.ErrIDTaken):
			return SaveResult{}, err
		default:
			return SaveResult{}, fmt.Errorf("masterxol/usecase: menyimpan master XOL: %w", err)
		}
	}

	caller = strings.TrimSpace(caller)
	if err := repo.SubmitToCommittee(ctx, saved.ID, caller, saved.RemarkPIC); err != nil {
		return SaveResult{}, fmt.Errorf("masterxol/usecase: mencatat pengajuan komite: %w", err)
	}
	saved.PIC = caller
	saved.CommitteeStatus = masterxol.CommitteePending

	submission := masterxol.CommitteeSubmission{
		MasterID:     saved.ID,
		Year:         saved.Year,
		ExchangeRate: saved.ExchangeRate,
		Remark:       saved.RemarkPIC,
		SubmittedBy:  caller,
	}
	if err := s.notifier.NotifyCommitteeSubmission(ctx, submission); err != nil {
		logging.From(ctx, s.logger).Error("pemberitahuan pengajuan komite gagal dikirim",
			slog.String("modul", "masterxol"),
			slog.String("portal", portalAlias),
			slog.String("master", saved.ID),
			slog.String("galat", err.Error()),
		)
	}

	return SaveResult{Master: saved, Warning: masterxol.ShareWarning(saved)}, nil
}

// DeleteMaster menghapus satu induk beserta seluruh anaknya.
//
// Kaskadenya dijelaskan di masterxol.Repo.DeleteMaster: sistem lama tidak berkaskade dan
// meninggalkan baris yatim di produksi, dan Work Owner memutuskan itu diperbaiki.
func (s *Service) DeleteMaster(ctx context.Context, portalAlias, id string) error {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return err
	}

	if err := repo.DeleteMaster(ctx, strings.TrimSpace(id)); err != nil {
		if errors.Is(err, masterxol.ErrNotFound) {
			return err
		}
		return fmt.Errorf("masterxol/usecase: menghapus master XOL %q: %w", id, err)
	}
	return nil
}

// DeleteBusiness menghapus satu baris grup bisnis dari sebuah induk.
func (s *Service) DeleteBusiness(ctx context.Context, portalAlias, masterID, businessID string) error {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return err
	}

	if err := repo.DeleteBusiness(ctx, strings.TrimSpace(masterID), strings.TrimSpace(businessID)); err != nil {
		if errors.Is(err, masterxol.ErrNotFound) {
			return err
		}
		return fmt.Errorf("masterxol/usecase: menghapus bisnis master XOL: %w", err)
	}
	return nil
}

// DeleteLayer menghapus satu lapisan beserta seluruh reas-nya.
func (s *Service) DeleteLayer(ctx context.Context, portalAlias, masterID, layerID string) error {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return err
	}

	if err := repo.DeleteLayer(ctx, strings.TrimSpace(masterID), strings.TrimSpace(layerID)); err != nil {
		if errors.Is(err, masterxol.ErrNotFound) || errors.Is(err, masterxol.ErrLayerNotFound) {
			return err
		}
		return fmt.Errorf("masterxol/usecase: menghapus layer XOL: %w", err)
	}
	return nil
}

// DeleteReinsurer menghapus satu baris reas dari sebuah lapisan.
func (s *Service) DeleteReinsurer(ctx context.Context, portalAlias, layerID, reinsurerID string) error {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return err
	}

	if err := repo.DeleteReinsurer(ctx, strings.TrimSpace(layerID), strings.TrimSpace(reinsurerID)); err != nil {
		if errors.Is(err, masterxol.ErrLayerNotFound) {
			return err
		}
		return fmt.Errorf("masterxol/usecase: menghapus reas layer: %w", err)
	}
	return nil
}

// TypeOption adalah satu pilihan Type XOL beserta labelnya.
type TypeOption struct {
	Code  masterxol.Type
	Label string
}

// FormOption adalah bekal awal layar: pilihan tahun dan pilihan Type XOL.
//
// Keduanya dikirim dalam satu permintaan. Satu layar sebaiknya dilayani satu permintaan
// (`docs/Steering/10-API-STRATEGY.md` §1) — memecahnya menjadi dua akan membuat form
// tampak setengah siap selama sesaat setiap kali dibuka.
type FormOption struct {
	Year []string
	Type []TypeOption
}

// Form mengembalikan bekal awal layar.
func (s *Service) Form(ctx context.Context, portalAlias string) (FormOption, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return FormOption{}, err
	}

	year, err := repo.ListYear(ctx)
	if err != nil {
		return FormOption{}, fmt.Errorf("masterxol/usecase: membaca daftar tahun: %w", err)
	}

	option := make([]TypeOption, 0, len(masterxol.KnownType()))
	for _, t := range masterxol.KnownType() {
		option = append(option, TypeOption{Code: t, Label: masterxol.TypeLabel(t)})
	}

	return FormOption{Year: year, Type: option}, nil
}

// BusinessGroup mengembalikan pilihan grup bisnis untuk sebuah Type XOL.
//
// Dipisahkan dari Form karena ia BERUBAH saat pengguna mengganti Type XOL — di layar lama
// pun dropdown-nya memanggil `ShowDetailGroupBisnisXol_Act` pada setiap perubahan.
// Menggabungkannya ke Form berarti mengambil ketiga daftar sekaligus padahal hanya satu
// yang dipakai.
func (s *Service) BusinessGroup(ctx context.Context, portalAlias string, t masterxol.Type) ([]masterxol.Business, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return nil, err
	}

	list, err := repo.ListBusinessGroup(ctx, t)
	if err != nil {
		return nil, fmt.Errorf("masterxol/usecase: membaca pilihan grup bisnis: %w", err)
	}
	return list, nil
}

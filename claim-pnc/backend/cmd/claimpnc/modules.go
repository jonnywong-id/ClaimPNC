// Perakitan sepuluh modul yang KODENYA sudah lengkap tetapi tidak pernah dirakit.
//
// # Kenapa berkas tersendiri
//
// Kesebelasnya datang dari cabang `fran-masuk-master` beserta usecase, penyimpanan SQL,
// penyimpanan memori, dan ujinya — hanya perakitannya yang tidak ikut. Menaruh perakitan
// itu di `main.go` akan menambah sekitar 500 baris pada berkas yang sudah 2.100 baris,
// sedangkan `docs/Steering/07` §2 aturan 3 menyebut berkas melebihi ~400 baris sebagai
// tanda perlu dipecah. Preseden memecahnya sudah ada: `registration.go`.
//
// Yang tinggal di `main.go` hanyalah dua field — `storage.extra` dan `assembly.extra` —
// beserta tiga titik panggil ke berkas ini.
//
// # Empat modul di antaranya menyentuh area yang pernah dikecualikan
//
// Master Bengkel, Master Sparepart, Master Panel, dan Master Supplier membaca tepat tabel
// yang `D-34` keluarkan dari lingkup migrasi — BENGKEL_HE, SPAREPART_HE, PANEL_HE,
// LOKASI_PANEL_HE, dan M_SUPPLIER. Modulnya terlanjur dibangun di cabang lain, dan Work
// Owner memutuskan pada 2026-09-22 bahwa SELURUH yang datang dari cabang itu harus ada.
//
// Keberatannya sudah disampaikan dan keputusannya ditegaskan. Ia dicatat di sini supaya
// pembaca berikutnya tahu bahwa keempat modul ini berada di luar `D-34` secara sadar,
// bukan karena kelalaian — dan supaya pencabutan `D-34` dapat ditulis sebagai keputusan
// tersendiri di Decision Log.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/portal"

	"claim-pnc/internal/inboxadmin"
	"claim-pnc/internal/inboxcompliance"
	"claim-pnc/internal/inboxmanageradmin"
	"claim-pnc/internal/masterautoclaim"
	"claim-pnc/internal/masterbengkel"
	"claim-pnc/internal/masterpanel"
	"claim-pnc/internal/masterpasal"
	"claim-pnc/internal/masterpenolakan"
	"claim-pnc/internal/mastersparepart"
	"claim-pnc/internal/mastersupplier"
	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/platform/db"
	"claim-pnc/internal/riwayatklaim"

	authhttp "claim-pnc/internal/auth/http"
	inboxadminhttp "claim-pnc/internal/inboxadmin/http"
	inboxadminmemory "claim-pnc/internal/inboxadmin/repo/memory"
	inboxadminsql "claim-pnc/internal/inboxadmin/repo/sqlstore"
	inboxadminusecase "claim-pnc/internal/inboxadmin/usecase"
	inboxcompliancehttp "claim-pnc/internal/inboxcompliance/http"
	inboxcompliancememory "claim-pnc/internal/inboxcompliance/repo/memory"
	inboxcompliancesql "claim-pnc/internal/inboxcompliance/repo/sqlstore"
	inboxcomplianceusecase "claim-pnc/internal/inboxcompliance/usecase"
	inboxmanageradminhttp "claim-pnc/internal/inboxmanageradmin/http"
	inboxmanageradminmemory "claim-pnc/internal/inboxmanageradmin/repo/memory"
	inboxmanageradminsql "claim-pnc/internal/inboxmanageradmin/repo/sqlstore"
	inboxmanageradminusecase "claim-pnc/internal/inboxmanageradmin/usecase"
	masterautoclaimhttp "claim-pnc/internal/masterautoclaim/http"
	masterautoclaimmemory "claim-pnc/internal/masterautoclaim/repo/memory"
	masterautoclaimsql "claim-pnc/internal/masterautoclaim/repo/sqlstore"
	masterautoclaimusecase "claim-pnc/internal/masterautoclaim/usecase"
	masterbengkelhttp "claim-pnc/internal/masterbengkel/http"
	masterbengkelmemory "claim-pnc/internal/masterbengkel/repo/memory"
	masterbengkelsql "claim-pnc/internal/masterbengkel/repo/sqlstore"
	masterbengkelusecase "claim-pnc/internal/masterbengkel/usecase"
	masterpanelhttp "claim-pnc/internal/masterpanel/http"
	masterpanelmemory "claim-pnc/internal/masterpanel/repo/memory"
	masterpanelsql "claim-pnc/internal/masterpanel/repo/sqlstore"
	masterpanelusecase "claim-pnc/internal/masterpanel/usecase"
	masterpasalhttp "claim-pnc/internal/masterpasal/http"
	masterpasalmemory "claim-pnc/internal/masterpasal/repo/memory"
	masterpasalsql "claim-pnc/internal/masterpasal/repo/sqlstore"
	masterpasalusecase "claim-pnc/internal/masterpasal/usecase"
	masterpenolakanhttp "claim-pnc/internal/masterpenolakan/http"
	masterpenolakanmemory "claim-pnc/internal/masterpenolakan/repo/memory"
	masterpenolakansql "claim-pnc/internal/masterpenolakan/repo/sqlstore"
	masterpenolakanusecase "claim-pnc/internal/masterpenolakan/usecase"
	masterspareparthttp "claim-pnc/internal/mastersparepart/http"
	mastersparepartmemory "claim-pnc/internal/mastersparepart/repo/memory"
	mastersparepartsql "claim-pnc/internal/mastersparepart/repo/sqlstore"
	mastersparepartusecase "claim-pnc/internal/mastersparepart/usecase"
	mastersupplierhttp "claim-pnc/internal/mastersupplier/http"
	mastersuppliermemory "claim-pnc/internal/mastersupplier/repo/memory"
	mastersuppliersql "claim-pnc/internal/mastersupplier/repo/sqlstore"
	mastersupplierusecase "claim-pnc/internal/mastersupplier/usecase"
	portalhttp "claim-pnc/internal/portal/http"
	riwayatklaimhttp "claim-pnc/internal/riwayatklaim/http"
	riwayatklaimmemory "claim-pnc/internal/riwayatklaim/repo/memory"
	riwayatklaimsql "claim-pnc/internal/riwayatklaim/repo/sqlstore"
	riwayatklaimusecase "claim-pnc/internal/riwayatklaim/usecase"
)

// extraSelectors memegang pemilih penyimpanan kesepuluh modul ini.
//
// Seluruhnya PER PORTAL, dengan satu pengecualian yang disengaja: `claimReport` adalah
// repo tunggal, karena modul Pelaporan Klaim memang menerima `Repo`, bukan pemilih.
// Membungkusnya menjadi pemilih di sini hanya akan memalsukan pemisahan entitas yang
// modulnya sendiri belum punya.
type extraSelectors struct {
	inboxAdmin        inboxadmin.RepoSelector
	inboxCompliance   inboxcompliance.RepoSelector
	inboxManagerAdmin inboxmanageradmin.RepoSelector

	// inboxManagerAdminLine membaca lini bisnis petugas, yang menentukan tab mana yang
	// boleh ia buka. Ia dipisah dari selector di atas karena membaca tabel yang BERBEDA
	// (`M_LOGIN_PNC`, milik modul Login) untuk pertanyaan yang berbeda: kewenangan, bukan
	// antrean.
	inboxManagerAdminLine inboxmanageradmin.LineBusinessRepoSelector

	autoClaim       masterautoclaim.RepoSelector
	workshop        masterbengkel.RepoSelector
	panel           masterpanel.RepoSelector
	clause          masterpasal.RepoSelector
	rejection       masterpenolakan.RepoSelector
	rejectionKomite masterpenolakan.RepoSelectorKomite
	sparepart       mastersparepart.RepoSelector
	supplier        mastersupplier.RepoSelector
	claimHistory    riwayatklaim.RepoSelector
	protection      riwayatklaim.ProtectionRepoSelector
}

// extraServices memegang layanan kesepuluh modul setelah terpasang.
type extraServices struct {
	inboxAdmin        *inboxadminusecase.Service
	inboxCompliance   *inboxcomplianceusecase.Service
	inboxManagerAdmin *inboxmanageradminusecase.Service

	autoClaim       *masterautoclaimusecase.Service
	workshop        *masterbengkelusecase.Service
	panel           *masterpanelusecase.Service
	clause          *masterpasalusecase.Service
	rejection       *masterpenolakanusecase.Service
	rejectionKomite *masterpenolakanusecase.ServiceKomite
	sparepart       *mastersparepartusecase.Service
	supplier        *mastersupplierusecase.Service
	claimHistory    *riwayatklaimusecase.Service
}

// setExtraOracleSelectors memasang pemilih di atas kolam koneksi entitas.
//
// Bentuknya sama persis dengan pemilih modul lain di `main.go`, dan jaminannya pun sama:
// portal yang tidak dikenal atau belum hidup menghasilkan galat dari `For()`, TIDAK PERNAH
// dialihkan ke koneksi utama sebagai cadangan (`R-20`).
func setExtraOracleSelectors(pool *db.Pool, store *storage) {
	store.extra.inboxAdmin = func(alias string) (inboxadmin.Repo, error) {
		conn, err := pool.For(alias)
		if err != nil {
			return nil, err
		}
		return inboxadminsql.NewRepo(conn), nil
	}
	store.extra.inboxCompliance = func(alias string) (inboxcompliance.Repo, error) {
		conn, err := pool.For(alias)
		if err != nil {
			return nil, err
		}
		return inboxcompliancesql.NewRepo(conn), nil
	}
	store.extra.inboxManagerAdmin = func(alias string) (inboxmanageradmin.Repo, error) {
		conn, err := pool.For(alias)
		if err != nil {
			return nil, err
		}
		return inboxmanageradminsql.NewRepo(conn), nil
	}
	// Lini bisnis dibaca dari koneksi portal yang SAMA dengan antreannya, dan itu bukan
	// kebetulan: `M_LOGIN_PNC` adalah tabel per entitas, sehingga membacanya dari portal
	// lain berarti menilai kewenangan dengan data badan hukum yang salah (`R-20`).
	store.extra.inboxManagerAdminLine = func(alias string) (inboxmanageradmin.LineBusinessRepo, error) {
		conn, err := pool.For(alias)
		if err != nil {
			return nil, err
		}
		return inboxmanageradminsql.NewRepo(conn), nil
	}
	store.extra.autoClaim = func(alias string) (masterautoclaim.Store, error) {
		conn, err := pool.For(alias)
		if err != nil {
			return nil, err
		}
		return masterautoclaimsql.NewRepo(conn), nil
	}
	store.extra.workshop = func(alias string) (masterbengkel.Store, error) {
		conn, err := pool.For(alias)
		if err != nil {
			return nil, err
		}
		return masterbengkelsql.NewRepo(conn), nil
	}
	store.extra.panel = func(alias string) (masterpanel.Store, error) {
		conn, err := pool.For(alias)
		if err != nil {
			return nil, err
		}
		return masterpanelsql.NewRepo(conn), nil
	}
	store.extra.clause = func(alias string) (masterpasal.Store, error) {
		conn, err := pool.For(alias)
		if err != nil {
			return nil, err
		}
		return masterpasalsql.NewRepo(conn), nil
	}
	store.extra.rejection = func(alias string) (masterpenolakan.Repo, error) {
		conn, err := pool.For(alias)
		if err != nil {
			return nil, err
		}
		return masterpenolakansql.NewRepo(conn), nil
	}
	store.extra.rejectionKomite = func(alias string) (masterpenolakan.RepoKomite, error) {
		conn, err := pool.For(alias)
		if err != nil {
			return nil, err
		}
		return masterpenolakansql.NewRepoKomite(conn), nil
	}
	store.extra.sparepart = func(alias string) (mastersparepart.Store, error) {
		conn, err := pool.For(alias)
		if err != nil {
			return nil, err
		}
		return mastersparepartsql.NewRepo(conn), nil
	}
	store.extra.supplier = func(alias string) (mastersupplier.Store, error) {
		conn, err := pool.For(alias)
		if err != nil {
			return nil, err
		}
		return mastersuppliersql.NewRepo(conn), nil
	}
	store.extra.claimHistory = func(alias string) (riwayatklaim.Repo, error) {
		conn, err := pool.For(alias)
		if err != nil {
			return nil, err
		}
		return riwayatklaimsql.NewRepo(conn), nil
	}
	store.extra.protection = func(alias string) (riwayatklaim.ProtectionRepo, error) {
		conn, err := pool.For(alias)
		if err != nil {
			return nil, err
		}
		return riwayatklaimsql.NewProtectionRepo(conn), nil
	}

}

// onlyPrimary menolak alias selain portal utama.
//
// Dipakai seluruh pemilih memori di berkas ini, dengan galat yang SAMA PERSIS seperti
// pemilih memori di `main.go`. Keseragaman itu disengaja: perilaku penolakannya ikut
// teruji saat pengembangan, bukan hanya nanti di produksi.
func onlyPrimary(primaryAlias, alias string) error {
	clean := strings.ToUpper(strings.TrimSpace(alias))
	if clean != strings.ToUpper(strings.TrimSpace(primaryAlias)) {
		return fmt.Errorf("%w: portal %q tidak tersedia tanpa basis data", portal.ErrNotReady, alias)
	}
	return nil
}

// setExtraMemorySelectors memasang pemilih di memori untuk menjalankan tanpa basis data.
//
// Hanya portal utama yang dilayani, dan alias lain DITOLAK — bukan diam-diam dialihkan.
// Menjalankan tanpa basis data tidak boleh mengubah aturan pemisahan entitas, karena
// justru di lingkungan itulah pelanggarannya paling mudah lolos (`R-20`).
func setExtraMemorySelectors(primaryAlias string, store *storage) {
	inboxAdminStore := inboxadminmemory.NewSampleStore()
	store.extra.inboxAdmin = func(alias string) (inboxadmin.Repo, error) {
		if err := onlyPrimary(primaryAlias, alias); err != nil {
			return nil, err
		}
		return inboxAdminStore, nil
	}

	inboxComplianceStore := inboxcompliancememory.NewSampleStore()
	store.extra.inboxCompliance = func(alias string) (inboxcompliance.Repo, error) {
		if err := onlyPrimary(primaryAlias, alias); err != nil {
			return nil, err
		}
		return inboxComplianceStore, nil
	}

	inboxManagerAdminStore := inboxmanageradminmemory.NewSampleStore()
	store.extra.inboxManagerAdmin = func(alias string) (inboxmanageradmin.Repo, error) {
		if err := onlyPrimary(primaryAlias, alias); err != nil {
			return nil, err
		}
		return inboxManagerAdminStore, nil
	}
	store.extra.inboxManagerAdminLine = func(alias string) (inboxmanageradmin.LineBusinessRepo, error) {
		if err := onlyPrimary(primaryAlias, alias); err != nil {
			return nil, err
		}
		return inboxManagerAdminStore, nil
	}

	autoClaimRepo := masterautoclaimmemory.NewSampleRepo()
	store.extra.autoClaim = func(alias string) (masterautoclaim.Store, error) {
		if err := onlyPrimary(primaryAlias, alias); err != nil {
			return nil, err
		}
		return autoClaimRepo, nil
	}

	workshopRepo := masterbengkelmemory.NewSampleRepo()
	store.extra.workshop = func(alias string) (masterbengkel.Store, error) {
		if err := onlyPrimary(primaryAlias, alias); err != nil {
			return nil, err
		}
		return workshopRepo, nil
	}

	panelRepo := masterpanelmemory.NewSampleRepo()
	store.extra.panel = func(alias string) (masterpanel.Store, error) {
		if err := onlyPrimary(primaryAlias, alias); err != nil {
			return nil, err
		}
		return panelRepo, nil
	}

	clauseRepo := masterpasalmemory.NewSampleRepo()
	store.extra.clause = func(alias string) (masterpasal.Store, error) {
		if err := onlyPrimary(primaryAlias, alias); err != nil {
			return nil, err
		}
		return clauseRepo, nil
	}

	rejectionRepo := masterpenolakanmemory.NewRepo(
		masterpenolakanmemory.SampleParents(),
		masterpenolakanmemory.SampleList()...,
	)
	store.extra.rejection = func(alias string) (masterpenolakan.Repo, error) {
		if err := onlyPrimary(primaryAlias, alias); err != nil {
			return nil, err
		}
		return rejectionRepo, nil
	}

	rejectionKomiteRepo := masterpenolakanmemory.NewRepoKomite(masterpenolakanmemory.SampleListKomite()...)
	store.extra.rejectionKomite = func(alias string) (masterpenolakan.RepoKomite, error) {
		if err := onlyPrimary(primaryAlias, alias); err != nil {
			return nil, err
		}
		return rejectionKomiteRepo, nil
	}

	sparepartRepo := mastersparepartmemory.NewSampleRepo()
	store.extra.sparepart = func(alias string) (mastersparepart.Store, error) {
		if err := onlyPrimary(primaryAlias, alias); err != nil {
			return nil, err
		}
		return sparepartRepo, nil
	}

	supplierRepo := mastersuppliermemory.NewSampleRepo()
	store.extra.supplier = func(alias string) (mastersupplier.Store, error) {
		if err := onlyPrimary(primaryAlias, alias); err != nil {
			return nil, err
		}
		return supplierRepo, nil
	}

	claimHistoryRepo := riwayatklaimmemory.NewRepo(riwayatklaimmemory.SampleClaims()...)
	store.extra.claimHistory = func(alias string) (riwayatklaim.Repo, error) {
		if err := onlyPrimary(primaryAlias, alias); err != nil {
			return nil, err
		}
		return claimHistoryRepo, nil
	}

	protectionRepo := riwayatklaimmemory.NewProtectionRepo(riwayatklaimmemory.SampleProtections()...)
	store.extra.protection = func(alias string) (riwayatklaim.ProtectionRepo, error) {
		if err := onlyPrimary(primaryAlias, alias); err != nil {
			return nil, err
		}
		return protectionRepo, nil
	}

}

// buildExtraServices menyusun layanan kesepuluh modul di balik seam-nya.
func buildExtraServices(store storage, logger *slog.Logger) (extraServices, error) {
	var (
		result extraServices
		err    error
	)

	if result.inboxAdmin, err = inboxadminusecase.NewService(inboxadminusecase.Options{
		RepoSelector: store.extra.inboxAdmin,
		Clock:        clock.System{},
		Logger:       logger,
	}); err != nil {
		return extraServices{}, err
	}

	if result.inboxCompliance, err = inboxcomplianceusecase.NewService(inboxcomplianceusecase.Options{
		RepoSelector: store.extra.inboxCompliance,
		Clock:        clock.System{},
		Logger:       logger,
	}); err != nil {
		return extraServices{}, err
	}

	if result.inboxManagerAdmin, err = inboxmanageradminusecase.NewService(
		inboxmanageradminusecase.Options{
			RepoSelector:         store.extra.inboxManagerAdmin,
			LineBusinessSelector: store.extra.inboxManagerAdminLine,
			Clock:                clock.System{},

			// Logger WAJIB terisi di modul ini, berbeda dari modul yang memakainya hanya
			// untuk peringatan. Ia yang mencatat SETIAP pembukaan antrean, dan catatan itu
			// satu-satunya kontrol pengimbang selama pemeriksaan peran belum ada
			// (`D-59`, `TKT-F3-004`). Lihat inboxmanageradmin/usecase.List.
			Logger: logger,
		}); err != nil {
		return extraServices{}, err
	}

	if result.autoClaim, err = masterautoclaimusecase.NewService(masterautoclaimusecase.Options{
		RepoSelector: store.extra.autoClaim,
	}); err != nil {
		return extraServices{}, err
	}

	if result.workshop, err = masterbengkelusecase.NewService(masterbengkelusecase.Options{
		RepoSelector: store.extra.workshop,
	}); err != nil {
		return extraServices{}, err
	}

	if result.panel, err = masterpanelusecase.NewService(masterpanelusecase.Options{
		RepoSelector: store.extra.panel,
	}); err != nil {
		return extraServices{}, err
	}

	if result.clause, err = masterpasalusecase.NewService(masterpasalusecase.Options{
		RepoSelector: store.extra.clause,
	}); err != nil {
		return extraServices{}, err
	}

	if result.rejection, err = masterpenolakanusecase.NewService(masterpenolakanusecase.Options{
		RepoSelector: store.extra.rejection,
		Clock:        clock.System{},
	}); err != nil {
		return extraServices{}, err
	}

	// Tanpa Clock, berbeda dari layanan induknya: penolakan komite tidak menuliskan
	// stempel waktu apa pun.
	if result.rejectionKomite, err = masterpenolakanusecase.NewServiceKomite(masterpenolakanusecase.OptionsKomite{
		RepoSelector: store.extra.rejectionKomite,
	}); err != nil {
		return extraServices{}, err
	}

	if result.sparepart, err = mastersparepartusecase.NewService(mastersparepartusecase.Options{
		RepoSelector: store.extra.sparepart,
		Clock:        clock.System{},
	}); err != nil {
		return extraServices{}, err
	}

	if result.supplier, err = mastersupplierusecase.NewService(mastersupplierusecase.Options{
		RepoSelector: store.extra.supplier,
		Clock:        clock.System{},
	}); err != nil {
		return extraServices{}, err
	}

	if result.claimHistory, err = riwayatklaimusecase.NewService(riwayatklaimusecase.Options{
		RepoSelector:       store.extra.claimHistory,
		ProtectionSelector: store.extra.protection,
		Clock:              clock.System{},
	}); err != nil {
		return extraServices{}, err
	}

	// Registrasi Klaim TIDAK dirakit di sini. Perakitan dan rutenya ada di main.go, yang
	// memasangnya di belakang penjaga portal utama (R-20) dan memakai login sebagai
	// identitas tugas. Salinan di berkas ini sempat membuat /registrasi terpasang dua kali,
	// dan chi menolak berjalan (2026-09-27).

	return result, nil
}

// mountExtra memasang rute kesepuluh modul di dalam kelompok yang sudah dijaga sesi.
//
// Pemeriksaan portal dipasang modulnya sendiri di dalam `Mount`, sama seperti modul master
// lain — kecuali Pelaporan Klaim dan Registrasi, yang `Mount`-nya memang tidak menerima
// `ActivePortalDeps`.
func mountExtra(
	protected chi.Router,
	service extraServices,
	portalDeps portalhttp.ActivePortalDeps,
	writeJSON func(http.ResponseWriter, *http.Request, int, any),
	writeError func(http.ResponseWriter, *http.Request, error),
	logger *slog.Logger,
) error {
	// Jembatan satu arah dari modul auth. Bentuknya diulang per modul karena tipe Caller
	// setiap modul BERBEDA — masing-masing mendeklarasikannya sendiri, dan itulah yang
	// membuat modul-modul ini tidak saling mengimpor.
	inboxAdminHandler := inboxadminhttp.NewHandler(inboxadminhttp.Options{
		Service: service.inboxAdmin,
		GetCaller: func(ctx context.Context) (inboxadminhttp.Caller, bool) {
			base, ok := authhttp.CallerFromContext(ctx)
			if !ok {
				return inboxadminhttp.Caller{}, false
			}
			return inboxadminhttp.Caller{Login: base.User.Login}, true
		},
		Logger:              logger,
		WriteJSON:           writeJSON,
		FallbackErrorWriter: writeError,
	})
	inboxadminhttp.Mount(protected, inboxAdminHandler, portalDeps)

	// Jembatan Caller di modul ini hanya dipakai jalur TULIS. Kedua jalur bacanya tidak
	// membutuhkannya — antreannya workbasket, yang isinya sama bagi setiap petugas —
	// sedangkan pengiriman ke Post Audit memerlukannya untuk MENCATAT siapa yang mengirim.
	inboxComplianceHandler := inboxcompliancehttp.NewHandler(inboxcompliancehttp.Options{
		Service: service.inboxCompliance,
		GetCaller: func(ctx context.Context) (inboxcompliancehttp.Caller, bool) {
			base, ok := authhttp.CallerFromContext(ctx)
			if !ok {
				return inboxcompliancehttp.Caller{}, false
			}
			return inboxcompliancehttp.Caller{Login: base.User.Login}, true
		},
		Logger:              logger,
		WriteJSON:           inboxcompliancehttp.JSONWriter(writeJSON),
		FallbackErrorWriter: inboxcompliancehttp.ErrorWriter(writeError),
	})
	inboxcompliancehttp.Mount(protected, inboxComplianceHandler, portalDeps)

	// Jembatan Caller modul ini membawa TIGA isian, bukan satu seperti modul inbox lain.
	//
	// Login dipakai mencatat siapa yang membuka antrean, DAN untuk mencari lini bisnisnya.
	// Unit organisasi menentukan pintu pengembang, mengikuti `pyContainerVisibleWhen` layar
	// Pega apa adanya — keputusan Work Owner 2026-09-26. Padanannya di sistem baru:
	//
	//	OperatorID.pyPosition  ->  M_LOGIN_PNC.LINE_BUSINESS  (dibaca usecase, per portal)
	//	OperatorID.pyOrgUnit   ->  variabel lingkungan UNIT_ORGANISASI_PENGGUNA
	//
	// # Yang pertama SENGAJA tidak diisi di sini, dan itu koreksi 2026-09-27
	//
	// Sampai tanggal itu baris ini menyalin `base.User.Position` — jabatan kepegawaian HCQ
	// seperti "IT SPECIALIST". Nilai itu tidak pernah cocok dengan `NONMBU`/`PA`/`TRAVEL`,
	// sehingga TIDAK SEORANG PUN melihat satu tab pun, dan layarnya tampak sengaja kosong.
	//
	// Keadaan itu sempat dicatat sebagai konsekuensi yang diterima. Pembacaan ulang export
	// membantahnya: di Pega `pyPosition` terisi KODE LINI BISNIS oleh administrator, dan
	// karena terisi, kontainernya tampil. Sekarang nilainya dibaca usecase dari
	// `M_LOGIN_PNC` milik portal yang aktif — bukan dari sesi, karena tabel itu per entitas.
	//
	// Unit organisasi tetap dari variabel lingkungan dan tetap BUKAN otorisasi: ia tidak
	// menjaga apa pun, hanya membuka ketiga tab bagi pengembang.
	inboxManagerAdminHandler := inboxmanageradminhttp.NewHandler(inboxmanageradminhttp.Options{
		Service: service.inboxManagerAdmin,
		GetCaller: func(ctx context.Context) (inboxmanageradminhttp.Caller, bool) {
			base, ok := authhttp.CallerFromContext(ctx)
			if !ok {
				return inboxmanageradminhttp.Caller{}, false
			}
			return inboxmanageradminhttp.Caller{
				Login:   base.User.Login,
				OrgUnit: userOrgUnit(),
			}, true
		},
		Logger:              logger,
		WriteJSON:           writeJSON,
		FallbackErrorWriter: writeError,
	})
	inboxmanageradminhttp.Mount(protected, inboxManagerAdminHandler, portalDeps)

	autoClaimHandler, err := masterautoclaimhttp.NewHandler(masterautoclaimhttp.Options{
		Service: service.autoClaim,
		Caller: func(ctx context.Context) (masterautoclaimhttp.Caller, bool) {
			base, ok := authhttp.CallerFromContext(ctx)
			if !ok {
				return masterautoclaimhttp.Caller{}, false
			}
			return masterautoclaimhttp.Caller{Login: base.User.Login}, true
		},
		Logger:        logger,
		WriteResponse: writeJSON,
		WriteError:    writeError,
	})
	if err != nil {
		return err
	}
	masterautoclaimhttp.Mount(protected, autoClaimHandler, portalDeps)

	workshopHandler, err := masterbengkelhttp.NewHandler(masterbengkelhttp.Options{
		Service: service.workshop,
		Caller: func(ctx context.Context) (masterbengkelhttp.Caller, bool) {
			base, ok := authhttp.CallerFromContext(ctx)
			if !ok {
				return masterbengkelhttp.Caller{}, false
			}
			return masterbengkelhttp.Caller{Login: base.User.Login}, true
		},
		Logger:        logger,
		WriteResponse: writeJSON,
		WriteError:    writeError,
	})
	if err != nil {
		return err
	}
	masterbengkelhttp.Mount(protected, workshopHandler, portalDeps)

	panelHandler, err := masterpanelhttp.NewHandler(masterpanelhttp.Options{
		Service: service.panel,
		Caller: func(ctx context.Context) (masterpanelhttp.Caller, bool) {
			base, ok := authhttp.CallerFromContext(ctx)
			if !ok {
				return masterpanelhttp.Caller{}, false
			}
			return masterpanelhttp.Caller{Login: base.User.Login}, true
		},
		Logger:        logger,
		WriteResponse: writeJSON,
		WriteError:    writeError,
	})
	if err != nil {
		return err
	}
	masterpanelhttp.Mount(protected, panelHandler, portalDeps)

	clauseHandler, err := masterpasalhttp.NewHandler(masterpasalhttp.Options{
		Service:       service.clause,
		Logger:        logger,
		WriteResponse: writeJSON,
		WriteError:    writeError,
	})
	if err != nil {
		return err
	}
	masterpasalhttp.Mount(protected, clauseHandler, portalDeps)

	rejectionHandler, err := masterpenolakanhttp.NewHandler(masterpenolakanhttp.Options{
		Service: service.rejection,
		Komite:  service.rejectionKomite,
		Caller: func(ctx context.Context) (masterpenolakanhttp.Caller, bool) {
			base, ok := authhttp.CallerFromContext(ctx)
			if !ok {
				return masterpenolakanhttp.Caller{}, false
			}
			return masterpenolakanhttp.Caller{Login: base.User.Login}, true
		},
		Logger:        logger,
		WriteResponse: writeJSON,
		WriteError:    writeError,
	})
	if err != nil {
		return err
	}
	masterpenolakanhttp.Mount(protected, rejectionHandler, portalDeps)

	sparepartHandler, err := masterspareparthttp.NewHandler(masterspareparthttp.Options{
		Service: service.sparepart,
		Caller: func(ctx context.Context) (masterspareparthttp.Caller, bool) {
			base, ok := authhttp.CallerFromContext(ctx)
			if !ok {
				return masterspareparthttp.Caller{}, false
			}
			return masterspareparthttp.Caller{Login: base.User.Login}, true
		},
		Logger:        logger,
		WriteResponse: writeJSON,
		WriteError:    writeError,
	})
	if err != nil {
		return err
	}
	masterspareparthttp.Mount(protected, sparepartHandler, portalDeps)

	supplierHandler, err := mastersupplierhttp.NewHandler(mastersupplierhttp.Options{
		Service: service.supplier,
		Caller: func(ctx context.Context) (mastersupplierhttp.Caller, bool) {
			base, ok := authhttp.CallerFromContext(ctx)
			if !ok {
				return mastersupplierhttp.Caller{}, false
			}
			return mastersupplierhttp.Caller{Login: base.User.Login}, true
		},
		Logger:        logger,
		WriteResponse: writeJSON,
		WriteError:    writeError,
	})
	if err != nil {
		return err
	}
	mastersupplierhttp.Mount(protected, supplierHandler, portalDeps)

	claimHistoryHandler := riwayatklaimhttp.NewHandler(riwayatklaimhttp.Options{
		Service: service.claimHistory,
		GetCaller: func(ctx context.Context) (riwayatklaimhttp.Caller, bool) {
			base, ok := authhttp.CallerFromContext(ctx)
			if !ok {
				return riwayatklaimhttp.Caller{}, false
			}
			return riwayatklaimhttp.Caller{Login: base.User.Login}, true
		},
		Logger:              logger,
		WriteJSON:           writeJSON,
		FallbackErrorWriter: writeError,
	})
	riwayatklaimhttp.Mount(protected, claimHistoryHandler, portalDeps)

	return nil
}

// userOrgUnit membaca unit organisasi pemanggil dari lingkungan.
//
// # Ini sakelar sementara, dan alasannya perlu dibaca sebelum dipakai
//
// Modul Inbox Manager Admin memisahkan ketiga tabnya menurut dua isian identitas yang di
// sistem lama datang dari `Data-Admin-Operator-ID`: `pyPosition` dan `pyOrgUnit`. Yang
// pertama punya padanan — `auth.User.Position`, diisi HCQ dari
// `EmpResponse.Placement.PositionName`. Yang kedua **tidak punya padanan sama sekali**:
// tidak ada isian unit organisasi di profil HCQ maupun di catatan pengguna lokal.
//
// Di layar Pega, satu nilai saja yang berarti — `Development`, yang membuka ketiga tab
// sekaligus. Tanpa penggantinya, tidak ada seorang pun yang dapat melihat lebih dari satu
// tab, termasuk saat modulnya diuji.
//
// Karena itu: satu variabel lingkungan, dibaca setiap permintaan, dipakai untuk SELURUH
// pengguna. Bentuknya mengikuti preseden `PERAN_PENGGUNA` di registration.go, dan
// peringatannya pun sama — **ia bukan otorisasi**. Ia tidak menjaga apa pun, dan tidak boleh
// dikira menjaga sesuatu: mengisinya `Development` di produksi berarti setiap pengguna yang
// dapat masuk melihat ketiga antrean.
//
// Begitu `TKT-F3-004` selesai dan unit organisasi datang dari catatan pengguna, fungsi ini
// dihapus.
//
// Nilainya dibaca setiap permintaan, bukan sekali saat start, supaya pengembang dapat
// mengubahnya tanpa menjalankan ulang aplikasi. Biayanya satu pembacaan variabel lingkungan
// per permintaan, dan itu tidak terukur dibanding satu perjalanan ke basis data.
func userOrgUnit() string {
	return strings.TrimSpace(os.Getenv("UNIT_ORGANISASI_PENGGUNA"))
}

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
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/portal"

	"claim-pnc/internal/inboxadmin"
	"claim-pnc/internal/inboxcompliance"
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
	registrasihttp "claim-pnc/internal/registrasi/http"
	registrasiusecase "claim-pnc/internal/registrasi/usecase"
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
	inboxAdmin      inboxadmin.RepoSelector
	inboxCompliance inboxcompliance.RepoSelector
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
	inboxAdmin      *inboxadminusecase.Service
	inboxCompliance *inboxcomplianceusecase.Service
	autoClaim       *masterautoclaimusecase.Service
	workshop        *masterbengkelusecase.Service
	panel           *masterpanelusecase.Service
	clause          *masterpasalusecase.Service
	rejection       *masterpenolakanusecase.Service
	rejectionKomite *masterpenolakanusecase.ServiceKomite
	sparepart       *mastersparepartusecase.Service
	supplier        *mastersupplierusecase.Service
	claimHistory    *riwayatklaimusecase.Service
	registration    *registrasiusecase.Service
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

	// Registrasi dirakit fungsi tersendiri di registration.go, yang sudah ada sejak cabang
	// asalnya. Basis datanya portal UTAMA, dan nil saat berjalan tanpa Oracle — fungsi itu
	// menanganinya sendiri dengan beralih ke penyimpanan memori.
	var primary *sql.DB
	if store.legacy != nil {
		primary = store.legacy.DB()
	}
	if result.registration, err = assembleRegistration(primary, logger); err != nil {
		return extraServices{}, err
	}

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

	// Registrasi membaca identitas dari PERMINTAAN, bukan dari konteks — bentuk Caller-nya
	// memang berbeda dari modul lain. Peran dan antrean datang dari sakelar sementara di
	// registration.go, yang menjelaskan sendiri kenapa ia bukan otorisasi.
	registrationHandler := registrasihttp.NewHandler(registrasihttp.Options{
		Service: service.registration,
		Logger:  logger,
		Caller: func(r *http.Request) (registrasiusecase.Caller, bool) {
			base, ok := authhttp.CallerFromContext(r.Context())
			if !ok {
				return registrasiusecase.Caller{}, false
			}
			return registrasiusecase.Caller{
				Identity:   base.User.Identity,
				Name:       base.User.Name,
				Roles:      userRoles(),
				Workbasket: userWorkbaskets(),
			}, true
		},
		WriteResponse: writeJSON,
	})
	registrasihttp.Mount(protected, registrationHandler)

	return nil
}

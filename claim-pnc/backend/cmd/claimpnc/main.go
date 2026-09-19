// Command claimpnc adalah satu-satunya titik masuk aplikasi Claim PNC.
//
// Aplikasi ini modular monolith (ADR-0001): satu binary yang memuat seluruh modul,
// dikompilasi tanpa dependensi runtime eksternal dan menyajikan API sekaligus berkas
// statis antarmuka.
//
// Berkas ini sengaja tipis. Tugasnya hanya tiga: membaca konfigurasi, merakit adapter
// di balik setiap seam, dan menyalakan server. **Tidak ada aturan bisnis di sini.**
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/auth"
	"claim-pnc/internal/auth/provider"
	"claim-pnc/internal/auth/repo/memory"
	"claim-pnc/internal/auth/repo/sqlstore"
	"claim-pnc/internal/auth/usecase"
	"claim-pnc/internal/masterrekening"
	"claim-pnc/internal/masterstatus"
	"claim-pnc/internal/masterstatusprogres"
	"claim-pnc/internal/menu"
	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/platform/config"
	"claim-pnc/internal/platform/db"
	"claim-pnc/internal/platform/httpserver"
	"claim-pnc/internal/platform/logging"
	"claim-pnc/internal/portal"
	"claim-pnc/spa"

	authhttp "claim-pnc/internal/auth/http"
	masterrekeningcashier "claim-pnc/internal/masterrekening/cashier"
	masterrekeninghttp "claim-pnc/internal/masterrekening/http"
	masterrekeningnotif "claim-pnc/internal/masterrekening/notification"
	masterrekeningmemory "claim-pnc/internal/masterrekening/repo/memory"
	masterrekeningsql "claim-pnc/internal/masterrekening/repo/sqlstore"
	masterrekeningusecase "claim-pnc/internal/masterrekening/usecase"
	masterstatushttp "claim-pnc/internal/masterstatus/http"
	masterstatusmemory "claim-pnc/internal/masterstatus/repo/memory"
	masterstatussql "claim-pnc/internal/masterstatus/repo/sqlstore"
	masterstatususecase "claim-pnc/internal/masterstatus/usecase"
	masterstatusprogreshttp "claim-pnc/internal/masterstatusprogres/http"
	masterstatusprogresmemory "claim-pnc/internal/masterstatusprogres/repo/memory"
	masterstatusprogressql "claim-pnc/internal/masterstatusprogres/repo/sqlstore"
	masterstatusprogresusecase "claim-pnc/internal/masterstatusprogres/usecase"
	menuhttp "claim-pnc/internal/menu/http"
	menumemory "claim-pnc/internal/menu/repo/memory"
	menusql "claim-pnc/internal/menu/repo/sqlstore"
	menuusecase "claim-pnc/internal/menu/usecase"
	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
	portalsql "claim-pnc/internal/portal/repo/sqlstore"
)

// defaultEnvFile dibaca bila ada. Nilai yang sudah ada di lingkungan proses menang atas
// isinya, sehingga satu perintah dapat menimpa satu nilai tanpa menyunting berkas.
const defaultEnvFile = ".env"

func main() {
	if err := run(); err != nil {
		// Kegagalan saat start ditulis ke stderr dan menghentikan proses. Aplikasi
		// yang setengah hidup lebih berbahaya daripada aplikasi yang tidak start.
		fmt.Fprintln(os.Stderr, "gagal menjalankan aplikasi:", err)
		os.Exit(1)
	}
}

func run() error {
	// Dua flag, keduanya untuk mode periksa. Aplikasi normal tidak memakai flag sama
	// sekali — seluruh konfigurasinya dari .env atau lingkungan (ADR-0025).
	checkMode := flag.Bool("periksa", false,
		"periksa integrasi basis data dan HCC/HCQ lalu berhenti; tidak menulis apa pun")
	testLogin := flag.String("login", "",
		"nama pengguna yang dicoba pada mode periksa; kata sandinya dibaca dari stdin")
	flag.Parse()

	if err := config.LoadEnvFile(defaultEnvFile); err != nil {
		return err
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if *checkMode {
		return check(cfg, *testLogin, os.Stdin, os.Stdout)
	}

	logger := logging.New(slog.LevelInfo)
	logger.Info("konfigurasi terbaca", slog.Any("konfigurasi", cfg.Summary()))

	assembly, err := build(cfg, logger)
	if err != nil {
		return err
	}
	defer assembly.close()

	spaFiles, err := spa.Files()
	if err != nil {
		logger.Warn("antarmuka tidak tersedia; aplikasi hanya melayani API",
			slog.String("sebab", err.Error()))
		spaFiles = nil
	}

	// Satu penulis JSON dan satu penulis galat dipakai bersama seluruh modul, supaya
	// bentuk respons dan header Cache-Control-nya tidak berbeda antarmodul.
	writeJSON := func(w http.ResponseWriter, r *http.Request, status int, body any) {
		authhttp.WriteJSON(w, r, status, body, logger)
	}
	writeAuthError := authhttp.WriteError(logger)

	handlerAuth := authhttp.NewHandler(assembly.auth, logger)
	handlerMasterStatus := masterstatushttp.NewHandler(masterstatushttp.Options{
		Service: assembly.masterStatus,
		Logger:  logger,
		// Galat yang bukan milik modul master diteruskan ke penulis galat auth,
		// sehingga galat sesi tetap dijawab dengan kode yang sudah dikenal frontend.
		WriteResponse:       writeJSON,
		FallbackErrorWriter: masterstatushttp.ErrorWriter(writeAuthError),
	})
	handlerPortal := portalhttp.NewHandler(portalhttp.Options{
		Repo:         assembly.portal,
		ReadyAliases: assembly.readyAliases,
		PrimaryAlias: cfg.PrimaryPortal,
		Logger:       logger,
		WriteResponse: func(w http.ResponseWriter, r *http.Request, status int, body any) {
			authhttp.WriteJSON(w, r, status, body, logger)
		},
		WriteError: authhttp.WriteError(logger),
	})

	// Galat portal dipetakan modul portal, sisanya diteruskan ke pemeta modul auth.
	// Urutan pembungkusnya menentukan: yang lebih khusus memeriksa lebih dulu. Satu
	// rantai untuk setiap modul bisnis, bukan satu tafsiran per modul.
	writePortalAwareError := portalhttp.WithPortalError(writeAuthError, writeJSON)

	progressStatusHandler, err := masterstatusprogreshttp.NewHandler(masterstatusprogreshttp.Options{
		Service:       assembly.masterStatusProgres,
		Logger:        logger,
		WriteResponse: writeJSON,
		WriteError:    masterstatusprogreshttp.ErrorWriter(writePortalAwareError),
	})
	if err != nil {
		return err
	}

	// Menu dirakit dari POOLDATA.M_MENU_APLIKASI_PNC dan M_OTORISASI_PNC. Jembatan
	// konteks pemanggilnya SATU ARAH dari modul auth, dipasang di sini supaya kedua
	// modul tetap tidak saling mengimpor.
	menuHandler, err := menuhttp.NewHandler(menuhttp.Options{
		Service: assembly.menu,
		Caller: func(ctx context.Context) (menuhttp.Caller, bool) {
			baseCtx, existing := authhttp.CallerFromContext(ctx)
			if !existing {
				return menuhttp.Caller{}, false
			}
			// Login yang DIKETIK pengguna, bukan NIK: itulah yang dicocokkan ke
			// M_LOGIN_GROUP_PNC.LOGIN_ID dan M_OTORISASI_PNC.LOGIN_ID_GROUP.
			return menuhttp.Caller{Login: baseCtx.User.Login}, true
		},
		Logger:        logger,
		WriteResponse: writeJSON,
		WriteError:    menuhttp.ErrorWriter(writeAuthError),
	})
	if err != nil {
		return err
	}

	// Bahan penentu portal aktif dipakai setiap modul bisnis yang menyentuh basis data
	// entitas. Ia dirakit sekali di sini supaya modul-modul berikutnya memakai
	// pemeriksaan yang sama persis, bukan masing-masing menafsirkannya sendiri.
	activePortalDeps := portalhttp.ActivePortalDeps{
		Repo:         assembly.portal,
		ReadyAliases: assembly.readyAliases,
		Logger:       logger,
		WriteError:   writePortalAwareError,
	}

	accountHandler := masterrekeninghttp.NewHandler(masterrekeninghttp.Options{
		Service: assembly.masterRekening,
		// Jembatan satu arah dari modul auth ke modul master rekening. Ia dipasang di
		// sini, bukan di dalam salah satu modul, supaya kedua modul tetap tidak saling
		// mengimpor — yang tahu keduanya hanyalah berkas perakitan ini.
		Caller: func(ctx context.Context) (masterrekeninghttp.Caller, bool) {
			baseCtx, existing := authhttp.CallerFromContext(ctx)
			if !existing {
				return masterrekeninghttp.Caller{}, false
			}
			return masterrekeninghttp.Caller{
				Identity: baseCtx.User.Identity,
				Name:     baseCtx.User.Name,
				Email:    baseCtx.User.Email,
			}, true
		},
		Logger: logger,
	})

	router := httpserver.Router(httpserver.Deps{
		Logger:   logger,
		SPAFiles: spaFiles,
		MountAPI: func(api chi.Router) {
			// Setiap modul memasang rutenya sendiri di sini. Modul berikutnya cukup
			// menambah satu baris; server tidak perlu tahu isinya.
			authhttp.Mount(api, handlerAuth, assembly.auth, logger)

			// List portal berada di balik sesi: pemilihnya ada di dalam aplikasi,
			// bukan di layar masuk (ADR-0030, berpindah portal tanpa login ulang).
			api.Group(func(protected chi.Router) {
				protected.Use(authhttp.Authenticate(assembly.auth, authhttp.WriteError(logger)))
				portalhttp.Mount(protected, handlerPortal)

				// Menu berada di balik sesi tetapi TIDAK di balik pemeriksaan portal:
				// peta menu dan kewenangannya hidup di basis data portal utama dan tidak
				// punya kolom entitas. Menuntut portal di sini akan membuat menunya gagal
				// justru saat pengguna belum memilih entitas.
				menuhttp.Mount(protected, menuHandler)

				// Master Status Progres 1. Rutenya memasang pemeriksaan portal sendiri
				// di dalam Mount — hanya pada rute yang benar-benar menyentuh basis
				// data entitas.
				masterstatusprogreshttp.Mount(protected, progressStatusHandler, activePortalDeps)
				// Master rekening memuat nama, NIK, nomor rekening, dan surel pihak
				// ketiga; tidak satu pun boleh terbaca tanpa sesi.
				masterrekeninghttp.Mount(protected, accountHandler)
				// Master data juga berada di balik sesi. Pemeriksaan peran — "apakah
				// pemanggil memiliki menu Master Data" (D-59) — belum ada di sini
				// karena TKT-F3-004 dan TKT-F3-005 belum dikerjakan; keadaannya sama
				// dengan seluruh rute lain hari ini.
				masterstatushttp.Mount(protected, handlerMasterStatus)
			})
		},
	})

	server := &http.Server{
		Addr:              cfg.Address,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}
	logger.Info("server menyala", slog.String("alamat", cfg.Address))
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("server berhenti: %w", err)
	}
	return nil
}

// assembly memegang seluruh modul yang sudah terpasang beserta cara menutupnya.
type assembly struct {
	auth           *usecase.Service
	portal         portal.Repo
	masterRekening *masterrekeningusecase.Service
	masterStatus   *masterstatususecase.Service

	// masterStatusProgres memakai pemilih repo per portal, bukan repo tunggal:
	// tabelnya ada di basis data SETIAP entitas (ADR-0030).
	masterStatusProgres *masterstatusprogresusecase.Service

	// menu menyusun peta menu beserta kewenangan pemakainya.
	menu *menuusecase.Service

	readyAliases func() []string
	close        func()
}

// storage memegang seluruh repo yang sudah terpasang di atas sumbernya.
type storage struct {
	user         auth.UserRepo
	session      auth.SessionRepo
	portal       portal.Repo
	masterStatus masterstatus.Repo

	account     masterrekening.Repo
	accountBank masterrekening.BankRepo

	// rekeningDiOracle menyatakan master rekening dipasang di atas POOLDATA.LST_ACCOUNT
	// yang sungguhan, bukan di memori. Adapter tiruan yang menulis jejak karangan
	// dilarang di atasnya — lihat rakitMasterRekening.
	accountInOracle bool

	// warisan bernilai nil bila koneksi Oracle tidak dibuka. Ia memberi akses baca ke
	// tiga tabel milik sistem lama: M_PORTAL_PNC, M_LOGIN_PNC, dan GCNM_CONNECT_REST.
	legacy *sqlstore.Legacy

	// progressStatusSelector memilih penyimpanan master status progres milik satu portal.
	//
	// Ia fungsi, bukan repo tunggal, karena tabelnya ada di basis data SETIAP entitas
	// (ADR-0030). Satu repo bersama akan menulis data seluruh entitas ke satu tempat,
	// kebocoran lintas badan hukum yang justru dicegah R-20.
	progressStatusSelector masterstatusprogres.RepoSelector

	// menu dibaca dari basis data portal UTAMA, sama seperti M_LOGIN_PNC dan
	// M_PORTAL_PNC: peta menu dan kewenangan pemakainya adalah data lingkup
	// identitas, bukan data bisnis milik satu badan hukum.
	menu menu.Repo

	readyAliases func() []string
	close        func()
}

// build menyusun seluruh modul di balik seam-nya masing-masing.
func build(cfg config.Config, logger *slog.Logger) (assembly, error) {
	production := cfg.Environment == config.Production

	store, err := buildStorage(cfg, production, logger)
	if err != nil {
		return assembly{}, err
	}

	identitySystem, err := buildIdentity(cfg, production, store.legacy)
	if err != nil {
		store.close()
		return assembly{}, err
	}

	service, err := usecase.NewService(usecase.Options{
		Identity:        identitySystem,
		UserRepo:        store.user,
		SessionRepo:     store.session,
		Clock:           clock.System{},
		SessionLifetime: cfg.Session.Lifetime,
	})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	progressStatusService, err := masterstatusprogresusecase.NewService(masterstatusprogresusecase.Options{
		RepoSelector: store.progressStatusSelector,
	})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	menuService, err := menuusecase.NewService(menuusecase.Options{Repo: store.menu})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	claimStatusService, err := masterstatususecase.NewService(masterstatususecase.Options{
		Repo: store.masterStatus,
	})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	return assembly{
		auth:                service,
		portal:              store.portal,
		masterRekening:      buildMasterRekening(cfg, store, logger),
		masterStatus:        claimStatusService,
		masterStatusProgres: progressStatusService,
		menu:                menuService,
		readyAliases:        store.readyAliases,
		close:               store.close,
	}, nil
}

// buildMasterRekening menyusun modul Master Rekening di balik seam-nya.
//
// Seam Kasir diisi klien HTTP nyata bila alamatnya sudah dikonfigurasi, dan tiruan bila
// belum. Perbedaannya diumumkan di log: layar yang tampak bekerja padahal pendaftaran
// ke Kasir tidak pernah terjadi adalah kegagalan yang tidak terlihat siapa pun sampai
// pembayaran pertama tertahan.
func buildMasterRekening(cfg config.Config, store storage, logger *slog.Logger) *masterrekeningusecase.Service {
	var (
		cashierSystem masterrekening.Cashier
		notifier      masterrekening.Notifier
	)

	if cfg.SMTP.Active() {
		notifier = masterrekeningnotif.NewSender(masterrekeningnotif.Config{
			Host:     cfg.SMTP.Host,
			Port:     cfg.SMTP.Port,
			User:     cfg.SMTP.User,
			Password: cfg.SMTP.Password,
			From:     cfg.SMTP.From,
			To:       cfg.SMTP.AlertRecipients,
			Timeout:  cfg.SMTP.Timeout,
		})
	} else {
		notifier = &masterrekeningnotif.Fake{}
		logger.Warn("pengirim surel tiruan dipakai",
			slog.String("akibat", "Tim IT TIDAK diberi tahu lewat surel bila pendaftaran ke Cashier gagal"),
			slog.String("perbaikan", "isi SMTP_HOST, SMTP_PORT, SMTP_DARI, dan SMTP_PENERIMA_PERINGATAN"))
	}

	switch {
	case cfg.Cashier.Active():
		cashierSystem = masterrekeningcashier.NewClient(masterrekeningcashier.Config{
			RegisterURL: cfg.Cashier.RegisterURL,
			UpdateURL:   cfg.Cashier.UpdateURL,
			User:        cfg.Cashier.User,
			Password:    cfg.Cashier.Password,
			Timeout:     cfg.Cashier.Timeout,
		})

	case store.accountInOracle:
		// KASIR TIRUAN DILARANG DI ATAS BASIS DATA SUNGGUHAN.
		//
		// Fake menjawab "berhasil" beserta nomor rekening Kasir karangan. Bila
		// jawaban itu ditulis ke POOLDATA.LST_ACCOUNT yang asli, kolom STS_SERVICE
		// dan ID_REKASIR akan memuat jejak pendaftaran yang tidak pernah terjadi —
		// dan tidak ada apa pun sesudahnya yang dapat membedakannya dari pendaftaran
		// yang sungguhan. Data palsu di master rekening lebih berbahaya daripada
		// langkah yang hilang.
		//
		// Seam dibiarkan nil. Usecase sudah menanganinya: rekening tetap dapat
		// disetujui komite, dan langkah pendaftaran ke Kasir dilewati tanpa
		// meninggalkan jejak apa pun.
		cashierSystem = nil
		logger.Warn("pendaftaran ke Cashier DILEWATI",
			slog.String("sebab", "alamat sistem Cashier belum dikonfigurasi, sedangkan master rekening membaca basis data sungguhan"),
			slog.String("akibat", "rekening yang disetujui komite TIDAK didaftarkan ke Cashier, dan tidak ada jejak Cashier yang ditulis"),
			slog.String("perbaikan", "isi KASIR_URL_DAFTAR_REKENING dan KASIR_URL_PERBARUI_REKENING"))

	default:
		// Penyimpanan di memori: tiruan aman dipakai dan memang berguna, karena ia
		// membuat seluruh alur dapat dicoba tanpa basis data dan tanpa jaringan.
		cashierSystem = masterrekeningcashier.NewFake()
		logger.Warn("sistem Cashier tiruan dipakai",
			slog.String("akibat", "rekening yang disetujui komite TIDAK didaftarkan ke Cashier yang sesungguhnya"),
			slog.String("perbaikan", "isi KASIR_URL_DAFTAR_REKENING dan KASIR_URL_PERBARUI_REKENING"))
	}

	return masterrekeningusecase.NewService(masterrekeningusecase.Options{
		Repo:        store.account,
		Bank:        store.accountBank,
		Cashier:     cashierSystem,
		Notifier:    notifier,
		Clock:       clock.System{},
		PortalAlias: cfg.PrimaryPortal,
	})
}

// needsOracle menyatakan apakah koneksi basis data harus dibuka.
//
// Dua sebab yang BERBEDA, dan memisahkannya penting:
//
//   - PENYIMPANAN=oracle  → tabel CPNC_PENGGUNA dan CPNC_SESI_AKTIF hidup di sana.
//   - IDENTITAS_ADAPTER=hcq → alamat layanan HCQ (GCNM_CONNECT_REST) dan daftar login
//     non-karyawan (M_LOGIN_PNC) dibaca dari sana, tanpa menyentuh tabel CPNC_ sama
//     sekali.
//
// Menyatukan keduanya — seperti yang saya lakukan mula-mula — memaksa migrasi 0001
// selesai sebelum integrasi HCC/HCQ dapat dicoba lewat layar, padahal keduanya tidak
// saling bergantung.
func needsOracle(cfg config.Config) bool {
	return cfg.Storage == config.StorageOracle ||
		cfg.IdentityAdapter == config.IdentityAdapterHCQ
}

// buildStorage membuka koneksi portal bila diperlukan dan memasang repo di atasnya.
func buildStorage(cfg config.Config, production bool, logger *slog.Logger) (storage, error) {
	if cfg.Storage == config.StorageMemory && production {
		// Session di memori satu instans melanggar tuntutan stateless (D-27): instans
		// kedua di belakang load balancer tidak akan mengenali sesi yang diterbitkan
		// instans pertama. Penolakannya ada di kode, bukan di nilai konfigurasi.
		return storage{}, errors.New(
			"penyimpanan memori menolak berjalan di lingkungan produksi: sesi wajib dikenali seluruh instans (D-27)")
	}

	store := storage{
		close:        func() {},
		readyAliases: func() []string { return nil },
	}

	if needsOracle(cfg) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		pool, err := db.NewPool(ctx, cfg.PrimaryPortal, portalParameters(cfg), func(alias string, err error) {
			// Portal yang gagal dibuka dicatat tetapi tidak menghentikan aplikasi:
			// pengisian kredensial tiap entitas berjalan bertahap, dan satu entitas
			// yang belum siap tidak boleh menghalangi entitas yang sudah siap.
			logger.Warn("portal tidak tersedia",
				slog.String("portal", alias),
				slog.String("sebab", err.Error()))
		})
		if err != nil {
			return storage{}, err
		}
		logger.Info("koneksi portal terbuka", slog.Any("portal", pool.Available()))

		primary := pool.Primary()
		store.legacy = sqlstore.NewLegacy(primary)
		store.portal = portalsql.NewRepo(primary)
		store.account = masterrekeningsql.NewRepo(primary)
		store.accountBank = masterrekeningsql.NewBankRepo(primary)
		store.accountInOracle = true
		store.readyAliases = pool.Available
		store.masterStatus = masterstatussql.NewRepo(primary)
		store.menu = menusql.NewRepo(primary)
		store.close = pool.Close

		// Setiap permintaan memilih koneksi entitasnya sendiri. Portal yang tidak
		// dikenal atau koneksinya belum hidup menghasilkan galat dari For(), TIDAK
		// pernah dialihkan ke koneksi utama sebagai cadangan.
		store.progressStatusSelector = func(alias string) (masterstatusprogres.Repo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return masterstatusprogressql.NewRepo(conn), nil
		}
	} else {
		store.portal = portalmemory.NewRepo(portalmemory.SampleList()...)
		store.account = masterrekeningmemory.NewRepo()
		store.accountBank = masterrekeningmemory.NewBankRepo(masterrekeningmemory.SampleBanks()...)
		// Ke-33 status nyata ikut dimuat, sehingga layar Master Status Klaim dapat
		// dicoba lengkap tanpa Oracle dan tanpa menunggu migrasi 0002.
		store.masterStatus = masterstatusmemory.NewRepo(masterstatusmemory.SampleList()...)
		store.readyAliases = func() []string { return []string{cfg.PrimaryPortal} }
		store.progressStatusSelector = progressStatusSelectorMemory(cfg.PrimaryPortal)
		// NewDevRepo, bukan NewSampleRepo: isi contoh m_login_group_pnc.csv hanya
		// memuat satu login, dan login provider tiruan tidak ada di dalamnya. Tanpa
		// itu, masuk saat pengembangan menghasilkan menu kosong yang tampak rusak.
		store.menu = menumemory.NewDevRepo()
	}

	switch cfg.Storage {
	case config.StorageOracle:
		primary := store.legacy.DB()
		store.user = sqlstore.NewUserRepo(primary)
		store.session = sqlstore.NewSessionRepo(primary)

	case config.StorageMemory:
		// Catatan dan sesi di memori. Dipakai bersama IDENTITAS_ADAPTER=hcq, ini
		// memungkinkan masuk dengan kredensial SUNGGUHAN sebelum migrasi 0001
		// dijalankan DBA — yang hilang hanya ketahanan sesi terhadap restart dan
		// pengenalan sesi lintas instans.
		if cfg.IdentityAdapter == config.IdentityAdapterHCQ {
			logger.Warn("identitas nyata dengan penyimpanan memori",
				slog.String("akibat", "sesi hilang saat restart dan tidak dikenali instans lain; hanya untuk pengujian"))
		}
		store.user = memory.NewUserRepo()
		store.session = memory.NewSessionRepo()

	default:
		store.close()
		return storage{}, fmt.Errorf("penyimpanan %q tidak dikenal", cfg.Storage)
	}

	return store, nil
}

// progressStatusSelectorMemory menyusun penyimpanan master status progres di memori.
//
// Satu portal mendapat satu penyimpanan, dibuat saat pertama diminta lalu dipakai
// kembali. Kalau dibuat ulang setiap permintaan, penambahan yang baru disimpan akan
// hilang pada permintaan berikutnya dan layarnya tampak rusak tanpa sebab.
//
// Hanya portal utama yang dilayani di sini, sejalan dengan readyAliases pada cabang
// tanpa Oracle yang juga menyebut portal utama saja. Memilih portal lain tanpa basis
// data karena itu ditolak dengan galat yang sama seperti di produksi: perilaku
// penolakannya ikut teruji saat pengembangan, bukan hanya nanti.
func progressStatusSelectorMemory(primaryAlias string) masterstatusprogres.RepoSelector {
	var lock sync.Mutex
	store := map[string]masterstatusprogres.Repo{}

	return func(alias string) (masterstatusprogres.Repo, error) {
		clean := strings.ToUpper(strings.TrimSpace(alias))
		if clean != strings.ToUpper(strings.TrimSpace(primaryAlias)) {
			return nil, fmt.Errorf("%w: portal %q tidak tersedia tanpa basis data", portal.ErrNotReady, alias)
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := masterstatusprogresmemory.NewRepo(masterstatusprogresmemory.SampleList()...)
		store[clean] = fresh
		return fresh, nil
	}
}

// buildIdentity menyusun rantai sumber identitas.
//
// Urutannya adalah aturan bisnis yang ditetapkan Work Owner 2026-09-16: HCC/HCQ lebih
// dulu untuk karyawan, lalu POOLDATA.M_LOGIN_PNC untuk non-karyawan.
func buildIdentity(cfg config.Config, production bool, legacy *sqlstore.Legacy) (auth.Identity, error) {
	switch cfg.IdentityAdapter {
	case config.IdentityAdapterFake:
		// Penolakan terhadap produksi ada di dalam provider, bukan hanya di sini —
		// satu nilai konfigurasi tidak boleh cukup untuk menyalakannya di produksi.
		return provider.NewFake(production, nil)

	case config.IdentityAdapterHCQ:
		if legacy == nil {
			return nil, errors.New("IDENTITAS_ADAPTER=hcq menuntut koneksi basis data: alamat layanan HCQ dan daftar login non-karyawan keduanya dibaca dari sana")
		}
		hcq, err := provider.NewHCQ(provider.HCQOptions{
			Katalog:     legacy,
			PortalAlias: cfg.PrimaryPortal,
			User:        cfg.HCQ.User,
			Password:    cfg.HCQ.Password,
			Timeout:     cfg.HCQ.Timeout,
		})
		if err != nil {
			return nil, err
		}
		local, err := provider.NewLocal(legacy)
		if err != nil {
			return nil, err
		}
		return provider.NewChain(
			provider.Link{Name: "hcq", Sumber: hcq},
			provider.Link{Name: "lokal", Sumber: local},
		)

	default:
		return nil, fmt.Errorf("adapter identitas %q tidak dikenal", cfg.IdentityAdapter)
	}
}

// portalParameters mengubah konfigurasi portal menjadi parameter koneksi, melewati
// portal yang variabel wajibnya belum terisi.
func portalParameters(cfg config.Config) []db.Parameter {
	alias := make([]string, 0, len(cfg.Portal))
	for a := range cfg.Portal {
		alias = append(alias, a)
	}
	sort.Strings(alias)

	parameter := make([]db.Parameter, 0, len(alias))
	for _, a := range alias {
		b := cfg.Portal[a]
		if !b.Complete() {
			continue
		}
		parameter = append(parameter, db.Parameter{
			Alias:              b.Alias,
			Host:               b.Host,
			Port:               b.Port,
			Service:            b.Service,
			User:               b.User,
			Password:           b.Password,
			MaxConnections:     b.MaxConnections,
			MaxIdle:            b.MaxIdle,
			ConnectionLifetime: b.ConnectionLifetime,
		})
	}
	return parameter
}

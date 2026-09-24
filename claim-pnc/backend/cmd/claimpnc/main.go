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
	"claim-pnc/internal/inboxanalystdoctor"
	"claim-pnc/internal/inboxautoclaim"
	"claim-pnc/internal/inboxclaimtreatynonprop"
	"claim-pnc/internal/inboxclaimtreatyprop"
	"claim-pnc/internal/inboxcloseclaim"
	"claim-pnc/internal/inboxkomunikasicabang"
	"claim-pnc/internal/inboxlaporanklaim"
	"claim-pnc/internal/inboxmanagerreceivepucl"
	"claim-pnc/internal/inboxoutstanding"
	"claim-pnc/internal/inboxprogressclaim"
	"claim-pnc/internal/inboxrclpucl"
	"claim-pnc/internal/inboxxol"
	"claim-pnc/internal/komite"
	"claim-pnc/internal/masterdominanfactor"
	"claim-pnc/internal/mastermasking"
	"claim-pnc/internal/masterpenyebabkerugian"
	"claim-pnc/internal/masterpicteknik"
	"claim-pnc/internal/masterrecovery"
	"claim-pnc/internal/masterrekening"
	"claim-pnc/internal/masterstatus"
	"claim-pnc/internal/masterstatusprogres"
	"claim-pnc/internal/mastersurveyors"
	"claim-pnc/internal/mastertipesurveyors"
	"claim-pnc/internal/masterxol"
	"claim-pnc/internal/menu"
	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/platform/config"
	"claim-pnc/internal/platform/db"
	"claim-pnc/internal/platform/httpserver"
	"claim-pnc/internal/platform/logging"
	"claim-pnc/internal/platform/random"
	"claim-pnc/internal/portal"
	"claim-pnc/spa"

	authhttp "claim-pnc/internal/auth/http"
	inboxautoclaimhttp "claim-pnc/internal/inboxautoclaim/http"
	inboxautoclaimmemory "claim-pnc/internal/inboxautoclaim/repo/memory"
	inboxautoclaimsql "claim-pnc/internal/inboxautoclaim/repo/sqlstore"
	inboxautoclaimusecase "claim-pnc/internal/inboxautoclaim/usecase"
	inboxclaimtreatynonprophttp "claim-pnc/internal/inboxclaimtreatynonprop/http"
	inboxclaimtreatynonpropmemory "claim-pnc/internal/inboxclaimtreatynonprop/repo/memory"
	inboxclaimtreatynonpropsql "claim-pnc/internal/inboxclaimtreatynonprop/repo/sqlstore"
	inboxclaimtreatynonpropusecase "claim-pnc/internal/inboxclaimtreatynonprop/usecase"

	inboxclaimtreatypropthttp "claim-pnc/internal/inboxclaimtreatyprop/http"
	inboxclaimtreatypropmemory "claim-pnc/internal/inboxclaimtreatyprop/repo/memory"
	inboxclaimtreatypropsql "claim-pnc/internal/inboxclaimtreatyprop/repo/sqlstore"
	inboxclaimtreatypropusecase "claim-pnc/internal/inboxclaimtreatyprop/usecase"

	inboxanalystdoctorhttp "claim-pnc/internal/inboxanalystdoctor/http"
	inboxanalystdoctormemory "claim-pnc/internal/inboxanalystdoctor/repo/memory"
	inboxanalystdoctorsql "claim-pnc/internal/inboxanalystdoctor/repo/sqlstore"
	inboxanalystdoctorusecase "claim-pnc/internal/inboxanalystdoctor/usecase"
	inboxcloseclaimhttp "claim-pnc/internal/inboxcloseclaim/http"
	inboxcloseclaimmemory "claim-pnc/internal/inboxcloseclaim/repo/memory"
	inboxcloseclaimsql "claim-pnc/internal/inboxcloseclaim/repo/sqlstore"
	inboxcloseclaimusecase "claim-pnc/internal/inboxcloseclaim/usecase"
	inboxkomunikasicabanghttp "claim-pnc/internal/inboxkomunikasicabang/http"
	inboxkomunikasicabangmemory "claim-pnc/internal/inboxkomunikasicabang/repo/memory"
	inboxkomunikasicabangsql "claim-pnc/internal/inboxkomunikasicabang/repo/sqlstore"
	inboxkomunikasicabangusecase "claim-pnc/internal/inboxkomunikasicabang/usecase"
	inboxlaporanklaimhttp "claim-pnc/internal/inboxlaporanklaim/http"
	inboxlaporanklaimmemory "claim-pnc/internal/inboxlaporanklaim/repo/memory"
	inboxlaporanklaimsql "claim-pnc/internal/inboxlaporanklaim/repo/sqlstore"
	inboxlaporanklaimusecase "claim-pnc/internal/inboxlaporanklaim/usecase"
	inboxmanagerreceivepuclhttp "claim-pnc/internal/inboxmanagerreceivepucl/http"
	inboxmanagerreceivepuclmemory "claim-pnc/internal/inboxmanagerreceivepucl/repo/memory"
	inboxmanagerreceivepuclsql "claim-pnc/internal/inboxmanagerreceivepucl/repo/sqlstore"
	inboxmanagerreceivepuclusecase "claim-pnc/internal/inboxmanagerreceivepucl/usecase"
	inboxoutstandinghttp "claim-pnc/internal/inboxoutstanding/http"
	inboxoutstandingmemory "claim-pnc/internal/inboxoutstanding/repo/memory"
	inboxoutstandingsql "claim-pnc/internal/inboxoutstanding/repo/sqlstore"
	inboxoutstandingusecase "claim-pnc/internal/inboxoutstanding/usecase"
	inboxprogressclaimhttp "claim-pnc/internal/inboxprogressclaim/http"
	inboxprogressclaimmemory "claim-pnc/internal/inboxprogressclaim/repo/memory"
	inboxprogressclaimsql "claim-pnc/internal/inboxprogressclaim/repo/sqlstore"
	inboxprogressclaimusecase "claim-pnc/internal/inboxprogressclaim/usecase"
	inboxrclpuclhttp "claim-pnc/internal/inboxrclpucl/http"
	inboxrclpuclmemory "claim-pnc/internal/inboxrclpucl/repo/memory"
	inboxrclpuclsql "claim-pnc/internal/inboxrclpucl/repo/sqlstore"
	inboxrclpuclusecase "claim-pnc/internal/inboxrclpucl/usecase"
	inboxxolhttp "claim-pnc/internal/inboxxol/http"
	inboxxolmemory "claim-pnc/internal/inboxxol/repo/memory"
	inboxxolsql "claim-pnc/internal/inboxxol/repo/sqlstore"
	inboxxolusecase "claim-pnc/internal/inboxxol/usecase"
	komitehttp "claim-pnc/internal/komite/http"
	komitememory "claim-pnc/internal/komite/repo/memory"
	komitesql "claim-pnc/internal/komite/repo/sqlstore"
	komiteusecase "claim-pnc/internal/komite/usecase"
	masterdominanfactorhttp "claim-pnc/internal/masterdominanfactor/http"
	masterdominanfactormemory "claim-pnc/internal/masterdominanfactor/repo/memory"
	masterdominanfactorsql "claim-pnc/internal/masterdominanfactor/repo/sqlstore"
	masterdominanfactorusecase "claim-pnc/internal/masterdominanfactor/usecase"
	mastermaskinghttp "claim-pnc/internal/mastermasking/http"
	mastermaskingmemory "claim-pnc/internal/mastermasking/repo/memory"
	mastermaskingsql "claim-pnc/internal/mastermasking/repo/sqlstore"
	mastermaskingusecase "claim-pnc/internal/mastermasking/usecase"
	masterpenyebabkerugianhttp "claim-pnc/internal/masterpenyebabkerugian/http"
	masterpenyebabkerugianmemory "claim-pnc/internal/masterpenyebabkerugian/repo/memory"
	masterpenyebabkerugiansql "claim-pnc/internal/masterpenyebabkerugian/repo/sqlstore"
	masterpenyebabkerugianusecase "claim-pnc/internal/masterpenyebabkerugian/usecase"
	masterpicteknikdirectory "claim-pnc/internal/masterpicteknik/directory"
	masterpicteknikhttp "claim-pnc/internal/masterpicteknik/http"
	masterpicteknikmemory "claim-pnc/internal/masterpicteknik/repo/memory"
	masterpictekniksql "claim-pnc/internal/masterpicteknik/repo/sqlstore"
	masterpicteknikusecase "claim-pnc/internal/masterpicteknik/usecase"
	masterrecoveryhttp "claim-pnc/internal/masterrecovery/http"
	masterrecoverymemory "claim-pnc/internal/masterrecovery/repo/memory"
	masterrecoverysql "claim-pnc/internal/masterrecovery/repo/sqlstore"
	masterrecoveryusecase "claim-pnc/internal/masterrecovery/usecase"
	masterrecoveryva "claim-pnc/internal/masterrecovery/virtualaccount"
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
	mastersurveyorsaccount "claim-pnc/internal/mastersurveyors/account"
	mastersurveyorscommittee "claim-pnc/internal/mastersurveyors/committee"
	mastersurveyorshttp "claim-pnc/internal/mastersurveyors/http"
	mastersurveyorsmemory "claim-pnc/internal/mastersurveyors/repo/memory"
	mastersurveyorssql "claim-pnc/internal/mastersurveyors/repo/sqlstore"
	mastersurveyorsusecase "claim-pnc/internal/mastersurveyors/usecase"
	mastertipesurveyorshttp "claim-pnc/internal/mastertipesurveyors/http"
	mastertipesurveyorsmemory "claim-pnc/internal/mastertipesurveyors/repo/memory"
	mastertipesurveyorssql "claim-pnc/internal/mastertipesurveyors/repo/sqlstore"
	mastertipesurveyorsusecase "claim-pnc/internal/mastertipesurveyors/usecase"
	masterxolhttp "claim-pnc/internal/masterxol/http"
	masterxolnotif "claim-pnc/internal/masterxol/notification"
	masterxolmemory "claim-pnc/internal/masterxol/repo/memory"
	masterxolsql "claim-pnc/internal/masterxol/repo/sqlstore"
	masterxolusecase "claim-pnc/internal/masterxol/usecase"
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
	} else {
		// Kapan antarmuka yang TERSEMAT dibangun — bukan kapan `npm run build` terakhir
		// dijalankan di folder frontend. Keduanya berbeda bila binary tidak ikut
		// dibangun ulang, dan perbedaan itu tidak meninggalkan jejak lain sama sekali:
		// aplikasi menyajikan layar versi lama tanpa satu pun galat, sehingga fitur yang
		// sudah diperbaiki tampak masih rusak.
		logger.Info("antarmuka tersemat", slog.String("dibangun", spaVersionText()))
	}

	// Satu penulis JSON dan satu penulis galat dipakai bersama seluruh modul, supaya
	// bentuk respons dan header Cache-Control-nya tidak berbeda antarmodul.
	writeJSON := func(w http.ResponseWriter, r *http.Request, status int, body any) {
		authhttp.WriteJSON(w, r, status, body, logger)
	}
	writeAuthError := authhttp.WriteError(logger)

	handlerAuth := authhttp.NewHandler(assembly.auth, logger)
	handlerKomite := komitehttp.NewHandler(komitehttp.Options{
		Service:             assembly.komite,
		Logger:              logger,
		WriteResponse:       writeJSON,
		FallbackErrorWriter: komitehttp.ErrorWriter(writeAuthError),
	})
	handlerKomiteInbox := komitehttp.NewInboxHandler(komitehttp.InboxHandlerOptions{
		Service: assembly.komiteInbox,
		// Jembatan satu arah dari modul auth ke modul Komite. Ia dipasang di sini, bukan
		// di dalam salah satu modul, supaya keduanya tetap tidak saling mengimpor — yang
		// tahu keduanya hanyalah berkas perakitan ini.
		//
		// Yang dijembatani LOGIN, bukan NIK. Inbox disaring terhadap `PXASSIGNEDOPERATORID`
		// pada worklist Pega, yang berisi nama seperti `ELLENSUPRIYATI` — dan Work Owner
		// menetapkan kunci pencocokannya adalah login yang DIKETIK pengguna
		// (`docs/keputusan-implementasi.md` §16.5).
		Caller: func(ctx context.Context) (komitehttp.InboxCaller, bool) {
			baseCtx, existing := authhttp.CallerFromContext(ctx)
			if !existing {
				return komitehttp.InboxCaller{}, false
			}
			return komitehttp.InboxCaller{
				Login: baseCtx.User.Login,
				Name:  baseCtx.User.Name,
			}, true
		},
		Logger:              logger,
		WriteResponse:       writeJSON,
		FallbackErrorWriter: komitehttp.ErrorWriter(writeAuthError),
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

	// Master Status Klaim dirakit SESUDAH penulis galat sadar-portal terbentuk: sejak
	// modul ini menjadi per portal, galat portalnya harus dipetakan modul portal — bukan
	// jatuh ke pemeta galat auth sebagai 500.
	handlerMasterStatus := masterstatushttp.NewHandler(masterstatushttp.Options{
		Service:             assembly.masterStatus,
		Logger:              logger,
		WriteResponse:       writeJSON,
		FallbackErrorWriter: masterstatushttp.ErrorWriter(writePortalAwareError),
	})

	// Master Dominan Factor dirakit dengan pola yang sama, dan dengan alasan yang sama:
	// seluruh rutenya menyentuh basis data entitas, sehingga galat portalnya harus
	// dipetakan modul portal — bukan jatuh ke pemeta galat auth sebagai 500.
	dominantFactorHandler := masterdominanfactorhttp.NewHandler(masterdominanfactorhttp.Options{
		Service:             assembly.masterDominanFactor,
		Logger:              logger,
		WriteResponse:       writeJSON,
		FallbackErrorWriter: masterdominanfactorhttp.ErrorWriter(writePortalAwareError),
	})

	// Master XOL dirakit dengan pola yang sama, ditambah satu bahan yang modul master
	// lain tidak punya: identitas pemanggil. Menyimpan di layar ini sekaligus MENGAJUKAN
	// struktur treaty ke komite — persis seperti sistem lama — dan pengajuan tanpa jejak
	// siapa yang mengajukan tidak punya arti (`D-59`).
	xolHandler, err := masterxolhttp.NewHandler(masterxolhttp.Options{
		Service: assembly.masterXOL,
		// Jembatan satu arah dari modul auth ke modul Master XOL. Ia dipasang di sini,
		// bukan di dalam salah satu modul, supaya kedua modul tetap tidak saling
		// mengimpor — yang tahu keduanya hanyalah berkas perakitan ini.
		Caller: func(ctx context.Context) (masterxolhttp.Caller, bool) {
			baseCtx, existing := authhttp.CallerFromContext(ctx)
			if !existing {
				return masterxolhttp.Caller{}, false
			}
			return masterxolhttp.Caller{
				Identity: baseCtx.User.Identity,
				Name:     baseCtx.User.Name,
			}, true
		},
		Logger:              logger,
		WriteResponse:       writeJSON,
		FallbackErrorWriter: masterxolhttp.ErrorWriter(writePortalAwareError),
	})
	if err != nil {
		return err
	}

	// Master Penyebab Kerugian dirakit dengan pola yang sama. Satu hal yang
	// membedakannya dari master lain: keterangan di sini menjadi kolom PENGELOMPOKAN
	// pada laporan Pega yang memakai `GROUP BY COL_DESC`, sehingga portal yang keliru
	// mengubah pengelompokan laporan — bukan hanya isi satu layar.
	causeOfLossHandler := masterpenyebabkerugianhttp.NewHandler(masterpenyebabkerugianhttp.Options{
		Service:             assembly.masterPenyebabKerugian,
		Logger:              logger,
		WriteResponse:       writeJSON,
		FallbackErrorWriter: masterpenyebabkerugianhttp.ErrorWriter(writePortalAwareError),
	})

	maskingHandler, err := mastermaskinghttp.NewHandler(mastermaskinghttp.Options{
		Service: assembly.masterMasking,
		// Jembatan satu arah dari modul auth ke modul Master Masking. Ia dipasang di sini,
		// bukan di dalam salah satu modul, supaya kedua modul tetap tidak saling mengimpor —
		// yang tahu keduanya hanyalah berkas perakitan ini.
		//
		// Identitas pemanggil mengisi kolom USERINPUT. Di modul ini ia lebih dari sekadar
		// jejak: yang dicatat adalah siapa yang memberi atau mencabut kewenangan membuka
		// nomor KTP, surel, dan nomor telepon nasabah. `D-59` menetapkan tidak ada
		// pemisahan tugas formal, sehingga catatan ini satu-satunya kontrol pengimbang.
		Caller: func(ctx context.Context) (mastermaskinghttp.Caller, bool) {
			baseCtx, existing := authhttp.CallerFromContext(ctx)
			if !existing {
				return mastermaskinghttp.Caller{}, false
			}
			return mastermaskinghttp.Caller{
				Identity: baseCtx.User.Identity,
				Name:     baseCtx.User.Name,
			}, true
		},
		Logger:        logger,
		WriteResponse: writeJSON,
		WriteError:    mastermaskinghttp.ErrorWriter(writePortalAwareError),
	})
	if err != nil {
		return err
	}

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

	// Inbox Auto Claim. Ia menyentuh basis data entitas, sehingga rutenya memakai
	// activePortalDeps yang sama dengan modul bisnis lain.
	autoClaimHandler, err := inboxautoclaimhttp.NewHandler(inboxautoclaimhttp.Options{
		Service: assembly.inboxAutoClaim,
		// Jembatan satu arah dari modul auth. Yang dibutuhkan hanya LOGIN pemanggil,
		// karena itulah yang tertulis di kolom USERINPUT dan tampil sebagai
		// "User Upload" di grid.
		Caller: func(ctx context.Context) (inboxautoclaimhttp.Caller, bool) {
			baseCtx, existing := authhttp.CallerFromContext(ctx)
			if !existing {
				return inboxautoclaimhttp.Caller{}, false
			}
			return inboxautoclaimhttp.Caller{Login: baseCtx.User.Login}, true
		},
		Logger:        logger,
		WriteResponse: writeJSON,
		WriteError:    inboxautoclaimhttp.ErrorWriter(writePortalAwareError),
	})
	if err != nil {
		return err
	}

	surveyorTypeHandler, err := mastertipesurveyorshttp.NewHandler(mastertipesurveyorshttp.Options{
		Service:       assembly.masterTipeSurveyors,
		Logger:        logger,
		WriteResponse: writeJSON,
		// Penulis galat yang sudah sadar portal: galat portal dipetakan modul portal,
		// sisanya diteruskan ke pemeta modul auth.
		WriteError: mastertipesurveyorshttp.ErrorWriter(writePortalAwareError),
	})
	if err != nil {
		return err
	}

	surveyorHandler, err := mastersurveyorshttp.NewHandler(mastersurveyorshttp.Options{
		Service: assembly.masterSurveyors,
		// Jembatan satu arah dari modul auth ke modul Master Surveyors. Ia dipasang di
		// sini, bukan di dalam salah satu modul, supaya kedua modul tetap tidak saling
		// mengimpor — yang tahu keduanya hanyalah berkas perakitan ini.
		//
		// Identity yang dipakai adalah yang SAMA dengan milik master rekening, karena
		// keduanya dibandingkan dengan kolom komite yang berisi Operator ID.
		Caller: func(ctx context.Context) (mastersurveyorshttp.Caller, bool) {
			baseCtx, existing := authhttp.CallerFromContext(ctx)
			if !existing {
				return mastersurveyorshttp.Caller{}, false
			}
			return mastersurveyorshttp.Caller{
				Identity: baseCtx.User.Identity,
				Name:     baseCtx.User.Name,
			}, true
		},
		Logger:        logger,
		WriteResponse: writeJSON,
		WriteError:    mastersurveyorshttp.ErrorWriter(writePortalAwareError),
	})
	if err != nil {
		return err
	}

	// Inbox Laporan Klaim. Jembatan pemanggilnya membawa LOGIN yang DIKETIK pengguna,
	// bukan NIK: itulah yang dicocokkan ke pxcreateoperator pada tabel warisan dan ke
	// sender pada percakapan.
	claimReportHandler, err := inboxlaporanklaimhttp.NewHandler(inboxlaporanklaimhttp.Options{
		Service: assembly.inboxLaporanKlaim,
		Caller: func(ctx context.Context) (inboxlaporanklaim.Caller, bool) {
			baseCtx, existing := authhttp.CallerFromContext(ctx)
			if !existing {
				return inboxlaporanklaim.Caller{}, false
			}
			// Cabang SENGAJA tidak dibawa dari sini. `baseCtx.User.BranchCode` adalah
			// kode cabang HCC/HCQ (`Placement.BranchCode`), dan layar itu membandingkan
			// terhadap POOLDATA.BRANCH.ID — sistem kode yang berbeda. Memakainya membuat
			// daftar tampil kosong tanpa satu pun pesan galat, dan itu benar-benar
			// terjadi. Penerjemahannya kini tugas BranchResolver.
			return inboxlaporanklaim.Caller{
				Login: baseCtx.User.Login,
				Name:  baseCtx.User.Name,
			}, true
		},
		Logger:        logger,
		WriteResponse: writeJSON,
		WriteError:    inboxlaporanklaimhttp.ErrorWriter(writePortalAwareError),
	})
	if err != nil {
		return err
	}

	picTeknikHandler, err := masterpicteknikhttp.NewHandler(masterpicteknikhttp.Options{
		Service:       assembly.masterPicTeknik,
		Logger:        logger,
		WriteResponse: writeJSON,
		WriteError:    masterpicteknikhttp.ErrorWriter(writePortalAwareError),
	})
	if err != nil {
		return err
	}

	recoveryHandler, err := masterrecoveryhttp.NewHandler(masterrecoveryhttp.Options{
		Service: assembly.masterRecovery,
		// Jembatan satu arah dari modul auth ke modul Master Recovery. Ia dipasang di
		// sini, bukan di dalam salah satu modul, supaya kedua modul tetap tidak saling
		// mengimpor — yang tahu keduanya hanyalah berkas perakitan ini.
		//
		// Identitas pemanggil mengisi kolom USERNAME pada batch dan INPUTOPERATOR pada
		// bukti bayar. Keduanya WAJIB: `D-59` menetapkan tidak ada pemisahan tugas formal,
		// sehingga jejak siapa-mengerjakan-apa adalah satu-satunya kontrol pengimbang yang
		// tersisa.
		Caller: func(ctx context.Context) (masterrecoveryhttp.Caller, bool) {
			baseCtx, existing := authhttp.CallerFromContext(ctx)
			if !existing {
				return masterrecoveryhttp.Caller{}, false
			}
			return masterrecoveryhttp.Caller{
				Identity: baseCtx.User.Identity,
				Name:     baseCtx.User.Name,
			}, true
		},
		Logger:        logger,
		WriteResponse: writeJSON,
		WriteError:    masterrecoveryhttp.ErrorWriter(writePortalAwareError),
	})
	if err != nil {
		return err
	}

	// Inbox XOL. Jembatan pemanggilnya membawa LOGIN, sama seperti View History Claim:
	// identitas yang dipakai sistem lama di layar ini adalah `OperatorID.pyUserIdentifier`,
	// bukan NIK.
	//
	// Modul ini MEMBACA SAJA (keputusan Work Owner 2026-09-20). Ketiga rute tulisnya ada
	// tetapi menolak dengan alasan — lihat inboxxolhttp.Mount.
	inboxXOLHandler := inboxxolhttp.NewHandler(inboxxolhttp.Options{
		Service: assembly.inboxXOL,
		GetCaller: func(ctx context.Context) (inboxxolhttp.Caller, bool) {
			baseCtx, existing := authhttp.CallerFromContext(ctx)
			if !existing {
				return inboxxolhttp.Caller{}, false
			}
			return inboxxolhttp.Caller{Login: baseCtx.User.Login}, true
		},
		Logger:    logger,
		WriteJSON: writeJSON,
		// Galat portal ikut dikenali, karena seluruh rute modul ini berada di balik
		// pemeriksaan portal.
		FallbackErrorWriter: inboxxolhttp.ErrorWriterFrom(writePortalAwareError),
	})
	if err != nil {
		return err
	}

	// Inbox Claim Treaty Prop (`MENU_ID 54`). Jembatan pemanggilnya membawa LOGIN dengan
	// alasan yang sama seperti Inbox XOL: yang dicocokkan ke `PXASSIGNEDOPERATORID` pada
	// tabel penugasan Pega adalah `OperatorID.pyUserIdentifier`, bukan NIK.
	//
	// Modul ini MEMBACA SAJA (keputusan Work Owner 2026-09-21). Rute tulisnya ada tetapi
	// menolak dengan alasan — lihat inboxclaimtreatypropthttp.Mount.
	claimTreatyPropHandler := inboxclaimtreatypropthttp.NewHandler(
		inboxclaimtreatypropthttp.Options{
			Service: assembly.inboxClaimTreatyProp,
			GetCaller: func(ctx context.Context) (inboxclaimtreatypropthttp.Caller, bool) {
				baseCtx, existing := authhttp.CallerFromContext(ctx)
				if !existing {
					return inboxclaimtreatypropthttp.Caller{}, false
				}
				return inboxclaimtreatypropthttp.Caller{Login: baseCtx.User.Login}, true
			},
			Logger:    logger,
			WriteJSON: writeJSON,
			// Galat portal ikut dikenali, karena seluruh rute modul ini berada di balik
			// pemeriksaan portal.
			FallbackErrorWriter: inboxclaimtreatypropthttp.ErrorWriter(writePortalAwareError),
		})

	// Inbox Claim Treaty Non Prop (`MENU_ID 55`). Layar SAUDARA dari yang di atas, dan
	// dirakit terpisah dengan sengaja: keduanya membaca tabel, kolom, dan penanda objek
	// kerja yang berbeda — lihat kepala `internal/inboxclaimtreatynonprop`.
	//
	// Jembatan pemanggilnya membawa LOGIN dengan alasan yang sama: yang dicocokkan ke
	// `PXASSIGNEDOPERATORID` adalah `OperatorID.pyUserIdentifier`, bukan NIK.
	claimTreatyNonPropHandler := inboxclaimtreatynonprophttp.NewHandler(
		inboxclaimtreatynonprophttp.Options{
			Service: assembly.inboxClaimTreatyNonProp,
			GetCaller: func(ctx context.Context) (inboxclaimtreatynonprophttp.Caller, bool) {
				baseCtx, existing := authhttp.CallerFromContext(ctx)
				if !existing {
					return inboxclaimtreatynonprophttp.Caller{}, false
				}
				return inboxclaimtreatynonprophttp.Caller{Login: baseCtx.User.Login}, true
			},
			Logger:    logger,
			WriteJSON: writeJSON,
			// Galat portal ikut dikenali, karena seluruh rute modul ini berada di balik
			// pemeriksaan portal.
			FallbackErrorWriter: inboxclaimtreatynonprophttp.ErrorWriter(writePortalAwareError),
		})

	// Inbox Manager Receive / PUCL (`MENU_ID 56`).
	//
	// Jembatan pemanggilnya membawa LOGIN seperti modul inbox lain, tetapi ALASANNYA
	// berbeda dan perlu dibaca sebelum disamakan: di sini login TIDAK dipakai menyaring
	// satu pun kueri. Layar ini pandangan penyelia atas pekerjaan seluruh petugas, dan
	// identitasnya dipakai untuk JEJAK — setiap pembukaan dicatat, bukan hanya yang
	// mencurigakan (lihat `internal/inboxmanagerreceivepucl/usecase`).
	managerReceivePUCLHandler := inboxmanagerreceivepuclhttp.NewHandler(
		inboxmanagerreceivepuclhttp.Options{
			Service: assembly.inboxManagerReceivePUCL,
			GetCaller: func(ctx context.Context) (inboxmanagerreceivepuclhttp.Caller, bool) {
				baseCtx, existing := authhttp.CallerFromContext(ctx)
				if !existing {
					return inboxmanagerreceivepuclhttp.Caller{}, false
				}
				return inboxmanagerreceivepuclhttp.Caller{Login: baseCtx.User.Login}, true
			},
			Logger:    logger,
			WriteJSON: writeJSON,
			// Galat portal ikut dikenali, karena seluruh rute modul ini berada di balik
			// pemeriksaan portal.
			FallbackErrorWriter: inboxmanagerreceivepuclhttp.ErrorWriter(writePortalAwareError),
		})

	// Inbox RCL/PUCL (`MENU_ID 61`).
	//
	// Jembatan pemanggilnya membawa LOGIN dengan alasan yang sama seperti Inbox Manager
	// Receive / PUCL, dan perlu dibaca sebelum disamakan dengan modul inbox lain: di sini
	// login TIDAK dipakai menyaring satu pun kueri. Antreannya BERSAMA — penyaringnya akun
	// `RCLPUCL`, bukan pengguna — sehingga setiap petugas melihat daftar yang sama.
	// Identitasnya dipakai untuk JEJAK, dan pada permintaan laporan harian rentang
	// tanggalnya ikut dicatat (lihat `internal/inboxrclpucl/usecase`).
	rclPUCLHandler := inboxrclpuclhttp.NewHandler(
		inboxrclpuclhttp.Options{
			Service: assembly.inboxRCLPUCL,
			GetCaller: func(ctx context.Context) (inboxrclpuclhttp.Caller, bool) {
				baseCtx, existing := authhttp.CallerFromContext(ctx)
				if !existing {
					return inboxrclpuclhttp.Caller{}, false
				}
				return inboxrclpuclhttp.Caller{Login: baseCtx.User.Login}, true
			},
			Logger:    logger,
			WriteJSON: writeJSON,
			// Galat portal ikut dikenali, karena seluruh rute modul ini berada di balik
			// pemeriksaan portal.
			FallbackErrorWriter: inboxrclpuclhttp.ErrorWriter(writePortalAwareError),
		})

	// Inbox Komunikasi Cabang. Jembatan pemanggilnya membawa LOGIN, dan di sini alasannya
	// paling keras di antara seluruh modul inbox: login BUKAN sekadar jejak, melainkan
	// bahan yang diterjemahkan menjadi KODE CABANG — dan kode cabang itulah batas datanya.
	//
	// Memakai NIK di sini akan membuat penerjemahan gagal pada setiap pengguna, karena yang
	// dicocokkan `GetIDCabang` adalah `V_HRD_MST.login_aplikasi`. Akibatnya bukan daftar
	// kosong melainkan yang lebih buruk: setiap petugas jatuh ke jalur kantor pusat dan
	// melihat percakapan yang bukan haknya (`P-5`, lihat inboxkomunikasicabang.BranchFilter).
	komunikasiCabangHandler := inboxkomunikasicabanghttp.NewHandler(
		inboxkomunikasicabanghttp.Options{
			Service: assembly.inboxKomunikasiCabang,
			GetCaller: func(ctx context.Context) (inboxkomunikasicabanghttp.Caller, bool) {
				baseCtx, existing := authhttp.CallerFromContext(ctx)
				if !existing {
					return inboxkomunikasicabanghttp.Caller{}, false
				}
				return inboxkomunikasicabanghttp.Caller{Login: baseCtx.User.Login}, true
			},
			Logger:    logger,
			WriteJSON: writeJSON,
			// Galat portal ikut dikenali, karena seluruh rute modul ini berada di balik
			// pemeriksaan portal.
			FallbackErrorWriter: inboxkomunikasicabanghttp.ErrorWriter(writePortalAwareError),
		})

	// Inbox Progress Claim. Jembatan pemanggilnya juga membawa LOGIN: itulah yang
	// dicocokkan ke `PEGA_DASHBOARDPNC.PIC` dan `MST_USER_TEKNIK.OPERATOR_ID`, dan
	// memakai NIK di sini akan membuat rekap per PIC kosong bagi setiap pengguna.
	inboxProgressClaimHandler := inboxprogressclaimhttp.NewHandler(
		inboxprogressclaimhttp.Options{
			Service: assembly.inboxProgressClaim,
			GetCaller: func(ctx context.Context) (inboxprogressclaimhttp.Caller, bool) {
				baseCtx, existing := authhttp.CallerFromContext(ctx)
				if !existing {
					return inboxprogressclaimhttp.Caller{}, false
				}
				return inboxprogressclaimhttp.Caller{Login: baseCtx.User.Login}, true
			},
			Logger:    logger,
			WriteJSON: writeJSON,
			// Galat portal ikut dikenali, karena seluruh rute modul ini berada di balik
			// pemeriksaan portal.
			FallbackErrorWriter: inboxprogressclaimhttp.ErrorWriter(writePortalAwareError),
		})

	// Inbox Analyst Doctor. Jembatan pemanggilnya membawa LOGIN, dan di modul ini ia bukan
	// kenyamanan melainkan syarat: Report Definition menyaring
	// `PC_ASSIGN_WORKLIST.PXASSIGNEDOPERATORID` dengan identitas pemanggil, sehingga memakai
	// NIK di sini akan membuat antrean tampak KOSONG bagi setiap pengguna — dan antrean
	// kosong tidak pernah dilaporkan siapa pun sebagai kerusakan.
	//
	// Clock disuntikkan karena kolom "Lama Waktu Klaim" dihitung darinya (`F-5`).
	inboxAnalystDoctorHandler := inboxanalystdoctorhttp.NewHandler(
		inboxanalystdoctorhttp.Options{
			Service: assembly.inboxAnalystDoctor,
			GetCaller: func(ctx context.Context) (inboxanalystdoctorhttp.Caller, bool) {
				baseCtx, existing := authhttp.CallerFromContext(ctx)
				if !existing {
					return inboxanalystdoctorhttp.Caller{}, false
				}
				return inboxanalystdoctorhttp.Caller{Login: baseCtx.User.Login}, true
			},
			Clock:     clock.System{},
			Logger:    logger,
			WriteJSON: writeJSON,
			// Galat portal ikut dikenali, karena seluruh rute modul ini berada di balik
			// pemeriksaan portal.
			FallbackErrorWriter: inboxanalystdoctorhttp.ErrorWriter(writePortalAwareError),
		})

	outstandingHandler := inboxoutstandinghttp.NewHandler(inboxoutstandinghttp.Options{
		Service: assembly.inboxOutstanding,
		// Jembatan satu arah dari modul auth, dipasang di sini supaya kedua modul tetap
		// tidak saling mengimpor.
		//
		// Hanya Login yang diambil: dari sanalah lini bisnis pemanggil dibaca, dan
		// batas data TIDAK PERNAH berasal dari badan permintaan maupun query string.
		GetCaller: func(ctx context.Context) (inboxoutstandinghttp.Caller, bool) {
			baseCtx, existing := authhttp.CallerFromContext(ctx)
			if !existing {
				return inboxoutstandinghttp.Caller{}, false
			}
			return inboxoutstandinghttp.Caller{Login: baseCtx.User.Login}, true
		},
		Logger:    logger,
		WriteJSON: writeJSON,
		// Galat portal dipetakan modul portal lebih dulu, sisanya jatuh ke pemeta galat
		// auth. Rantai yang sama dipakai modul master status progres.
		FallbackErrorWriter: inboxoutstandinghttp.ErrorWriter(writePortalAwareError),
	})

	closeClaimHandler := inboxcloseclaimhttp.NewHandler(inboxcloseclaimhttp.Options{
		Service: assembly.inboxCloseClaim,
		// Jembatan satu arah dari modul auth, dipasang di sini supaya kedua modul tetap
		// tidak saling mengimpor.
		//
		// DUA field diambil, berbeda dari modul yang hanya membaca: jejak permintaan
		// menyimpan NAMA pemohon bersama login-nya, supaya jejak itu tetap terbaca utuh
		// tanpa join ke tabel pengguna. Jejak yang namanya diambil lewat join akan berubah
		// ketika orangnya berganti nama — dan jejak yang dapat berubah bukan jejak.
		GetCaller: func(ctx context.Context) (inboxcloseclaimhttp.Caller, bool) {
			baseCtx, existing := authhttp.CallerFromContext(ctx)
			if !existing {
				return inboxcloseclaimhttp.Caller{}, false
			}
			return inboxcloseclaimhttp.Caller{
				Login: baseCtx.User.Login,
				Name:  baseCtx.User.Name,
			}, true
		},
		Logger:    logger,
		WriteJSON: writeJSON,
		// Galat portal dipetakan modul portal lebih dulu, sisanya jatuh ke pemeta galat
		// auth — rantai yang sama dipakai modul Inbox Outstanding tepat di atasnya.
		FallbackErrorWriter: inboxcloseclaimhttp.ErrorWriter(writePortalAwareError),
	})

	accountHandler := masterrekeninghttp.NewHandler(masterrekeninghttp.Options{
		// Adapter dari pemilih layanan bertipe konkret menjadi pemilih bertipe antarmuka.
		// Galatnya dikembalikan lebih dulu, bukan dibungkus: nil bertipe *Service yang
		// terlanjur masuk ke antarmuka akan terbaca sebagai layanan yang ada padahal
		// tidak.
		ServiceSelector: func(alias string) (masterrekeninghttp.Service, error) {
			service, err := assembly.masterRekening(alias)
			if err != nil {
				return nil, err
			}
			return service, nil
		},
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

				// Inbox Auto Claim memuat nomor polis, nilai klaim, dan nama
				// perusahaan rekanan; tidak satu pun boleh terbaca tanpa sesi.
				inboxautoclaimhttp.Mount(protected, autoClaimHandler, activePortalDeps)
				// Inbox Laporan Klaim. Seluruh rutenya memasang pemeriksaan portal di
				// dalam Mount — tidak satu pun yang boleh dilayani tanpa entitas yang
				// jelas, karena setiap rutenya menyentuh basis data entitas.
				inboxlaporanklaimhttp.Mount(protected, claimReportHandler, activePortalDeps)
				inboxoutstandinghttp.Mount(protected, outstandingHandler, activePortalDeps)
				// Inbox Close Claim — klaim yang sudah tutup, beserta permintaan
				// membukanya kembali dan menyalinnya.
				//
				// Satu-satunya modul inbox yang MENULIS. Yang ditulisnya bukan klaim
				// melainkan POOLDATA.CPNC_PERMINTAAN_KLAIM, tabel milik aplikasi ini
				// sendiri — `P-1` menetapkan klaim masih ditulis Pega selama masa paralel.
				//
				// Kewenangannya belum diperiksa per peran (TKT-F3-005). Di Pega, butir
				// menunya dijaga When rule yang membukanya bagi empat access group
				// ditambah TIGA Operator ID perorangan yang tertanam di dalam rule —
				// persis jenis hardcode yang D-15 hapus.
				inboxcloseclaimhttp.Mount(protected, closeClaimHandler, activePortalDeps)
				// Master Tipe Surveyors. Sama seperti di atas: pemeriksaan portal
				// dipasang di dalam Mount, karena SELURUH rutenya menyentuh basis
				// data entitas.
				mastertipesurveyorshttp.Mount(protected, surveyorTypeHandler, activePortalDeps)
				// Master Surveyors — daftar ORANGNYA. Barisnya memuat nama, alamat,
				// telepon, surel, dan nama login aplikasi seseorang; tidak satu pun
				// boleh terbaca tanpa sesi. Keputusan komite di dalamnya diperiksa
				// terhadap kolom KOMITE, dan itu satu-satunya kontrol kewenangan yang
				// benar-benar ada selama TKT-F3-005 belum dikerjakan (D-59).
				mastersurveyorshttp.Mount(protected, surveyorHandler, activePortalDeps)
				// Master PIC Teknik. Sama seperti di atas — seluruh rutenya menyentuh
				// entitas, termasuk pencarian direktori yang alamat layanannya pun dibaca
				// per entitas.
				masterpicteknikhttp.Mount(protected, picTeknikHandler, activePortalDeps)
				// Master Recovery. Sama seperti di atas, dengan satu hal yang lebih
				// berat: rutenya MENERBITKAN REKENING VIRTUAL dan MENCATAT NILAI UANG,
				// sehingga portal yang keliru bukan sekadar menampilkan data yang salah
				// — ia dapat mengarahkan dana ke rekening badan hukum lain (R-20).
				masterrecoveryhttp.Mount(protected, recoveryHandler, activePortalDeps)
				// Master rekening memuat nama, NIK, nomor rekening, dan surel pihak
				// ketiga; tidak satu pun boleh terbaca tanpa sesi.
				masterrekeninghttp.Mount(protected, accountHandler, activePortalDeps)
				// Master data juga berada di balik sesi. Pemeriksaan peran — "apakah
				// pemanggil memiliki menu Master Data" (D-59) — belum ada di sini
				// karena TKT-F3-004 dan TKT-F3-005 belum dikerjakan; keadaannya sama
				// dengan seluruh rute lain hari ini.
				masterstatushttp.Mount(protected, handlerMasterStatus, activePortalDeps)
				// Master Dominan Factor. Sama seperti di atas — seluruh rutenya
				// menyentuh basis data entitas. Satu hal yang membedakannya: nama
				// faktor di sini ikut terbaca LAPORAN Outstanding per Cabang lewat
				// LISTAGG, sehingga portal yang keliru mengubah isi laporan, bukan
				// hanya tampilan satu layar.
				masterdominanfactorhttp.Mount(protected, dominantFactorHandler, activePortalDeps)
				// Master XOL. Seluruh rutenya menyentuh basis data entitas, dan di modul
				// ini akibat salah entitas menjalar jauh: struktur treaty menentukan
				// pembagian klaim ke para reasuradur, sehingga limit dan share satu badan
				// hukum yang terbaca — apalagi tersimpan — di badan hukum lain akan
				// mengubah hasil perhitungan PLA/DLA (R-20).
				masterxolhttp.Mount(protected, xolHandler, activePortalDeps)
				// Master Penyebab Kerugian — TINGKAT GOLONGAN saja (MENU_ID 20).
				// Rinciannya (MENU_ID 38) butir menu tersendiri dan belum dibangun.
				// Keterangan di sini dibaca 19 rule Pega dan menjadi kolom
				// pengelompokan pada laporan, sehingga portal yang keliru mengubah
				// pengelompokan laporan entitas lain (`R-20`).
				masterpenyebabkerugianhttp.Mount(protected, causeOfLossHandler, activePortalDeps)
				// Master Masking. Portal yang keliru di sini berakibat paling berat
				// di antara seluruh master yang sudah dibangun: yang tampil bukan
				// daftar kode, melainkan daftar siapa yang boleh membuka nomor KTP
				// dan nomor telepon nasabah badan hukum lain (`R-20`).
				mastermaskinghttp.Mount(protected, maskingHandler, activePortalDeps)

				// Modul Komite memasang tiga kelompok rute sekaligus: master ambang di
				// bawah master/, perhitungan penjenjangan di bawah komite/, dan Inbox
				// Komite di bawah komite/inbox.
				//
				// Dua yang pertama DIBACA SAJA — tidak ada satu pun jalur yang menulis ke
				// POOLDATA.EMAILKOMITE selama masa paralel (P-1, keputusan Work Owner
				// 2026-09-17).
				//
				// Yang ketiga MENULIS, dan hanya ke satu tempat: tabel keputusan milik
				// aplikasi ini sendiri (migrasi 0004). Kasusnya tetap dibaca saja dari
				// tabel warisan.
				//
				// Isi layar ini memperlihatkan siapa yang berwenang menyetujui uang, dan
				// setiap barisnya memuat nilai klaim serta nama tertanggung. Tidak satu
				// pun boleh terbaca tanpa sesi.
				komitehttp.Mount(protected, handlerKomite, handlerKomiteInbox)

				// Inbox XOL memuat nilai klaim agregat satu perjanjian reasuransi,
				// nama reasuradur, dan alamat surelnya. Tidak satu pun boleh terbaca
				// tanpa sesi, dan seluruhnya dijaga pemeriksaan portal.
				//
				// Ia MEMBACA SAJA (keputusan Work Owner 2026-09-20): keempat tabel XOL
				// yang ditulis sistem lama tetap dimiliki Pega selama masa paralel
				// (`P-1`). Bedakan dari Master XOL di atas, yang MENULIS struktur
				// treaty-nya — keduanya menyentuh MST_XOL_PNC dan kerabatnya, dan hanya
				// satu di antaranya yang boleh menulis.
				inboxxolhttp.Mount(protected, inboxXOLHandler, activePortalDeps)

				// Inbox Claim Treaty Prop memuat nama tertanggung dan nama Ceding Co —
				// perusahaan asuransi yang mengalihkan risikonya kepada ASM. Keduanya
				// milik satu badan hukum, sehingga seluruh rutenya dijaga pemeriksaan
				// portal, termasuk rute keterangan layarnya.
				//
				// Ia MEMBACA SAJA (keputusan Work Owner 2026-09-21): pembuatan klaim
				// treaty menulis objek kerja di tabel yang masih dimiliki Pega selama
				// masa paralel (`P-1`).
				inboxclaimtreatypropthttp.Mount(
					protected, claimTreatyPropHandler, activePortalDeps)

				// Inbox Claim Treaty Non Prop memuat data yang sama sifatnya —
				// nama tertanggung dan nama Ceding Co milik satu badan hukum —
				// sehingga rutenya dijaga pemeriksaan portal yang sama.
				//
				// Ia punya satu rute yang tidak dimiliki layar Prop: ekspor berkas.
				// Berkas itu memuat data nasabah, dan justru karena ia terunduh ke
				// perangkat pengguna, pemeriksaan portalnya tidak boleh lebih longgar
				// daripada layarnya.
				inboxclaimtreatynonprophttp.Mount(
					protected, claimTreatyNonPropHandler, activePortalDeps)

				// Inbox Manager Receive / PUCL memuat nomor polis dan nama
				// tertanggung dari DUA antrean sekaligus, dan tidak satu pun
				// tabnya menyaring menurut pemanggil — ia memang pandangan
				// penyelia. Justru karena itu pemeriksaan portalnya tidak boleh
				// lebih longgar: yang terlihat di sini adalah seluruh berkas dan
				// seluruh klaim RCL/PUCL milik satu badan hukum.
				inboxmanagerreceivepuclhttp.Mount(
					protected, managerReceivePUCLHandler, activePortalDeps)

				// Inbox RCL/PUCL memuat nomor polis dan nama tertanggung dari
				// antrean BERSAMA — tidak satu pun tabnya menyaring menurut
				// pemanggil, karena penyaringnya akun antrean. Pemeriksaan
				// portalnya karena itu tidak boleh lebih longgar: yang terlihat
				// di sini adalah seluruh klaim RCL/PUCL milik satu badan hukum.
				//
				// Rute ekspornya menuntut hal yang sama dan sedikit lebih:
				// berkas laporan hariannya dapat diunduh dan dibawa keluar,
				// dengan rentang tanggal yang ditentukan penggunanya sendiri.
				inboxrclpuclhttp.Mount(
					protected, rclPUCLHandler, activePortalDeps)

				// Inbox Komunikasi Cabang memuat percakapan antarpetugas tentang
				// klaim yang sedang berjalan — milik satu badan hukum, bukan milik
				// badan hukum lain. Rutenya menuntut portal DAN disaring cabang;
				// batas kedua itu diselesaikan di dalam modulnya, bukan di sini.
				inboxkomunikasicabanghttp.Mount(
					protected, komunikasiCabangHandler, activePortalDeps)

				// Inbox Progress Claim memuat nama tertanggung, nomor polis, dan
				// catatan progres — seluruhnya milik satu badan hukum. Rutenya karena
				// itu menuntut portal, sama seperti Inbox Admin.
				inboxprogressclaimhttp.Mount(
					protected, inboxProgressClaimHandler, activePortalDeps)

				// Inbox Analyst Doctor memuat nama tertanggung dan klaim Personal
				// Accident. Rutenya menuntut portal karena alasan yang sama dengan
				// modul di atasnya, ditambah satu yang khas: `FR-R2` membatasi akses
				// data medis, dan pembatasan itu tidak bermakna bila datanya datang
				// dari entitas yang salah.
				inboxanalystdoctorhttp.Mount(
					protected, inboxAnalystDoctorHandler, activePortalDeps)
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
	auth   *usecase.Service
	portal portal.Repo
	// masterRekening memilih layanan milik satu portal entitas; sejak 2026-09-19
	// POOLDATA.LST_ACCOUNT dibaca dan ditulis per entitas, bukan dari portal utama saja.
	masterRekening func(string) (*masterrekeningusecase.Service, error)
	masterStatus   *masterstatususecase.Service

	// komite membaca master ambang dan menghitung penjenjangan persetujuan (B-7).
	komite *komiteusecase.Service

	// komiteInbox melayani layar Inbox Komite: daftar pekerjaan anggota komite dan
	// pencatatan keputusannya (`TKT-B07-002`, MENU_ID 52).
	komiteInbox *komiteusecase.InboxService

	// masterStatusProgres memakai pemilih repo per portal, bukan repo tunggal:
	// tabelnya ada di basis data SETIAP entitas (ADR-0030).
	masterStatusProgres *masterstatusprogresusecase.Service

	// masterTipeSurveyors juga per portal, dengan alasan yang sama: POOLDATA.M_SURVEYORS
	// ada di basis data setiap entitas, dan golongan surveyor satu badan hukum tidak
	// boleh terbaca dari badan hukum lain (R-20).
	masterTipeSurveyors *mastertipesurveyorsusecase.Service

	// masterSurveyors adalah daftar ORANGNYA, anak dari master tipe di atas. Per portal
	// dengan alasan yang sama, dan satu alasan tambahan yang lebih berat: barisnya memuat
	// nama, alamat, telepon, dan surel orang — juga nama login aplikasi mereka.
	masterSurveyors *mastersurveyorsusecase.Service

	// masterPicTeknik juga per portal: POOLDATA.MST_USER_TEKNIK ada di basis data setiap
	// entitas, dan daftar petugas satu badan hukum tidak boleh terbaca dari badan hukum
	// lain (R-20).
	masterPicTeknik *masterpicteknikusecase.Service

	// masterRecovery juga per portal: POOLDATA.MST_RECOVERY_ASM_PENJAMINAN dan master
	// virtual account-nya ada di basis data setiap entitas. Di modul ini R-20 menyentuh
	// hal yang paling berat akibatnya — nomor rekening virtual satu badan hukum yang
	// terbaca dari badan hukum lain berarti dana dapat diarahkan ke rekening yang keliru.
	masterRecovery *masterrecoveryusecase.Service

	// masterDominanFactor juga per portal: POOLDATA.M_DOMINAN_FACTOR ada di basis data
	// setiap entitas. Akibat portal yang keliru di sini halus tetapi luas — nama faktor
	// dominan ikut terbaca laporan Outstanding per Cabang lewat LISTAGG, sehingga yang
	// salah bukan satu layar melainkan isi laporan yang dibaca manajemen (`R-20`).
	masterDominanFactor *masterdominanfactorusecase.Service

	// masterXOL juga per portal: keempat tabel MST_XOL_* ada di basis data setiap
	// entitas. Akibat salah entitas di sini menjalar paling jauh di antara butir master —
	// struktur treaty menentukan pembagian klaim ke reasuradur, sehingga yang keliru
	// bukan satu layar melainkan nilai yang dihitung PLA/DLA sesudahnya (R-20).
	masterXOL *masterxolusecase.Service

	// masterPenyebabKerugian juga per portal: POOLDATA.M_CAUSE_OF_LOSS ada di basis data
	// setiap entitas. Akibat portal yang keliru di sini serupa dengan faktor dominan
	// tetapi jangkauannya lebih luas — keterangan penyebab kerugian dibaca 19 rule Pega
	// dan menjadi kolom PENGELOMPOKAN pada dasbor klaim per penyebab kerugian serta
	// laporan XOL per bisnis, sehingga yang keliru bukan satu layar melainkan
	// pengelompokan laporan yang dibaca manajemen (`R-20`).
	masterPenyebabKerugian *masterpenyebabkerugianusecase.Service

	// masterMasking juga per portal: POOLDATA.MST_PROTEKSI_DATA_PNC ada di basis data
	// setiap entitas. Di antara seluruh master yang sudah dibangun, inilah yang portal
	// kelirunya paling berat akibatnya — isinya adalah daftar SIAPA yang boleh melihat
	// nomor KTP, surel, dan nomor telepon nasabah tanpa disamarkan. Membacanya dari
	// entitas yang salah berarti membocorkan peta kewenangan data pribadi badan hukum
	// lain, dan menulisnya ke entitas yang salah berarti memberi orang kewenangan di
	// tempat yang bukan haknya — keduanya tanpa satu pun pesan galat (`R-20`).
	masterMasking *mastermaskingusecase.Service

	// menu menyusun peta menu beserta kewenangan pemakainya.
	menu *menuusecase.Service

	// inboxAutoClaim memakai pemilih repo per portal, sama seperti masterStatusProgres:
	// POOLDATA.TMP_BATCH_AUTO_CLAIM ada di basis data SETIAP entitas (ADR-0030).
	inboxAutoClaim *inboxautoclaimusecase.Service

	// inboxXOL melayani layar Inbox XOL (`MENU_ID 53`).
	inboxXOL *inboxxolusecase.Service

	// inboxClaimTreatyProp melayani layar Inbox Claim Treaty Prop (`MENU_ID 54`).
	//
	// Kedua tabel penugasan yang dibacanya ada di basis data SETIAP entitas (`ADR-0030`),
	// sama seperti modul inbox lain.
	inboxClaimTreatyProp *inboxclaimtreatypropusecase.Service

	// inboxClaimTreatyNonProp melayani layar Inbox Claim Treaty Non Prop (`MENU_ID 55`).
	//
	// Ia layar SAUDARA dari yang di atas dan sengaja berdiri sendiri: ketiga tabel yang
	// dibacanya, penanda objek kerjanya, dan kolom gridnya berbeda.
	inboxClaimTreatyNonProp *inboxclaimtreatynonpropusecase.Service

	// inboxManagerReceivePUCL melayani layar Inbox Manager Receive / PUCL (`MENU_ID 56`).
	//
	// Ia menyatukan DUA antrean yang kelas objek kerjanya berbeda — berkas penerimaan
	// dokumen dan klaim RCL/PUCL — karena begitulah harness `ReceiveDoucument_Harness`
	// menyusunnya.
	inboxManagerReceivePUCL *inboxmanagerreceivepuclusecase.Service
	inboxRCLPUCL            *inboxrclpuclusecase.Service

	// inboxKomunikasiCabang melayani layar Inbox Komunikasi Cabang (`MENU_ID 70`).
	//
	// Berbeda dari modul inbox di atasnya, layar ini DISARING menurut cabang pemanggilnya —
	// bukan antrean bersama. Batas itu diturunkan dari login lewat BranchResolver.
	inboxKomunikasiCabang *inboxkomunikasicabangusecase.Service

	// inboxProgressClaim melayani layar Inbox Progress Claim (`MENU_ID 65`).
	inboxProgressClaim *inboxprogressclaimusecase.Service

	// inboxAnalystDoctor melayani layar Inbox Analyst Doctor (`MENU_ID 60`) — antrean
	// penilaian medis milik satu petugas.
	inboxAnalystDoctor *inboxanalystdoctorusecase.Service

	// inboxLaporanKlaim melayani layar Inbox Laporan Klaim. Sama seperti master status
	// progres, ia memakai pemilih repo per portal: berkas laporan adalah data bisnis
	// milik satu badan hukum (ADR-0030).
	inboxLaporanKlaim *inboxlaporanklaimusecase.Service

	// inboxOutstanding melayani layar Inbox Outstanding — klaim yang masih berjalan.
	inboxOutstanding *inboxoutstandingusecase.Service

	// inboxCloseClaim melayani layar Inbox Close Claim — klaim yang sudah TUTUP.
	//
	// Ia kebalikan tepat dari inboxOutstanding tepat di atasnya: keduanya menyaring dua
	// nilai PYSTATUSWORK yang sama dengan arah yang berlawanan. Satu-satunya modul inbox
	// yang MENULIS, dan yang ditulisnya bukan klaim melainkan permintaan atas klaim.
	inboxCloseClaim *inboxcloseclaimusecase.Service

	readyAliases func() []string
	close        func()
}

// storage memegang seluruh repo yang sudah terpasang di atas sumbernya.
type storage struct {
	user    auth.UserRepo
	session auth.SessionRepo
	portal  portal.Repo
	// claimStatusSelector memilih penyimpanan master status klaim milik satu portal.
	// Sejak 2026-09-19 tabelnya dibaca per entitas, bukan dari portal utama saja.
	claimStatusSelector masterstatus.RepoSelector

	// komite adalah master ambang POOLDATA.EMAILKOMITE — DIBACA SAJA.
	komite komite.Repo

	// komiteInbox membaca kasus komite dari tabel warisan; komiteDecision menulis
	// keputusannya ke tabel milik aplikasi ini.
	//
	// Keduanya dinyatakan TERPISAH meski satu objek yang sama dapat mengisi keduanya
	// (adapter memori memang demikian). Pembelahannya mengikuti kepemilikan tabel:
	// yang satu tidak boleh menulis apa pun, yang lain menulis.
	komiteInbox    komite.InboxRepo
	komiteDecision komite.DecisionRepo

	// accountSelector memilih penyimpanan master rekening milik satu portal entitas.
	// Repo dan BankRepo dipilih bersamaan karena keduanya hidup di basis data yang sama.
	accountSelector func(alias string) (masterrekening.Repo, masterrekening.BankRepo, error)

	// rekeningDiOracle menyatakan master rekening dipasang di atas POOLDATA.LST_ACCOUNT
	// yang sungguhan, bukan di memori. Adapter tiruan yang menulis jejak karangan
	// dilarang di atasnya — lihat rakitMasterRekening.
	accountInOracle bool

	// warisan bernilai nil bila koneksi Oracle tidak dibuka. Ia memberi akses baca ke
	// tiga tabel milik sistem lama: M_PORTAL_PNC, M_LOGIN_PNC, dan GCNM_CONNECT_REST.
	legacy *sqlstore.Legacy

	// inboxXOLSelector memilih penyimpanan Inbox XOL milik satu portal.
	//
	// Fungsi, bukan repo tunggal, dengan alasan yang sama seperti selector di atasnya:
	// perjanjian XOL dan nilai klaimnya adalah data bisnis milik satu badan hukum
	// (`ADR-0030`). Satu repo bersama akan membaca perjanjian satu entitas dari basis
	// data entitas lain — kebocoran lintas badan hukum yang justru dicegah `R-20`.
	inboxXOLSelector inboxxol.RepoSelector

	// claimTreatyPropSelector memilih penyimpanan Inbox Claim Treaty Prop milik satu
	// portal, dengan alasan yang sama persis: barisnya memuat nama tertanggung dan nama
	// Ceding Co, dan keduanya milik satu badan hukum.
	claimTreatyPropSelector inboxclaimtreatyprop.RepoSelector

	// claimTreatyNonPropSelector memilih penyimpanan Inbox Claim Treaty Non Prop milik
	// satu portal, dengan alasan yang sama persis dengan selector di atasnya.
	claimTreatyNonPropSelector inboxclaimtreatynonprop.RepoSelector

	// managerReceivePUCLSelector memilih penyimpanan Inbox Manager Receive / PUCL milik
	// satu portal.
	//
	// Alasannya sama dengan selector di atasnya, dan di modul ini taruhannya paling besar:
	// tidak satu pun tabnya menyaring menurut pemanggil, sehingga jatuh ke koneksi bawaan
	// berarti memperlihatkan SELURUH antrean satu badan hukum kepada petugas badan hukum
	// lain (`R-20`).
	managerReceivePUCLSelector inboxmanagerreceivepucl.RepoSelector
	rclPUCLSelector            inboxrclpucl.RepoSelector

	// komunikasiCabangSelector memilih penyimpanan percakapan milik satu portal.
	komunikasiCabangSelector inboxkomunikasicabang.RepoSelector

	// komunikasiCabangBranch menerjemahkan login petugas menjadi kode cabang klaimnya.
	//
	// Ia SALINAN seam yang sama dengan claimReportBranch, bukan pemakaian ulangnya, dan itu
	// disengaja: seam dideklarasikan di paket yang MEMAKAINYA (`08-TECHNICAL-STRATEGY.md`
	// §2 aturan 2), sehingga kedua modul dapat berpindah ke API pengganti DB Link (`D-25`,
	// `R-03`) pada waktu yang berbeda tanpa saling menunggu.
	//
	// Ia hidup di basis data PORTAL UTAMA, bukan per entitas: HRD dan master pengguna
	// asuransi adalah data lingkup identitas, sama seperti M_LOGIN_PNC dan M_PORTAL_PNC.
	komunikasiCabangBranch inboxkomunikasicabang.BranchResolver

	// inboxProgressClaimSelector memilih penyimpanan progres klaim milik satu portal.
	//
	// Ia fungsi dengan alasan yang sama: progres klaim satu badan hukum bukan progres
	// badan hukum lain, dan barisnya memuat nama tertanggung (`ADR-0030`, `R-20`).
	inboxProgressClaimSelector inboxprogressclaim.RepoSelector

	// inboxAnalystDoctorSelector memilih penyimpanan antrean penilaian medis milik satu
	// portal.
	//
	// Alasannya sama dengan selector di atasnya, ditambah satu yang lebih berat: barisnya
	// adalah klaim Personal Accident, dan `FR-R2` memperlakukan data medis secara khusus.
	// Jatuh ke koneksi bawaan di sini bukan sekadar menampilkan entitas yang salah — ia
	// menampilkan data medis entitas yang salah.
	inboxAnalystDoctorSelector inboxanalystdoctor.RepoSelector

	// claimReportSelector memilih penyimpanan berkas laporan klaim milik satu portal.
	//
	// Alasannya sama dengan progressStatusSelector di bawah, ditambah satu yang khas
	// modul ini: ia membaca DUA tabel sekaligus — tabel warisan Pega dan tabel milik
	// aplikasi ini — dan keduanya hidup di basis data entitas yang sama.
	claimReportSelector inboxlaporanklaim.RepoSelector

	// outstandingSelector memilih penyimpanan klaim milik satu portal.
	outstandingSelector inboxoutstanding.RepoSelector

	// outstandingLines membaca M_LOGIN_PNC.LINEBUSINESS, pengganti OperatorID.pyPosition.
	outstandingLines inboxoutstanding.LineBusinessRepo
	// claimReportBranch menerjemahkan login petugas menjadi kode cabang klaimnya.
	//
	// Ia TIDAK diambil dari profil HCC/HCQ: kode cabang yang dipakai layar itu adalah
	// POOLDATA.BRANCH.ID, diturunkan lewat HRD dan master pengguna asuransi — sistem kode
	// yang berbeda dari Placement.BranchCode. Lihat inboxlaporanklaim.BranchResolver.
	//
	// Ia hidup di basis data PORTAL UTAMA, bukan per entitas: HRD dan master pengguna
	// adalah data lingkup identitas, sama seperti M_LOGIN_PNC dan M_PORTAL_PNC.
	claimReportBranch inboxlaporanklaim.BranchResolver

	// closeClaimSelector memilih penyimpanan klaim TUTUP milik satu portal.
	closeClaimSelector inboxcloseclaim.RepoSelector

	// closeClaimRequests memilih penyimpanan PERMINTAAN ReOpen dan Copy Klaim milik satu
	// portal.
	//
	// Ia terpisah dari closeClaimSelector meski keduanya melayani satu layar, dan
	// pembelahannya mengikuti kepemilikan tabel: yang pertama membaca tabel milik Pega,
	// yang kedua menulis tabel milik aplikasi ini sendiri (`P-1`).
	closeClaimRequests inboxcloseclaim.RequestRepoSelector

	// progressStatusSelector memilih penyimpanan master status progres milik satu portal.
	//
	// Ia fungsi, bukan repo tunggal, karena tabelnya ada di basis data SETIAP entitas
	// (ADR-0030). Satu repo bersama akan menulis data seluruh entitas ke satu tempat,
	// kebocoran lintas badan hukum yang justru dicegah R-20.
	progressStatusSelector masterstatusprogres.RepoSelector

	// surveyorTypeSelector memilih penyimpanan master tipe surveyor milik satu portal,
	// dengan alasan yang sama persis.
	surveyorTypeSelector mastertipesurveyors.RepoSelector

	// surveyorSelector memilih penyimpanan Master Surveyors — daftar ORANGNYA, bukan
	// tipenya — milik satu portal, dengan alasan yang sama persis.
	surveyorSelector mastersurveyors.RepoSelector

	// picTeknikSelector memilih penyimpanan master PIC teknik milik satu portal, dengan
	// alasan yang sama persis.
	picTeknikSelector masterpicteknik.RepoSelector

	// dominantFactorSelector memilih penyimpanan master faktor dominan milik satu portal,
	// dengan alasan yang sama persis.
	dominantFactorSelector masterdominanfactor.RepoSelector

	// xolSelector memilih penyimpanan Master XOL milik satu portal, dengan alasan yang
	// sama persis.
	xolSelector masterxol.RepoSelector

	// causeOfLossSelector memilih penyimpanan master penyebab kerugian milik satu portal,
	// dengan alasan yang sama persis.
	causeOfLossSelector masterpenyebabkerugian.RepoSelector

	// recoverySelector memilih penyimpanan Master Recovery milik satu portal, dengan
	// alasan yang sama persis.
	recoverySelector masterrecovery.RepoSelector

	// maskingSelector memilih penyimpanan Master Masking milik satu portal, dengan alasan
	// yang sama persis — dan di modul ini akibat kelalaiannya yang paling berat.
	maskingSelector mastermasking.RepoSelector

	// menu dibaca dari basis data portal UTAMA, sama seperti M_LOGIN_PNC dan
	// M_PORTAL_PNC: peta menu dan kewenangan pemakainya adalah data lingkup
	// identitas, bukan data bisnis milik satu badan hukum.
	menu menu.Repo

	// autoClaimSelector memilih penyimpanan Inbox Auto Claim milik satu portal.
	//
	// Alasannya sama dengan progressStatusSelector: tabelnya ada di basis data SETIAP
	// entitas, dan satu repo bersama akan menulis data seluruh entitas ke satu tempat —
	// kebocoran lintas badan hukum yang justru dicegah R-20.
	autoClaimSelector inboxautoclaim.RepoSelector

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
		RepoSelector: store.claimStatusSelector,
	})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	claimReportService, err := inboxlaporanklaimusecase.NewService(inboxlaporanklaimusecase.Options{
		RepoSelector:   store.claimReportSelector,
		BranchResolver: store.claimReportBranch,
		Clock:          clock.System{},
	})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	maskingService, err := mastermaskingusecase.NewService(mastermaskingusecase.Options{
		RepoSelector: store.maskingSelector,
		// Waktu datang dari jam yang sama dengan modul lain, bukan dari SYSDATE basis data
		// seperti procedure lama. `docs/Steering/07` §4.4 menetapkan konversi dan sumber
		// waktu berada di satu tempat; itulah yang menutup `R-12`.
		Now: clock.System{}.Now,
	})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	xolService, err := masterxolusecase.NewService(masterxolusecase.Options{
		RepoSelector: store.xolSelector,
		Notifier:     buildXOLNotifier(cfg, logger),
		Logger:       logger,
	})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	dominantFactorService, err := masterdominanfactorusecase.NewService(masterdominanfactorusecase.Options{
		RepoSelector: store.dominantFactorSelector,
	})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	causeOfLossService, err := masterpenyebabkerugianusecase.NewService(masterpenyebabkerugianusecase.Options{
		RepoSelector: store.causeOfLossSelector,
	})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	surveyorTypeService, err := mastertipesurveyorsusecase.NewService(mastertipesurveyorsusecase.Options{
		RepoSelector: store.surveyorTypeSelector,
	})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	// Master Surveyors — daftar ORANGNYA, anak dari master tipe di atas.
	//
	// Dua seam-nya dirakit di sini karena keduanya menyangkut keputusan yang bukan milik
	// modul master:
	//
	//   - Accounts: pembuatan akun aplikasi surveyor. Sistem lama membuat operator Pega
	//     lewat GCNMCreateOperator; `P-1` melarang Go menulis tabel operator milik Pega
	//     selama masa paralel, dan `F-3` yang memiliki identitas di sistem baru masih
	//     terhalang kontrak HCC/HCQ (`R-14`). Pengisi seam di tahap ini MENCATAT
	//     permintaannya tanpa membuat akun — dan itu keputusan yang dicatat, bukan
	//     kelalaian. Lihat paket mastersurveyors/account.
	//   - Committee: penetapan komite penentu, yang di sistem lama dibaca dari
	//     POOLDATA.EMAILKOMITE — master milik `B-7`, bukan milik modul ini. Kueri mana
	//     persisnya yang dipakai jalur surveyor tidak dapat dibaca dari export (`R-16`),
	//     sehingga yang dipasang sekarang adalah penetapan tetap.
	surveyorService, err := mastersurveyorsusecase.NewService(mastersurveyorsusecase.Options{
		RepoSelector: store.surveyorSelector,
		Committee:    mastersurveyorscommittee.Fixed{},
		Accounts:     mastersurveyorsaccount.NewRecorder(logger),
		Clock:        clock.System{},
	})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	employeeDirectory, err := buildEmployeeDirectory(cfg, store.legacy, logger)
	if err != nil {
		store.close()
		return assembly{}, err
	}

	picTeknikService, err := masterpicteknikusecase.NewService(masterpicteknikusecase.Options{
		RepoSelector: store.picTeknikSelector,
		Directory:    employeeDirectory,
	})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	recoveryIssuer, err := buildVirtualAccountIssuer(cfg, store.legacy, logger)
	if err != nil {
		store.close()
		return assembly{}, err
	}

	recoveryService, err := masterrecoveryusecase.NewService(masterrecoveryusecase.Options{
		RepoSelector: store.recoverySelector,
		Issuer:       recoveryIssuer,
	})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	autoClaimService, err := inboxautoclaimusecase.NewService(inboxautoclaimusecase.Options{
		RepoSelector: store.autoClaimSelector,
	})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	inboxXOLService, err := inboxxolusecase.NewService(inboxxolusecase.Options{
		RepoSelector: store.inboxXOLSelector,
	})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	claimTreatyPropService, err := inboxclaimtreatypropusecase.NewService(
		inboxclaimtreatypropusecase.Options{
			RepoSelector: store.claimTreatyPropSelector,

			// Logger diberikan supaya pembukaan antrean tanpa penyaring kepemilikan
			// ("See All Claim") tercatat. Sampai pemeriksaan peran ada (`TKT-F3-005`),
			// jejak di log adalah satu-satunya hal yang menyatakan siapa memakainya.
			Logger: logger,
		})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	claimTreatyNonPropService, err := inboxclaimtreatynonpropusecase.NewService(
		inboxclaimtreatynonpropusecase.Options{
			RepoSelector: store.claimTreatyNonPropSelector,

			// Alasan yang sama dengan layar Prop, ditambah satu yang khas modul ini:
			// ekspor berkas memakai penyaring yang sama, sehingga satu unduhan dengan
			// "See All Claim" mengeluarkan nama tertanggung seluruh petugas ke berkas
			// yang tersimpan di perangkat pengguna. Jejaknya di log adalah satu-satunya
			// hal yang menyatakan itu terjadi.
			Logger: logger,
		})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	managerReceivePUCLService, err := inboxmanagerreceivepuclusecase.NewService(
		inboxmanagerreceivepuclusecase.Options{
			RepoSelector: store.managerReceivePUCLSelector,

			// Logger di sini WAJIB, bukan pelengkap. Modul lain mencatat hanya saat
			// penyaring kepemilikan dilepas; di modul ini penyaring itu memang tidak
			// pernah ada — layarnya pandangan penyelia, dan SETIAP pembukaannya dicatat.
			//
			// Sampai pemeriksaan peran ada (`TKT-F3-004`), jejak itulah satu-satunya
			// kontrol yang menyatakan siapa membuka antrean seluruh petugas (`D-59`).
			Logger: logger,
		})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	rclPUCLService, err := inboxrclpuclusecase.NewService(
		inboxrclpuclusecase.Options{
			RepoSelector: store.rclPUCLSelector,

			// Logger WAJIB, dengan alasan yang sama seperti modul di atasnya DITAMBAH
			// satu: antrean layar ini bersama, sehingga tidak ada penyaring kepemilikan
			// sama sekali — dan berkas laporan hariannya dapat diunduh dengan rentang
			// tanggal yang ditentukan penggunanya sendiri. Rentang yang lebar adalah hal
			// yang harus dapat ditelusuri setelahnya (`D-59`).
			Logger: logger,
		})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	komunikasiCabangService, err := inboxkomunikasicabangusecase.NewService(
		inboxkomunikasicabangusecase.Options{
			RepoSelector: store.komunikasiCabangSelector,

			// BranchResolver WAJIB — dan modul ini menolak dibentuk tanpanya.
			//
			// Tanpa penerjemah, batas data layar ini tidak dapat ditentukan sama sekali,
			// dan satu-satunya jalan yang tersisa adalah menampilkan percakapan siapa saja.
			// Membiarkannya nil lalu "menanganinya nanti" adalah persis cara batas data
			// menghilang tanpa ada yang menyadarinya.
			BranchResolver: store.komunikasiCabangBranch,

			// Logger WAJIB, dengan satu alasan tambahan yang khas layar ini: petugas yang
			// kode cabangnya TIDAK terbaca dilayani sebagai kantor pusat (`P-5`), dan itu
			// pelebaran batas data yang tidak menghasilkan satu pun galat. Jejaknya adalah
			// satu-satunya hal yang dapat menjawab "siapa saja yang terkena" bila keputusan
			// itu kelak ditinjau ulang (`D-59`).
			Logger: logger,
		})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	// Logger disuntikkan dengan alasan yang mirip, tetapi ambangnya berbeda: yang diawasi
	// di sini adalah rekap per PIC, satu-satunya bagian layar ini yang TIDAK dipaginasi —
	// mengikuti sistem lama yang juga tidak memaginasinya.
	inboxProgressClaimService, err := inboxprogressclaimusecase.NewService(
		inboxprogressclaimusecase.Options{
			RepoSelector: store.inboxProgressClaimSelector,
			Clock:        clock.System{},
			Logger:       logger,
		})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	// Inbox Analyst Doctor tidak menerima Clock di sini: waktu hanya dibutuhkan saat
	// menyusun jawaban — kolom "Lama Waktu Klaim" — bukan saat mengambil antreannya.
	// Menaruhnya di usecase akan menambah ketergantungan yang tidak dipakai satu baris pun.
	inboxAnalystDoctorService, err := inboxanalystdoctorusecase.NewService(
		inboxanalystdoctorusecase.Options{
			RepoSelector: store.inboxAnalystDoctorSelector,
		})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	outstandingService, err := inboxoutstandingusecase.NewService(
		store.outstandingSelector,
		store.outstandingLines,
	)
	if err != nil {
		store.close()
		return assembly{}, err
	}

	// Pembangkit pengenal dan jam dipasok di sini, bukan dibaca modul dari jam sistem.
	//
	// Keduanya seam supaya waktu permintaan dapat diuji secara deterministik — dan supaya
	// tidak ada satu pun tempat di dalam modul yang memanggil time.Now() sendiri, yang
	// persis pola `Set7Hours` sistem lama yang menambah tujuh jam manual di 118 titik.
	closeClaimService, err := inboxcloseclaimusecase.NewService(inboxcloseclaimusecase.Options{
		Claims:   store.closeClaimSelector,
		Requests: store.closeClaimRequests,
		IDs:      inboxcloseclaimmemory.IDGenerator{},
		Clock:    clock.System{},
	})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	// Policy tidak dipasok: DefaultPolicy dipakai — mode kumulatif, dan hanya
	// Non-MBU yang memakai pita dengan batas Rp 100.000.000 (`D-52`, `D-70`).
	//
	// Ia BELUM bergantung pada portal yang sedang melayani, dan itu batas yang disadari:
	// entitas Simasnet memakai mode satu-penyetuju, dan entitas SMI memakai batas pita
	// USD 7.000 — keduanya ada di rule yang sama (`Activity/SetEmailKomite-Act.xml`).
	// Menyambungkannya ke portal aktif adalah `TKT-F6-002`, yang menuntut portal melekat
	// pada permintaan alih-alih pada keadaan global (`R-20`).
	//
	// Randomizer dipasok sekarang meski portal ASM tidak memakainya: bila kelak portal
	// diganti ke mode satu-penyetuju, pemilihannya langsung acak — bukan diam-diam
	// selalu jatuh ke orang yang sama.
	komiteService, err := komiteusecase.NewService(komiteusecase.Options{
		Repo:       store.komite,
		Randomizer: random.System{},
	})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	// Inbox Komite memakai jam sistem dalam UTC, sama dengan modul lain. Ia dipasok
	// eksplisit — bukan dibiarkan memakai bawaan — supaya jelas terbaca bahwa Aging
	// dihitung dari jam SERVER, bukan jam peramban. Satu kenyataan tidak boleh punya dua
	// umur.
	komiteInboxService, err := komiteusecase.NewInboxService(komiteusecase.InboxOptions{
		Cases:     store.komiteInbox,
		Decisions: store.komiteDecision,
		IDs:       komitememory.IDGenerator{},
		Clock:     clock.System{},
	})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	return assembly{
		auth:                    service,
		portal:                  store.portal,
		masterRekening:          buildMasterRekening(cfg, store, logger),
		masterStatus:            claimStatusService,
		masterStatusProgres:     progressStatusService,
		komite:                  komiteService,
		komiteInbox:             komiteInboxService,
		masterTipeSurveyors:     surveyorTypeService,
		masterSurveyors:         surveyorService,
		masterPicTeknik:         picTeknikService,
		masterRecovery:          recoveryService,
		masterDominanFactor:     dominantFactorService,
		masterXOL:               xolService,
		masterPenyebabKerugian:  causeOfLossService,
		masterMasking:           maskingService,
		menu:                    menuService,
		inboxAutoClaim:          autoClaimService,
		inboxXOL:                inboxXOLService,
		inboxClaimTreatyProp:    claimTreatyPropService,
		inboxClaimTreatyNonProp: claimTreatyNonPropService,
		inboxManagerReceivePUCL: managerReceivePUCLService,
		inboxRCLPUCL:            rclPUCLService,
		inboxKomunikasiCabang:   komunikasiCabangService,
		inboxProgressClaim:      inboxProgressClaimService,
		inboxAnalystDoctor:      inboxAnalystDoctorService,
		inboxLaporanKlaim:       claimReportService,
		inboxOutstanding:        outstandingService,
		inboxCloseClaim:         closeClaimService,
		readyAliases:            store.readyAliases,
		close:                   store.close,
	}, nil
}

// buildMasterRekening menyusun modul Master Rekening di balik seam-nya, SATU LAYANAN PER
// PORTAL.
//
// Seam Kasir diisi klien HTTP nyata bila alamatnya sudah dikonfigurasi, dan tiruan bila
// belum. Perbedaannya diumumkan di log: layar yang tampak bekerja padahal pendaftaran
// ke Kasir tidak pernah terjadi adalah kegagalan yang tidak terlihat siapa pun sampai
// pembayaran pertama tertahan.
//
// # Kenapa satu layanan per portal (2026-09-19)
//
// Seam Kasir, seam Notifier, dan jam sama untuk seluruh entitas — ketiganya konfigurasi
// tingkat aplikasi. Yang BERBEDA per entitas ada dua, dan keduanya menentukan hasil:
//
//   - penyimpanannya, karena POOLDATA.LST_ACCOUNT ada di basis data setiap entitas;
//   - `PortalAlias`, yang menentukan apakah rekening yang disetujui DIDAFTARKAN KE KASIR
//     (`portalsRegisteredWithCashier` di usecase/decide.go).
//
// Sebelum penyelarasan ini, alias yang dipakai selalu portal UTAMA — sehingga keputusan
// "daftarkan ke Kasir atau tidak" dijawab dengan entitas yang salah bagi setiap pengguna
// yang sedang melihat entitas lain. Sekarang ia dijawab dengan entitas yang benar-benar
// dipilih.
//
// Layanannya dibuat saat pertama diminta lalu dipakai kembali; membuatnya ulang setiap
// permintaan berarti membuang seluruh keadaan seam-nya tanpa alasan.
func buildMasterRekening(cfg config.Config, store storage, logger *slog.Logger) func(string) (*masterrekeningusecase.Service, error) {
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

	var lock sync.Mutex
	cache := map[string]*masterrekeningusecase.Service{}

	return func(alias string) (*masterrekeningusecase.Service, error) {
		clean := strings.ToUpper(strings.TrimSpace(alias))

		lock.Lock()
		defer lock.Unlock()
		if existing, already := cache[clean]; already {
			return existing, nil
		}

		accountRepo, bankRepo, err := store.accountSelector(clean)
		if err != nil {
			return nil, err
		}

		fresh := masterrekeningusecase.NewService(masterrekeningusecase.Options{
			Repo:     accountRepo,
			Bank:     bankRepo,
			Cashier:  cashierSystem,
			Notifier: notifier,
			Clock:    clock.System{},
			// Alias entitas YANG DIMINTA, bukan portal utama. Inilah yang membuat
			// keputusan pendaftaran ke Kasir dijawab dengan entitas yang benar.
			PortalAlias: clean,
		})
		cache[clean] = fresh
		return fresh, nil
	}
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
		store.accountSelector = func(alias string) (masterrekening.Repo, masterrekening.BankRepo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, nil, err
			}
			return masterrekeningsql.NewRepo(conn), masterrekeningsql.NewBankRepo(conn), nil
		}
		store.accountInOracle = true
		store.readyAliases = pool.Available
		store.menu = menusql.NewRepo(primary)

		// Master ambang komite dipasang pada koneksi UTAMA, dan itu CACAT YANG DISADARI —
		// bukan sekadar sementara.
		//
		// Work Owner menegaskan 2026-09-19 bahwa setiap server punya POOLDATA-nya sendiri,
		// dan isi EMAILKOMITE BERBEDA antar server: baris SIMASNET hanya ada di POOLDATA
		// server Simasnet. Selama repo ini terpasang pada koneksi utama, portal mana pun
		// yang dipilih pengguna akan membaca tangga ambang milik portal UTAMA.
		//
		// Akibatnya bukan galat melainkan angka yang salah tanpa tanda: layar menampilkan
		// jenjang persetujuan entitas lain, dan tidak ada yang terlihat keliru. Itu kelas
		// kegagalan yang sama dengan `R-20`.
		//
		// Hari ini belum menimbulkan kerugian karena hanya portal utama yang dilayani.
		// Memperbaikinya adalah `TKT-F6-002` — koneksi diambil dari portal AKTIF, yang
		// menuntut portal melekat pada permintaan alih-alih pada keadaan global.
		store.komite = komitesql.NewRepo(primary)

		// Inbox Komite membaca tabel WARISAN — `DATAPEGA.PC_ASM_FW_GCNMFW_WORK`,
		// `PC_ASSIGN_WORKLIST`, `T_CLAIM_KOMITE_LIST`, `T_CLAIM_DATA_RESULTS_AI` — dan
		// menulis keputusannya ke tabel MILIK APLIKASI INI (`CPNC_KOMITE_KEPUTUSAN`,
		// migrasi 0004). Keduanya dipasang pada koneksi yang sama.
		//
		// Cacat portalnya SAMA PERSIS dengan master ambang di atas, dan di sini akibatnya
		// lebih berat: yang salah portal bukan angka acuan melainkan DAFTAR PEKERJAAN
		// beserta nilai klaim dan nama tertanggung milik badan hukum lain. Itu `R-20`
		// secara harfiah, dan penutupannya `TKT-F6-002`.
		store.komiteInbox = komitesql.NewInboxRepo(primary)
		store.komiteDecision = komitesql.NewDecisionRepo(primary)
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

		store.autoClaimSelector = func(alias string) (inboxautoclaim.Repo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return inboxautoclaimsql.NewRepo(conn), nil
		}

		store.claimReportSelector = func(alias string) (inboxlaporanklaim.Repo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return inboxlaporanklaimsql.NewRepo(conn, clock.System{}), nil
		}

		// Klaim dibaca dari basis data entitasnya sendiri, dengan aturan yang sama:
		// portal yang tidak dikenal atau belum siap menghasilkan galat dari For(),
		// TIDAK pernah dialihkan ke koneksi utama.
		store.outstandingSelector = func(alias string) (inboxoutstanding.Repo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return inboxoutstandingsql.NewRepo(conn), nil
		}

		// Lini bisnis dibaca dari portal UTAMA — lihat komentar field-nya.
		//
		// Kolom LINEBUSINESS belum ada sampai migrasi 0004 dijalankan DBA, sehingga
		// pembacaannya gagal di setiap lingkungan hari ini. Kegagalan itu ditangani
		// usecase sebagai "lini tidak diketahui" dan dicatat di log; layar tetap
		// berjalan dengan seluruh lini terlihat, persis perilaku Pega.
		store.outstandingLines = inboxoutstandingsql.NewLineBusinessRepo(primary)

		// Klaim TUTUP dibaca dari basis data entitasnya sendiri, dengan aturan yang sama
		// seperti klaim berjalan di atasnya.
		store.closeClaimSelector = func(alias string) (inboxcloseclaim.Repo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return inboxcloseclaimsql.NewRepo(conn), nil
		}

		// Permintaan ReOpen dan Copy Klaim ditulis ke basis data entitas yang SAMA dengan
		// klaimnya — bukan ke portal utama.
		//
		// Ini bukan pilihan kerapian: permintaan atas klaim milik satu badan hukum yang
		// tercatat di basis data badan hukum lain adalah kebocoran yang persis `R-20`
		// larang, dan pada modul ini akibatnya melampaui tampilan.
		//
		// Tabelnya dibuat migrasi `0006`, yang BELUM dijalankan DBA di lingkungan mana pun.
		// Sampai itu terjadi, pembacaannya gagal dan usecase menanganinya sebagai
		// "permintaan tidak dapat dibaca" — daftarnya tetap tampil, penandanya tidak muncul,
		// dan kegagalannya dicatat di log.
		store.closeClaimRequests = func(alias string) (inboxcloseclaim.RequestRepo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return inboxcloseclaimsql.NewRequestRepo(conn), nil
		}

		store.surveyorTypeSelector = func(alias string) (mastertipesurveyors.Repo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return mastertipesurveyorssql.NewRepo(conn), nil
		}
		store.surveyorSelector = func(alias string) (mastersurveyors.Repo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return mastersurveyorssql.NewRepo(conn), nil
		}
		store.claimStatusSelector = func(alias string) (masterstatus.Repo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return masterstatussql.NewRepo(conn), nil
		}
		store.maskingSelector = func(alias string) (mastermasking.Repo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return mastermaskingsql.NewRepo(conn), nil
		}
		store.picTeknikSelector = func(alias string) (masterpicteknik.Repo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return masterpictekniksql.NewRepo(conn), nil
		}
		store.dominantFactorSelector = func(alias string) (masterdominanfactor.Repo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return masterdominanfactorsql.NewRepo(conn), nil
		}
		store.xolSelector = func(alias string) (masterxol.Repo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return masterxolsql.NewRepo(conn), nil
		}
		store.causeOfLossSelector = func(alias string) (masterpenyebabkerugian.Repo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return masterpenyebabkerugiansql.NewRepo(conn), nil
		}
		store.recoverySelector = func(alias string) (masterrecovery.Repo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return masterrecoverysql.NewRepo(conn), nil
		}

		store.inboxXOLSelector = func(alias string) (inboxxol.Repo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return inboxxolsql.NewRepo(conn), nil
		}

		store.claimTreatyPropSelector = func(alias string) (inboxclaimtreatyprop.Repo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return inboxclaimtreatypropsql.NewRepo(conn), nil
		}

		store.claimTreatyNonPropSelector = func(
			alias string,
		) (inboxclaimtreatynonprop.Repo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return inboxclaimtreatynonpropsql.NewRepo(conn), nil
		}

		store.managerReceivePUCLSelector = func(
			alias string,
		) (inboxmanagerreceivepucl.Repo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return inboxmanagerreceivepuclsql.NewRepo(conn), nil
		}

		store.rclPUCLSelector = func(alias string) (inboxrclpucl.Repo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return inboxrclpuclsql.NewRepo(conn), nil
		}

		store.komunikasiCabangSelector = func(
			alias string,
		) (inboxkomunikasicabang.Repo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return inboxkomunikasicabangsql.NewRepo(conn), nil
		}

		store.inboxProgressClaimSelector = func(alias string) (inboxprogressclaim.Repo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return inboxprogressclaimsql.NewRepo(conn), nil
		}

		store.inboxAnalystDoctorSelector = func(alias string) (inboxanalystdoctor.Repo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return inboxanalystdoctorsql.NewRepo(conn), nil
		}

		store.claimReportBranch = inboxlaporanklaimsql.NewBranchResolver(primary)

		// Penerjemah cabang KEDUA, milik modul Inbox Komunikasi Cabang.
		//
		// Ia membaca objek yang sama lewat kueri yang sama, dan itu BUKAN pengulangan yang
		// terlewat: seam-nya dideklarasikan di paket yang memakainya, sehingga kedua modul
		// dapat berpindah ke API pengganti DB Link (`D-25`) pada waktu yang berbeda.
		//
		// Keduanya menunjuk `primary` dengan alasan yang sama — HRD dan master pengguna
		// asuransi adalah data lingkup identitas, bukan data per entitas.
		store.komunikasiCabangBranch = inboxkomunikasicabangsql.NewBranchResolver(primary)
	} else {
		store.portal = portalmemory.NewRepo(portalmemory.SampleList()...)
		store.accountSelector = accountSelectorMemory(cfg.PrimaryPortal)
		// Ke-33 status nyata ikut dimuat, sehingga layar Master Status Klaim dapat
		// dicoba lengkap tanpa Oracle dan tanpa menunggu migrasi 0002.
		store.readyAliases = func() []string { return []string{cfg.PrimaryPortal} }
		// Isi master ambang yang sebenarnya ikut dimuat, sehingga layar Ambang Komite
		// dan simulasi penjenjangan dapat dicoba lengkap tanpa Oracle — termasuk
		// ketujuh kasus pada spec B-7.
		store.komite = komitememory.NewSampleRepo()
		// Kasus komite contoh ikut dimuat, sehingga layar Inbox Komite dapat dicoba
		// LENGKAP — termasuk alur keputusannya — tanpa Oracle dan tanpa menunggu migrasi
		// 0004. Seluruh isinya KARANGAN, berbeda dari master ambang di atas: tidak ada
		// satu pun ekstrak antrean komite yang pernah diserahkan kepada kami. Lihat
		// repo/memory/inbox_sample.go.
		inboxStore := komitememory.NewSampleInboxStore()
		store.komiteInbox = inboxStore
		store.komiteDecision = inboxStore
		store.progressStatusSelector = progressStatusSelectorMemory(cfg.PrimaryPortal)
		store.autoClaimSelector = autoClaimSelectorMemory(cfg.PrimaryPortal)
		store.claimReportSelector = claimReportSelectorMemory(cfg.PrimaryPortal)

		// memisahkan di produksi adalah KONEKSI basis data yang berbeda.
		//
		// Klaim contohnya mencakup lima Group Panel, sehingga batas data per lini dapat
		// dicoba tanpa Oracle dan tanpa menunggu migrasi 0004. Seluruh isinya karangan —
		// lihat repo/memory/sample.go.
		outstandingMemory := inboxoutstandingmemory.NewRepoWithSamples()
		store.outstandingSelector = func(string) (inboxoutstanding.Repo, error) {
			return outstandingMemory, nil
		}
		store.outstandingLines = outstandingMemory

		// Inbox Close Claim memakai SATU penyimpanan untuk klaim dan permintaannya.
		//
		// Berbeda dari perakitan SQL di atas, yang memisahkan keduanya karena tabelnya
		// dimiliki sistem yang berbeda. Di memori tidak ada kepemilikan tabel, dan
		// menyatukannya justru yang membuat permintaan yang dicatat langsung terlihat pada
		// daftar — persis perilaku yang hendak dicoba tanpa Oracle.
		//
		// Seluruh isinya karangan — lihat repo/memory/sample.go.
		closeClaimMemory := inboxcloseclaimmemory.NewStoreWithSamples()
		store.closeClaimSelector = func(string) (inboxcloseclaim.Repo, error) {
			return closeClaimMemory, nil
		}
		store.closeClaimRequests = func(string) (inboxcloseclaim.RequestRepo, error) {
			return closeClaimMemory, nil
		}
		// Keempat tipe surveyor nyata ikut dimuat, sehingga layar Master Tipe Surveyors
		// dapat dicoba lengkap tanpa Oracle.
		store.surveyorTypeSelector = surveyorTypeSelectorMemory(cfg.PrimaryPortal)
		// Empat contoh surveyor ikut dimuat, satu per posisi persetujuan, sehingga kelima
		// tab layar Master Surveyors dapat dicoba tanpa Oracle DAN tanpa menunggu migrasi
		// 0004 — yang berbeda dari migrasi 0003 bersifat WAJIB.
		store.surveyorSelector = surveyorSelectorMemory(cfg.PrimaryPortal)
		// Ke-33 status nyata ikut dimuat, sehingga layar Master Status Klaim dapat dicoba
		// lengkap tanpa Oracle dan tanpa menunggu migrasi 0002.
		store.claimStatusSelector = claimStatusSelectorMemory(cfg.PrimaryPortal)
		// Sepuluh contoh faktor dominan ikut dimuat — dan TEKS-nya dikarang, bukan isi
		// master yang sebenarnya. Isi aslinya belum pernah diterima dari DBA; lihat
		// peringatan di masterdominanfactor/repo/memory/sample.go.
		store.dominantFactorSelector = dominantFactorSelectorMemory(cfg.PrimaryPortal)
		// Empat contoh Master XOL ikut dimuat, dan bentuknya MENIRU produksi termasuk
		// keanehannya: nomor berlubang, Type XOL kosong, lapisan tanpa isi, dan satu
		// lapisan yang total share-nya nol. Keempatnya nyata — lihat peringatan di
		// masterxol/repo/memory/sample.go.
		store.xolSelector = xolSelectorMemory(cfg.PrimaryPortal)
		// Sepuluh contoh penyebab kerugian ikut dimuat — dan KETERANGANNYA dikarang,
		// bukan isi master yang sebenarnya; hanya bentuk ID-nya yang diturunkan dari
		// bukti. Salah satunya berketerangan KOSONG, supaya keputusan "keterangan kosong
		// diterima" dapat dicoba sungguhan tanpa Oracle. Lihat peringatan di
		// masterpenyebabkerugian/repo/memory/sample.go.
		store.causeOfLossSelector = causeOfLossSelectorMemory(cfg.PrimaryPortal)
		// Empat contoh PIC teknik ikut dimuat — salah satunya NONAKTIF, supaya keputusan
		// "daftar hanya menampilkan yang aktif" dapat dicoba sungguhan tanpa Oracle.
		store.picTeknikSelector = picTeknikSelectorMemory(cfg.PrimaryPortal)
		// Dua principal contoh dan dua acuan polis ikut dimuat, sehingga layar Master
		// Recovery dapat dicoba utuh tanpa Oracle — termasuk penolakan "polis tidak
		// ditemukan" untuk nomor selain keduanya.
		store.recoverySelector = recoverySelectorMemory(cfg.PrimaryPortal)
		// Lima contoh masking ikut dimuat — satu di antaranya NONAKTIF dan satu tanpa sub
		// modul, supaya dua keputusan yang paling mudah salah dapat dicoba sungguhan tanpa
		// Oracle: bahwa "hapus" hanya menonaktifkan, dan bahwa sub modul boleh kosong.
		store.maskingSelector = maskingSelectorMemory(cfg.PrimaryPortal)
		store.claimReportBranch = inboxlaporanklaimmemory.NewBranchResolver(
			inboxlaporanklaimmemory.SampleBranchOfLogin())

		// Pemetaan contohnya SENGAJA berbeda dari milik Inbox Laporan Klaim: di sana
		// `adminpnc` berada di cabang 1001, di sini ia berada di kantor pusat.
		//
		// Perbedaan itu bukan kelalaian. Layar ini punya jalur kantor pusat yang tidak ada
		// di layar itu, dan tanpa saksi yang cabangnya BENAR-BENAR terbaca sebagai kantor
		// pusat, jalur itu hanya dapat dicapai lewat kegagalan penerjemahan — sehingga
		// "petugas pusat" dan "cabang tidak terbaca" tidak akan pernah dapat dibedakan saat
		// pengembangan.
		store.komunikasiCabangBranch = inboxkomunikasicabangmemory.NewSampleBranchResolver()
		// NewDevRepo, bukan NewSampleRepo: isi contoh m_login_group_pnc.csv hanya
		// memuat satu login, dan login provider tiruan tidak ada di dalamnya. Tanpa
		// itu, masuk saat pengembangan menghasilkan menu kosong yang tampak rusak.
		store.menu = menumemory.NewDevRepo()
		store.inboxXOLSelector = inboxXOLSelectorMemory(cfg.PrimaryPortal)
		store.claimTreatyPropSelector = claimTreatyPropSelectorMemory(cfg.PrimaryPortal)
		store.claimTreatyNonPropSelector = claimTreatyNonPropSelectorMemory(cfg.PrimaryPortal)
		// Sepuluh baris contoh ikut dimuat, dan lima di antaranya sengaja TIDAK muncul di
		// tab mana pun — berkas tanpa Group Panel, klaim yang bocor ke tabel penugasan per
		// orang, klaim yang sudah selesai, dan klaim di antrean bersama lain. Tanpa baris
		// yang tertolak, layar pengembangan tidak dapat menunjukkan bahwa penyaringnya
		// benar-benar bekerja.
		store.managerReceivePUCLSelector = managerReceivePUCLSelectorMemory(cfg.PrimaryPortal)
		store.rclPUCLSelector = rclPUCLSelectorMemory(cfg.PrimaryPortal)
		// Sepuluh percakapan contoh ikut dimuat, empat di antaranya SENGAJA tertolak —
		// percakapan yang sudah ditutup, yang tanpa pengirim, yang tanpa pesan, dan yang
		// milik cabang lain. Tanpa keempatnya, penyaring yang hilang tidak akan ketahuan
		// saat pengembangan.
		store.komunikasiCabangSelector = komunikasiCabangSelectorMemory(cfg.PrimaryPortal)
		store.inboxProgressClaimSelector = inboxProgressClaimSelectorMemory(cfg.PrimaryPortal)
		store.inboxAnalystDoctorSelector = inboxAnalystDoctorSelectorMemory(cfg.PrimaryPortal)
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

// autoClaimSelectorMemory menyusun penyimpanan Inbox Auto Claim di memori.
//
// Bentuknya sengaja sama persis dengan progressStatusSelectorMemory, termasuk
// penyimpanan per portal yang dibuat sekali lalu dipakai kembali: unggahan yang baru
// disimpan harus tetap ada pada permintaan berikutnya, dan penyimpanan yang dibuat ulang
// tiap permintaan akan membuat layar tampak kehilangan data tanpa sebab.
//
// Hanya portal utama yang dilayani, sejalan dengan readyAliases pada cabang tanpa Oracle.
// Memilih portal lain karena itu ditolak dengan galat yang sama seperti di produksi —
// perilaku penolakannya ikut teruji saat pengembangan, bukan hanya nanti.
func autoClaimSelectorMemory(primaryAlias string) inboxautoclaim.RepoSelector {
	var lock sync.Mutex
	store := map[string]inboxautoclaim.Repo{}

	return func(alias string) (inboxautoclaim.Repo, error) {
		clean := strings.ToUpper(strings.TrimSpace(alias))
		if clean != strings.ToUpper(strings.TrimSpace(primaryAlias)) {
			return nil, fmt.Errorf("%w: portal %q tidak tersedia tanpa basis data", portal.ErrNotReady, alias)
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := inboxautoclaimmemory.NewSampleRepo()
		store[clean] = fresh
		return fresh, nil
	}
}

// claimReportSelectorMemory menyusun penyimpanan berkas laporan klaim di memori.
//
// Bentuknya sama persis dengan progressStatusSelectorMemory, dan alasannya pun sama:
// satu penyimpanan per portal, dibuat saat pertama diminta lalu dipakai kembali. Kalau
// dibuat ulang setiap permintaan, berkas yang baru dibuat lewat tombol "Buat Baru" akan
// hilang pada permintaan berikutnya dan layarnya tampak rusak tanpa sebab.
func claimReportSelectorMemory(primaryAlias string) inboxlaporanklaim.RepoSelector {
	var lock sync.Mutex
	store := map[string]inboxlaporanklaim.Repo{}
	systemClock := clock.System{}

	return func(alias string) (inboxlaporanklaim.Repo, error) {
		clean := strings.ToUpper(strings.TrimSpace(alias))
		if clean != strings.ToUpper(strings.TrimSpace(primaryAlias)) {
			return nil, fmt.Errorf("%w: portal %q tidak tersedia tanpa basis data", portal.ErrNotReady, alias)
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := inboxlaporanklaimmemory.NewRepo(inboxlaporanklaimmemory.SampleOptions(systemClock))
		store[clean] = fresh
		return fresh, nil
	}
}

// surveyorTypeSelectorMemory menyusun penyimpanan master tipe surveyor di memori.
//
// Bentuknya sama dengan progressStatusSelectorMemory dan alasannya pun sama: satu portal
// mendapat satu penyimpanan, dibuat saat pertama diminta lalu dipakai kembali. Kalau
// dibuat ulang setiap permintaan, penambahan yang baru disimpan akan hilang pada
// permintaan berikutnya dan layarnya tampak rusak tanpa sebab.
//
// Hanya portal utama yang dilayani, sejalan dengan readyAliases pada cabang tanpa Oracle.
// Memilih portal lain tanpa basis data karena itu ditolak dengan galat yang sama seperti
// di produksi — perilaku penolakannya ikut teruji saat pengembangan, bukan hanya nanti.
func surveyorTypeSelectorMemory(primaryAlias string) mastertipesurveyors.RepoSelector {
	var lock sync.Mutex
	store := map[string]mastertipesurveyors.Repo{}

	return func(alias string) (mastertipesurveyors.Repo, error) {
		clean := strings.ToUpper(strings.TrimSpace(alias))
		if clean != strings.ToUpper(strings.TrimSpace(primaryAlias)) {
			return nil, fmt.Errorf("%w: portal %q tidak tersedia tanpa basis data", portal.ErrNotReady, alias)
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := mastertipesurveyorsmemory.NewRepo(mastertipesurveyorsmemory.SampleList()...)
		store[clean] = fresh
		return fresh, nil
	}
}

// surveyorSelectorMemory menyusun penyimpanan Master Surveyors di memori.
//
// Bentuknya sama persis dengan surveyorTypeSelectorMemory, dan kesamaannya disengaja:
// keduanya melayani modul yang datanya hidup per entitas, sehingga keduanya WAJIB menolak
// portal selain portal utama alih-alih diam-diam melayaninya dari satu tempat (`R-20`).
func surveyorSelectorMemory(primaryAlias string) mastersurveyors.RepoSelector {
	var lock sync.Mutex
	store := map[string]mastersurveyors.Repo{}

	return func(alias string) (mastersurveyors.Repo, error) {
		clean := strings.ToUpper(strings.TrimSpace(alias))
		if clean != strings.ToUpper(strings.TrimSpace(primaryAlias)) {
			return nil, fmt.Errorf("%w: portal %q tidak tersedia tanpa basis data", portal.ErrNotReady, alias)
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := mastersurveyorsmemory.NewRepo()
		store[clean] = fresh
		return fresh, nil
	}
}

// buildEmployeeDirectory menyusun seam direktori pegawai milik modul Master PIC Teknik.
//
// # Kenapa pilihannya mengikuti adapter identitas
//
// Keduanya menembak API yang sama dan membaca alamatnya dari baris POOLDATA.GCNM_CONNECT_REST
// yang sama. Menyalakan salah satunya saja akan menghasilkan keadaan yang membingungkan:
// pengguna dapat MASUK lewat HCQ sungguhan, tetapi nama yang dicari layar Master PIC
// Teknik datang dari daftar karangan — atau sebaliknya.
//
// `legacy` adalah pembaca katalog layanan yang dipakai ulang apa adanya dari modul auth.
// Ia disuntikkan sebagai antarmuka sempit `ServiceCatalog` yang dideklarasikan modul
// masterpicteknik sendiri, sehingga kedua modul tetap tidak saling mengimpor — yang tahu
// keduanya hanyalah berkas perakitan ini.
func buildEmployeeDirectory(
	cfg config.Config,
	legacy *sqlstore.Legacy,
	logger *slog.Logger,
) (masterpicteknik.EmployeeDirectory, error) {
	if cfg.IdentityAdapter != config.IdentityAdapterHCQ || legacy == nil {
		// Direktori tiruan. Ia menjaga layar tetap dapat dicoba utuh tanpa Oracle,
		// termasuk jalur penolakan "ID operator tidak terdaftar".
		//
		// Perbedaannya diumumkan di log, bukan dibiarkan senyap: layar yang tampak bekerja
		// padahal nama yang ditemukannya karangan adalah kegagalan yang tidak terlihat
		// siapa pun sampai petugas pertama tersimpan dengan nama yang salah.
		logger.Warn("direktori pegawai memakai daftar tiruan",
			slog.String("modul", "masterpicteknik"),
			slog.String("sebab", "IDENTITAS_ADAPTER bukan hcq, atau koneksi basis data tidak dibuka"),
		)
		return masterpicteknikdirectory.NewFake(masterpicteknikdirectory.SampleEmployees()...), nil
	}

	return masterpicteknikdirectory.New(masterpicteknikdirectory.Options{
		Catalog:  legacy,
		User:     cfg.HCQ.User,
		Password: cfg.HCQ.Password,
		Timeout:  cfg.HCQ.Timeout,
		// Galat "baris tidak terdaftar" milik modul auth diteruskan sebagai nilai, bukan
		// diimpor tipenya: itulah yang membuat modul ini dapat MEMBEDAKAN katalog yang
		// belum diisi dari jaringan yang sedang putus, tanpa bergantung pada modul auth.
		NotRegistered: provider.ErrServiceNotRegistered,
	})
}

// picTeknikSelectorMemory menyusun penyimpanan master PIC teknik di memori.
//
// Bentuknya sama persis dengan surveyorTypeSelectorMemory, dan kesamaannya disengaja:
// hanya portal UTAMA yang dilayani, dan alias lain DITOLAK — bukan diam-diam dialihkan ke
// portal utama. Menjalankan tanpa basis data tidak boleh mengubah aturan pemisahan
// entitas, karena justru di lingkungan itulah pelanggarannya paling mudah lolos (`R-20`).
//
// Instans disimpan per alias supaya perubahan yang disimpan pengguna bertahan selama
// aplikasi hidup; membuat repo baru setiap permintaan akan membuang setiap penyuntingan.
func picTeknikSelectorMemory(primaryAlias string) masterpicteknik.RepoSelector {
	var lock sync.Mutex
	store := map[string]masterpicteknik.Repo{}

	return func(alias string) (masterpicteknik.Repo, error) {
		clean := strings.ToUpper(strings.TrimSpace(alias))
		if clean != strings.ToUpper(strings.TrimSpace(primaryAlias)) {
			return nil, fmt.Errorf("%w: portal %q tidak tersedia tanpa basis data", portal.ErrNotReady, alias)
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := masterpicteknikmemory.NewRepo(masterpicteknikmemory.SampleList()...)
		store[clean] = fresh
		return fresh, nil
	}
}

// recoverySelectorMemory menyusun penyimpanan Master Recovery di memori.
//
// Bentuknya sama persis dengan picTeknikSelectorMemory, dan kesamaannya disengaja: hanya
// portal UTAMA yang dilayani, dan alias lain DITOLAK — bukan diam-diam dialihkan ke portal
// utama. Menjalankan tanpa basis data tidak boleh mengubah aturan pemisahan entitas,
// karena justru di lingkungan itulah pelanggarannya paling mudah lolos (`R-20`).
//
// Instans disimpan per alias supaya batch dan virtual account yang baru dicatat bertahan
// selama aplikasi hidup; membuat repo baru setiap permintaan akan membuat penerbitan VA
// tampak selalu "baru" dan perilaku pakai-ulangnya tidak pernah dapat dicoba.
func recoverySelectorMemory(primaryAlias string) masterrecovery.RepoSelector {
	var lock sync.Mutex
	store := map[string]masterrecovery.Repo{}

	return func(alias string) (masterrecovery.Repo, error) {
		clean := strings.ToUpper(strings.TrimSpace(alias))
		if clean != strings.ToUpper(strings.TrimSpace(primaryAlias)) {
			return nil, fmt.Errorf("%w: portal %q tidak tersedia tanpa basis data", portal.ErrNotReady, alias)
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := masterrecoverymemory.NewSampleRepo()
		store[clean] = fresh
		return fresh, nil
	}
}

// buildVirtualAccountIssuer menyusun seam penerbit rekening virtual milik Master Recovery.
//
// # Kenapa pilihannya mengikuti PENYIMPANAN, bukan adapter identitas
//
// Berbeda dari direktori pegawai, yang mengikuti `IDENTITAS_ADAPTER` karena keduanya
// menembak API yang sama. Penerbit VA menembak layanan yang berbeda, dan alamatnya dibaca
// dari POOLDATA.GCNM_CONNECT_REST — baris `TYPESERVICE='GENERATEDVA'`. Tanpa koneksi
// Oracle, alamat itu tidak dapat dibaca sama sekali, sehingga yang menentukan adalah ada
// atau tidaknya koneksi.
//
// # Kenapa tiruan BUKAN sekadar kenyamanan di sini
//
// Alamat yang terdaftar menunjuk layanan Pega yang MENERBITKAN REKENING SUNGGUHAN.
// Menembaknya dari lingkungan pengembangan meninggalkan rekening nyata yang tidak diminta
// siapa pun, pada sistem yang dipakai orang lain.
//
// Perbedaannya diumumkan di log, bukan dibiarkan senyap: layar yang tampak bekerja padahal
// nomor yang ditampilkannya karangan adalah kegagalan yang tidak terlihat siapa pun sampai
// dana pertama dikirim ke nomor itu.
func buildVirtualAccountIssuer(
	cfg config.Config,
	legacy *sqlstore.Legacy,
	logger *slog.Logger,
) (masterrecovery.VirtualAccountIssuer, error) {
	if legacy == nil {
		logger.Warn("penerbit virtual account memakai nomor tiruan",
			slog.String("modul", "masterrecovery"),
			slog.String("sebab", "koneksi basis data tidak dibuka, sehingga alamat layanan pada POOLDATA.GCNM_CONNECT_REST tidak dapat dibaca"),
		)
		return masterrecoveryva.NewFake(), nil
	}

	return masterrecoveryva.NewPega(masterrecoveryva.Options{
		Catalog: legacy,
		// Kredensial HCQ dipakai ulang sebagai Basic Auth bila terisi. Layanan ini tidak
		// diketahui menuntut autentikasi — 18 dari 21 Connect REST di sistem lama
		// ber-`pyUseAuthentication=false` (`D-73`) — dan bila keduanya kosong, header
		// Authorization tidak dikirim sama sekali.
		User:     cfg.HCQ.User,
		Password: cfg.HCQ.Password,
		// Galat "baris tidak terdaftar" milik modul auth diteruskan sebagai nilai, bukan
		// diimpor tipenya: itulah yang membuat modul ini dapat MEMBEDAKAN katalog yang
		// belum diisi dari jaringan yang sedang putus, tanpa bergantung pada modul auth.
		NotRegistered: provider.ErrServiceNotRegistered,
	})
}

// accountSelectorMemory menyusun penyimpanan master rekening di memori.
//
// Repo dan BankRepo dipilih bersamaan karena keduanya hidup di basis data yang sama;
// memisahkannya akan membuka kemungkinan rekening satu entitas dipasangkan dengan daftar
// bank entitas lain.
func accountSelectorMemory(primaryAlias string) func(string) (masterrekening.Repo, masterrekening.BankRepo, error) {
	var lock sync.Mutex
	type pair struct {
		account masterrekening.Repo
		bank    masterrekening.BankRepo
	}
	store := map[string]pair{}

	return func(alias string) (masterrekening.Repo, masterrekening.BankRepo, error) {
		clean := strings.ToUpper(strings.TrimSpace(alias))
		if clean != strings.ToUpper(strings.TrimSpace(primaryAlias)) {
			return nil, nil, fmt.Errorf("%w: portal %q tidak tersedia tanpa basis data", portal.ErrNotReady, alias)
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing.account, existing.bank, nil
		}
		fresh := pair{
			account: masterrekeningmemory.NewRepo(),
			bank:    masterrekeningmemory.NewBankRepo(masterrekeningmemory.SampleBanks()...),
		}
		store[clean] = fresh
		return fresh.account, fresh.bank, nil
	}
}

// claimStatusSelectorMemory menyusun penyimpanan master status klaim di memori.
//
// Bentuknya sama dengan kedua pemilih memori lainnya, dan alasannya pun sama: satu portal
// mendapat satu penyimpanan, dibuat saat pertama diminta lalu dipakai kembali. Kalau
// dibuat ulang setiap permintaan, perubahan yang baru disimpan akan hilang pada permintaan
// berikutnya dan layarnya tampak rusak tanpa sebab.
func claimStatusSelectorMemory(primaryAlias string) masterstatus.RepoSelector {
	var lock sync.Mutex
	store := map[string]masterstatus.Repo{}

	return func(alias string) (masterstatus.Repo, error) {
		clean := strings.ToUpper(strings.TrimSpace(alias))
		if clean != strings.ToUpper(strings.TrimSpace(primaryAlias)) {
			return nil, fmt.Errorf("%w: portal %q tidak tersedia tanpa basis data", portal.ErrNotReady, alias)
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := masterstatusmemory.NewRepo(masterstatusmemory.SampleList()...)
		store[clean] = fresh
		return fresh, nil
	}
}

// buildXOLNotifier memilih pengisi seam masterxol.Notifier.
//
// Pengirim SMTP dipasang hanya bila konfigurasinya LENGKAP — termasuk daftar penerimanya.
// Bila belum, yang dipasang adalah tiruan yang mencatat, dan penyimpanan Master XOL tetap
// berhasil: datanya sudah tersimpan dan pengajuannya sudah tercatat sebelum pemberitahuan
// dikirim, sehingga menggagalkan permintaan pada titik itu hanya akan membuat pengguna
// menyimpan ulang — dan mengajukan dua kali.
//
// Penerimanya datang dari XOL_PENERIMA_KOMITE, bukan dari kode. Sistem lama menuliskan
// dua alamat perorangan langsung di dalam activity-nya dan menimpa hasil pencariannya
// dengan keduanya; `D-15` melarang pola itu dibawa dan `D-67` melarang akun pribadi.
//
// Tempat yang benar bagi daftar ini kelak adalah master Penerima Notifikasi (`F-4`), yang
// belum dibangun.
func buildXOLNotifier(cfg config.Config, logger *slog.Logger) masterxol.Notifier {
	smtpConfig := masterxolnotif.Config{
		Host:     cfg.SMTP.Host,
		Port:     cfg.SMTP.Port,
		User:     cfg.SMTP.User,
		Password: cfg.SMTP.Password,
		From:     cfg.SMTP.From,
		To:       cfg.SMTP.XOLCommitteeRecipients,
		Timeout:  cfg.SMTP.Timeout,
	}
	if smtpConfig.Complete() {
		return masterxolnotif.NewSender(smtpConfig)
	}

	logger.Warn("pemberitahuan komite Master XOL tidak dikirim: SMTP atau penerimanya belum lengkap",
		slog.String("modul", "masterxol"),
		slog.String("perbaikan", "isi SMTP_HOST, SMTP_PORT, SMTP_DARI, dan XOL_PENERIMA_KOMITE"))
	return masterxolnotif.NewFake()
}

// xolSelectorMemory menyusun penyimpanan Master XOL di memori.
//
// Bentuknya sama dengan pemilih memori lainnya, dan alasannya pun sama: satu portal
// mendapat satu penyimpanan, dibuat saat pertama diminta lalu dipakai kembali. Kalau
// dibuat ulang setiap permintaan, induk yang baru disimpan akan hilang pada permintaan
// berikutnya dan layarnya tampak rusak tanpa sebab.
func xolSelectorMemory(primaryAlias string) masterxol.RepoSelector {
	var lock sync.Mutex
	store := map[string]masterxol.Repo{}

	return func(alias string) (masterxol.Repo, error) {
		clean := strings.ToUpper(strings.TrimSpace(alias))
		if clean != strings.ToUpper(strings.TrimSpace(primaryAlias)) {
			return nil, fmt.Errorf("%w: portal %q tidak tersedia tanpa basis data", portal.ErrNotReady, alias)
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := masterxolmemory.NewSampleRepo()
		store[clean] = fresh
		return fresh, nil
	}
}

// dominantFactorSelectorMemory menyusun penyimpanan master faktor dominan di memori.
//
// Bentuknya sama dengan pemilih memori lainnya, dan alasannya pun sama: satu portal
// mendapat satu penyimpanan, dibuat saat pertama diminta lalu dipakai kembali. Kalau
// dibuat ulang setiap permintaan, perubahan yang baru disimpan akan hilang pada permintaan
// berikutnya dan layarnya tampak rusak tanpa sebab.
func dominantFactorSelectorMemory(primaryAlias string) masterdominanfactor.RepoSelector {
	var lock sync.Mutex
	store := map[string]masterdominanfactor.Repo{}

	return func(alias string) (masterdominanfactor.Repo, error) {
		clean := strings.ToUpper(strings.TrimSpace(alias))
		if clean != strings.ToUpper(strings.TrimSpace(primaryAlias)) {
			return nil, fmt.Errorf("%w: portal %q tidak tersedia tanpa basis data", portal.ErrNotReady, alias)
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := masterdominanfactormemory.NewRepo(masterdominanfactormemory.SampleList()...)
		store[clean] = fresh
		return fresh, nil
	}
}

// causeOfLossSelectorMemory menyusun penyimpanan master penyebab kerugian di memori.
//
// Bentuknya sama dengan pemilih memori lainnya, dan alasannya pun sama: satu portal
// mendapat satu penyimpanan, dibuat saat pertama diminta lalu dipakai kembali. Kalau
// dibuat ulang setiap permintaan, perubahan yang baru disimpan akan hilang pada permintaan
// berikutnya dan layarnya tampak rusak tanpa sebab.
func causeOfLossSelectorMemory(primaryAlias string) masterpenyebabkerugian.RepoSelector {
	var lock sync.Mutex
	store := map[string]masterpenyebabkerugian.Repo{}

	return func(alias string) (masterpenyebabkerugian.Repo, error) {
		clean := strings.ToUpper(strings.TrimSpace(alias))
		if clean != strings.ToUpper(strings.TrimSpace(primaryAlias)) {
			return nil, fmt.Errorf("%w: portal %q tidak tersedia tanpa basis data", portal.ErrNotReady, alias)
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := masterpenyebabkerugianmemory.NewRepo(masterpenyebabkerugianmemory.SampleList()...)
		store[clean] = fresh
		return fresh, nil
	}
}

// maskingSelectorMemory memilih penyimpanan Master Masking di memori.
//
// Hanya portal utama yang dilayani. Portal lain menghasilkan ErrNotReady, bukan diam-diam
// memakai penyimpanan yang sama — tanpa basis data, memakai satu penyimpanan untuk semua
// entitas akan membuat perpindahan portal tampak berhasil padahal datanya itu-itu juga,
// dan justru menyembunyikan kelas kesalahan yang `R-20` peringatkan.
func maskingSelectorMemory(primaryAlias string) mastermasking.RepoSelector {
	var lock sync.Mutex
	store := map[string]mastermasking.Repo{}

	return func(alias string) (mastermasking.Repo, error) {
		clean := strings.ToUpper(strings.TrimSpace(alias))
		if clean != strings.ToUpper(strings.TrimSpace(primaryAlias)) {
			return nil, fmt.Errorf("%w: portal %q tidak tersedia tanpa basis data", portal.ErrNotReady, alias)
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		// Cabang tambahan ikut dimuat supaya menambah data untuk cabang yang BELUM punya
		// baris dapat dicoba. Tanpa itu, satu-satunya cabang yang dapat dipilih adalah yang
		// sudah terpakai — dan setiap penambahan akan ditolak sebagai pasangan ganda.
		fresh := mastermaskingmemory.NewRepo(mastermaskingmemory.SampleList()...).
			WithBranches(mastermaskingmemory.SampleBranches()...)
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

// inboxXOLSelectorMemory menyusun penyimpanan Inbox XOL di memori.
//
// Satu portal mendapat satu penyimpanan, dibuat saat pertama diminta lalu dipakai
// kembali — alasannya sama dengan selector memori lain di berkas ini.
//
// Isinya contoh yang mencakup SELURUH jalur layar, termasuk dua yang paling mudah
// terlewat: perjanjian tanpa group business, dan baris treaty inward yang kursnya tidak
// ditemukan. Seluruhnya karangan — lihat inboxxol/repo/memory/sample.go.
//
// Hanya portal utama yang dilayani, sejalan dengan readyAliases pada cabang tanpa Oracle.
// Memilih portal lain tanpa basis data karena itu ditolak dengan galat yang sama seperti
// di produksi.
func inboxXOLSelectorMemory(primaryAlias string) inboxxol.RepoSelector {
	var lock sync.Mutex
	store := map[string]inboxxol.Repo{}

	return func(alias string) (inboxxol.Repo, error) {
		clean, err := matchPrimaryPortal(alias, primaryAlias)
		if err != nil {
			return nil, err
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := inboxxolmemory.NewSampleRepo()
		store[clean] = fresh
		return fresh, nil
	}
}

// claimTreatyPropSelectorMemory menyusun penyimpanan Inbox Claim Treaty Prop di memori.
//
// Satu portal mendapat satu penyimpanan, dibuat saat pertama diminta lalu dipakai
// kembali — alasannya sama dengan selector memori lain di berkas ini.
//
// Isinya contoh yang mencakup ketiga penyaring sekaligus: penugasan milik dua petugas
// berbeda, antrean teknik, satu baris tanpa penanda `CLMP`, dan satu baris di antrean
// lain. Seluruhnya karangan — lihat inboxclaimtreatyprop/repo/memory/sample.go.
//
// Hanya portal utama yang dilayani, sejalan dengan readyAliases pada cabang tanpa Oracle.
func claimTreatyPropSelectorMemory(primaryAlias string) inboxclaimtreatyprop.RepoSelector {
	var lock sync.Mutex
	store := map[string]inboxclaimtreatyprop.Repo{}

	return func(alias string) (inboxclaimtreatyprop.Repo, error) {
		clean, err := matchPrimaryPortal(alias, primaryAlias)
		if err != nil {
			return nil, err
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := inboxclaimtreatypropmemory.NewSampleStore()
		store[clean] = fresh
		return fresh, nil
	}
}

// claimTreatyNonPropSelectorMemory menyusun penyimpanan Inbox Claim Treaty Non Prop di
// memori; alasannya sama dengan claimTreatyPropSelectorMemory di atas.
//
// Isi contohnya mencakup KEEMPAT penyaring layar ini sekaligus: penugasan milik dua petugas
// berbeda, antrean teknik, satu baris yang nomor polisnya belum terbit, satu baris
// ber-awalan `CLMP-` milik layar saudaranya, dan satu baris ber-awalan `KMTNP-`. Dua yang
// terakhir yang membuktikan penyaring awalan tidak mencampur kedua layar treaty — lihat
// inboxclaimtreatynonprop/repo/memory/sample.go.
//
// Hanya portal utama yang dilayani, sejalan dengan readyAliases pada cabang tanpa Oracle.
func claimTreatyNonPropSelectorMemory(
	primaryAlias string,
) inboxclaimtreatynonprop.RepoSelector {
	var lock sync.Mutex
	store := map[string]inboxclaimtreatynonprop.Repo{}

	return func(alias string) (inboxclaimtreatynonprop.Repo, error) {
		clean, err := matchPrimaryPortal(alias, primaryAlias)
		if err != nil {
			return nil, err
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := inboxclaimtreatynonpropmemory.NewSampleStore()
		store[clean] = fresh
		return fresh, nil
	}
}

// managerReceivePUCLSelectorMemory menyusun penyimpanan Inbox Manager Receive / PUCL di
// memori; alasannya sama dengan claimTreatyPropSelectorMemory di atas.
//
// Isi contohnya mencakup KEEMPAT penyaring layar ini sekaligus, dan lima dari sepuluh
// barisnya sengaja TERTOLAK: berkas tanpa Group Panel, klaim yang berada di tabel penugasan
// per orang, klaim yang sudah selesai, dan klaim di antrean bersama lain. Baris yang lolos
// saja tidak membuktikan apa pun — yang membuktikan penyaringnya bekerja adalah baris yang
// seharusnya tidak muncul dan memang tidak muncul. Lihat
// inboxmanagerreceivepucl/repo/memory/sample.go.
//
// Hanya portal utama yang dilayani, sejalan dengan readyAliases pada cabang tanpa Oracle.
func managerReceivePUCLSelectorMemory(
	primaryAlias string,
) inboxmanagerreceivepucl.RepoSelector {
	var lock sync.Mutex
	store := map[string]inboxmanagerreceivepucl.Repo{}

	return func(alias string) (inboxmanagerreceivepucl.Repo, error) {
		clean, err := matchPrimaryPortal(alias, primaryAlias)
		if err != nil {
			return nil, err
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := inboxmanagerreceivepuclmemory.NewSampleStore()
		store[clean] = fresh
		return fresh, nil
	}
}

// rclPUCLSelectorMemory menyusun penyimpanan Inbox RCL/PUCL di memori; alasannya sama
// dengan claimTreatyPropSelectorMemory di atas.
//
// Isi contohnya mencakup KEENAM penyaring layar ini, dan enam dari sebelas barisnya sengaja
// TERTOLAK: penanda kasus yang berbeda, klaim yang sudah disetujui, klaim yang penanda
// persetujuannya KOSONG, klaim yang sudah selesai, klaim di antrean bersama lain, dan klaim
// Personal Accident di luar antrean. Baris yang lolos saja tidak membuktikan apa pun — yang
// membuktikan penyaringnya bekerja adalah baris yang seharusnya tidak muncul dan memang
// tidak muncul.
//
// Baris terakhir punya tugas tambahan: ia TIDAK muncul di tab mana pun tetapi IKUT di
// laporan harian, dan itulah satu-satunya hal yang membuktikan laporan dan tabel memang
// berbeda isinya. Lihat inboxrclpucl/repo/memory/sample.go.
//
// Hanya portal utama yang dilayani, sejalan dengan readyAliases pada cabang tanpa Oracle.
func rclPUCLSelectorMemory(primaryAlias string) inboxrclpucl.RepoSelector {
	var lock sync.Mutex
	store := map[string]inboxrclpucl.Repo{}

	return func(alias string) (inboxrclpucl.Repo, error) {
		clean, err := matchPrimaryPortal(alias, primaryAlias)
		if err != nil {
			return nil, err
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := inboxrclpuclmemory.NewSampleStore()
		store[clean] = fresh
		return fresh, nil
	}
}

// komunikasiCabangSelectorMemory menyusun penyimpanan percakapan cabang di memori.
//
// Sepuluh baris contohnya memegang satu janji: SETIAP penyaring punya baris yang cocok
// MAUPUN yang tidak. Yang membuktikan penyaringnya bekerja bukan baris yang muncul,
// melainkan baris yang seharusnya tidak muncul dan memang tidak muncul — di sini ada empat,
// masing-masing untuk kanal percakapan, pengirim kosong, pesan kosong, dan batas cabang.
//
// Satu baris punya tugas tambahan: KOM-0006 dibalas TANPA penjawab tercatat, sehingga ia
// muncul di tabel tetapi tidak terhitung di pencacah mana pun. Itulah satu-satunya hal yang
// membuktikan selisih satu kolom antara grid dan pencacah benar-benar ditiru.
//
// Hanya portal utama yang dilayani, sejalan dengan readyAliases pada cabang tanpa Oracle.
func komunikasiCabangSelectorMemory(primaryAlias string) inboxkomunikasicabang.RepoSelector {
	var lock sync.Mutex
	store := map[string]inboxkomunikasicabang.Repo{}

	return func(alias string) (inboxkomunikasicabang.Repo, error) {
		clean, err := matchPrimaryPortal(alias, primaryAlias)
		if err != nil {
			return nil, err
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := inboxkomunikasicabangmemory.NewSampleStore()
		store[clean] = fresh
		return fresh, nil
	}
}

// inboxProgressClaimSelectorMemory menyusun penyimpanan progres klaim di memori;
// alasannya sama dengan claimTreatyPropSelectorMemory di atas.
//
// Hanya portal utama yang dilayani, sejalan dengan readyAliases pada cabang tanpa Oracle.
// Memilih portal lain tanpa basis data karena itu ditolak dengan galat yang sama seperti di
// produksi: perilaku penolakannya ikut teruji saat pengembangan, bukan hanya nanti.
func inboxProgressClaimSelectorMemory(primaryAlias string) inboxprogressclaim.RepoSelector {
	var lock sync.Mutex
	store := map[string]inboxprogressclaim.Repo{}

	return func(alias string) (inboxprogressclaim.Repo, error) {
		clean, err := matchPrimaryPortal(alias, primaryAlias)
		if err != nil {
			return nil, err
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := inboxprogressclaimmemory.NewSampleStore()
		store[clean] = fresh
		return fresh, nil
	}
}

// inboxAnalystDoctorSelectorMemory menyusun penyimpanan antrean penilaian medis di memori;
// alasannya sama dengan inboxProgressClaimSelectorMemory di atas.
//
// Hanya portal utama yang dilayani, sejalan dengan readyAliases pada cabang tanpa Oracle.
// Memilih portal lain tanpa basis data karena itu ditolak dengan galat yang sama seperti di
// produksi: perilaku penolakannya ikut teruji saat pengembangan, bukan hanya nanti.
//
// Satu salinan per portal, bukan satu yang dibagi. Modul ini memang tidak menulis, sehingga
// hari ini tidak ada yang dapat saling menimpa — tetapi berbagi penyimpanan antarportal
// adalah bentuk kebocoran yang persis dilarang `R-20`, dan mencegahnya sejak awal jauh lebih
// murah daripada menemukannya kelak.
func inboxAnalystDoctorSelectorMemory(primaryAlias string) inboxanalystdoctor.RepoSelector {
	var lock sync.Mutex
	store := map[string]inboxanalystdoctor.Repo{}

	return func(alias string) (inboxanalystdoctor.Repo, error) {
		clean, err := matchPrimaryPortal(alias, primaryAlias)
		if err != nil {
			return nil, err
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := inboxanalystdoctormemory.NewSampleStore()
		store[clean] = fresh
		return fresh, nil
	}
}

// matchPrimaryPortal menyeragamkan alias dan menolak portal selain portal utama.
//
// Penolakannya memakai portal.ErrNotReady, galat yang sama dengan yang dihasilkan
// produksi saat kredensial sebuah entitas belum diisi — sehingga jalur penolakannya
// berperilaku sama di kedua lingkungan.
func matchPrimaryPortal(alias, primaryAlias string) (string, error) {
	clean := strings.ToUpper(strings.TrimSpace(alias))
	if clean != strings.ToUpper(strings.TrimSpace(primaryAlias)) {
		return "", fmt.Errorf("%w: portal %q tidak tersedia tanpa basis data", portal.ErrNotReady, alias)
	}
	return clean, nil
}

// spaVersionText menyebut kapan antarmuka tersemat dibangun, dalam bentuk yang aman
// ditampilkan meski penandanya tidak ada.
//
// Binary yang dikompilasi sebelum penanda ini diperkenalkan tetap dapat berjalan; yang
// hilang hanyalah kemampuan menjawab "antarmuka versi mana yang sedang disajikan".
func spaVersionText() string {
	if v := spa.Version(); v != "" {
		return v
	}
	return "tidak diketahui (dibangun sebelum penanda versi ada)"
}

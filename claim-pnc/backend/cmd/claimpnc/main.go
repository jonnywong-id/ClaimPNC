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
	"claim-pnc/internal/inboxacceptopenprotection"
	"claim-pnc/internal/inboxautoclaim"
	"claim-pnc/internal/inboxclaimtreatynonprop"
	"claim-pnc/internal/inboxclaimtreatyprop"
	"claim-pnc/internal/inboxlaporanklaim"
	"claim-pnc/internal/inboxoutstanding"
	"claim-pnc/internal/inboxprogressclaim"
	"claim-pnc/internal/inboxxol"
	"claim-pnc/internal/inputreqprotection"
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
	"claim-pnc/internal/portal"
	"claim-pnc/spa"

	authhttp "claim-pnc/internal/auth/http"
	inboxacceptopenprotectionhttp "claim-pnc/internal/inboxacceptopenprotection/http"
	inboxacceptopenprotectionmemory "claim-pnc/internal/inboxacceptopenprotection/repo/memory"
	inboxacceptopenprotectionsql "claim-pnc/internal/inboxacceptopenprotection/repo/sqlstore"
	inboxacceptopenprotectionusecase "claim-pnc/internal/inboxacceptopenprotection/usecase"
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

	inboxlaporanklaimhttp "claim-pnc/internal/inboxlaporanklaim/http"
	inboxlaporanklaimmemory "claim-pnc/internal/inboxlaporanklaim/repo/memory"
	inboxlaporanklaimsql "claim-pnc/internal/inboxlaporanklaim/repo/sqlstore"
	inboxlaporanklaimusecase "claim-pnc/internal/inboxlaporanklaim/usecase"
	inboxoutstandinghttp "claim-pnc/internal/inboxoutstanding/http"
	inboxoutstandingmemory "claim-pnc/internal/inboxoutstanding/repo/memory"
	inboxoutstandingsql "claim-pnc/internal/inboxoutstanding/repo/sqlstore"
	inboxoutstandingusecase "claim-pnc/internal/inboxoutstanding/usecase"
	inboxprogressclaimhttp "claim-pnc/internal/inboxprogressclaim/http"
	inboxprogressclaimmemory "claim-pnc/internal/inboxprogressclaim/repo/memory"
	inboxprogressclaimsql "claim-pnc/internal/inboxprogressclaim/repo/sqlstore"
	inboxprogressclaimusecase "claim-pnc/internal/inboxprogressclaim/usecase"
	inboxxolhttp "claim-pnc/internal/inboxxol/http"
	inboxxolmemory "claim-pnc/internal/inboxxol/repo/memory"
	inboxxolsql "claim-pnc/internal/inboxxol/repo/sqlstore"
	inboxxolusecase "claim-pnc/internal/inboxxol/usecase"
	inputreqprotectionhttp "claim-pnc/internal/inputreqprotection/http"
	inputreqprotectionmemory "claim-pnc/internal/inputreqprotection/repo/memory"
	inputreqprotectionsql "claim-pnc/internal/inputreqprotection/repo/sqlstore"
	inputreqprotectionusecase "claim-pnc/internal/inputreqprotection/usecase"
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
	}

	// Satu penulis JSON dan satu penulis galat dipakai bersama seluruh modul, supaya
	// bentuk respons dan header Cache-Control-nya tidak berbeda antarmodul.
	writeJSON := func(w http.ResponseWriter, r *http.Request, status int, body any) {
		authhttp.WriteJSON(w, r, status, body, logger)
	}
	writeAuthError := authhttp.WriteError(logger)

	handlerAuth := authhttp.NewHandler(assembly.auth, logger)
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
			return inboxlaporanklaim.Caller{
				Login: baseCtx.User.Login,
				Name:  baseCtx.User.Name,
				// Cabang menentukan batas data seluruh layar ini. Ia datang dari profil
				// HCC/HCQ; pengguna non-karyawan tidak memilikinya, dan berkas baru
				// karena itu ditolak dengan pesan yang menyebutkan sebabnya.
				BranchCode: baseCtx.User.BranchCode,
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

	// Input Req Protection. Pembuat permintaan diambil dari SESI, bukan dari badan
	// permintaan — ia satu-satunya jejak siapa yang meminta pembukaan proteksi.
	protectionRequestHandler := inputreqprotectionhttp.NewHandler(inputreqprotectionhttp.Options{
		Service: assembly.inputReqProtection,
		GetCaller: func(ctx context.Context) (inputreqprotectionhttp.Caller, bool) {
			baseCtx, existing := authhttp.CallerFromContext(ctx)
			if !existing {
				return inputreqprotectionhttp.Caller{}, false
			}
			return inputreqprotectionhttp.Caller{Login: baseCtx.User.Login}, true
		},
		Logger:              logger,
		WriteJSON:           writeJSON,
		FallbackErrorWriter: inputreqprotectionhttp.ErrorWriter(writePortalAwareError),
	})

	// Inbox Accept Open Protection. Pelaku akseptasi juga diambil dari sesi: `D-59`
	// menetapkan tidak ada pemisahan tugas formal, sehingga kolom DIAKSEP_OLEH adalah
	// satu-satunya kontrol pengimbang yang tersisa atas persetujuan ini.
	protectionAcceptHandler := inboxacceptopenprotectionhttp.NewHandler(
		inboxacceptopenprotectionhttp.Options{
			Service: assembly.inboxAcceptOpenProtection,
			GetCaller: func(ctx context.Context) (inboxacceptopenprotectionhttp.Caller, bool) {
				baseCtx, existing := authhttp.CallerFromContext(ctx)
				if !existing {
					return inboxacceptopenprotectionhttp.Caller{}, false
				}
				return inboxacceptopenprotectionhttp.Caller{Login: baseCtx.User.Login}, true
			},
			Logger:              logger,
			WriteJSON:           writeJSON,
			FallbackErrorWriter: inboxacceptopenprotectionhttp.ErrorWriter(writePortalAwareError),
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
				// Input Req Protection dan Inbox Accept Open Protection. Keduanya
				// menyentuh POOLDATA.T_CLAIM_OPENPROTECTION di basis data entitas,
				// sehingga pemeriksaan portal dipasang di dalam Mount masing-masing.
				//
				// Barisnya memuat nomor polis dan nomor klaim; tidak satu pun boleh
				// terbaca tanpa sesi. Layar akseptasi bahkan MENULIS persetujuan atas
				// pembukaan proteksi, dan sampai TKT-F3-005 dikerjakan, yang tersisa
				// sebagai kontrol hanyalah jejak DIAKSEP_OLEH (D-59).
				inputreqprotectionhttp.Mount(protected, protectionRequestHandler, activePortalDeps)
				inboxacceptopenprotectionhttp.Mount(protected, protectionAcceptHandler, activePortalDeps)
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

				// Inbox Progress Claim memuat nama tertanggung, nomor polis, dan
				// catatan progres — seluruhnya milik satu badan hukum. Rutenya karena
				// itu menuntut portal, sama seperti Inbox Admin.
				inboxprogressclaimhttp.Mount(
					protected, inboxProgressClaimHandler, activePortalDeps)
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

	// inboxProgressClaim melayani layar Inbox Progress Claim (`MENU_ID 65`).
	inboxProgressClaim *inboxprogressclaimusecase.Service

	// inboxLaporanKlaim melayani layar Inbox Laporan Klaim. Sama seperti master status
	// progres, ia memakai pemilih repo per portal: berkas laporan adalah data bisnis
	// milik satu badan hukum (ADR-0030).
	inboxLaporanKlaim *inboxlaporanklaimusecase.Service

	// inboxOutstanding melayani layar Inbox Outstanding — klaim yang masih berjalan.
	inboxOutstanding *inboxoutstandingusecase.Service

	// inputReqProtection melayani layar Input Req Protection — permintaan pembukaan
	// proteksi beserta form inputnya.
	inputReqProtection *inputreqprotectionusecase.Service

	// inboxAcceptOpenProtection melayani layar Inbox Accept Open Protection — antrean
	// akseptasi atas permintaan yang sama.
	//
	// Kedua modul menyentuh SATU tabel, tetapi menulis kolom yang berbeda: yang pertama
	// kolom pembuatan, yang kedua kolom akseptasi. Pembagian itu yang menjaga P-1 tetap
	// berlaku tanpa menggabungkan keduanya menjadi satu modul.
	inboxAcceptOpenProtection *inboxacceptopenprotectionusecase.Service

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

	// inboxProgressClaimSelector memilih penyimpanan progres klaim milik satu portal.
	//
	// Ia fungsi dengan alasan yang sama: progres klaim satu badan hukum bukan progres
	// badan hukum lain, dan barisnya memuat nama tertanggung (`ADR-0030`, `R-20`).
	inboxProgressClaimSelector inboxprogressclaim.RepoSelector

	// claimReportSelector memilih penyimpanan berkas laporan klaim milik satu portal.
	//
	// Alasannya sama dengan progressStatusSelector di bawah, ditambah satu yang khas
	// modul ini: ia membaca DUA tabel sekaligus — tabel warisan Pega dan tabel milik
	// aplikasi ini — dan keduanya hidup di basis data entitas yang sama.
	claimReportSelector inboxlaporanklaim.RepoSelector

	// outstandingSelector memilih penyimpanan klaim milik satu portal.
	outstandingSelector inboxoutstanding.RepoSelector

	// protectionRequestSelector memilih penyimpanan permintaan proteksi milik satu portal.
	//
	// Ia fungsi, bukan repo tunggal, karena POOLDATA.T_CLAIM_OPENPROTECTION ada di basis
	// data SETIAP entitas (ADR-0030).
	protectionRequestSelector inputreqprotection.RepoSelector

	// protectionAcceptSelector memilih penyimpanan antrean akseptasi milik satu portal.
	//
	// Ia menunjuk tabel yang SAMA dengan protectionRequestSelector; yang berbeda adalah
	// kolom yang ditulisnya.
	protectionAcceptSelector inboxacceptopenprotection.RepoSelector

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
		RepoSelector: store.claimReportSelector,
		Clock:        clock.System{},
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

	outstandingService, err := inboxoutstandingusecase.NewService(store.outstandingSelector)
	if err != nil {
		store.close()
		return assembly{}, err
	}

	// Jam dan zona keduanya diserahkan, bukan dibaca di dalam modul: aturan "proteksi ganda
	// di hari yang sama" bergantung pada TANGGAL WIB, dan aturan yang membaca jam sendiri
	// tidak dapat diuji tanpa menunggu pergantian hari.
	protectionRequestService, err := inputreqprotectionusecase.NewService(
		inputreqprotectionusecase.Options{Protections: store.protectionRequestSelector},
	)
	if err != nil {
		store.close()
		return assembly{}, err
	}

	protectionAcceptService, err := inboxacceptopenprotectionusecase.NewService(
		inboxacceptopenprotectionusecase.Options{Protections: store.protectionAcceptSelector},
	)
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
		inboxProgressClaim:      inboxProgressClaimService,
		inboxLaporanKlaim:       claimReportService,
		inboxOutstanding:        outstandingService,

		inputReqProtection:        protectionRequestService,
		inboxAcceptOpenProtection: protectionAcceptService,

		readyAliases: store.readyAliases,
		close:        store.close,
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

		// Open Protection: DUA repo di atas SATU tabel, POOLDATA.T_CLAIM_OPENPROTECTION.
		//
		// Tabelnya dibuat Work Owner pada 2026-09-23 dan berisi 16 kolom; nama kolom pada
		// kedua adapter dibaca langsung dari katalog Oracle, bukan dari usulan —
		// keduanya sempat berbeda. Rinciannya di docs/kolom-open-protection.md.
		//
		// Keduanya menulis kolom yang BERBEDA pada tahap hidup yang berbeda: yang pertama
		// kolom pembuatan, yang kedua APPROVALSTATUS/RESOLVEDBY/RESOLVEDATETIME. Itulah
		// yang menjaga `P-1` tetap berlaku tanpa menggabungkan kedua modul.
		//
		// Portal yang tidak dikenal atau belum siap menghasilkan galat dari For(), TIDAK
		// pernah dialihkan ke koneksi utama (`R-20`).
		// Repo proteksi dan repo master tipe dibentuk dari SATU koneksi yang sama, dan
		// dikembalikan bersamaan. Memilih keduanya lewat dua pemanggilan terpisah membuka
		// kemungkinan proteksi dibaca dari portal yang satu dan nama tipenya dari portal
		// yang lain — kelas cacat yang tidak menghasilkan galat apa pun.
		store.protectionRequestSelector = func(alias string) (inputreqprotection.Stores, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return inputreqprotection.Stores{}, err
			}
			return inputreqprotection.Stores{
				Protections: inputreqprotectionsql.NewRepo(conn),
				Types:       inputreqprotectionsql.NewTypeRepo(conn),
				Claims:      inputreqprotectionsql.NewClaimRepo(conn),
			}, nil
		}
		store.protectionAcceptSelector = func(alias string) (inboxacceptopenprotection.Repo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return inboxacceptopenprotectionsql.NewRepo(conn), nil
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

		store.inboxProgressClaimSelector = func(alias string) (inboxprogressclaim.Repo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return inboxprogressclaimsql.NewRepo(conn), nil
		}
	} else {
		store.portal = portalmemory.NewRepo(portalmemory.SampleList()...)
		store.accountSelector = accountSelectorMemory(cfg.PrimaryPortal)
		// Ke-33 status nyata ikut dimuat, sehingga layar Master Status Klaim dapat
		// dicoba lengkap tanpa Oracle dan tanpa menunggu migrasi 0002.
		store.readyAliases = func() []string { return []string{cfg.PrimaryPortal} }
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

		// Open Protection: DUA penyimpanan memori yang berbeda, dan itu disengaja.
		//
		// Di produksi keduanya membaca tabel yang sama, tetapi menulis kolom yang berbeda
		// (`P-1`). Memakai satu objek bersama di sini akan membuat uji coba lokal
		// menyembunyikan kesalahan pemetaan antar keduanya — misalnya modul akseptasi yang
		// diam-diam ikut mengubah keterangan.
		//
		// Akibat yang harus disadari saat mencoba lokal: proteksi yang baru dibuat di layar
		// Input Req Protection TIDAK muncul di layar akseptasi, dan sebaliknya. Keduanya
		// menyatu hanya setelah tabelnya ada. Seluruh isi contohnya karangan — lihat
		// masing-masing repo/memory/sample.go.
		// Master tipe proteksi ikut dimuat berisi kesembilan tipe yang benar-benar ada di
		// `POOLDATA.M_CLAIM_PROTECTION_TYPE`, sehingga pilihan tipe pada form menampilkan
		// nama yang SAMA dengan produksi tanpa Oracle. Ia salinan, bukan cadangan — tidak
		// ada jalur yang jatuh ke sini saat master di Oracle kosong.
		protectionRequestMemory := inputreqprotectionmemory.NewRepoWithSamples()
		protectionTypeMemory := inputreqprotectionmemory.NewTypeRepoWithSamples()
		// Klaim contoh ikut dimuat supaya form tipe 7 dan 8 dapat dicoba utuh tanpa
		// Oracle — termasuk field turunan yang tidak dapat diketik.
		protectionClaimMemory := inputreqprotectionmemory.NewClaimRepoWithSamples()
		store.protectionRequestSelector = func(string) (inputreqprotection.Stores, error) {
			return inputreqprotection.Stores{
				Protections: protectionRequestMemory,
				Types:       protectionTypeMemory,
				Claims:      protectionClaimMemory,
			}, nil
		}

		protectionAcceptMemory := inboxacceptopenprotectionmemory.NewRepoWithSamples()
		store.protectionAcceptSelector = func(string) (inboxacceptopenprotection.Repo, error) {
			return protectionAcceptMemory, nil
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
		// NewDevRepo, bukan NewSampleRepo: isi contoh m_login_group_pnc.csv hanya
		// memuat satu login, dan login provider tiruan tidak ada di dalamnya. Tanpa
		// itu, masuk saat pengembangan menghasilkan menu kosong yang tampak rusak.
		store.menu = menumemory.NewDevRepo()
		store.inboxXOLSelector = inboxXOLSelectorMemory(cfg.PrimaryPortal)
		store.claimTreatyPropSelector = claimTreatyPropSelectorMemory(cfg.PrimaryPortal)
		store.claimTreatyNonPropSelector = claimTreatyNonPropSelectorMemory(cfg.PrimaryPortal)
		store.inboxProgressClaimSelector = inboxProgressClaimSelectorMemory(cfg.PrimaryPortal)
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

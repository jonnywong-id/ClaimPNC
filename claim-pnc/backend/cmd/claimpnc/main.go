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
	"claim-pnc/internal/detailpenyebab"
	"claim-pnc/internal/inboxinvestigator"
	"claim-pnc/internal/inboxreceivetka"
	"claim-pnc/internal/masterautoclaim"
	"claim-pnc/internal/masterbengkel"
	"claim-pnc/internal/mastergroupingsparepart"
	"claim-pnc/internal/masterkategorisparepart"
	"claim-pnc/internal/masterlogin"
	"claim-pnc/internal/masterpanel"
	"claim-pnc/internal/masterpasal"
	"claim-pnc/internal/masterpasalai"
	"claim-pnc/internal/masterpenolakan"
	"claim-pnc/internal/masterreas"
	"claim-pnc/internal/masterrekening"
	"claim-pnc/internal/mastersparepart"
	"claim-pnc/internal/masterstatus"
	"claim-pnc/internal/masterstatusprogres"
	"claim-pnc/internal/mastersupplier"
	"claim-pnc/internal/mastertipesparepart"
	"claim-pnc/internal/menu"
	"claim-pnc/internal/pelaporanklaim"
	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/platform/config"
	"claim-pnc/internal/platform/db"
	"claim-pnc/internal/platform/httpserver"
	"claim-pnc/internal/platform/logging"
	"claim-pnc/internal/portal"
	"claim-pnc/internal/riwayatklaim"
	"claim-pnc/spa"

	authhttp "claim-pnc/internal/auth/http"
	detailpenyebabhttp "claim-pnc/internal/detailpenyebab/http"
	detailpenyebabmemory "claim-pnc/internal/detailpenyebab/repo/memory"
	detailpenyebabsql "claim-pnc/internal/detailpenyebab/repo/sqlstore"
	detailpenyebabusecase "claim-pnc/internal/detailpenyebab/usecase"
	inboxinvestigatorhttp "claim-pnc/internal/inboxinvestigator/http"
	inboxinvestigatormemory "claim-pnc/internal/inboxinvestigator/repo/memory"
	inboxinvestigatorsql "claim-pnc/internal/inboxinvestigator/repo/sqlstore"
	inboxinvestigatorusecase "claim-pnc/internal/inboxinvestigator/usecase"
	inboxreceivetkahttp "claim-pnc/internal/inboxreceivetka/http"
	inboxreceivetkanotif "claim-pnc/internal/inboxreceivetka/notification"
	inboxreceivetkamemory "claim-pnc/internal/inboxreceivetka/repo/memory"
	inboxreceivetkasql "claim-pnc/internal/inboxreceivetka/repo/sqlstore"
	inboxreceivetkausecase "claim-pnc/internal/inboxreceivetka/usecase"
	masterautoclaimhttp "claim-pnc/internal/masterautoclaim/http"
	masterautoclaimmemory "claim-pnc/internal/masterautoclaim/repo/memory"
	masterautoclaimsql "claim-pnc/internal/masterautoclaim/repo/sqlstore"
	masterautoclaimusecase "claim-pnc/internal/masterautoclaim/usecase"
	masterbengkelhttp "claim-pnc/internal/masterbengkel/http"
	masterbengkelmemory "claim-pnc/internal/masterbengkel/repo/memory"
	masterbengkelsql "claim-pnc/internal/masterbengkel/repo/sqlstore"
	masterbengkelusecase "claim-pnc/internal/masterbengkel/usecase"
	mastergroupingspareparthttp "claim-pnc/internal/mastergroupingsparepart/http"
	mastergroupingsparepartmemory "claim-pnc/internal/mastergroupingsparepart/repo/memory"
	mastergroupingsparepartsql "claim-pnc/internal/mastergroupingsparepart/repo/sqlstore"
	mastergroupingsparepartusecase "claim-pnc/internal/mastergroupingsparepart/usecase"
	masterkategorispareparthttp "claim-pnc/internal/masterkategorisparepart/http"
	masterkategorisparepartmemory "claim-pnc/internal/masterkategorisparepart/repo/memory"
	masterkategorisparepartsql "claim-pnc/internal/masterkategorisparepart/repo/sqlstore"
	masterkategorisparepartusecase "claim-pnc/internal/masterkategorisparepart/usecase"
	masterloginhttp "claim-pnc/internal/masterlogin/http"
	masterloginmemory "claim-pnc/internal/masterlogin/repo/memory"
	masterloginsql "claim-pnc/internal/masterlogin/repo/sqlstore"
	masterloginusecase "claim-pnc/internal/masterlogin/usecase"
	masterpanelhttp "claim-pnc/internal/masterpanel/http"
	masterpanelmemory "claim-pnc/internal/masterpanel/repo/memory"
	masterpanelsql "claim-pnc/internal/masterpanel/repo/sqlstore"
	masterpanelusecase "claim-pnc/internal/masterpanel/usecase"
	masterpasalhttp "claim-pnc/internal/masterpasal/http"
	masterpasalmemory "claim-pnc/internal/masterpasal/repo/memory"
	masterpasalsql "claim-pnc/internal/masterpasal/repo/sqlstore"
	masterpasalusecase "claim-pnc/internal/masterpasal/usecase"
	masterpasalaihttp "claim-pnc/internal/masterpasalai/http"
	masterpasalaimemory "claim-pnc/internal/masterpasalai/repo/memory"
	masterpasalaisql "claim-pnc/internal/masterpasalai/repo/sqlstore"
	masterpasalaiusecase "claim-pnc/internal/masterpasalai/usecase"
	masterpenolakanhttp "claim-pnc/internal/masterpenolakan/http"
	masterpenolakanmemory "claim-pnc/internal/masterpenolakan/repo/memory"
	masterpenolakansql "claim-pnc/internal/masterpenolakan/repo/sqlstore"
	masterpenolakanusecase "claim-pnc/internal/masterpenolakan/usecase"
	masterreashttp "claim-pnc/internal/masterreas/http"
	masterreasmemory "claim-pnc/internal/masterreas/repo/memory"
	masterreassql "claim-pnc/internal/masterreas/repo/sqlstore"
	masterreasusecase "claim-pnc/internal/masterreas/usecase"
	masterrekeningcashier "claim-pnc/internal/masterrekening/cashier"
	masterrekeninghttp "claim-pnc/internal/masterrekening/http"
	masterrekeningnotif "claim-pnc/internal/masterrekening/notification"
	masterrekeningmemory "claim-pnc/internal/masterrekening/repo/memory"
	masterrekeningsql "claim-pnc/internal/masterrekening/repo/sqlstore"
	masterrekeningusecase "claim-pnc/internal/masterrekening/usecase"
	masterspareparthttp "claim-pnc/internal/mastersparepart/http"
	mastersparepartmemory "claim-pnc/internal/mastersparepart/repo/memory"
	mastersparepartsql "claim-pnc/internal/mastersparepart/repo/sqlstore"
	mastersparepartusecase "claim-pnc/internal/mastersparepart/usecase"
	masterstatushttp "claim-pnc/internal/masterstatus/http"
	masterstatusmemory "claim-pnc/internal/masterstatus/repo/memory"
	masterstatussql "claim-pnc/internal/masterstatus/repo/sqlstore"
	masterstatususecase "claim-pnc/internal/masterstatus/usecase"
	masterstatusprogreshttp "claim-pnc/internal/masterstatusprogres/http"
	masterstatusprogresmemory "claim-pnc/internal/masterstatusprogres/repo/memory"
	masterstatusprogressql "claim-pnc/internal/masterstatusprogres/repo/sqlstore"
	masterstatusprogresusecase "claim-pnc/internal/masterstatusprogres/usecase"
	mastersupplierhttp "claim-pnc/internal/mastersupplier/http"
	mastersuppliermemory "claim-pnc/internal/mastersupplier/repo/memory"
	mastersuppliersql "claim-pnc/internal/mastersupplier/repo/sqlstore"
	mastersupplierusecase "claim-pnc/internal/mastersupplier/usecase"
	mastertipespareparthttp "claim-pnc/internal/mastertipesparepart/http"
	mastertipesparepartmemory "claim-pnc/internal/mastertipesparepart/repo/memory"
	mastertipesparepartsql "claim-pnc/internal/mastertipesparepart/repo/sqlstore"
	mastertipesparepartusecase "claim-pnc/internal/mastertipesparepart/usecase"
	menuhttp "claim-pnc/internal/menu/http"
	menumemory "claim-pnc/internal/menu/repo/memory"
	menusql "claim-pnc/internal/menu/repo/sqlstore"
	menuusecase "claim-pnc/internal/menu/usecase"
	pelaporanklaimhttp "claim-pnc/internal/pelaporanklaim/http"
	pelaporanklaimmemory "claim-pnc/internal/pelaporanklaim/repo/memory"
	pelaporanklaimsql "claim-pnc/internal/pelaporanklaim/repo/sqlstore"
	pelaporanklaimusecase "claim-pnc/internal/pelaporanklaim/usecase"
	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
	portalsql "claim-pnc/internal/portal/repo/sqlstore"
	riwayatklaimhttp "claim-pnc/internal/riwayatklaim/http"
	riwayatklaimmemory "claim-pnc/internal/riwayatklaim/repo/memory"
	riwayatklaimsql "claim-pnc/internal/riwayatklaim/repo/sqlstore"
	riwayatklaimusecase "claim-pnc/internal/riwayatklaim/usecase"
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

	clauseAIHandler, err := masterpasalaihttp.NewHandler(masterpasalaihttp.Options{
		Service:       assembly.masterPasalAI,
		Logger:        logger,
		WriteResponse: writeJSON,
		WriteError:    masterpasalaihttp.ErrorWriter(writePortalAwareError),
	})
	if err != nil {
		return err
	}

	// Master Status Progres 2 memakai penulis galat yang SAMA PERSIS dengan tingkat 1,
	// bukan rantai baru: keduanya memetakan galat lewat satu fungsi petakanGalat, sehingga
	// satu jenis galat tidak pernah dijawab dua bentuk yang berbeda.
	progressStatus2Handler, err := masterstatusprogreshttp.NewHandler2(masterstatusprogreshttp.Options2{
		Service:       assembly.masterStatusProgres2,
		Logger:        logger,
		WriteResponse: writeJSON,
		WriteError:    masterstatusprogreshttp.ErrorWriter(writePortalAwareError),
	})
	if err != nil {
		return err
	}

	// Master Penolakan Klaim memakai penulis galat yang SAMA dengan master status
	// progres: ia pun menyentuh basis data entitas, sehingga galat portal harus dijawab
	// dengan kode yang sudah dikenal frontend.
	rejectionHandler, err := masterpenolakanhttp.NewHandler(masterpenolakanhttp.Options{
		Service: assembly.masterPenolakan,
		Komite:  assembly.masterPenolakanKomite,
		// Jembatan satu arah dari modul auth. Ia dipasang di sini, bukan di dalam salah
		// satu modul, supaya kedua modul tetap tidak saling mengimpor — yang tahu
		// keduanya hanyalah berkas perakitan ini.
		//
		// Login yang DIKETIK pengguna, bukan NIK: kolom USER_INPUT pada
		// POOLDATA.MST_PENOLAKAN_KLAIM_2 sudah berisi login pada baris-baris lama.
		Caller: func(ctx context.Context) (masterpenolakanhttp.Caller, bool) {
			baseCtx, existing := authhttp.CallerFromContext(ctx)
			if !existing {
				return masterpenolakanhttp.Caller{}, false
			}
			return masterpenolakanhttp.Caller{Login: baseCtx.User.Login}, true
		},
		Logger:        logger,
		WriteResponse: writeJSON,
		WriteError:    masterpenolakanhttp.ErrorWriter(writePortalAwareError),
	})
	if err != nil {
		return err
	}

	// Master Auto Claim memakai penulis galat yang SAMA dengan modul bisnis lain: ia
	// menyentuh basis data entitas, sehingga galat portal harus dijawab dengan kode yang
	// sudah dikenal frontend.
	autoClaimHandler, err := masterautoclaimhttp.NewHandler(masterautoclaimhttp.Options{
		Service: assembly.masterAutoClaim,
		// Jembatan satu arah dari modul auth, dipasang di sini supaya kedua modul tetap
		// tidak saling mengimpor.
		//
		// Login yang DIKETIK pengguna, bukan NIK — dan di modul ini pilihan itu
		// MENENTUKAN, bukan sekadar rapi: nilainya dibandingkan dengan kolom KOMITE,
		// yang berisi OPERATOR_ID dari POOLDATA.EMAILKOMITE. Memakai NIK akan membuat
		// tab Komite Approval selalu kosong, tanpa satu pun galat.
		Caller: func(ctx context.Context) (masterautoclaimhttp.Caller, bool) {
			baseCtx, existing := authhttp.CallerFromContext(ctx)
			if !existing {
				return masterautoclaimhttp.Caller{}, false
			}
			return masterautoclaimhttp.Caller{Login: baseCtx.User.Login}, true
		},
		Logger:        logger,
		WriteResponse: writeJSON,
		WriteError:    masterautoclaimhttp.ErrorWriter(writePortalAwareError),
	})
	if err != nil {
		return err
	}

	// Master Bengkel memakai penulis galat yang SAMA dengan modul bisnis lain: ia
	// menyentuh basis data entitas, sehingga galat portal harus dijawab dengan kode yang
	// sudah dikenal frontend.
	workshopHandler, err := masterbengkelhttp.NewHandler(masterbengkelhttp.Options{
		Service: assembly.masterBengkel,
		// Jembatan satu arah dari modul auth, dipasang di sini supaya kedua modul tetap
		// tidak saling mengimpor.
		//
		// Login yang DIKETIK pengguna, bukan NIK — sama seperti modul master lain. Di
		// modul ini ia TIDAK pernah tersimpan: POOLDATA.BENGKEL_HE tidak punya satu pun
		// kolom pencatat pelaku maupun waktu, sehingga identitas pemanggil hanya masuk
		// log. Itu keterbatasan tabelnya, dan sudah dicatat pada usecase.Actor.
		Caller: func(ctx context.Context) (masterbengkelhttp.Caller, bool) {
			baseCtx, existing := authhttp.CallerFromContext(ctx)
			if !existing {
				return masterbengkelhttp.Caller{}, false
			}
			return masterbengkelhttp.Caller{Login: baseCtx.User.Login}, true
		},
		Logger:        logger,
		WriteResponse: writeJSON,
		WriteError:    masterbengkelhttp.ErrorWriter(writePortalAwareError),
	})
	if err != nil {
		return err
	}

	// Master Panel memakai penulis galat yang SAMA dengan modul bisnis lain: ia menyentuh
	// basis data entitas, sehingga galat portal harus dijawab dengan kode yang sudah
	// dikenal frontend.
	panelHandler, err := masterpanelhttp.NewHandler(masterpanelhttp.Options{
		Service: assembly.masterPanel,
		// Jembatan satu arah dari modul auth, dipasang di sini supaya kedua modul tetap
		// tidak saling mengimpor.
		//
		// Login yang DIKETIK pengguna, bukan NIK — sama seperti modul master lain. Di
		// modul ini ia TIDAK pernah tersimpan: POOLDATA.PANEL_HE tidak punya satu pun
		// kolom pencatat pelaku maupun waktu, sehingga identitas pemanggil hanya masuk
		// log. Itu keterbatasan tabelnya, dan sudah dicatat pada usecase.Actor.
		Caller: func(ctx context.Context) (masterpanelhttp.Caller, bool) {
			baseCtx, existing := authhttp.CallerFromContext(ctx)
			if !existing {
				return masterpanelhttp.Caller{}, false
			}
			return masterpanelhttp.Caller{Login: baseCtx.User.Login}, true
		},
		Logger:        logger,
		WriteResponse: writeJSON,
		WriteError:    masterpanelhttp.ErrorWriter(writePortalAwareError),
	})
	if err != nil {
		return err
	}

	// Master Sparepart memakai penulis galat yang SAMA dengan modul bisnis lain: ia
	// menyentuh basis data entitas, sehingga galat portal harus dijawab dengan kode yang
	// sudah dikenal frontend.
	sparepartHandler, err := masterspareparthttp.NewHandler(masterspareparthttp.Options{
		Service: assembly.masterSparepart,
		// Jembatan satu arah dari modul auth, dipasang di sini supaya kedua modul tetap
		// tidak saling mengimpor.
		//
		// Login yang DIKETIK pengguna, bukan NIK — sama seperti modul master lain. Di modul
		// ini ia BENAR-BENAR TERSIMPAN ke kolom USER_UPDATE, berbeda dari Master Panel yang
		// tabelnya tidak punya kolom pencatat pelaku sama sekali. Kolom itu sudah berisi
		// login pada baris-baris lama, dan menuliskan NIK ke kolom yang sama akan membuat dua
		// bentuk identitas hidup berdampingan tanpa cara membedakannya.
		Caller: func(ctx context.Context) (masterspareparthttp.Caller, bool) {
			baseCtx, existing := authhttp.CallerFromContext(ctx)
			if !existing {
				return masterspareparthttp.Caller{}, false
			}
			return masterspareparthttp.Caller{Login: baseCtx.User.Login}, true
		},
		Logger:        logger,
		WriteResponse: writeJSON,
		WriteError:    masterspareparthttp.ErrorWriter(writePortalAwareError),
	})
	if err != nil {
		return err
	}

	// Master Grouping Sparepart memakai penulis galat yang SAMA dengan modul bisnis lain: ia
	// menyentuh basis data entitas, sehingga galat portal harus dijawab dengan kode yang
	// sudah dikenal frontend.
	groupingHandler, err := mastergroupingspareparthttp.NewHandler(
		mastergroupingspareparthttp.Options{
			Service: assembly.masterGroupingSparepart,
			// Jembatan satu arah dari modul auth, dipasang di sini supaya kedua modul tetap
			// tidak saling mengimpor.
			//
			// Login yang DIKETIK pengguna, bukan NIK — sama seperti modul master lain. Di
			// modul ini ia TIDAK tersimpan ke basis data: kedua tabelnya tidak punya satu pun
			// kolom pencatat pelaku. Ia dipakai untuk mencatat siapa yang mengubah apa di log,
			// satu-satunya tempat yang tersedia sampai S-5 Jejak Audit dibangun.
			Caller: func(ctx context.Context) (mastergroupingspareparthttp.Caller, bool) {
				baseCtx, existing := authhttp.CallerFromContext(ctx)
				if !existing {
					return mastergroupingspareparthttp.Caller{}, false
				}
				return mastergroupingspareparthttp.Caller{Login: baseCtx.User.Login}, true
			},
			Logger:        logger,
			WriteResponse: writeJSON,
			WriteError:    mastergroupingspareparthttp.ErrorWriter(writePortalAwareError),
		})
	if err != nil {
		return err
	}

	// Master Kategori Sparepart memakai penulis galat yang SAMA dengan modul bisnis lain: ia
	// menyentuh basis data entitas, sehingga galat portal harus dijawab dengan kode yang
	// sudah dikenal frontend.
	//
	// Ia MENERIMA Caller meski tabelnya tidak punya kolom pencatat pelaku. Login-nya tidak
	// tersimpan di basis data — ia hanya masuk ke log, dan log itulah satu-satunya tempat
	// siapa yang menyetujui sebuah kategori terekam. Lihat masterkategorisparepart/usecase.Actor.
	partCategoryHandler, err := masterkategorispareparthttp.NewHandler(
		masterkategorispareparthttp.Options{
			Service: assembly.masterKategoriSparepart,
			Caller: func(ctx context.Context) (masterkategorispareparthttp.Caller, bool) {
				baseCtx, existing := authhttp.CallerFromContext(ctx)
				if !existing {
					return masterkategorispareparthttp.Caller{}, false
				}
				return masterkategorispareparthttp.Caller{Login: baseCtx.User.Login}, true
			},
			Logger:        logger,
			WriteResponse: writeJSON,
			WriteError:    masterkategorispareparthttp.ErrorWriter(writePortalAwareError),
		})
	if err != nil {
		return err
	}

	// Master Tipe Sparepart memakai penulis galat yang SAMA dengan modul bisnis lain: ia
	// menyentuh basis data entitas, sehingga galat portal harus dijawab dengan kode yang
	// sudah dikenal frontend.
	//
	// Ia MENERIMA Caller meski tabelnya tidak punya kolom pencatat pelaku. Login-nya tidak
	// tersimpan di basis data — ia hanya masuk ke log, dan log itulah satu-satunya tempat
	// siapa yang menyetujui sebuah tipe terekam. Lihat mastertipesparepart/usecase.Actor.
	partTypeHandler, err := mastertipespareparthttp.NewHandler(
		mastertipespareparthttp.Options{
			Service: assembly.masterTipeSparepart,
			Caller: func(ctx context.Context) (mastertipespareparthttp.Caller, bool) {
				baseCtx, existing := authhttp.CallerFromContext(ctx)
				if !existing {
					return mastertipespareparthttp.Caller{}, false
				}
				return mastertipespareparthttp.Caller{Login: baseCtx.User.Login}, true
			},
			Logger:        logger,
			WriteResponse: writeJSON,
			WriteError:    mastertipespareparthttp.ErrorWriter(writePortalAwareError),
		})
	if err != nil {
		return err
	}

	// Master Login memakai penulis galat yang SAMA dengan modul bisnis lain.
	//
	// Ia MENERIMA Caller, dan di sini Caller bukan sekadar pengisi log seperti pada kedua
	// modul di atas: login pemanggil dipakai MENURUNKAN kolom LOGINLEADER setiap baris baru
	// — `Activity/CNMInsertMstLoginSurveyor_act` mencari leader milik pengguna yang
	// menyimpan, lalu menuliskannya ke baris yang dibuat. Tanpa identitas itu, setiap baris
	// baru lahir tanpa tautan tim. Lihat masterlogin/usecase.Service.Create.
	// Detail Penyebab Kerugian. Caller-nya HANYA pengisi log — POOLDATA.D_CAUSE_OF_LOSS
	// hanya punya D_COL_ID dan JSONDATA, tanpa satu pun kolom pencatat pelaku maupun waktu.
	// Itu keterbatasan tabelnya, dan menambahkannya menempuh `D-63`; lihat
	// detailpenyebab/usecase.Actor.
	causeOfLossDetailHandler, err := detailpenyebabhttp.NewHandler(
		detailpenyebabhttp.Options{
			Service: assembly.detailPenyebab,
			Caller: func(ctx context.Context) (detailpenyebabhttp.Caller, bool) {
				baseCtx, existing := authhttp.CallerFromContext(ctx)
				if !existing {
					return detailpenyebabhttp.Caller{}, false
				}
				return detailpenyebabhttp.Caller{Login: baseCtx.User.Login}, true
			},
			Logger:        logger,
			WriteResponse: writeJSON,
			WriteError:    detailpenyebabhttp.ErrorWriter(writePortalAwareError),
		})
	if err != nil {
		return err
	}

	surveyorLoginHandler, err := masterloginhttp.NewHandler(
		masterloginhttp.Options{
			Service: assembly.masterLogin,
			Caller: func(ctx context.Context) (masterloginhttp.Caller, bool) {
				baseCtx, existing := authhttp.CallerFromContext(ctx)
				if !existing {
					return masterloginhttp.Caller{}, false
				}
				return masterloginhttp.Caller{Login: baseCtx.User.Login}, true
			},
			Logger:        logger,
			WriteResponse: writeJSON,
			WriteError:    masterloginhttp.ErrorWriter(writePortalAwareError),
		})
	if err != nil {
		return err
	}

	// Master Pasal Kerugian memakai penulis galat yang SAMA dengan modul bisnis lain: ia
	// menyentuh basis data entitas, sehingga galat portal harus dijawab dengan kode yang
	// sudah dikenal frontend.
	//
	// Ia TIDAK menerima Caller, dan itu bukan kelalaian: POOLDATA.V_M_DATA_PASAL hanya
	// punya tiga kolom — IDDATA, IDPASAL, JSONPASAL — sehingga tidak ada tempat menuliskan
	// siapa dan kapan. Akibatnya perubahan dan penghapusan di layar itu tidak meninggalkan
	// jejak di basis data; keterbatasan itu dicatat pada masterpasalhttp.Handler.
	clauseHandler, err := masterpasalhttp.NewHandler(masterpasalhttp.Options{
		Service:       assembly.masterPasal,
		Logger:        logger,
		WriteResponse: writeJSON,
		WriteError:    masterpasalhttp.ErrorWriter(writePortalAwareError),
	})
	if err != nil {
		return err
	}

	// Inbox Investigator (MENU_ID 48) — antrean pekerjaan di workbasket `InvestigatorPNC`.
	//
	// Ia TIDAK menerima Caller, dan itu akibat langsung dari bentuk layarnya: yang
	// ditampilkan adalah isi WORKBASKET, yaitu antrean bersama yang belum bertuan (`D-26`).
	// Tidak ada satu baris pun yang diturunkan dari identitas pemanggil.
	//
	// Itu berubah begitu layar kerjanya dibangun: mengambil pekerjaan dari antrean menuntut
	// identitas pengambilnya, dan modul itulah yang akan membutuhkannya.
	investigatorInboxHandler, err := inboxinvestigatorhttp.NewHandler(
		inboxinvestigatorhttp.Options{
			Service:       assembly.inboxInvestigator,
			Logger:        logger,
			WriteResponse: writeJSON,
			WriteError:    inboxinvestigatorhttp.ErrorWriter(writePortalAwareError),
		})
	if err != nil {
		return err
	}

	// Inbox Receive TKA (MENU_ID 49) atas POOLDATA.T_CLAIM_TKA_H.
	//
	// Ia TIDAK menerima Caller, dan berbeda dari modul baca-saja, di sini ketiadaannya
	// adalah UTANG yang disadari — bukan akibat bentuk layarnya. Modul ini mengubah
	// `TGLDOKLENGKAP` pada data klaim, sehingga siapa pelakunya adalah keterangan yang
	// seharusnya tercatat sebagai jejak audit.
	//
	// Modul Jejak Audit (`S-5`) belum ada dan daftar peristiwa wajib auditnya masih
	// ditunggu dari Compliance (`ADR-0026`). Sampai itu tiba, pelakunya hanya tercatat di
	// log aplikasi lewat middleware permintaan. Itu tidak memadai: `D-59` menetapkan
	// satuan izin adalah MENU dan tidak ada pemisahan tugas, sehingga jejak audit adalah
	// satu-satunya kontrol pengimbang yang tersisa.
	receiveTKAHandler, err := inboxreceivetkahttp.NewHandler(
		inboxreceivetkahttp.Options{
			Service:       assembly.inboxReceiveTKA,
			Logger:        logger,
			WriteResponse: writeJSON,
			WriteError:    inboxreceivetkahttp.ErrorWriter(writePortalAwareError),
		})
	if err != nil {
		return err
	}

	// Master Reas (MENU_ID 35) atas POOLDATA.T_REINSURER.
	//
	// Ia TIDAK menerima Caller, dan itu bukan kelalaian melainkan akibat langsung dari
	// modulnya yang hanya MEMBACA: tidak ada baris yang diturunkan dari identitas
	// pemanggil, dan tidak ada perubahan yang perlu dicatat pelakunya.
	//
	// Alasan modul ini hanya membaca ada pada banner paket masterreas — ringkasnya,
	// satu-satunya penulis tabel itu di sistem lama adalah alur PLA/DLA lewat
	// `Database/UPDATEREAS.prc`, dipanggil `UpdateDetailPLA2` dan `UpdateDetailDLA2`,
	// bukan layar master ini.
	reinsurerMemberHandler, err := masterreashttp.NewHandler(masterreashttp.Options{
		Service:       assembly.masterReas,
		Logger:        logger,
		WriteResponse: writeJSON,
		WriteError:    masterreashttp.ErrorWriter(writePortalAwareError),
	})
	if err != nil {
		return err
	}

	// Master Supplier memakai penulis galat yang SAMA dengan modul bisnis lain: ia
	// menyentuh basis data entitas, sehingga galat portal harus dijawab dengan kode yang
	// sudah dikenal frontend.
	supplierHandler, err := mastersupplierhttp.NewHandler(mastersupplierhttp.Options{
		Service: assembly.masterSupplier,
		// Jembatan satu arah dari modul auth, dipasang di sini supaya kedua modul tetap
		// tidak saling mengimpor.
		//
		// Login yang DIKETIK pengguna, bukan NIK — sama seperti modul master lain.
		// Berbeda dari Master Bengkel dan Master Panel, di modul ini ia BENAR-BENAR
		// TERSIMPAN: ia menjadi kunci USERKLAIMID di dalam dokumen supplier dan kolom
		// USER_REQ pada baris permintaan persetujuan. Keduanya ditulis sistem lama juga
		// (`CreateNewMasterSupplier_post` step 6), sehingga jejaknya bukan tambahan.
		Caller: func(ctx context.Context) (mastersupplierhttp.Caller, bool) {
			baseCtx, existing := authhttp.CallerFromContext(ctx)
			if !existing {
				return mastersupplierhttp.Caller{}, false
			}
			return mastersupplierhttp.Caller{Login: baseCtx.User.Login}, true
		},
		Logger:        logger,
		WriteResponse: writeJSON,
		WriteError:    mastersupplierhttp.ErrorWriter(writePortalAwareError),
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

	claimReportHandler := pelaporanklaimhttp.NewHandler(pelaporanklaimhttp.Options{
		Service: assembly.pelaporanKlaim,
		// Jembatan satu arah dari modul auth ke modul pelaporan klaim, dipasang di sini
		// supaya kedua modul tetap tidak saling mengimpor — yang tahu keduanya hanyalah
		// berkas perakitan ini.
		//
		// Kode cabang diambil dari profil pengguna, bukan dari badan permintaan.
		//
		// Di sistem lama ia dibaca `GetIDCabang` lewat DB Link `@ASMD`
		// (`Activity/CreateNewCaseRCV-Act.xml` step 6), dan API penggantinya (`D-25`,
		// `R-03`) belum ada. Yang dipakai sebagai gantinya adalah `Placement.BranchCode`
		// dari HCC/HCQ, yang memang sudah dipetakan ke `User.BranchCode` justru untuk
		// keperluan batas data per cabang (`11-SECURITY.md` §3.2, dicatat di
		// `keputusan-implementasi.md` §9.5).
		//
		// Ia KOSONG untuk pengguna non-karyawan — `POOLDATA.M_LOGIN_PNC` tidak memuat
		// cabang. Dalam keadaan itu nilai dari form yang dipakai; usecase menanganinya.
		GetCaller: func(ctx context.Context) (pelaporanklaimhttp.Caller, bool) {
			baseCtx, existing := authhttp.CallerFromContext(ctx)
			if !existing {
				return pelaporanklaimhttp.Caller{}, false
			}
			return pelaporanklaimhttp.Caller{
				Login:      baseCtx.User.Login,
				BranchCode: baseCtx.User.BranchCode,
			}, true
		},
		Logger:              logger,
		WriteJSON:           writeJSON,
		FallbackErrorWriter: pelaporanklaimhttp.ErrorWriter(writeAuthError),
	})

	// View History Claim. Jembatan pemanggilnya membawa LOGIN, bukan NIK: itulah yang
	// dicocokkan ke kolom LOGIN pada POOLDATA.MST_PROTEKSI_DATA_PNC, dan memakai NIK di
	// sini akan membuat setiap pengguna tampak belum terdaftar di gerbang proteksi.
	claimHistoryHandler := riwayatklaimhttp.NewHandler(riwayatklaimhttp.Options{
		Service: assembly.riwayatKlaim,
		GetCaller: func(ctx context.Context) (riwayatklaimhttp.Caller, bool) {
			baseCtx, existing := authhttp.CallerFromContext(ctx)
			if !existing {
				return riwayatklaimhttp.Caller{}, false
			}
			return riwayatklaimhttp.Caller{Login: baseCtx.User.Login}, true
		},
		Logger:    logger,
		WriteJSON: writeJSON,
		// Galat portal ikut dikenali, karena seluruh rute modul ini berada di balik
		// pemeriksaan portal.
		FallbackErrorWriter: riwayatklaimhttp.ErrorWriter(writePortalAwareError),
	})

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
				// Master Status Progres 2. SELURUH rutenya dipasangi pemeriksaan portal —
				// termasuk daftar induknya, yang dibaca dari tabel tingkat 1 milik entitas
				// yang bersangkutan, bukan daftar tetap milik aplikasi.
				masterstatusprogreshttp.Mount2(protected, progressStatus2Handler, activePortalDeps)
				// Master Penolakan Klaim. SATU pemasangan untuk DUA tab — Penolakan
				// Klaim dan Penolakan Komite — karena keduanya satu layar dan satu
				// butir menu (MENU_ID 25). Seluruh rutenya dipasangi pemeriksaan
				// portal; tidak ada satu pun yang isinya milik aplikasi.
				masterpenolakanhttp.Mount(protected, rejectionHandler, activePortalDeps)
				// Master Auto Claim. SELURUH rutenya dipasangi pemeriksaan portal —
				// master maupun ketiga lookup-nya dibaca dari basis data entitas, dan
				// dua entitas punya sumber bisnis serta daftar client yang berbeda.
				masterautoclaimhttp.Mount(protected, autoClaimHandler, activePortalDeps)
				// Master Bengkel. SELURUH rutenya dipasangi pemeriksaan portal —
				// POOLDATA.BENGKEL_HE dan ketiga tabel acuannya ada di basis data setiap
				// entitas, dan dua entitas punya daftar cabang yang berbeda.
				masterbengkelhttp.Mount(protected, workshopHandler, activePortalDeps)
				// Master Panel. Rutenya memasang pemeriksaan portal sendiri di dalam
				// Mount — SELURUHNYA kecuali daftar pilihan Lokasi dan Sisi, yang
				// isinya konstanta yang ditanam di activity Pega dan bukan dibaca dari
				// basis data entitas mana pun.
				masterpanelhttp.Mount(protected, panelHandler, activePortalDeps)
				// Master Sparepart. SELURUH rutenya dipasangi pemeriksaan portal —
				// termasuk daftar pilihannya, yang berbeda dari Master Panel: kategori
				// dan tipe suku cadang dibaca dari basis data entitas, bukan dari
				// konstanta yang ditanam di activity Pega.
				masterspareparthttp.Mount(protected, sparepartHandler, activePortalDeps)
				// Master Grouping Sparepart. SELURUH rutenya dipasangi pemeriksaan
				// portal — kedua tabel groupingnya DAN keempat sumber acuannya
				// (PANEL_HE, LOKASI_PANEL_HE, SPAREPART_HE, branddetail) ada di basis
				// data setiap entitas. Tidak ada satu pun rutenya yang isinya konstanta
				// milik aplikasi.
				mastergroupingspareparthttp.Mount(protected, groupingHandler, activePortalDeps)
				// Master Kategori Sparepart. SELURUH rutenya dipasangi pemeriksaan
				// portal — tabelnya ada di basis data setiap entitas, dan tidak ada
				// satu pun rutenya yang isinya milik aplikasi.
				//
				// Ia MENULIS tabel yang dibaca Master Sparepart di atas sebagai daftar
				// acuan. Satu tabel, satu penulis (P-1).
				masterkategorispareparthttp.Mount(protected, partCategoryHandler,
					activePortalDeps)
				// Master Tipe Sparepart. SELURUH rutenya dipasangi pemeriksaan portal —
				// tabelnya ada di basis data setiap entitas, dan `/pilihan` membaca
				// tabel kategori entitas itu juga.
				//
				// Ia MENULIS tabel yang dibaca Master Sparepart sebagai daftar acuan
				// Tipe, dan MEMBACA tabel yang ditulis Master Kategori Sparepart di
				// atas. Satu tabel, satu penulis (P-1).
				mastertipespareparthttp.Mount(protected, partTypeHandler,
					activePortalDeps)
				// Master Login. SELURUH rutenya dipasangi pemeriksaan portal —
				// POOLDATA.MST_LOGIN_SURVEYOR ada di basis data setiap entitas, dan
				// isinya menentukan siapa yang boleh bekerja sebagai surveyor pada
				// badan hukum itu. Tidak ada satu pun rutenya yang isinya milik
				// aplikasi: kelima isiannya diketik bebas, dan dua kolom sisanya
				// diturunkan server.
				masterloginhttp.Mount(protected, surveyorLoginHandler,
					activePortalDeps)
				// Detail Penyebab Kerugian. Rutenya memasang pemeriksaan portal sendiri
				// di dalam Mount — SELURUHNYA kecuali daftar Status Aktif, yang isinya
				// konstanta domain dan sama di keempat portal.
				//
				// Yang dipisahkan pemeriksaan itu bukan sekadar daftar acuan: sebab
				// kerugian yang dapat dipilih menentukan bagaimana klaim dinilai pada
				// badan hukum itu.
				detailpenyebabhttp.Mount(protected, causeOfLossDetailHandler,
					activePortalDeps)
				// Master Pasal Kerugian. Rutenya memasang pemeriksaan portal sendiri di
				// dalam Mount — SELURUHNYA kecuali daftar Kategori, yang isinya milik
				// aplikasi dan bukan dibaca dari basis data entitas mana pun.
				masterpasalhttp.Mount(protected, clauseHandler, activePortalDeps)

				// Master Pasal AI (MENU_ID 36). Baca-saja, satu rute.
				masterpasalaihttp.Mount(protected, clauseAIHandler, activePortalDeps)
				// Inbox Investigator (MENU_ID 48). Modul INBOX pertama, dan modul
				// pertama yang berada di bawah awalan `/inbox/...` — kelompok menu
				// tersendiri di sistem lama (`MENU_ID 2`, induk dari 30 butir).
				//
				// SELURUH rutenya dipasangi pemeriksaan portal: antrean pekerjaan ada
				// di basis data setiap entitas, dan barisnya memuat nama tertanggung
				// serta nama peserta klaim badan hukum itu.
				inboxinvestigatorhttp.Mount(protected, investigatorInboxHandler,
					activePortalDeps)
				// Inbox Receive TKA (MENU_ID 49). Modul INBOX kedua, dan yang
				// PERTAMA yang menulis.
				//
				// SELURUH rutenya dipasangi pemeriksaan portal, dan pada rute
				// tulisnya itu lebih dari sekadar mencegah kebocoran baca:
				// tanpa pemeriksaan portal, satu permintaan dapat mengubah
				// `TGLDOKLENGKAP` pada klaim milik badan hukum lain (`R-20`).
				inboxreceivetkahttp.Mount(protected, receiveTKAHandler,
					activePortalDeps)
				// Master Reas. SELURUH rutenya dipasangi pemeriksaan portal —
				// POOLDATA.T_REINSURER ada di basis data SETIAP entitas, dan isinya
				// menentukan mitra reasuransi mana yang menerima pemberitahuan klaim
				// badan hukum itu beserta surel tujuannya. Hanya rute BACA yang
				// terdaftar; lihat banner paket masterreas.
				masterreashttp.Mount(protected, reinsurerMemberHandler,
					activePortalDeps)
				// Master Supplier. SELURUH rutenya dipasangi pemeriksaan portal —
				// M_SUPPLIER dan keempat tabel acuannya ada di basis data setiap
				// entitas, dan dua entitas punya daftar cabang serta supplier yang
				// berbeda. Tidak ada satu pun rutenya yang isinya milik aplikasi.
				mastersupplierhttp.Mount(protected, supplierHandler, activePortalDeps)
				// Master rekening memuat nama, NIK, nomor rekening, dan surel pihak
				// ketiga; tidak satu pun boleh terbaca tanpa sesi.
				masterrekeninghttp.Mount(protected, accountHandler)
				// Master data juga berada di balik sesi. Pemeriksaan peran — "apakah
				// pemanggil memiliki menu Master Data" (D-59) — belum ada di sini
				// karena TKT-F3-004 dan TKT-F3-005 belum dikerjakan; keadaannya sama
				// dengan seluruh rute lain hari ini.
				masterstatushttp.Mount(protected, handlerMasterStatus)

				// Pelaporan Klaim memuat nama tertanggung, nomor polis, kronologi
				// kejadian, dan alamat surel pelapor. Tidak satu pun boleh terbaca
				// tanpa sesi.
				pelaporanklaimhttp.Mount(protected, claimReportHandler)

				// View History Claim memuat nama tertanggung, nomor polis, dan tanggal
				// lahir peserta — satu pencarian dapat mengembalikan seluruh riwayat
				// klaim seorang nasabah. Selain sesi, ia dijaga gerbang proteksi data
				// yang jatahnya berkurang tiap kali layar dibuka.
				riwayatklaimhttp.Mount(protected, claimHistoryHandler, activePortalDeps)
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

	// masterStatusProgres2 memakai pemilih repo per portal dengan alasan yang sama, dan
	// menerima pemilih tingkat 1 sebagai bahan kedua: penambahan tingkat 2 membaca baris
	// induknya untuk memastikan induk itu ada dan menyalin namanya.
	masterStatusProgres2 *masterstatusprogresusecase.Service2

	// masterAutoClaim melayani layar Master Auto Claim — daftar Sumber Bisnis yang
	// klaimnya boleh dibuat otomatis. Ia memakai pemilih repo per portal dengan alasan
	// yang sama seperti master lain: POOLDATA.M_AUTO_CLAIM_PNC beserta ketiga tabel
	// acuannya ada di basis data SETIAP entitas (ADR-0030).
	masterAutoClaim *masterautoclaimusecase.Service

	// masterBengkel melayani layar Master Bengkel — daftar bengkel rekanan beserta
	// syarat kerja samanya. Ia memakai pemilih repo per portal dengan alasan yang sama
	// seperti master lain: POOLDATA.BENGKEL_HE beserta ketiga tabel acuannya ada di basis
	// data SETIAP entitas (ADR-0030).
	masterBengkel *masterbengkelusecase.Service

	// masterPanel melayani layar Master Panel — daftar panel bodi kendaraan berat
	// beserta perlakuan klaim yang berlaku atasnya. Ia memakai pemilih repo per portal
	// dengan alasan yang sama seperti master lain: POOLDATA.PANEL_HE dan tabel anaknya
	// POOLDATA.LOKASI_PANEL_HE ada di basis data SETIAP entitas (ADR-0030).
	masterPanel *masterpanelusecase.Service

	// masterSparepart melayani layar Master Sparepart — daftar suku cadang alat berat
	// beserta harga, dimensi, dan batas stoknya. Ia memakai pemilih repo per portal dengan
	// alasan yang sama seperti master lain: POOLDATA.SPAREPART_HE beserta kedua tabel
	// acuannya ada di basis data SETIAP entitas (ADR-0030).
	masterSparepart *mastersparepartusecase.Service

	// masterGroupingSparepart melayani layar Master Grouping Sparepart — penautan suku
	// cadang ke panel bodi pada sebuah kendaraan, dikelompokkan menurut nomor rangka. Ia
	// memakai pemilih repo per portal dengan alasan yang sama seperti master lain:
	// POOLDATA.SPAREPART_HE_VIN_KEY, POOLDATA.SPAREPART_HE_VIN_GROUP, beserta keempat sumber
	// acuannya ada di basis data SETIAP entitas (ADR-0030).
	masterGroupingSparepart *mastergroupingsparepartusecase.Service

	// masterKategoriSparepart melayani layar Master Kategori Sparepart — penggolongan suku
	// cadang yang menjadi acuan Master Sparepart dan Master Tipe Sparepart. Ia memakai
	// pemilih repo per portal dengan alasan yang sama seperti master lain:
	// POOLDATA.GCNM_M_SPAREPART_CATEGORY ada di basis data SETIAP entitas (ADR-0030).
	masterKategoriSparepart *masterkategorisparepartusecase.Service

	// masterTipeSparepart melayani layar Master Tipe Sparepart — penggolongan tingkat kedua
	// di bawah kategori, yang menjadi acuan Tipe pada Master Sparepart. Ia memakai pemilih
	// repo per portal dengan alasan yang sama seperti master lain:
	// POOLDATA.GCNM_M_SPAREPART_TYPE ada di basis data SETIAP entitas (ADR-0030).
	masterTipeSparepart *mastertipesparepartusecase.Service

	// masterPasal melayani layar Master Pasal Kerugian — daftar baku butir ketentuan polis
	// yang dirujuk saat klaim dinilai. Ia memakai pemilih repo per portal dengan alasan
	// yang sama seperti master lain: POOLDATA.V_M_DATA_PASAL dan POOLDATA.BUSINESS ada di
	// basis data SETIAP entitas (ADR-0030).
	masterPasal *masterpasalusecase.Service

	// masterPasalAI melayani layar Master Pasal AI — daftar wording polis yang dipakai
	// penilaian AI atas klaim. Ia BACA-SAJA: layar lamanya tidak punya jalur tulis sama
	// sekali.
	//
	// Pemilih repo per portal dengan alasan yang sama seperti master lain: tabelnya ada di
	// basis data SETIAP entitas (ADR-0030).
	masterPasalAI *masterpasalaiusecase.Service

	// masterSupplier melayani layar Master Supplier — daftar supplier rekanan beserta
	// syarat dagangnya. Ia memakai pemilih repo per portal dengan alasan yang sama
	// seperti master lain: M_SUPPLIER dan keempat tabel acuannya ada di basis data
	// SETIAP entitas (ADR-0030).
	masterSupplier *mastersupplierusecase.Service

	// masterLogin melayani layar Master Login (MENU_ID 37) atas
	// POOLDATA.MST_LOGIN_SURVEYOR. Ia memakai pemilih repo per portal dengan alasan yang
	// sama seperti master lainnya — tabelnya ada di basis data SETIAP entitas (ADR-0030) —
	// ditambah satu yang khas modul ini: isinya menentukan siapa yang boleh bekerja sebagai
	// surveyor pada badan hukum itu.
	masterLogin *masterloginusecase.Service

	// detailPenyebab melayani layar Detail Penyebab Kerugian (MENU_ID 38) atas
	// POOLDATA.D_CAUSE_OF_LOSS — rincian di bawah Master Penyebab Kerugian, yang menjadi
	// pilihan petugas saat klaim diregistrasi. Ia memakai pemilih repo per portal dengan
	// alasan yang sama seperti master lainnya: tabelnya ada di basis data SETIAP entitas
	// (ADR-0030).
	detailPenyebab *detailpenyebabusecase.Service

	// inboxInvestigator melayani layar Inbox Investigator (MENU_ID 48) — antrean pekerjaan
	// yang menunggu di workbasket `InvestigatorPNC`.
	//
	// Modul INBOX pertama di aplikasi ini; pembedaan inbox dari layar master dan layar
	// pencarian ditetapkan `D-79`. Ia memakai pemilih repo per portal dengan alasan yang
	// sama seperti modul lain, dan di sini taruhannya termasuk yang tertinggi: antrean satu
	// badan hukum memuat nama tertanggung dan nama peserta klaimnya (ADR-0030, R-20).
	inboxInvestigator *inboxinvestigatorusecase.Service

	// inboxReceiveTKA melayani layar Inbox Receive TKA (MENU_ID 49) — klaim TKA yang
	// tanggal kelengkapan dokumennya belum diisi.
	//
	// Modul INBOX kedua, dan yang PERTAMA yang menulis. Ia mengubah `TGLDOKLENGKAP` pada
	// data klaim, sehingga taruhan pemilih repo per portal di sini melampaui kebocoran
	// baca: portal yang keliru berarti mengubah tanggal pada klaim milik badan hukum lain
	// (ADR-0030, R-20).
	inboxReceiveTKA *inboxreceivetkausecase.Service

	// masterReas melayani layar Master Reas (MENU_ID 35) atas POOLDATA.T_REINSURER —
	// daftar mitra reasuransi penerima pemberitahuan PLA, Pre-DLA, dan DLA. Ia memakai
	// pemilih repo per portal dengan alasan yang sama seperti master lainnya: tabelnya ada
	// di basis data SETIAP entitas (ADR-0030).
	//
	// Satu-satunya layanan master yang HANYA MEMBACA; alasannya ada pada banner paket
	// masterreas.
	masterReas *masterreasusecase.Service

	// masterPenolakan melayani tab Penolakan Klaim pada layar Master Penolakan Klaim.
	// Ia memakai pemilih repo per portal dengan alasan yang sama seperti master status
	// progres: kedua tabelnya ada di basis data SETIAP entitas (ADR-0030).
	masterPenolakan *masterpenolakanusecase.Service

	// masterPenolakanKomite melayani tab Penolakan Komite pada layar yang SAMA. Ia
	// layanan tersendiri karena tabelnya tidak sekerabat dan tidak punya satu pun kolom
	// yang menghubungkannya dengan kedua tabel di atas.
	masterPenolakanKomite *masterpenolakanusecase.ServiceKomite

	// menu menyusun peta menu beserta kewenangan pemakainya.
	menu *menuusecase.Service

	// pelaporanKlaim melayani alur Pelaporan Klaim (Receive Document).
	pelaporanKlaim *pelaporanklaimusecase.Service

	// riwayatKlaim melayani layar View History Claim (`MENU_ID 76`).
	riwayatKlaim *riwayatklaimusecase.Service

	readyAliases func() []string
	close        func()
}

// storage memegang seluruh repo yang sudah terpasang di atas sumbernya.
type storage struct {
	user           auth.UserRepo
	session        auth.SessionRepo
	portal         portal.Repo
	masterStatus   masterstatus.Repo
	pelaporanKlaim pelaporanklaim.Repo

	account     masterrekening.Repo
	accountBank masterrekening.BankRepo

	// rekeningDiOracle menyatakan master rekening dipasang di atas POOLDATA.LST_ACCOUNT
	// yang sungguhan, bukan di memori. Adapter tiruan yang menulis jejak karangan
	// dilarang di atasnya — lihat rakitMasterRekening.
	accountInOracle bool

	// warisan bernilai nil bila koneksi Oracle tidak dibuka. Ia memberi akses baca ke
	// tiga tabel milik sistem lama: M_PORTAL_PNC, M_LOGIN_PNC, dan GCNM_CONNECT_REST.
	legacy *sqlstore.Legacy

	// claimHistorySelector dan claimProtectionSelector memilih penyimpanan View History
	// Claim milik satu portal.
	//
	// Keduanya fungsi, bukan repo tunggal, karena riwayat klaim DAN jatah proteksi
	// seorang pengguna adalah data bisnis milik satu badan hukum (`ADR-0030`). Satu repo
	// bersama akan membaca riwayat satu entitas dari basis data entitas lain — kebocoran
	// lintas badan hukum yang justru dicegah `R-20`.
	claimHistorySelector    riwayatklaim.RepoSelector
	claimProtectionSelector riwayatklaim.ProtectionRepoSelector

	// progressStatusSelector memilih penyimpanan master status progres milik satu portal.
	//
	// Ia fungsi, bukan repo tunggal, karena tabelnya ada di basis data SETIAP entitas
	// (ADR-0030). Satu repo bersama akan menulis data seluruh entitas ke satu tempat,
	// kebocoran lintas badan hukum yang justru dicegah R-20.
	progressStatusSelector masterstatusprogres.RepoSelector

	// progressStatus2Selector memilih penyimpanan tingkat 2 milik satu portal, dengan
	// alasan yang sama persis: POOLDATA.GCNM_MST_PROGRESS ada di basis data setiap entitas.
	progressStatus2Selector masterstatusprogres.RepoSelector2

	// clauseAISelector memilih penyimpanan Master Pasal AI milik satu portal.
	//
	// POOLDATA.MST_PASAL_AI ada di basis data setiap entitas (ADR-0030), sehingga satu repo
	// bersama akan menampilkan wording polis satu badan hukum kepada pengguna badan hukum
	// lain — kebocoran yang justru dicegah R-20.
	clauseAISelector masterpasalai.RepoSelector

	// autoClaimSelector memilih penyimpanan Master Auto Claim milik satu portal.
	// POOLDATA.M_AUTO_CLAIM_PNC dan ketiga tabel acuannya ada di basis data setiap
	// entitas.
	autoClaimSelector masterautoclaim.RepoSelector

	// workshopSelector memilih penyimpanan Master Bengkel milik satu portal.
	// POOLDATA.BENGKEL_HE dan ketiga tabel acuannya ada di basis data setiap entitas.
	workshopSelector masterbengkel.RepoSelector

	// panelSelector memilih penyimpanan Master Panel milik satu portal.
	// POOLDATA.PANEL_HE dan tabel anaknya POOLDATA.LOKASI_PANEL_HE ada di basis data
	// setiap entitas.
	panelSelector masterpanel.RepoSelector

	// sparepartSelector memilih penyimpanan Master Sparepart milik satu portal.
	sparepartSelector mastersparepart.RepoSelector

	// groupingSelector memilih penyimpanan Master Grouping Sparepart milik satu portal.
	// Kedua tabel groupingnya dan keempat sumber acuannya ada di basis data setiap entitas.
	groupingSelector mastergroupingsparepart.RepoSelector

	// partCategorySelector memilih penyimpanan Master Kategori Sparepart milik satu portal.
	// POOLDATA.GCNM_M_SPAREPART_CATEGORY ada di basis data setiap entitas — tabel yang sama
	// yang dibaca sparepartSelector sebagai daftar acuan, dan ditulis oleh yang ini.
	partCategorySelector masterkategorisparepart.RepoSelector

	// partTypeSelector memilih penyimpanan Master Tipe Sparepart milik satu portal.
	// POOLDATA.GCNM_M_SPAREPART_TYPE ada di basis data setiap entitas — tabel yang sama yang
	// dibaca sparepartSelector sebagai daftar acuan Tipe, dan ditulis oleh yang ini.
	//
	// Repo yang sama juga MEMBACA GCNM_M_SPAREPART_CATEGORY untuk dropdown Kategorinya;
	// yang menulis tabel itu tetap partCategorySelector (P-1).
	partTypeSelector mastertipesparepart.RepoSelector

	// surveyorLoginSelector memilih penyimpanan Master Login milik satu portal.
	//
	// POOLDATA.MST_LOGIN_SURVEYOR ada di basis data setiap entitas, dan pemisahannya di
	// sini lebih berarti daripada pada master penggolongan: isinya menentukan siapa yang
	// boleh bekerja sebagai surveyor pada satu badan hukum. Baris yang bocor ke portal lain
	// bukan sekadar data yang salah tempat — ia orang yang muncul di daftar entitas yang
	// bukan haknya (R-20).
	surveyorLoginSelector masterlogin.RepoSelector

	// causeOfLossDetailSelector memilih penyimpanan Detail Penyebab Kerugian milik satu
	// portal. POOLDATA.D_CAUSE_OF_LOSS ada di basis data setiap entitas, dan isinya
	// menentukan sebab kerugian apa saja yang dapat dipilih pada badan hukum itu — sehingga
	// baris yang bocor ke portal lain mengubah cara klaim entitas itu dinilai, bukan
	// sekadar menampilkan baris yang salah tempat (R-20).
	causeOfLossDetailSelector detailpenyebab.RepoSelector

	// investigatorInboxSelector memilih penyimpanan Inbox Investigator milik satu portal.
	// Antrean pekerjaan ada di basis data setiap entitas; baris yang bocor ke portal lain
	// menampakkan nama tertanggung dan nama peserta klaim badan hukum yang bukan haknya
	// (R-20).
	investigatorInboxSelector inboxinvestigator.RepoSelector

	// receiveTKASelector memilih penyimpanan Inbox Receive TKA milik satu portal.
	// POOLDATA.T_CLAIM_TKA_H ada di basis data setiap entitas, dan modul ini MENULIS —
	// pemilih yang keliru bukan hanya menampakkan pekerjaan badan hukum lain, melainkan
	// mengubah tanggal pada klaimnya (R-20).
	receiveTKASelector inboxreceivetka.RepoSelector

	// reinsurerMemberSelector memilih penyimpanan Master Reas milik satu portal.
	// POOLDATA.T_REINSURER ada di basis data setiap entitas, dan isinya menentukan ke mana
	// pemberitahuan klaim satu badan hukum dikirim. Baris yang bocor ke portal lain
	// menampakkan daftar mitra reasuransi beserta surel tujuannya kepada entitas yang bukan
	// haknya (R-20).
	reinsurerMemberSelector masterreas.RepoSelector

	// clauseSelector memilih penyimpanan Master Pasal Kerugian milik satu portal.
	// POOLDATA.V_M_DATA_PASAL dan master lini bisnisnya ada di basis data setiap entitas.
	clauseSelector masterpasal.RepoSelector

	// supplierSelector memilih penyimpanan Master Supplier milik satu portal.
	// M_SUPPLIER, antrean POOLDATA.PROTEKSI_KLAIMMBU, dan keempat tabel acuannya ada di
	// basis data setiap entitas.
	supplierSelector mastersupplier.RepoSelector

	// rejectionSelector memilih penyimpanan Master Penolakan Klaim milik satu portal.
	// POOLDATA.MST_PENOLAKAN_KLAIM_1 dan _2 ada di basis data setiap entitas.
	rejectionSelector masterpenolakan.RepoSelector

	// rejectionKomiteSelector memilih penyimpanan Master Penolakan Komite milik satu
	// portal — POOLDATA.MST_REJECTED_KOMITE, juga per entitas.
	rejectionKomiteSelector masterpenolakan.RepoSelectorKomite

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

	clauseAIService, err := masterpasalaiusecase.NewService(masterpasalaiusecase.Options{
		RepoSelector: store.clauseAISelector,
	})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	progressStatus2Service, err := masterstatusprogresusecase.NewService2(masterstatusprogresusecase.Options2{
		RepoSelector:   store.progressStatus2Selector,
		ParentSelector: store.progressStatusSelector,
	})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	autoClaimService, err := masterautoclaimusecase.NewService(masterautoclaimusecase.Options{
		RepoSelector: store.autoClaimSelector,
	})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	// Master Pasal Kerugian TIDAK menerima Clock: tabelnya tidak punya kolom waktu maupun
	// kolom pelaku sama sekali — hanya IDDATA, IDPASAL, dan JSONPASAL.
	clauseService, err := masterpasalusecase.NewService(masterpasalusecase.Options{
		RepoSelector: store.clauseSelector,
	})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	// Master Supplier MENERIMA Clock, berbeda dari Master Pasal Kerugian dan Master
	// Bengkel. Ia menulis dua jejak waktu: kunci TGL_INSERT di dalam dokumen supplier —
	// dalam bentuk teks dd/MM/yyyy zona WIB, terikat karena dokumennya dibaca bersama Pega
	// — dan kolom TGL_INPUT pada baris permintaan persetujuan, yang bentuknya bebas karena
	// tidak dibaca layar mana pun.
	supplierService, err := mastersupplierusecase.NewService(mastersupplierusecase.Options{
		RepoSelector: store.supplierSelector,
		Clock:        clock.System{},
	})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	workshopService, err := masterbengkelusecase.NewService(masterbengkelusecase.Options{
		RepoSelector: store.workshopSelector,
	})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	// Master Panel TIDAK menerima Clock: kedua tabelnya tidak punya kolom waktu maupun
	// kolom pelaku sama sekali — kelima belas kolom induknya terbaca lengkap dari
	// BrowseMasterPanel_HE_RD, dan tidak satu pun menyebut siapa atau kapan.
	panelService, err := masterpanelusecase.NewService(masterpanelusecase.Options{
		RepoSelector: store.panelSelector,
	})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	// Master Sparepart MENERIMA Clock, berbeda dari Master Panel dan Master Bengkel:
	// POOLDATA.SPAREPART_HE punya kolom TGL_UPDATE_HARGA, dan
	// `Activity/UpdateSparepartHE_act` mengisinya dengan @DateTime.CurrentDateTime().
	// Jam yang sama dipakai modul auth, sehingga waktu di seluruh aplikasi berasal dari satu
	// sumber dan tidak ada satu pun penambahan 7 jam manual yang menyelinap masuk.
	sparepartService, err := mastersparepartusecase.NewService(mastersparepartusecase.Options{
		RepoSelector: store.sparepartSelector,
		Clock:        clock.System{},
	})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	// Master Grouping Sparepart TIDAK menerima Clock, berbeda dari Master Sparepart: kedua
	// tabelnya tidak punya satu pun kolom waktu, dan `Activity/UpdateGroupingSparepartHE_act`
	// tidak memanggil @DateTime.CurrentDateTime() sama sekali. Menyuntikkan jam yang tidak
	// akan pernah dipakai hanya akan menyesatkan pembaca berikutnya.
	groupingService, err := mastergroupingsparepartusecase.NewService(
		mastergroupingsparepartusecase.Options{
			RepoSelector: store.groupingSelector,
		})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	// Master Kategori Sparepart TIDAK menerima Clock, berbeda dari Master Sparepart yang
	// bertetangga dengannya: POOLDATA.GCNM_M_SPAREPART_CATEGORY tidak punya satu pun kolom
	// waktu, sehingga tidak ada yang perlu distempel. Menyerahkan jam yang tidak pernah
	// dipakai hanya akan menyesatkan pembaca berikutnya.
	partCategoryService, err := masterkategorisparepartusecase.NewService(
		masterkategorisparepartusecase.Options{
			RepoSelector: store.partCategorySelector,
		})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	// Master Tipe Sparepart juga TIDAK menerima Clock, dengan alasan yang sama seperti
	// tetangganya di atas: POOLDATA.GCNM_M_SPAREPART_TYPE tidak punya satu pun kolom waktu
	// maupun kolom pelaku.
	partTypeService, err := mastertipesparepartusecase.NewService(
		mastertipesparepartusecase.Options{
			RepoSelector: store.partTypeSelector,
		})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	// Master Login juga TIDAK menerima Clock: POOLDATA.MST_LOGIN_SURVEYOR tidak punya satu
	// pun kolom waktu maupun kolom pelaku. Ketujuh kolomnya terbaca lengkap dari ketiga
	// rule SQL yang menyentuhnya, dan tidak satu pun menampung siapa atau kapan.
	surveyorLoginService, err := masterloginusecase.NewService(
		masterloginusecase.Options{
			RepoSelector: store.surveyorLoginSelector,
		})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	// Detail Penyebab Kerugian juga TIDAK menerima Clock, dan alasannya sama:
	// POOLDATA.D_CAUSE_OF_LOSS hanya punya D_COL_ID dan JSONDATA — tidak ada satu pun kolom
	// waktu maupun kolom pelaku untuk distempel.
	causeOfLossDetailService, err := detailpenyebabusecase.NewService(
		detailpenyebabusecase.Options{
			RepoSelector: store.causeOfLossDetailSelector,
		})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	// Inbox Investigator TIDAK menerima Clock maupun Logger: ia hanya membaca, dan tidak
	// ada satu pun nilainya yang bergantung pada jam dinding. Kolom "Lama Masuk Inbox"
	// menampilkan tanggal survei apa adanya — bukan durasi yang dihitung. Lihat
	// inboxinvestigator.Task.SurveyDate.
	investigatorInboxService, err := inboxinvestigatorusecase.NewService(
		inboxinvestigatorusecase.Options{
			RepoSelector: store.investigatorInboxSelector,
		})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	// Inbox Receive TKA MENERIMA Logger — berbeda dari Inbox Investigator — karena ia
	// menulis: pengisian tanggal adalah peristiwa bisnis, dan sampai modul Jejak Audit
	// (`S-5`) ada, log aplikasi adalah satu-satunya tempat peristiwa itu tercatat.
	//
	// Ia TIDAK menerima Clock. Satu-satunya tanggal yang ditulisnya datang dari pengguna,
	// persis seperti `Param.Tanggal` pada activity lama; tidak ada nilai yang bergantung
	// pada jam dinding.
	receiveTKAService, err := inboxreceivetkausecase.NewService(
		inboxreceivetkausecase.Options{
			RepoSelector: store.receiveTKASelector,
			Notifier:     buildReceiveTKANotifier(cfg, logger),
			Logger:       logger,
		})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	// Master Reas TIDAK menerima Clock maupun Logger di layanannya: ia hanya membaca,
	// sehingga tidak ada peristiwa yang perlu distempel maupun dicatat. Lihat
	// masterreasusecase.NewService.
	reinsurerMemberService, err := masterreasusecase.NewService(
		masterreasusecase.Options{
			RepoSelector: store.reinsurerMemberSelector,
		})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	// Master Penolakan Klaim menerima Clock karena ia menulis kolom TANGGALKIRIM.
	// Jam yang sama dipakai modul auth, sehingga waktu di seluruh aplikasi berasal dari
	// satu sumber dan tidak ada satu pun penambahan 7 jam manual yang menyelinap masuk.
	rejectionService, err := masterpenolakanusecase.NewService(masterpenolakanusecase.Options{
		RepoSelector: store.rejectionSelector,
		Clock:        clock.System{},
	})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	// Tab Penolakan Komite TIDAK menerima Clock: tabelnya tidak punya kolom waktu maupun
	// kolom pelaku sama sekali.
	rejectionKomiteService, err := masterpenolakanusecase.NewServiceKomite(masterpenolakanusecase.OptionsKomite{
		RepoSelector: store.rejectionKomiteSelector,
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

	claimReportService, err := pelaporanklaimusecase.NewService(pelaporanklaimusecase.Options{
		Repo:  store.pelaporanKlaim,
		Clock: clock.System{},
	})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	claimHistoryService, err := riwayatklaimusecase.NewService(riwayatklaimusecase.Options{
		RepoSelector:       store.claimHistorySelector,
		ProtectionSelector: store.claimProtectionSelector,
		Clock:              clock.System{},
	})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	return assembly{
		auth:                 service,
		portal:               store.portal,
		masterRekening:       buildMasterRekening(cfg, store, logger),
		masterStatus:         claimStatusService,
		masterStatusProgres:  progressStatusService,
		masterStatusProgres2: progressStatus2Service,
		masterPasalAI:        clauseAIService,

		masterAutoClaim:         autoClaimService,
		masterBengkel:           workshopService,
		masterPanel:             panelService,
		masterSparepart:         sparepartService,
		masterGroupingSparepart: groupingService,
		masterKategoriSparepart: partCategoryService,
		masterTipeSparepart:     partTypeService,
		masterLogin:             surveyorLoginService,
		detailPenyebab:          causeOfLossDetailService,
		inboxInvestigator:       investigatorInboxService,
		inboxReceiveTKA:         receiveTKAService,
		masterReas:              reinsurerMemberService,
		masterPasal:             clauseService,
		masterSupplier:          supplierService,

		masterPenolakan:       rejectionService,
		masterPenolakanKomite: rejectionKomiteService,

		pelaporanKlaim: claimReportService,
		riwayatKlaim:   claimHistoryService,

		menu:         menuService,
		readyAliases: store.readyAliases,
		close:        store.close,
	}, nil
}

// buildMasterRekening menyusun modul Master Rekening di balik seam-nya.
//
// Seam Kasir diisi klien HTTP nyata bila alamatnya sudah dikonfigurasi, dan tiruan bila
// belum. Perbedaannya diumumkan di log: layar yang tampak bekerja padahal pendaftaran
// ke Kasir tidak pernah terjadi adalah kegagalan yang tidak terlihat siapa pun sampai
// pembayaran pertama tertahan.
// buildReceiveTKANotifier menyiapkan pengirim pemberitahuan kelengkapan dokumen TKA.
//
// # nil BUKAN kegagalan, dan itu sengaja
//
// Modul lain memakai pengirim TIRUAN ketika SMTP belum dikonfigurasi. Di sini yang
// dikembalikan justru nil, dan bedanya terlihat oleh pengguna: layar membedakan
// "pemberitahuan belum dipasang" dari "pemberitahuan gagal dikirim", dan pengirim tiruan
// yang selalu berhasil akan melaporkan keadaan KETIGA yang tidak benar — bahwa surelnya
// terkirim.
//
// Pengisian tanggalnya sendiri tetap berjalan penuh. Pemberitahuan adalah akibat samping,
// bukan syarat; lihat usecase.Complete.
//
// # Penerimanya TIDAK pernah diturunkan dari siapa yang menekan tombol
//
// `Activity/SubmitTanggalLengkapTKA-Act.xml` memilih penerimanya dengan bercabang pada tiga
// Operator ID yang tertanam di dalam rule, salah satunya menunjuk akun surel pribadi di
// jalur produksi. Percabangan itu dicabut (`D-15`, `D-67`); yang berlaku adalah satu daftar
// dari `SMTP_PENERIMA_TKA`.
func buildReceiveTKANotifier(
	cfg config.Config,
	logger *slog.Logger,
) inboxreceivetka.Notifier {
	if !cfg.SMTP.TKAActive() {
		logger.Warn("pemberitahuan kelengkapan dokumen TKA tidak aktif",
			slog.String("akibat",
				"tanggal tetap tersimpan, tetapi tidak ada yang diberi tahu lewat surel"),
			slog.String("perbaikan",
				"isi SMTP_HOST, SMTP_PORT, SMTP_DARI, dan SMTP_PENERIMA_TKA"))
		return nil
	}

	return inboxreceivetkanotif.NewSender(inboxreceivetkanotif.Config{
		Host:     cfg.SMTP.Host,
		Port:     cfg.SMTP.Port,
		User:     cfg.SMTP.User,
		Password: cfg.SMTP.Password,
		From:     cfg.SMTP.From,
		To:       cfg.SMTP.TKARecipients,
		Timeout:  cfg.SMTP.Timeout,
	})
}

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
		store.pelaporanKlaim = pelaporanklaimsql.NewRepo(primary)
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
		store.progressStatus2Selector = func(alias string) (masterstatusprogres.Repo2, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return masterstatusprogressql.NewRepo2(conn), nil
		}
		store.autoClaimSelector = func(alias string) (masterautoclaim.Store, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return masterautoclaimsql.NewRepo(conn), nil
		}
		store.workshopSelector = func(alias string) (masterbengkel.Store, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return masterbengkelsql.NewRepo(conn), nil
		}
		store.panelSelector = func(alias string) (masterpanel.Store, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return masterpanelsql.NewRepo(conn), nil
		}
		store.sparepartSelector = func(alias string) (mastersparepart.Store, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return mastersparepartsql.NewRepo(conn), nil
		}
		store.groupingSelector = func(alias string) (mastergroupingsparepart.Store, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return mastergroupingsparepartsql.NewRepo(conn), nil
		}
		store.partCategorySelector = func(alias string) (masterkategorisparepart.Store, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return masterkategorisparepartsql.NewRepo(conn), nil
		}
		store.partTypeSelector = func(alias string) (mastertipesparepart.Store, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return mastertipesparepartsql.NewRepo(conn), nil
		}
		store.surveyorLoginSelector = func(alias string) (masterlogin.Repo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return masterloginsql.NewRepo(conn), nil
		}
		store.clauseAISelector = func(alias string) (masterpasalai.Repo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return masterpasalaisql.NewRepo(conn), nil
		}
		store.causeOfLossDetailSelector = func(alias string) (detailpenyebab.Store, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return detailpenyebabsql.NewRepo(conn), nil
		}
		store.investigatorInboxSelector = func(alias string) (inboxinvestigator.Repo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return inboxinvestigatorsql.NewRepo(conn), nil
		}
		store.receiveTKASelector = func(alias string) (inboxreceivetka.Repo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return inboxreceivetkasql.NewRepo(conn), nil
		}
		store.reinsurerMemberSelector = func(alias string) (masterreas.Repo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return masterreassql.NewRepo(conn), nil
		}
		store.clauseSelector = func(alias string) (masterpasal.Store, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return masterpasalsql.NewRepo(conn), nil
		}
		store.supplierSelector = func(alias string) (mastersupplier.Store, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return mastersuppliersql.NewRepo(conn), nil
		}
		store.rejectionSelector = func(alias string) (masterpenolakan.Repo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return masterpenolakansql.NewRepo(conn), nil
		}
		store.rejectionKomiteSelector = func(alias string) (masterpenolakan.RepoKomite, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return masterpenolakansql.NewRepoKomite(conn), nil
		}

		store.claimHistorySelector = func(alias string) (riwayatklaim.Repo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return riwayatklaimsql.NewRepo(conn), nil
		}

		store.claimProtectionSelector = func(alias string) (riwayatklaim.ProtectionRepo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return riwayatklaimsql.NewProtectionRepo(conn), nil
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
		store.progressStatus2Selector = progressStatus2SelectorMemory(store.progressStatusSelector)
		store.autoClaimSelector = autoClaimSelectorMemory(cfg.PrimaryPortal)
		store.workshopSelector = workshopSelectorMemory(cfg.PrimaryPortal)
		store.panelSelector = panelSelectorMemory(cfg.PrimaryPortal)
		store.sparepartSelector = sparepartSelectorMemory(cfg.PrimaryPortal)
		store.groupingSelector = groupingSelectorMemory(cfg.PrimaryPortal)
		store.partCategorySelector = partCategorySelectorMemory(cfg.PrimaryPortal)
		store.partTypeSelector = partTypeSelectorMemory(cfg.PrimaryPortal)
		store.surveyorLoginSelector = surveyorLoginSelectorMemory(cfg.PrimaryPortal)
		store.clauseAISelector = clauseAISelectorMemory(cfg.PrimaryPortal)
		store.causeOfLossDetailSelector = causeOfLossDetailSelectorMemory(cfg.PrimaryPortal)
		store.reinsurerMemberSelector = reinsurerMemberSelectorMemory(cfg.PrimaryPortal)
		store.investigatorInboxSelector = investigatorInboxSelectorMemory(cfg.PrimaryPortal)
		store.receiveTKASelector = receiveTKASelectorMemory(cfg.PrimaryPortal)
		store.clauseSelector = clauseSelectorMemory(cfg.PrimaryPortal)
		store.supplierSelector = supplierSelectorMemory(cfg.PrimaryPortal)
		store.rejectionSelector = rejectionSelectorMemory(cfg.PrimaryPortal)
		store.rejectionKomiteSelector = rejectionKomiteSelectorMemory(cfg.PrimaryPortal)
		// NewDevRepo, bukan NewSampleRepo: isi contoh m_login_group_pnc.csv hanya
		// memuat satu login, dan login provider tiruan tidak ada di dalamnya. Tanpa
		// itu, masuk saat pengembangan menghasilkan menu kosong yang tampak rusak.
		store.menu = menumemory.NewDevRepo()
		// Laporan contoh mencakup kelima tahap, sehingga seluruh tab layar Pelaporan
		// Klaim dapat dicoba tanpa Oracle dan tanpa menunggu migrasi 0003. Seluruh
		// isinya karangan — lihat repo/memory/sample.go.
		store.pelaporanKlaim = pelaporanklaimmemory.NewRepo(pelaporanklaimmemory.SampleReports()...)
		store.claimHistorySelector = claimHistorySelectorMemory(cfg.PrimaryPortal)
		store.claimProtectionSelector = claimProtectionSelectorMemory(cfg.PrimaryPortal)
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

// progressStatus2SelectorMemory menyusun penyimpanan tingkat 2 di memori.
//
// Ia TIDAK memeriksa alias portalnya sendiri, melainkan menanyakannya ke pemilih tingkat
// 1: portal yang ditolak di sana ditolak di sini dengan galat yang sama persis. Dua
// pemeriksaan terpisah atas hal yang sama akan berbeda begitu salah satunya disunting —
// dan yang dipertaruhkan pada R-20 adalah pemisahan data antar badan hukum.
//
// Repo tingkat 1 yang dikembalikan pemilih itu DIPAKAI LANGSUNG sebagai induk, bukan
// disalin. Dengan begitu status progres 1 yang baru ditambahkan lewat layarnya langsung
// muncul di dropdown tingkat 2 — perilaku yang sama dengan adapter SQL, yang membaca
// tabel induk di dalam transaksi yang sama.
func progressStatus2SelectorMemory(parentSelector masterstatusprogres.RepoSelector) masterstatusprogres.RepoSelector2 {
	var lock sync.Mutex
	store := map[string]masterstatusprogres.Repo2{}

	return func(alias string) (masterstatusprogres.Repo2, error) {
		parent, err := parentSelector(alias)
		if err != nil {
			return nil, err
		}

		memoryParent, usable := parent.(*masterstatusprogresmemory.Repo)
		if !usable {
			// Tidak mungkin terjadi pada rakitan yang ada; dinyatakan supaya cacat
			// perakitan gagal keras, bukan diam-diam menyajikan induk yang kosong.
			return nil, fmt.Errorf("perakitan: repo tingkat 1 portal %q bukan adapter memori", alias)
		}

		clean := strings.ToUpper(strings.TrimSpace(alias))

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := masterstatusprogresmemory.NewRepo2(memoryParent, masterstatusprogresmemory.SampleList2()...)
		store[clean] = fresh
		return fresh, nil
	}
}

// autoClaimSelectorMemory menyusun penyimpanan Master Auto Claim di memori.
//
// Satu portal mendapat satu penyimpanan, dibuat saat pertama diminta lalu dipakai
// kembali. Kalau dibuat ulang setiap permintaan, penambahan yang baru disimpan akan
// hilang pada permintaan berikutnya dan layarnya tampak rusak tanpa sebab.
//
// Hanya portal utama yang dilayani, sejalan dengan readyAliases pada cabang tanpa
// Oracle. Memilih portal lain tanpa basis data karena itu ditolak dengan galat yang
// sama seperti di produksi.
//
// Penyimpanan contoh memuat ketiga tabel acuannya sekaligus — sumber bisnis, client,
// dan bank — sehingga seluruh alur layar dapat dicoba tanpa Oracle: mencari sumber
// bisnis, menambah, menolak yang sudah ada, lalu menyetujui lewat tab komite.
func autoClaimSelectorMemory(primaryAlias string) masterautoclaim.RepoSelector {
	var lock sync.Mutex
	store := map[string]masterautoclaim.Store{}

	return func(alias string) (masterautoclaim.Store, error) {
		clean, err := memoryPortal(alias, primaryAlias)
		if err != nil {
			return nil, err
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := masterautoclaimmemory.NewSampleRepo()
		store[clean] = fresh
		return fresh, nil
	}
}

// workshopSelectorMemory menyusun penyimpanan Master Bengkel di memori.
//
// Satu portal mendapat satu penyimpanan, dibuat saat pertama diminta lalu dipakai
// kembali. Kalau dibuat ulang setiap permintaan, penambahan yang baru disimpan akan
// hilang pada permintaan berikutnya dan layarnya tampak rusak tanpa sebab — dan pada
// modul ini akibatnya lebih jauh: nomor urut ID_BENGKEL ikut mundur, sehingga dua
// bengkel dapat lahir dengan kunci yang sama.
//
// Hanya portal utama yang dilayani, sejalan dengan readyAliases pada cabang tanpa
// Oracle. Memilih portal lain tanpa basis data karena itu ditolak dengan galat yang sama
// seperti di produksi.
//
// Penyimpanan contoh memuat ketiga tabel acuannya sekaligus — cabang, kota, dan bank —
// sehingga seluruh alur layar dapat dicoba tanpa Oracle: menambah, menolak nama yang
// sudah ada, menyunting, lalu menyetujui borongan lewat tab Waiting Approval.
func workshopSelectorMemory(primaryAlias string) masterbengkel.RepoSelector {
	var lock sync.Mutex
	store := map[string]masterbengkel.Store{}

	return func(alias string) (masterbengkel.Store, error) {
		clean, err := memoryPortal(alias, primaryAlias)
		if err != nil {
			return nil, err
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := masterbengkelmemory.NewSampleRepo()
		store[clean] = fresh
		return fresh, nil
	}
}

// panelSelectorMemory menyusun penyimpanan Master Panel di memori.
//
// Satu portal mendapat satu penyimpanan, dibuat saat pertama diminta lalu dipakai
// kembali. Kalau dibuat ulang setiap permintaan, penambahan yang baru disimpan akan hilang
// pada permintaan berikutnya dan layarnya tampak rusak tanpa sebab — dan pada modul ini
// akibatnya lebih jauh: nomor urut ID_PANEL ikut mundur, sehingga dua panel dapat lahir
// dengan kunci yang sama.
//
// Hanya portal utama yang dilayani, sejalan dengan readyAliases pada cabang tanpa Oracle.
// Memilih portal lain tanpa basis data karena itu ditolak dengan galat yang sama seperti
// di produksi.
//
// Penyimpanan contoh memuat panel dengan DUA, SATU, dan NOL lokasi, sehingga seluruh alur
// layar dapat dicoba tanpa Oracle — termasuk panel tanpa lokasi sama sekali, keadaan sah
// yang paling mudah terlupa diuji.
func panelSelectorMemory(primaryAlias string) masterpanel.RepoSelector {
	var lock sync.Mutex
	store := map[string]masterpanel.Store{}

	return func(alias string) (masterpanel.Store, error) {
		clean, err := memoryPortal(alias, primaryAlias)
		if err != nil {
			return nil, err
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := masterpanelmemory.NewSampleRepo()
		store[clean] = fresh
		return fresh, nil
	}
}

// sparepartSelectorMemory menyusun penyimpanan Master Sparepart di memori.
//
// Satu portal mendapat satu penyimpanan, dibuat saat pertama diminta lalu dipakai kembali.
// Kalau dibuat ulang setiap permintaan, penambahan yang baru disimpan akan hilang pada
// permintaan berikutnya dan layarnya tampak rusak tanpa sebab — dan pada modul ini
// akibatnya lebih jauh: nomor urut ID ikut mundur, sehingga dua sparepart dapat lahir
// dengan kunci yang sama.
//
// Hanya portal utama yang dilayani, sejalan dengan readyAliases pada cabang tanpa Oracle.
// Memilih portal lain tanpa basis data karena itu ditolak dengan galat yang sama seperti di
// produksi.
//
// Penyimpanan contoh memuat ketiga status persetujuan sekaligus beserta kedua daftar
// acuannya, sehingga seluruh alur layar dapat dicoba tanpa Oracle — termasuk baris yang
// kolom pilihannya kosong dan baris yang belum pernah distempel tanggal harga, dua keadaan
// sah yang paling mudah terlupa diuji.
func sparepartSelectorMemory(primaryAlias string) mastersparepart.RepoSelector {
	var lock sync.Mutex
	store := map[string]mastersparepart.Store{}

	return func(alias string) (mastersparepart.Store, error) {
		clean, err := memoryPortal(alias, primaryAlias)
		if err != nil {
			return nil, err
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := mastersparepartmemory.NewSampleRepo()
		store[clean] = fresh
		return fresh, nil
	}
}

// groupingSelectorMemory menyusun penyimpanan Master Grouping Sparepart di memori.
//
// Satu portal mendapat satu penyimpanan, dibuat saat pertama diminta lalu dipakai kembali.
// Kalau dibuat ulang setiap permintaan, penambahan yang baru disimpan akan hilang pada
// permintaan berikutnya dan layarnya tampak rusak tanpa sebab — dan pada modul ini akibatnya
// lebih jauh: pencacah ID DAN nomor grup ikut mundur, sehingga dua grouping dapat lahir dengan
// kunci yang sama dan dua kendaraan berbeda dapat berbagi satu nomor grup.
//
// Hanya portal utama yang dilayani, sejalan dengan readyAliases pada cabang tanpa Oracle.
// Memilih portal lain tanpa basis data karena itu ditolak dengan galat yang sama seperti di
// produksi.
//
// Penyimpanan contoh memuat ketiga status persetujuan sekaligus beserta keempat sumber
// acuannya, dan DUA baris yang berbagi satu nomor grup — tanpa itu, layar tidak pernah
// memperlihatkan apa gunanya modul ini.
func groupingSelectorMemory(primaryAlias string) mastergroupingsparepart.RepoSelector {
	var lock sync.Mutex
	store := map[string]mastergroupingsparepart.Store{}

	return func(alias string) (mastergroupingsparepart.Store, error) {
		clean, err := memoryPortal(alias, primaryAlias)
		if err != nil {
			return nil, err
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := mastergroupingsparepartmemory.NewSampleRepo()
		store[clean] = fresh
		return fresh, nil
	}
}

// partCategorySelectorMemory menyusun penyimpanan Master Kategori Sparepart di memori.
//
// Satu portal mendapat satu penyimpanan, dibuat saat pertama diminta lalu dipakai kembali.
// Kalau dibuat ulang setiap permintaan, penambahan yang baru disimpan akan hilang pada
// permintaan berikutnya — dan pada modul ini akibatnya lebih jauh: nomor urut yang lahir
// dari `max+1` ikut mundur, sehingga dua kategori dapat lahir dengan kunci yang sama.
//
// Hanya portal utama yang dilayani, sejalan dengan readyAliases pada cabang tanpa Oracle.
// Memilih portal lain tanpa basis data karena itu ditolak dengan galat yang sama seperti di
// produksi.
//
// # Satu hal yang TIDAK terjadi di modus memori, dan itu perlu disadari saat mencoba
//
// Modul ini dan Master Sparepart membaca tabel yang SAMA di produksi, tetapi punya
// penyimpanan memori SENDIRI-SENDIRI di sini. Kategori yang ditambahkan lewat layar ini
// karena itu tidak muncul di dropdown Kategori pada layar Master Sparepart selama aplikasi
// berjalan tanpa Oracle.
//
// Menyatukan keduanya akan menuntut salah satu modul mengimpor penyimpanan modul lain —
// tautan yang tidak ada di produksi, dan yang membuat modul selesai harus disunting setiap
// kali tetangganya berubah. Keterbatasan ini dibiarkan dan dicatat, bukan ditambal.
func partCategorySelectorMemory(primaryAlias string) masterkategorisparepart.RepoSelector {
	var lock sync.Mutex
	store := map[string]masterkategorisparepart.Store{}

	return func(alias string) (masterkategorisparepart.Store, error) {
		clean, err := memoryPortal(alias, primaryAlias)
		if err != nil {
			return nil, err
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := masterkategorisparepartmemory.NewSampleRepo()
		store[clean] = fresh
		return fresh, nil
	}
}

// partTypeSelectorMemory menyusun penyimpanan Master Tipe Sparepart di memori.
//
// Satu portal mendapat satu penyimpanan, dibuat saat pertama diminta lalu dipakai kembali.
// Kalau dibuat ulang setiap permintaan, penambahan yang baru disimpan akan hilang pada
// permintaan berikutnya — dan pada modul ini akibatnya lebih jauh: nomor urut yang lahir
// dari `max+1` ikut mundur, sehingga dua tipe dapat lahir dengan kunci yang sama.
//
// Hanya portal utama yang dilayani, sejalan dengan readyAliases pada cabang tanpa Oracle.
// Memilih portal lain tanpa basis data karena itu ditolak dengan galat yang sama seperti di
// produksi.
//
// # Dua hal yang TIDAK terjadi di modus memori, dan keduanya perlu disadari saat mencoba
//
//  1. Modul ini dan Master Sparepart membaca tabel tipe yang SAMA di produksi, tetapi punya
//     penyimpanan memori SENDIRI-SENDIRI di sini. Tipe yang ditambahkan lewat layar ini
//     tidak muncul di dropdown Tipe pada layar Master Sparepart selama berjalan tanpa
//     Oracle.
//  2. Dropdown Kategori pada layar ini dilayani SALINAN acuan milik penyimpanan ini sendiri
//     (lihat SampleCategoryList), bukan oleh penyimpanan Master Kategori Sparepart.
//     Kategori yang ditambahkan di layar itu karena itu tidak muncul di sini.
//
// Menyatukan keduanya akan menuntut salah satu modul mengimpor penyimpanan modul lain —
// tautan yang tidak ada di produksi, dan yang membuat modul selesai harus disunting setiap
// kali tetangganya berubah. Keterbatasan ini dibiarkan dan dicatat, bukan ditambal.
func partTypeSelectorMemory(primaryAlias string) mastertipesparepart.RepoSelector {
	var lock sync.Mutex
	store := map[string]mastertipesparepart.Store{}

	return func(alias string) (mastertipesparepart.Store, error) {
		clean, err := memoryPortal(alias, primaryAlias)
		if err != nil {
			return nil, err
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := mastertipesparepartmemory.NewSampleRepo()
		store[clean] = fresh
		return fresh, nil
	}
}

// surveyorLoginSelectorMemory menyusun penyimpanan Master Login di memori.
//
// Satu portal mendapat satu penyimpanan, dibuat saat pertama diminta lalu dipakai kembali.
// Kalau dibuat ulang setiap permintaan, penambahan yang baru disimpan akan hilang pada
// permintaan berikutnya — dan pada modul ini akibatnya khas: LOGINLEADER baris baru
// diturunkan dengan MEMBACA penyimpanan yang sama, sehingga penyimpanan yang lahir kembali
// akan membuat setiap penambahan tampak seolah pemanggilnya tidak pernah terdaftar.
func surveyorLoginSelectorMemory(primaryAlias string) masterlogin.RepoSelector {
	var lock sync.Mutex
	store := map[string]masterlogin.Repo{}

	return func(alias string) (masterlogin.Repo, error) {
		clean, err := memoryPortal(alias, primaryAlias)
		if err != nil {
			return nil, err
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := masterloginmemory.NewSampleRepo()
		store[clean] = fresh
		return fresh, nil
	}
}

// clauseAISelectorMemory menyusun penyimpanan Master Pasal AI di memori.
//
// Alasannya sama dengan progressStatusSelectorMemory: satu portal satu penyimpanan, dibuat
// saat pertama diminta lalu dipakai kembali, dan hanya portal utama yang dilayani.
//
// Bedanya satu: **inilah satu-satunya jalur yang benar-benar menampilkan layar ini hari
// ini.** Pada penyimpanan Oracle, modul ini menjawab 503 sampai kueri `GetListDataPasalAI`
// dan `CountDataPasalAI` diterima dari Tim Pega.
func clauseAISelectorMemory(primaryAlias string) masterpasalai.RepoSelector {
	var lock sync.Mutex
	store := map[string]masterpasalai.Repo{}

	return func(alias string) (masterpasalai.Repo, error) {
		clean := strings.ToUpper(strings.TrimSpace(alias))
		if clean != strings.ToUpper(strings.TrimSpace(primaryAlias)) {
			return nil, fmt.Errorf("%w: portal %q tidak tersedia tanpa basis data", portal.ErrNotReady, alias)
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := masterpasalaimemory.NewRepo(masterpasalaimemory.SampleClause())
		store[clean] = fresh
		return fresh, nil
	}
}

// causeOfLossDetailSelectorMemory menyusun penyimpanan Detail Penyebab Kerugian di memori.
//
// Satu portal mendapat satu penyimpanan, dibuat saat pertama diminta lalu dipakai kembali.
// Kalau dibuat ulang setiap permintaan, penambahan yang baru disimpan akan hilang pada
// permintaan berikutnya — dan pada modul ini akibatnya khas: nomor urut D_COL_ID ikut lahir
// kembali, sehingga baris berikutnya menerima ID yang sudah dipakai dan ditolak sebagai
// kunci ganda.
func causeOfLossDetailSelectorMemory(primaryAlias string) detailpenyebab.RepoSelector {
	var lock sync.Mutex
	store := map[string]detailpenyebab.Store{}

	return func(alias string) (detailpenyebab.Store, error) {
		clean, err := memoryPortal(alias, primaryAlias)
		if err != nil {
			return nil, err
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := detailpenyebabmemory.NewSampleRepo()
		store[clean] = fresh
		return fresh, nil
	}
}

// investigatorInboxSelectorMemory menyusun penyimpanan Inbox Investigator di memori.
//
// Satu portal mendapat satu penyimpanan, dibuat saat pertama diminta lalu dipakai kembali.
//
// Seperti Master Reas, alasannya BUKAN menjaga penambahan agar tidak hilang — modul ini
// tidak menulis apa pun. Yang dijaga adalah IDENTITAS penyimpanannya: repo yang lahir
// kembali setiap permintaan membuat galat yang dipasang lewat SetError menghilang di antara
// dua permintaan, sehingga jalur gagal tidak dapat dicoba saat pengembangan.
//
// Portal selain yang utama DITOLAK dengan `portal.ErrNotReady` lewat memoryPortal — bukan
// diberi penyimpanan kosong. Pembedaannya penting: "belum tersedia" dan "antreannya memang
// kosong" adalah dua hal berbeda, dan menjawab yang pertama dengan daftar kosong membuat
// petugas menyimpulkan tidak ada pekerjaan padahal basis datanya belum tersambung.
func investigatorInboxSelectorMemory(primaryAlias string) inboxinvestigator.RepoSelector {
	var lock sync.Mutex
	store := map[string]inboxinvestigator.Repo{}

	return func(alias string) (inboxinvestigator.Repo, error) {
		clean, err := memoryPortal(alias, primaryAlias)
		if err != nil {
			return nil, err
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := inboxinvestigatormemory.NewSampleRepo()
		store[clean] = fresh
		return fresh, nil
	}
}

// receiveTKASelectorMemory menyusun penyimpanan Inbox Receive TKA di memori.
//
// Satu portal mendapat satu penyimpanan, dibuat saat pertama diminta lalu dipakai kembali.
//
// Di sini alasannya BUKAN sekadar menjaga identitas penyimpanan seperti pada modul
// baca-saja: modul ini MENULIS. Repo yang lahir kembali setiap permintaan akan memunculkan
// lagi baris yang barusan diisi, sehingga jalur "baris hilang setelah dikerjakan" — ciri
// kedua Inbox pada `D-79` — tidak dapat dicoba sama sekali saat pengembangan.
//
// Portal selain yang utama DITOLAK dengan `portal.ErrNotReady` lewat memoryPortal — bukan
// diberi penyimpanan kosong. Pembedaannya penting: "belum tersedia" dan "memang tidak ada
// pekerjaannya" adalah dua hal berbeda.
func receiveTKASelectorMemory(primaryAlias string) inboxreceivetka.RepoSelector {
	var lock sync.Mutex
	store := map[string]inboxreceivetka.Repo{}

	return func(alias string) (inboxreceivetka.Repo, error) {
		clean, err := memoryPortal(alias, primaryAlias)
		if err != nil {
			return nil, err
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := inboxreceivetkamemory.NewSampleRepo()
		store[clean] = fresh
		return fresh, nil
	}
}

// reinsurerMemberSelectorMemory menyusun penyimpanan Master Reas di memori.
//
// Satu portal mendapat satu penyimpanan, dibuat saat pertama diminta lalu dipakai kembali.
//
// Berbeda dari pemilih modul lain, alasannya BUKAN menjaga penambahan agar tidak hilang —
// modul ini tidak menulis apa pun. Yang dijaga adalah IDENTITAS penyimpanannya: repo yang
// lahir kembali setiap permintaan membuat galat yang dipasang lewat SetError menghilang di
// antara dua permintaan, sehingga jalur gagal tidak dapat dicoba saat pengembangan.
func reinsurerMemberSelectorMemory(primaryAlias string) masterreas.RepoSelector {
	var lock sync.Mutex
	store := map[string]masterreas.Repo{}

	return func(alias string) (masterreas.Repo, error) {
		clean, err := memoryPortal(alias, primaryAlias)
		if err != nil {
			return nil, err
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := masterreasmemory.NewSampleRepo()
		store[clean] = fresh
		return fresh, nil
	}
}

// clauseSelectorMemory menyusun penyimpanan Master Pasal Kerugian di memori.
//
// Satu portal mendapat satu penyimpanan, dibuat saat pertama diminta lalu dipakai
// kembali. Kalau dibuat ulang setiap permintaan, penambahan yang baru disimpan akan
// hilang pada permintaan berikutnya dan layarnya tampak rusak tanpa sebab — dan pada
// modul ini akibatnya lebih jauh: nomor urut IDDATA ikut mundur, sehingga dua pasal dapat
// lahir dengan kunci yang sama.
//
// Hanya portal utama yang dilayani, sejalan dengan readyAliases pada cabang tanpa
// Oracle. Memilih portal lain tanpa basis data karena itu ditolak dengan galat yang sama
// seperti di produksi.
//
// Penyimpanan contoh memuat master lini bisnisnya sekaligus, sehingga seluruh alur layar
// dapat dicoba tanpa Oracle: menambah, memilih lini bisnis dari daftar, menyunting, lalu
// menghapus.
func clauseSelectorMemory(primaryAlias string) masterpasal.RepoSelector {
	var lock sync.Mutex
	store := map[string]masterpasal.Store{}

	return func(alias string) (masterpasal.Store, error) {
		clean, err := memoryPortal(alias, primaryAlias)
		if err != nil {
			return nil, err
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := masterpasalmemory.NewSampleRepo()
		store[clean] = fresh
		return fresh, nil
	}
}

// supplierSelectorMemory menyusun penyimpanan Master Supplier di memori.
//
// Satu portal mendapat satu penyimpanan, dibuat saat pertama diminta lalu dipakai
// kembali. Kalau dibuat ulang setiap permintaan, penambahan yang baru disimpan akan
// hilang pada permintaan berikutnya dan layarnya tampak rusak tanpa sebab.
//
// Di modul ini pemakaian ulang itu punya akibat kedua yang tidak dimiliki master lain:
// antrean permintaan persetujuan ikut tersimpan di instans yang sama, sehingga alur
// "tambah supplier lalu periksa antreannya" dapat dicoba tanpa Oracle sama sekali.
//
// Hanya portal utama yang dilayani, sejalan dengan readyAliases pada cabang tanpa Oracle.
// Memilih portal lain tanpa basis data karena itu ditolak dengan galat yang sama seperti
// di produksi: perilaku penolakannya ikut teruji saat pengembangan, bukan hanya nanti.
func supplierSelectorMemory(primaryAlias string) mastersupplier.RepoSelector {
	var lock sync.Mutex
	store := map[string]mastersupplier.Store{}

	return func(alias string) (mastersupplier.Store, error) {
		clean, err := memoryPortal(alias, primaryAlias)
		if err != nil {
			return nil, err
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := mastersuppliermemory.NewSampleRepo()
		store[clean] = fresh
		return fresh, nil
	}
}

// rejectionSelectorMemory menyusun penyimpanan Master Penolakan Klaim di memori.
//
// Satu portal mendapat satu penyimpanan, dibuat saat pertama diminta lalu dipakai
// kembali. Kalau dibuat ulang setiap permintaan, penambahan yang baru disimpan akan
// hilang pada permintaan berikutnya dan layarnya tampak rusak tanpa sebab.
//
// Hanya portal utama yang dilayani, sejalan dengan readyAliases pada cabang tanpa Oracle.
// Memilih portal lain tanpa basis data karena itu ditolak dengan galat yang sama seperti
// di produksi: perilaku penolakannya ikut teruji saat pengembangan, bukan hanya nanti.
func rejectionSelectorMemory(primaryAlias string) masterpenolakan.RepoSelector {
	var lock sync.Mutex
	store := map[string]masterpenolakan.Repo{}

	return func(alias string) (masterpenolakan.Repo, error) {
		clean, err := memoryPortal(alias, primaryAlias)
		if err != nil {
			return nil, err
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := masterpenolakanmemory.NewRepo(
			masterpenolakanmemory.SampleParents(),
			masterpenolakanmemory.SampleList()...,
		)
		store[clean] = fresh
		return fresh, nil
	}
}

// rejectionKomiteSelectorMemory menyusun penyimpanan Master Penolakan Komite di memori.
//
// Ia TIDAK menumpang pada rejectionSelectorMemory seperti halnya tingkat 2 master status
// progres menumpang pada tingkat 1: kedua tabel itu memang tidak berhubungan, dan
// menautkannya di sini akan menyiratkan hubungan yang tidak ada.
func rejectionKomiteSelectorMemory(primaryAlias string) masterpenolakan.RepoSelectorKomite {
	var lock sync.Mutex
	store := map[string]masterpenolakan.RepoKomite{}

	return func(alias string) (masterpenolakan.RepoKomite, error) {
		clean, err := memoryPortal(alias, primaryAlias)
		if err != nil {
			return nil, err
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := masterpenolakanmemory.NewRepoKomite(masterpenolakanmemory.SampleListKomite()...)
		store[clean] = fresh
		return fresh, nil
	}
}

// memoryPortal menormalkan alias portal dan menolak yang bukan portal utama.
//
// Satu fungsi untuk kedua pemilih di atas supaya keduanya menolak dengan galat yang sama
// persis. Dua pemeriksaan terpisah atas hal yang sama akan berbeda begitu salah satunya
// disunting — dan yang dipertaruhkan pada R-20 adalah pemisahan data antar badan hukum.
func memoryPortal(alias, primaryAlias string) (string, error) {
	clean := strings.ToUpper(strings.TrimSpace(alias))
	if clean != strings.ToUpper(strings.TrimSpace(primaryAlias)) {
		return "", fmt.Errorf("%w: portal %q tidak tersedia tanpa basis data", portal.ErrNotReady, alias)
	}
	return clean, nil
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

// claimHistorySelectorMemory menyusun penyimpanan riwayat klaim di memori.
//
// Satu portal mendapat satu penyimpanan, dibuat saat pertama diminta lalu dipakai
// kembali — alasannya sama dengan progressStatusSelectorMemory.
//
// Hanya portal utama yang dilayani, sejalan dengan readyAliases pada cabang tanpa Oracle.
// Memilih portal lain tanpa basis data karena itu ditolak dengan galat yang sama seperti
// di produksi: perilaku penolakannya ikut teruji saat pengembangan, bukan hanya nanti.
func claimHistorySelectorMemory(primaryAlias string) riwayatklaim.RepoSelector {
	var lock sync.Mutex
	store := map[string]riwayatklaim.Repo{}

	return func(alias string) (riwayatklaim.Repo, error) {
		clean, err := matchPrimaryPortal(alias, primaryAlias)
		if err != nil {
			return nil, err
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := riwayatklaimmemory.NewRepo(riwayatklaimmemory.SampleClaims()...)
		store[clean] = fresh
		return fresh, nil
	}
}

// claimProtectionSelectorMemory menyusun gerbang proteksi data di memori.
//
// Ia dipakai kembali antar permintaan, dan itu MENENTUKAN di sini: jatah pencarian
// dihitung dari pemakaian yang tercatat, dan penyimpanan yang dibuat ulang setiap
// permintaan akan mengembalikan jatah penuh setiap kali — sehingga jalur "jatah habis"
// tidak akan pernah dapat dicoba tanpa Oracle.
//
// Baris proteksi contohnya sengaja berbeda keadaan supaya keempat jalur gerbang dapat
// dicoba: lolos, hampir habis, sudah habis, dan belum terdaftar. Lihat
// riwayatklaim/repo/memory/sample.go.
func claimProtectionSelectorMemory(primaryAlias string) riwayatklaim.ProtectionRepoSelector {
	var lock sync.Mutex
	store := map[string]riwayatklaim.ProtectionRepo{}

	return func(alias string) (riwayatklaim.ProtectionRepo, error) {
		clean, err := matchPrimaryPortal(alias, primaryAlias)
		if err != nil {
			return nil, err
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := riwayatklaimmemory.NewProtectionRepo(riwayatklaimmemory.SampleProtections()...)
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

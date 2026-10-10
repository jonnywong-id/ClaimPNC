package main

import (
	"bufio"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"claim-pnc/internal/auth"
	"claim-pnc/internal/auth/provider"
	"claim-pnc/internal/auth/repo/sqlstore"
	"claim-pnc/internal/inboxcloseclaim"
	"claim-pnc/internal/inboxcompliance"
	"claim-pnc/internal/inboxinvestigator"
	"claim-pnc/internal/inboxosclaimpercabang"
	inboxosclaimpercabangsql "claim-pnc/internal/inboxosclaimpercabang/repo/sqlstore"
	"claim-pnc/internal/inboxoutstanding"
	"claim-pnc/internal/inboxreceivetka"
	"claim-pnc/internal/inboxservicecenter"
	"claim-pnc/internal/masterautoclaim"
	"claim-pnc/internal/masterbengkel"
	"claim-pnc/internal/masterdominanfactor"
	"claim-pnc/internal/mastergroupingsparepart"
	"claim-pnc/internal/masterkategorisparepart"
	"claim-pnc/internal/masterpanel"
	"claim-pnc/internal/masterpasal"
	"claim-pnc/internal/masterpenolakan"
	"claim-pnc/internal/masterpenyebabkerugian"
	"claim-pnc/internal/mastersparepart"
	"claim-pnc/internal/masterstatus"
	"claim-pnc/internal/mastersupplier"
	"claim-pnc/internal/mastersurveyors"
	"claim-pnc/internal/mastertipesparepart"
	"claim-pnc/internal/masterxol"
	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/platform/config"
	"claim-pnc/internal/platform/db"
	"claim-pnc/internal/portal"
	"claim-pnc/internal/registrasi"
	"claim-pnc/internal/riwayatklaim"

	casestudyclaimsql "claim-pnc/internal/casestudyclaim/repo/sqlstore"
	daftardetaildokumentravelsql "claim-pnc/internal/daftardetaildokumentravel/repo/sqlstore"
	daftardetailtipedokumensql "claim-pnc/internal/daftardetailtipedokumen/repo/sqlstore"
	daftarobjekdokumensql "claim-pnc/internal/daftarobjekdokumen/repo/sqlstore"
	daftartipedokumensql "claim-pnc/internal/daftartipedokumen/repo/sqlstore"
	daftartipedokumenbisnissql "claim-pnc/internal/daftartipedokumenbisnis/repo/sqlstore"
	detailpenyebabsql "claim-pnc/internal/detailpenyebab/repo/sqlstore"
	inboxadminsql "claim-pnc/internal/inboxadmin/repo/sqlstore"
	inboxanalystdoctorsql "claim-pnc/internal/inboxanalystdoctor/repo/sqlstore"
	"claim-pnc/internal/inboxautoclaim"
	inboxautoclaimsql "claim-pnc/internal/inboxautoclaim/repo/sqlstore"
	"claim-pnc/internal/inboxbandinghargasalvage"
	inboxbandinghargasalvagesql "claim-pnc/internal/inboxbandinghargasalvage/repo/sqlstore"
	"claim-pnc/internal/inboxclaimtreatynonprop"
	inboxclaimtreatynonpropsql "claim-pnc/internal/inboxclaimtreatynonprop/repo/sqlstore"
	"claim-pnc/internal/inboxclaimtreatyprop"
	inboxclaimtreatypropsql "claim-pnc/internal/inboxclaimtreatyprop/repo/sqlstore"
	inboxcloseclaimsql "claim-pnc/internal/inboxcloseclaim/repo/sqlstore"
	inboxcompliancesql "claim-pnc/internal/inboxcompliance/repo/sqlstore"
	inboxinvestigatorsql "claim-pnc/internal/inboxinvestigator/repo/sqlstore"
	"claim-pnc/internal/inboxkomunikasicabang"
	inboxkomunikasicabangsql "claim-pnc/internal/inboxkomunikasicabang/repo/sqlstore"
	inboxlaporanklaimsql "claim-pnc/internal/inboxlaporanklaim/repo/sqlstore"
	"claim-pnc/internal/inboxmanager"
	inboxmanagersql "claim-pnc/internal/inboxmanager/repo/sqlstore"
	"claim-pnc/internal/inboxmanageradmin"
	inboxmanageradminsql "claim-pnc/internal/inboxmanageradmin/repo/sqlstore"
	"claim-pnc/internal/inboxmanagerreceivepucl"
	inboxmanagerreceivepuclsql "claim-pnc/internal/inboxmanagerreceivepucl/repo/sqlstore"
	inboxoutstandingsql "claim-pnc/internal/inboxoutstanding/repo/sqlstore"
	"claim-pnc/internal/inboxpladla"
	inboxpladlasql "claim-pnc/internal/inboxpladla/repo/sqlstore"
	"claim-pnc/internal/inboxpladlapredla"
	inboxpladlapredlasql "claim-pnc/internal/inboxpladlapredla/repo/sqlstore"
	inboxprogressclaimsql "claim-pnc/internal/inboxprogressclaim/repo/sqlstore"
	inboxrclsql "claim-pnc/internal/inboxrcl/repo/sqlstore"
	"claim-pnc/internal/inboxrclpucl"
	inboxrclpuclsql "claim-pnc/internal/inboxrclpucl/repo/sqlstore"
	inboxreceivetkasql "claim-pnc/internal/inboxreceivetka/repo/sqlstore"
	"claim-pnc/internal/inboxsalvage"
	inboxsalvagesql "claim-pnc/internal/inboxsalvage/repo/sqlstore"
	inboxservicecentersql "claim-pnc/internal/inboxservicecenter/repo/sqlstore"
	inboxsurveysql "claim-pnc/internal/inboxsurvey/repo/sqlstore"
	"claim-pnc/internal/inputacceptation"
	inputacceptationsql "claim-pnc/internal/inputacceptation/repo/sqlstore"
	"claim-pnc/internal/komite"
	komitesql "claim-pnc/internal/komite/repo/sqlstore"
	masterautoclaimsql "claim-pnc/internal/masterautoclaim/repo/sqlstore"
	masterbengkelsql "claim-pnc/internal/masterbengkel/repo/sqlstore"
	mastercolsql "claim-pnc/internal/mastercolsimasonline/repo/sqlstore"
	masterdominanfactorsql "claim-pnc/internal/masterdominanfactor/repo/sqlstore"
	mastergroupingsparepartsql "claim-pnc/internal/mastergroupingsparepart/repo/sqlstore"
	masterkategorisparepartsql "claim-pnc/internal/masterkategorisparepart/repo/sqlstore"
	masterloginsql "claim-pnc/internal/masterlogin/repo/sqlstore"
	masterpanelsql "claim-pnc/internal/masterpanel/repo/sqlstore"
	masterpasalsql "claim-pnc/internal/masterpasal/repo/sqlstore"
	masterpenolakansql "claim-pnc/internal/masterpenolakan/repo/sqlstore"
	masterpenyebabkerugiansql "claim-pnc/internal/masterpenyebabkerugian/repo/sqlstore"
	masterpicteknikdirectory "claim-pnc/internal/masterpicteknik/directory"
	masterpictekniksql "claim-pnc/internal/masterpicteknik/repo/sqlstore"
	masterreassql "claim-pnc/internal/masterreas/repo/sqlstore"
	masterrecoverysql "claim-pnc/internal/masterrecovery/repo/sqlstore"
	masterrecoveryva "claim-pnc/internal/masterrecovery/virtualaccount"
	mastersparepartsql "claim-pnc/internal/mastersparepart/repo/sqlstore"
	masterstatussql "claim-pnc/internal/masterstatus/repo/sqlstore"
	mastersuppliersql "claim-pnc/internal/mastersupplier/repo/sqlstore"
	mastersurveyorssql "claim-pnc/internal/mastersurveyors/repo/sqlstore"
	mastertipesparepartsql "claim-pnc/internal/mastertipesparepart/repo/sqlstore"
	masterxolsql "claim-pnc/internal/masterxol/repo/sqlstore"
	monitoringslinkojksql "claim-pnc/internal/monitoringslinkojk/repo/sqlstore"
	"claim-pnc/internal/outstandingclaim"
	outstandingclaimsql "claim-pnc/internal/outstandingclaim/repo/sqlstore"
	portalsql "claim-pnc/internal/portal/repo/sqlstore"
	registrasisql "claim-pnc/internal/registrasi/repo/sqlstore"
	reportklaimsql "claim-pnc/internal/reportklaim/repo/sqlstore"
	"claim-pnc/internal/reportkpi"
	reportkpisql "claim-pnc/internal/reportkpi/repo/sqlstore"
	riwayatklaimsql "claim-pnc/internal/riwayatklaim/repo/sqlstore"
)

// check menjalankan pemeriksaan integrasi dan mencetak hasilnya, lalu berhenti.
//
// # Kenapa mode ini ada
//
// Login yang sesungguhnya MENULIS ke CPNC_PENGGUNA dan CPNC_SESI_AKTIF, dan kedua tabel
// itu baru ada setelah DBA menjalankan migrasi 0001. Tanpa mode ini, integrasi HCC/HCQ
// dan POOLDATA.M_LOGIN_PNC tidak dapat dicoba sama sekali sebelum migrasi selesai —
// padahal keduanya justru yang paling ingin dibuktikan lebih dulu.
//
// Mode ini karena itu **tidak menulis apa pun**. Ia hanya membaca, memanggil HCQ, dan
// melaporkan apa yang ditemukannya.
func check(cfg config.Config, login string, passwordSource io.Reader, out io.Writer) error {
	print := func(format string, content ...any) {
		_, _ = fmt.Fprintf(out, format+"\n", content...)
	}

	print("Check integrasi Claim PNC")
	print("  lingkungan       : %s", cfg.Environment)
	print("  portal utama     : %s", cfg.PrimaryPortal)
	print("  adapter identitas: %s", cfg.IdentityAdapter)
	// Antarmuka disematkan saat `go build`, bukan saat `npm run build`. Tanpa baris ini,
	// binary yang dibangun sebelum antarmukanya diperbaiki tidak dapat dibedakan dari
	// yang sesudahnya — dan perbedaan itu tampak sebagai fitur yang tidak bekerja.
	print("  antarmuka        : dibangun %s", spaVersionText())
	print("")

	if cfg.Storage != config.StorageOracle {
		return fmt.Errorf("mode periksa menuntut PENYIMPANAN=oracle; sekarang %q.\n"+
			"    Alamat layanan HCQ dan daftar login non-karyawan keduanya dibaca dari basis data",
			cfg.Storage)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	pool, err := db.NewPool(ctx, cfg.PrimaryPortal, portalParameters(cfg), func(alias string, err error) {
		print("  [lewat] portal %-5s tidak dapat dibuka: %v", alias, err)
	})
	if err != nil {
		return fmt.Errorf("koneksi basis data gagal: %w", err)
	}
	defer pool.Close()
	print("  [ok]    koneksi basis data: %s", strings.Join(pool.Available(), ", "))

	primary := pool.Primary()
	legacy := sqlstore.NewLegacy(primary)

	// Koneksi KEDUA portal utama — pengganti DB Link `@ASMD`.
	//
	// Ia dibuka di sini, bukan di dalam pemeriksaannya, supaya ketiadaannya terbaca
	// sebagai satu keadaan rakitan alih-alih sebagai kegagalan satu pemeriksaan. Dan ia
	// TIDAK menghentikan mode periksa: aplikasi memang berjalan tanpanya.
	anekaPool := db.NewOptionalPool(ctx, anekaParameters(cfg), func(alias string, err error) {
		print("  [lewat] koneksi kedua portal %-5s tidak dapat dibuka: %v", alias, err)
	})
	defer anekaPool.Close()

	var anekaPrimary *sql.DB
	if anekaPool.Has(cfg.PrimaryPortal) {
		if conn, err := anekaPool.For(cfg.PrimaryPortal); err == nil {
			anekaPrimary = conn
		}
	}

	portalList := checkPortal(ctx, portalsql.NewRepo(primary), print)
	address := checkCatalog(ctx, legacy, cfg.PrimaryPortal, print)
	checkAppTables(ctx, legacy, print)
	checkLoginTable(ctx, legacy, print)
	checkClaimStatus(ctx, masterstatussql.NewRepo(primary), print)
	checkRejection(ctx, primary, print)
	checkMasterAutoClaim(ctx, primary, print)
	checkBengkel(ctx, primary, print)
	checkPanel(ctx, primary, print)
	checkSparepart(ctx, primary, print)
	checkGrouping(ctx, primary, print)
	checkPartCategory(ctx, primary, print)
	checkPartType(ctx, primary, print)
	checkPasal(ctx, primary, print)
	checkSupplier(ctx, primary, print)
	checkSurveyorLogin(ctx, primary, print)
	checkReinsurerMember(ctx, primary, print)
	checkCauseOfLossDetail(ctx, primary, print)
	checkClaimHistoryGate(ctx, riwayatklaimsql.NewProtectionRepo(primary), print)
	checkInvestigatorInbox(ctx, primary, print)
	checkReceiveTKAInbox(ctx, primary, print)
	checkSimasOnlineCauseOfLoss(ctx, mastercolsql.NewRepo(primary), mastercolsql.NewBusinessRepo(primary), print)
	checkTravelDocumentDetail(ctx, primary, print)
	checkAutoClaim(ctx, inboxautoclaimsql.NewRepo(primary), print)
	checkAutoClaimTabsDiffer(ctx, inboxautoclaimsql.NewRepo(primary), print)
	checkAutoClaimPaging(ctx, inboxautoclaimsql.NewRepo(primary), print)
	checkAutoClaimEveryCompany(ctx, inboxautoclaimsql.NewRepo(primary), print)
	checkAutoClaimUploadColumns(ctx, primary, print)
	checkAutoClaimDetail(ctx, inboxautoclaimsql.NewRepo(primary), print)
	checkAutoClaimPremiumSource(ctx, primary, print)
	checkAutoClaimUploadLookups(ctx, inboxautoclaimsql.NewRepo(primary), print)
	checkAutoClaimPremiumCheck(ctx, primary, inboxautoclaimsql.NewRepo(primary), print)
	checkPicTeknik(ctx, masterpictekniksql.NewRepo(primary), legacy, cfg.PrimaryPortal, print)
	checkRecovery(ctx, masterrecoverysql.NewRepo(primary), legacy, cfg.PrimaryPortal, print)
	checkDominantFactor(ctx, masterdominanfactorsql.NewRepo(primary), print)
	checkCauseOfLoss(ctx, masterpenyebabkerugiansql.NewRepo(primary), print)
	checkXOL(ctx, masterxolsql.NewRepo(primary), print)
	checkSurveyors(ctx, mastersurveyorssql.NewRepo(primary), print)
	checkDocumentType(ctx, daftartipedokumensql.NewRepo(primary), print)
	checkDocumentObject(ctx, daftarobjekdokumensql.NewRepo(primary), daftarobjekdokumensql.NewBusinessRepo(primary), print)
	checkBusinessDocumentRule(ctx, daftartipedokumenbisnissql.NewRepo(primary), print)
	checkDetailDocumentType(ctx, daftardetailtipedokumensql.NewRepo(primary), daftardetailtipedokumensql.NewReferenceRepo(primary), print)
	checkAssembledModules(ctx, primary, print)
	checkClaimTreatyProp(ctx, inboxclaimtreatypropsql.NewRepo(primary), print)
	checkOutstandingClaim(ctx, outstandingclaimsql.NewRepo(primary),
		inboxclaimtreatypropsql.NewRepo(primary), print)
	checkClaimTreatyNonProp(ctx, inboxclaimtreatynonpropsql.NewRepo(primary), print)
	checkInputAcceptation(ctx, inputacceptationsql.NewRepo(primary),
		inboxclaimtreatynonpropsql.NewRepo(primary), print)
	checkOSClaimPerCabang(ctx, inboxosclaimpercabangsql.NewRepo(primary), print)
	checkManagerReceivePUCL(ctx, inboxmanagerreceivepuclsql.NewRepo(primary), print)
	checkRCLPUCL(ctx, inboxrclpuclsql.NewRepo(primary), print)
	checkSalvage(ctx, inboxsalvagesql.NewRepo(primary), print)
	checkBandingHargaSalvage(ctx, inboxbandinghargasalvagesql.NewRepo(primary), print)
	checkPLADLAQueue(ctx, inboxpladlapredlasql.NewRepo(primary), print)
	checkPLADLAReinsurer(ctx, inboxpladlasql.NewRepo(primary), print)
	checkPLADLACommunicationFunnel(ctx, primary, login, print)
	checkReportKPI(ctx, reportkpisql.NewRepo(primary), print)
	checkReportKPIPICTeknik(ctx, reportkpisql.NewRepo(primary), print)
	checkReportKlaim(ctx, reportklaimsql.NewRepo(primary, anekaPrimary), print)
	checkKomunikasiCabang(ctx,
		inboxkomunikasicabangsql.NewRepo(primary),
		inboxkomunikasicabangsql.NewBranchResolver(primary),
		print)
	checkInboxProgressClaim(ctx, inboxprogressclaimsql.NewRepo(primary), print)
	checkInboxAnalystDoctor(ctx, inboxanalystdoctorsql.NewRepo(primary), print)
	checkInboxSurvey(ctx, inboxsurveysql.NewRepo(primary), print)
	checkInboxRCL(ctx, inboxrclsql.NewRepo(primary), print)
	checkCaseStudyClaim(ctx, casestudyclaimsql.NewRepo(primary), print)
	checkKomiteInbox(ctx, primary, login, print)
	checkClaimReport(ctx, inboxlaporanklaimsql.NewRepo(primary, clock.System{}), print)
	checkOutstanding(ctx, inboxoutstandingsql.NewRepo(primary), login, print)
	checkClaimReportBranch(ctx, inboxlaporanklaimsql.NewBranchResolver(primary), print)
	checkCloseClaim(ctx,
		inboxcloseclaimsql.NewRepo(primary),
		inboxcloseclaimsql.NewRequestRepo(primary),
		print)
	checkMonitoringSlinkOJK(ctx, primary, print)
	checkRegistration(ctx, primary, print)

	print("")
	if login == "" {
		print("Selesai. Tambahkan -login <nama pengguna> untuk mencoba masuk sungguhan.")
		_ = portalList
		return nil
	}
	if address == "" {
		print("Percobaan masuk dilewati: alamat layanan HCQ belum dapat dibaca.")
		return nil
	}
	return checkLogin(ctx, cfg, legacy, login, passwordSource, print)
}

func checkPortal(ctx context.Context, repo portal.Repo, print func(string, ...any)) []portal.Portal {
	list, err := repo.List(ctx)
	if err != nil {
		print("  [GAGAL] POOLDATA.M_PORTAL_PNC: %v", err)
		return nil
	}
	print("  [ok]    POOLDATA.M_PORTAL_PNC: %d portal terbaca", len(list))
	for _, p := range list {
		print("            %-5s %s", p.Alias, p.Name)
	}
	return list
}

func checkCatalog(ctx context.Context, legacy *sqlstore.Legacy, alias string, print func(string, ...any)) string {
	address, err := legacy.ServiceAddress(ctx, alias, "HCQ-LOGIN")
	switch {
	case errors.Is(err, provider.ErrServiceNotRegistered):
		print("  [GAGAL] POOLDATA.GCNM_CONNECT_REST: tidak ada baris APP=%q TYPESERVICE='HCQ-LOGIN'", alias)
		print("            Mintakan barisnya ke DBA, atau ubah PORTAL_UTAMA ke portal yang sudah punya.")
		return ""
	case err != nil:
		print("  [GAGAL] POOLDATA.GCNM_CONNECT_REST: %v", err)
		return ""
	}
	print("  [ok]    alamat layanan HCQ untuk APP=%q: %s", alias, address)
	return address
}

// checkAppTables membedakan "migrasi belum dijalankan" dari "akun tidak punya hak
// baca" — dua sebab yang tampak mirip tetapi perbaikannya berbeda jauh.
func checkAppTables(ctx context.Context, legacy *sqlstore.Legacy, print func(string, ...any)) {
	tabel := []struct{ name, query string }{
		{"CPNC_PENGGUNA", "user_check_table"},
		{"CPNC_SESI_AKTIF", "session_check_table"},
	}
	missing := 0
	for _, t := range tabel {
		if err := legacy.CheckTable(ctx, t.query); err != nil {
			print("  [BELUM] %s tidak dapat dibaca: %v", t.name, err)
			missing++
			continue
		}
		print("  [ok]    %s dapat dibaca", t.name)
	}
	if missing > 0 {
		print("            Migrasi backend/migrations/0001 tampaknya belum dijalankan DBA.")
		print("            Mode periksa TETAP bisa mencoba masuk — ia tidak menulis apa pun.")
		print("            Yang belum bisa adalah masuk lewat aplikasi, karena itu menyimpan sesi.")
	}
}

// checkClaimReport melaporkan kesiapan tabel Inbox Laporan Klaim, termasuk kolom
// T_CLAIM_PNC yang ditulisnya untuk baris berkas.
//
// Ia memeriksa DUA hal yang sifatnya berbeda, dan membedakannya penting:
//
//	POOLDATA.CPNC_LAPORAN_KLAIM   tabel BARU, dibuat migrasi 0003 — dibaca DAN ditulis
//	POOLDATA.T_CLAIMLIST_ADMIN    sumber daftar, diisi proses lain — hanya DIBACA
//
// Yang pertama belum ada sampai DBA menjalankan migrasinya; yang kedua sudah ada, dan
// kegagalannya berarti akun aplikasi tidak diberi hak baca. Dua sebab yang tampak mirip
// di layar tetapi perbaikannya berbeda jauh.
// checkMonitoringSlinkOJK melaporkan kesiapan ketiga tabel modul Monitoring SLINK OJK.
//
// # Kenapa modul ini butuh pemeriksanya sendiri
//
// Ketiga tabelnya tidak disentuh modul lain mana pun. Tanpa pemeriksa, ketiadaan atau
// salah nama kolomnya baru ketahuan ketika seorang pelapor membuka layarnya dan mendapat
// galat 500 — dan yang gagal di sana adalah laporan ke OJK.
//
// Ketiganya dilaporkan TERPISAH karena akibat kegagalannya berbeda:
//
//   - T_CLAIM_SLIK_OJK gagal   → segmen D01 kosong sama sekali.
//   - T_CLAIM_OBJECTLIST gagal → segmen F06 kosong sama sekali.
//   - T_GENERAL gagal          → hanya kolom "Operasi Data" yang tidak dapat dihitung;
//     kedua segmen lainnya tetap berjalan.
func checkMonitoringSlinkOJK(ctx context.Context, primary *sql.DB, print func(string, ...any)) {
	results := monitoringslinkojksql.Probe(ctx, primary)

	failed := 0
	for _, result := range results {
		if result.OK() {
			print("  [ok]    %s", result.Name)
			continue
		}
		failed++
		print("  [BELUM] %s: %v", result.Name, result.Err)
	}

	if failed == 0 {
		return
	}
	print("            Ketiganya HANYA DIBACA modul ini; yang mengisinya adalah jalur")
	print("            akseptasi sistem lama (InsertAdjustmentList, InsertAdjustmentListKredit).")
	print("            Bila gagal, yang kurang adalah hak baca akun aplikasi — bukan migrasi,")
	print("            karena modul ini tidak membawa migrasi sama sekali.")
}

func checkClaimReport(ctx context.Context, repo *inboxlaporanklaimsql.Repo, print func(string, ...any)) {
	if err := repo.CheckTable(ctx); err != nil {
		print("  [BELUM] Inbox Laporan Klaim belum siap: %v", err)
		print("            CPNC_LAPORAN_KLAIM dibuat migrasi backend/migrations/0003,")
		print("            dan ia dijalankan di SETIAP portal entitas — bukan hanya portal utama.")
		print("            T_CLAIMLIST_ADMIN diisi proses lain dan hanya dibaca; bila justru ia")
		print("            yang gagal, yang kurang adalah hak baca akun aplikasi, bukan migrasinya.")
		return
	}
	print("  [ok]    tabel Inbox Laporan Klaim dapat dibaca, dan kolom baris berkas di T_CLAIM_PNC tersedia")

	// Pengisian otomatis form saat Nomor Polis diisi membaca POOLDATA.T_GENERAL. Nomor
	// kosong tidak cocok dengan baris mana pun, sehingga yang teruji hanya hak baca dan
	// kolomnya — tidak ada data polis yang dibaca.
	if _, _, err := repo.FindPolicy(ctx, ""); err != nil {
		print("  [BELUM] POOLDATA.T_GENERAL tidak dapat dibaca: %v", err)
		print("            Tanpanya, form Input Receive Document tidak dapat mengisi data polis.")
		return
	}
	print("  [ok]    POOLDATA.T_GENERAL dapat dibaca — data polis form Input Receive Document")
}

// checkClaimReportBranch melaporkan apakah cabang klaim petugas dapat diterjemahkan.
//
// Pemeriksaan ini ada karena kegagalannya TIDAK terlihat sebagai galat di layar. Bila
// penerjemahan cabang gagal, daftar Inbox Laporan Klaim tetap tampil — hanya saja tanpa
// batas cabang, sehingga petugas cabang melihat berkas seluruh cabang. Itu keadaan yang
// harus diketahui operator sebelum pengguna melaporkannya.
//
// Dua sebab kegagalan yang sifatnya berbeda:
//
//   - POOLDATA.BRANCH tidak dapat dibaca → hak baca akun aplikasi.
//   - DB link @asmd.sinarmas.co.id mati → HRDASM.V_HRD_MST dan LST_USER_ASURANSI ada di
//     basis data lain, dan `D-25` memang menugaskan penggantiannya dengan API (`R-03`).
//     Sampai API itu ada, layar ini bergantung pada DB link tersebut.
func checkClaimReportBranch(ctx context.Context, resolver *inboxlaporanklaimsql.BranchResolver, print func(string, ...any)) {
	if err := resolver.CheckTable(ctx); err != nil {
		print("  [BELUM] penerjemahan cabang klaim belum dapat dijalankan: %v", err)
		print("            Sumbernya POOLDATA.BRANCH + LST_USER_ASURANSI dan HRDASM.V_HRD_MST")
		print("            lewat DB link @asmd.sinarmas.co.id, sama seperti GetIDCabang di Pega.")
		print("            Selama gagal, daftar Inbox Laporan Klaim TIDAK dibatasi per cabang —")
		print("            layarnya memberitahu, tetapi batas cabangnya memang tidak berlaku.")
		return
	}
	print("  [ok]    cabang klaim petugas dapat diterjemahkan dari login")
}

// checkClaimStatus melaporkan kesiapan POOLDATA.M_STS_CLAIM sesudah migrasi 0002.
//
// Ia melaporkan JUMLAH BARIS, bukan sekadar "dapat dibaca". Angka itulah yang menjawab
// acceptance criteria TKT-F4-005 — "master status memuat tepat 33 kode, dihitung dan
// dilaporkan angkanya" — dan angka yang meleset menandakan langkah pemindahan isi pada
// migrasi 0002 tidak berjalan sebagaimana mestinya.
func checkClaimStatus(ctx context.Context, repo *masterstatussql.Repo, print func(string, ...any)) {
	if err := repo.CheckTable(ctx); err != nil {
		print("  [BELUM] POOLDATA.M_STS_CLAIM belum siap: %v", err)
		print("            Kolom LSC_NOTE dan OLD_LSC_ID ditambahkan migrasi 0002.")
		print("            Selama belum dijalankan, layar Master Status Klaim tidak dapat")
		print("            dipakai terhadap Oracle — tetapi seluruh bagian lain tetap jalan.")
		return
	}

	list, err := repo.List(ctx)
	if err != nil {
		print("  [GAGAL] POOLDATA.M_STS_CLAIM tidak dapat dibaca isinya: %v", err)
		return
	}

	print("  [ok]    POOLDATA.M_STS_CLAIM dapat dibaca: %d status", len(list))
	if empty := countEmptyLabels(list); empty > 0 {
		// Label kosong akan tampil sebagai status kosong di 23 rule Pega yang membaca
		// V_STS_CLAIM, termasuk laporan TAT dan KPI. Ia harus terlihat di sini, bukan
		// ditemukan pengguna di laporan.
		print("  [WASPADA] %d status berlabel kosong — periksa langkah 2 migrasi 0002", empty)
	}
}

func countEmptyLabels(list []masterstatus.ClaimStatus) int {
	empty := 0
	for _, s := range list {
		if strings.TrimSpace(s.Label) == "" {
			empty++
		}
	}
	return empty
}

// checkSimasOnlineCauseOfLoss memeriksa kesiapan Master COL Simas Online.
//
// Namanya dibedakan dari checkCauseOfLoss di bawah dengan sengaja: keduanya memang
// "penyebab kerugian", tetapi modul dan tabelnya berbeda.
//
// Ketiganya diperiksa TERPISAH — tabel COL, tabel pemetaan bisnis, dan master bisnis —
// karena ketiganya gagal karena sebab yang berbeda, dan menyatukan laporannya membuat
// pembaca menebak mana yang sebenarnya kurang:
//
//	M_CAUSE_OF_LOSS_ONLINE          hak baca; tabelnya sendiri sudah ada
//	M_CAUSE_OF_LOSS_ONLINE_DETAIL   hak baca; tabelnya sendiri sudah ada
//	POOLDATA.BUSINESS               hak baca belum diberikan; tabel itu milik GISFW dan
//	                                sampai modul ini aplikasi tidak pernah menyentuhnya
func checkSimasOnlineCauseOfLoss(
	ctx context.Context,
	repo *mastercolsql.Repo,
	business *mastercolsql.BusinessRepo,
	print func(string, ...any),
) {
	if err := repo.CheckTable(ctx); err != nil {
		print("  [BELUM] Master COL Simas Online belum siap: %v", err)
		print("            Yang dibaca modul ini adalah POOLDATA.M_CAUSE_OF_LOSS_ONLINE")
		print("            dan POOLDATA.M_CAUSE_OF_LOSS_ONLINE_DETAIL — keduanya sudah ada")
		print("            di basis data, jadi kegagalan di sini hampir pasti HAK BACA,")
		print("            bukan tabel yang belum dibuat.")
	} else {
		list, err := repo.List(ctx)
		if err != nil {
			print("  [GAGAL] POOLDATA.M_CAUSE_OF_LOSS_ONLINE tidak dapat dibaca isinya: %v", err)
		} else {
			print("  [ok]    POOLDATA.M_CAUSE_OF_LOSS_ONLINE dapat dibaca: %d cause of loss", len(list))
		}
	}

	if err := business.CheckTable(ctx); err != nil {
		print("  [GAGAL] POOLDATA.BUSINESS tidak dapat dibaca: %v", err)
		print("            Tabel itu milik GISFW dan HANYA DIBACA. Mintakan GRANT SELECT")
		print("            untuk akun aplikasi — tanpa itu, isian Bisnis pada layar COL")
		print("            Simas Online kosong dan setiap penyimpanan yang memilih bisnis")
		print("            akan ditolak.")
		return
	}

	list, err := business.List(ctx)
	if err != nil {
		print("  [GAGAL] POOLDATA.BUSINESS tidak dapat dibaca isinya: %v", err)
		return
	}
	print("  [ok]    POOLDATA.BUSINESS dapat dibaca: %d bisnis", len(list))
}

// checkDominantFactor melaporkan kesiapan POOLDATA.M_DOMINAN_FACTOR.
//
// Berbeda dari checkClaimStatus, di sini TIDAK ADA angka yang diharapkan: isi master ini
// belum pernah diterima dari DBA, sehingga berapa pun jumlah barisnya tidak dapat disebut
// benar atau salah. Yang dilaporkan karena itu apa adanya — dan angka itulah yang menjadi
// jawaban permintaan yang sedang menggantung.
//
// Modul ini tidak menuntut satu pun migrasi: kedua kolom yang dipakainya, ID dan NAME,
// sudah ada sejak tabelnya dibuat. Kegagalan di sini karena itu hampir pasti soal HAK
// BACA, bukan soal migrasi yang belum dijalankan.
func checkDominantFactor(ctx context.Context, repo *masterdominanfactorsql.Repo, print func(string, ...any)) {
	if err := repo.CheckTable(ctx); err != nil {
		print("  [BELUM] POOLDATA.M_DOMINAN_FACTOR tidak dapat dibaca: %v", err)
		print("            Modul ini TIDAK menuntut migrasi — kolom ID dan NAME sudah ada.")
		print("            Periksa hak SELECT, INSERT, dan UPDATE akun aplikasi atas tabel itu.")
		return
	}

	list, err := repo.List(ctx)
	if err != nil {
		print("  [GAGAL] POOLDATA.M_DOMINAN_FACTOR tidak dapat dibaca isinya: %v", err)
		return
	}

	print("  [ok]    POOLDATA.M_DOMINAN_FACTOR dapat dibaca: %d faktor dominan", len(list))

	// Nama kosong DITERIMA modul ini (keputusan Work Owner 2026-09-20), sehingga ini
	// bukan galat. Ia tetap dilaporkan karena akibatnya terlihat di tempat lain: laporan
	// Outstanding per Cabang merangkai nama faktor dengan LISTAGG, dan nama kosong muncul
	// di sana sebagai entri kosong di antara koma.
	if empty := countEmptyFactorNames(list); empty > 0 {
		print("  [CATATAN] %d faktor bernama kosong — sah, tetapi akan tampil sebagai", empty)
		print("            entri kosong pada LISTAGG laporan Outstanding per Cabang.")
	}

	// ID bukan bilangan tidak akan menghentikan aplikasi ini — ia hanya dipindahkan ke
	// belakang daftar. Tetapi ia MENGHENTIKAN procedure lama, yang memakai to_number(ID)
	// saat membentuk nomor berikutnya. Selama Pega masih dapat menulis tabel ini,
	// keadaan itu perlu diketahui.
	if odd := countNonNumericIDs(list); odd > 0 {
		print("  [WASPADA] %d baris ber-ID bukan bilangan — PEGA_M_DOMINAN_FACTOR akan", odd)
		print("            gagal dengan ORA-01722 bila penambahan dilakukan dari Pega.")
	}
}

// checkXOL melaporkan kesiapan keempat tabel Master XOL, dan memeriksa TIGA hal yang
// tidak dapat diketahui dari "tabelnya dapat dibaca" saja.
//
// Ketiganya dipilih karena masing-masing pernah benar-benar terjadi, dan seluruhnya
// terbukti di portal ASM pada 2026-09-20:
//
//  1. **Layer dan reas yatim.** Hapus di sistem lama tidak berkaskade, sehingga induk yang
//     terbuang meninggalkan anaknya. Induk 10003 sudah terhapus tetapi 2 layer dan 3 baris
//     bisnisnya masih ada. Aplikasi ini berkaskade, jadi ia tidak akan menambah yatim baru
//     — tetapi yang sudah ada tidak hilang sendiri, dan pembersihannya menempuh DBA.
//  2. **Layer yang total share-nya bukan 100%.** Sah menurut aturan modul ini — ia
//     peringatan, bukan penolakan — tetapi angkanya menyatakan berapa banyak struktur
//     treaty yang pembagian klaimnya belum lengkap.
//  3. **CONVERT_LIMIT yang tidak sama dengan LIMIT × KURSVALUE.** Itu satu-satunya cara
//     mengetahui apakah ada baris yang nilainya pernah disimpan dengan kurs yang berbeda.
//
// Modul ini TIDAK menuntut satu pun migrasi: keempat tabelnya beserta seluruh kolom yang
// dipakai sudah ada. Kegagalan di sini karena itu hampir pasti soal HAK BACA.
func checkXOL(ctx context.Context, repo *masterxolsql.Repo, print func(string, ...any)) {
	if err := repo.CheckTable(ctx); err != nil {
		print("  [BELUM] Tabel MST_XOL_* tidak dapat dibaca: %v", err)
		print("            Modul ini TIDAK menuntut migrasi — keempat tabelnya sudah ada.")
		print("            Periksa hak SELECT, INSERT, UPDATE, dan DELETE akun aplikasi atas")
		print("            MST_XOL_PNC, MST_XOL_BUSINESS, MST_XOL_LAYER, dan MST_XOL_REAS.")
		return
	}

	list, err := repo.List(ctx)
	if err != nil {
		print("  [GAGAL] POOLDATA.MST_XOL_PNC tidak dapat dibaca isinya: %v", err)
		return
	}
	print("  [ok]    POOLDATA.MST_XOL_PNC dapat dibaca: %d master XOL", len(list))

	// Pilihan Tahun dan pilihan grup bisnis dibaca dari tabel milik SISTEM LAIN
	// (M_TREATYYEAR, BUSINESS, BUSINESSGROUP, PROPORTIONALARRG). Hak bacanya tidak dijamin
	// oleh hak baca atas MST_XOL_*, sehingga ia diperiksa tersendiri — tanpa ini, layar
	// terbuka tetapi kedua dropdown-nya kosong tanpa sebab yang terbaca.
	year, err := repo.ListYear(ctx)
	if err != nil {
		print("  [GAGAL] Pilihan Tahun tidak dapat dibaca dari POOLDATA.M_TREATYYEAR: %v", err)
	} else {
		print("  [ok]    Pilihan Tahun terbaca: %d tahun treaty", len(year))
	}

	business, err := repo.ListBusinessGroup(ctx, masterxol.TypeProperty)
	if err != nil {
		print("  [GAGAL] Pilihan grup bisnis tidak dapat dibaca: %v", err)
	} else {
		print("  [ok]    Pilihan grup bisnis Type 1 terbaca: %d pilihan", len(business))
	}

	// Ketiga angka di bawah dihitung dengan MEMBACA tiap induk satu per satu, bukan dengan
	// kueri agregat. Jumlah induknya delapan di produksi, sehingga biayanya tidak berarti
	// — dan yang dipakai adalah jalur baca yang sama dengan yang dipakai layar, sehingga
	// pemeriksaan ini sekaligus menguji jalur itu.
	var layerCount, incomplete, mismatched int
	for _, m := range list {
		full, err := repo.Get(ctx, m.ID)
		if err != nil {
			print("  [GAGAL] Master XOL %s tidak dapat dibaca utuh: %v", m.ID, err)
			return
		}
		for _, l := range full.Layer {
			layerCount++
			if l.TotalShare() != masterxol.FullShare {
				incomplete++
			}
			if l.ConvertedLimit != masterxol.ConvertedLimit(l.Limit, full.ExchangeRate) {
				mismatched++
			}
		}
	}
	print("  [ok]    Layer terbaca utuh beserta reas-nya: %d layer", layerCount)

	// Baris yatim dilaporkan TERPISAH dari hitungan di atas, dan memang harus: layer yatim
	// tidak pernah ikut terbaca lewat Get, karena induknya tidak ada untuk dibuka. Tanpa
	// hitungan tersendiri, keberadaannya tidak akan pernah terlihat dari layar mana pun.
	if orphan, err := repo.OrphanCount(ctx); err != nil {
		print("  [GAGAL] Jumlah baris yatim tidak dapat dihitung: %v", err)
	} else if total := orphan["layer"] + orphan["bisnis"] + orphan["reas"]; total > 0 {
		print("  [WASPADA] %d baris anak yatim: %d layer, %d bisnis, %d reas.",
			total, orphan["layer"], orphan["bisnis"], orphan["reas"])
		print("            Induknya sudah terhapus lewat layar lama, yang TIDAK berkaskade.")
		print("            Aplikasi ini berkaskade sehingga tidak menambah yang baru; yang")
		print("            sudah ada dibersihkan lewat jalur DBA (`D-63`).")
	}

	if incomplete > 0 {
		print("  [CATATAN] %d layer total share-nya belum 100%% — sah menurut aturan modul ini,", incomplete)
		print("            tetapi pembagian klaim pada layer itu belum lengkap.")
	}
	if mismatched > 0 {
		print("  [WASPADA] %d layer CONVERT_LIMIT-nya tidak sama dengan LIMIT × KURSVALUE.", mismatched)
		print("            Aplikasi ini menghitung ulang saat membaca, jadi layar tetap benar —")
		print("            tetapi nilai yang TERSIMPAN berbeda sampai barisnya disimpan ulang.")
	}
}

// checkCauseOfLoss melaporkan kesiapan POOLDATA.M_CAUSE_OF_LOSS sesudah migrasi 0005.
//
// Ia melaporkan TIGA hal, bukan sekadar "dapat dibaca", dan ketiganya menjawab pertanyaan
// yang berbeda:
//
//  1. Apakah kolom COL_DESC sudah ada — yakni apakah migrasi 0005 sudah dijalankan.
//  2. Berapa barisnya — angka yang menggantikan daftar contoh yang sekarang dikarang.
//  3. Berapa baris yang JSON-nya ada tetapi kolomnya kosong — inilah yang mengukur celah
//     penulis kedua, dan tidak ada modul lain yang punya pertanyaan seperti ini.
func checkCauseOfLoss(ctx context.Context, repo *masterpenyebabkerugiansql.Repo, print func(string, ...any)) {
	if err := repo.CheckTable(ctx); err != nil {
		print("  [BELUM] POOLDATA.M_CAUSE_OF_LOSS belum siap: %v", err)
		print("            Kolom COL_DESC diisi migrasi 0005 — dan mungkin baru DITAMBAHKAN")
		print("            olehnya; apakah kolomnya sudah ada belum pernah diperiksa (R-08).")
		print("            Selama belum dijalankan, layar Master Penyebab Kerugian tidak dapat")
		print("            dipakai terhadap Oracle — tetapi seluruh bagian lain tetap jalan.")
		return
	}

	list, err := repo.List(ctx)
	if err != nil {
		print("  [GAGAL] POOLDATA.M_CAUSE_OF_LOSS tidak dapat dibaca isinya: %v", err)
		return
	}
	print("  [ok]    POOLDATA.M_CAUSE_OF_LOSS dapat dibaca: %d penyebab kerugian", len(list))

	// Keterangan kosong DITERIMA modul ini (keputusan Work Owner 2026-09-20), sehingga ini
	// bukan galat. Ia tetap dilaporkan karena akibatnya terlihat di tempat lain: dasbor
	// klaim per penyebab kerugian dan laporan XOL per bisnis mengelompokkan hasilnya
	// dengan GROUP BY COL_DESC, dan keterangan kosong muncul di sana sebagai kelompok
	// tanpa nama.
	if empty := countEmptyCauseDescriptions(list); empty > 0 {
		print("  [CATATAN] %d penyebab kerugian berketerangan kosong — sah, tetapi akan", empty)
		print("            tampil sebagai kelompok tanpa nama pada laporan yang memakai")
		print("            GROUP BY COL_DESC.")
	}

	// Inilah pemeriksaan yang khas modul ini, dan ia mengukur satu hal yang tidak dapat
	// dilihat dari layar mana pun: berapa baris yang ditulis PENULIS KEDUA — layar Simas
	// Online (MENU_ID 21) yang masih hidup di Pega dan hanya mengisi JSON_DATA.
	//
	// Nol berarti seluruh baris siap. Angka yang naik dari waktu ke waktu berarti layar
	// itu masih dipakai, dan barisnya tampil TANPA KETERANGAN di 19 rule pembaca tanpa
	// satu pun pesan galat.
	pending, err := repo.CountPendingJSON(ctx)
	if err != nil {
		// Bukan kegagalan yang menghentikan apa pun: kolom JSON_DATA mungkin sudah
		// dibuang, atau haknya tidak diberikan. Layarnya tetap berfungsi penuh.
		print("  [CATATAN] jumlah baris yang belum dipindahkan tidak dapat dihitung: %v", err)
		return
	}
	if pending > 0 {
		print("  [WASPADA] %d baris punya JSON_DATA tetapi COL_DESC kosong.", pending)
		print("            Dua sebab yang mungkin, dan keduanya perlu ditindaklanjuti:")
		print("            (a) langkah 1 migrasi 0005 belum dijalankan, atau kunci JSON-nya salah;")
		print("            (b) layar Simas Online (MENU_ID 21) masih menulis tabel ini dari Pega.")
		print("            Baris itu tampil TANPA KETERANGAN di 19 rule pembaca, tanpa galat.")
		return
	}
	print("  [ok]    tidak ada baris yang tertinggal di JSON_DATA")
}

func countEmptyCauseDescriptions(list []masterpenyebabkerugian.CauseOfLoss) int {
	empty := 0
	for _, c := range list {
		if strings.TrimSpace(c.Description) == "" {
			empty++
		}
	}
	return empty
}

func countEmptyFactorNames(list []masterdominanfactor.DominantFactor) int {
	empty := 0
	for _, f := range list {
		if strings.TrimSpace(f.Name) == "" {
			empty++
		}
	}
	return empty
}

func countNonNumericIDs(list []masterdominanfactor.DominantFactor) int {
	odd := 0
	for _, f := range list {
		if _, isNumber := masterdominanfactor.NumericID(f.ID); !isNumber {
			odd++
		}
	}
	return odd
}

// checkPicTeknik melaporkan kesiapan ketiga bahan modul Master PIC Teknik.
//
// Ketiganya diperiksa TERPISAH karena ketiganya dapat gagal sendiri-sendiri, dan tindak
// lanjutnya berbeda:
//
//	tabel  MST_USER_TEKNIK     objek lama yang sudah pasti ada; gagal = soal hak baca
//	view   V_MST_USER_TEKNIS   nama kolomnya BELUM terverifikasi dari export
//	baris  GCNM_CONNECT_REST   tanpa ini, menambah PIC tidak mungkin sama sekali
//
// Yang kedua dan ketiga adalah dua pertanyaan terbuka ke DBA. Menaruhnya di sini membuat
// jawabannya terbaca dengan satu perintah, bukan ditemukan pengguna sebagai layar yang
// gagal dimuat.
func checkPicTeknik(
	ctx context.Context,
	repo *masterpictekniksql.Repo,
	legacy *sqlstore.Legacy,
	alias string,
	print func(string, ...any),
) {
	if err := repo.CheckTable(ctx); err != nil {
		print("  [GAGAL] POOLDATA.MST_USER_TEKNIK tidak dapat dibaca: %v", err)
	} else {
		print("  [ok]    POOLDATA.MST_USER_TEKNIK dapat dibaca")
	}

	// View TIDAK lagi diperiksa. Daftar maupun form sama-sama membaca
	// POOLDATA.MST_USER_TEKNIK — tempat datanya benar-benar disimpan — sehingga
	// ketersediaan view tidak lagi menentukan apa pun. Memeriksanya hanya akan
	// menghasilkan peringatan atas sesuatu yang tidak dipakai.

	// Direktori pegawai dipakai SETIAP kali PIC ditambah atau diubah: nama tidak pernah
	// diketik, selalu dicari. Tanpa barisnya, layar tetap dapat menampilkan daftar tetapi
	// tidak dapat menyimpan satu pun perubahan.
	address, err := legacy.ServiceAddress(ctx, alias, masterpicteknikdirectory.DefaultServiceKind)
	switch {
	case errors.Is(err, provider.ErrServiceNotRegistered):
		print("  [BELUM] alamat layanan direktori pegawai belum terdaftar:")
		print("            APP=%q TYPESERVICE=%q di POOLDATA.GCNM_CONNECT_REST",
			alias, masterpicteknikdirectory.DefaultServiceKind)
		print("            Tanpa baris ini, menambah dan mengubah PIC Teknik tidak dapat")
		print("            dilakukan — nama petugas dicari ke layanan itu, tidak diketik.")
	case err != nil:
		print("  [GAGAL] alamat layanan direktori pegawai tidak dapat dibaca: %v", err)
	default:
		// Alamatnya tidak dicetak: ia hostname sistem internal, dan aturan penulisan
		// `D-69` melarang menuliskannya di keluaran yang dapat tersalin ke mana-mana.
		_ = address
		print("  [ok]    alamat layanan direktori pegawai terbaca untuk APP=%q", alias)
	}
}

// checkRecovery melaporkan kesiapan keempat bahan modul Master Recovery.
//
// Keempatnya diperiksa TERPISAH karena keempatnya dapat gagal sendiri-sendiri, dan tindak
// lanjutnya berbeda jauh:
//
//	tabel   MST_RECOVERY_ASM_PENJAMINAN  batch-nya sendiri; gagal = modul tidak dapat dipakai
//	tabel   MST_VIRTUAL_ACCOUNT_PNC      pilihan principal; gagal = isian tidak punya pilihan
//	tabel   DATA_ATTACHFILE              bukti bayar; gagal = unggahan tidak dapat disimpan
//	DB Link MST_DET_SALES@ASMD           identitas polis; gagal TIDAK menghalangi pencatatan
//	baris   GCNM_CONNECT_REST            penerbit VA; gagal = VA baru tidak dapat diterbitkan
//
// Perbedaan derajat itu yang membuat pemeriksaan ini berguna: DB Link yang mati hanya
// membuat empat kolom identitas kosong — batch tetap tersimpan — sementara tabel batch
// yang tidak terbaca membuat seluruh layar tidak dapat dipakai. Melaporkan keduanya
// dengan nada yang sama akan menyesatkan orang yang membacanya.
func checkRecovery(
	ctx context.Context,
	repo *masterrecoverysql.Repo,
	legacy *sqlstore.Legacy,
	alias string,
	print func(string, ...any),
) {
	if err := repo.CheckTable(ctx); err != nil {
		print("  [GAGAL] tabel Master Recovery tidak dapat dibaca: %v", err)
	} else {
		print("  [ok]    POOLDATA.MST_RECOVERY_ASM_PENJAMINAN, MST_VIRTUAL_ACCOUNT_PNC,")
		print("            dan DATA_ATTACHFILE dapat dibaca")
	}

	// Tabel baris klaim dibuat migrasi kita sendiri, bukan warisan — kegagalannya berarti
	// migrasinya belum dijalankan, bukan soal hak akses.
	if err := repo.CheckClaimLineTable(ctx); err != nil {
		print("  [BELUM] POOLDATA.CPNC_RECOVERY_BARIS_KLAIM tidak dapat dibaca: %v", err)
		print("            Migrasi backend/migrations/0013 tampaknya belum dijalankan DBA.")
		print("            Akibatnya TIDAK terbatas: daftar polis disimpan di tabel ini, dan")
		print("            karena ia satu transaksi dengan kepala batch, SELURUH penyimpanan")
		print("            batch akan gagal selama tabelnya belum ada.")
	} else {
		print("  [ok]    POOLDATA.CPNC_RECOVERY_BARIS_KLAIM dapat dibaca")
	}

	// Kegagalan di sini sengaja bertanda [BELUM], bukan [GAGAL]: ia tidak menghalangi
	// pencatatan batch sama sekali.
	if err := repo.CheckPolicyLink(ctx); err != nil {
		print("  [BELUM] DB Link MST_DET_SALES@ASMD tidak dapat ditembak: %v", err)
		print("            Akibatnya TERBATAS: batch recovery tetap tersimpan, tetapi")
		print("            keempat kolom identitas polis (LBU_ID, LDC_ID, LAG_AGEN_ID,")
		print("            LMO_ID) kosong. Penggantinya adalah API Master Sales pada")
		print("            D-25, yang belum dibangun (R-03).")
	} else {
		print("  [ok]    DB Link MST_DET_SALES@ASMD dapat ditembak")
	}

	// Penerbit VA dipakai setiap kali principal BARU didaftarkan. Tanpa barisnya, layar
	// tetap dapat mencatat batch untuk principal yang sudah punya VA — hanya penerbitan
	// yang baru yang tidak mungkin.
	address, err := legacy.ServiceAddress(ctx, alias, masterrecoveryva.DefaultServiceKind)
	switch {
	case errors.Is(err, provider.ErrServiceNotRegistered):
		print("  [BELUM] alamat layanan penerbit virtual account belum terdaftar:")
		print("            APP=%q TYPESERVICE=%q di POOLDATA.GCNM_CONNECT_REST",
			alias, masterrecoveryva.DefaultServiceKind)
		print("            Tanpa baris ini, VA untuk principal BARU tidak dapat")
		print("            diterbitkan. Principal yang sudah punya VA tetap dapat dipakai.")
	case err != nil:
		print("  [GAGAL] alamat layanan penerbit virtual account tidak dapat dibaca: %v", err)
	default:
		// Alamatnya tidak dicetak: ia hostname sistem internal, dan aturan penulisan
		// `D-69` melarang menuliskannya di keluaran yang dapat tersalin ke mana-mana.
		_ = address
		print("  [ok]    alamat layanan penerbit virtual account terbaca untuk APP=%q", alias)
	}
}

func checkLoginTable(ctx context.Context, legacy *sqlstore.Legacy, print func(string, ...any)) {
	if err := legacy.CheckTable(ctx, "local_login_check_table"); err != nil {
		print("  [GAGAL] POOLDATA.M_LOGIN_PNC tidak dapat dibaca: %v", err)
		return
	}
	print("  [ok]    POOLDATA.M_LOGIN_PNC dapat dibaca")
}

// checkLogin menjalankan rantai identitas yang sama persis dengan yang dipakai
// aplikasi — HCQ dulu, lalu POOLDATA.M_LOGIN_PNC — tanpa menerbitkan sesi.
func checkLogin(
	ctx context.Context,
	cfg config.Config,
	legacy *sqlstore.Legacy,
	login string,
	passwordSource io.Reader,
	print func(string, ...any),
) error {
	password, err := readPassword(passwordSource)
	if err != nil {
		return err
	}

	rantai, err := buildIdentity(cfg, false, legacy)
	if err != nil {
		return err
	}

	print("Mencoba masuk sebagai %q …", login)
	profile, err := rantai.Verify(ctx, auth.Credential{Username: login, Password: password})
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrWrongCredential):
			print("  [TOLAK] kredensial tidak diterima HCQ maupun POOLDATA.M_LOGIN_PNC.")
			print("          Keduanya menjawab; jadi jalur integrasinya hidup, kredensialnya yang tidak cocok.")
		case errors.Is(err, auth.ErrUserInactive):
			print("  [TOLAK] akun ada tetapi tidak aktif.")
		case errors.Is(err, auth.ErrIdentitySystemUnreachable):
			print("  [GAGAL] sistem identitas tidak dapat dihubungi: %v", err)
		default:
			print("  [GAGAL] %v", err)
		}
		return nil
	}

	print("  [ok]    diterima")
	print("            jenis      : %s", profile.Kind)
	print("            identitas  : %s", profile.Identity)
	print("            nama       : %s", profile.Name)
	printIfPresent(print, "login      ", profile.Login)
	printIfPresent(print, "email      ", profile.Email)
	printIfPresent(print, "perusahaan ", profile.Company)
	printIfPresent(print, "cabang     ", profile.Branch)
	printIfPresent(print, "kode cabang", profile.BranchCode)
	printIfPresent(print, "jabatan    ", profile.Position)
	if profile.ActiveAtSource != nil {
		print("            aktif di sumber: %t (direkam, belum dipakai menolak)", *profile.ActiveAtSource)
	}
	return nil
}

func printIfPresent(print func(string, ...any), label, value string) {
	if strings.TrimSpace(value) != "" {
		print("            %s: %s", label, value)
	}
}

// readPassword membaca kata sandi dari stdin.
//
// Ia sengaja TIDAK diterima sebagai argumen baris perintah: argumen tersimpan di riwayat
// shell dan terlihat oleh siapa pun yang menjalankan daftar proses.
func readPassword(source io.Reader) (string, error) {
	if source == nil {
		source = os.Stdin
	}
	rowScanner := bufio.NewScanner(source)
	if !rowScanner.Scan() {
		if err := rowScanner.Err(); err != nil {
			return "", fmt.Errorf("membaca kata sandi dari stdin: %w", err)
		}
		return "", errors.New("kata sandi tidak terbaca dari stdin")
	}
	password := strings.TrimRight(rowScanner.Text(), "\r\n")
	if password == "" {
		return "", errors.New("kata sandi kosong")
	}
	return password, nil
}

// checkTravelDocumentDetail memeriksa kesiapan Daftar Detail Dokumen Travel.
//
// Kedua objeknya diperiksa TERPISAH, karena keduanya gagal karena sebab yang berbeda dan
// menyatukan laporannya membuat pembaca menebak mana yang sebenarnya kurang:
//
//	V_LST_DOC_TRAVEL   hak baca belum diberikan, atau view-nya memang tidak ada
//	M_DOCTRAVEL        milik modul Master Dokumen Travel; hanya dibaca di sini
//
// V_LST_DOC_TRAVEL_COVERAGE dan M_PLANTRAVEL TIDAK ikut diperiksa, karena modul ini tidak
// menyentuh keduanya: grid Plan dan Jaminan tidak ada di layar Pega yang berjalan
// (Work Owner, 2026-10-03).
//
// # Yang sengaja TIDAK diperiksa di sini
//
// Tabel dasar dan urutan yang dipakai jalur tulis. Nama keduanya belum terverifikasi
// (`R-16` — activity penyimpannya hilang dari export), dan memeriksa urutan berarti
// MENGHABISKAN satu nomor — efek samping yang tidak pantas dimiliki mode periksa.
// Verifikasinya ada di migrations/0006, dijalankan DBA sekali.
//
// Akibat yang harus disadari pembaca laporan ini: seluruh baris [ok] di bawah hanya
// membuktikan jalur BACA siap. Jalur TULIS belum terbukti sampai migrasi 0006 dijawab.
func checkTravelDocumentDetail(ctx context.Context, primary *sql.DB, print func(string, ...any)) {
	repo := daftardetaildokumentravelsql.NewRepo(primary)
	if err := repo.CheckTable(ctx); err != nil {
		print("  [BELUM] Daftar Detail Dokumen Travel belum siap: %v", err)
		print("            View V_LST_DOC_TRAVEL adalah objek warisan Pega. Mintakan")
		print("            GRANT SELECT untuk akun aplikasi — lihat migrations/0006.")
		print("            Selama belum diberikan, layarnya tidak dapat dipakai terhadap")
		print("            Oracle; bagian lain tetap jalan.")
	} else {
		list, err := repo.List(ctx)
		if err != nil {
			print("  [GAGAL] POOLDATA.V_LST_DOC_TRAVEL tidak dapat dibaca isinya: %v", err)
		} else {
			print("  [ok]    POOLDATA.V_LST_DOC_TRAVEL dapat dibaca: %d detail dokumen", len(list))
		}
	}

	document := daftardetaildokumentravelsql.NewDocumentRepo(primary)
	if err := document.CheckTable(ctx); err != nil {
		print("  [GAGAL] POOLDATA.M_DOCTRAVEL tidak dapat dibaca: %v", err)
		print("            Tanpa itu isian ID Dokumen pada layar Daftar Detail kosong.")
		print("            Kodenya tetap dapat diketik sendiri, sehingga penyimpanan")
		print("            tidak ikut gagal — yang hilang hanya daftar pilihannya.")
	} else if list, err := document.List(ctx); err != nil {
		print("  [GAGAL] POOLDATA.M_DOCTRAVEL tidak dapat dibaca isinya: %v", err)
	} else {
		print("  [ok]    POOLDATA.M_DOCTRAVEL dapat dibaca: %d dokumen travel", len(list))
	}
}

// checkSurveyors melaporkan kesiapan POOLDATA.D_SURVEYORS — daftar ORANGNYA.
//
// Ia sempat TIDAK ADA di mode periksa ini, dan ketiadaannya berakibat nyata: layarnya
// gagal memuat di produksi sementara laporan periksa menyatakan semuanya siap. Yang
// diperiksa sekarang dua langkah, karena keduanya gagal dengan sebab yang berbeda —
// tabelnya tidak terbaca, atau terbaca tetapi kueri daftarnya yang menolak.
func checkSurveyors(ctx context.Context, repo *mastersurveyorssql.Repo, print func(string, ...any)) {
	if err := repo.CheckTable(ctx); err != nil {
		print("  [BELUM] Master Surveyors belum siap: %v", err)
		print("            Periksa keberadaan POOLDATA.D_SURVEYORS beserta hak SELECT akun")
		print("            aplikasi atasnya.")
		return
	}

	list, total, err := repo.List(ctx, mastersurveyors.Filter{})
	if err != nil {
		print("  [GAGAL] POOLDATA.D_SURVEYORS tidak dapat dibaca isinya: %v", err)
		return
	}
	print("  [ok]    POOLDATA.D_SURVEYORS dapat dibaca: %d surveyor (%d terbaca)", total, len(list))
}

// checkDocumentType melaporkan kesiapan POOLDATA.LST_DOC_TYPE.
//
// Ditambahkan bersama checkSurveyors di atas dan dengan alasan yang sama: modulnya sudah
// dipakai layar, tetapi kesiapannya tidak pernah ikut diperiksa.
func checkDocumentType(ctx context.Context, repo *daftartipedokumensql.Repo, print func(string, ...any)) {
	if err := repo.CheckTable(ctx); err != nil {
		print("  [BELUM] Daftar Tipe Dokumen belum siap: %v", err)
		print("            Kolomnya dibuat migrasi 0005_daftar_tipe_dokumen. Selama belum")
		print("            dijalankan, layarnya tidak dapat dipakai terhadap Oracle.")
		return
	}

	list, err := repo.List(ctx)
	if err != nil {
		print("  [GAGAL] POOLDATA.LST_DOC_TYPE tidak dapat dibaca isinya: %v", err)
		return
	}
	print("  [ok]    POOLDATA.LST_DOC_TYPE dapat dibaca: %d tipe dokumen", len(list))
}

// checkDocumentObject melaporkan kesiapan Daftar Objek Dokumen.
//
// # Kenapa laporannya lebih rinci daripada modul master lain
//
// Karena dua dari tiga objek yang dipakainya adalah DUGAAN. Jalur simpan layar lama —
// `CNMInsertLstDocObj_act` dan `SetsLstDocObjValue_act` — hilang dari export (`R-16`), dan
// tidak ada `PEGA_LST_DOC_OBJ.prc` di `Database/`. Nama tabel dasar dan nama tabel pemetaan
// bisnisnya karena itu diturunkan dari pola tabel bersaudaranya, bukan dibaca.
//
// Laporan ini adalah tempat dugaan itu dibuktikan benar atau salah — sebelum pengguna
// pertama menekan Simpan, bukan sesudahnya. Karena itu ketiga objeknya diperiksa SATU PER
// SATU: mengetahui MANA yang gagal adalah seluruh gunanya.
// checkDetailDocumentType melaporkan kesiapan Daftar Detail Tipe Dokumen (MENU_ID 41).
//
// Diperiksa TIGA langkah, bukan satu, karena ketiganya gagal dengan sebab yang berbeda dan
// menuntut tindakan yang berbeda pula:
//
//	baca     view induk ada dan dapat dibaca        -> layarnya dapat menampilkan data
//	tulis    kolom tabel dasarnya sesuai dugaan     -> layarnya dapat MENYIMPAN
//	rujukan  master tipe dokumen dapat dibaca       -> dropdown-nya terisi
//
// Langkah kedua yang paling penting: nama kolom tabel dasarnya diturunkan dari nama kolom
// view-nya, bukan dibaca dari procedure lama yang hanya menyebut (ID, JSON_DATA). Tanpa
// pemisahan ini, layar yang dapat menampilkan data tetapi gagal menyimpan akan terbaca
// [ok] sampai pengguna pertama menekan Simpan.
func checkDetailDocumentType(
	ctx context.Context,
	repo *daftardetailtipedokumensql.Repo,
	reference *daftardetailtipedokumensql.ReferenceRepo,
	print func(string, ...any),
) {
	if err := repo.CheckTable(ctx); err != nil {
		print("  [BELUM] Daftar Detail Tipe Dokumen belum siap: %v", err)
		print("            Dua kemungkinan, dan galat di atas membedakannya:")
		print("            - ORA-00942: POOLDATA.V_LST_DET_TYPE_DOC atau")
		print("              POOLDATA.V_LST_DOC_TYPE tidak ada di basis data entitas ini.")
		print("            - galat hak akses: mintakan GRANT SELECT untuk akun aplikasi.")
		print("            Selama belum selesai, layarnya tidak dapat dipakai terhadap")
		print("            Oracle; bagian lain tetap jalan.")
		return
	}

	list, err := repo.List(ctx)
	if err != nil {
		print("  [GAGAL] POOLDATA.V_LST_DET_TYPE_DOC tidak dapat dibaca isinya: %v", err)
		return
	}
	print("  [ok]    POOLDATA.V_LST_DET_TYPE_DOC dapat dibaca: %d detail tipe dokumen", len(list))

	// Jalur TULIS diperiksa terpisah, dan kegagalannya BUKAN sekadar catatan: nama kolom
	// tabel dasarnya diturunkan dari nama kolom view-nya, bukan dibaca dari procedure —
	// procedure lama hanya menyebut (ID, JSON_DATA). Bila tabelnya ternyata masih
	// berbentuk itu, kueri ini gagal dengan ORA-00904.
	if err := repo.CheckWriteTable(ctx); err != nil {
		print("  [GAGAL] tabel dasar Daftar Detail Tipe Dokumen tidak dapat ditulis: %v", err)
		print("            DAFTARNYA tetap dapat dimuat. Yang terblokir hanya MENYIMPAN.")
		print("            POOLDATA.LST_DET_TYPE_DOC sudah diverifikasi lengkap kolomnya")
		print("            pada 2026-09-23, jadi sebab yang paling mungkin adalah hak")
		print("            akses: mintakan GRANT INSERT, UPDATE untuk akun aplikasi.")
	}

	// Master rujukan diperiksa terakhir, dan kegagalannya CATATAN — bukan kegagalan.
	// Detail Dokumen tetap dapat disunting dan disimpan; yang kosong hanya dropdown ID
	// Tipe Dokumen.
	if err := reference.CheckTable(ctx); err != nil {
		print("  [catat] master tipe dokumen tidak dapat dibaca: %v", err)
		print("            Layar tetap dapat dipakai; yang kosong hanya isi dropdown")
		print("            ID Tipe Dokumen.")
	}
}

func checkDocumentObject(
	ctx context.Context,
	repo *daftarobjekdokumensql.Repo,
	business *daftarobjekdokumensql.BusinessRepo,
	print func(string, ...any),
) {
	if err := repo.CheckTable(ctx); err != nil {
		print("  [GAGAL] Daftar Objek Dokumen belum siap: %v", err)
		print("            Yang dipakai modul ini hanya POOLDATA.LST_DOC_OBJ beserta kolom")
		print("            ID, KET_DOC_OBJ, OLD_ID, dan JSON_DATA. Keempatnya SUDAH ADA di")
		print("            basis data, sehingga kegagalan di sini menyangkut HAK AKSES —")
		print("            bukan migrasi yang belum dijalankan.")
		return
	}

	list, err := repo.List(ctx)
	if err != nil {
		print("  [GAGAL] POOLDATA.LST_DOC_OBJ tidak dapat dibaca isinya: %v", err)
		return
	}

	// Baris tanpa nama dihitung dan dilaporkan, karena itulah gejala yang paling mungkin
	// dilaporkan pengguna — dan sebabnya ada di DATA, bukan di aplikasi.
	//
	// Isi master ini tinggal di JSON_DATA; kolom KET_DOC_OBJ kosong pada seluruh baris
	// warisan. Modul membaca keduanya, sehingga nama tetap tampil. Baris yang masih kosong
	// di KEDUANYA memang tidak punya nama di mana pun.
	tanpaNama := 0
	for _, row := range list {
		if strings.TrimSpace(row.Description) == "" {
			tanpaNama++
		}
	}

	print("  [ok]    POOLDATA.LST_DOC_OBJ dapat dibaca: %d objek dokumen", len(list))
	if tanpaNama > 0 {
		print("  [catat] %d baris tanpa nama di kolom MAUPUN di dokumen JSON-nya", tanpaNama)
		print("            Layar menampilkannya sebagai baris kosong. Itu isi datanya,")
		print("            bukan cacat pembacaan.")
	}

	// Master bisnis diperiksa terpisah: ia milik GISFW (`D-03`) dan hak bacanya diminta
	// sendiri ke DBA.
	//
	// Kegagalannya MENGHALANGI penyimpanan, dan karena itu dilaporkan sebagai GAGAL — bukan
	// sebagai catatan seperti pada modul Master COL Simas Online. Pemetaan di sini hanya
	// menyimpan ID bisnis, sehingga tanpa master tidak ada cara mengubah nama yang diketik
	// pengguna menjadi sesuatu yang dapat disimpan.
	if err := business.CheckTable(ctx); err != nil {
		print("  [GAGAL] POOLDATA.BUSINESS tidak dapat dibaca: %v", err)
		print("            Objek dokumen TANPA bisnis tetap dapat disimpan; yang memakai")
		print("            isian Bisnis akan ditolak sampai hak bacanya diberikan.")
	}
}

// checkBusinessDocumentRule melaporkan kesiapan modul Daftar Tipe Dokumen Bisnis.
//
// # Kenapa keempat master rujukannya TIDAK ikut diperiksa di sini
//
// Keempatnya sudah diperiksa modul PEMILIKNYA masing-masing: POOLDATA.BUSINESS dan
// V_LST_DOC_OBJ oleh Daftar Objek Dokumen, V_LST_DOC_TYPE oleh Daftar Tipe Dokumen, dan
// V_LST_DET_TYPE_DOC oleh Daftar Detail Tipe Dokumen. Memeriksanya lagi di sini hanya
// menggandakan baris laporan tanpa menambah keterangan — dan bila salah satunya gagal,
// pembacanya akan melihat kegagalan yang sama dilaporkan dua kali dari dua modul berbeda.
//
// Kegagalan keempatnya pun TIDAK membuat layar ini tidak dapat dipakai: yang hilang hanya
// SARAN pada isian, karena keempat isian rujukannya boleh diketik sendiri — persis seperti
// autocomplete ber-`pyAllowFreeFormInput=true` di Pega.
func checkBusinessDocumentRule(
	ctx context.Context,
	repo *daftartipedokumenbisnissql.Repo,
	print func(string, ...any),
) {
	if err := repo.CheckTable(ctx); err != nil {
		print("  [BELUM] Daftar Tipe Dokumen Bisnis belum siap: %v", err)
		print("            Kedua tabelnya SUDAH ADA di sistem lama — modul ini tidak menuntut")
		print("            perubahan bentuk apa pun. Yang dituntut migrasi")
		print("            0010_daftar_tipe_dokumen_bisnis hanyalah HAK AKSES, ditambah enam")
		print("            pertanyaan ke DBA. Kegagalan di sini karena itu hampir selalu")
		print("            berarti grant-nya belum diberikan, bukan objeknya belum dibuat.")
		return
	}

	list, err := repo.ListBusinesses(ctx)
	if err != nil {
		print("  [GAGAL] POOLDATA.LST_TYPE_DOC_BUSINESS tidak dapat dibaca isinya: %v", err)
		return
	}
	print("  [ok]    POOLDATA.LST_TYPE_DOC_BUSINESS dapat dibaca: %d lini bisnis beraturan dokumen", len(list))

	// Yang TIDAK dibuktikan mode ini, dan perlu dinyatakan supaya laporannya tidak terbaca
	// sebagai "siap": jalur TULIS. Ia bergantung pada urutan LST_TYPE_DOC_BUSINESS_SEQ dan
	// pada keunikan kolom ID — keduanya pertanyaan yang masih terbuka di migrasi 0010
	// Bagian 1, dan memeriksanya dari sini berarti MENGHABISKAN satu nomor urut.
	print("            Yang terbukti hanya jalur BACA. Jalur simpan menunggu jawaban DBA")
	print("            atas migrasi 0010 Bagian 1 — terutama apakah kolom ID benar-benar unik.")
}

// checkAssembledModules melaporkan kesiapan sepuluh modul yang perakitannya dipulihkan di
// modules.go.
//
// # Kenapa satu fungsi, bukan sepuluh
//
// Yang diperiksa di sini SATU hal yang sama untuk semuanya: apakah tabelnya terbaca. Tidak
// ada angka yang berarti khusus per modul seperti "33 status" pada Master Status Klaim,
// sehingga sepuluh fungsi terpisah hanya akan mengulang bentuk yang sama sepuluh kali.
//
// Ia ditambahkan setelah tiga layar ditemukan gagal di produksi sementara laporan periksa
// menyatakan semuanya siap — celahnya bukan pada modulnya, melainkan pada apa yang
// diperiksa laporan ini.
func checkAssembledModules(ctx context.Context, primary *sql.DB, print func(string, ...any)) {
	autoClaim := masterautoclaimsql.NewRepo(primary)
	workshop := masterbengkelsql.NewRepo(primary)
	panel := masterpanelsql.NewRepo(primary)
	clause := masterpasalsql.NewRepo(primary)
	rejection := masterpenolakansql.NewRepo(primary)
	supplier := mastersuppliersql.NewRepo(primary)
	inboxCompliance := inboxcompliancesql.NewRepo(primary)
	inboxServiceCenter := inboxservicecentersql.NewRepo(primary)

	// Tab Registrasi SC dicari lewat FindTab, bukan disusun di sini, supaya pemeriksaan ini
	// memakai tab yang BENAR-BENAR terdaftar.
	serviceCenterTab, serviceCenterTabKnown := inboxservicecenter.FindTab(
		inboxservicecenter.TabRegistration)
	inboxManagerAdmin := inboxmanageradminsql.NewRepo(primary)
	inboxManager := inboxmanagersql.NewRepo(primary)

	// Tab Compliance dicari lewat FindTab, bukan disusun di sini, supaya pemeriksaan ini
	// memakai tab yang BENAR-BENAR terdaftar. Tab yang tidak ditemukan membuat kueri
	// daftarnya dilewati — bukan memanggil repo dengan tab kosong yang pasti gagal dan
	// terbaca seperti tabel yang tidak dapat dibaca.
	complianceTab, complianceTabKnown := inboxcompliance.FindTab(inboxcompliance.TabCompliance)

	// `list` menjalankan kueri DAFTAR yang sesungguhnya, bukan sekadar menyentuh tabelnya.
	//
	// Pembedaan itu yang menentukan: `CheckTable` hanya membuktikan tabelnya ada dan
	// terbaca, sedangkan yang membuat layar gagal biasanya kueri daftarnya — ia menyentuh
	// lebih banyak kolom, dan sering lebih dari satu tabel. Master Surveyors gagal persis
	// begitu: tabelnya ada, satu kolomnya tidak.
	probe := []struct {
		name  string
		check func(context.Context) error
		list  func(context.Context) (int, error)
	}{
		{"Inbox Admin", inboxadminsql.NewRepo(primary).CheckTable, nil},

		// Kueri daftarnya ikut dijalankan, bukan hanya tabelnya disentuh, dan di modul ini
		// pembedaan itu paling berharga: `list_compliance` menyentuh dua kolom yang
		// keberadaannya DISIMPULKAN, bukan dibuktikan dari DDL yang belum ada (`R-08`) —
		// `PC_ASM_FW_GCNMFW_WORK.PYORIGUSERID` dan
		// `T_CLAIM_PNC.COMPLIANCE_CREATEDATE`. Bila salah satunya ternyata bernama lain,
		// perintah `-periksa` yang menemukannya saat start, bukan petugas Compliance yang
		// menemukannya saat membuka layar.
		{"Inbox Compliance", inboxCompliance.CheckTable, func(ctx context.Context) (int, error) {
			if !complianceTabKnown {
				return 0, fmt.Errorf(
					"tab %q tidak terdaftar di inboxcompliance.Tabs()",
					inboxcompliance.TabCompliance)
			}

			page, err := inboxCompliance.List(
				ctx,
				inboxcompliance.Query{
					Tab:        complianceTab,
					Workbasket: inboxcompliance.WorkbasketCompliance,
				},
				inboxcompliance.Pagination{Page: 1, Size: 1},
			)
			return page.Total, err
		}},

		// Jalur TULIS modul Inbox Compliance diperiksa terpisah dari jalur bacanya.
		//
		// Kegagalannya berakibat berbeda, sehingga melaporkannya sebagai satu baris akan
		// menyesatkan: tanpa sequence, kedua tab tetap terbaca utuh dan yang gagal hanya
		// tombol Kirim ke Post Audit. Baris ini yang menemukannya saat start — kalau
		// tidak, petugas Compliance yang menemukannya saat menekan tombol, dengan galat
		// Oracle yang berbunyi "sequence does not exist" dan tidak menyebut sebabnya.
		{"Inbox Compliance (kirim ke Post Audit)",
			inboxCompliance.CheckPostAuditWritable, nil},

		// Tabel keputusan form Compliance Checker, dibuat migrasi 0012.
		//
		// Baris tersendiri dengan alasan yang sama seperti di atas: tanpa tabel ini kedua
		// tab tetap terbaca utuh dan daftar tetap tampil — yang gagal hanya tombol
		// "Simpan Data" pada form, dan galatnya baru muncul setelah petugas mengisi
		// seluruh form lalu menekan tombolnya.
		{"Inbox Compliance (simpan keputusan)",
			inboxCompliance.CheckDecisionWritable, nil},

		// Kueri daftarnya ikut dijalankan, dan di modul ini pembedaan itu justru paling
		// berharga: kueri grid aslinya TIDAK ADA di export (`R-16`) dan disusun ulang dari
		// tiga rule sekelas — lihat kepala inboxservicecenter.sql. Nama kolom yang meleset
		// karena itu bukan kemungkinan teoretis.
		//
		// Tab Registrasi SC yang dipakai, bukan tab lain, karena ia satu-satunya yang
		// menempuh cabang `STS_APPROVAL IS NULL`. Cabang itulah yang paling mudah salah
		// ditulis, sebab `= NULL` tidak pernah benar dan gagalnya DIAM: kuerinya berjalan,
		// hasilnya nol baris, dan layar terbaca seperti antrean yang memang kosong.
		{"Inbox Manager Admin", inboxManagerAdmin.CheckTable, func(ctx context.Context) (int, error) {
			tab, known := inboxmanageradmin.FindTab(inboxmanageradmin.TabNonMBU)
			if !known {
				return 0, fmt.Errorf(
					"tab %q tidak terdaftar di inboxmanageradmin.Tabs()",
					inboxmanageradmin.TabNonMBU)
			}

			rows, err := inboxManagerAdmin.List(ctx, inboxmanageradmin.Query{Tab: tab})
			return len(rows), err
		}},

		// Inbox Manager membaca TIGA tabel yang berbeda, dan CheckTable memeriksa
		// ketiganya tersendiri supaya pesan gagalnya menyebut satu hal saja.
		//
		// Yang dijalankan di bawah adalah kueri PENCACAH, bukan kueri daftar, dan itu
		// disengaja: pencacahnya menyentuh kesepuluh sumber layar ini sekaligus — termasuk
		// view `SPAREPART_HE` yang saat diperiksa 2026-09-28 berstatus INVALID. Kueri
		// daftar hanya akan menyentuh satu antrean.
		//
		// Pencacah yang sumbernya tidak terbaca TIDAK menggagalkan pemeriksaan ini: ia
		// dilaporkan per antrean lewat Counter.Unavailable, persis seperti yang dibaca
		// penyelia di layar. Angka di bawah karena itu jumlah pencacah yang BERHASIL, dan
		// selisihnya terhadap sepuluh adalah jumlah sumber yang sedang rusak.
		//
		// Yang TIDAK dapat dibuktikan perintah ini: apakah akun aplikasi punya hak TULIS
		// atas kesembilan tabel persetujuan. Memeriksanya menuntut menulis sungguhan, dan
		// perintah pemeriksa tidak boleh meninggalkan jejak di basis data mana pun.
		{"Inbox Manager", inboxManager.CheckTable, func(ctx context.Context) (int, error) {
			counters, err := inboxManager.Counters(ctx, inboxmanager.Caller{Login: "-periksa"})
			if err != nil {
				return 0, err
			}

			terbaca := 0
			for _, counter := range counters {
				if counter.Unavailable == "" {
					terbaca++
				}
			}
			return terbaca, nil
		}},

		{"Inbox Service Center", inboxServiceCenter.CheckTable, func(ctx context.Context) (int, error) {
			if !serviceCenterTabKnown {
				return 0, fmt.Errorf(
					"tab %q tidak terdaftar di inboxservicecenter.Tabs()",
					inboxservicecenter.TabRegistration)
			}

			page, err := inboxServiceCenter.List(
				ctx,
				inboxservicecenter.Query{
					Tab:      serviceCenterTab,
					Approval: inboxservicecenter.ApprovalFilter{MatchNull: true},
					// Login karangan: yang diperiksa adalah kuerinya dapat berjalan dan
					// kolomnya terbaca, bukan ada tidaknya baris milik seseorang.
					Caller: inboxservicecenter.Caller{Login: "PERIKSA"},
				},
				inboxservicecenter.Pagination{Page: 1, Size: 1},
			)
			if err != nil {
				return 0, err
			}

			// Kueri RINCIAN ikut dijalankan, dan justru inilah yang paling perlu.
			//
			// Ia menyebut **83 nama kolom** yang disusun ulang dari tiga rule sekelas —
			// bukan disalin dari satu rule yang ada. Satu nama yang meleset menghasilkan
			// ORA-00904 yang hanya menyebut kolom PERTAMA yang salah, sehingga menemukannya
			// lewat layar berarti menemukannya satu per satu.
			//
			// Login dan ID karangan: yang diperiksa keberadaan kolomnya, bukan barisnya.
			// Baris yang tidak ditemukan karena itu BUKAN kegagalan — ia jawaban yang
			// diharapkan.
			_, err = inboxServiceCenter.FindDetail(ctx, inboxservicecenter.DetailQuery{
				ID:     "PERIKSA",
				Caller: inboxservicecenter.Caller{Login: "PERIKSA"},
			})
			if err != nil && !errors.Is(err, inboxservicecenter.ErrNotFound) {
				return 0, err
			}

			// Riwayat progres menyentuh tabel yang BERBEDA
			// (`POOLDATA.PROGRESS_SERVICECENTER_CLAIM`), sehingga hak bacanya perlu
			// dibuktikan tersendiri. Riwayat kosong bukan kegagalan.
			if _, err := inboxServiceCenter.ListProgress(ctx, "PERIKSA"); err != nil {
				return 0, err
			}

			return page.Total, nil
		}},
		{"Master Auto Claim", autoClaim.CheckTable, func(ctx context.Context) (int, error) {
			row, err := autoClaim.List(ctx, masterautoclaim.Filter{})
			return len(row), err
		}},
		{"Master Bengkel", workshop.CheckTable, func(ctx context.Context) (int, error) {
			row, err := workshop.List(ctx, masterbengkel.Filter{})
			return len(row), err
		}},
		{"Master Panel", panel.CheckTable, func(ctx context.Context) (int, error) {
			row, err := panel.List(ctx, masterpanel.Filter{})
			return len(row), err
		}},
		{"Master Pasal Kerugian", clause.CheckTable, func(ctx context.Context) (int, error) {
			row, err := clause.List(ctx)
			return len(row), err
		}},
		{"Master Penolakan Klaim", rejection.CheckTable, func(ctx context.Context) (int, error) {
			row, err := rejection.ListParent(ctx)
			return len(row), err
		}},
		{"Master Sparepart", mastersparepartsql.NewRepo(primary).CheckTable, nil},
		{"Master Supplier", supplier.CheckTable, func(ctx context.Context) (int, error) {
			row, err := supplier.List(ctx, mastersupplier.Filter{})
			return len(row), err
		}},
	}

	for _, p := range probe {
		if err := p.check(ctx); err != nil {
			print("  [BELUM] %s belum siap: %v", p.name, err)
			continue
		}
		if p.list != nil {
			if n, err := p.list(ctx); err != nil {
				print("  [GAGAL] %s: tabelnya terbaca, tetapi kueri DAFTARNYA menolak: %v", p.name, err)
				continue
			} else {
				print("  [ok]    %s dapat dibaca: %d baris", p.name, n)
				continue
			}
		}
		print("  [ok]    %s dapat dibaca", p.name)
	}

	// View History Claim tidak punya CheckTable — ia membaca beberapa tabel sekaligus
	// menurut tipe pencarian yang dipilih, sehingga tidak ada satu tabel yang mewakilinya.
	print("  [CATATAN] View History Claim tidak diperiksa di sini: ia membaca tabel yang")
	print("            berbeda menurut tipe pencarian, tanpa satu tabel yang mewakilinya.")
}

// checkAutoClaim melaporkan kesiapan kedua tabel yang dipakai Inbox Auto Claim.
//
// Berbeda dari checkClaimStatus, ia TIDAK melaporkan jumlah baris. Dua sebab, dan
// keduanya disengaja:
//
//   - POOLDATA.TMP_BATCH_AUTO_CLAIM adalah tabel TRANSAKSI yang isinya berubah setiap
//     hari. Angka barisnya tidak menjawab pertanyaan apa pun tentang kesiapan.
//   - Menghitungnya berarti memindai tabel yang dapat sangat besar, dan mode periksa
//     dijalankan terhadap produksi.
//
// Yang dilaporkan hanya "dapat dibaca atau tidak" — dan itulah yang membedakan
// "tabelnya tidak ada" dari "akun aplikasi belum diberi hak baca".
// checkAutoClaimEveryCompany menyaring SETIAP perusahaan, bukan hanya yang pertama.
//
// Pemeriksaan sebelumnya mengambil satu perusahaan contoh, dan itu meloloskan cacat yang
// hanya mengenai sebagian: kode yang berbeda besar-kecil hurufnya, berspasi, atau punya
// lebih dari satu baris master. Laporan Work Owner 2026-09-20 — menyaring satu perusahaan
// tetapi baris perusahaan lain ikut tampil — tidak dapat ditangkap satu contoh.
//
// Dua hal diperiksa sekaligus untuk tiap perusahaan:
//
//	jumlahnya cocok    -> penyaringnya benar-benar menyaring
//	barisnya satu kode -> tidak ada baris perusahaan lain yang lolos
//
// Ditambah pemeriksaan baris kembar: dua baris dengan (kode, batch, tanggal) yang sama
// berarti join ke master MENGGANDAKAN baris — master punya lebih dari satu baris untuk
// kode itu.
func checkAutoClaimEveryCompany(ctx context.Context, repo *inboxautoclaimsql.Repo, print func(string, ...any)) {
	for _, source := range inboxautoclaim.AllSource() {
		summary, err := repo.SummarizeCompany(ctx, source)
		if err != nil {
			print("  [BELUM] tab %-15s ringkasan ditolak: %v", source.Label(), err)
			continue
		}

		bocor, selisih, kembar := 0, 0, 0
		for _, c := range summary.Company {
			page, err := repo.ListBatch(ctx, inboxautoclaim.BatchFilter{
				Source:      source,
				CompanyCode: c.Code,
				Page:        inboxautoclaim.PageRequest{Number: 1, Size: 500},
			})
			if err != nil {
				print("  [BELUM] tab %-15s penyaring ditolak: %v", source.Label(), err)
				break
			}
			if page.Total != c.BatchCount {
				selisih++
			}

			terlihat := map[string]bool{}
			for _, b := range page.Item {
				if b.CompanyCode != c.Code {
					bocor++
				}
				kunci := b.CompanyCode + "\x00" + b.BatchNumber + "\x00" + b.ProcessedDate
				if terlihat[kunci] {
					kembar++
				}
				terlihat[kunci] = true
			}
		}

		switch {
		case bocor > 0:
			print("  [BELUM] tab %-15s %d baris LOLOS penyaring — perusahaannya bukan yang diminta",
				source.Label(), bocor)
		case selisih > 0:
			print("  [BELUM] tab %-15s %d perusahaan: jumlah ringkasan dan hasil penyaring berbeda",
				source.Label(), selisih)
		case kembar > 0:
			print("  [BELUM] tab %-15s %d baris KEMBAR — (kode, batch, tanggal) yang sama muncul dua kali",
				source.Label(), kembar)
			print("            Join ke M_AUTO_CLAIM_PNC menggandakan baris: master punya lebih")
			print("            dari satu baris untuk kode perusahaan yang sama.")
		default:
			print("  [ok]    tab %-15s seluruh %d perusahaan tersaring benar, tanpa baris kembar",
				source.Label(), len(summary.Company))
		}
	}
}

// checkAutoClaimPremiumSource memeriksa sumber alamat layanan cek premi dan kolom polis.
//
// Work Owner menetapkan (2026-09-29) alamat layanan `CekPremiAutoKlaim` dibaca dari
// POOLDATA.GCNM_CONNECT_REST per portal: `app = <alias portal> AND typeservice = 'PREMI'`.
// Pemeriksaan ini hanya mencetak NAMA KOLOM dan JUMLAH baris per portal — SERVICENAME
// berisi alamat layanan dan TIDAK PERNAH dicetak (D-69).
//
// Kolom T_GENERAL ikut dicetak (nama dan tipe saja) karena JSON_POLIS.DATA_JSONBLOB
// dinyatakan tidak dipakai lagi, padahal Pega membaca periode, status, dan mata uang polis
// dari sana.
func checkAutoClaimPremiumSource(ctx context.Context, primary *sql.DB, print func(string, ...any)) {
	columns := func(owner, table string) []string {
		rows, err := primary.QueryContext(ctx,
			"SELECT COLUMN_NAME, DATA_TYPE FROM ALL_TAB_COLUMNS WHERE OWNER = :1 AND TABLE_NAME = :2 ORDER BY COLUMN_ID",
			owner, table)
		if err != nil {
			return nil
		}
		defer func() { _ = rows.Close() }()
		var out []string
		for rows.Next() {
			var name, kind string
			if rows.Scan(&name, &kind) == nil {
				out = append(out, name+"("+kind+")")
			}
		}
		return out
	}

	connect := columns("POOLDATA", "GCNM_CONNECT_REST")
	if len(connect) == 0 {
		print("  [BELUM] sumber cek premi : POOLDATA.GCNM_CONNECT_REST tidak terlihat di katalog")
	} else {
		print("  [ok]    sumber cek premi : GCNM_CONNECT_REST kolom %s", strings.Join(connect, ","))
		// PREMI dipakai pemeriksaan saat unggah; PREMI-API dipakai tab Cek Premi
		// (`CekPremi-Act` → GetPremiumPaid_SPK).
		for _, kind := range []string{"PREMI", "PREMI-API"} {
			rows, err := primary.QueryContext(ctx,
				"SELECT APP, COUNT(1) FROM POOLDATA.GCNM_CONNECT_REST WHERE TYPESERVICE = :1 GROUP BY APP ORDER BY APP", kind)
			if err != nil {
				print("  [BELUM] sumber cek premi : baris %s tidak terbaca: %v", kind, err)
				continue
			}
			var perApp []string
			for rows.Next() {
				var app sql.NullString
				var count int
				if rows.Scan(&app, &count) == nil {
					perApp = append(perApp, fmt.Sprintf("%q=%d", app.String, count))
				}
			}
			_ = rows.Close()
			print("          baris %s per APP: %s", kind, listOrDash(perApp))
		}

		// Rule GetPremiumPaid_SPK menanam alamatnya sendiri; aplikasi baru membacanya dari
		// katalog. Yang dicetak hanya apakah alamat katalog berakhir di layanan yang sama —
		// alamatnya sendiri tidak pernah dicetak (D-69).
		var matching, total int
		if err := primary.QueryRowContext(ctx,
			`SELECT COUNT(CASE WHEN SERVICENAME LIKE '%/getPaymentDataSumbis' THEN 1 END), COUNT(1)
			   FROM POOLDATA.GCNM_CONNECT_REST WHERE TYPESERVICE = 'PREMI-API'`).Scan(&matching, &total); err == nil {
			print("          PREMI-API menunjuk getPaymentDataSumbis: %d dari %d baris", matching, total)
		}
	}

	// Tab Cek Premi: daftar Bisnis dari POOLDATA.BUSINESS (BrowseBusiness_RD) dan daftar
	// Sumber Bisnis dari M_AUTO_CLAIM_PNC (BrowseAutoKlaim, approval='1').
	if business := columns("POOLDATA", "BUSINESS"); len(business) > 0 {
		print("          kolom POOLDATA.BUSINESS: %s", strings.Join(business, ","))
	} else {
		print("  [BELUM] Cek Premi        : POOLDATA.BUSINESS tidak terlihat di katalog")
	}
	var approved int
	if err := primary.QueryRowContext(ctx,
		"SELECT COUNT(1) FROM POOLDATA.M_AUTO_CLAIM_PNC WHERE APPROVAL = '1'").Scan(&approved); err != nil {
		print("  [BELUM] Cek Premi        : M_AUTO_CLAIM_PNC approval='1' tidak terbaca: %v", err)
	} else {
		print("          M_AUTO_CLAIM_PNC approval='1': %d baris", approved)
	}

	// T_GENERAL dipanggil tanpa skema di kueri penerima klaim; pemiliknya dicari.
	var owner sql.NullString
	_ = primary.QueryRowContext(ctx,
		"SELECT MIN(OWNER) FROM ALL_TAB_COLUMNS WHERE TABLE_NAME = 'T_GENERAL'").Scan(&owner)
	if general := columns(owner.String, "T_GENERAL"); len(general) > 0 {
		print("          kolom %s.T_GENERAL: %s", owner.String, strings.Join(general, ","))
	}
}

// checkAutoClaimPremiumCheck menjalankan kueri tab Cek Premi terhadap Oracle.
//
// Yang dicetak hanya JUMLAH pilihan dan apakah kueri total klaim berjalan — tidak ada nama
// perusahaan, kode, maupun nilai klaim yang dicetak.
func checkAutoClaimPremiumCheck(ctx context.Context, primary *sql.DB, repo *inboxautoclaimsql.Repo, print func(string, ...any)) {
	choices, err := repo.PremiumCheckChoices(ctx)
	if err != nil {
		print("  [GAGAL] Cek Premi        : pilihan tidak terbaca: %v", err)
		return
	}
	print("  [ok]    Cek Premi        : %d bisnis, %d sumber bisnis", len(choices.Business), len(choices.SourceOfBusiness))
	if len(choices.Business) == 0 || len(choices.SourceOfBusiness) == 0 {
		return
	}
	total, err := repo.SucceededClaimTotal(ctx, inboxautoclaim.PremiumCheckQuery{
		BusinessCode:     choices.Business[0].Code,
		SourceOfBusiness: choices.SourceOfBusiness[0].Code,
	})
	if err != nil {
		print("  [GAGAL] Cek Premi        : total klaim tidak terbaca: %v", err)
		return
	}
	print("  [ok]    Cek Premi        : kueri total klaim berjalan (terisi=%t)", total != "")

	// Pasangan yang PASTI punya klaim sukses, supaya SUM(NILAIKLAIM) teruji pada data
	// nyata — kolomnya belum tentu NUMBER. Pasangannya tidak dicetak.
	var business, source sql.NullString
	if err := primary.QueryRowContext(ctx, `SELECT G.BUSINESSCODE, G.SOURCEOFBUSINESS
		  FROM POOLDATA.TMP_BATCH_CLAIM_KREDIT A
		  JOIN POOLDATA.T_GENERAL G ON G.NOPOLIS = A.NOPOLIS
		 WHERE A.TMP_MESSAGE = :1 AND G.BUSINESSCODE IS NOT NULL AND G.SOURCEOFBUSINESS IS NOT NULL
		 FETCH NEXT 1 ROWS ONLY`, inboxautoclaim.MessageSuccess).Scan(&business, &source); err != nil {
		print("  [BELUM] Cek Premi        : tidak ada klaim Kredit sukses untuk menguji penjumlahan: %v", err)
		return
	}
	total, err = repo.SucceededClaimTotal(ctx, inboxautoclaim.PremiumCheckQuery{
		BusinessCode: business.String, SourceOfBusiness: source.String,
	})
	if err != nil {
		print("  [GAGAL] Cek Premi        : penjumlahan klaim sukses gagal: %v", err)
		return
	}
	print("  [ok]    Cek Premi        : penjumlahan klaim sukses berjalan pada data nyata (terisi=%t)", total != "")
}

// checkAutoClaimUploadLookups menjalankan kueri pemeriksaan unggahan terhadap Oracle.
//
// Polis contohnya diambil dari rincian batch yang sudah ada dan TIDAK dicetak; yang
// dicetak hanya apakah setiap kueri berjalan dan menemukan sesuatu. Tidak ada penulisan.
func checkAutoClaimUploadLookups(ctx context.Context, repo *inboxautoclaimsql.Repo, print func(string, ...any)) {
	for _, source := range inboxautoclaim.AllSource() {
		first, err := repo.ListBatch(ctx, inboxautoclaim.BatchFilter{
			Source: source, Page: inboxautoclaim.PageRequest{Number: 1, Size: 1},
		})
		if err != nil || len(first.Item) == 0 {
			continue
		}
		b := first.Item[0]
		lines, err := repo.ListLine(ctx, inboxautoclaim.LineQuery{
			Source: source, CompanyCode: b.CompanyCode, BatchNumber: b.BatchNumber,
			Page: inboxautoclaim.PageRequest{Number: 1, Size: 1},
		})
		if err != nil || len(lines.Item) == 0 {
			continue
		}
		policyNo := lines.Item[0].PolicyNo

		seq, seqFound, err := repo.FindPolicyProductSeq(ctx, policyNo)
		if err != nil {
			print("  [BELUM] pencarian unggah %-15s prodke: %v", source.Label(), err)
			continue
		}
		detail, detailFound, err := repo.FindPolicyDetail(ctx, policyNo, seq)
		if err != nil {
			print("  [BELUM] pencarian unggah %-15s data polis T_GENERAL: %v", source.Label(), err)
			continue
		}
		_, currencyFound, err := repo.CurrencyID(ctx, detail.Currency)
		if err != nil {
			print("  [BELUM] pencarian unggah %-15s mata uang: %v", source.Label(), err)
			continue
		}
		_, err = repo.HasOpenProtection(ctx, policyNo, inboxautoclaim.OpenProtectionPremiumType)
		if err != nil {
			print("  [BELUM] pencarian unggah %-15s open protection: %v", source.Label(), err)
			continue
		}
		_, err = repo.ContractClaimed(ctx, b.CompanyCode, "PERIKSA-TIDAK-ADA")
		if err != nil {
			print("  [BELUM] pencarian unggah %-15s kontrak ganda: %v", source.Label(), err)
			continue
		}
		print("  [ok]    pencarian unggah %-15s prodke=%v polis=%v periode=%v mata-uang=%v",
			source.Label(), seqFound, detailFound, detail.StartDate != "" && detail.EndDate != "", currencyFound)
	}
}

// checkAutoClaimDetail membuka rincian dan ekspor batch pertama tiap tab.
//
// Menjawab cacat yang lolos seluruh uji: kueri rincian dan ekspor menyebut kolom tabel
// ANEKA, sehingga tombol Detail dan Export di tab Kredit dan Travel gagal ORA-00904. Yang
// dicetak hanya JUMLAH baris — bukan isinya.
func checkAutoClaimDetail(ctx context.Context, repo *inboxautoclaimsql.Repo, print func(string, ...any)) {
	for _, source := range inboxautoclaim.AllSource() {
		first, err := repo.ListBatch(ctx, inboxautoclaim.BatchFilter{
			Source: source, Page: inboxautoclaim.PageRequest{Number: 1, Size: 1},
		})
		if err != nil || len(first.Item) == 0 {
			print("  [lewat] rincian tab %-15s tidak ada batch untuk dibuka", source.Label())
			continue
		}
		b := first.Item[0]
		query := inboxautoclaim.LineQuery{
			Source: source, CompanyCode: b.CompanyCode, BatchNumber: b.BatchNumber,
			Page: inboxautoclaim.PageRequest{Number: 1, Size: 15},
		}
		page, err := repo.ListLine(ctx, query)
		if err != nil {
			print("  [BELUM] rincian tab %-15s ditolak: %v", source.Label(), err)
			continue
		}
		exported, err := repo.ExportLine(ctx, query)
		if err != nil {
			print("  [BELUM] ekspor tab %-15s ditolak: %v", source.Label(), err)
			continue
		}
		print("  [ok]    rincian tab %-15s %d baris; ekspor %d baris", source.Label(), page.Total, len(exported))
	}
}

// checkAutoClaimPaging memastikan halaman 2 tidak mengulang baris halaman 1.
//
// Ini bukan kerapian. `OFFSET … FETCH NEXT` memotong hasil menurut URUTAN, dan bila
// urutannya tidak menentukan satu susunan tunggal, basis data boleh menyusun baris yang
// seri dengan cara berbeda pada setiap eksekusi. Akibatnya satu baris dapat muncul di dua
// halaman sementara baris lain tidak pernah muncul sama sekali — tanpa galat apa pun.
//
// Gejalanya di layar: menekan Next lalu Previous menampilkan isi yang berbeda, dan
// sebagian batch "hilang". Pada tabel dengan 100 baris grid, itu bukan kemungkinan
// teoretis.
func checkAutoClaimPaging(ctx context.Context, repo *inboxautoclaimsql.Repo, print func(string, ...any)) {
	for _, source := range inboxautoclaim.AllSource() {
		const size = 15

		satu, err := repo.ListBatch(ctx, inboxautoclaim.BatchFilter{
			Source: source, Page: inboxautoclaim.PageRequest{Number: 1, Size: size},
		})
		if err != nil {
			print("  [BELUM] tab %-15s halaman 1 ditolak: %v", source.Label(), err)
			continue
		}
		if satu.Total <= size {
			print("  [ok]    tab %-15s hanya satu halaman (%d baris)", source.Label(), satu.Total)
			continue
		}

		dua, err := repo.ListBatch(ctx, inboxautoclaim.BatchFilter{
			Source: source, Page: inboxautoclaim.PageRequest{Number: 2, Size: size},
		})
		if err != nil {
			print("  [BELUM] tab %-15s halaman 2 ditolak: %v", source.Label(), err)
			continue
		}

		pada1 := map[string]bool{}
		for _, b := range satu.Item {
			pada1[b.CompanyCode+"\x00"+b.BatchNumber+"\x00"+b.ProcessedDate] = true
		}
		ulang := 0
		for _, b := range dua.Item {
			if pada1[b.CompanyCode+"\x00"+b.BatchNumber+"\x00"+b.ProcessedDate] {
				ulang++
			}
		}
		if ulang > 0 {
			print("  [BELUM] tab %-15s %d dari %d baris halaman 2 MENGULANG halaman 1",
				source.Label(), ulang, len(dua.Item))
			print("            Urutan kueri tidak menentukan satu susunan tunggal, sehingga")
			print("            OFFSET memotong di tempat yang berbeda pada tiap eksekusi.")
			continue
		}
		print("  [ok]    tab %-15s halaman 1 dan 2 tidak beririsan (%d + %d dari %d)",
			source.Label(), len(satu.Item), len(dua.Item), satu.Total)
	}
}

// checkAutoClaimUploadColumns memastikan setiap kolom yang ditulis unggahan ADA di tabel
// tiap tab.
//
// Ketiga tab memakai satu kueri sisip yang sama (`auto_claim_insert_upload`), padahal di
// Pega tabel Asuransi Kredit diisi dengan susunan kolom yang BERBEDA
// (`RDB List/InsertTempAsuransiKredit-SQL.xml`: NOASURANSI, TYPEKLAIM, TANGGALBAYAR, …).
// DDL ketiga tabel tidak ada di repo, sehingga satu-satunya cara memastikannya adalah
// katalog basis data. Kolom yang hilang berarti unggahan ke tab itu gagal dengan galat 500,
// bukan dengan pesan yang dapat diperbaiki pengguna.
//
// Yang dicetak hanya NAMA KOLOM — metadata skema, bukan data nasabah.
func checkAutoClaimUploadColumns(ctx context.Context, primary *sql.DB, print func(string, ...any)) {
	// Kolom yang ditulis kueri sisip masing-masing tab (auto_claim_line_insert*).
	writtenBy := map[inboxautoclaim.Source][]string{
		inboxautoclaim.SourceAneka: {
			"BATCH", "NOPOLIS", "PRODKE", "TGLPROSES", "USERINPUT", "IDPEGA", "TGLKEJADIAN",
			"TGLLAPOR", "CURRENCY", "COL_ID", "NILAIKLAIM", "NOTE", "KEYWORD", "TMP_MESSAGE",
			"NOAKSEPTASI", "OBJECTNAME", "FLAGTIDAKBAYAR",
		},
		inboxautoclaim.SourceKredit: {
			"BATCH", "NOPOLIS", "PRODKE", "TGLPROSES", "USERINPUT", "IDPEGA", "ACCEPTNO",
			"TMP_MESSAGE", "CURRENCY", "NILAIKLAIM", "NOASURANSI", "TYPEKLAIM", "TANGGALBAYAR",
		},
		inboxautoclaim.SourceTravel: {
			"BATCH", "NOPOLIS", "PRODKE", "TGLPROSES", "USERINPUT", "IDPEGA", "NOAKSEPTASI",
			"TMP_MESSAGE", "TGLKEJADIAN", "CURRENCY", "NILAIKLAIM", "FLAGTIDAKBAYAR",
			"REPORTDESCRIPTION",
		},
	}
	// Kolom yang diisi Pega tetapi TIDAK ditulis modul ini; dicetak bila ada di tabel
	// supaya kesenjangannya terlihat, bukan dianggap galat.
	legacyOnly := []string{"PROPOSEVALUE", "DEDUCTIBLE"}

	for _, source := range inboxautoclaim.AllSource() {
		written := writtenBy[source]
		info, _ := source.Info()
		owner, table, _ := strings.Cut(info.Table, ".")

		rows, err := primary.QueryContext(ctx,
			"SELECT COLUMN_NAME, DATA_TYPE FROM ALL_TAB_COLUMNS WHERE OWNER = :1 AND TABLE_NAME = :2",
			owner, table)
		if err != nil {
			print("  [gagal] kolom unggahan %-7s: katalog tidak terbaca: %v", source, err)
			continue
		}
		present := map[string]bool{}
		dataType := map[string]string{}
		for rows.Next() {
			var name, kind string
			if err := rows.Scan(&name, &kind); err == nil {
				present[strings.ToUpper(name)] = true
				dataType[strings.ToUpper(name)] = kind
			}
		}
		_ = rows.Close()
		if kind, exists := dataType["TANGGALBAYAR"]; exists {
			// Tanggal bayar dikirim sebagai nilai tanggal (paymentDate). Bila kolomnya
			// ternyata teks, pengirimannya harus diubah.
			print("          TANGGALBAYAR bertipe %s", kind)
		}

		if len(present) == 0 {
			print("  [gagal] kolom unggahan %-7s: tabel %s tidak terlihat di katalog", source, info.Table)
			continue
		}

		var missing, extra []string
		for _, column := range append([]string{info.CompanyColumn}, written...) {
			if !present[column] {
				missing = append(missing, column)
			}
		}
		for _, column := range legacyOnly {
			if present[column] {
				extra = append(extra, column)
			}
		}

		status := "[ok]   "
		if len(missing) > 0 {
			status = "[gagal]"
		}
		print("  %s kolom unggahan %-7s: %d kolom; tidak ada: %s; kolom Pega belum ditulis: %s",
			status, source, len(present), listOrDash(missing), listOrDash(extra))
		if len(missing) > 0 {
			// Susunan tabelnya dicetak utuh: itulah yang dibutuhkan untuk memperbaikinya,
			// dan DDL-nya tidak ada di repo.
			all := make([]string, 0, len(present))
			for name := range present {
				all = append(all, name)
			}
			sort.Strings(all)
			print("          kolom tabel %s: %s", info.Table, strings.Join(all, ","))
		}
	}
}

func listOrDash(list []string) string {
	if len(list) == 0 {
		return "-"
	}
	return strings.Join(list, ",")
}

// checkAutoClaimTabsDiffer memastikan ketiga tab benar-benar membaca tabel yang berbeda.
//
// Ia menjawab satu laporan yang tidak dapat dijawab oleh jumlah saja: "tab ANEKA
// menampilkan data Asuransi Kredit". Yang dibandingkan KUNCI barisnya — pasangan
// (kode perusahaan, nomor batch) — bukan namanya, sehingga keluarannya tidak pernah memuat
// nama perusahaan mana pun.
//
// Tumpang tindih tidak selalu berarti cacat: satu perusahaan boleh mengirim ke lebih dari
// satu lini, dan nomor batch dihitung per tabel sehingga angka yang sama wajar muncul di
// dua tabel. Yang MUSTAHIL benar adalah dua tab yang isinya sama persis.
func checkAutoClaimTabsDiffer(ctx context.Context, repo *inboxautoclaimsql.Repo, print func(string, ...any)) {
	kunci := map[inboxautoclaim.Source]map[string]bool{}
	for _, source := range inboxautoclaim.AllSource() {
		page, err := repo.ListBatch(ctx, inboxautoclaim.BatchFilter{
			Source: source,
			Page:   inboxautoclaim.PageRequest{Number: 1, Size: 200},
		})
		if err != nil {
			print("  [BELUM] tab %-15s tidak dapat dibaca untuk perbandingan: %v", source.Label(), err)
			return
		}
		set := map[string]bool{}
		for _, b := range page.Item {
			set[b.CompanyCode+"\x00"+b.BatchNumber] = true
		}
		kunci[source] = set
	}

	all := inboxautoclaim.AllSource()
	for i := 0; i < len(all); i++ {
		for j := i + 1; j < len(all); j++ {
			a, b := kunci[all[i]], kunci[all[j]]
			if len(a) == 0 || len(b) == 0 {
				continue
			}
			sama := 0
			for k := range a {
				if b[k] {
					sama++
				}
			}
			if sama == len(a) && sama == len(b) {
				print("  [BELUM] tab %s dan %s mengembalikan ISI YANG SAMA PERSIS (%d baris)",
					all[i].Label(), all[j].Label(), sama)
				print("            Keduanya membaca tabel yang berbeda, jadi ini berarti")
				print("            penyaring tab tidak sampai ke kueri.")
				continue
			}
			print("  [ok]    tab %-15s vs %-15s berbeda (%d dan %d baris, %d beririsan)",
				all[i].Label(), all[j].Label(), len(a), len(b), sama)
		}
	}
}

// checkSparepartStore melaporkan POOLDATA.M_SPAREPART_HE_BU sendirian.
//
// Ia dipanggil pada jalur GAGAL checkSparepart, bukan pada jalur normal, dan itu
// disengaja: kalau SPAREPART_HE tidak dapat dibaca, pertanyaan yang tersisa bukan lagi
// "berapa barisnya" melainkan **"apakah datanya masih ada"**.
//
// Keduanya adalah dua objek yang berbeda. Bila SPAREPART_HE memang view di atas
// JSONDATA milik M_SPAREPART_HE_BU — dugaan yang dicatat di banner mastersparepart.sql —
// maka view yang rusak TIDAK berarti datanya hilang, dan membedakan keduanya menentukan
// apakah yang diminta ke DBA adalah "perbaiki definisinya" atau "pulihkan datanya".
func checkSparepartStore(
	ctx context.Context,
	repo *mastersparepartsql.Repo,
	print func(string, ...any),
) {
	if err := repo.CheckJSONMirror(ctx); err != nil {
		print("  [catat] POOLDATA.M_SPAREPART_HE_BU juga tidak dapat dibaca: %v", err)

		if strings.Contains(err.Error(), "ORA-00942") {
			// ORA-00942 tidak membedakan "objeknya tidak ada" dari "objeknya ada tetapi
			// akun ini tidak diberi hak bacanya". Keduanya menuntut tindakan yang
			// berbeda, dan menebak salah satunya akan membuang satu putaran dengan DBA.
			print("            ORA-00942 berarti objeknya tidak ADA **atau** ada tetapi")
			print("            akun aplikasi tidak diberi hak bacanya — keduanya tidak")
			print("            dapat dibedakan dari sini.")
			print("            Patut diduga KEDUANYA SATU SEBAB: bila SPAREPART_HE adalah")
			print("            view di atas objek ini, objek yang hilang membuat view-nya")
			print("            ikut tidak dapat dikompilasi (ORA-04063 di atas).")
			print("            Yang menjawabnya:")
			print("              SELECT owner, object_type, status FROM all_objects")
			print("               WHERE object_name='M_SPAREPART_HE_BU';")
			print("              SELECT referenced_owner, referenced_name, referenced_type")
			print("                FROM all_dependencies")
			print("               WHERE owner='POOLDATA' AND name='SPAREPART_HE';")
			return
		}

		print("            Mintakan ke DBA keadaan kedua objek itu, bukan hanya salah satunya.")
		return
	}

	total, err := repo.CountJSONMirror(ctx)
	if err != nil {
		print("  [catat] POOLDATA.M_SPAREPART_HE_BU terbaca, tetapi barisnya tidak dapat dihitung: %v", err)
		return
	}

	print("  [ok]    POOLDATA.M_SPAREPART_HE_BU tetap terbaca: %d baris", total)
	print("            Datanya ADA. Yang rusak hanyalah objek yang membacanya, sehingga")
	print("            yang diminta ke DBA adalah memperbaiki definisinya — bukan")
	print("            memulihkan data.")
}

// checkSparepartNumbering melaporkan kesiapan penomoran ID.
func checkAutoClaim(ctx context.Context, repo *inboxautoclaimsql.Repo, print func(string, ...any)) {
	if err := repo.CheckTable(ctx); err != nil {
		print("  [BELUM] tabel Inbox Auto Claim belum dapat dibaca: %v", err)
		print("            Yang dibutuhkan: POOLDATA.TMP_BATCH_AUTO_CLAIM dan")
		print("            POOLDATA.M_AUTO_CLAIM_PNC. Keduanya milik sistem lama dan TIDAK")
		print("            dibuat migrasi mana pun — yang diperlukan hanya hak baca serta")
		print("            hak tulis pada yang pertama.")
		return
	}
	print("  [ok]    POOLDATA.TMP_BATCH_AUTO_CLAIM, M_AUTO_CLAIM_PNC, dan CURRENCY dapat dibaca")

	// Kueri ringkasan dijalankan sungguhan, bukan sekadar diperiksa keberadaan tabelnya.
	//
	// Sebabnya konkret: dua cacat modul ini — TGLPROSES yang bagian PRIMARY KEY, dan
	// `ORA-01008` dari bind yang diulang — LOLOS seluruh uji dan baru muncul saat Oracle
	// benar-benar menguraikan pernyataannya. Penyimpanan memori tidak menguraikan SQL,
	// jadi satu-satunya cara menangkap kelas cacat itu lebih awal adalah menjalankannya.
	//
	// Yang dicetak hanya JUMLAH, tidak pernah nama perusahaan: perintah ini dijalankan
	// terhadap basis data berisi data nasabah, dan keluarannya sering ditempel ke tiket.
	// KETIGA tab diperiksa, bukan hanya yang bawaan. Ketiganya membaca tabel yang
	// berbeda, sehingga satu tab yang berhasil tidak menjamin dua lainnya.
	for _, source := range inboxautoclaim.AllSource() {
		summary, err := repo.SummarizeCompany(ctx, source)
		if err != nil {
			print("  [BELUM] tab %-15s ringkasan ditolak basis data: %v", source.Label(), err)
			continue
		}

		// Penyaring perusahaan diuji SUNGGUHAN, memakai kode dari ringkasan tab ini.
		// Ia pernah rusak dengan cara yang tidak menghasilkan galat apa pun: kodenya
		// di-uppercase sebelum dibandingkan sementara kolomnya tidak, sehingga tabelnya
		// selalu kosong.
		var sample inboxautoclaim.CompanySummary
		for _, c := range summary.Company {
			if c.BatchCount > 0 {
				sample = c
				break
			}
		}
		if sample.Code == "" {
			print("  [ok]    tab %-15s terbaca: %d perusahaan, %d batch (belum ada isinya)",
				source.Label(), len(summary.Company), summary.Total)
			continue
		}

		page, err := repo.ListBatch(ctx, inboxautoclaim.BatchFilter{
			Source: source, CompanyCode: sample.Code,
		})
		switch {
		case err != nil:
			print("  [BELUM] tab %-15s penyaring perusahaan ditolak: %v", source.Label(), err)
		case page.Total != sample.BatchCount:
			print("  [BELUM] tab %-15s penyaring menghasilkan %d batch, ringkasan menyebut %d",
				source.Label(), page.Total, sample.BatchCount)
			print("            Keduanya WAJIB sama. Selisihnya berarti kode perusahaan diubah")
			print("            di salah satu jalur — spasi di ujung, atau besar kecil huruf.")
		default:
			print("  [ok]    tab %-15s %d perusahaan, %d batch; penyaring cocok (%d batch)",
				source.Label(), len(summary.Company), summary.Total, page.Total)
		}

		// Kode yang sama diuji SEKALI LAGI setelah dipangkas spasinya, karena itulah yang
		// benar-benar sampai ke repo lewat HTTP: handler memanggil `strings.TrimSpace`
		// pada `?perusahaan=`.
		//
		// Bila kolomnya bertipe CHAR, Oracle memadatkan nilainya dengan spasi. Kode yang
		// utuh cocok, kode yang dipangkas TIDAK — dan pemeriksaan di atas tidak akan
		// menangkapnya karena ia memakai nilai mentah. Gejalanya di layar persis
		// "penyaringnya tidak berfungsi": tidak ada galat, hanya tabel yang tidak berubah.
		trimmed := strings.TrimSpace(sample.Code)
		if trimmed == sample.Code {
			continue
		}
		viaHTTP, err := repo.ListBatch(ctx, inboxautoclaim.BatchFilter{
			Source: source, CompanyCode: trimmed,
		})
		if err != nil || viaHTTP.Total != page.Total {
			print("  [BELUM] tab %-15s kode perusahaan BERSPASI di ujung (%d karakter -> %d)",
				source.Label(), len(sample.Code), len(trimmed))
			print("            Lewat HTTP kodenya dipangkas, dan penyaringnya menghasilkan")
			print("            %d batch, bukan %d. Kolomnya kemungkinan CHAR, bukan VARCHAR2.",
				viaHTTP.Total, page.Total)
		}
	}
}

// checkClaimTreatyProp memeriksa modul Inbox Claim Treaty Prop (`MENU_ID 54`).
//
// Tiga hal diperiksa, dan ketiganya pernah menjadi sebab layar terbuka tetapi kosong di
// modul lain:
//
//  1. hak baca atas kedua tabel penugasan dan atas POOLDATA.JSON_KLAIM;
//  2. apakah `JSON_VALUE` benar-benar dapat dijalankan terhadap DATA_JSONBLOB;
//  3. apakah antrean teknik memang berisi — akun antreannya literal di kueri lama, dan
//     bila namanya berubah di produksi, tabnya kosong tanpa satu pun galat.
func checkClaimTreatyProp(
	ctx context.Context,
	repo *inboxclaimtreatypropsql.Repo,
	print func(string, ...any),
) {
	if err := repo.CheckTable(ctx); err != nil {
		print("  [BELUM] Tabel antrean treaty tidak dapat dibaca: %v", err)
		print("            Modul ini TIDAK menuntut migrasi — seluruh tabelnya milik Pega.")
		print("            Periksa hak SELECT akun aplikasi atas DATAPEGA.PC_ASSIGN_WORKLIST,")
		print("            DATAPEGA.PC_ASSIGN_WORKBASKET, dan POOLDATA.JSON_KLAIM.")
		return
	}
	print("  [ok]    Tabel antrean treaty dan POOLDATA.JSON_KLAIM dapat dibaca")

	// Satu halaman saja. Yang diperiksa adalah apakah kuerinya BERJALAN — termasuk
	// JSON_VALUE atas DATA_JSONBLOB, yang menuntut kolomnya benar-benar berisi JSON yang
	// sah. Bila blob-nya rusak, Oracle menolak di sini, bukan di layar pengguna.
	page := inboxclaimtreatyprop.Pagination{Page: 1, Size: 5}

	technical, found := inboxclaimtreatyprop.FindTab(inboxclaimtreatyprop.TabTechnical)
	if !found {
		print("  [GAGAL] Tab antrean teknik tidak terdaftar di modul")
		return
	}

	result, err := repo.List(ctx, inboxclaimtreatyprop.Query{Tab: technical}, page)
	if err != nil {
		print("  [GAGAL] Antrean teknik tidak dapat dibaca: %v", err)
		print("            Bila galatnya menyebut JSON, isi POOLDATA.JSON_KLAIM.DATA_JSONBLOB")
		print("            kemungkinan bukan JSON yang sah pada sebagian baris.")
		return
	}

	print("  [ok]    Antrean teknik terbaca: %d pekerjaan menunggu", result.Total)
	if result.Total == 0 {
		print("            Kosong BUKAN berarti gagal — tetapi periksa apakah akun antrean")
		print("            masih bernama %q di produksi. Nama itu literal di kueri lama,",
			inboxclaimtreatyprop.TechnicalWorkbasket)
		print("            dan bila berubah, tab ini kosong tanpa satu pun galat.")
	}

	// Tanggal Kejadian diperiksa tersendiri karena ia perbaikan `P-5` modul ini: di sistem
	// lama kolomnya SELALU kosong pada antrean teknik. Bila ia tetap kosong di sini,
	// sebabnya bukan lagi alias yang tertukar melainkan isi blob — dan itu temuan yang
	// berbeda, yang harus terbaca berbeda pula.
	for _, item := range result.Items {
		if item.LossDate == "" {
			print("  [PERIKSA] %s: Tanggal Kejadian kosong di DATA_JSONBLOB ($.DateOfLoss).",
				item.ClaimID)
			print("            Alias kueri sudah dibetulkan, jadi sebabnya ada di isi blob.")
			break
		}
	}
}

// checkClaimTreatyNonProp memeriksa modul Inbox Claim Treaty Non Prop (`MENU_ID 55`).
//
// Ia TERPISAH dari pemeriksaan layar Prop di atas, dan bukan karena kerapian: modul ini
// menyentuh satu tabel yang tidak disentuh layar Prop
// (`DATAPEGA.PC_ASM_FW_GCNMFW_WORK`) dan membaca kolom JSON yang BERBEDA pada tabel yang
// sama (`DATA_JSON`, bukan `DATA_JSONBLOB`). Hak baca atas yang satu tidak menyatakan apa
// pun tentang yang lain, dan kolom JSON yang salah tidak menghasilkan galat — ia hanya
// mengosongkan dua kolom di layar.
func checkClaimTreatyNonProp(
	ctx context.Context,
	repo *inboxclaimtreatynonpropsql.Repo,
	print func(string, ...any),
) {
	if err := repo.CheckTable(ctx); err != nil {
		print("  [BELUM] Tabel antrean treaty non-prop tidak dapat dibaca: %v", err)
		print("            Modul ini TIDAK menuntut migrasi — seluruh tabelnya milik Pega.")
		print("            Periksa hak SELECT akun aplikasi atas DATAPEGA.PC_ASSIGN_WORKLIST,")
		print("            DATAPEGA.PC_ASSIGN_WORKBASKET, DATAPEGA.PC_ASM_FW_GCNMFW_WORK,")
		print("            dan POOLDATA.JSON_KLAIM.")
		return
	}
	print("  [ok]    Ketiga tabel antrean treaty non-prop dapat dibaca")

	// Satu halaman saja. Yang diperiksa adalah apakah kuerinya BERJALAN — termasuk
	// JSON_VALUE atas DATA_JSON, yang menuntut kolomnya benar-benar berisi JSON yang sah.
	page := inboxclaimtreatynonprop.Pagination{Page: 1, Size: 5}

	technical, found := inboxclaimtreatynonprop.FindTab(
		inboxclaimtreatynonprop.TabTechnical)
	if !found {
		print("  [GAGAL] Tab antrean teknik tidak terdaftar di modul")
		return
	}

	result, err := repo.List(ctx, inboxclaimtreatynonprop.Query{Tab: technical}, page)
	if err != nil {
		print("  [GAGAL] Antrean teknik non-prop tidak dapat dibaca: %v", err)
		print("            Bila galatnya menyebut JSON, isi POOLDATA.JSON_KLAIM.DATA_JSON")
		print("            kemungkinan bukan JSON yang sah pada sebagian baris.")
		print("            Perhatikan kolomnya DATA_JSON, BUKAN DATA_JSONBLOB yang dibaca")
		print("            layar Treaty Prop — keduanya kolom berbeda pada tabel yang sama.")
		return
	}

	print("  [ok]    Antrean teknik non-prop terbaca: %d pekerjaan menunggu", result.Total)
	if result.Total == 0 {
		print("            Kosong BUKAN berarti gagal — tetapi periksa apakah akun antrean")
		print("            masih bernama %q di produksi. Nama itu literal di kueri lama,",
			inboxclaimtreatynonprop.TechnicalWorkbasket)
		print("            dan bila berubah, tab ini kosong tanpa satu pun galat.")
		print("            Periksa pula apakah nomor klaim non-prop masih berawalan %q.",
			inboxclaimtreatynonprop.ClaimPrefix)
	}

	// Kolom yang berasal dari PC_ASM_FW_GCNMFW_WORK diperiksa tersendiri.
	//
	// Gabungan ke tabel itu LEFT JOIN, sehingga penugasan yang objek kerjanya tidak
	// ditemukan tetap muncul — dengan nama tertanggung, Ceding Co, dan umur yang kosong
	// seluruhnya. Itu tidak menghasilkan galat apa pun, dan di layar terbaca seperti data
	// yang memang belum diisi. Bila seluruh baris begitu, yang salah adalah kunci
	// gabungannya, bukan datanya.
	for _, item := range result.Items {
		if item.InsuredName == "" && item.CedingCompany == "" {
			print("  [PERIKSA] %s: seluruh kolom dari PC_ASM_FW_GCNMFW_WORK kosong.",
				item.ClaimID)
			print("            Bila ini terjadi pada SEMUA baris, gabungan")
			print("            PXREFOBJECTKEY = PZINSKEY tidak menemukan pasangannya.")
			break
		}
	}
}

func checkManagerReceivePUCL(
	ctx context.Context,
	repo *inboxmanagerreceivepuclsql.Repo,
	print func(string, ...any),
) {
	if err := repo.CheckTable(ctx); err != nil {
		print("  [BELUM] Tabel Inbox Manager Receive / PUCL tidak dapat dibaca: %v", err)
		print("            Modul ini TIDAK menuntut migrasi — seluruh tabelnya milik Pega.")
		print("            Periksa hak SELECT akun aplikasi atas")
		print("            DATAPEGA.PC_ASM_FW_GCNMFW_WORK, DATAPEGA.PC_ASSIGN_WORKLIST,")
		print("            DATAPEGA.PC_ASSIGN_WORKBASKET, dan POOLDATA.T_CLAIM_RECIVEDCLAIM.")
		return
	}
	print("  [ok]    Keempat tabel Inbox Manager Receive / PUCL dapat dibaca")

	page := inboxmanagerreceivepucl.Pagination{Page: 1, Size: 5}

	// KEDUA tab diperiksa, bukan satu.
	//
	// Tab Receive membaca tabel penugasan yang berbeda dari tab RCL/PUCL, dan keduanya
	// dipisahkan penyaring yang justru paling mungkin keliru. Memeriksa satu tab saja akan
	// menyatakan modulnya sehat sementara separuhnya belum tersentuh.
	counts := map[string]int{}

	// firstReceiveReference menyimpan kunci berkas Receive pertama yang terbaca, dipakai
	// memeriksa kueri LAYAR KERJA sesudah perulangan ini. Ia diambil dari hasil nyata, bukan
	// dikarang: kunci karangan selalu menghasilkan "tidak ditemukan", sehingga pemeriksaannya
	// tidak akan pernah menyentuh satu kolom pun.
	firstReceiveReference := ""

	for _, code := range []string{
		inboxmanagerreceivepucl.TabReceive,
		inboxmanagerreceivepucl.TabRCLPUCL,
	} {
		tab, found := inboxmanagerreceivepucl.FindTab(code)
		if !found {
			print("  [GAGAL] Tab %s tidak terdaftar di modul", code)
			return
		}

		result, err := repo.List(
			ctx, inboxmanagerreceivepucl.Query{Tab: tab}, page)
		if err != nil {
			print("  [GAGAL] Tab %q tidak dapat dibaca: %v", tab.Name, err)
			print("            Bila galatnya menyebut kolom, periksa apakah nama kolom")
			print("            pada DATAPEGA.PC_ASM_FW_GCNMFW_WORK masih sama — DDL tabel")
			print("            itu belum pernah diterima (`R-08`), dan seluruh nama kolom")
			print("            di modul ini dibaca dari kueri Pega, bukan dari DDL.")
			return
		}

		counts[code] = result.Total
		print("  [ok]    Tab %q terbaca: %d baris", tab.Name, result.Total)

		// Kedua kolom dari tabel cermin diperiksa pada tab Receive saja — hanya di sana
		// gabungannya dipakai.
		if !tab.OpensReceiveDocument {
			continue
		}

		if len(result.Items) > 0 {
			firstReceiveReference = result.Items[0].Reference
		}

		// Jenis Klaim diperiksa TERPISAH dari isi tabel cermin, karena keduanya gagal karena
		// sebab yang berbeda: yang satu Group Panel yang berubah nilainya, yang lain
		// gabungan yang tidak cocok.
		//
		// Sejak kedua daftar Receive digabung menjadi satu tab, Group Panel yang berubah
		// TIDAK lagi mengosongkan sebuah tab — ia hanya membuat seluruh baris terbaca
		// "NONMBU". Itu jauh lebih sulit terlihat, sehingga justru perlu disebut di sini.
		hasPA := false
		for _, item := range result.Items {
			if item.ClaimType == inboxmanagerreceivepucl.ClaimTypePA {
				hasPA = true
				break
			}
		}
		if len(result.Items) > 0 && !hasPA {
			print("  [PERIKSA] Tidak ada satu pun berkas berjenis klaim %q pada halaman ini.",
				inboxmanagerreceivepucl.ClaimTypePA)
			print("            Periksa apakah Group Panel Personal Accident masih bernilai")
			print("            %q di produksi. Kode itu menggantikan",
				inboxmanagerreceivepucl.GroupPanelPA)
			print("            `.ReceiveDocument.TypeOfClaim` yang tidak punya kolom basis")
			print("            data; bila berbeda, SELURUH baris terbaca NONMBU tanpa satu")
			print("            pun galat.")
		}

		for _, item := range result.Items {
			if item.SenderName == "" && item.DocumentReceivedDate == "" {
				print("  [PERIKSA] %s: Nama Pengirim dan Tanggal Terima Dokumen kosong.",
					item.CaseID)
				print("            Keduanya dibaca dari POOLDATA.T_CLAIM_RECIVEDCLAIM lewat")
				print("            LEFT JOIN CLAIMID = PZINSKEY. Bila SELURUH baris begitu,")
				print("            tabel itu kosong atau kunci gabungannya tidak cocok —")
				print("            bukan datanya yang belum diisi.")
				break
			}
		}
	}

	if counts[inboxmanagerreceivepucl.TabReceive] == 0 {
		print("  [PERIKSA] Tab Receive kosong.")
		print("            Penyaringnya `GROUPPANEL_1 IS NOT NULL` — gabungan tepat dari")
		print("            kedua penyaring grid Pega. Bila kolom itu kosong di seluruh baris")
		print("            berkelas ReceiveDocument, tab ini kosong tanpa satu pun galat,")
		print("            persis seperti kedua grid di Pega.")
	}

	// LAYAR KERJA diperiksa tersendiri, dan alasannya nyata: kuerinya membaca EMPAT BELAS
	// kolom yang tidak disentuh kueri grid mana pun — tiga belas dari
	// POOLDATA.T_CLAIM_RECIVEDCLAIM ditambah NOTREGISTNOTE_1. Grid yang sehat karena itu
	// tidak menyatakan apa pun tentang layar kerjanya.
	checkReceiveDocument(ctx, repo, firstReceiveReference, print)

	if counts[inboxmanagerreceivepucl.TabRCLPUCL] == 0 {
		print("  [PERIKSA] Tab RCL/PUCL kosong.")
		print("            Periksa apakah akun antrean bersama masih bernama %q.",
			inboxmanagerreceivepucl.RCLPUCLWorkbasket)
		print("            Penyaring itu TIDAK ADA di Report Definition layar ini — ia")
		print("            diambil dari RDB List/CountKlaimPUCL-SQL.xml dan")
		print("            ReminderPUCL-SQL.xml. Bila namanya berubah, tab ini kosong tanpa")
		print("            satu pun galat.")
	}
}

// checkReceiveDocument melaporkan kesiapan kueri LAYAR KERJA penerimaan dokumen.
//
// # Kenapa ia terpisah dari pemeriksaan grid
//
// Karena yang dibacanya memang berbeda. Kueri grid membaca sembilan kolom; kueri layar kerja
// membaca dua puluh tiga, dan EMPAT BELAS di antaranya tidak disentuh kueri mana pun di
// aplikasi ini sebelumnya — tiga belas kolom POOLDATA.T_CLAIM_RECIVEDCLAIM ditambah
// NOTREGISTNOTE_1.
//
// Nama ketiga belas kolom itu dibaca dari pernyataan `update` di
// `Database/PROCINSERTDATARECIVEDKLAIM.prc`, bukan dari DDL — DDL tabelnya memang belum
// pernah diterima (`R-08`). Bila procedure di produksi sudah berbeda dari salinan yang
// diekspor, kuerinya gagal pada pemakaian PERTAMA di produksi. Pemeriksaan ini yang
// memindahkan kegagalan itu ke sini.
//
// # Kenapa kuncinya diambil dari hasil nyata
//
// Karena kunci karangan selalu menghasilkan "tidak ditemukan", dan jawaban itu tidak menyentuh
// satu kolom pun — sehingga pemeriksaannya akan lulus meski seluruh nama kolomnya salah.
func checkReceiveDocument(
	ctx context.Context,
	repo inboxmanagerreceivepucl.Repo,
	reference string,
	print func(string, ...any),
) {
	if reference == "" {
		print("  [LEWAT]  Layar kerja penerimaan dokumen tidak diperiksa — tab Receive")
		print("            kosong, sehingga tidak ada kunci berkas nyata untuk mencobanya.")
		return
	}

	doc, err := repo.Document(ctx, reference)
	if err != nil {
		print("  [GAGAL] Layar kerja penerimaan dokumen tidak dapat dibaca: %v", err)
		print("            Kueri ini membaca 13 kolom POOLDATA.T_CLAIM_RECIVEDCLAIM dan")
		print("            NOTREGISTNOTE_1, yang tidak disentuh kueri lain di aplikasi ini.")
		print("            Namanya dibaca dari Database/PROCINSERTDATARECIVEDKLAIM.prc,")
		print("            bukan dari DDL (`R-08`) — bila galatnya menyebut sebuah kolom,")
		print("            bandingkan procedure di produksi dengan salinan yang diekspor.")
		return
	}
	print("  [ok]    Layar kerja penerimaan dokumen terbaca: %s", doc.CaseID)

	// Tabel cermin diperiksa TERPISAH dari kuerinya, karena keduanya gagal karena sebab yang
	// berbeda: yang satu nama kolom, yang lain gabungan yang tidak pernah cocok.
	//
	// `LEFT JOIN` membuat kegagalan gabungan TIDAK menghasilkan galat apa pun — layarnya
	// terbuka, seluruh isiannya kosong, dan tidak ada apa pun yang menandakannya.
	if doc.SenderName == "" && doc.ReceivedAt == "" && doc.Chronology == "" {
		print("  [PERIKSA] Seluruh isian dari POOLDATA.T_CLAIM_RECIVEDCLAIM kosong pada")
		print("            berkas ini. Tabel itu digabung LEFT JOIN CLAIMID = PZINSKEY dan")
		print("            TIDAK PERNAH DIBACA sistem lama, sehingga kelengkapan isinya")
		print("            belum terverifikasi. Bila SELURUH berkas begitu, tabelnya kosong")
		print("            atau kunci gabungannya tidak cocok — bukan datanya yang belum")
		print("            diisi, dan layar kerjanya akan tampil kosong tanpa satu pun galat.")
	}
}

// checkPartCategory melaporkan kesiapan POOLDATA.GCNM_M_SPAREPART_CATEGORY.
//
// Tabelnya warisan Pega dan TIDAK dibuat migrasi aplikasi ini, sehingga "belum dapat dibaca"
// di sini berarti tabelnya memang tidak ada di entitas itu, atau akun aplikasi belum diberi
// hak bacanya — keduanya urusan DBA.
//
// # Lima hal dilaporkan, dan tiga di antaranya tidak ada padanannya di master lain
//
//  1. Jumlah baris per status — apakah ketiga tab layar akan terisi.
//  2. Ketersediaan penomoran. Berbeda dari Master Sparepart dan Master Panel, penomoran di
//     sini TIDAK memakai sequence maupun kode situs: ia `MAX(PART_CATEGORY_ID)+1` atas
//     tabelnya sendiri. Yang dapat gagal karena itu bukan ketiadaan sequence melainkan
//     tipe kolom yang ternyata bukan angka.
//  3. **Baris berstatus di luar '0', '1', dan '2'.** Baris seperti itu tidak muncul di satu
//     pun tab — ia ada di basis data tetapi tidak dapat dilihat maupun diputuskan siapa pun
//     dari layar. Sistem lama punya cacat yang sama dan tidak melaporkannya.
//  4. **Nama yang dipakai lebih dari satu baris.** Modul ini menolak nama ganda, tetapi
//     tidak ada constraint unik yang menjaganya (R-08). Baris kembar yang sudah terlanjur
//     ada membuat penyimpanan yang sebenarnya sah ikut tertolak.
//  5. **Sparepart yang menunjuk kategori yang tidak ada.** Ia pemeriksaan dari arah
//     sebaliknya terhadap checkSparepartLookup, dan ia ada di sini karena modul INILAH yang
//     kelak menolak sebuah kategori — sedangkan penolakan tidak memutuskan tautan yang
//     sudah ada.
func checkPartCategory(ctx context.Context, primary *sql.DB, print func(string, ...any)) {
	repo := masterkategorisparepartsql.NewRepo(primary)

	if err := repo.CheckTable(ctx); err != nil {
		print("  [BELUM] POOLDATA.GCNM_M_SPAREPART_CATEGORY belum dapat dibaca: %v", err)
		print("            Tabelnya warisan Pega dan TIDAK dibuat migrasi aplikasi ini.")
		print("            Mintakan hak bacanya ke DBA.")
		print("            Tabel yang sama dibaca modul Master Sparepart sebagai daftar")
		print("            acuan Kategori; bila yang ini gagal, dropdown di sana pun kosong.")
		return
	}

	total := 0
	pendingCount := 0
	for _, s := range []masterkategorisparepart.ApprovalStatus{
		masterkategorisparepart.StatusApproved,
		masterkategorisparepart.StatusPending,
		masterkategorisparepart.StatusRejected,
	} {
		count, err := repo.CountByStatus(ctx, s)
		if err != nil {
			print("  [GAGAL] GCNM_M_SPAREPART_CATEGORY status %q tidak dapat dihitung: %v", s, err)
			return
		}
		if s == masterkategorisparepart.StatusPending {
			pendingCount = count
		}
		total += count
		print("            status %q %-16s %d baris", s, s.Label(), count)
	}
	print("  [ok]    POOLDATA.GCNM_M_SPAREPART_CATEGORY dapat dibaca: %d baris berstatus dikenal", total)

	if pendingCount > 0 {
		print("  [catat] %d kategori sparepart menunggu persetujuan.", pendingCount)
	}

	checkPartCategoryNumbering(ctx, repo, print)
	checkPartCategoryIntegrity(ctx, repo, total, print)
}

// checkPartCategoryNumbering melaporkan kesiapan penomoran PART_CATEGORY_ID.
//
// Berbeda dari checkSparepartNumbering, ia TIDAK memakai satu nomor urut: penomoran di sini
// `MAX(...)+1` atas tabelnya sendiri, bukan sequence, sehingga membacanya tidak mengubah
// apa pun. Nomor yang dilaporkan adalah nomor yang benar-benar akan terpakai bila ada
// penambahan saat ini juga.
//
// Kegagalannya hampir selalu berarti satu hal: PART_CATEGORY_ID ternyata bukan kolom angka.
// Bila itu terjadi, penerbitan kunci tidak dapat dijalankan sama sekali, dan asumsi yang
// dipakai seluruh berkas masterkategorisparepart.sql harus ditinjau ulang.
func checkPartCategoryNumbering(
	ctx context.Context,
	repo *masterkategorisparepartsql.Repo,
	print func(string, ...any),
) {
	id, err := repo.NextID(ctx)
	if err != nil {
		print("  [GAGAL] penomoran ID kategori sparepart tidak dapat dijalankan: %v", err)
		print("            Penambahan kategori baru akan gagal. Penyebab paling mungkin:")
		print("            PART_CATEGORY_ID bukan kolom angka, sehingga MAX(...)+1 gagal.")
		print("            Bila benar begitu, asumsi masterkategorisparepart.sql harus")
		print("            ditinjau ulang — bukan hanya kueri penomorannya.")
		return
	}
	print("  [ok]    penomoran ID kategori sparepart siap; berikutnya: %s", id)
	print("            (angka berurut dari MAX(PART_CATEGORY_ID)+1, bukan sequence)")
	print("            (tidak ada nomor yang terpakai oleh pemeriksaan ini)")
}

// checkPartCategoryIntegrity melaporkan tiga keadaan data yang tidak dijaga constraint apa
// pun, dan yang ketiganya baru terlihat sebagai keluhan pengguna bila tidak diperiksa.
//
// Ketiganya BUKAN kegagalan: aplikasi tetap berjalan dengan ketiganya. Yang dilaporkan
// adalah keadaan, supaya ia diketahui sebelum petugas menanyakannya.
func checkPartCategoryIntegrity(
	ctx context.Context,
	repo *masterkategorisparepartsql.Repo,
	knownStatusRows int,
	print func(string, ...any),
) {
	all, err := repo.CountAll(ctx)
	if err != nil {
		print("  [GAGAL] jumlah seluruh baris kategori tidak dapat dihitung: %v", err)
		return
	}
	if all == 0 {
		print("  [catat] Tabelnya KOSONG. Dropdown Kategori pada layar Master Sparepart")
		print("            karena itu juga kosong, dan sparepart baru tidak dapat")
		print("            digolongkan sampai ada kategori yang disetujui di sini.")
		return
	}

	if unknown, err := repo.CountUnknownStatus(ctx); err != nil {
		print("  [GAGAL] baris berstatus tak dikenal tidak dapat dihitung: %v", err)
	} else if unknown > 0 {
		print("  [catat] %d baris ber-APPROVAL di luar '0', '1', '2' (%d lainnya dikenal).",
			unknown, knownStatusRows)
		print("            Baris seperti itu TIDAK muncul di satu pun tab — ada di basis")
		print("            data, tetapi tidak dapat dilihat maupun diputuskan dari layar.")
		print("            Sistem lama punya cacat yang sama dan tidak melaporkannya.")
	}

	if duplicate, err := repo.CountDuplicateName(ctx); err != nil {
		print("  [GAGAL] nama kategori ganda tidak dapat dihitung: %v", err)
	} else if duplicate > 0 {
		print("  [catat] %d nama kategori dipakai lebih dari satu baris.", duplicate)
		print("            Tidak ada constraint unik yang menjaganya (R-08). Akibatnya")
		print("            menyimpan salah satu baris kembar itu akan ditolak dengan")
		print("            \"nama sudah dipakai\" — atas nama baris itu sendiri.")
	}

	if orphan, err := repo.CountOrphanSparepart(ctx); err != nil {
		print("  [GAGAL] sparepart tanpa kategori yang sah tidak dapat dihitung: %v", err)
	} else if orphan > 0 {
		print("  [catat] %d sparepart menunjuk kategori yang tidak ada di tabel ini.", orphan)
		print("            Barisnya tetap terbaca dan tetap dapat disunting; yang tidak")
		print("            tampil hanyalah NAMA kategorinya.")
	}
}

// checkPartType melaporkan kesiapan POOLDATA.GCNM_M_SPAREPART_TYPE.
//
// Tabelnya warisan Pega dan TIDAK dibuat migrasi aplikasi ini, sehingga "belum dapat dibaca"
// di sini berarti tabelnya memang tidak ada di entitas itu, atau akun aplikasi belum diberi
// hak bacanya — keduanya urusan DBA.
//
// # Enam hal dilaporkan, dan dua di antaranya khas modul ini
//
//  1. Jumlah baris per status — apakah ketiga tab layar akan terisi.
//  2. Ketersediaan penomoran, sama seperti Master Kategori Sparepart: `MAX(...)+1` atas
//     tabelnya sendiri, bukan sequence.
//  3. Baris berstatus di luar '0', '1', dan '2' — tidak muncul di satu pun tab.
//  4. Nama yang dipakai lebih dari satu baris. Pencacahnya TIDAK mengelompokkan menurut
//     kategori, meniru cakupan ValidationSparepartType apa adanya.
//  5. **Tipe yang menunjuk kategori yang tidak ada.** Inilah baris yang di sistem lama
//     HILANG dari layar karena inner join-nya, dan yang di modul ini justru TETAP terlihat.
//     Selisih perilaku itu disengaja, dan jumlahnya dilaporkan di sini supaya ia dapat
//     dijelaskan SEBELUM muncul sebagai selisih pada uji kesetaraan gerbang 1.
//  6. **Sparepart yang menunjuk tipe yang tidak ada.** Pemeriksaan dari arah sebaliknya,
//     ada di sini karena modul INILAH yang kelak menolak sebuah tipe — sedangkan penolakan
//     tidak memutuskan tautan yang sudah ada.
func checkPartType(ctx context.Context, primary *sql.DB, print func(string, ...any)) {
	repo := mastertipesparepartsql.NewRepo(primary)

	if err := repo.CheckTable(ctx); err != nil {
		print("  [BELUM] POOLDATA.GCNM_M_SPAREPART_TYPE belum dapat dibaca: %v", err)
		print("            Tabelnya warisan Pega dan TIDAK dibuat migrasi aplikasi ini.")
		print("            Mintakan hak bacanya ke DBA.")
		print("            Tabel yang sama dibaca modul Master Sparepart sebagai daftar")
		print("            acuan Tipe; bila yang ini gagal, dropdown di sana pun kosong.")
		return
	}

	total := 0
	pendingCount := 0
	for _, s := range []mastertipesparepart.ApprovalStatus{
		mastertipesparepart.StatusApproved,
		mastertipesparepart.StatusPending,
		mastertipesparepart.StatusRejected,
	} {
		count, err := repo.CountByStatus(ctx, s)
		if err != nil {
			print("  [GAGAL] GCNM_M_SPAREPART_TYPE status %q tidak dapat dihitung: %v", s, err)
			return
		}
		if s == mastertipesparepart.StatusPending {
			pendingCount = count
		}
		total += count
		print("            status %q %-16s %d baris", s, s.Label(), count)
	}
	print("  [ok]    POOLDATA.GCNM_M_SPAREPART_TYPE dapat dibaca: %d baris berstatus dikenal", total)

	if pendingCount > 0 {
		print("  [catat] %d tipe sparepart menunggu persetujuan.", pendingCount)
	}

	checkPartTypeNumbering(ctx, repo, print)
	checkPartTypeIntegrity(ctx, repo, total, print)
}

// checkPartTypeNumbering melaporkan kesiapan penomoran PART_SECTION_ID.
//
// Ia TIDAK memakai satu nomor urut: penomoran di sini `MAX(...)+1` atas tabelnya sendiri,
// bukan sequence, sehingga membacanya tidak mengubah apa pun. Nomor yang dilaporkan adalah
// nomor yang benar-benar akan terpakai bila ada penambahan saat ini juga.
//
// Kegagalannya hampir selalu berarti satu hal: PART_SECTION_ID ternyata bukan kolom angka.
// Bila itu terjadi, penerbitan kunci tidak dapat dijalankan sama sekali, dan asumsi yang
// dipakai seluruh berkas mastertipesparepart.sql harus ditinjau ulang.
func checkPartTypeNumbering(
	ctx context.Context,
	repo *mastertipesparepartsql.Repo,
	print func(string, ...any),
) {
	id, err := repo.NextID(ctx)
	if err != nil {
		print("  [GAGAL] penomoran ID tipe sparepart tidak dapat dijalankan: %v", err)
		print("            Penambahan tipe baru akan gagal. Penyebab paling mungkin:")
		print("            PART_SECTION_ID bukan kolom angka, sehingga MAX(...)+1 gagal.")
		print("            Bila benar begitu, asumsi mastertipesparepart.sql harus")
		print("            ditinjau ulang — bukan hanya kueri penomorannya.")
		return
	}
	print("  [ok]    penomoran ID tipe sparepart siap; berikutnya: %s", id)
	print("            (angka berurut dari MAX(PART_SECTION_ID)+1, bukan sequence)")
	print("            (tidak ada nomor yang terpakai oleh pemeriksaan ini)")
}

// checkPartTypeIntegrity melaporkan empat keadaan data yang tidak dijaga constraint apa pun,
// dan yang keempatnya baru terlihat sebagai keluhan pengguna bila tidak diperiksa.
//
// Keempatnya BUKAN kegagalan: aplikasi tetap berjalan dengan keempatnya. Yang dilaporkan
// adalah keadaan, supaya ia diketahui sebelum petugas menanyakannya.
func checkPartTypeIntegrity(
	ctx context.Context,
	repo *mastertipesparepartsql.Repo,
	knownStatusRows int,
	print func(string, ...any),
) {
	all, err := repo.CountAll(ctx)
	if err != nil {
		print("  [GAGAL] jumlah seluruh baris tipe tidak dapat dihitung: %v", err)
		return
	}
	if all == 0 {
		print("  [catat] Tabelnya KOSONG. Dropdown Tipe pada layar Master Sparepart karena")
		print("            itu juga kosong, dan sparepart baru tidak dapat ditautkan ke")
		print("            tipe mana pun sampai ada tipe yang disetujui di sini.")
		return
	}

	if unknown, err := repo.CountUnknownStatus(ctx); err != nil {
		print("  [GAGAL] baris berstatus tak dikenal tidak dapat dihitung: %v", err)
	} else if unknown > 0 {
		print("  [catat] %d baris ber-APPROVAL di luar '0', '1', '2' (%d lainnya dikenal).",
			unknown, knownStatusRows)
		print("            Baris seperti itu TIDAK muncul di satu pun tab — ada di basis")
		print("            data, tetapi tidak dapat dilihat maupun diputuskan dari layar.")
		print("            Sistem lama punya cacat yang sama dan tidak melaporkannya.")
	}

	if duplicate, err := repo.CountDuplicateName(ctx); err != nil {
		print("  [GAGAL] nama tipe ganda tidak dapat dihitung: %v", err)
	} else if duplicate > 0 {
		print("  [catat] %d nama tipe dipakai lebih dari satu baris.", duplicate)
		print("            Tidak ada constraint unik yang menjaganya (R-08). Akibatnya")
		print("            menyimpan salah satu baris kembar itu akan ditolak dengan")
		print("            \"nama sudah dipakai\" — atas nama baris itu sendiri.")
		print("            Termasuk baris yang namanya sama di kategori BERBEDA:")
		print("            ValidationSparepartType tidak menyaring kategori sama sekali.")
	}

	if orphan, err := repo.CountOrphanCategory(ctx); err != nil {
		print("  [GAGAL] tipe tanpa kategori yang sah tidak dapat dihitung: %v", err)
	} else if orphan > 0 {
		print("  [catat] %d tipe menunjuk kategori yang tidak ada di master kategori.", orphan)
		print("            SELISIH PERILAKU YANG DISENGAJA: di Pega baris ini HILANG dari")
		print("            layar karena inner join-nya, di sini ia TETAP terlihat dengan")
		print("            kolom Kategori kosong. Angka di atas adalah jumlah baris yang")
		print("            akan tampak berlebih pada uji kesetaraan gerbang 1.")
		print("            Barisnya hanya dapat disimpan ulang setelah kategorinya dipilih.")
	}

	if orphan, err := repo.CountOrphanSparepart(ctx); err != nil {
		print("  [GAGAL] sparepart tanpa tipe yang sah tidak dapat dihitung: %v", err)
	} else if orphan > 0 {
		print("  [catat] %d sparepart menunjuk tipe yang tidak ada di tabel ini.", orphan)
		print("            Barisnya tetap terbaca dan tetap dapat disunting; yang tidak")
		print("            tampil hanyalah NAMA tipenya.")
	}
}

// checkGrouping melaporkan kesiapan kedua tabel Master Grouping Sparepart.
//
// Keduanya warisan Pega dan TIDAK dibuat migrasi aplikasi ini, sehingga "belum dapat dibaca"
// di sini berarti tabelnya memang tidak ada di entitas itu, atau akun aplikasi belum diberi
// hak bacanya — keduanya urusan DBA.
//
// # Lima hal dilaporkan, dan yang KELIMA adalah alasan utama fungsi ini ada
//
//  1. Jumlah baris per status — apakah ketiga tab layar akan terisi.
//
//  2. **Berapa baris induk yang TIDAK punya pendamping.** Kedua sistem menggabungkan tabelnya
//     dengan INNER JOIN, sehingga baris seperti itu tidak muncul di layar mana pun — di Pega
//     maupun di sini. Jumlahnya menjelaskan selisih antara "jumlah baris tabel" dan "jumlah
//     baris yang terlihat" sebelum ada yang mengiranya cacat.
//
//  3. **Berapa baris yang NO_RANGKA kedua tabelnya berbeda.** Sistem lama MENAMPILKAN
//     `B.NO_RANGKA` tetapi MEMERIKSA keunikan atas `A.NO_RANGKA`; selama keduanya sama,
//     perbedaannya tidak terlihat. Rinciannya di banner mastergroupingsparepart.sql.
//
//  4. Ketersediaan penomoran ID dan nomor grup. Keduanya `MAX+1`, bukan sequence, dan
//     keduanya dibaca dari dua tempat sekaligus selama masa paralel.
//
//  5. **Apakah kolom NAMA pada POOLDATA.LOKASI_PANEL_HE berisi nama PANEL atau nama LOKASI.**
//     Modul ini membaca daftar Sisi lewat `id_panel` DAN `nama`, meniru
//     `RDB List/GetDataSisiPanel-SQL.xml` — dan nilai yang dikirimkannya berasal dari
//     autocomplete atas `BrowseMasterPanel_HE_RD`, yakni NAMA PANEL. Modul Master Panel
//     berasumsi sebaliknya dan menulis NAMA := LOKASI_PANEL.
//
//     Bila asumsi Master Panel yang salah, jalur tulisnya akan MEMATIKAN daftar Sisi di layar
//     ini tanpa satu pun pesan galat. Pemeriksaan ini yang menjawabnya dari data nyata alih-
//     alih dari tebakan salah satu pihak; lihat LookupRepo.ListSides.
func checkGrouping(ctx context.Context, primary *sql.DB, print func(string, ...any)) {
	repo := mastergroupingsparepartsql.NewRepo(primary)

	if err := repo.CheckTable(ctx); err != nil {
		print("  [BELUM] POOLDATA.SPAREPART_HE_VIN_KEY belum dapat dibaca: %v", err)
		print("            Tabelnya warisan Pega dan TIDAK dibuat migrasi aplikasi ini.")
		print("            Mintakan hak bacanya ke DBA.")
		return
	}
	if err := repo.CheckGroupTable(ctx); err != nil {
		print("  [BELUM] POOLDATA.SPAREPART_HE_VIN_GROUP belum dapat dibaca: %v", err)
		print("            Tanpa tabel pendamping ini, layar Master Grouping Sparepart")
		print("            TIDAK akan menampilkan satu baris pun — kedua sistem")
		print("            menggabungkannya dengan INNER JOIN.")
		return
	}

	total := 0
	pendingCount := 0
	for _, s := range []mastergroupingsparepart.ApprovalStatus{
		mastergroupingsparepart.StatusApproved,
		mastergroupingsparepart.StatusPending,
		mastergroupingsparepart.StatusRejected,
	} {
		count, err := repo.CountByStatus(ctx, s)
		if err != nil {
			print("  [GAGAL] SPAREPART_HE_VIN_KEY status %q tidak dapat dihitung: %v", s, err)
			return
		}
		if s == mastergroupingsparepart.StatusPending {
			pendingCount = count
		}
		total += count
		print("            status %q %-16s %d baris", s, s.Label(), count)
	}
	print("  [ok]    POOLDATA.SPAREPART_HE_VIN_KEY dapat dibaca: %d baris terlihat di layar",
		total)

	if pendingCount > 0 {
		print("  [catat] %d grouping menunggu persetujuan.", pendingCount)
	}

	checkGroupingCompanion(ctx, repo, print)
	checkGroupingNumbering(ctx, repo, print)
	checkGroupingLookup(ctx, repo, print)
	checkGroupingPanelNameColumn(ctx, repo, print)
}

// checkGroupingCompanion melaporkan hubungan kedua tabel.
//
// Dua angka, dan keduanya menjelaskan hal yang sama dari sisi berbeda: berapa banyak baris
// yang TIDAK terlihat, dan berapa banyak yang terlihat dengan nomor rangka yang berbeda dari
// yang diperiksa keunikannya.
func checkGroupingCompanion(
	ctx context.Context,
	repo *mastergroupingsparepartsql.Repo,
	print func(string, ...any),
) {
	all, err := repo.CountAll(ctx)
	if err != nil {
		print("  [GAGAL] jumlah seluruh baris induk tidak dapat dihitung: %v", err)
		return
	}

	orphan, err := repo.CountWithoutGroup(ctx)
	if err != nil {
		print("  [GAGAL] baris tanpa pendamping tidak dapat dihitung: %v", err)
		return
	}

	switch {
	case orphan == 0:
		print("  [ok]    seluruh %d baris induk punya pendamping di _VIN_GROUP", all)
	default:
		print("  [WASPADA] %d dari %d baris induk TIDAK punya pendamping", orphan, all)
		print("            Seluruhnya tidak muncul di layar, baik di Pega maupun di sini:")
		print("            kedua sistem menggabungkannya dengan INNER JOIN. Angka ini yang")
		print("            menjelaskan selisihnya sebelum ada yang mengiranya cacat.")
	}

	mismatch, err := repo.CountChassisMismatch(ctx)
	if err != nil {
		print("  [catat] selisih NO_RANGKA antartabel tidak dapat dihitung: %v", err)
		return
	}
	if mismatch == 0 {
		print("  [ok]    NO_RANGKA sama pada kedua tabel di seluruh baris yang terlihat")
		return
	}

	print("  [WASPADA] %d baris punya NO_RANGKA yang BERBEDA antara kedua tabelnya", mismatch)
	print("            Sistem lama MENAMPILKAN nomor rangka tabel pendamping tetapi")
	print("            MEMERIKSA keunikan atas nomor rangka tabel induk. Pada baris ini")
	print("            keduanya tidak sepakat, sehingga baris yang tampil sebagai duplikat")
	print("            dapat lolos pemeriksaan — dan sebaliknya.")
	print("            Penyimpanan modul ini menulis keduanya dengan nilai yang sama,")
	print("            sehingga selisih baru tidak dapat lahir; yang lama tidak diperbaiki.")
}

// checkGroupingNumbering melaporkan kesiapan penomoran ID dan nomor grup.
//
// Ia benar-benar MENGAMBIL satu nomor, dan itu disengaja meski mode periksa tidak menulis apa
// pun: keduanya `MAX+1` yang tidak menyisakan jejak, sehingga memanggilnya TIDAK memakai satu
// nomor pun — berbeda dari sequence pada ketiga master alat berat lain.
func checkGroupingNumbering(
	ctx context.Context,
	repo *mastergroupingsparepartsql.Repo,
	print func(string, ...any),
) {
	id, err := repo.NextID(ctx)
	if err != nil {
		print("  [GAGAL] ID grouping berikutnya tidak dapat diterbitkan: %v", err)
		print("            Penambahan akan gagal pada permintaan pertama.")
	} else {
		print("  [ok]    ID grouping berikutnya: %s", id)
		print("            Bentuknya angka polos, TANPA kode situs dan tanpa pengisian nol —")
		print("            meniru PEGA_M_GROUPING_SPAREPART_HE.prc:11, yang berbeda dari")
		print("            ketiga master alat berat lain.")
	}

	group, err := repo.NextGroupNumber(ctx)
	if err != nil {
		print("  [GAGAL] nomor grup berikutnya tidak dapat diterbitkan: %v", err)
	} else {
		print("  [ok]    nomor grup berikutnya: %s", group)
	}

	if broken, err := repo.CountUnreadableGroupNumber(ctx); err != nil {
		print("  [catat] nomor grup yang tidak terbaca tidak dapat dihitung: %v", err)
	} else if broken > 0 {
		print("  [catat] %d nomor grup tidak dapat dibaca sebagai angka.", broken)
		print("            Nilainya dilewati saat menerbitkan nomor baru, sehingga ia tidak")
		print("            menghentikan apa pun — tetapi baris yang memakainya tidak akan")
		print("            pernah dapat diikuti sebagai grup.")
	}

	// Penyimpanan JSON milik Pega. Perbandingan jumlah barisnya adalah cara termurah
	// mengetahui apakah keduanya satu sumber; lihat banner mastergroupingsparepart.sql.
	if err := repo.CheckJSONMirror(ctx); err != nil {
		print("  [catat] POOLDATA.M_SPAREPART_HE_VIN_KEY belum dapat dibaca: %v", err)
		print("            Penomoran modul ini tetap berjalan, tetapi ia tidak lagi dapat")
		print("            menghindari ID yang sedang diterbitkan Pega selama masa paralel.")
		return
	}

	mirror, err := repo.CountJSONMirror(ctx)
	if err != nil {
		print("  [catat] jumlah baris penyimpanan JSON tidak dapat dihitung: %v", err)
		return
	}
	live, err := repo.CountAll(ctx)
	if err != nil {
		return
	}

	switch {
	case mirror == live:
		print("  [ok]    SPAREPART_HE_VIN_KEY dan M_SPAREPART_HE_VIN_KEY sama-sama %d baris",
			live)
	default:
		print("  [WASPADA] SPAREPART_HE_VIN_KEY %d baris, M_SPAREPART_HE_VIN_KEY %d baris",
			live, mirror)
		print("            Keduanya TIDAK satu sumber. Modul ini menulis kolom bernama ke")
		print("            yang pertama dan berhenti menulis JSONDATA ke yang kedua (D-02,")
		print("            D-68); bila Pega masih membaca yang kedua, tulisan di sini tidak")
		print("            akan pernah sampai ke sana. Angkanya perlu dibawa ke DBA.")
	}
}

// checkGroupingLookup melaporkan keempat sumber acuan layar ini.
func checkGroupingLookup(
	ctx context.Context,
	repo *mastergroupingsparepartsql.Repo,
	print func(string, ...any),
) {
	if err := repo.CheckVehicleTypeTable(ctx); err != nil {
		print("  [catat] tabel branddetail belum dapat dibaca: %v", err)
		print("            Isian Tipe Kendaraan akan tampil tanpa pilihan. Perhatikan bahwa")
		print("            namanya memang DISEBUT TANPA SKEMA, mengikuti kueri aslinya —")
		print("            yang terpakai adalah skema bawaan akun koneksi.")
	} else if vehicle, err := repo.ListVehicleTypes(ctx); err != nil {
		print("  [catat] daftar tipe kendaraan tidak dapat dibaca: %v", err)
	} else {
		print("  [ok]    %d tipe kendaraan aktif pada lini ANEKA", len(vehicle))
	}

	if panel, err := repo.ListPanels(ctx); err != nil {
		print("  [catat] daftar panel tidak dapat dibaca: %v", err)
	} else {
		print("  [ok]    %d panel disetujui dan dapat dipilih", len(panel))
		if len(panel) == 0 {
			print("            Tanpa satu pun panel disetujui, isian Nama Panel tidak dapat")
			print("            diisi sama sekali — dan Nama Panel adalah isian WAJIB.")
		}
	}

	if orphan, err := repo.CountOrphanPart(ctx); err != nil {
		print("  [catat] grouping tanpa sparepart yang sah tidak dapat dihitung: %v", err)
	} else if orphan > 0 {
		print("  [catat] %d grouping menunjuk nomor sparepart yang tidak ada di", orphan)
		print("            POOLDATA.SPAREPART_HE. Barisnya tetap TERBACA, tetapi TIDAK DAPAT")
		print("            DISIMPAN ULANG tanpa lebih dulu memperbaiki nomornya — modul ini")
		print("            menolak nomor yang tidak ketemu, meniru SetDataSparepart.")
	}

	if orphan, err := repo.CountOrphanPanel(ctx); err != nil {
		print("  [catat] grouping tanpa panel yang sah tidak dapat dihitung: %v", err)
	} else if orphan > 0 {
		print("  [catat] %d grouping menunjuk nama panel yang tidak ada di", orphan)
		print("            POOLDATA.PANEL_HE. Barisnya tetap terbaca; menyimpannya ulang")
		print("            menuntut memilih panel dari daftar.")
	}
}

// checkGroupingPanelNameColumn menjawab pertanyaan yang menentukan apakah daftar Sisi terisi.
//
// Kolom `NAMA` pada POOLDATA.LOKASI_PANEL_HE tidak pernah muncul sebagai kolom yang DIBACA di
// seluruh export — hanya sebagai penyaring pada `RDB List/GetDataSisiPanel-SQL.xml`. Dua
// pembacaan sama-sama masuk akal, dan keduanya tidak dapat benar bersamaan:
//
//	(a) NAMA berisi nama LOKASI      -> asumsi modul Master Panel, yang menulis NAMA := LOKASI_PANEL
//	(b) NAMA berisi nama PANEL       -> yang dituntut modul ini, karena nilainya berasal dari
//	                                    autocomplete atas BrowseMasterPanel_HE_RD
//
// Yang dilaporkan di sini adalah PERBANDINGAN KEDUANYA terhadap data nyata. Modul Master
// Panel tidak disunting dari sini; yang dikerjakan adalah menyediakan angkanya supaya
// keputusannya diambil atas bukti.
func checkGroupingPanelNameColumn(
	ctx context.Context,
	repo *mastergroupingsparepartsql.Repo,
	print func(string, ...any),
) {
	if err := repo.CheckChildNameColumn(ctx); err != nil {
		print("  [BELUM] kolom NAMA pada POOLDATA.LOKASI_PANEL_HE belum dapat dibaca: %v", err)
		print("            Tanpa kolom itu, daftar Sisi TIDAK akan pernah terisi — dan Sisi")
		print("            adalah isian WAJIB pada layar ini.")
		return
	}

	rows, err := repo.CountChildRows(ctx)
	if err != nil {
		print("  [catat] jumlah baris lokasi panel tidak dapat dihitung: %v", err)
		return
	}
	if rows == 0 {
		print("  [WASPADA] POOLDATA.LOKASI_PANEL_HE kosong")
		print("            Daftar Sisi tidak akan pernah terisi, sehingga tidak satu pun")
		print("            grouping dapat ditambahkan. Isi Master Panel lebih dulu.")
		return
	}

	asPanel, err := repo.CountChildNameAsPanel(ctx)
	if err != nil {
		print("  [catat] NAMA tidak dapat dibandingkan dengan nama panel: %v", err)
		return
	}
	asLocation, err := repo.CountChildNameAsLocation(ctx)
	if err != nil {
		print("  [catat] NAMA tidak dapat dibandingkan dengan LOKASI_PANEL: %v", err)
		return
	}

	print("            NAMA = nama PANEL   : %d dari %d baris", asPanel, rows)
	print("            NAMA = LOKASI_PANEL : %d dari %d baris", asLocation, rows)

	switch {
	case asPanel >= rows && asLocation < rows:
		print("  [ok]    NAMA berisi NAMA PANEL. Daftar Sisi layar ini akan terisi.")
		print("            KONSEKUENSI UNTUK MASTER PANEL: jalur tulisnya mengisi NAMA")
		print("            dengan nama LOKASI, dan itu akan MEMATIKAN daftar Sisi di sini")
		print("            tanpa satu pun pesan galat. Bawa temuan ini ke Work Owner")
		print("            sebelum jalur tulis Master Panel dinyalakan di produksi.")
	case asLocation >= rows && asPanel < rows:
		print("  [ok]    NAMA berisi NAMA LOKASI. Asumsi Master Panel TERBUKTI.")
		print("            KONSEKUENSI UNTUK MODUL INI: daftar Sisi TIDAK akan terisi,")
		print("            karena nilai yang dikirimkannya adalah nama PANEL. Kueri")
		print("            grouping_side_list harus ditinjau sebelum layar dipakai.")
	case asPanel >= rows && asLocation >= rows:
		print("  [ok]    Keduanya sama — nama panel dan nama lokasinya memang sepadan pada")
		print("            seluruh baris, sehingga kedua pembacaan menghasilkan hal yang")
		print("            sama. Tidak ada yang perlu diputuskan hari ini.")
	default:
		print("  [WASPADA] Tidak satu pun pembacaan cocok pada SELURUH baris.")
		print("            Kolom NAMA berisi sesuatu yang bukan keduanya, setidaknya pada")
		print("            sebagian baris. Mintakan DDL dan contoh isi kedua kolom ke DBA")
		print("            (R-08) sebelum layar Master Grouping Sparepart dipakai.")
	}
}

// checkSurveyorLogin melaporkan kesiapan POOLDATA.MST_LOGIN_SURVEYOR.
//
// Tabelnya warisan Pega dan TIDAK dibuat migrasi aplikasi ini, sehingga "belum dapat dibaca"
// di sini berarti tabelnya memang tidak ada di entitas itu, atau akun aplikasi belum diberi
// hak bacanya — keduanya urusan DBA.
//
// # Empat keadaan dilaporkan, dan ketiganya khas modul ini
//
//  1. Jumlah baris. TANPA rincian per status: tabelnya tidak punya kolom APPROVAL, dan
//     layar lamanya tidak bertab — berbeda dari seluruh master di rumpun sparepart.
//  2. **LOGIN yang dipakai lebih dari satu baris, dan LOGIN yang kosong.** Keduanya
//     berakibat LANGSUNG, bukan sekadar merepotkan: setiap pernyataan simpan menyaring
//     `where login = ...`, sehingga satu penyimpanan mengubah SELURUH baris berlogin sama
//     sekaligus. Tidak ada constraint unik yang menjaganya (R-08), dan sistem lama pun
//     tidak punya — pemeriksaan gandanya bahkan menembak tabel operator Pega, bukan tabel
//     ini.
//  3. **LOGINLEADER yang menunjuk login yang tidak ada.** Ia belum berakibat apa pun hari
//     ini; ia baru berarti bila cakupan daftar kelak diputuskan disaring per tim. Lihat
//     masterlogin.Filter.
//  4. **EMAIL atau TELP yang kosong.** Keduanya WAJIB di layar Pega tetapi tidak pernah
//     ditegakkan di server sana, sehingga baris tanpa surel benar-benar mungkin ada. Modul
//     ini MENOLAKNYA saat disimpan ulang — dan jumlahnya perlu diketahui sebelum petugas
//     menemukan bahwa baris yang selama ini tersimpan tidak lagi dapat disimpan.
func checkSurveyorLogin(ctx context.Context, primary *sql.DB, print func(string, ...any)) {
	repo := masterloginsql.NewRepo(primary)

	if err := repo.CheckTable(ctx); err != nil {
		print("  [BELUM] POOLDATA.MST_LOGIN_SURVEYOR belum dapat dibaca: %v", err)
		print("            Tabelnya warisan Pega dan TIDAK dibuat migrasi aplikasi ini.")
		print("            Mintakan hak baca dan tulisnya ke DBA.")
		print("            Ketujuh kolom yang dipakai: NAMA, LOGIN, EMAIL, TELP, ALAMAT,")
		print("            STSLOGIN, LOGINLEADER.")
		return
	}

	total, err := repo.CountAll(ctx)
	if err != nil {
		print("  [GAGAL] MST_LOGIN_SURVEYOR tidak dapat dihitung: %v", err)
		return
	}
	print("  [ok]    POOLDATA.MST_LOGIN_SURVEYOR dapat dibaca: %d baris", total)
	print("            (tanpa rincian status — tabelnya tidak punya kolom APPROVAL)")

	checkSurveyorLoginKey(ctx, repo, print)
	checkSurveyorLoginIntegrity(ctx, repo, print)
}

// checkSurveyorLoginKey melaporkan keadaan KUNCI tabel — dan inilah pemeriksaan terpenting
// di modul ini.
//
// LOGIN adalah satu-satunya kunci, tidak dijaga constraint apa pun (R-08), dan dipakai
// sebagai penyaring pada SETIAP pernyataan simpan. Dua keadaan merusaknya, dan keduanya
// merusak dalam diam:
//
//	kembar  satu penyimpanan mengubah lebih dari satu orang sekaligus
//	kosong  barisnya tidak dapat dibuka maupun disunting dari layar mana pun
//
// Keduanya BUKAN kegagalan aplikasi: layar tetap berjalan. Yang dilaporkan adalah keadaan,
// supaya ia diketahui sebelum perubahan mengenai orang yang salah.
func checkSurveyorLoginKey(
	ctx context.Context,
	repo *masterloginsql.Repo,
	print func(string, ...any),
) {
	duplicate, errOne := repo.CountDuplicateKey(ctx)
	empty, errTwo := repo.CountEmptyKey(ctx)
	if errOne != nil || errTwo != nil {
		print("  [catat] keadaan kunci LOGIN tidak dapat diperiksa")
		return
	}

	if duplicate == 0 && empty == 0 {
		print("  [ok]    seluruh LOGIN terisi dan tidak ada yang kembar")
		return
	}

	if duplicate > 0 {
		print("  [PENTING] %d LOGIN dipakai lebih dari satu baris.", duplicate)
		print("            Setiap penyimpanan menyaring `where login = ...`, sehingga satu")
		print("            perubahan mengenai SELURUH baris berlogin sama sekaligus — dan")
		print("            layar hanya menampilkan salah satunya tanpa menyebut ada yang")
		print("            lain. Mintakan pembersihannya ke DBA sebelum jalur tulis dipakai.")
	}
	if empty > 0 {
		print("  [PENTING] %d baris LOGIN-nya kosong.", empty)
		print("            Baris seperti itu tidak dapat dibuka maupun disunting dari layar:")
		print("            kuncinya kosong. Ia terbaca di daftar, dan tombol Ubah-nya tidak")
		print("            akan menemukan barisnya.")
	}
}

// checkSurveyorLoginIntegrity melaporkan dua keadaan data yang tidak dijaga apa pun.
//
// Keduanya BUKAN kegagalan — aplikasi tetap berjalan dengan keduanya. Yang dilaporkan
// adalah keadaan, supaya ia diketahui sebelum petugas menanyakannya.
func checkSurveyorLoginIntegrity(
	ctx context.Context,
	repo *masterloginsql.Repo,
	print func(string, ...any),
) {
	orphan, errOne := repo.CountOrphanLeader(ctx)
	missing, errTwo := repo.CountMissingContact(ctx)
	if errOne != nil || errTwo != nil {
		print("  [catat] keutuhan LOGINLEADER dan kelengkapan kontak tidak dapat diperiksa")
		return
	}

	if orphan > 0 {
		print("  [catat] %d baris LOGINLEADER-nya menunjuk login yang tidak ada.", orphan)
		print("            Belum berakibat apa pun hari ini: daftarnya memuat seluruh baris.")
		print("            Ia baru berarti bila cakupan daftar kelak disaring per tim —")
		print("            pertanyaan terbuka pada masterlogin.Filter.")
	}

	if missing > 0 {
		print("  [catat] %d baris EMAIL atau TELP-nya kosong.", missing)
		print("            Keduanya WAJIB di layar Pega, tetapi kewajiban itu tidak pernah")
		print("            ditegakkan di server sana. Modul ini MENOLAKNYA saat barisnya")
		print("            disimpan ulang, sehingga petugas yang membuka baris seperti itu")
		print("            harus melengkapinya lebih dulu. Baris yang tidak disentuh tetap")
		print("            terbaca apa adanya.")
	} else {
		print("  [ok]    seluruh baris punya EMAIL dan TELP")
	}
}

// checkInvestigatorInbox melaporkan kesiapan antrean Inbox Investigator.
//
// # Kenapa pemeriksaan ini berbeda dari pemeriksaan modul master
//
// Ia menyentuh EMPAT tabel di DUA skema sekaligus — DATAPEGA untuk header pekerjaan dan
// penugasannya, POOLDATA untuk objek pertanggungan dan hasil survei. Hak baca yang kurang
// pada salah satu skema saja tidak menghasilkan galat yang terbaca petugas: layarnya gagal
// memuat, dan sebabnya hanya terlihat di log server.
//
// Keempatnya warisan Pega dan TIDAK dibuat migrasi aplikasi ini, sehingga "belum dapat
// dibaca" berarti tabelnya tidak ada di entitas itu, atau akun aplikasi belum diberi hak
// bacanya — keduanya urusan DBA.
//
// # Hak BACA saja yang dibutuhkan
//
// Modul ini tidak menulis satu baris pun; mengambil pekerjaan dari antrean dan mencatat
// hasil investigasi terjadi di layar kerja yang belum dibangun. Selama itu benar, `P-1`
// terpenuhi tanpa negosiasi kepemilikan: Pega tetap satu-satunya penulis tabelnya sendiri.
//
// # Dua angka dilaporkan, dan yang kedua menjawab pertanyaan yang layar tidak dapat jawab
//
//  1. **Berapa pekerjaan yang menunggu.** Layar memotong pada 500 baris; angka ini tidak.
//     Ia satu-satunya cara mengetahui seberapa jauh antreannya melampaui batas itu.
//  2. **Berapa yang tidak punya baris survei.** Tanggal survei mengisi kolom KESEMBILAN
//     grid — yang captionnya di layar lama berbunyi "Lama Masuk Inbox" meski isinya tanggal.
//     Pekerjaan tanpa baris survei karena itu tampil dengan sel kosong di kolom itu. Bila
//     angkanya besar, kolom itu kosong bagi sebagian besar antrean — keadaan yang hanya
//     dapat diketahui dari data produksi, bukan dari export.
func checkInvestigatorInbox(ctx context.Context, primary *sql.DB, print func(string, ...any)) {
	repo := inboxinvestigatorsql.NewRepo(primary)

	if err := repo.CheckTable(ctx); err != nil {
		print("  [BELUM] antrean Inbox Investigator belum dapat dibaca: %v", err)
		print("            Empat tabel warisan Pega, di DUA skema:")
		print("              DATAPEGA.PC_ASM_FW_GCNMFW_WORK   header pekerjaan")
		print("              DATAPEGA.PC_ASSIGN_WORKBASKET    antrean bersama")
		print("              POOLDATA.T_CLAIM_OBJECTLIST      nama peserta")
		print("              POOLDATA.T_SURVEYORLIST          tanggal survei")
		print("            Mintakan hak BACA keempatnya ke DBA — modul ini tidak menulis.")
		return
	}

	waiting, err := repo.CountWaiting(ctx)
	if err != nil {
		print("  [GAGAL] antrean Inbox Investigator tidak dapat dihitung: %v", err)
		return
	}
	print("  [ok]    antrean Inbox Investigator dapat dibaca: %d pekerjaan menunggu",
		waiting)
	print("            (workbasket %q, tanpa yang berstatus Resolved-Completed)",
		inboxinvestigator.Workbasket)

	if waiting > inboxinvestigator.MaxRows {
		print("  [catat] antrean MELAMPAUI batas %d baris yang dikirim ke layar",
			inboxinvestigator.MaxRows)
		print("            Layar menyatakan dirinya terpotong, tidak memotong diam-diam")
		print("            seperti pyMaxRecords=500 pada sistem lama. Tetapi %d pekerjaan",
			waiting-inboxinvestigator.MaxRows)
		print("            tetap tidak terlihat tanpa memakai penyaring pencarian.")
	}

	checkInvestigatorInboxSurvey(ctx, repo, waiting, print)
}

// checkInvestigatorInboxSurvey melaporkan pekerjaan yang tidak punya baris survei.
//
// Lihat butir 2 pada checkInvestigatorInbox untuk alasan angka ini layak dilaporkan.
func checkInvestigatorInboxSurvey(
	ctx context.Context,
	repo *inboxinvestigatorsql.Repo,
	waiting int,
	print func(string, ...any),
) {
	if waiting == 0 {
		return
	}

	without, err := repo.CountWithoutSurvey(ctx)
	if err != nil {
		print("  [catat] pekerjaan tanpa baris survei tidak dapat diperiksa")
		return
	}

	if without == 0 {
		print("  [ok]    seluruh pekerjaan punya tanggal survei")
		return
	}

	print("  [catat] %d dari %d pekerjaan TIDAK punya tanggal survei", without, waiting)
	print("            Kolom kesembilan pada baris itu tampil KOSONG. Captionnya di layar")
	print("            lama berbunyi \"Lama Masuk Inbox\" meski isinya tanggal survei —")
	print("            ketidakcocokan itu dibawa apa adanya dari Pega (P-5).")
}

// checkReceiveTKAInbox melaporkan kesiapan sumber daftar Inbox Receive TKA.
//
// # Sumbernya sama dengan Report Definition Pega
//
// `DATAPEGA.PC_ASM_FW_GCNMFW_WORK`, disaring `TKA_1 = '1'` — kolom yang sama yang dipakai
// layar lama. Angka "pekerjaan menunggu" di bawah karena itu dapat dibandingkan LANGSUNG
// dengan jumlah baris pada layar Pega; selisihnya berarti ada yang perlu ditelusuri.
//
// # Modul ini MENULIS, dan hanya satu kolom
//
// `POOLDATA.T_CLAIM_PNC.TGLDOKLENGKAP`. Tabel engine Pega tidak pernah disentuh, karena
// nilai yang ditulis ke sana akan tertimpa dari BLOB kasus tanpa satu pun tanda.
func checkReceiveTKAInbox(ctx context.Context, primary *sql.DB, print func(string, ...any)) {
	repo := inboxreceivetkasql.NewRepo(primary)

	if err := repo.CheckTable(ctx); err != nil {
		print("  [BELUM] daftar Inbox Receive TKA belum dapat dibaca: %v", err)
		print("            Tiga tabel, di DUA skema:")
		print("              DATAPEGA.PC_ASM_FW_GCNMFW_WORK   antrean klaim TKA")
		print("              POOLDATA.T_CLAIM_PNC             klaim sebenarnya")
		print("              POOLDATA.T_GENERAL               nama peserta, dari polis")
		print("            Mintakan hak BACA ketiganya ke DBA.")
		return
	}

	if err := repo.CheckClaimColumn(ctx); err != nil {
		print("  [BELUM] kolom POOLDATA.T_CLAIM_PNC.TGLDOKLENGKAP tidak dapat dibaca: %v", err)
	}
	print("  [catat] Submit modul ini MENULIS satu kolom:")
	print("              POOLDATA.T_CLAIM_PNC.TGLDOKLENGKAP")
	print("            Tabel engine Pega sengaja TIDAK ditulis — nilainya akan tertimpa dari")
	print("            BLOB kasus. Kolom itu hari ini dimiliki Pega (P-1); serah-terima")
	print("            kepemilikan tulisnya menempuh D-63. Hak UPDATE tidak diperiksa di sini.")

	waiting, err := repo.CountWaiting(ctx)
	if err != nil {
		print("  [GAGAL] daftar Inbox Receive TKA tidak dapat dihitung: %v", err)
		return
	}
	print("  [ok]    daftar Inbox Receive TKA dapat dibaca: %d pekerjaan menunggu", waiting)
	print("            (TKA_1='1', tanggal dokumen belum diisi, belum Resolved-Completed)")
	print("            Angka ini SEHARUSNYA sama dengan jumlah baris pada layar TKA di Pega.")

	if waiting > inboxreceivetka.MaxRows {
		print("  [catat] daftar MELAMPAUI batas %d baris yang dikirim ke layar",
			inboxreceivetka.MaxRows)
		print("            Layar menyatakan dirinya terpotong, tidak memotong diam-diam")
		print("            seperti pyMaxRecords=500 pada sistem lama. Tetapi %d pekerjaan",
			waiting-inboxreceivetka.MaxRows)
		print("            tetap tidak terlihat tanpa memakai penyaring pencarian.")
	}

	checkReceiveTKADivergence(ctx, repo, waiting, print)
	checkReceiveTKARegisterDate(ctx, repo, print)
}

// checkReceiveTKADivergence melaporkan ketiga selisih yang dapat terjadi antara layar ini
// dan layar Pega.
func checkReceiveTKADivergence(
	ctx context.Context,
	repo *inboxreceivetkasql.Repo,
	waiting int,
	print func(string, ...any),
) {
	if pegaOnly, err := repo.CountPegaOnly(ctx); err == nil && pegaOnly > 0 {
		print("  [catat] %d pekerjaan sudah diisi lewat aplikasi ini tetapi MASIH tampil di Pega",
			pegaOnly)
		print("            Itu harga dari tidak menulis ke tabel engine Pega, dan memang")
		print("            disengaja. Bila menumpuk, petugas Pega akan mengisi ulang tanggal")
		print("            yang sebenarnya sudah diisi — angkanya perlu dipantau.")
	}

	if orphan, err := repo.CountOrphanClaim(ctx); err == nil && orphan > 0 {
		print("  [catat] %d pekerjaan klaimnya TIDAK ADA di POOLDATA.T_CLAIM_PNC", orphan)
		print("            Baris itu TETAP TAMPIL — gabungannya sengaja LEFT, sama seperti")
		print("            Pega — tetapi Submit atasnya ditolak, dan layar mematikan isiannya")
		print("            lebih dulu. Yang perlu ditinjau kelengkapan T_CLAIM_PNC.")
	}

	if waiting == 0 {
		return
	}

	if missing, err := repo.CountMissingParticipant(ctx); err == nil && missing > 0 {
		print("  [catat] %d dari %d pekerjaan nama pesertanya tidak ditemukan di tabel polis",
			missing, waiting)
		print("            `.Policy.TheInsured` tidak di-expose sebagai kolom pada tabel kerja")
		print("            Pega, sehingga nilainya diambil dari POOLDATA.T_GENERAL lewat")
		print("            NOPOLIS + PRODKE. Bila angkanya besar, penggantinya keliru.")
	}
}

// checkReceiveTKARegisterDate memastikan format kolom tanggal registrasi seragam.
//
// Kolomnya `VARCHAR2(32)`, bukan tanggal, dan isinya terverifikasi berformat `yyyymmdd`
// pada DUA baris saja. Kueri ini memperlihatkan apakah bentuk itu berlaku pada seluruh
// daftar — bentuk yang menyimpang diurai menjadi kosong, dan kolom Aging pada baris itu
// tampil sebagai tanda hubung.
func checkReceiveTKARegisterDate(
	ctx context.Context,
	repo *inboxreceivetkasql.Repo,
	print func(string, ...any),
) {
	sample, err := repo.SampleRegisteredOn(ctx)
	if err != nil || len(sample) == 0 {
		return
	}

	var odd []string
	for _, one := range sample {
		if len(one) < 8 {
			odd = append(odd, one)
			continue
		}
		if _, parseErr := time.Parse("20060102", one[:8]); parseErr != nil {
			odd = append(odd, one)
		}
	}

	if len(odd) == 0 {
		print("  [ok]    kolom REGISTERDATE_1 berformat yyyymmdd pada seluruh %d contoh",
			len(sample))
		return
	}

	print("  [catat] kolom REGISTERDATE_1: %d dari %d contoh bentuknya TIDAK dikenali",
		len(odd), len(sample))
	print("            Contohnya: %s", strings.Join(odd, " | "))
	print("            Baris seperti itu tampil dengan kolom Aging bertanda hubung, bukan")
	print("            dengan lama menunggu yang salah.")
}

// checkReinsurerMember melaporkan kesiapan POOLDATA.T_REINSURER.
//
// Tabelnya warisan Pega dan TIDAK dibuat migrasi aplikasi ini, sehingga "belum dapat dibaca"
// di sini berarti tabelnya memang tidak ada di entitas itu, atau akun aplikasi belum diberi
// hak bacanya — keduanya urusan DBA.
//
// # Hak BACA saja yang dibutuhkan, dan itu perlu disebut ke DBA
//
// Modul Master Reas tidak menulis satu baris pun; lihat banner paket masterreas. Meminta hak
// tulis untuk tabel ini berarti meminta kewenangan yang tidak dipakai — dan pada tabel yang
// menentukan ke mana pemberitahuan klaim dikirim, kewenangan yang menganggur adalah risiko
// yang tidak berimbalan.
//
// # Empat keadaan dilaporkan, dan dua di antaranya khas tabel ini
//
//  1. Jumlah baris. TANPA rincian per status: tabelnya tidak punya kolom persetujuan, dan
//     layar lamanya tidak bertab.
//  2. **Kunci alami yang kembar** — REINSURERID + REINSURERNAME + TYPE. Ia kunci yang
//     dipakai `UPDATEREAS` untuk memutuskan menyisipkan atau memperbarui, dan tidak dijaga
//     constraint apa pun yang diketahui (`R-08`).
//  3. **LOGIN yang dipakai lebih dari satu kode reas, dan LOGIN yang kosong.** Inilah
//     pemeriksaan bertaruh paling tinggi di modul ini; lihat checkReinsurerMemberLogin.
//  4. **EMAIL kosong, dan perusahaan tanpa baris cadangan.** Keduanya berakibat sama:
//     pemberitahuan PLA/DLA yang tidak pernah sampai, tanpa satu pun galat.
func checkReinsurerMember(ctx context.Context, primary *sql.DB, print func(string, ...any)) {
	repo := masterreassql.NewRepo(primary)

	if err := repo.CheckTable(ctx); err != nil {
		print("  [BELUM] POOLDATA.T_REINSURER belum dapat dibaca: %v", err)
		print("            Tabelnya warisan Pega dan TIDAK dibuat migrasi aplikasi ini.")
		print("            Mintakan hak BACA-nya ke DBA — modul Master Reas tidak menulis.")
		print("            Keenam kolom yang dipakai: REINSURERID, REINSURERNAME, LOGIN,")
		print("            EMAIL, COUNTRY, TYPE. COUNTRYID sengaja tidak dibaca.")
		return
	}

	total, err := repo.CountAll(ctx)
	if err != nil {
		print("  [GAGAL] T_REINSURER tidak dapat dihitung: %v", err)
		return
	}
	print("  [ok]    POOLDATA.T_REINSURER dapat dibaca: %d baris", total)
	print("            (tanpa rincian status — tabelnya tidak punya kolom persetujuan)")

	checkReinsurerMemberKey(ctx, repo, print)
	checkReinsurerMemberLogin(ctx, repo, print)
	checkReinsurerMemberDelivery(ctx, repo, print)
}

// checkReinsurerMemberKey melaporkan kunci alami yang kembar.
//
// Kuncinya TIGA kolom — REINSURERID + REINSURERNAME + TYPE — dibaca dari
// `Database/UPDATEREAS.prc`, yang memeriksa keberadaan baris dengan ketiganya sekaligus.
//
// Baris kembar berakibat pada KEDUA arah: `UPDATEREAS` memperbarui seluruhnya sekaligus
// karena UPDATE-nya tidak membatasi jumlah baris, sementara pembacaan PLA/DLA memakai
// `fetch next 1 row only` dan hanya melihat salah satunya. Surel yang dipakai mengirim
// dokumen karena itu belum tentu surel yang terakhir diubah petugas.
func checkReinsurerMemberKey(
	ctx context.Context,
	repo *masterreassql.Repo,
	print func(string, ...any),
) {
	duplicate, err := repo.CountDuplicateKey(ctx)
	if err != nil {
		print("  [catat] keadaan kunci alami tidak dapat diperiksa")
		return
	}

	if duplicate == 0 {
		print("  [ok]    tidak ada kunci alami yang kembar (REINSURERID + NAMA + TYPE)")
		return
	}

	print("  [PENTING] %d kunci alami dipakai lebih dari satu baris.", duplicate)
	print("            UPDATEREAS memperbarui SELURUHNYA sekaligus, sementara pembacaan")
	print("            PLA/DLA memakai `fetch next 1 row only` dan hanya melihat salah")
	print("            satunya. Surel yang dipakai mengirim dokumen karena itu belum tentu")
	print("            surel yang terakhir diubah petugas. Mintakan pembersihannya ke DBA.")
}

// checkReinsurerMemberLogin melaporkan keadaan LOGIN — pemeriksaan bertaruh paling tinggi
// di modul ini.
//
// LOGIN menentukan klaim mana yang dilihat seorang mitra reasuransi: lima kueri inbox
// menyaringnya dengan `where login = {OperatorID.pyUserIdentifier}`. Dua keadaan
// merusaknya, dan keduanya merusak dalam diam:
//
//	berbagi  satu orang berpotensi melihat klaim milik mitra lain
//	kosong   mitranya tidak akan pernah melihat klaimnya sendiri
//
// Sistem lama TAHU keadaan pertama mungkin terjadi, dan menyelesaikannya dengan memilih
// sembarang satu — `RDB List/GetPNCList_PLA1-SQL.xml` membaca `order by reinsurerid desc
// fetch next 1 row only`. Yang dilaporkan di sini adalah berapa kali ia benar-benar terjadi.
func checkReinsurerMemberLogin(
	ctx context.Context,
	repo *masterreassql.Repo,
	print func(string, ...any),
) {
	shared, errOne := repo.CountSharedLogin(ctx)
	empty, errTwo := repo.CountEmptyLogin(ctx)
	if errOne != nil || errTwo != nil {
		print("  [catat] keadaan LOGIN tidak dapat diperiksa")
		return
	}

	if shared == 0 && empty == 0 {
		print("  [ok]    seluruh LOGIN terisi dan tidak ada yang dipakai dua kode reas")
		return
	}

	if shared > 0 {
		print("  [PENTING] %d LOGIN dipakai lebih dari satu kode reasuransi.", shared)
		print("            LOGIN menentukan klaim mana yang dilihat seorang mitra — lima")
		print("            kueri inbox menyaringnya. Satu login yang menunjuk beberapa kode")
		print("            berarti seseorang berpotensi melihat klaim milik mitra lain, dan")
		print("            layarnya tampil normal tanpa satu pun tanda. Bawa temuan ini ke")
		print("            Work Owner sebelum layar mitra reasuransi dibuka (R-20).")
	}
	if empty > 0 {
		print("  [PENTING] %d baris LOGIN-nya kosong.", empty)
		print("            Mitra pada baris itu tidak dapat dipakai masuk oleh siapa pun,")
		print("            sehingga ia tidak akan pernah melihat klaimnya sendiri — dan")
		print("            tidak ada apa pun di layar lama yang menunjukkannya.")
	}
}

// checkReinsurerMemberDelivery melaporkan dua keadaan yang membuat pemberitahuan PLA/DLA
// tidak sampai.
//
// Keduanya BUKAN kegagalan — aplikasi tetap berjalan dengan keduanya. Yang dilaporkan adalah
// keadaan, supaya ia diketahui sebelum seseorang menanyakan mengapa dokumennya tidak pernah
// diterima mitra.
func checkReinsurerMemberDelivery(
	ctx context.Context,
	repo *masterreassql.Repo,
	print func(string, ...any),
) {
	missing, errOne := repo.CountMissingEmail(ctx)
	orphan, errTwo := repo.CountWithoutFallback(ctx)
	if errOne != nil || errTwo != nil {
		print("  [catat] kelengkapan surel dan baris cadangan tidak dapat diperiksa")
		return
	}

	if missing > 0 {
		print("  [PENTING] %d baris EMAIL-nya kosong.", missing)
		print("            Surel pada baris ini adalah tujuan pemberitahuan PLA, Pre-DLA,")
		print("            dan DLA. Baris tanpa surel gagal dalam diam: dokumennya terbit,")
		print("            tercatat terkirim, dan tidak pernah sampai ke siapa pun.")
	} else {
		print("  [ok]    seluruh baris punya EMAIL tujuan")
	}

	if orphan > 0 {
		print("  [catat] %d kode reasuransi tidak punya baris cadangan (TYPE '1').", orphan)
		print("            BrowseEmailReas memakai baris TYPE '1' ketika tidak ada baris")
		print("            yang cocok dengan jenis dokumen yang sedang dikirim. Tanpa baris")
		print("            itu, jenis dokumen lain tidak menemukan surel tujuannya sama")
		print("            sekali. Keadaan ini dapat tercipta sistem lama sendiri: UPDATEREAS")
		print("            mengubah baris '1' menjadi tipe yang diminta alih-alih menyisipkan")
		print("            baris baru, sehingga cadangannya habis terpakai.")
	}
}

// checkCauseOfLossDetail memeriksa kelima objek yang dipakai modul Detail Penyebab
// Kerugian.
//
// # Kenapa LIMA, bukan satu
//
// Modul ini menyentuh objek yang kewenangannya berbeda-beda, dan hak akses atas salah
// satunya TIDAK menyiratkan hak atas yang lain:
//
//	POOLDATA.D_CAUSE_OF_LOSS             DITULIS  — tabel fisik, D_COL_ID + JSONDATA
//	POOLDATA.V_D_CAUSE_OF_LOSS           dibaca   — view berkolom atas tabel di atas
//	POOLDATA.V_D_CAUSE_OF_LOSS_BUSINESS  dibaca   — lini bisnis per detail
//	POOLDATA.V_M_CAUSE_OF_LOSS           dibaca   — master induk
//	POOLDATA.BUSINESS                    dibaca   — master lini bisnis, milik GISFW
//
// Yang paling mudah terlewat adalah pasangan pertama: hak BACA atas view tidak berarti hak
// TULIS atas tabel di belakangnya. Layar akan tampil penuh dan baru gagal saat petugas
// menekan Simpan.
//
// # Yang TIDAK dapat diperiksa di sini, dan itu penting
//
// Pemetaan kunci JSON ke kolom view adalah REKONSTRUKSI — definisi view-nya tidak ada di
// export (`R-08`). Bila kuncinya berbeda dari yang diduga, kolom view akan KOSONG tanpa
// satu pun galat.
//
// Pemeriksaan ini hanya dapat menunjukkan gejalanya: bila barisnya ada tetapi kolom
// DESCRIPTION seluruhnya kosong, hampir pasti kuncinya berbeda. Itulah sebabnya jumlah
// baris ber-DESCRIPTION terisi ikut dihitung, bukan hanya jumlah barisnya.
func checkCauseOfLossDetail(ctx context.Context, primary *sql.DB, print func(string, ...any)) {
	repo := detailpenyebabsql.NewRepo(primary)

	result := repo.CheckTables(ctx)
	blocked := false
	for _, object := range []string{
		"POOLDATA.V_D_CAUSE_OF_LOSS",
		"POOLDATA.D_CAUSE_OF_LOSS",
		"POOLDATA.V_D_CAUSE_OF_LOSS_BUSINESS",
		"POOLDATA.V_M_CAUSE_OF_LOSS",
		"POOLDATA.BUSINESS",
	} {
		if err := result[object]; err != nil {
			print("  [BELUM] %s belum dapat dibaca: %v", object, err)
			blocked = true
			continue
		}
		print("  [ok]    %s dapat dibaca", object)
	}

	if blocked {
		print("            Seluruhnya warisan Pega dan TIDAK dibuat migrasi aplikasi ini.")
		print("            Mintakan hak BACA kelimanya, dan hak TULIS pada")
		print("            POOLDATA.D_CAUSE_OF_LOSS — itu satu-satunya yang ditulis modul ini.")
		print("            Dibutuhkan pula sequence D_CAUSE_SEQ dan baris CURRENT_SITE='1'")
		print("            pada POOLDATA.M_SITE_DATABASE untuk menerbitkan D_COL_ID.")
		return
	}

	// Gejala kunci JSON yang tidak cocok — lihat doc comment di atas.
	var total, described int
	err := primary.QueryRowContext(ctx,
		`SELECT COUNT(D_COL_ID) FROM POOLDATA.V_D_CAUSE_OF_LOSS`).Scan(&total)
	if err != nil {
		print("  [GAGAL] V_D_CAUSE_OF_LOSS tidak dapat dihitung: %v", err)
		return
	}
	err = primary.QueryRowContext(ctx,
		`SELECT COUNT(D_COL_ID) FROM POOLDATA.V_D_CAUSE_OF_LOSS WHERE DESCRIPTION IS NOT NULL`).
		Scan(&described)
	if err != nil {
		print("  [GAGAL] V_D_CAUSE_OF_LOSS tidak dapat dihitung: %v", err)
		return
	}

	print("  [ok]    POOLDATA.V_D_CAUSE_OF_LOSS dapat dibaca: %d baris", total)
	if total > 0 && described == 0 {
		print("  [CURIGA] SELURUH %d barisnya ber-DESCRIPTION kosong.", total)
		print("            Kolom view terisi dari kunci di dalam JSONDATA, dan kunci yang")
		print("            dipakai modul ini REKONSTRUKSI (`R-08`). Kosong seluruhnya hampir")
		print("            pasti berarti kuncinya berbeda — mintakan definisi view-nya ke DBA")
		print("            SEBELUM modul ini dipakai menyimpan.")
		return
	}
	print("            %d di antaranya ber-DESCRIPTION terisi", described)
}

// checkRCLPUCL memeriksa modul Inbox RCL/PUCL (`MENU_ID 61`).
//
// # Kenapa pemeriksaannya lebih rinci daripada modul inbox lain
//
// Karena EMPAT hal di modul ini gagal TANPA GALAT bila kesimpulannya keliru, dan ketiga tab
// layar ini punya kolom yang IDENTIK — sehingga tidak ada apa pun di antarmuka yang
// menandakan isinya tertukar:
//
//   - Kolom `MSIG` nyaris tidak pernah terisi — satu baris dari 7.722 pada portal ASM,
//     dihitung 2026-09-30. Tab "Klaim MSIG" karena itu nyaris selalu kosong, dan tab
//     "Kelengkapan Dokumen" menampung selebihnya.
//   - `PUCL_APPROVE <> '1'` tidak menangkap nilai kosong. Klaim yang penandanya belum
//     pernah diisi hilang dari DUA tab sekaligus.
//   - Penyaring antrean DIHAPUS 2026-10-01: kolomnya berisi nama orang, bukan nama
//     antrean, sehingga menyaringnya mengosongkan ketiga tab. `TC_PNC_PUCL` tabel khusus
//     RCL/PUCL, jadi penyaring itu tidak lagi menyeleksi apa pun.
//   - `STATUS_CASE = '0'`. Artinya tidak diketahui, dan tidak ada master yang
//     menerjemahkannya di export mana pun. Bila nilainya berbeda di produksi, tab
//     "Cetak Surat" kosong sementara dua tab lain terisi normal.
//
// Keempatnya diperiksa di sini supaya kekeliruannya ketahuan saat `-periksa` dijalankan,
// bukan saat pengguna melaporkan "tabnya kosong".
//
// # Sejak 2026-10-01 ada SEBAB KELIMA yang mengosongkan seluruh layar
//
// Ketiga tab dan layar kerja kini membaca `POOLDATA.TC_PNC_PUCL`, tabel datar milik aplikasi
// ini. Tabel itu BARU, dan ia kosong sampai proses pengisi berjalan — sehingga layar yang
// kosong di sini belum tentu berarti antreannya kosong. Keduanya dibedakan di bawah.
func checkRCLPUCL(
	ctx context.Context,
	repo *inboxrclpuclsql.Repo,
	print func(string, ...any),
) {
	if err := repo.CheckTable(ctx); err != nil {
		print("  [BELUM] Inbox RCL/PUCL tidak dapat dibaca: %v", err)
		print("            Bila galatnya menyebut POOLDATA.TC_PNC_PUCL, tabel datarnya belum")
		print("            dibuat di portal ini — jalankan Database/CREATE_TABLE_3.SQL")
		print("            (menempuh `D-63`: DBA, atas persetujuan Work Owner).")
		print("            Bila galatnya menyebut KOLOM tabel itu, periksa namanya terhadap")
		print("            CREATE_TABLE_3.SQL: akhiran `_1` dibuang dan kata dipisah garis")
		print("            bawah, sehingga nama bentuk lama gagal dengan ORA-00904.")
		print("            Bila galatnya menyebut DATAPEGA.*, yang mati HANYA tombol unduh")
		print("            tab Cetak Surat — ketiga tabnya sendiri tidak lagi membacanya.")
		return
	}
	print("  [ok]    Tabel datar, kolom penyaring, dan tabel anak Inbox RCL/PUCL dapat dibaca")
	print("            Tabel anak memasok isian LAYAR KERJA yang diturunkan:")
	print("            T_CLAIM_OBJECTLIST  -> Nama Peserta DAN UP (keduanya ObjectName,")
	print("                                   dan itu memang benar — dikonfirmasi 2026-09-24),")
	print("            T_CLAIM_ADJUSTMENT  -> Jumlah Tagihan.")
	print("            TUJUH kolom dibaca dari TC_PNC_PUCL meski BELUM ada di")
	print("            CREATE_TABLE_3.SQL: PERIHAL, KETERANGAN1..3, dan ketiga parameter")
	print("            tindakan ID_OBJECT, ID_COVERAGE, ID_ADJUSTMENT. Portal yang tabelnya")
	print("            dibuat dari berkas DDL bersama akan gagal ORA-00904 pada klaim")
	print("            PERTAMA yang dibuka — bukan saat build.")
	print("            Email Tertanggung dibaca dari T_CLAIM_PNC.EMAIL_LOD, digabung lewat")
	print("            CLAIMNO — BUKAN CLAIMID, yang di tabel itu berawalan kunci Pega.")
	print("            TIGA isian sisanya properti clipboard Pega; ia tidak punya kolom,")
	print("            sehingga tidak ada yang perlu diperiksa di sini.")

	page := inboxrclpucl.Pagination{Page: 1, Size: 5}
	counts := map[string]int{}

	// Bentuk tanggal yang BENAR-BENAR digambar, diambil dari baris pertama yang ada.
	//
	// Ia diperiksa karena kegagalannya tidak menghasilkan galat: keempat kolom tanggal
	// modul ini bertipe `TIMESTAMP(6)`, dan driver mengembalikannya sebagai teks ISO
	// ber-offset. Sampai 2026-09-30 teks itu sampai ke layar apa adanya — bentuk yang tidak
	// pernah muncul di Pega. Portal yang tipe kolomnya berbeda akan menampakkannya di sini,
	// bukan lewat laporan pengguna.
	shape := ""

	// Ketiga tab diperiksa, bukan satu.
	//
	// Ketiganya membaca tabel yang sama dan dipisahkan HANYA oleh penyaring — dan justru
	// penyaring itulah yang paling mungkin keliru. Memeriksa satu tab saja akan menyatakan
	// modulnya sehat sementara dua pertiganya belum tersentuh.
	for _, code := range []string{
		inboxrclpucl.TabCetakSurat,
		inboxrclpucl.TabKelengkapanDokumen,
		inboxrclpucl.TabKlaimMSIG,
	} {
		tab, found := inboxrclpucl.FindTab(code)
		if !found {
			print("  [GAGAL] Tab %s tidak terdaftar di modul", code)
			return
		}

		result, err := repo.List(ctx, inboxrclpucl.Query{Tab: tab}, page)
		if err != nil {
			print("  [GAGAL] Tab %q tidak dapat dibaca: %v", tab.Name, err)
			return
		}

		counts[code] = result.Total
		print("  [ok]    Tab %q terbaca: %d baris", tab.Name, result.Total)

		for _, item := range result.Items {
			if shape == "" && item.InboxEntryAt != "" {
				shape = item.InboxEntryAt
			}
		}
	}

	if shape != "" {
		print("  [ok]    Tanggal digambar sebagai %q", shape)
		if strings.Contains(shape, "T") {
			print("  [PERIKSA] Bentuk di atas masih teks ISO mentah, bukan tanggal.")
			print("            Yang menggambarnya inboxrclpucl.DisplayTimeText; bentuk yang")
			print("            tidak dikenalinya dilewatkan apa adanya, sehingga inilah")
			print("            tandanya kolom di portal ini menyimpan bentuk lain.")
		}
	}

	if counts[inboxrclpucl.TabCetakSurat] == 0 &&
		counts[inboxrclpucl.TabKelengkapanDokumen] == 0 &&
		counts[inboxrclpucl.TabKlaimMSIG] == 0 {
		print("  [PERIKSA] KETIGA tab kosong sekaligus.")
		reportEmptyRCLPUCL(ctx, repo, print)
		return
	}

	// Satu keadaan yang BERDIRI SENDIRI dari terisi atau tidaknya sebuah tab.
	//
	// Ia dicetak selalu, bukan hanya saat ada tab yang kosong, karena ia menyangkut baris
	// yang memang TIDAK AKAN pernah muncul di tab mana pun — tab yang terisi tidak
	// membuktikan apa pun tentangnya.
	//
	// Ia BUKAN lagi pertanyaan terbuka. Work Owner memutuskan 2026-09-30 mengikuti Pega
	// apa adanya, sehingga penyaringnya tidak akan diubah. Kuerinya tetap dicetak untuk
	// satu keperluan yang tersisa: menjawab laporan "klaim saya hilang" dengan angka,
	// bukan dengan dugaan.
	print("  [CATATAN] Klaim yang suratnya SUDAH dicetak tetapi PUCLAPPROVE_1 kosong")
	print("            keluar dari tab \"Cetak Surat\" DAN tidak masuk tab mana pun — di")
	print("            sini maupun di Pega. Keputusan Work Owner 2026-09-30: ikuti Pega")
	print("            apa adanya. Bila ada yang melapor klaimnya hilang, inilah sebabnya,")
	print("            dan ini kueri yang menghitungnya:")
	print("            SELECT COUNT(*) FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK")
	print("             WHERE TANGGALCETAKDOKUMENPUCL_1 IS NOT NULL")
	print("               AND PUCLAPPROVE_1 IS NULL;")

	// Klaim yang ditandai tercetak TANPA suratnya terbit.
	//
	// Diperiksa SELALU, bukan hanya saat sebuah tab kosong: klaim seperti ini tergambar
	// normal di tab "Kelengkapan Dokumen" berdampingan dengan klaim yang suratnya memang
	// ada, dan tidak ada satu pun layar yang membedakan keduanya. Tab yang terisi justru
	// tidak membuktikan apa pun tentangnya.
	if tanpaSurat, err := repo.CountClaimsMissingLetter(ctx); err != nil {
		print("  [PERIKSA] Jumlah klaim bersurat-gagal tidak dapat dihitung: %v", err)
	} else if tanpaSurat == 0 {
		print("  [ok]    Tidak ada klaim yang ditandai tercetak tanpa suratnya terbit")
	} else {
		print("  [PERIKSA] %d klaim ditandai suratnya tercetak, tetapi suratnya TIDAK ada.",
			tanpaSurat)
		print("            Tombol \"Download Dokumen\" menandai dulu, baru menerbitkan PDF.")
		print("            Kegagalan penerbitan sengaja tidak membatalkan penandaan, sehingga")
		print("            klaimnya tetap berpindah ke tab \"Kelengkapan Dokumen\" dan")
		print("            petugas hanya membaca satu kalimat yang lewat. Sesudahnya klaim")
		print("            ini tidak dapat dibedakan dari klaim yang suratnya memang ada.")
		print("            Sebab yang sudah diketahui: sampai 2026-10-05 kueri work_object_key")
		print("            mencari kunci klaim di DATAPEGA.PC_ASM_FW_GCNMFW_WORK, yang tidak")
		print("            memuat klaim PNCN.* sama sekali. Itu sudah diperbaiki.")
		print("            Pemulihannya: jalankan migrations/0015_pucl_cetak_tanpa_surat.up.sql")
		print("            (DBA, atas persetujuan Work Owner — `D-63`). Klaimnya kembali ke")
		print("            tab \"Cetak Surat\" dan tombolnya dapat ditekan ulang.")
		print("            Angka yang TIDAK turun sesudah itu berarti sebabnya yang lain;")
		print("            cari \"surat rcl/pucl gagal diterbitkan\" di log peladen.")
	}

	if counts[inboxrclpucl.TabCetakSurat] == 0 {
		print("  [PERIKSA] Tab \"Cetak Surat\" kosong sementara tab lain terisi.")
		print("            Periksa apakah STATUSCASE_1 masih bernilai %q untuk klaim yang",
			inboxrclpucl.ExpiryStatusActive)
		print("            suratnya belum dicetak. Arti kolom itu tidak diketahui, dan")
		print("            Work Owner memutuskan 2026-09-30 untuk mengikuti Pega apa")
		print("            adanya — sehingga peringatan inilah satu-satunya yang akan")
		print("            menyebut sebabnya bila nilainya kelak berubah.")
	}

	if counts[inboxrclpucl.TabKlaimMSIG] == 0 {
		print("  [CATATAN] Tab \"Klaim MSIG\" kosong, dan itu BUKAN kerusakan.")
		print("            MSIG_1 menandai klaim yang datanya dari atau untuk perusahaan")
		print("            MSIG (Work Owner, 2026-09-30), dan kolom itu nyaris tidak pernah")
		print("            terisi: hitungan langsung pada portal ASM 2026-09-30 menemukan")
		print("            SATU baris dari 7.722. Kosong di portal ini karena itu wajar.")
		print("            Nilai pembandingnya (%q) DISALIN dari penyaring Report",
			inboxrclpucl.MSIGMarker)
		print("            Definition Pega, bukan ditebak, sehingga tab ini kosong di sini")
		print("            persis bila ia kosong juga di Pega. Tidak ada yang perlu")
		print("            dipastikan: seluruh klaim bersurat lain berada di tab")
		print("            \"Kelengkapan Dokumen\".")
	}

	if counts[inboxrclpucl.TabKelengkapanDokumen] == 0 {
		print("  [PERIKSA] Tab \"Kelengkapan Dokumen\" kosong.")
		print("            Kemungkinan terbesarnya BUKAN antrean yang sepi melainkan")
		print("            penyaring PUCLAPPROVE_1 <> %q — %q berarti PUCL sudah",
			inboxrclpucl.PUCLReturnedToAnalyst, inboxrclpucl.PUCLReturnedToAnalyst)
		print("            mengembalikan klaimnya ke Analyst, %q berarti klaimnya masih",
			inboxrclpucl.PUCLWithPUCL)
		print("            di tangan PUCL (Work Owner, 2026-09-30). Perbandingan itu tidak")
		print("            pernah bernilai benar untuk nilai KOSONG, sehingga klaim yang")
		print("            penandanya belum pernah diisi tidak muncul. Perilakunya")
		print("            direplikasi dari Pega dengan sengaja; periksa sebarannya:")
		print("            SELECT PUCLAPPROVE_1, COUNT(*) FROM")
		print("             DATAPEGA.PC_ASM_FW_GCNMFW_WORK GROUP BY PUCLAPPROVE_1;")
	}
}

// reportEmptyRCLPUCL menjelaskan ketiga tab yang kosong DENGAN ANGKA.
//
// # Kenapa angka, bukan daftar kemungkinan
//
// Karena daftar kemungkinan menyuruh orang memeriksanya satu per satu di SQL*Plus, dan
// sebagian besar yang membaca keluaran ini tidak akan melakukannya. Angka per penyaring
// menunjuk langsung penyaring mana yang mengosongkan layar.
//
// Penyaring yang menghasilkan NOL sementara tabelnya BERISI adalah jawabannya. Tabel yang
// kosong adalah jawaban yang berbeda, dan keduanya terbaca sama di layar.
func reportEmptyRCLPUCL(
	ctx context.Context,
	repo *inboxrclpuclsql.Repo,
	print func(string, ...any),
) {
	d, err := repo.DiagnoseEmpty(ctx)
	if err != nil {
		print("            Sebabnya tidak dapat dihitung: %v", err)
		return
	}

	if d.Total == 0 {
		print("            POOLDATA.TC_PNC_PUCL KOSONG — nol baris.")
		print("            Tabelnya baru, dan proses pengisinya tampaknya belum berjalan di")
		print("            portal ini. Yang kurang DATA-nya, bukan kuerinya.")
		return
	}

	print("            Tabelnya BERISI %d baris, jadi yang mengosongkan layar adalah salah",
		d.Total)
	print("            satu penyaring. Berapa baris yang lolos tiap penyaring, satu per satu:")
	print("")
	print("              status kerja bukan 'selesai'      %d", d.NotCompleted)
	print("              tab 1  surat BELUM dicetak        %d", d.WithoutLetter)
	print("              tab 1  STATUS_CASE = %-4q         %d",
		inboxrclpucl.ExpiryStatusActive, d.CaseStatusMatch)
	print("              tab 2+3  surat SUDAH dicetak      %d", d.WithLetter)
	print("              tab 2+3  PUCL_APPROVE <> %-4q     %d",
		inboxrclpucl.PUCLReturnedToAnalyst, d.StillWithPUCL)
	print("              tab 2  bukan jalur MSIG           %d", d.NotMSIG)
	print("              tab 3  jalur MSIG                 %d", d.MSIG)
	print("")
	print("            Penyaring yang bernilai NOL di atas itulah yang mengosongkan tabnya.")
	print("            Seluruhnya replikasi penyaring Pega, bukan tambahan aplikasi ini —")
	print("            jadi baris yang tertolak memang tidak terlihat di Pega pula.")

	if d.StillWithPUCL == 0 {
		print("")
		print("            PERHATIKAN `PUCL_APPROVE <> %q` meloloskan NOL baris.",
			inboxrclpucl.PUCLReturnedToAnalyst)
		print("            Nilai %q berarti PUCL SUDAH mengembalikan klaimnya ke Analyst",
			inboxrclpucl.PUCLReturnedToAnalyst)
		print("            (Work Owner, 2026-09-30), sehingga klaim itu memang bukan lagi")
		print("            pekerjaan PUCL. Bila baris uji dimaksudkan tampil, isikan %q.",
			inboxrclpucl.PUCLWithPUCL)
	}
}

// checkKomunikasiCabang memastikan layar Inbox Komunikasi Cabang dapat dijalankan.
//
// Tiga hal diperiksa, dan ketiganya gagal dengan cara yang berbeda:
//
//   - Tabel percakapan tidak terbaca → layar tidak dapat dipakai sama sekali.
//   - Tabel lampiran tidak terbaca → layar tetap dapat dipakai, lampirannya saja kosong.
//   - Penerjemahan cabang gagal → layar TERTUTUP bagi semua orang, dan itu disengaja.
//
// Yang ketiga patut dibaca dua kali. Di modul Inbox Laporan Klaim, penerjemahan cabang yang
// gagal hanya melebarkan daftar. Di sini ia MENUTUP layar, karena batas datanya memang
// batas itu — dan `P-5` menetapkan petugas yang cabangnya tidak terbaca dilayani sebagai
// kantor pusat, sehingga tidak ada cara membedakan "DB Link mati" dari "Anda petugas pusat"
// selain dengan menolak.
func checkKomunikasiCabang(
	ctx context.Context,
	repo *inboxkomunikasicabangsql.Repo,
	resolver *inboxkomunikasicabangsql.BranchResolver,
	print func(string, ...any),
) {
	if err := repo.CheckTable(ctx); err != nil {
		print("  [BELUM] Inbox Komunikasi Cabang tidak dapat dibaca: %v", err)
		print("            Modul ini TIDAK menuntut migrasi — seluruh tabelnya milik Pega.")
		print("            Periksa hak SELECT akun aplikasi atas POOLDATA.M_KOMUNIKASI_PNC,")
		print("            POOLDATA.D_KOMUNIKASI_PNC, POOLDATA.M_KOMUNIKASI_CABANG,")
		print("            POOLDATA.V_LST_DOC_TYPE, dan POOLDATA.V_LST_DET_TYPE_DOC.")
		print("            Bila galatnya menyebut M_KOMUNIKASI_CABANG.CREATEDDATE: nama kolom")
		print("            itu DITEBAK. Tidak satu pun INSERT mengisinya, dan DDL-nya belum")
		print("            ada (`R-08`). Sebutkan nama kolom tanggal yang sebenarnya, dan")
		print("            perbaikannya satu kata di detail_thread.")
		print("            Bila galatnya menyebut KOLOM, kolom itu memang tidak ada —")
		print("            seluruh nama kolom modul ini dibaca dari kueri Pega, bukan dari")
		print("            DDL, yang belum pernah diterima (`R-08`).")
		return
	}
	print("  [ok]    Tabel percakapan dan tabel lampiran Inbox Komunikasi Cabang terbaca")

	if err := resolver.CheckTable(ctx); err != nil {
		print("  [BELUM] penerjemahan cabang komunikasi belum dapat dijalankan: %v", err)
		print("            Sumbernya POOLDATA.BRANCH + LST_USER_ASURANSI dan HRDASM.V_HRD_MST")
		print("            lewat DB link @asmd.sinarmas.co.id, sama seperti GetIDCabang di Pega.")
		print("            BERBEDA dari Inbox Laporan Klaim: selama gagal, layar ini TERTUTUP")
		print("            bagi semua orang — bukan melebar. Batas datanya memang cabang itu,")
		print("            dan melayaninya tanpa batas berarti menampilkan percakapan cabang")
		print("            lain tanpa satu pun galat (`R-20`).")
		return
	}
	print("  [ok]    cabang petugas dapat diterjemahkan dari login")

	// Kedua pencacah dijalankan pada batas KANTOR PUSAT.
	//
	// Bukan pada cabang tertentu: `-periksa` tidak punya petugas yang sedang masuk, dan
	// kantor pusat adalah satu-satunya batas yang dapat disusun tanpa login siapa pun.
	// Angkanya karena itu menyatakan kesehatan kueri, bukan beban kerja sebuah cabang.
	filter := inboxkomunikasicabang.ResolveBranch(
		inboxkomunikasicabang.HeadOfficeBranch, true)

	summary, err := repo.Summarize(ctx, filter)
	if err != nil {
		print("  [GAGAL] pencacah percakapan tidak dapat dijalankan: %v", err)
		return
	}
	print("  [ok]    Percakapan kantor pusat: %d belum dijawab, %d sudah dijawab",
		summary.NotAnswered, summary.Answered)

	// Daftar cabang untuk pemilih tujuan "Kirim Pesan".
	//
	// Ia diperiksa TERPISAH karena sumbernya pun terpisah — `POOLDATA.V_D_SURVEYORS`, bukan
	// tabel percakapan — sehingga kegagalannya berakibat berbeda: layar tetap dapat dibaca
	// dan dibalas, hanya pembuatan percakapan baru yang lumpuh.
	//
	// Ia juga satu-satunya kueri modul ini yang PENYARINGNYA DITEBAK: kueri asli yang mengisi
	// pemilih itu tidak ada di export mana pun, dan yang terbaca hanyalah kelas halamannya.
	// Angka di bawah adalah cara tercepat menguji tebakan itu terhadap basis data sungguhan.
	branches, err := repo.Branches(ctx)
	if err != nil {
		print("  [BELUM] daftar cabang tujuan tidak dapat dibaca: %v", err)
		print("            Sumbernya POOLDATA.V_D_SURVEYORS, kolom BRANCH, BRANCHNAME, EMAIL.")
		print("            Selama gagal, layar tetap dapat DIBACA dan DIBALAS — yang lumpuh")
		print("            hanya tombol \"Kirim Pesan\".")
		print("            PERHATIAN: kueri ini penyaringnya DITEBAK. Kueri Pega yang")
		print("            sebenarnya mengisi pemilih cabang tidak ada di export mana pun")
		print("            (`R-16`); yang terbaca hanya kelas halamannya. Bila galatnya")
		print("            menyebut kolom, kolom itu memang bukan yang dipakai Pega.")
		return
	}
	print("  [ok]    Daftar cabang tujuan terbaca: %d cabang", len(branches))

	if len(branches) == 0 {
		print("  [PERIKSA] Tidak ada satu pun cabang yang dapat dipilih sebagai tujuan.")
		print("            Tombol \"Kirim Pesan\" akan tampil tetapi tidak dapat dipakai untuk")
		print("            mengirim ke cabang. Jalankan:")
		print("              SELECT COUNT(DISTINCT BRANCH) FROM POOLDATA.V_D_SURVEYORS")
		print("               WHERE BRANCH IS NOT NULL AND BRANCHNAME IS NOT NULL;")
	}

	if summary.Total() == 0 {
		print("  [PERIKSA] Tidak ada satu pun percakapan kantor pusat yang berjalan.")
		print("            Periksa apakah kanal percakapan masih bernama %q:",
			inboxkomunikasicabang.CaseOpen)
		print("            SELECT CASEID, COUNT(*) FROM POOLDATA.M_KOMUNIKASI_PNC")
		print("             GROUP BY CASEID;")
		print("            Nilai penutupnya %q, dan keduanya BERAWALAN sama — sehingga",
			inboxkomunikasicabang.CaseClosed)
		print("            kanal yang berubah mengosongkan layar tanpa satu pun galat.")
		return
	}

	// Kedua tab dijalankan, bukan satu.
	//
	// Keduanya membaca tabel yang sama dan dipisahkan HANYA oleh satu penyaring — dan
	// justru penyaring itulah yang paling mungkin keliru. Memeriksa satu tab saja akan
	// menyatakan modulnya sehat sementara separuhnya belum tersentuh.
	page := inboxkomunikasicabang.Pagination{Page: 1, Size: 5}
	rows := map[string]int{}

	for _, code := range []string{
		inboxkomunikasicabang.TabNotAnswered,
		inboxkomunikasicabang.TabAnswered,
	} {
		tab, found := inboxkomunikasicabang.FindTab(code)
		if !found {
			print("  [GAGAL] Tab %s tidak terdaftar di modul", code)
			return
		}

		result, err := repo.List(
			ctx,
			inboxkomunikasicabang.Query{Tab: tab, Branch: filter},
			page,
		)
		if err != nil {
			print("  [GAGAL] Tab %q tidak dapat dibaca: %v", tab.Name, err)
			return
		}

		rows[code] = result.Total
		print("  [ok]    Tab %q terbaca: %d baris", tab.Name, result.Total)
	}

	// Selisih antara pencacah dan tabel BUKAN cacat, dan justru karena itu ia dijelaskan
	// di sini — orang yang membandingkan keempat angka di atas akan mengira ada yang rusak.
	if rows[inboxkomunikasicabang.TabAnswered] != summary.Answered ||
		rows[inboxkomunikasicabang.TabNotAnswered] != summary.NotAnswered {
		print("  [catatan] Angka pencacah dan jumlah baris tabel BERBEDA, dan itu memang")
		print("            perilaku sistem lama. Pencacah memeriksa DUA kolom balasan")
		print("            (REPLYFROM dan REPLYMESSAGE) sementara tabel hanya memeriksa")
		print("            satu, dan pencacah TIDAK menyaring pengirim maupun pesan kosong")
		print("            sementara tabel menyaringnya. Sebarannya:")
		print("            SELECT COUNT(*) FROM POOLDATA.M_KOMUNIKASI_PNC")
		print("             WHERE CASEID = '%s' AND REPLYMESSAGE IS NOT NULL",
			inboxkomunikasicabang.CaseOpen)
		print("               AND REPLYFROM IS NULL;")
		print("            Baris sejumlah itu muncul di tabel tetapi tidak di pencacah.")
	}
}

// checkInboxProgressClaim memastikan tabel yang dibaca layar Inbox Progress Claim
// terjangkau.
//
// Sama seperti Inbox Admin, modul ini TIDAK menuntut satu pun migrasi: seluruh tabel yang
// dibacanya sudah ada dan milik sistem lama. Yang dapat gagal karena itu bukan "tabelnya
// belum dibuat", melainkan "akun aplikasi belum diberi hak SELECT atasnya".
func checkInboxProgressClaim(
	ctx context.Context,
	repo *inboxprogressclaimsql.Repo,
	print func(string, ...any),
) {
	if err := repo.CheckTable(ctx); err != nil {
		print("  [BELUM] POOLDATA.PEGA_DASHBOARDPNC tidak dapat dibaca: %v", err)
		print("            Tanpa hak baca atasnya, seluruh bagian Inbox Progress Claim")
		print("            kosong. Tabel ini milik sistem lama dan tidak dibuat migrasi")
		print("            mana pun.")
		return
	}
	print("  [ok]    POOLDATA.PEGA_DASHBOARDPNC dapat dibaca")
	print("            Catatan: modul ini juga membaca GCNM_PROGRESS_CLAIM,")
	print("            GCNM_PROGRESS_POSISI_PNC, GCNM_MST_PROGRESS_KLAIM,")
	print("            GCNM_MST_PROGRESS, MST_USER_TEKNIK, dan T_CLAIM_PNC.")
	print("            Penyaring Cabang BELUM aktif — sumbernya DB Link ke HRD yang")
	print("            belum punya API pengganti (R-03).")
}

// checkInboxAnalystDoctor memastikan tabel DAN seluruh kolom yang dibaca layar Inbox Analyst
// Doctor terjangkau.
//
// # Kenapa modul ini diperiksa dalam DUA langkah, tidak seperti modul lain
//
// Karena dua hal yang berbeda dapat gagal di sini, dan keduanya menuntut tindakan yang
// berbeda pula:
//
//	tabel   hak SELECT belum diberikan — sama seperti modul lain
//	kolom   ada nama kolom yang tidak ada di tabelnya — khas modul ini
//
// # Yang pernah terjadi, dan kenapa pemeriksaan ini akhirnya menangkapnya
//
// `Report Definition/InboxAnalystDoctor_RD-RD.xml` menandai `.ClaimData.isComplianceTransfer`
// dan `.ClaimData.AnalystDoctorRemaks` sebagai `unexposed` — keduanya hidup di dalam blob
// Pega, bukan sebagai kolom SQL. Kueri modul ini sempat menebak nama kolomnya dengan
// mengikuti konvensi `_1`, dan tebakan itu TERNYATA SALAH: katalog Oracle membuktikan
// `ISCOMPLIANCETRANSFER_1` maupun `ANALYSTDOCTORREMAKS_1` tidak ada (2026-10-09). Selama itu
// berlaku, layarnya gagal ORA-00904 pada setiap permintaan.
//
// Penyaringnya kini memakai `PC_ASSIGN_WORKLIST.PXTASKLABEL = 'Analyst Doctor'` — nama tahap
// pada `Flow/Register_Flow.xml` `Assignment13` — dan kolom "Komentar dari PIC Teknis" tidak
// lagi diambil dari SQL.
//
// Inilah satu-satunya tempat keadaan semacam itu diketahui SEBELUM ada pengguna yang membuka
// layarnya. Tanpa pemeriksaan ini, yang pertama menemukannya adalah petugas medis yang
// layarnya gagal dimuat.
func checkInboxAnalystDoctor(
	ctx context.Context,
	repo *inboxanalystdoctorsql.Repo,
	print func(string, ...any),
) {
	if err := repo.CheckTables(ctx); err != nil {
		print("  [BELUM] Tabel Inbox Analyst Doctor tidak dapat dibaca: %v", err)
		print("            Dibutuhkan hak SELECT atas DATAPEGA.PC_ASM_FW_GCNMFW_WORK dan")
		print("            DATAPEGA.PC_ASSIGN_WORKLIST. Keduanya milik sistem lama dan tidak")
		print("            dibuat migrasi mana pun.")
		return
	}
	print("  [ok]    Tabel Inbox Analyst Doctor dapat dibaca")

	if err := repo.CheckColumns(ctx); err != nil {
		print("  [BELUM] Ada kolom antrean Analyst Doctor yang tidak ada: %v", err)
		print("            Galat Oracle di atas MENYEBUT nama kolom yang salah; mulailah")
		print("            dari sana. Yang dibaca layar ini:")
		print("              PC_ASM_FW_GCNMFW_WORK  PYID, POLICYNO, QQNAME, BRANCHNAME,")
		print("                                     PYORIGUSERID, USERTEKNIS_1,")
		print("                                     PXCREATEDATETIME, PYSTATUSWORK")
		print("              PC_ASSIGN_WORKLIST     PXTASKLABEL, PXASSIGNEDOPERATORID")
		return
	}
	print("  [ok]    Seluruh kolom antrean Analyst Doctor ada")
	print("            Antrean dikenali dari PXTASKLABEL = 'Analyst Doctor', nama tahap pada")
	print("            Flow/Register_Flow.xml Assignment13 — bukan dari isComplianceTransfer,")
	print("            yang ternyata tidak punya kolom di basis data ini.")
	print("            Kolom \"Komentar dari PIC Teknis\" karena itu SELALU kosong; mengisinya")
	print("            menuntut kolom baru dari DBA, bukan tebakan nama.")
	print("            Catatan: antrean ini disaring dengan Operator ID pemanggil, sehingga")
	print("            pengguna tanpa tugas Analyst Doctor melihatnya kosong — dan itu")
	print("            jawaban yang benar, bukan kerusakan.")
}

// checkInboxSurvey memeriksa prasyarat layar My Work (MENU_ID 50) terhadap Oracle sungguhan.
//
// # Kenapa modul ini paling perlu diperiksa di antara seluruh inbox
//
// Karena ia satu-satunya yang membaca tabel yang BELUM pernah dibaca modul mana pun di
// aplikasi ini — `POOLDATA.T_SURVEYORLIST` — dan karena sumber datanya BERGESER DUA KALI:
// layar lamanya membaca objek kerja `Work-SurveyClaim` di tabel Pega, penggantinya yang
// pertama terbukti tidak memuat satu pun baris jenis itu, dan sejak 2026-09-29 header klaimnya
// diambil dari `POOLDATA.T_CLAIM_PNC` dengan nama kolom yang BERBEDA JAUH.
//
// Tanpa pemeriksaan ini, yang pertama menemukan keadaan itu adalah seorang adjuster yang
// layarnya gagal dimuat.
func checkInboxSurvey(
	ctx context.Context,
	repo *inboxsurveysql.Repo,
	print func(string, ...any),
) {
	if err := repo.CheckTables(ctx); err != nil {
		print("  [BELUM] Tabel My Work tidak dapat dibaca: %v", err)
		print("            Dibutuhkan hak SELECT atas POOLDATA.T_SURVEYORLIST,")
		print("            POOLDATA.T_CLAIM_PNC, dan POOLDATA.MST_LOGIN_SURVEYOR.")
		print("            T_SURVEYORLIST yang paling patut diperiksa: ia tabel yang BELUM")
		print("            pernah dibaca modul mana pun, sehingga hak bacanya belum pernah")
		print("            terbukti.")
		return
	}
	print("  [ok]    Tabel My Work dapat dibaca")

	if err := repo.CheckColumns(ctx); err != nil {
		print("  [BELUM] Kolom antrean My Work tidak lengkap: %v", err)
		print("            Penamaan kolom klaim BERBEDA dari tabel datar yang dipakai modul")
		print("            lain — CLAIMNO bukan PYID, NOPOLIS bukan POLICYNO, PICTEKNIK bukan")
		print("            USERTEKNIS_1 — dan salah satu saja menjatuhkan seluruh layar.")
		print("            Satu kolom memang BELUM dikonfirmasi DBA:")
		print("              LOSSTYPE  ->  kolom \"Cause Of Loss\"")
		print("            Yang diminta ke DBA — satu kueri katalog:")
		print("              SELECT TABLE_NAME, COLUMN_NAME, DATA_TYPE, NUM_DISTINCT")
		print("                FROM ALL_TAB_COLUMNS WHERE OWNER = 'POOLDATA'")
		print("                 AND TABLE_NAME IN ('T_CLAIM_PNC', 'T_SURVEYORLIST');")
		return
	}
	print("  [ok]    Kolom antrean My Work ada")

	// Keempat kolom adjuster diperiksa TERPISAH, dan kegagalannya DIHARAPKAN.
	//
	// Ia satu-satunya cara mengetahui kolomnya sudah tiba tanpa mencobanya secara kebetulan.
	// Selama belum ada, empat dari tujuh tab dan tiga dari tiga belas kolom layar menyatakan
	// dirinya belum tersedia — bukan menampilkan kosong.
	// Keberadaan kolom dilaporkan PER KOLOM, lewat katalog.
	//
	// Probe berbasis parsing bersifat semua-atau-tidak: satu kolom yang belum ada membuat
	// kolom lain yang sudah ada ikut terbaca belum ada. Itu persis keadaan 2026-09-30.
	columns, err := repo.CheckNewColumns(ctx)
	if err != nil {
		print("  [catat] Katalog kolom tidak dapat dibaca: %v", err)
		return
	}

	ada := func(punya bool) string {
		if punya {
			return "ADA  "
		}
		return "belum"
	}
	print("  [catat] Kelima kolom yang ditunggu di POOLDATA.T_SURVEYORLIST:")
	print("            ADJUSTERACCEPT %s · PYSTATUSWORK %s · REFNO %s",
		ada(columns.Accept), ada(columns.WorkStatus), ada(columns.Reference))
	print("            ADJUSTERPIC %s · RESCHEDULELOCATION %s",
		ada(columns.AdjusterPIC), ada(columns.SurveyLocation))

	if !columns.All() {
		print("            Yang belum ada menahan: tab Outstanding/ALL/Invoice (ADJUSTERACCEPT),")
		print("            tab Close dan penyaring berkas tutup (PYSTATUSWORK), setengah kotak")
		print("            cari dan kolom Reference No (REFNO), kolom PIC Loss Adjuster")
		print("            (ADJUSTERPIC), dan kolom Location (RESCHEDULELOCATION).")
		print("            Perubahan skema menempuh D-63 — lihat")
		print("            docs/permintaan-kolom-t-surveyorlist.md")
	}

	// Keterisian diperiksa TERPISAH dari keberadaan kolom.
	//
	// Keduanya diperbaiki langkah yang berbeda: kolom lewat ALTER oleh DBA, isinya oleh
	// procedure. Menyatukannya membuat "kolomnya ada tetapi masih kosong" terbaca siap — dan
	// menghidupkan tab atas dasar itu menghasilkan tab kosong yang terbaca "tidak ada
	// pekerjaan".
	if !columns.Accept && !columns.Reference {
		return
	}

	filled, err := repo.CheckFilledColumns(ctx)
	if err != nil {
		print("  [catat] Keterisian kolom tidak dapat diukur: %v", err)
		return
	}

	print("  [catat] Keterisian, dari %d baris: ADJUSTERACCEPT %d · REFNO %d · PYSTATUSWORK %d",
		filled.TotalRows, filled.Accept, filled.Reference, filled.WorkStatus)
	print("            ADJUSTERPIC %d · RESCHEDULELOCATION %d",
		filled.AdjusterPIC, filled.SurveyLocation)

	if filled.AdjusterPIC == 0 {
		print("  [BELUM] ADJUSTERPIC ADA tetapi SELURUHNYA kosong")
		print("            Kolom PIC Loss Adjuster tetap menggambar SURVEYOR_NAME sebagai")
		print("            pengganti. Diukur 2026-10-03: pada 1.796 berkas adjuster eksternal")
		print("            itu ORANG YANG BERBEDA, bukan nama lain untuk orang yang sama.")
	}
	if filled.SurveyLocation == 0 {
		print("  [BELUM] RESCHEDULELOCATION ADA tetapi SELURUHNYA kosong")
		print("            Kolom Location tetap menggambar LOCATION_SURVEY sebagai pengganti —")
		print("            berbeda dari Pega pada 660 dari 2.427 berkas.")
	}
	if filled.WorkStatus == 0 {
		print("  [BELUM] PYSTATUSWORK ADA tetapi SELURUHNYA kosong")
		print("            Tab Close belum dapat dihitung, dan berkas survei yang sudah ditutup")
		print("            di Pega masih ikut ditampilkan pada tab yang berjalan.")
	}
	if filled.Accept == 0 {
		print("  [BELUM] ADJUSTERACCEPT ADA tetapi SELURUHNYA kosong")
		print("            Tab Outstanding menyaring kolom itu IS NULL, sehingga menghidupkannya")
		print("            sekarang akan menampilkan SELURUH antrean sebagai \"belum")
		print("            dikonfirmasi adjuster\" — layar terisi wajar, isinya salah.")
		print("            Yang dibutuhkan: procedure pengisinya, bukan ALTER lagi.")
	}
	if filled.Reference == 0 {
		print("  [BELUM] REFNO ADA tetapi SELURUHNYA kosong")
		print("            Kolom Reference No akan tampil kosong di setiap baris, dan kotak")
		print("            cari tidak akan pernah menemukan apa pun lewat nomor referensi.")
	}

	// KPI diperiksa TERAKHIR dan kegagalannya tidak menghentikan apa pun.
	//
	// Tab KPI membaca tabel LAIN yang diisi procedure terpisah. Ketiadaannya mengosongkan
	// satu tab, bukan merusak layar — dan menyamakannya dengan kegagalan di atas akan
	// membuat modul yang sebenarnya siap terbaca sebagai belum siap.
	if err := repo.CheckKPI(ctx); err != nil {
		print("  [catat] Tab KPI My Work belum dapat dipakai: %v", err)
		print("            Ini TIDAK menghalangi tab INBOX. POOLDATA.DETAIL_KPI_ADJUSTER")
		print("            diisi Database/INSERT_KPIADJUSTER.prc, dan ketiadaannya hanya")
		print("            mengosongkan satu tab.")
		return
	}
	print("  [ok]    Tabel KPI adjuster dapat dibaca")
	print("            Catatan: antrean ini disaring NAMA SURVEYOR yang diturunkan dari")
	print("            login lewat POOLDATA.MST_LOGIN_SURVEYOR. Pengguna yang belum")
	print("            terdaftar di sana menerima 403 yang menyebut sebabnya — bukan")
	print("            antrean kosong.")
}

// checkInboxRCL memastikan tabel dan kolom yang dibaca layar Inbox RCL terjangkau.
//
// Sumbernya POOLDATA.TC_PNC_PUCL (keputusan Work Owner 2026-10-05) dan POOLDATA.M_LOGIN_PNC.
// Dua langkah, karena sebab gagalnya berbeda: hak baca, atau kolom ALASAN_DOKTER_REJECT_RCL
// yang ditambahkan Work Owner pada 2026-10-05.
func checkInboxRCL(
	ctx context.Context,
	repo *inboxrclsql.Repo,
	print func(string, ...any),
) {
	if err := repo.CheckTables(ctx); err != nil {
		print("  [BELUM] Tabel Inbox RCL tidak dapat dibaca: %v", err)
		print("            Dibutuhkan hak SELECT atas POOLDATA.TC_PNC_PUCL dan POOLDATA.M_LOGIN_PNC.")
		return
	}
	print("  [ok]    Tabel Inbox RCL dapat dibaca (TC_PNC_PUCL, M_LOGIN_PNC)")

	if err := repo.CheckColumns(ctx); err != nil {
		print("  [BELUM] Kolom ALASAN_DOKTER_REJECT_RCL belum ada di TC_PNC_PUCL: %v", err)
		return
	}
	print("  [ok]    Kolom ALASAN_DOKTER_REJECT_RCL ada")
	print("            Catatan: antrean disaring dengan LOGIN_ID pemanggil yang aktif di")
	print("            POOLDATA.M_LOGIN_PNC, RCL_PUCL 1 (RCL) atau 3 (MSIG).")
}

// checkOutstanding menjalankan kueri Inbox Outstanding terhadap Oracle sungguhan.
//
// Inilah satu-satunya jalan membuktikan kuerinya sah selama migrasi 0001 belum dijalankan:
// layarnya sendiri menuntut sesi, dan sesi disimpan di `CPNC_SESI_AKTIF` yang belum ada.
//
// Yang dibuktikan di sini: SQL-nya diterima Oracle, penanda bind-nya benar, dan
// pembacaan 21 kolomnya cocok dengan tipe kolom yang sebenarnya. Yang TIDAK dibuktikan:
// kesetaraan hasilnya dengan layar Pega — itu gerbang 1, milik `S-8`.
func checkOutstanding(ctx context.Context, repo *inboxoutstandingsql.Repo, login string, print func(string, ...any)) {
	// Daftar TANPA pemilik tidak lagi mungkin, dan itu memang yang dikehendaki: layar ini
	// menyaring `PXASSIGNEDOPERATORID = pemanggil`, sehingga memeriksanya tanpa identitas
	// berarti memeriksa sesuatu yang tidak pernah dijalankan aplikasi.
	// Unduhan diperiksa LEBIH DULU, dan sengaja TANPA -login.
	//
	// Ia tidak menyaring pemilik pekerjaan sama sekali, sehingga ia satu-satunya bagian
	// modul ini yang dapat dibuktikan tanpa identitas. Menaruhnya sesudah penjagaan login
	// di bawah akan membuatnya ikut terlewat — persis anggapan keliru yang membuat export
	// dulu memanggil ulang daftar.
	checkOutstandingExport(ctx, repo, login, print)

	if login == "" {
		print("  [lewat] My Inbox: butuh -login <nama pengguna> — daftarnya menyaring per pemilik")
		return
	}

	// Identitas lama dibaca dulu, dan hasilnya DICETAK.
	//
	// Tanpa baris ini, "0 pekerjaan" punya dua sebab yang tampak sama: memang tidak ada
	// pekerjaan, atau identitas login tidak pernah dipetakan ke nama operator Pega.
	legacy, err := repo.LegacyOperatorFor(ctx, login)
	if err != nil {
		print("  [BELUM] POOLDATA.T_ACCESS_GROUP_PNC tidak dapat dibaca: %v", err)
		return
	}
	if legacy == "" {
		print("  [catat] %s tidak punya identitas lama — hanya identitas ini yang dicocokkan", login)
	} else {
		print("  [ok]    identitas lama %s: %s", login, legacy)
	}

	page, err := repo.List(ctx, inboxoutstanding.Filter{
		AssignedTo:       login,
		AssignedToLegacy: legacy,
		Limit:            5,
	})
	if err != nil {
		print("  [BELUM] POOLDATA.T_CLAIMLIST_ADMIN tidak dapat dibaca: %v", err)
		print("            Tabelnya milik sistem lama, bukan dibuat migrasi. Selama belum ada,")
		print("            layar My Inbox tidak dapat dipakai terhadap Oracle.")
		return
	}

	print("  [ok]    POOLDATA.T_CLAIMLIST_ADMIN dapat dibaca: %d pekerjaan milik %s", page.Total, login)

	// Ringkasan donut diperiksa terhadap Oracle sungguhan.
	//
	// Yang dibuktikan di sini bukan angkanya melainkan JUMLAHNYA: irisan-irisan wajib
	// menjumlah menjadi total. Donut yang bagian-bagiannya tidak menjumlah menjadi
	// keseluruhan tetap tergambar rapi, dan tidak ada galat yang menandainya.
	ringkasan, err := repo.SummarizeDocumentStatus(ctx, inboxoutstanding.Filter{
		AssignedTo:       login,
		AssignedToLegacy: legacy,
	})
	if err != nil {
		print("  [BELUM] ringkasan status dokumen tidak dapat dihitung: %v", err)
	} else {
		// Hanya tab yang BENAR-BENAR dihitung ikut dijumlahkan.
		//
		// Yang jumlahnya nil belum punya kueri, dan memperlakukannya sebagai nol akan
		// membuat pemeriksaan ini lulus karena alasan yang salah.
		jumlah := 0
		for _, s := range ringkasan.Status {
			if s.Count == nil {
				print("            %-28s (belum dihitung)", s.Label)
				continue
			}
			print("            %-28s %d klaim", s.Label, *s.Count)

			// "ALL Case" adalah totalnya sendiri, bukan salah satu bagiannya.
			if s.Status == inboxoutstanding.StatusAll {
				continue
			}
			jumlah += *s.Count
		}
		if jumlah != ringkasan.Total {
			print("  [GAGAL] tab berjumlah %d, total %d — donut akan berbohong",
				jumlah, ringkasan.Total)
		} else {
			print("  [ok]    tab yang terhitung menjumlah tepat menjadi %d", ringkasan.Total)
		}
	}

	for _, c := range page.Claims {
		nomor := c.ClaimNumber
		if nomor == "" {
			nomor = "(belum bernomor)"
		}
		aging := "—"
		if c.AgingDays != nil {
			aging = fmt.Sprintf("%d", *c.AgingDays)
		}
		// Nomor polis dan nama tertanggung SENGAJA tidak dicetak (`D-69`) — keluaran mode
		// periksa sering disalin ke tiket dan percakapan.
		print("            %-16s %-8s panel %-4s aging %-5s %s",
			nomor, c.DisplayStatus(), c.GroupPanel, aging, c.CurrentStage)
	}
}

// checkCloseClaim menjalankan kueri Inbox Close Claim terhadap Oracle sungguhan, lalu
// memeriksa tabel permintaannya secara TERPISAH.
//
// # Kenapa dua pemeriksaan, bukan satu
//
// Keduanya dapat gagal karena sebab yang sama sekali berbeda, dan menyatukannya akan
// menyembunyikan yang kedua:
//
//	kueri daftar     membaca tabel MILIK PEGA yang sudah ada — yang dapat gagal hanyalah
//	                 hak SELECT-nya, atau kuerinya sendiri yang keliru
//	tabel permintaan dibuat migrasi `0006` yang BELUM dijalankan DBA di mana pun, sehingga
//	                 kegagalannya hari ini adalah keadaan yang DIHARAPKAN
//
// Layar tetap berguna meski yang kedua gagal: daftarnya tampil, hanya penanda "permintaan
// terkirim" yang tidak muncul dan kedua tombolnya menjawab galat. Perilaku itu disengaja dan
// diuji (`usecase.ListResult.PendingLookupError`).
func checkCloseClaim(
	ctx context.Context,
	repo *inboxcloseclaimsql.Repo,
	requests *inboxcloseclaimsql.RequestRepo,
	print func(string, ...any),
) {
	// Tanpa penyaring: memeriksa tabelnya, bukan kewenangan seseorang.
	page, err := repo.List(ctx, inboxcloseclaim.Filter{Limit: 5})
	if err != nil {
		print("  [BELUM] Klaim tutup tidak dapat dibaca: %v", err)
		print("            Kuerinya menempuh DATAPEGA.PC_ASM_FW_GCNMFW_WORK,")
		print("            POOLDATA.BUSINESS, POOLDATA.BUSINESSGROUP, POOLDATA.V_STS_CLAIM,")
		print("            dan POOLDATA.T_CLAIM_ADJUSTMENT. Kelimanya milik sistem lama dan")
		print("            tidak dibuat migrasi mana pun — yang kurang hampir pasti hak")
		print("            SELECT atas salah satunya.")
	} else {
		print("  [ok]    Klaim tutup dapat dibaca: %d klaim sudah tutup", page.Total)
		for _, c := range page.Claims {
			nomor := c.ClaimNumber
			if nomor == "" {
				nomor = "(belum bernomor)"
			}
			transfer := "belum transfer"
			if c.TransferredToCashier {
				transfer = "sudah transfer"
			}
			// Nomor polis dan nama tertanggung SENGAJA tidak dicetak (`D-69`) — keluaran
			// mode periksa sering disalin ke tiket dan percakapan.
			print("            %-16s %-7s panel %-4s %-15s %s",
				nomor, c.DisplayStatus(), c.GroupPanel, transfer, c.ClaimStatusLabel)
		}
	}

	if err := requests.CheckTable(ctx); err != nil {
		print("  [BELUM] POOLDATA.CPNC_PERMINTAAN_KLAIM tidak dapat dibaca: %v", err)
		print("            Tabelnya dibuat migrasi 0006, yang BELUM dijalankan di lingkungan")
		print("            mana pun. Sampai itu terjadi: daftar klaim tutup TETAP tampil,")
		print("            tetapi penanda permintaan tidak muncul dan tombol ReOpen maupun")
		print("            Copy Klaim menjawab galat.")
		print("            Migrasi ini dijalankan EMPAT KALI — sekali per portal (D-75).")
		return
	}
	print("  [ok]    POOLDATA.CPNC_PERMINTAAN_KLAIM dapat dibaca")
	print("            Catatan: akun aplikasi hanya perlu SELECT dan INSERT. Hak UPDATE dan")
	print("            DELETE sengaja TIDAK diberikan — yang memindahkan STATUS ke")
	print("            'dijalankan' adalah pelaksana, dengan akunnya sendiri.")
}

// checkRegistration memeriksa modul Registrasi Klaim terhadap Oracle sungguhan.
//
// # Ia TIDAK mendaftarkan klaim
//
// Mode periksa tidak menulis apa pun, dan pendaftaran menulis ke lima tabel sekaligus.
// Yang diperiksa di sini adalah PRASYARATNYA: tabel yang ditulisnya dapat dibaca, dan
// ketiga sumber baca-saja menjawab.
//
// Seam Penugasan sengaja TIDAK dipanggil: `Assign` menaikkan pencacah beban petugas,
// sehingga memeriksanya akan mengubah data — persis yang mode ini janjikan tidak dilakukan.
func checkRegistration(ctx context.Context, primary *sql.DB, print func(string, ...any)) {
	print("")
	print("Registrasi Klaim (B-2)")

	// Tabel yang DITULIS pendaftaran. Urutannya mengikuti urutan penulisannya, supaya
	// yang gagal lebih dulu adalah yang paling awal dibutuhkan.
	//
	// # Kenapa sebagian memeriksa KOLOM, bukan sekadar keberadaan tabel
	//
	// Pemeriksaan keberadaan saja terbukti tidak cukup. T_CLAIM_SPREADING lolos
	// "[ok] dapat dibaca" selama modul menulis kolom yang tidak ada di sana — tabelnya
	// memang ada, hanya bentuknya lain (ORA-00904, 2026-09-26). Yang diperiksa karena itu
	// adalah KONTRAK: kolom yang benar-benar dipakai modul. Satu kolom yang hilang membuat
	// SELECT ini gagal dengan nama kolomnya, jauh sebelum petugas menekan Simpan.
	tabel := []struct {
		nama   string
		sumber string
		kolom  string // kosong berarti cukup keberadaan tabel
	}{
		{"POOLDATA.T_CLAIM_PNC", "tabel warisan, bentuknya ditetapkan Work Owner (diubah 2026-09-26)",
			"CLAIMID, CLAIMNO, PORTAL, NOPOLIS, GROUPPANEL, POLIS_JENIS_BISNIS, POLIS_MATA_UANG, QQNAME, " +
				"BRANCHCODE, DATEOFLOSS, REPORTDATE, RECEIVEDATE, LOCATION, KRONOLOGI, REPORTERNAME, NO_HP, " +
				"REPORTADDRESS, PELAPOR_HUBUNGAN, PELAPOR_HUBUNGAN_LAIN, CURRENCY, NOMOR_SLIK, EXGRATIA, " +
				"PICTEKNIK, RCVID, RCLPUCL, STATUSWORK, STATUSCLAIM, ADMINKLAIM, REGISTERDATE, FLAG_NOLL, " +
				"SOBNAME, SOBNAMEID, BRANCHNAME, BUSINESSCODE, BUSINESSNAME, PRODKE, TYPEOFCOINS, " +
				"COINSNAME, LEADER_MEMBER, SHAREASM, POLISLEADER, " +
				// Migrasi 0012 — wilayah kejadian dan Prinsip Mengenal Nasabah.
				"COUNTRY, COUNTRYID, PROVINCE, PROVINCEID, CITY, CITYID, DISTRICT, DISTRICTID, " +
				"RW, RWID, POSTALCODE, CUSTOMERPRINCIPLE, SUSPICIOUSCOMMENT"},
		{"POOLDATA.T_CLAIM_OBJECTLIST", "migrasi 0008 — 3 kolom tambahan",
			"CLAIMID, OBJECTID, OBJECTNAME, LOKASI, URUTAN, DIHAPUS_PADA"},
		{"POOLDATA.T_CLAIM_OBJECTCOVERAGE", "migrasi 0008 — 3 kolom tambahan",
			"CLAIMID, OBJECTID, OBJECTCOVERAGEID, CAUSEOFLOSSID, SUMTSI, URUTAN_OBJEK, URUTAN, DIHAPUS_PADA, COVERAGENAME"},
		// Sumber objek, coverage, dan spreading saat klaim dibuka — dibaca, tidak pernah ditulis.
		{"POOLDATA.T_PERSONLIST", "tabel polis — objek PA dan Travel", "NOPOLIS, PRODKE, INDEXOBJECT, PYFULLNAME, COVERAGEDATA"},
		{"POOLDATA.T_PROPERTYLIST", "tabel polis — objek Fire", "NOPOLIS, PRODKE, INDEXOBJECT, OBJECTNO, OBJECTNAME, ASMADDRESS, FLAGDELETE, COVERAGELIST, PROPERTYITEMLIST"},
		{"POOLDATA.T_CARGOLIST", "tabel polis — objek Marine Cargo", "NOPOLIS, PRODKE, INDEXOBJECT, GOODSNAME, CONVEYANCENOTE, COVERAGEDATA"},
		{"POOLDATA.T_ANEKALIST", "tabel polis — objek Aneka", "NOPOLIS, PRODKE, INDEXOBJECT, OBJECTNAME, ASMADDRESS, COVERAGELIST"},
		// Tahap Input Estimasi.
		{"POOLDATA.TC_PNC_OBJECTITEM", "Database/CREATE_TABLE_2.sql — item objek",
			"CLAIMID, OBJECTID, OBJECTCOVERAGEID, OBJECTITEMID, OBJECTITEMNAME, DESKRIPSIOBJECT, SUMESTIMATION, DIBUAT_OLEH, DIBUAT_PADA, DIUBAH_OLEH, DIUBAH_PADA, DIHAPUS_OLEH, DIHAPUS_PADA"},
		{"POOLDATA.T_CLAIM_ESTIMASI", "tabel warisan — baris estimasi",
			"CLAIMID, OBJECTID, OBJECTCOVERAGEID, OBJECTITEMID, ESTIMASIID, ESTIMATIONTYPE, KURSID, ESTIMATIONVALUE, KURSVALUE, CONVERTVALUE, ESTIMATIONDATE, DIBUAT_OLEH, DIBUAT_PADA, PRINTFACECLAIM, CFSDATE"},
		// Tombol Download Claim Face Sheet.
		{"POOLDATA.TC_PNC_CFS", "revisi Claim Face Sheet (CFSList)", "CLAIMID, OBJECTID, OBJECTCOVERAGEID, REVISI, CFSDATE, FILENAME"},
		{"POOLDATA.TC_PNC_CFS_ESTIMASI", "reserve per revisi Claim Face Sheet", "CLAIMID, OBJECTID, OBJECTCOVERAGEID, REVISI, URUTAN, CURRENCY, ESTIMATIONDATE, ESTIMATIONVALUE"},
		{"POOLDATA.V_D_CAUSE_OF_LOSS", "master penyebab kerugian — Nature of Loss (dibaca)", "D_COL_ID, DESCRIPTION"},
		{"DATAPEGA.PR_OPERATORS", "operator Pega — nama PIC Admin (dibaca)", "PYUSERIDENTIFIER, PYUSERNAME"},
		{"POOLDATA.M_LOGIN_PNC", "login non-karyawan — nama PIC Admin (dibaca)", "LOGIN_ID, LOGIN_NAME"},
		// Tombol Print PLA.
		{"POOLDATA.T_PLALIST", "PLA yang terbit (ditulis untuk klaim PNCN)", "CLAIMID, OBJECTID, OBJECTCOVERAGEID, NOPLA, NILAIPLA, PLAREINSURER, REVISI, TIPEPLA, TGLPLA, NOTES, REINSCODE, CURRENCYPOLIS, PERCENTPLA, ESTIMASI, ESTIMASISHARE, EMAILPLA, LOGIN, COUNTRY, JSON_PLA"},
		{"POOLDATA.PLA", "log penomoran PLA (PLA_DLA.prc)", "KEY, ID_PLA, KODE, ID_SITE, TAHUN, COUNT"},
		{"POOLDATA.M_SITE_DATABASE", "site aktif pada nomor PLA (dibaca)", "ID, CURRENT_SITE"},
		{"POOLDATA.T_REINSURER", "master penerima PLA (dibaca)", "REINSURERID, REINSURERNAME, LOGIN, EMAIL, COUNTRY"},
		{"POOLDATA.MTTD", "tanda tangan PLA (dibaca)", "ID, NAME, JSONDATA"},
		{"POOLDATA.CURRENCY", "master mata uang", "ID, CURRENCY"},
		{"POOLDATA.V_STS_CLAIM", "master status klaim", "LSC_ID, LSC_NOTE"},
		// Tab Survey, Unggah Dokumen, Progress Claim & Komunikasi — hanya dibaca.
		{"POOLDATA.T_SURVEYORLIST", "tabel warisan — hasil survey (dibaca)",
			"CASEID, PNCCASEID, SURVEYTYPE, SURVEYOR_NAME, SURVEYDATE, LOCATION_SURVEY, OBJECT_NAME, LOCATION_OBJECT, INDEX_SURVEY, STS_SURVEY, KETERANGAN, TGLINPUT"},
		{"POOLDATA.LST_TYPE_DOC_BUSINESS", "master jenis dokumen per bisnis (dibaca)",
			"BUSINESSID, DOCUMENT_TYPE_ID, DOC_TYPE_DT_ID, DETAIL_DOKUMEN, STS_WAJIB, OBJECT_DOC_ID, MIN_DOC"},
		{"POOLDATA.LST_DOC_TYPE", "master jenis induk dokumen (dibaca)", "ID, TYPE_DOCUMENT, JSON_DATA"},
		{"POOLDATA.COVERAGE_DOC_BUSINESS", "master dokumen wajib per coverage PA (dibaca)", "ID, BUSINESSID, COVERAGEID"},
		{"POOLDATA.DATA_ATTACHFILE", "tabel warisan — lampiran (dibaca)",
			"DATAID, ATTACHNAME, ATTACHMIMETYPE, ATTACHNOTE, CATEGORY, SUB_CATEGORY, IMAGEID, INPUTOPERATOR, INPUTDATE, IDPEGA"},
		{"POOLDATA.GCNM_PROGRESS_CLAIM", "tabel warisan — progres klaim (dibaca)",
			"ID_UPDATE, TGL_INPUT, PNCCASEID, STATUS_PROGRESS1, STATUS_PROGRESS2, KETERANGAN, NEXT_FOLLOWUP, USER_INPUT, POSISIID"},
		{"POOLDATA.GCNM_MST_PROGRESS_KLAIM", "master status progres 1 (dibaca)", "ID_PROGRESS, STS_PROGRESS1"},
		{"POOLDATA.GCNM_MST_PROGRESS", "master status progres 2 (dibaca)", "ID_MST, STS_PROGRESS2"},
		{"POOLDATA.M_KOMUNIKASI_PNC", "tabel warisan — komunikasi (dibaca)",
			"CASEID, KOMUNIKASIID, CREATEDDATE, SENDER, SENDERNAME, MESSAGE, REPLYMESSAGE, REPLYFROMNAME, CREATEDATEREPLY, KOMUNIKASISTATUS, CASECLAIM"},
		{"POOLDATA.T_CLAIM_SPREADING", "Database/CREATE_TABLE_2.sql — milik Work Owner, dijalankan DBA (D-63)",
			"CLAIMID, OBJECTID, OBJECTCOVERAGEID, TREATYTYPE, TREATYNAME, SHAREPERCENTAGE, URUTAN"},
		{"POOLDATA.CPNC_TUGAS", "migrasi 0009 — tabel baru", ""},
		{"POOLDATA.CPNC_JEJAK_AUDIT", "migrasi 0009 — tabel baru", ""},
		{"POOLDATA.CPNC_NOTIFIKASI", "migrasi 0011 — tabel baru", ""},
	}
	for _, t := range tabel {
		pilih := "COUNT(*)"
		if t.kolom != "" {
			pilih = t.kolom
		}
		// WHERE 1 = 0 membuktikan tabel dan kolomnya ada tanpa memindai isinya.
		baris, err := primary.QueryContext(ctx,
			"SELECT "+pilih+" FROM "+t.nama+" WHERE 1 = 0")
		if err == nil {
			_ = baris.Close()
		}
		if err != nil {
			print("  [BELUM] %s tidak cocok: %v", t.nama, err)
			print("            Sumbernya %s.", t.sumber)
			print("            Dijalankan EMPAT KALI — sekali per portal (D-75).")
			continue
		}
		if t.kolom != "" {
			print("  [ok]    %s — kolom yang ditulis modul tersedia", t.nama)
			continue
		}
		print("  [ok]    %s dapat dibaca", t.nama)
	}

	// FLAG_NOLL menentukan subjek pemberitahuan berikutnya — biasa atau "(REVISE)".
	// Tanpa kolomnya, setiap pemberitahuan terbaca sebagai yang pertama.
	var abaikan int64
	if err := primary.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM POOLDATA.T_CLAIM_PNC WHERE FLAG_NOLL IS NOT NULL AND 1 = 0").
		Scan(&abaikan); err != nil {
		print("  [BELUM] kolom FLAG_NOLL tidak ada: %v", err)
		print("            Dibuat migrasi 0010. Tanpa kolom ini, Notice of Large Losses")
		print("            kedua atas klaim yang sama tidak dapat ditandai sebagai revisi.")
	} else {
		print("  [ok]    kolom FLAG_NOLL dapat dibaca")
	}

	// Nomor klaim diturunkan dari isi T_CLAIM_PNC, bukan dari tabel pencacah (Work Owner,
	// 2026-09-24). Mencetak nomor berikutnya membuktikan seluruh rantainya berjalan:
	// penyaring tahun, SUBSTR, dan TO_NUMBER.
	var terakhir int64
	pola := fmt.Sprintf("PNCN.%02d.%%", time.Now().Year()%100)
	err := primary.QueryRowContext(ctx,
		`SELECT COALESCE(MAX(TO_NUMBER(SUBSTR(CLAIMNO, 9))), 0)
		   FROM POOLDATA.T_CLAIM_PNC
		  WHERE CLAIMNO LIKE :1`, pola).Scan(&terakhir)
	if err != nil {
		print("  [BELUM] nomor klaim berikutnya tidak dapat dihitung: %v", err)
	} else {
		print("  [ok]    nomor klaim berikutnya: PNCN.%02d.%d",
			time.Now().Year()%100, terakhir+1)
	}

	// Kurs. `D-48` menetapkan kurs yang dipakai adalah kurs TANGGAL KEJADIAN, dan kurs
	// yang tidak ditemukan MENOLAK klaim — tidak ada nilai bawaan.
	//
	// Mata uang disebut dengan KODE ANGKA, bukan simbol ISO: 10001 USD, 10026 IDR.
	// Itulah yang dibawa polis pada $.Currency. Memeriksanya dengan "USD" akan menjawab
	// "tidak ada" dan jawaban itu menyesatkan.
	kurs := registrasisql.NewExchangeRateSource(primary)
	for _, m := range []struct{ kode, nama string }{{"10001", "USD"}, {"10026", "IDR"}} {
		nilai, err := kurs.Find(ctx, m.kode, time.Now())
		if err != nil {
			print("  [BELUM] kurs %s (%s) tidak dapat dibaca: %v", m.nama, m.kode, err)
			print("            Sumbernya POOLDATA.M_CURRENCYSTANDARD. Tanpa kurs pada tanggal")
			print("            kejadian, klaim DITOLAK — itu perilaku yang D-48 tetapkan,")
			print("            bukan cacat. Termasuk klaim rupiah: IDR pun dibaca dari tabel")
			print("            ini, tanpa cabang khusus.")
			continue
		}
		print("  [ok]    kurs %s (%s) terbaca: %.4f", m.nama, m.kode,
			float64(nilai)/float64(registrasi.ExchangeRateOne))
	}

	// Ambang Notice of Large Losses dan penerimanya.
	param := registrasisql.NewParameter(primary)
	if ambang, err := param.LargeLossThreshold(ctx); err != nil {
		print("  [BELUM] ambang Notice of Large Losses tidak dapat dibaca: %v", err)
		print("            Sumbernya POOLDATA.M_PARAMETER baris PNC.AMBANG_KERUGIAN_BESAR.")
	} else {
		print("  [ok]    ambang Notice of Large Losses: Rp %d", int64(ambang)/100)
	}
	penerima, err := param.LargeLossRecipients(ctx, registrasi.LineFire)
	if err != nil && !strings.Contains(err.Error(), "belum diisi") {
		print("  [BELUM] penerima Notice of Large Losses tidak dapat dibaca: %v", err)
	} else if err != nil || len(penerima) == 0 {
		print("  [BELUM] penerima Notice of Large Losses KOSONG")
		print("            Barisnya PNC.PENERIMA_KERUGIAN_BESAR pada POOLDATA.M_PARAMETER")
		print("            belum diisi, dan daftarnya ditunggu dari Work Owner berupa")
		print("            mailbox fungsional — bukan akun pribadi (D-67).")
		print("            Pendaftaran klaim TIDAK terhalang: peristiwanya tetap terbit dan")
		print("            tercatat, hanya tanpa tujuan. Itu mengikuti sistem lama, yang")
		print("            merakit penerima dari lima sumber dan tidak pernah menghentikan")
		print("            registrasi karena salah satunya kosong.")
		print("            Satu sumber lain juga belum ada: email cabang/GL/Pincab, yang")
		print("            di sistem lama datang dari RDB rule GetEmailCabang_SQL —")
		print("            dirujuk 2 activity, TIDAK ADA di export (R-16).")
	} else {
		// Alamatnya SENGAJA tidak dicetak (`D-69`); yang perlu diketahui hanyalah
		// daftarnya sudah terisi.
		print("  [ok]    penerima Notice of Large Losses terisi: %d alamat", len(penerima))
	}

	// Polis dibaca dengan nomor yang pasti tidak ada. Jawaban "tidak ditemukan"
	// membuktikan kuerinya jalan dan JSON_POLIS terbaca, TANPA menyentuh data nasabah
	// mana pun (`D-69`).
	polis := registrasisql.NewPolicyRepo(primary)
	_, err = polis.Get(ctx, "PERIKSA.TIDAK.ADA")
	switch {
	case err == nil:
		print("  [BELUM] POOLDATA.JSON_POLIS menjawab polis untuk nomor yang tidak ada")
	case strings.Contains(err.Error(), "tidak ditemukan"):
		print("  [ok]    POOLDATA.JSON_POLIS dapat dibaca")
	default:
		print("  [BELUM] POOLDATA.JSON_POLIS tidak dapat dibaca: %v", err)
	}

	// Tautan balik ke berkas Receive Document. Yang dilaporkan adalah sebaran posisi
	// berkas MILIK APLIKASI INI — itulah yang berpindah saat tombol Register Klaim
	// ditekan, dan satu-satunya tanda yang dilihat petugas bahwa tombolnya bekerja.
	var belumDiserahkan, belumRegistrasi, outstanding, tanpaTab int64
	err = primary.QueryRowContext(ctx, `
		SELECT SUM(CASE WHEN TRANSFERASM IS NULL AND NOKLAIM IS NULL     THEN 1 ELSE 0 END),
		       SUM(CASE WHEN TRANSFERASM IS NOT NULL AND NOKLAIM IS NULL THEN 1 ELSE 0 END),
		       SUM(CASE WHEN TRANSFERASM IS NOT NULL AND NOKLAIM IS NOT NULL THEN 1 ELSE 0 END),
		       SUM(CASE WHEN TRANSFERASM IS NULL AND NOKLAIM IS NOT NULL THEN 1 ELSE 0 END)
		  FROM POOLDATA.T_CLAIM_RECIVEDCLAIM
		 WHERE CLAIMID LIKE 'RCVN.%'`).
		Scan(&belumDiserahkan, &belumRegistrasi, &outstanding, &tanpaTab)
	if err != nil {
		print("  [BELUM] sebaran posisi berkas laporan tidak dapat dibaca: %v", err)
	} else {
		print("  [ok]    berkas laporan milik aplikasi ini: %d Not Transferred · %d Not Registered · %d Outstanding",
			belumDiserahkan, belumRegistrasi, outstanding)
		if tanpaTab > 0 {
			// Kombinasi ini tidak dikembalikan kueri posisi mana pun, sehingga berkasnya
			// tidak muncul di tab apa pun. Ia hanya dapat lahir bila NOKLAIM dipasang
			// sebelum TRANSFERASM — urutan yang dijaga reportlink.sql.
			print("  [BELUM] %d berkas ber-NOKLAIM tetapi belum diserahkan — TIDAK muncul", tanpaTab)
			print("            di tab mana pun. Urutan penulisannya terbalik.")
		}
	}

	// Persetujuan / Akseptasi LOD menulis tujuh kolom baru T_CLAIM_ADJUSTMENT. Tanpanya klaim
	// tetap dapat dimuat (kolomnya dibaca terpisah), tetapi Simpan akseptasi gagal.
	if _, err := primary.ExecContext(ctx, `
		SELECT TANGGALBOLEHBAYAR, RECEIVEDATEANALIST, ACCEPTANCEVALUELOD, TIPEAKSEPTASI,
		       KOMITEACCEPTED, REMARKACCEPTED, UPLOADNOTELOD
		  FROM POOLDATA.T_CLAIM_ADJUSTMENT
		 WHERE 1 = 0`); err != nil {
		print("  [BELUM] kolom isian akseptasi belum ada di T_CLAIM_ADJUSTMENT: %v", err)
		print("            Jalankan migrations/0013_akseptasi_lod.up.sql (DBA, D-63). Sampai itu,")
		print("            tombol Simpan Persetujuan / Akseptasi gagal; klaim tetap dapat dibuka.")
	} else {
		print("  [ok]    kolom isian akseptasi (migrasi 0013) ada di T_CLAIM_ADJUSTMENT")
	}

	print("            Seam Penugasan tidak diperiksa di sini: memanggilnya menaikkan")
	print("            pencacah beban petugas, dan mode ini tidak menulis apa pun.")
}

// checkBandingHargaSalvage memeriksa kesiapan modul Inbox Banding Harga Salvage.
//
// # Kenapa modul ini PERLU diperiksa, sementara modul baca lain sering tidak
//
// Karena DDL tabel intinya belum pernah dibaca. Seluruh nama kolom
// `POOLDATA.T_CLAIM_CHEKER_SALVAGE` disimpulkan dari teks kueri Pega, dan SATU di antaranya
// — `NOTEKOMITE` — bahkan disimpulkan dari nama properti gridnya, bukan dari kueri mana pun.
//
// Kolom yang ternyata tidak ada akan menggagalkan SELURUH layar, bukan mengosongkan satu
// kolom. Pemeriksaan di sini membuat kekeliruan itu terbaca saat aplikasi start, lengkap
// dengan nama kolom yang salah — bukan sebagai layar galat yang dilaporkan pengguna.
//
// Satu nama memang PERNAH keliru: kolom catatan komite sempat ditulis `NOTEKOMITE`,
// disimpulkan dari nama properti gridnya, sampai `UpdateDataReqSalvage` membuktikan namanya
// `NOTEAPPROVE`.
//
// # Sejak modul ini MENULIS, pemeriksaannya bertambah
//
// Tombol Approve dan Reject menyentuh dua tabel LAIN — `SALAVAGEDOCUMENT` dan
// `DETAIL_PNC_SALVAGE` — dan nama kolom keduanya pun disimpulkan dari teks kueri Pega. Kolom
// yang ternyata bernama lain menggagalkan seluruh transaksi keputusan, dan itu baru ketahuan
// ketika seorang komite menekan Simpan atas putusan yang sudah ia pertimbangkan.
//
// Ia tetap tidak menulis apa pun dan tidak membaca satu baris pun: yang dibaca adalah katalog
// kolom. Menguji hak tulis dengan benar-benar menulis akan meninggalkan baris percobaan di
// tabel produksi.
func checkBandingHargaSalvage(
	ctx context.Context,
	repo *inboxbandinghargasalvagesql.Repo,
	print func(string, ...any),
) {
	if err := repo.CheckTable(ctx); err != nil {
		print("  [BELUM] POOLDATA.T_CLAIM_CHEKER_SALVAGE tidak terbaca utuh: %v", err)
		print("            Nama kolom modul ini disimpulkan dari kueri Pega, bukan dari DDL")
		print("            — yang belum pernah diterima. Bila kolomnya bernama lain, yang")
		print("            disesuaikan adalah kuerinya, bukan tabelnya.")
		return
	}
	print("  [ok]    POOLDATA.T_CLAIM_CHEKER_SALVAGE punya kedua belas kolom yang dibaca")

	// KEDUA tab dijalankan, bukan satu: keduanya membaca tabel yang BERBEDA —
	// T_CLAIM_CHEKER_SALVAGE dan PNC_SALVAGE — sehingga satu kueri yang berhasil tidak
	// menyatakan apa pun tentang yang lain.
	caller := inboxbandinghargasalvage.Caller{Login: "pemeriksa-kesiapan"}
	page := inboxbandinghargasalvage.Pagination{Page: 1, Size: 5}

	for _, tab := range inboxbandinghargasalvage.Tabs() {
		query, err := inboxbandinghargasalvage.NewQuery(
			inboxbandinghargasalvage.QueryInput{Tab: tab.Code}, caller)
		if err != nil {
			print("  [BELUM] Tab %q tidak dapat disusun: %v", tab.Code, err)
			continue
		}

		if _, err := repo.List(ctx, query, page); err != nil {
			print("  [BELUM] Kueri tab %q gagal: %v", tab.Name, err)
			continue
		}
		print("  [ok]    Kueri tab %q berjalan", tab.Name)
	}

	print("            Nol baris BUKAN kegagalan: kedua tab menyaring menurut NAMA KOMITE,")
	print("            dan akun pemeriksa bukan komite mana pun.")

	// Tabel tujuan PENULISAN diperiksa terpisah, dan kegagalannya tidak menghentikan
	// pemeriksaan di atas: layar ini tetap dapat dibaca meski tombolnya tidak dapat dipakai.
	found, err := repo.CheckWriteTargets(ctx)
	switch {
	case err != nil:
		print("  [BELUM] Katalog kolom tabel tujuan penulisan tidak terbaca: %v", err)
		print("            Periksa hak SELECT akun aplikasi atas ALL_TAB_COLUMNS.")
	case found < inboxbandinghargasalvagesql.WriteTargetColumns:
		print("  [BELUM] Tabel tujuan penulisan hanya punya %d dari %d kolom yang ditulis",
			found, inboxbandinghargasalvagesql.WriteTargetColumns)
		print("            POOLDATA.SALAVAGEDOCUMENT   IDBALAILELANG, NOKLAIM, TIPEDOCSALVAGE")
		print("            POOLDATA.DETAIL_PNC_SALVAGE HARGAITEM, IDSALVAGE, IDDETAILSALVAGE")
		print("            Tombol Approve/Reject akan gagal SELURUHNYA saat ditekan, karena")
		print("            keempat pernyataannya berjalan dalam satu transaksi.")
	default:
		print("  [ok]    Kedua tabel tujuan penulisan punya keenam kolom yang ditulis")
		print("            Hak INSERT/UPDATE-nya TIDAK diuji di sini — mengujinya berarti")
		print("            menulis baris percobaan ke tabel produksi. Pastikan ke DBA bahwa")
		print("            akun aplikasi punya UPDATE atas ketiga tabelnya.")
	}
}

// checkSalvage memeriksa kesiapan modul Inbox Salvage.
//
// # Kenapa pemeriksaannya BERBEDA dari modul inbox lain
//
// Karena modul ini MENULIS. Modul inbox lain hanya membutuhkan hak SELECT; modul ini
// membutuhkan SELECT, INSERT, dan UPDATE atas dua tabel — dan hak yang kurang baru
// ketahuan saat petugas menekan Submit, yakni pada saat yang paling buruk.
//
// Pemeriksaan di sini TIDAK menulis apa pun. Ia membaca katalog kolom dan menjalankan
// ketiga keluarga kueri, lalu menyatakan apa yang harus diperiksa DBA bila salah satunya
// gagal. Menguji hak tulis dengan benar-benar menulis akan meninggalkan baris percobaan di
// tabel produksi.
func checkSalvage(
	ctx context.Context,
	repo *inboxsalvagesql.Repo,
	print func(string, ...any),
) {
	found, err := repo.Ready(ctx)
	if err != nil {
		print("  [BELUM] Katalog kolom Inbox Salvage tidak dapat dibaca: %v", err)
		print("            Periksa hak SELECT akun aplikasi atas ALL_TAB_COLUMNS.")
		return
	}

	// Sembilan kolom diperiksa — seluruh kolom yang dibaca grid keluarga C.
	const wantedColumns = 9
	if found < wantedColumns {
		print("  [BELUM] POOLDATA.PNC_SALVAGE hanya punya %d dari %d kolom yang dibaca",
			found, wantedColumns)
		print("            Nama kolom modul ini dibaca dari kueri Pega, bukan dari DDL —")
		print("            yang belum pernah diterima (`R-08`). Bila kolomnya memang")
		print("            bernama lain, kuerinya yang harus disesuaikan, bukan tabelnya.")
		return
	}
	print("  [ok]    POOLDATA.PNC_SALVAGE punya kesembilan kolom yang dibaca grid")

	caller := inboxsalvage.Caller{Login: "pemeriksa-kesiapan"}
	page := inboxsalvage.Pagination{Page: 1, Size: 5}

	// KETIGA keluarga kueri dijalankan, bukan satu.
	//
	// Ketiganya membaca TABEL YANG BERBEDA — keluarga A dan B membaca T_CLAIM_PNC,
	// keluarga C membaca PNC_SALVAGE beserta agregat DETAIL_PNC_SALVAGE — sehingga satu
	// kueri yang berhasil tidak menyatakan apa pun tentang dua lainnya.
	perFamily := map[inboxsalvage.Family]int{}
	failed := false

	// ID satu pengajuan sungguhan, dan satu nomor klaim sungguhan — keduanya dipakai
	// menguji kedua jalur panel Detail di bawah.
	sampleID := ""
	sampleClaim := ""

	for _, tab := range inboxsalvage.Tabs() {
		// Cukup SATU tab per keluarga: yang berbeda antartab di dalam satu keluarga
		// hanyalah nilai penyaringnya, bukan bentuk kuerinya.
		if _, already := perFamily[tab.Family]; already {
			continue
		}

		query, err := inboxsalvage.NewQuery(
			inboxsalvage.QueryInput{Tab: tab.Code}, caller)
		if err != nil {
			print("  [BELUM] Daftar %q tidak dapat disusun: %v", tab.Name, err)
			failed = true
			continue
		}

		result, err := repo.List(ctx, query, page)
		if err != nil {
			print("  [BELUM] Daftar %q tidak dapat dibaca: %v", tab.Name, err)
			failed = true
			continue
		}
		perFamily[tab.Family] = result.Total

		if tab.Family == inboxsalvage.FamilySalvage && sampleID == "" {
			for _, row := range result.Items {
				if row.SalvageID != "" {
					sampleID = row.SalvageID
					break
				}
			}
		}

		if tab.Family != inboxsalvage.FamilySalvage && sampleClaim == "" {
			for _, row := range result.Items {
				if row.ClaimNo != "" {
					sampleClaim = row.ClaimNo
					break
				}
			}
		}
	}

	if !failed {
		print("  [ok]    Ketiga keluarga kueri Inbox Salvage dapat dijalankan")
		for family, total := range perFamily {
			print("            keluarga %-12s %d baris", family, total)
		}
	}

	// Kueri panel Detail diuji TERSENDIRI, dan alasannya bukan kelengkapan.
	//
	// Keduanya membaca tabel yang sama dengan daftar tetapi dengan bentuk yang berbeda —
	// satu baris tunggal dengan dua puluh delapan kolom, dan agregat atas
	// DETAIL_PNC_SALVAGE. Daftar yang berhasil dibaca tidak menyatakan apa pun tentang
	// keduanya; cacat tipe data pada salah satu kolom yang HANYA dibaca panel ini tidak
	// akan pernah muncul saat daftarnya dibuka.
	switch {
	case sampleID == "":
		print("  [lewat] Panel Detail Salvage tidak diuji — tidak ada satu pun pengajuan")
		print("            pada lima baris pertama keluarga salvage. Bukan kegagalan;")
		print("            kuerinya belum pernah dijalankan terhadap basis data ini.")

	default:
		detail, err := repo.Detail(ctx, sampleID)
		switch {
		case err != nil:
			print("  [BELUM] Panel Detail Salvage tidak dapat dibaca: %v", err)
			print("            Kueri kepala panel membaca DUA PULUH DELAPAN kolom, di")
			print("            antaranya kolom yang tidak dibaca daftar mana pun.")

		default:
			print("  [ok]    Panel Detail Salvage dapat dibaca (kepala dan grid barang)")

			// Riwayat pengajuan ini WAJIB memuat dirinya sendiri. Nol baris di sini
			// berarti kuerinya berjalan tetapi tidak menemukan apa-apa — dan itu
			// kegagalan yang diam, bukan keberhasilan.
			if len(detail.History) == 0 {
				print("  [BELUM] Riwayat pengajuan ini KOSONG, padahal pengajuannya ada.")
				print("            Kueri riwayat berjalan tetapi tidak menemukan barisnya")
				print("            sendiri — periksa nama kolom NOKLAIM di PNC_SALVAGE.")
			} else {
				print("  [ok]    Riwayat pengajuan terbaca (%d baris, termasuk dirinya)",
					len(detail.History))
			}
		}
	}

	// Jalur KEDUA panel: dibuka dari baris yang berupa klaim.
	//
	// Diuji tersendiri karena ia menempuh dua kueri yang TIDAK dipakai jalur pertama —
	// pencarian pengajuan terakhir milik klaim, dan pembacaan kepala klaimnya. Keduanya
	// menyentuh tabel yang berbeda pula: `DETAIL_PNC_SALVAGE` dan `T_CLAIM_PNC`.
	switch {
	case sampleClaim == "":
		print("  [lewat] Panel Detail lewat nomor klaim tidak diuji — tidak ada satu pun")
		print("            baris pada lima baris pertama keluarga berbasis klaim.")

	default:
		detail, err := repo.DetailByClaim(ctx, sampleClaim)
		switch {
		case err != nil:
			print("  [BELUM] Panel Detail lewat nomor klaim tidak dapat dibaca: %v", err)
			print("            Ia menempuh POOLDATA.DETAIL_PNC_SALVAGE dan")
			print("            POOLDATA.T_CLAIM_PNC, bukan hanya POOLDATA.PNC_SALVAGE.")

		case detail.HasSubmission:
			print("  [ok]    Panel Detail lewat nomor klaim dapat dibaca (ada pengajuan)")

		default:
			print("  [ok]    Panel Detail lewat nomor klaim dapat dibaca")
			print("            Klaim contoh belum punya pengajuan salvage — itu keadaan")
			print("            yang SAH: klaimnya masuk ke form \"Menambahkan Data")
			print("            Salvage\", bukan ke panel rincian.")
		}

		if err == nil {
			print("  [ok]    Grid \"Detail History Salvage\" dapat dibaca (%d baris)",
				len(detail.History))
		}
	}

	counts, err := repo.Counts(ctx, caller)
	if err != nil {
		print("  [BELUM] Tabel ringkas Inbox Salvage tidak dapat dihitung: %v", err)
		return
	}
	print("  [ok]    Tabel ringkas Inbox Salvage dapat dihitung (%d baris)", len(counts))

	print("            CATATAN — angka ringkas TIDAK selalu sama dengan jumlah baris")
	print("            daftarnya. Tiga baris menghitung populasi yang BERBEDA dari daftar")
	print("            yang dibukanya, dan itu keadaan di Pega yang sengaja direplikasi")
	print("            (`P-5`). Lihat inboxsalvage.CountRows.")
	print("            HAK TULIS TIDAK DIUJI di sini — mengujinya berarti menulis baris")
	print("            percobaan. Akun aplikasi membutuhkan INSERT dan UPDATE atas")
	print("            POOLDATA.PNC_SALVAGE dan INSERT atas POOLDATA.DETAIL_PNC_SALVAGE.")
}

// checkReportKPI memeriksa kesiapan tab KPI Adjuster pada Report KPI PNC.
//
// # Apa yang benar-benar dijawab pemeriksaan ini
//
// Tiga hal, dan ketiganya adalah hal yang TIDAK dapat diketahui dari kode:
//
//  1. Tabelnya ada dan akun aplikasi boleh membacanya. Modul ini tidak menuntut satu pun
//     migrasi — `POOLDATA.DETAIL_KPI_ADJUSTER` sudah ada dan diisi Pega — sehingga
//     kegagalan di sini hampir pasti soal hak SELECT, bukan soal objek yang belum dibuat.
//
//  2. Tabelnya BERISI. Tabel kosong bukan kegagalan: Pega baru mengisinya ketika seseorang
//     membuka tab KPI Adjuster di sana. Tetapi ia perlu DISEBUT, karena layar yang kosong
//     dengan tabel kosong dan layar yang kosong karena penyaringnya keliru terlihat sama
//     persis bagi pengguna.
//
//  3. Nilai `TIPE` yang benar-benar ada. Kedua tipe yang dikenal modul ini — OUTSTANDING
//     dan FINAL — dibaca dari LITERAL di dalam `GetSummaryKPIAdjusterALL-SQL.xml`, bukan
//     dari master mana pun. Bila produksi memuat nilai ketiga, dropdown tidak akan pernah
//     menampilkannya dan barisnya tidak akan pernah terlihat — tanpa satu pun galat.
//
// Butir ketiga itulah alasan utama fungsi ini ada.
func checkReportKPI(
	ctx context.Context,
	repo *reportkpisql.Repo,
	print func(string, ...any),
) {
	state, err := repo.CheckSource(ctx)
	if err != nil {
		print("  [BELUM] Report KPI (tab KPI Adjuster) tidak dapat dibaca: %v", err)
		print("            Modul ini TIDAK menuntut migrasi — %s sudah ada dan diisi Pega.",
			reportkpi.SourceTable)
		print("            Periksa hak SELECT akun aplikasi atas tabel itu.")
		return
	}

	print("  [ok]    %s dapat dibaca: %d baris", reportkpi.SourceTable, state.Rows)

	if state.Rows == 0 {
		print("  [PERIKSA] Tabelnya KOSONG, dan itu belum tentu keliru.")
		print("            Barisnya ditulis Pega saat seseorang membuka tab KPI Adjuster")
		print("            di sana; entitas yang belum pernah memakainya memang kosong.")
		print("            Yang perlu dipastikan: apakah entitas ini memang belum pernah")
		print("            memakai layar itu — bukan bahwa tabelnya salah.")
		return
	}

	// Nilai TIPE dibandingkan dengan yang dikenal modul. Yang tidak dikenal disebut satu
	// per satu, karena satu nilai asing berarti ada baris yang tidak akan pernah terlihat.
	known := map[string]bool{}
	for _, t := range reportkpi.ReportTypes() {
		known[string(t.Code)] = true
	}

	var unknown []string
	for _, value := range state.Types {
		if !known[value] {
			unknown = append(unknown, value)
		}
	}

	print("  [ok]    Nilai TIPE di basis data: %s", strings.Join(state.Types, ", "))

	if len(unknown) > 0 {
		print("  [PERIKSA] %d nilai TIPE TIDAK dikenal modul: %s",
			len(unknown), strings.Join(unknown, ", "))
		print("            Kedua tipe yang dikenal (OUTSTANDING, FINAL) dibaca dari literal")
		print("            di dalam GetSummaryKPIAdjusterALL-SQL.xml, bukan dari master.")
		print("            Baris ber-tipe di atas TIDAK akan pernah terlihat di layar,")
		print("            dan tidak ada galat yang menandakannya. Tambahkan tipenya di")
		print("            internal/reportkpi/component.go setelah artinya dipastikan.")
		return
	}

	print("            Seluruhnya dikenal modul, sehingga tidak ada baris yang tersembunyi.")
	print("            Catatan: modul ini MEMBACA saja. Yang mengisi tabel itu adalah")
	print("            Pega lewat INSERT_KPIADJUSTER; selama masa paralel tepat satu")
	print("            sistem yang boleh menulisnya (P-1).")
}

// checkReportKlaim memeriksa kesiapan modul Report Klaim (28 panel ekspor).
//
// # Apa yang benar-benar dijawab pemeriksaan ini
//
// Dua hal, dan keduanya adalah keadaan yang TIDAK terlihat sebagai kegagalan di layar:
//
//  1. Koneksi KEDUA portal (`ANEKA_<PORTAL_ALIAS>_*`) terpasang dan kalender liburnya
//     dapat dibaca. Tanpanya, kolom hari kerja pada Report TAT ditandai tidak diketahui —
//     bukan salah, tetapi juga bukan angka yang dapat dipakai menilai kinerja. Berkasnya
//     tetap terunduh, dan tidak ada satu pun pesan galat yang muncul.
//
//  2. Laporan mana yang kuerinya BELUM selesai dipindahkan dari export. Ia disebutkan
//     per lini bisnis, karena laporan yang sama berjalan normal pada lini lain — dan
//     menyebut nama laporannya saja akan terbaca seolah seluruh laporan itu mati.
//
// Butir kedua dihitung dari rencana dan berkas .sql yang ada, bukan dari daftar tulisan
// tangan, supaya kemajuan pemindahan ikut terbaca tanpa menyunting berkas ini.
func checkReportKlaim(
	ctx context.Context,
	repo *reportklaimsql.Repo,
	print func(string, ...any),
) {
	state := repo.CheckSource(ctx)

	switch {
	case !state.SecondConnection:
		print("  [PERIKSA] Report Klaim: koneksi KEDUA portal tidak terpasang.")
		print("            Isi blok ANEKA_<PORTAL_ALIAS>_* di .env bila kolom hari kerja")
		print("            pada Report TAT memang harus terisi. Tanpanya laporan tetap")
		print("            terunduh, dan kolom itu ditandai tidak diketahui (R-03).")
	case state.HolidayError != nil:
		print("  [PERIKSA] Report Klaim: kalender libur tidak dapat dibaca: %v", state.HolidayError)
		print("            Koneksi keduanya hidup, jadi ini hampir pasti soal hak SELECT")
		print("            akun aplikasi atas GENERAL.HRD_LBR — bukan objek yang belum ada.")
	case state.HolidayDays == 0:
		print("  [PERIKSA] Report Klaim: kalender libur %d KOSONG.", state.HolidayYear)
		print("            Perhitungan hari kerja akan memotong akhir pekan saja, dan")
		print("            hasilnya terlihat wajar. Pastikan tahun berjalan memang belum")
		print("            diisi, bukan bahwa tabelnya salah.")
	default:
		print("  [ok]    Report Klaim: kalender libur %d dapat dibaca: %d hari",
			state.HolidayYear, state.HolidayDays)
	}

	belum := reportklaimsql.NotPorted()
	if len(belum) == 0 {
		print("  [ok]    Seluruh kueri laporan sudah dipindahkan dari export.")
		return
	}

	print("  [PERIKSA] %d kombinasi laporan x lini bisnis kuerinya BELUM dipindahkan:", len(belum))
	for _, item := range belum {
		print("            %-28s lini %-4s (%s)", item.Report, item.BusinessLine, item.Query)
	}
	print("            Permintaannya DITOLAK dengan sebab yang menyebut keduanya, bukan")
	print("            dijawab berkas kosong. Lini lain pada laporan yang sama tetap jalan.")
}

// checkReportKPIPICTeknik memeriksa kesiapan tab KPI PIC Teknik.
//
// # Kenapa pemeriksaan ini terpisah dari checkReportKPI
//
// Karena tabnya membaca tabel yang BERBEDA. Tab KPI Adjuster membaca penilaian yang sudah
// jadi; tab ini menghitungnya sendiri, dan SELURUH nilainya berasal dari tangga
// `POOLDATA.M_KPI_PNC`. Satu tangga yang hilang membuat seluruh kolom Nilai kosong — tanpa
// satu pun galat.
//
// # Yang dijawabnya, dan tidak dapat diketahui dari kode
//
//  1. Keempat tangga ada dan berisi.
//  2. ARAH tiap tangga. Dua di antaranya tersusun MENURUN di produksi hari ini, dan itulah
//     sebab nilai tertinggi justru diberikan kepada persentase terkecil. Arahnya dibaca dari
//     DATA — kalau kelak isinya diperbaiki, pemeriksaan ini yang pertama menunjukkannya.
//  3. Lubang dan tumpang tindih antar pita. Tumpang tindih di titik batas ADA hari ini dan
//     itu yang membuat hasil Pega di sana bergantung urutan baris.
func checkReportKPIPICTeknik(
	ctx context.Context,
	repo *reportkpisql.Repo,
	print func(string, ...any),
) {
	jobs := []struct {
		job      string
		expected bool
	}{
		{reportkpi.JobProgress, true},
		{reportkpi.JobAnalysis, false},
		{reportkpi.JobAcceptance, false},
		{reportkpi.JobSLA, true},
	}

	for _, item := range jobs {
		state, err := repo.CheckBands(ctx, item.job)
		if err != nil {
			print("  [BELUM] Tangga nilai %q tidak dapat dibaca: %v", item.job, err)
			print("            Seluruh kolom Nilai tab KPI PIC Teknik berasal dari %s.",
				reportkpi.BandTable)
			print("            Periksa hak SELECT akun aplikasi atas tabel itu.")
			return
		}

		if state.Bands == 0 {
			print("  [PERIKSA] Tangga nilai %q KOSONG di entitas ini.", item.job)
			print("            Komponen itu akan tampil tanpa nilai — bukan bernilai nol.")
			continue
		}

		arah := "menaik"
		if state.Descending {
			arah = "MENURUN"
		}
		print("  [ok]    Tangga %q: %d pita, tersusun %s", item.job, state.Bands, arah)

		// Arah yang BERBEDA dari yang direplikasi modul adalah kabar penting, bukan galat.
		// Ia berarti isi tabelnya sudah diperbaiki di produksi, dan replikasi cacatnya di
		// modul ini menjadi selisih yang harus dicabut.
		if state.Descending != item.expected {
			print("  [PERIKSA] Arahnya BERBEDA dari yang direplikasi modul.")
			print("            Modul mereplikasi keadaan Pega saat dibaca: %q tersusun %s.",
				item.job, arahDari(item.expected))
			print("            Bila isi tabelnya sudah diperbaiki, butir selisih terencana")
			print("            tentang tangga terbalik perlu dicabut — lihat")
			print("            internal/reportkpi/screen.go.")
		}

		for _, gap := range state.Gaps {
			print("  [PERIKSA] Tangga %q: %s", item.job, gap)
		}
	}

	print("            Catatan: tumpang tindih di titik batas MEMANG ada di produksi, dan")
	print("            itu sebab hasil Pega di sana bergantung urutan baris. Modul ini")
	print("            membacanya terurut ID dan memakai yang pertama cocok.")
}

// arahDari menggambar arah tangga sebagai kata.
func arahDari(descending bool) string {
	if descending {
		return "MENURUN"
	}
	return "menaik"
}

// checkRejection melaporkan kesiapan ketiga tabel Master Penolakan Klaim.
//
// Ketiganya tabel WARISAN — tidak satu pun dibuat migrasi aplikasi ini, sehingga "tidak
// dapat dibaca" di sini berarti tabelnya memang tidak ada di entitas itu, atau akun
// aplikasi belum diberi hak bacanya. Keduanya urusan DBA, bukan urusan migrasi yang
// tertinggal — itulah sebabnya pesannya berbeda dari checkClaimStatus.
//
// Ia melaporkan JUMLAH BARIS pada kedua tabel yang dibaca layar. Isi tabelnya tidak ikut
// dikirim bersama export (tidak ada CSV-nya di `Database/`), sehingga angka inilah
// satu-satunya cara mengetahui seberapa banyak data yang sebenarnya ada sebelum layarnya
// dibuka.
func checkRejection(ctx context.Context, primary *sql.DB, print func(string, ...any)) {
	repo := masterpenolakansql.NewRepo(primary)
	if err := repo.CheckTable(ctx); err != nil {
		print("  [BELUM] tabel Master Penolakan Klaim belum dapat dibaca: %v", err)
		print("            POOLDATA.MST_PENOLAKAN_KLAIM_1 dan _2 adalah tabel warisan Pega;")
		print("            keduanya TIDAK dibuat migrasi aplikasi ini. Mintakan hak bacanya ke DBA.")
	} else {
		parent, errParent := repo.ListParent(ctx)
		rows, errRows := repo.List(ctx)
		switch {
		case errParent != nil:
			print("  [GAGAL] POOLDATA.MST_PENOLAKAN_KLAIM_1 tidak dapat dibaca isinya: %v", errParent)
		case errRows != nil:
			print("  [GAGAL] POOLDATA.MST_PENOLAKAN_KLAIM_2 tidak dapat dibaca isinya: %v", errRows)
		default:
			print("  [ok]    Master Penolakan Klaim: %d status penolakan 1, %d status penolakan 2",
				len(parent), len(rows))
			if duplicate := countDuplicateNames(parent); duplicate > 0 {
				// Akibat langsung cacat MASTERPENOLAKANKLAIM1 yang menyisipkan baris baru
				// pada setiap simpan. Angkanya dilaporkan supaya besarnya terlihat sebelum
				// ada yang memutuskan apa yang harus dilakukan terhadap baris-baris itu.
				print("  [WASPADA] %d nama Status Penolakan 1 kembar — bekas cacat MASTERPENOLAKANKLAIM1", duplicate)
				print("            Penambahan baru tidak lagi menerbitkan kembar; baris lama dibiarkan.")
			}
		}
	}

	komite := masterpenolakansql.NewRepoKomite(primary)
	if err := komite.CheckTable(ctx); err != nil {
		print("  [BELUM] POOLDATA.MST_REJECTED_KOMITE belum dapat dibaca: %v", err)
		return
	}
	list, err := komite.List(ctx)
	if err != nil {
		print("  [GAGAL] POOLDATA.MST_REJECTED_KOMITE tidak dapat dibaca isinya: %v", err)
		return
	}
	print("  [ok]    POOLDATA.MST_REJECTED_KOMITE dapat dibaca: %d catatan", len(list))
}

// checkPasal melaporkan kesiapan POOLDATA.V_M_DATA_PASAL beserta master lini bisnisnya.
//
// Tabelnya warisan Pega dan TIDAK dibuat migrasi aplikasi ini, sehingga "belum dapat
// dibaca" di sini berarti tabelnya memang tidak ada di entitas itu, atau akun aplikasi
// belum diberi hak bacanya — keduanya urusan DBA.
//
// # Empat hal dilaporkan, dan yang KEDUA adalah alasan utama fungsi ini ada
//
//  1. Jumlah baris — apakah layarnya akan terisi.
//  2. **Berapa baris yang dokumen JSON-nya tidak dapat diurai.** Seluruh isi pasal selain
//     No Pasal hidup di dalam satu kolom CLOB yang ditulis Pega, dan bentuknya tidak
//     pernah dapat dibaca dari export — hanya diturunkan dari `json_value` pada kueri
//     lama. Satu baris yang tidak terurai akan menggagalkan pembacaan daftar di layar,
//     dan mode periksa ini satu-satunya cara mengetahuinya SEBELUM pengguna membukanya.
//  3. Sebaran kategori — apakah kode selain "1" dan "2" benar-benar ada di data nyata.
//     Bila ada, kodenya harus dibawa ke Work Owner: daftar pilihan dropdown-nya tidak ada
//     di export (`R-16`), dan "Notifikasi" hari ini diperlakukan sebagai cabang `else`.
//  4. Master lini bisnis, yang tanpanya isian Bisnis tidak dapat diisi.
func checkPasal(ctx context.Context, primary *sql.DB, print func(string, ...any)) {
	repo := masterpasalsql.NewRepo(primary)

	if err := repo.CheckTable(ctx); err != nil {
		print("  [BELUM] POOLDATA.V_M_DATA_PASAL belum dapat dibaca: %v", err)
		print("            Tabelnya warisan Pega dan TIDAK dibuat migrasi aplikasi ini.")
		print("            Mintakan hak bacanya ke DBA.")
		return
	}

	list, err := repo.List(ctx)
	if err != nil {
		// Kegagalan di sini hampir pasti berarti satu baris dokumennya rusak; galatnya
		// menyebut IDDATA-nya. Itu justru temuan yang dicari fungsi ini.
		print("  [GAGAL] POOLDATA.V_M_DATA_PASAL tidak dapat dibaca: %v", err)
		print("            Bila galatnya menyebut sebuah IDDATA, dokumen JSON baris itu rusak")
		print("            dan HARUS diperbaiki sebelum layar Master Pasal Kerugian dipakai.")
		return
	}

	unknownCategory := map[string]int{}
	for _, clause := range list {
		switch clause.Category {
		case masterpasal.CategoryPolicyCoverage, masterpasal.CategoryException, masterpasal.CategoryNotification:
		default:
			unknownCategory[clause.Category]++
		}
	}

	print("  [ok]    Master Pasal Kerugian: %d pasal terbaca, seluruh dokumen JSON-nya utuh", len(list))

	for code, count := range unknownCategory {
		print("  [WASPADA] %d pasal berkategori %q, kode yang TIDAK dikenal", count, code)
		print("            Ketiga pilihan yang diketahui hanya \"1\", \"2\", dan kosong; daftar")
		print("            pilihan aslinya tidak ada di export (R-16). Bawa kode ini ke Work")
		print("            Owner sebelum layarnya dipakai menyunting baris tersebut.")
	}

	if err := repo.CheckBusinessTable(ctx); err != nil {
		print("  [BELUM] POOLDATA.BUSINESS belum dapat dibaca: %v", err)
		print("            Tanpa itu isian Bisnis pada form tidak dapat diisi.")
		return
	}
	print("  [ok]    POOLDATA.BUSINESS terbaca; isian Bisnis dapat diisi")
}

// checkBengkel melaporkan kesiapan POOLDATA.BENGKEL_HE beserta acuannya.
//
// Tabelnya warisan Pega dan TIDAK dibuat migrasi aplikasi ini, sehingga "belum dapat
// dibaca" di sini berarti tabelnya memang tidak ada di entitas itu, atau akun aplikasi
// belum diberi hak bacanya — keduanya urusan DBA.
//
// # Lima hal dilaporkan, dan yang KETIGA adalah alasan utama fungsi ini ada
//
//  1. Jumlah baris per status — apakah ketiga tab layar akan terisi.
//  2. Ketersediaan penomoran: kode situs dan sequence. Tanpa keduanya, penambahan gagal
//     pada permintaan pertama, dan gagalnya baru terlihat saat petugas menekan Simpan.
//  3. **Apakah BENGKEL_HE dan M_BENGKEL_HE satu sumber atau dua.** Sistem lama MENULIS
//     dokumen JSON ke M_BENGKEL_HE dan MEMBACA kolom dari BENGKEL_HE; modul ini menulis
//     kolom langsung ke BENGKEL_HE. Bila keduanya ternyata dua tabel terpisah yang
//     disinkronkan, penulisan di sini tidak akan pernah sampai ke Pega — dan Pega tidak
//     akan pernah sampai ke sini. Perbandingan jumlah baris adalah cara termurah
//     mengetahuinya tanpa DDL (R-08). Rinciannya di banner masterbengkel.sql.
//  4. Ketiga tabel acuan — cabang, kota, bank — yang tanpanya form tidak dapat diisi.
//  5. Bengkel rekanan yang TIDAK punya login aplikasi. Ia keadaan yang di Pega tidak
//     terlihat di layar mana pun, dan akibatnya bengkel itu tidak akan pernah dapat masuk.
func checkBengkel(ctx context.Context, primary *sql.DB, print func(string, ...any)) {
	repo := masterbengkelsql.NewRepo(primary)

	if err := repo.CheckTable(ctx); err != nil {
		print("  [BELUM] POOLDATA.BENGKEL_HE belum dapat dibaca: %v", err)
		print("            Tabelnya warisan Pega dan TIDAK dibuat migrasi aplikasi ini.")
		print("            Mintakan hak bacanya ke DBA.")
		return
	}

	total := 0
	pendingCount := 0
	for _, s := range []masterbengkel.ApprovalStatus{
		masterbengkel.StatusApproved,
		masterbengkel.StatusPending,
		masterbengkel.StatusRejected,
	} {
		count, err := repo.CountByStatus(ctx, s)
		if err != nil {
			print("  [GAGAL] POOLDATA.BENGKEL_HE status %q tidak dapat dihitung: %v", s, err)
			return
		}
		if s == masterbengkel.StatusPending {
			pendingCount = count
		}
		total += count
		print("            status %q %-16s %d baris", s, s.Label(), count)
	}
	print("  [ok]    POOLDATA.BENGKEL_HE dapat dibaca: %d baris berstatus dikenal", total)

	if pendingCount > 0 {
		print("  [catat] %d bengkel menunggu persetujuan.", pendingCount)
	}

	checkBengkelNumbering(ctx, repo, print)
	checkBengkelMirror(ctx, repo, print)
	checkBengkelLookup(ctx, repo, print)
	checkBengkelPartnerLogin(ctx, repo, print)
}

// checkBengkelNumbering melaporkan kesiapan penomoran ID_BENGKEL.
//
// Ia benar-benar MENGAMBIL satu nomor urut, dan itu disengaja meski mode periksa tidak
// menulis apa pun: sequence yang dapat dibaca tetapi tidak dapat diambil nomornya adalah
// keadaan yang tidak terlihat dari pemeriksaan mana pun yang lebih lembut.
//
// Satu nomor yang terpakai tidak berakibat apa pun — deretnya memang tidak menjanjikan
// kesinambungan, dan Pega pun melompati nomor setiap kali penyimpanannya gagal.
func checkBengkelNumbering(ctx context.Context, repo *masterbengkelsql.Repo, print func(string, ...any)) {
	id, err := repo.NextID(ctx)
	if err != nil {
		print("  [GAGAL] penomoran ID_BENGKEL tidak dapat dijalankan: %v", err)
		print("            Penambahan bengkel baru akan gagal. Yang dibutuhkan:")
		print("            POOLDATA.M_SITE_DATABASE berbaris CURRENT_SITE='1', dan")
		print("            sequence POOLDATA.BENGKEL_HE_SEQ yang dapat diambil nomornya.")
		return
	}
	print("  [ok]    penomoran ID_BENGKEL siap; contoh berikutnya: %s", id)
	print("            (satu nomor urut terpakai oleh pemeriksaan ini; tidak ada baris ditulis)")
}

// checkBengkelMirror membandingkan BENGKEL_HE dengan M_BENGKEL_HE.
//
// Lihat butir 3 pada checkBengkel untuk alasan kenapa perbandingan ini menentukan.
func checkBengkelMirror(ctx context.Context, repo *masterbengkelsql.Repo, print func(string, ...any)) {
	if err := repo.CheckJSONMirror(ctx); err != nil {
		print("  [catat] POOLDATA.M_BENGKEL_HE tidak dapat dibaca: %v", err)
		print("            Tabel JSON milik Pega. Tanpa akses bacanya, kesamaan kedua")
		print("            sumber tidak dapat diperiksa dari sini — mintakan ke DBA.")
		return
	}

	flat, errFlat := repo.CountAll(ctx)
	mirror, errMirror := repo.CountJSONMirror(ctx)
	if errFlat != nil || errMirror != nil {
		print("  [catat] jumlah baris kedua sumber tidak dapat dibandingkan")
		return
	}

	switch {
	case flat == mirror:
		print("  [ok]    BENGKEL_HE dan M_BENGKEL_HE sama-sama %d baris", flat)
		print("            Sejalan dengan dugaan bahwa keduanya SATU sumber (view atas JSON).")
		print("            Tetap mintakan kepastiannya ke DBA sebelum modul ini menulis di produksi.")
	default:
		print("  [WASPADA] BENGKEL_HE %d baris, M_BENGKEL_HE %d baris — TIDAK sama", flat, mirror)
		print("            Keduanya kemungkinan DUA tabel terpisah yang disinkronkan.")
		print("            Bila benar, penulisan modul ini TIDAK akan sampai ke Pega dan")
		print("            sebaliknya. Jangan aktifkan jalur tulis sebelum DBA memastikan")
		print("            mana yang menjadi sumbernya. Lihat banner masterbengkel.sql.")
	}
}

// checkBengkelLookup melaporkan ketiga tabel acuan yang menyuapi form.
func checkBengkelLookup(ctx context.Context, repo *masterbengkelsql.Repo, print func(string, ...any)) {
	branch, err := repo.ListBranches(ctx)
	switch {
	case err != nil:
		print("  [GAGAL] daftar cabang bengkel tidak dapat dibaca: %v", err)
	case len(branch) == 0:
		print("  [WASPADA] daftar cabang bengkel KOSONG")
		print("            Kuerinya menyaring LDI_ID='0076' dan STS_AKTIF='1' —")
		print("            keduanya tertanam di rule lama. Bila entitas ini memakai kode")
		print("            aplikasi yang berbeda, dropdown Cabang akan selalu kosong.")
	default:
		print("  [ok]    daftar cabang bengkel: %d cabang", len(branch))
	}

	bank, err := repo.ListBanks(ctx)
	switch {
	case err != nil:
		print("  [GAGAL] daftar bank tidak dapat dibaca: %v", err)
	case len(bank) == 0:
		print("  [WASPADA] GENERAL.LST_BANK_GROUP tidak mengembalikan satu bank pun")
	default:
		print("  [ok]    daftar bank: %d bank", len(bank))
	}

	// Kota dicoba dengan kata kunci yang pasti ada pada daftar mana pun berbahasa
	// Indonesia. Yang diuji bukan hasilnya melainkan apakah tabelnya dapat dibaca —
	// nama tabelnya disebut TANPA skema di seluruh rule lama, sehingga yang terbaca
	// bergantung pada skema bawaan akun koneksi.
	switch city, err := repo.SearchCities(ctx, "JAKARTA"); {
	case err != nil:
		print("  [GAGAL] tabel CITY tidak dapat dibaca: %v", err)
		print("            Nama tabelnya disebut tanpa skema di seluruh rule lama, sehingga")
		print("            yang terbaca mengikuti skema bawaan akun koneksi. Mintakan")
		print("            sinonim atau hak bacanya ke DBA.")
	case len(city) == 0:
		print("  [catat] tabel CITY dapat dibaca, tetapi tanpa hasil untuk kata kunci contoh")
	default:
		print("  [ok]    tabel CITY dapat dibaca: %d kota cocok dengan kata kunci contoh", len(city))
	}
}

// checkBengkelPartnerLogin melaporkan bengkel rekanan yang tidak punya login aplikasi.
//
// Keadaan itu tidak terlihat di layar Pega mana pun, dan akibatnya nyata: bengkelnya
// tidak akan pernah dapat masuk. Ia dilaporkan di sini, bukan ditolak diam-diam — baris
// lama dibaca apa adanya, dan yang baru sudah ditolak lebih dulu oleh Input.Check.
func checkBengkelPartnerLogin(ctx context.Context, repo *masterbengkelsql.Repo, print func(string, ...any)) {
	missing := 0
	for _, s := range []masterbengkel.ApprovalStatus{
		masterbengkel.StatusApproved,
		masterbengkel.StatusPending,
	} {
		list, err := repo.List(ctx, masterbengkel.Filter{Status: s})
		if err != nil {
			return
		}
		for _, w := range list {
			if w.PartnerStatus != masterbengkel.PartnerStatusNonPartner && strings.TrimSpace(w.Login) == "" {
				missing++
			}
		}
	}
	if missing > 0 {
		print("  [WASPADA] %d bengkel berstatus rekanan TANPA login aplikasi", missing)
		print("            Bengkelnya tidak akan pernah dapat masuk, dan tidak ada satu pun")
		print("            layar di sistem lama yang menjelaskan sebabnya.")
	}
}

// checkPanel melaporkan kesiapan POOLDATA.PANEL_HE beserta tabel anaknya.
//
// Kedua tabelnya warisan Pega dan TIDAK dibuat migrasi aplikasi ini, sehingga "belum dapat
// dibaca" di sini berarti tabelnya memang tidak ada di entitas itu, atau akun aplikasi
// belum diberi hak bacanya — keduanya urusan DBA.
//
// # Lima hal dilaporkan, dan yang KELIMA adalah alasan utama fungsi ini ada
//
//  1. Jumlah baris per status — apakah ketiga tab layar akan terisi.
//  2. Ketersediaan penomoran: kode situs dan sequence. Tanpa keduanya, penambahan gagal
//     pada permintaan pertama, dan gagalnya baru terlihat saat petugas menekan Simpan.
//  3. Apakah PANEL_HE dan M_PANEL_HE satu sumber atau dua. Sistem lama MENULIS dokumen
//     JSON ke M_PANEL_HE dan MEMBACA kolom dari PANEL_HE; modul ini menulis kolom langsung
//     ke PANEL_HE. Bila keduanya ternyata dua tabel terpisah yang disinkronkan, penulisan
//     di sini tidak akan pernah sampai ke Pega — dan Pega tidak akan pernah sampai ke
//     sini. Rinciannya di banner masterpanel.sql.
//  4. Tabel anak LOKASI_PANEL_HE: dapat dibaca, berapa barisnya, dan berapa yang induknya
//     sudah tidak ada.
//  5. **Apakah kolom NAMA memuat hal yang sama dengan LOKASI_PANEL.** Modul ini MENULIS
//     keduanya dengan nilai yang sama, dan itu asumsi yang disimpulkan dari dua kueri yang
//     saling melengkapi — bukan dari DDL, yang belum ada (R-08). Bila asumsinya salah,
//     penulisan modul ini akan merusak kolom yang dibaca modul Grouping Sparepart.
//     Perbandingan ini adalah cara termurah membuktikannya tanpa DDL.
func checkPanel(ctx context.Context, primary *sql.DB, print func(string, ...any)) {
	repo := masterpanelsql.NewRepo(primary)

	if err := repo.CheckTable(ctx); err != nil {
		print("  [BELUM] POOLDATA.PANEL_HE belum dapat dibaca: %v", err)
		print("            Tabelnya warisan Pega dan TIDAK dibuat migrasi aplikasi ini.")
		print("            Mintakan hak bacanya ke DBA.")
		return
	}

	total := 0
	pendingCount := 0
	for _, s := range []masterpanel.ApprovalStatus{
		masterpanel.StatusApproved,
		masterpanel.StatusPending,
		masterpanel.StatusRejected,
	} {
		count, err := repo.CountByStatus(ctx, s)
		if err != nil {
			print("  [GAGAL] POOLDATA.PANEL_HE status %q tidak dapat dihitung: %v", s, err)
			return
		}
		if s == masterpanel.StatusPending {
			pendingCount = count
		}
		total += count
		print("            status %q %-16s %d baris", s, s.Label(), count)
	}
	print("  [ok]    POOLDATA.PANEL_HE dapat dibaca: %d baris berstatus dikenal", total)

	if pendingCount > 0 {
		print("  [catat] %d panel menunggu persetujuan.", pendingCount)
	}

	checkPanelNumbering(ctx, repo, print)
	checkPanelMirror(ctx, repo, print)
	checkPanelLocation(ctx, repo, print)
}

// checkPanelNumbering melaporkan kesiapan penomoran ID_PANEL.
//
// Ia benar-benar MENGAMBIL satu nomor urut, dan itu disengaja meski mode periksa tidak
// menulis apa pun: sequence yang dapat dibaca tetapi tidak dapat diambil nomornya adalah
// keadaan yang tidak terlihat dari pemeriksaan mana pun yang lebih lembut.
//
// Satu nomor yang terpakai tidak berakibat apa pun — deretnya memang tidak menjanjikan
// kesinambungan, dan Pega pun melompati nomor setiap kali penyimpanannya gagal.
func checkPanelNumbering(ctx context.Context, repo *masterpanelsql.Repo, print func(string, ...any)) {
	id, err := repo.NextID(ctx)
	if err != nil {
		print("  [GAGAL] penomoran ID_PANEL tidak dapat dijalankan: %v", err)
		print("            Penambahan panel baru akan gagal. Yang dibutuhkan:")
		print("            POOLDATA.M_SITE_DATABASE berbaris CURRENT_SITE='1', dan")
		print("            sequence POOLDATA.PANEL_HE_SEQ yang dapat diambil nomornya.")
		return
	}
	print("  [ok]    penomoran ID_PANEL siap; contoh berikutnya: %s", id)
	print("            (enam digit nomor urut, bukan sepuluh seperti ID_BENGKEL)")
	print("            (satu nomor urut terpakai oleh pemeriksaan ini; tidak ada baris ditulis)")
}

// checkPanelMirror membandingkan PANEL_HE dengan M_PANEL_HE.
//
// Lihat butir 3 pada checkPanel untuk alasan kenapa perbandingan ini menentukan.
func checkPanelMirror(ctx context.Context, repo *masterpanelsql.Repo, print func(string, ...any)) {
	if err := repo.CheckJSONMirror(ctx); err != nil {
		print("  [catat] POOLDATA.M_PANEL_HE tidak dapat dibaca: %v", err)
		print("            Tabel JSON milik Pega. Tanpa akses bacanya, kesamaan kedua")
		print("            sumber tidak dapat diperiksa dari sini — mintakan ke DBA.")
		return
	}

	flat, errFlat := repo.CountAll(ctx)
	mirror, errMirror := repo.CountJSONMirror(ctx)
	if errFlat != nil || errMirror != nil {
		print("  [catat] jumlah baris kedua sumber tidak dapat dibandingkan")
		return
	}

	switch {
	case flat == mirror:
		print("  [ok]    PANEL_HE dan M_PANEL_HE sama-sama %d baris", flat)
		print("            Sejalan dengan dugaan bahwa keduanya SATU sumber (view atas JSON).")
		print("            Tetap mintakan kepastiannya ke DBA sebelum modul ini menulis di produksi.")
	default:
		print("  [WASPADA] PANEL_HE %d baris, M_PANEL_HE %d baris — TIDAK sama", flat, mirror)
		print("            Keduanya kemungkinan DUA tabel terpisah yang disinkronkan.")
		print("            Bila benar, penulisan modul ini TIDAK akan sampai ke Pega dan")
		print("            sebaliknya. Jangan aktifkan jalur tulis sebelum DBA memastikan")
		print("            mana yang menjadi sumbernya. Lihat banner masterpanel.sql.")
	}
}

// checkPanelLocation melaporkan tabel anak POOLDATA.LOKASI_PANEL_HE.
//
// Butir terakhirnya — perbandingan NAMA dengan LOKASI_PANEL — adalah satu-satunya
// pemeriksaan di seluruh mode periksa yang menguji ASUMSI PENULISAN, bukan sekadar
// ketersediaan. Lihat butir 5 pada checkPanel.
func checkPanelLocation(ctx context.Context, repo *masterpanelsql.Repo, print func(string, ...any)) {
	if err := repo.CheckLocationTable(ctx); err != nil {
		print("  [BELUM] POOLDATA.LOKASI_PANEL_HE belum dapat dibaca: %v", err)
		print("            Tanpa tabel ini, daftar lokasi pada setiap panel tidak dapat")
		print("            ditampilkan maupun disimpan. Mintakan hak bacanya ke DBA.")
		print("            Keempat kolom yang dituntut: ID_PANEL, LOKASI_PANEL, SISI_PANEL, NAMA.")
		return
	}

	total, err := repo.CountLocation(ctx)
	if err != nil {
		print("  [GAGAL] baris POOLDATA.LOKASI_PANEL_HE tidak dapat dihitung: %v", err)
		return
	}
	print("  [ok]    POOLDATA.LOKASI_PANEL_HE dapat dibaca: %d baris lokasi", total)

	if orphan, err := repo.CountLocationOrphan(ctx); err != nil {
		print("  [catat] lokasi tanpa induk tidak dapat dihitung: %v", err)
	} else if orphan > 0 {
		print("  [WASPADA] %d baris lokasi induknya TIDAK ADA di PANEL_HE", orphan)
		print("            Modul ini tidak dapat menghasilkannya — penyisipannya selalu")
		print("            berdampingan dengan induknya di dalam satu transaksi. Baris itu")
		print("            warisan, dan tidak akan pernah muncul di layar mana pun.")
	}

	mismatch, err := repo.CountLocationNameMismatch(ctx)
	if err != nil {
		print("  [catat] kolom NAMA tidak dapat dibandingkan dengan LOKASI_PANEL: %v", err)
		return
	}

	switch {
	case mismatch == 0:
		print("  [ok]    kolom NAMA sama dengan LOKASI_PANEL pada seluruh %d baris", total)
		print("            Asumsi penulisan modul ini TERBUKTI untuk data yang ada:")
		print("            panel_location_insert mengisi keduanya dengan nilai yang sama.")
	default:
		print("  [WASPADA] %d dari %d baris punya NAMA yang BERBEDA dari LOKASI_PANEL", mismatch, total)
		print("            Asumsi penulisan modul ini SALAH: keduanya dua hal berbeda.")
		print("            JANGAN aktifkan jalur tulis sebelum masterpanel.sql diperbaiki —")
		print("            penulisan sekarang akan merusak kolom yang dibaca modul")
		print("            Grouping Sparepart lewat RDB List/GetDataSisiPanel-SQL.xml.")
		print("            Mintakan DDL kedua tabel ke DBA (R-08).")
	}
}

// checkSparepart melaporkan kesiapan POOLDATA.SPAREPART_HE beserta kedua tabel acuannya.
//
// Ketiga tabelnya warisan Pega dan TIDAK dibuat migrasi aplikasi ini, sehingga "belum dapat
// dibaca" di sini berarti tabelnya memang tidak ada di entitas itu, atau akun aplikasi belum
// diberi hak bacanya — keduanya urusan DBA.
//
// # Empat hal dilaporkan, dan yang KEEMPAT adalah alasan utama fungsi ini ada
//
//  1. Jumlah baris per status — apakah ketiga tab layar akan terisi.
//  2. Ketersediaan penomoran: kode situs dan sequence. Tanpa keduanya, penambahan gagal pada
//     permintaan pertama, dan gagalnya baru terlihat saat petugas menekan Simpan.
//  3. Apakah SPAREPART_HE dan M_SPAREPART_HE_BU satu sumber atau dua. Sistem lama MENULIS
//     dokumen JSON ke M_SPAREPART_HE_BU dan MEMBACA kolom dari SPAREPART_HE; modul ini
//     menulis kolom langsung ke SPAREPART_HE. Bila keduanya ternyata dua tabel terpisah yang
//     disinkronkan, penulisan di sini tidak akan pernah sampai ke Pega — dan sebaliknya.
//     Rinciannya di banner mastersparepart.sql.
//  4. **Apakah kedua tabel acuan terbaca, dan apakah baris lama menunjuk kategori atau tipe
//     yang tidak ada di sana.** Dropdown layar ini hanya menawarkan kategori dan tipe yang
//     sudah disetujui, meniru `Activity/BrowseTipeKategoriPart`. Baris lama yang menunjuk
//     kategori yang kemudian ditolak karena itu akan tampil tanpa nama — keadaan yang sah,
//     tetapi yang jumlahnya perlu diketahui sebelum petugas menanyakannya.
func checkSparepart(ctx context.Context, primary *sql.DB, print func(string, ...any)) {
	repo := mastersparepartsql.NewRepo(primary)

	if err := repo.CheckTable(ctx); err != nil {
		print("  [BELUM] POOLDATA.SPAREPART_HE belum dapat dibaca: %v", err)
		print("            Objeknya warisan Pega dan TIDAK dibuat migrasi aplikasi ini.")
		print("            Catatan: kueri pemeriksaan ini menyebut PROD_DATE dan")
		print("            TGL_UPDATE_HARGA; bila yang gagal justru salah satunya, tipe")
		print("            kolomnya berbeda dari asumsi mastersparepart.sql.")
		if strings.Contains(err.Error(), "ORA-04063") {
			// ORA-04063 BUKAN soal hak akses. Ia berarti objeknya ADA tetapi
			// definisinya sendiri tidak dapat dikompilasi — pada sebuah view, biasanya
			// karena objek yang dirujuknya berubah atau hilang.
			//
			// Dibedakan dengan sengaja: "mintakan hak bacanya ke DBA" akan menyesatkan
			// di sini, dan menghabiskan satu putaran percakapan dengan DBA untuk hal
			// yang bukan penyebabnya.
			print("            ORA-04063 berarti objeknya ADA tetapi definisinya RUSAK —")
			print("            bukan soal hak akses. Bila ia view, kemungkinan besar objek")
			print("            yang dirujuknya berubah atau hilang. Yang menjawabnya:")
			print("              SELECT object_type, status FROM all_objects")
			print("               WHERE owner='POOLDATA' AND object_name='SPAREPART_HE';")
			print("              SELECT line, position, text FROM all_errors")
			print("               WHERE owner='POOLDATA' AND name='SPAREPART_HE' ORDER BY sequence;")
			print("            Pega membaca objek yang SAMA, sehingga layar Master Sparepart")
			print("            di Pega semestinya ikut gagal — patut dipastikan ke tim Pega.")
		} else {
			print("            Mintakan hak bacanya ke DBA.")
		}

		// Pemeriksaan TIDAK berhenti di sini, berbeda dari master lain.
		//
		// Bila SPAREPART_HE memang view di atas M_SPAREPART_HE_BU, pertanyaan DBA
		// berikutnya pasti "apakah datanya masih ada" — dan menjawabnya menuntut satu
		// pemeriksaan lagi yang justru TIDAK menyentuh objek yang rusak.
		checkSparepartStore(ctx, repo, print)
		return
	}

	total := 0
	pendingCount := 0
	for _, s := range []mastersparepart.ApprovalStatus{
		mastersparepart.StatusApproved,
		mastersparepart.StatusPending,
		mastersparepart.StatusRejected,
	} {
		count, err := repo.CountByStatus(ctx, s)
		if err != nil {
			print("  [GAGAL] POOLDATA.SPAREPART_HE status %q tidak dapat dihitung: %v", s, err)
			return
		}
		if s == mastersparepart.StatusPending {
			pendingCount = count
		}
		total += count
		print("            status %q %-16s %d baris", s, s.Label(), count)
	}
	print("  [ok]    POOLDATA.SPAREPART_HE dapat dibaca: %d baris berstatus dikenal", total)

	if pendingCount > 0 {
		print("  [catat] %d sparepart menunggu persetujuan.", pendingCount)
	}

	checkSparepartNumbering(ctx, repo, print)
	checkSparepartMirror(ctx, repo, print)
	checkSparepartLookup(ctx, repo, print)
}

// checkSparepartNumbering melaporkan kesiapan penomoran ID.
//
// Ia benar-benar MENGAMBIL satu nomor urut, dan itu disengaja meski mode periksa tidak
// menulis apa pun: sequence yang dapat dibaca tetapi tidak dapat diambil nomornya adalah
// keadaan yang tidak terlihat dari pemeriksaan mana pun yang lebih lembut.
//
// Satu nomor yang terpakai tidak berakibat apa pun — deretnya memang tidak menjanjikan
// kesinambungan, dan Pega pun melompati nomor setiap kali penyimpanannya gagal.
func checkSparepartNumbering(
	ctx context.Context,
	repo *mastersparepartsql.Repo,
	print func(string, ...any),
) {
	id, err := repo.NextID(ctx)
	if err != nil {
		print("  [GAGAL] penomoran ID sparepart tidak dapat dijalankan: %v", err)
		print("            Penambahan sparepart baru akan gagal. Yang dibutuhkan:")
		print("            POOLDATA.M_SITE_DATABASE berbaris CURRENT_SITE='1', dan")
		print("            sequence POOLDATA.SPAREPART_HE_SEQ yang dapat diambil nomornya.")
		return
	}
	print("  [ok]    penomoran ID sparepart siap; contoh berikutnya: %s", id)
	print("            (sepuluh digit nomor urut, bukan enam seperti ID_PANEL)")
	print("            (satu nomor urut terpakai oleh pemeriksaan ini; tidak ada baris ditulis)")
}

// checkSparepartMirror membandingkan SPAREPART_HE dengan M_SPAREPART_HE_BU.
//
// Lihat butir 3 pada checkSparepart untuk alasan kenapa perbandingan ini menentukan.
func checkSparepartMirror(
	ctx context.Context,
	repo *mastersparepartsql.Repo,
	print func(string, ...any),
) {
	if err := repo.CheckJSONMirror(ctx); err != nil {
		print("  [catat] POOLDATA.M_SPAREPART_HE_BU tidak dapat dibaca: %v", err)
		print("            Tabel JSON milik Pega. Tanpa akses bacanya, kesamaan kedua")
		print("            sumber tidak dapat diperiksa dari sini — mintakan ke DBA.")
		return
	}

	flat, errFlat := repo.CountAll(ctx)
	mirror, errMirror := repo.CountJSONMirror(ctx)
	if errFlat != nil || errMirror != nil {
		print("  [catat] jumlah baris kedua sumber tidak dapat dibandingkan")
		return
	}

	switch {
	case flat == mirror:
		print("  [ok]    SPAREPART_HE dan M_SPAREPART_HE_BU sama-sama %d baris", flat)
		print("            Sejalan dengan dugaan bahwa keduanya SATU sumber (view atas JSON).")
		print("            Tetap mintakan kepastiannya ke DBA sebelum modul ini menulis di produksi.")
	default:
		print("  [WASPADA] SPAREPART_HE %d baris, M_SPAREPART_HE_BU %d baris — TIDAK sama",
			flat, mirror)
		print("            Keduanya kemungkinan DUA tabel terpisah yang disinkronkan.")
		print("            Akhiran _BU pada nama tabel Pega memperkuat dugaan itu, dan")
		print("            artinya tidak disebut di mana pun dalam export.")
		print("            Jangan aktifkan jalur tulis sebelum DBA memastikan mana yang")
		print("            menjadi sumbernya. Lihat banner mastersparepart.sql.")
	}
}

// checkSparepartLookup melaporkan kedua tabel acuan Kategori dan Tipe.
//
// Butir terakhirnya — jumlah baris yang menunjuk acuan yang tidak ada — adalah yang
// menentukan berapa banyak baris lama akan tampil tanpa nama kategori atau tipe. Lihat butir
// 4 pada checkSparepart.
func checkSparepartLookup(
	ctx context.Context,
	repo *mastersparepartsql.Repo,
	print func(string, ...any),
) {
	categoryReadable := repo.CheckCategoryTable(ctx) == nil
	typeReadable := repo.CheckTypeTable(ctx) == nil

	if !categoryReadable {
		print("  [BELUM] POOLDATA.GCNM_M_SPAREPART_CATEGORY belum dapat dibaca.")
		print("            Dropdown Kategori akan kosong; sparepart tetap dapat disimpan")
		print("            tanpa kategori, dan nama kategori baris lama tidak akan tampil.")
	}
	if !typeReadable {
		print("  [BELUM] POOLDATA.GCNM_M_SPAREPART_TYPE belum dapat dibaca.")
		print("            Dropdown Tipe akan kosong, dengan akibat yang sama.")
	}
	if !categoryReadable || !typeReadable {
		return
	}

	category, errCategory := repo.ListCategories(ctx)
	partType, errType := repo.ListTypes(ctx)
	if errCategory != nil || errType != nil {
		print("  [catat] daftar acuan Kategori atau Tipe tidak dapat dibaca isinya")
		return
	}
	print("  [ok]    acuan sparepart terbaca: %d kategori, %d tipe yang sudah disetujui",
		len(category), len(partType))

	if len(category) == 0 || len(partType) == 0 {
		print("  [catat] salah satu daftar acuan KOSONG pada penyaring status disetujui.")
		print("            Layar Master Sparepart hanya menawarkan acuan yang sudah")
		print("            disetujui, meniru Activity/BrowseTipeKategoriPart. Bila acuannya")
		print("            memang masih menunggu, layar Master Kategori dan Master Tipe")
		print("            (MENU_ID 33 dan 34) yang harus menyetujuinya lebih dulu.")
	}

	orphanCategory, errOne := repo.CountOrphanCategory(ctx)
	orphanType, errTwo := repo.CountOrphanType(ctx)
	if errOne != nil || errTwo != nil {
		print("  [catat] baris yang acuannya tidak ada tidak dapat dihitung")
		return
	}

	switch {
	case orphanCategory == 0 && orphanType == 0:
		print("  [ok]    seluruh baris menunjuk kategori dan tipe yang ada di acuan")
	default:
		print("  [catat] %d baris menunjuk kategori yang tidak ada di acuan, %d menunjuk tipe",
			orphanCategory, orphanType)
		print("            Barisnya tetap terbaca dan tetap dapat disunting; yang tidak")
		print("            tampil hanyalah NAMA acuannya. Menyimpan ulang baris seperti itu")
		print("            menuntut petugas memilih kategori atau tipe yang sah.")
	}
}

// checkSupplier melaporkan kesiapan M_SUPPLIER beserta acuannya.
//
// Tabelnya warisan Pega dan TIDAK dibuat migrasi aplikasi ini, sehingga "belum dapat
// dibaca" di sini berarti tabelnya memang tidak ada di entitas itu, atau akun aplikasi
// belum diberi hak bacanya — keduanya urusan DBA.
//
// # Lima hal dilaporkan, dan yang KEDUA adalah alasan utama fungsi ini ada
//
//  1. Jumlah baris — apakah layarnya akan terisi.
//  2. **Berapa di antaranya yang dokumen JSON-nya benar-benar terbaca.** Seluruh isi
//     supplier tinggal di satu kolom JSONDATA, dan `JSON_VALUE` menjawab NULL alih-alih
//     gagal ketika kolomnya bukan JSON yang sah atau kuncinya dinamai lain. Tanpa
//     pemeriksaan ini, layar akan menampilkan sederet baris berisi kolom kosong tanpa satu
//     pun pesan galat — kelas kegagalan yang tidak dimiliki modul master lain.
//  3. Ketersediaan penomoran: kode situs dan sequence. Tanpa keduanya, penambahan gagal
//     pada permintaan pertama, dan gagalnya baru terlihat saat petugas menekan Simpan.
//  4. Antrean persetujuan POOLDATA.PROTEKSI_KLAIMMBU, yang tanpanya setiap penambahan
//     gagal separuh jalan — supplier tersimpan, permintaannya tidak.
//  5. Keempat tabel acuan — cabang, kota, negara, bank — yang tanpanya form tidak dapat
//     diisi.
func checkSupplier(ctx context.Context, primary *sql.DB, print func(string, ...any)) {
	repo := mastersuppliersql.NewRepo(primary)

	if err := repo.CheckTable(ctx); err != nil {
		print("  [BELUM] M_SUPPLIER belum dapat dibaca: %v", err)
		print("            Tabelnya warisan Pega dan TIDAK dibuat migrasi aplikasi ini.")
		print("            Mintakan hak bacanya ke DBA.")
		return
	}

	total, err := repo.CountAll(ctx)
	if err != nil {
		print("  [GAGAL] M_SUPPLIER tidak dapat dihitung: %v", err)
		return
	}
	print("  [ok]    M_SUPPLIER dapat dibaca: %d baris", total)

	checkSupplierDocument(ctx, repo, total, print)
	checkSupplierNumbering(ctx, repo, print)
	checkSupplierApproval(ctx, repo, print)
	checkSupplierLookup(ctx, repo, print)
}

// checkSupplierNumbering melaporkan kesiapan penomoran ID supplier.
//
// Ia benar-benar MENGAMBIL satu nomor urut, dan itu disengaja meski mode periksa tidak
// menulis apa pun: sequence yang dapat dibaca tetapi tidak dapat diambil nomornya adalah
// keadaan yang tidak terlihat dari pemeriksaan mana pun yang lebih lembut.
//
// Satu nomor yang terpakai tidak berakibat apa pun — deretnya memang tidak menjanjikan
// kesinambungan, dan Pega pun melompati nomor setiap kali penyimpanannya gagal.
func checkSupplierNumbering(ctx context.Context, repo *mastersuppliersql.Repo, print func(string, ...any)) {
	id, err := repo.NextID(ctx)
	if err != nil {
		print("  [GAGAL] penomoran ID supplier tidak dapat dijalankan: %v", err)
		print("            Penambahan supplier baru akan gagal. Yang dibutuhkan:")
		print("            POOLDATA.M_SITE_DATABASE berbaris CURRENT_SITE='1', dan")
		print("            sequence SUPPLIER_SEQ yang dapat diambil nomornya.")
		return
	}
	print("  [ok]    penomoran ID supplier siap; contoh berikutnya: %s", id)
	print("            (satu nomor urut terpakai oleh pemeriksaan ini; tidak ada baris ditulis)")
}

// checkSupplierCode melaporkan sandi yang dipakai kelima dropdown bersandi.
//
// Ia ada karena daftar pilihan kelimanya TIDAK ADA di export: isiannya `pxDropdown`
// bersumber `associated`, artinya daftarnya hidup di rule Field Value yang tidak ikut
// diekspor (`R-16`). Yang ditawarkan layar karena itu digabung dari sandi yang artinya
// terbukti di activity dan sandi yang benar-benar ada di data.
//
// Melaporkannya di sini membuat selisih antara keduanya terlihat SEBELUM layarnya dipakai
// — dan itulah bahan yang dibutuhkan saat meminta daftar aslinya ke Work Owner.
func checkSupplierCode(ctx context.Context, repo *mastersuppliersql.Repo, print func(string, ...any)) {
	set, err := repo.ListCodes(ctx)
	if err != nil {
		print("  [catat] sandi dropdown tidak dapat dihitung: %v", err)
		return
	}

	print("  [catat] sandi yang dipakai baris yang ada — status rekanan %d, status supply %d,",
		len(set.PartnerStatus), len(set.SupplyType))
	print("            jenis supplier %d, status aktif %d, autopayment %d",
		len(set.SupplierType), len(set.Active), len(set.AutoPayment))
	print("            Daftar pilihan aslinya TIDAK ada di export (R-16). Yang ditawarkan")
	print("            layar adalah sandi di atas ditambah yang artinya terbukti di activity.")
	print("            Mintakan daftar Field Value-nya ke Work Owner.")
}

// checkSupplierApproval melaporkan kesiapan antrean persetujuan.
//
// Hak BACA yang ada di sini tidak menjamin hak TULIS, dan yang dibutuhkan penambahan
// supplier adalah yang kedua. Itu dinyatakan terang-terangan supaya laporan hijau di sini
// tidak dibaca sebagai jaminan.
func checkSupplierApproval(ctx context.Context, repo *mastersuppliersql.Repo, print func(string, ...any)) {
	if err := repo.CheckApprovalTable(ctx); err != nil {
		print("  [BELUM] POOLDATA.PROTEKSI_KLAIMMBU belum dapat dibaca: %v", err)
		print("            Setiap penambahan supplier menyisipkan satu baris ke sana.")
		print("            Tanpa itu penyimpanan gagal separuh jalan: supplier tersimpan,")
		print("            permintaan persetujuannya tidak. Mintakan haknya ke DBA.")
		return
	}

	pending, err := repo.CountApprovalPending(ctx)
	if err != nil {
		print("  [catat] permintaan persetujuan supplier tidak dapat dihitung: %v", err)
		return
	}
	print("  [ok]    POOLDATA.PROTEKSI_KLAIMMBU terbaca; %d permintaan supplier di posisi awal", pending)
	print("            (hak BACA terbukti; hak TULIS belum — itu yang dibutuhkan penambahan)")

	if pending > 0 {
		print("  [catat] %d permintaan menunggu, dan layar pemutusnya BELUM ADA di aplikasi ini.", pending)
		print("            Sisi pemutus antrean itu tidak ada di export Pega sama sekali (R-16).")
	}
}

// checkSupplierDocument melaporkan berapa baris yang dokumen JSON-nya terbaca.
//
// Lihat butir 2 pada checkSupplier untuk alasan kenapa pemeriksaan ini menentukan.
func checkSupplierDocument(
	ctx context.Context,
	repo *mastersuppliersql.Repo,
	total int,
	print func(string, ...any),
) {
	readable, err := repo.CountReadable(ctx)
	if err != nil {
		print("  [GAGAL] dokumen JSON M_SUPPLIER tidak dapat dibaca: %v", err)
		print("            Bila galatnya menyebut JSON, kolom JSONDATA berisi sesuatu yang")
		print("            bukan JSON yang sah. Layar Master Supplier TIDAK boleh dipakai")
		print("            sebelum itu diperbaiki.")
		return
	}

	switch {
	case total == 0:
		print("  [catat] belum ada satu pun supplier; bentuk dokumennya belum dapat diperiksa")
	case readable == total:
		print("  [ok]    seluruh %d dokumen JSON-nya terbaca; kunci NAMA ada di semuanya", total)
	case readable == 0:
		print("  [GAGAL] TIDAK SATU PUN dari %d baris punya kunci NAMA yang terbaca", total)
		print("            Kolom JSONDATA kemungkinan besar memakai nama kunci yang BERBEDA")
		print("            dari yang dibaca RDB List/GetDataEditMasterSupller-SQL.xml.")
		print("            Layarnya akan menampilkan baris kosong tanpa galat — JSON_VALUE")
		print("            menjawab NULL alih-alih gagal. Bawa temuan ini ke DBA sebelum")
		print("            layar Master Supplier dipakai.")
	default:
		print("  [WASPADA] %d dari %d baris tidak punya kunci NAMA yang terbaca", total-readable, total)
		print("            Baris itu akan tampil tanpa nama di layar, dan pemeriksaan nama")
		print("            ganda tidak akan pernah menangkapnya. Periksa isi JSONDATA-nya.")
	}
}

// checkSupplierLookup melaporkan keempat tabel acuan yang menyuapi form.
//
// Kegagalan salah satunya TIDAK menghentikan pemeriksaan yang lain: form yang kehilangan
// satu dropdown masih dapat dipakai sebagian, dan mengetahui ketiga sisanya siap jauh
// lebih berguna daripada berhenti pada yang pertama gagal.
func checkSupplierLookup(ctx context.Context, repo *mastersuppliersql.Repo, print func(string, ...any)) {
	branch, errBranch := repo.ListBranches(ctx)
	if errBranch != nil {
		print("  [BELUM] daftar cabang (M_BRANCH) belum dapat dibaca: %v", errBranch)
	} else {
		print("  [ok]    daftar cabang terbaca: %d cabang", len(branch))
	}

	// Kata kuncinya sengaja satu huruf yang umum, bukan nama kota sungguhan: yang diuji
	// adalah tabelnya dapat dibaca, bukan isinya memuat kota tertentu.
	city, errCity := repo.SearchCities(ctx, "ja")
	if errCity != nil {
		print("  [BELUM] tabel kota (CITY) belum dapat dibaca: %v", errCity)
	} else {
		print("  [ok]    lookup kota terbaca: %d kota cocok dengan contoh kata kunci", len(city))
	}

	country, errCountry := repo.ListCountries(ctx)
	if errCountry != nil {
		print("  [BELUM] daftar negara (COUNTRY) belum dapat dibaca: %v", errCountry)
	} else {
		print("  [ok]    daftar negara terbaca: %d negara", len(country))
	}

	bank, errBank := repo.ListBanks(ctx)
	if errBank != nil {
		print("  [BELUM] daftar bank (GENERAL.LST_BANK_GROUP) belum dapat dibaca: %v", errBank)
	} else {
		print("  [ok]    daftar bank terbaca: %d bank", len(bank))
	}

	checkSupplierCode(ctx, repo, print)
}

// checkClaimHistoryGate melaporkan kesiapan gerbang proteksi data layar View History
// Claim sesudah migrasi 0004.
//
// # Kenapa ia diperiksa terpisah dari tabel aplikasi lain
//
// Karena kesiapannya menempuh DUA pihak, bukan satu. Tabel jejaknya dibuat DBA lewat
// migrasi 0004; tetapi baris proteksi penggunanya didaftarkan lewat layar Master Proteksi
// Data MILIK SISTEM LAMA. Salah satunya belum selesai berarti layar menolak setiap
// pengguna — dan penolakan itu benar menurut aturan, sehingga tidak akan tampak sebagai
// galat di mana pun kecuali di sini.
func checkClaimHistoryGate(
	ctx context.Context,
	repo *riwayatklaimsql.ProtectionRepo,
	print func(string, ...any),
) {
	ready, err := repo.TableReady(ctx)
	switch {
	case err != nil:
		print("  [GAGAL] POOLDATA.%s tidak dapat diperiksa: %v", riwayatklaimsql.TableName, err)
		return
	case !ready:
		print("  [BELUM] POOLDATA.%s belum ada", riwayatklaimsql.TableName)
		print("            Tabelnya dibuat migrasi 0004. Selama belum dijalankan, layar")
		print("            View History Claim tidak dapat dipakai terhadap Oracle —")
		print("            tetapi seluruh bagian lain tetap jalan.")
		return
	}
	print("  [ok]    POOLDATA.%s dapat dibaca", riwayatklaimsql.TableName)

	// Master proteksi dibaca dengan login yang PASTI tidak ada, sehingga pemeriksaan ini
	// tidak menyentuh data siapa pun. Yang diuji hanyalah apakah tabelnya dapat dibaca
	// akun aplikasi — hak SELECT atasnya diberikan migrasi 0004 langkah 4.
	if _, _, err := repo.Find(ctx, "\x00periksa", riwayatklaim.ModuleKey); err != nil {
		print("  [BELUM] POOLDATA.MST_PROTEKSI_DATA_PNC tidak dapat dibaca: %v", err)
		print("            Tanpa hak baca atasnya, gerbang proteksi menolak SETIAP")
		print("            pengguna. Lihat langkah 4 migrasi 0004.")
		return
	}
	print("  [ok]    POOLDATA.MST_PROTEKSI_DATA_PNC dapat dibaca")
	print("            Catatan: pendaftaran pengguna untuk MODUL=%q dilakukan lewat", riwayatklaim.ModuleKey)
	print("            layar Master Proteksi Data milik sistem lama, bukan oleh aplikasi ini.")
}

// countDuplicateNames menghitung berapa baris tingkat 1 yang namanya sudah dipakai baris
// lain.
func countDuplicateNames(list []masterpenolakan.RejectionStatus) int {
	seen := make(map[string]bool, len(list))
	duplicate := 0
	for _, parent := range list {
		name := strings.ToUpper(strings.TrimSpace(parent.Name))
		if seen[name] {
			duplicate++
			continue
		}
		seen[name] = true
	}
	return duplicate
}

// checkMasterAutoClaim melaporkan kesiapan POOLDATA.M_AUTO_CLAIM_PNC beserta acuannya.
//
// Tabelnya warisan Pega dan TIDAK dibuat migrasi aplikasi ini, sehingga "belum dapat
// dibaca" di sini berarti tabelnya memang tidak ada di entitas itu, atau akun aplikasi
// belum diberi hak bacanya — keduanya urusan DBA.
//
// Tiga hal dilaporkan, dan ketiganya menjawab pertanyaan yang berbeda:
//
//  1. Jumlah baris per status. Itu yang memberi tahu apakah keempat tab layar akan
//     terisi, sebelum layarnya dibuka.
//  2. Berapa baris yang benar-benar DAPAT DIPAKAI klaim otomatis. Ia bukan sekadar
//     jumlah yang disetujui: baris yang disetujui tetapi CLAIM_ALLOWED-nya bukan "1"
//     tidak pernah lolos penyaring GetReceiverClaimAsuransiKredit, dan selisih kedua
//     angka itu adalah baris yang tampak benar tetapi tidak pernah dipakai.
//  3. Ada tidaknya penyetuju komite. Tanpa itu, SETIAP baris baru lahir tanpa penyetuju
//     dan tertahan di Waiting Approval selamanya — kegagalan senyap yang tidak terlihat
//     di layar mana pun.
func checkMasterAutoClaim(ctx context.Context, primary *sql.DB, print func(string, ...any)) {
	repo := masterautoclaimsql.NewRepo(primary)

	if err := repo.CheckTable(ctx); err != nil {
		print("  [BELUM] POOLDATA.M_AUTO_CLAIM_PNC belum dapat dibaca: %v", err)
		print("            Tabelnya warisan Pega dan TIDAK dibuat migrasi aplikasi ini.")
		print("            Mintakan hak bacanya ke DBA.")
		return
	}

	total := 0
	usable := 0
	for _, s := range []masterautoclaim.ApprovalStatus{
		masterautoclaim.StatusApproved,
		masterautoclaim.StatusPending,
		masterautoclaim.StatusRejected,
	} {
		list, err := repo.List(ctx, masterautoclaim.Filter{Status: s})
		if err != nil {
			print("  [GAGAL] POOLDATA.M_AUTO_CLAIM_PNC status %q tidak dapat dibaca: %v", s, err)
			return
		}
		total += len(list)
		for _, ac := range list {
			if ac.Usable() {
				usable++
			}
		}
		print("            status %q %-16s %d baris", s, s.Label(), len(list))
	}
	print("  [ok]    POOLDATA.M_AUTO_CLAIM_PNC dapat dibaca: %d baris", total)

	approved, err := repo.List(ctx, masterautoclaim.Filter{Status: masterautoclaim.StatusApproved})
	if err == nil && len(approved) != usable {
		print("  [WASPADA] %d baris disetujui tetapi hanya %d yang CLAIM_ALLOWED-nya \"1\"",
			len(approved), usable)
		print("            Selisihnya TIDAK akan dipakai pembuatan klaim otomatis, dan tidak ada")
		print("            satu pun tanda di layar yang menjelaskannya.")
	}

	switch operator, err := repo.Committee(ctx); {
	case err != nil:
		print("  [GAGAL] POOLDATA.EMAILKOMITE tidak dapat dibaca: %v", err)
	case operator == "":
		print("  [WASPADA] tidak ada penyetuju komite untuk Master Auto Claim")
		print("            POOLDATA.EMAILKOMITE tidak punya baris TYPE_BUSINESS='BONDING'")
		print("            dengan STS_AKTIF='1'. Setiap baris BARU akan tertahan di Waiting")
		print("            Approval tanpa pernah muncul di tab Komite Approval siapa pun.")
	default:
		print("  [ok]    penyetuju komite Master Auto Claim: %s", operator)
	}
}

// checkOutstandingExport membuktikan kueri unduhan sah dan TIDAK terikat pemilik.
//
// Ia mencetak jumlah baris untuk kelima cakupan sekaligus. Angka-angka itulah yang
// membedakan "cakupannya bekerja" dari "cakupannya diabaikan": bila kelimanya sama, berarti
// penyaring lini bisnis tidak menggigit sama sekali.
func checkOutstandingExport(ctx context.Context, repo *inboxoutstandingsql.Repo, login string, print func(string, ...any)) {
	cakupan := []struct {
		nama string
		line inboxoutstanding.LineBusiness
	}{
		{"tanpa cakupan", inboxoutstanding.LineUnknown},
		{"NONMBU", inboxoutstanding.LineNonMBU},
		{"PA", inboxoutstanding.LinePA},
		{"TRAVEL", inboxoutstanding.LineTravel},
		{"BONDING", inboxoutstanding.LineBonding},
	}

	for i, c := range cakupan {
		// Limit 1: yang dicari jumlahnya, bukan isinya.
		page, err := repo.Export(ctx, inboxoutstanding.ExportFilter{LineBusiness: c.line, Limit: 1})
		if err != nil {
			print("  [BELUM] unduhan My Inbox tidak dapat dijalankan: %v", err)
			return
		}
		if i == 0 {
			print("  [ok]    unduhan My Inbox berjalan TANPA penyaring pemilik pekerjaan")
		}
		print("            cakupan %-14s %d baris", c.nama, page.Total)
	}

	if login == "" {
		return
	}
	line, err := repo.LineBusinessFor(ctx, login)
	if err != nil {
		print("  [BELUM] POOLDATA.M_LOGIN_PNC.LINE_BUSINESS tidak dapat dibaca: %v", err)
		return
	}
	if line == inboxoutstanding.LineUnknown {
		print("  [catat] %s belum punya LINE_BUSINESS — unduhannya memakai cakupan penuh", login)
		return
	}
	print("  [ok]    lini bisnis %s: %s", login, line)
}

// checkOSClaimPerCabang memeriksa layar Inbox OS Claim per Cabang (`MENU_ID 69`).
//
// Yang diperiksa tiga hal, dan ketiganya dipisah karena tindak lanjutnya berbeda:
//
//	tabel layar        gagal -> layarnya tidak dapat dibuka sama sekali
//	tabel ekspor       gagal -> hanya tombol ekspor yang mati, termasuk bila DB Link padam
//	satu halaman nyata gagal -> kuerinya berjalan tetapi ada yang tidak terbaca
func checkOSClaimPerCabang(
	ctx context.Context,
	repo *inboxosclaimpercabangsql.Repo,
	print func(string, ...any),
) {
	if err := repo.CheckTable(ctx); err != nil {
		print("  [BELUM] Tabel OS klaim per cabang tidak dapat dibaca: %v", err)
		print("            Modul ini TIDAK menuntut migrasi — seluruh tabelnya milik Pega.")
		print("            Periksa hak SELECT akun aplikasi atas POOLDATA.T_CLAIM_PNC,")
		print("            DATAPEGA.PC_ASM_FW_GCNMFW_WORK, POOLDATA.GCNM_PROGRESS_CLAIM,")
		print("            POOLDATA.GCNM_MST_PROGRESS, POOLDATA.T_CLAIM_ESTIMASI,")
		print("            POOLDATA.T_SURVEYORLIST, POOLDATA.T_CLAIM_OBJECTCOVERAGE,")
		print("            dan POOLDATA.BRANCH.")
		return
	}
	print("  [ok]    Kedelapan tabel OS klaim per cabang dapat dibaca")

	// Tabel ekspor diperiksa TERPISAH, dan kegagalannya tidak menghentikan pemeriksaan.
	// Salah satunya `treaty_loss@asmd` — DB Link yang sedang padam mematikan ekspor, bukan
	// layarnya.
	if err := repo.CheckExportTable(ctx); err != nil {
		print("  [BELUM] Tabel ekspor OS per cabang tidak dapat dibaca: %v", err)
		print("            Hanya tombol Export To Excel yang terdampak; layarnya tetap jalan.")
		print("            Periksa DB Link asmd.sinarmas.co.id dan hak SELECT atas")
		print("            treaty_loss, POOLDATA.T_GENERAL, POOLDATA.T_CLAIM_DOMINANFACTOR,")
		print("            serta POOLDATA.M_DOMINAN_FACTOR.")
	} else {
		print("  [ok]    Tabel ekspor OS per cabang dapat dibaca, termasuk lewat DB Link")
	}

	// Satu cabang nyata diambil dari datanya sendiri, bukan dikarang: kode cabang karangan
	// akan selalu menghasilkan nol baris, dan nol baris tidak membuktikan kuerinya berjalan.
	branch, detailBranch, found, err := repo.AnyBranchWithClaims(ctx)
	switch {
	case err != nil:
		print("  [GAGAL] Kode cabang contoh tidak dapat dibaca: %v", err)
		return
	case !found:
		print("  [lewat] Tidak ada satu pun klaim outstanding; kueri daftar tidak dicoba")
		return
	}

	// Jalurnya sama persis dengan layar: kode cabang RINCI dari HCQ, diterjemahkan lebih dulu.
	// Memberi kueri daftar kode klaim secara langsung akan melewati terjemahan itu — dan
	// terjemahan itulah satu-satunya langkah yang bila salah menghasilkan layar kosong tanpa
	// satu pun galat.
	resolved, known, err := repo.BranchOf(ctx, detailBranch)
	switch {
	case err != nil:
		print("  [GAGAL] Cabang tidak dapat diterjemahkan dari kode rinci %q: %v",
			detailBranch, err)
		return
	case !known:
		print("  [GAGAL] Kode cabang rinci %q tidak dikenal POOLDATA.BRANCH", detailBranch)
		print("            Periksa kolom OLDID; tanpa ini seluruh layar menolak pemanggil.")
		return
	case resolved.Code != branch:
		print("  [GAGAL] Terjemahan cabang meleset: kode rinci %q menghasilkan %q, "+
			"seharusnya %q", detailBranch, resolved.Code, branch)
		return
	}
	print("  [ok]    Kode rinci %s diterjemahkan menjadi cabang %s (%s)",
		detailBranch, resolved.Code, resolved.Name)

	page := inboxosclaimpercabang.Pagination{Page: 1, Size: 5}
	result, err := repo.List(ctx, inboxosclaimpercabang.Query{Branch: resolved}, page)
	if err != nil {
		print("  [GAGAL] Kueri daftar OS per cabang gagal: %v", err)
		return
	}
	print("  [ok]    Cabang %s punya %d klaim outstanding (%d baris dibaca)",
		branch, result.Total, len(result.Items))

	stalled := 0
	for _, item := range result.Items {
		if item.ProgressStalled {
			stalled++
		}
	}
	print("  [info]  %d dari %d baris contoh bertanda progres mandek",
		stalled, len(result.Items))

	// Popup Detail diperiksa TERPISAH, dan kegagalannya tidak menghentikan pemeriksaan:
	// tabel yang hanya dipakai popup yang tidak terbaca membuat satu tombol tidak bekerja,
	// bukan seluruh layar cabang.
	if err := repo.CheckDetailTable(ctx); err != nil {
		print("  [BELUM] Tabel popup Detail tidak dapat dibaca: %v", err)
		print("            Hanya tombol Detail yang terdampak; daftarnya tetap jalan.")
		return
	}
	print("  [ok]    Tabel popup Detail dapat dibaca")

	if len(result.Items) == 0 {
		return
	}

	// Popup dijalankan atas klaim NYATA dari hasil di atas — bukan nomor karangan. Nomor
	// karangan akan selalu menjawab "tidak ditemukan", dan jawaban itu tidak membuktikan
	// kelima kueri popup berjalan; ia justru menyembunyikan gabungan yang rusak.
	sample := result.Items[0].ClaimNumber
	detail, found, err := repo.FindDetail(ctx,
		inboxosclaimpercabang.Query{Branch: resolved}, sample)
	switch {
	case err != nil:
		print("  [GAGAL] Kueri popup Detail gagal untuk klaim contoh: %v", err)
		return
	case !found:
		print("  [GAGAL] Klaim %s ada di daftar tetapi TIDAK ditemukan popup Detail.", sample)
		print("            Kedua penyaring seharusnya sama persis; selisihnya berarti")
		print("            salah satu kueri kehilangan penyaring cabang atau outstanding.")
		return
	}
	print("  [ok]    Popup Detail klaim %s: %d objek · %d catatan progres · %d pesan adjuster",
		sample, len(detail.Objects), len(detail.ProgressHistory),
		len(detail.AdjusterMessages))

	// Popup TIDAK boleh menjawab klaim cabang lain. Diuji dengan nomor yang memang ada,
	// tetapi diminta atas nama cabang yang berbeda — satu-satunya bentuk kebocoran `R-20`
	// yang tidak menghasilkan galat apa pun bila terjadi.
	lain := inboxosclaimpercabang.Branch{Code: "000000", Name: "CABANG BUKAN MILIKNYA"}
	if _, bocor, err := repo.FindDetail(ctx,
		inboxosclaimpercabang.Query{Branch: lain}, sample); err != nil {
		print("  [GAGAL] Uji batas cabang popup Detail gagal dijalankan: %v", err)
	} else if bocor {
		print("  [GAGAL] Popup Detail mengembalikan klaim %s untuk cabang LAIN.", sample)
		print("            Ini kebocoran data antarbadan hukum (R-20), bukan cacat tampilan.")
	} else {
		print("  [ok]    Popup Detail menolak klaim yang bukan milik cabang pemanggil")
	}
}

// checkPLADLAQueue menjalankan ketiga kueri daftar modul Inbox PLA, DLA, Pre DLA.
//
// # Kenapa KETIGANYA, bukan satu
//
// Karena ketiganya membaca TABEL YANG BERBEDA — `T_PLALIST`, `T_DLALIST`, dan
// `T_PREDLALIST` — dan penyaringnya pun berbeda bentuk. Satu kueri yang berhasil tidak
// menyatakan apa pun tentang dua lainnya, dan ketiga tabel itu tidak punya DDL di export
// (`R-08`) sehingga nama kolomnya dibaca dari kueri Pega, bukan dari skema.
//
// Rentang tanggal ikut diuji pada satu daftar. Ia satu-satunya bagian yang memakai bind
// bertipe tanggal, dan itu kelas galat yang hanya muncul di Oracle — bukan di uji yang
// membaca teks SQL.
func checkPLADLAQueue(
	ctx context.Context,
	repo *inboxpladlapredlasql.Repo,
	print func(string, ...any),
) {
	caller := inboxpladlapredla.Caller{Login: "pemeriksa-kesiapan"}
	page := inboxpladlapredla.Pagination{Page: 1, Size: 5}

	sampleKey := ""
	sampleTab := inboxpladlapredla.Tab{}
	preDLAKey := ""
	failed := false

	for _, tab := range inboxpladlapredla.Tabs() {
		query, err := inboxpladlapredla.NewQuery(
			inboxpladlapredla.QueryInput{Tab: tab.Code}, caller)
		if err != nil {
			print("  [BELUM] Daftar %q tidak dapat disusun: %v", tab.Name, err)
			failed = true
			continue
		}

		result, err := repo.List(ctx, query, page)
		if err != nil {
			print("  [BELUM] Antrean %q tidak dapat dibaca: %v", tab.Name, err)
			print("            Periksa POOLDATA.T_PLALIST, T_DLALIST, dan T_PREDLALIST.")
			failed = true
			continue
		}

		print("  [ok]    Antrean Inbox %s dapat dibaca (%d baris)", tab.Name, result.Total)

		if sampleKey == "" && tab.HasDocuments() {
			for _, row := range result.Items {
				if row.ClaimKey != "" {
					sampleKey = row.ClaimKey
					sampleTab = tab
					break
				}
			}
		}

		if preDLAKey == "" && tab.HasPrintAction {
			for _, row := range result.Items {
				if row.ClaimKey != "" {
					preDLAKey = row.ClaimKey
					break
				}
			}
		}
	}

	if failed {
		return
	}

	// Rentang tanggal diuji TERPISAH, dan hanya sekali.
	//
	// Bentuk klausanya sama di ketiga kueri; yang diuji adalah apakah bind bertipe
	// tanggal diterima Oracle sama sekali.
	from := time.Now().AddDate(-1, 0, 0)
	to := time.Now()
	ranged, err := inboxpladlapredla.NewQuery(
		inboxpladlapredla.QueryInput{Tab: inboxpladlapredla.DefaultTab,
			From: &from, To: &to, Search: "PNC"},
		caller)
	if err != nil {
		print("  [BELUM] Penyaring rentang tanggal tidak dapat disusun: %v", err)
		return
	}
	if _, err := repo.List(ctx, ranged, page); err != nil {
		print("  [BELUM] Penyaring rentang tanggal ditolak basis data: %v", err)
		print("            Ia satu-satunya bind bertipe tanggal di modul ini.")
		return
	}
	print("  [ok]    Penyaring rentang tanggal dan pencarian diterima basis data")

	if sampleKey == "" {
		print("  [catatan] Tidak ada baris contoh; grid rincian tidak diuji.")
	} else if _, err := repo.Documents(ctx, sampleTab, sampleKey); err != nil {
		print("  [BELUM] Grid rincian %q tidak dapat dibaca: %v", sampleTab.Name, err)
	} else {
		print("  [ok]    Grid rincian Inbox %s dapat dibaca", sampleTab.Name)
	}

	// Panel "Print Pre DLA" diuji TERPISAH, dan itu bukan pengulangan grid rincian.
	//
	// Ia satu-satunya kueri modul ini yang menyentuh KEDUA tabel lampiran Pega
	// (`PC_LINK_ATTACHMENT` dan `PC_DATA_WORKATTACH`), dan satu-satunya yang memakai
	// `PXATTACHNAME` tanpa nama tabel — kolom yang tidak diketahui milik tabel mana
	// karena DDL keduanya tidak ada di export (`R-08`). Bila tebakan Oracle berbeda dari
	// dugaan kami, yang muncul adalah galat "column ambiguously defined", dan itu hanya
	// terlihat dengan menjalankannya sungguhan.
	if preDLAKey == "" {
		print("  [catatan] Tidak ada baris contoh Pre DLA; panel cetak tidak diuji.")
		return
	}

	rows, err := repo.PrintPreDLA(ctx, preDLAKey)
	if err != nil {
		print("  [BELUM] Panel \"Print Pre DLA\" tidak dapat dibaca: %v", err)
		print("            Periksa PXATTACHNAME — ia dipakai tanpa nama tabel, persis " +
			"seperti kueri Pega.")
		return
	}
	print("  [ok]    Panel \"Print Pre DLA\" dapat dibaca (%d baris)", len(rows))

	if len(rows) > 0 {
		return
	}

	// Kosong BUKAN kegagalan: klaim itu boleh saja belum punya Pre-DLA yang lampirannya
	// cocok. Tetapi kosong punya EMPAT sebab yang tampak sama, dan menebaknya memakan
	// waktu lebih lama daripada menghitungnya.
	print("  [catatan] Panelnya kosong untuk klaim contoh. Menghitung tahap mana yang " +
		"menggugurkan barisnya:")

	diag, err := repo.DiagnosePrintPreDLA(ctx, preDLAKey)
	if err != nil {
		print("            Diagnosa tidak dapat dijalankan: %v", err)
		return
	}

	print("            Pre-DLA pada klaim ini        : %d", diag.PreDLARows)
	print("            NODLA tepat 11 karakter       : %d", diag.AdviceNoIs11)
	print("            punya lampiran apa pun        : %d", diag.HasAttachment)
	print("            lampiran berkategori 'DLA'    : %d", diag.CategoryIsDLA)

	switch {
	case diag.PreDLARows == 0:
		print("            -> Klaim contoh memang belum punya Pre-DLA. Bukan cacat.")
	case diag.AdviceNoIs11 == 0:
		// Sebab ini yang paling sering, dan yang paling tidak terduga.
		print("            -> SEBABNYA DI SINI. Pencocokan nama berkas Pega " +
			"(SUBSTR(PXATTACHNAME,-15,11) = NODLA) hanya dapat cocok bila NODLA " +
			"tepat 11 karakter. Tidak ada satu pun yang 11 karakter, sehingga " +
			"panel TIDAK AKAN PERNAH berisi — di Pega sekalipun.")
	case diag.CategoryIsDLA == 0 && diag.HasAttachment > 0:
		print("            -> Lampirannya ada tetapi TIDAK berkategori 'DLA'.")
	case diag.HasAttachment == 0:
		print("            -> Belum ada lampiran sama sekali pada klaim ini.")
	default:
		print("            -> Ketiga tahap awal lolos, sehingga yang menggugurkan " +
			"adalah pencocokan nama berkasnya. Periksa akhiran nama berkas: " +
			"ia harus NODLA ditambah tepat 4 karakter, misalnya '.pdf'.")
	}
}

// checkPLADLAReinsurer menjalankan kueri modul Inbox PLA DLA — layar milik reasuradur.
//
// # Yang diuji di sini BERBEDA dari modul di atasnya
//
// Seluruh kuerinya menyaring lewat `POOLDATA.T_REINSURER.LOGIN`, sehingga yang dibuktikan
// bukan hanya "kueri berjalan" melainkan juga "rantai reasuradurnya diterima". Login
// pemeriksa TIDAK terdaftar sebagai mitra, dan itu memang yang diharapkan: daftarnya
// menjawab nol baris, bukan galat.
//
// Nol baris di sini BUKAN tanda kegagalan. Yang gagal adalah kuerinya yang ditolak.
func checkPLADLAReinsurer(
	ctx context.Context,
	repo *inboxpladlasql.Repo,
	print func(string, ...any),
) {
	const login = "pemeriksa-kesiapan"

	codes, err := repo.ReinsurerCodes(ctx, login)
	if err != nil {
		print("  [BELUM] POOLDATA.T_REINSURER tidak dapat dibaca: %v", err)
		print("            Tanpa tabel itu, layar Inbox PLA DLA menolak setiap mitra.")
		return
	}
	print("  [ok]    POOLDATA.T_REINSURER dapat dibaca (%d kode untuk login uji)",
		len(codes))

	// Kode contoh dipakai supaya kuerinya benar-benar dijalankan dengan penyaring yang
	// terisi. Login pemeriksa tidak terdaftar, sehingga tanpa ini seluruh kueri berjalan
	// dengan senarai kosong dan bagian penyaringnya tidak pernah teruji.
	probe := codes
	if len(probe) == 0 {
		probe = []string{"PEMERIKSA"}
	}

	caller := inboxpladla.Caller{Login: login}
	page := inboxpladla.Pagination{Page: 1, Size: 5}

	for _, tab := range inboxpladla.Tabs() {
		query, err := inboxpladla.NewQuery(
			inboxpladla.QueryInput{Tab: tab.Code}, caller, probe)
		if err != nil {
			print("  [BELUM] Daftar reasuradur %q tidak dapat disusun: %v", tab.Name, err)
			continue
		}

		result, err := repo.List(ctx, query, page)
		if err != nil {
			print("  [BELUM] Daftar reasuradur %q tidak dapat dibaca: %v", tab.Name, err)
			if tab.Source == inboxpladla.SourceCommunication {
				print("            Ketiga daftar komunikasi membaca " +
					"POOLDATA.M_KOMUNIKASI_PNC.")
			}
			continue
		}
		print("  [ok]    Daftar reasuradur %s dapat dibaca (%d baris)",
			tab.Name, result.Total)

		if _, err := repo.Counts(ctx, query); err != nil {
			print("  [BELUM] Ringkasan status %q tidak dapat dihitung: %v", tab.Name, err)
		}
	}

	checkPLADLADetail(ctx, repo, login, probe, print)
}

// checkPLADLADetail menjalankan kueri layar RINCIAN modul Inbox PLA DLA.
//
// # Kenapa ia terpisah, dan kenapa ia penting
//
// Keenam kuerinya menyentuh tabel yang TIDAK disentuh layar induknya — `T_DOC_REAS`,
// `DATA_ATTACHFILE`, `M_KOMUNIKASI_PNC`, ditambah dua master jenis dokumen. Hak baca yang
// kurang pada salah satunya tidak terlihat saat layar induk dibuka: daftarnya tergambar
// lengkap, dan galatnya baru muncul ketika seorang mitra menekan "Detail Claim".
//
// Kunci klaim yang dipakai adalah kunci KARANGAN yang pasti tidak ada. Yang diuji adalah
// keterbacaan tabelnya, bukan isinya — sehingga "tidak ditemukan" di sini adalah hasil
// yang BENAR, bukan kegagalan.
func checkPLADLADetail(
	ctx context.Context,
	repo *inboxpladlasql.Repo,
	login string,
	codes []string,
	print func(string, ...any),
) {
	scope := inboxpladla.DetailScope{
		ClaimKey:       "ASM-FW-GCNMFW-WORK PEMERIKSA-KESIAPAN",
		Login:          login,
		ReinsurerCodes: codes,
	}

	if _, err := repo.ClaimHeader(ctx, scope); err != nil &&
		!errors.Is(err, inboxpladla.ErrRowNotFound) {
		print("  [BELUM] Kepala klaim rincian tidak dapat dibaca: %v", err)
	} else {
		print("  [ok]    Kepala klaim rincian dapat dibaca")
	}

	for _, kind := range []inboxpladla.AdviceKind{
		inboxpladla.AdviceKindPLA, inboxpladla.AdviceKindDLA,
	} {
		if _, err := repo.Advices(ctx, scope, kind); err != nil {
			print("  [BELUM] Grid %s rincian tidak dapat dibaca: %v", kind, err)
		} else {
			print("  [ok]    Grid %s rincian dapat dibaca", kind)
		}
	}

	if _, err := repo.Documents(
		ctx, scope, "PEMERIKSA", inboxpladla.AdviceKindPLA,
	); err != nil {
		print("  [BELUM] Dokumen rincian tidak dapat dibaca: %v", err)
		print("            Dibutuhkan hak SELECT atas POOLDATA.T_DOC_REAS,")
		print("            POOLDATA.DATA_ATTACHFILE, POOLDATA.V_LST_DOC_TYPE, dan")
		print("            POOLDATA.V_LST_DET_TYPE_DOC.")
	} else {
		print("  [ok]    Dokumen rincian dapat dibaca")
	}

	// Isi dokumen dibaca dari kolom BLOB. Hak baca atasnya dapat berbeda dari hak baca
	// kolom lain pada tabel yang sama, dan galatnya hanya muncul saat berkasnya diunduh.
	if _, err := repo.DocumentContent(ctx, scope, "0"); err != nil &&
		!errors.Is(err, inboxpladla.ErrDocumentNotFound) {
		print("  [BELUM] Isi dokumen tidak dapat dibaca: %v", err)
	} else {
		print("  [ok]    Isi dokumen dapat dibaca")
	}

	if _, err := repo.Conversations(ctx, scope); err != nil {
		print("  [BELUM] Riwayat komunikasi tidak dapat dibaca: %v", err)
		print("            Dibutuhkan hak SELECT atas POOLDATA.M_KOMUNIKASI_PNC.")
		return
	}
	print("  [ok]    Riwayat komunikasi dapat dibaca")

	// Pernyataan BALASAN sengaja TIDAK dijalankan.
	//
	// Ia satu-satunya pernyataan yang MENULIS di modul ini, dan pemeriksa kesiapan tidak
	// boleh menulis ke tabel yang masih dimiliki Pega (`P-1`). Hak tulisnya baru akan
	// terlihat saat seorang mitra benar-benar membalas — dan itu diterima secara sadar.
	print("  [catatan] Hak TULIS balasan komunikasi tidak diuji di sini: " +
		"pemeriksa kesiapan tidak menulis apa pun.")
}

// checkCaseStudyClaim membuktikan keempat tabel layar Case Study Claim dapat dibaca.
//
// # Kenapa pemeriksaan ini berharga
//
// Kuerinya menyentuh EMPAT tabel sekaligus, dan salah satunya — `T_CLAIM_ADJUSTMENT` —
// hanya muncul di dalam subkueri. Hak baca yang kurang pada tabel itu tidak terlihat saat
// layar dibuka: layarnya tergambar, penyaringnya terisi, dan galatnya baru muncul saat
// pengguna menekan "Lihat Data".
//
// Satu kolom patut disebut khusus: `PEGA_DASHBOARDPNC.THNREGIS`. Ia yang dibandingkan
// penyaring rentang, dan TIPENYA BELUM PERNAH DITERIMA (`R-08`). Kueri ini
// membandingkannya terhadap teks, persis seperti kueri lama membandingkannya terhadap
// keluaran `TO_CHAR`; bila kolomnya ternyata NUMBER, Oracle mengubah teksnya menjadi angka
// dan hasilnya sama. Yang tidak sama adalah bila ia bertipe lain sama sekali — dan itulah
// yang akan terlihat di sini.
func checkCaseStudyClaim(
	ctx context.Context,
	repo *casestudyclaimsql.Repo,
	print func(string, ...any),
) {
	if err := repo.CheckTable(ctx); err != nil {
		print("  [BELUM] Tabel Case Study Claim tidak dapat dibaca: %v", err)
		print("            Dibutuhkan hak SELECT atas POOLDATA.PEGA_DASHBOARDPNC,")
		print("            POOLDATA.T_CLAIM_PNC, POOLDATA.BUSINESS, dan")
		print("            POOLDATA.T_CLAIM_ADJUSTMENT. Keempatnya milik sistem lama dan")
		print("            tidak dibuat migrasi mana pun.")
		print("            Bila galatnya menyebut KOLOM, kolom itu memang tidak ada —")
		print("            seluruh nama kolom modul ini dibaca dari kueri Pega, bukan dari")
		print("            DDL, yang belum pernah diterima (`R-08`).")
		return
	}
	print("  [ok]    Keempat tabel Case Study Claim dapat dibaca")

	print("            Catatan: layar ini hanya memuat klaim yang salah satu baris")
	print("            settlement-nya melampaui Rp 5.000.000.000. Daftar yang kosong pada")
	print("            periode tertentu karena itu jawaban yang benar, bukan kerusakan.")
	print("            Modul ini MENULIS satu kolom — T_CLAIM_PNC.REMARKRECOMENDATION —")
	print("            dan penulisan itu menuntut serah-terima kepemilikan tulis (`D-63`).")
}

// checkPLADLACommunicationFunnel menjelaskan MENGAPA sebuah daftar komunikasi kosong.
//
// # Kenapa langkah ini ada
//
// Ketiga daftar komunikasi (`NOT ANSWERED`, `NOT REPLIED FROM ASM`, `REPLIED FROM ASM`)
// menyaring lewat EMPAT syarat berturut-turut, dan ketika hasilnya nol, layar tidak dapat
// membedakan yang mana penyebabnya:
//
//  1. `SENDER` / `COMMUNICATE_TO` cocok dengan login pemanggil
//  2. `KOMUNIKASISTATUS` bernilai `0` atau `1` — NULL tidak cocok dengan keduanya
//  3. `CASEID` berpasangan dengan satu baris `T_CLAIM_PNC`
//  4. klaim itu belum `Resolved-Completed` / `Resolved-Rejected`
//
// "Daftar kosong" karena syarat 1 berarti **login mitranya salah**; karena syarat 2 berarti
// **datanya yang tidak lengkap**; karena syarat 4 berarti **klaimnya memang sudah selesai**.
// Ketiganya menuntut tindakan yang sama sekali berbeda, dan menebaknya dari layar tidak
// mungkin.
//
// Langkah ini menghitung keempatnya sebagai CORONG, sehingga yang terbaca bukan "nol"
// melainkan pada langkah mana angkanya jatuh ke nol.
//
// # Ia tidak menulis apa pun
//
// Hanya `SELECT COUNT(...)`. Ia aman dijalankan terhadap basis data yang dipakai bersama
// Pega (`P-1`).
func checkPLADLACommunicationFunnel(
	ctx context.Context,
	primary *sql.DB,
	login string,
	print func(string, ...any),
) {
	clean := strings.TrimSpace(login)
	if clean == "" {
		print("  [lewat] Corong daftar komunikasi tidak diperiksa — jalankan ulang " +
			"dengan -login <login mitra> untuk menjelaskan daftar yang kosong.")
		return
	}

	// Satu kueri untuk kedua sisi percakapan. Dipisah menjadi dua kueri akan membuat
	// keduanya dapat dibaca pada keadaan basis data yang berbeda bila ada yang menulis
	// di antaranya.
	const funnel = `
SELECT COUNT(CASE WHEN UPPER(TRIM(k.SENDER))        = UPPER(TRIM(:1)) THEN 1 END),
       COUNT(CASE WHEN UPPER(TRIM(k.COMMUNICATE_TO)) = UPPER(TRIM(:2)) THEN 1 END),
       COUNT(CASE WHEN UPPER(TRIM(k.SENDER)) = UPPER(TRIM(:3))
                   AND k.KOMUNIKASISTATUS IS NULL THEN 1 END),
       COUNT(CASE WHEN UPPER(TRIM(k.SENDER)) = UPPER(TRIM(:4))
                   AND k.KOMUNIKASISTATUS = '0' THEN 1 END),
       COUNT(CASE WHEN UPPER(TRIM(k.SENDER)) = UPPER(TRIM(:5))
                   AND k.KOMUNIKASISTATUS = '1' THEN 1 END),
       COUNT(CASE WHEN UPPER(TRIM(k.SENDER)) = UPPER(TRIM(:6))
                   AND k.KOMUNIKASISTATUS = '0'
                   AND c.CLAIMID IS NOT NULL THEN 1 END),
       COUNT(CASE WHEN UPPER(TRIM(k.SENDER)) = UPPER(TRIM(:7))
                   AND k.KOMUNIKASISTATUS = '0'
                   AND c.STATUSWORK NOT IN ('Resolved-Completed', 'Resolved-Rejected')
                  THEN 1 END)
  FROM POOLDATA.M_KOMUNIKASI_PNC k
  LEFT JOIN POOLDATA.T_CLAIM_PNC c ON c.CLAIMID = k.CASEID`

	var (
		asSender, asRecipient        int
		statusNull, status0, status1 int
		withClaim, claimOpen         int
	)

	row := primary.QueryRowContext(ctx, funnel,
		clean, clean, clean, clean, clean, clean, clean)
	if err := row.Scan(&asSender, &asRecipient, &statusNull,
		&status0, &status1, &withClaim, &claimOpen); err != nil {
		print("  [BELUM] Corong daftar komunikasi tidak terbaca: %v", err)
		return
	}

	print("  [info]  Corong daftar komunikasi untuk login %q:", clean)
	print("            percakapan dengan SENDER          = login : %d", asSender)
	print("            percakapan dengan COMMUNICATE_TO  = login : %d", asRecipient)

	if asSender == 0 && asRecipient == 0 {
		print("  [BELUM] Login itu tidak muncul sebagai pihak mana pun di " +
			"POOLDATA.M_KOMUNIKASI_PNC.")
		print("            Ketiga daftar komunikasi memang kosong, dan itu BENAR.")
		print("            Bila Anda meminjam lewat REAS_MITRA_PENGEMBANGAN, mitra yang " +
			"dipinjam bukan pemilik percakapan yang Anda cari.")
		showCommunicationParties(ctx, primary, print)
		return
	}

	print("            di antara SENDER — KOMUNIKASISTATUS NULL   : %d", statusNull)
	print("            di antara SENDER — KOMUNIKASISTATUS '0'    : %d  (NOT REPLIED FROM ASM)", status0)
	print("            di antara SENDER — KOMUNIKASISTATUS '1'    : %d  (REPLIED FROM ASM)", status1)
	print("            di antara yang '0' — CASEID punya klaim    : %d", withClaim)
	print("            di antara yang '0' — klaimnya belum selesai: %d  <- yang TAMPIL", claimOpen)

	switch {
	case statusNull > 0 && status0 == 0:
		print("  [BELUM] %d percakapan ber-KOMUNIKASISTATUS NULL, bukan '0'.", statusNull)
		print("            Baris NULL tidak cocok dengan '0' maupun '1', sehingga ia tidak " +
			"muncul di satu pun dari ketiga daftar.")
		print("            Ini PERILAKU YANG SAMA dengan Pega — kuerinya pun membandingkan " +
			"nilai, bukan menangani NULL. Perbaikannya ada di sisi data.")
	case status0 > 0 && withClaim == 0:
		print("  [BELUM] CASEID percakapan itu tidak berpasangan dengan satu pun baris " +
			"POOLDATA.T_CLAIM_PNC.")
		print("            Periksa apakah CASEID memuat awalan kelas Pega " +
			"('ASM-FW-GCNMFW-WORK ...') seperti T_CLAIM_PNC.CLAIMID.")
	case withClaim > 0 && claimOpen == 0:
		print("  [info]  Klaimnya sudah Resolved-Completed/Resolved-Rejected, sehingga " +
			"sengaja tidak ditampilkan.")
		print("            Pega menyaringnya dengan cara yang sama.")
	case claimOpen > 0:
		print("  [ok]    %d baris SEHARUSNYA tampil di NOT REPLIED FROM ASM.", claimOpen)
		print("            Bila layarnya tetap kosong, selisihnya ada di lapisan aplikasi " +
			"— bukan di kueri.")
	}
}

// showCommunicationParties menyebut beberapa login yang BENAR-BENAR punya percakapan.
//
// Tanpa ini, "login Anda tidak punya percakapan" tetap menyisakan pertanyaan berikutnya:
// lalu login siapa yang punya. Jawabannya dibutuhkan untuk mengisi
// REAS_MITRA_PENGEMBANGAN, dan mencarinya dengan menebak satu per satu tidak masuk akal.
func showCommunicationParties(
	ctx context.Context,
	primary *sql.DB,
	print func(string, ...any),
) {
	const parties = `
SELECT pihak, jumlah FROM (
  SELECT TRIM(k.SENDER) AS pihak, COUNT(*) AS jumlah
    FROM POOLDATA.M_KOMUNIKASI_PNC k
   WHERE k.SENDER IS NOT NULL
   GROUP BY TRIM(k.SENDER)
   ORDER BY COUNT(*) DESC)
 FETCH NEXT 10 ROWS ONLY`

	rows, err := primary.QueryContext(ctx, parties)
	if err != nil {
		print("            (daftar pengirim tidak terbaca: %v)", err)
		return
	}
	defer rows.Close()

	print("            Login yang punya percakapan sebagai SENDER (10 terbanyak):")
	found := false
	for rows.Next() {
		var pihak sql.NullString
		var jumlah int
		if err := rows.Scan(&pihak, &jumlah); err != nil {
			break
		}
		found = true
		print("              %-40s %d", strings.TrimSpace(pihak.String), jumlah)
	}
	if !found {
		print("              (tidak ada satu pun)")
	}
}

// checkKomiteInbox menjelaskan MENGAPA layar Inbox Komite berperilaku seperti yang terlihat.
//
// # Kenapa langkah ini ada
//
// Inbox Komite adalah satu-satunya inbox yang selama ini TIDAK punya langkah periksa, dan
// ketiadaannya persis yang membuat kegagalan 2026-09-28 tidak terlihat sampai dilaporkan
// pengguna: ketiga kueri daftarnya menggabungkan `POOLDATA.CPNC_KOMITE_KEPUTUSAN`, dan
// tabel itu dibuat migrasi `0004` yang belum pernah dijalankan di lingkungan mana pun.
// Akibatnya seluruh layar mati dengan `ORA-00942`, sementara 1.542 kasus komite di tabel
// warisan baik-baik saja.
//
// `inbox_check_table` yang sudah ada tidak menangkapnya — ia hanya menyentuh tabel Pega.
//
// # Yang dibedakan di sini
//
// "Daftar kosong" pada layar ini punya empat sebab yang menuntut tindakan berbeda, dan
// tidak satu pun dapat dibedakan dari layarnya:
//
//  1. tabel warisan tidak dapat dibaca        → hak akses akun aplikasi
//  2. jejak keputusan belum ada               → migrasi 0004, urusan DBA (`D-63`)
//  3. tidak ada kasus Work-Komite sama sekali  → basis datanya memang kosong
//  4. ada kasusnya, tetapi bukan milik login itu → pemetaan identitas ke OPERATOR_ID
//
// Butir 4 sangat mungkin terjadi selama pemetaan identitas HCC/HCQ ke `OPERATOR_ID` belum
// ada (`ADR-0024`), dan ia paling mudah tertukar dengan butir 3.
//
// # Ia tidak menulis apa pun
//
// Hanya SELECT. Aman dijalankan terhadap basis data yang dipakai bersama Pega (`P-1`).
func checkKomiteInbox(
	ctx context.Context,
	primary *sql.DB,
	login string,
	print func(string, ...any),
) {
	kasus := komitesql.NewInboxRepo(primary)
	keputusan := komitesql.NewDecisionRepo(primary)
	transfer := komitesql.NewTransferRepo(primary)

	if err := kasus.CheckTables(ctx); err != nil {
		print("  [BELUM] Tabel warisan Inbox Komite tidak dapat dibaca: %v", err)
		print("            Dibutuhkan hak SELECT atas DATAPEGA.PC_ASM_FW_GCNMFW_WORK,")
		print("            DATAPEGA.PC_ASSIGN_WORKLIST, dan POOLDATA.T_CLAIM_KOMITE_LIST.")
		return
	}
	print("  [ok]    Tabel warisan Inbox Komite dapat dibaca (ketiganya)")

	// Rincian "Lihat Detail Transfer" memakai DUA tabel lain, dan ketiadaannya berakibat
	// berbeda: daftarnya tetap jalan, yang mati hanya layar rinciannya.
	if err := transfer.CheckTables(ctx); err != nil {
		print("  [BELUM] Tabel rincian transfer tidak dapat dibaca: %v", err)
		print("            Dibutuhkan hak SELECT atas POOLDATA.T_CLAIM_ADJUSTMENT,")
		print("            T_CLAIM_KOMITE_LIST, T_CLAIM_PNC, dan T_CLAIM_OBJECTCOVERAGE.")
		print("            Daftar inbox TIDAK terpengaruh; yang gagal hanya layar rincian")
		print("            saat sebuah case ditekan.")
	} else {
		print("  [ok]    Keempat tabel rincian transfer dapat dibaca")
		print("            T_CLAIM_ADJUSTMENT · T_CLAIM_KOMITE_LIST · T_CLAIM_PNC ·")
		print("            T_CLAIM_OBJECTCOVERAGE (blok klaim dan analisis komite)")
	}

	// Jejak keputusan diperiksa TERPISAH dari tabel warisan, karena akibat ketiadaannya
	// berbeda sama sekali: yang satu mematikan layar, yang lain hanya mematikan tombolnya.
	jejakSiap := true
	if err := keputusan.CheckTable(ctx); err != nil {
		jejakSiap = false
		print("  [BELUM] POOLDATA.CPNC_KOMITE_KEPUTUSAN belum dapat dipakai: %v", err)
		print("            Jalankan migrations/0004_komite_keputusan.up.sql (DBA, `D-63`).")
		print("            AKIBATNYA SEKARANG: daftar kasus TETAP tampil — kueri daftar tidak")
		print("            menyentuh tabel ini sama sekali (Work Owner, 2026-09-28) — tetapi")
		print("            keputusan komite TIDAK dapat dicatat, dan kotak Diterima/Ditolak")
		print("            hanya berisi riwayat keputusan Pega.")
	} else {
		print("  [ok]    POOLDATA.CPNC_KOMITE_KEPUTUSAN dapat dibaca; keputusan dapat dicatat")
	}

	// Jumlah seluruh kasus dihitung TANPA memandang pemilik.
	//
	// Inilah yang memisahkan "basis datanya memang kosong" dari "ada pekerjaannya, tetapi
	// bukan milik login yang diperiksa" — dua keadaan yang di layar sama-sama terbaca
	// sebagai tabel kosong, dan tindakannya sama sekali berbeda.
	var seluruhnya int
	err := primary.QueryRowContext(ctx,
		`SELECT COUNT(1) FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK
          WHERE PXOBJCLASS = 'ASM-FW-GCNMFW-Work-Komite'`).Scan(&seluruhnya)
	if err != nil {
		print("  [BELUM] Jumlah kasus komite tidak dapat dihitung: %v", err)
		return
	}
	print("  [ok]    Kasus Work-Komite di basis data ini: %d", seluruhnya)

	// Penyaring tahun `F1` pada InboxRegisterKomite_RD dihitung TERPISAH.
	//
	// Ia memotong daftar secara berarti — pada ASM, 417 menjadi 189 — dan tanpa angka ini
	// "kenapa case lama tidak muncul" tidak dapat dijawab selain dengan menebak.
	var lolosTahun int
	if err := primary.QueryRowContext(ctx,
		`SELECT COUNT(1)
           FROM DATAPEGA.PC_ASSIGN_WORKLIST w
           JOIN DATAPEGA.PC_ASM_FW_GCNMFW_WORK a ON a.PZINSKEY = w.PXREFOBJECTKEY
          WHERE w.PXOBJCLASS = 'Assign-Worklist'
            AND a.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-Komite'
            AND a.PXCREATEDATETIME >= :1`,
		komite.InboxEarliestCreatedAt()).Scan(&lolosTahun); err == nil {
		print("  [ok]    Ditugaskan DAN dibuat sejak %d: %d — inilah yang dapat muncul di inbox",
			komite.InboxEarliestYear, lolosTahun)
		print("            Penyaring tahun berasal dari `F1` pada InboxRegisterKomite_RD;")
		print("            case yang lebih tua memang TIDAK pernah muncul di layar Pega.")
	}
	if seluruhnya == 0 {
		print("            Basis data ini memang belum punya kasus komite; inbox siapa pun")
		print("            akan kosong, dan itu BUKAN cacat.")
		return
	}

	if login == "" {
		print("  [lewat] Isi inbox tidak diperiksa — jalankan ulang dengan -login <operator>")
		print("            untuk membedakan 'tidak ada pekerjaan' dari 'identitasnya tidak")
		print("            cocok dengan satu pun OPERATOR_ID di data warisan' (`ADR-0024`).")
		return
	}

	// Ketiga kotak dihitung lewat kueri yang BENAR-BENAR dipakai layar, bukan lewat kueri
	// tiruan. Dengan begitu langkah ini sekaligus membuktikan pernyataannya sah, penanda
	// bind-nya benar, dan pembacaan kolomnya cocok dengan tipe kolom yang sebenarnya —
	// termasuk pada jalur `_warisan` yang baru.
	penyaring := komite.InboxFilter{Operator: login}
	ringkasan, err := kasus.Summarize(ctx, penyaring)
	if err != nil {
		print("  [BELUM] Kueri ringkasan inbox komite gagal: %v", err)
		return
	}
	print("  [ok]    Inbox %s — outstanding %d · diterima %d · ditolak %d",
		komite.OperatorKey(login),
		ringkasan.Outstanding, ringkasan.Accepted, ringkasan.Rejected)

	halaman, err := kasus.ListCases(ctx, penyaring)
	if err != nil {
		print("  [BELUM] Kueri daftar inbox komite gagal: %v", err)
		return
	}
	print("  [ok]    Kueri daftar berjalan; %d baris cocok, halaman pertama %d baris",
		halaman.Total, len(halaman.Cases))

	if halaman.Total == 0 && seluruhnya > 0 {
		print("  [WASPADA] Ada %d kasus komite, tetapi tidak satu pun milik %q.",
			seluruhnya, komite.OperatorKey(login))
		print("            Kemungkinan terbesarnya BUKAN kueri, melainkan pemetaan identitas:")
		print("            inbox menyaring PXASSIGNEDOPERATORID, dan pemetaan HCC/HCQ ke")
		print("            OPERATOR_ID belum ada (`ADR-0024`). Coba ulangi dengan OPERATOR_ID")
		print("            seperti yang tertulis di DATAPEGA.PC_ASSIGN_WORKLIST.")
	}

	if !jejakSiap {
		print("            Catatan: angka di atas dihitung TANPA keputusan milik aplikasi ini,")
		print("            karena jejaknya belum ada. Ia akan berubah setelah migrasi 0004")
		print("            dijalankan dan keputusan pertama tercatat.")
	}
}

// checkInputAcceptation memeriksa modul Acceptation Claim — akseptasi klaim treaty non-prop.
//
// # Kenapa pemeriksaan ini penting justru di modul ini
//
// Karena ~50 isian dan 13 gridnya dibaca dari SATU dokumen JSON yang bentuknya belum pernah
// diperiksa (`R-08`). Jalur yang salah tidak menghasilkan galat apa pun — ia hanya
// mengosongkan selnya. Menghitung berapa jalur yang benar-benar ditemukan adalah satu-satunya
// cara membedakan "klaim ini memang belum diisi" dari "seluruh jalurnya salah".
//
// Nomor klaimnya diambil dari antrean Inbox Claim Treaty Non Prop, bukan dikarang, supaya
// pemeriksaan ini tidak pernah menyentuh nomor yang tidak ada.
func checkInputAcceptation(
	ctx context.Context,
	repo *inputacceptationsql.Repo,
	queue *inboxclaimtreatynonpropsql.Repo,
	print func(string, ...any),
) {
	if err := repo.CheckTables(ctx); err != nil {
		print("  [BELUM] Tabel akseptasi klaim treaty non-prop tidak dapat dibaca: %v", err)
		print("            Modul ini TIDAK menuntut migrasi — kedua tabelnya milik Pega.")
		print("            Periksa hak SELECT akun aplikasi atas")
		print("            DATAPEGA.PC_ASM_FW_GCNMFW_WORK dan POOLDATA.JSON_KLAIM.")
		return
	}
	print("  [ok]    Tabel akseptasi klaim treaty non-prop dapat dibaca")

	// BEBERAPA klaim dicoba, bukan satu.
	//
	// Antrean ini memuat baris yang objek kerjanya tidak punya pasangan — `CLMNP-1` salah
	// satunya — dan klaim seperti itu tidak punya dokumen sama sekali. Satu sampel buruk
	// membuat pemeriksaan ini melaporkan "0 dari 40 jalur ditemukan" yang tidak menyatakan
	// apa pun tentang benar-tidaknya jalur di section.go.
	samples := nonPropClaimIDs(ctx, queue, 10)
	if len(samples) == 0 {
		print("            Tidak ada klaim treaty non-prop di antrean untuk diperiksa. Itu")
		print("            BUKAN kegagalan — hanya berarti jalur dokumennya belum teruji.")
		return
	}

	// Isian yang terhalang tidak ikut dihitung: ia memang tidak punya jalur.
	expected := 0
	for _, field := range inputacceptation.Fields() {
		if !field.Blocked {
			expected++
		}
	}

	var (
		sample string
		detail inputacceptation.Detail
		found  int
	)

	for _, candidate := range samples {
		q, err := inputacceptation.NewQuery(
			candidate, inputacceptation.Caller{Login: "-periksa"})
		if err != nil {
			continue
		}

		got, err := repo.Find(ctx, q)
		if err != nil {
			print("  [GAGAL] Akseptasi klaim %s tidak dapat dibaca: %v", candidate, err)
			print("            Bila galatnya menyebut JSON, isi")
			print("            POOLDATA.JSON_KLAIM.DATA_JSON kemungkinan bukan JSON yang sah.")
			print("            Perhatikan kolomnya DATA_JSON, BUKAN DATA_JSONBLOB yang")
			print("            dibaca layar Outstanding Claim — keduanya kolom berbeda.")
			return
		}

		// Klaim pertama yang BENAR-BENAR punya dokumen yang dipakai. Klaim tanpa dokumen
		// dilewati, bukan dilaporkan sebagai kegagalan jalur.
		sample, detail, found = candidate, got, len(got.Values)
		if found > 0 {
			break
		}
	}

	if found == 0 {
		print("  [PERIKSA] %d klaim dicoba dan TIDAK satu pun punya isi dokumen.",
			len(samples))
		print("            Dua kemungkinan, dan keduanya menuntut tindakan berbeda:")
		print("            klaim-klaim itu belum punya baris di POOLDATA.JSON_KLAIM, atau")
		print("            seluruh jalur di section.go salah. Bandingkan isi DATA_JSON")
		print("            salah satunya dengan internal/inputacceptation/section.go.")
		return
	}
	print("  [ok]    Akseptasi klaim %s terbaca: %d dari %d isian ditemukan di dokumen",
		sample, found, expected)

	switch {
	case found == 0:
		print("  [PERIKSA] TIDAK SATU PUN jalur ditemukan. Dua kemungkinan, dan keduanya")
		print("            menuntut tindakan berbeda: klaim ini belum punya baris di")
		print("            POOLDATA.JSON_KLAIM, atau seluruh jalur di section.go salah.")
		print("            Bandingkan isi DATA_JSON klaim ini dengan daftar jalur di")
		print("            internal/inputacceptation/section.go sebelum menyimpulkan.")
	case found*2 < expected:
		print("  [PERIKSA] Kurang dari separuh jalur ditemukan. Bentuk dokumen kemungkinan")
		print("            berbeda dari yang dibaca dari section — lihat log peringatan")
		print("            modul untuk rincian isian mana yang hilang.")
	}

	grids := 0
	for _, rows := range detail.Grids {
		if len(rows) > 0 {
			grids++
		}
	}
	print("  [ok]    %d dari %d tabel berisi baris pada klaim %s",
		grids, len(inputacceptation.GridList()), sample)
}

// nonPropClaimIDs mengambil beberapa nomor klaim treaty non-prop dari antrean teknik.
//
// Antrean teknik dipilih karena ia antrean BERSAMA: isinya tidak bergantung pada siapa yang
// menjalankan pemeriksaan, sedangkan antrean Admin menyaring menurut petugas.
//
// Diambil BEBERAPA, bukan satu, karena antrean ini memuat baris yang objek kerjanya tidak
// punya pasangan — dan klaim seperti itu tidak punya dokumen untuk diperiksa.
func nonPropClaimIDs(
	ctx context.Context, queue *inboxclaimtreatynonpropsql.Repo, limit int,
) []string {
	technical, found := inboxclaimtreatynonprop.FindTab(inboxclaimtreatynonprop.TabTechnical)
	if !found {
		return nil
	}

	page, err := queue.List(ctx,
		inboxclaimtreatynonprop.Query{Tab: technical},
		inboxclaimtreatynonprop.Pagination{Page: 1, Size: limit},
	)
	if err != nil {
		return nil
	}

	result := make([]string, 0, len(page.Items))
	for _, item := range page.Items {
		if item.ClaimID != "" {
			result = append(result, item.ClaimID)
		}
	}
	return result
}

// checkOutstandingClaim memeriksa modul Outstanding Claim — rincian klaim treaty.
//
// Tiga hal yang diperiksa, dan yang ketiga tidak dimiliki modul lain:
//
//  1. hak baca atas DATAPEGA.PC_ASM_FW_GCNMFW_WORK dan POOLDATA.JSON_KLAIM;
//  2. apakah kueri rinciannya benar-benar berjalan terhadap klaim yang ada;
//  3. BERAPA BANYAK jalur dokumen klaim yang ditemukan — karena bentuk dokumen itu belum
//     pernah diperiksa (`R-08`), dan layar berisi 97 isian kosong terbaca sama persis entah
//     karena klaimnya memang belum diisi atau karena seluruh jalurnya salah.
//
// Pemeriksaan ketiga itulah yang paling berguna di sini: ia menjawab "apakah kita membaca
// dokumen yang benar" sebelum ada satu pun pengguna yang membuka layarnya.
func checkOutstandingClaim(
	ctx context.Context,
	repo *outstandingclaimsql.Repo,
	queue *inboxclaimtreatypropsql.Repo,
	print func(string, ...any),
) {
	if err := repo.CheckTable(ctx); err != nil {
		print("  [BELUM] Tabel rincian klaim treaty tidak dapat dibaca: %v", err)
		print("            Modul ini TIDAK menuntut migrasi — kedua tabelnya milik Pega.")
		print("            Periksa hak SELECT akun aplikasi atas")
		print("            DATAPEGA.PC_ASM_FW_GCNMFW_WORK dan POOLDATA.JSON_KLAIM.")
		return
	}
	print("  [ok]    Tabel rincian klaim treaty dapat dibaca")

	// Satu klaim nyata dibutuhkan untuk memeriksa jalur dokumennya, dan nomornya tidak
	// dapat ditebak. Ia diambil dari antrean modul di atasnya — bukan dikarang — supaya
	// pemeriksaan ini tidak pernah menyentuh nomor klaim yang tidak ada.
	sample := firstTreatyClaimID(ctx, queue)
	if sample == "" {
		print("            Tidak ada klaim treaty di antrean untuk diperiksa. Itu BUKAN")
		print("            kegagalan — hanya berarti jalur dokumennya belum dapat diuji.")
		return
	}

	query, err := outstandingclaim.NewQuery(sample, outstandingclaim.Caller{Login: "-periksa"})
	if err != nil {
		print("  [GAGAL] Permintaan rincian tidak terbentuk: %v", err)
		return
	}

	detail, err := repo.Find(ctx, query)
	if err != nil {
		print("  [GAGAL] Rincian klaim %s tidak dapat dibaca: %v", sample, err)
		print("            Bila galatnya menyebut JSON, isi")
		print("            POOLDATA.JSON_KLAIM.DATA_JSONBLOB bukan JSON yang sah.")
		return
	}

	// Isian yang terhalang tidak ikut dihitung: ia memang tidak punya jalur.
	expected := 0
	for _, field := range outstandingclaim.Fields() {
		if !field.Blocked {
			expected++
		}
	}

	found := len(detail.Values)
	print("  [ok]    Rincian klaim %s terbaca: %d dari %d isian ditemukan di dokumen",
		sample, found, expected)

	switch {
	case found == 0:
		print("  [PERIKSA] TIDAK SATU PUN jalur ditemukan. Dua kemungkinan, dan keduanya")
		print("            menuntut tindakan berbeda: klaim ini belum punya baris di")
		print("            POOLDATA.JSON_KLAIM, atau seluruh jalur di section.go salah.")
		print("            Bandingkan isi DATA_JSONBLOB klaim ini dengan daftar jalur di")
		print("            internal/outstandingclaim/section.go sebelum menyimpulkan.")
	case found*2 < expected:
		print("  [PERIKSA] Kurang dari separuh jalur ditemukan. Bentuk dokumen kemungkinan")
		print("            berbeda dari yang dibaca dari section — lihat log peringatan")
		print("            modul untuk rincian isian mana yang hilang.")
	}

	grids := 0
	for _, rows := range detail.Grids {
		if len(rows) > 0 {
			grids++
		}
	}
	print("  [ok]    %d grid berisi baris pada klaim %s", grids, sample)
}

// firstTreatyClaimID mengambil satu nomor klaim treaty dari antrean, atau teks kosong.
//
// Ia memakai penyimpanan modul Inbox Claim Treaty Prop, bukan kueri tersendiri, supaya
// pemeriksaan ini tidak menambah satu pun kueri yang harus dipelihara — dan supaya nomor yang
// diperiksa memang nomor yang benar-benar tampil di layar antrean.
//
// Antrean TEKNIK yang dipakai, bukan antrean milik pemanggil: `-periksa` berjalan tanpa
// pengguna, sehingga antrean per orang selalu kosong baginya.
//
// Galat DITELAN di sini dan dijawab teks kosong. Kegagalan membaca antrean bukan temuan modul
// ini — ia sudah dilaporkan checkClaimTreatyProp tepat sebelumnya, dan melaporkannya dua kali
// membuat satu masalah terbaca sebagai dua.
func firstTreatyClaimID(ctx context.Context, queue *inboxclaimtreatypropsql.Repo) string {
	technical, found := inboxclaimtreatyprop.FindTab(inboxclaimtreatyprop.TabTechnical)
	if !found {
		return ""
	}

	page, err := queue.List(ctx,
		inboxclaimtreatyprop.Query{Tab: technical},
		inboxclaimtreatyprop.Pagination{Page: 1, Size: 1},
	)
	if err != nil || len(page.Items) == 0 {
		return ""
	}
	return page.Items[0].ClaimID
}

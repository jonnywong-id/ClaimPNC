package main

import (
	"bufio"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"claim-pnc/internal/auth"
	"claim-pnc/internal/auth/provider"
	"claim-pnc/internal/auth/repo/sqlstore"
	"claim-pnc/internal/inboxcloseclaim"
	"claim-pnc/internal/inboxcompliance"
	"claim-pnc/internal/inboxinvestigator"
	"claim-pnc/internal/inboxoutstanding"
	"claim-pnc/internal/inboxreceivetka"
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
	"claim-pnc/internal/riwayatklaim"

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
	"claim-pnc/internal/inboxclaimtreatynonprop"
	inboxclaimtreatynonpropsql "claim-pnc/internal/inboxclaimtreatynonprop/repo/sqlstore"
	"claim-pnc/internal/inboxclaimtreatyprop"
	inboxclaimtreatypropsql "claim-pnc/internal/inboxclaimtreatyprop/repo/sqlstore"
	inboxcloseclaimsql "claim-pnc/internal/inboxcloseclaim/repo/sqlstore"
	inboxcompliancesql "claim-pnc/internal/inboxcompliance/repo/sqlstore"
	inboxinvestigatorsql "claim-pnc/internal/inboxinvestigator/repo/sqlstore"
	inboxlaporanklaimsql "claim-pnc/internal/inboxlaporanklaim/repo/sqlstore"
	"claim-pnc/internal/inboxmanagerreceivepucl"
	inboxmanagerreceivepuclsql "claim-pnc/internal/inboxmanagerreceivepucl/repo/sqlstore"
	inboxoutstandingsql "claim-pnc/internal/inboxoutstanding/repo/sqlstore"
	inboxprogressclaimsql "claim-pnc/internal/inboxprogressclaim/repo/sqlstore"
	"claim-pnc/internal/inboxkomunikasicabang"
	inboxkomunikasicabangsql "claim-pnc/internal/inboxkomunikasicabang/repo/sqlstore"
	"claim-pnc/internal/inboxrclpucl"
	inboxrclpuclsql "claim-pnc/internal/inboxrclpucl/repo/sqlstore"
	inboxreceivetkasql "claim-pnc/internal/inboxreceivetka/repo/sqlstore"
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
	portalsql "claim-pnc/internal/portal/repo/sqlstore"
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
	checkClaimTreatyNonProp(ctx, inboxclaimtreatynonpropsql.NewRepo(primary), print)
	checkManagerReceivePUCL(ctx, inboxmanagerreceivepuclsql.NewRepo(primary), print)
	checkRCLPUCL(ctx, inboxrclpuclsql.NewRepo(primary), print)
	checkKomunikasiCabang(ctx,
		inboxkomunikasicabangsql.NewRepo(primary),
		inboxkomunikasicabangsql.NewBranchResolver(primary),
		print)
	checkInboxProgressClaim(ctx, inboxprogressclaimsql.NewRepo(primary), print)
	checkInboxAnalystDoctor(ctx, inboxanalystdoctorsql.NewRepo(primary), print)
	checkClaimReport(ctx, inboxlaporanklaimsql.NewRepo(primary, clock.System{}), print)
	checkOutstanding(ctx, inboxoutstandingsql.NewRepo(primary), login, print)
	checkClaimReportBranch(ctx, inboxlaporanklaimsql.NewBranchResolver(primary), print)
	checkCloseClaim(ctx,
		inboxcloseclaimsql.NewRepo(primary),
		inboxcloseclaimsql.NewRequestRepo(primary),
		print)

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

// checkClaimReport melaporkan kesiapan kedua tabel Inbox Laporan Klaim.
//
// Ia memeriksa DUA hal yang sifatnya berbeda, dan membedakannya penting:
//
//	POOLDATA.CPNC_LAPORAN_KLAIM   tabel BARU, dibuat migrasi 0003 — dibaca DAN ditulis
//	POOLDATA.T_CLAIMLIST_ADMIN    sumber daftar, diisi proses lain — hanya DIBACA
//
// Yang pertama belum ada sampai DBA menjalankan migrasinya; yang kedua sudah ada, dan
// kegagalannya berarti akun aplikasi tidak diberi hak baca. Dua sebab yang tampak mirip
// di layar tetapi perbaikannya berbeda jauh.
func checkClaimReport(ctx context.Context, repo *inboxlaporanklaimsql.Repo, print func(string, ...any)) {
	if err := repo.CheckTable(ctx); err != nil {
		print("  [BELUM] Inbox Laporan Klaim belum siap: %v", err)
		print("            CPNC_LAPORAN_KLAIM dibuat migrasi backend/migrations/0003,")
		print("            dan ia dijalankan di SETIAP portal entitas — bukan hanya portal utama.")
		print("            T_CLAIMLIST_ADMIN diisi proses lain dan hanya dibaca; bila justru ia")
		print("            yang gagal, yang kurang adalah hak baca akun aplikasi, bukan migrasinya.")
		return
	}
	print("  [ok]    kedua tabel Inbox Laporan Klaim dapat dibaca")
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

	if err := repo.CheckView(ctx); err != nil {
		print("  [BELUM] POOLDATA.V_MST_USER_TEKNIS tidak dapat dibaca: %v", err)
		print("            Daftar PIC Teknik dibaca dari view ini karena TOTAL_JOB tidak ada")
		print("            di tabelnya. Nama kolomnya belum terverifikasi dari export —")
		print("            mintakan definisinya ke DBA:")
		print("              SELECT text FROM all_views")
		print("               WHERE owner = 'POOLDATA' AND view_name = 'V_MST_USER_TEKNIS';")
		print("            Bila kolomnya berbeda, yang disesuaikan hanya kueri")
		print("            technician_list di repo/sqlstore/technician.sql.")
	} else {
		print("  [ok]    POOLDATA.V_MST_USER_TEKNIS dapat dibaca beserta TOTAL_JOB")
	}

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
// Keempat objeknya diperiksa TERPISAH, karena keempatnya gagal karena sebab yang berbeda
// dan menyatukan laporannya membuat pembaca menebak mana yang sebenarnya kurang:
//
//	V_LST_DOC_TRAVEL             hak baca belum diberikan, atau view-nya memang tidak ada
//	V_LST_DOC_TRAVEL_COVERAGE    idem
//	M_DOCTRAVEL                  milik modul Master Dokumen Travel; hanya dibaca di sini
//	M_PLANTRAVEL                 milik GISFW; hanya dibaca, dan belum pernah disentuh
//	                             aplikasi ini sampai modul ini ada
//
// # Yang sengaja TIDAK diperiksa di sini
//
// Tabel dasar dan urutan yang dipakai jalur tulis. Nama ketiganya belum terverifikasi
// (`R-16` — activity penyimpannya hilang dari export), dan memeriksa urutan berarti
// MENGHABISKAN satu nomor — efek samping yang tidak pantas dimiliki mode periksa.
// Verifikasinya ada di migrations/0006, dijalankan DBA sekali.
//
// Akibat yang harus disadari pembaca laporan ini: seluruh baris [ok] di bawah hanya
// membuktikan jalur BACA siap. Jalur TULIS belum terbukti sampai migrasi 0005 dijawab.
func checkTravelDocumentDetail(ctx context.Context, primary *sql.DB, print func(string, ...any)) {
	repo := daftardetaildokumentravelsql.NewRepo(primary)
	if err := repo.CheckTable(ctx); err != nil {
		print("  [BELUM] Daftar Detail Dokumen Travel belum siap: %v", err)
		print("            Kedua view V_LST_DOC_TRAVEL dan V_LST_DOC_TRAVEL_COVERAGE")
		print("            adalah objek warisan Pega. Mintakan GRANT SELECT untuk akun")
		print("            aplikasi — lihat migrations/0006. Selama belum diberikan,")
		print("            layarnya tidak dapat dipakai terhadap Oracle; bagian lain")
		print("            tetap jalan.")
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

	plan := daftardetaildokumentravelsql.NewPlanRepo(primary)
	if err := plan.CheckTable(ctx); err != nil {
		print("  [GAGAL] POOLDATA.M_PLANTRAVEL tidak dapat dibaca: %v", err)
		print("            Tabel itu milik GISFW dan HANYA DIBACA. Mintakan GRANT SELECT")
		print("            untuk akun aplikasi. Perhatikan juga nama kolomnya belum")
		print("            pernah diverifikasi — bila galatnya menyebut kolom, bukan")
		print("            tabel, lihat bagian M_PLANTRAVEL pada migrations/0006.")
		return
	}

	plans, err := plan.ListPlans(ctx)
	if err != nil {
		print("  [GAGAL] POOLDATA.M_PLANTRAVEL tidak dapat dibaca isinya: %v", err)
		return
	}
	coverages, err := plan.ListCoverages(ctx)
	if err != nil {
		print("  [GAGAL] jaminan travel tidak dapat dibaca: %v", err)
		return
	}
	print("  [ok]    POOLDATA.M_PLANTRAVEL dapat dibaca: %d plan, %d jaminan", len(plans), len(coverages))
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
//	baca     kedua view ada dan dapat dibaca        -> layarnya dapat menampilkan data
//	tulis    kolom tabel dasarnya sesuai dugaan     -> layarnya dapat MENYIMPAN
//	rujukan  keempat master dapat dibaca            -> daftar pilihannya terisi
//
// Langkah kedua yang paling penting, dan ia satu-satunya yang membuktikan dugaan nama
// kolom pada migrasi 0009 LANGKAH 0 Q1 benar. Tanpa pemisahan ini, layar yang dapat
// menampilkan data tetapi gagal menyimpan akan terbaca [ok] sampai pengguna pertama
// menekan Simpan.
func checkDetailDocumentType(
	ctx context.Context,
	repo *daftardetailtipedokumensql.Repo,
	reference *daftardetailtipedokumensql.ReferenceRepo,
	print func(string, ...any),
) {
	if err := repo.CheckTable(ctx); err != nil {
		print("  [BELUM] Daftar Detail Tipe Dokumen belum siap: %v", err)
		print("            Dua kemungkinan, dan galat di atas membedakannya:")
		print("            - ORA-00942 pada LST_DET_TYPE_DOC_BISNIS: tabel anaknya BELUM")
		print("              ADA. Daftar lini bisnis masih hidup di dalam JSON_DATA")
		print("              induknya. Keputusan Work Owner 2026-09-23 menetapkan modul")
		print("              ini tidak lagi menyentuh JSON, sehingga tabelnya harus")
		print("              dibuat — migrations/0009 LANGKAH 1.")
		print("            - galat hak akses: mintakan GRANT untuk akun aplikasi,")
		print("              migrations/0009 LANGKAH 2.")
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
		print("            DAFTARNYA tetap dapat dimuat. Yang terblokir adalah MEMBUKA")
		print("            SATU BARIS dan MENYIMPAN — keduanya menyentuh tabel anaknya.")
		print("")
		print("            Sebabnya sudah diverifikasi ke katalog Oracle 2026-09-23:")
		print("            POOLDATA.LST_DET_TYPE_DOC sudah lengkap kolomnya, tetapi")
		print("            POOLDATA.LST_DET_TYPE_DOC_BISNIS BELUM ADA — daftar bisnis")
		print("            masih hidup di dalam JSON_DATA induknya, dan view anaknya")
		print("            adalah JSON_TABLE atas kolom itu.")
		print("")
		print("            Yang harus diminta ke DBA ada di migrations/0009 LANGKAH 1:")
		print("            membuat tabel anaknya, memindahkan isi JSON yang sudah ada,")
		print("            lalu mendefinisikan ulang V_LST_DET_TYPE_DOC_BISNIS agar")
		print("            membacanya. Sampai itu selesai, layarnya BACA-SAJA.")
	}

	// Keempat master rujukan diperiksa terakhir, dan kegagalannya hanya CATATAN: keempat
	// kodenya boleh diketik sendiri — layar lama pun memakai autocomplete yang menerima
	// ketikan di luar daftar — sehingga yang hilang hanya kenyamanan memilih.
	if err := reference.CheckTable(ctx); err != nil {
		print("  [catat] master rujukan Daftar Detail Tipe Dokumen tidak dapat dibaca: %v", err)
		print("            Layar tetap dapat dipakai; yang hilang hanya SARAN pada isian")
		print("            Tipe Dokumen, Dokumen kolom ID, Objek Dokumen, dan ID Bisnis.")
	}
}

func checkDocumentObject(
	ctx context.Context,
	repo *daftarobjekdokumensql.Repo,
	business *daftarobjekdokumensql.BusinessRepo,
	print func(string, ...any),
) {
	if err := repo.CheckTable(ctx); err != nil {
		print("  [BELUM] Daftar Objek Dokumen belum siap: %v", err)
		print("            Objeknya disiapkan migrasi 0008_daftar_objek_dokumen, yang ditulis")
		print("            sebagai DAFTAR PERTANYAAN untuk DBA — bukan DDL yang tinggal")
		print("            dijalankan. Bagian 0-nya menanyakan tiga nama yang masih dugaan:")
		print("            POOLDATA.LST_DOC_OBJ, POOLDATA.SET_LST_DOC_OBJ, dan")
		print("            POOLDATA.LST_DOC_OBJ_BUSINESS. Selama belum dijawab, layarnya")
		print("            tidak dapat dipakai terhadap Oracle.")
		return
	}

	list, err := repo.List(ctx)
	if err != nil {
		print("  [GAGAL] POOLDATA.V_LST_DOC_OBJ tidak dapat dibaca isinya: %v", err)
		return
	}
	print("  [ok]    POOLDATA.V_LST_DOC_OBJ dapat dibaca: %d objek dokumen", len(list))

	// Master bisnis diperiksa terpisah: ia milik GISFW (`D-03`) dan hak bacanya diminta
	// sendiri ke DBA. Kegagalannya TIDAK membuat layar tidak dapat dipakai — nama bisnis
	// boleh diketik sendiri — sehingga ia dilaporkan sebagai catatan, bukan sebagai gagal.
	if err := business.CheckTable(ctx); err != nil {
		print("  [catat] POOLDATA.BUSINESS tidak dapat dibaca: %v", err)
		print("            Layar tetap dapat dipakai; yang hilang hanya SARAN nama bisnis,")
		print("            karena namanya memang boleh diketik sendiri.")
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

// checkManagerReceivePUCL memeriksa modul Inbox Manager Receive / PUCL (`MENU_ID 56`).
//
// # Kenapa pemeriksaannya lebih rinci daripada modul inbox lain
//
// Karena TIGA hal di modul ini dibangun di atas kesimpulan yang belum dapat diverifikasi
// tanpa basis data nyata, dan ketiganya gagal TANPA GALAT bila kesimpulannya keliru:
//
//   - Group Panel `002` sebagai pengganti `.ReceiveDocument.TypeOfClaim`. Bila kodenya
//     berbeda di produksi, tab PA kosong dan seluruh isinya pindah ke tab NONMBU.
//   - Akun antrean `RCLPUCL`. Ia diambil dari dua kueri Pega lain karena Report Definition
//     tab itu tidak punya penyaring antrean sama sekali. Bila namanya berubah, tab RCL/PUCL
//     kosong.
//   - POOLDATA.T_CLAIM_RECIVEDCLAIM. Tabel itu TIDAK PERNAH DIBACA sistem lama, sehingga
//     kelengkapan isinya belum terverifikasi. Gabungannya LEFT JOIN, sehingga tabel yang
//     kosong menghasilkan dua kolom kosong — bukan galat.
//
// Ketiganya diperiksa di sini supaya kekeliruannya ketahuan saat `-periksa` dijalankan,
// bukan saat pengguna melaporkan "tabnya kosong".
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

	// Ketiga tab diperiksa, bukan satu.
	//
	// Kedua tab Receive membaca tabel penugasan yang berbeda dari tab RCL/PUCL, dan
	// keduanya dipisahkan penyaring yang justru paling mungkin keliru. Memeriksa satu tab
	// saja akan menyatakan modulnya sehat sementara dua pertiganya belum tersentuh.
	counts := map[string]int{}

	for _, code := range []string{
		inboxmanagerreceivepucl.TabReceivePA,
		inboxmanagerreceivepucl.TabReceiveNonMBU,
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
		if tab.ClaimType == "" {
			continue
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

	if counts[inboxmanagerreceivepucl.TabReceivePA] == 0 {
		print("  [PERIKSA] Tab Receive PA kosong.")
		print("            Periksa apakah Group Panel Personal Accident masih bernilai %q",
			inboxmanagerreceivepucl.GroupPanelPA)
		print("            di produksi. Kode itu menggantikan `.ReceiveDocument.TypeOfClaim`")
		print("            yang tidak punya kolom basis data; bila berbeda, seluruh isi tab")
		print("            ini pindah ke tab NONMBU tanpa satu pun galat.")
	}

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
//   - Kolom `MSIG_1` tampaknya tidak pernah terisi. Bila memang begitu, tab "Klaim MSIG"
//     selalu kosong dan tab "Kelengkapan Dokumen" menampung seluruhnya.
//   - `PUCLAPPROVE_1 <> '1'` tidak menangkap nilai kosong. Klaim yang penandanya belum
//     pernah diisi hilang dari DUA tab sekaligus.
//   - Akun antrean `RCLPUCL`. Bila namanya berubah, KETIGA tab kosong sekaligus.
//   - `STATUSCASE_1 = '0'`. Artinya tidak diketahui, dan tidak ada master yang
//     menerjemahkannya di export mana pun. Bila nilainya berbeda di produksi, tab
//     "Cetak Surat" kosong sementara dua tab lain terisi normal.
//
// Keempatnya diperiksa di sini supaya kekeliruannya ketahuan saat `-periksa` dijalankan,
// bukan saat pengguna melaporkan "tabnya kosong".
func checkRCLPUCL(
	ctx context.Context,
	repo *inboxrclpuclsql.Repo,
	print func(string, ...any),
) {
	if err := repo.CheckTable(ctx); err != nil {
		print("  [BELUM] Inbox RCL/PUCL tidak dapat dibaca: %v", err)
		print("            Modul ini TIDAK menuntut migrasi — seluruh tabelnya milik Pega.")
		print("            Bila galatnya menyebut TABEL, periksa hak SELECT akun aplikasi")
		print("            atas DATAPEGA.PC_ASM_FW_GCNMFW_WORK dan")
		print("            DATAPEGA.PC_ASSIGN_WORKBASKET.")
		print("            Bila galatnya menyebut KOLOM, kolom itu memang tidak ada —")
		print("            seluruh nama kolom modul ini dibaca dari kueri Pega, bukan dari")
		print("            DDL, yang belum pernah diterima (`R-08`).")
		return
	}
	print("  [ok]    Tabel, kolom penyaring, dan tabel anak Inbox RCL/PUCL dapat dibaca")
	print("            Tabel anak memasok isian LAYAR KERJA yang diturunkan:")
	print("            T_CLAIM_OBJECTLIST  -> Nama Peserta DAN UP (keduanya ObjectName,")
	print("                                   dan itu memang benar — dikonfirmasi 2026-09-24),")
	print("            T_CLAIM_ADJUSTMENT  -> Jumlah Tagihan.")
	print("            Sembilan isian lain adalah properti clipboard Pega; ia tidak punya")
	print("            kolom, sehingga tidak ada yang perlu diperiksa di sini.")

	page := inboxrclpucl.Pagination{Page: 1, Size: 5}
	counts := map[string]int{}

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
	}

	if counts[inboxrclpucl.TabCetakSurat] == 0 &&
		counts[inboxrclpucl.TabKelengkapanDokumen] == 0 &&
		counts[inboxrclpucl.TabKlaimMSIG] == 0 {
		print("  [PERIKSA] KETIGA tab kosong sekaligus.")
		print("            Periksa apakah akun antrean bersama masih bernama %q.",
			inboxrclpucl.RCLPUCLWorkbasket)
		print("            Ketiga tab memakai akun yang sama, sehingga namanya yang")
		print("            berubah mengosongkan seluruh layar tanpa satu pun galat.")
		return
	}

	if counts[inboxrclpucl.TabCetakSurat] == 0 {
		print("  [PERIKSA] Tab \"Cetak Surat\" kosong sementara tab lain terisi.")
		print("            Periksa apakah STATUSCASE_1 masih bernilai %q untuk klaim yang",
			inboxrclpucl.ExpiryStatusActive)
		print("            suratnya belum dicetak. Arti kolom itu tidak diketahui — tidak")
		print("            ada master yang menerjemahkannya di export mana pun — dan nilai")
		print("            yang berbeda mengosongkan tab ini saja.")
	}

	if counts[inboxrclpucl.TabKlaimMSIG] == 0 {
		print("  [PERIKSA] Tab \"Klaim MSIG\" kosong. Ini yang DIPERKIRAKAN terjadi.")
		print("            Kolom MSIG_1 tidak muncul di inventaris kolom terisi yang")
		print("            dibaca dari katalog Oracle pada 2026-09-22, sehingga ia")
		print("            tampaknya ada tetapi belum pernah diisi. Bila memang begitu,")
		print("            seluruh klaim bersurat berada di tab \"Kelengkapan Dokumen\".")
		print("            Mohon DBA memastikannya:")
		print("            SELECT COUNT(*) FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK")
		print("             WHERE MSIG_1 IS NOT NULL;")
	}

	if counts[inboxrclpucl.TabKelengkapanDokumen] == 0 {
		print("  [PERIKSA] Tab \"Kelengkapan Dokumen\" kosong.")
		print("            Kemungkinan terbesarnya BUKAN antrean yang sepi melainkan")
		print("            penyaring PUCLAPPROVE_1 <> %q: perbandingan itu tidak pernah",
			inboxrclpucl.PUCLApproved)
		print("            bernilai benar untuk nilai KOSONG, sehingga klaim yang")
		print("            penandanya belum pernah diisi tidak muncul. Perilakunya")
		print("            direplikasi dari Pega dengan sengaja; periksa sebarannya:")
		print("            SELECT PUCLAPPROVE_1, COUNT(*) FROM")
		print("             DATAPEGA.PC_ASM_FW_GCNMFW_WORK GROUP BY PUCLAPPROVE_1;")
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
		print("            POOLDATA.D_KOMUNIKASI_PNC, POOLDATA.V_LST_DOC_TYPE, dan")
		print("            POOLDATA.V_LST_DET_TYPE_DOC.")
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

// checkInboxAnalystDoctor memastikan tabel DAN dua kolom yang dibaca layar Inbox Analyst
// Doctor terjangkau.
//
// # Kenapa modul ini diperiksa dalam DUA langkah, tidak seperti modul lain
//
// Karena dua hal yang berbeda dapat gagal di sini, dan yang kedua SUDAH DIDUGA akan gagal:
//
//	tabel   hak SELECT belum diberikan — sama seperti modul lain
//	kolom   nama kolomnya belum dikonfirmasi DBA — khas modul ini
//
// `Report Definition/InboxAnalystDoctor_RD-RD.xml` menandai `.ClaimData.isComplianceTransfer`
// dan `.ClaimData.AnalystDoctorRemaks` sebagai `unexposed` — keduanya hidup di dalam blob
// Pega, bukan sebagai kolom SQL. Nama kolom yang dipakai mengikuti konvensi `_1` yang berlaku
// pada properti `ClaimData` lain, dan konvensi itu belum dibuktikan untuk kedua nama ini.
//
// Inilah satu-satunya tempat keadaan itu diketahui SEBELUM ada pengguna yang membuka
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
		print("  [BELUM] Kolom ISCOMPLIANCETRANSFER_1 / ANALYSTDOCTORREMAKS_1 tidak ada: %v", err)
		print("            INI SUDAH DIDUGA. Kedua properti Pega-nya ditandai `unexposed`,")
		print("            sehingga keduanya tidak punya kolom SQL yang terbukti. Yang")
		print("            pertama adalah PENYARING UTAMA layar ini; tanpanya layar tidak")
		print("            dapat dipakai sama sekali terhadap Oracle.")
		print("            Yang diminta ke DBA — satu kueri katalog:")
		print("              SELECT COLUMN_NAME, DATA_TYPE, NUM_DISTINCT FROM ALL_TAB_COLUMNS")
		print("               WHERE OWNER = 'DATAPEGA' AND TABLE_NAME = 'PC_ASM_FW_GCNMFW_WORK'")
		print("                 AND (COLUMN_NAME LIKE '%%COMPLIANCE%%'")
		print("                      OR COLUMN_NAME LIKE '%%ANALYSTDOCTOR%%');")
		return
	}
	print("  [ok]    Kolom ISCOMPLIANCETRANSFER_1 dan ANALYSTDOCTORREMAKS_1 ada")
	print("            Catatan: antrean ini disaring dengan Operator ID pemanggil, sehingga")
	print("            pengguna tanpa tugas Analyst Doctor melihatnya kosong — dan itu")
	print("            jawaban yang benar, bukan kerusakan.")
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

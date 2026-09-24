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
	"claim-pnc/internal/inboxcompliance"
	"claim-pnc/internal/masterautoclaim"
	"claim-pnc/internal/masterbengkel"
	"claim-pnc/internal/masterdominanfactor"
	"claim-pnc/internal/masterpanel"
	"claim-pnc/internal/masterpenyebabkerugian"
	"claim-pnc/internal/masterstatus"
	"claim-pnc/internal/mastersupplier"
	"claim-pnc/internal/mastersurveyors"
	"claim-pnc/internal/masterxol"
	"claim-pnc/internal/pelaporanklaim"
	"claim-pnc/internal/platform/config"
	"claim-pnc/internal/platform/db"
	"claim-pnc/internal/portal"

	daftardetaildokumentravelsql "claim-pnc/internal/daftardetaildokumentravel/repo/sqlstore"
	daftardetailtipedokumensql "claim-pnc/internal/daftardetailtipedokumen/repo/sqlstore"
	daftarobjekdokumensql "claim-pnc/internal/daftarobjekdokumen/repo/sqlstore"
	daftartipedokumensql "claim-pnc/internal/daftartipedokumen/repo/sqlstore"
	daftartipedokumenbisnissql "claim-pnc/internal/daftartipedokumenbisnis/repo/sqlstore"
	inboxadminsql "claim-pnc/internal/inboxadmin/repo/sqlstore"
	inboxcompliancesql "claim-pnc/internal/inboxcompliance/repo/sqlstore"
	masterautoclaimsql "claim-pnc/internal/masterautoclaim/repo/sqlstore"
	masterbengkelsql "claim-pnc/internal/masterbengkel/repo/sqlstore"
	mastercolsql "claim-pnc/internal/mastercolsimasonline/repo/sqlstore"
	masterdominanfactorsql "claim-pnc/internal/masterdominanfactor/repo/sqlstore"
	masterpanelsql "claim-pnc/internal/masterpanel/repo/sqlstore"
	masterpasalsql "claim-pnc/internal/masterpasal/repo/sqlstore"
	masterpenolakansql "claim-pnc/internal/masterpenolakan/repo/sqlstore"
	masterpenyebabkerugiansql "claim-pnc/internal/masterpenyebabkerugian/repo/sqlstore"
	masterpicteknikdirectory "claim-pnc/internal/masterpicteknik/directory"
	masterpictekniksql "claim-pnc/internal/masterpicteknik/repo/sqlstore"
	masterrecoverysql "claim-pnc/internal/masterrecovery/repo/sqlstore"
	masterrecoveryva "claim-pnc/internal/masterrecovery/virtualaccount"
	mastersparepartsql "claim-pnc/internal/mastersparepart/repo/sqlstore"
	masterstatussql "claim-pnc/internal/masterstatus/repo/sqlstore"
	mastersuppliersql "claim-pnc/internal/mastersupplier/repo/sqlstore"
	mastersurveyorssql "claim-pnc/internal/mastersurveyors/repo/sqlstore"
	masterxolsql "claim-pnc/internal/masterxol/repo/sqlstore"
	pelaporanklaimsql "claim-pnc/internal/pelaporanklaim/repo/sqlstore"
	portalsql "claim-pnc/internal/portal/repo/sqlstore"
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
	checkSimasOnlineCauseOfLoss(ctx, mastercolsql.NewRepo(primary), mastercolsql.NewBusinessRepo(primary), print)
	checkTravelDocumentDetail(ctx, primary, print)
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
	claimReport := pelaporanklaimsql.NewRepo(primary)
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
		{"Inbox Laporan Klaim", claimReport.CheckTable, func(ctx context.Context) (int, error) {
			page, err := claimReport.List(ctx, pelaporanklaim.Filter{})
			return page.Total, err
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

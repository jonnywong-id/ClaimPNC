package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"claim-pnc/internal/auth"
	"claim-pnc/internal/auth/provider"
	"claim-pnc/internal/auth/repo/sqlstore"
	"claim-pnc/internal/inboxoutstanding"
	"claim-pnc/internal/masterdominanfactor"
	"claim-pnc/internal/masterpenyebabkerugian"
	"claim-pnc/internal/masterstatus"
	"claim-pnc/internal/masterxol"
	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/platform/config"
	"claim-pnc/internal/platform/db"
	"claim-pnc/internal/portal"

	"claim-pnc/internal/inboxautoclaim"
	inboxautoclaimsql "claim-pnc/internal/inboxautoclaim/repo/sqlstore"
	"claim-pnc/internal/inboxclaimtreatynonprop"
	inboxclaimtreatynonpropsql "claim-pnc/internal/inboxclaimtreatynonprop/repo/sqlstore"

	"claim-pnc/internal/inboxclaimtreatyprop"
	inboxclaimtreatypropsql "claim-pnc/internal/inboxclaimtreatyprop/repo/sqlstore"
	inboxlaporanklaimsql "claim-pnc/internal/inboxlaporanklaim/repo/sqlstore"
	"claim-pnc/internal/inboxmanagerreceivepucl"
	inboxmanagerreceivepuclsql "claim-pnc/internal/inboxmanagerreceivepucl/repo/sqlstore"
	inboxoutstandingsql "claim-pnc/internal/inboxoutstanding/repo/sqlstore"
	inboxprogressclaimsql "claim-pnc/internal/inboxprogressclaim/repo/sqlstore"
	masterdominanfactorsql "claim-pnc/internal/masterdominanfactor/repo/sqlstore"
	masterpenyebabkerugiansql "claim-pnc/internal/masterpenyebabkerugian/repo/sqlstore"
	masterpicteknikdirectory "claim-pnc/internal/masterpicteknik/directory"
	masterpictekniksql "claim-pnc/internal/masterpicteknik/repo/sqlstore"
	masterrecoverysql "claim-pnc/internal/masterrecovery/repo/sqlstore"
	masterrecoveryva "claim-pnc/internal/masterrecovery/virtualaccount"
	masterstatussql "claim-pnc/internal/masterstatus/repo/sqlstore"
	masterxolsql "claim-pnc/internal/masterxol/repo/sqlstore"
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
	checkAutoClaim(ctx, inboxautoclaimsql.NewRepo(primary), print)
	checkAutoClaimTabsDiffer(ctx, inboxautoclaimsql.NewRepo(primary), print)
	checkAutoClaimPaging(ctx, inboxautoclaimsql.NewRepo(primary), print)
	checkAutoClaimEveryCompany(ctx, inboxautoclaimsql.NewRepo(primary), print)
	checkPicTeknik(ctx, masterpictekniksql.NewRepo(primary), legacy, cfg.PrimaryPortal, print)
	checkRecovery(ctx, masterrecoverysql.NewRepo(primary), legacy, cfg.PrimaryPortal, print)
	checkDominantFactor(ctx, masterdominanfactorsql.NewRepo(primary), print)
	checkCauseOfLoss(ctx, masterpenyebabkerugiansql.NewRepo(primary), print)
	checkXOL(ctx, masterxolsql.NewRepo(primary), print)
	checkClaimTreatyProp(ctx, inboxclaimtreatypropsql.NewRepo(primary), print)
	checkClaimTreatyNonProp(ctx, inboxclaimtreatynonpropsql.NewRepo(primary), print)
	checkManagerReceivePUCL(ctx, inboxmanagerreceivepuclsql.NewRepo(primary), print)
	checkInboxProgressClaim(ctx, inboxprogressclaimsql.NewRepo(primary), print)
	checkClaimReport(ctx, inboxlaporanklaimsql.NewRepo(primary, clock.System{}), print)
	checkOutstanding(ctx, inboxoutstandingsql.NewRepo(primary), print)
	checkClaimReportBranch(ctx, inboxlaporanklaimsql.NewBranchResolver(primary), print)

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

// checkOutstanding menjalankan kueri Inbox Outstanding terhadap Oracle sungguhan.
//
// Inilah satu-satunya jalan membuktikan kuerinya sah selama migrasi 0001 belum dijalankan:
// layarnya sendiri menuntut sesi, dan sesi disimpan di `CPNC_SESI_AKTIF` yang belum ada.
//
// Yang dibuktikan di sini: SQL-nya diterima Oracle, penanda bind-nya benar, dan
// pembacaan 21 kolomnya cocok dengan tipe kolom yang sebenarnya. Yang TIDAK dibuktikan:
// kesetaraan hasilnya dengan layar Pega — itu gerbang 1, milik `S-8`.
func checkOutstanding(ctx context.Context, repo *inboxoutstandingsql.Repo, print func(string, ...any)) {
	// Tanpa batas lini: memeriksa tabelnya, bukan kewenangan seseorang.
	page, err := repo.List(ctx, inboxoutstanding.Filter{
		Scope: inboxoutstanding.LineScope{Unrestricted: true},
		Limit: 5,
	})
	if err != nil {
		print("  [BELUM] POOLDATA.T_CLAIMLIST_ADMIN tidak dapat dibaca: %v", err)
		print("            Tabelnya milik sistem lama, bukan dibuat migrasi. Selama belum ada,")
		print("            layar Inbox Outstanding tidak dapat dipakai terhadap Oracle.")
		return
	}

	print("  [ok]    POOLDATA.T_CLAIMLIST_ADMIN dapat dibaca: %d klaim masih berjalan", page.Total)

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

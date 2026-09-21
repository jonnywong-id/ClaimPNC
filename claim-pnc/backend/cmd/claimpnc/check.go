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
	"claim-pnc/internal/masterautoclaim"
	"claim-pnc/internal/masterbengkel"
	"claim-pnc/internal/masterpanel"
	"claim-pnc/internal/masterpasal"
	"claim-pnc/internal/masterpenolakan"
	"claim-pnc/internal/mastersparepart"
	"claim-pnc/internal/masterstatus"
	"claim-pnc/internal/pelaporanklaim"
	"claim-pnc/internal/platform/config"
	"claim-pnc/internal/platform/db"
	"claim-pnc/internal/portal"
	"claim-pnc/internal/riwayatklaim"

	inboxadminsql "claim-pnc/internal/inboxadmin/repo/sqlstore"
	inboxprogressclaimsql "claim-pnc/internal/inboxprogressclaim/repo/sqlstore"
	masterautoclaimsql "claim-pnc/internal/masterautoclaim/repo/sqlstore"
	masterbengkelsql "claim-pnc/internal/masterbengkel/repo/sqlstore"
	masterpanelsql "claim-pnc/internal/masterpanel/repo/sqlstore"
	masterpasalsql "claim-pnc/internal/masterpasal/repo/sqlstore"
	masterpenolakansql "claim-pnc/internal/masterpenolakan/repo/sqlstore"
	mastersparepartsql "claim-pnc/internal/mastersparepart/repo/sqlstore"
	masterstatussql "claim-pnc/internal/masterstatus/repo/sqlstore"
	mastersuppliersql "claim-pnc/internal/mastersupplier/repo/sqlstore"
	pelaporanklaimsql "claim-pnc/internal/pelaporanklaim/repo/sqlstore"
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
	checkAutoClaim(ctx, primary, print)
	checkBengkel(ctx, primary, print)
	checkPanel(ctx, primary, print)
	checkSparepart(ctx, primary, print)
	checkPasal(ctx, primary, print)
	checkSupplier(ctx, primary, print)
	checkClaimReport(ctx, pelaporanklaimsql.NewRepo(primary), print)
	checkClaimHistoryGate(ctx, riwayatklaimsql.NewProtectionRepo(primary), print)
	checkInboxAdmin(ctx, inboxadminsql.NewRepo(primary), print)
	checkInboxProgressClaim(ctx, inboxprogressclaimsql.NewRepo(primary), print)

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

// checkAutoClaim melaporkan kesiapan POOLDATA.M_AUTO_CLAIM_PNC beserta acuannya.
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
func checkAutoClaim(ctx context.Context, primary *sql.DB, print func(string, ...any)) {
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

// checkClaimReport melaporkan kesiapan POOLDATA.CPNC_LAPORAN_KLAIM sesudah migrasi
// 0003.
//
// Ia melaporkan jumlah laporan PER TAHAP, bukan sekadar "dapat dibaca". Angka itulah yang
// membedakan tabel yang sudah dipakai dari tabel yang baru dibuat dan masih kosong — dan
// pada tabel yang sudah berisi, ia sekaligus memperlihatkan apakah ada laporan yang
// tertahan lama di satu tahap.
func checkClaimReport(ctx context.Context, repo *pelaporanklaimsql.Repo, print func(string, ...any)) {
	if err := repo.CheckTable(ctx); err != nil {
		print("  [BELUM] POOLDATA.CPNC_LAPORAN_KLAIM belum siap: %v", err)
		print("            Tabelnya dibuat migrasi 0003. Selama belum dijalankan, layar")
		print("            Pelaporan Klaim tidak dapat dipakai terhadap Oracle — tetapi")
		print("            seluruh bagian lain tetap jalan.")
		return
	}

	summary, err := repo.Summary(ctx, pelaporanklaim.Filter{})
	if err != nil {
		print("  [GAGAL] POOLDATA.CPNC_LAPORAN_KLAIM tidak dapat dihitung isinya: %v", err)
		return
	}

	total := 0
	for _, count := range summary {
		total += count
	}
	print("  [ok]    POOLDATA.CPNC_LAPORAN_KLAIM dapat dibaca: %d laporan", total)

	// Urutannya tetap, mengikuti perjalanan laporan — bukan urutan map, yang berubah
	// setiap kali proses dijalankan dan membuat dua keluaran tidak dapat dibandingkan.
	for _, stage := range []pelaporanklaim.Stage{
		pelaporanklaim.StageNotTransferred,
		pelaporanklaim.StageNotRegistered,
		pelaporanklaim.StageRegistered,
		pelaporanklaim.StageAccepted,
		pelaporanklaim.StageRejected,
	} {
		print("            %-20s %d", stage.Label(), summary[stage])
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
		print("            Tabelnya warisan Pega dan TIDAK dibuat migrasi aplikasi ini.")
		print("            Mintakan hak bacanya ke DBA.")
		print("            Catatan: kueri pemeriksaan ini menyebut PROD_DATE dan")
		print("            TGL_UPDATE_HARGA; bila yang gagal justru salah satunya, tipe")
		print("            kolomnya berbeda dari asumsi mastersparepart.sql.")
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

// checkInboxAdmin memastikan tabel yang dibaca layar Inbox Admin terjangkau.
//
// Berbeda dengan modul lain, modul ini TIDAK menuntut satu pun migrasi: seluruh tabel yang
// dibacanya sudah ada dan milik sistem lama. Yang dapat gagal karena itu bukan "tabelnya
// belum dibuat", melainkan "akun aplikasi belum diberi hak SELECT atasnya".
func checkInboxAdmin(
	ctx context.Context,
	repo *inboxadminsql.Repo,
	print func(string, ...any),
) {
	if err := repo.CheckTable(ctx); err != nil {
		print("  [BELUM] DATAPEGA.PC_ASM_FW_GCNMFW_WORK tidak dapat dibaca: %v", err)
		print("            Tanpa hak baca atasnya, seluruh tab Inbox Admin kosong.")
		print("            Tabel ini milik sistem lama dan tidak dibuat migrasi mana pun.")
		return
	}
	print("  [ok]    DATAPEGA.PC_ASM_FW_GCNMFW_WORK dapat dibaca")
	print("            Catatan: penyaring Cabang dan Korwil BELUM aktif — sumbernya")
	print("            DB Link ke HRD yang belum punya API pengganti (R-03).")
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

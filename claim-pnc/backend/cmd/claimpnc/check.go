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
	"claim-pnc/internal/masterstatus"
	"claim-pnc/internal/platform/config"
	"claim-pnc/internal/platform/db"
	"claim-pnc/internal/portal"

	"claim-pnc/internal/inboxautoclaim"
	inboxautoclaimsql "claim-pnc/internal/inboxautoclaim/repo/sqlstore"
	masterstatussql "claim-pnc/internal/masterstatus/repo/sqlstore"
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
	checkAutoClaim(ctx, inboxautoclaimsql.NewRepo(primary), print)
	checkAutoClaimTabsDiffer(ctx, inboxautoclaimsql.NewRepo(primary), print)
	checkAutoClaimPaging(ctx, inboxautoclaimsql.NewRepo(primary), print)
	checkAutoClaimEveryCompany(ctx, inboxautoclaimsql.NewRepo(primary), print)

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

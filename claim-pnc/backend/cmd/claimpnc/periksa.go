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
	"claim-pnc/internal/platform/config"
	"claim-pnc/internal/platform/db"
	"claim-pnc/internal/portal"
	portalsql "claim-pnc/internal/portal/repo/sqlstore"
)

// periksa menjalankan pemeriksaan integrasi dan mencetak hasilnya, lalu berhenti.
//
// # Kenapa mode ini ada
//
// Masuk yang sesungguhnya MENULIS ke CPNC_PENGGUNA dan CPNC_SESI_AKTIF, dan kedua tabel
// itu baru ada setelah DBA menjalankan migrasi 0001. Tanpa mode ini, integrasi HCC/HCQ
// dan POOLDATA.M_LOGIN_PNC tidak dapat dicoba sama sekali sebelum migrasi selesai —
// padahal keduanya justru yang paling ingin dibuktikan lebih dulu.
//
// Mode ini karena itu **tidak menulis apa pun**. Ia hanya membaca, memanggil HCQ, dan
// melaporkan apa yang ditemukannya.
func periksa(konf config.Konfigurasi, login string, sumberSandi io.Reader, keluaran io.Writer) error {
	cetak := func(format string, isi ...any) {
		_, _ = fmt.Fprintf(keluaran, format+"\n", isi...)
	}

	cetak("Periksa integrasi Claim PNC")
	cetak("  lingkungan       : %s", konf.Lingkungan)
	cetak("  portal utama     : %s", konf.PortalUtama)
	cetak("  adapter identitas: %s", konf.AdapterIdentitas)
	cetak("")

	if konf.Penyimpanan != config.PenyimpananOracle {
		return fmt.Errorf("mode periksa menuntut PENYIMPANAN=oracle; sekarang %q.\n"+
			"    Alamat layanan HCQ dan daftar login non-karyawan keduanya dibaca dari basis data",
			konf.Penyimpanan)
	}

	ctx, batal := context.WithTimeout(context.Background(), 60*time.Second)
	defer batal()

	kumpulan, err := db.KumpulanBaru(ctx, konf.PortalUtama, parameterPortal(konf), func(alias string, err error) {
		cetak("  [lewat] portal %-5s tidak dapat dibuka: %v", alias, err)
	})
	if err != nil {
		return fmt.Errorf("koneksi basis data gagal: %w", err)
	}
	defer kumpulan.Tutup()
	cetak("  [ok]    koneksi basis data: %s", strings.Join(kumpulan.Tersedia(), ", "))

	utama := kumpulan.Utama()
	warisan := sqlstore.WarisanBaru(utama)

	daftarPortal := periksaPortal(ctx, portalsql.RepoBaru(utama), cetak)
	alamat := periksaKatalog(ctx, warisan, konf.PortalUtama, cetak)
	periksaTabelAplikasi(ctx, warisan, cetak)
	periksaTabelLogin(ctx, warisan, cetak)

	cetak("")
	if login == "" {
		cetak("Selesai. Tambahkan -login <nama pengguna> untuk mencoba masuk sungguhan.")
		_ = daftarPortal
		return nil
	}
	if alamat == "" {
		cetak("Percobaan masuk dilewati: alamat layanan HCQ belum dapat dibaca.")
		return nil
	}
	return periksaMasuk(ctx, konf, warisan, login, sumberSandi, cetak)
}

func periksaPortal(ctx context.Context, repo portal.Repo, cetak func(string, ...any)) []portal.Portal {
	daftar, err := repo.Daftar(ctx)
	if err != nil {
		cetak("  [GAGAL] POOLDATA.M_PORTAL_PNC: %v", err)
		return nil
	}
	cetak("  [ok]    POOLDATA.M_PORTAL_PNC: %d portal terbaca", len(daftar))
	for _, p := range daftar {
		cetak("            %-5s %s", p.Alias, p.Nama)
	}
	return daftar
}

func periksaKatalog(ctx context.Context, warisan *sqlstore.Warisan, alias string, cetak func(string, ...any)) string {
	alamat, err := warisan.AlamatLayanan(ctx, alias, "HCQ-LOGIN")
	switch {
	case errors.Is(err, provider.ErrLayananTidakTerdaftar):
		cetak("  [GAGAL] POOLDATA.GCNM_CONNECT_REST: tidak ada baris APP=%q TYPESERVICE='HCQ-LOGIN'", alias)
		cetak("            Mintakan barisnya ke DBA, atau ubah PORTAL_UTAMA ke portal yang sudah punya.")
		return ""
	case err != nil:
		cetak("  [GAGAL] POOLDATA.GCNM_CONNECT_REST: %v", err)
		return ""
	}
	cetak("  [ok]    alamat layanan HCQ untuk APP=%q: %s", alias, alamat)
	return alamat
}

// periksaTabelAplikasi membedakan "migrasi belum dijalankan" dari "akun tidak punya hak
// baca" — dua sebab yang tampak mirip tetapi perbaikannya berbeda jauh.
func periksaTabelAplikasi(ctx context.Context, warisan *sqlstore.Warisan, cetak func(string, ...any)) {
	tabel := []struct{ nama, kueri string }{
		{"CPNC_PENGGUNA", "pengguna_periksa_tabel"},
		{"CPNC_SESI_AKTIF", "sesi_periksa_tabel"},
	}
	kurang := 0
	for _, t := range tabel {
		if err := warisan.PeriksaTabel(ctx, t.kueri); err != nil {
			cetak("  [BELUM] %s tidak dapat dibaca: %v", t.nama, err)
			kurang++
			continue
		}
		cetak("  [ok]    %s dapat dibaca", t.nama)
	}
	if kurang > 0 {
		cetak("            Migrasi backend/migrations/0001 tampaknya belum dijalankan DBA.")
		cetak("            Mode periksa TETAP bisa mencoba masuk — ia tidak menulis apa pun.")
		cetak("            Yang belum bisa adalah masuk lewat aplikasi, karena itu menyimpan sesi.")
	}
}

func periksaTabelLogin(ctx context.Context, warisan *sqlstore.Warisan, cetak func(string, ...any)) {
	if err := warisan.PeriksaTabel(ctx, "login_lokal_periksa_tabel"); err != nil {
		cetak("  [GAGAL] POOLDATA.M_LOGIN_PNC tidak dapat dibaca: %v", err)
		return
	}
	cetak("  [ok]    POOLDATA.M_LOGIN_PNC dapat dibaca")
}

// periksaMasuk menjalankan rantai identitas yang sama persis dengan yang dipakai
// aplikasi — HCQ dulu, lalu POOLDATA.M_LOGIN_PNC — tanpa menerbitkan sesi.
func periksaMasuk(
	ctx context.Context,
	konf config.Konfigurasi,
	warisan *sqlstore.Warisan,
	login string,
	sumberSandi io.Reader,
	cetak func(string, ...any),
) error {
	sandi, err := bacaSandi(sumberSandi)
	if err != nil {
		return err
	}

	rantai, err := rakitIdentitas(konf, false, warisan)
	if err != nil {
		return err
	}

	cetak("Mencoba masuk sebagai %q …", login)
	profil, err := rantai.Verifikasi(ctx, auth.Kredensial{NamaPengguna: login, KataSandi: sandi})
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrKredensialSalah):
			cetak("  [TOLAK] kredensial tidak diterima HCQ maupun POOLDATA.M_LOGIN_PNC.")
			cetak("          Keduanya menjawab; jadi jalur integrasinya hidup, kredensialnya yang tidak cocok.")
		case errors.Is(err, auth.ErrPenggunaTidakAktif):
			cetak("  [TOLAK] akun ada tetapi tidak aktif.")
		case errors.Is(err, auth.ErrSistemTidakTerhubung):
			cetak("  [GAGAL] sistem identitas tidak dapat dihubungi: %v", err)
		default:
			cetak("  [GAGAL] %v", err)
		}
		return nil
	}

	cetak("  [ok]    diterima")
	cetak("            jenis      : %s", profil.Jenis)
	cetak("            identitas  : %s", profil.Identitas)
	cetak("            nama       : %s", profil.Nama)
	cetakBilaAda(cetak, "login      ", profil.Login)
	cetakBilaAda(cetak, "email      ", profil.Email)
	cetakBilaAda(cetak, "perusahaan ", profil.Perusahaan)
	cetakBilaAda(cetak, "cabang     ", profil.Cabang)
	cetakBilaAda(cetak, "kode cabang", profil.KodeCabang)
	cetakBilaAda(cetak, "jabatan    ", profil.Jabatan)
	if profil.AktifDiSumber != nil {
		cetak("            aktif di sumber: %t (direkam, belum dipakai menolak)", *profil.AktifDiSumber)
	}
	return nil
}

func cetakBilaAda(cetak func(string, ...any), label, nilai string) {
	if strings.TrimSpace(nilai) != "" {
		cetak("            %s: %s", label, nilai)
	}
}

// bacaSandi membaca kata sandi dari stdin.
//
// Ia sengaja TIDAK diterima sebagai argumen baris perintah: argumen tersimpan di riwayat
// shell dan terlihat oleh siapa pun yang menjalankan daftar proses.
func bacaSandi(sumber io.Reader) (string, error) {
	if sumber == nil {
		sumber = os.Stdin
	}
	pemindai := bufio.NewScanner(sumber)
	if !pemindai.Scan() {
		if err := pemindai.Err(); err != nil {
			return "", fmt.Errorf("membaca kata sandi dari stdin: %w", err)
		}
		return "", errors.New("kata sandi tidak terbaca dari stdin")
	}
	sandi := strings.TrimRight(pemindai.Text(), "\r\n")
	if sandi == "" {
		return "", errors.New("kata sandi kosong")
	}
	return sandi, nil
}

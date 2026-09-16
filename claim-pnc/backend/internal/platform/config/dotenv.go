package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// MuatBerkasEnv membaca berkas .env dan menaruh isinya ke variabel lingkungan proses.
//
// Nilai yang SUDAH ada di lingkungan tidak ditimpa. Urutannya disengaja: variabel yang
// disetel operator saat menjalankan aplikasi harus menang atas berkas, supaya satu
// perintah dapat menimpa satu nilai tanpa menyunting berkasnya.
//
// Berkas yang tidak ada bukan galat — di server, nilai datang dari lingkungan proses,
// bukan dari berkas (ADR-0025).
//
// Format yang didukung sengaja sempit:
//
//	# komentar
//	NAMA=nilai
//	NAMA="nilai dengan spasi"
//	NAMA='nilai apa adanya'
//
// Tidak ada interpolasi `${LAIN}`, tidak ada `export`, dan tidak ada nilai
// multi-baris. Format yang lebih pintar berarti lebih banyak cara salah membaca
// kredensial.
func MuatBerkasEnv(jalur string) error {
	berkas, err := os.Open(jalur)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("config: membuka %s: %w", jalur, err)
	}
	defer func() { _ = berkas.Close() }()

	pemindai := bufio.NewScanner(berkas)
	nomor := 0
	for pemindai.Scan() {
		nomor++
		baris := strings.TrimSpace(pemindai.Text())
		if baris == "" || strings.HasPrefix(baris, "#") {
			continue
		}
		nama, nilai, ada := strings.Cut(baris, "=")
		if !ada {
			return fmt.Errorf("config: %s baris %d tidak berbentuk NAMA=nilai", jalur, nomor)
		}
		nama = strings.TrimSpace(nama)
		if nama == "" {
			return fmt.Errorf("config: %s baris %d tidak punya nama variabel", jalur, nomor)
		}
		if _, sudahAda := os.LookupEnv(nama); sudahAda {
			continue
		}
		if err := os.Setenv(nama, lepasKutip(strings.TrimSpace(nilai))); err != nil {
			return fmt.Errorf("config: menyetel %s: %w", nama, err)
		}
	}
	if err := pemindai.Err(); err != nil {
		return fmt.Errorf("config: membaca %s: %w", jalur, err)
	}
	return nil
}

// lepasKutip membuang sepasang kutip pembungkus bila ada. Kata sandi basis data kerap
// memuat karakter seperti `#` atau spasi yang menuntut kutip.
func lepasKutip(nilai string) string {
	if len(nilai) >= 2 {
		awal, akhir := nilai[0], nilai[len(nilai)-1]
		if (awal == '"' && akhir == '"') || (awal == '\'' && akhir == '\'') {
			return nilai[1 : len(nilai)-1]
		}
	}
	// Komentar di belakang nilai hanya dikenali bila didahului spasi, supaya kata sandi
	// yang memuat '#' tidak terpotong diam-diam.
	if potong := strings.Index(nilai, " #"); potong >= 0 {
		return strings.TrimSpace(nilai[:potong])
	}
	return nilai
}

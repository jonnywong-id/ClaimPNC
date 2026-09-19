package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// LoadEnvFile membaca berkas .env dan menaruh isinya ke variabel lingkungan proses.
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
func LoadEnvFile(filePath string) error {
	files, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("config: membuka %s: %w", filePath, err)
	}
	defer func() { _ = files.Close() }()

	pemindai := bufio.NewScanner(files)
	nomor := 0
	for pemindai.Scan() {
		nomor++
		rows := strings.TrimSpace(pemindai.Text())
		if rows == "" || strings.HasPrefix(rows, "#") {
			continue
		}
		name, value, existing := strings.Cut(rows, "=")
		if !existing {
			return fmt.Errorf("config: %s baris %d tidak berbentuk NAMA=nilai", filePath, nomor)
		}
		name = strings.TrimSpace(name)
		if name == "" {
			return fmt.Errorf("config: %s baris %d tidak punya nama variabel", filePath, nomor)
		}
		if _, exists := os.LookupEnv(name); exists {
			continue
		}
		if err := os.Setenv(name, unquote(strings.TrimSpace(value))); err != nil {
			return fmt.Errorf("config: menyetel %s: %w", name, err)
		}
	}
	if err := pemindai.Err(); err != nil {
		return fmt.Errorf("config: membaca %s: %w", filePath, err)
	}
	return nil
}

// unquote membuang sepasang kutip pembungkus bila ada. Kata sandi basis data kerap
// memuat karakter seperti `#` atau spasi yang menuntut kutip.
func unquote(value string) string {
	if len(value) >= 2 {
		start, end := value[0], value[len(value)-1]
		if (start == '"' && end == '"') || (start == '\'' && end == '\'') {
			return value[1 : len(value)-1]
		}
	}
	// Komentar di belakang nilai hanya dikenali bila didahului spasi, supaya kata sandi
	// yang memuat '#' tidak terpotong diam-diam.
	if trimmed := strings.Index(value, " #"); trimmed >= 0 {
		return strings.TrimSpace(value[:trimmed])
	}
	return value
}

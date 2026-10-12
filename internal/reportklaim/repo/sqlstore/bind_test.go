package sqlstore

import (
	"fmt"
	"regexp"
	"strconv"
	"testing"
	"time"

	"claim-pnc/internal/reportklaim"
)

// penandaBind menangkap penanda bind Oracle (`:1`, `:2`, …).
//
// Tanda titik dua yang diikuti huruf SENGAJA tidak ditangkap — ia penanda bernama, dan
// tidak satu pun kueri modul ini memakainya.
var penandaBind = regexp.MustCompile(`:(\d+)`)

// TestJumlahBindCocokDenganParameter menolak kueri yang penanda bind-nya tidak sejajar
// dengan parameter yang dikirim rencananya.
//
// # Kenapa uji ini ada
//
// Ia lahir dari satu laporan nyata pada 2026-10-01: SELURUH tombol ekspor menjawab
// "Terjadi kesalahan pada sistem", kecuali satu — dan yang satu itu kebetulan
// satu-satunya laporan yang TIDAK punya satu pun parameter bind. Pola seperti itu
// menunjuk ke parameter, bukan ke dua puluh tujuh kueri.
//
// Ketidakcocokan bind adalah kelas cacat yang paling mahal ditemukan di tempat lain:
//
//   - Ia TIDAK tertangkap kompilator — kueri hanyalah teks.
//   - Ia TIDAK tertangkap uji yang memakai basis data palsu.
//   - Ia baru muncul sebagai ORA-01008 ketika seseorang benar-benar menekan tombolnya,
//     dan di layar ia tampil sebagai galat sistem yang tidak menyebutkan apa pun.
//
// Yang diperiksa dua hal, dan keduanya diperlukan:
//
//	jumlah   penanda tertinggi harus SAMA dengan banyaknya parameter
//	kerapatan  :1 … :N harus utuh — melompati satu nomor adalah galat yang sama
func TestJumlahBindCocokDenganParameter(t *testing.T) {
	// Satu penyaring yang MENGISI seluruh medan, supaya penyusun parameter yang
	// bercabang pada isi penyaring mengambil cabang terpanjangnya.
	filter := reportklaim.Filter{
		From:             time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:               time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC),
		BusinessCode:     "B1",
		ComplianceStatus: "1",
		Detail:           true,
		Entity:           "ASM",
	}

	for code, p := range plans {
		// Diperiksa untuk SETIAP lini bisnis, bukan satu saja: tiga laporan memakai
		// kueri DAN parameter yang berbeda untuk Non-MBU, dan ketidakcocokannya dapat
		// mengenai satu cabang saja.
		for _, line := range lineYangDiuji() {
			f := filter
			f.BusinessLine = line

			name := p.query(f)
			if !hasQuery(name) {
				continue // kueri yang memang belum dipindahkan
			}

			t.Run(fmt.Sprintf("%s/%s", code, line), func(t *testing.T) {
				periksaBind(t, name, getQuery(name), len(p.args(f)))
			})
		}
	}
}

// periksaBind membandingkan penanda di dalam satu kueri dengan banyaknya parameter.
func periksaBind(t *testing.T, name, query string, args int) {
	t.Helper()

	terpakai := map[int]bool{}
	tertinggi := 0
	for _, m := range penandaBind.FindAllStringSubmatch(query, -1) {
		n, err := strconv.Atoi(m[1])
		if err != nil {
			continue
		}
		terpakai[n] = true
		if n > tertinggi {
			tertinggi = n
		}
	}

	if tertinggi != args {
		t.Errorf("kueri %q memakai penanda sampai :%d tetapi menerima %d parameter",
			name, tertinggi, args)
		return
	}
	for n := 1; n <= tertinggi; n++ {
		if !terpakai[n] {
			t.Errorf("kueri %q melompati penanda :%d — penomorannya harus rapat 1..%d",
				name, n, tertinggi)
		}
	}
}

// lineYangDiuji mengembalikan seluruh lini bisnis yang dapat dipilih, ditambah lini
// kosong — keadaan sebelum pengguna memilih apa pun.
func lineYangDiuji() []reportklaim.BusinessLine {
	line := []reportklaim.BusinessLine{""}
	for _, o := range reportklaim.BusinessLineOptions() {
		line = append(line, o.Value)
	}
	return line
}

package sqlstore

import (
	"strings"
	"testing"
)

// TestCommentQuotesBalancedPerLine menjaga jebakan go-ora yang sudah menggigit dua kali.
//
// Pengurai bind go-ora TIDAK melewati komentar. Kutip yang terbuka di satu baris komentar
// dan tertutup di baris lain membuat penanda bind di antaranya terbaca sebagai literal,
// dan pernyataan yang sebenarnya sah ditolak dengan ORA-00900.
//
// Gejalanya menyesatkan dan mahal: galatnya hanya muncul saat kueri dijalankan terhadap
// Oracle, pesannya tidak menyebut baris mana pun, dan membuang satu baris komentar justru
// membuatnya bertahan. Uji ini memindahkan penemuannya dari "saat petugas menekan tombol"
// menjadi "saat berkasnya disimpan".
//
// Yang diperiksa hanya baris KOMENTAR. Kutip di dalam SQL-nya sendiri memang berpasangan
// lintas baris pada beberapa kueri, dan itu sah.
func TestCommentQuotesBalancedPerLine(t *testing.T) {
	berkas, err := queryFiles.ReadDir(".")
	if err != nil {
		t.Fatalf("membaca direktori kueri: %v", err)
	}

	diperiksa := 0
	for _, f := range berkas {
		if !strings.HasSuffix(f.Name(), ".sql") {
			continue
		}
		diperiksa++

		isi, err := queryFiles.ReadFile(f.Name())
		if err != nil {
			t.Fatalf("membaca %s: %v", f.Name(), err)
		}

		for nomor, baris := range strings.Split(string(isi), "\n") {
			teks := strings.TrimSpace(strings.TrimRight(baris, "\r"))
			if !strings.HasPrefix(teks, "--") {
				continue
			}
			for _, kutip := range []string{`"`, `'`} {
				if strings.Count(teks, kutip)%2 != 0 {
					t.Errorf("%s:%d kutip %s tidak berpasangan di dalam komentar:\n  %s\n"+
						"Kutip yang membentang antar-baris komentar membuat go-ora "+
						"salah membaca penanda bind (ORA-00900). Tutup kutipnya di baris "+
						"yang sama, atau buang tanda kutipnya.",
						f.Name(), nomor+1, kutip, teks)
				}
			}
		}
	}

	if diperiksa == 0 {
		t.Fatal("tidak ada berkas .sql yang diperiksa — pemindaiannya tidak bekerja")
	}
	t.Logf("%d berkas kueri diperiksa", diperiksa)
}

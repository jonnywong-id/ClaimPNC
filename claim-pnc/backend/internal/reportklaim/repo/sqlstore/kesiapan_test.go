package sqlstore

import (
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/reportklaim"
)

// Setiap laporan yang katalognya nyatakan SIAP wajib benar-benar punya kueri.
//
// # Kenapa uji ini ada
//
// Katalog dan daftar rencana adalah dua berkas terpisah, dan keduanya disunting pada saat
// yang berbeda. Satu laporan yang ditandai `Ready()` tanpa entri di `plans` akan lolos
// kompilasi, lolos seluruh uji lain, muncul sebagai kartu bertombol di layar — lalu gagal
// tepat ketika pengguna menekan tombolnya.
//
// Pesannya pun tidak membantu pengguna: "laporan %q tidak punya kueri" adalah kalimat
// untuk pengembang. Uji ini memindahkan penemuannya dari layar pengguna ke sini.
//
// Kebalikannya sudah dijaga `TestTidakAdaKueriYatim` — kueri yang tidak dipakai rencana
// mana pun. Keduanya bersama menutup kedua arah.
func TestSetiapLaporanSiapPunyaKueri(t *testing.T) {
	// Penyaring kosong cukup: yang diuji keberadaan kuerinya, bukan isinya. Laporan yang
	// kuerinya bercabang menurut lini diuji tiap cabangnya di bawah.
	for _, r := range reportklaim.Catalog() {
		if !r.Availability.Ready {
			continue
		}

		p, ada := plans[r.Code]
		require.Truef(t, ada,
			"laporan %q (%s) ditandai siap tetapi tidak punya rencana kueri", r.Code, r.Title)

		for _, f := range kombinasiPenyaring {
			name := p.query(f)
			require.Truef(t, hasQuery(name),
				"laporan %q lini %q menunjuk kueri %q yang tidak ada",
				r.Code, f.BusinessLine, name)
		}
	}
}

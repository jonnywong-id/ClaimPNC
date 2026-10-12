// Package asset menyimpan berkas biner bersama — logo dan gambar lain yang dicetak di
// dokumen aplikasi — dan menyerahkannya menurut IDENTITAS, bukan menurut nama berkas.
//
// # Kenapa identitas, bukan jalur berkas
//
// Di Pega berkas seperti ini adalah Binary File (`webweb/LogoASM.PNG`) yang dirujuk langsung
// dari template HTML. Di sini satu identitas — mis. `LogoASM` — dipakai modul mana pun yang
// membutuhkannya, dan setiap pemakai menentukan UKURANNYA sendiri: DLA dan PLA
// menggambarnya 55 mm, layar atau dokumen lain boleh lebih kecil atau lebih besar. Mengganti
// logo cukup mengganti berkasnya di sini, tanpa menyentuh satu pun pemakai.
//
// Berkas ditanam ke binary (`go:embed`), sehingga tidak ada jalur berkas yang dapat hilang
// saat aplikasi dipasang di mesin lain.
package asset

import (
	"embed"
	"fmt"
	"strings"
)

// Identitas berkas yang tersedia.
const (
	// LogoASM adalah logo PT Asuransi Sinar Mas — kop dokumen cetak portal ASM. Sumbernya
	// `Binary File/LogoForm.jpg` dari Work Owner (2026-10-12), padanan `webweb/LogoASM.PNG`
	// pada template DLA Pega.
	LogoASM = "LogoASM"
)

//go:embed LogoASM.jpg
var files embed.FS

// berkas memetakan identitas ke nama berkas di dalam paket ini.
var berkas = map[string]string{
	LogoASM: "LogoASM.jpg",
}

// portalLogo memetakan portal ke identitas logo kopnya. Portal yang tidak terdaftar di sini
// BELUM punya logo dan dicetak tanpa logo — bukan memakai logo portal lain, karena setiap
// portal adalah badan hukum tersendiri (`ADR-0030`).
var portalLogo = map[string]string{
	"ASM": LogoASM,
}

// Get mengembalikan isi berkas untuk satu identitas. Identitas yang tidak dikenal adalah galat.
func Get(id string) ([]byte, error) {
	name, ok := berkas[id]
	if !ok {
		return nil, fmt.Errorf("asset: identitas %q tidak dikenal", id)
	}
	b, err := files.ReadFile(name)
	if err != nil {
		return nil, fmt.Errorf("asset: membaca %s: %w", name, err)
	}
	return b, nil
}

// MustGet sama dengan Get, tetapi panik bila gagal. Hanya untuk perakitan saat start: berkas
// tertanam di binary, sehingga kegagalannya adalah cacat build, bukan keadaan saat berjalan.
func MustGet(id string) []byte {
	b, err := Get(id)
	if err != nil {
		panic(err)
	}
	return b
}

// PortalLogos mengembalikan logo kop seluruh portal yang punya logo, dikunci alias portal
// (huruf besar). Dipakai renderer dokumen cetak yang memilih logo menurut portal klaim.
func PortalLogos() map[string][]byte {
	out := make(map[string][]byte, len(portalLogo))
	for portal, id := range portalLogo {
		out[strings.ToUpper(portal)] = MustGet(id)
	}
	return out
}

package sqlstore

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/reportklaim"
)

// polaAlias menangkap `AS "X"` — satu-satunya cara kueri modul ini menamai kolomnya.
var polaAlias = regexp.MustCompile(`(?i)\bas\s*"([^"]+)"`)

// aliasKueri mengumpulkan seluruh nama kolom yang dihasilkan sebuah kueri.
//
// Ia ikut menangkap alias pada anak-kueri, dan itu diterima: kelebihan nama hanya membuat
// uji ini lebih longgar, tidak pernah membuatnya lolos pada kolom yang benar-benar tidak
// punya sumber.
func aliasKueri(sql string) map[string]bool {
	out := map[string]bool{}
	for _, m := range polaAlias.FindAllStringSubmatch(sql, -1) {
		out[strings.ToLower(m[1])] = true
	}
	return out
}

// Setiap kolom yang ditulis ke berkas HARUS punya sumber — alias kueri, atau kolom
// turunan yang benar-benar diisi penghitungnya.
//
// # Kenapa uji ini yang paling berharga di berkas ini
//
// Karena kolom tanpa sumber TIDAK menghasilkan galat apa pun. Ia menghasilkan sel kosong,
// dan sel kosong pada berkas berisi puluhan ribu baris tidak dapat dibedakan dari data
// yang memang tidak ada. Tiga cacat semacam itu sudah ditemukan di export dengan membaca
// tangan — `IsCFS` pada Reject Klaim, `month CopyFrom` dan `KomiteAccepted` pada Data
// Komite — dan ketiganya sudah berjalan bertahun-tahun tanpa ada yang menyadarinya.
//
// Sejak uji ini ada, salah ketik satu huruf pada alias membuat rakitan gagal, bukan
// membuat satu kolom diam-diam kosong.
//
// # Kenapa penghitung kolom turunan dijalankan, bukan didaftar
//
// Daftar tulisan tangan akan menyimpang begitu sebuah kolom turunan ditambahkan. Di sini
// penghitungnya benar-benar dijalankan atas baris kosong, dan kunci yang ditulisnya
// dihitung sebagai sumber. Repo tanpa koneksi memang menghasilkan master yang tidak
// tersedia — dan itu justru yang dikehendaki: yang diuji nama kolomnya, bukan isinya.
func TestSetiapKolomBerkasPunyaSumber(t *testing.T) {
	repo := NewRepo(nil, nil)
	var kurang []string

	for _, report := range reportklaim.CatalogInPegaOrder() {
		p, ada := plans[report.Code]
		if !ada {
			continue
		}

		for _, f := range kombinasiPenyaring {
			kolom := report.Columns(f)
			if len(kolom) == 0 {
				continue
			}
			name := p.query(f)
			if !hasQuery(name) {
				continue
			}

			sumber := aliasKueri(getQuery(name))
			if p.derive != nil {
				hitung, err := p.derive(t.Context(), repo, f)
				require.NoErrorf(t, err, "menyiapkan kolom turunan %s", report.Code)
				if hitung != nil {
					row := reportklaim.Row{}
					hitung(row)
					for field := range row {
						sumber[strings.ToLower(field)] = true
					}
				}
			}

			kosong := map[string]bool{}
			for _, name := range kolomTanpaSumber[report.Code] {
				kosong[strings.ToLower(name)] = true
			}

			for _, c := range kolom {
				if sumber[strings.ToLower(c.Field)] {
					require.Falsef(t, kosong[strings.ToLower(c.Field)],
						"laporan %s: %q terdaftar di kolomTanpaSumber tetapi ternyata PUNYA sumber; buang namanya",
						report.Code, c.Field)
					continue
				}
				if kosong[strings.ToLower(c.Field)] {
					continue
				}
				// Dikumpulkan, bukan digagalkan seketika: satu alias yang salah ketik
				// jarang berdiri sendiri, dan melaporkannya satu per satu berarti
				// menjalankan uji ini belasan kali untuk menemukan daftar yang sama.
				kurang = append(kurang, fmt.Sprintf(
					"laporan %s lini %q: kolom %q (judul %q) tidak dihasilkan kueri %q maupun kolom turunannya",
					report.Code, f.BusinessLine, c.Field, c.Header, name))
			}
		}
	}

	require.Emptyf(t, kurang, "%d kolom berkas tanpa sumber:\n%s",
		len(kurang), strings.Join(kurang, "\n"))
}

// Kolom bantu TIDAK boleh ikut ke berkas.
//
// Ia kebalikan uji di atas, dan sama pentingnya: kolom bantu yang lupa dibuang muncul di
// berkas sebagai lajur tanpa judul yang menggeser seluruh kolom sesudahnya pada pembaca
// yang membaca menurut posisi.
func TestKolomBantuTidakIkutKeBerkas(t *testing.T) {
	repo := NewRepo(nil, nil)

	bantu := map[reportklaim.Code][]string{
		reportklaim.CodeTAT:                 tatHelperColumn,
		reportklaim.CodeCloseKlaim:          closeNonMBUHelperColumn,
		reportklaim.CodeTemporaryCloseKlaim: closeNonMBUHelperColumn,
		reportklaim.CodeKomite:              komiteNonMBUHelperColumn,
	}

	for code, daftar := range bantu {
		report, ok := reportklaim.Find(code)
		require.True(t, ok)

		p := plans[code]
		f := reportklaim.Filter{BusinessLine: reportklaim.BusinessLineNonMBU}
		if code == reportklaim.CodeTAT {
			f = reportklaim.Filter{BusinessLine: reportklaim.BusinessLinePA}
		}

		hitung, err := p.derive(t.Context(), repo, f)
		require.NoError(t, err)
		require.NotNilf(t, hitung, "laporan %s seharusnya punya kolom turunan pada lini ini", code)

		row := reportklaim.Row{}
		for _, name := range daftar {
			row[name] = "isi apa pun"
		}
		hitung(row)

		for _, name := range daftar {
			_, masih := row[name]
			require.Falsef(t, masih, "laporan %s: kolom bantu %q tidak dibuang", code, name)
		}

		// Dan kolom bantu itu memang bukan kolom berkas — bila ia ternyata terdaftar di
		// katalog, membuangnya adalah cacat, bukan kerapian.
		for _, c := range report.Columns(f) {
			for _, name := range daftar {
				require.NotEqualf(t, name, c.Field,
					"laporan %s: %q terdaftar sebagai kolom berkas tetapi dibuang penghitung", code, name)
			}
		}
	}
}

// Nama kolom bantu tidak boleh bertabrakan dengan nama kolom berkas laporan mana pun.
//
// Tabrakan seperti itu membuat penghitung membuang kolom yang justru harus ditulis, dan
// akibatnya hanya terlihat pada satu laporan tertentu.
func TestNamaKolomBantuTidakBertabrakan(t *testing.T) {
	semua := append(append(append([]string{}, tatHelperColumn...),
		closeNonMBUHelperColumn...), komiteNonMBUHelperColumn...)

	for _, report := range reportklaim.CatalogInPegaOrder() {
		for _, f := range kombinasiPenyaring {
			for _, c := range report.Columns(f) {
				for _, name := range semua {
					require.NotEqualf(t, strings.ToLower(name), strings.ToLower(c.Field),
						"laporan %s lini %q memakai %q sebagai kolom berkas, dan nama itu dipakai sebagai kolom bantu",
						report.Code, f.BusinessLine, c.Field)
				}
			}
		}
	}
}

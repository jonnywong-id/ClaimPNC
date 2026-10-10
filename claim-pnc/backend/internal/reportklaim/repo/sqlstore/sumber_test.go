package sqlstore

import (
	"html"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Uji ini menyapu satu kelas cacat yang BASIS DATA TIDAK DAPAT MENEMUKAN:
// kolom yang di Pega berupa CASE atau sub-kueri, tetapi dipindahkan menjadi kolom biasa.
//
// # Kenapa basis data tidak bisa menemukannya
//
// Kolom biasa itu SAH. Ia ada, tipenya benar, kuerinya jalan — ia hanya mengambil nilai
// dari tempat yang salah. `EXPLAIN PLAN` lulus, menjalankannya lulus, dan berkasnya terisi
// dengan angka yang salah tanpa satu pun tanda.
//
// Ini benar-benar terjadi. Pada `report_os_komite`, dua kolom occupation yang di Pega
// memilih di antara TIGA tabel menurut Group Panel dipindahkan menjadi dua kolom
// `pega_dashboardpnc`. Salah satunya menyalak karena kolomnya memang tidak ada; yang
// satunya ADA, dan akan diam selamanya.
//
// # Yang TIDAK dianggap cacat
//
// Mengganti sub-kueri berkorelasi dengan JOIN adalah penulisan ulang yang setara dan
// disengaja — `report_klaim_harian` (tabel turunan beragregat), `report_pengiriman_pla`
// (satu LEFT JOIN menggantikan empat sub-kueri), dan kolom progres yang memakai LEFT JOIN
// ber-MAX. Ketiganya ada di daftar penulisanUlangSetara di bawah, disebut satu per satu
// supaya penambahan berikutnya menuntut seseorang menyadarinya.
func TestTidakAdaKolomYangDipendekkanDariSumbernya(t *testing.T) {
	akar := akarRepo(t)
	if akar == "" {
		t.Skip("export rule Pega tidak ada di pohon ini — uji ini hanya berjalan di repositori lengkap")
	}

	// penulisanUlangSetara: kueri -> alias kolom yang SENGAJA ditulis ulang menjadi JOIN.
	//
	// Masing-masing sudah dibandingkan ke sumbernya dan terbukti menghasilkan nilai yang
	// sama; yang berubah hanya bentuknya.
	penulisanUlangSetara := map[string]map[string]string{
		"report_klaim_harian": {
			"ALASANTERLAMBAT":        "tabel turunan adj — MAX(asm_share)",
			"COMPLIANCEREMARK":       "tabel turunan adj — SUM(total_claim * currencyvalue)",
			"CONVEYANCE":             "tabel turunan adj — SUM(grossvalue * currencyvalue)",
			"CLOSECLAIMNOTE":         "tabel turunan adj — SUM(individual_risk_value * currencyvalue)",
			"COMMENTKOMITECLOSECASE": "tabel turunan adj — SUM(total_claim * share)",
			"COMPLIANCEPOSAUDITBYR":  "tabel turunan adj — LOC + salvage",
			"USERNAME":               "tabel turunan adj — nilai bersih",
		},
		"report_close_klaim_nonmbu": {
			"PYNOTE":         "LEFT JOIN gcnm_progress_claim ber-MAX(id_update)",
			"STATUSRECEIVER": "LEFT JOIN gcnm_progress_claim ber-MAX(id_update)",
		},
		"report_komite_nonmbu": {
			"PYNOTE": "LEFT JOIN gcnm_progress_claim ber-MAX(id_update)",
		},
		"report_pengiriman_pla": {
			"ALASANKLAIM": "LEFT JOIN pc_asm_fw_gcnmfw_work menggantikan empat sub-kueri",
			"KEYWORD":     "idem",
			"OTHER":       "idem; pemformatan tanggalnya pindah ke Go",
			"CASEID":      "idem",
			"USERTEKNIS":  "idem",
		},
	}

	dibandingkan := 0
	// Dibaca dari BERKASNYA, bukan dari peta `query`: pemuatnya membuang komentar kepala,
	// dan baris `-- Asal:` justru di situ. Membacanya dari peta menghasilkan nol
	// perbandingan — dan nol perbandingan yang lulus diam-diam adalah uji yang tidak
	// menguji apa pun. Itu sebabnya ada ambang di akhir fungsi ini.
	for name, text := range kueriBerkomentar(t) {
		asal := asalRule(text)
		if asal == "" {
			continue
		}
		xml, err := os.ReadFile(filepath.Join(akar, filepath.FromSlash(asal)))
		if err != nil {
			continue
		}
		sumber := browseSQL(string(xml))
		if sumber == "" {
			continue
		}

		punyaPega := kolomSelect(sumber)
		if len(punyaPega) == 0 {
			continue
		}
		dibandingkan++

		for alias, punyaSaya := range kolomSelect(text) {
			pega, ada := punyaPega[alias]
			if !ada || !mengambilKeputusan(pega) || !sebutanKolomBiasa(punyaSaya) {
				continue
			}
			require.Containsf(t, penulisanUlangSetara[name], alias,
				"kueri %q kolom %q: sumbernya mengambil keputusan (CASE/sub-kueri) tetapi di sini "+
					"hanya menyebut %q. Bila itu penulisan ulang yang setara, daftarkan di "+
					"penulisanUlangSetara beserta alasannya.\n  Pega: %s",
				name, alias, punyaSaya, ringkas(pega))
		}
	}

	require.GreaterOrEqual(t, dibandingkan, 20,
		"terlalu sedikit kueri yang berhasil dibandingkan — uji ini kemungkinan tidak menguji apa pun")
}

// kueriBerkomentar membaca berkas .sql APA ADANYA — berikut komentar kepalanya.
func kueriBerkomentar(t *testing.T) map[string]string {
	t.Helper()

	berkas, err := filepath.Glob("*.sql")
	require.NoError(t, err)

	out := map[string]string{}
	for _, f := range berkas {
		isi, err := os.ReadFile(f)
		require.NoError(t, err)

		for _, blok := range strings.Split(string(isi), "-- name: ")[1:] {
			baris := strings.SplitN(blok, "\n", 2)
			if len(baris) != 2 {
				continue
			}
			out[strings.TrimSpace(baris[0])] = blok
		}
	}
	return out
}

// akarRepo mencari direktori yang memuat export rule Pega, naik dari berkas uji ini.
func akarRepo(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	require.NoError(t, err)

	for i := 0; i < 8; i++ {
		if st, err := os.Stat(filepath.Join(dir, "RDB List")); err == nil && st.IsDir() {
			return dir
		}
		induk := filepath.Dir(dir)
		if induk == dir {
			break
		}
		dir = induk
	}
	return ""
}

var polaAsal = regexp.MustCompile("(?m)^-- Asal: `([^`]+\\.xml)`")

func asalRule(text string) string {
	m := polaAsal.FindStringSubmatch(text)
	if m == nil {
		return ""
	}
	return m[1]
}

func browseSQL(xml string) string {
	const buka, tutup = "<pyBrowseSQL>", "</pyBrowseSQL>"
	i := strings.Index(xml, buka)
	j := strings.Index(xml, tutup)
	if i < 0 || j < i {
		return ""
	}
	return html.UnescapeString(xml[i+len(buka) : j])
}

var polaAliasSumber = regexp.MustCompile(`(?is)\bAS\s+"([^"]+)"\s*$`)

// kolomSelect memetakan alias kolom keluaran ke ekspresinya.
func kolomSelect(sql string) map[string]string {
	naik := strings.ToUpper(sql)
	iSel := strings.Index(naik, "SELECT")
	if iSel < 0 {
		return nil
	}
	iFrom := fromTingkatAtas(sql, naik, iSel)
	if iFrom < 0 {
		return nil
	}

	out := map[string]string{}
	for _, kolom := range pecahKoma(sql[iSel+len("SELECT") : iFrom]) {
		var isi []string
		for _, l := range strings.Split(kolom, "\n") {
			if !strings.HasPrefix(strings.TrimSpace(l), "--") {
				isi = append(isi, l)
			}
		}
		bersih := strings.TrimSpace(strings.Join(isi, "\n"))

		m := polaAliasSumber.FindStringSubmatchIndex(bersih)
		if m == nil {
			continue
		}
		out[strings.ToUpper(bersih[m[2]:m[3]])] = strings.TrimSpace(bersih[:m[0]])
	}
	return out
}

// fromTingkatAtas mencari FROM yang BUKAN milik sub-kueri.
func fromTingkatAtas(sql, naik string, mulai int) int {
	dalam, kutip := 0, false
	for i := mulai; i+5 < len(sql); i++ {
		switch c := sql[i]; {
		case c == '\'':
			kutip = !kutip
		case kutip:
		case c == '(':
			dalam++
		case c == ')':
			dalam--
		case dalam == 0 && (c == ' ' || c == '\n' || c == '\t' || c == '\r'):
			if naik[i+1:i+5] == "FROM" && (i+5 >= len(naik) || naik[i+5] == ' ' || naik[i+5] == '\n' || naik[i+5] == '\t' || naik[i+5] == '\r') {
				return i
			}
		}
	}
	return -1
}

// pecahKoma memecah pada koma TINGKAT ATAS saja.
func pecahKoma(teks string) []string {
	var out []string
	dalam, kutip, kutipGanda, mulai := 0, false, false, 0

	for i := 0; i < len(teks); i++ {
		switch c := teks[i]; {
		case c == '\'' && !kutipGanda:
			kutip = !kutip
		case c == '"' && !kutip:
			kutipGanda = !kutipGanda
		case kutip || kutipGanda:
		case c == '(':
			dalam++
		case c == ')':
			dalam--
		case c == ',' && dalam == 0:
			out = append(out, teks[mulai:i])
			mulai = i + 1
		}
	}
	return append(out, teks[mulai:])
}

var (
	polaKeputusan  = regexp.MustCompile(`(?i)\bCASE\b|\bSELECT\b|\bDECODE\b`)
	polaKolomBiasa = regexp.MustCompile(`^[A-Za-z_][\w$]*(\s*\.\s*[A-Za-z_][\w$]*)?$`)
	polaSpasi      = regexp.MustCompile(`\s+`)
)

func mengambilKeputusan(e string) bool { return polaKeputusan.MatchString(e) }
func sebutanKolomBiasa(e string) bool  { return polaKolomBiasa.MatchString(strings.TrimSpace(e)) }
func ringkas(e string) string {
	s := polaSpasi.ReplaceAllString(e, " ")
	if len(s) > 160 {
		s = s[:160] + "…"
	}
	return s
}

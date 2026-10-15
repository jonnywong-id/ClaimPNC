package masterreas_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterreas"
	"claim-pnc/internal/masterreas/repo/memory"
)

// TestNaturalKeyMemakaiTigaKolom membuktikan kunci baris mengikuti `UPDATEREAS.prc`, yang
// memeriksa keberadaan baris dengan REINSURERID + REINSURERNAME + TYPE sekaligus.
//
// Bukan REINSURERID saja: satu perusahaan reasuransi memang punya beberapa baris, satu per
// jenis dokumen, dan memperlakukan kodenya sebagai kunci akan membuat ketiganya dianggap
// baris yang sama.
func TestNaturalKeyMemakaiTigaKolom(t *testing.T) {
	satu := masterreas.Member{ReinsurerID: "RE-001", ReinsurerName: "Nusantara", Type: "1"}
	dua := masterreas.Member{ReinsurerID: "RE-001", ReinsurerName: "Nusantara", Type: "2"}

	require.NotEqual(t, satu.NaturalKey(), dua.NaturalKey(),
		"dua baris yang hanya berbeda TYPE harus punya kunci yang berbeda")
}

// TestNaturalKeyTidakBertabrakanLewatPemisah membuktikan pemisahnya bukan tanda baca biasa.
//
// Bila pemisahnya tanda hubung, dua baris berbeda di bawah ini akan menghasilkan kunci yang
// sama persis — dan layar akan menganggap keduanya satu baris. Nama perusahaan reasuransi
// memang memuat tanda baca; "Andalas Re - Syariah" bukan bentuk yang mengada-ada.
func TestNaturalKeyTidakBertabrakanLewatPemisah(t *testing.T) {
	satu := masterreas.Member{ReinsurerID: "RE", ReinsurerName: "001-Andalas", Type: "1"}
	dua := masterreas.Member{ReinsurerID: "RE-001", ReinsurerName: "Andalas", Type: "1"}

	require.NotEqual(t, satu.NaturalKey(), dua.NaturalKey())
}

// TestIsFallbackHanyaUntukTipeSatu membuktikan baris cadangan dikenali dari TYPE '1' saja.
//
// Nilainya dibaca dari dua tempat yang saling menguatkan — `BrowseEmailReas` yang memakainya
// sebagai cadangan (`or type = '1'`), dan `UPDATEREAS` yang menaikkan baris '1' menjadi tipe
// yang diminta.
func TestIsFallbackHanyaUntukTipeSatu(t *testing.T) {
	require.True(t, masterreas.Member{Type: "1"}.IsFallback())
	require.True(t, masterreas.Member{Type: " 1 "}.IsFallback(),
		"spasi ujung tidak boleh mengubah artinya; kolomnya bisa saja CHAR (R-08)")
	require.False(t, masterreas.Member{Type: "2"}.IsFallback())
	require.False(t, masterreas.Member{Type: ""}.IsFallback(),
		"TYPE kosong BUKAN cadangan — baris yang lahir dari GetListDataLoginReas memang "+
			"tidak punya TYPE sama sekali, dan itu justru keadaan yang perlu terlihat")
}

// TestFilterCleanMemangkasSpasi membuktikan kata kunci dipangkas sebelum dipakai.
//
// Tanpa itu, pencarian yang tidak sengaja diawali spasi akan menjadi pola LIKE `% budi%`
// dan tidak menemukan apa pun — kegagalan yang tidak dapat dijelaskan dari layar.
func TestFilterCleanMemangkasSpasi(t *testing.T) {
	require.Equal(t, "andalas", masterreas.Filter{Keyword: "  andalas  "}.Clean().Keyword)
	require.Equal(t, "", masterreas.Filter{Keyword: "   "}.Clean().Keyword)
}

// TestSampleListKonsisten membuktikan contoh pengembangan benar-benar memuat keadaan yang
// diklaim doc comment-nya.
//
// Uji ini menjaga contoh tetap berguna: bila kelak seseorang merapikan datanya — menyamakan
// login yang berbagi, atau menambahkan baris cadangan yang hilang — layar tidak akan pernah
// diuji terhadap bentuk data yang sebenarnya ada di produksi.
func TestSampleListKonsisten(t *testing.T) {
	list := memory.SampleList()
	require.Len(t, list, 7)

	tipePerReinsurer := map[string][]string{}
	loginPemakai := map[string]map[string]struct{}{}
	adaCountryKosong := false

	for _, one := range list {
		tipePerReinsurer[one.ReinsurerID] = append(tipePerReinsurer[one.ReinsurerID], one.Type)

		if loginPemakai[one.Login] == nil {
			loginPemakai[one.Login] = map[string]struct{}{}
		}
		loginPemakai[one.Login][one.ReinsurerID] = struct{}{}

		if strings.TrimSpace(one.Country) == "" {
			adaCountryKosong = true
		}

		require.NotEmpty(t, one.Email, "setiap contoh harus punya surel tujuan")
		require.Contains(t, one.Email, "@contoh.invalid",
			"surel contoh WAJIB memakai domain yang tidak dapat diselesaikan DNS (D-69)")
	}

	require.Len(t, tipePerReinsurer["RE-001"], 3,
		"satu perusahaan dengan tiga TYPE dibutuhkan agar kolom Tipe terbukti berguna")

	require.NotContains(t, tipePerReinsurer["RE-004"], masterreas.FallbackType,
		"satu perusahaan tanpa baris cadangan dibutuhkan agar keadaan itu dapat dicoba")

	require.Len(t, loginPemakai["AndalasRe"], 2,
		"satu login yang dipakai dua kode reas dibutuhkan agar risiko R-20 terlihat di layar")

	require.True(t, adaCountryKosong,
		"satu baris tanpa COUNTRY dibutuhkan; baris dari GetListDataLoginReas memang begitu")
}

package masterstatus_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterstatus"
	"claim-pnc/internal/masterstatus/repo/memory"
)

// Acceptance criteria TKT-F4-005: "Master status memuat TEPAT 33 kode 1134–1166 beserta
// labelnya — dihitung dan dilaporkan angkanya."
func TestMasterHoldsExactly33Codes(t *testing.T) {
	list := memory.SampleList()
	require.Len(t, list, 33, "V_STS_CLAIM memuat 33 kode; R-06 ditutup dengan angka ini")

	require.Equal(t, "1134", list[0].Code, "kode terkecil")
	require.Equal(t, "1166", list[32].Code, "kode terbesar")
}

// Kode pada SALINAN CSV berurutan tanpa lompatan, karena ia dibentuk
// id_site || lpad(urutan, 3, '0'). Lompatan berarti ada baris yang hilang dari salinan
// master di repo ini.
//
// Basis data yang berjalan justru PUNYA lompatan — lihat uji berikutnya.
func TestCodesInCSVCopyAreConsecutive(t *testing.T) {
	list := memory.SampleList()

	for i, s := range list {
		diharapkan := 1134 + i
		require.Equal(t, itoa(diharapkan), s.Code,
			"kode ke-%d seharusnya %d; deretnya tidak boleh berlubang", i+1, diharapkan)
	}
}

// Selisih antara CSV yang diserahkan Work Owner dan basis data yang berjalan.
//
// Pembacaan langsung POOLDATA.M_STS_CLAIM pada 2026-09-17 menemukan 32 baris, bukan 33:
// kode 1165 tidak ada di sana. Sebabnya belum dijelaskan.
//
// Uji ini tidak menguji kebenaran apa pun — ia MENGUNCI FAKTA supaya tidak hilang
// diam-diam. Bila kelak 1165 dihapus dari daftar contoh karena "produksi memang tidak
// punya", uji ini gagal dan memaksa pertanyaannya dibicarakan lebih dulu.
func TestDifferenceWithProductionDatabaseRecorded(t *testing.T) {
	const codeOnlyInCSV = "1165"

	existing := false
	for _, s := range memory.SampleList() {
		if s.Code == codeOnlyInCSV {
			existing = true
			require.Equal(t, "Rejected Chasier", s.Label)
		}
	}
	require.True(t, existing,
		"kode %s ada di Database/v_sts_claim.csv tetapi TIDAK ada di POOLDATA.M_STS_CLAIM "+
			"per 2026-09-17; selisihnya belum dijelaskan Work Owner dan tidak boleh dihapus "+
			"dari daftar ini tanpa jawaban", codeOnlyInCSV)
}

// Acceptance criteria TKT-F4-005: "Sebelas kode pertama (1134–1144) menyimpan penomoran
// lama 01–11 sebagai kolom terpisah, sehingga data historis tetap terbaca."
func TestFirstElevenCodesCarryLegacyNumbering(t *testing.T) {
	list := memory.SampleList()

	for i := 0; i < 11; i++ {
		require.NotEmpty(t, list[i].LegacyCode,
			"kode %s seharusnya membawa penomoran lama", list[i].Code)
		require.Equal(t, zeroPadded(i+1), list[i].LegacyCode,
			"penomoran lama kode %s", list[i].Code)
	}
	for i := 11; i < len(list); i++ {
		require.Empty(t, list[i].LegacyCode,
			"kode %s tidak pernah punya penomoran lama", list[i].Code)
	}
}

// Arti kode TIDAK BOLEH disimpulkan dari pemakaiannya di rule. Tiga kesimpulan yang
// pernah diambil begitu terbukti salah seluruhnya (R-06); uji ini mengunci arti yang
// sebenarnya supaya kesalahan itu tidak dapat masuk kembali lewat perubahan data contoh.
func TestMeaningOfThreeCodesOnceWronglyInferred(t *testing.T) {
	arti := map[string]string{}
	for _, s := range memory.SampleList() {
		arti[s.Code] = s.Label
	}

	require.Equal(t, "Close Claim for this object", arti["1143"], "1143 BUKAN status awal")
	require.Equal(t, "LOD Report", arti["1150"], "1150 BUKAN penanda terdaftar")
	require.Equal(t, "Analyst", arti["1151"], "1151 BUKAN Investigator")
}

// Label kosong ditolak. Ini perbedaan yang DISENGAJA dari sistem lama, yang menandai
// isian ini pyRequired=false — dan label kosong akan tampil sebagai status kosong di 23
// rule Pega yang membaca V_STS_CLAIM.
func TestEmptyLabelRejected(t *testing.T) {
	for _, input := range []string{"", " ", "\t", "   \n  "} {
		pelanggaran := masterstatus.CheckLabel(input)
		require.Len(t, pelanggaran, 1, "masukan %q seharusnya melanggar tepat satu aturan", input)
		require.Equal(t, masterstatus.FieldLabel, pelanggaran[0].Field,
			"pelanggaran harus menunjuk kolomnya, supaya layar dapat menandainya")
	}
}

func TestFilledLabelAccepted(t *testing.T) {
	require.Empty(t, masterstatus.CheckLabel("Paid"))
	require.Empty(t, masterstatus.CheckLabel("  Close Claim for this object  "),
		"spasi tepi dibuang sebelum diperiksa")
}

func TestTooLongLabelRejected(t *testing.T) {
	pas := strings.Repeat("a", masterstatus.MaxLabelLength)
	require.Empty(t, masterstatus.CheckLabel(pas), "tepat di batas masih diterima")

	lebih := strings.Repeat("a", masterstatus.MaxLabelLength+1)
	require.Len(t, masterstatus.CheckLabel(lebih), 1, "satu karakter di atas batas ditolak")
}

// Panjang dihitung dalam rune, bukan byte. Satu huruf beraksen memakan dua byte, dan
// menghitung byte akan membuat batasnya terasa berubah-ubah bagi pengguna.
func TestLabelLengthCountedInRunesNotBytes(t *testing.T) {
	// 100 rune, 200 byte.
	seratusRune := strings.Repeat("é", masterstatus.MaxLabelLength)
	require.Len(t, []byte(seratusRune), 2*masterstatus.MaxLabelLength, "prasyarat uji")

	require.Empty(t, masterstatus.CheckLabel(seratusRune),
		"100 huruf beraksen masih 100 karakter, bukan 200")
}

// Violation dikumpulkan SELURUHNYA, tidak berhenti pada yang pertama — meniru perilaku
// sistem lama yang menampilkan semua pesan validasi sekaligus.
func TestAllViolationsReturnedAtOnce(t *testing.T) {
	// Satu masukan yang melanggar dua aturan tidak mungkin dibuat di sini: kosong dan
	// kepanjangan saling meniadakan. Yang diuji adalah bentuk galatnya membawa senarai,
	// bukan satu pesan.
	err := masterstatus.NewValidationError([]masterstatus.Violation{
		{Field: masterstatus.FieldLabel, Message: "satu"},
		{Field: masterstatus.FieldLabel, Message: "dua"},
	})
	require.Error(t, err)

	var validasi *masterstatus.ValidationError
	require.ErrorAs(t, err, &validasi)
	require.Len(t, validasi.Violation, 2)
}

func TestEmptyValidationErrorIsTrulyNil(t *testing.T) {
	err := masterstatus.NewValidationError(nil)
	require.NoError(t, err, "nil bertipe harus benar-benar nil supaya if err != nil terbaca apa adanya")
}

// "Paid" dan "PAID  " adalah status yang sama bagi pengguna. Ekspresi yang sama dipakai
// indeks unik di basis data (UPPER(TRIM(LSC_NOTE))), sehingga keduanya tidak dapat
// berbeda pendapat.
func TestLabelKeyIgnoresCaseAndSpaces(t *testing.T) {
	require.Equal(t, masterstatus.LabelKey("Paid"), masterstatus.LabelKey("PAID"))
	require.Equal(t, masterstatus.LabelKey("Paid"), masterstatus.LabelKey("  paid  "))
	require.NotEqual(t, masterstatus.LabelKey("Paid"), masterstatus.LabelKey("Unpaid"))
}

// OLD_LSC_ID tersimpan sebagai CHAR berisi padding ("01  "). Spasi itu tidak pernah
// dimaksudkan sebagai bagian nilainya.
func TestCleanStripsCharColumnPadding(t *testing.T) {
	rapi := masterstatus.ClaimStatus{
		Code:       " 1134 ",
		Label:      "  Abbreviated Report  ",
		LegacyCode: "01  ",
	}.Clean()

	require.Equal(t, "1134", rapi.Code)
	require.Equal(t, "Abbreviated Report", rapi.Label)
	require.Equal(t, "01", rapi.LegacyCode)
}

// Label pada master yang ada semuanya unik. Bila kelak tidak, indeks unik migrasi 0002
// akan gagal dibuat — dan itu harus diketahui dari sini, bukan dari DBA saat migrasi.
func TestAllLabelsInMasterAreUnique(t *testing.T) {
	terlihat := map[string]string{}
	for _, s := range memory.SampleList() {
		key := masterstatus.LabelKey(s.Label)
		previous, existing := terlihat[key]
		require.False(t, existing, "label %q dipakai dua kali: kode %s dan %s", s.Label, previous, s.Code)
		terlihat[key] = s.Code
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

func zeroPadded(n int) string {
	s := itoa(n)
	if len(s) < 2 {
		return "0" + s
	}
	return s
}

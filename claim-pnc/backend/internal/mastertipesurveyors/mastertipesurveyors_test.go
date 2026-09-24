package mastertipesurveyors_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/mastertipesurveyors"
	"claim-pnc/internal/mastertipesurveyors/repo/memory"
)

// ── Isi master, dibaca langsung dari basis data ──────────────────────────────────

// Keempat baris ini DIBACA dari POOLDATA.V_M_SURVEYORS portal ASM pada 2026-09-19, bukan
// disalin dari dokumen mana pun. Uji ini mengunci keadaan itu supaya perubahannya kelak
// terlihat sebagai uji yang gagal, bukan sebagai daftar yang diam-diam bergeser.
func TestMasterHoldsExactlyFourTypes(t *testing.T) {
	list := memory.SampleList()
	require.Len(t, list, 4, "portal ASM memuat tepat empat tipe surveyor per 2026-09-19")
}

// Kodenya dibentuk `id_site || lpad(urutan, 3, '0')` — situs "1", urutan 1..4 — sehingga
// hasilnya berurutan tanpa lompatan. Uji ini yang membuktikan pembacaan itu, bukan
// kebetulan.
func TestCodesAreConsecutive(t *testing.T) {
	expected := []string{"1001", "1002", "1003", "1004"}

	actual := make([]string, 0, 4)
	for _, tipe := range memory.SampleList() {
		actual = append(actual, tipe.Code)
	}
	require.Equal(t, expected, actual)
}

// Tiga kode DIPATOK LANGSUNG di kueri Pega. Uji ini adalah pengingat yang dapat gagal:
// mengganti salah satu kode berarti memutus kueri yang menyebutnya.
//
//	1002  RDB List/BrowseSurveyorTypeLossAdjuster-SQL.xml   m_survey_id in ('1002')
//	1003  RDB List/BrowseSurveyorTypeExpert-SQL.xml         m_survey_id in ('1003')
//	1004  RDB List/BrowseSurveyorTypeSurveyAgent-SQL.xml    m_survey_id in ('1004')
func TestCodesPinnedInPegaQueriesStillPresent(t *testing.T) {
	byCode := map[string]string{}
	for _, tipe := range memory.SampleList() {
		byCode[tipe.Code] = tipe.Description
	}

	require.Equal(t, "LOSS ADJUSTER", byCode["1002"])
	require.Equal(t, "EXPERT", byCode["1003"])
	require.Equal(t, "SURVEY AGENT", byCode["1004"])
}

// OLD_M_SURVEY_ID KOSONG pada seluruh empat baris — berbeda dari Master Status Klaim yang
// sebelas kode pertamanya membawa penomoran lama. Dicatat sebagai uji supaya layar tidak
// dibangun dengan anggapan kolom itu pasti berisi.
func TestLegacyCodeEmptyOnEveryRow(t *testing.T) {
	for _, tipe := range memory.SampleList() {
		require.Emptyf(t, tipe.LegacyCode,
			"OLD_M_SURVEY_ID kosong pada seluruh baris; %q ternyata berisi", tipe.Code)
	}
}

// Tidak ada dua tipe bernama sama. Ini prasyarat langkah 1 migrasi 0003: indeks uniknya
// akan GAGAL dibuat bila ada nama ganda.
func TestAllDescriptionsUnique(t *testing.T) {
	seen := map[string]string{}
	for _, tipe := range memory.SampleList() {
		key := mastertipesurveyors.DescriptionKey(tipe.Description)
		previous, clash := seen[key]
		require.Falsef(t, clash, "%q dan %q memakai nama yang sama", previous, tipe.Code)
		seen[key] = tipe.Code
	}
}

// ── Aturan isian ────────────────────────────────────────────────────────────────

func TestEmptyDescriptionRejected(t *testing.T) {
	for _, masukan := range []string{"", "   ", "\t\n"} {
		violation := mastertipesurveyors.CheckDescription(masukan)
		require.Len(t, violation, 1, "masukan %q seharusnya ditolak", masukan)
		require.Equal(t, mastertipesurveyors.FieldDescription, violation[0].Field)
	}
}

func TestFilledDescriptionAccepted(t *testing.T) {
	require.Empty(t, mastertipesurveyors.CheckDescription("INTERNAL SURVEYOR"))
	require.Empty(t, mastertipesurveyors.CheckDescription("  Loss Adjuster  "))
}

func TestTooLongDescriptionRejected(t *testing.T) {
	tepatBatas := strings.Repeat("A", mastertipesurveyors.MaxDescriptionLength)
	require.Empty(t, mastertipesurveyors.CheckDescription(tepatBatas),
		"tepat pada batas masih diterima")

	lewatSatu := strings.Repeat("A", mastertipesurveyors.MaxDescriptionLength+1)
	require.Len(t, mastertipesurveyors.CheckDescription(lewatSatu), 1)
}

// Panjang dihitung dalam rune, bukan byte. Satu huruf beraksen memakan dua byte, dan
// menghitung byte akan membuat batasnya terasa berubah-ubah bagi pengguna.
func TestDescriptionLengthCountedInRunesNotBytes(t *testing.T) {
	// 100 rune, tetapi 200 byte.
	aksen := strings.Repeat("é", mastertipesurveyors.MaxDescriptionLength)
	require.Len(t, []byte(aksen), 2*mastertipesurveyors.MaxDescriptionLength)

	require.Empty(t, mastertipesurveyors.CheckDescription(aksen),
		"100 rune masih di dalam batas walau 200 byte")
}

// Spasi tepi dibuang SEBELUM panjangnya dihitung, sehingga menempelkan spasi tidak dapat
// dipakai untuk menembus batas maupun untuk menyelundupkan isian kosong.
func TestEdgeSpacesNotCounted(t *testing.T) {
	tepatBatas := "  " + strings.Repeat("A", mastertipesurveyors.MaxDescriptionLength) + "  "
	require.Empty(t, mastertipesurveyors.CheckDescription(tepatBatas))
}

// ── Galat validasi ──────────────────────────────────────────────────────────────

// Galat validasi memuat SELURUH pelanggaran, bukan yang pertama saja. Di modul ini hanya
// ada satu isian, sehingga yang diuji adalah bentuk pembawanya — yang akan menampung
// pelanggaran kedua begitu isian kedua ditambahkan.
func TestValidationErrorCarriesFieldAndMessage(t *testing.T) {
	err := mastertipesurveyors.NewValidationError(mastertipesurveyors.CheckDescription(""))
	require.Error(t, err)

	var validationError *mastertipesurveyors.ValidationError
	require.ErrorAs(t, err, &validationError)
	require.Len(t, validationError.Violation, 1)
	require.Equal(t, mastertipesurveyors.FieldDescription, validationError.Violation[0].Field)
	require.NotEmpty(t, validationError.Violation[0].Message)
	require.Contains(t, err.Error(), mastertipesurveyors.FieldDescription)
}

// Tanpa pelanggaran, yang dikembalikan harus nil YANG BENAR-BENAR NIL — bukan pointer nil
// terbungkus interface, yang akan membuat `if err != nil` di pemanggil bernilai true.
func TestEmptyValidationErrorIsTrulyNil(t *testing.T) {
	require.Nil(t, mastertipesurveyors.NewValidationError(nil))
	require.Nil(t, mastertipesurveyors.NewValidationError([]mastertipesurveyors.Violation{}))

	err := mastertipesurveyors.NewValidationError(nil)
	require.True(t, err == nil, "harus nil sejati, bukan pointer nil di dalam interface")
}

// ── Perapian dan kunci keunikan ─────────────────────────────────────────────────

// M_SURVEY_ID dan OLD_M_SURVEY_ID keduanya CHAR(4), dan Oracle memadatkan nilai CHAR
// dengan spasi tanpa memberi tanda apa pun.
func TestCleanStripsCharColumnPadding(t *testing.T) {
	kotor := mastertipesurveyors.SurveyorType{
		Code:        "1001",
		Description: "  INTERNAL SURVEYOR  ",
		LegacyCode:  "01  ",
	}
	bersih := kotor.Clean()

	require.Equal(t, "1001", bersih.Code)
	require.Equal(t, "INTERNAL SURVEYOR", bersih.Description)
	require.Equal(t, "01", bersih.LegacyCode)
}

// Perbandingan keunikan mengabaikan besar-kecil huruf dan spasi tepi. Ekspresinya HARUS
// sepadan dengan indeks `UPPER(TRIM(DESCRIPTION))` pada migrasi 0003 — bila keduanya
// berbeda, aplikasi akan menerima nama yang kemudian ditolak basis data.
func TestDescriptionKeyIgnoresCaseAndSpaces(t *testing.T) {
	require.Equal(t,
		mastertipesurveyors.DescriptionKey("Expert"),
		mastertipesurveyors.DescriptionKey("  EXPERT  "))
	require.NotEqual(t,
		mastertipesurveyors.DescriptionKey("EXPERT"),
		mastertipesurveyors.DescriptionKey("EXPERTS"))
}

// Besar-kecil huruf yang DIKETIK pengguna tidak diubah.
//
// Keempat nilai yang ada memang huruf besar, tetapi tidak ada satu pun rule yang
// menyeragamkannya: `@toUpperCase` di ValidasiMasterSurveyor bekerja atas
// TempDetailSurveyors.NAME, yaitu nama SURVEYOR di D_SURVEYORS — bukan deskripsi tipe.
// Menambahkan penyeragaman di sini berarti mengarang perilaku yang tidak pernah ada.
func TestCheckDoesNotChangeLetterCase(t *testing.T) {
	require.Empty(t, mastertipesurveyors.CheckDescription("Loss Adjuster"),
		"huruf kecil diterima apa adanya; hanya PERBANDINGAN yang diseragamkan")
}

package inputacceptation_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inputacceptation"
)

// Kunci isian dan kunci grid adalah KONTRAK API. Dua isian bernama sama berarti salah satunya
// tidak pernah sampai ke layar, dan yang hilang tidak menghasilkan galat apa pun — ia hanya
// kosong.
func TestKunciIsianTidakBoleh_Ganda(t *testing.T) {
	seen := map[string]string{}
	for _, group := range inputacceptation.Groups() {
		for _, field := range group.Fields {
			if previous, clash := seen[field.Key]; clash {
				t.Fatalf("isian %q dipakai dua kali: kelompok %s dan %s",
					field.Key, previous, group.Code)
			}
			seen[field.Key] = group.Code
		}
	}
}

func TestKunciGridTidakBoleh_Ganda(t *testing.T) {
	seen := map[string]bool{}
	for _, grid := range inputacceptation.GridList() {
		require.Falsef(t, seen[grid.Code], "grid %q terdaftar dua kali", grid.Code)
		seen[grid.Code] = true
	}
}

// Setiap grid yang disebut sebuah kelompok WAJIB ada, dan setiap grid WAJIB disebut tepat satu
// kelompok.
//
// Grid yang tidak disebut kelompok mana pun tidak akan pernah digambar; grid yang disebut dua
// kelompok akan digambar dua kali dengan isi yang sama.
func TestSetiapGridDisebutTepatSatuKelompok(t *testing.T) {
	disebut := map[string]int{}
	for _, group := range inputacceptation.Groups() {
		for _, code := range group.Grids {
			_, known := inputacceptation.FindGrid(code)
			require.Truef(t, known, "kelompok %s menyebut grid %q yang tidak ada",
				group.Code, code)
			disebut[code]++
		}
	}

	for _, grid := range inputacceptation.GridList() {
		require.Equalf(t, 1, disebut[grid.Code],
			"grid %q disebut %d kelompok, seharusnya tepat satu",
			grid.Code, disebut[grid.Code])
	}
}

// Isian yang TIDAK terhalang wajib punya jalur baca; yang terhalang wajib TIDAK punya.
//
// Jalur kosong pada isian yang tidak terhalang berarti isian itu selalu kosong tanpa ada yang
// menyatakan kenapa. Jalur berisi pada isian yang terhalang lebih buruk lagi: ia undangan
// untuk membacanya dari tempat yang salah.
func TestJalurBacaSejalanDenganKeadaanTerhalang(t *testing.T) {
	for _, field := range inputacceptation.Fields() {
		if field.Blocked {
			require.Emptyf(t, field.Path,
				"isian %q terhalang tetapi masih punya jalur baca", field.Key)
			require.NotEmptyf(t, field.BlockedReason,
				"isian %q terhalang tanpa alasan yang dapat ditampilkan", field.Key)
			require.NotEmptyf(t, field.BlockedOwner,
				"isian %q terhalang tanpa pemilik; penghalang tanpa alamat tidak pernah "+
					"hilang", field.Key)
			continue
		}
		require.NotEmptyf(t, field.Path, "isian %q tidak punya jalur baca", field.Key)
	}
}

// Setiap grid yang tidak terhalang wajib punya jalur senarai dan minimal satu kolom.
func TestSetiapGridPunyaJalurDanKolom(t *testing.T) {
	for _, grid := range inputacceptation.GridList() {
		if grid.Blocked {
			continue
		}
		require.NotEmptyf(t, grid.Path, "grid %q tidak punya jalur senarai", grid.Code)
		require.NotEmptyf(t, grid.Columns, "grid %q tidak punya satu pun kolom", grid.Code)

		seen := map[string]bool{}
		for _, column := range grid.Columns {
			require.Falsef(t, seen[column.Key],
				"grid %q memakai kunci kolom %q dua kali", grid.Code, column.Key)
			seen[column.Key] = true
			require.NotEmptyf(t, column.Path,
				"kolom %s.%s tidak punya jalur baca", grid.Code, column.Key)
		}
	}
}

// Isian terhalang tidak boleh sekaligus dapat diubah.
//
// Isian yang tidak dapat dibaca tetapi dapat dikirim balik adalah isian yang menulis ke tempat
// yang tidak diketahui.
func TestIsianTerhalangTidakDapatDiubah(t *testing.T) {
	editable := map[string]bool{}
	for _, key := range inputacceptation.EditableFields() {
		editable[key] = true
	}

	for _, field := range inputacceptation.Fields() {
		if field.Blocked {
			require.Falsef(t, editable[field.Key],
				"isian %q terhalang tetapi terdaftar dapat diubah", field.Key)
		}
	}
}

// Kesembilan isian yang terhalang memang kesembilan itu, bukan bertambah diam-diam.
//
// Daftarnya ditulis lengkap supaya penambahan isian terhalang baru — yang berarti layar makin
// kosong — tidak dapat lolos tanpa seseorang menyuntingnya di sini.
func TestDaftarIsianTerhalangTetap(t *testing.T) {
	want := []string{
		"ri_type", "class_of_business", "ceding_name", "sob_name",
		"bordereaux", "bordereaux_note", "accounting_mode", "teritorial_scope",
		"asm_share",
	}

	got := []string{}
	for _, field := range inputacceptation.Fields() {
		if field.Blocked {
			got = append(got, field.Key)
		}
	}

	require.ElementsMatch(t, want, got)
}

// Alasan terhalang menyebut HALAMAN yang benar.
//
// "Class Of Business" terhalang oleh halaman OfferFacIn, kedelapan lainnya oleh TreatyInMaster.
// Menyamakan alasannya akan mengirim Tim Pega mencari rule yang salah.
func TestAlasanTerhalangMenyebutHalamanYangBenar(t *testing.T) {
	for _, field := range inputacceptation.Fields() {
		if !field.Blocked {
			continue
		}
		if field.Key == "class_of_business" {
			require.Containsf(t, field.BlockedReason, "OfferFacIn",
				"isian %q terhalang oleh halaman OfferFacIn", field.Key)
			continue
		}
		require.Containsf(t, field.BlockedReason, "TreatyInMaster",
			"isian %q terhalang oleh halaman TreatyInMaster", field.Key)
	}
}

// Ketiga belas grid memang ketiga belas yang terbaca di section, dengan jalur senarainya.
//
// Jalur senarai yang salah TIDAK menghasilkan galat — gridnya hanya kosong. Uji ini yang
// membuat pertukaran jalur ketahuan.
func TestJalurSenaraiGridSesuaiSection(t *testing.T) {
	want := map[string]string{
		inputacceptation.GridInterest:        "InterestList",
		inputacceptation.GridInterestTotal:   "TotalInterestInsured",
		inputacceptation.GridClaimAmount:     "ListClaimAmount",
		inputacceptation.GridSpreadLoss:      "CNPSpreadLoss",
		inputacceptation.GridSpreadingRisk:   "SpreadingRisk",
		inputacceptation.GridTotalEstimation: "ListTotalEstimation",
		inputacceptation.GridReinstatement:   "ReinstatementList",
		inputacceptation.GridSpreadingClaim:  "SpreadingClaim",
		inputacceptation.GridSpreadBreakQS:   "SpreadingBreakQS",
		inputacceptation.GridAdjustment:      "AdjustmentList",
		inputacceptation.GridSpreadAdjust:    "SpreadingAdjustment",
		inputacceptation.GridSpreadAdjustQS:  "SpreadingAdjustmentQS",
		inputacceptation.GridSuggest:         "SuggestList",
	}

	got := map[string]string{}
	for _, grid := range inputacceptation.GridList() {
		got[grid.Code] = grid.Path
	}

	require.Equal(t, want, got)
}

// Judul kolom dipertahankan APA ADANYA, termasuk yang tidak konsisten.
//
// Section yang sama memakai "Share (%)" dengan spasi pada grid Claim Spreded dan "Share(%)"
// tanpa spasi pada grid Claim Spreaded. `D-13` menetapkan tampilan meniru Pega; merapikannya
// berarti pengguna mencari kolom yang tidak ada.
func TestJudulKolomYangTidakKonsistenDipertahankan(t *testing.T) {
	judul := func(code, key string) string {
		grid, known := inputacceptation.FindGrid(code)
		require.Truef(t, known, "grid %q tidak ada", code)
		for _, column := range grid.Columns {
			if column.Key == key {
				return column.Title
			}
		}
		t.Fatalf("kolom %s.%s tidak ada", code, key)
		return ""
	}

	require.Equal(t, "Share (%)",
		judul(inputacceptation.GridSpreadingClaim, "share_pct"))
	require.Equal(t, "Share(%)",
		judul(inputacceptation.GridSpreadAdjust, "share_pct"))
}

// KnownField mengenali seluruh isian yang digambar dan menolak yang tidak.
func TestKnownFieldMengikutiKatalog(t *testing.T) {
	for _, field := range inputacceptation.Fields() {
		require.Truef(t, inputacceptation.KnownField(field.Key),
			"isian %q ada di katalog tetapi tidak dikenal KnownField", field.Key)
	}
	require.False(t, inputacceptation.KnownField("isian_yang_tidak_pernah_ada"))
}

// Judul isian tidak boleh berupa label bawaan kontrol Pega.
//
// Section membawa "Dropdown", "Number", "Text Input", dan "Checkbox" sebagai label sel — itu
// nama KONTROL, bukan judul isian. Membawanya apa adanya akan menghasilkan layar berisi enam
// isian yang semuanya berjudul "Number".
func TestJudulIsianBukanLabelBawaanKontrol(t *testing.T) {
	bawaan := []string{"Dropdown", "Number", "Text Input", "Checkbox", "Date time"}

	for _, field := range inputacceptation.Fields() {
		for _, label := range bawaan {
			require.NotEqualf(t, label, strings.TrimSpace(field.Title),
				"isian %q memakai label bawaan kontrol %q sebagai judul", field.Key, label)
		}
	}
}

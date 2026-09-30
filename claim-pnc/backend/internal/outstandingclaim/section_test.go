package outstandingclaim_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/outstandingclaim"
)

func TestEveryFieldKeyIsUnique(t *testing.T) {
	// Kunci ganda tidak menghasilkan galat apa pun: yang belakangan menimpa yang duluan di
	// peta isian, sehingga satu isian menghilang dari layar tanpa satu pun tanda.
	seen := map[string]string{}
	for _, field := range outstandingclaim.Fields() {
		previous, clash := seen[field.Key]
		require.Falsef(t, clash,
			"kunci %q dipakai dua isian: %q dan %q", field.Key, previous, field.Title)
		seen[field.Key] = field.Title
	}
}

func TestEveryReadableFieldHasAPath(t *testing.T) {
	// Isian yang tidak terhalang WAJIB punya jalur. Tanpa jalur ia tidak pernah terisi,
	// tetapi juga tidak menyatakan alasannya — sel kosong yang tidak dapat dijelaskan
	// siapa pun.
	for _, field := range outstandingclaim.Fields() {
		if field.Blocked {
			continue
		}
		require.NotEmptyf(t, field.Path,
			"isian %q (%s) tidak terhalang tetapi tidak punya jalur", field.Key, field.Title)
	}
}

func TestBlockedFieldHasNoPath(t *testing.T) {
	// Kebalikannya: isian terhalang tidak boleh punya jalur. Jalur yang menganggur pada
	// isian terhalang adalah tebakan yang menunggu dipakai — dan di blok Treaty
	// Information, menebak berarti menampilkan pita share reasuransi milik treaty lain.
	for _, field := range outstandingclaim.Fields() {
		if !field.Blocked {
			continue
		}
		require.Emptyf(t, field.Path,
			"isian %q terhalang tetapi punya jalur %q", field.Key, field.Path)
	}
}

func TestEveryGridIsReachableFromAGroup(t *testing.T) {
	// Grid yang tidak disebut kelompok mana pun tidak pernah digambar. Ia bukan galat
	// melainkan kode mati yang kelak dikira siap dipakai.
	listed := map[string]bool{}
	for _, grid := range outstandingclaim.GridList() {
		listed[grid.Code] = true
	}

	for _, code := range []string{
		outstandingclaim.GridInterest,
		outstandingclaim.GridInterestTotal,
		outstandingclaim.GridClaimAmount,
		outstandingclaim.GridSpreadingRisk,
		outstandingclaim.GridEstimation,
		outstandingclaim.GridEstimationTotal,
		outstandingclaim.GridSpreadingClaim,
		outstandingclaim.GridSpreadingBreak,
		outstandingclaim.GridAttachment,
		outstandingclaim.GridSuggestion,
	} {
		require.Truef(t, listed[code], "grid %q tidak disebut kelompok mana pun", code)
	}
}

func TestEveryGroupGridExists(t *testing.T) {
	// Kebalikannya: kelompok tidak boleh menyebut grid yang tidak ada. Bila ia menyebutnya,
	// layar menggambar tabel tanpa kolom — bukan galat, hanya kotak kosong.
	for _, group := range outstandingclaim.Groups() {
		for _, code := range group.Grids {
			_, found := outstandingclaim.FindGrid(code)
			require.Truef(t, found,
				"kelompok %q menyebut grid %q yang tidak ada", group.Code, code)
		}
	}
}

func TestEveryReadableGridHasPathAndColumns(t *testing.T) {
	for _, grid := range outstandingclaim.GridList() {
		if grid.Blocked {
			continue
		}
		require.NotEmptyf(t, grid.Path, "grid %q tidak punya jalur senarai", grid.Code)
		require.NotEmptyf(t, grid.Columns, "grid %q tidak punya kolom", grid.Code)

		for _, column := range grid.Columns {
			require.NotEmptyf(t, column.Path,
				"kolom %q pada grid %q tidak punya jalur", column.Key, grid.Code)
		}
	}
}

func TestBlockedGridNamesItsOwner(t *testing.T) {
	// Penghalang tanpa alamat tidak pernah hilang — pelajaran yang sudah tercatat di `D-36`.
	for _, grid := range outstandingclaim.GridList() {
		if !grid.Blocked {
			continue
		}
		require.NotEmptyf(t, grid.BlockedReason, "grid %q terhalang tanpa alasan", grid.Code)
		require.NotEmptyf(t, grid.BlockedOwner, "grid %q terhalang tanpa pemilik", grid.Code)
	}
}

func TestTreatyMasterFieldsAreAllBlocked(t *testing.T) {
	// Kedelapan isian `.TreatyInMaster.*` terhalang bersama, karena penghalangnya satu dan
	// sama: tidak ada rule pemuat halaman `ASM-FW-GISFW-Int-TREATY_IN` di export.
	//
	// Uji ini menjaga agar tidak ada satu pun di antaranya yang kelak "diisi" dari kolom
	// yang kebetulan mirip. `ceding_name` paling rawan: ada `CEDINGCONAME` di tabel objek
	// kerja dan `$.QuotationData.CedingCoName` di dokumen klaim, dan keduanya BELUM tentu
	// treaty yang sama.
	blocked := map[string]bool{
		"ri_type": true, "ceding_name": true, "sob_name": true, "bordeaux": true,
		"bordereaux_note": true, "accounting_mode": true, "teritorial_scope": true,
		"asm_share": true,
	}

	for _, field := range outstandingclaim.Fields() {
		if blocked[field.Key] {
			require.Truef(t, field.Blocked,
				"isian %q berasal dari TreatyInMaster dan tidak boleh diisi tanpa rule "+
					"pemuatnya", field.Key)
		}
	}
}

func TestNewDetailDropsUnknownKeys(t *testing.T) {
	// Isian yang tidak pernah digambar tetapi ikut dikirim adalah data yang keluar dari
	// server tanpa satu pun alasan — dan di layar ini isinya memuat nama tertanggung dan
	// nilai klaim.
	detail := outstandingclaim.NewDetail("CLMP-1", "KEY", "New", "OP",
		map[string]string{
			"insured_name":              "PT Contoh",
			"isian_yang_tidak_digambar": "rahasia",
		},
		map[string][]outstandingclaim.GridRow{
			outstandingclaim.GridInterest: {{"object_name": "Gudang"}},
			"grid_yang_tidak_ada":         {{"apa_saja": "x"}},
		},
	)

	require.Equal(t, "PT Contoh", detail.Get("insured_name"))
	require.NotContains(t, detail.Values, "isian_yang_tidak_digambar")
	require.Len(t, detail.Rows(outstandingclaim.GridInterest), 1)
	require.NotContains(t, detail.Grids, "grid_yang_tidak_ada")
}

func TestNewDetailKeepsBlockedFieldsOut(t *testing.T) {
	// Isian terhalang dikenal section.go, sehingga KnownField menerimanya. Yang tidak boleh
	// terjadi adalah ia terisi — dan satu-satunya jalan ke sana adalah pengisi seam yang
	// memetiknya dari jalur yang ditebak. readDocument melewatinya; uji ini menjaga agar
	// kelak tidak ada pengisi lain yang mengisinya tanpa sengaja.
	require.True(t, outstandingclaim.KnownField("ceding_name"))

	for _, field := range outstandingclaim.Fields() {
		if field.Key == "ceding_name" {
			require.True(t, field.Blocked)
		}
	}
}

func TestQueryRejectsEmptyClaimID(t *testing.T) {
	_, err := outstandingclaim.NewQuery("   ", outstandingclaim.Caller{Login: "ADMIN"})

	var validation *outstandingclaim.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Len(t, validation.Violations, 1)
	require.Equal(t, outstandingclaim.FieldClaimID, validation.Violations[0].Field)
}

func TestQueryRejectsUnknownCaller(t *testing.T) {
	// Identitas diwajibkan meski tidak menyaring apa pun: setiap pembukaan dicatat, dan
	// catatan tanpa pelaku tidak menjelaskan apa pun saat ditelusuri kemudian.
	_, err := outstandingclaim.NewQuery("CLMP-70", outstandingclaim.Caller{Login: "  "})
	require.ErrorIs(t, err, outstandingclaim.ErrCallerUnknown)
}

func TestQueryTrimsClaimID(t *testing.T) {
	// Nomor klaim datang dari alamat yang diketik atau disalin orang, dan salinan sering
	// membawa spasi di ujungnya.
	query, err := outstandingclaim.NewQuery("  CLMP-70 ", outstandingclaim.Caller{Login: "A"})
	require.NoError(t, err)
	require.Equal(t, "CLMP-70", query.ClaimID)
}

func TestGridListOrderIsStable(t *testing.T) {
	// Urutan grid diambil dari Groups, bukan dari peta — urutan peta di Go tidak ditetapkan,
	// dan grid yang berpindah tempat setiap permintaan adalah layar yang tidak dapat
	// dipercaya.
	first := codesOf(outstandingclaim.GridList())
	for range 20 {
		require.Equal(t, first, codesOf(outstandingclaim.GridList()))
	}
}

func TestGroupsReturnsACopy(t *testing.T) {
	// Pemanggil yang menulisi hasilnya tidak boleh dapat mengubah bentuk layar bagi seluruh
	// permintaan berikutnya.
	groups := outstandingclaim.Groups()
	require.NotEmpty(t, groups[0].Fields)
	groups[0].Fields[0].Title = "DIUBAH"

	require.NotEqual(t, "DIUBAH", outstandingclaim.Groups()[0].Fields[0].Title)
}

func TestFieldTitlesFollowPegaEvenWhenOdd(t *testing.T) {
	// `D-13` menetapkan tampilan meniru Pega, termasuk judul yang salah ketik dan yang
	// huruf besarnya tidak konsisten. Uji ini menjaga agar tidak ada yang "merapikannya"
	// tanpa keputusan — merapikannya membuat pembandingan berdampingan gagal pada kolom
	// yang sebenarnya benar.
	titles := map[string]string{}
	for _, field := range outstandingclaim.Fields() {
		titles[field.Key] = field.Title
	}

	require.Equal(t, "TERITORIAL SCOPE", titles["teritorial_scope"])
	require.Equal(t, "StartDateTreaty", titles["start_date_treaty"])
	require.Equal(t, ".TypeDeductible", titles["type_deductible"])

	// Judul yang SAMA dipakai dua isian berbeda di blok Treaty Information, dan itu
	// tertulis begitu di Pega.
	require.Equal(t, titles["class_of_business"], titles["treaty_group_name"])

	grid, found := outstandingclaim.FindGrid(outstandingclaim.GridEstimationTotal)
	require.True(t, found)
	require.Equal(t, "Geoss Estimate Treaty (100%)", grid.Columns[1].Title)
	require.Equal(t, "Esstimation ASM", grid.Columns[2].Title)
}

func TestNoFieldKeyLooksLikeAPegaProperty(t *testing.T) {
	// Kunci kontrak tidak boleh menyalin nama properti Pega. Nama seperti `CARI10` tidak
	// menyatakan apa pun, dan membawanya ke kontrak membuat salah artinya ikut terbawa
	// (`D-19`).
	for _, field := range outstandingclaim.Fields() {
		require.Falsef(t, strings.HasPrefix(field.Key, "."),
			"kunci %q menyalin nama properti Pega", field.Key)
		require.Falsef(t, strings.Contains(strings.ToUpper(field.Key), "CARI"),
			"kunci %q menyalin alias CARI warisan", field.Key)
	}
}

func codesOf(grids []outstandingclaim.Grid) []string {
	result := make([]string, 0, len(grids))
	for _, grid := range grids {
		result = append(result, grid.Code)
	}
	return result
}

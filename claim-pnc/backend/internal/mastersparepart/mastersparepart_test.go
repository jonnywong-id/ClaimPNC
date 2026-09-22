package mastersparepart_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/mastersparepart"
)

// validInput adalah isian terkecil yang lolos pemeriksaan.
//
// Hanya keempat isian wajib yang terisi — nomor, nama, kode, dan harga. Sengaja begitu:
// setiap uji di bawah mengubah SATU hal dari dasar yang terbukti lolos, sehingga kegagalan
// selalu menunjuk perubahan itu dan bukan sesuatu yang kebetulan ikut terbawa.
func validInput() mastersparepart.Input {
	return mastersparepart.Input{
		Number:       "1R-0716",
		Name:         "FILTER OLI MESIN",
		Code:         "FLT-ENG-001",
		SellingPrice: "1250000",
	}
}

// violationFields mengembalikan nama isian yang dilaporkan sebuah galat pemeriksaan.
func violationFields(t *testing.T, err error) []string {
	t.Helper()

	var failure *mastersparepart.ValidationError
	require.ErrorAs(t, err, &failure)

	field := make([]string, 0, len(failure.Violation))
	for _, one := range failure.Violation {
		field = append(field, one.Field)
	}
	return field
}

func TestCheckAcceptsMinimumInput(t *testing.T) {
	require.NoError(t, validInput().Clean().Check())
}

// Keempat isian wajib berasal dari `pyRequired=true` pada
// `Section/BrowseMasterSparepartHEApproval-Section.xml`, dan HANYA keempat itu.
//
// Uji ini menjaga jumlahnya juga, bukan hanya daftarnya: isian kelima yang diam-diam
// diwajibkan akan menolak baris yang di layar lama tersimpan tanpa keluhan.
func TestCheckRequiresExactlyFourFields(t *testing.T) {
	err := mastersparepart.Input{}.Clean().Check()
	require.ElementsMatch(t,
		[]string{"nomor_sparepart", "nama_sparepart", "kode_sparepart", "harga_jual"},
		violationFields(t, err))
}

// Seluruh pelanggaran dilaporkan sekaligus, bukan berhenti pada yang pertama (P-5).
//
// `Activity/ValidateMasterSparepart` menyusun tiga pesan berdampingan — `Local.errmsg`,
// `errmsg1`, `errmsg2` — lalu menampilkan ketiganya bersamaan.
func TestCheckReportsEveryViolationAtOnce(t *testing.T) {
	input := validInput()
	input.Name = ""
	input.SellingPrice = "bukan angka"
	input.Weight = "juga bukan"

	field := violationFields(t, input.Clean().Check())
	require.ElementsMatch(t, []string{"nama_sparepart", "harga_jual", "berat"}, field)
}

func TestCleanTrimsEveryField(t *testing.T) {
	input := mastersparepart.Input{
		Number:         "  1R-0716  ",
		Name:           "  FILTER  ",
		Code:           "  FLT  ",
		SellingPrice:   "  1250000  ",
		CategoryID:     "  KAT01  ",
		TypeID:         "  TIP01  ",
		Weight:         "  850  ",
		Length:         "  12  ",
		Width:          "  13  ",
		Height:         "  14  ",
		MinStock:       "  5  ",
		MaxStock:       "  40  ",
		OrderQuantity:  "  10  ",
		ProductionDate: "  2025-11-04  ",
		Substitute:     "  FILTER ALT  ",
		Kind:           "  ORIGINAL  ",
		Unit:           "  PCS  ",
		ActiveStatus:   "  1  ",
		PartStatus:     "  READY  ",
	}

	clean := input.Clean()
	require.Equal(t, "1R-0716", clean.Number)
	require.Equal(t, "FILTER", clean.Name)
	require.Equal(t, "FLT", clean.Code)
	require.Equal(t, "1250000", clean.SellingPrice)
	require.Equal(t, "KAT01", clean.CategoryID)
	require.Equal(t, "TIP01", clean.TypeID)
	require.Equal(t, "850", clean.Weight)
	require.Equal(t, "12", clean.Length)
	require.Equal(t, "13", clean.Width)
	require.Equal(t, "14", clean.Height)
	require.Equal(t, "5", clean.MinStock)
	require.Equal(t, "40", clean.MaxStock)
	require.Equal(t, "10", clean.OrderQuantity)
	require.Equal(t, "2025-11-04", clean.ProductionDate)
	require.Equal(t, "FILTER ALT", clean.Substitute)
	require.Equal(t, "ORIGINAL", clean.Kind)
	require.Equal(t, "PCS", clean.Unit)
	require.Equal(t, "1", clean.ActiveStatus)
	require.Equal(t, "READY", clean.PartStatus)
}

// Ketiga kunci alami TIDAK di-UPPERCASE oleh Clean.
//
// Rule validasi Pega membandingkan `upper(...)`, tetapi yang di-uppercase di sana adalah
// PEMBANDINGNYA — bukan nilai yang disimpan. Memaksa huruf besar akan mengubah tampilan
// setiap baris yang disunting.
func TestCleanKeepsLetterCase(t *testing.T) {
	input := mastersparepart.Input{Name: "Filter Oli Mesin", Number: "1r-0716", Code: "flt-001"}

	clean := input.Clean()
	require.Equal(t, "Filter Oli Mesin", clean.Name)
	require.Equal(t, "1r-0716", clean.Number)
	require.Equal(t, "flt-001", clean.Code)
}

func TestCheckPrice(t *testing.T) {
	for name, tc := range map[string]struct {
		price    string
		rejected bool
	}{
		"bulat":               {"1250000", false},
		"desimal titik":       {"1250000.50", false},
		"desimal koma":        {"1250000,50", false},
		"nol":                 {"0", false},
		"kosong":              {"", true},
		"huruf":               {"seribu", true},
		"angka lalu huruf":    {"1250abc", true},
		"negatif":             {"-1", true},
		"terlalu besar":       {"100000000001", true},
		"tepat di batas atas": {"100000000000", false},
		"NaN":                 {"NaN", true},
		"Inf":                 {"Inf", true},
		"dua koma":            {"1,250,000", true},
	} {
		t.Run(name, func(t *testing.T) {
			input := validInput()
			input.SellingPrice = tc.price

			err := input.Clean().Check()
			if !tc.rejected {
				require.NoError(t, err)
				return
			}
			require.Contains(t, violationFields(t, err), "harga_jual")
		})
	}
}

// Ketujuh angka lain BOLEH kosong, dan harus berupa angka bila diisi.
func TestCheckOptionalNumbers(t *testing.T) {
	field := map[string]func(*mastersparepart.Input, string){
		"berat":             func(i *mastersparepart.Input, v string) { i.Weight = v },
		"panjang":           func(i *mastersparepart.Input, v string) { i.Length = v },
		"lebar":             func(i *mastersparepart.Input, v string) { i.Width = v },
		"tinggi":            func(i *mastersparepart.Input, v string) { i.Height = v },
		"stock_minimal":     func(i *mastersparepart.Input, v string) { i.MinStock = v },
		"stock_maximal":     func(i *mastersparepart.Input, v string) { i.MaxStock = v },
		"kuantitas_pesanan": func(i *mastersparepart.Input, v string) { i.OrderQuantity = v },
	}

	for name, set := range field {
		t.Run(name+" kosong diterima", func(t *testing.T) {
			input := validInput()
			set(&input, "")
			require.NoError(t, input.Clean().Check())
		})

		t.Run(name+" bukan angka ditolak", func(t *testing.T) {
			input := validInput()
			set(&input, "sepuluh")
			require.Contains(t, violationFields(t, input.Clean().Check()), name)
		})

		t.Run(name+" negatif ditolak", func(t *testing.T) {
			input := validInput()
			set(&input, "-1")
			require.Contains(t, violationFields(t, input.Clean().Check()), name)
		})
	}
}

// Stok minimal tidak boleh melampaui stok maksimal — SELISIH YANG DIRENCANAKAN terhadap
// sistem lama, yang tidak memeriksanya sama sekali.
func TestCheckStockRange(t *testing.T) {
	for name, tc := range map[string]struct {
		min, max string
		rejected bool
	}{
		"minimal di bawah maksimal": {"5", "40", false},
		"sama besar":                {"5", "5", false},
		"minimal di atas maksimal":  {"41", "40", true},
		"maksimal kosong":           {"41", "", false},
		"minimal kosong":            {"", "40", false},
		"keduanya kosong":           {"", "", false},
		// Salah satunya bukan angka: pesannya sudah dilaporkan checkNumbers, dan
		// mengulanginya di sini hanya akan membingungkan.
		"maksimal bukan angka": {"41", "empat puluh", true},
	} {
		t.Run(name, func(t *testing.T) {
			input := validInput()
			input.MinStock = tc.min
			input.MaxStock = tc.max

			err := input.Clean().Check()
			if !tc.rejected {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			if tc.max == "empat puluh" {
				// Yang dilaporkan adalah kolom yang bukan angkanya, BUKAN rentangnya.
				require.Contains(t, violationFields(t, err), "stock_maximal")
				require.NotContains(t, violationFields(t, err), "stock_minimal")
				return
			}
			require.Contains(t, violationFields(t, err), "stock_minimal")
		})
	}
}

// Part Substitusi TIDAK diperiksa keberadaannya.
//
// `Activity/ValidasiSparepart` memeriksanya, tetapi ia dipanggil dari layar LAIN — bukan
// dari jalur simpan Master Sparepart. Memindahkannya ke sini akan menolak baris yang hari
// ini tersimpan tanpa keluhan.
func TestCheckDoesNotVerifySubstitute(t *testing.T) {
	input := validInput()
	input.Substitute = "SPAREPART YANG TIDAK PERNAH ADA"

	require.NoError(t, input.Clean().Check())
}

func TestCheckLength(t *testing.T) {
	for name, tc := range map[string]struct {
		field string
		max   int
		set   func(*mastersparepart.Input, string)
	}{
		"nama_sparepart": {"nama_sparepart", mastersparepart.MaxNameLength,
			func(i *mastersparepart.Input, v string) { i.Name = v }},
		"nomor_sparepart": {"nomor_sparepart", mastersparepart.MaxNumberLength,
			func(i *mastersparepart.Input, v string) { i.Number = v }},
		"kode_sparepart": {"kode_sparepart", mastersparepart.MaxCodeLength,
			func(i *mastersparepart.Input, v string) { i.Code = v }},
		"part_substitusi": {"part_substitusi", mastersparepart.MaxNameLength,
			func(i *mastersparepart.Input, v string) { i.Substitute = v }},
		"tanggal_produksi": {"tanggal_produksi", mastersparepart.MaxDateLength,
			func(i *mastersparepart.Input, v string) { i.ProductionDate = v }},
		"satuan": {"satuan", mastersparepart.MaxMarkLength,
			func(i *mastersparepart.Input, v string) { i.Unit = v }},
	} {
		t.Run(name, func(t *testing.T) {
			input := validInput()
			tc.set(&input, strings.Repeat("x", tc.max))
			require.NoError(t, input.Clean().Check())

			tc.set(&input, strings.Repeat("x", tc.max+1))
			require.Contains(t, violationFields(t, input.Clean().Check()), tc.field)
		})
	}
}

// Batas panjang di sini DIULANG di SparepartForm.tsx. Uji ini tidak dapat membaca berkas
// TypeScript-nya; yang dijaganya adalah angkanya tidak berubah tanpa disadari, sehingga
// perubahan di satu tempat memaksa perubahan di sini juga — dan review melihat keduanya.
func TestLengthLimitsAreTheOnesTheFormRepeats(t *testing.T) {
	require.Equal(t, 100, mastersparepart.MaxNameLength)
	require.Equal(t, 50, mastersparepart.MaxNumberLength)
	require.Equal(t, 50, mastersparepart.MaxCodeLength)
	require.Equal(t, 20, mastersparepart.MaxNumericText)
	require.Equal(t, 30, mastersparepart.MaxDateLength)
	require.Equal(t, 30, mastersparepart.MaxMarkLength)
}

func TestApprovalStatus(t *testing.T) {
	require.Equal(t, "Approve", mastersparepart.StatusApproved.Label())
	require.Equal(t, "Reject", mastersparepart.StatusRejected.Label())
	require.Equal(t, "Waiting Approval", mastersparepart.StatusPending.Label())
	require.Empty(t, mastersparepart.ApprovalStatus("9").Label())

	require.True(t, mastersparepart.StatusApproved.Known())
	require.True(t, mastersparepart.StatusRejected.Known())
	require.True(t, mastersparepart.StatusPending.Known())
	require.False(t, mastersparepart.ApprovalStatus("").Known())
	require.False(t, mastersparepart.ApprovalStatus("3").Known())
}

// Bentuk ID meniru `Database/PEGA_M_SPAREPART_HE.prc:21` — kode situs ditambah nomor urut
// SEPULUH digit bertambal nol.
func TestComposeID(t *testing.T) {
	require.Equal(t, "SP0000000001", mastersparepart.ComposeID("SP", 1, 10))
	require.Equal(t, "SP0000123456", mastersparepart.ComposeID("SP", 123456, 10))
	require.Equal(t, "SP1234567890", mastersparepart.ComposeID("SP", 1234567890, 10))

	// Kode situs yang berspasi dipangkas, supaya kunci tidak memuat spasi di tengah.
	require.Equal(t, "SP0000000001", mastersparepart.ComposeID("  SP  ", 1, 10))

	// Nomor urut yang LEBIH PANJANG dari lebarnya dibiarkan tumbuh, tidak dipotong. LPAD
	// Oracle memotongnya dari kanan dan menghasilkan kunci yang bertabrakan diam-diam.
	require.Equal(t, "SP12345678901", mastersparepart.ComposeID("SP", 12345678901, 10))
}

func TestParseNumber(t *testing.T) {
	for _, accepted := range []string{"0", "1250000", "1250000.5", "1250000,5", "-1"} {
		value, err := mastersparepart.ParseNumber(accepted)
		require.NoError(t, err, accepted)
		require.False(t, value != value, "hasilnya tidak boleh NaN")
	}

	for _, rejected := range []string{"", "seribu", "1250abc", "NaN", "Inf", "1,250,000"} {
		_, err := mastersparepart.ParseNumber(rejected)
		require.Error(t, err, rejected)
	}
}

func TestOneViolation(t *testing.T) {
	err := mastersparepart.OneViolation("nama_sparepart", "Sudah dipakai.")
	require.ElementsMatch(t, []string{"nama_sparepart"}, violationFields(t, err))
	require.Contains(t, err.Error(), "nama_sparepart")
	require.Contains(t, err.Error(), "Sudah dipakai.")
}

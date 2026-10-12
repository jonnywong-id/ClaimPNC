package masterpanel_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterpanel"
)

// validInput adalah isian yang lolos seluruh aturan.
//
// Setiap uji berangkat dari sini lalu merusak SATU hal, sehingga yang membuatnya ditolak
// selalu jelas — bukan hasil beberapa kesalahan yang menumpuk.
func validInput() masterpanel.Input {
	return masterpanel.Input{
		Name:                "Pintu Depan",
		RepairStatus:        "1",
		EditQuantityStatus:  "1",
		PremiumRepairStatus: "0",
		ShatterStatus:       "0",
		StickerStatus:       "1",
		SideStatus:          "1",
		SevereDamageStatus:  "0",
		ActiveStatus:        "1",
		ExclusionC:          "0",
		Location: []masterpanel.PanelLocation{
			{Name: "KIRI", Side: masterpanel.SideLeft},
		},
	}
}

// violationFields mengumpulkan nama isian dari sebuah galat validasi.
func violationFields(t *testing.T, err error) []string {
	t.Helper()

	var validationError *masterpanel.ValidationError
	require.ErrorAs(t, err, &validationError)

	field := make([]string, 0, len(validationError.Violation))
	for _, one := range validationError.Violation {
		field = append(field, one.Field)
	}
	return field
}

func TestValidInputPasses(t *testing.T) {
	require.NoError(t, validInput().Clean().Check())
}

// Kesepuluh isian induk WAJIB, dan itu bukan tambahan: kesepuluhnya bertanda
// `pyRequired=true` pada Section/BrowsePanelHEApproval-Section.xml.
func TestEveryParentFieldIsRequired(t *testing.T) {
	blank := map[string]func(*masterpanel.Input){
		"nama_panel":            func(i *masterpanel.Input) { i.Name = "" },
		"status_repair":         func(i *masterpanel.Input) { i.RepairStatus = "" },
		"status_edit_quantity":  func(i *masterpanel.Input) { i.EditQuantityStatus = "" },
		"status_premium_repair": func(i *masterpanel.Input) { i.PremiumRepairStatus = "" },
		"status_pecah":          func(i *masterpanel.Input) { i.ShatterStatus = "" },
		"status_sticker":        func(i *masterpanel.Input) { i.StickerStatus = "" },
		"status_sisi":           func(i *masterpanel.Input) { i.SideStatus = "" },
		"status_rusak_parah":    func(i *masterpanel.Input) { i.SevereDamageStatus = "" },
		"status_aktif":          func(i *masterpanel.Input) { i.ActiveStatus = "" },
		"exclusion_c":           func(i *masterpanel.Input) { i.ExclusionC = "" },
	}

	for field, blankOut := range blank {
		t.Run(field, func(t *testing.T) {
			input := validInput()
			blankOut(&input)

			err := input.Clean().Check()
			require.Error(t, err)
			require.Contains(t, violationFields(t, err), field)
		})
	}
}

// Seluruh pelanggaran dikirim sekaligus, bukan yang pertama saja (P-5).
func TestAllViolationsReturnedAtOnce(t *testing.T) {
	err := masterpanel.Input{}.Clean().Check()
	require.Error(t, err)

	// Sepuluh isian induk kosong. Daftar lokasi kosong TIDAK menambah pelanggaran —
	// panel tanpa lokasi adalah keadaan yang sah.
	require.Len(t, violationFields(t, err), 10)
}

// Panel TANPA lokasi sama sekali adalah keadaan yang SAH.
//
// Layar lama tidak mewajibkan satu pun baris lokasi, dan GetLokasiSisiPanel yang
// mengembalikan nol baris tidak diperlakukan sebagai galat di mana pun.
func TestPanelWithoutLocationIsValid(t *testing.T) {
	input := validInput()
	input.Location = nil

	require.NoError(t, input.Clean().Check())
}

// Baris lokasi yang kosong SELURUHNYA dibuang diam-diam, bukan ditolak.
//
// Layar yang menambah baris lalu membiarkannya kosong adalah hal yang wajar; menolaknya
// memaksa pengguna menghapus baris yang tidak pernah ia isi.
func TestBlankLocationRowIsDropped(t *testing.T) {
	input := validInput()
	input.Location = append(input.Location, masterpanel.PanelLocation{Name: "  ", Side: " "})

	clean := input.Clean()
	require.NoError(t, clean.Check())
	require.Len(t, clean.Location, 1)
	require.Equal(t, "KIRI", clean.Location[0].Name)
}

// Baris lokasi yang sisinya terisi tetapi namanya kosong DITOLAK — ia bukan baris kosong
// melainkan baris setengah jadi.
func TestLocationWithoutNameRejected(t *testing.T) {
	input := validInput()
	input.Location = []masterpanel.PanelLocation{{Name: "", Side: masterpanel.SideLeft}}

	err := input.Clean().Check()
	require.Error(t, err)
	require.Contains(t, violationFields(t, err), "lokasi.0.lokasi_panel")
}

// Sisi di luar tiga sandi yang dikenal ditolak.
//
// SELISIH YANG DIRENCANAKAN terhadap Pega: `Activity/GetSisiPanel-Act.xml` membaca apa pun
// yang bukan "-" dan bukan "1" sebagai "KANAN", sehingga nilai rusak tampil sebagai nilai
// yang sah.
func TestUnknownSideRejected(t *testing.T) {
	input := validInput()
	input.Location = []masterpanel.PanelLocation{{Name: "KIRI", Side: masterpanel.Side("9")}}

	err := input.Clean().Check()
	require.Error(t, err)
	require.Contains(t, violationFields(t, err), "lokasi.0.sisi_panel")
}

func TestEveryKnownSideAccepted(t *testing.T) {
	for _, side := range []masterpanel.Side{
		masterpanel.SideNone,
		masterpanel.SideLeft,
		masterpanel.SideRight,
	} {
		t.Run(string(side), func(t *testing.T) {
			input := validInput()
			input.Location = []masterpanel.PanelLocation{{Name: "DEPAN", Side: side}}

			require.NoError(t, input.Clean().Check())
			require.True(t, side.Known())
			require.NotEmpty(t, side.Label())
		})
	}
}

// Lokasi yang sama pada sisi yang sama tidak boleh muncul dua kali.
//
// DITAMBAHKAN terhadap sistem lama, yang tidak memeriksanya — dan dua baris kembar membuat
// GetDataSisiPanel, yang membaca satu nilai saja, mengembalikan baris yang mana pun lebih
// dulu ditemukan.
func TestDuplicateLocationRejected(t *testing.T) {
	input := validInput()
	input.Location = []masterpanel.PanelLocation{
		{Name: "KIRI", Side: masterpanel.SideLeft},
		{Name: "KIRI", Side: masterpanel.SideLeft},
	}

	err := input.Clean().Check()
	require.Error(t, err)
	require.Contains(t, violationFields(t, err), "lokasi.1.lokasi_panel")
}

// Lokasi yang sama pada sisi yang BERBEDA tetap diterima — "KIRI" sisi kiri dan "KIRI"
// sisi kanan adalah dua baris yang berbeda.
func TestSameLocationDifferentSideAccepted(t *testing.T) {
	input := validInput()
	input.Location = []masterpanel.PanelLocation{
		{Name: "KIRI", Side: masterpanel.SideLeft},
		{Name: "KIRI", Side: masterpanel.SideRight},
	}

	require.NoError(t, input.Clean().Check())
}

// Nama lokasi TIDAK dibatasi pada kelima pilihan LocationOptions.
//
// Baris lama dapat memuat lokasi lain, dan menolaknya berarti baris yang hari ini sah
// tidak dapat disimpan ulang.
func TestLocationOutsideTheFiveOptionsAccepted(t *testing.T) {
	input := validInput()
	input.Location = []masterpanel.PanelLocation{{Name: "ATAP", Side: masterpanel.SideNone}}

	require.NoError(t, input.Clean().Check())
}

func TestLocationRowLimit(t *testing.T) {
	input := validInput()
	input.Location = nil
	for i := 0; i <= masterpanel.MaxLocationRows; i++ {
		input.Location = append(input.Location, masterpanel.PanelLocation{
			Name: "LOKASI" + strings.Repeat("X", i%5),
			Side: masterpanel.SideNone,
		})
	}

	err := input.Clean().Check()
	require.Error(t, err)
	require.Contains(t, violationFields(t, err), "lokasi")
}

// Clean memangkas spasi DAN menaikkan nama lokasi menjadi huruf besar.
//
// Yang kedua bukan kerapian: GetDataSisiPanel mencocokkan kolom itu dengan `=` biasa tanpa
// UPPER di kedua sisi, sehingga satu baris bertuliskan "Kiri" tidak akan pernah ditemukan
// modul Grouping Sparepart.
func TestCleanTrimsAndUppercasesLocation(t *testing.T) {
	input := masterpanel.Input{
		Name:                "  Pintu Depan  ",
		RepairStatus:        " 1 ",
		EditQuantityStatus:  " 1 ",
		PremiumRepairStatus: " 0 ",
		ShatterStatus:       " 0 ",
		StickerStatus:       " 1 ",
		SideStatus:          " 1 ",
		SevereDamageStatus:  " 0 ",
		ActiveStatus:        " 1 ",
		ExclusionC:          " 0 ",
		Location: []masterpanel.PanelLocation{
			{Name: "  kiri  ", Side: masterpanel.Side("  1  ")},
		},
	}

	clean := input.Clean()
	require.Equal(t, "Pintu Depan", clean.Name)
	require.Equal(t, "1", clean.RepairStatus)
	require.Equal(t, "0", clean.ExclusionC)
	require.Equal(t, "KIRI", clean.Location[0].Name)
	require.Equal(t, masterpanel.SideLeft, clean.Location[0].Side)
}

// Nama panel TIDAK dinaikkan menjadi huruf besar.
//
// Hanya nama LOKASI yang dinaikkan, dan alasannya ada pada uji di atas. Nama panel
// disimpan apa adanya — pencocokan gandanya memakai UPPER di kedua sisi, sehingga tidak
// ada yang memaksanya seragam.
func TestPanelNameKeepsItsCase(t *testing.T) {
	input := validInput()
	input.Name = "Pintu Depan"

	require.Equal(t, "Pintu Depan", input.Clean().Name)
}

func TestApprovalStatusKnown(t *testing.T) {
	for _, s := range []masterpanel.ApprovalStatus{
		masterpanel.StatusPending,
		masterpanel.StatusApproved,
		masterpanel.StatusRejected,
	} {
		require.True(t, s.Known(), "%q harus dikenal", s)
		require.NotEmpty(t, s.Label(), "%q harus punya label", s)
	}

	require.False(t, masterpanel.ApprovalStatus("3").Known())
	require.Empty(t, masterpanel.ApprovalStatus("3").Label())
}

// Sandinya tetap "0", "1", "2" seperti kolom APPROVAL, karena tabelnya masih dibaca
// sistem lama selama masa paralel (ADR-0004).
func TestApprovalStatusCodesMatchThePegaColumn(t *testing.T) {
	require.Equal(t, masterpanel.ApprovalStatus("0"), masterpanel.StatusPending)
	require.Equal(t, masterpanel.ApprovalStatus("1"), masterpanel.StatusApproved)
	require.Equal(t, masterpanel.ApprovalStatus("2"), masterpanel.StatusRejected)
}

// Sandi sisi mengikuti Activity/SetLokasiSisiPanel-Act.xml apa adanya.
func TestSideCodesMatchThePegaActivity(t *testing.T) {
	require.Equal(t, masterpanel.Side("-"), masterpanel.SideNone)
	require.Equal(t, masterpanel.Side("1"), masterpanel.SideLeft)
	require.Equal(t, masterpanel.Side("2"), masterpanel.SideRight)

	require.Equal(t, "-", masterpanel.SideNone.Label())
	require.Equal(t, "KIRI", masterpanel.SideLeft.Label())
	require.Equal(t, "KANAN", masterpanel.SideRight.Label())
}

// Kelima pilihan lokasi mengikuti urutan yang sama dengan Pega — itulah urutan yang
// dilihat petugas pada dropdown hari ini.
func TestLocationOptionsFollowThePegaOrder(t *testing.T) {
	require.Equal(t,
		[]string{"KIRI", "KANAN", "DEPAN", "BELAKANG", "LAIN-LAIN"},
		masterpanel.LocationOptions)
}

// Bentuk kunci meniru PEGA_M_PANEL_HE.prc:21 — kode situs ditambah ENAM digit.
func TestComposeID(t *testing.T) {
	require.Equal(t, "01000001", masterpanel.ComposeID("01", 1, 6))
	require.Equal(t, "01000042", masterpanel.ComposeID("01", 42, 6))
	require.Equal(t, "01999999", masterpanel.ComposeID("01", 999999, 6))
	require.Equal(t, "01000001", masterpanel.ComposeID("  01  ", 1, 6))
}

// Nomor urut yang lebih panjang dari lebar yang diminta TIDAK dipotong.
//
// LPAD Oracle memotongnya dari kanan, sehingga kuncinya bertabrakan diam-diam. Di sini ia
// dibiarkan tumbuh: kuncinya menjadi lebih panjang, dan itu terlihat.
func TestComposeIDDoesNotTruncate(t *testing.T) {
	require.Equal(t, "011000000", masterpanel.ComposeID("01", 1000000, 6))
}

func TestErrorsNameTheModule(t *testing.T) {
	for _, err := range []error{
		masterpanel.ErrNotFound,
		masterpanel.ErrNameTaken,
		masterpanel.ErrUnknownStatus,
	} {
		require.True(t, strings.HasPrefix(err.Error(), "masterpanel: "),
			"pesan %q harus menyebut modulnya", err.Error())
	}
}

func TestOneViolationIsAValidationError(t *testing.T) {
	err := masterpanel.OneViolation("nama_panel", "Nama tersebut telah digunakan.")

	var validationError *masterpanel.ValidationError
	require.True(t, errors.As(err, &validationError))
	require.Len(t, validationError.Violation, 1)
	require.Equal(t, "nama_panel", validationError.Violation[0].Field)
}

// Batas panjang di sini HARUS sama dengan yang ditulis di PanelForm.tsx.
//
// Keduanya asumsi yang disadari — DDL kedua tabel belum ada (R-08) — dan uji ini ada
// supaya duplikasinya tetap terlihat alih-alih menjadi dua angka yang diam-diam berbeda.
func TestLengthLimitsAreTheOnesTheFormRepeats(t *testing.T) {
	require.Equal(t, 100, masterpanel.MaxNameLength)
	require.Equal(t, 20, masterpanel.MaxCodeLength)
	require.Equal(t, 50, masterpanel.MaxLocationLength)
	require.Equal(t, 250, masterpanel.MaxReasonLength)
	require.Equal(t, 50, masterpanel.MaxLocationRows)
}

func TestTooLongValuesRejected(t *testing.T) {
	input := validInput()
	input.Name = strings.Repeat("A", masterpanel.MaxNameLength+1)

	err := input.Clean().Check()
	require.Error(t, err)
	require.Contains(t, violationFields(t, err), "nama_panel")
}

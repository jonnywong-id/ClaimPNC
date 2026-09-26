package mastergroupingsparepart_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/mastergroupingsparepart"
)

// validInput adalah isian yang lolos seluruh pemeriksaan.
//
// Setiap uji berangkat darinya lalu merusak SATU hal — dengan begitu yang diuji benar-benar
// hal itu, bukan kebetulan ada isian lain yang juga salah.
func validInput() mastergroupingsparepart.Input {
	return mastergroupingsparepart.Input{
		PartNumber:       "SP-1001",
		PanelID:          "PNL02",
		PanelName:        "KABIN",
		PanelSide:        "1",
		ChassisNumber:    "MHFXW1234K5678901",
		VehicleType:      "EXCAVATOR",
		GroupWithChassis: "",
		Note:             "Pemasangan awal unit.",
	}
}

func TestValidInputPasses(t *testing.T) {
	require.NoError(t, validInput().Clean().Check())
}

// Keempat isian wajib adalah persis anggota kunci alami.
//
// Bukan karena layar lama menandainya `pyRequired` — ia tidak menandai satu pun — melainkan
// karena kunci yang salah satu anggotanya kosong tidak dapat membedakan dua baris. Lihat
// Input.Check.
func TestRequiredFieldsAreExactlyTheNaturalKey(t *testing.T) {
	cases := map[string]func(*mastergroupingsparepart.Input){
		"nomor_sparepart": func(i *mastergroupingsparepart.Input) { i.PartNumber = "" },
		"nama_panel":      func(i *mastergroupingsparepart.Input) { i.PanelName = "" },
		"no_rangka":       func(i *mastergroupingsparepart.Input) { i.ChassisNumber = "" },
		"sisi":            func(i *mastergroupingsparepart.Input) { i.PanelSide = "" },
	}

	for field, breakIt := range cases {
		t.Run(field, func(t *testing.T) {
			input := validInput()
			breakIt(&input)

			err := input.Clean().Check()
			require.Error(t, err)
			require.Contains(t, violationFields(t, err), field)
		})
	}
}

// Ketiga isian yang BUKAN anggota kunci alami boleh kosong.
//
// Tidak satu pun rule di export memeriksanya, dan mewajibkannya akan menolak baris yang hari
// ini tersimpan tanpa keluhan.
func TestOptionalFieldsMayBeEmpty(t *testing.T) {
	input := validInput()
	input.PanelID = ""
	input.VehicleType = ""
	input.Note = ""

	require.NoError(t, input.Clean().Check())
}

// SELURUH pelanggaran dilaporkan sekaligus, bukan yang pertama saja.
//
// Kesetaraan perilaku (`P-5`): `Activity/UpdateGroupingSparepartHE_act` menyusun dua pesan
// galatnya berdampingan lalu menampilkan keduanya lewat `Page-Set-Messages`.
func TestAllViolationsReportedTogether(t *testing.T) {
	input := mastergroupingsparepart.Input{}

	err := input.Clean().Check()
	require.Error(t, err)
	require.Len(t, violationFields(t, err), 4,
		"keempat isian wajib yang kosong harus dilaporkan bersamaan")
}

// Spasi di kedua ujung dipangkas SEBELUM diperiksa, sehingga isian berisi spasi saja ditolak.
func TestWhitespaceOnlyIsTreatedAsEmpty(t *testing.T) {
	input := validInput()
	input.PanelName = "   "

	err := input.Clean().Check()
	require.Error(t, err)
	require.Contains(t, violationFields(t, err), "nama_panel")
}

// Clean memangkas, dan TIDAK meng-uppercase.
//
// Kunci alaminya memang dibandingkan tanpa memandang besar-kecil huruf, tetapi yang
// di-uppercase adalah PEMBANDINGNYA — bukan nilai yang disimpan. Memaksa huruf besar akan
// mengubah tampilan setiap baris yang disunting.
func TestCleanTrimsButNeverUppercases(t *testing.T) {
	input := mastergroupingsparepart.Input{
		PartNumber:    "  sp-1001  ",
		PanelName:     " Kabin ",
		ChassisNumber: " mhfxw1234k5678901 ",
		PanelSide:     " 1 ",
	}.Clean()

	require.Equal(t, "sp-1001", input.PartNumber)
	require.Equal(t, "Kabin", input.PanelName)
	require.Equal(t, "mhfxw1234k5678901", input.ChassisNumber)
	require.Equal(t, "1", input.PanelSide)
}

// Menggabungkan baris dengan nomor rangkanya sendiri ditolak.
//
// SELISIH YANG DIRENCANAKAN terhadap sistem lama, yang tidak memeriksanya; lihat
// Input.checkGroupTarget.
func TestGroupingWithOwnChassisIsRejected(t *testing.T) {
	input := validInput()
	input.GroupWithChassis = input.ChassisNumber

	err := input.Clean().Check()
	require.Error(t, err)
	require.Contains(t, violationFields(t, err), "grouping_dengan_no_rangka")
}

// Pemeriksaannya mengabaikan besar-kecil huruf, sepasang dengan pencarian grupnya sendiri.
func TestGroupingWithOwnChassisIgnoresCase(t *testing.T) {
	input := validInput()
	input.GroupWithChassis = strings.ToLower(input.ChassisNumber)

	require.Error(t, input.Clean().Check())
}

// Menggabungkan ke nomor rangka LAIN tetap diterima — itu memang gunanya isian ini.
func TestGroupingWithAnotherChassisIsAccepted(t *testing.T) {
	input := validInput()
	input.GroupWithChassis = "MHFZZ9876K1234567"

	require.NoError(t, input.Clean().Check())
}

func TestLengthLimitsAreEnforced(t *testing.T) {
	cases := []struct {
		field string
		limit int
		set   func(*mastergroupingsparepart.Input, string)
	}{
		{"nomor_sparepart", mastergroupingsparepart.MaxPartNumberLength,
			func(i *mastergroupingsparepart.Input, v string) { i.PartNumber = v }},
		{"nama_panel", mastergroupingsparepart.MaxPanelNameLength,
			func(i *mastergroupingsparepart.Input, v string) { i.PanelName = v }},
		{"no_rangka", mastergroupingsparepart.MaxChassisLength,
			func(i *mastergroupingsparepart.Input, v string) { i.ChassisNumber = v }},
		{"tipe_kendaraan", mastergroupingsparepart.MaxVehicleTypeLength,
			func(i *mastergroupingsparepart.Input, v string) { i.VehicleType = v }},
		{"catatan", mastergroupingsparepart.MaxNoteLength,
			func(i *mastergroupingsparepart.Input, v string) { i.Note = v }},
	}

	for _, one := range cases {
		t.Run(one.field, func(t *testing.T) {
			input := validInput()
			one.set(&input, strings.Repeat("x", one.limit+1))

			err := input.Clean().Check()
			require.Error(t, err)
			require.Contains(t, violationFields(t, err), one.field)

			exact := validInput()
			one.set(&exact, strings.Repeat("x", one.limit))
			require.NoError(t, exact.Clean().Check(), "panjang tepat di batas harus diterima")
		})
	}
}

// Batas panjang yang diulang di GroupingForm.tsx harus sama dengan yang di sini.
//
// Duplikasi itu disadari; uji ini yang menjaganya tetap terlihat. Bila salah satu berubah,
// KEDUA tempat harus ikut berubah.
func TestLengthLimitsAreTheOnesTheFormRepeats(t *testing.T) {
	require.Equal(t, 50, mastergroupingsparepart.MaxPartNumberLength)
	require.Equal(t, 100, mastergroupingsparepart.MaxPanelNameLength)
	require.Equal(t, 50, mastergroupingsparepart.MaxChassisLength)
	require.Equal(t, 30, mastergroupingsparepart.MaxSideLength)
	require.Equal(t, 100, mastergroupingsparepart.MaxVehicleTypeLength)
	require.Equal(t, 500, mastergroupingsparepart.MaxNoteLength)
}

func TestApprovalStatusLabels(t *testing.T) {
	require.Equal(t, "Waiting Approval", mastergroupingsparepart.StatusPending.Label())
	require.Equal(t, "Approve", mastergroupingsparepart.StatusApproved.Label())
	require.Equal(t, "Reject", mastergroupingsparepart.StatusRejected.Label())
	require.Empty(t, mastergroupingsparepart.ApprovalStatus("9").Label())
}

func TestApprovalStatusKnown(t *testing.T) {
	for _, status := range []mastergroupingsparepart.ApprovalStatus{"0", "1", "2"} {
		require.Truef(t, status.Known(), "status %q harus dikenal", status)
	}
	for _, status := range []mastergroupingsparepart.ApprovalStatus{"", "3", "01", "x"} {
		require.Falsef(t, status.Known(), "status %q tidak boleh dikenal", status)
	}
}

// Sandi sisi di luar ketiganya dikembalikan APA ADANYA.
//
// Sistem lama memaksanya menjadi "KANAN" — `@If(.NAME=="1","KIRI","KANAN")` menjawab KANAN
// untuk apa pun yang bukan "-" dan bukan "1" — sehingga nilai rusak tampil sebagai nilai yang
// sah. SELISIH YANG DIRENCANAKAN, dan perlakuan yang sama sudah diambil Master Panel.
func TestSideLabel(t *testing.T) {
	require.Equal(t, "-", mastergroupingsparepart.SideLabel("-"))
	require.Equal(t, "KIRI", mastergroupingsparepart.SideLabel("1"))
	require.Equal(t, "KANAN", mastergroupingsparepart.SideLabel("2"))

	require.Equal(t, "9", mastergroupingsparepart.SideLabel("9"),
		"sandi tak dikenal tidak boleh dipaksa menjadi KANAN")
	require.Equal(t, "", mastergroupingsparepart.SideLabel(""))
}

// Bentuk nomor grup meniru `UpdateGroupingSparepartHE_act` langkah 7: `"000" + nomor`.
//
// Ia perangkaian teks, bukan pengisian nol sampai lebar tertentu — sehingga lebarnya
// BERTAMBAH mulai nomor 10. Ditiru apa adanya; lihat ComposeGroupNumber.
func TestComposeGroupNumberFollowsThePegaShape(t *testing.T) {
	require.Equal(t, "0001", mastergroupingsparepart.ComposeGroupNumber(1))
	require.Equal(t, "0009", mastergroupingsparepart.ComposeGroupNumber(9))
	require.Equal(t, "00010", mastergroupingsparepart.ComposeGroupNumber(10),
		"lebarnya memang bertambah; itu bentuk yang sudah ada di basis data")
	require.Equal(t, "000123", mastergroupingsparepart.ComposeGroupNumber(123))
}

// ParseGroupNumber memaafkan KEDUA bentuk yang ditulis sistem lama.
//
// `GetNewNoGroup` menghasilkan bentuk berawalan nol, sedangkan `GetNoGroup` mengembalikannya
// lewat `TO_NUMBER` yang membuang awalannya — sehingga satu grup dapat tersimpan dalam dua
// bentuk teks.
func TestParseGroupNumberAcceptsBothShapes(t *testing.T) {
	for _, one := range []struct {
		text string
		want int64
	}{
		{"0001", 1},
		{"1", 1},
		{"00010", 10},
		{"10", 10},
		{"000123", 123},
		{"", 0},
		{"000", 0},
		{"  0007  ", 7},
	} {
		got, err := mastergroupingsparepart.ParseGroupNumber(one.text)
		require.NoErrorf(t, err, "%q harus dapat diurai", one.text)
		require.Equalf(t, one.want, got, "%q", one.text)
	}
}

// Nilai yang bukan angka menghasilkan GALAT, bukan nol diam-diam.
//
// Nol akan membuat penerbitan nomor berikutnya mengulang nomor yang sudah dipakai.
func TestParseGroupNumberRejectsNonNumeric(t *testing.T) {
	for _, text := range []string{"abc", "00A1", "1.5", "-1", "+1", "1 2"} {
		_, err := mastergroupingsparepart.ParseGroupNumber(text)
		require.Errorf(t, err, "%q tidak boleh diurai sebagai nomor grup", text)
	}
}

// ComposeGroupNumber dan ParseGroupNumber harus bolak-balik.
//
// Tanpa ini, nomor grup yang diterbitkan hari ini dapat tidak terbaca saat menerbitkan nomor
// berikutnya — dan dua grup akan berbagi satu nomor.
func TestGroupNumberRoundTrips(t *testing.T) {
	for _, number := range []int64{1, 9, 10, 99, 100, 1000, 999999} {
		text := mastergroupingsparepart.ComposeGroupNumber(number)
		got, err := mastergroupingsparepart.ParseGroupNumber(text)
		require.NoErrorf(t, err, "nomor grup %q hasil terbitan sendiri harus terbaca", text)
		require.Equal(t, number, got)
	}
}

func TestKeyOfReadsAllFourColumns(t *testing.T) {
	key := mastergroupingsparepart.KeyOf(mastergroupingsparepart.Grouping{
		PartNumber:    "SP-1001",
		PanelName:     "KABIN",
		ChassisNumber: "MHFXW1234K5678901",
		PanelSide:     "1",
		// Kolom di luar kunci alami sengaja diisi: bila salah satunya bocor ke dalam kunci,
		// uji ini yang menangkapnya.
		Note:        "catatan",
		VehicleType: "EXCAVATOR",
	})

	require.Equal(t, mastergroupingsparepart.NaturalKey{
		PartNumber:    "SP-1001",
		PanelName:     "KABIN",
		ChassisNumber: "MHFXW1234K5678901",
		PanelSide:     "1",
	}, key)
}

func TestOneViolationCarriesFieldAndMessage(t *testing.T) {
	err := mastergroupingsparepart.OneViolation("nomor_sparepart", "Pesan.")

	var violation *mastergroupingsparepart.ValidationError
	require.ErrorAs(t, err, &violation)
	require.Len(t, violation.Violation, 1)
	require.Equal(t, "nomor_sparepart", violation.Violation[0].Field)
	require.Equal(t, "Pesan.", violation.Violation[0].Message)
}

// violationFields mengambil nama isian yang dilaporkan sebuah galat validasi.
func violationFields(t *testing.T, err error) []string {
	t.Helper()

	var violation *mastergroupingsparepart.ValidationError
	require.ErrorAs(t, err, &violation)

	result := make([]string, 0, len(violation.Violation))
	for _, one := range violation.Violation {
		result = append(result, one.Field)
	}
	return result
}

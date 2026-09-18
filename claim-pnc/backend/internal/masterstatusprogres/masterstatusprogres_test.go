package masterstatusprogres_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterstatusprogres"
)

// Nama isian wajib. Ini aturan pertama yang menolak isian pengguna, dan pesannya
// menyebut apa yang harus ia lakukan — bukan hanya bahwa ada yang salah.
func TestNameIsRequired(t *testing.T) {
	input := masterstatusprogres.Input{Name: "   ", PositionCode: "002"}.Clean()

	err := input.Check()
	require.Error(t, err)

	var validationErr *masterstatusprogres.ValidationError
	require.ErrorAs(t, err, &validationErr)
	require.Len(t, validationErr.Violation, 1)
	require.Equal(t, "nama", validationErr.Violation[0].Field)
}

// Seluruh pelanggaran dikembalikan sekaligus, bukan yang pertama saja.
//
// Ini kesetaraan perilaku dengan Pega (P-5), bukan selera: `InputRegister_act`
// menampilkan semua pesan bersamaan, dan mengembalikan satu per satu akan membuat
// pengguna menebak isian mana lagi yang salah.
func TestAllViolationsReturnedAtOnce(t *testing.T) {
	input := masterstatusprogres.Input{Name: "", PositionCode: ""}.Clean()

	err := input.Check()
	var validationErr *masterstatusprogres.ValidationError
	require.ErrorAs(t, err, &validationErr)
	require.Len(t, validationErr.Violation, 2, "nama kosong DAN posisi kosong, keduanya dilaporkan")

	fields := map[string]bool{}
	for _, p := range validationErr.Violation {
		fields[p.Field] = true
	}
	require.True(t, fields["nama"])
	require.True(t, fields["kode_posisi"])
}

// Kode posisi di luar keempat yang dikenali ditolak.
//
// Di sistem lama kendali ini diberikan oleh dropdown. API dapat ditembak langsung tanpa
// melewati layar, sehingga kendalinya harus ada di server — kalau tidak, ia hilang.
func TestUnknownPositionCodeRejected(t *testing.T) {
	input := masterstatusprogres.Input{Name: "DOKUMEN DITERIMA", PositionCode: "999"}.Clean()

	err := input.Check()
	var validationErr *masterstatusprogres.ValidationError
	require.ErrorAs(t, err, &validationErr)
	require.Len(t, validationErr.Violation, 1)
	require.Equal(t, "kode_posisi", validationErr.Violation[0].Field)
}

func TestValidInputPasses(t *testing.T) {
	for _, position := range masterstatusprogres.ListPositions() {
		input := masterstatusprogres.Input{Name: "MENUNGGU DOKUMEN", PositionCode: position.Code}.Clean()
		require.NoError(t, input.Check(), "posisi %q seharusnya sah", position.Code)
	}
}

func TestTooLongNameRejected(t *testing.T) {
	tooLong := strings.Repeat("A", masterstatusprogres.MaxNameLength+1)
	input := masterstatusprogres.Input{Name: tooLong, PositionCode: "002"}.Clean()

	err := input.Check()
	var validationErr *masterstatusprogres.ValidationError
	require.ErrorAs(t, err, &validationErr)
	require.Equal(t, "nama", validationErr.Violation[0].Field)

	// Tepat di batas harus lolos. Kasus "tepat di batas" adalah tempat aturan seperti
	// ini paling sering salah (`14-TESTING-STRATEGY.md` §3.1).
	exact := masterstatusprogres.Input{
		Name:         strings.Repeat("A", masterstatusprogres.MaxNameLength),
		PositionCode: "002",
	}.Clean()
	require.NoError(t, exact.Check())
}

// Spasi di ujung isian dipangkas SEBELUM diperiksa, bukan sesudah — kalau tidak, nama
// berisi spasi saja akan lolos karena panjangnya bukan nol.
func TestCleanTrimsAndNormalisesCase(t *testing.T) {
	clean := masterstatusprogres.Input{Name: "  DOKUMEN DITERIMA  ", PositionCode: " 002 "}.Clean()

	require.Equal(t, "DOKUMEN DITERIMA", clean.Name)
	require.Equal(t, "002", clean.PositionCode)
}

// Keempat posisi adalah yang benar-benar ada di
// `Activity/ViewStatusProgress_act-Act.xml`. Bila daftarnya kelak pindah menjadi master
// data `F-4`, uji ini yang mengingatkan bahwa nilainya pernah ditetapkan di sini.
func TestPositionListMatchesLegacySystem(t *testing.T) {
	list := masterstatusprogres.ListPositions()
	require.Len(t, list, 4)

	require.Equal(t, "002", list[0].Code)
	require.Equal(t, "REGISTER", list[0].Name)
	require.Equal(t, "004", list[1].Code)
	require.Equal(t, "SURVEY", list[1].Name)
	require.Equal(t, "006", list[2].Code)
	require.Equal(t, "KOMITE", list[2].Name)
	require.Equal(t, "007", list[3].Code)
	require.Equal(t, "AKSEPTASI", list[3].Name)
}

// Kode 003 dan 005 tidak dipakai jalur ini; keduanya harus tetap tidak dikenal.
func TestFindPosition(t *testing.T) {
	p, known := masterstatusprogres.FindPosition("006")
	require.True(t, known)
	require.Equal(t, "KOMITE", p.Name)

	// Kolom STATUS bisa bertipe CHAR berlebar tetap yang memadatkan nilainya dengan
	// spasi tanpa memberi tanda apa pun (R-08: DDL belum ada).
	p, known = masterstatusprogres.FindPosition("  006  ")
	require.True(t, known)
	require.Equal(t, "KOMITE", p.Name)

	_, known = masterstatusprogres.FindPosition("003")
	require.False(t, known)
	_, known = masterstatusprogres.FindPosition("")
	require.False(t, known)
}

// Kode yang tidak dikenal ditampilkan APA ADANYA, tidak disembunyikan.
//
// Baris lama dapat memuat kode di luar keempat yang dikenali — kolom STATUS tidak punya
// constraint yang membatasinya — dan menyembunyikannya membuat baris tampak kosong
// tanpa sebab. Pengguna perlu melihat apa yang benar-benar tersimpan.
func TestPositionNameDoesNotHideForeignCode(t *testing.T) {
	require.Equal(t, "REGISTER", masterstatusprogres.PositionName("002"))
	require.Equal(t, "999", masterstatusprogres.PositionName("999"))
	require.Equal(t, "", masterstatusprogres.PositionName("   "))
}

// Bentuk nomor direplikasi apa adanya dari
// `Activity/InsertMstStatusProgress1_act-Act.xml`: `"0" + nomor urut`.
func TestFormatIDFollowsLegacySystem(t *testing.T) {
	require.Equal(t, "01", masterstatusprogres.FormatID(1))
	require.Equal(t, "09", masterstatusprogres.FormatID(9))

	// Cacat yang ikut terbawa, dicatat sebagai uji supaya tidak disangka rancangan:
	// lebarnya tidak dipadatkan, sehingga urutan teksnya tidak sama dengan urutan
	// penerbitannya.
	require.Equal(t, "010", masterstatusprogres.FormatID(10))
	require.True(t, masterstatusprogres.FormatID(10) < masterstatusprogres.FormatID(9),
		"sebagai teks, 010 mendahului 09 — inilah sebab daftar diurutkan di basis data")
}

// ListPositions mengembalikan salinan; pemanggil yang mengubahnya tidak boleh merusak
// daftar bagi pemanggil berikutnya.
func TestPositionListCannotBeMutatedByCaller(t *testing.T) {
	first := masterstatusprogres.ListPositions()
	first[0].Name = "DIRUSAK"

	second := masterstatusprogres.ListPositions()
	require.Equal(t, "REGISTER", second[0].Name)
}

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
	input := masterstatusprogres.Input{Name: "   ", PositionCode: "REGISTER"}.Clean()

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
	input := masterstatusprogres.Input{Name: tooLong, PositionCode: "REGISTER"}.Clean()

	err := input.Check()
	var validationErr *masterstatusprogres.ValidationError
	require.ErrorAs(t, err, &validationErr)
	require.Equal(t, "nama", validationErr.Violation[0].Field)

	// Tepat di batas harus lolos. Kasus "tepat di batas" adalah tempat aturan seperti
	// ini paling sering salah (`14-TESTING-STRATEGY.md` §3.1).
	exact := masterstatusprogres.Input{
		Name:         strings.Repeat("A", masterstatusprogres.MaxNameLength),
		PositionCode: "REGISTER",
	}.Clean()
	require.NoError(t, exact.Check())
}

// Spasi di ujung isian dipangkas SEBELUM diperiksa, bukan sesudah — kalau tidak, nama
// berisi spasi saja akan lolos karena panjangnya bukan nol.
func TestCleanTrimsAndNormalisesCase(t *testing.T) {
	clean := masterstatusprogres.Input{Name: "  DOKUMEN DITERIMA  ", PositionCode: " REGISTER "}.Clean()

	require.Equal(t, "DOKUMEN DITERIMA", clean.Name)
	require.Equal(t, "REGISTER", clean.PositionCode)
}

// Kesembilan baris ini adalah isi dropdown "Posisi" pada layar Pega yang sedang berjalan,
// beserta urutannya. Rule yang mengisinya TIDAK ADA di export (`R-16`), sehingga tidak
// ada berkas yang dapat dibandingkan — uji inilah satu-satunya tempat daftarnya
// dikunci, dan yang akan mengingatkan bahwa nilainya pernah ditetapkan di sini bila
// kelak ia pindah menjadi master data `F-4`.
//
// Urutannya ikut diuji, bukan hanya isinya: urutan dropdown adalah yang dilihat
// pengguna, dan mengurutkannya ulang secara diam-diam mengubah layar (`D-13`).
func TestPositionListMatchesLegacySystem(t *testing.T) {
	list := masterstatusprogres.ListPositions()

	order := make([]string, 0, len(list))
	for _, p := range list {
		// Nilai simpanan dan labelnya memang satu properti yang sama di sistem lama —
		// dropdown-nya mengikat pyValue dan pyPrompt ke `.CaseID`. Kalau keduanya sampai
		// berbeda, ada yang menambahkan pemetaan yang tidak pernah ada.
		require.Equal(t, p.Code, p.Name, "nilai simpanan dan label harus sama persis")
		order = append(order, p.Code)
	}

	require.Equal(t, []string{
		"All",
		"REGISTER",
		"KOMITE",
		"SURVEY",
		"AKSEPTASI",
		"OUTSTANDING",
		"BENGKEL",
		"PROCUREMENT",
		// "All" memang terulang. `pyHasNoSelection = false` pada sel dropdown-nya
		// membuktikan Pega tidak menyisipkan baris kosong, jadi pengulangan ini ada di
		// datanya sendiri — bukan baris prompt yang salah terbaca.
		"All",
	}, order)
}

// Nilai yang tidak ada di daftar harus tetap tidak dikenal.
func TestFindPosition(t *testing.T) {
	p, known := masterstatusprogres.FindPosition("KOMITE")
	require.True(t, known)
	require.Equal(t, "KOMITE", p.Name)

	// Kolom STATUS bisa bertipe CHAR berlebar tetap yang memadatkan nilainya dengan
	// spasi tanpa memberi tanda apa pun (R-08: DDL belum ada).
	p, known = masterstatusprogres.FindPosition("  KOMITE  ")
	require.True(t, known)
	require.Equal(t, "KOMITE", p.Name)

	// Beda huruf besar-kecil diabaikan, dan yang dikembalikan adalah EJAAN BAKUNYA —
	// itulah yang membuat Clean dapat membakukan isian sebelum disimpan.
	p, known = masterstatusprogres.FindPosition("all")
	require.True(t, known)
	require.Equal(t, "All", p.Code)

	_, known = masterstatusprogres.FindPosition("002")
	require.False(t, known, "kode angka bukan lagi nilai yang dikenali; lihat koreksi 2026-09-20")
	_, known = masterstatusprogres.FindPosition("")
	require.False(t, known)
}

// Kode yang tidak dikenal ditampilkan APA ADANYA, tidak disembunyikan.
//
// Baris lama dapat memuat kode di luar keempat yang dikenali — kolom STATUS tidak punya
// constraint yang membatasinya — dan menyembunyikannya membuat baris tampak kosong
// tanpa sebab. Pengguna perlu melihat apa yang benar-benar tersimpan.
func TestPositionNameDoesNotHideForeignCode(t *testing.T) {
	require.Equal(t, "REGISTER", masterstatusprogres.PositionName("REGISTER"))
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
	require.Equal(t, "All", second[0].Name)
}

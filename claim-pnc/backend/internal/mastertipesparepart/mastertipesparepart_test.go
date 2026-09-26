package mastertipesparepart_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/mastertipesparepart"
)

// validInput adalah isian yang lolos seluruh aturan, dipakai sebagai titik awal uji yang
// hanya ingin merusak SATU hal.
func validInput() mastertipesparepart.Input {
	return mastertipesparepart.Input{Name: "HYDRAULIC PUMP", CategoryID: "2"}
}

// violationFields mengembalikan nama isian yang dilaporkan sebuah galat pemeriksaan.
func violationFields(t *testing.T, err error) []string {
	t.Helper()

	var failure *mastertipesparepart.ValidationError
	require.ErrorAs(t, err, &failure)

	field := make([]string, 0, len(failure.Violation))
	for _, one := range failure.Violation {
		field = append(field, one.Field)
	}
	return field
}

func TestCheckAcceptsCompleteInput(t *testing.T) {
	require.NoError(t, validInput().Clean().Check())
}

// Nama wajib diisi. Kewajiban ini DITAMBAHKAN terhadap sistem lama, yang menyimpan apa pun
// yang diketik termasuk kosong; alasannya ada pada doc comment Input.Check.
func TestCheckRequiresName(t *testing.T) {
	input := validInput()
	input.Name = ""
	require.Equal(t, []string{"nama_tipe_sparepart"},
		violationFields(t, input.Clean().Check()))
}

// Kategori wajib dipilih — isian yang TIDAK ada padanannya di Master Kategori Sparepart,
// dan yang membuat modul ini master pertama di rumpun sparepart dengan kunci asing.
func TestCheckRequiresCategory(t *testing.T) {
	input := validInput()
	input.CategoryID = ""
	require.Equal(t, []string{"id_kategori_sparepart"},
		violationFields(t, input.Clean().Check()))
}

// SELURUH pelanggaran dikembalikan sekaligus, bukan yang pertama saja.
//
// `P-5` menuntutnya, meniru `InputRegister_act` yang menampilkan semua pesan galat dalam
// satu kali. Pada modul berisi dua isian wajib, bentuk itu benar-benar terpakai — dan
// inilah uji yang membuktikannya, karena satu isian saja tidak dapat membuktikan apa pun
// tentang bentuk jamak.
func TestCheckReportsEveryViolationAtOnce(t *testing.T) {
	err := mastertipesparepart.Input{}.Clean().Check()
	require.ElementsMatch(t,
		[]string{"nama_tipe_sparepart", "id_kategori_sparepart"},
		violationFields(t, err))
}

// Isian yang hanya berisi spasi diperlakukan sama dengan kosong.
//
// Tanpa Clean lebih dulu, " " akan lolos sebagai nama sepanjang satu karakter — dan
// tersimpan sebagai tipe yang tampak kosong di dropdown Master Sparepart. Hal yang sama
// berlaku pada kunci kategori, yang berspasi tidak akan pernah cocok dengan baris mana pun.
func TestCheckTreatsBlankAsEmpty(t *testing.T) {
	err := mastertipesparepart.Input{Name: "   ", CategoryID: "  "}.Clean().Check()
	require.ElementsMatch(t,
		[]string{"nama_tipe_sparepart", "id_kategori_sparepart"},
		violationFields(t, err))
}

func TestCheckRejectsNameLongerThanLimit(t *testing.T) {
	input := validInput()
	input.Name = strings.Repeat("A", mastertipesparepart.MaxNameLength+1)
	require.Equal(t, []string{"nama_tipe_sparepart"},
		violationFields(t, input.Clean().Check()))
}

func TestCheckAcceptsNameExactlyAtLimit(t *testing.T) {
	input := validInput()
	input.Name = strings.Repeat("A", mastertipesparepart.MaxNameLength)
	require.NoError(t, input.Clean().Check())
}

// Clean memangkas spasi pada KEDUA isian, dan TIDAK meng-uppercase.
//
// Rule validasi lamanya memang membandingkan `upper(...)`, tetapi yang di-uppercase di sana
// adalah PEMBANDINGNYA — bukan nilai yang disimpan. Memaksa huruf besar akan mengubah
// tampilan setiap baris yang disunting.
func TestCleanTrimsBothFieldsAndKeepsLetterCase(t *testing.T) {
	clean := mastertipesparepart.Input{
		Name:       "  Hydraulic Pump  ",
		CategoryID: "  2  ",
	}.Clean()

	require.Equal(t, "Hydraulic Pump", clean.Name)
	require.Equal(t, "2", clean.CategoryID)
}

// Ketiga sandi status dan labelnya. Sandinya tidak boleh berubah — tabelnya dibaca sistem
// lama DAN dibaca modul mastersparepart di aplikasi ini sendiri (ADR-0004).
func TestApprovalStatusLabel(t *testing.T) {
	require.Equal(t, "Waiting Approval", mastertipesparepart.StatusPending.Label())
	require.Equal(t, "Approve", mastertipesparepart.StatusApproved.Label())
	require.Equal(t, "Reject", mastertipesparepart.StatusRejected.Label())
	require.Empty(t, mastertipesparepart.ApprovalStatus("9").Label())
}

func TestApprovalStatusKnown(t *testing.T) {
	for _, s := range []mastertipesparepart.ApprovalStatus{"0", "1", "2"} {
		require.True(t, s.Known(), "status %q seharusnya dikenal", s)
	}
	for _, s := range []mastertipesparepart.ApprovalStatus{"", "3", "01", " 1"} {
		require.False(t, s.Known(), "status %q seharusnya TIDAK dikenal", s)
	}
}

// Sandi statusnya harus sama persis dengan yang dipakai mastersparepart untuk menyaring
// daftar acuan Tipe.
//
// Modul itu menyaring `APPROVAL = '1'` lewat lookup.go-nya, mengikuti
// `RDB List/BrowseTipeSparepart-SQL.xml`. Bila modul INI menuliskan sandi lain untuk
// "disetujui", tipe yang baru disetujui tidak akan pernah muncul di dropdown layar Master
// Sparepart — dan tidak ada satu pun pesan galat yang menandainya.
//
// Uji ini menjaga kesepakatan itu dari sisi modul yang MENULIS. Ia tidak mengimpor modul
// tetangganya — cukup menegaskan sandinya secara harfiah, karena itulah yang tertulis di
// basis data.
func TestApprovedCodeMatchesLookupFilterUsedBySparepartScreen(t *testing.T) {
	require.Equal(t, "1", string(mastertipesparepart.StatusApproved),
		"sandi disetujui wajib '1'; mastersparepart/lookup.go menyaring APPROVAL = '1'")
	require.Equal(t, "0", string(mastertipesparepart.StatusPending))
	require.Equal(t, "2", string(mastertipesparepart.StatusRejected))
}

func TestOneViolationCarriesFieldAndMessage(t *testing.T) {
	err := mastertipesparepart.OneViolation("nama_tipe_sparepart", "pesan uji")

	var failure *mastertipesparepart.ValidationError
	require.ErrorAs(t, err, &failure)
	require.Len(t, failure.Violation, 1)
	require.Equal(t, "nama_tipe_sparepart", failure.Violation[0].Field)
	require.Equal(t, "pesan uji", failure.Violation[0].Message)
	require.Contains(t, err.Error(), "pesan uji")
}

// MaxNameLength diulang di PartTypeForm.tsx, dan kedua tempat harus sama.
//
// Uji ini menjaga duplikasi itu tetap TERLIHAT: bila angkanya diubah di sini saja, uji gagal
// dan menyebut berkas mana yang harus ikut berubah. Pola yang sama dipakai Master Sparepart
// dan Master Kategori Sparepart.
//
// Berkas frontend dibaca lewat jalur relatif dari paket ini. Bila strukturnya berpindah, uji
// ini DILEWATI alih-alih gagal — ia menjaga kesamaan angka, bukan tata letak repositori.
func TestMaxNameLengthMatchesFrontendForm(t *testing.T) {
	path := filepath.Join("..", "..", "..", "frontend", "src", "modules",
		"master-tipe-sparepart", "PartTypeForm.tsx")

	content, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("berkas form frontend tidak dapat dibaca (%v); uji kesamaan angka dilewati", err)
	}

	want := "const MAX_NAME_LENGTH = 100"
	require.Contains(t, string(content), want,
		"PartTypeForm.tsx harus memakai batas yang sama dengan MaxNameLength (%d)",
		mastertipesparepart.MaxNameLength)
	require.Equal(t, 100, mastertipesparepart.MaxNameLength,
		"bila MaxNameLength berubah, PartTypeForm.tsx harus ikut berubah")
}

// Batas namanya sama dengan kedua master tetangganya.
//
// Nilai kolom ini muncul sebagai label pada layar Master Sparepart berdampingan dengan nama
// kategori, dan tiga batas berbeda pada tiga layar bertetangga hanya akan membingungkan.
// Uji ini menegaskannya secara harfiah alih-alih mengimpor konstanta modul lain — mengimpor
// akan membuat perubahan di satu master diam-diam menyeret master lain.
func TestMaxNameLengthMatchesNeighbouringMasters(t *testing.T) {
	require.Equal(t, 100, mastertipesparepart.MaxNameLength,
		"batas nama disamakan dengan masterkategorisparepart dan mastersparepart")
}

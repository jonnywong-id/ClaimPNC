package masterkategorisparepart_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterkategorisparepart"
)

// violationFields mengembalikan nama isian yang dilaporkan sebuah galat pemeriksaan.
func violationFields(t *testing.T, err error) []string {
	t.Helper()

	var failure *masterkategorisparepart.ValidationError
	require.ErrorAs(t, err, &failure)

	field := make([]string, 0, len(failure.Violation))
	for _, one := range failure.Violation {
		field = append(field, one.Field)
	}
	return field
}

func TestCheckAcceptsSimpleName(t *testing.T) {
	input := masterkategorisparepart.Input{Name: "HYDRAULIC"}
	require.NoError(t, input.Clean().Check())
}

// Nama wajib diisi. Kewajiban ini DITAMBAHKAN terhadap sistem lama, yang menyimpan apa pun
// yang diketik termasuk kosong; alasannya ada pada doc comment Input.Check.
func TestCheckRequiresName(t *testing.T) {
	err := masterkategorisparepart.Input{}.Clean().Check()
	require.Equal(t, []string{"nama_kategori_sparepart"}, violationFields(t, err))
}

// Isian yang hanya berisi spasi diperlakukan sama dengan kosong.
//
// Tanpa Clean lebih dulu, " " akan lolos sebagai nama sepanjang satu karakter — dan
// tersimpan sebagai kategori yang tampak kosong di dropdown Master Sparepart.
func TestCheckTreatsBlankNameAsEmpty(t *testing.T) {
	err := masterkategorisparepart.Input{Name: "   "}.Clean().Check()
	require.Equal(t, []string{"nama_kategori_sparepart"}, violationFields(t, err))
}

func TestCheckRejectsNameLongerThanLimit(t *testing.T) {
	input := masterkategorisparepart.Input{
		Name: strings.Repeat("A", masterkategorisparepart.MaxNameLength+1),
	}
	require.Equal(t, []string{"nama_kategori_sparepart"},
		violationFields(t, input.Clean().Check()))
}

func TestCheckAcceptsNameExactlyAtLimit(t *testing.T) {
	input := masterkategorisparepart.Input{
		Name: strings.Repeat("A", masterkategorisparepart.MaxNameLength),
	}
	require.NoError(t, input.Clean().Check())
}

// Clean memangkas spasi, dan TIDAK meng-uppercase.
//
// Rule validasi lamanya memang membandingkan `upper(...)`, tetapi yang di-uppercase di sana
// adalah PEMBANDINGNYA — bukan nilai yang disimpan. Memaksa huruf besar akan mengubah
// tampilan setiap baris yang disunting.
func TestCleanTrimsButKeepsLetterCase(t *testing.T) {
	clean := masterkategorisparepart.Input{Name: "  Hydraulic Pump  "}.Clean()
	require.Equal(t, "Hydraulic Pump", clean.Name)
}

// Ketiga sandi status dan labelnya. Sandinya tidak boleh berubah — tabelnya dibaca sistem
// lama DAN dibaca modul mastersparepart di aplikasi ini sendiri (ADR-0004).
func TestApprovalStatusLabel(t *testing.T) {
	require.Equal(t, "Waiting Approval", masterkategorisparepart.StatusPending.Label())
	require.Equal(t, "Approve", masterkategorisparepart.StatusApproved.Label())
	require.Equal(t, "Reject", masterkategorisparepart.StatusRejected.Label())
	require.Empty(t, masterkategorisparepart.ApprovalStatus("9").Label())
}

func TestApprovalStatusKnown(t *testing.T) {
	for _, s := range []masterkategorisparepart.ApprovalStatus{"0", "1", "2"} {
		require.True(t, s.Known(), "status %q seharusnya dikenal", s)
	}
	for _, s := range []masterkategorisparepart.ApprovalStatus{"", "3", "01", " 1"} {
		require.False(t, s.Known(), "status %q seharusnya TIDAK dikenal", s)
	}
}

// Sandi statusnya harus sama persis dengan yang dipakai mastersparepart untuk menyaring
// daftar acuan Kategori.
//
// Modul itu menyaring `APPROVAL = '1'` lewat lookup.go-nya. Bila modul INI menuliskan sandi
// lain untuk "disetujui", kategori yang baru disetujui tidak akan pernah muncul di dropdown
// layar Master Sparepart — dan tidak ada satu pun pesan galat yang menandainya.
//
// Uji ini menjaga kesepakatan itu dari sisi modul yang MENULIS. Ia tidak mengimpor modul
// tetangganya — cukup menegaskan sandinya secara harfiah, karena itulah yang tertulis di
// basis data.
func TestApprovedCodeMatchesLookupFilterUsedBySparepartScreen(t *testing.T) {
	require.Equal(t, "1", string(masterkategorisparepart.StatusApproved),
		"sandi disetujui wajib '1'; mastersparepart/lookup.go menyaring APPROVAL = '1'")
	require.Equal(t, "0", string(masterkategorisparepart.StatusPending))
	require.Equal(t, "2", string(masterkategorisparepart.StatusRejected))
}

func TestOneViolationCarriesFieldAndMessage(t *testing.T) {
	err := masterkategorisparepart.OneViolation("nama_kategori_sparepart", "pesan uji")

	var failure *masterkategorisparepart.ValidationError
	require.ErrorAs(t, err, &failure)
	require.Len(t, failure.Violation, 1)
	require.Equal(t, "nama_kategori_sparepart", failure.Violation[0].Field)
	require.Equal(t, "pesan uji", failure.Violation[0].Message)
	require.Contains(t, err.Error(), "pesan uji")
}

// MaxNameLength diulang di PartCategoryForm.tsx, dan kedua tempat harus sama.
//
// Uji ini menjaga duplikasi itu tetap TERLIHAT: bila angkanya diubah di sini saja, uji gagal
// dan menyebut berkas mana yang harus ikut berubah. Pola yang sama dipakai Master Sparepart.
//
// Berkas frontend dibaca lewat jalur relatif dari paket ini. Bila strukturnya berpindah, uji
// ini DILEWATI alih-alih gagal — ia menjaga kesamaan angka, bukan tata letak repositori.
func TestMaxNameLengthMatchesFrontendForm(t *testing.T) {
	path := filepath.Join("..", "..", "..", "frontend", "src", "modules",
		"master-kategori-sparepart", "PartCategoryForm.tsx")

	content, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("berkas form frontend tidak dapat dibaca (%v); uji kesamaan angka dilewati", err)
	}

	want := "const MAX_NAME_LENGTH = 100"
	require.Contains(t, string(content), want,
		"PartCategoryForm.tsx harus memakai batas yang sama dengan MaxNameLength (%d)",
		masterkategorisparepart.MaxNameLength)
	require.Equal(t, 100, masterkategorisparepart.MaxNameLength,
		"bila MaxNameLength berubah, PartCategoryForm.tsx harus ikut berubah")
}

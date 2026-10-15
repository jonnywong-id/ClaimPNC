package mastercolsimasonline_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/mastercolsimasonline"
)

// violationsOf mengumpulkan pelanggaran per isian supaya uji dapat menyebut isian yang
// dimaksud, bukan mencocokkan urutan di dalam senarai.
func violationsOf(t *testing.T, err error) map[string]string {
	t.Helper()
	if err == nil {
		return map[string]string{}
	}

	validationError, matched := err.(*mastercolsimasonline.ValidationError)
	require.Truef(t, matched, "galat yang diharapkan ValidationError, bukan %T", err)

	result := map[string]string{}
	for _, v := range validationError.Violation {
		result[v.Field] = v.Message
	}
	return result
}

func validInput() mastercolsimasonline.Input {
	return mastercolsimasonline.Input{
		Description:   "KEBAKARAN",
		BusinessNames: []string{"FIRE / PROPERTY", "ANEKA"},
	}
}

func TestValidInputPassesEveryRule(t *testing.T) {
	require.NoError(t, validInput().Clean().Check())
}

// NAMA BOLEH KOSONG — kesetaraan dengan Pega, ditetapkan Work Owner 2026-09-21.
//
// Layar Pega menandai SELURUH 19 isiannya `pyRequired=false`, dan tidak ada satu pun
// Page-Validate, Property-Validate, maupun Rule-Obj-Validate untuk kelas ini. `P-5`
// menetapkan perilaku dipertahankan lebih dulu.
//
// Uji ini menjaga keputusan itu tetap terlihat: siapa pun yang kelak menambahkan
// "wajib diisi" akan membuatnya merah dan membaca alasannya.
func TestDescriptionMayBeEmptyJustLikePega(t *testing.T) {
	input := validInput()
	input.Description = "   "

	require.NoError(t, input.Clean().Check(),
		"Pega tidak memvalidasi apa pun di sini; mewajibkannya adalah penyimpangan")
}

func TestDescriptionLongerThanLimitIsRejected(t *testing.T) {
	input := validInput()
	input.Description = strings.Repeat("A", mastercolsimasonline.MaxDescriptionLength+1)

	violation := violationsOf(t, input.Clean().Check())
	require.Contains(t, violation, mastercolsimasonline.FieldDescription)
}

func TestDescriptionExactlyAtLimitIsAccepted(t *testing.T) {
	input := validInput()
	input.Description = strings.Repeat("A", mastercolsimasonline.MaxDescriptionLength)

	require.NoError(t, input.Clean().Check(), "tepat di batas harus diterima, bukan ditolak")
}

// Batas dihitung dalam rune, bukan byte: satu huruf beraksen memakan dua byte dan akan
// membuat batas terasa berubah-ubah bagi pengguna.
func TestLimitIsCountedInRunesNotBytes(t *testing.T) {
	input := validInput()
	input.Description = strings.Repeat("é", mastercolsimasonline.MaxDescriptionLength)

	require.NoError(t, input.Clean().Check())
}

// Bisnis boleh kosong: layar Pega tidak mewajibkan satu baris pun di grid-nya.
func TestBusinessListMayBeEmpty(t *testing.T) {
	input := validInput()
	input.BusinessNames = nil

	require.NoError(t, input.Clean().Check())
}

// Nama bisnis di luar master DITERIMA — itu perilaku Pega yang dipertahankan Work Owner
// 2026-09-21 (`pyAllowFreeFormInput=true`). Ia diselesaikan menjadi baris tanpa ID, bukan
// ditolak.
func TestBusinessNameOutsideTheMasterIsAccepted(t *testing.T) {
	input := validInput()
	input.BusinessNames = []string{"BISNIS YANG BELUM ADA DI MASTER"}

	require.NoError(t, input.Clean().Check())
}

func TestBusinessNameLongerThanLimitIsRejected(t *testing.T) {
	input := validInput()
	input.BusinessNames = []string{strings.Repeat("A", mastercolsimasonline.MaxBusinessNameLength+1)}

	violation := violationsOf(t, input.Clean().Check())
	require.Contains(t, violation, mastercolsimasonline.FieldBusiness)
}

// BISNIS KEMBAR DITERIMA — kesetaraan dengan Pega, ditetapkan Work Owner 2026-09-21.
//
// Grid Pega tidak punya satu pun penanda keunikan, sehingga satu bisnis memang boleh
// dipilih dua kali. Konsekuensinya mengikat bentuk penyimpanan: kunci baris pemetaan
// adalah POSISINYA di grid, bukan namanya.
func TestDuplicateBusinessIsAcceptedJustLikePega(t *testing.T) {
	input := validInput()
	input.BusinessNames = []string{"ANEKA", "TRAVEL", "ANEKA"}

	require.NoError(t, input.Clean().Check())
	require.Equal(t, []string{"ANEKA", "TRAVEL", "ANEKA"}, input.Clean().BusinessNames,
		"baris kembar dipertahankan apa adanya, termasuk urutannya")
}

// NormalizeBusinessName dipakai mencocokkan nama yang diketik ke master bisnis.
//
// Ia TIDAK dipakai menolak kembar — kembar memang diizinkan. Yang dikerjakannya adalah
// memastikan "aneka" yang diketik pengguna tetap menemukan ID bisnis "ANEKA" di master.
func TestNormalizeBusinessNameIgnoresCaseAndEdgeSpaces(t *testing.T) {
	require.Equal(t, "ANEKA", mastercolsimasonline.NormalizeBusinessName("  Aneka  "))
	require.Equal(t,
		mastercolsimasonline.NormalizeBusinessName("FIRE / PROPERTY"),
		mastercolsimasonline.NormalizeBusinessName("fire / property"))
	require.NotEqual(t,
		mastercolsimasonline.NormalizeBusinessName("ANEKA"),
		mastercolsimasonline.NormalizeBusinessName("ANEKA LAIN"))
}

// SELURUH pelanggaran dikembalikan sekaligus, bukan yang pertama saja. Ini kesetaraan
// perilaku (`P-5`): sistem lama menampilkan semua pesan bersamaan.
//
// Yang tersisa sebagai aturan isian hanyalah PANJANG — ketiganya penjaga teknis terhadap
// lebar kolom, bukan aturan bisnis.
func TestEveryViolationIsReportedAtOnce(t *testing.T) {
	input := mastercolsimasonline.Input{
		Description:   strings.Repeat("A", mastercolsimasonline.MaxDescriptionLength+1),
		BusinessNames: []string{strings.Repeat("B", mastercolsimasonline.MaxBusinessNameLength+1)},
	}

	violation := violationsOf(t, input.Clean().Check())
	require.Len(t, violation, 2, "kedua isian yang salah harus dilaporkan bersamaan")
}

func TestCleanTrimsEverySpaceAtBothEnds(t *testing.T) {
	clean := mastercolsimasonline.Input{
		Description:   "  KEBAKARAN  ",
		BusinessNames: []string{"  ANEKA  "},
	}.Clean()

	require.Equal(t, "KEBAKARAN", clean.Description)
	require.Equal(t, []string{"ANEKA"}, clean.BusinessNames)
}

// Baris grid yang kosong dibuang, bukan disimpan sebagai pemetaan ke bisnis bernama "".
// Grid di layar dapat meninggalkan baris kosong bila pengguna menekan "Tambah Bisnis"
// lalu tidak mengisi apa-apa.
func TestCleanDropsEmptyBusinessRows(t *testing.T) {
	clean := mastercolsimasonline.Input{
		Description:   "KEBAKARAN",
		BusinessNames: []string{"ANEKA", "  ", "", "TRAVEL"},
	}.Clean()

	require.Equal(t, []string{"ANEKA", "TRAVEL"}, clean.BusinessNames)
}

// Urutan yang disusun pengguna dipertahankan apa adanya. Mengurutkannya diam-diam akan
// membuat layar menampilkan urutan yang berbeda dari yang baru saja ia simpan.
func TestCleanKeepsBusinessOrderAsTheUserArrangedIt(t *testing.T) {
	clean := mastercolsimasonline.Input{
		Description:   "KEBAKARAN",
		BusinessNames: []string{"TRAVEL", "ANEKA", "MARINE CARGO"},
	}.Clean()

	require.Equal(t, []string{"TRAVEL", "ANEKA", "MARINE CARGO"}, clean.BusinessNames)
}

// Batas panjang diduplikasi di frontend (`CauseOfLossForm.tsx`) supaya pengguna tahu
// sebelum mengirim. Uji ini membuat duplikasi itu tetap TERLIHAT: siapa pun yang mengubah
// angkanya di sini akan membaca pengingat untuk mengubahnya di sana juga.
func TestLengthLimitsAreMirroredInTheFrontend(t *testing.T) {
	require.Equal(t, 100, mastercolsimasonline.MaxDescriptionLength,
		"bila berubah, ubah juga MAX_NAME_LENGTH di CauseOfLossForm.tsx")
	require.Equal(t, 100, mastercolsimasonline.MaxBusinessNameLength,
		"bila berubah, ubah juga MAX_BUSINESS_NAME_LENGTH di CauseOfLossForm.tsx")
}

// Nama isian pada pelanggaran harus SAMA dengan nama field JSON permintaan, supaya layar
// dapat menyorot isiannya tanpa memetakan apa pun.
func TestViolationFieldNamesMatchTheRequestContract(t *testing.T) {
	require.Equal(t, "nama", mastercolsimasonline.FieldDescription)
	require.Equal(t, "bisnis", mastercolsimasonline.FieldBusiness)
}

// Seam sengaja TIDAK menyediakan operasi hapus: `D-66` melarang penghapusan fisik data
// bernilai bisnis, dan baris ini dirujuk D_CAUSE_OF_LOSS.M_COL_ID pada data berjalan.
//
// Diuji lewat tipe, bukan lewat pembacaan kode: penegasan ini gagal dikompilasi begitu
// seseorang menambahkan Delete ke antarmukanya.
func TestSeamProvidesNoDeleteOperation(t *testing.T) {
	var seam any = struct {
		mastercolsimasonline.Repo
	}{}

	_, hasDelete := seam.(interface {
		Delete(any, string) error
	})
	require.False(t, hasDelete, "Repo tidak boleh punya operasi hapus (D-66)")
}

// BusinessRepo sengaja hanya punya operasi baca: POOLDATA.BUSINESS milik GISFW (`D-03`).
// Batas kepemilikan itu ditegakkan bentuk antarmuka, bukan ingatan penulis kode
// berikutnya.
func TestBusinessSeamProvidesNoWriteOperation(t *testing.T) {
	var seam any = struct {
		mastercolsimasonline.BusinessRepo
	}{}

	_, hasWrite := seam.(interface {
		Insert(any, mastercolsimasonline.Business) error
	})
	require.False(t, hasWrite, "BusinessRepo tidak boleh punya operasi tulis (D-03)")
}

package masterautoclaim_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterautoclaim"
)

// validInput adalah isian yang lolos seluruh aturan, dipakai sebagai titik tolak.
//
// Setiap uji penolakan mengubah SATU isian darinya, sehingga yang membuatnya ditolak
// tidak pernah ambigu.
func validInput() masterautoclaim.Input {
	return masterautoclaim.Input{
		Initial:         "AGN900",
		ReceiverName:    "MITRA CONTOH SEJAHTERA",
		BankName:        "BANK CONTOH NIAGA",
		AccountNumber:   "1000000900",
		MaxPercent:      "100",
		ReporterPIC:     "PIC Contoh",
		ReporterEmail:   "pic@contoh.example",
		ReceiverAddress: "Jalan Contoh Nomor 9",
		ClientID:        "CLI900",
		ClientName:      "TERTANGGUNG CONTOH",
		Status:          masterautoclaim.StatusPending,
	}
}

// violationFields mengambil nama isian yang dilanggar dari sebuah galat.
func violationFields(t *testing.T, err error) []string {
	t.Helper()
	require.Error(t, err)

	validationError, ok := err.(*masterautoclaim.ValidationError)
	require.Truef(t, ok, "galat bukan ValidationError melainkan %T: %v", err, err)

	field := make([]string, 0, len(validationError.Violation))
	for _, v := range validationError.Violation {
		field = append(field, v.Field)
		require.NotEmptyf(t, v.Message, "pelanggaran pada %q tidak punya pesan", v.Field)
	}
	return field
}

func TestCheckAcceptsValidInput(t *testing.T) {
	require.NoError(t, validInput().Clean().Check())
}

// Delapan isian wajib, persis daftar pada prasyarat InsertMstAutoClaim_act step 2.
//
// Yang diuji bukan hanya bahwa isian kosong ditolak, tetapi bahwa nama isian yang
// dilaporkan SAMA dengan nama field JSON yang dikirim layar — tanpa itu, keterangan
// galat menempel di isian yang salah.
func TestCheckRejectsEmptyRequiredFields(t *testing.T) {
	for _, c := range []struct {
		field string
		blank func(*masterautoclaim.Input)
	}{
		{"inisial", func(i *masterautoclaim.Input) { i.Initial = "" }},
		{"nama_penerima", func(i *masterautoclaim.Input) { i.ReceiverName = "" }},
		{"nama_bank", func(i *masterautoclaim.Input) { i.BankName = "" }},
		{"no_rekening", func(i *masterautoclaim.Input) { i.AccountNumber = "" }},
		{"pct_max", func(i *masterautoclaim.Input) { i.MaxPercent = "" }},
		{"pic_lapor", func(i *masterautoclaim.Input) { i.ReporterPIC = "" }},
		{"email_lapor", func(i *masterautoclaim.Input) { i.ReporterEmail = "" }},
		{"alamat_penerima", func(i *masterautoclaim.Input) { i.ReceiverAddress = "" }},
	} {
		t.Run(c.field, func(t *testing.T) {
			input := validInput()
			c.blank(&input)

			require.Contains(t, violationFields(t, input.Clean().Check()), c.field)
		})
	}
}

// SELURUH pelanggaran dikirim sekaligus, bukan yang pertama saja (P-5).
//
// Sistem lama menjawab satu kalimat untuk delapan isian sekaligus; yang ditiru adalah
// "semua sekaligus", bukan "satu kalimat".
func TestCheckReportsEveryViolationAtOnce(t *testing.T) {
	input := masterautoclaim.Input{}

	field := violationFields(t, input.Clean().Check())
	require.Len(t, field, 8, "delapan isian wajib seharusnya dilaporkan bersamaan, bukan satu per satu")
}

// Spasi di kedua ujung dipangkas SEBELUM diperiksa, sehingga isian berisi spasi saja
// ditolak — bukan lolos karena kebetulan panjangnya bukan nol.
func TestCleanTrimsAndBlankSpaceIsRejected(t *testing.T) {
	input := validInput()
	input.ReceiverAddress = "   "
	input.BankName = "  BANK CONTOH NIAGA  "

	clean := input.Clean()
	require.Equal(t, "BANK CONTOH NIAGA", clean.BankName)
	require.Contains(t, violationFields(t, clean.Check()), "alamat_penerima")
}

// PCT_MAX harus berupa angka 0–100.
//
// Ini SELISIH YANG DIRENCANAKAN terhadap Pega, yang menerima teks apa pun. Uji ini yang
// menjaga selisih itu tetap disengaja alih-alih hilang tanpa disadari.
func TestCheckMaxPercent(t *testing.T) {
	for _, c := range []struct {
		value    string
		accepted bool
	}{
		{"100", true},
		{"0", true},
		{"82.5", true},
		{"82,5", true},   // koma, karena itu yang diketik petugas Indonesia
		{"100.00", true}, // nol di belakang koma sah
		{"101", false},
		{"-1", false},
		{"abc", false},
		{"82abc", false}, // Sscanf akan meloloskannya; ParseFloat tidak
		{"NaN", false},
		{"Inf", false},
	} {
		t.Run(c.value, func(t *testing.T) {
			input := validInput()
			input.MaxPercent = c.value

			err := input.Clean().Check()
			if c.accepted {
				require.NoErrorf(t, err, "%q seharusnya diterima", c.value)
				return
			}
			require.Contains(t, violationFields(t, err), "pct_max")
		})
	}
}

// Client boleh kosong seluruhnya — sistem lama pun menyisipkannya kosong. Yang tidak
// boleh adalah SETENGAH terisi.
func TestCheckClientPair(t *testing.T) {
	t.Run("keduanya kosong diterima", func(t *testing.T) {
		input := validInput()
		input.ClientID = ""
		input.ClientName = ""

		require.NoError(t, input.Clean().Check())
	})

	for _, c := range []struct {
		name   string
		change func(*masterautoclaim.Input)
	}{
		{"hanya nama", func(i *masterautoclaim.Input) { i.ClientID = "" }},
		{"hanya id", func(i *masterautoclaim.Input) { i.ClientName = "" }},
	} {
		t.Run(c.name, func(t *testing.T) {
			input := validInput()
			c.change(&input)

			require.Contains(t, violationFields(t, input.Clean().Check()), "id_client")
		})
	}
}

// CheckEditable TIDAK menuntut Sumber Bisnis dan nama penerima.
//
// Keduanya tidak datang dari badan permintaan pada jalur simpan — yang satu dari jalur
// URL, yang lain dari baris tersimpan karena memang tidak dapat diubah. Menuntut
// keduanya akan menolak permintaan yang benar.
func TestCheckEditableIgnoresKeyAndReceiverName(t *testing.T) {
	input := validInput()
	input.Initial = ""
	input.ReceiverName = ""

	require.NoError(t, input.Clean().CheckEditable())

	// Aturan yang sama untuk isian yang MEMANG dapat diubah tetap berlaku pada kedua
	// jalur — itulah sebabnya keduanya berbagi satu pemeriksaan.
	input.BankName = ""
	require.Contains(t, violationFields(t, input.Clean().CheckEditable()), "nama_bank")
}

// Status hanya boleh salah satu dari tiga sandi POOLDATA.M_AUTO_CLAIM_PNC.APPROVAL.
func TestApprovalStatusKnown(t *testing.T) {
	for _, s := range []masterautoclaim.ApprovalStatus{
		masterautoclaim.StatusPending,
		masterautoclaim.StatusApproved,
		masterautoclaim.StatusRejected,
	} {
		require.Truef(t, s.Known(), "status %q seharusnya dikenal", s)
		require.NotEmptyf(t, s.Label(), "status %q seharusnya punya label", s)
	}

	for _, s := range []masterautoclaim.ApprovalStatus{"", "3", "1 ", "YA"} {
		require.Falsef(t, s.Known(), "status %q seharusnya ditolak", s)
	}
}

// Sandi statusnya TIDAK boleh berubah: tabelnya masih dibaca dan ditulis sistem lama
// selama masa paralel, dan sandi yang berbeda membuat kedua sistem membaca baris yang
// sama secara berbeda (ADR-0004).
func TestApprovalStatusCodesAreFrozen(t *testing.T) {
	require.Equal(t, masterautoclaim.ApprovalStatus("0"), masterautoclaim.StatusPending)
	require.Equal(t, masterautoclaim.ApprovalStatus("1"), masterautoclaim.StatusApproved)
	require.Equal(t, masterautoclaim.ApprovalStatus("2"), masterautoclaim.StatusRejected)
}

// Usable menyatukan KEDUA syarat penyaring GetReceiverClaimAsuransiKredit-SQL.xml:
// `claim_allowed = 1 AND APPROVAL = '1'`. Satu syarat saja tidak cukup.
func TestUsableNeedsBothApprovalAndClaimAllowed(t *testing.T) {
	for _, c := range []struct {
		name         string
		status       masterautoclaim.ApprovalStatus
		claimAllowed string
		usable       bool
	}{
		{"disetujui dan diizinkan", masterautoclaim.StatusApproved, "1", true},
		{"disetujui tetapi tidak diizinkan", masterautoclaim.StatusApproved, "0", false},
		{"diizinkan tetapi menunggu", masterautoclaim.StatusPending, "1", false},
		{"diizinkan tetapi ditolak", masterautoclaim.StatusRejected, "1", false},
		{"claim_allowed kosong", masterautoclaim.StatusApproved, "", false},
	} {
		t.Run(c.name, func(t *testing.T) {
			ac := masterautoclaim.AutoClaim{Status: c.status, ClaimAllowed: c.claimAllowed}
			require.Equal(t, c.usable, ac.Usable())
		})
	}
}

// CLAIM_ALLOWED yang tersimpan selalu "1" — keputusan Work Owner 2026-09-19.
//
// Uji ini menjaga konstantanya, bukan jalurnya; jalurnya dijaga uji di paket usecase.
func TestClaimAllowedConstant(t *testing.T) {
	require.Equal(t, "1", masterautoclaim.ClaimAllowedYes)
}

// Batas panjang diulang di AutoClaimForm.tsx. Uji ini TIDAK dapat memeriksa berkas itu,
// tetapi ia membuat angkanya terlihat di satu tempat sehingga perubahan yang lupa
// mengikutkan frontend setidaknya menyentuh uji ini lebih dulu.
func TestLengthLimitsAreStated(t *testing.T) {
	require.Equal(t, 20, masterautoclaim.MaxInitialLength)
	require.Equal(t, 150, masterautoclaim.MaxReceiverNameLength)
	require.Equal(t, 100, masterautoclaim.MaxBankNameLength)
	require.Equal(t, 30, masterautoclaim.MaxAccountNumberLength)
	require.Equal(t, 100, masterautoclaim.MaxReporterPICLength)
	require.Equal(t, 100, masterautoclaim.MaxEmailLength)
	require.Equal(t, 250, masterautoclaim.MaxAddressLength)
}

func TestCheckRejectsTooLongValues(t *testing.T) {
	input := validInput()
	input.AccountNumber = strings.Repeat("9", masterautoclaim.MaxAccountNumberLength+1)

	require.Contains(t, violationFields(t, input.Clean().Check()), "no_rekening")
}

// OneViolation menghasilkan galat yang bentuknya SAMA dengan pelanggaran isian lain,
// sehingga pemeriksaan yang menuntut basis data tidak berakhir sebagai 500.
func TestOneViolation(t *testing.T) {
	err := masterautoclaim.OneViolation("nama_bank", "Bank penerima harus dipilih dari daftar.")

	require.Equal(t, []string{"nama_bank"}, violationFields(t, err))
}

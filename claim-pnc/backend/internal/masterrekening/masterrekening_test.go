package masterrekening_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterrekening"
)

func TestCheckNamesAllEmptyFieldsAtOnce(t *testing.T) {
	// Pengguna yang mengisi sembilan kolom berhak tahu seluruh yang kurang dalam satu
	// kali, bukan menemukan satu kesalahan baru pada setiap kali menekan simpan.
	err := masterrekening.Account{}.Check()
	require.Error(t, err)

	var validasi *masterrekening.ValidationError
	require.ErrorAs(t, err, &validasi)

	assert.ElementsMatch(t, []string{
		"nomor_rekening", "nama_pemilik", "nama_bank", "cabang_bank", "alamat_bank",
		"kode_bank", "tipe_rekening", "email", "nik",
	}, key(validasi.Field))
}

func TestCheckAcceptsCompleteAccount(t *testing.T) {
	require.NoError(t, completeAccount().Check())
}

func TestCheckRejectsEmailWithImplausibleShape(t *testing.T) {
	r := completeAccount()
	r.Email = "bukan-email"

	var validasi *masterrekening.ValidationError
	require.ErrorAs(t, r.Check(), &validasi)
	assert.Contains(t, validasi.Field, "email")
}

func TestCommitteeCannotApproveWithoutPassbookAndNote(t *testing.T) {
	// Dua syarat ini hanya berlaku saat MENYETUJUI. Komite tidak boleh menyetujui
	// rekening yang buktinya tidak dapat dilihat, dan alasannya harus tercatat.
	r := completeAccount()
	r.DocumentID = ""
	r.Note = ""

	var validasi *masterrekening.ValidationError
	require.ErrorAs(t, r.CheckBeforeApproval(), &validasi)
	assert.ElementsMatch(t, []string{"id_dokumen", "catatan"}, key(validasi.Field))
}

func TestCommitteeCanApproveOnceProofAndNoteExist(t *testing.T) {
	r := completeAccount()
	r.DocumentID = "DOK-001"
	r.Note = "Disetujui atasan, buku rekening sesuai."

	require.NoError(t, r.CheckBeforeApproval())
}

func TestNewNumberIsAlwaysAllowedToRegister(t *testing.T) {
	assert.True(t, masterrekening.CanBeResubmitted(nil))
}

func TestNumberRejectedByCommitteeMayBeResubmitted(t *testing.T) {
	rejected := completeAccount()
	rejected.Status = masterrekening.StatusRejected

	assert.True(t, masterrekening.CanBeResubmitted([]masterrekening.Account{rejected}))
}

func TestPendingOrApprovedNumberMayNotBeResubmitted(t *testing.T) {
	for _, status := range []masterrekening.ApprovalStatus{
		masterrekening.StatusPending,
		masterrekening.StatusApproved,
	} {
		r := completeAccount()
		r.Status = status
		assert.Falsef(t, masterrekening.CanBeResubmitted([]masterrekening.Account{r}),
			"status %q seharusnya menahan pengajuan ulang", status)
	}
}

func TestAccountUsableOnlyWhenApprovedAndActive(t *testing.T) {
	// Dua syarat, bukan satu. Rekening yang dinonaktifkan setelah disetujui tidak
	// boleh lagi menjadi tujuan pembayaran.
	kasus := []struct {
		name   string
		status masterrekening.ApprovalStatus
		active bool
		mau    bool
	}{
		{"disetujui dan aktif", masterrekening.StatusApproved, true, true},
		{"disetujui tetapi nonaktif", masterrekening.StatusApproved, false, false},
		{"menunggu walau aktif", masterrekening.StatusPending, true, false},
		{"ditolak walau aktif", masterrekening.StatusRejected, true, false},
	}
	for _, k := range kasus {
		t.Run(k.name, func(t *testing.T) {
			r := completeAccount()
			r.Status = k.status
			r.Active = k.active
			assert.Equal(t, k.mau, r.Usable())
		})
	}
}

func TestTrimCashierResponseTakesPartAfterBracket(t *testing.T) {
	// Sistem lama memangkasnya di dalam SQL dengan SUBSTR/INSTR. Pemangkasannya
	// pindah ke Go; hasilnya wajib sama.
	assert.Equal(t, "Account sudah terdaftar",
		masterrekening.TrimCashierResponse("[ERR-01] Account sudah terdaftar"))

	// Pesan tanpa kurung siku dikembalikan utuh, bukan menjadi kosong.
	assert.Equal(t, "Berhasil", masterrekening.TrimCashierResponse("Berhasil"))
	assert.Equal(t, "", masterrekening.TrimCashierResponse(""))
}

func TestStatusLabelsFollowLegacyScreenWording(t *testing.T) {
	assert.Equal(t, "Menunggu", masterrekening.StatusPending.Label())
	assert.Equal(t, "Committee Approve", masterrekening.StatusApproved.Label())
	assert.Equal(t, "Committee Reject", masterrekening.StatusRejected.Label())
	assert.Equal(t, "", masterrekening.ApprovalStatus("7").Label())
}

func completeAccount() masterrekening.Account {
	return masterrekening.Account{
		Number:      "1234567890",
		OwnerName:   "BENGKEL CONTOH SEJAHTERA",
		BankName:    "BANK CONTOH",
		BankBranch:  "JAKARTA PUSAT",
		BankAddress: "JL. CONTOH NO. 1",
		BankCode:    "014",
		AccountType: "BIASA",
		Email:       "keuangan@contoh.co.id",
		NIK:         "3171000000000000",
		Active:      true,
		Status:      masterrekening.StatusPending,
	}
}

func key(m map[string]string) []string {
	result := make([]string, 0, len(m))
	for k := range m {
		result = append(result, k)
	}
	return result
}

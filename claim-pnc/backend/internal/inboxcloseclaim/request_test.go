package inboxcloseclaim_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxcloseclaim"
)

func validRequest() inboxcloseclaim.ClaimRequest {
	return inboxcloseclaim.ClaimRequest{
		ID:          "abc123",
		Kind:        inboxcloseclaim.RequestReopen,
		ClaimID:     "ASM-FW-GCNMFW-WORK PNCN.26.0001",
		Status:      inboxcloseclaim.RequestPending,
		ActorLogin:  "BUDI",
		RequestedAt: time.Now().UTC(),
	}
}

// TestValidRequestPasses memastikan permintaan yang lengkap tidak ditolak.
func TestValidRequestPasses(t *testing.T) {
	require.NoError(t, validRequest().Normalize().Validate())
}

// TestValidateCollectsEveryViolation menjaga `12-CROSSCUTTING` §1.2 butir 1.
//
// Seluruh pelanggaran dikembalikan SEKALIGUS, meniru perilaku Pega yang menampilkan semua
// pesan bersamaan. Mengembalikannya satu per satu akan membuat pengguna menekan tombol
// berkali-kali untuk menemukan kesalahan berikutnya.
func TestValidateCollectsEveryViolation(t *testing.T) {
	err := inboxcloseclaim.ClaimRequest{Kind: "entah"}.Normalize().Validate()
	require.Error(t, err)

	var validation *inboxcloseclaim.ValidationError
	require.True(t, errors.As(err, &validation))

	fields := map[string]bool{}
	for _, violation := range validation.Violations {
		fields[violation.Field] = true
	}

	require.True(t, fields[inboxcloseclaim.FieldKind], "jenis yang tidak dikenal harus dilaporkan")
	require.True(t, fields[inboxcloseclaim.FieldClaimID], "klaim kosong harus dilaporkan")
	require.True(t, fields[inboxcloseclaim.FieldActor], "pemohon kosong harus dilaporkan")
	require.True(t, fields[inboxcloseclaim.FieldRequestedAt], "waktu kosong harus dilaporkan")
	require.Len(t, validation.Violations, 4, "keempatnya dikembalikan sekaligus, bukan satu per satu")
}

// TestReasonLengthIsCheckedInGo menjaga pesan yang dapat dibaca.
//
// Tanpa pemeriksaan ini, alasan yang terlalu panjang dijawab Oracle dengan ORA-12899 yang
// menyebut nama kolom internal — galat yang tidak berarti apa pun bagi pengguna.
func TestReasonLengthIsCheckedInGo(t *testing.T) {
	request := validRequest()
	request.Reason = strings.Repeat("a", inboxcloseclaim.MaxReasonLength+1)

	err := request.Normalize().Validate()
	require.Error(t, err)

	var validation *inboxcloseclaim.ValidationError
	require.True(t, errors.As(err, &validation))
	require.Equal(t, inboxcloseclaim.FieldReason, validation.Violations[0].Field)
}

// TestReasonLengthCountsRunes menjaga batas dihitung dalam KARAKTER, bukan byte.
//
// Kolomnya `VARCHAR2(1500)` — panjang dalam byte bila basis datanya tidak memakai semantik
// karakter. Yang dijamin di sini adalah tidak ada teks sah yang ditolak lebih awal; batas
// byte-nya tetap urusan basis data.
func TestReasonLengthCountsRunes(t *testing.T) {
	request := validRequest()
	request.Reason = strings.Repeat("é", inboxcloseclaim.MaxReasonLength)

	require.NoError(t, request.Normalize().Validate())
}

// TestNormalizeUppercasesTheLogin menjaga kunci pencocokan identitas.
//
// Export memuat tiga nama peran yang muncul dalam dua kapitalisasi sekaligus (`D-58`), dan
// perbandingan yang hanya satu sisinya diseragamkan tidak pernah cocok — gagalnya diam.
func TestNormalizeUppercasesTheLogin(t *testing.T) {
	request := validRequest()
	request.ActorLogin = "  budi_santoso  "

	require.Equal(t, "BUDI_SANTOSO", request.Normalize().ActorLogin)
}

// TestNormalizeDefaultsStatusToPending menjaga aplikasi hanya pernah menulis 'menunggu'.
func TestNormalizeDefaultsStatusToPending(t *testing.T) {
	request := validRequest()
	request.Status = ""

	require.Equal(t, inboxcloseclaim.RequestPending, request.Normalize().Status)
}

// TestNormalizeLeavesZeroTimeAlone menjaga waktu yang lupa diisi tertangkap Validate.
//
// Mengisinya diam-diam dengan waktu sekarang di dalam Normalize akan menyembunyikan cacat
// pemrograman — dan jejak yang waktunya diisi di tempat yang salah bukan jejak.
func TestNormalizeLeavesZeroTimeAlone(t *testing.T) {
	request := validRequest()
	request.RequestedAt = time.Time{}

	require.True(t, request.Normalize().RequestedAt.IsZero())
	require.Error(t, request.Normalize().Validate())
}

// TestParseRequestKind menjaga kedua jenis beserta penolakan nilai lain.
func TestParseRequestKind(t *testing.T) {
	kind, known := inboxcloseclaim.ParseRequestKind("REOPEN")
	require.True(t, known)
	require.Equal(t, inboxcloseclaim.RequestReopen, kind)

	kind, known = inboxcloseclaim.ParseRequestKind(" salin ")
	require.True(t, known)
	require.Equal(t, inboxcloseclaim.RequestCopy, kind)

	_, known = inboxcloseclaim.ParseRequestKind("hapus")
	require.False(t, known, "jenis yang tidak ada di layar tidak boleh diterima")
}

// TestReopenEffectMatchesWorkOwnerDecision menjaga niat yang ditetapkan 2026-09-23.
//
// Nilainya BUKAN hasil pembacaan export — export tidak memuat satu pun rule yang menulis
// status `1164` maupun menyentuh `PYREOPEN*`. Ia keputusan Work Owner, dan uji ini yang
// membuatnya tidak berubah tanpa disadari.
func TestReopenEffectMatchesWorkOwnerDecision(t *testing.T) {
	require.Equal(t, "New", inboxcloseclaim.EfekStatusKerjaReopen)
	require.Equal(t, "1164", inboxcloseclaim.EfekStatusKlaimReopen)
	require.Equal(t, "polis_objek_coverage", inboxcloseclaim.LingkupSalinBaku)
}

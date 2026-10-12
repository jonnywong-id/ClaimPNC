package masterpenolakan_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterpenolakan"
)

func TestCommitteeNoteIsRequired(t *testing.T) {
	// Aturan ini TIDAK ADA di sistem lama — layar Pega meneruskan isian apa adanya,
	// sehingga catatan kosong pun tersimpan. Ditambahkan mengikuti keputusan yang sama
	// pada Master Status Klaim (2026-09-17).
	violation := violationsOf(t, masterpenolakan.InputKomite{}.Clean().Check())
	require.Contains(t, violation, "catatan")
}

func TestCommitteeNoteWithWhitespaceOnlyIsRejected(t *testing.T) {
	violation := violationsOf(t, masterpenolakan.InputKomite{Note: "  \t "}.Clean().Check())
	require.Contains(t, violation, "catatan")
}

func TestCommitteeNoteExactlyAtLimitIsAccepted(t *testing.T) {
	input := masterpenolakan.InputKomite{
		Note: strings.Repeat("A", masterpenolakan.MaxNoteLengthKomite),
	}.Clean()

	require.NoError(t, input.Check())
}

func TestCommitteeNoteLongerThanLimitIsRejected(t *testing.T) {
	input := masterpenolakan.InputKomite{
		Note: strings.Repeat("A", masterpenolakan.MaxNoteLengthKomite+1),
	}.Clean()

	violation := violationsOf(t, input.Check())
	require.Contains(t, violation, "catatan")
}

func TestFirstCommitteeIDIsOneHundredEleven(t *testing.T) {
	// Angka 111 direplikasi apa adanya dari `INSERTMASTERREJECTEDKOMITE.prc:9`. Tidak ada
	// keterangan apa pun tentang asalnya di sumbernya; ia dipertahankan karena P-5.
	require.Equal(t, 111, masterpenolakan.FirstIDKomite)
	require.Equal(t, "111", masterpenolakan.FormatIDKomite(masterpenolakan.FirstIDKomite))
}

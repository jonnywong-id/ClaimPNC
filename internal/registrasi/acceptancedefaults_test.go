package registrasi_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/registrasi"
)

// AcceptationLOD_PreAct: Tipe Akseptasi selalu = Tipe Pembayaran; Nama Komite = anggota
// terakhir yang menyetujui; Nilai LOD = Gross × Share ASM / 100 — dua terakhir hanya Non-MBU.
func TestDefaultAcceptanceFollowsPreActivity(t *testing.T) {
	line := registrasi.SettlementLine{PaymentType: "2", Gross: registrasi.Rupiah(10_000_000), ShareASM: registrasi.Percent(400_000)}
	fire := registrasi.Policy{Line: registrasi.LineFire}
	members := []registrasi.CommitteeMember{
		{Operator: "KEDUA", Level: 2, Decision: registrasi.DecisionApprove},
		{Operator: "PERTAMA", Level: 1, Decision: registrasi.DecisionApprove},
	}

	d := registrasi.DefaultAcceptance(line, fire, "Gudang", members[:1], "")
	require.Equal(t, "2", d.Type)
	require.Equal(t, "KEDUA", d.CommitteeName)
	require.True(t, d.HasLODValue)
	require.Equal(t, registrasi.Rupiah(4_000_000), d.LODValue)
	require.False(t, d.LocationMissing)

	// Dua anggota atau lebih: pengganti dari pengaturan; tanpa pengaturan, anggota terakhir
	// (urut jenjang) yang menyetujui.
	require.Equal(t, "PENGGANTI", registrasi.DefaultAcceptance(line, fire, "Gudang", members, "PENGGANTI").CommitteeName)
	require.Equal(t, "KEDUA", registrasi.DefaultAcceptance(line, fire, "Gudang", members, "").CommitteeName)

	// Anggota yang belum/tidak menyetujui tidak dipakai.
	pending := []registrasi.CommitteeMember{{Operator: "MENUNGGU", Level: 1, Decision: "0"}}
	require.Equal(t, "", registrasi.DefaultAcceptance(line, fire, "Gudang", pending, "").CommitteeName)

	require.True(t, registrasi.DefaultAcceptance(line, fire, "  ", nil, "").LocationMissing)

	// Lini lain: hanya Tipe Akseptasi.
	pa := registrasi.DefaultAcceptance(line, registrasi.Policy{Line: registrasi.LinePersonalAccident}, "", members, "PENGGANTI")
	require.Equal(t, registrasi.AcceptanceDefaults{Type: "2"}, pa)
}

// Langkah 17: akseptasi Non-MBU tanpa lokasi kejadian ditolak dengan pesan Pega.
func TestAcceptanceWithoutLocationIsRejectedForNonMBU(t *testing.T) {
	line := registrasi.SettlementLine{PaymentType: "2"}
	_, err := registrasi.ValidateAcceptance(line, registrasi.Policy{Line: registrasi.LineFire}, registrasi.AcceptanceForm{}, registrasi.AcceptanceCheck{})
	require.ErrorContains(t, err, string(registrasi.ViolationAcceptanceLocation))

	_, err = registrasi.ValidateAcceptance(line, registrasi.Policy{Line: registrasi.LineFire}, registrasi.AcceptanceForm{}, registrasi.AcceptanceCheck{Location: "Gudang"})
	require.NotContains(t, err.Error(), string(registrasi.ViolationAcceptanceLocation))
}

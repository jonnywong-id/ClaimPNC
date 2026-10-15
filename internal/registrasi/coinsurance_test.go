package registrasi_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/registrasi"
)

// TestDeriveCoinsuranceFollowsPegaProcedure menjaga porting
// Database/PEGA_CONVERT_JSONKLAIM_PNC.prc baris 315–373.
//
// Setiap kasus menyebut cabang PL/SQL yang dijaganya. Hasilnya ditulis ke COINSNAME,
// LEADER_MEMBER, dan SHAREASM pada POOLDATA.T_CLAIM_PNC — kolom yang dibaca laporan lama.
func TestDeriveCoinsuranceFollowsPegaProcedure(t *testing.T) {
	const asm = "PT ASURANSI SINAR MAS"
	share := func(p float64) registrasi.Percent { return registrasi.Percent(p * 10_000) }
	row := func(leader, name string, p float64) registrasi.CoinsuranceRow {
		return registrasi.CoinsuranceRow{Leader: leader, CoinsName: name, PercentShare: share(p), HasShare: true}
	}

	cases := []struct {
		name      string
		typeCoins string
		rows      []registrasi.CoinsuranceRow
		ceding    string
		facShare  registrasi.Percent
		want      registrasi.Coinsurance
	}{
		{
			name: "tanpa CoinsList: Sinar Mas leader penuh (baris 356-359)",
			want: registrasi.Coinsurance{Name: registrasi.OwnCompany, Role: "LEADER", ShareASM: registrasi.PercentFull, HasShare: true},
		},
		{
			name: "Sinar Mas leader: nama dan share dari barisnya (baris 338-347)",
			rows: []registrasi.CoinsuranceRow{row("true", asm, 60), row("false", "PT LAIN", 40)},
			want: registrasi.Coinsurance{Name: asm, Role: "LEADER", ShareASM: share(60), HasShare: true},
		},
		{
			name: "perusahaan lain leader: Sinar Mas MEMBER dengan sharenya sendiri (baris 348)",
			rows: []registrasi.CoinsuranceRow{row("true", "PT LAIN", 70), row("false", asm, 30)},
			want: registrasi.Coinsurance{Name: "PT LAIN", Role: "MEMBER", ShareASM: share(30), HasShare: true},
		},
		{
			// Di Oracle nama kosong adalah NULL: LIKE dan NOT LIKE sama-sama tidak benar,
			// sehingga baris leader tanpa nama TIDAK menjadi MEMBER.
			name: "leader tanpa nama jatuh ke cabang NULL: LEADER 100, nama kosong (baris 339, 350-352)",
			rows: []registrasi.CoinsuranceRow{row("true", "", 50)},
			want: registrasi.Coinsurance{Name: "", Role: "LEADER", ShareASM: registrasi.PercentFull, HasShare: true},
		},
		{
			name: "tidak ada baris yang memberi peran: bawaan Sinar Mas (baris 363-367)",
			rows: []registrasi.CoinsuranceRow{row("false", "PT LAIN", 40)},
			want: registrasi.Coinsurance{Name: registrasi.OwnCompany, Role: "LEADER", ShareASM: registrasi.PercentFull, HasShare: true},
		},
		{
			name:      "fakultatif masuk menimpa seluruhnya (baris 369-373)",
			typeCoins: "F",
			rows:      []registrasi.CoinsuranceRow{row("true", asm, 60)},
			ceding:    "PT CEDING",
			facShare:  share(25),
			want:      registrasi.Coinsurance{Name: "PT CEDING", Role: "FAC IN", ShareASM: share(25), HasShare: true},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := registrasi.DeriveCoinsurance(c.typeCoins, c.rows, c.ceding, c.facShare, c.facShare != 0)
			require.Equal(t, c.want, got)
		})
	}
}

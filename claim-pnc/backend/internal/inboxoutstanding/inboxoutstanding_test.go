package inboxoutstanding_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxoutstanding"
)

// Nama uji menyebut ATURANNYA, bukan nama fungsinya — `14-TESTING-STRATEGY.md` §3.2.
// Daftar uji dengan begitu terbaca sebagai daftar aturan yang berlaku.

func TestStatusTampilDiturunkanDariStatusProses(t *testing.T) {
	cases := []struct {
		statusProses string
		want         inboxoutstanding.DisplayStatus
	}{
		// Nilai `PYSTATUSWORK` sebagaimana benar-benar tersimpan.
		//
		// Uji ini sempat menegaskan "BERJALAN", "SELESAI", "DITOLAK" — nilai tabel yang
		// salah pakai. Ia lulus, karena kode dan uji sama-sama dikarang dari premis yang
		// sama. Yang membongkarnya satu baris data produksi, bukan uji mana pun.
		{"New", inboxoutstanding.DisplayOnProgress},
		{"Resolved-Completed", inboxoutstanding.DisplayClose},
		{"Resolved-Rejected", inboxoutstanding.DisplayReject},

		// Spasi di ujung tidak boleh mengubah label: kolom Oracle `CHAR` melapisi
		// nilainya dengan spasi, dan `T_CLAIMLIST_ADMIN` diisi sistem lama.
		{"  New  ", inboxoutstanding.DisplayOnProgress},

		// Status yang TIDAK dikenali tampil apa adanya, bukan dipaksa jadi "On Progress".
		//
		// Inilah yang dulu menyembunyikan cacatnya: setiap nilai jatuh ke default dan
		// keluar sebagai "On Progress" — termasuk klaim yang sudah ditutup.
		{"ENTAH_APA", inboxoutstanding.DisplayStatus("ENTAH_APA")},
		{"", inboxoutstanding.DisplayStatus("")},
	}

	for _, c := range cases {
		claim := inboxoutstanding.OutstandingClaim{ProcessStatus: c.statusProses}
		require.Equal(t, c.want, claim.DisplayStatus(), "status proses %q", c.statusProses)
	}
}

func TestUmurKlaimDihitungTerhadapTanggalWIBBukanSelisihJam(t *testing.T) {
	wib := time.FixedZone("WIB", 7*60*60)

	// Klaim didaftarkan 23.00 WIB tanggal 1, dilihat 01.00 WIB tanggal 2.
	// Selisihnya hanya dua jam, tetapi bagi pengguna ia sudah berumur SATU hari.
	registered := time.Date(2026, 9, 1, 23, 0, 0, 0, wib)
	now := time.Date(2026, 9, 2, 1, 0, 0, 0, wib)

	claim := inboxoutstanding.OutstandingClaim{RegisteredAt: registered}
	require.Equal(t, 1, claim.AgeInDays(now, wib))
}

func TestUmurKlaimTidakBergeserOlehPenyimpananUTC(t *testing.T) {
	wib := time.FixedZone("WIB", 7*60*60)

	// Didaftarkan 2 September pukul 03.00 WIB — yaitu 1 September 20.00 UTC.
	// Bila dihitung terhadap tanggal UTC, umurnya akan terbaca satu hari lebih tua.
	registered := time.Date(2026, 9, 1, 20, 0, 0, 0, time.UTC)
	now := time.Date(2026, 9, 2, 10, 0, 0, 0, time.UTC) // 2 Sept 17.00 WIB

	claim := inboxoutstanding.OutstandingClaim{RegisteredAt: registered}
	require.Equal(t, 0, claim.AgeInDays(now, wib), "keduanya jatuh pada tanggal WIB yang sama")
}

func TestUmurKlaimTidakPernahNegatif(t *testing.T) {
	wib := time.FixedZone("WIB", 7*60*60)
	now := time.Date(2026, 9, 1, 12, 0, 0, 0, wib)

	claim := inboxoutstanding.OutstandingClaim{
		RegisteredAt: time.Date(2026, 9, 5, 12, 0, 0, 0, wib),
	}
	require.Equal(t, 0, claim.AgeInDays(now, wib))
}

func TestUmurKlaimNolBilaTanggalPendaftaranKosong(t *testing.T) {
	claim := inboxoutstanding.OutstandingClaim{}
	require.Equal(t, 0, claim.AgeInDays(time.Now(), time.UTC))
}

func TestBatasDataPerLiniBisnis(t *testing.T) {
	excluded := []string{"10008", "10010", "10015", "10023"}

	// Nilainya disalin dari potongan WHERE sistem lama — lihat komentar ScopeFor.
	cases := []struct {
		line             string
		wantUnrestricted bool
		wantPanels       []string
		wantExcluded     []string
	}{
		{inboxoutstanding.LinePA, false, []string{"002"}, nil},
		{inboxoutstanding.LineTravel, false, []string{"005"}, nil},
		{inboxoutstanding.LineNonMBU, false, []string{"003", "004", "006"}, excluded},

		// BONDING melihat SELURUH Group Panel, dikurangi kelompok bisnis yang
		// dikecualikan — ia karena itu bukan "tanpa batas".
		{inboxoutstanding.LineBonding, false, nil, excluded},

		// Kosong dan tidak dikenali sama-sama jatuh ke tanpa batas (`K-4`).
		{"", true, nil, nil},
		{"TRAVELOKA", true, nil, nil},
	}

	for _, c := range cases {
		scope := inboxoutstanding.ScopeFor(c.line)
		require.Equal(t, c.wantUnrestricted, scope.Unrestricted, "lini %q", c.line)
		require.Equal(t, c.wantPanels, scope.GroupPanels, "lini %q", c.line)
		require.Equal(t, c.wantExcluded, scope.ExcludedBusinessGroups, "lini %q", c.line)
	}
}

// BONDING adalah satu-satunya lini yang batasnya SELURUHNYA berupa pengecualian.
//
// Uji terpisah karena mudah salah: scope tanpa Group Panel terlihat seperti scope kosong,
// dan scope kosong gagal tertutup. Yang membedakannya adalah daftar pengecualian.
func TestBondingMenyaringLewatPengecualianBukanLini(t *testing.T) {
	scope := inboxoutstanding.ScopeFor(inboxoutstanding.LineBonding)

	require.False(t, scope.Unrestricted, "BONDING punya batas, bukan tanpa batas")
	require.Empty(t, scope.GroupPanels, "BONDING tidak menyaring Group Panel")
	require.NotEmpty(t, scope.ExcludedBusinessGroups, "batasnya ada pada pengecualian")
}

func TestBatasDataTidakPekaBesarKecilHurufDanSpasiTepi(t *testing.T) {
	// Kolom LOGIN_ID di sistem lama tidak diseragamkan, dan LINEBUSINESS yang menyusulnya
	// tidak punya alasan untuk lebih rapi.
	scope := inboxoutstanding.ScopeFor("  pa  ")
	require.False(t, scope.Unrestricted)
	require.Equal(t, []string{"002"}, scope.GroupPanels)
}

func TestFilterDinormalkanKeNilaiYangMasukAkal(t *testing.T) {
	f := inboxoutstanding.Filter{
		Search:     "  PNCN.26  ",
		Stage:      "  Komite ",
		BranchCode: " JKT ",
		Limit:      0,
		Offset:     -5,
	}.Normalize()

	require.Equal(t, "PNCN.26", f.Search)
	require.Equal(t, "Komite", f.Stage)
	require.Equal(t, "JKT", f.BranchCode)
	require.Equal(t, inboxoutstanding.DefaultLimit, f.Limit)
	require.Equal(t, 0, f.Offset)
}

func TestFilterMemangkasLimitYangMelebihiBatas(t *testing.T) {
	f := inboxoutstanding.Filter{Limit: 5000}.Normalize()
	require.Equal(t, inboxoutstanding.MaxLimit, f.Limit)
}

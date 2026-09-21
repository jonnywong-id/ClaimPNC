package inboxxol_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxxol"
)

func TestTipePemberitahuanDibacaTanpaPeduliHurufBesarKecil(t *testing.T) {
	for _, raw := range []string{"PLA", "pla", " Pla "} {
		parsed, ok := inboxxol.ParseAdviceType(raw)
		require.True(t, ok, "gagal membaca %q", raw)
		require.Equal(t, inboxxol.AdvicePLA, parsed)
	}
	parsed, ok := inboxxol.ParseAdviceType("dla")
	require.True(t, ok)
	require.Equal(t, inboxxol.AdviceDLA, parsed)
}

// TestTipePemberitahuanKosongDitolak menjaga teks kosong tidak diam-diam berarti
// "kedua tipe".
//
// Kalau kosong diperlakukan sah, salah ketik tidak dapat dibedakan dari "tidak disaring",
// dan pencarian yang seharusnya gagal justru mengembalikan daftar.
func TestTipePemberitahuanKosongDitolak(t *testing.T) {
	for _, raw := range []string{"", "   ", "PLADLA", "LA"} {
		_, ok := inboxxol.ParseAdviceType(raw)
		require.False(t, ok, "tipe %q seharusnya ditolak", raw)
	}
}

// TestGroupBusinessTanpaNamaMenjadiTreatyInward menguji pengganti
// `Database/GET_GROUPBUSINESS_XOL.fnc`, yang menulis 'TREATY INWARD' saat NOTE-nya NULL.
func TestGroupBusinessTanpaNamaMenjadiTreatyInward(t *testing.T) {
	require.Equal(t, inboxxol.TreatyInwardLabel, inboxxol.BusinessGroup{ID: "99"}.DisplayName())
	require.Equal(t, inboxxol.TreatyInwardLabel, inboxxol.BusinessGroup{ID: "99", Name: "  "}.DisplayName())
	require.Equal(t, "Fire", inboxxol.BusinessGroup{ID: "10", Name: "Fire"}.DisplayName())
}

func TestNamaGroupBusinessDirangkaiDenganKoma(t *testing.T) {
	master := inboxxol.MasterXOL{BusinessGroups: []inboxxol.BusinessGroup{
		{ID: "10", Name: "Fire"},
		{ID: "20", Name: "Marine Cargo"},
		{ID: "99", Name: ""},
	}}
	require.Equal(t, "Fire, Marine Cargo, TREATY INWARD", master.BusinessGroupNames())
	require.Equal(t, []string{"10", "20", "99"}, master.BusinessGroupIDs())
}

func TestPerjanjianTanpaGroupBusinessTidakMenghasilkanKodeApaPun(t *testing.T) {
	master := inboxxol.MasterXOL{ID: "XOL-003"}
	require.Empty(t, master.BusinessGroupNames())
	require.Nil(t, master.BusinessGroupIDs())
}

// TestKodeGroupBusinessKosongDibuang menjaga baris master yang kolomnya kosong tidak
// menjadi penyaring kosong di dalam klausa IN.
func TestKodeGroupBusinessKosongDibuang(t *testing.T) {
	master := inboxxol.MasterXOL{BusinessGroups: []inboxxol.BusinessGroup{
		{ID: "10", Name: "Fire"},
		{ID: "   ", Name: "Entah"},
	}}
	require.Equal(t, []string{"10"}, master.BusinessGroupIDs())
}

func TestPerjanjianMenungguKomiteDikenaliDariStatusNol(t *testing.T) {
	require.True(t, inboxxol.MasterXOL{CommitteeStatus: "0"}.AwaitingCommittee())
	require.True(t, inboxxol.MasterXOL{CommitteeStatus: " 0 "}.AwaitingCommittee())
	require.False(t, inboxxol.MasterXOL{CommitteeStatus: "1"}.AwaitingCommittee())
	require.False(t, inboxxol.MasterXOL{}.AwaitingCommittee())
}

// TestNomorPemberitahuanMerakitRevisi menguji perakitan yang di sistem lama dikerjakan
// SQL lewat `CASE REVISI WHEN '0' THEN … ELSE … || ' / ' || REVISI END`.
func TestNomorPemberitahuanMerakitRevisi(t *testing.T) {
	require.Equal(t, "PLA/2024/0001",
		inboxxol.Advice{Number: "PLA/2024/0001", Revision: "0"}.DisplayNumber())

	require.Equal(t, "PLA/2024/0001",
		inboxxol.Advice{Number: "PLA/2024/0001", Revision: ""}.DisplayNumber())

	require.Equal(t, "PLA/2024/0001 / 2",
		inboxxol.Advice{Number: "PLA/2024/0001", Revision: "2"}.DisplayNumber())
}

// TestPenyaringKosongDikenali menjaga klausa `IN ()` tidak pernah terbentuk.
//
// Di Oracle ia galat sintaksis; di sebagian dialek lain ia mengembalikan SELURUH baris —
// yang berarti menampilkan klaim group business yang tidak ditanggung perjanjian itu.
func TestPenyaringKosongDikenali(t *testing.T) {
	require.True(t, inboxxol.ClaimFilter{}.Empty())
	require.True(t, inboxxol.ClaimFilter{Year: "2024"}.Empty())
	require.True(t, inboxxol.ClaimFilter{BusinessGroupIDs: []string{"10"}}.Empty())
	require.False(t, inboxxol.ClaimFilter{Year: "2024", BusinessGroupIDs: []string{"10"}}.Empty())
}

// TestPenyaringTreatyInwardTidakMenuntutGroupBusiness menguji pembedaan yang menentukan:
// treaty inward dibaca dari tabel yang TIDAK punya kolom group business.
//
// Menyamakan keduanya akan menghilangkan baris treaty inward pada perjanjian yang group
// business-nya belum diisi — padahal justru di situ treaty inward paling mungkin ada.
func TestPenyaringTreatyInwardTidakMenuntutGroupBusiness(t *testing.T) {
	filter := inboxxol.BreakdownFilter{LossDate: "12/03/2024", CauseOfLoss: "BANJIR"}
	require.True(t, filter.Empty(), "rincian klaim sendiri menuntut group business")
	require.False(t, filter.TreatyEmpty(), "treaty inward tidak menuntut group business")

	require.True(t, inboxxol.BreakdownFilter{CauseOfLoss: "BANJIR"}.TreatyEmpty())
	require.True(t, inboxxol.BreakdownFilter{LossDate: "12/03/2024"}.TreatyEmpty())
}

func TestPenyaringPemberitahuanMenuntutKetigaIsian(t *testing.T) {
	require.True(t, inboxxol.AdviceFilter{CauseOfLoss: "BANJIR", Type: inboxxol.AdvicePLA}.Empty())
	require.True(t, inboxxol.AdviceFilter{Year: "2024", Type: inboxxol.AdvicePLA}.Empty())
	require.True(t, inboxxol.AdviceFilter{Year: "2024", CauseOfLoss: "BANJIR"}.Empty())
	require.False(t, inboxxol.AdviceFilter{
		Year: "2024", CauseOfLoss: "BANJIR", Type: inboxxol.AdviceDLA,
	}.Empty())
}

func TestIdentitasPemanggilDipangkas(t *testing.T) {
	require.Equal(t, "PICTEKNIK01", inboxxol.Caller{Login: "  PICTEKNIK01 "}.Clean().Login)
}

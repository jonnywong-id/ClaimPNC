package memory_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxbandinghargasalvage"
	"claim-pnc/internal/inboxbandinghargasalvage/repo/memory"
)

// Uji di berkas ini memeriksa PENULISAN keputusan, termasuk ketiga langkah tambahannya.
//
// Seluruhnya memakai baris contoh, karena di situlah pasangan baris yang dibutuhkan sudah
// tersedia — khususnya PNC-0460/1, satu barang yang ditangani DUA komite berjenjang.

func putusan(
	t *testing.T, login, detail, salvage string, setujui bool,
) inboxbandinghargasalvage.DecisionCommand {
	t.Helper()

	command, err := inboxbandinghargasalvage.NewDecisionCommand(
		inboxbandinghargasalvage.DecisionInput{
			DetailObject: detail,
			SalvageID:    salvage,
			RequestPrice: "8200000.00",
			Note:         "sudah diperiksa",
			Approve:      setujui,
		},
		inboxbandinghargasalvage.Caller{Login: login},
	)
	require.NoError(t, err)
	return command
}

// penulis menyerahkan penyimpanan contoh beserta penulisnya, keduanya berbagi baris yang sama.
func penulis(t *testing.T) (*memory.Store, *memory.Writer) {
	t.Helper()

	store := memory.NewSampleStore()
	return store, memory.NewWriter(store)
}

func TestKeputusanTercatatDanBarisnyaHilangDariAntrean(t *testing.T) {
	store, writer := penulis(t)
	ctx := context.Background()

	sebelum, err := store.Count(ctx, kueri(t, inboxbandinghargasalvage.TabRequest, memory.SampleOwner, ""))
	require.NoError(t, err)

	result, err := writer.Decide(ctx,
		putusan(t, memory.SampleOwner, "PNC-0451/1", "451", true))
	require.NoError(t, err)
	require.True(t, result.Recorded)

	sesudah, err := store.Count(ctx, kueri(t, inboxbandinghargasalvage.TabRequest, memory.SampleOwner, ""))
	require.NoError(t, err)
	require.Equal(t, sebelum-1, sesudah,
		"barisnya hilang dari antrean karena TGLAPPROVE sudah terisi")
}

// Penekanan tombol kedua tidak menimpa putusan pertama — itulah gunanya penyaring
// `TGLAPPROVE IS NULL`, dan itulah yang dibaca pengguna sebagai jawaban atas klik ganda.
func TestPenekananKeduaTidakMengubahApaPun(t *testing.T) {
	_, writer := penulis(t)
	ctx := context.Background()
	command := putusan(t, memory.SampleOwner, "PNC-0451/1", "451", true)

	_, err := writer.Decide(ctx, command)
	require.NoError(t, err)

	_, err = writer.Decide(ctx, command)
	require.ErrorIs(t, err, inboxbandinghargasalvage.ErrAlreadyDecided)
}

func TestBarisMilikKomiteLainTidakDapatDiputus(t *testing.T) {
	_, writer := penulis(t)

	// PNC-0456/1 milik komite lain.
	_, err := writer.Decide(context.Background(),
		putusan(t, memory.SampleOwner, "PNC-0456/1", "456", true))

	require.ErrorIs(t, err, inboxbandinghargasalvage.ErrAlreadyDecided,
		"kepemilikan dan 'sudah diputus' sengaja dijawab satu galat yang sama")
}

// Bagi komite biasa, menyetujui HANYA mencatat putusan — harganya tidak berubah. Ini perilaku
// layar lama apa adanya; lihat PlanDecision.
func TestKomiteBiasaMenyetujuiTanpaMengubahHarga(t *testing.T) {
	store, writer := penulis(t)
	ctx := context.Background()

	result, err := writer.Decide(ctx,
		putusan(t, memory.SampleOwner, "PNC-0451/1", "451", true))
	require.NoError(t, err)
	require.False(t, result.PriceApplied)
	require.False(t, result.DocumentMarked)

	// Harga barangnya tetap seperti semula pada riwayat putusan.
	decisions, err := store.ListDecisions(ctx,
		riwayat(t, "PNCN.26.0451", memory.SampleOwner))
	require.NoError(t, err)
	require.Len(t, decisions, 1)
	require.Equal(t, "12500000.00", decisions[0].ItemPrice,
		"harga barang tidak tersentuh oleh persetujuan komite biasa")
}

// Jenjang TERAKHIR yang menyetujui-lah yang menimpa harga barang.
func TestJenjangTerakhirMenyetujuiMenimpaHargaBarang(t *testing.T) {
	store, writer := penulis(t)
	ctx := context.Background()

	result, err := writer.Decide(ctx,
		putusan(t, "DANIELLISWANDI", "PNC-0461/1", "461", true))
	require.NoError(t, err)
	require.True(t, result.PriceApplied)

	decisions, err := store.ListDecisions(ctx,
		riwayat(t, "PNCN.26.0461", "DANIELLISWANDI"))
	require.NoError(t, err)
	require.Len(t, decisions, 1)
	require.Equal(t, "8200000.00", decisions[0].ItemPrice,
		"harga tandingan balai lelang menggantikan harga barang")
}

// Penolakan jenjang pertama menutup jenjang berikutnya sekaligus — tidak ada gunanya
// menanyakan persetujuan atas harga yang sudah ditolak.
func TestPenolakanJenjangPertamaIkutMenutupJenjangBerikutnya(t *testing.T) {
	store, writer := penulis(t)
	ctx := context.Background()

	result, err := writer.Decide(ctx,
		putusan(t, "BAMBANGSETIADJIGUNAWAN", "PNC-0460/1", "460", false))
	require.NoError(t, err)
	require.True(t, result.DocumentMarked)
	require.False(t, result.PriceApplied)

	// Baris komite berikutnya atas barang yang sama ikut tertutup, dengan status yang sama.
	decisions, err := store.ListDecisions(ctx,
		riwayat(t, "PNCN.26.0460", "DANIELLISWANDI"))
	require.NoError(t, err)
	require.Len(t, decisions, 1)
	require.Equal(t, inboxbandinghargasalvage.DecisionRejected, decisions[0].Status)

	// Dan ia tidak dapat memutuskannya lagi.
	_, err = writer.Decide(ctx,
		putusan(t, "DANIELLISWANDI", "PNC-0460/1", "460", true))
	require.ErrorIs(t, err, inboxbandinghargasalvage.ErrAlreadyDecided)
}

// Penolakan oleh komite biasa TIDAK merembet ke siapa pun.
func TestPenolakanKomiteBiasaTidakMerembet(t *testing.T) {
	_, writer := penulis(t)

	result, err := writer.Decide(context.Background(),
		putusan(t, memory.SampleOwner, "PNC-0451/1", "451", false))

	require.NoError(t, err)
	require.True(t, result.DocumentMarked, "penandaan dokumen berlaku bagi siapa pun yang menolak")
	require.False(t, result.PriceApplied)
}

// Catatan komite tersimpan dan terbaca kembali di panel riwayat.
func TestCatatanKomiteTerbacaKembaliDiRiwayat(t *testing.T) {
	store, writer := penulis(t)
	ctx := context.Background()

	command := putusan(t, memory.SampleOwner, "PNC-0451/1", "451", false)
	_, err := writer.Decide(ctx, command)
	require.NoError(t, err)

	decisions, err := store.ListDecisions(ctx, riwayat(t, "PNCN.26.0451", memory.SampleOwner))
	require.NoError(t, err)
	require.Len(t, decisions, 1)
	require.Equal(t, "Tidak setuju", inboxbandinghargasalvage.DecisionLabel(decisions[0].Status))
	require.NotNil(t, decisions[0].ApprovedAt, "tanggal putusan ikut terisi, bukan dibiarkan kosong")
}

// riwayat menyusun kueri panel rincian.
func riwayat(t *testing.T, noKlaim, login string) inboxbandinghargasalvage.DecisionQuery {
	t.Helper()

	query, err := inboxbandinghargasalvage.NewDecisionQuery(
		noKlaim, inboxbandinghargasalvage.Caller{Login: login})
	require.NoError(t, err)
	return query
}

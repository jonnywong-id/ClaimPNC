package reportklaim

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// tanggaContoh meniru bentuk `POOLDATA.GCNM_FEE_SCALE`: indeks berurutan, LOSS_AMOUNT dan
// FEE menaik. Angkanya dibuat bulat supaya interpolasinya dapat dihitung di kepala.
func tanggaContoh() *FeeScale {
	return NewFeeScale([]FeeBand{
		{Index: 1, LossAmount: 100, Fee: 10},
		{Index: 2, LossAmount: 200, Fee: 30},
		{Index: 3, LossAmount: 400, Fee: 70},
	})
}

// Nilai yang masih di dalam pita PERTAMA menghasilkan nilai tetap, bukan hasil
// interpolasi. Ini perilaku `ehasil := 1650000` pada fungsi aslinya.
func TestFeeDiDalamPitaPertamaMemakaiNilaiTetap(t *testing.T) {
	fee, ok := tanggaContoh().Fee(50)

	require.True(t, ok)
	require.Equal(t, FeeBeforeFirstBand, fee)
}

// Tepat di batas atas pita pertama masih pita pertama: penyaringnya `LOSS_AMOUNT >= nilai`,
// bukan `>`.
func TestFeeTepatDiBatasPitaPertamaMasihNilaiTetap(t *testing.T) {
	fee, ok := tanggaContoh().Fee(100)

	require.True(t, ok)
	require.Equal(t, FeeBeforeFirstBand, fee)
}

// Interpolasi linear di antara dua pita.
//
//	150 berada di tengah 100..200, jadi fee = 10 + 0,5 × (30 − 10) = 20
func TestFeeDiAntaraDuaPitaDiinterpolasi(t *testing.T) {
	fee, ok := tanggaContoh().Fee(150)

	require.True(t, ok)
	require.InDelta(t, 20.0, fee, 1e-9)
}

// Tepat di batas atas sebuah pita menghasilkan fee pita itu sendiri — bukti interpolasinya
// bersambung di ujung, bukan melompat.
func TestFeeTepatDiBatasPitaMenghasilkanFeePitaItu(t *testing.T) {
	fee, ok := tanggaContoh().Fee(200)

	require.True(t, ok)
	require.InDelta(t, 30.0, fee, 1e-9)
}

// Di atas seluruh pita: fee pita PANGKAL ditambah 2% dari kelebihannya.
//
// Tangga ini sengaja memberi nomor TopBandIndex pada pita teratasnya, karena pangkalnya
// memang dicari menurut NOMOR, bukan menurut urutan.
//
//	500 melampaui 400, jadi fee = 70 + (500 − 400) × 0,02 = 72
func TestFeeDiAtasSeluruhPitaDiekstrapolasi(t *testing.T) {
	tangga := NewFeeScale([]FeeBand{
		{Index: TopBandIndex - 2, LossAmount: 100, Fee: 10},
		{Index: TopBandIndex - 1, LossAmount: 200, Fee: 30},
		{Index: TopBandIndex, LossAmount: 400, Fee: 70},
	})

	fee, ok := tangga.Fee(500)

	require.True(t, ok)
	require.InDelta(t, 72.0, fee, 1e-9)
}

// Pangkal ekstrapolasi adalah pita bernomor TopBandIndex, PERSIS seperti sistem lama —
// bukan pita ber-indeks terbesar.
//
// Uji ini menahan kesetaraan, bukan kebenaran. Master yang bertambah menjadi 18 pita akan
// tetap berpangkal pada pita ke-17, dan itu memang yang dilakukan `WHERE INDEX_FEE = 17`
// pada fungsi aslinya. Memperbaikinya adalah keputusan Work Owner, bukan keputusan modul.
func TestEkstrapolasiMemakaiPitaBernomorTopBandIndex(t *testing.T) {
	bands := []FeeBand{{Index: 1, LossAmount: 100, Fee: 10}}
	for i := 2; i <= TopBandIndex; i++ {
		bands = append(bands, FeeBand{Index: i, LossAmount: float64(i) * 100, Fee: float64(i) * 10})
	}
	// Satu pita DI ATAS pangkal — nilainya sengaja jauh berbeda supaya tertukar terlihat.
	bands = append(bands, FeeBand{Index: TopBandIndex + 1, LossAmount: 9000, Fee: 9000})

	fee, ok := NewFeeScale(bands).Fee(10000)

	require.True(t, ok)
	// Pita 17: LossAmount 1700, Fee 170 → 170 + (10000 − 1700) × 0,02 = 336
	require.InDelta(t, 336.0, fee, 1e-9, "pangkalnya pita 17, bukan pita 18")
}

// Tanpa pita bernomor TopBandIndex, ekstrapolasi tidak dapat dilakukan — sistem lama
// melempar NO_DATA_FOUND dan menggagalkan seluruh kueri.
func TestEkstrapolasiTanpaPitaPangkalTidakDapatDihitung(t *testing.T) {
	_, ok := tanggaContoh().Fee(10000)
	require.False(t, ok, "tangga contoh hanya punya tiga pita, tidak ada pita 17")
}

// Tangga yang sumbernya tidak terbaca TIDAK menjawab nol. Nol pada kolom fee terbaca
// sebagai "adjuster tidak dibayar", dan itu angka yang salah, bukan angka yang hilang.
func TestTanggaTidakTersediaTidakMenjawabNol(t *testing.T) {
	_, ok := UnavailableFeeScale().Fee(150)
	require.False(t, ok)

	_, ok = NewFeeScale(nil).Fee(150)
	require.False(t, ok, "daftar kosong diperlakukan sebagai tidak tersedia")
}

// Pita indeks sebelumnya yang hilang membuat satu SEL kosong, bukan menjatuhkan laporan.
// Sistem lama melempar NO_DATA_FOUND dan menggagalkan seluruh kueri.
func TestPitaSebelumnyaHilangMengosongkanSatuSelSaja(t *testing.T) {
	tangga := NewFeeScale([]FeeBand{
		{Index: 1, LossAmount: 100, Fee: 10},
		{Index: 3, LossAmount: 400, Fee: 70}, // indeks 2 sengaja tidak ada
	})

	_, ok := tangga.Fee(300)
	require.False(t, ok)

	fee, ok := tangga.Fee(50)
	require.True(t, ok, "pita lain tetap dapat dijawab")
	require.Equal(t, FeeBeforeFirstBand, fee)
}

// Dua pita ber-LOSS_AMOUNT sama TIDAK pernah sampai ke pembagian.
//
// Pemilihannya mengambil INDEKS TERKECIL di antara pita yang menampung nilai. Bila pita
// `k-1` punya batas yang sama dengan pita `k`, ia ikut menampung nilai yang sama — maka
// yang terpilih adalah `k-1`, dan pasangan berimpit itu tidak pernah menjadi pembagi.
//
// Uji ini mengunci penalarannya, bukan menutupi cacat: penjaga `span == 0` di dalam Fee
// memang tidak dapat dicapai, dan dibiarkan berdiri hanya sebagai jaring bila aturan
// pemilihannya kelak berubah.
func TestDuaPitaBerimpitDiserapPemilihanIndeksTerkecil(t *testing.T) {
	tangga := NewFeeScale([]FeeBand{
		{Index: 1, LossAmount: 100, Fee: 10},
		{Index: 2, LossAmount: 100, Fee: 30},
	})

	fee, ok := tangga.Fee(100)

	require.True(t, ok)
	require.Equal(t, FeeBeforeFirstBand, fee, "yang terpilih pita 1, sehingga nilai tetap yang berlaku")
}

// Urutan indeks tidak harus sejalan dengan urutan LOSS_AMOUNT — tidak ada apa pun di
// master yang menjaminnya, dan penelusurannya memang tidak mengandaikannya.
func TestIndeksTidakHarusSejalanDenganUrutanNilai(t *testing.T) {
	tangga := NewFeeScale([]FeeBand{
		{Index: 3, LossAmount: 400, Fee: 70},
		{Index: 1, LossAmount: 100, Fee: 10},
		{Index: 2, LossAmount: 200, Fee: 30},
	})

	fee, ok := tangga.Fee(150)

	require.True(t, ok)
	require.InDelta(t, 20.0, fee, 1e-9)
}

func TestNilaiNegatifTidakDihitung(t *testing.T) {
	_, ok := tanggaContoh().Fee(-1)
	require.False(t, ok)
}

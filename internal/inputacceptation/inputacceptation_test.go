package inputacceptation_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inputacceptation"
)

var caller = inputacceptation.Caller{Login: "ADMINNONPROP1"}

func mustQuery(t *testing.T) inputacceptation.Query {
	t.Helper()
	q, err := inputacceptation.NewQuery("CLMNP-1001", caller)
	require.NoError(t, err)
	return q
}

// Pemanggil tanpa login DITOLAK, dan ditolak sebagai galat tersendiri.
//
// Setiap pembukaan layar ini dicatat beserta pelakunya — itu satu-satunya jejak yang ada
// sampai pemeriksaan kewenangan menu dibangun. Melayani permintaan tanpa identitas berarti
// membuat jejak itu bolong.
func TestPemanggilTanpaLoginDitolak(t *testing.T) {
	_, err := inputacceptation.NewQuery("CLMNP-1001", inputacceptation.Caller{})
	require.ErrorIs(t, err, inputacceptation.ErrCallerUnknown)

	_, err = inputacceptation.NewQuery("CLMNP-1001", inputacceptation.Caller{Login: "   "})
	require.ErrorIs(t, err, inputacceptation.ErrCallerUnknown)
}

// Hanya nomor klaim BERAWALAN `CLMNP-` yang dilayani.
//
// `PC_ASM_FW_GCNMFW_WORK` memuat objek kerja seluruh jenis klaim. Tanpa penyaring ini, alamat
// layar dapat diisi nomor klaim PNC biasa dan layar akan menggambarnya dengan susunan
// akseptasi treaty — ~50 isian yang hampir seluruhnya kosong, tanpa satu pun tanda bahwa yang
// dibuka adalah jenis klaim yang berbeda.
//
// `CLMP-` milik layar Prop sengaja ikut diuji: ketiga huruf pertamanya sama, dan penyaring
// yang longgar akan meloloskannya.
func TestHanyaNomorKlaimNonPropDilayani(t *testing.T) {
	for _, nomor := range []string{"CLMNP-1001", "clmnp-1001"} {
		q, err := inputacceptation.NewQuery(nomor, caller)
		require.NoErrorf(t, err, "nomor %q seharusnya diterima", nomor)
		require.Equal(t, nomor, q.ClaimID, "nomor klaim dibawa apa adanya")
	}

	for _, nomor := range []string{"CLMP-70", "PNC-1865", "PNCN.26.8125", "KMTNP-3001"} {
		_, err := inputacceptation.NewQuery(nomor, caller)

		var validation *inputacceptation.ValidationError
		require.ErrorAsf(t, err, &validation, "nomor %q seharusnya ditolak", nomor)
		require.Equal(t, inputacceptation.FieldClaimID, validation.Violations[0].Field)
	}
}

func TestNomorKlaimKosongDitolak(t *testing.T) {
	_, err := inputacceptation.NewQuery("   ", caller)

	var validation *inputacceptation.ValidationError
	require.ErrorAs(t, err, &validation)
}

// NewDetail MENOLAK kunci yang tidak dikenal katalog.
//
// Kunci yang tidak dikenal bukan sekadar kelebihan: ia isian yang tidak akan pernah digambar,
// sehingga keberadaannya di jawaban API hanya menyesatkan pembacanya.
func TestNewDetailMenolakKunciAsing(t *testing.T) {
	_, err := inputacceptation.NewDetail("CLMNP-1001", "REF", "New", "ADMIN",
		map[string]string{"isian_karangan": "x"}, nil)

	var unknown *inputacceptation.UnknownFieldError
	require.ErrorAs(t, err, &unknown)
	require.Equal(t, "isian_karangan", unknown.Field)

	_, err = inputacceptation.NewDetail("CLMNP-1001", "REF", "New", "ADMIN", nil,
		map[string][]inputacceptation.GridRow{"tabel_karangan": {}})

	var unknownGrid *inputacceptation.UnknownGridError
	require.ErrorAs(t, err, &unknownGrid)
	require.Equal(t, "tabel_karangan", unknownGrid.Grid)
}

// Grid yang sumbernya nil menjadi senarai KOSONG, bukan tetap nil.
//
// `[]` dan `null` ditangani berbeda oleh klien, dan yang kedua memaksa setiap layar
// memeriksanya lebih dulu.
func TestGridNilMenjadiSenaraiKosong(t *testing.T) {
	detail, err := inputacceptation.NewDetail("CLMNP-1001", "REF", "New", "ADMIN", nil,
		map[string][]inputacceptation.GridRow{inputacceptation.GridAdjustment: nil})
	require.NoError(t, err)

	rows := detail.Rows(inputacceptation.GridAdjustment)
	require.NotNil(t, rows)
	require.Empty(t, rows)
}

// Isian yang hanya DITAMPILKAN tidak dapat dikirim balik.
//
// Mengabaikannya diam-diam membuat pengguna mengira perubahannya tersimpan, dan pada layar
// yang menetapkan nilai akseptasi itu kekeliruan bernilai uang.
func TestSubmitMenolakIsianYangHanyaDitampilkan(t *testing.T) {
	_, err := inputacceptation.NewSaveCommand(mustQuery(t), "REF",
		map[string]string{"treaty_id": "TNP-DIUBAH"}, nil)

	var validation *inputacceptation.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Len(t, validation.Violations, 1)
	require.Equal(t, "treaty_id", validation.Violations[0].Field)
	require.Contains(t, validation.Violations[0].Message, "tidak dapat diubah")
}

func TestSubmitMenolakIsianYangTidakAdaDiLayar(t *testing.T) {
	_, err := inputacceptation.NewSaveCommand(mustQuery(t), "REF",
		map[string]string{"isian_karangan": "x"}, nil)

	var validation *inputacceptation.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Contains(t, validation.Violations[0].Message, "tidak ada di layar")
}

// Isian terhalang TIDAK dapat dikirim balik, meski ia digambar.
//
// Isian yang tidak dapat dibaca tetapi dapat ditulis adalah isian yang menulis ke tempat yang
// tidak diketahui.
func TestSubmitMenolakIsianTerhalang(t *testing.T) {
	_, err := inputacceptation.NewSaveCommand(mustQuery(t), "REF",
		map[string]string{"ceding_name": "Diisi Sendiri"}, nil)

	var validation *inputacceptation.ValidationError
	require.ErrorAs(t, err, &validation)
}

func TestSubmitMenerimaIsianYangDapatDiubah(t *testing.T) {
	cmd, err := inputacceptation.NewSaveCommand(mustQuery(t), "REF",
		map[string]string{"dla_no_ceding": "DLA/2026/0002"}, nil)
	require.NoError(t, err)

	require.Equal(t, "DLA/2026/0002", cmd.Values["dla_no_ceding"])
	require.Equal(t, "REF", cmd.Reference)
	require.Equal(t, "CLMNP-1001", cmd.Query.ClaimID)
}

// Kolom grid yang hanya DITAMPILKAN tidak dapat dikirim balik.
//
// Kolom read-only pada grid adalah hasil hitungan server — alokasi kerugian, premi
// reinstatement, nilai akseptasi. Menerimanya dari klien berarti membiarkan klien menimpanya.
func TestSubmitMenolakKolomGridYangHanyaDitampilkan(t *testing.T) {
	_, err := inputacceptation.NewSaveCommand(mustQuery(t), "REF", nil,
		map[string][]inputacceptation.GridRow{
			inputacceptation.GridAdjustment: {
				{"acceptation_no": "AKS/KARANGAN/0001"},
			},
		})

	var validation *inputacceptation.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Equal(t,
		inputacceptation.GridAdjustment+".acceptation_no",
		validation.Violations[0].Field)
	require.Contains(t, validation.Violations[0].Message, "baris 1")
}

func TestSubmitMenerimaKolomGridYangDapatDiubah(t *testing.T) {
	cmd, err := inputacceptation.NewSaveCommand(mustQuery(t), "REF", nil,
		map[string][]inputacceptation.GridRow{
			inputacceptation.GridAdjustment: {{"type": "Interim"}},
		})
	require.NoError(t, err)
	require.Equal(t, "Interim", cmd.Grids[inputacceptation.GridAdjustment][0]["type"])
}

// SELURUH pelanggaran dikumpulkan, bukan yang pertama saja.
//
// Layar ini mengirim ~11 isian dan lima grid dalam satu Submit. Mengembalikan pelanggaran satu
// per satu memaksa pengguna menekan Submit berkali-kali untuk menemukan kesalahan berikutnya.
func TestSubmitMengumpulkanSeluruhPelanggaran(t *testing.T) {
	_, err := inputacceptation.NewSaveCommand(mustQuery(t), "REF",
		map[string]string{
			"treaty_id":      "x",
			"isian_karangan": "y",
		},
		map[string][]inputacceptation.GridRow{
			inputacceptation.GridAdjustment: {{"status": "9"}},
		})

	var validation *inputacceptation.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Len(t, validation.Violations, 3)
}

// Galat ErrWriteNotOwned dapat dikenali pemanggil lewat errors.Is.
//
// Lapisan transport membedakannya dari galat lain untuk menjawab 409 beserta alasannya, bukan
// 500 tanpa keterangan.
func TestGalatKepemilikanDapatDikenali(t *testing.T) {
	wrapped := errors.Join(errors.New("konteks"), inputacceptation.ErrWriteNotOwned)
	require.ErrorIs(t, wrapped, inputacceptation.ErrWriteNotOwned)
}

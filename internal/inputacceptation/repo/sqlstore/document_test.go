package sqlstore

import (
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inputacceptation"
)

// Dokumen kosong BUKAN galat.
//
// Gabungan ke JSON_KLAIM adalah LEFT JOIN, sehingga klaim yang belum punya baris di sana
// mengembalikan NULL — dan klaim seperti itu tetap dapat dibuka, hanya isinya yang kosong.
func TestDokumenKosongBukanGalat(t *testing.T) {
	doc, err := parseDocument("")
	require.NoError(t, err)
	require.Empty(t, doc)

	doc, err = parseDocument("   ")
	require.NoError(t, err)
	require.Empty(t, doc)
}

// Dokumen yang ADA tetapi rusak JUSTRU galat.
//
// Menelannya menjadi rincian kosong berarti kerusakan data tersaji kepada pengguna sebagai
// "klaim ini memang belum diisi".
func TestDokumenRusakMenghasilkanGalat(t *testing.T) {
	_, err := parseDocument(`{"NoClaim": `)
	require.Error(t, err)
	require.Contains(t, err.Error(), "mengurai dokumen klaim")
}

// Angka dibaca APA ADANYA, bukan lewat float64.
//
// Hampir setiap angka di layar ini adalah nilai uang atau persentase share reasuransi, dan
// float64 membulatkannya tanpa satu pun galat. `I-12` menetapkan nilai uang disimpan presisi
// penuh dan hanya dibulatkan saat ditampilkan.
func TestAngkaBesarTidakDibulatkan(t *testing.T) {
	doc, err := parseDocument(`{"TotalSumInsuredIDR": 1234567890123.45}`)
	require.NoError(t, err)

	value, exists := doc.lookup("TotalSumInsuredIDR")
	require.True(t, exists)
	require.Equal(t, "1234567890123.45", text(value))
}

// "Tidak ada" dan "ada tetapi kosong" adalah dua keadaan yang BERBEDA.
//
// Yang pertama menunjuk jalur yang salah, yang kedua menunjuk data yang belum diisi. Tanpa
// pembedaan itu, jalur dokumen yang salah tidak punya satu pun tanda.
func TestJalurTidakAdaDibedakanDariNilaiKosong(t *testing.T) {
	doc, err := parseDocument(`{"NoClaim": "", "PolicyData": {"PolicyNo": "99.002"}}`)
	require.NoError(t, err)

	value, exists := doc.lookup("NoClaim")
	require.True(t, exists, "isian yang ada tetapi kosong tetap ADA")
	require.Equal(t, "", text(value))

	_, exists = doc.lookup("IsianYangTidakPernahAda")
	require.False(t, exists)

	// Jalur bertingkat ditelusuri titik demi titik.
	value, exists = doc.lookup("PolicyData.PolicyNo")
	require.True(t, exists)
	require.Equal(t, "99.002", text(value))
}

// Senarai yang berisi SATU objek dapat ditulis Pega sebagai objek tunggal.
//
// Membuangnya membuat grid tampak kosong padahal berisi. Bentuk dokumen ini belum dapat
// diperiksa (`R-08`), jadi keduanya diterima.
func TestObjekTunggalDibacaSebagaiSatuBaris(t *testing.T) {
	columns := []inputacceptation.GridColumn{
		{Key: "type", Path: "Type"},
		{Key: "acceptation_no", Path: "AcceptedNo"},
	}

	doc, err := parseDocument(`{"AdjustmentList": {"Type": "Final", "AcceptedNo": "A-1"}}`)
	require.NoError(t, err)

	value, exists := doc.lookup("AdjustmentList")
	require.True(t, exists)

	got := rows(value, columns)
	require.Len(t, got, 1)
	require.Equal(t, "Final", got[0]["type"])
	require.Equal(t, "A-1", got[0]["acceptation_no"])
}

// Seluruh isian dan grid katalog dipetik, dan yang tidak ditemukan DIHITUNG.
//
// Hitungan itu satu-satunya tanda yang dimiliki jalur dokumen yang salah: layar berisi ~50
// isian kosong terbaca sama persis, entah karena klaimnya memang belum diisi atau karena
// seluruh jalurnya salah.
func TestReadDocumentMenghitungJalurYangTidakDitemukan(t *testing.T) {
	doc, err := parseDocument(`{
		"NoClaim": "CLMNP-1001",
		"IDMaster": "TNP-2026-01",
		"AdjustmentList": [{"Type": "Final", "AcceptedNo": "A-1", "AcceptanceStatus": "1"}]
	}`)
	require.NoError(t, err)

	values, gridRows, stats := readDocument(doc)

	require.Equal(t, "CLMNP-1001", values["claim_no"])
	require.Equal(t, "TNP-2026-01", values["treaty_id"])
	require.Equal(t, 2, stats.FieldsFound)
	require.Positive(t, stats.FieldsMissing, "isian yang tidak ada wajib terhitung")

	require.Equal(t, 1, stats.GridsFound)
	require.Positive(t, stats.GridsMissing)
	require.Len(t, gridRows[inputacceptation.GridAdjustment], 1)
	require.Equal(t, "Final", gridRows[inputacceptation.GridAdjustment][0]["type"])
}

// Isian yang TERHALANG tidak pernah dipetik, dan tidak pernah dihitung sebagai hilang.
//
// Ia memang tidak punya jalur; menghitungnya sebagai hilang akan menaikkan angka "tidak
// ditemukan" secara tetap dan menutupi jalur yang benar-benar salah.
func TestIsianTerhalangTidakDihitungSebagaiHilang(t *testing.T) {
	doc, err := parseDocument(`{}`)
	require.NoError(t, err)

	values, _, stats := readDocument(doc)

	for _, field := range inputacceptation.Fields() {
		if !field.Blocked {
			continue
		}
		_, present := values[field.Key]
		require.Falsef(t, present, "isian terhalang %q ikut dipetik", field.Key)
	}

	dapatDibaca := 0
	for _, field := range inputacceptation.Fields() {
		if !field.Blocked {
			dapatDibaca++
		}
	}
	require.Equal(t, dapatDibaca, stats.FieldsMissing)
}

// Hasil readDocument selalu diterima NewDetail.
//
// Keduanya berdiri di atas katalog yang sama, dan uji ini yang menangkap saat salah satunya
// bergeser — sebelum pengguna membuka layar dan menemukan galat internal.
func TestHasilPembacaanSelaluDiterimaNewDetail(t *testing.T) {
	doc, err := parseDocument(`{"NoClaim": "CLMNP-1001", "AdjustmentList": []}`)
	require.NoError(t, err)

	values, gridRows, _ := readDocument(doc)

	_, err = inputacceptation.NewDetail(
		"CLMNP-1001", "REF", "New", "ADMIN", values, gridRows)
	require.NoError(t, err)
}

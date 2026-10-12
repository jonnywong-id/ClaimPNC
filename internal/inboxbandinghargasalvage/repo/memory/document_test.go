package memory_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxbandinghargasalvage"
	"claim-pnc/internal/inboxbandinghargasalvage/repo/memory"
)

// Uji di berkas ini memeriksa dialog "Lihat File".
//
// Yang paling dijaga bukan isinya melainkan PENYARING KEPEMILIKAN — penyaring yang tidak ada
// di sistem lama dan sengaja ditambahkan. Tanpa uji, penambahan semacam itu mudah hilang saat
// kuerinya kelak disunting.

func dokumen(t *testing.T) (*memory.Store, *memory.DocumentStore) {
	t.Helper()

	store := memory.NewSampleStore()
	return store, memory.NewSampleDocumentStore(store)
}

func permintaanDokumen(
	t *testing.T, login, detail, salvage string,
) inboxbandinghargasalvage.DocumentQuery {
	t.Helper()

	q, err := inboxbandinghargasalvage.NewDocumentQuery(
		detail, salvage, inboxbandinghargasalvage.Caller{Login: login})
	require.NoError(t, err)
	return q
}

func TestDaftarDokumenMenampilkanYangBelumDitandaiDitolak(t *testing.T) {
	_, dok := dokumen(t)

	rows, err := dok.ListDocuments(context.Background(),
		permintaanDokumen(t, memory.SampleOwner, "PNC-0451/1", "451"))

	require.NoError(t, err)
	require.Len(t, rows, 2, "dokumen yang sudah ditandai ditolak tidak ikut")

	nama := []string{rows[0].Name, rows[1].Name}
	require.NotContains(t, nama, "banding-sebelumnya.pdf")
}

// Terbaru di atas — sama seperti `ORDER BY TGLINS DESC`.
func TestDaftarDokumenTerbaruDiAtas(t *testing.T) {
	_, dok := dokumen(t)

	rows, err := dok.ListDocuments(context.Background(),
		permintaanDokumen(t, memory.SampleOwner, "PNC-0451/1", "451"))

	require.NoError(t, err)
	require.Equal(t, "penawaran-balai-lelang.pdf", rows[0].Name)
}

// Kategori adalah KONSTANTA, bukan kolom basis data. Uji ini menjaganya tetap begitu: bila
// kelak ada yang mengisinya dari tabel, nilainya akan berubah dan uji ini menyatakannya.
func TestKategoriDokumenSelaluKonstanta(t *testing.T) {
	_, dok := dokumen(t)

	rows, err := dok.ListDocuments(context.Background(),
		permintaanDokumen(t, memory.SampleOwner, "PNC-0451/1", "451"))
	require.NoError(t, err)

	for _, row := range rows {
		require.Equal(t, "BandingHarga", row.Category())
	}
}

/*
Penyaring kepemilikan — yang TIDAK ada di sistem lama.

Kueri lama hanya menyaring kedua id, dan keduanya dikirim pemanggil. Siapa pun yang tahu
sepasang id karena itu dapat membaca daftar dokumen banding komite lain. Kedua uji di
bawah menjaga penambahan itu tetap ada.
*/
func TestDaftarDokumenKomiteLainTidakTerbaca(t *testing.T) {
	_, dok := dokumen(t)

	rows, err := dok.ListDocuments(context.Background(),
		permintaanDokumen(t, memory.SampleOwner, "PNC-0456/1", "456"))

	require.NoError(t, err)
	require.Empty(t, rows, "banding itu ditangani komite lain")
}

func TestIsiDokumenKomiteLainDitolak(t *testing.T) {
	_, dok := dokumen(t)

	_, err := dok.DocumentContent(context.Background(), "9004",
		permintaanDokumen(t, memory.SampleOwner, "PNC-0456/1", "456"))

	require.ErrorIs(t, err, inboxbandinghargasalvage.ErrDocumentNotFound)
}

// Menebak id dokumen milik banding lain TIDAK cukup: id-nya harus cocok dengan banding yang
// disebut permintaan, dan banding itu harus milik komite pemanggil.
func TestIsiDokumenTidakDapatDiambilDenganMenebakId(t *testing.T) {
	_, dok := dokumen(t)

	// Id 9004 nyata, tetapi ia milik banding lain — disebut di sini bersama banding yang
	// memang milik pemanggil.
	_, err := dok.DocumentContent(context.Background(), "9004",
		permintaanDokumen(t, memory.SampleOwner, "PNC-0451/1", "451"))

	require.ErrorIs(t, err, inboxbandinghargasalvage.ErrDocumentNotFound)
}

func TestIsiDokumenDiserahkanBesertaTipenya(t *testing.T) {
	_, dok := dokumen(t)

	isi, err := dok.DocumentContent(context.Background(), "9001",
		permintaanDokumen(t, memory.SampleOwner, "PNC-0451/1", "451"))

	require.NoError(t, err)
	require.Equal(t, "penawaran-balai-lelang.pdf", isi.Name)
	require.Equal(t, "application/pdf", isi.MIMEType)
	require.NotEmpty(t, isi.Content)
}

func TestDokumenTanpaIdentitasDitolak(t *testing.T) {
	_, err := inboxbandinghargasalvage.NewDocumentQuery(
		"PNC-0451/1", "451", inboxbandinghargasalvage.Caller{})

	require.ErrorIs(t, err, inboxbandinghargasalvage.ErrCallerUnknown)
}

func TestDokumenTanpaKeduaIdDitolakDenganSeluruhPelanggarannya(t *testing.T) {
	_, err := inboxbandinghargasalvage.NewDocumentQuery(
		"", "  ", inboxbandinghargasalvage.Caller{Login: memory.SampleOwner})

	var validation *inboxbandinghargasalvage.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Len(t, validation.Violations, 2)
}

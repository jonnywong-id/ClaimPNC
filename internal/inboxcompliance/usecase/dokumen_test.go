package usecase_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxcompliance"
)

/*
Tiga uji DICABUT 2026-10-08 bersama grid dokumennya.

`TestDokumenKlaimTerbaca`, `TestDokumenKosongBukanGalat`, dan
`TestDokumenDibacaUntukSemuaLini` seluruhnya memeriksa pembacaan dokumen SAAT FORM
DIBUKA — perilaku yang tidak ada di Pega. Gridnya memang ada di
`Section/CompliancePNC-Section.xml`, tetapi sumbernya `TempDocumentAttach` berkelas
`Code-Pega-List`: halaman klipboard yang hanya diisi saat menyimpan, bukan dimuat dari
basis data saat form dibuka.

Yang di bawah ini DIPERTAHANKAN, karena perilakunya tetap dipakai: penggantian surat
penolakan mencari dokumen berdasarkan nama, dan DeleteDocument memeriksa kepemilikannya.
Keduanya menempuh `FindDocuments` yang sama.
*/

// Baris TANPA kunci penyimpanan dibuang — meniru `IMAGEID IS NOT NULL` pada kuerinya.
//
// Baris begitu adalah unggahan yang belum selesai: berkasnya tidak dapat dibuka. Bila ia
// ikut terbawa, penggantian surat penolakan dapat menghapus baris yang berkasnya tidak
// pernah ada, dan DeleteDocument akan mencoba membuang berkas yatim.
func TestDokumenTanpaKunciPenyimpananDibuang(t *testing.T) {
	t.Parallel()

	store := paStore()
	store.SeedDocuments(claimKey,
		inboxcompliance.Document{ID: "1", Name: "selesai.pdf", StorageID: "IMG-1"},
		inboxcompliance.Document{ID: "2", Name: "belum-selesai.pdf"}, // tanpa StorageID
	)

	dokumen, err := store.FindDocuments(context.Background(), claimKey)
	require.NoError(t, err)
	require.Len(t, dokumen, 1, "baris tanpa IMAGEID tidak boleh ikut terbawa")
	require.Equal(t, "selesai.pdf", dokumen[0].Name)
}

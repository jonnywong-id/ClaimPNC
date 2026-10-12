package sqlstore

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Dokumen klaim Pega: PDFType dan PrintDateLOD per adjustment, berkunci seperti
// PEGA_CONVERT_JSONKLAIM_PNC.prc — ObjectID, CoverageID, pxListSubscript atau urutan + 1.
// Nilai KARANGAN (D-69).
func TestDokumenKlaimMemberiTipePDFDanTanggalCetakLOD(t *testing.T) {
	body := []byte(`{"ObjectList":[{"ObjectID":"OBJ1","ObjectCoverageList":[{"CoverageID":"1","AdjustmentList":[
		{"PDFType":"15","PrintDateLOD":"20260930T005900.000 GMT"},
		{"pxListSubscript":"2","PDFType":3}]}]}]}`)
	facts, ok := parseLODDocument(body)
	require.True(t, ok)
	require.Equal(t, "15", facts[lodKey("OBJ1", "1", "1")].lodType)
	require.Equal(t, time.Date(2026, 9, 30, 0, 59, 0, 0, time.UTC), facts[lodKey("OBJ1", "1", "1")].printedAt.UTC())
	require.Equal(t, "3", facts[lodKey("OBJ1", "1", "2")].lodType, "angka JSON dibaca sebagai teks")
	require.True(t, facts[lodKey("OBJ1", "1", "2")].printedAt.IsZero())

	_, ok = parseLODDocument([]byte("bukan json"))
	require.False(t, ok)

	// Kolom DATE hasil konversi hanya tanggal (tengah malam WIB) — dilengkapi dari dokumen.
	wib := time.FixedZone("WIB", 7*3600)
	require.True(t, dateOnlyWIB(time.Date(2026, 9, 30, 0, 0, 0, 0, wib)))
	require.False(t, dateOnlyWIB(time.Date(2026, 9, 30, 7, 59, 0, 0, wib)))
}

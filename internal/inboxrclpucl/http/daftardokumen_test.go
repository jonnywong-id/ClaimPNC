package inboxrclpuclhttp

import (
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxrclpucl"
)

// TestDaftarDokumenMengikutiPenyaringPega menjaga penyaring daftar "Lihat Dokumen".
//
// # Kenapa penyaring ini ada
//
// `GCNMGetAllAttachments` — report definition di balik tombol itu — berjalan di kelas
// `Link-Attachment` dan menyaring atas `pyCategory`, yang isinya NAMA kategori lampiran.
// Baris yang `CATEGORY`-nya kode angka berasal dari mekanisme lain dan TIDAK pernah tergambar
// di layar Pega.
//
// Dibandingkan langsung pada satu klaim 2026-10-02: Pega menggambar **4** lampiran, daftar
// kami menggambar **9**. Keenam selisihnya bernama `duplicated.JPG` berkategori `10064`.
func TestDaftarDokumenMengikutiPenyaringPega(t *testing.T) {
	jawaban := toDocumentListResponse([]inboxrclpucl.Document{
		{ID: "A", Name: "PUCL.pdf", Category: "Notification", PegaVisible: true},
		{ID: "B", Name: "Kwitansi.pdf", Category: "LOD", PegaVisible: true},
		{ID: "C", Name: "duplicated.JPG", Category: "10064", PegaVisible: false},
		{ID: "D", Name: "duplicated.JPG", Category: "10064", PegaVisible: false},
	}, "ASM")

	require.Len(t, jawaban.Dokumen, 2, "hanya yang tergambar di Pega yang digambar")
	require.Equal(t, "A", jawaban.Dokumen[0].ID)
	require.Equal(t, "B", jawaban.Dokumen[1].ID)
}

// TestYangDisaringDIHITUNG menjaga kejujuran daftar yang dipersempit.
//
// Daftar yang diam-diam lebih pendek adalah kegagalan yang tidak menghasilkan satu pun galat:
// dokumen yang dicari petugas hilang, dan tidak ada yang memberi tahu bahwa ia disembunyikan.
//
// Karena itu jumlahnya WAJIB ikut terkirim — bukan sekadar barisnya dibuang.
func TestYangDisaringDIHITUNG(t *testing.T) {
	jawaban := toDocumentListResponse([]inboxrclpucl.Document{
		{ID: "A", PegaVisible: true},
		{ID: "B", PegaVisible: false},
		{ID: "C", PegaVisible: false},
		{ID: "D", PegaVisible: false},
	}, "ASM")

	require.Equal(t, 3, jawaban.Disaring)
	require.Len(t, jawaban.Dokumen, 1)

	// Klaim yang seluruh lampirannya tergambar TIDAK melaporkan apa pun tersaring.
	bersih := toDocumentListResponse([]inboxrclpucl.Document{
		{ID: "A", PegaVisible: true},
	}, "ASM")
	require.Zero(t, bersih.Disaring)
}

// TestDaftarKosongTetapSenaraiKosong menjaga bentuk jawabannya.
//
// Seluruh baris tersaring menghasilkan `[]`, bukan `null`. Layar membedakan "tidak ada
// dokumen" dari "daftarnya gagal dibaca", dan `null` membuat keduanya terlihat sama.
func TestDaftarKosongTetapSenaraiKosong(t *testing.T) {
	jawaban := toDocumentListResponse([]inboxrclpucl.Document{
		{ID: "A", PegaVisible: false},
	}, "ASM")

	require.NotNil(t, jawaban.Dokumen)
	require.Empty(t, jawaban.Dokumen)
	require.Equal(t, 1, jawaban.Disaring)
}

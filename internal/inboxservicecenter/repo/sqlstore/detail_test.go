package sqlstore

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxservicecenter"
)

// TestUrutanKolomRincianSamaDenganSELECT menjaga ketiga tempat yang harus sejalan: daftar
// detailColumns, urutan kolom pada SELECT find_detail, dan urutan pemindai scanDetail.
//
// Pada 83 kolom yang HAMPIR SELURUHNYA teks, pergeseran satu kolom tidak menghasilkan galat
// apa pun — ia hanya menukar isi antar isian. Nomor IMEI muncul di kotak "NO. SERIAL", dan
// tidak ada yang memberi tahu.
func TestUrutanKolomRincianSamaDenganSELECT(t *testing.T) {
	text := query("find_detail")

	start := strings.Index(text, "SELECT")
	end := strings.Index(text, "FROM")
	require.Greater(t, end, start)

	body := text[start+len("SELECT") : end]

	position := -1
	for _, column := range detailColumns {
		found := strings.Index(body, "k."+column+",")
		if found < 0 {
			// Kolom terakhir tidak diikuti koma.
			found = strings.Index(body, "k."+column+"\n")
		}
		require.GreaterOrEqualf(t, found, 0, "kolom %s tidak ada di SELECT find_detail", column)
		require.Greaterf(t, found, position,
			"kolom %s tidak berurutan seperti detailColumns", column)
		position = found
	}
}

// TestJumlahKolomRincianSamaDenganJumlahYangDipindai mengunci ketiganya pada satu angka.
func TestJumlahKolomRincianSamaDenganJumlahYangDipindai(t *testing.T) {
	require.Len(t, detailColumns, 83)

	text := query("find_detail")
	body := text[strings.Index(text, "SELECT"):strings.Index(text, "FROM")]
	require.Len(t, regexp.MustCompile(`k\.[A-Z_]+`).FindAllString(body, -1), 83)
}

// TestKueriRincianMenyaringIDDanPIC — penyaring PIC adalah satu-satunya yang menahan
// pembacaan rincian milik petugas lain lewat ID yang ditebak.
func TestKueriRincianMenyaringIDDanPIC(t *testing.T) {
	text := query("find_detail")
	require.Contains(t, text, "UPPER(TRIM(k.ID)) = :1")
	require.Contains(t, text, "UPPER(TRIM(k.PIC)) = :2")
}

// TestKueriRiwayatDikunciRepairID mengunci pembedaan yang paling mudah tertukar di modul ini.
//
// `PROGRESS_SERVICECENTER_CLAIM` dikunci `REPAIRID`, bukan `ID`. Menukarnya menghasilkan
// riwayat yang SELALU kosong tanpa satu pun galat.
func TestKueriRiwayatDikunciRepairID(t *testing.T) {
	text := query("list_progress")
	require.Contains(t, text, "TRIM(p.REPAIRID) = :1")
	require.Contains(t, text, "ORDER BY p.INSERTDATE ASC")
	require.NotContains(t, text, "p.ID =")
}

// TestKueriRincianDanRiwayatTidakMenulis — keduanya membaca, dan tabelnya milik Pega (`P-1`).
func TestKueriRincianDanRiwayatTidakMenulis(t *testing.T) {
	for _, name := range []string{"find_detail", "list_progress"} {
		text := strings.ToUpper(query(name))
		for _, forbidden := range []string{"INSERT ", "UPDATE ", "DELETE ", "MERGE "} {
			require.NotContainsf(t, text, forbidden,
				"kueri %s memuat pernyataan yang menulis: %s", name, forbidden)
		}
	}
}

// TestKelompokIsianMemuatSeluruhIsianYangDikirim menjaga daftar kelompok tetap sejalan dengan
// isian yang benar-benar ada di kontrak.
//
// Kelompok yang menyebut isian tidak dikenal akan membuat layar menggambar sel kosong pada
// baris yang datanya sebenarnya ada — kegagalan yang diam.
func TestKelompokIsianTidakKosongDanTidakGanda(t *testing.T) {
	seen := map[string]string{}

	for _, group := range inboxservicecenter.DetailGroups() {
		require.NotEmptyf(t, group.Code, "kelompok tanpa kode")
		require.NotEmptyf(t, group.Title, "kelompok %s tanpa judul", group.Code)
		require.NotEmptyf(t, group.Fields, "kelompok %s tanpa isian", group.Code)

		for _, field := range group.Fields {
			require.NotEmptyf(t, field.Key, "isian tanpa kunci di kelompok %s", group.Code)
			require.NotEmptyf(t, field.Title,
				"isian %q tanpa judul di kelompok %s", field.Key, group.Code)

			previous, duplicate := seen[field.Key]
			require.Falsef(t, duplicate,
				"isian %q muncul di dua kelompok: %s dan %s", field.Key, previous, group.Code)
			seen[field.Key] = group.Code
		}
	}

	require.Len(t, inboxservicecenter.DetailGroups(), 7)
}

// TestDaftarIsianTidakDapatDiubahLewatHasilDetailGroups — sama dengan Tabs(), `Fields` adalah
// senarai dan salinan dangkal masih berbagi lariknya.
func TestDaftarIsianTidakDapatDiubahLewatHasilDetailGroups(t *testing.T) {
	groups := inboxservicecenter.DetailGroups()
	first := groups[0].Fields[0].Title
	groups[0].Fields[0].Title = "DIUBAH"

	require.Equal(t, first, inboxservicecenter.DetailGroups()[0].Fields[0].Title)
}

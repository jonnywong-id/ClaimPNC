package inboxinvestigatorhttp_test

import (
	"encoding/csv"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	portalhttp "claim-pnc/internal/portal/http"
)

// exportRoute adalah jalur tombol "Export Data Investigation".
const exportRoute = route + "/ekspor"

// exportQuery menyusun ketiga kendali ekspor dengan pengodean yang benar.
func exportQuery(from, to, investigated string) string {
	return url.Values{
		"dari":        {from},
		"sampai":      {to},
		"investigasi": {investigated},
	}.Encode()
}

// callRaw menjalankan satu permintaan dan mengembalikan badan respons MENTAH.
//
// Berbeda dari call, yang menguraikan badan sebagai JSON: berkas CSV bukan JSON, dan
// menguraikannya sebagai JSON akan menghasilkan peta kosong yang membuat setiap uji lulus
// tanpa memeriksa apa pun.
func callRaw(t *testing.T, p *testServer, path, portalAlias string) (*http.Response, string) {
	t.Helper()

	request, err := http.NewRequest(http.MethodGet, p.server.URL+path, nil)
	require.NoError(t, err)
	if p.token != "" {
		request.Header.Set("Authorization", "Bearer "+p.token)
	}
	if portalAlias != "" {
		request.Header.Set(portalhttp.HeaderPortal, portalAlias)
	}

	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)
	t.Cleanup(func() { _ = response.Body.Close() })

	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	return response, string(body)
}

// exportCSV menjalankan ekspor dan menguraikan berkasnya.
func exportCSV(t *testing.T, p *testServer, query, portalAlias string) [][]string {
	t.Helper()

	response, body := callRaw(t, p, exportRoute+"?"+query, portalAlias)
	require.Equal(t, http.StatusOK, response.StatusCode, body)

	records, err := csv.NewReader(strings.NewReader(body)).ReadAll()
	require.NoError(t, err)
	return records
}

// keysOf mengambil kunci peta sebagai senarai, untuk dibandingkan dengan Subset.
func keysOf(set map[string]bool) []string {
	list := make([]string, 0, len(set))
	for key := range set {
		list = append(list, key)
	}
	return list
}

// Judul berkasnya SAMA PERSIS dengan sistem lama.
//
// Ketiga belas judul itu ditulis sebagai satu teks tetap di
// `Activity/ExportDataInvestigator-Act.xml:2332`, dan pengguna mencocokkan berkas baru
// dengan berkas lama kolom per kolom. Ejaan "Remaks" yang terbaca seperti salah ketik
// TETAP dipakai, dengan alasan yang sama.
func TestExportHeaderMatchesTheLegacyFile(t *testing.T) {
	p := newTestServer(t)
	records := exportCSV(t, p, exportQuery("2026-09-01", "2026-09-30", "1"), "ASM")

	require.NotEmpty(t, records)
	require.Equal(t, []string{
		"Tanggal Investigasi",
		"Alamat RS Klinik",
		"Asuransi Lain",
		"Pasien",
		"Perusahaan",
		"Tidak ada pembayaran",
		"IsInvestigated",
		"Konfirmasi Model Kwitansi",
		"NoRekap Medis",
		"NoTelp DiHubungi",
		"Pasien Terdaftar",
		"Remaks",
		"Jenis Rumah Sakit",
	}, records[0])
}

// Dropdown "Pilih Investigation" menyaring isi berkas.
func TestExportRespectsTheInvestigationChoice(t *testing.T) {
	p := newTestServer(t)

	sudah := exportCSV(t, p, exportQuery("2026-09-01", "2026-09-30", "1"), "ASM")
	belum := exportCSV(t, p, exportQuery("2026-09-01", "2026-09-30", "0"), "ASM")

	require.Greater(t, len(sudah), 1, "berkas 'sudah' harus punya baris isi")
	require.Len(t, belum, 2, "contoh hanya punya satu baris 'belum'")

	const kolomInvestigated = 6
	for _, row := range sudah[1:] {
		require.Equal(t, "1", row[kolomInvestigated])
	}
	require.Equal(t, "0", belum[1][kolomInvestigated])
}

// Jenis rumah sakit ditulis sebagai TEKS, bukan kode.
//
// Satu-satunya kolom yang diterjemahkan; lihat ExportRow.HospitalKindLabel.
func TestExportWritesTheHospitalKindAsText(t *testing.T) {
	p := newTestServer(t)
	records := exportCSV(t, p, exportQuery("2026-09-01", "2026-09-30", "1"), "ASM")

	const kolomJenis = 12
	seen := map[string]bool{}
	for _, row := range records[1:] {
		seen[row[kolomJenis]] = true
	}
	require.Subset(t, []string{"Rumah Sakit", "NON Rumah Sakit"}, keysOf(seen))
}

// Berkasnya diunduh, bukan ditampilkan, dan tidak boleh mengendap di cache perantara.
//
// Isinya memuat alamat rumah sakit dan nomor rekam medis — data medis yang `FR-R2` batasi
// aksesnya.
func TestExportIsDownloadedAndNeverCached(t *testing.T) {
	p := newTestServer(t)
	response, _ := callRaw(t, p,
		exportRoute+"?"+exportQuery("2026-09-01", "2026-09-30", "1"), "ASM")

	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Contains(t, response.Header.Get("Content-Type"), "text/csv")
	require.Contains(t, response.Header.Get("Content-Disposition"), "attachment")
	require.Equal(t, "no-store", response.Header.Get("Cache-Control"))
}

// Nama berkasnya menyebut rentang dan pilihannya.
//
// Nama yang seragam membuat dua unduhan dengan penyaring berbeda saling menimpa di folder
// unduhan tanpa satu pun peringatan.
func TestExportFilenameNamesItsFilter(t *testing.T) {
	p := newTestServer(t)
	response, _ := callRaw(t, p,
		exportRoute+"?"+exportQuery("2026-09-01", "2026-09-30", "1"), "ASM")

	disposition := response.Header.Get("Content-Disposition")
	require.Contains(t, disposition, "2026-09-01")
	require.Contains(t, disposition, "2026-09-30")
	require.Contains(t, disposition, "sudah-investigasi")
}

// Penyaring yang tidak lengkap dijawab 400 beserta kalimat yang menyebut isiannya.
//
// 400, bukan berkas kosong. Berkas kosong tanpa keterangan persis yang dihasilkan sistem
// lama ketika dropdown belum dipilih, dan pengguna tidak punya cara membedakannya dari
// "memang tidak ada datanya".
func TestExportRejectsAnIncompleteFilter(t *testing.T) {
	p := newTestServer(t)

	for name, probe := range map[string]struct {
		query    string
		mentions string
	}{
		"tanpa pilihan investigasi": {
			query: exportQuery("2026-09-01", "2026-09-30", ""), mentions: "Pilih Investigation",
		},
		"pilihan investigasi tidak dikenal": {
			query: exportQuery("2026-09-01", "2026-09-30", "2"), mentions: "Pilih Investigation",
		},
		"tanggal Dari kosong": {
			query: exportQuery("", "2026-09-30", "1"), mentions: "Dari",
		},
		"tanggal Sampai kosong": {
			query: exportQuery("2026-09-01", "", "1"), mentions: "Sampai",
		},
		"tanggal Dari bukan tanggal": {
			query: exportQuery("01/09/2026", "2026-09-30", "1"), mentions: "Dari",
		},
		"rentang terbalik": {
			query: exportQuery("2026-09-30", "2026-09-01", "1"), mentions: "lebih awal",
		},
	} {
		t.Run(name, func(t *testing.T) {
			response, body := callRaw(t, p, exportRoute+"?"+probe.query, "ASM")
			require.Equal(t, http.StatusBadRequest, response.StatusCode, body)

			var content map[string]any
			require.NoError(t, json.Unmarshal([]byte(body), &content))
			require.Equal(t, "permintaan_cacat", content["kode"])
			require.Contains(t, content["pesan"], probe.mentions)
		})
	}
}

// Ekspor TIDAK dapat dijalankan tanpa menyebut portal.
//
// Berkasnya memuat data medis milik satu badan hukum. Permintaan tanpa portal ditolak,
// tidak pernah dilayani portal utama sebagai cadangan (`R-20`, `TKT-F6-002`).
func TestExportRequiresPortal(t *testing.T) {
	p := newTestServer(t)
	response, _ := callRaw(t, p,
		exportRoute+"?"+exportQuery("2026-09-01", "2026-09-30", "1"), "")
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
}

// Ekspor berada di balik middleware sesi, sama seperti daftarnya.
func TestExportRequiresSession(t *testing.T) {
	p := newTestServer(t)
	p.token = ""
	response, _ := callRaw(t, p,
		exportRoute+"?"+exportQuery("2026-09-01", "2026-09-30", "1"), "ASM")
	require.Equal(t, http.StatusUnauthorized, response.StatusCode)
}

// Portal yang tidak punya satu baris pun menghasilkan berkas berisi judul saja.
//
// Berkas berjudul tanpa isi adalah jawaban yang benar: ia membuktikan ekspornya berjalan
// dan rentangnya memang kosong. Itu berbeda dari galat, dan berbeda pula dari berkas yang
// gagal terbit.
func TestExportOfAnEmptyPortalStillHasItsHeader(t *testing.T) {
	p := newTestServer(t)
	records := exportCSV(t, p, exportQuery("2026-09-01", "2026-09-30", "1"), "ASI")
	require.Len(t, records, 1)
}

package dashboardclaimhttp_test

import (
	"encoding/csv"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// ============================================================================
// BERKAS UNDUHAN WAJIB SAMA SUSUNANNYA DENGAN BERKAS PEGA
// ============================================================================
//
// Uji ini lahir dari satu temuan: keempat berkas unduhan kita BERBEDA susunan kolomnya dari
// berkas Pega yang dipakai pengguna hari ini, dan tidak ada satu pun uji yang menyadarinya
// karena jalur unduhan sama sekali belum diuji.
//
// Judul yang diharapkan di bawah DISALIN MENTAH dari parameter `CSVPropHeaders` pada
// `pxConvertResultsToCSV` di activity Pega-nya, lalu hanya dibuang spasi di depan tiap
// judulnya. Menuliskannya sebagai satu string yang dipotong koma — bukan sebagai daftar Go
// yang sudah rapi — disengaja: bentuk itu dapat ditempelkan berdampingan dengan XML-nya dan
// dibaca sebaris, sehingga selisih satu kolom pun terlihat tanpa menghitung.

const (
	// Activity/ExportOutstanding_Act-Act.xml
	judulPegaOutstanding = "No Klaim, No Polis,TSI,EstimasiValue(IDR),Nama Tertanggung," +
		" Nama Bisnis, Sumber Bisnis, Nama Cabang, Nama Admin, Tgl Kejadian," +
		" Tgl Register, Tgl Transfer PIC, Posisi Klaim, Login Terakhir, PIC Teknik,Report Date"

	// Activity/ExportDataCloseClaim-Act.xml
	judulPegaCloseClaim = "No Klaim,No Polis,Nama Tertanggung,Nama Bisnis,Sumber Bisnis," +
		"Nama Cabang,Tanggal Pendaftaran,PIC Teknik, Admin PNC"

	// Activity/ExportSurveyorClaim-Act.xml — cabang adjuster
	judulPegaLossAdjuster = "Appointment No,Reference No,Claim No,Policy No,Adjuster," +
		"PIC Adjuster,Insured Name,PIC ASM,Status"

	// Activity/ExportSurveyorClaim-Act.xml — cabang internal
	judulPegaInternalSurveyor = "Nomor Case, No Klaim, No Polis, Nama Tertanggung," +
		" Tanggal Reschedule, Lokasi Reschedule, Status"
)

// judulPega memotong satu string `CSVPropHeaders` menjadi daftar judul yang sudah dirapikan.
func judulPega(raw string) []string {
	bagian := strings.Split(raw, ",")
	for i := range bagian {
		bagian[i] = strings.TrimSpace(bagian[i])
	}
	return bagian
}

func TestUnduhanMemakaiSusunanKolomPega(t *testing.T) {
	t.Parallel()

	for _, uji := range []struct {
		tile  string
		judul string
	}{
		{"outstanding", judulPegaOutstanding},
		{"close-claim", judulPegaCloseClaim},
		{"loss-adjuster", judulPegaLossAdjuster},
		{"internal-surveyor", judulPegaInternalSurveyor},
	} {
		t.Run(uji.tile, func(t *testing.T) {
			t.Parallel()

			baris := unduh(t, uji.tile)
			require.NotEmpty(t, baris, "berkas unduhan tidak boleh kosong sama sekali")
			require.Equal(t, judulPega(uji.judul), baris[0])
		})
	}
}

// TestUnduhanOutstandingMenaruhNilaiDiKolomYangBenar menjaga PEMETAANNYA, bukan hanya
// judulnya.
//
// Judul yang benar di atas kolom yang salah adalah kegagalan yang paling mahal di sini: ia
// terbaca wajar, tidak menimbulkan galat apa pun, dan baru ketahuan ketika seseorang memakai
// angkanya. Enam belas kolom Outstanding karena itu diperiksa satu per satu.
func TestUnduhanOutstandingMenaruhNilaiDiKolomYangBenar(t *testing.T) {
	t.Parallel()

	baris := unduh(t, "outstanding")
	require.GreaterOrEqual(t, len(baris), 2)

	judul := baris[0]
	data := cariBaris(t, baris, "PNCN.26.0001")

	// Dipasangkan judul->nilai supaya kegagalannya menyebut NAMA kolomnya, bukan indeks.
	harus := map[string]string{
		"No Klaim":           "PNCN.26.0001",
		"No Polis":           "POLIS-CONTOH-0001",
		"TSI":                "93980904506106",
		"EstimasiValue(IDR)": "17964.07",
		"Nama Tertanggung":   "Tertanggung Contoh A",
		"Nama Bisnis":        "Aneka",
		"Sumber Bisnis":      "Cabang",
		"Nama Cabang":        "Jakarta",
		"Nama Admin":         "ADMIN_ASAL_1",
		"Posisi Klaim":       "Register",
		"Login Terakhir":     "OPERATOR_PEMEGANG_1",
		"PIC Teknik":         "PIC Contoh 1",
	}

	for nama, nilai := range harus {
		require.Equal(t, nilai, nilaiKolom(t, judul, data, nama), "kolom %q", nama)
	}

	// Tanggal: bentuknya `dd/mm/yyyy`, mengikuti `to_char(…,'dd/mm/yyyy')` Pega.
	for _, nama := range []string{"Tgl Kejadian", "Tgl Register", "Tgl Transfer PIC", "Report Date"} {
		require.Regexp(t, `^\d{2}/\d{2}/\d{4}$`, nilaiKolom(t, judul, data, nama), "kolom %q", nama)
	}
}

// TestUnduhanOutstandingMembacaJalurExport menjaga bahwa unduhan memakai
// ListOutstandingExport, bukan ListOutstanding.
//
// Ganda memori sengaja MENGOSONGKAN kelima kolom itu pada ListOutstanding, persis seperti
// kueri layar yang memang tidak membacanya. Jadi bila jalurnya tertukar, TSI dan
// EstimasiValue menjadi kosong di sini — bukan menjadi galat, dan bukan pula tetap lulus.
func TestUnduhanOutstandingMembacaJalurExport(t *testing.T) {
	t.Parallel()

	baris := unduh(t, "outstanding")
	judul := baris[0]
	data := cariBaris(t, baris, "PNCN.26.0001")

	require.NotEmpty(t, nilaiKolom(t, judul, data, "TSI"),
		"TSI kosong: unduhan membaca kueri layar, bukan kueri unduhan")
	require.NotEmpty(t, nilaiKolom(t, judul, data, "EstimasiValue(IDR)"))
	require.NotEmpty(t, nilaiKolom(t, judul, data, "Nama Admin"))
	require.NotEmpty(t, nilaiKolom(t, judul, data, "Login Terakhir"))
	require.NotEmpty(t, nilaiKolom(t, judul, data, "Tgl Transfer PIC"))
}

// TestUnduhanMemakaiNamaBerkasPega menjaga nama berkas yang sampai ke pengguna.
//
// Ia bagian dari "sama dengan Pega" yang paling cepat terlihat: berkas lama dan baru harus
// berjajar dengan nama yang sama di folder unduhan.
func TestUnduhanMemakaiNamaBerkasPega(t *testing.T) {
	t.Parallel()

	for tile, nama := range map[string]string{
		"outstanding":       "Laporan Outstanding",
		"close-claim":       "Data Close Claim",
		"loss-adjuster":     "Data Lost Adjuster",
		"internal-surveyor": "Data Internal Surveyor",
	} {
		t.Run(tile, func(t *testing.T) {
			t.Parallel()

			rekam := get(t, testServer(t), "/api/dashboard-claim/"+tile+"/unduh")
			require.Equal(t, http.StatusOK, rekam.Code)
			require.Contains(t, rekam.Header().Get("Content-Disposition"), `filename="`+nama+`-`)
		})
	}
}

// unduh mengambil satu berkas unduhan dan mengurainya sebagai CSV.
func unduh(t *testing.T, tile string) [][]string {
	t.Helper()

	rekam := get(t, testServer(t), "/api/dashboard-claim/"+tile+"/unduh")
	require.Equal(t, http.StatusOK, rekam.Code, "badan: %s", rekam.Body.String())
	require.Contains(t, rekam.Header().Get("Content-Type"), "text/csv")

	baris, err := csv.NewReader(strings.NewReader(rekam.Body.String())).ReadAll()
	require.NoError(t, err)
	return baris
}

// cariBaris mengambil baris data yang kolom pertamanya cocok.
func cariBaris(t *testing.T, baris [][]string, kunci string) []string {
	t.Helper()

	for _, satu := range baris[1:] {
		if len(satu) > 0 && satu[0] == kunci {
			return satu
		}
	}
	t.Fatalf("baris %q tidak ada di berkas unduhan", kunci)
	return nil
}

// nilaiKolom membaca satu sel MENURUT JUDULNYA, bukan menurut indeksnya.
//
// Dengan begitu, kolom yang bergeser akan gagal pada judul yang hilang — pesan yang menyebut
// apa yang sebenarnya rusak — bukan pada perbandingan nilai yang membingungkan.
func nilaiKolom(t *testing.T, judul, baris []string, nama string) string {
	t.Helper()

	for i, j := range judul {
		if j == nama {
			require.Less(t, i, len(baris), "baris lebih pendek daripada judulnya")
			return baris[i]
		}
	}
	t.Fatalf("kolom %q tidak ada; yang ada: %v", nama, judul)
	return ""
}

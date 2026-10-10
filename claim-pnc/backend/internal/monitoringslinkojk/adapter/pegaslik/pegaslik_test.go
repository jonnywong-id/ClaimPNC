package pegaslik_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/monitoringslinkojk"
	"claim-pnc/internal/monitoringslinkojk/adapter/pegaslik"
)

// orang menyusun satu pengiriman perorangan yang sah.
func orang() monitoringslinkojk.Submission {
	return monitoringslinkojk.Submission{
		ID:         41,
		ClaimID:    "PNCN.26.0101",
		ContractNo: "KTR-2026-0101",
		Debtor: monitoringslinkojk.Debtor{
			CustomerType:  monitoringslinkojk.CustomerPerson,
			TransactionID: "41",
			FirstName:     "CONTOH NAMA",
			Gender:        "L",
			DateOfBirth:   "17/08/1985",
			Addresses: []monitoringslinkojk.DebtorAddress{{
				Street:     "Jalan Contoh Nomor 1",
				PostalCode: "12345",
				Primary:    true,
				Phones:     []monitoringslinkojk.DebtorPhone{{Number: "0210000000"}},
			}},
		},
	}
}

// jawab membentuk server tiruan yang menjawab satu badan JSON apa adanya.
func jawab(t *testing.T, status int, body string, rekam *[]byte) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if rekam != nil {
				isi, _ := io.ReadAll(r.Body)
				*rekam = isi
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			_, _ = io.WriteString(w, body)
		}))
	t.Cleanup(server.Close)
	return server
}

// ============================================================================
// ALAMAT BELUM ADA
// ============================================================================

// Portal tanpa alamat menolak — dan TIDAK menembak alamat portal lain.
//
// Ini uji terpenting di berkas ini. `D-75` menetapkan satu alamat per entitas, dan jatuh ke
// alamat bawaan berarti mendaftarkan klien satu badan hukum ke sistem badan hukum lain —
// kegagalan yang tidak terlihat sebagai galat apa pun (`R-20`).
func TestAlamatKosongMenolakTanpaMemanggil(t *testing.T) {
	client := pegaslik.NewClient(pegaslik.Config{})

	_, err := client.Send(context.Background(), orang())

	require.ErrorIs(t, err, monitoringslinkojk.ErrSenderNotConfigured)
}

// ============================================================================
// BENTUK PERMINTAAN
// ============================================================================

// Badan permintaannya memakai nama properti Pega apa adanya.
//
// Satu nama yang meleset tidak menghasilkan galat — ia menghasilkan field yang diam-diam
// kosong di sisi penerima. Uji ini yang menangkapnya.
func TestBadanPermintaanMemakaiNamaPropertiPega(t *testing.T) {
	var dikirim []byte
	server := jawab(t, http.StatusOK, `{"ClientID":"CL-9","ErrMsg":""}`, &dikirim)

	client := pegaslik.NewClient(pegaslik.Config{BaseURL: server.URL})
	_, err := client.Send(context.Background(), orang())
	require.NoError(t, err)

	var isi map[string]any
	require.NoError(t, json.Unmarshal(dikirim, &isi))

	page := isi["RequestService"].(map[string]any)["NBWorkPage"].(map[string]any)

	policy := page["Policy"].(map[string]any)
	require.Equal(t, "1", policy["CustomerType"])
	require.Equal(t, "41", policy["IdTransaction"],
		"kunci idempotensi = nomor urut pengiriman")

	person := page["Customer_P"].(map[string]any)
	require.Equal(t, "CONTOH NAMA", person["pyFirstName"])
	require.Equal(t, "L", person["ASMGender"])
	require.Equal(t, "17/08/1985", person["ASMDateOfBirth"])

	_, adaPerusahaan := page["Customer_C"]
	require.False(t, adaPerusahaan,
		"cabang perusahaan tidak boleh ikut terkirim pada CustomerType=1")

	alamat := page["AddressList"].([]any)[0].(map[string]any)
	require.Equal(t, "Jalan Contoh Nomor 1", alamat["ASMAddress"])
	require.Equal(t, "12345", alamat["ASMZipCode"])

	telepon := alamat["ASMTelfax"].([]any)[0].(map[string]any)
	require.Equal(t, "0210000000", telepon["TelfaxNumber"])
}

// Cabang perusahaan mengirim Customer_C, dan TIDAK mengirim Customer_P.
func TestCabangPerusahaan(t *testing.T) {
	var dikirim []byte
	server := jawab(t, http.StatusOK, `{"ClientID":"CL-7"}`, &dikirim)

	submission := orang()
	submission.Debtor = monitoringslinkojk.Debtor{
		CustomerType:  monitoringslinkojk.CustomerCompany,
		TransactionID: "41",
		CompanyName:   "PT CONTOH",
		CompanyID:     "COM-1",
	}

	client := pegaslik.NewClient(pegaslik.Config{BaseURL: server.URL})
	_, err := client.Send(context.Background(), submission)
	require.NoError(t, err)

	var isi map[string]any
	require.NoError(t, json.Unmarshal(dikirim, &isi))
	page := isi["RequestService"].(map[string]any)["NBWorkPage"].(map[string]any)

	require.Equal(t, "2", page["Policy"].(map[string]any)["CustomerType"])
	require.Equal(t, "PT CONTOH", page["Customer_C"].(map[string]any)["pyCompany"])

	_, adaPerorangan := page["Customer_P"]
	require.False(t, adaPerorangan)
}

// ============================================================================
// PENOLAKAN YANG DICEGAT DI MUKA
// ============================================================================

// Muatan tanpa nama ditolak SEBELUM dikirim.
//
// Penerimanya sendiri menolaknya (`pyFirstName==""` -> "Data Client tidak boleh kosong"),
// tetapi penolakan itu tetap meninggalkan baris pengiriman di tabel kita. Mencegatnya di
// muka membuat baris semacam itu tidak pernah ada.
func TestNamaKosongDitolakTanpaMemanggil(t *testing.T) {
	var dikirim []byte
	server := jawab(t, http.StatusOK, `{"ClientID":"CL-1"}`, &dikirim)

	submission := orang()
	submission.Debtor.FirstName = "   "

	client := pegaslik.NewClient(pegaslik.Config{BaseURL: server.URL})
	_, err := client.Send(context.Background(), submission)

	require.ErrorIs(t, err, monitoringslinkojk.ErrEmptyClient)
	require.Empty(t, dikirim, "permintaan tidak boleh sampai terkirim")
}

func TestNamaPerusahaanKosongDitolak(t *testing.T) {
	submission := orang()
	submission.Debtor = monitoringslinkojk.Debtor{
		CustomerType: monitoringslinkojk.CustomerCompany,
	}

	client := pegaslik.NewClient(pegaslik.Config{BaseURL: "https://contoh.invalid"})
	_, err := client.Send(context.Background(), submission)

	require.ErrorIs(t, err, monitoringslinkojk.ErrEmptyCompany)
}

// ============================================================================
// JAWABAN
// ============================================================================

// Jawaban berhasil mengembalikan ClientID dan nomor transaksi yang KITA kirim.
func TestJawabanBerhasil(t *testing.T) {
	server := jawab(t, http.StatusOK, `{"ClientID":"CL-9","ErrMsg":""}`, nil)

	client := pegaslik.NewClient(pegaslik.Config{BaseURL: server.URL})
	result, err := client.Send(context.Background(), orang())

	require.NoError(t, err)
	require.Equal(t, "CL-9", result.ClientID)
	require.Equal(t, "41", result.TransactionID)
}

// `ErrMsg` terisi berarti DITOLAK — meski HTTP-nya 200.
//
// Ini jalur yang paling mudah terlewat, dan akibatnya paling buruk: tanpa pemeriksaan ini
// penolakan tercatat sebagai pengiriman berhasil, dan pada laporan regulator itu kebohongan
// yang tidak terlihat oleh siapa pun.
func TestErrMsgTerisiBerartiDitolak(t *testing.T) {
	server := jawab(t, http.StatusOK,
		`{"ClientID":"","ErrMsg":"Data Client tidak boleh kosong"}`, nil)

	client := pegaslik.NewClient(pegaslik.Config{BaseURL: server.URL})
	_, err := client.Send(context.Background(), orang())

	require.Error(t, err)
	require.NotErrorIs(t, err, monitoringslinkojk.ErrSenderNotConfigured,
		"ditolak BUKAN belum tersedia — keduanya menuntut tindakan yang berbeda")
	require.Contains(t, err.Error(), "Data Client tidak boleh kosong")
}

// 5xx disamakan dengan belum tersedia: layanannya yang bermasalah, bukan datanya.
func TestLimaRatusDisamakanDenganBelumTersedia(t *testing.T) {
	server := jawab(t, http.StatusBadGateway, `{}`, nil)

	client := pegaslik.NewClient(pegaslik.Config{BaseURL: server.URL})
	_, err := client.Send(context.Background(), orang())

	require.ErrorIs(t, err, monitoringslinkojk.ErrSenderNotConfigured)
}

// 4xx BUKAN belum tersedia: permintaannya sampai dan ditolak.
func TestEmpatRatusBukanBelumTersedia(t *testing.T) {
	server := jawab(t, http.StatusBadRequest, `{}`, nil)

	client := pegaslik.NewClient(pegaslik.Config{BaseURL: server.URL})
	_, err := client.Send(context.Background(), orang())

	require.Error(t, err)
	require.NotErrorIs(t, err, monitoringslinkojk.ErrSenderNotConfigured)
}

// ============================================================================
// ALAMAT DAN OTENTIKASI
// ============================================================================

// Jalur bawaannya mengikuti pola layanan REST Pega.
func TestJalurBawaan(t *testing.T) {
	var jalur string
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			jalur = r.URL.Path
			_, _ = io.WriteString(w, `{"ClientID":"CL-1"}`)
		}))
	t.Cleanup(server.Close)

	client := pegaslik.NewClient(pegaslik.Config{BaseURL: server.URL})
	_, err := client.Send(context.Background(), orang())
	require.NoError(t, err)

	require.Equal(t,
		"/prweb/PRRestService/ASMFWInternalWork/Data-Portal/ASMRequestServiceCreateClient",
		jalur)
}

// Tanpa kredensial, permintaannya dikirim TANPA otentikasi — keadaan layanan ini hari ini.
func TestTanpaKredensialTidakMengirimOtentikasi(t *testing.T) {
	var punyaAuth bool
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			_, _, punyaAuth = r.BasicAuth()
			_, _ = io.WriteString(w, `{"ClientID":"CL-1"}`)
		}))
	t.Cleanup(server.Close)

	client := pegaslik.NewClient(pegaslik.Config{BaseURL: server.URL})
	_, err := client.Send(context.Background(), orang())
	require.NoError(t, err)

	require.False(t, punyaAuth)
}

// Dengan kredensial, Basic Auth terkirim — tanpa menyentuh kode.
func TestKredensialMengirimBasicAuth(t *testing.T) {
	var user string
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			user, _, _ = r.BasicAuth()
			_, _ = io.WriteString(w, `{"ClientID":"CL-1"}`)
		}))
	t.Cleanup(server.Close)

	client := pegaslik.NewClient(pegaslik.Config{
		BaseURL: server.URL, User: "pelapor", Password: "rahasia"})
	_, err := client.Send(context.Background(), orang())
	require.NoError(t, err)

	require.Equal(t, "pelapor", user)
}

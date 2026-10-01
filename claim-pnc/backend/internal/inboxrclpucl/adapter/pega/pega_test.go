package pega_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxrclpucl"
	"claim-pnc/internal/inboxrclpucl/adapter/pega"
)

func contoh() inboxrclpucl.ClaimActionCommand {
	return inboxrclpucl.ClaimActionCommand{
		Kind:         inboxrclpucl.ActionSendToAnalyst,
		CaseNumber:   "PNC-700001",
		IDObject:     "OBJ-7",
		IDCoverage:   "COV-8",
		IDAdjustment: "ADJ-9",
		Note:         "Dokumen sudah lengkap.",
		Caller:       "petugascontoh",
	}
}

func TestAlamatKosongBerartiBelumTersedia(t *testing.T) {
	// Keadaan HARI INI: layanannya belum dibangun. Ia TIDAK boleh membuat aplikasi gagal
	// menyala maupun layarnya gagal digambar — yang gagal hanyalah tindakannya, dengan galat
	// yang dapat dikenali transport dan diterjemahkan menjadi kalimat yang berguna.
	err := pega.NewClient(pega.Config{}).Perform(context.Background(), contoh())
	require.ErrorIs(t, err, inboxrclpucl.ErrPegaServiceUnavailable)
}

func TestParameterDikirimDenganNamaPUCLPost(t *testing.T) {
	// Nama parameter adalah KONTRAK dengan Pega, bukan gaya penamaan kami. `Status` wajib
	// `"1"` — nilai itulah yang membedakan "kirim" dari "cetak" (kosong) dan "tolak" (`0`) —
	// dan ketiga id wajib sampai APA ADANYA, karena `PUCLPost` memakai `idCov` untuk memilih
	// BARIS adjustment mana yang disetujui.
	var terima map[string]any
	var jalur, metode string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jalur, metode = r.URL.Path, r.Method
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &terima)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := pega.NewClient(pega.Config{BaseURL: srv.URL}).
		Perform(context.Background(), contoh())
	require.NoError(t, err)

	require.Equal(t, "1", terima["Status"], `Status "1" yang meneruskan klaim ke Analyst`)
	require.Equal(t, "dokumen", terima["tipe"])
	require.Equal(t, "OBJ-7", terima["idObj"])
	require.Equal(t, "COV-8", terima["idCov"])
	require.Equal(t, "ADJ-9", terima["idAdj"])
	require.Equal(t, "PNC-700001", terima["caseNumber"])

	// Pelakunya ikut: Pega mencatat pelaku pada objek kerja, dan tanpa ini jejaknya menunjuk
	// akun integrasi alih-alih orangnya.
	require.Equal(t, "petugascontoh", terima["caller"])

	// Jalur dan metodenya ikut dijaga: keduanya bagian kontrak, dan `PUCLPost` menimbulkan
	// akibat sehingga tidak boleh dipanggil dengan GET.
	// Nama layanannya BUKAN "PUCLPost": itu nama ACTIVITY, dan memakainya untuk layanan
	// menghasilkan dua rule bernama sama di Pega.
	require.Equal(t, "/ActionClaimPUCL", jalur)
	require.NotContains(t, jalur, "PUCLPost")
	require.Equal(t, "POST", metode)
}

func TestGagalMenghubungiTerbacaSebagaiBelumTersedia(t *testing.T) {
	// Bagi petugas yang menekan tombol, "belum dibangun" dan "tidak dapat dihubungi" berarti
	// hal yang sama: tindakannya belum dapat dijalankan dari sini.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	err := pega.NewClient(pega.Config{BaseURL: srv.URL}).
		Perform(context.Background(), contoh())
	require.ErrorIs(t, err, inboxrclpucl.ErrPegaServiceUnavailable)
}

func TestPenolakanBUKANKetidaktersediaan(t *testing.T) {
	// 4xx berarti permintaannya SAMPAI dan ditolak. Membungkusnya sebagai "belum tersedia"
	// akan menyuruh orang menunggu Tim Pega, padahal yang salah ada di permintaan kita.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	err := pega.NewClient(pega.Config{BaseURL: srv.URL}).
		Perform(context.Background(), contoh())
	require.Error(t, err)
	require.False(t, errors.Is(err, inboxrclpucl.ErrPegaServiceUnavailable))
}

func TestJalurDanKredensialDapatDiaturTanpaUbahKode(t *testing.T) {
	// Nama resource ditentukan Tim Pega, bukan kami. Menjadikannya konstanta berarti nama
	// yang berbeda dari usulan menuntut perubahan kode, bangun ulang, dan rilis — untuk satu
	// kata. Uji ini menjaga keduanya tetap dapat diatur dari konfigurasi.
	var jalur, auth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jalur = r.URL.Path
		auth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := pega.NewClient(pega.Config{
		BaseURL:  srv.URL,
		Path:     "TindakanKlaimPNC",
		User:     "svc-claimpnc",
		Password: "rahasia",
	}).Perform(context.Background(), contoh())
	require.NoError(t, err)

	require.Equal(t, "/TindakanKlaimPNC", jalur)
	require.NotEmpty(t, auth, "Basic Auth wajib terkirim saat kredensialnya diisi")
}

func TestTanpaKredensialTIDAKMengirimHeaderOtentikasi(t *testing.T) {
	// Mengirim header kosong lebih buruk daripada tidak mengirim: sebagian peladen menolaknya
	// sebagai kredensial yang salah, bukan sebagai permintaan tanpa kredensial — dan galatnya
	// lalu terbaca seperti sandi yang keliru.
	var ada bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, ada = r.Header["Authorization"]
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := pega.NewClient(pega.Config{BaseURL: srv.URL}).
		Perform(context.Background(), contoh())
	require.NoError(t, err)
	require.False(t, ada)
}

func TestTiapTindakanMengirimParameterPUCLPostYangBENAR(t *testing.T) {
	// Pemetaannya dibaca dari rangkaian aksi tiap tombol di section-nya. Satu nilai yang
	// tertukar membuat Pega mengerjakan tindakan yang BERBEDA — mencetak surat alih-alih
	// meneruskan klaim, atau menolak alih-alih mengirim — dan `PUCLPost` tidak menolaknya.
	//
	// Keempat nilai di bawah disalin dari `pyBehaviors` tiap tombol — termasuk SPASI di ujung
	// "Wait for Complete PUCL Document ", yang ada di Pega dan terbawa ke kolom riwayat apa
	// adanya. Membuangnya mengubah data yang tersimpan, sehingga ia dijaga di sini.
	for _, c := range []struct {
		kind                                 inboxrclpucl.ClaimActionKind
		status, tipe, statusCase, statusNote string
	}{
		{inboxrclpucl.ActionPrintLetter, "", "cetak", "1", "Wait for Complete PUCL Document "},
		{inboxrclpucl.ActionRejectClaim, "0", "dokumen", "", ""},
		{inboxrclpucl.ActionSendToAnalyst, "1", "dokumen", "", ""},
		// Tanpa `tipe`: jalur ini memakai `InsertHistoryClaimPNC`, bukan `InsertMitraPA`.
		{inboxrclpucl.ActionSendToPICTeknik, "1", "", "", "send by PUCL to PIC Teknis"},
		{inboxrclpucl.ActionSave, "", "", "", ""},
	} {
		var terima map[string]any
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			b, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(b, &terima)
			w.WriteHeader(http.StatusOK)
		}))

		cmd := contoh()
		cmd.Kind = c.kind
		err := pega.NewClient(pega.Config{BaseURL: srv.URL}).Perform(context.Background(), cmd)
		srv.Close()

		require.NoErrorf(t, err, "tindakan %s", c.kind)
		require.Equalf(t, string(c.kind), terima["aksi"], "tindakan %s", c.kind)
		require.Equalf(t, c.status, terima["Status"], "Status tindakan %s", c.kind)
		require.Equalf(t, c.tipe, terima["tipe"], "tipe tindakan %s", c.kind)
		require.Equalf(t, c.statusCase, terima["statusCase"], "statusCase tindakan %s", c.kind)
		require.Equalf(t, c.statusNote, terima["statusNote"], "statusNote tindakan %s", c.kind)
	}
}

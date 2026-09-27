package gateway_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/archivedokumenklaim"
	"claim-pnc/internal/archivedokumenklaim/gateway"
)

// katalogTetap adalah pengisi ServiceCatalog yang selalu menjawab satu alamat.
type katalogTetap struct {
	address string
	err     error

	askedApp  string
	askedKind string
}

func (k *katalogTetap) ServiceAddress(_ context.Context, app, kind string) (string, error) {
	k.askedApp, k.askedKind = app, kind
	return k.address, k.err
}

func shipment() archivedokumenklaim.Shipment {
	return archivedokumenklaim.Shipment{
		ID:          42,
		BoxName:     "BOX-A-01",
		FillingCode: "FIL-2024-001",
		RequestedAt: time.Date(2026, 9, 24, 3, 15, 0, 0, time.UTC),
	}
}

// Badan permintaan adalah KONTRAK dengan sistem Arsip milik tim lain.
//
// Nama fieldnya, bentuk NoDokumen, dan bentuk tanggalnya ditiru persis dari
// `Activity/SendDataArchiveDOcumentByService-Act.xml`. Mengubah salah satunya berarti
// mengubah kontrak dengan sistem yang tidak kita miliki.
func TestBadanPermintaanMengikutiKontrakSistemArsip(t *testing.T) {
	var captured map[string]any
	var contentType string

	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			contentType = r.Header.Get("Content-Type")
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &captured)

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"ResponseCode":"200","ResponseMessage":"OK"}`))
		}))
	defer server.Close()

	katalog := &katalogTetap{address: server.URL}
	client, err := gateway.NewHTTP(gateway.Options{Catalog: katalog})
	require.NoError(t, err)

	receipt, err := client.Send(context.Background(), "asm", shipment())
	require.NoError(t, err)

	require.Equal(t, "application/json", contentType)
	require.Equal(t, "42/BOX-A-01/FIL-2024-001", captured["NoDokumen"])
	require.Equal(t, "24/09/2026", captured["TglRequest"],
		"bentuk tanggalnya dd/mm/yyyy, persis activity lama")

	require.Equal(t, "200", receipt.Code)
	require.Equal(t, "OK", receipt.Note)
	require.Equal(t, int64(42), receipt.ID)
	require.Contains(t, receipt.Request, "NoDokumen",
		"badan permintaan ikut tersimpan sebagai jejak — kolom HITARCHIVE")
}

// Alamat dicari menurut portal DAN jenis layanan.
//
// Tiap portal entitas boleh punya alamat Arsip sendiri; memakai satu alamat untuk semua
// akan mengarsipkan berkas satu badan hukum ke sistem badan hukum lain (`R-20`).
func TestAlamatDicariMenurutPortal(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"ResponseCode":"200","ResponseMessage":"OK"}`))
		}))
	defer server.Close()

	katalog := &katalogTetap{address: server.URL}
	client, err := gateway.NewHTTP(gateway.Options{Catalog: katalog})
	require.NoError(t, err)

	_, err = client.Send(context.Background(), "simasnet", shipment())
	require.NoError(t, err)

	require.Equal(t, "simasnet", katalog.askedApp)
	require.Equal(t, gateway.ArchiveServiceKind, katalog.askedKind)
}

// Alamat yang belum terdaftar DIBEDAKAN dari layanan yang menolak.
//
// Keduanya ditangani orang yang berbeda — yang pertama DBA, yang kedua tim Arsip. Pesannya
// menyebutkan tepat apa yang kurang supaya pelapornya tidak perlu menebak.
func TestAlamatKosongMenghasilkanGalatTersendiri(t *testing.T) {
	client, err := gateway.NewHTTP(gateway.Options{Catalog: &katalogTetap{address: "  "}})
	require.NoError(t, err)

	_, err = client.Send(context.Background(), "asm", shipment())
	require.ErrorIs(t, err, archivedokumenklaim.ErrServiceAddress)
	require.Contains(t, err.Error(), gateway.ArchiveServiceKind)
}

func TestKatalogGagalDiperlakukanSebagaiAlamatKosong(t *testing.T) {
	katalog := &katalogTetap{err: errors.New("tabel tidak dapat dibaca")}

	client, err := gateway.NewHTTP(gateway.Options{Catalog: katalog})
	require.NoError(t, err)

	_, err = client.Send(context.Background(), "asm", shipment())
	require.ErrorIs(t, err, archivedokumenklaim.ErrServiceAddress)
}

func TestStatusNonDuaRatusDianggapGagal(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"ResponseCode":"500"}`))
		}))
	defer server.Close()

	client, err := gateway.NewHTTP(gateway.Options{Catalog: &katalogTetap{address: server.URL}})
	require.NoError(t, err)

	_, err = client.Send(context.Background(), "asm", shipment())
	require.ErrorIs(t, err, archivedokumenklaim.ErrServiceFailed)
}

// Jawaban 200 yang BUKAN JSON dianggap gagal, bukan dianggap berhasil dengan kode kosong.
//
// Ini kelas cacat yang sudah pernah menggigit aplikasi ini: jawaban 200 berisi HTML yang
// diurai sebagai kosong menghasilkan layar yang tidak menampilkan apa pun tanpa satu pun
// petunjuk.
func TestJawabanBukanJSONDianggapGagal(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte(`<!doctype html><html><body>OK</body></html>`))
		}))
	defer server.Close()

	client, err := gateway.NewHTTP(gateway.Options{Catalog: &katalogTetap{address: server.URL}})
	require.NoError(t, err)

	_, err = client.Send(context.Background(), "asm", shipment())
	require.ErrorIs(t, err, archivedokumenklaim.ErrServiceFailed)
}

func TestGatewayMenolakRakitanTanpaKatalog(t *testing.T) {
	_, err := gateway.NewHTTP(gateway.Options{})
	require.Error(t, err)
}

// Perekam menyusun badan permintaan dengan bentuk yang SAMA dengan klien nyata.
//
// Kalau bentuknya berbeda, isi kolom HITARCHIVE di pengembangan tidak akan menyerupai
// isinya di produksi — dan justru kolom itulah satu-satunya bukti yang tersisa bila kelak
// pengiriman dipersoalkan.
func TestPerekamMenyusunBadanYangSamaDenganKlienNyata(t *testing.T) {
	recorder := gateway.NewRecorder()

	receipt, err := recorder.Send(context.Background(), "asm", shipment())
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal([]byte(receipt.Request), &decoded))
	require.Equal(t, "42/BOX-A-01/FIL-2024-001", decoded["NoDokumen"])
	require.Equal(t, "24/09/2026", decoded["TglRequest"])

	require.Len(t, recorder.Sent(), 1)
}

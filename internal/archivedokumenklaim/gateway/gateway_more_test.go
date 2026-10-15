package gateway_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/archivedokumenklaim"
	"claim-pnc/internal/archivedokumenklaim/gateway"
)

// Alamat yang tidak dapat dijadikan permintaan HTTP dianggap kegagalan layanan.
func TestAlamatCacatDianggapGagal(t *testing.T) {
	client, err := gateway.NewHTTP(gateway.Options{Catalog: &katalogTetap{address: "://cacat"}})
	require.NoError(t, err)

	_, err = client.Send(context.Background(), "asm", shipment())
	require.ErrorIs(t, err, archivedokumenklaim.ErrServiceFailed)
	require.ErrorContains(t, err, "menyusun permintaan")
}

// Layanan yang tidak dapat dihubungi dianggap kegagalan layanan.
func TestLayananTidakTerhubungDianggapGagal(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	address := server.URL
	server.Close()

	client, err := gateway.NewHTTP(gateway.Options{
		Catalog: &katalogTetap{address: address},
		Client:  &http.Client{},
	})
	require.NoError(t, err)

	_, err = client.Send(context.Background(), "asm", shipment())
	require.ErrorIs(t, err, archivedokumenklaim.ErrServiceFailed)
}

// Jawaban dipangkas spasinya sebelum disimpan.
func TestJawabanDipangkas(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"ResponseCode":" 201 ","ResponseMessage":" diterima "}`))
		}))
	defer server.Close()

	client, err := gateway.NewHTTP(gateway.Options{Catalog: &katalogTetap{address: server.URL}})
	require.NoError(t, err)

	receipt, err := client.Send(context.Background(), "asm", shipment())
	require.NoError(t, err)
	require.Equal(t, "201", receipt.Code)
	require.Equal(t, "diterima", receipt.Note)
	require.Equal(t, shipment().RequestedAt, receipt.SentAt)
}

func TestPerekamMengembalikanGalatYangDiatur(t *testing.T) {
	recorder := gateway.NewRecorder()
	recorder.Err = errors.New("layanan mati")

	_, err := recorder.Send(context.Background(), "asm", shipment())
	require.EqualError(t, err, "layanan mati")
	require.Empty(t, recorder.Sent(), "pengiriman yang gagal tidak terekam")
}

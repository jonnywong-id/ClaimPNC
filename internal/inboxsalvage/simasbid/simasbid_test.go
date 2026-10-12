package simasbid_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxsalvage"
	"claim-pnc/internal/inboxsalvage/simasbid"
)

func submission() inboxsalvage.AuctionSubmission {
	return inboxsalvage.AuctionSubmission{
		ItemID:          "77",
		ClaimNo:         "PNC-2044",
		SalvageID:       "77",
		ItemName:        "Mesin Genset",
		ItemDescription: "Terbakar sebagian",
		Location:        "Gudang Cakung",
		InJabodetabek:   true,
		Price:           "15000000",
		EntityFlag:      "ASM",
	}
}

// Badan permintaan mengikuti susunan `MyServicePage.ClaimData.Lelang` pada rule lama.
//
// Nama fieldnya adalah KONTRAK milik sistem lain — ia sampai ke SimasBid apa adanya lewat
// `@GCNM.GetPageJSONString()`. Uji ini yang menjaga penggantian nama di sisi Go tidak
// diam-diam mengubah apa yang diterima pihak sana.
func TestSendSalvagePostsTheLelangShape(t *testing.T) {
	var got map[string]any

	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, http.MethodPost, r.Method)
			require.Equal(t, "application/json", r.Header.Get("Content-Type"))

			raw, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			require.NoError(t, json.Unmarshal(raw, &got))

			_, _ = io.WriteString(w,
				`{"ResponseService":{"ResponseMessage":"Sukses","ResponseID":"SB-9"}}`)
		}))
	defer server.Close()

	client := simasbid.NewClient(simasbid.Config{URL: server.URL})

	receipt, err := client.SendSalvage(context.Background(), submission())
	require.NoError(t, err)
	require.True(t, receipt.Accepted())
	require.Equal(t, "SB-9", receipt.AuctionID)

	lelang := got["ClaimData"].(map[string]any)["Lelang"].(map[string]any)
	require.Equal(t, "77", lelang["IDObject"])
	require.Equal(t, "Mesin Genset", lelang["ObjectName"])
	require.Equal(t, "Gudang Cakung", lelang["Location"])
	require.Equal(t, "Terbakar sebagian", lelang["ItemDescription"])
	require.Equal(t, "15000000", lelang["Harga"])
	require.Equal(t, "ASM", lelang["Flag"])

	// TEKS, bukan boolean JSON — bentuk itu dari rule lama, yang mengirimkannya sebagai
	// `@if(Param.isjabodatabek=="","false",Param.isjabodatabek)`.
	require.Equal(t, "true", lelang["FlagJabodetabek"])

	// Keduanya konstanta di rule lama, bukan turunan data pengajuan.
	require.Equal(t, "1", lelang["Type"])
	require.Equal(t, "1", lelang["SalvageStatus"])

	// Senarai KOSONG, bukan null: penerima yang menguraikan JSON dengan skema ketat sering
	// menolak field yang berubah bentuk.
	require.Equal(t, []any{}, lelang["DocumentList"])
}

// Pesan pada akar jawaban ikut terbaca, bukan hanya yang di dalam `ResponseService`.
//
// Nama field JSON-nya tidak ada di export sama sekali — yang diketahui hanyalah activity
// lama membacanya sebagai `ResponseService.ResponseMessage`. Pembacaan yang longgar adalah
// akibat langsung dari ketidaktahuan itu, dan ia dinyatakan di banner paket.
func TestSendSalvageReadsTheMessageFromEitherPlace(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, `{"ResponseMessage":"Sukses","IDSimasBid":"SB-1"}`)
		}))
	defer server.Close()

	client := simasbid.NewClient(simasbid.Config{URL: server.URL})

	receipt, err := client.SendSalvage(context.Background(), submission())
	require.NoError(t, err)
	require.True(t, receipt.Accepted())
	require.Equal(t, "SB-1", receipt.AuctionID)
}

// Kode HTTP di luar 2xx BUKAN galat Go.
//
// Ia jawaban yang sah: pengajuannya sampai ke sana dan ditolak. Mengembalikannya sebagai
// galat akan membuat pemanggil memperlakukannya seperti jaringan yang putus — dan
// penolakannya tidak pernah tercatat pada baris pengajuan.
func TestSendSalvageTreatsARejectionAsAnAnswerNotAnError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = io.WriteString(w, `{"ResponseMessage":"ID sudah terdaftar"}`)
		}))
	defer server.Close()

	client := simasbid.NewClient(simasbid.Config{URL: server.URL})

	receipt, err := client.SendSalvage(context.Background(), submission())
	require.NoError(t, err)
	require.False(t, receipt.Accepted())
	require.Equal(t, "ID sudah terdaftar", receipt.Message)
}

// Jawaban yang bukan JSON tetap sampai ke petugas sebagai PESAN.
//
// Balai lelang yang menjawab halaman galat HTML sudah menerima permintaannya; menelan
// jawabannya menjadi "gagal menghubungi" menyembunyikan satu-satunya keterangan yang ada.
func TestSendSalvageSummarizesAnAnswerThatIsNotJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusBadGateway)
			_, _ = io.WriteString(w, "<html><body>Bad Gateway</body></html>")
		}))
	defer server.Close()

	client := simasbid.NewClient(simasbid.Config{URL: server.URL})

	receipt, err := client.SendSalvage(context.Background(), submission())
	require.NoError(t, err)
	require.False(t, receipt.Accepted())
	require.Contains(t, receipt.Message, "HTTP 502")
	require.Contains(t, receipt.Message, "Bad Gateway")
}

// Jawaban yang panjang DIPOTONG sebelum dibawa ke layar dan ke basis data.
func TestSendSalvageTrimsAVeryLongAnswer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, strings.Repeat("x", 5000))
		}))
	defer server.Close()

	client := simasbid.NewClient(simasbid.Config{URL: server.URL})

	receipt, err := client.SendSalvage(context.Background(), submission())
	require.NoError(t, err)
	require.Less(t, len(receipt.Message), 300)
}

// Alamat yang kosong menghasilkan galat BERNAMA, bukan permintaan ke alamat kosong.
//
// Pemanggil membedakannya dengan errors.Is, dan dari situlah layar tahu bahwa yang perlu
// dihubungi adalah Tim Infra — bukan balai lelang.
func TestSendSalvageWithoutAnAddressIsNamed(t *testing.T) {
	client := simasbid.NewClient(simasbid.Config{})

	_, err := client.SendSalvage(context.Background(), submission())
	require.True(t, errors.Is(err, inboxsalvage.ErrAuctionNotAvailable))
}

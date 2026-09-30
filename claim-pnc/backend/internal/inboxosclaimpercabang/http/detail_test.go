package inboxosclaimpercabanghttp_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	oscabanghttp "claim-pnc/internal/inboxosclaimpercabang/http"
)

func TestDetailReturnsTheFourSections(t *testing.T) {
	recorder := get(t, testServer(t), "/api/inbox-os-claim-per-cabang/PNC-9001")
	require.Equal(t, http.StatusOK, recorder.Code)

	body := decode(t, recorder)

	header := body["ringkasan"].(map[string]any)
	require.Equal(t, "PNC-9001", header["no_klaim"])
	require.Equal(t, "Aneka", header["cob"])
	require.Equal(t, "Terjadi kebakaran pada malam hari.", header["kronologi"])
	require.Equal(t, "Menunggu hasil survei", header["note_pic"])

	require.Len(t, body["objek"], 2)
	require.Len(t, body["riwayat_progres"], 2)
	require.Len(t, body["komunikasi_adjuster"], 2)

	require.Equal(t, "100099", body["cabang"].(map[string]any)["kode"])
	require.NotEmpty(t, body["selisih_terencana"])
}

func TestDetailSendsMoneyAsTextNotNumber(t *testing.T) {
	// Angka JSON dibaca JavaScript sebagai float64, dan nilai uang tidak pernah float
	// (`I-12`). Uji ini gagal begitu tipenya berubah menjadi angka.
	recorder := get(t, testServer(t), "/api/inbox-os-claim-per-cabang/PNC-9001")
	header := decode(t, recorder)["ringkasan"].(map[string]any)

	require.IsType(t, "", header["total_reserve"])
	require.IsType(t, "", header["total_sum_insured"])
	require.Equal(t, "250000.00", header["total_reserve"])
	require.Equal(t, "5000000.00", header["total_sum_insured"])
}

func TestDetailSendsTimestampsWithTheirHour(t *testing.T) {
	// Riwayat progres dan komunikasi membawa JAM, sedangkan kolom tanggal pada grid tidak.
	// Jam itu yang membedakan dua catatan pada hari yang sama; membuangnya membuat urutan
	// barisnya tampak acak.
	recorder := get(t, testServer(t), "/api/inbox-os-claim-per-cabang/PNC-9001")
	body := decode(t, recorder)

	progress := body["riwayat_progres"].([]any)[0].(map[string]any)
	require.Equal(t, "2024-02-01 16:00", progress["tanggal_input"],
		"cap waktu harus WIB, bukan UTC")
	require.Equal(t, "2024-02-08", progress["tanggal_next_followup"],
		"tanggal follow-up digambar tanpa jam di layar lama")

	// Objek memakai tanggal tanpa jam.
	objects := body["objek"].([]any)
	require.Equal(t, "", objects[0].(map[string]any)["tanggal_lahir"])
}

func TestDetailMarksIncomingAndOutgoingMessages(t *testing.T) {
	recorder := get(t, testServer(t), "/api/inbox-os-claim-per-cabang/PNC-9001")
	messages := decode(t, recorder)["komunikasi_adjuster"].([]any)

	require.Equal(t, true, messages[0].(map[string]any)["internal"])
	require.Equal(t, false, messages[1].(map[string]any)["internal"])
}

func TestDetailOfAnotherBranchIsNotFoundAndDoesNotSayWhy(t *testing.T) {
	// PNC-9004 ada, tetapi milik BANDUNG. Pesannya tidak boleh menyebutkan bahwa nomor itu
	// ada di cabang lain — kalau menyebutkan, endpoint ini menjadi alat memastikan keberadaan
	// klaim badan hukum lain (`R-20`).
	recorder := get(t, testServer(t), "/api/inbox-os-claim-per-cabang/PNC-9004")
	require.Equal(t, http.StatusNotFound, recorder.Code)

	body := decode(t, recorder)
	require.Equal(t, "klaim_tidak_ditemukan", body["kode"])
	require.NotContains(t, body["pesan"], "cabang lain")
	require.NotContains(t, body["pesan"], "BANDUNG")
}

func TestDetailWithoutBranchGetsTheLegacyNotice(t *testing.T) {
	server := buildServer(t, oscabanghttp.Caller{Login: "MITRA1"}, true)

	recorder := get(t, server, "/api/inbox-os-claim-per-cabang/PNC-9001")
	require.Equal(t, http.StatusConflict, recorder.Code)
	require.Equal(t, "cabang_tidak_diketahui", decode(t, recorder)["kode"])
}

func TestExportPathIsNotSwallowedByTheDetailRoute(t *testing.T) {
	// `/ekspor` dan `/{nomor}` berbagi bentuk jalur yang sama. Bila pencocokannya terbalik,
	// tombol Export akan menjawab "klaim tidak ditemukan" — kegagalan yang terbaca sebagai
	// masalah data, bukan sebagai masalah rute.
	recorder := get(t, testServer(t), "/api/inbox-os-claim-per-cabang/ekspor")

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Header().Get("Content-Type"), "csv")
}

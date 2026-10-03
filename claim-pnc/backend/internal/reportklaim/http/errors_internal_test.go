package reportklaimhttp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	reportklaimsql "claim-pnc/internal/reportklaim/repo/sqlstore"
)

// Penolakan basis data dijawab dengan kodenya sendiri, dan pesan ORA-nya TIDAK ikut
// terkirim ke klien.
//
// # Kenapa uji ini ada
//
// Ia mengunci dua hal yang berlawanan arah, dan keduanya pernah dilanggar sekaligus.
//
// Sebelum 2026-10-01, galat basis data jatuh ke penulis galat umum dan sampai ke
// pengguna sebagai "Terjadi kesalahan pada sistem" — tanpa kode tersendiri, tanpa ID
// permintaan, tanpa satu pun hal yang dapat ditindaklanjuti. Pesan ORA-nya ada, tetapi
// hanya di konsol peladen.
//
// Godaan memperbaikinya adalah mengirim pesan ORA itu ke layar. Itu justru melanggar
// `11-CROSSCUTTING.md` §1.2 aturan 5: pesan ORA menyebut nama tabel dan kolom. Uji ini
// karena itu menuntut KEDUANYA — kode beserta ID permintaan terkirim, pesan ORA tidak.
func TestPenolakanBasisDataDijawabDenganKodeSendiri(t *testing.T) {
	var log bytes.Buffer
	h := &Handler{
		logger: slog.New(slog.NewJSONHandler(&log, nil)),
		writeResponse: func(w http.ResponseWriter, _ *http.Request, status int, body any) {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(status)
			_ = json.NewEncoder(w).Encode(body)
		},
	}

	// Bentuk galatnya sama persis dengan yang disusun Stream: penanda, lalu galat
	// aslinya, keduanya di dalam satu rantai.
	oracle := fmt.Errorf("ORA-00942: table or view POOLDATA.T_CLAIM_PNC does not exist")
	err := fmt.Errorf("%w: laporan %q kueri %q: %w",
		reportklaimsql.ErrQueryFailed, "pla", "report_pla", oracle)

	rec := httptest.NewRecorder()
	h.writeModuleError(rec, httptest.NewRequest(http.MethodGet, "/report-klaim/pla/ekspor", nil), err)

	require.Equal(t, http.StatusInternalServerError, rec.Code)

	var body struct {
		Kode  string `json:"kode"`
		Pesan string `json:"pesan"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, CodeQueryFailed, body.Kode)

	// Yang BOLEH sampai ke pengguna: arahan, dan nomor permintaan untuk dicocokkan.
	require.Contains(t, body.Pesan, "nomor permintaan")

	// Yang TIDAK BOLEH: isi pesan Oracle, termasuk nama tabelnya.
	require.NotContains(t, body.Pesan, "ORA-")
	require.NotContains(t, body.Pesan, "T_CLAIM_PNC")

	// Tetapi ia harus ADA di log — kalau tidak, kegagalannya tetap buntu.
	require.Contains(t, log.String(), "ORA-00942")
	require.Contains(t, log.String(), "T_CLAIM_PNC")
}

// Galat yang BUKAN penolakan basis data tetap diteruskan ke penulis galat umum.
//
// Tanpa uji ini, penanda baru dapat diam-diam menelan seluruh galat lain — dan
// perbedaan antara "basis data menolak" dan "ada cacat" hilang lagi, kali ini ke arah
// sebaliknya.
func TestGalatLainTetapDiteruskan(t *testing.T) {
	diteruskan := false
	h := &Handler{
		writeError: func(w http.ResponseWriter, _ *http.Request, _ error) {
			diteruskan = true
			w.WriteHeader(http.StatusInternalServerError)
		},
	}

	h.writeModuleError(
		httptest.NewRecorder(),
		httptest.NewRequest(http.MethodGet, "/report-klaim/pla/ekspor", nil),
		fmt.Errorf("cacat pemrograman mana pun"),
	)

	require.True(t, diteruskan, "galat tak dikenal harus diteruskan, bukan ditelan")
}

// Penanda ErrQueryFailed tidak boleh kehilangan galat aslinya saat dibungkus.
//
// Ia dibungkus dengan DUA kata kerja %w sekaligus, dan itu mudah rusak tanpa terlihat:
// mengganti salah satunya menjadi %v membuat salah satu sisi rantai putus — penanda
// hilang, atau pesan ORA-nya hilang. Keduanya gagal diam-diam.
func TestRantaiGalatMemuatKeduanya(t *testing.T) {
	oracle := fmt.Errorf("ORA-01008: not all variables bound")
	err := fmt.Errorf("%w: laporan %q kueri %q: %w",
		reportklaimsql.ErrQueryFailed, "tat", "report_tat", oracle)

	require.ErrorIs(t, err, reportklaimsql.ErrQueryFailed)
	require.ErrorIs(t, err, oracle)
	require.True(t, strings.Contains(err.Error(), "report_tat"),
		"nama kuerinya harus ikut, supaya log menunjuk kueri yang tepat")
}

package inboxsalvagehttp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxsalvage"
	"claim-pnc/internal/portal"
)

// Setiap konstanta Field… menunjuk isian Row yang benar. Kolom yang tidak dikenali menjadi
// teks kosong, bukan panik.
func TestGridCellReadsEveryKnownField(t *testing.T) {
	row := inboxsalvage.Row{
		ClaimNo: "c", SalvageID: "s", InputDate: "i", LossDate: "l", PIC: "p",
		BusinessName: "b", ObjectName: "o", SalvageType: "t", SalvageLocation: "loc",
		AuctionStatus: "a", EstimateValue: "e", Email: "m", Remark: "r",
		RequestValue: "rv", SubmissionType: "st", Aging: "ag", Note: "n",
	}

	cases := map[string]string{
		inboxsalvage.FieldClaimNo:         "c",
		inboxsalvage.FieldSalvageID:       "s",
		inboxsalvage.FieldInputDate:       "i",
		inboxsalvage.FieldLossDate:        "l",
		inboxsalvage.FieldPIC:             "p",
		inboxsalvage.FieldBusinessName:    "b",
		inboxsalvage.FieldObjectName:      "o",
		inboxsalvage.FieldSalvageType:     "t",
		inboxsalvage.FieldSalvageLocation: "loc",
		inboxsalvage.FieldAuctionStatus:   "a",
		inboxsalvage.FieldEstimateValue:   "e",
		inboxsalvage.FieldEmail:           "m",
		inboxsalvage.FieldRemark:          "r",
		inboxsalvage.FieldRequestValue:    "rv",
		inboxsalvage.FieldSubmissionType:  "st",
		inboxsalvage.FieldAging:           "ag",
		inboxsalvage.FieldNote:            "n",
		"kolom_tidak_dikenal":             "",
	}
	for key, want := range cases {
		require.Equalf(t, want, gridCell(row, key), "kolom %q", key)
	}
}

func TestExportFilenameFallsBackWhenTheTabHasNoCode(t *testing.T) {
	require.Equal(t, "inbox-salvage.csv", exportFilename(inboxsalvage.Tab{}))
	require.Equal(t, "inbox-salvage-tba.csv", exportFilename(inboxsalvage.Tab{Code: "tba"}))
}

func TestTruncationNoticeWithoutColumnsIsEmpty(t *testing.T) {
	require.Empty(t, truncationNotice(0, 10))

	notice := truncationNotice(2, 60_000)
	require.Len(t, notice, 2)
	require.Contains(t, notice[0], "50000 baris dari 60000")
	require.Empty(t, notice[1])
}

// Handler yang dipanggil TANPA middleware portal tetap menolak, bukan melayani portal
// bawaan. Penjagaan ini berlapis: middleware menolak lebih dulu, handler menolak lagi.
func TestHandlersRejectARequestThatNeverPassedThePortalCheck(t *testing.T) {
	var captured error
	handler := NewHandler(Options{
		WriteJSON: func(w http.ResponseWriter, _ *http.Request, status int, body any) {
			w.WriteHeader(status)
			_ = json.NewEncoder(w).Encode(body)
		},
		FallbackErrorWriter: func(w http.ResponseWriter, _ *http.Request, err error) {
			captured = err
			w.WriteHeader(http.StatusBadRequest)
		},
	})

	for name, serve := range map[string]http.HandlerFunc{
		"metadata": handler.Metadata,
		"tindakan": handler.RejectWrite,
		"daftar":   handler.List,
	} {
		t.Run(name, func(t *testing.T) {
			captured = nil
			recorder := httptest.NewRecorder()
			serve(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
			require.Equal(t, http.StatusBadRequest, recorder.Code)
			require.ErrorIs(t, captured, portal.ErrNotStated)
		})
	}
}

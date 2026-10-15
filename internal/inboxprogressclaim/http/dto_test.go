package inboxprogressclaimhttp

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxprogressclaim"
	"claim-pnc/internal/inboxprogressclaim/usecase"
)

// day membentuk tanggal UTC tanpa jam.
func day(year int, month time.Month, date int) *time.Time {
	at := time.Date(year, month, date, 0, 0, 0, 0, time.UTC)
	return &at
}

// viewOf mengambil satu region beserta kolomnya.
func viewOf(t *testing.T, code string) inboxprogressclaim.View {
	t.Helper()

	view, found := inboxprogressclaim.FindView(code)
	require.True(t, found)
	return view
}

func TestPositionsAreJoinedPositionally(t *testing.T) {
	// Padanan antarkolom bersifat positional: nilai ke-n pada `posisi` sepadan dengan
	// nilai ke-n pada kedua status dan pada tenggatnya. Posisi yang belum punya tenggat
	// karena itu menyumbang teks KOSONG, bukan dilewati — melewatinya akan menggeser
	// seluruh nilai sesudahnya dan memasangkan tenggat sebuah posisi dengan nama posisi
	// yang lain.
	row := toClaimRowDTO(inboxprogressclaim.ClaimRow{
		Positions: []inboxprogressclaim.Position{
			{Name: "SURVEY", Status1: "Berjalan", Status2: "", NextFollowUp: nil},
			{
				Name: "KOMITE", Status1: "", Status2: "Berkas lengkap",
				NextFollowUp: day(2026, time.September, 19),
			},
		},
	})

	require.Equal(t, "SURVEY, KOMITE", row.Position)
	require.Equal(t, "Berjalan, ", row.ProgressStatus1)
	require.Equal(t, ", Berkas lengkap", row.ProgressStatus2)
	require.Equal(t, ", 2026-09-19", row.NextFollowUp)
	require.Equal(t, 2, row.PositionCount)
}

func TestClaimWithoutPositionsSendsEmptyText(t *testing.T) {
	row := toClaimRowDTO(inboxprogressclaim.ClaimRow{ClaimNumber: "PNCN.26.0104"})

	require.Empty(t, row.Position)
	require.Empty(t, row.NextFollowUp)
	require.Equal(t, 0, row.PositionCount)
}

func TestDatesAreSentWithoutTime(t *testing.T) {
	// Jam sengaja tidak dikirim: yang dikirim sistem lama SALAH — `hh:mm:ss` pada Oracle
	// menempatkan BULAN di bagian menit.
	row := toClaimRowDTO(inboxprogressclaim.ClaimRow{
		RegisterDate: day(2026, time.August, 3),
		LossDate:     nil,
	})

	require.Equal(t, "2026-08-03", *row.RegisterDate)
	require.Nil(t, row.LossDate, "tanggal kosong dikirim null, bukan teks kosong")
}

func TestClaimViewSendsClaimRows(t *testing.T) {
	response := toListResponse(usecase.Listed{
		Query: inboxprogressclaim.Query{View: viewOf(t, inboxprogressclaim.ViewOutstanding)},
		Claims: inboxprogressclaim.ClaimPage{
			Items:      []inboxprogressclaim.ClaimRow{{ClaimNumber: "PNCN.26.0101"}},
			Total:      1,
			Pagination: inboxprogressclaim.Pagination{Page: 1, Size: 15},
		},
	}, "asm")

	rows, ok := response.Rows.([]ClaimRowDTO)
	require.True(t, ok, "region klaim harus mengirim baris klaim")
	require.Len(t, rows, 1)
	require.Equal(t, 1, response.Pagination.Total)
	require.Equal(t, "asm", response.Portal)
}

func TestPerPICViewSendsSummaryRowsInOnePage(t *testing.T) {
	// Rekap ini tidak dipaginasi, mengikuti sistem lama. Keterangan halamannya tetap diisi
	// supaya layar tidak perlu bercabang saat menampilkan jumlah baris.
	response := toListResponse(usecase.Listed{
		Query:   inboxprogressclaim.Query{View: viewOf(t, inboxprogressclaim.ViewPerPIC)},
		PICRows: []inboxprogressclaim.PICSummary{{PIC: "ADMINKLAIM", ClaimCount: 12}},
	}, "asm")

	rows, ok := response.Rows.([]PICRowDTO)
	require.True(t, ok, "rekap per PIC harus mengirim baris rekap")
	require.Len(t, rows, 1)
	require.Equal(t, 12, rows[0].ClaimCount)
	require.Equal(t, 1, response.Pagination.TotalPages)
}

func TestRowsAreNeverNull(t *testing.T) {
	// `[]` dan `null` ditangani berbeda oleh klien, dan yang kedua memaksa setiap layar
	// memeriksanya lebih dulu. Region Evaluasi adalah yang paling mudah terlewat karena
	// ia memang tidak punya data.
	for _, code := range []string{
		inboxprogressclaim.ViewOutstanding,
		inboxprogressclaim.ViewPerPIC,
		inboxprogressclaim.ViewEvaluation,
	} {
		response := toListResponse(usecase.Listed{
			Query: inboxprogressclaim.Query{View: viewOf(t, code)},
		}, "asm")

		encoded, err := json.Marshal(response)
		require.NoError(t, err)
		require.NotContainsf(t, string(encoded), `"baris":null`,
			"bagian %s mengirim baris null", code)
	}
}

func TestDeadControlsRideAlongWithTheirView(t *testing.T) {
	// Pengguna yang menekan penyaring lalu melihat hasil yang tidak berubah akan
	// melaporkannya sebagai kerusakan, berulang kali, sampai ada yang menjelaskan bahwa
	// memang begitu di sistem lama.
	outstanding := toViewDTO(viewOf(t, inboxprogressclaim.ViewOutstanding))
	require.Len(t, outstanding.DeadControls, 2)
	require.NotEmpty(t, outstanding.DeadControls[0].Reason)

	perPIC := toViewDTO(viewOf(t, inboxprogressclaim.ViewPerPIC))
	require.Empty(t, perPIC.DeadControls)
	require.True(t, perPIC.SupportsBusinessFilter)
}

func TestColumnsCarryBothKeyAndField(t *testing.T) {
	// Region Outstanding menggambar SATU isian di DUA kolom. Bila identitas kolom diambil
	// dari nama isiannya, kedua kolom itu bertabrakan di layar.
	view := toViewDTO(viewOf(t, inboxprogressclaim.ViewOutstanding))

	keys := map[string]bool{}
	fields := 0
	for _, column := range view.Columns {
		require.NotEmpty(t, column.Field)
		require.False(t, keys[column.Key], "kunci kolom tidak boleh ganda")
		keys[column.Key] = true

		if column.Field == inboxprogressclaim.FieldRegisterDate {
			fields++
		}
	}
	require.Equal(t, 2, fields)
}

func TestMetadataCarriesLimitations(t *testing.T) {
	// Keterbatasan dikirim sebagai DATA supaya ia hilang dengan sendirinya begitu
	// penghalangnya hilang — tanpa menyunting frontend.
	meta := toMetadataResponse(usecase.Metadata{
		Views:       inboxprogressclaim.Views(),
		DefaultView: inboxprogressclaim.DefaultView,
		PageSize:    inboxprogressclaim.DefaultPageSize,
	}, "asm")

	require.NotEmpty(t, meta.Limitations)
	require.Equal(t, "asm", meta.Portal)
	require.Equal(t, 15, meta.PageSize)
}

func TestValidationErrorBecomes422WithDetails(t *testing.T) {
	// 422, bukan 400: permintaannya berbentuk benar, isinya yang melanggar aturan.
	// Frontend menanganinya berbeda.
	status, body, recognized := mapError(inboxprogressclaim.NewValidationError(
		[]inboxprogressclaim.Violation{
			{Field: inboxprogressclaim.FieldBusiness, Message: "Pilih lini bisnis lebih dulu."},
		}))

	require.True(t, recognized)
	require.Equal(t, http.StatusUnprocessableEntity, status)
	require.Equal(t, CodeValidationFail, body.Code)
	require.Len(t, body.Details, 1)
	require.Equal(t, "bisnis", body.Details[0].Field)
}

func TestUnknownCallerBecomes409(t *testing.T) {
	// 409, bukan 401: sesinya sah — yang tidak lengkap adalah profil di dalamnya.
	// Menjawab 401 akan membuat layar melempar pengguna ke halaman masuk, lalu
	// mengembalikannya ke galat yang sama.
	status, body, recognized := mapError(inboxprogressclaim.ErrCallerUnknown)

	require.True(t, recognized)
	require.Equal(t, http.StatusConflict, status)
	require.Equal(t, CodeCallerUnknown, body.Code)
}

func TestForeignErrorIsHandedOver(t *testing.T) {
	// Galat yang bukan milik modul ini harus dapat dibedakan dari galat internal, supaya
	// penulis galat portal dan auth tetap kebagian menjawabnya.
	_, _, recognized := mapError(errors.New("galat milik modul lain"))
	require.False(t, recognized)
}

package sqlstore

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxinvestigator"
)

var exportColumns = []string{
	"INVESTIGATED_AT", "HOSPITAL_ADDRESS", "PAID_BY_OTHER_INSURER", "PAID_BY_PATIENT",
	"PAID_BY_COMPANY", "NO_PAYMENT", "INVESTIGATED", "RECEIPT_CONFIRMATION",
	"MEDICAL_RECORD_NUMBER", "PHONE_CALLED", "PATIENT_REGISTERED", "REMARKS",
	"HOSPITAL_KIND",
}

func sampleFilter() inboxinvestigator.ExportFilter {
	return inboxinvestigator.ExportFilter{
		From:         time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
		To:           time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC),
		Investigated: inboxinvestigator.InvestigatedYes,
	}
}

// Batas atas digeser SATU HARI sebelum dikirim ke basis data.
//
// Itulah yang membuat hari yang diketik pengguna IKUT terbawa seluruhnya, meski kuerinya
// membandingkan dengan tanda kurang-dari. Tanpa pergeseran itu, klaim yang ditransfer pada
// hari "Sampai" akan hilang dari berkas tanpa satu pun tanda.
func TestExportShiftsTheUpperBoundByOneDay(t *testing.T) {
	repo, mock := newMock(t)
	filter := sampleFilter()

	mock.ExpectQuery(sqlNamed("investigator_export")).
		WithArgs(
			filter.From,
			time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
			inboxinvestigator.InvestigatedYes,
			inboxinvestigator.MaxExportRows+1,
		).
		WillReturnRows(sqlmock.NewRows(exportColumns))

	rows, err := repo.Export(context.Background(), filter)
	require.NoError(t, err)
	require.Empty(t, rows)
	require.NoError(t, mock.ExpectationsWereMet())
}

// Ketiga belas kolom dipetakan ke field yang benar, dan NULL menjadi teks kosong.
func TestExportMapsEveryColumn(t *testing.T) {
	repo, mock := newMock(t)
	investigated := time.Date(2026, 9, 21, 2, 15, 0, 0, time.UTC)

	mock.ExpectQuery(sqlNamed("investigator_export")).
		WillReturnRows(sqlmock.NewRows(exportColumns).
			AddRow(investigated, " Jalan Contoh 1 ", "false", "true", "false", "false",
				"1", "1", " RM-1 ", "021-0", "1", " Catatan ", "1").
			AddRow(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil))

	rows, err := repo.Export(context.Background(), sampleFilter())
	require.NoError(t, err)
	require.Equal(t, []inboxinvestigator.ExportRow{
		{
			InvestigatedAt:      &investigated,
			HospitalAddress:     "Jalan Contoh 1",
			PaidByOtherInsurer:  "false",
			PaidByPatient:       "true",
			PaidByCompany:       "false",
			NoPayment:           "false",
			Investigated:        "1",
			ReceiptConfirmation: "1",
			MedicalRecordNumber: "RM-1",
			PhoneCalled:         "021-0",
			PatientRegistered:   "1",
			Remarks:             "Catatan",
			HospitalKindCode:    "1",
		},
		{},
	}, rows)
	require.NoError(t, mock.ExpectationsWereMet())
}

// Penyaring yang tidak sah ditolak SEBELUM basis data disentuh.
//
// Menyentuhnya lebih dulu berarti menjalankan kueri yang sudah pasti tidak berguna, dan
// pada rentang yang lebar itu bukan biaya yang kecil.
func TestExportRejectsAnInvalidFilterWithoutQuerying(t *testing.T) {
	repo, mock := newMock(t)

	_, err := repo.Export(context.Background(), inboxinvestigator.ExportFilter{})
	require.ErrorIs(t, err, inboxinvestigator.ErrExportFilterInvalid)
	require.NoError(t, mock.ExpectationsWereMet())
}

// Galat basis data dibungkus dengan nama modul, bukan diteruskan telanjang.
func TestExportWrapsDatabaseFailure(t *testing.T) {
	repo, mock := newMock(t)
	mock.ExpectQuery(sqlNamed("investigator_export")).WillReturnError(errOracle)

	_, err := repo.Export(context.Background(), sampleFilter())
	require.ErrorIs(t, err, errOracle)
	require.Contains(t, err.Error(), "inboxinvestigator/sqlstore")
}

// Waktu Pega di dalam JSON diurai menjadi UTC, dan yang tidak terbaca menjadi nil.
//
// Nil, bukan galat: dokumen JSON ditulis sistem lain yang masih berjalan, dan satu dokumen
// bernilai aneh harus membuat SATU sel kosong — bukan menggagalkan seluruh antrean.
func TestParsePegaMoment(t *testing.T) {
	for name, probe := range map[string]struct {
		input string
		want  *time.Time
	}{
		"bentuk baku": {
			input: "20260922T010000.000 GMT",
			want:  ptr(time.Date(2026, 9, 22, 1, 0, 0, 0, time.UTC)),
		},
		"berspasi di kedua ujung": {
			input: "  20260922T010000.000 GMT  ",
			want:  ptr(time.Date(2026, 9, 22, 1, 0, 0, 0, time.UTC)),
		},
		"tanpa akhiran zona": {
			input: "20260922T010000.000",
			want:  ptr(time.Date(2026, 9, 22, 1, 0, 0, 0, time.UTC)),
		},
		"kosong":         {input: "", want: nil},
		"hanya spasi":    {input: "   ", want: nil},
		"bukan waktu":    {input: "belum disurvei", want: nil},
		"bentuk lain":    {input: "2026-09-22T01:00:00Z", want: nil},
		"tanggal mentah": {input: "20260922", want: nil},
	} {
		t.Run(name, func(t *testing.T) {
			got := parsePegaMoment(probe.input)
			if probe.want == nil {
				require.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			require.True(t, probe.want.Equal(*got), "%v != %v", probe.want, got)
			require.Equal(t, time.UTC, got.Location())
		})
	}
}

func ptr(moment time.Time) *time.Time { return &moment }

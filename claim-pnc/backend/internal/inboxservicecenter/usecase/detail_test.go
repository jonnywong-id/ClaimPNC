package usecase_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxservicecenter"
	"claim-pnc/internal/inboxservicecenter/repo/memory"
	"claim-pnc/internal/inboxservicecenter/usecase"
)

func detail(
	t *testing.T,
	service *usecase.Service,
	id, login string,
) (usecase.Detailed, error) {
	t.Helper()
	return service.Detail(
		context.Background(),
		portalUtama,
		inboxservicecenter.Caller{Login: login},
		id,
	)
}

func TestRincianTerbukaBagiPICNyaSendiri(t *testing.T) {
	service := newService(t, memory.NewSampleStore())

	got, err := detail(t, service, "SC-000101", memory.SampleOwner)
	require.NoError(t, err)

	require.Equal(t, "SC-000101", got.Claim.ID)
	require.Equal(t, "1000101", got.Claim.RepairID)

	// Ketujuh kelompok ikut dikirim supaya layar tidak perlu memuat daftarnya sendiri.
	require.Len(t, got.Groups, 7)
}

// TestRincianMilikPICLainDijawabTidakDitemukan adalah uji kebocoran, bukan uji penyaring.
//
// Tanpa penyaring PIC, `SC-000103` dapat dibaca siapa pun yang sudah masuk hanya dengan
// menaikkan angka pada jalurnya — beserta nama nasabah dan nomor IMEI-nya.
func TestRincianMilikPICLainDijawabTidakDitemukan(t *testing.T) {
	service := newService(t, memory.NewSampleStore())

	_, err := detail(t, service, "SC-000103", memory.SampleOwner)
	require.ErrorIs(t, err, inboxservicecenter.ErrNotFound)
}

// TestKlaimTidakAdaDanBukanMilikSendiriDijawabSAMA — membedakan keduanya memberi tahu
// penanya bahwa sebuah ID nyata.
func TestKlaimTidakAdaDanBukanMilikSendiriDijawabSama(t *testing.T) {
	service := newService(t, memory.NewSampleStore())

	_, adaTapiBukanMilikSaya := detail(t, service, "SC-000103", memory.SampleOwner)
	_, memangTidakAda := detail(t, service, "SC-999999", memory.SampleOwner)

	require.ErrorIs(t, adaTapiBukanMilikSaya, inboxservicecenter.ErrNotFound)
	require.ErrorIs(t, memangTidakAda, inboxservicecenter.ErrNotFound)
	require.Equal(t, adaTapiBukanMilikSaya.Error(), memangTidakAda.Error())
}

func TestRincianTanpaIDDitolakSebagaiGalatValidasi(t *testing.T) {
	service := newService(t, memory.NewSampleStore())

	_, err := detail(t, service, "   ", memory.SampleOwner)

	var validation *inboxservicecenter.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Equal(t, inboxservicecenter.FieldID, validation.Violations[0].Field)
}

func TestRincianTanpaIdentitasDitolak(t *testing.T) {
	service := newService(t, memory.NewSampleStore())

	_, err := detail(t, service, "SC-000101", "")
	require.ErrorIs(t, err, inboxservicecenter.ErrCallerUnknown)
}

// TestRiwayatProgresDiambilDenganRepairIDBukanID mengunci pembedaan yang paling mudah
// tertukar: tabel riwayat dikunci REPAIRID (`1000101`), bukan ID (`SC-000101`).
//
// Bila tertukar, riwayatnya selalu kosong dan tidak ada satu pun galat yang muncul.
func TestRiwayatProgresDiambilDenganRepairIDBukanID(t *testing.T) {
	service := newService(t, memory.NewSampleStore())

	got, err := detail(t, service, "SC-000101", memory.SampleOwner)
	require.NoError(t, err)

	require.Len(t, got.Progress, 2)
	require.Equal(t, "1000101", got.Progress[0].ClaimID)

	// Terlama di atas — `ORDER BY INSERTDATE ASC`, seperti kueri lamanya.
	require.Contains(t, got.Progress[0].Note, "Unit diterima")
	require.Contains(t, got.Progress[1].Note, "Pengecekan selesai")
}

// TestRiwayatKosongBukanGalat — klaim yang belum pernah dicatat progresnya memang tidak punya
// barisnya, dan itu keadaan biasa pada tab Registrasi SC.
func TestRiwayatKosongBukanGalat(t *testing.T) {
	service := newService(t, memory.NewSampleStore())

	got, err := detail(t, service, "SC-000102", memory.SampleOwner)
	require.NoError(t, err)

	require.Empty(t, got.Progress)
	require.NotNil(t, got.Progress, "senarai kosong, bukan nil")
}

// TestRincianTerisiRapatTerbacaPadaKetujuhKelompok memastikan isian contoh benar-benar
// menembus seluruh lapisan — bukan hanya kelompok pertama.
func TestRincianTerisiRapatTerbacaPadaKetujuhKelompok(t *testing.T) {
	service := newService(t, memory.NewSampleStore())

	got, err := detail(t, service, "SC-000101", memory.SampleOwner)
	require.NoError(t, err)

	require.Equal(t, "SIMAS INSURTECH", got.Claim.Insurance) // General Information
	require.Equal(t, "Contoh Model X", got.Claim.Model)      // Informasi Unit
	require.Equal(t, "LCD-01", got.Claim.SymptomCode)        // Informasi Perbaikan
	require.NotNil(t, got.Claim.AcknowledgeDate)             // Estimasi Date
	require.Equal(t, "2214500", got.Claim.TotalFee)          // Estimasi Biaya
	require.Equal(t, "Y", got.Claim.ChargerAdaptor)          // Accessories Unit
	require.Equal(t, "Repair Submitted", got.Claim.RepairStatusName())
	require.Equal(t, "Belum diajukan", got.Claim.ApprovalStatusName())
}

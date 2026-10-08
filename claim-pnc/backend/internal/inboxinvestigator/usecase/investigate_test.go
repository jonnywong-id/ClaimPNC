package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxinvestigator"
	"claim-pnc/internal/inboxinvestigator/repo/memory"
)

func emptyQueue() inboxinvestigator.Repo { return memory.NewRepo(memory.Options{}) }

// form menyusun formulir terisi minimal yang sah.
func form(ref string) inboxinvestigator.Investigation {
	return inboxinvestigator.Investigation{
		ClaimRef:     ref,
		Investigated: inboxinvestigator.InvestigatedYes,
	}
}

// Tanggal Investigasi diisi saat formulir DIBUKA, bukan saat disimpan.
//
// Itu pra-proses `PresetInvestigation` pada Flow Action `InputInvestigator`:
// `TanggalInvestigasi := @CurrentDateTime()`.
func TestMembukaFormulirMengisiTanggalInvestigasi(t *testing.T) {
	service, _ := serviceAndForms(t, emptyQueue())

	one, err := service.OpenInvestigation(context.Background(), "asm", "ref-1")
	require.NoError(t, err)
	require.NotNil(t, one.InvestigatedAt)
	require.True(t, formAt.Equal(*one.InvestigatedAt))
	require.Equal(t, "ref-1", one.ClaimRef)
	require.Equal(t, 1, one.SurveyIndex)
	require.Equal(t, 1, one.Index)
}

// Formulir yang dibuka ULANG mempertahankan tanggal investigasi pertamanya.
//
// Menimpanya setiap kali dibuka akan membuat tanggal itu selalu menunjuk hari ini, dan
// kolom "Tanggal Investigasi" pada berkas ekspor kehilangan maknanya.
func TestMembukaUlangTidakMenimpaTanggalInvestigasi(t *testing.T) {
	service, forms := serviceAndForms(t, emptyQueue())

	earlier := time.Date(2026, 9, 1, 2, 0, 0, 0, time.UTC)
	first := form("ref-1")
	first.InvestigatedAt = &earlier
	require.NoError(t, forms.Save(context.Background(), first,
		inboxinvestigator.Transition{}, "uji"))

	again, err := service.OpenInvestigation(context.Background(), "asm", "ref-1")
	require.NoError(t, err)
	require.NotNil(t, again.InvestigatedAt)
	require.True(t, earlier.Equal(*again.InvestigatedAt),
		"tanggal investigasi pertama tertimpa")
}

// Menyimpan formulir MEMINDAHKAN klaim ke Analyst.
//
// Ketiga nilainya dibaca dari `SetStatusInvestigator_Act`. Uji ini yang gagal bila
// salah satunya berubah tanpa bukti baru.
func TestMenyimpanMemindahkanKlaimKeAnalyst(t *testing.T) {
	service, forms := serviceAndForms(t, emptyQueue())

	move, err := service.SubmitInvestigation(
		context.Background(), "asm", form("ref-1"), false, "ADMINUJI")
	require.NoError(t, err)

	require.Equal(t, "5", move.SurveyStatus)
	require.Equal(t, "5", move.PNCStatus)
	require.Equal(t, "1151", move.ClaimStatus)
	require.True(t, formAt.Equal(move.At))

	stored, found := forms.MoveOf("ref-1")
	require.True(t, found, "perpindahan tidak tercatat di penyimpanan")
	require.Equal(t, move, stored)
}

// Satu waktu dipakai untuk Tanggal Investigasi DAN perpindahan.
//
// Dua pemanggilan jam dapat berbeda sedetik, dan selisih itu terbaca sebagai klaim yang
// berpindah ke Analyst sebelum investigasinya selesai.
func TestSatuWaktuUntukTanggalDanPerpindahan(t *testing.T) {
	service, forms := serviceAndForms(t, emptyQueue())

	move, err := service.SubmitInvestigation(
		context.Background(), "asm", form("ref-1"), false, "ADMINUJI")
	require.NoError(t, err)

	saved, found, err := forms.Load(context.Background(), "ref-1")
	require.NoError(t, err)
	require.True(t, found)
	require.NotNil(t, saved.InvestigatedAt)
	require.True(t, move.At.Equal(*saved.InvestigatedAt))
}

// Isian yang TIDAK tampil dikosongkan sebelum disimpan.
//
// Pengguna yang mengisi "Nama Rumah Sakit" lalu mengubah tempat kejadian menjadi "bukan
// rumah sakit" meninggalkan nilai yang tidak lagi terlihat di layarnya; menyimpannya
// berarti menyimpan sesuatu yang tidak dapat dikoreksi siapa pun.
func TestIsianYangTidakTampilDikosongkan(t *testing.T) {
	service, forms := serviceAndForms(t, emptyQueue())

	one := form("ref-1")
	one.HospitalKindCode = "0" // bukan rumah sakit
	one.HospitalName = "RS Contoh"
	one.HospitalAddress = "Jalan Contoh"
	one.OtherPlaceName = "Tempat Contoh"
	one.PaidByCompany = "false"
	one.PaidByOtherInsurer = "false"
	one.OtherInsurer = "Asuransi Contoh"

	_, err := service.SubmitInvestigation(context.Background(), "asm", one, false, "ADMINUJI")
	require.NoError(t, err)

	saved, found, err := forms.Load(context.Background(), "ref-1")
	require.NoError(t, err)
	require.True(t, found)

	require.Empty(t, saved.HospitalName, "Nama Rumah Sakit tidak tampil, harus kosong")
	require.Empty(t, saved.HospitalAddress, "Alamat RS/Klinik tidak tampil, harus kosong")
	require.Empty(t, saved.OtherInsurer, "Nama Asuransi Lain tidak tampil, harus kosong")
	require.Equal(t, "Tempat Contoh", saved.OtherPlaceName, "yang tampil justru dikosongkan")
}

// Isian yang tampil TIDAK dikosongkan.
func TestIsianYangTampilDipertahankan(t *testing.T) {
	service, forms := serviceAndForms(t, emptyQueue())

	one := form("ref-1")
	one.HospitalKindCode = "1" // rumah sakit
	one.HospitalName = "RS Contoh"
	one.HospitalAddress = "Jalan Contoh"
	one.PaidByOtherInsurer = "true"
	one.OtherInsurer = "Asuransi Contoh"

	_, err := service.SubmitInvestigation(context.Background(), "asm", one, true, "ADMINUJI")
	require.NoError(t, err)

	saved, _, err := forms.Load(context.Background(), "ref-1")
	require.NoError(t, err)
	require.Equal(t, "RS Contoh", saved.HospitalName)
	require.Equal(t, "Jalan Contoh", saved.HospitalAddress)
	require.Equal(t, "Asuransi Contoh", saved.OtherInsurer)
}

// Formulir tanpa pilihan "Dapat Diinvestigasi" DITOLAK, dan tidak disimpan.
//
// Nilainya menentukan isi berkas Export Data Investigation; baris yang nilainya kosong
// tidak akan pernah muncul di berkas mana pun — hilang tanpa satu pun tanda.
func TestFormulirTanpaPilihanInvestigasiDitolak(t *testing.T) {
	service, forms := serviceAndForms(t, emptyQueue())

	one := form("ref-1")
	one.Investigated = ""

	_, err := service.SubmitInvestigation(context.Background(), "asm", one, false, "ADMINUJI")
	require.ErrorIs(t, err, inboxinvestigator.ErrInvestigationInvalid)
	require.Contains(t, err.Error(), "Dapat Diinvestigasi")

	_, found, err := forms.Load(context.Background(), "ref-1")
	require.NoError(t, err)
	require.False(t, found, "formulir yang ditolak tetap tersimpan")
}

// Portal yang tidak dilayani ditolak SEBELUM satu baris pun ditulis.
func TestPortalTidakDikenalDitolakPadaJalurTulis(t *testing.T) {
	service, _ := serviceAndForms(t, emptyQueue())

	_, err := service.SubmitInvestigation(
		context.Background(), "asi", form("ref-1"), false, "ADMINUJI")
	require.Error(t, err)

	_, err = service.OpenInvestigation(context.Background(), "asi", "ref-1")
	require.Error(t, err)
}

// Tabel yang belum ada dikembalikan sebagai galat yang DAPAT DIKENALI.
//
// Lapisan transport memetakannya menjadi 503 beserta kalimat yang menyebut administrator;
// tanpa pembedaan itu, petugas membaca "sistem rusak" dan mencoba berulang kali.
func TestTabelBelumAdaDapatDikenali(t *testing.T) {
	service, forms := serviceAndForms(t, emptyQueue())
	forms.SetError(inboxinvestigator.ErrInvestigationStoreMissing)

	_, err := service.SubmitInvestigation(
		context.Background(), "asm", form("ref-1"), false, "ADMINUJI")
	require.ErrorIs(t, err, inboxinvestigator.ErrInvestigationStoreMissing)
}

// Kegagalan penyimpanan lain DIBUNGKUS, bukan diteruskan telanjang.
func TestKegagalanPenyimpananDibungkus(t *testing.T) {
	service, forms := serviceAndForms(t, emptyQueue())
	failure := errors.New("oracle mati")
	forms.SetError(failure)

	_, err := service.SubmitInvestigation(
		context.Background(), "asm", form("ref-1"), false, "ADMINUJI")
	require.ErrorIs(t, err, failure)
	require.Contains(t, err.Error(), "menyimpan hasil investigasi")
}

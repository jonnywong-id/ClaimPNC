package usecase_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/mastersurveyors"
	"claim-pnc/internal/mastersurveyors/account"
	"claim-pnc/internal/mastersurveyors/committee"
	"claim-pnc/internal/mastersurveyors/repo/memory"
	"claim-pnc/internal/mastersurveyors/usecase"
)

const portalAlias = "ASM"

// newService merakit layanan lengkap di atas penyimpanan memori.
//
// Pengisi seam akunnya dikembalikan juga supaya uji dapat memeriksa APA yang diminta —
// bukan hanya bahwa permintaannya tidak gagal.
func newService(t *testing.T) (*usecase.Service, *account.Recorder) {
	t.Helper()

	// SATU penyimpanan per uji, dibentuk sekali lalu ditutup dalam closure.
	//
	// Membentuknya ulang di dalam selector akan membuat baris yang baru disimpan tidak
	// terbaca pada langkah berikutnya — dan sebagian uji di berkas ini akan lulus karena
	// alasan yang salah.
	repo := memory.NewRepo()

	recorder := account.NewRecorder(nil)
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (mastersurveyors.Repo, error) {
			require.Equal(t, portalAlias, alias)
			return repo, nil
		},
		Committee: committee.Fixed{Identity: "KOMITEUJI"},
		Accounts:  recorder,
	})
	require.NoError(t, err)
	return service, recorder
}

func submission() usecase.Submission {
	return usecase.Submission{
		TypeCode: "1002",
		Name:     "Adjuster Baru",
		Address:  "Jalan Uji Nomor 1",
		Phone:    "0210000099",
		Email:    "adjuster.baru@contoh.invalid",
	}
}

func submitter() usecase.Submitter {
	return usecase.Submitter{Identity: "3171999", Name: "Petugas Contoh"}
}

// TestSubmitMenetapkanStatusMenungguDanKomite menguji dua hal yang dilakukan langkah 6
// dan 7 `CNMInsertDetailSurveyors_act`: menetapkan komite, dan menaruh barisnya di posisi
// menunggu.
func TestSubmitMenetapkanStatusMenungguDanKomite(t *testing.T) {
	service, _ := newService(t)

	saved, err := service.Submit(context.Background(), portalAlias, submission(), submitter())
	require.NoError(t, err)

	require.Equal(t, mastersurveyors.StatusPending, saved.Status)
	require.Equal(t, "KOMITEUJI", saved.Committee)
	require.Equal(t, "3171999", saved.CreatedBy)

	// ID dibuat penyimpanan, tidak pernah datang dari pemanggil.
	require.NotEmpty(t, saved.ID)
}

// TestSubmitMenolakNamaGandaTanpaPeduliSpasiDanHuruf menguji aturan dari
// `ValidasiMasterSurveyor` sampai ke tingkat alur, bukan hanya fungsi NameKey.
func TestSubmitMenolakNamaGandaTanpaPeduliSpasiDanHuruf(t *testing.T) {
	service, _ := newService(t)

	_, err := service.Submit(context.Background(), portalAlias, submission(), submitter())
	require.NoError(t, err)

	kembar := submission()
	kembar.Name = "  adjusterbaru  "

	_, err = service.Submit(context.Background(), portalAlias, kembar, submitter())
	require.ErrorIs(t, err, mastersurveyors.ErrNameTaken)
}

// TestSubmitMemintaAkunHanyaBilaAdaLoginAplikasi menguji bahwa surveyor eksternal tanpa
// login TIDAK memicu permintaan akun.
//
// Memintanya tanpa nama pengguna akan gagal dengan alasan yang menyesatkan, dan surveyor
// eksternal memang tidak masuk ke aplikasi ini.
func TestSubmitMemintaAkunHanyaBilaAdaLoginAplikasi(t *testing.T) {
	service, recorder := newService(t)

	_, err := service.Submit(context.Background(), portalAlias, submission(), submitter())
	require.NoError(t, err)
	require.Empty(t, recorder.Recorded(), "surveyor tanpa login tidak boleh meminta akun")

	internal := submission()
	internal.TypeCode = mastersurveyors.InternalTypeCode
	internal.Name = "Surveyor Internal Baru"
	internal.AppLogin = "SURVEYORBARU"

	_, err = service.Submit(context.Background(), portalAlias, internal, submitter())
	require.NoError(t, err)

	recorded := recorder.Recorded()
	require.Len(t, recorded, 1)
	require.Equal(t, portalAlias, recorded[0].PortalAlias)
	require.Equal(t, "SURVEYORBARU", recorded[0].Request.UserID)
	require.Equal(t, "SURVEYORBARU123456", recorded[0].Request.Password)
	require.True(t, recorded[0].Request.MustChangePassword)
	require.Equal(t, "Internal", recorded[0].Request.Unit)
}

// TestSubmitMenolakSurveyorInternalTanpaLogin menguji langkah 8 di tingkat alur.
func TestSubmitMenolakSurveyorInternalTanpaLogin(t *testing.T) {
	service, _ := newService(t)

	in := submission()
	in.TypeCode = mastersurveyors.InternalTypeCode
	in.Name = "Internal Tanpa Login"

	_, err := service.Submit(context.Background(), portalAlias, in, submitter())

	var violation *mastersurveyors.ValidationError
	require.ErrorAs(t, err, &violation)
	require.Contains(t, violation.Field, "login_aplikasi")
}

// TestUpdateMempertahankanKomiteDanPengaju menguji field mana yang dibawa apa adanya.
func TestUpdateMempertahankanKomiteDanPengaju(t *testing.T) {
	service, _ := newService(t)

	saved, err := service.Submit(context.Background(), portalAlias, submission(), submitter())
	require.NoError(t, err)

	changed := submission()
	changed.Address = "Jalan Uji Nomor 2"

	updated, err := service.Update(context.Background(), portalAlias, saved.ID, changed, submitter())
	require.NoError(t, err)

	require.Equal(t, "Jalan Uji Nomor 2", updated.Address)
	require.Equal(t, saved.Committee, updated.Committee)
	require.Equal(t, saved.CreatedBy, updated.CreatedBy)
	require.Equal(t, "3171999", updated.UpdatedBy)
}

// TestUpdateMengembalikanSurveyorYangSudahDisetujuiKeAntreanKomite adalah uji pengaman yang
// paling penting di berkas ini.
//
// Ia meniru langkah 4 `CNMInsertDetailSurveyors_act`, yang berjalan TANPA SYARAT
// (`pyStepsPreCondition: true`) dan menyetel `APPROVAL := Param.approval` — sementara
// tombol Simpan pada tab Approve dan Reject mengirim `approval = "0"`.
//
// Tanpa perilaku ini, nama login dan alamat surveyor dapat diubah SETELAH komite
// menyetujuinya, dan komite tidak pernah melihat perubahannya. Uji ini yang menjaga agar
// perilaku itu tidak hilang lagi — ia sempat tidak dibawa, dan dibetulkan 2026-09-20.
func TestUpdateMengembalikanSurveyorYangSudahDisetujuiKeAntreanKomite(t *testing.T) {
	service, _ := newService(t)

	saved, err := service.Submit(context.Background(), portalAlias, submission(), submitter())
	require.NoError(t, err)

	disetujui, err := service.Decide(context.Background(), portalAlias, saved.ID,
		usecase.Decision{Status: mastersurveyors.StatusApproved, Note: "Lengkap."},
		usecase.Committee{Identity: "KOMITEUJI"},
	)
	require.NoError(t, err)
	require.Equal(t, mastersurveyors.StatusApproved, disetujui.Status)
	require.NotNil(t, disetujui.DecidedAt)

	changed := submission()
	changed.AppLogin = "LOGINBARU"

	updated, err := service.Update(context.Background(), portalAlias, saved.ID, changed, submitter())
	require.NoError(t, err)

	require.Equal(t, mastersurveyors.StatusPending, updated.Status,
		"menyunting surveyor yang sudah disetujui wajib mengembalikannya ke antrean komite")
	require.False(t, updated.Assignable(), "ia tidak boleh lagi dapat ditugaskan")

	// Jejak keputusan sebelumnya ikut dibuang: baris berstatus menunggu yang memuat
	// tanggal dan catatan keputusan adalah keadaan yang tidak dapat dibaca siapa pun.
	require.Nil(t, updated.DecidedAt)
	require.Empty(t, updated.Note)

	// Komitenya TETAP — ia ditetapkan sekali saat pengajuan pertama.
	require.Equal(t, saved.Committee, updated.Committee)
}

// TestUpdateMengembalikanSurveyorYangDitolakKeAntreanKomite adalah pasangan uji di atas.
//
// Tanpa ini, surveyor yang ditolak tidak akan pernah dapat diajukan ulang: memperbaiki
// datanya tidak akan mengembalikannya ke antrean, dan ia tertinggal di tab Reject
// selamanya.
func TestUpdateMengembalikanSurveyorYangDitolakKeAntreanKomite(t *testing.T) {
	service, _ := newService(t)

	saved, err := service.Submit(context.Background(), portalAlias, submission(), submitter())
	require.NoError(t, err)

	_, err = service.Decide(context.Background(), portalAlias, saved.ID,
		usecase.Decision{Status: mastersurveyors.StatusRejected, Note: "Data kurang."},
		usecase.Committee{Identity: "KOMITEUJI"},
	)
	require.NoError(t, err)

	changed := submission()
	changed.Address = "Jalan Uji Diperbaiki"

	updated, err := service.Update(context.Background(), portalAlias, saved.ID, changed, submitter())
	require.NoError(t, err)

	require.Equal(t, mastersurveyors.StatusPending, updated.Status)
	require.Empty(t, updated.Note)
}

// TestDecideMencatatKeputusanDanWaktunya menguji jalur berhasil keputusan komite.
func TestDecideMencatatKeputusanDanWaktunya(t *testing.T) {
	service, _ := newService(t)

	saved, err := service.Submit(context.Background(), portalAlias, submission(), submitter())
	require.NoError(t, err)

	decided, err := service.Decide(context.Background(), portalAlias, saved.ID,
		usecase.Decision{Status: mastersurveyors.StatusApproved, Note: "Lengkap."},
		usecase.Committee{Identity: "KOMITEUJI", Name: "Komite Uji"},
	)
	require.NoError(t, err)

	require.Equal(t, mastersurveyors.StatusApproved, decided.Status)
	require.Equal(t, "Lengkap.", decided.Note)
	require.NotNil(t, decided.DecidedAt, "waktu keputusan wajib tercatat — ia jejak audit")
	require.Equal(t, "1", decided.CommitteeTransferred)
	require.True(t, decided.Assignable())
}

// TestDecideMenolakKomiteYangBukanDitunjuk menguji satu-satunya kontrol kewenangan yang
// benar-benar ada di modul ini.
func TestDecideMenolakKomiteYangBukanDitunjuk(t *testing.T) {
	service, _ := newService(t)

	saved, err := service.Submit(context.Background(), portalAlias, submission(), submitter())
	require.NoError(t, err)

	_, err = service.Decide(context.Background(), portalAlias, saved.ID,
		usecase.Decision{Status: mastersurveyors.StatusApproved},
		usecase.Committee{Identity: "ORANGLAIN"},
	)
	require.ErrorIs(t, err, mastersurveyors.ErrNotAssignedCommittee)
}

// TestDecideMenerimaHurufBesarKecilYangBerbeda menguji kelonggaran yang disengaja.
//
// `docs/Steering/11-SECURITY.md` §3.1 mencatat identitas di sistem lama muncul dalam dua
// kapitalisasi berbeda karena perbandingan rule lama tidak konsisten soal itu.
func TestDecideMenerimaHurufBesarKecilYangBerbeda(t *testing.T) {
	service, _ := newService(t)

	saved, err := service.Submit(context.Background(), portalAlias, submission(), submitter())
	require.NoError(t, err)

	_, err = service.Decide(context.Background(), portalAlias, saved.ID,
		usecase.Decision{Status: mastersurveyors.StatusApproved},
		usecase.Committee{Identity: "komiteuji"},
	)
	require.NoError(t, err)
}

// TestDecideMenolakKeputusanKedua menguji bahwa keputusan komite tidak dianulir lewat
// layar ini.
func TestDecideMenolakKeputusanKedua(t *testing.T) {
	service, _ := newService(t)

	saved, err := service.Submit(context.Background(), portalAlias, submission(), submitter())
	require.NoError(t, err)

	by := usecase.Committee{Identity: "KOMITEUJI"}
	_, err = service.Decide(context.Background(), portalAlias, saved.ID,
		usecase.Decision{Status: mastersurveyors.StatusApproved}, by)
	require.NoError(t, err)

	_, err = service.Decide(context.Background(), portalAlias, saved.ID,
		usecase.Decision{Status: mastersurveyors.StatusRejected}, by)
	require.ErrorIs(t, err, mastersurveyors.ErrAlreadyDecided)
}

// TestDecideMenolakStatusYangBukanKeputusan menguji bahwa "belum memutuskan" tidak dapat
// dikirim sebagai keputusan.
func TestDecideMenolakStatusYangBukanKeputusan(t *testing.T) {
	service, _ := newService(t)

	saved, err := service.Submit(context.Background(), portalAlias, submission(), submitter())
	require.NoError(t, err)

	for _, status := range []mastersurveyors.ApprovalStatus{mastersurveyors.StatusPending, "9", ""} {
		_, err = service.Decide(context.Background(), portalAlias, saved.ID,
			usecase.Decision{Status: status},
			usecase.Committee{Identity: "KOMITEUJI"},
		)
		require.ErrorIs(t, err, mastersurveyors.ErrUnknownStatus, "status %q", status)
	}
}

// TestPortalTidakDikenalDitolak menguji bahwa portal yang tidak dikenal menghasilkan
// galat — TIDAK PERNAH dialihkan ke portal utama sebagai cadangan (`R-20`).
func TestPortalTidakDikenalDitolak(t *testing.T) {
	recorder := account.NewRecorder(nil)
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (mastersurveyors.Repo, error) {
			if alias != portalAlias {
				return nil, context.Canceled // galat apa pun; yang diuji adalah ia TIDAK nil
			}
			return memory.NewRepo(), nil
		},
		Committee: committee.Fixed{},
		Accounts:  recorder,
	})
	require.NoError(t, err)

	_, _, err = service.List(context.Background(), "ENTITASLAIN", mastersurveyors.Filter{})
	require.Error(t, err)
}

// TestNewServiceMenolakBahanTidakLengkap menguji bahwa rakitan setengah jadi gagal saat
// start, bukan saat pengguna sedang bekerja.
func TestNewServiceMenolakBahanTidakLengkap(t *testing.T) {
	selector := func(string) (mastersurveyors.Repo, error) { return memory.NewRepo(), nil }

	_, err := usecase.NewService(usecase.Options{Committee: committee.Fixed{}, Accounts: account.NewRecorder(nil)})
	require.Error(t, err)

	_, err = usecase.NewService(usecase.Options{RepoSelector: selector, Accounts: account.NewRecorder(nil)})
	require.Error(t, err)

	_, err = usecase.NewService(usecase.Options{RepoSelector: selector, Committee: committee.Fixed{}})
	require.Error(t, err)
}

package usecase_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxrclpucl"
	"claim-pnc/internal/inboxrclpucl/usecase"
)

// reminderRepo adalah Repo tiruan yang HANYA melayani ReminderRecipients.
//
// Ia menyematkan inboxrclpucl.Repo supaya metode lain tetap memenuhi antarmuka tanpa
// ditulis satu per satu. Memanggil yang tidak dipasang akan panik — dan itu disengaja:
// uji yang diam-diam memakai jalur lain lebih buruk daripada uji yang berhenti.
type reminderRepo struct {
	inboxrclpucl.Repo

	recipients []inboxrclpucl.ReminderRecipient
	err        error

	// asked merekam nomor yang BENAR-BENAR diminta ke repo.
	asked []string
}

func (r *reminderRepo) ReminderRecipients(
	_ context.Context,
	claimNumbers []string,
) ([]inboxrclpucl.ReminderRecipient, error) {
	r.asked = append(r.asked, claimNumbers...)
	return r.recipients, r.err
}

// reminderNotifier merekam pengiriman, dan boleh gagal pada nomor tertentu.
type reminderNotifier struct {
	sent   []string
	failOn map[string]error
}

func (n *reminderNotifier) SendReminder(
	_ context.Context,
	recipient inboxrclpucl.ReminderRecipient,
) error {
	if err, bad := n.failOn[recipient.ClaimNumber]; bad {
		return err
	}
	n.sent = append(n.sent, recipient.ClaimNumber)
	return nil
}

func reminderService(
	t *testing.T,
	repo inboxrclpucl.Repo,
	notifier inboxrclpucl.ReminderNotifier,
) *usecase.Service {
	t.Helper()

	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (inboxrclpucl.Repo, error) { return repo, nil },
		Reminders:    notifier,
	})
	require.NoError(t, err)
	return service
}

func TestSendReminderMengirimKeKlaimYangPunyaAlamat(t *testing.T) {
	repo := &reminderRepo{recipients: []inboxrclpucl.ReminderRecipient{
		{ClaimNumber: "PNC-1", InsuredName: "Tertanggung Satu", Email: "satu@contoh.invalid"},
	}}
	notifier := &reminderNotifier{}

	hasil, err := reminderService(t, repo, notifier).
		SendReminder(context.Background(), "ASM", inboxrclpucl.Caller{Login: "PETUGAS"},
			[]string{"PNC-1"})

	require.NoError(t, err)
	require.Equal(t, []string{"PNC-1"}, notifier.sent)
	require.Equal(t, 1, hasil.Count(inboxrclpucl.ReminderSent))
}

func TestSendReminderMelaporkanKlaimTanpaAlamatAlihAlihMengirim(t *testing.T) {
	// Inilah yang menggantikan kalimat yang dulu ditulis tetap di layar — "kolom itu kosong
	// pada seluruh baris". Keadaan basis data dilaporkan PER KLAIM saat pengiriman dicoba,
	// bukan ditebak di muka lalu menjadi salah begitu kolomnya diisi.
	repo := &reminderRepo{recipients: []inboxrclpucl.ReminderRecipient{
		{ClaimNumber: "PNC-1", InsuredName: "Tertanggung Satu", Email: ""},
	}}
	notifier := &reminderNotifier{}

	hasil, err := reminderService(t, repo, notifier).
		SendReminder(context.Background(), "ASM", inboxrclpucl.Caller{}, []string{"PNC-1"})

	require.NoError(t, err, "satu klaim tanpa alamat BUKAN kegagalan seluruh permintaan")
	require.Empty(t, notifier.sent, "tidak boleh ada surel ke alamat kosong")
	require.Equal(t, 1, hasil.Count(inboxrclpucl.ReminderNoEmail))
}

func TestSendReminderMelaporkanNomorYangTidakMemenuhiSyarat(t *testing.T) {
	// Repo hanya mengembalikan klaim yang memenuhi penyaring antrean Kelengkapan Dokumen.
	// Nomor yang tidak kembali WAJIB tetap punya barisnya sendiri di hasil — tanpa itu ia
	// hilang diam-diam, dan petugas mengira pengingatnya terkirim.
	repo := &reminderRepo{recipients: nil}
	notifier := &reminderNotifier{}

	hasil, err := reminderService(t, repo, notifier).
		SendReminder(context.Background(), "ASM", inboxrclpucl.Caller{}, []string{"PNC-9"})

	require.NoError(t, err)
	require.Empty(t, notifier.sent)
	require.Len(t, hasil.Outcomes, 1)
	require.Equal(t, inboxrclpucl.ReminderNotEligible, hasil.Outcomes[0].Status)
}

func TestSendReminderTidakMembatalkanSisanyaKetikaSatuGagal(t *testing.T) {
	repo := &reminderRepo{recipients: []inboxrclpucl.ReminderRecipient{
		{ClaimNumber: "PNC-1", Email: "satu@contoh.invalid"},
		{ClaimNumber: "PNC-2", Email: "dua@contoh.invalid"},
	}}
	notifier := &reminderNotifier{
		failOn: map[string]error{"PNC-1": errors.New("relay menolak")},
	}

	hasil, err := reminderService(t, repo, notifier).
		SendReminder(context.Background(), "ASM", inboxrclpucl.Caller{},
			[]string{"PNC-1", "PNC-2"})

	require.NoError(t, err)
	require.Equal(t, []string{"PNC-2"}, notifier.sent, "yang kedua tetap dikirim")
	require.Equal(t, 1, hasil.Count(inboxrclpucl.ReminderFailed))
	require.Equal(t, 1, hasil.Count(inboxrclpucl.ReminderSent))
}

func TestSendReminderTidakMembocorkanGalatMentahKeLayar(t *testing.T) {
	// Galat SMTP dapat memuat nama host internal. Yang sampai ke layar adalah kalimat yang
	// layak dibaca petugas; yang lengkap masuk log.
	repo := &reminderRepo{recipients: []inboxrclpucl.ReminderRecipient{
		{ClaimNumber: "PNC-1", Email: "satu@contoh.invalid"},
	}}
	notifier := &reminderNotifier{
		failOn: map[string]error{"PNC-1": errors.New("dial tcp relay-internal.contoh:587")},
	}

	hasil, err := reminderService(t, repo, notifier).
		SendReminder(context.Background(), "ASM", inboxrclpucl.Caller{}, []string{"PNC-1"})

	require.NoError(t, err)
	require.Len(t, hasil.Outcomes, 1)
	require.NotContains(t, hasil.Outcomes[0].Reason, "relay-internal")
	require.NotEmpty(t, hasil.Outcomes[0].Reason)
}

func TestSendReminderMenolakPermintaanKosongDanBerlebihan(t *testing.T) {
	service := reminderService(t, &reminderRepo{}, &reminderNotifier{})

	_, err := service.SendReminder(context.Background(), "ASM", inboxrclpucl.Caller{}, nil)
	require.ErrorIs(t, err, inboxrclpucl.ErrNoReminderRecipients)

	// Spasi dan senarai berisi teks kosong dihitung sebagai TIDAK ADA, bukan satu klaim.
	_, err = service.SendReminder(context.Background(), "ASM", inboxrclpucl.Caller{},
		[]string{"  ", ""})
	require.ErrorIs(t, err, inboxrclpucl.ErrNoReminderRecipients)

	terlalu := make([]string, inboxrclpucl.MaxReminderRecipients+1)
	for i := range terlalu {
		terlalu[i] = "PNC-" + string(rune('A'+i%26)) + string(rune('A'+i/26))
	}
	_, err = service.SendReminder(context.Background(), "ASM", inboxrclpucl.Caller{}, terlalu)
	require.ErrorIs(t, err, inboxrclpucl.ErrTooManyReminderRecipients)
}

func TestSendReminderMembuangNomorKembar(t *testing.T) {
	// Dua baris terpilih yang bernomor sama akan menghasilkan DUA surel ke orang yang sama.
	repo := &reminderRepo{recipients: []inboxrclpucl.ReminderRecipient{
		{ClaimNumber: "PNC-1", Email: "satu@contoh.invalid"},
	}}
	notifier := &reminderNotifier{}

	hasil, err := reminderService(t, repo, notifier).
		SendReminder(context.Background(), "ASM", inboxrclpucl.Caller{},
			[]string{"PNC-1", "PNC-1", " PNC-1 "})

	require.NoError(t, err)
	require.Equal(t, []string{"PNC-1"}, repo.asked, "repo hanya ditanyai sekali")
	require.Len(t, hasil.Outcomes, 1)
	require.Equal(t, 1, hasil.Count(inboxrclpucl.ReminderSent))
}

func TestSendReminderMenolakKetikaPengirimBelumTerpasang(t *testing.T) {
	// Nil BUKAN kekeliruan tatanan: SMTP belum tentu terkonfigurasi di setiap lingkungan.
	// Yang tidak boleh terjadi adalah tombol yang tampak bekerja lalu diam.
	service := reminderService(t, &reminderRepo{}, nil)

	_, err := service.SendReminder(context.Background(), "ASM", inboxrclpucl.Caller{},
		[]string{"PNC-1"})
	require.ErrorIs(t, err, usecase.ErrReminderUnavailable)
}

func TestReminderSubjectMenirukanPegaApaAdanya(t *testing.T) {
	// Termasuk salah ketik "AACCIDENT" dan pemenggalan pada huruf `Q` PERTAMA — bukan pada
	// `" QQ "`. Keduanya perilaku Pega, dan `P-5` menetapkan perilaku dipertahankan lebih
	// dulu; memperbaiki teks yang dibaca pihak LUAR adalah keputusan Work Owner.
	require.Equal(t,
		"REMINDER PROSES ULANG KLAIM PERSONAL AACCIDENT - A/N PT.SUMMIT OTO FINANCE",
		inboxrclpucl.ReminderSubject("PT.SUMMIT OTO FINANCE QQ YAZIER ARAFAH"))

	// Nama tanpa Q dipakai utuh.
	require.Equal(t,
		"REMINDER PROSES ULANG KLAIM PERSONAL AACCIDENT - A/N ETY ROHETI",
		inboxrclpucl.ReminderSubject("ETY ROHETI"))
}

func TestSendReminderMembedakanTidakTerjangkauDariDitolak(t *testing.T) {
	// Keduanya menuntut tindakan yang BERBEDA, dan menyebutnya dengan kalimat yang sama
	// menyuruh orang mencari di tempat yang keliru.
	//
	// Itu benar-benar terjadi 2026-10-12: layar berbunyi "ditolak peladen surel" padahal
	// `SMTP_HOST` tidak teresolusi — peladennya tidak pernah dihubungi, apalagi menolak.
	for _, c := range []struct {
		nama     string
		galat    error
		memuat   string
		dilarang string
	}{
		{
			nama:     "tidak terjangkau",
			galat:    fmt.Errorf("dial: %w: no such host", inboxrclpucl.ErrReminderUnreachable),
			memuat:   "tidak dapat dihubungi",
			dilarang: "menolak",
		},
		{
			nama:     "ditolak",
			galat:    fmt.Errorf("rcpt: %w: 550", inboxrclpucl.ErrReminderRejected),
			memuat:   "menolak",
			dilarang: "tidak dapat dihubungi",
		},
	} {
		t.Run(c.nama, func(t *testing.T) {
			repo := &reminderRepo{recipients: []inboxrclpucl.ReminderRecipient{
				{ClaimNumber: "PNC-1", Email: "satu@contoh.invalid"},
			}}
			notifier := &reminderNotifier{failOn: map[string]error{"PNC-1": c.galat}}

			hasil, err := reminderService(t, repo, notifier).
				SendReminder(context.Background(), "ASM", inboxrclpucl.Caller{},
					[]string{"PNC-1"})

			require.NoError(t, err)
			require.Len(t, hasil.Outcomes, 1)

			alasan := hasil.Outcomes[0].Reason
			require.Contains(t, alasan, c.memuat)
			require.NotContains(t, alasan, c.dilarang)

			// Galat mentah tetap tidak bocor ke layar.
			require.NotContains(t, alasan, "no such host")
			require.NotContains(t, alasan, "550")
		})
	}
}

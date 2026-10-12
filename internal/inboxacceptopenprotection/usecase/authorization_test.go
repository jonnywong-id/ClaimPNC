package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxacceptopenprotection"
	"claim-pnc/internal/inboxacceptopenprotection/usecase"
)

// Uji di berkas ini menjaga penegakan `When/IsOpenProtectionPNC-When.xml` beserta pemisahan
// antrean pada `Section/InputProtection_Section-Section.xml:1592` dan `:6036`.
//
// Sumber kewenangannya `POOLDATA.M_LOGIN_GROUP_PNC.GROUP_ID`, yang berisi nama access group
// Pega tanpa awalan `GCNMFW:` (Work Owner, 2026-09-25).

func TestLoginDiLuarKelimaAccessGroupDitolak(t *testing.T) {
	// `PncAdmin` TIDAK termasuk kelima access group `IsOpenProtectionPNC`. Ia peran yang
	// paling banyak dipakai, sehingga justru ia yang paling mungkin salah terbuka.
	at := time.Date(2026, time.September, 25, 10, 0, 0, 0, wib)
	service, repo := bangunDenganGroup(t, at, map[string][]string{
		"ADMINCONTOH": {"PncAdmin"},
	})
	repo.Add(lengkap("OPCN.26.0001", "1", at))

	_, err := service.List(context.Background(), usecase.ListQuery{
		PortalAlias: portal,
		Login:       "ADMINCONTOH",
		Queue:       inboxacceptopenprotection.QueueNonPremium,
	})
	require.ErrorIs(t, err, inboxacceptopenprotection.ErrForbidden)
}

func TestLoginTanpaSatuPunGroupDitolak(t *testing.T) {
	// Tabel yang BELUM diisi harus menutup layar, bukan membukanya. Arah sebaliknya akan
	// membuat setiap orang yang berhasil masuk dapat menyetujui pembukaan proteksi.
	at := time.Date(2026, time.September, 25, 10, 0, 0, 0, wib)
	service, _ := bangunDenganGroup(t, at, map[string][]string{})

	_, err := service.List(context.Background(), usecase.ListQuery{
		PortalAlias: portal,
		Login:       "SIAPAPUN",
		Queue:       inboxacceptopenprotection.QueueNonPremium,
	})
	require.ErrorIs(t, err, inboxacceptopenprotection.ErrForbidden)
}

func TestHanyaPncCollectionYangMembukaAntreanPremi(t *testing.T) {
	// `Section/InputProtection_Section-Section.xml:6036` menampilkan grid PREMI hanya bagi
	// `AccessGroup == 'GCNMFW:PncCollection'`.
	at := time.Date(2026, time.September, 25, 10, 0, 0, 0, wib)
	service, repo := bangunDenganGroup(t, at, map[string][]string{
		"TEKNIKCONTOH":  {"CaseManager"},
		"KOLEKSICONTOH": {"PncCollection"},
	})
	repo.Add(lengkap("OPCN.26.0002", inboxacceptopenprotection.TypePremium, at))

	premi := usecase.ListQuery{
		PortalAlias: portal,
		Queue:       inboxacceptopenprotection.QueuePremium,
	}

	premi.Login = "TEKNIKCONTOH"
	_, err := service.List(context.Background(), premi)
	require.ErrorIs(t, err, inboxacceptopenprotection.ErrForbidden,
		"CaseManager tidak boleh membuka antrean PREMI")

	premi.Login = "KOLEKSICONTOH"
	page, err := service.List(context.Background(), premi)
	require.NoError(t, err)
	require.Equal(t, 1, page.Total)
}

func TestPncCollectionTidakMembukaAntreanNonPremi(t *testing.T) {
	// Kebalikannya juga berlaku: `:1592` menampilkan grid NON PREMI bagi yang BUKAN
	// PncCollection. Melewatkan arah ini akan membuat pemisahannya berlaku sebelah.
	at := time.Date(2026, time.September, 25, 10, 0, 0, 0, wib)
	service, _ := bangunDenganGroup(t, at, map[string][]string{
		"KOLEKSICONTOH": {"PncCollection"},
	})

	_, err := service.List(context.Background(), usecase.ListQuery{
		PortalAlias: portal,
		Login:       "KOLEKSICONTOH",
		Queue:       inboxacceptopenprotection.QueueNonPremium,
	})
	require.ErrorIs(t, err, inboxacceptopenprotection.ErrForbidden)
}

func TestKeduaGroupMembukaKeduaAntrean(t *testing.T) {
	// Keadaan yang TIDAK punya padanan di Pega: di sana satu operator berada di tepat satu
	// access group. `M_LOGIN_GROUP_PNC` berkunci (LOGIN_ID, GROUP_ID), sehingga banyak
	// group mungkin — dan yang dipilih adalah GABUNGAN, bukan salah satu.
	at := time.Date(2026, time.September, 25, 10, 0, 0, 0, wib)
	service, _ := bangunDenganGroup(t, at, map[string][]string{
		"KEDUACONTOH": {"CaseManager", "PncCollection"},
	})

	queues, err := service.Queues(context.Background(), "KEDUACONTOH")
	require.NoError(t, err)
	require.Equal(t, []inboxacceptopenprotection.Queue{
		inboxacceptopenprotection.QueueNonPremium,
		inboxacceptopenprotection.QueuePremium,
	}, queues, "urutannya mengikuti urutan grid layar lama: NON PREMI lebih dulu")
}

func TestKapitalisasiGroupTidakMenentukanKewenangan(t *testing.T) {
	// `D-58` mencatat Pega tidak konsisten kapitalisasinya — ViewClaimPNC/VIEWCLAIMPNC dan
	// PncReceive/PNCRECEIVE hidup berdampingan — dan menuntut sistem baru menormalkannya.
	// Penolakan karena kapitalisasi akan tampak seperti gangguan, bukan seperti keputusan.
	//
	// Bentuk berawalan ikut diuji: barisnya diisi manusia, dan satu baris yang telanjur
	// memuat `GCNMFW:` tidak boleh menolak pengguna yang sah.
	at := time.Date(2026, time.September, 25, 10, 0, 0, 0, wib)
	service, _ := bangunDenganGroup(t, at, map[string][]string{
		"HURUFBESAR": {"PNCCOLLECTION"},
		"HURUFKECIL": {"pnccollection"},
		"BERAWALAN":  {"GCNMFW:PncCollection"},
	})

	for _, login := range []string{"HURUFBESAR", "HURUFKECIL", "BERAWALAN"} {
		queues, err := service.Queues(context.Background(), login)
		require.NoError(t, err, "login %s", login)
		require.Equal(t, []inboxacceptopenprotection.Queue{
			inboxacceptopenprotection.QueuePremium,
		}, queues, "login %s seharusnya dikenali sebagai PncCollection", login)
	}
}

func TestKeputusanAtasAntreanYangBukanHaknyaDitolak(t *testing.T) {
	// Pembacaan cukup menuntut kewenangan atas LAYAR; KEPUTUSAN menuntut kewenangan atas
	// antrean baris yang bersangkutan.
	//
	// Tipe proteksinya diambil dari yang TERSIMPAN, bukan dari apa pun yang dikirim klien.
	at := time.Date(2026, time.September, 25, 10, 0, 0, 0, wib)
	service, repo := bangunDenganGroup(t, at, map[string][]string{
		"TEKNIKCONTOH": {"CaseManager"},
	})
	repo.Add(lengkap("OPCN.26.0002", inboxacceptopenprotection.TypePremium, at))

	_, err := service.Decide(context.Background(), usecase.DecideCommand{
		PortalAlias: portal,
		Number:      "OPCN.26.0002",
		Decision:    inboxacceptopenprotection.DecisionApprove,
		By:          "TEKNIKCONTOH",
	})
	require.ErrorIs(t, err, inboxacceptopenprotection.ErrForbidden)

	// Penolakan kewenangan tidak boleh menyisakan perubahan apa pun: barisnya tetap
	// MENUNGGU keputusan.
	tetap, err := service.Get(context.Background(), portal, "OPCN.26.0002", "TEKNIKCONTOH")
	require.NoError(t, err)
	require.True(t, tetap.Pending(), "baris tidak boleh berubah setelah keputusan ditolak")
}

func TestPembacaanRincianTidakTundukPadaAntrean(t *testing.T) {
	// Sebuah proteksi dapat berpindah antrean setelah tautannya disimpan — tipenya disunting
	// pemohon. Menolak PEMBACAAN karenanya akan menampilkan 403 pada baris yang tadinya sah
	// dibuka. Yang dijaga ketat adalah keputusannya.
	at := time.Date(2026, time.September, 25, 10, 0, 0, 0, wib)
	service, repo := bangunDenganGroup(t, at, map[string][]string{
		"TEKNIKCONTOH": {"CaseManager"},
	})
	repo.Add(lengkap("OPCN.26.0002", inboxacceptopenprotection.TypePremium, at))

	p, err := service.Get(context.Background(), portal, "OPCN.26.0002", "TEKNIKCONTOH")
	require.NoError(t, err, "pembacaan cukup menuntut kewenangan atas layar")
	require.Equal(t, inboxacceptopenprotection.TypePremium, p.Type)
}

func TestServiceTanpaPembacaGroupDitolakSaatDirakit(t *testing.T) {
	// Perakitan yang lupa memasangnya tidak boleh berjalan tanpa pemeriksaan kewenangan
	// sama sekali — dan lupanya tidak menghasilkan galat apa pun saat dijalankan.
	_, err := usecase.NewService(usecase.Options{
		Protections: func(string) (inboxacceptopenprotection.Repo, error) { return nil, nil },
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "access group")
}

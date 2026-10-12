package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxmanager"
	"claim-pnc/internal/inboxmanager/repo/memory"
	"claim-pnc/internal/inboxmanager/usecase"
	"claim-pnc/internal/platform/clock"
)

const portalUtama = "ASM"

// newService merakit layanan di atas penyimpanan memori.
func newService(t *testing.T, store *memory.Store, at time.Time) *usecase.Service {
	t.Helper()

	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (inboxmanager.Repo, error) {
			if alias != portalUtama {
				return nil, errors.New("portal tidak dikenal: " + alias)
			}
			return store, nil
		},
		LineBusinessSelector: func(alias string) (inboxmanager.LineBusinessRepo, error) {
			if alias != portalUtama {
				return nil, errors.New("portal tidak dikenal: " + alias)
			}
			return store, nil
		},
		Clock: clock.FixedAt(at),
	})
	require.NoError(t, err)

	return service
}

// TestLiniBisnisDibacaDariPenyimpananBukanDariPemanggil mengunci koreksi 2026-09-27.
//
// `Caller` sengaja membawa nilai yang SALAH dan penyimpanan membawa yang BENAR. Bila nilai
// pemanggil dipakai lagi — yakni bila lini bisnis kembali diambil dari jabatan kepegawaian HCQ
// — uji ini gagal.
func TestLiniBisnisDibacaDariPenyimpananBukanDariPemanggil(t *testing.T) {
	store := memory.NewStrictStore()
	store.SetLineBusiness("JONNY", inboxmanager.LineNonMBU)

	service := newService(t, store, time.Now())

	meta, err := service.Metadata(context.Background(), portalUtama, inboxmanager.Caller{
		Login: "JONNY",

		// Nilai yang SALAH, seperti yang dulu datang dari sesi.
		LineBusiness: "IT SPECIALIST",
	})
	require.NoError(t, err)

	require.Equal(t, inboxmanager.LineNonMBU, meta.LineBusiness)
	require.Len(t, meta.Tabs, 13, "petugas NONMBU melihat seluruh tab")
}

// TestPetugasTanpaLiniBisnisTetapMelihatDuaBelasTab mengunci keadaan paling umum di produksi.
//
// Berbeda dari modul Inbox Manager Admin — yang seluruh tabnya dibatasi lini bisnis — layar ini
// hanya membatasi SATU tab. Petugas yang kolomnya belum diisi karena itu tetap dapat bekerja,
// dan layarnya tidak kosong.
func TestPetugasTanpaLiniBisnisTetapMelihatDuaBelasTab(t *testing.T) {
	store := memory.NewStrictStore()
	service := newService(t, store, time.Now())

	meta, err := service.Metadata(context.Background(), portalUtama,
		inboxmanager.Caller{Login: "TANPA_LINI"})
	require.NoError(t, err)

	require.Empty(t, meta.LineBusiness)
	require.Len(t, meta.Tabs, 12)
	require.Equal(t, inboxmanager.TabOutstanding, meta.DefaultTab)
}

// TestPortalLainDitolak menahan `R-20` — kebocoran yang tidak menghasilkan galat.
//
// Di modul ini taruhannya lebih besar daripada modul baca-saja: rute keputusan MENULIS, dan
// menulis ke basis data badan hukum yang salah tidak dapat dibatalkan dengan memuat ulang
// halaman.
func TestPortalLainDitolak(t *testing.T) {
	store := memory.NewSampleStore()
	service := newService(t, store, time.Now())

	_, err := service.Metadata(context.Background(), "PORTAL_LAIN",
		inboxmanager.Caller{Login: "JONNY"})
	require.Error(t, err)

	_, err = service.Decide(context.Background(), "PORTAL_LAIN", usecase.DecideInput{
		Tab:     inboxmanager.TabKategoriSparepart,
		Verdict: inboxmanager.VerdictApprove,
		Keys:    []string{"KAT-01"},
	}, inboxmanager.Caller{Login: "JONNY"})
	require.Error(t, err)
}

// TestPeriodeBawaanTabProduktivitasAdalahBulanBerjalan mengunci nilai bawaan yang WAJIB ada.
//
// Tab itu tidak dapat berjalan tanpa periode — kedelapan pencacahnya dibangun dari
// perbandingan dua periode — sehingga periodenya harus punya nilai bawaan.
func TestPeriodeBawaanTabProduktivitasAdalahBulanBerjalan(t *testing.T) {
	store := memory.NewSampleStore()
	service := newService(t, store, time.Date(2026, 9, 28, 10, 0, 0, 0, time.UTC))

	view, err := service.List(context.Background(), portalUtama, inboxmanager.QueryInput{
		Tab: inboxmanager.TabProduktivitas,
	}, inboxmanager.Caller{Login: "JONNY"})
	require.NoError(t, err)

	require.Equal(t, time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), view.Period.From)
	require.Equal(t, time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), view.Period.Until)
	require.Equal(t, time.Date(2025, 9, 1, 0, 0, 0, 0, time.UTC), view.Period.PriorFrom)
}

// TestTabKlaimTIDAKDiberiPeriodeBawaan mengunci perbedaan yang disengaja.
//
// Kueri Dashboard Klaim di Pega menyisipkan penyaring periodenya lewat pola `{ASIS}`, dan pola
// itu memang dapat berisi teks kosong — artinya seluruh periode. Memaksakan bulan berjalan di
// sana akan memperkecil angkanya tanpa satu pun tanda.
func TestTabKlaimTIDAKDiberiPeriodeBawaan(t *testing.T) {
	store := memory.NewSampleStore()
	service := newService(t, store, time.Date(2026, 9, 28, 10, 0, 0, 0, time.UTC))

	view, err := service.List(context.Background(), portalUtama, inboxmanager.QueryInput{
		Tab: inboxmanager.TabKlaim,
	}, inboxmanager.Caller{Login: "JONNY"})
	require.NoError(t, err)

	require.True(t, view.Period.Empty())
}

// TestPencacahApprovalMasterAdalahJumlahAnaknya mengunci selisih yang menguntungkan.
//
// `Activity/CountDashbroardManager` tidak punya kueri pencacah untuk tab itu; yang ditulisnya
// angka tetap `"1"` — sebuah penanda, bukan hitungan.
func TestPencacahApprovalMasterAdalahJumlahAnaknya(t *testing.T) {
	store := memory.NewSampleStore()
	service := newService(t, store, time.Now())

	counters, err := service.Counters(context.Background(), portalUtama,
		inboxmanager.Caller{Login: "JONNY"})
	require.NoError(t, err)

	total, overview := 0, inboxmanager.Counter{}
	for _, counter := range counters {
		switch {
		case counter.TabCode == inboxmanager.TabApprovalMaster:
			overview = counter
		case counter.Parent == inboxmanager.TabApprovalMaster && counter.Unavailable == "":
			total += counter.Count
		}
	}

	require.Equal(t, inboxmanager.TabApprovalMaster, overview.TabCode)
	require.Equal(t, total, overview.Count)
}

// TestPencacahMenyatakanSumberYangTidakTerbaca mengunci keadaan nyata per 2026-09-28.
//
// `POOLDATA.SPAREPART_HE` adalah view berstatus INVALID. Angka nol yang sesungguhnya berarti
// "tidak terbaca" akan membuat penyelia mengira antreannya kosong.
func TestPencacahMenyatakanSumberYangTidakTerbaca(t *testing.T) {
	store := memory.NewSampleStore()
	service := newService(t, store, time.Now())

	counters, err := service.Counters(context.Background(), portalUtama,
		inboxmanager.Caller{Login: "JONNY"})
	require.NoError(t, err)

	var sparepart, overview inboxmanager.Counter
	for _, counter := range counters {
		if counter.TabCode == inboxmanager.TabMasterSparepart {
			sparepart = counter
		}
		if counter.TabCode == inboxmanager.TabApprovalMaster {
			overview = counter
		}
	}

	require.NotEmpty(t, sparepart.Unavailable)
	require.NotEmpty(t, overview.Unavailable,
		"ringkasan wajib menyatakan angkanya belum lengkap saat ada sumber yang rusak")
}

// TestKeputusanMenyatakanBarisYangSudahDiputuskanOrangLain mengunci angka yang sampai ke layar.
//
// Setiap pernyataan keputusan ikut menyaring status menunggu, sehingga baris yang sudah
// diputuskan orang lain tidak berubah. Di Pega keadaan itu tidak dapat diketahui sama sekali:
// keputusan terakhir menimpa yang sebelumnya tanpa jejak.
func TestKeputusanMenyatakanBarisYangSudahDiputuskanOrangLain(t *testing.T) {
	store := memory.NewSampleStore()
	service := newService(t, store, time.Now())

	result, err := service.Decide(context.Background(), portalUtama, usecase.DecideInput{
		Tab:     inboxmanager.TabMasterBengkel,
		Verdict: inboxmanager.VerdictApprove,

		// BGK-002 ada; BGK-999 tidak — seolah sudah diputuskan orang lain sementara
		// daftar di layar masih yang lama.
		Keys: []string{"BGK-002", "BGK-999"},
	}, inboxmanager.Caller{Login: "JONNY"})
	require.NoError(t, err)

	require.Equal(t, 2, result.Requested)
	require.Equal(t, 1, result.Changed)
	require.Equal(t, 1, result.Stale())
}

// TestBarisYangDiputuskanHilangDariAntrean memastikan penyimpanan memori berperilaku sama
// dengan SQL pada hal yang paling sering diuji.
func TestBarisYangDiputuskanHilangDariAntrean(t *testing.T) {
	store := memory.NewSampleStore()
	service := newService(t, store, time.Now())

	caller := inboxmanager.Caller{Login: "JONNY"}

	before, err := service.List(context.Background(), portalUtama,
		inboxmanager.QueryInput{Tab: inboxmanager.TabMasterBengkel}, caller)
	require.NoError(t, err)
	require.Equal(t, 2, before.Queue.Total)

	_, err = service.Decide(context.Background(), portalUtama, usecase.DecideInput{
		Tab:     inboxmanager.TabMasterBengkel,
		Verdict: inboxmanager.VerdictReject,
		Keys:    []string{"BGK-001"},
		Reason:  "alamat tidak sesuai",
	}, caller)
	require.NoError(t, err)

	after, err := service.List(context.Background(), portalUtama,
		inboxmanager.QueryInput{Tab: inboxmanager.TabMasterBengkel}, caller)
	require.NoError(t, err)
	require.Equal(t, 1, after.Queue.Total)
}

// TestAntreanNomorRangkaButuhLiniBisnisNonMBU menegakkan batas tab di jalur TULIS pula.
//
// Memeriksanya hanya pada jalur baca akan menyisakan lubang: permintaan keputusan dapat
// disusun tangan tanpa pernah membuka tabnya.
func TestAntreanNomorRangkaButuhLiniBisnisNonMBU(t *testing.T) {
	store := memory.NewSampleStore()
	store.SetLineBusiness("PETUGAS_PA", inboxmanager.LinePA)

	service := newService(t, store, time.Now())

	_, err := service.Decide(context.Background(), portalUtama, usecase.DecideInput{
		Tab:     inboxmanager.TabNomorRangka,
		Verdict: inboxmanager.VerdictApprove,
		Keys:    []string{"PNC-1001|AVANZA|MHF001|MHF002"},
	}, inboxmanager.Caller{Login: "PETUGAS_PA"})

	require.ErrorIs(t, err, inboxmanager.ErrTabNotAllowed)
}

// TestDashboardDanAntreanTidakPernahTerisiBersamaan mengunci bentuk jawaban.
//
// Layar memilih cara menggambar dari jenis tabnya; jawaban yang mengisi keduanya akan membuat
// layar menggambar tabel keputusan di atas baris yang tidak punya kunci.
func TestDashboardDanAntreanTidakPernahTerisiBersamaan(t *testing.T) {
	store := memory.NewSampleStore()
	service := newService(t, store, time.Now())
	caller := inboxmanager.Caller{Login: "JONNY"}

	for _, tab := range inboxmanager.Tabs() {
		view, err := service.List(context.Background(), portalUtama,
			inboxmanager.QueryInput{Tab: tab.Code}, caller)

		// Master Sparepart sengaja dicontohkan sebagai antrean yang sumbernya TIDAK dapat
		// dibaca — keadaan nyata di basis data sejak 2026-09-28. Ia menjawab galat yang
		// DIKENALI, bukan galat umum, sehingga layar dapat menjelaskan sebabnya.
		if tab.Code == inboxmanager.TabMasterSparepart {
			require.ErrorIs(t, err, inboxmanager.ErrSourceUnavailable)
			continue
		}
		require.NoErrorf(t, err, "tab %s", tab.Code)

		switch tab.Kind {
		case inboxmanager.KindDashboard:
			require.NotEmptyf(t, view.Dashboard.Panels, "tab %s", tab.Code)
			require.Emptyf(t, view.Queue.Rows, "tab %s", tab.Code)
		case inboxmanager.KindQueue:
			require.Emptyf(t, view.Dashboard.Panels, "tab %s", tab.Code)
		case inboxmanager.KindOverview:
			require.Emptyf(t, view.Dashboard.Panels, "tab %s", tab.Code)
			require.Emptyf(t, view.Queue.Rows, "tab %s", tab.Code)
		}
	}
}

// TestIdentitasKosongDitolakSebelumMenyentuhPenyimpanan mengunci syarat yang paling mendasar.
//
// Modul ini MENULIS, dan setiap keputusan dicatat atas nama pemanggilnya.
func TestIdentitasKosongDitolakSebelumMenyentuhPenyimpanan(t *testing.T) {
	store := memory.NewSampleStore()
	service := newService(t, store, time.Now())

	_, err := service.Metadata(context.Background(), portalUtama, inboxmanager.Caller{})
	require.ErrorIs(t, err, inboxmanager.ErrCallerUnknown)

	_, err = service.Decide(context.Background(), portalUtama, usecase.DecideInput{
		Tab:     inboxmanager.TabKategoriSparepart,
		Verdict: inboxmanager.VerdictApprove,
		Keys:    []string{"KAT-01"},
	}, inboxmanager.Caller{})
	require.ErrorIs(t, err, inboxmanager.ErrCallerUnknown)
}

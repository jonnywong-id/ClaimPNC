package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxacceptopenprotection"
	"claim-pnc/internal/inboxacceptopenprotection/repo/memory"
	"claim-pnc/internal/inboxacceptopenprotection/usecase"
)

const portal = "asm"

var wib = time.FixedZone("WIB", 7*60*60)

func bangun(t *testing.T, at time.Time) (*usecase.Service, *memory.Repo) {
	t.Helper()

	repo := memory.NewRepo()
	service, err := usecase.NewService(usecase.Options{
		Protections: func(alias string) (inboxacceptopenprotection.Repo, error) {
			if alias != portal {
				return nil, errors.New("portal tidak dikenal")
			}
			return repo, nil
		},
		Now: func() time.Time { return at },
	})
	require.NoError(t, err)

	return service, repo
}

// lengkap membentuk proteksi yang memenuhi ketiga syarat antrean akseptasi.
func lengkap(number, protectionType string, at time.Time) inboxacceptopenprotection.Protection {
	return inboxacceptopenprotection.Protection{
		Number:       number,
		PolicyNumber: "99.001.2026.00000001",
		ClaimNumber:  "PNCN.26.0007",
		Type:         protectionType,
		InputDate:    at,
		CreatedBy:    "ADMINCONTOH",
		AcceptStatus: inboxacceptopenprotection.AcceptPending,
	}
}

func TestAntreanPremiHanyaMemuatTipeDua(t *testing.T) {
	// `InboxOpenProtection2_RD_collection` menyaring `.TypeProtection = "2"`.
	at := time.Date(2026, time.September, 23, 10, 0, 0, 0, wib)
	service, repo := bangun(t, at)

	repo.Add(
		lengkap("OPCN.26.0001", "1", at),
		lengkap("OPCN.26.0002", inboxacceptopenprotection.TypePremium, at),
		lengkap("OPCN.26.0003", "7", at),
	)

	page, err := service.List(context.Background(), usecase.ListQuery{
		PortalAlias: portal,
		Queue:       inboxacceptopenprotection.QueuePremium,
	})
	require.NoError(t, err)

	require.Equal(t, 1, page.Total)
	require.Equal(t, "OPCN.26.0002", page.Protections[0].Number)
}

func TestAntreanNonPremiMemuatSeluruhTipeSelainDua(t *testing.T) {
	// `InboxOpenProtection2_RD` menyaring `!= "2"`. Tipe yang belum dikenal karena itu
	// tetap muncul di sini, bukan lenyap dari kedua antrean.
	at := time.Date(2026, time.September, 23, 10, 0, 0, 0, wib)
	service, repo := bangun(t, at)

	repo.Add(
		lengkap("OPCN.26.0001", "1", at),
		lengkap("OPCN.26.0002", inboxacceptopenprotection.TypePremium, at),
		lengkap("OPCN.26.0003", "7", at),
		lengkap("OPCN.26.0004", "99", at), // tipe yang belum dikenal
	)

	page, err := service.List(context.Background(), usecase.ListQuery{
		PortalAlias: portal,
		Queue:       inboxacceptopenprotection.QueueNonPremium,
	})
	require.NoError(t, err)

	require.Equal(t, 3, page.Total)
	for _, p := range page.Protections {
		require.NotEqual(t, inboxacceptopenprotection.TypePremium, p.Type)
	}
}

func TestProteksiTanpaNomorKlaimTidakSampaiKeMejaAkseptasi(t *testing.T) {
	// Inilah yang membedakan layar ini dari layar pemohon: `CaseID IS NOT NULL`.
	at := time.Date(2026, time.September, 23, 10, 0, 0, 0, wib)
	service, repo := bangun(t, at)

	rancangan := lengkap("OPCN.26.0005", "1", at)
	rancangan.ClaimNumber = ""
	repo.Add(rancangan)

	page, err := service.List(context.Background(), usecase.ListQuery{
		PortalAlias: portal,
		Queue:       inboxacceptopenprotection.QueueNonPremium,
	})
	require.NoError(t, err)
	require.Zero(t, page.Total)
}

func TestProteksiYangSudahDiputuskanHilangDariAntrean(t *testing.T) {
	at := time.Date(2026, time.September, 23, 10, 0, 0, 0, wib)
	service, repo := bangun(t, at)

	disetujui := lengkap("OPCN.26.0006", "1", at)
	disetujui.AcceptStatus = inboxacceptopenprotection.AcceptApproved
	repo.Add(disetujui)

	page, err := service.List(context.Background(), usecase.ListQuery{
		PortalAlias: portal,
		Queue:       inboxacceptopenprotection.QueueNonPremium,
	})
	require.NoError(t, err)
	require.Zero(t, page.Total)
}

func TestProteksiYangSudahDiputuskanTetapDapatDibukaLewatNomor(t *testing.T) {
	// Form harus tetap terbuka supaya pesannya dapat menjelaskan apa yang terjadi, bukan
	// sekadar "tidak ditemukan".
	at := time.Date(2026, time.September, 23, 10, 0, 0, 0, wib)
	service, repo := bangun(t, at)

	disetujui := lengkap("OPCN.26.0006", "1", at)
	disetujui.AcceptStatus = inboxacceptopenprotection.AcceptApproved
	repo.Add(disetujui)

	p, err := service.Get(context.Background(), portal, "OPCN.26.0006")
	require.NoError(t, err)
	require.True(t, p.Approved())
}

func TestPersetujuanMencatatPelakuDanWaktunya(t *testing.T) {
	// `D-59` menetapkan tidak ada pemisahan tugas formal, sehingga jejak inilah
	// satu-satunya kontrol pengimbang yang tersisa.
	at := time.Date(2026, time.September, 23, 10, 0, 0, 0, wib)
	service, repo := bangun(t, at)
	repo.Add(lengkap("OPCN.26.0001", "1", at))

	saved, err := service.Decide(context.Background(), usecase.DecideCommand{
		PortalAlias: portal,
		Number:      "OPCN.26.0001",
		Decision:    inboxacceptopenprotection.DecisionApprove,
		By:          "KOLEKSICONTOH",
	})
	require.NoError(t, err)

	require.Equal(t, inboxacceptopenprotection.AcceptApproved, saved.AcceptStatus)
	require.Equal(t, "KOLEKSICONTOH", saved.AcceptedBy)
	require.NotNil(t, saved.AcceptedAt)
	require.True(t, saved.AcceptedAt.Equal(at))
}

func TestPenolakanMenyimpanKodeDuaBukanKodeSatu(t *testing.T) {
	at := time.Date(2026, time.September, 23, 10, 0, 0, 0, wib)
	service, repo := bangun(t, at)
	repo.Add(lengkap("OPCN.26.0001", "1", at))

	saved, err := service.Decide(context.Background(), usecase.DecideCommand{
		PortalAlias: portal,
		Number:      "OPCN.26.0001",
		Decision:    inboxacceptopenprotection.DecisionReject,
		By:          "KOLEKSICONTOH",
	})
	require.NoError(t, err)
	require.Equal(t, inboxacceptopenprotection.AcceptRejected, saved.AcceptStatus)
}

func TestKeputusanKeduaAtasBarisYangSamaDitolak(t *testing.T) {
	// Antrean BERSAMA: dua petugas melihat baris yang sama, dan menekan tombol bersamaan
	// bukan kejadian langka.
	at := time.Date(2026, time.September, 23, 10, 0, 0, 0, wib)
	service, repo := bangun(t, at)
	repo.Add(lengkap("OPCN.26.0001", "1", at))

	_, err := service.Decide(context.Background(), usecase.DecideCommand{
		PortalAlias: portal, Number: "OPCN.26.0001",
		Decision: inboxacceptopenprotection.DecisionApprove, By: "PETUGAS-A",
	})
	require.NoError(t, err)

	_, err = service.Decide(context.Background(), usecase.DecideCommand{
		PortalAlias: portal, Number: "OPCN.26.0001",
		Decision: inboxacceptopenprotection.DecisionReject, By: "PETUGAS-B",
	})
	require.ErrorIs(t, err, inboxacceptopenprotection.ErrAlreadyDecided)
}

func TestKeputusanAsingDitolakTanpaNilaiBawaan(t *testing.T) {
	// Memperlakukan nilai asing sebagai "tolak" akan menolak proteksi yang tidak seorang
	// pun bermaksud menolaknya; sebagai "setuju" jauh lebih buruk.
	at := time.Date(2026, time.September, 23, 10, 0, 0, 0, wib)
	service, repo := bangun(t, at)
	repo.Add(lengkap("OPCN.26.0001", "1", at))

	_, err := service.Decide(context.Background(), usecase.DecideCommand{
		PortalAlias: portal, Number: "OPCN.26.0001",
		Decision: inboxacceptopenprotection.Decision("mungkin"), By: "KOLEKSICONTOH",
	})
	require.ErrorIs(t, err, inboxacceptopenprotection.ErrUnknownDecision)
}

func TestProteksiBelumLengkapTidakDapatDiakseptasiMeskiNomornyaDiketahui(t *testing.T) {
	// Ia tidak muncul di antrean, tetapi tautan yang disimpan masih dapat membukanya.
	at := time.Date(2026, time.September, 23, 10, 0, 0, 0, wib)
	service, repo := bangun(t, at)

	rancangan := lengkap("OPCN.26.0005", "1", at)
	rancangan.ClaimNumber = ""
	repo.Add(rancangan)

	_, err := service.Decide(context.Background(), usecase.DecideCommand{
		PortalAlias: portal, Number: "OPCN.26.0005",
		Decision: inboxacceptopenprotection.DecisionApprove, By: "KOLEKSICONTOH",
	})
	require.ErrorIs(t, err, inboxacceptopenprotection.ErrIncomplete)
}

func TestPelakuAkseptasiDiambilDariSesi(t *testing.T) {
	at := time.Date(2026, time.September, 23, 10, 0, 0, 0, wib)
	service, repo := bangun(t, at)
	repo.Add(lengkap("OPCN.26.0001", "1", at))

	_, err := service.Decide(context.Background(), usecase.DecideCommand{
		PortalAlias: portal, Number: "OPCN.26.0001",
		Decision: inboxacceptopenprotection.DecisionApprove, By: "",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "identitas pemanggil kosong")
}

func TestAntreanTidakDikenaliJatuhKeNonPremiBukanKePremi(t *testing.T) {
	// Arah jatuhnya dipilih sengaja: antrean PREMI menyangkut penagihan premi dan
	// pemiliknya satu peran tertentu.
	f := inboxacceptopenprotection.Filter{Queue: inboxacceptopenprotection.Queue("entah")}
	require.Equal(t, inboxacceptopenprotection.QueueNonPremium, f.Normalize().Queue)
}

func TestPortalTidakDikenalDitolak(t *testing.T) {
	at := time.Date(2026, time.September, 23, 10, 0, 0, 0, wib)
	service, _ := bangun(t, at)

	_, err := service.List(context.Background(), usecase.ListQuery{
		PortalAlias: "entitas-lain",
		Queue:       inboxacceptopenprotection.QueueNonPremium,
	})
	require.Error(t, err)
}

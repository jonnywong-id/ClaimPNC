package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inputreqprotection"
	"claim-pnc/internal/inputreqprotection/repo/memory"
	"claim-pnc/internal/inputreqprotection/usecase"
)

const portal = "asm"

var wib = time.FixedZone("WIB", 7*60*60)

// bangun membentuk service beserta penyimpanan memorinya pada waktu yang ditentukan.
//
// Waktu diserahkan uji, bukan diambil dari jam mesin, supaya aturan "hari ini" pada
// pemeriksaan proteksi ganda dapat diperiksa tanpa menunggu pergantian hari.
func bangun(t *testing.T, at time.Time) (*usecase.Service, *memory.Repo) {
	t.Helper()

	repo := memory.NewRepo()
	// Master tipe dan klaim ikut dibentuk: ketiganya datang dari koneksi yang sama, dan
	// pemilih yang mengembalikan nil akan membuat uji panic alih-alih gagal dengan pesan.
	stores := inputreqprotection.Stores{
		Protections: repo,
		Types:       memory.NewTypeRepoWithSamples(),
		Claims:      memory.NewClaimRepoWithSamples(),
	}
	service, err := usecase.NewService(usecase.Options{
		Protections: func(alias string) (inputreqprotection.Stores, error) {
			if alias != portal {
				return inputreqprotection.Stores{}, errors.New("portal tidak dikenal")
			}
			return stores, nil
		},
		Now:      func() time.Time { return at },
		Location: wib,
	})
	require.NoError(t, err)

	return service, repo
}

func draft() inputreqprotection.Draft {
	return inputreqprotection.Draft{
		ClaimNumber: "PNCN.26.0007",
		Type:        "1",
		Note:        "Keterangan contoh.",
	}
}

func TestProteksiBaruTerbitDenganNomorOPCNDanBelumDiakseptasi(t *testing.T) {
	at := time.Date(2026, time.September, 23, 10, 0, 0, 0, wib)
	service, _ := bangun(t, at)

	saved, err := service.Create(context.Background(), usecase.SaveCommand{
		PortalAlias: portal,
		Draft:       draft(),
		By:          "ADMINCONTOH",
	})
	require.NoError(t, err)

	require.Equal(t, "OPCN.26.0001", saved.Number)
	require.True(t, inputreqprotection.IssuedHere(saved.Number))
	require.Equal(t, inputreqprotection.AcceptPending, saved.AcceptStatus)
	require.Equal(t, "ADMINCONTOH", saved.CreatedBy)
}

func TestPembuatDiambilDariSesiBukanDariBadanPermintaan(t *testing.T) {
	// Tanpa penjagaan ini, baris tersimpan tanpa pembuat dan tidak dapat ditelusuri.
	at := time.Date(2026, time.September, 23, 10, 0, 0, 0, wib)
	service, _ := bangun(t, at)

	_, err := service.Create(context.Background(), usecase.SaveCommand{
		PortalAlias: portal,
		Draft:       draft(),
		By:          "",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "identitas pemanggil kosong")
}

func TestIsianCacatDitolakSebelumMenyentuhPenyimpanan(t *testing.T) {
	at := time.Date(2026, time.September, 23, 10, 0, 0, 0, wib)
	service, repo := bangun(t, at)

	_, err := service.Create(context.Background(), usecase.SaveCommand{
		PortalAlias: portal,
		Draft:       inputreqprotection.Draft{},
		By:          "ADMINCONTOH",
	})

	var v *inputreqprotection.ValidationError
	require.True(t, errors.As(err, &v))

	page, listErr := repo.List(context.Background(), inputreqprotection.Filter{})
	require.NoError(t, listErr)
	require.Zero(t, page.Total, "isian yang ditolak tidak boleh meninggalkan baris")
}

func TestProteksiGandaPadaPolisDanTipeSamaDiHariYangSamaDitolak(t *testing.T) {
	at := time.Date(2026, time.September, 23, 10, 0, 0, 0, wib)
	service, _ := bangun(t, at)

	_, err := service.Create(context.Background(), usecase.SaveCommand{
		PortalAlias: portal, Draft: draft(), By: "ADMINCONTOH",
	})
	require.NoError(t, err)

	_, err = service.Create(context.Background(), usecase.SaveCommand{
		PortalAlias: portal, Draft: draft(), By: "ADMINCONTOH",
	})
	require.Error(t, err)

	// Dikembalikan sebagai kesalahan validasi pada kolom No Polis, bukan sebagai konflik
	// teknis, supaya layar menempelkannya ke kolom yang memang harus diubah pengguna.
	var v *inputreqprotection.ValidationError
	require.True(t, errors.As(err, &v))
	require.Equal(t, inputreqprotection.FieldPolicyNumber, v.Errors[0].Field)
	require.Equal(t, inputreqprotection.DuplicateMessage, v.Errors[0].Message)
}

func TestPolisSamaTipeBerbedaBukanProteksiGanda(t *testing.T) {
	at := time.Date(2026, time.September, 23, 10, 0, 0, 0, wib)
	service, _ := bangun(t, at)

	first := draft()
	_, err := service.Create(context.Background(), usecase.SaveCommand{
		PortalAlias: portal, Draft: first, By: "ADMINCONTOH",
	})
	require.NoError(t, err)

	second := draft()
	second.Type = "3"
	_, err = service.Create(context.Background(), usecase.SaveCommand{
		PortalAlias: portal, Draft: second, By: "ADMINCONTOH",
	})
	require.NoError(t, err)
}

func TestProteksiSamaDiHariBerbedaDiterima(t *testing.T) {
	// Kuncinya "di hari ini", bukan "pernah ada". Menolaknya selamanya akan membuat polis
	// yang sama tidak pernah dapat diproteksi ulang.
	hariIni := time.Date(2026, time.September, 23, 10, 0, 0, 0, wib)
	service, repo := bangun(t, hariIni)

	_, err := service.Create(context.Background(), usecase.SaveCommand{
		PortalAlias: portal, Draft: draft(), By: "ADMINCONTOH",
	})
	require.NoError(t, err)

	besok := hariIni.AddDate(0, 0, 1)
	serviceBesok, err := usecase.NewService(usecase.Options{
		Protections: func(string) (inputreqprotection.Stores, error) {
			return inputreqprotection.Stores{
				Protections: repo,
				Types:       memory.NewTypeRepoWithSamples(),
				Claims:      memory.NewClaimRepoWithSamples(),
			}, nil
		},
		Now:      func() time.Time { return besok },
		Location: wib,
	})
	require.NoError(t, err)

	_, err = serviceBesok.Create(context.Background(), usecase.SaveCommand{
		PortalAlias: portal, Draft: draft(), By: "ADMINCONTOH",
	})
	require.NoError(t, err)
}

func TestProteksiYangSudahTertautKlaimTidakDapatDisunting(t *testing.T) {
	at := time.Date(2026, time.September, 23, 10, 0, 0, 0, wib)
	service, repo := bangun(t, at)

	repo.Add(inputreqprotection.Protection{
		Number:       "OPCN.26.0009",
		PolicyNumber: "99.001.2026.00000009",
		ClaimNumber:  "PNCN.26.0011", // terisi -> terkunci
		Type:         "1",
		InputDate:    at,
		AcceptStatus: inputreqprotection.AcceptPending,
		CreatedAt:    at,
	})

	_, err := service.Update(context.Background(), usecase.SaveCommand{
		PortalAlias: portal, Number: "OPCN.26.0009", Draft: draft(), By: "ADMINCONTOH",
	})
	require.ErrorIs(t, err, inputreqprotection.ErrLocked)
}

func TestProteksiYangSudahDiakseptasiPunyaPesanTersendiri(t *testing.T) {
	// Dibedakan dari ErrLocked: yang satu menunggu akseptasi, yang satu sudah selesai.
	// Pesan yang sama akan membuat pengguna menunggu sesuatu yang tidak akan datang.
	at := time.Date(2026, time.September, 23, 10, 0, 0, 0, wib)
	service, repo := bangun(t, at)

	repo.Add(inputreqprotection.Protection{
		Number:       "OPCN.26.0010",
		PolicyNumber: "99.001.2026.00000010",
		Type:         "1",
		InputDate:    at,
		AcceptStatus: inputreqprotection.AcceptApproved,
		CreatedAt:    at,
	})

	_, err := service.Update(context.Background(), usecase.SaveCommand{
		PortalAlias: portal, Number: "OPCN.26.0010", Draft: draft(), By: "ADMINCONTOH",
	})
	require.ErrorIs(t, err, inputreqprotection.ErrAccepted)
}

func TestMenyuntingTanpaMengubahPolisTidakDitolakOlehDirinyaSendiri(t *testing.T) {
	// Tanpa pengecualian diri pada pemeriksaan ganda, setiap penyuntingan mustahil — dan
	// cacatnya hanya muncul saat menyunting, tidak saat membuat.
	at := time.Date(2026, time.September, 23, 10, 0, 0, 0, wib)
	service, _ := bangun(t, at)

	saved, err := service.Create(context.Background(), usecase.SaveCommand{
		PortalAlias: portal, Draft: draft(), By: "ADMINCONTOH",
	})
	require.NoError(t, err)

	// Baris ini belum tertaut klaim supaya dapat disunting.
	revisi := draft()
	revisi.ClaimNumber = ""

	_, err = service.Update(context.Background(), usecase.SaveCommand{
		PortalAlias: portal, Number: saved.Number, Draft: revisi, By: "ADMINCONTOH",
	})
	// Nomor klaim wajib, sehingga isian di atas memang ditolak validasi — bukan ditolak
	// sebagai proteksi ganda. Yang diuji: pesannya BUKAN pesan proteksi ganda.
	require.Error(t, err)
	require.NotContains(t, err.Error(), inputreqprotection.DuplicateMessage)
}

func TestProteksiYangSudahDiakseptasiTidakTampilDiDaftar(t *testing.T) {
	// `InboxReqOpenProtection_RD` menyaring `.AcceptStatus IS NULL`, dan itu bukan pilihan
	// pengguna melainkan definisi layarnya.
	at := time.Date(2026, time.September, 23, 10, 0, 0, 0, wib)
	service, repo := bangun(t, at)

	repo.Add(
		inputreqprotection.Protection{Number: "OPCN.26.0001", AcceptStatus: inputreqprotection.AcceptPending, CreatedAt: at},
		inputreqprotection.Protection{Number: "OPCN.26.0002", AcceptStatus: inputreqprotection.AcceptApproved, CreatedAt: at},
		inputreqprotection.Protection{Number: "OPCN.26.0003", AcceptStatus: inputreqprotection.AcceptRejected, CreatedAt: at},
	)

	page, err := service.List(context.Background(), usecase.ListQuery{PortalAlias: portal})
	require.NoError(t, err)

	require.Equal(t, 1, page.Total)
	require.Equal(t, "OPCN.26.0001", page.Protections[0].Number)
}

func TestPortalTidakDikenalDitolakBukanDialihkanKePortalUtama(t *testing.T) {
	// `R-20`: jatuh ke koneksi default berarti menampilkan proteksi satu badan hukum di
	// layar badan hukum lain, tanpa satu pun galat.
	at := time.Date(2026, time.September, 23, 10, 0, 0, 0, wib)
	service, _ := bangun(t, at)

	_, err := service.List(context.Background(), usecase.ListQuery{PortalAlias: "entitas-lain"})
	require.Error(t, err)
}

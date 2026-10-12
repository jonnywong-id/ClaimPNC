package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxrclpucl"
	"claim-pnc/internal/inboxrclpucl/repo/memory"
	"claim-pnc/internal/inboxrclpucl/suratpdf"
	"claim-pnc/internal/inboxrclpucl/usecase"
	"claim-pnc/internal/platform/clock"
)

// repoMenolakPindah adalah penyimpanan memori yang perpindahan tahapnya SELALU ditolak,
// dan yang mencatat apakah kedua penulisan lain sempat dijalankan.
//
// Ia membungkus penyimpanan sungguhan alih-alih menirunya seluruhnya: yang hendak diuji
// adalah URUTAN pemanggilan, bukan isi penyimpanannya.
type repoMenolakPindah struct {
	*memory.Store

	menandaiPUCLSelesai bool
	menyimpanIsian      bool
}

func (r *repoMenolakPindah) MoveToSendToAnalyst(context.Context, string, string) error {
	// Penolakan yang BENAR-BENAR terjadi di produksi pada klaim PNCN.26.31: klaimnya belum
	// punya PIC Teknik, sehingga tidak ada yang dapat menerimanya di tahap Send To Analis.
	return inboxrclpucl.ErrTechnicalPICUnknown
}

func (r *repoMenolakPindah) ReturnToAnalyst(ctx context.Context, ref, caller string) error {
	r.menandaiPUCLSelesai = true
	return r.Store.ReturnToAnalyst(ctx, ref, caller)
}

func (r *repoMenolakPindah) SaveReceipt(
	ctx context.Context,
	ref string,
	in inboxrclpucl.ReceiptInput,
	caller string,
) error {
	r.menyimpanIsian = true
	return r.Store.SaveReceipt(ctx, ref, in, caller)
}

// TestKirimYangDitolakTidakMenulisApaPun menjaga agar klaim tidak pernah hilang dari setiap
// layar ketika perpindahan tahapnya ditolak.
//
// # Kegagalan yang dijaga di sini pernah terjadi
//
// Sampai 2026-10-05 urutannya: simpan isian → tandai PUCL selesai → pindahkan tahap. Ketiganya
// menutup transaksinya sendiri-sendiri, sehingga penolakan pada langkah terakhir meninggalkan
// `PUCL_APPROVE = '1'` tanpa satu pun tugas Send To Analis terbuka. Klaimnya keluar dari
// SELURUH tab RCL/PUCL dan belum menjadi pekerjaan siapa pun — hilang dari setiap layar,
// tanpa satu pun galat yang menyebutkan sebabnya.
//
// Yang diuji karena itu bukan pesan galatnya melainkan APA YANG TIDAK TERTULIS.
func TestKirimYangDitolakTidakMenulisApaPun(t *testing.T) {
	store := memory.NewSampleStore()
	penolak := &repoMenolakPindah{Store: store}

	s, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (inboxrclpucl.Repo, error) { return penolak, nil },
		Letters:      suratpdf.Renderer{},
		Clock:        clock.FixedAt(time.Date(2026, 10, 2, 7, 5, 9, 0, time.UTC)),
	})
	require.NoError(t, err)

	// Klaim dicari dari ANTREANNYA, bukan dari baris contoh mana pun: yang dibuktikan adalah
	// klaim itu MASIH di antrean sesudah penolakan, dan klaim yang tidak pernah ada di sana
	// tidak dapat membuktikannya.
	detail, kind := klaimDiAntrean(t, s, store)

	_, err = aksiIsi(t, s, detail.Reference, kind, isianLengkap())
	require.ErrorIs(t, err, inboxrclpucl.ErrTechnicalPICUnknown)

	require.False(t, penolak.menandaiPUCLSelesai,
		"penolakan perpindahan tetap menandai PUCL selesai; klaimnya keluar dari seluruh "+
			"tab RCL/PUCL tanpa pernah menjadi pekerjaan siapa pun")
	require.False(t, penolak.menyimpanIsian,
		"penolakan perpindahan tetap menulis isian; tidak satu pun tulisan boleh terjadi "+
			"ketika tindakannya sendiri ditolak")

	require.True(t, adaDiAntrean(t, s, detail.Reference),
		"klaim hilang dari antrean RCL/PUCL meski tindakannya ditolak")
}

// TestKirimYangDitolakValidasiLebihDuluDariPerpindahan menjaga isian yang kosong tetap
// ditolak sebagai kesalahan pengisian, bukan sebagai penolakan perpindahan.
//
// Urutan baru mengerjakan perpindahan lebih dulu, dan tanpa penjaga ini validasi isian dapat
// ikut berpindah ke belakang — sehingga petugas yang lupa mengisi catatan menerima kalimat
// tentang PIC Teknik alih-alih tentang isian yang kosong.
func TestKirimYangDitolakValidasiLebihDuluDariPerpindahan(t *testing.T) {
	store := memory.NewSampleStore()
	penolak := &repoMenolakPindah{Store: store}

	s, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (inboxrclpucl.Repo, error) { return penolak, nil },
		Letters:      suratpdf.Renderer{},
		Clock:        clock.FixedAt(time.Date(2026, 10, 2, 7, 5, 9, 0, time.UTC)),
	})
	require.NoError(t, err)

	detail, kind := klaimDiAntrean(t, s, store)

	_, err = aksiIsi(t, s, detail.Reference, kind, inboxrclpucl.ReceiptInput{})

	var invalid *inboxrclpucl.ValidationError
	require.ErrorAs(t, err, &invalid,
		"isian kosong harus ditolak sebagai kesalahan pengisian, bukan sebagai penolakan "+
			"perpindahan tahap")
}

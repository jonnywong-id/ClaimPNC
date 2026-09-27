package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxoutstanding"
	"claim-pnc/internal/inboxoutstanding/usecase"
)

// recordingRepo mencatat filter yang diterimanya, supaya uji dapat memeriksa APA yang
// dikirim ke penyimpanan — bukan hanya apa yang dikembalikan.
type recordingRepo struct {
	got inboxoutstanding.Filter

	// gotExport mencatat penyaring unduhan, terpisah dari got.
	//
	// Keduanya sengaja DUA field, bukan satu: yang diuji justru bahwa kedua jalur menyaring
	// hal yang berbeda, dan satu field bersama akan menyembunyikan perbedaannya.
	gotExport inboxoutstanding.ExportFilter

	// line adalah lini bisnis yang dikembalikan LineBusinessFor.
	line inboxoutstanding.LineBusiness

	// legacy adalah identitas lama yang dikembalikan LegacyOperatorFor.
	legacy string

	// gotSummary mencatat penyaring yang diterima ringkasan.
	gotSummary inboxoutstanding.Filter
}

func (r *recordingRepo) LegacyOperatorFor(_ context.Context, _ string) (string, error) {
	return r.legacy, nil
}

func (r *recordingRepo) SummarizeDocumentStatus(_ context.Context, f inboxoutstanding.Filter) (inboxoutstanding.Summary, error) {
	r.gotSummary = f
	return inboxoutstanding.Summary{}, nil
}

func (r *recordingRepo) List(_ context.Context, f inboxoutstanding.Filter) (inboxoutstanding.Page, error) {
	r.got = f
	return inboxoutstanding.Page{}, nil
}

func (r *recordingRepo) Export(_ context.Context, f inboxoutstanding.ExportFilter) (inboxoutstanding.Page, error) {
	r.gotExport = f
	return inboxoutstanding.Page{}, nil
}

func (r *recordingRepo) LineBusinessFor(_ context.Context, _ string) (inboxoutstanding.LineBusiness, error) {
	return r.line, nil
}

// selectorFor membungkus satu repo menjadi pemilih yang mengabaikan alias.
//
// Dipakai uji yang tidak sedang memeriksa pemilihan portal itu sendiri.
func selectorFor(repo inboxoutstanding.Repo) inboxoutstanding.RepoSelector {
	return func(string) (inboxoutstanding.Repo, error) { return repo, nil }
}

// testPortal adalah alias portal yang dipakai uji yang tidak sedang menguji pemilihan
// portal itu sendiri.
const testPortal = "ASM"

func TestIdentitasKosongDitolakBukanDilayaniTanpaBatas(t *testing.T) {
	repo := &recordingRepo{}
	service, err := usecase.NewService(selectorFor(repo))
	require.NoError(t, err)

	_, err = service.List(context.Background(), usecase.Query{LoginID: "", PortalAlias: testPortal})
	require.Error(t, err,
		"rute dilindungi sesi; identitas kosong adalah cacat yang tidak boleh disembunyikan")
}

func TestPenyaringDariLayarDiteruskanApaAdanya(t *testing.T) {
	repo := &recordingRepo{}
	service, err := usecase.NewService(selectorFor(repo))
	require.NoError(t, err)

	_, err = service.List(context.Background(), usecase.Query{
		LoginID:    "BUDISANTOSO",
		Search:     "PNCN.26",
		Stage:      "Komite",
		BranchCode: "JKT",
		Limit:      10,
		Offset:     20,
	})
	require.NoError(t, err)

	require.Equal(t, "PNCN.26", repo.got.Search)
	require.Equal(t, "Komite", repo.got.Stage)
	require.Equal(t, "JKT", repo.got.BranchCode)
	require.Equal(t, 10, repo.got.Limit)
	require.Equal(t, 20, repo.got.Offset)
}

func TestServiceMenolakDibentukTanpaSeamnya(t *testing.T) {
	_, err := usecase.NewService(nil)
	require.Error(t, err)
}

// Portal yang tidak dapat dipilih menghentikan permintaan.
//
// Ini inti `R-20`: jatuh ke koneksi mana pun sebagai cadangan berarti menampilkan klaim
// satu badan hukum di layar badan hukum lain — kegagalan yang TIDAK terlihat sebagai
// galat, karena layarnya tampil normal dan angkanya masuk akal.
func TestPortalYangTidakDapatDipilihMenghentikanPermintaan(t *testing.T) {
	boom := errors.New("portal belum siap")
	selector := func(string) (inboxoutstanding.Repo, error) { return nil, boom }

	service, err := usecase.NewService(selector)
	require.NoError(t, err)

	_, err = service.List(context.Background(), usecase.Query{
		LoginID:     "SIAPAPUN",
		PortalAlias: "ENTAH",
	})
	require.ErrorIs(t, err, boom, "galat pemilihan portal diteruskan apa adanya")
}

// Portal yang diminta benar-benar diteruskan ke pemilih.
//
// Tanpa uji ini, sebuah cacat yang mengirim alias kosong ke pemilih tidak akan tertangkap:
// pemilih yang lalai akan mengembalikan koneksi utama, dan hasilnya tampak benar.
func TestAliasPortalDiteruskanKePemilihApaAdanya(t *testing.T) {
	var diterima string
	selector := func(alias string) (inboxoutstanding.Repo, error) {
		diterima = alias
		return &recordingRepo{}, nil
	}

	service, err := usecase.NewService(selector)
	require.NoError(t, err)

	_, err = service.List(context.Background(), usecase.Query{
		LoginID:     "SIAPAPUN",
		PortalAlias: "SIMASNET",
	})
	require.NoError(t, err)
	require.Equal(t, "SIMASNET", diterima)
}

// Unduhan TIDAK membawa pemilik pekerjaan ke penyimpanan.
//
// Ini uji yang paling menentukan pada berkas ini. Ia memeriksa APA yang dikirim ke
// penyimpanan, bukan apa yang dikembalikan — dan justru di situlah cacatnya dulu
// bersembunyi: export memanggil ulang List, sehingga `AssignedTo` ikut terkirim tanpa satu
// baris kode pun yang menyatakannya, dan hasilnya tampak wajar.
func TestUnduhanTidakMembawaPemilikPekerjaan(t *testing.T) {
	repo := &recordingRepo{line: inboxoutstanding.LineNonMBU}
	service, err := usecase.NewService(selectorFor(repo))
	require.NoError(t, err)

	_, err = service.Export(context.Background(), usecase.ExportQuery{
		LoginID:     "BUDISANTOSO",
		PortalAlias: testPortal,
	})
	require.NoError(t, err)

	require.Empty(t, repo.got.AssignedTo,
		"jalur daftar tidak boleh ikut terpakai saat mengunduh")
	require.Equal(t, inboxoutstanding.LineNonMBU, repo.gotExport.LineBusiness,
		"yang membatasi unduhan adalah lini bisnis, bukan identitas")
}

// Lini bisnis pemanggil menentukan cakupan unduhan.
func TestLiniBisnisPemanggilMenentukanCakupanUnduhan(t *testing.T) {
	for _, lini := range []inboxoutstanding.LineBusiness{
		inboxoutstanding.LinePA,
		inboxoutstanding.LineTravel,
		inboxoutstanding.LineBonding,
		inboxoutstanding.LineNonMBU,
		inboxoutstanding.LineUnknown,
	} {
		repo := &recordingRepo{line: lini}
		service, err := usecase.NewService(selectorFor(repo))
		require.NoError(t, err)

		_, err = service.Export(context.Background(), usecase.ExportQuery{
			LoginID:     "SIAPAPUN",
			PortalAlias: testPortal,
		})
		require.NoError(t, err)
		require.Equal(t, lini, repo.gotExport.LineBusiness, string(lini))
	}
}

// Petugas tanpa lini bisnis tetap dilayani, tidak ditolak.
//
// Sistem lama menyetel fragmen cakupannya menjadi kosong dan mengembalikan seluruh klaim
// yang masih berjalan (`ExportDataDetailKlaim-Act`, step terakhir). Perilaku itu
// dipertahankan (`P-5`); menolaknya akan membuat export gagal bagi petugas yang kolomnya
// belum terisi — dan kolom itu baru terisi pada sebagian petugas.
func TestPetugasTanpaLiniBisnisTetapDapatMengunduh(t *testing.T) {
	repo := &recordingRepo{line: inboxoutstanding.LineUnknown}
	service, err := usecase.NewService(selectorFor(repo))
	require.NoError(t, err)

	hasil, err := service.Export(context.Background(), usecase.ExportQuery{
		LoginID:     "BELUMTERISI",
		PortalAlias: testPortal,
	})
	require.NoError(t, err)
	require.Equal(t, inboxoutstanding.LineUnknown, hasil.LineBusiness)
}

// Ringkasan menempuh identitas yang SAMA dengan daftar.
//
// Bila keduanya menyimpang, angka pada donut menghitung populasi yang berbeda dari isi
// grid — dan tidak ada galat yang menandainya, karena keduanya sama-sama masuk akal.
func TestRingkasanMemakaiIdentitasYangSamaDenganDaftar(t *testing.T) {
	repo := &recordingRepo{legacy: "NAMALAMA"}
	service, err := usecase.NewService(selectorFor(repo))
	require.NoError(t, err)

	q := usecase.Query{
		LoginID:     "orang@contoh.co.id",
		PortalAlias: testPortal,
		Search:      "PNCN.26",
		BranchCode:  "JKT",
	}

	_, err = service.List(context.Background(), q)
	require.NoError(t, err)
	_, err = service.Summary(context.Background(), q)
	require.NoError(t, err)

	require.Equal(t, repo.got.AssignedTo, repo.gotSummary.AssignedTo)
	require.Equal(t, repo.got.AssignedToLegacy, repo.gotSummary.AssignedToLegacy)
	require.Equal(t, "NAMALAMA", repo.gotSummary.AssignedToLegacy)

	// Penyaring layar berlaku pada keduanya: donut harus meringkas apa yang sedang dilihat,
	// bukan seluruh inbox.
	require.Equal(t, "PNCN.26", repo.gotSummary.Search)
	require.Equal(t, "JKT", repo.gotSummary.BranchCode)
}

// Ringkasan menolak identitas kosong, sama seperti daftar.
func TestRingkasanMenolakIdentitasKosong(t *testing.T) {
	repo := &recordingRepo{}
	service, err := usecase.NewService(selectorFor(repo))
	require.NoError(t, err)

	_, err = service.Summary(context.Background(), usecase.Query{PortalAlias: testPortal})
	require.Error(t, err)
}

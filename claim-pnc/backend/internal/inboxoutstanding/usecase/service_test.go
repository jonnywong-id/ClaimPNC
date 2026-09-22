package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxoutstanding"
	"claim-pnc/internal/inboxoutstanding/repo/memory"
	"claim-pnc/internal/inboxoutstanding/usecase"
)

// stubLines memungkinkan uji menyuntikkan kegagalan pembacaan lini bisnis — keadaan yang
// TIDAK dapat dihasilkan adapter memori, karena di sana "tidak ada baris" memang bukan
// kegagalan.
type stubLines struct {
	line string
	err  error
}

func (s stubLines) LineBusinessFor(context.Context, string) (string, error) {
	return s.line, s.err
}

// recordingRepo mencatat filter yang diterimanya, supaya uji dapat memeriksa APA yang
// dikirim ke penyimpanan — bukan hanya apa yang dikembalikan.
type recordingRepo struct {
	got inboxoutstanding.Filter
}

func (r *recordingRepo) List(_ context.Context, f inboxoutstanding.Filter) (inboxoutstanding.Page, error) {
	r.got = f
	return inboxoutstanding.Page{}, nil
}

// selectorFor membungkus satu repo menjadi pemilih yang mengabaikan alias.
//
// Dipakai uji yang tidak sedang memeriksa pemilihan portal itu sendiri.
func selectorFor(repo inboxoutstanding.Repo) inboxoutstanding.RepoSelector {
	return func(string) (inboxoutstanding.Repo, error) { return repo, nil }
}

// testPortal adalah alias portal yang dipakai uji; nilainya tidak penting selama pemilih
// mengabaikannya.
const testPortal = "ASM"

func TestBatasDataDiturunkanDariIdentitasBukanDariPermintaan(t *testing.T) {
	repo := &recordingRepo{}
	service, err := usecase.NewService(selectorFor(repo), stubLines{line: inboxoutstanding.LinePA})
	require.NoError(t, err)

	_, err = service.List(context.Background(), usecase.Query{LoginID: "DEWILESTARI", PortalAlias: testPortal})
	require.NoError(t, err)

	require.False(t, repo.got.Scope.Unrestricted)
	require.Equal(t, []string{"002"}, repo.got.Scope.GroupPanels,
		"batas data harus berasal dari lini bisnis pemanggil")
}

// Kegagalan membaca lini bisnis TIDAK boleh mematikan layar.
//
// Kolom LINEBUSINESS belum ada sampai migrasi 0004 dijalankan, sehingga pembacaannya gagal
// di setiap lingkungan hari ini. Menghentikan permintaan berarti layar mati total sampai
// perubahan skema selesai.
func TestGagalMembacaLiniBisnisTidakMematikanLayar(t *testing.T) {
	repo := &recordingRepo{}
	boom := errors.New("ORA-00904: identifier tidak sah")

	service, err := usecase.NewService(selectorFor(repo), stubLines{err: boom})
	require.NoError(t, err)

	result, err := service.List(context.Background(), usecase.Query{LoginID: "SIAPAPUN", PortalAlias: testPortal})
	require.NoError(t, err, "permintaan tetap dilayani")
	require.True(t, repo.got.Scope.Unrestricted, "jatuh ke tanpa batas, seperti Pega")
	require.ErrorIs(t, result.LineLookupError, boom,
		"kegagalannya tetap dibawa supaya pemanggil dapat mencatatnya")
}

// Kegagalan dan "tidak punya lini" menghasilkan batas data yang SAMA, sehingga keduanya
// harus dapat dibedakan lewat jalur lain. Bila tidak, kegagalan basis data menjadi tidak
// terlihat oleh siapa pun.
func TestTanpaLiniBisnisDapatDibedakanDariGagalMembacanya(t *testing.T) {
	repo := &recordingRepo{}
	service, err := usecase.NewService(selectorFor(repo), stubLines{line: ""})
	require.NoError(t, err)

	result, err := service.List(context.Background(), usecase.Query{LoginID: "SIAPAPUN", PortalAlias: testPortal})
	require.NoError(t, err)
	require.True(t, result.Scope.Unrestricted)
	require.NoError(t, result.LineLookupError, "tidak punya lini bukan kegagalan")
}

func TestIdentitasKosongDitolakBukanDilayaniTanpaBatas(t *testing.T) {
	repo := &recordingRepo{}
	service, err := usecase.NewService(selectorFor(repo), stubLines{})
	require.NoError(t, err)

	_, err = service.List(context.Background(), usecase.Query{LoginID: "", PortalAlias: testPortal})
	require.Error(t, err,
		"rute dilindungi sesi; identitas kosong adalah cacat yang tidak boleh disembunyikan")
}

func TestPenyaringDariLayarDiteruskanApaAdanya(t *testing.T) {
	repo := &recordingRepo{}
	service, err := usecase.NewService(selectorFor(repo), stubLines{line: inboxoutstanding.LineNonMBU})
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
	_, err := usecase.NewService(nil, stubLines{})
	require.Error(t, err)

	_, err = usecase.NewService(selectorFor(&recordingRepo{}), nil)
	require.Error(t, err)
}

// Uji ujung-ke-ujung ringan terhadap adapter memori: petugas PA hanya melihat klaim PA.
func TestPetugasPAHanyaMelihatKlaimLiniNya(t *testing.T) {
	repo := memory.NewRepoWithSamples()
	service, err := usecase.NewService(selectorFor(repo), repo)
	require.NoError(t, err)

	result, err := service.List(context.Background(), usecase.Query{
		LoginID: "DEWILESTARI", // terdaftar sebagai PA di data contoh
		Limit:   inboxoutstanding.MaxLimit,
	})
	require.NoError(t, err)
	require.NotEmpty(t, result.Page.Claims)

	for _, c := range result.Page.Claims {
		require.Equal(t, "002", c.GroupPanel,
			"klaim lini lain tidak boleh terlihat oleh petugas PA")
	}
}

// Pengguna yang tidak terdaftar melihat SELURUH lini — perilaku Pega yang Work Owner
// tetapkan ditiru apa adanya (`K-4`).
func TestPenggunaTanpaLiniMelihatSeluruhLini(t *testing.T) {
	repo := memory.NewRepoWithSamples()
	service, err := usecase.NewService(selectorFor(repo), repo)
	require.NoError(t, err)

	result, err := service.List(context.Background(), usecase.Query{
		LoginID: "ADMINPNC", // sengaja tidak terdaftar di data contoh
		Limit:   inboxoutstanding.MaxLimit,
	})
	require.NoError(t, err)
	require.True(t, result.Scope.Unrestricted)

	panels := map[string]bool{}
	for _, c := range result.Page.Claims {
		panels[c.GroupPanel] = true
	}
	require.Greater(t, len(panels), 1, "lebih dari satu lini terlihat")
}

// Portal yang tidak dapat dipilih menghentikan permintaan.
//
// Ini inti `R-20`: jatuh ke koneksi mana pun sebagai cadangan berarti menampilkan klaim
// satu badan hukum di layar badan hukum lain — kegagalan yang TIDAK terlihat sebagai
// galat, karena layarnya tampil normal dan angkanya masuk akal.
func TestPortalYangTidakDapatDipilihMenghentikanPermintaan(t *testing.T) {
	boom := errors.New("portal belum siap")
	selector := func(string) (inboxoutstanding.Repo, error) { return nil, boom }

	service, err := usecase.NewService(selector, stubLines{})
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

	service, err := usecase.NewService(selector, stubLines{})
	require.NoError(t, err)

	_, err = service.List(context.Background(), usecase.Query{
		LoginID:     "SIAPAPUN",
		PortalAlias: "SIMASNET",
	})
	require.NoError(t, err)
	require.Equal(t, "SIMASNET", diterima)
}

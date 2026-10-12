package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxanalystdoctor"
	"claim-pnc/internal/inboxanalystdoctor/repo/memory"
	"claim-pnc/internal/inboxanalystdoctor/usecase"
)

// recordingRepo mencatat argumen yang diterimanya.
//
// Ia dipakai membuktikan apa yang DITERUSKAN ke penyimpanan, bukan hanya apa yang
// dikembalikan — dua hal yang mudah tampak benar bersamaan padahal salah satunya salah.
type recordingRepo struct {
	inner *memory.Store

	lastOperator string
	lastFilter   inboxanalystdoctor.Filter
	calls        int
}

func (r *recordingRepo) List(
	ctx context.Context,
	operator string,
	f inboxanalystdoctor.Filter,
) (inboxanalystdoctor.Page, error) {
	r.lastOperator = operator
	r.lastFilter = f
	r.calls++
	return r.inner.List(ctx, operator, f)
}

// failingRepo selalu gagal, dipakai memastikan galat penyimpanan tidak tertelan.
type failingRepo struct{ err error }

func (r failingRepo) List(
	context.Context, string, inboxanalystdoctor.Filter,
) (inboxanalystdoctor.Page, error) {
	return inboxanalystdoctor.Page{}, r.err
}

const portalUtama = "ASM"

// newService membentuk layanan di atas repo yang diberikan.
func newService(t *testing.T, repo inboxanalystdoctor.Repo) *usecase.Service {
	t.Helper()

	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (inboxanalystdoctor.Repo, error) {
			if alias != portalUtama {
				return nil, errors.New("portal tidak tersedia")
			}
			return repo, nil
		},
	})
	require.NoError(t, err)
	return service
}

func TestNewServiceMenolakTanpaRepoSelector(t *testing.T) {
	_, err := usecase.NewService(usecase.Options{})

	require.Error(t, err)
}

// TestIdentitasDiperiksaSEBELUMPenyimpananDisentuh mengunci urutan pemeriksaan.
//
// Dua hal bergantung padanya. Yang pertama hemat: memilih repo untuk permintaan yang pasti
// ditolak adalah perjalanan yang terbuang. Yang kedua lebih penting: urutan ini membuat
// kegagalan sesi terbaca sebagai kegagalan sesi — bukan sebagai antrean kosong, yang tidak
// pernah dilaporkan siapa pun.
func TestIdentitasDiperiksaSEBELUMPenyimpananDisentuh(t *testing.T) {
	repo := &recordingRepo{inner: memory.NewSampleStore()}
	service := newService(t, repo)

	_, err := service.List(context.Background(), portalUtama,
		inboxanalystdoctor.Caller{}, inboxanalystdoctor.Filter{})

	require.ErrorIs(t, err, inboxanalystdoctor.ErrCallerUnknown)
	require.Zero(t, repo.calls, "penyimpanan tidak boleh disentuh tanpa identitas")
}

func TestPortalTidakDikenalMenghasilkanGalat(t *testing.T) {
	service := newService(t, &recordingRepo{inner: memory.NewSampleStore()})

	_, err := service.List(context.Background(), "ENTITAS-LAIN",
		inboxanalystdoctor.Caller{Login: memory.SampleOperator}, inboxanalystdoctor.Filter{})

	require.Error(t, err)
}

// TestLoginPemanggilDiteruskanApaAdanyaKePenyimpanan menjaga batas kewenangan.
//
// Bila yang diteruskan bukan login pemanggil — misalnya kosong, atau nilai dari isian layar —
// antrean yang tampil bukan milik orang yang membukanya.
func TestLoginPemanggilDiteruskanApaAdanyaKePenyimpanan(t *testing.T) {
	repo := &recordingRepo{inner: memory.NewSampleStore()}
	service := newService(t, repo)

	_, err := service.List(context.Background(), portalUtama,
		inboxanalystdoctor.Caller{Login: memory.SampleOperator}, inboxanalystdoctor.Filter{})

	require.NoError(t, err)
	require.Equal(t, memory.SampleOperator, repo.lastOperator)
}

// TestPenyaringDinormalkanSebelumDipakaiDanDikembalikan menguji keduanya sekaligus.
//
// Yang dikembalikan ke layar WAJIB angka yang benar-benar dipakai. Permintaan `batas=5000`
// dipangkas menjadi 100, dan tanpa mengembalikan 100, bilah halaman akan menghitung jumlah
// halaman dari angka yang tidak pernah berlaku.
func TestPenyaringDinormalkanSebelumDipakaiDanDikembalikan(t *testing.T) {
	repo := &recordingRepo{inner: memory.NewSampleStore()}
	service := newService(t, repo)

	listed, err := service.List(context.Background(), portalUtama,
		inboxanalystdoctor.Caller{Login: memory.SampleOperator},
		inboxanalystdoctor.Filter{Search: "  0311  ", Limit: 5000, Offset: -3})

	require.NoError(t, err)

	require.Equal(t, "0311", repo.lastFilter.Search)
	require.Equal(t, inboxanalystdoctor.MaxLimit, repo.lastFilter.Limit)
	require.Equal(t, 0, repo.lastFilter.Offset)

	require.Equal(t, "0311", listed.Filter.Search)
	require.Equal(t, inboxanalystdoctor.MaxLimit, listed.Filter.Limit)
}

func TestGalatPenyimpananDiteruskanDenganKonteks(t *testing.T) {
	sentinel := errors.New("koneksi terputus")
	service := newService(t, failingRepo{err: sentinel})

	_, err := service.List(context.Background(), portalUtama,
		inboxanalystdoctor.Caller{Login: memory.SampleOperator}, inboxanalystdoctor.Filter{})

	require.ErrorIs(t, err, sentinel)
	require.Contains(t, err.Error(), "antrean Analyst Doctor",
		"galat harus menyebut apa yang sedang diambil")
}

// TestSetiapKolomPunyaKunciDanJudul menjaga keterangan layar tetap utuh.
//
// Kolom tanpa kunci tidak dapat dipetakan ke field JSON mana pun; kolom tanpa judul digambar
// sebagai kepala kosong. Keduanya tidak menghasilkan galat — hanya layar yang salah.
func TestSetiapKolomPunyaKunciDanJudul(t *testing.T) {
	for _, column := range usecase.Columns() {
		require.NotEmpty(t, column.Key)
		require.NotEmptyf(t, column.Title, "kolom %s tidak punya judul", column.Key)
	}
}

// TestJudulKolomSamaPersisDenganHarness mengunci kedelapan judul terhadap buktinya.
//
// Sumbernya rule `pyCaption …` pada `Harness/inboxAnalystDoctor_Harness-Harness.xml`.
// `D-13` menetapkan teks yang dilihat pengguna mengikuti layar lama apa adanya, sehingga
// "memperbaiki" salah satunya adalah perubahan yang tidak punya dasar.
func TestJudulKolomSamaPersisDenganHarness(t *testing.T) {
	var titles []string
	for _, column := range usecase.Columns() {
		titles = append(titles, column.Title)
	}

	require.Equal(t, []string{
		"Nomor Case",
		"No Polis",
		"Nama Tertanggung",
		"Nama Cabang",
		"Tanggal Pendaftaran",
		"Nama Admin",
		"Komentar dari PIC Teknis",
		"Lama Waktu Klaim",
	}, titles)
}

// TestSalahKetikCaptionPegaTidakDibawa menjaga §4.7 tidak terulang.
//
// Harness memuat DUA rule caption yang nyaris sama — "Komentar dari PIC Teknis" dan
// "Komentar dari PIC Tekniks". Yang kedua salah ketik, dan salah ketik tidak dibawa; sama
// seperti `Broswse*` dan `Complience` yang juga tidak dibawa.
func TestSalahKetikCaptionPegaTidakDibawa(t *testing.T) {
	for _, column := range usecase.Columns() {
		require.NotEqual(t, "Komentar dari PIC Tekniks", column.Title)
	}
}

// TestKolomYangBelumTerbawaMenjelaskanDirinya menjaga sel kosong tidak tersamar.
//
// Kolom "Komentar dari PIC Teknis" masih kosong terhadap Oracle karena properti Pega-nya
// tidak terekspos. Tanpa keterangan, sel kosong terbaca sebagai data yang memang tidak ada —
// dan tidak ada seorang pun yang menanyakannya.
func TestKolomYangBelumTerbawaMenjelaskanDirinya(t *testing.T) {
	var found bool
	for _, column := range usecase.Columns() {
		if column.Key == "komentar_pic_teknis" {
			found = true
			require.NotEmpty(t, column.Note)

			// Keterangannya menyebut SEBABNYA, bukan sekadar "belum ada". Pembacanya
			// pengguna, sehingga kata `unexposed` sengaja tidak dipakai — yang dipakai
			// padanan Indonesianya.
			require.Contains(t, column.Note, "tidak terekspos")
		}
	}
	require.True(t, found, "kolom komentar_pic_teknis wajib ada")
}

// TestSelisihTerencanaDanKeterbatasanTidakKosong menjaga keduanya sampai ke layar.
//
// `D-54` menetapkan selisih di luar 13 butir `P-5` dinyatakan, bukan disimpan sebagai catatan
// teknis. Senarai kosong di sini berarti layar tidak menyatakan apa pun.
func TestSelisihTerencanaDanKeterbatasanTidakKosong(t *testing.T) {
	require.NotEmpty(t, usecase.PlannedDifferences())
	require.NotEmpty(t, usecase.Limitations())
}

// TestColumnsMengembalikanSalinan menjaga daftar kolom tidak dapat diubah dari luar.
func TestColumnsMengembalikanSalinan(t *testing.T) {
	first := usecase.Columns()
	first[0].Title = "DIUBAH"

	require.Equal(t, "Nomor Case", usecase.Columns()[0].Title)
}

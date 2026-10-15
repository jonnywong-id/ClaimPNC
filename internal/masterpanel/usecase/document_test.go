package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterpanel"
	masterpanelmemory "claim-pnc/internal/masterpanel/repo/memory"
	"claim-pnc/internal/masterpanel/usecase"
)

// layananUnggah merakit layanan beserta repo memori dan pengunggah palsu.
func layananUnggah(t *testing.T) (*usecase.Service, *masterpanelmemory.Repo, *masterpanelmemory.FakeUploader) {
	t.Helper()
	repo := masterpanelmemory.NewSampleRepo()
	uploader := &masterpanelmemory.FakeUploader{}
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (masterpanel.Store, error) { return repo, nil },
		Uploader:     uploader,
	})
	require.NoError(t, err)
	return service, repo, uploader
}

func perintahUnggah() usecase.UploadCommand {
	return usecase.UploadCommand{
		PortalAlias: "asm",
		PanelID:     masterpanelmemory.SampleList()[0].ID,
		FileName:    "panel.pdf",
		Content:     []byte("isi berkas"),
		Note:        "Foto panel",
		By:          usecase.Actor{Login: "PETUGAS"},
	}
}

func TestUnggahMenyimpanDanMenautkanDokumen(t *testing.T) {
	service, repo, uploader := layananUnggah(t)
	cmd := perintahUnggah()

	doc, err := service.UploadDocument(context.Background(), cmd, nil)
	require.NoError(t, err)

	require.NotEmpty(t, doc.DataID, "DATAID diterbitkan saat penyimpanan")
	require.NotEmpty(t, doc.ImageID, "IMAGEID datang dari layanan penyimpanan")
	require.Equal(t, cmd.PanelID, doc.PanelID)
	require.Equal(t, "PETUGAS", doc.UploadedBy)
	require.Len(t, uploader.Uploaded, 1, "berkasnya benar-benar dikirim ke penyimpanan")

	stored, err := repo.DocumentOf(context.Background(), cmd.PanelID)
	require.NoError(t, err)
	require.Equal(t, doc.DataID, stored.DataID)
}

// Panelnya dibaca SEBELUM berkasnya dikirim. Terbalik, satu salah ketik ID akan
// meninggalkan berkas yatim di layanan penyimpanan untuk kesalahan yang dapat diketahui
// dengan satu pembacaan murah.
func TestPanelTidakAdaDitolakSEBELUMBerkasTerkirim(t *testing.T) {
	service, _, uploader := layananUnggah(t)
	cmd := perintahUnggah()
	cmd.PanelID = "99999999"

	_, err := service.UploadDocument(context.Background(), cmd, nil)

	require.ErrorIs(t, err, masterpanel.ErrNotFound)
	require.Empty(t, uploader.Uploaded, "tidak boleh ada berkas yang terkirim")
}

func TestUnggahBerikutnyaMenggantiDokumenSebelumnya(t *testing.T) {
	service, repo, _ := layananUnggah(t)
	cmd := perintahUnggah()

	first, err := service.UploadDocument(context.Background(), cmd, nil)
	require.NoError(t, err)

	cmd.FileName = "panel-revisi.pdf"
	second, err := service.UploadDocument(context.Background(), cmd, nil)
	require.NoError(t, err)

	require.NotEqual(t, first.DataID, second.DataID, "setiap unggahan menerbitkan DATAID baru")

	stored, err := repo.DocumentOf(context.Background(), cmd.PanelID)
	require.NoError(t, err)
	require.Equal(t, second.DataID, stored.DataID,
		"satu panel memegang satu dokumen: tautannya menunjuk yang terbaru")
	require.Equal(t, "panel-revisi.pdf", stored.Name)
}

func TestTanpaPengunggahJalurUnggahDitolakDenganSebabnya(t *testing.T) {
	repo := masterpanelmemory.NewSampleRepo()
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (masterpanel.Store, error) { return repo, nil },
	})
	require.NoError(t, err)

	require.False(t, service.UploadAvailable(),
		"layar harus dapat menanyakan ketersediaannya sebelum menggambar tombol")

	_, err = service.UploadDocument(context.Background(), perintahUnggah(), nil)

	var upload *masterpanel.DocumentUploadError
	require.ErrorAs(t, err, &upload)
	require.Equal(t, masterpanel.UploadMisconfigured, upload.Kind)
}

// Layanan penyimpanan yang gagal TIDAK boleh meninggalkan baris metadata. Baris yang
// menunjuk ke IMAGEID yang tidak ada adalah dokumen yang terlihat ada tetapi tidak dapat
// dibuka — kegagalan yang baru ketahuan saat seseorang mengkliknya.
func TestPenyimpananGagalTidakMeninggalkanMetadata(t *testing.T) {
	service, repo, uploader := layananUnggah(t)
	uploader.Failure = errors.New("layanan penyimpanan mati")
	cmd := perintahUnggah()

	_, err := service.UploadDocument(context.Background(), cmd, nil)
	require.Error(t, err)

	_, err = repo.DocumentOf(context.Background(), cmd.PanelID)
	require.ErrorIs(t, err, masterpanel.ErrDocumentMissing)
}

// Metadata yang gagal SETELAH berkas terkirim digolongkan UploadHalfDone, bukan
// UploadUnavailable. Perbedaannya menentukan: yang kedua mengundang pengulangan, dan
// mengulang di sini menumpuk berkas ganda di layanan penyimpanan.
func TestMetadataGagalDigolongkanSeparuhJalan(t *testing.T) {
	repo := masterpanelmemory.NewSampleRepo()
	uploader := &masterpanelmemory.FakeUploader{}
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (masterpanel.Store, error) { return repo, nil },
		Uploader:     uploader,
	})
	require.NoError(t, err)

	cmd := perintahUnggah()
	// Panelnya terbaca dulu, lalu repo dibuat gagal tepat sebelum penyimpanan metadata.
	repo.SetError(nil)
	_, err = repo.Get(context.Background(), cmd.PanelID)
	require.NoError(t, err)

	gagal := &repoGagalSaatSimpan{Repo: repo, failure: errors.New("basis data menolak")}
	service, err = usecase.NewService(usecase.Options{
		RepoSelector: func(string) (masterpanel.Store, error) { return gagal, nil },
		Uploader:     uploader,
	})
	require.NoError(t, err)

	_, err = service.UploadDocument(context.Background(), cmd, nil)

	var upload *masterpanel.DocumentUploadError
	require.ErrorAs(t, err, &upload)
	require.Equal(t, masterpanel.UploadHalfDone, upload.Kind)
	require.Len(t, uploader.Uploaded, 1, "berkasnya memang sudah terkirim")
}

// repoGagalSaatSimpan meneruskan segalanya ke repo asli kecuali penyimpanan dokumen.
//
// Dibuat di sini, bukan sebagai sakelar di dalam repo memori, supaya repo memori tetap
// meniru perilaku adapter SQL apa adanya — menambahkan sakelar kegagalan per method akan
// membuatnya menerima keadaan yang tidak mungkin terjadi di produksi.
type repoGagalSaatSimpan struct {
	*masterpanelmemory.Repo
	failure error
}

func (r *repoGagalSaatSimpan) SaveDocument(
	context.Context, masterpanel.PanelDocument,
) (string, error) {
	return "", r.failure
}

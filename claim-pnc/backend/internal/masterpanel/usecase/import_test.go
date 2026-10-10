package usecase_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterpanel"
	masterpanelmemory "claim-pnc/internal/masterpanel/repo/memory"
	"claim-pnc/internal/masterpanel/usecase"
)

const csvHeader = "NAME,STS_REPAIR,STS_EDIT_QTY,STS_PREMIUM_REPAIR,STS_PECAH," +
	"STS_STICKER,STS_SISI,STS_RUSAK_PARAH,STS_AKTIF,EXCLUSION_C"

func layananImpor(t *testing.T) (*usecase.Service, *masterpanelmemory.Repo) {
	t.Helper()
	repo := masterpanelmemory.NewSampleRepo()
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (masterpanel.Store, error) { return repo, nil },
	})
	require.NoError(t, err)
	return service, repo
}

func imporPanel(t *testing.T, service *usecase.Service, isi string) usecase.ImportReport {
	t.Helper()
	report, err := service.ImportPanelCSV(
		context.Background(), "asm", []byte(isi), "", usecase.Actor{Login: "PETUGAS"}, nil)
	require.NoError(t, err)
	return report
}

func TestImporPanelBaruMenambahBaris(t *testing.T) {
	service, repo := layananImpor(t)

	report := imporPanel(t, service, csvHeader+"\nKap Mesin,GANTI,YA,YA,YA,YA,YA,YA,YA,YA\n")

	require.Equal(t, 1, report.Total)
	require.Equal(t, 1, report.Created)
	require.Equal(t, 0, report.Updated)
	require.Equal(t, 0, report.Failed)
	require.Len(t, report.Rows, 1)
	require.Equal(t, usecase.ImportCreated, report.Rows[0].Outcome)
	require.NotEmpty(t, report.Rows[0].ID, "ID diterbitkan server")

	found, err := repo.FindByName(context.Background(), "Kap Mesin")
	require.NoError(t, err)
	require.Equal(t, masterpanel.StatusPending, found.Status,
		"baris hasil unggah masuk antrean persetujuan, sama seperti jalur Simpan")
}

// Kuncinya NAMA — `ValidationMasterPanel-SQL.xml` mencocokkan `upper(trim(name))`, dan itu
// satu-satunya kueri pencarian baris modul ini selain lewat ID.
func TestImporPanelBernamaSamaMemperbaruiBarisYangAda(t *testing.T) {
	service, repo := layananImpor(t)
	ada := masterpanelmemory.SampleList()[0]

	report := imporPanel(t, service,
		csvHeader+"\n"+ada.Name+",JASA,YA,YA,YA,YA,YA,YA,YA,YA\n")

	require.Equal(t, 1, report.Updated)
	require.Equal(t, 0, report.Created)
	require.Equal(t, ada.ID, report.Rows[0].ID, "ID lama dipakai ulang, bukan diterbitkan baru")

	found, err := repo.Get(context.Background(), ada.ID)
	require.NoError(t, err)
	require.Equal(t, "2", found.RepairStatus, "JASA diterjemahkan menjadi 2")
}

// Pencocokan nama tidak peka huruf besar-kecil maupun spasi, sejalan dengan
// `upper(trim(name))` pada kueri Pega.
func TestPencocokanNamaTidakPekaKapitalisasi(t *testing.T) {
	service, _ := layananImpor(t)
	ada := masterpanelmemory.SampleList()[0]

	report := imporPanel(t, service,
		csvHeader+"\n  "+strings.ToUpper(ada.Name)+"  ,GANTI,YA,YA,YA,YA,YA,YA,YA,YA\n")

	require.Equal(t, 1, report.Updated, "baris yang sama tidak boleh terbaca sebagai baris baru")
}

// Satu baris yang gagal TIDAK membatalkan yang lain — perilaku Pega, yang memutar baris
// satu per satu tanpa transaksi yang membungkus keseluruhan.
func TestBarisGagalTidakMembatalkanBarisLain(t *testing.T) {
	service, repo := layananImpor(t)

	report := imporPanel(t, service, csvHeader+
		"\nPanel Sah,GANTI,YA,YA,YA,YA,YA,YA,YA,YA"+
		"\n,GANTI,YA,YA,YA,YA,YA,YA,YA,YA"+ // nama kosong -> ditolak validasi
		"\nPanel Sah Dua,GANTI,YA,YA,YA,YA,YA,YA,YA,YA\n")

	require.Equal(t, 3, report.Total)
	require.Equal(t, 2, report.Created)
	require.Equal(t, 1, report.Failed)

	require.Equal(t, usecase.ImportFailed, report.Rows[1].Outcome)
	require.NotEmpty(t, report.Rows[1].Message, "sebab kegagalannya harus disebutkan")
	require.Equal(t, 3, report.Rows[1].Line, "nomor baris mengikuti berkas, header sebagai baris 1")

	_, err := repo.FindByName(context.Background(), "Panel Sah Dua")
	require.NoError(t, err, "baris sesudah yang gagal tetap masuk")
}

// Laporan memuat SELURUH baris, bukan yang gagal saja: pengguna harus dapat membedakan
// "semua berhasil" dari "separuh berkas tidak terbaca".
func TestLaporanMemuatSeluruhBaris(t *testing.T) {
	service, _ := layananImpor(t)

	report := imporPanel(t, service, csvHeader+
		"\nSatu,GANTI,YA,YA,YA,YA,YA,YA,YA,YA\nDua,GANTI,YA,YA,YA,YA,YA,YA,YA,YA\n")

	require.Len(t, report.Rows, 2)
	require.Equal(t, report.Total, len(report.Rows))
}

func TestBerkasCacatDitolakSeluruhnya(t *testing.T) {
	service, _ := layananImpor(t)

	_, err := service.ImportPanelCSV(context.Background(), "asm",
		[]byte("NAME\nPintu\n"), "", usecase.Actor{Login: "PETUGAS"}, nil)

	require.ErrorIs(t, err, masterpanel.ErrCSVHeaderMissing)
}

// --- lokasi ---

func imporLokasi(t *testing.T, service *usecase.Service, isi string) usecase.ImportReport {
	t.Helper()
	report, err := service.ImportLocationCSV(
		context.Background(), "asm", []byte(isi), usecase.Actor{Login: "PETUGAS"}, nil)
	require.NoError(t, err)
	return report
}

// Unggah lokasi MENAMBAH, tidak mengganti. Berkas berisi satu baris untuk satu panel tidak
// boleh memusnahkan lokasi lain panel itu.
func TestImporLokasiMenambahBukanMengganti(t *testing.T) {
	service, repo := layananImpor(t)
	ada := masterpanelmemory.SampleList()[0]
	require.Len(t, ada.Location, 2, "panel contoh memang sudah punya dua lokasi")

	report := imporLokasi(t, service,
		"NAME,pyLabel,STS_SISI\n"+ada.Name+",DEPAN,DEPAN\n")

	require.Equal(t, 1, report.Created)
	require.Equal(t, 0, report.Failed)

	found, err := repo.Get(context.Background(), ada.ID)
	require.NoError(t, err)
	require.Len(t, found.Location, 3, "dua lokasi lama TETAP ada, satu ditambahkan")
}

// Lokasi yang SUDAH ADA tetap ditambahkan lagi — Pega memakai `LOKASI(<APPEND>)` tanpa satu
// pun pemeriksaan, sehingga mengunggah berkas yang sama dua kali menggandakan barisnya.
//
// Dilaporkan `diperbarui`, bukan `gagal`: barisnya memang masuk.
func TestImporLokasiYangSudahAdaTetapDitambahkanSepertiPega(t *testing.T) {
	service, repo := layananImpor(t)
	ada := masterpanelmemory.SampleList()[0]
	lama := ada.Location[0]

	report := imporLokasi(t, service,
		"NAME,pyLabel,STS_SISI\n"+ada.Name+","+lama.Name+","+lama.Side.Label()+"\n")

	require.Equal(t, 0, report.Failed)
	require.Equal(t, 1, report.Updated, "dilaporkan diperbarui karena lokasinya sudah ada")

	found, err := repo.Get(context.Background(), ada.ID)
	require.NoError(t, err)
	require.Len(t, found.Location, len(ada.Location)+1,
		"Pega MENGGANDAKAN barisnya; dedup adalah penyimpangan")
}

// Akibat langsung dari meniru Pega: panel yang lokasinya telanjur ganda TIDAK dapat
// disimpan ulang dari form sampai salah satunya dihapus — form memakai aturan penuh.
//
// Diuji supaya akibatnya tercatat, bukan ditemukan pengguna.
func TestLokasiGandaDitolakSaatDisimpanDariForm(t *testing.T) {
	service, repo := layananImpor(t)
	ada := masterpanelmemory.SampleList()[0]
	lama := ada.Location[0]

	imporLokasi(t, service,
		"NAME,pyLabel,STS_SISI\n"+ada.Name+","+lama.Name+","+lama.Side.Label()+"\n")

	found, err := repo.Get(context.Background(), ada.ID)
	require.NoError(t, err)

	_, err = service.Save(context.Background(), "asm", ada.ID,
		inputDariPanel(found), usecase.Actor{Login: "PETUGAS"}, nil)

	var invalid *masterpanel.ValidationError
	require.ErrorAs(t, err, &invalid, "form menolak lokasi ganda, dan itu memang aturannya")
}

// inputDariPanel menyalin panel menjadi Input, seperti yang dilakukan layar saat menyimpan.
func inputDariPanel(p masterpanel.Panel) masterpanel.Input {
	return masterpanel.Input{
		Name: p.Name, RepairStatus: p.RepairStatus, EditQuantityStatus: p.EditQuantityStatus,
		PremiumRepairStatus: p.PremiumRepairStatus, ShatterStatus: p.ShatterStatus,
		StickerStatus: p.StickerStatus, SideStatus: p.SideStatus,
		SevereDamageStatus: p.SevereDamageStatus, ActiveStatus: p.ActiveStatus,
		ExclusionC: p.ExclusionC, Location: p.Location,
	}
}

// Panel yang tidak ditemukan dilaporkan per baris, bukan menggagalkan seluruh berkas.
func TestLokasiUntukPanelTidakDikenalDilaporkanPerBaris(t *testing.T) {
	service, _ := layananImpor(t)
	ada := masterpanelmemory.SampleList()[0]

	report := imporLokasi(t, service, "NAME,pyLabel,STS_SISI\n"+
		"Panel Yang Tidak Ada,DEPAN,DEPAN\n"+
		ada.Name+",BELAKANG,BELAKANG\n")

	require.Equal(t, 2, report.Total)
	require.Equal(t, 1, report.Failed)
	require.Equal(t, 1, report.Created)
	require.Contains(t, report.Rows[0].Message, "tidak ditemukan")
}

// Beberapa baris untuk satu panel disimpan SEKALI, bukan sekali per baris.
func TestBeberapaLokasiSatuPanelDisimpanSekali(t *testing.T) {
	service, repo := layananImpor(t)
	ada := masterpanelmemory.SampleList()[0]

	report := imporLokasi(t, service, "NAME,pyLabel,STS_SISI\n"+
		ada.Name+",DEPAN,DEPAN\n"+
		ada.Name+",BELAKANG,BELAKANG\n")

	require.Equal(t, 2, report.Created)

	found, err := repo.Get(context.Background(), ada.ID)
	require.NoError(t, err)
	require.Len(t, found.Location, 4)
}

// Unggah lokasi TIDAK boleh diam-diam menghapus isian induk panelnya.
func TestImporLokasiTidakMengubahIsianInduk(t *testing.T) {
	service, repo := layananImpor(t)
	ada := masterpanelmemory.SampleList()[0]

	imporLokasi(t, service, "NAME,pyLabel,STS_SISI\n"+ada.Name+",DEPAN,DEPAN\n")

	found, err := repo.Get(context.Background(), ada.ID)
	require.NoError(t, err)
	require.Equal(t, ada.Name, found.Name)
	require.Equal(t, ada.RepairStatus, found.RepairStatus)
	require.Equal(t, ada.ExclusionC, found.ExclusionC)
}

// --- berkas CSV ditautkan sebagai dokumen, seperti Pega ---

// layananImporDenganPengunggah merakit layanan beserta pengunggah palsu.
func layananImporDenganPengunggah(
	t *testing.T,
) (*usecase.Service, *masterpanelmemory.Repo, *masterpanelmemory.FakeUploader) {
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

// `PNCUploadMasterPanel_Act` memanggil `PNCSaveAttachmentToDB` SEKALI lalu menyetel
// `TempPanel.CoverID` pada setiap baris — satu dokumen, banyak panel.
func TestBerkasCSVDisimpanSekaliDanDitautkanKeSetiapBaris(t *testing.T) {
	service, repo, uploader := layananImporDenganPengunggah(t)

	report, err := service.ImportPanelCSV(context.Background(), "asm",
		[]byte(csvHeader+"\nPanel Satu,GANTI,YA,YA,YA,YA,YA,YA,YA,YA"+
			"\nPanel Dua,JASA,YA,YA,YA,YA,YA,YA,YA,YA\n"),
		"master-panel.csv", usecase.Actor{Login: "PETUGAS"}, nil)
	require.NoError(t, err)
	require.Equal(t, 2, report.Created)

	require.Len(t, uploader.Uploaded, 1,
		"berkasnya dikirim SEKALI, bukan sekali per baris")
	require.Equal(t, "master-panel.csv", uploader.Uploaded[0].FileName)

	satu, err := repo.FindByName(context.Background(), "PANEL SATU")
	require.NoError(t, err)
	dua, err := repo.FindByName(context.Background(), "PANEL DUA")
	require.NoError(t, err)

	require.NotEmpty(t, satu.DocumentID)
	require.Equal(t, satu.DocumentID, dua.DocumentID,
		"kedua panel menunjuk baris lampiran yang SAMA")
}

// Akibat yang disadari: baris yang sudah punya dokumen KEHILANGAN tautannya, diganti berkas
// CSV. Jalur lokasi justru mempertahankannya — asimetri itu ada di sistem lama.
func TestUnggahMasterMenimpaTautanDokumenYangSudahAda(t *testing.T) {
	service, repo, _ := layananImporDenganPengunggah(t)
	ada := masterpanelmemory.SampleList()[0]

	_, err := repo.SaveDocument(context.Background(), masterpanel.PanelDocument{
		ImageID: "IMGLAMA", Name: "foto-lama.pdf", UploadedBy: "PETUGAS", PanelID: ada.ID,
	})
	require.NoError(t, err)

	sebelum, err := repo.Get(context.Background(), ada.ID)
	require.NoError(t, err)
	require.NotEmpty(t, sebelum.DocumentID)

	_, err = service.ImportPanelCSV(context.Background(), "asm",
		[]byte(csvHeader+"\n"+ada.Name+",GANTI,YA,YA,YA,YA,YA,YA,YA,YA\n"),
		"master-panel.csv", usecase.Actor{Login: "PETUGAS"}, nil)
	require.NoError(t, err)

	sesudah, err := repo.Get(context.Background(), ada.ID)
	require.NoError(t, err)
	require.NotEqual(t, sebelum.DocumentID, sesudah.DocumentID,
		"Pega menimpanya; meniru itu berarti tautan lama memang hilang")
}

// Tanpa pengunggah, unggahan barisnya TETAP berjalan — hanya tanpa lampiran. Membatalkan
// seluruh berkas karena lampirannya gagal menukar kerugian kecil dengan kerugian besar.
func TestTanpaPengunggahBarisTetapMasukTanpaLampiran(t *testing.T) {
	service, repo := layananImpor(t)

	report := imporPanel(t, service, csvHeader+"\nKap Mesin,GANTI,YA,YA,YA,YA,YA,YA,YA,YA\n")
	require.Equal(t, 1, report.Created)
	require.Equal(t, 0, report.Failed)

	found, err := repo.FindByName(context.Background(), "KAP MESIN")
	require.NoError(t, err)
	require.Empty(t, found.DocumentID)
}

// Unggah LOKASI tidak melampirkan apa pun: `PNCUploadLokasiSisiPanel_Act` justru
// MEMPERTAHANKAN CoverID yang sudah ada.
func TestUnggahLokasiTidakMenyentuhTautanDokumen(t *testing.T) {
	service, repo, uploader := layananImporDenganPengunggah(t)
	ada := masterpanelmemory.SampleList()[0]

	_, err := repo.SaveDocument(context.Background(), masterpanel.PanelDocument{
		ImageID: "IMGLAMA", Name: "foto-lama.pdf", UploadedBy: "PETUGAS", PanelID: ada.ID,
	})
	require.NoError(t, err)
	sebelum, err := repo.Get(context.Background(), ada.ID)
	require.NoError(t, err)

	imporLokasi(t, service, "NAME,pyLabel,STS_SISI\n"+ada.Name+",DEPAN,DEPAN\n")

	sesudah, err := repo.Get(context.Background(), ada.ID)
	require.NoError(t, err)
	require.Equal(t, sebelum.DocumentID, sesudah.DocumentID)
	require.Empty(t, uploader.Uploaded, "jalur lokasi tidak mengunggah berkas apa pun")
}

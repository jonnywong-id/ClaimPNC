package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/archivedokumenklaim"
	"claim-pnc/internal/archivedokumenklaim/gateway"
	"claim-pnc/internal/archivedokumenklaim/repo/memory"
	"claim-pnc/internal/archivedokumenklaim/usecase"
)

const portalAlias = "asm"

// fixedClock adalah jam tetap, supaya waktu pengiriman dapat diuji secara pasti.
type fixedClock struct{ at time.Time }

func (c fixedClock) Now() time.Time { return c.at }

// build merakit layanan beserta repo memori dan perekam gateway.
func build(t *testing.T) (*usecase.Service, *memory.Repo, *gateway.Recorder) {
	t.Helper()

	repo := memory.NewSampleRepo()
	recorder := gateway.NewRecorder()

	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (archivedokumenklaim.Repo, error) {
			require.Equal(t, portalAlias, alias)
			return repo, nil
		},
		Gateway: recorder,
		Clock:   fixedClock{at: time.Date(2026, 9, 24, 10, 0, 0, 0, time.UTC)},
	})
	require.NoError(t, err)

	return service, repo, recorder
}

func caller(position string) archivedokumenklaim.Caller {
	return archivedokumenklaim.Caller{Login: "USERARSIP1", Position: position}
}

func TestLayananMenolakRakitanTanpaSeam(t *testing.T) {
	_, err := usecase.NewService(usecase.Options{})
	require.Error(t, err)
}

func TestBukaLayarMenyerahkanIsiDropdown(t *testing.T) {
	service, _, _ := build(t)

	opened, err := service.Open(context.Background(), portalAlias, caller("NONMBU"))
	require.NoError(t, err)

	require.Len(t, opened.ClaimSearchTypes, 3)
	require.NotEmpty(t, opened.DocumentTypes)
	require.NotEmpty(t, opened.DocumentKinds)
	require.ElementsMatch(t,
		[]string{
			archivedokumenklaim.GroupPanelPersonalAccident,
			archivedokumenklaim.GroupPanelTravel,
		},
		opened.BranchScope.ExcludedGroupPanels,
	)
}

func TestBukaLayarTanpaIdentitasDitolak(t *testing.T) {
	service, _, _ := build(t)

	_, err := service.Open(context.Background(), portalAlias, archivedokumenklaim.Caller{})
	require.ErrorIs(t, err, archivedokumenklaim.ErrCallerUnknown)
}

// Pencarian kata kunci mencocokkan PERSIS, bukan sebagian — persis kueri lama.
func TestPencarianKataKunciCocokPersis(t *testing.T) {
	service, _, _ := build(t)

	found, err := service.Search(context.Background(), portalAlias,
		archivedokumenklaim.CriteriaInput{
			Mode:    string(archivedokumenklaim.ModeKeyword),
			Keyword: "BOX-A-01",
		},
		archivedokumenklaim.Pagination{},
	)
	require.NoError(t, err)
	require.Equal(t, 2, found.Page.Total, "dua berkas memakai boks ini")

	sebagian, err := service.Search(context.Background(), portalAlias,
		archivedokumenklaim.CriteriaInput{
			Mode:    string(archivedokumenklaim.ModeKeyword),
			Keyword: "BOX-A",
		},
		archivedokumenklaim.Pagination{},
	)
	require.NoError(t, err)
	require.Zero(t, sebagian.Page.Total, "pencarian sebagian TIDAK cocok, sama seperti Pega")
}

// Nama tipe dan jenis dokumen kosong bila kodenya tidak ada di master.
//
// Ia meniru LEFT JOIN pada kueri SQL, dan barisnya tetap tampil — bukan disembunyikan.
func TestNamaDokumenKosongSaatKodeTidakAdaDiMaster(t *testing.T) {
	service, _, _ := build(t)

	found, err := service.Search(context.Background(), portalAlias,
		archivedokumenklaim.CriteriaInput{
			Mode:    string(archivedokumenklaim.ModeKeyword),
			Keyword: "PNCN.26.0001",
		},
		archivedokumenklaim.Pagination{},
	)
	require.NoError(t, err)
	require.Len(t, found.Page.Files, 1)
	require.Equal(t, "9999", found.Page.Files[0].DocumentTypeCode)
	require.Empty(t, found.Page.Files[0].DocumentTypeName)
	require.Empty(t, found.Page.Files[0].DocumentKindName)
}

func TestPencarianKlaimByPolisMempersempit(t *testing.T) {
	service, _, _ := build(t)

	claims, err := service.SearchClaims(context.Background(), portalAlias,
		string(archivedokumenklaim.ClaimByPolicy), "POL-2024-000006")
	require.NoError(t, err)
	require.Len(t, claims, 2, "satu polis wajar membuahkan beberapa klaim")
}

// Tipe "No Klaim" mencocokkan KETIGA kolom sekaligus — persis kueri lama.
//
// Dropdown "Tipe Input Archive" karena itu praktis tidak mempersempit apa pun, dan itu
// perilaku sistem lama, bukan kelalaian pembacaan.
func TestPencarianKlaimByNomorJugaMencocokkanNamaDanPolis(t *testing.T) {
	service, _, _ := build(t)

	claims, err := service.SearchClaims(context.Background(), portalAlias,
		string(archivedokumenklaim.ClaimByNumber), "CONTOH TERTANGGUNG ENAM")
	require.NoError(t, err)
	require.Len(t, claims, 2)
}

func TestSimpanBerkasBaruMenerbitkanNomor(t *testing.T) {
	service, repo, _ := build(t)

	saved, err := service.Save(context.Background(), portalAlias, caller("NONMBU"), "01",
		validDraftInput())
	require.NoError(t, err)
	require.True(t, saved.Created)
	require.Equal(t, int64(6), saved.ID, "nomor tertinggi contoh 5, berikutnya 6")

	file, exists, err := repo.FindByID(context.Background(), saved.ID)
	require.NoError(t, err)
	require.True(t, exists)
	require.Equal(t, "USERARSIP1", file.InputUser)
	require.Equal(t, archivedokumenklaim.BranchStatusPending, file.BranchStatus,
		"berkas baru WAJIB masuk daftar kirim ke cabang")
}

// Menyimpan berkas LANGSUNG mengirimkannya ke layanan Arsip.
//
// Perilaku sistem lama: `Activity/SaveAttachArchiveToDatabase-Act.xml` langkah 5 memanggil
// `SendDataArchiveDOcumentByService` tepat setelah prosedur penyisipannya selesai.
// Direplikasi atas keputusan Work Owner 2026-09-25.
func TestSimpanLangsungMengirimKeLayananArsip(t *testing.T) {
	service, repo, recorder := build(t)

	saved, err := service.Save(context.Background(), portalAlias, caller("NONMBU"), "01",
		validDraftInput())
	require.NoError(t, err)
	require.True(t, saved.Sent)
	require.Equal(t, "200", saved.ServiceCode)
	require.Empty(t, saved.SendError)

	shipments := recorder.Sent()
	require.Len(t, shipments, 1)
	require.Equal(t, "6/BOX-A-01/FIL-2024-001", shipments[0].DocumentNumber())

	file, _, err := repo.FindByID(context.Background(), saved.ID)
	require.NoError(t, err)
	require.Equal(t, "200", file.ServiceCode, "jawaban layanan ikut tersimpan")
	require.NotNil(t, file.SentDate)
}

// Berkas yang baru disimpan TETAP muncul di daftar kirim ke cabang.
//
// Jalur Simpan menyimpan jawaban layanan tetapi TIDAK menandai CABANGSTATUS — hanya jalur
// Dokument Cabang yang menandainya. Akibatnya berkas yang sama dikirim DUA KALI, dan itu
// perilaku sistem lama yang direplikasi.
//
// Uji ini menahannya supaya tidak "diperbaiki" tanpa keputusan: menandainya di jalur
// Simpan akan menghapus pengiriman kedua.
func TestBerkasBaruTetapMunculDiDaftarKirimCabang(t *testing.T) {
	service, _, _ := build(t)

	saved, err := service.Save(context.Background(), portalAlias, caller("NONMBU"), "01",
		validDraftInput())
	require.NoError(t, err)
	require.True(t, saved.Sent)

	pending, err := service.PendingBranch(context.Background(), portalAlias,
		caller("PICTEKNIK"), archivedokumenklaim.Pagination{})
	require.NoError(t, err)

	var found bool
	for _, file := range pending.Page.Files {
		if file.ID == saved.ID {
			found = true
			require.Equal(t, archivedokumenklaim.BranchStatusPending, file.BranchStatus)
		}
	}
	require.True(t, found,
		"berkas yang sudah dikirim saat disimpan TETAP menunggu di daftar kirim ke cabang")
}

// Pengiriman yang gagal TIDAK menggagalkan penyimpanan.
//
// Barisnya sudah tersimpan — di sistem lama pun prosedurnya COMMIT sebelum pengiriman
// dijalankan. Mengembalikan galat akan membuat layar melaporkan "gagal menyimpan" atas
// berkas yang sebenarnya ADA, dan pengguna akan menyimpannya lagi.
func TestPengirimanGagalSaatSimpanTidakMenggagalkanPenyimpanan(t *testing.T) {
	service, repo, recorder := build(t)
	recorder.Err = archivedokumenklaim.ErrServiceFailed

	saved, err := service.Save(context.Background(), portalAlias, caller("NONMBU"), "01",
		validDraftInput())
	require.NoError(t, err, "penyimpanan TIDAK boleh gagal karena layanan Arsip bermasalah")
	require.False(t, saved.Sent)
	require.NotEmpty(t, saved.SendError)

	file, exists, err := repo.FindByID(context.Background(), saved.ID)
	require.NoError(t, err)
	require.True(t, exists, "barisnya tetap tersimpan")
	require.Empty(t, file.ServiceCode)
	require.Nil(t, file.SentDate)
}

func TestSimpanBerkasYangTidakAdaDitolak(t *testing.T) {
	service, _, _ := build(t)

	input := validDraftInput()
	input.ID = 9999

	_, err := service.Save(context.Background(), portalAlias, caller("NONMBU"), "01", input)
	require.ErrorIs(t, err, archivedokumenklaim.ErrNotFound)
}

// Pengubahan TIDAK menimpa USERINPUT.
//
// Kolom itu menyatakan siapa yang MENGARSIPKAN, bukan siapa yang terakhir menyunting —
// persis cabang `update` prosedur lama, yang memang tidak menyentuhnya.
func TestPengubahanTidakMenimpaUserInput(t *testing.T) {
	service, repo, _ := build(t)

	input := validDraftInput()
	input.ID = 3
	input.SheetCount = 99

	_, err := service.Save(context.Background(), portalAlias,
		archivedokumenklaim.Caller{Login: "PETUGASLAIN"}, "02", input)
	require.NoError(t, err)

	file, _, err := repo.FindByID(context.Background(), 3)
	require.NoError(t, err)
	require.Equal(t, 99, file.SheetCount, "isian yang diubah ikut tersimpan")
	require.Equal(t, "USERARSIP2", file.InputUser, "pengarsipnya TIDAK berubah")
}

// Daftar kirim ke cabang menyaring menurut jabatan DI SERVER.
func TestDaftarKirimCabangDisaringJabatan(t *testing.T) {
	service, _, _ := build(t)

	nonMBU, err := service.PendingBranch(context.Background(), portalAlias,
		caller(archivedokumenklaim.PositionNonMBU), archivedokumenklaim.Pagination{})
	require.NoError(t, err)

	for _, file := range nonMBU.Page.Files {
		require.NotEqual(t, archivedokumenklaim.GroupPanelPersonalAccident, file.GroupPanel)
		require.NotEqual(t, archivedokumenklaim.GroupPanelTravel, file.GroupPanel)
	}

	semua, err := service.PendingBranch(context.Background(), portalAlias,
		caller("PICTEKNIK"), archivedokumenklaim.Pagination{})
	require.NoError(t, err)
	require.Greater(t, semua.Page.Total, nonMBU.Page.Total,
		"jabatan tanpa saringan melihat lebih banyak")
}

// Berkas yang sudah dikirim TIDAK muncul lagi di daftar.
func TestDaftarKirimCabangHanyaYangBelumTerkirim(t *testing.T) {
	service, _, _ := build(t)

	pending, err := service.PendingBranch(context.Background(), portalAlias,
		caller("PICTEKNIK"), archivedokumenklaim.Pagination{})
	require.NoError(t, err)

	for _, file := range pending.Page.Files {
		require.NotEqual(t, archivedokumenklaim.BranchStatusSent, file.BranchStatus)
	}
}

func TestKirimKeCabangMenyimpanJawabanLayanan(t *testing.T) {
	service, repo, recorder := build(t)

	sent, err := service.SendToBranch(context.Background(), portalAlias, caller("PICTEKNIK"), 1)
	require.NoError(t, err)
	require.Equal(t, "200", sent.Code)

	shipments := recorder.Sent()
	require.Len(t, shipments, 1)
	require.Equal(t, "1/BOX-A-01/FIL-2024-001", shipments[0].DocumentNumber())

	file, _, err := repo.FindByID(context.Background(), 1)
	require.NoError(t, err)
	require.Equal(t, archivedokumenklaim.BranchStatusSent, file.BranchStatus)
	require.NotNil(t, file.SentDate, "Tanggal Kirim Dok ikut terisi")
	require.Equal(t, "200", file.ServiceCode)
}

// Mengirim dua kali DITOLAK di server.
//
// Sistem lama tidak memeriksanya sama sekali: menekan tombolnya dua kali mengirim berkas
// yang sama dua kali ke sistem Arsip. Menonaktifkan tombol di layar hanyalah kenyamanan
// tampilan (`D-59`), sehingga pemeriksaannya harus ada di sini.
func TestKirimKeCabangDuaKaliDitolak(t *testing.T) {
	service, _, recorder := build(t)

	_, err := service.SendToBranch(context.Background(), portalAlias, caller("PICTEKNIK"), 1)
	require.NoError(t, err)

	_, err = service.SendToBranch(context.Background(), portalAlias, caller("PICTEKNIK"), 1)
	require.ErrorIs(t, err, archivedokumenklaim.ErrAlreadySent)

	require.Len(t, recorder.Sent(), 1, "layanan Arsip hanya ditembak sekali")
}

func TestKirimBerkasYangTidakAdaDitolak(t *testing.T) {
	service, _, recorder := build(t)

	_, err := service.SendToBranch(context.Background(), portalAlias, caller("PICTEKNIK"), 9999)
	require.ErrorIs(t, err, archivedokumenklaim.ErrNotFound)
	require.Empty(t, recorder.Sent(), "layanan Arsip tidak ditembak untuk berkas yang tidak ada")
}

// Pengiriman yang GAGAL tidak menandai berkasnya terkirim.
//
// Tanpa ini, berkas yang tidak pernah sampai ke sistem Arsip akan hilang dari daftar dan
// tidak akan pernah dikirim ulang — kegagalan yang tidak meninggalkan satu pun jejak di
// layar.
func TestPengirimanGagalTidakMenandaiBerkas(t *testing.T) {
	service, repo, recorder := build(t)
	recorder.Err = archivedokumenklaim.ErrServiceFailed

	_, err := service.SendToBranch(context.Background(), portalAlias, caller("PICTEKNIK"), 1)
	require.ErrorIs(t, err, archivedokumenklaim.ErrServiceFailed)

	file, _, err := repo.FindByID(context.Background(), 1)
	require.NoError(t, err)
	require.Equal(t, archivedokumenklaim.BranchStatusPending, file.BranchStatus)
	require.Nil(t, file.SentDate)
}

// Kode filling disusun dari pemakaian yang sudah ada — rekonstruksi, bukan master.
func TestKodeFillingDisusunDariPemakaian(t *testing.T) {
	service, _, _ := build(t)

	codes, err := service.FillingCodes(context.Background(), portalAlias, "")
	require.NoError(t, err)
	require.NotEmpty(t, codes)

	var found bool
	for _, code := range codes {
		if code.Code == "FIL-2024-001" {
			require.Equal(t, "BOX-A-01", code.BoxName)
			require.Equal(t, 2, code.UsageCount)
			found = true
		}
	}
	require.True(t, found, "kode yang dipakai dua berkas muncul dengan jumlahnya")
}

func TestKodeFillingDapatDisaring(t *testing.T) {
	service, _, _ := build(t)

	codes, err := service.FillingCodes(context.Background(), portalAlias, "2026")
	require.NoError(t, err)
	require.Len(t, codes, 1)
	require.Equal(t, "FIL-2026-003", codes[0].Code)
}

func validDraftInput() archivedokumenklaim.DraftInput {
	received := time.Date(2024, 5, 20, 0, 0, 0, 0, time.UTC)
	loss := time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC)

	return archivedokumenklaim.DraftInput{
		ClaimNumber:          "PNC-100009",
		PolicyNumber:         "POL-2024-000009",
		InsuredName:          "CONTOH TERTANGGUNG SEMBILAN",
		LossDate:             &loss,
		TechnicalPIC:         "PICTEKNIK1",
		GroupPanel:           "006",
		DocumentReceivedDate: &received,
		SheetCount:           12,
		DocumentTypeCode:     "0001",
		DocumentKindCode:     "000101",
		BoxName:              "BOX-A-01",
		FillingCode:          "FIL-2024-001",
	}
}

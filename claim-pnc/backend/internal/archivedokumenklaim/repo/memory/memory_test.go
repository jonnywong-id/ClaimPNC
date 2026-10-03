package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/archivedokumenklaim"
	"claim-pnc/internal/archivedokumenklaim/repo/memory"
)

var ctx = context.Background()

func date(year int, month time.Month, day int) *time.Time {
	moment := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	return &moment
}

func TestDataContohTerisiNamaDokumen(t *testing.T) {
	repo := memory.NewSampleRepo()

	file, exists, err := repo.FindByID(ctx, 1)
	require.NoError(t, err)
	require.True(t, exists)
	require.Equal(t, "Dokumen Klaim", file.DocumentTypeName)
	require.Equal(t, "Laporan Kerugian", file.DocumentKindName)

	_, exists, err = repo.FindByID(ctx, 999)
	require.NoError(t, err)
	require.False(t, exists)

	require.Len(t, memory.SampleClaims(), 3)
	require.Len(t, memory.SampleDocumentTypes(), 3)
	require.Len(t, memory.SampleDocumentKinds(), 5)
}

// Pencarian kata kunci mencocokkan salah satu dari tiga kolom tanpa peka huruf.
func TestSearchKataKunci(t *testing.T) {
	repo := memory.NewSampleRepo()

	byInsured, err := repo.Search(ctx, archivedokumenklaim.Criteria{
		Mode: archivedokumenklaim.ModeKeyword, Keyword: "contoh tertanggung tiga",
	}, archivedokumenklaim.Pagination{})
	require.NoError(t, err)
	require.Equal(t, 1, byInsured.Total)
	require.Equal(t, int64(3), byInsured.Files[0].ID)

	byBox, err := repo.Search(ctx, archivedokumenklaim.Criteria{
		Mode: archivedokumenklaim.ModeKeyword, Keyword: "BOX-A-01",
	}, archivedokumenklaim.Pagination{})
	require.NoError(t, err)
	require.Equal(t, 2, byBox.Total)
	require.Equal(t, int64(2), byBox.Files[0].ID, "urutan menurun menurut ID")
}

// Rentang tanggal input dibandingkan per tanggal kalender, inklusif di kedua ujung.
func TestSearchRentangTanggalInput(t *testing.T) {
	afternoon := time.Date(2024, 4, 10, 15, 30, 0, 0, time.UTC)
	repo := memory.NewRepo(memory.Options{Files: []archivedokumenklaim.ArchiveFile{
		{ID: 1, InputDate: date(2024, 3, 20)},
		{ID: 2, InputDate: &afternoon},
		{ID: 3, InputDate: date(2024, 5, 15)},
		{ID: 4},
	}})

	page, err := repo.Search(ctx, archivedokumenklaim.Criteria{
		Mode: archivedokumenklaim.ModeInputDate,
		From: date(2024, 4, 1),
		To:   date(2024, 4, 10),
	}, archivedokumenklaim.Pagination{})
	require.NoError(t, err)
	require.Equal(t, 1, page.Total, "berkas sore hari pada tanggal akhir tetap ikut")
	require.Equal(t, int64(2), page.Files[0].ID)

	before, err := repo.Search(ctx, archivedokumenklaim.Criteria{
		Mode: archivedokumenklaim.ModeInputDate,
		From: date(2024, 5, 1),
	}, archivedokumenklaim.Pagination{})
	require.NoError(t, err)
	require.Equal(t, 1, before.Total)
	require.Equal(t, int64(3), before.Files[0].ID)
}

func TestPaginasiMemotongHalaman(t *testing.T) {
	repo := memory.NewSampleRepo()

	criteria := archivedokumenklaim.Criteria{Mode: archivedokumenklaim.ModeInputDate}

	second, err := repo.Search(ctx, criteria, archivedokumenklaim.Pagination{Page: 2, Size: 2})
	require.NoError(t, err)
	require.Equal(t, 5, second.Total)
	require.Len(t, second.Files, 2)
	require.Equal(t, int64(3), second.Files[0].ID)

	last, err := repo.Search(ctx, criteria, archivedokumenklaim.Pagination{Page: 3, Size: 2})
	require.NoError(t, err)
	require.Len(t, last.Files, 1)

	beyond, err := repo.Search(ctx, criteria, archivedokumenklaim.Pagination{Page: 9, Size: 2})
	require.NoError(t, err)
	require.Equal(t, 5, beyond.Total)
	require.Empty(t, beyond.Files)
	require.NotNil(t, beyond.Files)
}

// Lini bisnis kosong lolos saringan; yang sudah terkirim tidak tampak.
func TestPendingBranchMenyaringLiniDanStatus(t *testing.T) {
	repo := memory.NewRepo(memory.Options{Files: []archivedokumenklaim.ArchiveFile{
		{ID: 1, GroupPanel: "002", BranchStatus: "0"},
		{ID: 2, GroupPanel: "", BranchStatus: "0"},
		{ID: 3, GroupPanel: "006", BranchStatus: "1"},
		{ID: 4, GroupPanel: "006", BranchStatus: "0"},
	}})

	page, err := repo.PendingBranch(ctx,
		archivedokumenklaim.BranchScopeFor(archivedokumenklaim.PositionPA),
		archivedokumenklaim.Pagination{})
	require.NoError(t, err)

	ids := []int64{}
	for _, file := range page.Files {
		ids = append(ids, file.ID)
	}
	require.Equal(t, []int64{4, 2}, ids)
}

func TestSearchClaims(t *testing.T) {
	repo := memory.NewSampleRepo()

	byPolicy, err := repo.SearchClaims(ctx, archivedokumenklaim.ClaimCriteria{
		Type: archivedokumenklaim.ClaimByPolicy, Value: "pol-2024-000006",
	})
	require.NoError(t, err)
	require.Len(t, byPolicy, 2)

	byName, err := repo.SearchClaims(ctx, archivedokumenklaim.ClaimCriteria{
		Type: archivedokumenklaim.ClaimByNumber, Value: "PNCN.26.0002",
	})
	require.NoError(t, err)
	require.Len(t, byName, 1)

	none, err := repo.SearchClaims(ctx, archivedokumenklaim.ClaimCriteria{
		Type: archivedokumenklaim.ClaimByPolicy, Value: "PNCN.26.0002",
	})
	require.NoError(t, err)
	require.Empty(t, none, "tipe polis hanya mencocokkan nomor polis")
}

// Repo kosong menerbitkan nomor 1 untuk berkas pertama.
func TestSaveMenerbitkanNomorDanMengisiNama(t *testing.T) {
	repo := memory.NewRepo(memory.Options{
		Types: memory.SampleDocumentTypes(),
		Kinds: memory.SampleDocumentKinds(),
	})

	id, err := repo.Save(ctx, archivedokumenklaim.Draft{
		ClaimNumber:      "PNC-1",
		DocumentTypeCode: "0002",
		DocumentKindCode: "000201",
		BoxName:          "BOX",
		FillingCode:      "FIL",
		InputUser:        "USER1",
		SheetCount:       2,
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), id)

	second, err := repo.Save(ctx, archivedokumenklaim.Draft{ClaimNumber: "PNC-2"})
	require.NoError(t, err)
	require.Equal(t, int64(2), second)

	file, _, err := repo.FindByID(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, "Dokumen Survey", file.DocumentTypeName)
	require.Equal(t, "Berita Acara Survey", file.DocumentKindName)
	require.Equal(t, "USER1", file.InputUser)
	require.Equal(t, archivedokumenklaim.BranchStatusPending, file.BranchStatus)

	// Pengubahan tidak menyentuh USERINPUT maupun CABANGSTATUS.
	_, err = repo.Save(ctx, archivedokumenklaim.Draft{
		ID: 1, ClaimNumber: "PNC-1B", InputUser: "LAIN", DocumentTypeCode: "0009",
	})
	require.NoError(t, err)
	file, _, err = repo.FindByID(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, "PNC-1B", file.ClaimNumber)
	require.Equal(t, "USER1", file.InputUser)
	require.Empty(t, file.DocumentTypeName, "kode yang tidak ada di master mengosongkan nama")

	_, err = repo.Save(ctx, archivedokumenklaim.Draft{ID: 77, ClaimNumber: "X"})
	require.ErrorIs(t, err, archivedokumenklaim.ErrNotFound)
}

func TestJawabanLayananStoreDanMark(t *testing.T) {
	repo := memory.NewSampleRepo()
	sentAt := time.Date(2026, 9, 24, 3, 0, 0, 0, time.UTC)

	require.NoError(t, repo.StoreReceipt(ctx, archivedokumenklaim.Receipt{
		ID: 1, Code: "200", Note: "OK", SentAt: sentAt,
	}))
	file, _, err := repo.FindByID(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, "200", file.ServiceCode)
	require.Equal(t, sentAt, *file.SentDate)
	require.Equal(t, archivedokumenklaim.BranchStatusPending, file.BranchStatus)

	require.NoError(t, repo.MarkSent(ctx, archivedokumenklaim.Receipt{ID: 1, SentAt: sentAt}))
	file, _, err = repo.FindByID(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, archivedokumenklaim.BranchStatusSent, file.BranchStatus)

	require.ErrorIs(t, repo.MarkSent(ctx, archivedokumenklaim.Receipt{ID: 99}),
		archivedokumenklaim.ErrNotFound)
}

func TestPilihanDokumenDisalin(t *testing.T) {
	repo := memory.NewSampleRepo()

	types, err := repo.DocumentTypes(ctx)
	require.NoError(t, err)
	require.Equal(t, memory.SampleDocumentTypes(), types)

	kinds, err := repo.DocumentKinds(ctx)
	require.NoError(t, err)
	require.Equal(t, memory.SampleDocumentKinds(), kinds)

	types[0].Name = "DIUBAH"
	again, err := repo.DocumentTypes(ctx)
	require.NoError(t, err)
	require.Equal(t, "Dokumen Klaim", again[0].Name, "mengubah hasil tidak mengubah isi repo")
}

// Kode filling dikumpulkan per pasangan kode dan boks, diurutkan, dan kode kosong dilewati.
func TestFillingCodesMengelompokkanDanMengurutkan(t *testing.T) {
	repo := memory.NewRepo(memory.Options{Files: []archivedokumenklaim.ArchiveFile{
		{ID: 1, FillingCode: "FIL-B", BoxName: "BOX-2"},
		{ID: 2, FillingCode: "FIL-A", BoxName: "BOX-9"},
		{ID: 3, FillingCode: "FIL-A", BoxName: "BOX-1"},
		{ID: 4, FillingCode: "FIL-A", BoxName: "BOX-1"},
		{ID: 5, FillingCode: "  ", BoxName: "BOX-X"},
	}})

	all, err := repo.FillingCodes(ctx, "")
	require.NoError(t, err)
	require.Equal(t, []archivedokumenklaim.FillingCodeOption{
		{Code: "FIL-A", BoxName: "BOX-1", UsageCount: 2},
		{Code: "FIL-A", BoxName: "BOX-9", UsageCount: 1},
		{Code: "FIL-B", BoxName: "BOX-2", UsageCount: 1},
	}, all)

	byBox, err := repo.FillingCodes(ctx, "box-2")
	require.NoError(t, err)
	require.Len(t, byBox, 1)
	require.Equal(t, "FIL-B", byBox[0].Code)
}

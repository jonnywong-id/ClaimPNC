package masterdokumentravel_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterdokumentravel"
	"claim-pnc/internal/masterdokumentravel/repo/memory"
)

// Nama uji di berkas ini sengaja menyatakan ATURANNYA, bukan nama fungsinya, sehingga
// daftar uji terbaca sebagai dokumentasi aturan yang selalu mutakhir
// (`14-TESTING-STRATEGY.md` §3.2).

func TestDOCIDIsSiteCodeFollowedByFiveDigitSequence(t *testing.T) {
	// Bentuk yang direplikasi dari `Database/DOCTRAVEL_CVG.prc:20`:
	//   id_site || lpad(to_Char(DOCTRAVEL_SEQ.nextval),5,'0')
	require.Equal(t, "100001", masterdokumentravel.FormatID("1", 1))
	require.Equal(t, "100042", masterdokumentravel.FormatID("1", 42))
	require.Equal(t, "199999", masterdokumentravel.FormatID("1", 99999))
}

func TestSequenceBeyondFiveDigitsIsNotTruncated(t *testing.T) {
	// LPAD Oracle memanjangkan, tidak memotong. Memotongnya di sini akan menghasilkan
	// DOCID GANDA — dua jenis dokumen berbagi satu kunci yang dirujuk tabel lain — dan
	// itu jauh lebih buruk daripada penyisipan yang gagal dengan pesan jelas.
	require.Equal(t, "1100000", masterdokumentravel.FormatID("1", 100000))
}

func TestSiteCodeSpacingIsNotCarriedIntoTheID(t *testing.T) {
	// Kolom ID pada M_SITE_DATABASE dapat bertipe CHAR dan memadatkan nilainya dengan
	// spasi. Membiarkannya akan menghasilkan DOCID berisi spasi di tengah.
	require.Equal(t, "100001", masterdokumentravel.FormatID(" 1 ", 1))
}

func TestEdgeSpacesAreTrimmedFromTheTitle(t *testing.T) {
	// Pemangkasan ini bukan validasi: ia menjaga apa yang disimpan sama dengan apa yang
	// dibaca kembali, karena pembacaan memang memangkas padding kolom CHAR.
	require.Equal(t, "Paspor", masterdokumentravel.Input{Name: "  Paspor  "}.Clean().Name)
}

// TestEmptyTitleIsAcceptedLikeInPega mengunci keputusan Work Owner 2026-09-21.
//
// Layar Pega tidak memvalidasi apa pun — `Section/BrowseMasterDocumentTravel-Section.xml`
// tidak memuat satu pun `pyRequired=true`, dan `DOCTRAVEL_CVG.prc` menyisipkan tanpa
// memeriksa. Uji ini ada supaya penambahan validasi di kemudian hari adalah keputusan
// yang disadari, bukan kelalaian yang menyelinap.
func TestEmptyTitleIsAcceptedLikeInPega(t *testing.T) {
	repo := memory.NewRepo()

	saved, err := repo.InsertNew(context.Background(), masterdokumentravel.Input{Name: ""}.Clean())
	require.NoError(t, err)
	require.Empty(t, saved.Name)
	require.NotEmpty(t, saved.ID, "DOCID tetap diterbitkan walau judulnya kosong")
}

// TestDuplicateTitleIsAcceptedLikeInPega mengunci separuh lainnya dari keputusan itu.
//
// Berbeda dari Master Status Klaim, yang Work Owner putuskan untuk diperketat pada
// 2026-09-17, master ini tidak menolak judul ganda.
func TestDuplicateTitleIsAcceptedLikeInPega(t *testing.T) {
	repo := memory.NewRepo()
	ctx := context.Background()

	first, err := repo.InsertNew(ctx, masterdokumentravel.Input{Name: "Paspor"})
	require.NoError(t, err)
	second, err := repo.InsertNew(ctx, masterdokumentravel.Input{Name: "Paspor"})
	require.NoError(t, err)

	require.NotEqual(t, first.ID, second.ID, "keduanya baris berbeda dengan DOCID berbeda")
}

func TestNewDocumentContinuesTheHighestSequenceAlreadyUsed(t *testing.T) {
	// Tanpa ini, penambahan pertama pada repo yang sudah berisi akan menerbitkan DOCID
	// yang bentrok dengan baris yang ada.
	repo := memory.NewRepo(memory.SampleList()...)

	saved, err := repo.InsertNew(context.Background(), masterdokumentravel.Input{Name: "Visa"})
	require.NoError(t, err)
	require.Equal(t, "100007", saved.ID)
}

func TestListIsSortedByDOCIDLikeTheOldReportDefinition(t *testing.T) {
	// `BrowseMstDocTravel_RD-RD.xml` mengurutkan DOCID menaik.
	repo := memory.NewRepo(
		masterdokumentravel.TravelDocument{ID: "100003", Name: "Ketiga"},
		masterdokumentravel.TravelDocument{ID: "100001", Name: "Pertama"},
		masterdokumentravel.TravelDocument{ID: "100002", Name: "Kedua"},
	)

	list, err := repo.List(context.Background())
	require.NoError(t, err)
	require.Equal(t, []string{"100001", "100002", "100003"}, idsOf(list))
}

func TestUnknownDOCIDIsReportedAsNotFound(t *testing.T) {
	repo := memory.NewRepo(memory.SampleList()...)
	ctx := context.Background()

	_, err := repo.Get(ctx, "999999")
	require.ErrorIs(t, err, masterdokumentravel.ErrNotFound)

	err = repo.Update(ctx, masterdokumentravel.TravelDocument{ID: "999999", Name: "Apa saja"})
	require.ErrorIs(t, err, masterdokumentravel.ErrNotFound)
}

func TestUpdateChangesTheTitleAndNeverTheDOCID(t *testing.T) {
	// DOCID dirujuk V_LST_DOC_TRAVEL dan dokumen klaim yang sudah terunggah; mengubahnya
	// memutus keduanya.
	repo := memory.NewRepo(masterdokumentravel.TravelDocument{ID: "100001", Name: "Paspor"})
	ctx := context.Background()

	require.NoError(t, repo.Update(ctx, masterdokumentravel.TravelDocument{ID: "100001", Name: "Paspor Baru"}))

	after, err := repo.Get(ctx, "100001")
	require.NoError(t, err)
	require.Equal(t, "100001", after.ID)
	require.Equal(t, "Paspor Baru", after.Name)
}

// TestSeamProvidesNoDeleteOperation menjaga `ADR-0012` tetap terpasang di tingkat tipe.
//
// Layar Pega tidak punya tombol hapus dan procedure-nya hanya mengenal INSERT dan
// UPDATE. Menghapus satu baris master akan membuat setiap dokumen klaim yang menyimpan
// DOCID itu kehilangan artinya.
func TestSeamProvidesNoDeleteOperation(t *testing.T) {
	var seam any = memory.NewRepo()

	_, hasDelete := seam.(interface {
		Delete(ctx context.Context, id string) error
	})
	require.False(t, hasDelete, "Repo tidak boleh menyediakan operasi hapus")
}

func idsOf(list []masterdokumentravel.TravelDocument) []string {
	result := make([]string, 0, len(list))
	for _, doc := range list {
		result = append(result, doc.ID)
	}
	return result
}

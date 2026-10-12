package memory_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxosclaimpercabang"
	"claim-pnc/internal/inboxosclaimpercabang/repo/memory"
)

func TestFindDetailReturnsTheFourSections(t *testing.T) {
	store := memory.NewSampleStore()

	detail, found, err := store.FindDetail(t.Context(), cilegon(), "PNC-9001")
	require.NoError(t, err)
	require.True(t, found)

	require.Equal(t, "PNC-9001", detail.ClaimNumber)
	require.Equal(t, "CLAIM-0001", detail.ClaimKey)
	require.Len(t, detail.Objects, 2)
	require.Len(t, detail.ProgressHistory, 2)
	require.Len(t, detail.AdjusterMessages, 2)
}

func TestFindDetailJoinsDominantFactorsInOrder(t *testing.T) {
	// Faktor dominan pada contoh melekat pada CLAIM-0002, yakni PNC-9002. Urutannya mengikuti
	// `idx_dominanfactor` apa adanya — tidak diurutkan ulang dengan aturan yang dapat
	// berselisih dengan kuerinya.
	store := memory.NewSampleStore()

	detail, found, err := store.FindDetail(t.Context(), cilegon(), "PNC-9002")
	require.NoError(t, err)
	require.True(t, found)

	require.Equal(t, "Kelalaian Tertanggung, Dokumen Tidak Lengkap", detail.DominantFactors,
		"faktor dominan harus dirangkai dengan pemisah yang sama dengan kueri lama")
}

func TestFindDetailMarksIncomingAndOutgoingMessagesDifferently(t *testing.T) {
	// Bila penandanya selalu bernilai sama, layar menggambar percakapan seolah satu pihak
	// saja — dan tidak ada uji yang akan menangkapnya.
	store := memory.NewSampleStore()

	detail, _, err := store.FindDetail(t.Context(), cilegon(), "PNC-9001")
	require.NoError(t, err)

	require.True(t, detail.AdjusterMessages[0].Internal)
	require.False(t, detail.AdjusterMessages[1].Internal)
}

func TestFindDetailUsesDashWhenThereAreNoDominantFactors(t *testing.T) {
	// `"-"` itu yang selama ini terbaca di layar, dan lembar kerja yang sudah menyaring nilai
	// itu akan berhenti bekerja bila ia berubah menjadi kosong.
	store := memory.NewSampleStore()

	detail, found, err := store.FindDetail(t.Context(), cilegon(), "PNC-9001")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "-", detail.DominantFactors)
}

func TestFindDetailRefusesClaimsOfAnotherBranch(t *testing.T) {
	// Inilah kebocoran yang tidak menghasilkan satu pun galat: popup terisi, angkanya masuk
	// akal, dan yang salah hanya MILIK SIAPA datanya (`R-20`).
	//
	// PNC-9004 ada di penyimpanan, tetapi milik BANDUNG.
	store := memory.NewSampleStore()

	_, found, err := store.FindDetail(t.Context(), cilegon(), "PNC-9004")
	require.NoError(t, err)
	require.False(t, found, "klaim cabang lain ikut terbuka lewat popup")
}

func TestFindDetailRefusesClaimsThatTheListFiltersOut(t *testing.T) {
	// Popup tidak boleh lebih longgar daripada daftarnya. PNC-9005 sudah selesai dan PNC-9006
	// tidak punya tanggal registrasi; keduanya tidak tampil di grid, sehingga isinya pun tidak
	// boleh terbaca lewat nomor.
	store := memory.NewSampleStore()

	for _, number := range []string{"PNC-9005", "PNC-9006"} {
		_, found, err := store.FindDetail(t.Context(), cilegon(), number)
		require.NoErrorf(t, err, "klaim %s", number)
		require.Falsef(t, found, "klaim %s tidak tampil di daftar tetapi terbuka di popup",
			number)
	}
}

func TestFindDetailWithoutClaimNumberIsNotAnError(t *testing.T) {
	store := memory.NewSampleStore()

	_, found, err := store.FindDetail(t.Context(), cilegon(), "   ")
	require.NoError(t, err, "nomor kosong bukan galat")
	require.False(t, found)
}

func TestFindDetailCopiesItsSlices(t *testing.T) {
	// Pemanggil yang mengubah hasilnya tidak boleh mengubah isi penyimpanan. Pada pengisi
	// memori, irisan yang dibagikan apa adanya membuat satu permintaan merusak permintaan
	// berikutnya — cacat yang tidak pernah muncul pada pengisi SQL.
	store := memory.NewSampleStore()

	first, _, err := store.FindDetail(t.Context(), cilegon(), "PNC-9001")
	require.NoError(t, err)
	first.Objects[0].Name = "DIUBAH PEMANGGIL"

	second, _, err := store.FindDetail(t.Context(), cilegon(), "PNC-9001")
	require.NoError(t, err)
	require.Equal(t, "GUDANG A", second.Objects[0].Name)
}

func TestDetailPlannedDifferencesAreStated(t *testing.T) {
	require.NotEmpty(t, inboxosclaimpercabang.DetailPlannedDifferences,
		"selisih popup harus sampai ke pengguna, bukan berhenti di komentar kode")
}

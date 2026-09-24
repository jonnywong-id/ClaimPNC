package memory_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxkomunikasicabang"
	"claim-pnc/internal/inboxkomunikasicabang/repo/memory"
)

// headOffice adalah batas data petugas kantor pusat yang cabangnya TERBACA.
func headOffice() inboxkomunikasicabang.BranchFilter {
	return inboxkomunikasicabang.ResolveBranch(
		inboxkomunikasicabang.HeadOfficeBranch, true)
}

// branch1001 adalah batas data petugas cabang 1001.
func branch1001() inboxkomunikasicabang.BranchFilter {
	return inboxkomunikasicabang.ResolveBranch("1001", true)
}

// listOf menjalankan satu tab dan mengembalikan barisnya.
func listOf(
	t *testing.T, tabCode string, filter inboxkomunikasicabang.BranchFilter,
) []inboxkomunikasicabang.Conversation {
	t.Helper()

	store := memory.NewSampleStore()
	tab, found := inboxkomunikasicabang.FindTab(tabCode)
	require.True(t, found)

	page, err := store.List(
		context.Background(),
		inboxkomunikasicabang.Query{
			Tab:    tab,
			Branch: filter,
			Caller: inboxkomunikasicabang.Caller{Login: "pictekniks"},
		},
		inboxkomunikasicabang.Pagination{Page: 1, Size: 100},
	)
	require.NoError(t, err)
	return page.Items
}

// idsOf mengumpulkan nomor percakapan dari sederet baris.
func idsOf(items []inboxkomunikasicabang.Conversation) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return ids
}

func TestAClosedConversationDisappearsFromBothTabs(t *testing.T) {
	// Tombol "Selesai Komunikasi" mengubah `CASEID` menjadi `CABANG SELESAI`, dan kedua grid
	// menyaring `CASEID = 'CABANG'`. Itulah yang membuat layar ini benar-benar Inbox menurut
	// `D-79`: barisnya HILANG begitu pekerjaannya selesai.
	//
	// KOM-0007 ber-`CABANG SELESAI`, dan nilainya BERAWALAN `CABANG` — sehingga penyaring
	// yang ditulis sebagai awalan akan meloloskannya.
	for _, tabCode := range []string{
		inboxkomunikasicabang.TabNotAnswered,
		inboxkomunikasicabang.TabAnswered,
	} {
		require.NotContains(t, idsOf(listOf(t, tabCode, headOffice())), "KOM-0007")
	}
}

func TestAConversationWithoutSenderOrMessageIsHidden(t *testing.T) {
	// `SENDER IS NOT NULL AND MESSAGE IS NOT NULL` pada kedua kueri grid.
	ids := idsOf(listOf(t, inboxkomunikasicabang.TabNotAnswered, headOffice()))

	require.NotContains(t, ids, "KOM-0008", "baris tanpa pengirim tidak boleh tampil")
	require.NotContains(t, ids, "KOM-0009", "baris tanpa pesan tidak boleh tampil")
}

func TestAnotherBranchConversationIsNeverVisible(t *testing.T) {
	// KOM-0010 berasal dari cabang 1003 menuju 1004. Tidak satu pun petugas contoh boleh
	// melihatnya — termasuk kantor pusat, karena asal maupun tujuannya bukan `1`.
	for _, filter := range []inboxkomunikasicabang.BranchFilter{
		headOffice(), branch1001(),
	} {
		ids := idsOf(listOf(t, inboxkomunikasicabang.TabNotAnswered, filter))
		require.NotContains(t, ids, "KOM-0010")
	}
}

func TestAConversationIsVisibleToBothEndsOfIt(t *testing.T) {
	// Penyaringnya `OR`, bukan `AND`: percakapan terlihat oleh cabang yang MENGIRIM maupun
	// yang MENERIMA. Menukarnya dengan `AND` mengosongkan seluruh layar.
	//
	// KOM-0005 berasal dari cabang 1001 menuju pusat, sehingga KEDUANYA harus melihatnya.
	require.Contains(t,
		idsOf(listOf(t, inboxkomunikasicabang.TabNotAnswered, branch1001())), "KOM-0005")
	require.Contains(t,
		idsOf(listOf(t, inboxkomunikasicabang.TabNotAnswered, headOffice())), "KOM-0005")
}

func TestAnUnresolvedBranchSeesExactlyWhatHeadOfficeSees(t *testing.T) {
	// Keputusan Work Owner 2026-09-24 (`P-5`): keduanya dipetakan ke penyaring yang SAMA.
	// Yang membedakannya hanya penanda `Resolved`, yang dipakai menjelaskan keadaannya —
	// bukan mengubah apa yang terlihat.
	unresolved := inboxkomunikasicabang.ResolveBranch("", false)

	require.Equal(t,
		idsOf(listOf(t, inboxkomunikasicabang.TabNotAnswered, headOffice())),
		idsOf(listOf(t, inboxkomunikasicabang.TabNotAnswered, unresolved)),
	)
}

func TestTheTwoTabsPartitionTheSameConversationsWithoutOverlap(t *testing.T) {
	notAnswered := idsOf(listOf(t, inboxkomunikasicabang.TabNotAnswered, headOffice()))
	answered := idsOf(listOf(t, inboxkomunikasicabang.TabAnswered, headOffice()))

	require.NotEmpty(t, notAnswered)
	require.NotEmpty(t, answered)

	for _, id := range notAnswered {
		require.NotContains(t, answered, id,
			"satu percakapan tidak boleh berada di kedua tab sekaligus")
	}
}

func TestNotAnsweredTabIsOrderedOldestFirst(t *testing.T) {
	// `ORDER BY CREATEDDATE ASC` — antrean dibaca dari yang paling lama menunggu.
	// KOM-0003 bertanggal 2026-08-28, paling awal di antara yang belum dijawab.
	ids := idsOf(listOf(t, inboxkomunikasicabang.TabNotAnswered, headOffice()))

	require.NotEmpty(t, ids)
	require.Equal(t, "KOM-0003", ids[0])
}

func TestAnsweredTabIsOrderedNewestReplyFirstWhichIsTheOppositeDirection(t *testing.T) {
	// `ORDER BY CREATEDATEREPLY DESC`. Kedua arah BERLAWANAN, dan itu perilaku sistem lama
	// apa adanya — bukan kekeliruan penyalinan.
	//
	// KOM-0004 dibalas 2026-09-12, paling akhir di antara percakapan cabang 1001.
	ids := idsOf(listOf(t, inboxkomunikasicabang.TabAnswered, branch1001()))

	require.NotEmpty(t, ids)
	require.Equal(t, "KOM-0004", ids[0])
}

func TestARepliedConversationWithoutARecordedReplierAppearsButIsCountedNowhere(t *testing.T) {
	// Selisih SATU KOLOM antara grid dan pencacah, dan ia ada di sistem lama:
	//
	//	grid     REPLYMESSAGE IS NOT NULL
	//	pencacah REPLYFROM IS NOT NULL AND REPLYMESSAGE IS NOT NULL
	//
	// KOM-0006 dibalas tanpa penjawab tercatat, sehingga ia MUNCUL di tab "Sudah Dijawab"
	// tetapi tidak terhitung di pencacah mana pun.
	answered := listOf(t, inboxkomunikasicabang.TabAnswered, branch1001())
	require.Contains(t, idsOf(answered), "KOM-0006")

	store := memory.NewSampleStore()
	summary, err := store.Summarize(context.Background(), branch1001())
	require.NoError(t, err)

	// Selisihnya diuji TERISOLASI pada pencacah "Answered" saja.
	//
	// Membandingkan TOTAL kedua pencacah dengan total kedua tab tidak akan menyatakan apa
	// pun tentang KOM-0006: pencacah juga menghitung baris yang grid-nya sembunyikan
	// (KOM-0008 dan KOM-0009), dan kedua efek itu saling menutupi. Yang satu mengurangi,
	// yang lain menambah — dan totalnya justru lebih besar.
	require.Len(t, answered, 3, "tab menampilkan KOM-0002, KOM-0004, dan KOM-0006")
	require.Equal(t, 2, summary.Answered,
		"pencacah menuntut REPLYFROM pula, sehingga KOM-0006 tidak terhitung")
}

func TestTheSenderNameColumnNeverReachesTheScreen(t *testing.T) {
	// Kueri lama memberi DUA kolom alias `UserName` yang sama; yang menang adalah yang
	// TERAKHIR — `COMMUNICATE_FROM`, bukan `SENDERNAME`.
	//
	// KOM-0001 menyimpan nama pengirim "PIC Teknik Surabaya", dan nama itu tidak boleh
	// muncul di isian mana pun.
	items := listOf(t, inboxkomunikasicabang.TabNotAnswered, headOffice())

	for _, item := range items {
		if item.ID != "KOM-0001" {
			continue
		}
		require.Equal(t, "1001", item.SenderOrigin,
			"asal harus kode cabang, bukan nama pengirim")
		require.Equal(t, "pictekniks", item.SenderOperator)
		require.NotContains(t, item.SenderOrigin, "PIC Teknik")
		return
	}
	t.Fatal("KOM-0001 tidak ditemukan di tab Belum Dijawab")
}

func TestSummaryCountsIgnoreTheSenderAndMessageFiltersThatTheGridApplies(t *testing.T) {
	// Kedua kueri pencacah TIDAK memuat `SENDER IS NOT NULL` maupun `MESSAGE IS NOT NULL`,
	// padahal kedua kueri grid memilikinya. KOM-0008 dan KOM-0009 karena itu IKUT terhitung
	// meski tidak pernah tampil.
	store := memory.NewSampleStore()

	summary, err := store.Summarize(context.Background(), headOffice())
	require.NoError(t, err)

	visible := len(listOf(t, inboxkomunikasicabang.TabNotAnswered, headOffice()))
	require.Greater(t, summary.NotAnswered, visible,
		"pencacah harus menghitung baris yang grid-nya sembunyikan")
}

func TestDetailRefusesAConversationBelongingToAnotherBranch(t *testing.T) {
	// Nomor percakapan berurutan dan mudah ditebak. Tanpa batas cabang di sini, layar detail
	// menjadi pintu samping ke percakapan cabang mana pun.
	store := memory.NewSampleStore()

	_, err := store.Detail(context.Background(), "KOM-0010", branch1001())
	require.ErrorIs(t, err, inboxkomunikasicabang.ErrConversationNotFound)
}

func TestDetailReachesAClosedConversationBecauseItsChannelIsNotFiltered(t *testing.T) {
	// Layar detail TIDAK menyaring kanal: percakapan yang sudah ditutup tetap dapat dibaca
	// utasnya bila nomornya diketahui. Menutupnya adalah aturan yang tidak dapat dibaca dari
	// export mana pun — kueri pemasok layar detail HILANG (`R-16`).
	store := memory.NewSampleStore()

	detail, err := store.Detail(context.Background(), "KOM-0007", headOffice())

	require.NoError(t, err)
	require.Equal(t, "KOM-0007", detail.ID)
}

func TestDetailCarriesAttachmentsAndDistinguishesUploadedFromPending(t *testing.T) {
	store := memory.NewSampleStore()

	detail, err := store.Detail(context.Background(), "KOM-0002", branch1001())
	require.NoError(t, err)
	require.Len(t, detail.Attachments, 2)

	require.True(t, detail.Attachments[0].Uploaded())
	require.False(t, detail.Attachments[1].Uploaded(),
		"lampiran tanpa tanggal unggah harus terbaca belum diunggah")
}

func TestDetailThreadIsAlwaysOldestFirstEvenForTheAnsweredTab(t *testing.T) {
	// Urutan menurun tab "Sudah Dijawab" berlaku untuk DAFTARNYA, bukan untuk isi satu
	// percakapan. Percakapan dibaca dari awal.
	store := memory.NewSampleStore()

	detail, err := store.Detail(context.Background(), "KOM-0004", branch1001())
	require.NoError(t, err)
	require.NotEmpty(t, detail.Messages)

	for i := 1; i < len(detail.Messages); i++ {
		require.LessOrEqual(t,
			detail.Messages[i-1].CreatedAt, detail.Messages[i].CreatedAt)
	}
}

func TestAnEmptyBranchFilterShowsNothingRatherThanEverything(t *testing.T) {
	// Tidak pernah terjadi lewat ResolveBranch, yang selalu mengisi Code. Penjagaan ini ada
	// supaya Repo yang dipanggil dengan BranchFilter kosong — misalnya dari kode yang ditulis
	// kemudian — TIDAK diam-diam menampilkan seluruh percakapan.
	items := listOf(t, inboxkomunikasicabang.TabNotAnswered,
		inboxkomunikasicabang.BranchFilter{})

	require.Empty(t, items)
}

func TestBranchResolverDistinguishesUnknownLoginFromABrokenSource(t *testing.T) {
	// Keduanya berakibat SANGAT berbeda: yang pertama dilayani sebagai kantor pusat (`P-5`),
	// yang kedua menutup layar. Seam yang tidak membedakannya menyembunyikan DB Link yang
	// mati di balik daftar yang tampak wajar.
	resolver := memory.NewSampleBranchResolver()

	code, resolved, err := resolver.Resolve(context.Background(), "tidakterdaftar")
	require.NoError(t, err, "login tak dikenal BUKAN galat")
	require.False(t, resolved)
	require.Empty(t, code)

	code, resolved, err = resolver.Resolve(context.Background(), "pictekniks")
	require.NoError(t, err)
	require.True(t, resolved)
	require.Equal(t, "1001", code)
}

func TestBranchResolverReportsAFailingSourceAsAnError(t *testing.T) {
	resolver := memory.NewSampleBranchResolver()
	resolver.SetError(context.DeadlineExceeded)

	_, resolved, err := resolver.Resolve(context.Background(), "pictekniks")

	require.Error(t, err)
	require.False(t, resolved)
}

func TestSampleBranchMappingReachesEveryVisibilityPath(t *testing.T) {
	// Tanpa kesejajaran ini, tidak satu pun login contoh dapat melihat satu pun baris — dan
	// seluruh uji di berkas ini akan lulus dengan daftar kosong.
	mapping := memory.SampleBranchOfLogin()

	require.Equal(t, inboxkomunikasicabang.HeadOfficeBranch, mapping["adminpnc"],
		"jalur kantor pusat butuh saksi yang cabangnya BENAR-BENAR terbaca")
	require.Equal(t, "1001", mapping["pictekniks"])
}

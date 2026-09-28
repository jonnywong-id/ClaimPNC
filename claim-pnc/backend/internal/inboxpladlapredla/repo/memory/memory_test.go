package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxpladlapredla"
	"claim-pnc/internal/inboxpladlapredla/repo/memory"
)

// Uji di berkas ini menegakkan aturan yang SAMA dengan yang ditegakkan penyimpanan SQL.
// Keduanya tidak boleh memahami penyaring tab secara berbeda — dan karena penyimpanan SQL
// tidak dapat diuji tanpa Oracle, di sinilah aturannya dijaga.

func caller() inboxpladlapredla.Caller {
	return inboxpladlapredla.Caller{Login: "JONNY"}
}

func date(year int, month time.Month, day int) *time.Time {
	moment := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	return &moment
}

// list menjalankan satu tab dengan penyaring tertentu dan mengembalikan nomor klaimnya.
func list(
	t *testing.T,
	store *memory.Store,
	input inboxpladlapredla.QueryInput,
) []string {
	t.Helper()

	query, err := inboxpladlapredla.NewQuery(input, caller())
	require.NoError(t, err)

	page, err := store.List(context.Background(), query,
		inboxpladlapredla.Pagination{Page: 1, Size: 100})
	require.NoError(t, err)

	numbers := []string{}
	for _, row := range page.Items {
		numbers = append(numbers, row.ClaimNo)
	}
	return numbers
}

// rowOf mencari satu baris menurut nomor klaimnya.
func rowOf(
	t *testing.T,
	store *memory.Store,
	tab, claimNo string,
) inboxpladlapredla.Row {
	t.Helper()

	query, err := inboxpladlapredla.NewQuery(
		inboxpladlapredla.QueryInput{Tab: tab}, caller())
	require.NoError(t, err)

	page, err := store.List(context.Background(), query,
		inboxpladlapredla.Pagination{Page: 1, Size: 100})
	require.NoError(t, err)

	for _, row := range page.Items {
		if row.ClaimNo == claimNo {
			return row
		}
	}

	t.Fatalf("baris %s tidak ada di daftar %s", claimNo, tab)
	return inboxpladlapredla.Row{}
}

// Personal Accident dan Travel dikecualikan KETIGA tab.
//
// Kedua klaimnya punya dokumen yang memenuhi syarat, sehingga yang menahannya benar-benar
// penyaring lini bisnis — bukan ketiadaan dokumen.
func TestPersonalAccidentAndTravelNeverAppearOnAnyList(t *testing.T) {
	store := memory.NewSampleStore()

	for _, tab := range []string{"pla", "dla", "pre-dla"} {
		numbers := list(t, store, inboxpladlapredla.QueryInput{Tab: tab})
		require.NotContains(t, numbers, "PNC-1003", "%s: Personal Accident lolos", tab)
		require.NotContains(t, numbers, "PNC-1004", "%s: Travel lolos", tab)
	}
}

// Tab PLA menuntut kode reasuradur sudah terisi.
func TestThePLAListSkipsAdvicesWithoutAReinsurerCode(t *testing.T) {
	store := memory.NewSampleStore()

	numbers := list(t, store, inboxpladlapredla.QueryInput{Tab: "pla"})

	require.Contains(t, numbers, "PNC-1001")
	require.NotContains(t, numbers, "PNC-1002",
		"PLA tanpa kode reasuradur belum menjadi pekerjaan")
}

// Tab DLA melewatkan dokumen yang sudah terkirim.
func TestTheDLAListSkipsAdvicesAlreadySent(t *testing.T) {
	store := memory.NewSampleStore()

	numbers := list(t, store, inboxpladlapredla.QueryInput{Tab: "dla"})

	require.Contains(t, numbers, "PNC-1001")
	require.NotContains(t, numbers, "PNC-1002", "DLA yang sudah terkirim tetap muncul")
}

// Dua pengecualian tambahan tab DLA berlaku DI SANA SAJA.
//
// Klaim ASNET dan klaim berkelompok bisnis 10008 tetap muncul di tab PLA — keduanya punya
// PLA yang memenuhi syarat.
func TestBranchAndBusinessGroupExclusionsApplyToTheDLAListOnly(t *testing.T) {
	store := memory.NewSampleStore()

	pla := list(t, store, inboxpladlapredla.QueryInput{Tab: "pla"})
	require.Contains(t, pla, "PNC-1005", "ASNET tidak dikecualikan di tab PLA")
	require.Contains(t, pla, "PNC-1007", "10008 tidak dikecualikan di tab PLA")

	dla := list(t, store, inboxpladlapredla.QueryInput{Tab: "dla"})
	require.NotContains(t, dla, "PNC-1005", "cabang ASNET lolos ke tab DLA")
	require.NotContains(t, dla, "PNC-1007", "kelompok bisnis 10008 lolos ke tab DLA")
}

// Klaim yang cabangnya KOSONG ikut tersaring keluar dari tab DLA.
//
// Di Oracle, `NULL <> 'ASNET'` menghasilkan UNKNOWN — bukan TRUE. Kueri lama tidak
// menulis `OR BRANCHNAME IS NULL`, dan perilakunya dibawa apa adanya (`P-5`).
//
// Ini uji yang akan gagal bila seseorang "memperbaiki" penyaringnya menjadi lebih masuk
// akal — dan perbaikan itu akan MENAMBAH baris ke antrean tanpa ada yang memintanya.
func TestAClaimWithAnEmptyBranchIsAlsoExcludedFromTheDLAList(t *testing.T) {
	store := memory.NewSampleStore()

	require.Contains(t, list(t, store, inboxpladlapredla.QueryInput{Tab: "pla"}),
		"PNC-1006", "cabang kosong tidak menahan tab PLA")

	require.NotContains(t, list(t, store, inboxpladlapredla.QueryInput{Tab: "dla"}),
		"PNC-1006", "cabang kosong seharusnya tersaring keluar dari tab DLA")
}

// Tab Pre DLA disaring Nomor Akseptasi, bukan penanda terkirim.
func TestThePreDLAListIsFilteredByAcceptanceNumber(t *testing.T) {
	store := memory.NewSampleStore()

	numbers := list(t, store, inboxpladlapredla.QueryInput{Tab: "pre-dla"})

	require.Contains(t, numbers, "PNC-1001", "Pre-DLA tanpa akseptasi harus muncul")
	require.NotContains(t, numbers, "PNC-1009",
		"Pre-DLA yang sudah ber-Nomor Akseptasi harus keluar dari antrean")
}

// Dokumen ber-`ISKIRIM = '0'` tetap membuat klaimnya masuk antrean — tetapi kolom tanggal
// advice-nya KOSONG.
//
// Kejanggalan Pega yang sengaja dibawa (`P-5`): penyaring keanggotaan menerima `'0'`,
// sedangkan sub-kueri tanggalnya hanya menerima `NULL`.
func TestAnAdviceMarkedZeroAppearsButLeavesTheDateColumnEmpty(t *testing.T) {
	store := memory.NewSampleStore()

	numbers := list(t, store, inboxpladlapredla.QueryInput{Tab: "pla"})
	require.Contains(t, numbers, "PNC-1008",
		"ISKIRIM '0' berarti belum terkirim, jadi barisnya harus muncul")

	row := rowOf(t, store, "pla", "PNC-1008")
	require.Empty(t, row.AdviceDate,
		"tanggal advice harus kosong — sub-kueri Pega hanya menerima ISKIRIM NULL")
}

// Tanggal advice adalah yang TERBARU di antara dokumen yang belum terkirim.
func TestTheAdviceDateIsTheLatestUnsentOne(t *testing.T) {
	store := memory.NewSampleStore()

	// PNC-1001 punya dua PLA: satu belum terkirim (10 Jan) dan satu sudah (12 Jan).
	// Yang digambar adalah yang BELUM terkirim, meski bukan yang paling baru.
	row := rowOf(t, store, "pla", "PNC-1001")
	require.Equal(t, "2026-01-10", row.AdviceDate)
}

// Rentang tanggal menyaring DOKUMEN, bukan tanggal registrasi klaimnya.
func TestTheDateRangeFiltersTheDocumentNotTheClaim(t *testing.T) {
	store := memory.NewSampleStore()

	// PLA PNC-1001 yang belum terkirim bertanggal 10 Januari 2026, sementara klaimnya
	// diregistrasi 5 Januari. Rentang yang hanya memuat tanggal registrasi TIDAK boleh
	// menemukan apa pun.
	onlyRegister := list(t, store, inboxpladlapredla.QueryInput{
		Tab:  "pla",
		From: date(2026, time.January, 1),
		To:   date(2026, time.January, 6),
	})
	require.NotContains(t, onlyRegister, "PNC-1001")

	onlyAdvice := list(t, store, inboxpladlapredla.QueryInput{
		Tab:  "pla",
		From: date(2026, time.January, 10),
		To:   date(2026, time.January, 10),
	})
	require.Contains(t, onlyAdvice, "PNC-1001")
}

// Pencarian mencocokkan kunci klaim, dan tidak peka huruf besar-kecil.
func TestSearchFindsTheClaimByItsWorkKey(t *testing.T) {
	store := memory.NewSampleStore()

	require.Equal(t, []string{"PNC-1001"},
		list(t, store, inboxpladlapredla.QueryInput{Tab: "pla", Search: "pnc-1001"}))

	require.Empty(t,
		list(t, store, inboxpladlapredla.QueryInput{Tab: "pla", Search: "PNC-9999"}))
}

// Urutan mengikuti tanggal registrasi, dengan nomor klaim sebagai pemutus seri.
//
// Tanpa pemutus seri, satu baris dapat muncul di dua halaman sekaligus sementara baris
// lain tidak muncul sama sekali.
func TestRowsAreOrderedByRegistrationDateThenClaimNumber(t *testing.T) {
	store := memory.NewStore()
	store.Seed(
		[]memory.Claim{
			{
				Key: "K/PNC-3", No: "PNC-3", GroupPanel: "003",
				BranchName: "JKT",
				RegisterDate: time.Date(
					2026, time.January, 1, 0, 0, 0, 0, time.UTC),
			},
			{
				Key: "K/PNC-1", No: "PNC-1", GroupPanel: "003",
				BranchName: "JKT",
				RegisterDate: time.Date(
					2026, time.January, 1, 0, 0, 0, 0, time.UTC),
			},
			{
				Key: "K/PNC-2", No: "PNC-2", GroupPanel: "003",
				BranchName: "JKT",
				RegisterDate: time.Date(
					2025, time.December, 31, 0, 0, 0, 0, time.UTC),
			},
		},
		[]memory.Advice{
			{
				ClaimKey: "K/PNC-3", Kind: inboxpladlapredla.KindPLA,
				No: "A3", ReinsCode: "R1",
				Date: time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC),
			},
			{
				ClaimKey: "K/PNC-1", Kind: inboxpladlapredla.KindPLA,
				No: "A1", ReinsCode: "R1",
				Date: time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC),
			},
			{
				ClaimKey: "K/PNC-2", Kind: inboxpladlapredla.KindPLA,
				No: "A2", ReinsCode: "R1",
				Date: time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC),
			},
		},
	)

	require.Equal(t, []string{"PNC-2", "PNC-1", "PNC-3"},
		list(t, store, inboxpladlapredla.QueryInput{Tab: "pla"}))
}

// Paginasi memotong baris dan tetap melaporkan jumlah seluruhnya.
func TestPaginationReportsTheFullTotal(t *testing.T) {
	store := memory.NewSampleStore()

	query, err := inboxpladlapredla.NewQuery(
		inboxpladlapredla.QueryInput{Tab: "pla"}, caller())
	require.NoError(t, err)

	first, err := store.List(context.Background(), query,
		inboxpladlapredla.Pagination{Page: 1, Size: 2})
	require.NoError(t, err)

	require.Len(t, first.Items, 2)
	require.Greater(t, first.Total, 2,
		"total harus menyebut SELURUH baris yang cocok, bukan yang tampil")
}

// Grid rincian mengembalikan SELURUH dokumen klaim itu, terkirim maupun belum.
//
// Kolom "Terkirim" dan "Tanggal Kirim" justru ada supaya perbedaannya terlihat.
func TestTheDocumentGridReturnsSentAndUnsentAlike(t *testing.T) {
	store := memory.NewSampleStore()

	tab, found := inboxpladlapredla.FindTab("pla")
	require.True(t, found)

	items, err := store.Documents(
		context.Background(), tab, "ASM-FW-GCNMFW-WORK PNC-1001")
	require.NoError(t, err)
	require.Len(t, items, 2)

	require.Equal(t, "", items[0].Sent, "yang pertama belum terkirim")
	require.Equal(t, "1", items[1].Sent, "yang kedua sudah terkirim")
	require.Equal(t, "2026-01-13", items[1].SentDate)
}

// Kolom yang hanya ada di salah satu tabel dikosongkan di tabel yang lain.
func TestRevisionIsEmptyOnDLADocumentsAndAcceptanceOnPLADocuments(t *testing.T) {
	store := memory.NewSampleStore()

	pla, _ := inboxpladlapredla.FindTab("pla")
	dla, _ := inboxpladlapredla.FindTab("dla")
	key := "ASM-FW-GCNMFW-WORK PNC-1001"

	plaItems, err := store.Documents(context.Background(), pla, key)
	require.NoError(t, err)
	require.Equal(t, "0", plaItems[0].Revision)
	require.Empty(t, plaItems[0].AcceptanceNo,
		"GetPLAList tidak mengambil NOAKSEP")

	dlaItems, err := store.Documents(context.Background(), dla, key)
	require.NoError(t, err)
	require.Equal(t, "AKS-2026-0001", dlaItems[0].AcceptanceNo)
	require.Empty(t, dlaItems[0].Revision,
		"GetDLAList tidak mengambil REVISI")
}

// Klaim yang ADA tetapi belum punya dokumen menghasilkan daftar KOSONG, bukan galat.
//
// Ia dipisahkan dari kunci yang salah, dan hanya yang kedua yang merupakan kekeliruan.
func TestAClaimWithoutDocumentsReturnsAnEmptyListNotAnError(t *testing.T) {
	store := memory.NewSampleStore()
	tab, _ := inboxpladlapredla.FindTab("pla")

	items, err := store.Documents(
		context.Background(), tab, "ASM-FW-GCNMFW-WORK PNC-1010")

	require.NoError(t, err)
	require.Empty(t, items)
}

// Kunci klaim yang tidak ada menghasilkan ErrRowNotFound.
//
// Penyebab paling mungkin bukan klaim yang benar-benar tidak ada, melainkan kunci yang
// benar dibuka pada portal yang salah (`R-20`).
func TestAnUnknownClaimKeyIsReportedAsNotFound(t *testing.T) {
	store := memory.NewSampleStore()
	tab, _ := inboxpladlapredla.FindTab("pla")

	_, err := store.Documents(
		context.Background(), tab, "ASM-FW-GCNMFW-WORK PNC-9999")

	require.ErrorIs(t, err, inboxpladlapredla.ErrRowNotFound)
}

// Menandai Pre-DLA terkirim mengisi tanggalnya dan mengubah penandanya.
func TestMarkingAPreDLASentRecordsTheDate(t *testing.T) {
	store := memory.NewSampleStore()
	store.Now = func() time.Time { return time.Date(2026, 3, 10, 9, 0, 0, 0, time.UTC) }

	key := "ASM-FW-GCNMFW-WORK PNC-1001"

	berubah, err := store.MarkPreDLASent(context.Background(), key, "PRE/2026/0001")
	require.NoError(t, err)
	require.True(t, berubah)

	rows, err := store.PrintPreDLA(context.Background(), key)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, "1", rows[0].Sent)
	require.Equal(t, "2026-03-10", rows[0].SentDate)
}

// Menekan tombol DUA KALI tidak menimpa tanggal kirim yang pertama.
//
// Ini pagar terhadap masalah `P-1`: selama Pega masih menulis tabel yang sama, penandaan
// dari sini tidak boleh menghapus jejak penandaan sebelumnya — baik oleh petugas lain
// maupun oleh Pega. Tanggal yang tertimpa tidak dapat dipulihkan.
func TestMarkingAnAlreadySentPreDLAChangesNothing(t *testing.T) {
	store := memory.NewSampleStore()
	store.Now = func() time.Time { return time.Date(2026, 3, 10, 9, 0, 0, 0, time.UTC) }

	key := "ASM-FW-GCNMFW-WORK PNC-1001"
	_, err := store.MarkPreDLASent(context.Background(), key, "PRE/2026/0001")
	require.NoError(t, err)

	// Penekanan kedua memakai jam yang berbeda; bila ia menimpa, tanggalnya bergeser.
	store.Now = func() time.Time { return time.Date(2026, 4, 1, 9, 0, 0, 0, time.UTC) }

	berubah, err := store.MarkPreDLASent(context.Background(), key, "PRE/2026/0001")
	require.NoError(t, err)
	require.False(t, berubah, "penekanan kedua seharusnya tidak mengubah apa pun")

	rows, err := store.PrintPreDLA(context.Background(), key)
	require.NoError(t, err)
	require.Equal(t, "2026-03-10", rows[0].SentDate, "tanggal pertama tertimpa")
}

// Nomor Pre-DLA yang tidak ada pada klaim itu tidak mengubah apa pun, dan bukan galat.
func TestMarkingAnUnknownPreDLAChangesNothing(t *testing.T) {
	store := memory.NewSampleStore()

	berubah, err := store.MarkPreDLASent(
		context.Background(), "ASM-FW-GCNMFW-WORK PNC-1001", "PRE/2026/9999")
	require.NoError(t, err)
	require.False(t, berubah)
}

// Klaim yang tidak ada TETAP galat — ia berbeda dari nomor yang tidak ada.
//
// Kunci klaim yang salah hampir selalu berarti panel dibuka pada portal yang keliru
// (`R-20`), dan itu harus terdengar berbeda dari "nomornya sudah terkirim".
func TestMarkingOnAnUnknownClaimIsAnError(t *testing.T) {
	store := memory.NewSampleStore()

	_, err := store.MarkPreDLASent(context.Background(), "ASM-FW-GCNMFW-WORK PNC-9999", "X")
	require.ErrorIs(t, err, inboxpladlapredla.ErrRowNotFound)
}

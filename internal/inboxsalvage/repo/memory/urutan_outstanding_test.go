package memory_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxsalvage"
	"claim-pnc/internal/inboxsalvage/repo/memory"
)

// Daftar Salvage Outstanding diurutkan TANGGAL KEJADIAN TERBARU DULU.
//
// # Kenapa urutannya diputuskan, bukan disalin
//
// Pega TIDAK mengurutkan daftar ini sama sekali — `GcnmSalvageData_OS_SQL` tidak memuat
// satu pun `ORDER BY`. Urutan barisnya di sana ditentukan jalur akses Oracle, berbeda
// antar pemuatan, dan tidak dapat ditiru dengan sengaja.
//
// Itulah yang dilaporkan Work Owner 2026-10-08: jumlah barisnya sudah sama di kedua
// sistem — predikatnya memang identik — tetapi baris yang tampil berbeda, karena hanya
// kita yang mengurutkan. Pilihan urutannya karena itu diputuskan: terbaru dulu.
func TestOutstandingIsOrderedByNewestLossDate(t *testing.T) {
	store := memory.NewStore()

	store.Seed([]memory.Claim{
		{ClaimNo: "PNC-1", SalvageStatus: "5", WorkStatus: "Open", LossDate: "2020-01-03"},
		{ClaimNo: "PNC-2683", SalvageStatus: "5", WorkStatus: "Open", LossDate: "2024-02-13"},
		{ClaimNo: "PNC-2679", SalvageStatus: "3", WorkStatus: "Open", LossDate: "2025-08-14"},
		{ClaimNo: "PNC-1006", SalvageStatus: "5", WorkStatus: "Open", LossDate: "2022-09-12"},
	}, nil)

	rows := outstandingRows(t, store)

	require.Equal(t,
		[]string{"PNC-2679", "PNC-2683", "PNC-1006", "PNC-1"},
		rows,
		"terbaru dulu — bukan menurut nomor klaim")
}

// Tanggal yang SAMA dipecah nomor klaim, dan itu bukan kerapian.
//
// Pada data yang dibandingkan Work Owner, tiga klaim berbagi tanggal 13/06/2025.
// Mengurutkan hanya dengan tanggal membuat urutan di dalam satu tanggal tidak ditentukan —
// dan paginasi di atas urutan yang tidak ditentukan dapat mengulang baris di halaman
// berikutnya atau melewatkannya sama sekali. Persis cacat yang hendak dihindari dengan
// tidak meniru ketiadaan `ORDER BY` milik Pega.
func TestOutstandingBreaksTiesByClaimNumber(t *testing.T) {
	store := memory.NewStore()

	store.Seed([]memory.Claim{
		{ClaimNo: "PNC-2673", SalvageStatus: "5", WorkStatus: "Open", LossDate: "2025-06-13"},
		{ClaimNo: "PNC-2671", SalvageStatus: "3", WorkStatus: "Open", LossDate: "2025-06-13"},
		{ClaimNo: "PNC-2672", SalvageStatus: "5", WorkStatus: "Open", LossDate: "2025-06-13"},
	}, nil)

	rows := outstandingRows(t, store)

	require.Equal(t,
		[]string{"PNC-2671", "PNC-2672", "PNC-2673"},
		rows,
		"tanggal sama diurutkan nomor klaim, naik")
}

// Tanggal kejadian yang KOSONG jatuh ke bawah, bukan ke puncak.
//
// Bawaan Oracle untuk `DESC` adalah `NULLS FIRST`, sehingga tanpa `NULLS LAST` yang tegas
// klaim tanpa tanggal akan menempati puncak daftar — tepat kebalikan dari "terbaru dulu".
// Uji ini menjaga pengisi memori sepadan dengan pengisi SQL pada titik itu.
func TestOutstandingPutsClaimsWithoutALossDateLast(t *testing.T) {
	store := memory.NewStore()

	store.Seed([]memory.Claim{
		{ClaimNo: "PNC-100", SalvageStatus: "5", WorkStatus: "Open", LossDate: ""},
		{ClaimNo: "PNC-200", SalvageStatus: "5", WorkStatus: "Open", LossDate: "2021-05-05"},
		{ClaimNo: "PNC-300", SalvageStatus: "5", WorkStatus: "Open", LossDate: "2023-07-07"},
	}, nil)

	rows := outstandingRows(t, store)

	require.Equal(t,
		[]string{"PNC-300", "PNC-200", "PNC-100"},
		rows,
		"yang tanpa tanggal berada di urutan terakhir")
}

// Daftar keluarga lain TIDAK ikut berubah urutannya.
//
// Work Owner menyatakan kedelapan daftar lain sudah cocok dengan Pega, sehingga mengubah
// urutannya akan merusak yang sudah benar. Hanya daftar Outstanding yang diputuskan.
func TestOtherClaimListsStillOrderByClaimNumber(t *testing.T) {
	store := memory.NewStore()

	store.Seed([]memory.Claim{
		{ClaimNo: "PNC-300", SalvageStatus: "3", WorkStatus: "Open", LossDate: "2025-01-01"},
		{ClaimNo: "PNC-100", SalvageStatus: "3", WorkStatus: "Open", LossDate: "2020-01-01"},
		{ClaimNo: "PNC-200", SalvageStatus: "3", WorkStatus: "Open", LossDate: "2023-01-01"},
	}, nil)

	page := list(t, store, inboxsalvage.TabEkonomis, "")

	var rows []string
	for _, row := range page.Items {
		rows = append(rows, row.ClaimNo)
	}

	require.Equal(t,
		[]string{"PNC-100", "PNC-200", "PNC-300"},
		rows,
		"daftar Ekonomis tetap menurut nomor klaim")
}

// outstandingRows mengambil nomor klaim daftar Outstanding, pada urutan tampilnya.
func outstandingRows(t *testing.T, store *memory.Store) []string {
	t.Helper()

	page := list(t, store, inboxsalvage.TabOutstanding, "")

	rows := make([]string, 0, len(page.Items))
	for _, row := range page.Items {
		rows = append(rows, row.ClaimNo)
	}
	return rows
}

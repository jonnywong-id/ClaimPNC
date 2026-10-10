package inboxsalvage_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxsalvage"
	"claim-pnc/internal/inboxsalvage/repo/memory"
)

// Baris "Outstanding" mencacah PERSIS apa yang daftarnya tampilkan — tidak lebih, tidak
// kurang.
//
// # Bukti angkanya, bukan bacaan rule-nya
//
// Export menunjuk dua penyaring yang saling meniadakan pada activity yang sama, dan dibaca
// apa adanya daftar ini memuat klaim yang BELUM ditandai. Pengukuran terhadap layar Pega
// sungguhan pada 2026-10-08 membantahnya:
//
//	daftar Outstanding di Pega           145 baris
//	"3 atau 5" + masih terbuka           145 baris   <- cocok
//	"belum ditandai" + masih terbuka     465 baris
//
// Ditambah satu pemeriksaan dari arah berbeda oleh Work Owner: satu klaim yang tampil di
// daftar Outstanding Pega juga tampil di daftar TBA, dan TBA menyaring STSSALVAGE='5'.
// Klaim ber-penanda tidak akan pernah lolos penyaring "belum ditandai".
//
// Lihat inboxsalvage.OutstandingSalvageStatuses untuk uraian lengkapnya.
func TestOnlyTheOutstandingCounterFiltersWorkStatus(t *testing.T) {
	var outstanding, lainnya int

	for _, row := range inboxsalvage.CountRows() {
		if row.Source != inboxsalvage.CountFromClaim {
			continue
		}

		if row.Tab == inboxsalvage.TabOutstanding {
			outstanding++
			require.True(t, row.ExcludesClosedWork,
				"baris Outstanding membuang klaim yang pekerjaannya sudah selesai, "+
					"supaya angkanya sepadan dengan daftarnya")
			continue
		}

		lainnya++
		require.False(t, row.ExcludesClosedWork,
			"baris %q mencacah menurut STSSALVAGE TANPA memandang status pekerjaan, "+
				"dan angkanya SUDAH cocok dengan Pega — mengubahnya akan merusak yang "+
				"sudah benar", row.Label)
	}

	require.Equal(t, 1, outstanding, "tepat satu baris Outstanding")
	require.Positive(t, lainnya, "baris sekeluarganya ikut diperiksa")
}

// Outstanding menghitung klaim ber-penanda 3 atau 5 yang pekerjaannya MASIH BERJALAN.
//
// Uji di atas menjaga penandanya; uji ini menjaga penanda itu benar-benar berakibat.
// Keduanya dibutuhkan: penanda yang dipasang tetapi tidak dibaca pengisi seam akan lolos
// dari uji pertama.
func TestOutstandingCountsOnlyMarkedClaimsStillInProgress(t *testing.T) {
	store := memory.NewStore()

	store.Seed([]memory.Claim{
		// Ditandai DAN masih terbuka — ketiganya masuk hitungan Outstanding.
		{ClaimNo: "PNC-3", SalvageStatus: "3", WorkStatus: "Open"},
		{ClaimNo: "PNC-4", SalvageStatus: "5", WorkStatus: "Open"},
		{ClaimNo: "PNC-5", SalvageStatus: "5", WorkStatus: "Pending"},

		// Belum ditandai — TIDAK masuk, sebanyak apa pun. Inilah premis yang dicabut
		// 2026-10-08, dan di sinilah ia akan tertangkap bila kembali.
		{ClaimNo: "PNC-1", SalvageStatus: "", WorkStatus: "Open"},
		{ClaimNo: "PNC-2", SalvageStatus: "", WorkStatus: "Open"},

		// Penanda di luar 3 dan 5 — juga TIDAK masuk.
		{ClaimNo: "PNC-6", SalvageStatus: "4", WorkStatus: "Open"},

		// Ditandai tetapi pekerjaannya sudah selesai — TIDAK masuk.
		{ClaimNo: "PNC-7", SalvageStatus: "3", WorkStatus: "Resolved-Completed"},
		{ClaimNo: "PNC-8", SalvageStatus: "5", WorkStatus: "Resolved-Rejected"},
	}, nil)

	counts, err := store.Counts(context.Background(), inboxsalvage.Caller{Login: "siapa"})
	require.NoError(t, err)

	angka := map[string]int{}
	for _, row := range counts {
		angka[row.Label] = row.Total
	}

	require.Equal(t, 3, angka["Outstanding"],
		"hanya yang ditandai 3 atau 5 DAN masih berjalan")

	// Baris Ekonomis dan TBA tetap mencacah TANPA memandang status pekerjaan — PNC-7 dan
	// PNC-8 ikut terhitung di sana. Inilah selisih yang disengaja, dan angkanya sudah
	// cocok dengan Pega pada perbandingan 2026-10-08.
	require.Equal(t, 2, angka["Ekonomis"], "Ekonomis ikut menghitung yang sudah selesai")
	require.Equal(t, 3, angka["TBA"], "TBA ikut menghitung yang sudah selesai")
}

// Outstanding TIDAK PERNAH melebihi jumlah Ekonomis dan TBA.
//
// Inilah relasi yang dulu tidak dijaga, dan ketiadaannya membuat premis yang keliru bertahan
// dua hari: angka Outstanding sempat melampaui keduanya, yang secara aritmetis mustahil bila
// ia benar-benar menyaring "3 atau 5". Menjaga RELASINYA menangkap kekeliruan yang menjaga
// nilai tidak akan menangkap.
func TestOutstandingNeverExceedsEkonomisPlusTBA(t *testing.T) {
	store := memory.NewStore()

	store.Seed([]memory.Claim{
		{ClaimNo: "PNC-1", SalvageStatus: "", WorkStatus: "Open"},
		{ClaimNo: "PNC-2", SalvageStatus: "", WorkStatus: "Open"},
		{ClaimNo: "PNC-3", SalvageStatus: "", WorkStatus: "Open"},
		{ClaimNo: "PNC-4", SalvageStatus: "3", WorkStatus: "Open"},
		{ClaimNo: "PNC-5", SalvageStatus: "5", WorkStatus: "Open"},
	}, nil)

	counts, err := store.Counts(context.Background(), inboxsalvage.Caller{Login: "siapa"})
	require.NoError(t, err)

	angka := map[string]int{}
	for _, row := range counts {
		angka[row.Label] = row.Total
	}

	require.Equal(t, 2, angka["Outstanding"])
	require.LessOrEqual(t, angka["Outstanding"], angka["Ekonomis"]+angka["TBA"],
		"Outstanding adalah HIMPUNAN BAGIAN dari Ekonomis digabung TBA — ia menyaring "+
			"penanda yang sama, ditambah syarat pekerjaan masih berjalan")
}

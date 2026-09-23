package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxcloseclaim"
	"claim-pnc/internal/inboxcloseclaim/repo/memory"
)

func store(t *testing.T) *memory.Store {
	t.Helper()
	return memory.NewStoreWithSamples()
}

func list(t *testing.T, s *memory.Store, f inboxcloseclaim.Filter) inboxcloseclaim.Page {
	t.Helper()
	page, err := s.List(context.Background(), f)
	require.NoError(t, err)
	return page
}

// TestOnlyClosedClaimsAppear adalah uji yang membedakan layar ini dari Inbox Outstanding.
//
// Keduanya menyaring dua nilai `PYSTATUSWORK` yang SAMA, dengan arah yang BERLAWANAN. Satu
// huruf yang salah membalik seluruh isi layar tanpa satu pun galat.
func TestOnlyClosedClaimsAppear(t *testing.T) {
	s := store(t)
	s.AddClaim(inboxcloseclaim.ClosedClaim{
		ClaimID:       "ASM-FW-GCNMFW-WORK PNCN.26.9999",
		ClaimNumber:   "PNCN.26.9999",
		ProcessStatus: "New", // masih berjalan
		GroupPanel:    "006",
	})

	page := list(t, s, inboxcloseclaim.Filter{})
	for _, claim := range page.Claims {
		require.NotEqual(t, "PNCN.26.9999", claim.ClaimNumber,
			"klaim yang masih berjalan tidak boleh muncul di Inbox Close Claim")
	}
}

// TestBondingFiltersWithIN menjaga perbedaan yang paling mudah terlewat.
//
// Pada modul Inbox Outstanding, BONDING memakai `NOT IN` atas keempat kelompok yang sama.
// Di sini ia `IN`. Menyeragamkan keduanya akan menampilkan lini yang salah tanpa galat.
func TestBondingFiltersWithIN(t *testing.T) {
	s := store(t)

	page := list(t, s, inboxcloseclaim.Filter{Business: inboxcloseclaim.BusinessBonding})
	require.NotEmpty(t, page.Claims, "contoh Bonding harus ada supaya uji ini bermakna")

	for _, claim := range page.Claims {
		require.Contains(t, []string{"10008", "10010", "10015", "10023"}, claim.BusinessGroupID,
			"BONDING memakai IN — hanya kelompok bisnis Bonding yang boleh tampil")
	}
}

// TestNonMBUExcludesBondingGroups menjaga kedua bagian penyaring NONMBU sekaligus.
func TestNonMBUExcludesBondingGroups(t *testing.T) {
	s := store(t)

	page := list(t, s, inboxcloseclaim.Filter{Business: inboxcloseclaim.BusinessNonMBU})
	for _, claim := range page.Claims {
		require.Contains(t, []string{"003", "004", "006"}, claim.GroupPanel)
		require.NotContains(t, []string{"10008", "10010", "10015", "10023"}, claim.BusinessGroupID)
	}
}

// TestNonMBUDoesNotInclude009 menjaga perbedaan kedua terhadap modul lain.
//
// `inboxadmin` memuat Group Panel `'009'` pada NONMBU; activity layar INI tidak. Keduanya
// disalin dari activity-nya masing-masing, dan `P-5` menetapkan perilaku dipertahankan.
func TestNonMBUDoesNotInclude009(t *testing.T) {
	s := store(t)
	s.AddClaim(inboxcloseclaim.ClosedClaim{
		ClaimID:         "ASM-FW-GCNMFW-WORK PNCN.26.0009",
		ClaimNumber:     "PNCN.26.0009",
		ProcessStatus:   inboxcloseclaim.StatusKerjaSelesai,
		GroupPanel:      "009",
		BusinessGroupID: "10005",
	})

	page := list(t, s, inboxcloseclaim.Filter{Business: inboxcloseclaim.BusinessNonMBU})
	for _, claim := range page.Claims {
		require.NotEqual(t, "PNCN.26.0009", claim.ClaimNumber,
			"Group Panel 009 tidak ada di penyaring NONMBU layar ini")
	}
}

// TestUnpaidFilterDropsRowsWithoutStatusCode adalah uji yang paling perlu dibaca utuh.
//
// Ia menegaskan perilaku yang TAMPAK SEPERTI CACAT dan ternyata memang dikehendaki: di
// Oracle `NULL <> '1163'` bernilai UNKNOWN, sehingga klaim tanpa kode status HILANG saat
// penyaring "BELUM LUNAS" dipakai.
//
// DIKONFIRMASI Work Owner 2026-09-23: klaim tanpa kode status **tidak** boleh terbaca
// sebagai belum lunas. Ia klaim yang keadaan bayarnya belum diketahui — bukan klaim yang
// diketahui belum dibayar.
//
// Uji ini karena itu menjaga keputusan, bukan sekadar merekam peniruan: menambahkan
// `IS NULL OR` akan memasukkan baris yang justru tidak boleh ada di sana, dan penambahannya
// tidak menghasilkan galat apa pun.
func TestUnpaidFilterDropsRowsWithoutStatusCode(t *testing.T) {
	s := store(t)

	semua := list(t, s, inboxcloseclaim.Filter{})
	adaTanpaKode := false
	for _, claim := range semua.Claims {
		if claim.ClaimStatusCode == "" {
			adaTanpaKode = true
		}
	}
	require.True(t, adaTanpaKode, "contoh tanpa kode status harus ada supaya uji ini bermakna")

	belumLunas := list(t, s, inboxcloseclaim.Filter{Payment: inboxcloseclaim.PaymentUnpaid})
	for _, claim := range belumLunas.Claims {
		require.NotEmpty(t, claim.ClaimStatusCode,
			"klaim tanpa kode status HILANG dari penyaring BELUM LUNAS — perilaku Oracle yang ditiru")
	}
}

// TestPaidFilterMatchesTheCode menjaga cabang LUNAS.
func TestPaidFilterMatchesTheCode(t *testing.T) {
	s := store(t)

	page := list(t, s, inboxcloseclaim.Filter{Payment: inboxcloseclaim.PaymentPaid})
	require.NotEmpty(t, page.Claims)
	for _, claim := range page.Claims {
		require.Equal(t, inboxcloseclaim.KodeStatusLunas, claim.ClaimStatusCode)
	}
}

// TestTransferFilterBothDirections menjaga kedua cabang Status Transfer.
func TestTransferFilterBothDirections(t *testing.T) {
	s := store(t)

	sudah := list(t, s, inboxcloseclaim.Filter{Transfer: inboxcloseclaim.TransferDone})
	require.NotEmpty(t, sudah.Claims)
	for _, claim := range sudah.Claims {
		require.True(t, claim.TransferredToCashier)
	}

	belum := list(t, s, inboxcloseclaim.Filter{Transfer: inboxcloseclaim.TransferNone})
	require.NotEmpty(t, belum.Claims)
	for _, claim := range belum.Claims {
		require.False(t, claim.TransferredToCashier)
	}
}

// TestSearchLooksAtPolicyAndClaimNumber menjaga kotak cari gabungan.
func TestSearchLooksAtPolicyAndClaimNumber(t *testing.T) {
	s := store(t)

	byClaim := list(t, s, inboxcloseclaim.Filter{Search: "pncn.26.0001"})
	require.Len(t, byClaim.Claims, 1, "pencarian tidak peka huruf besar-kecil")

	byPolicy := list(t, s, inboxcloseclaim.Filter{Search: "POL-CONTOH-0003"})
	require.Len(t, byPolicy.Claims, 1)
}

// TestSeparateFiltersNarrowTogether menjaga keempat penyaring teks hidup berdampingan.
//
// Di sistem lama keempatnya disisipkan ke kueri yang sama lewat penanda `{ASIS:…}` yang
// berbeda, sehingga keempatnya berlaku bersamaan — bukan saling menggantikan.
func TestSeparateFiltersNarrowTogether(t *testing.T) {
	s := store(t)

	page := list(t, s, inboxcloseclaim.Filter{
		PolicyNumber: "POL-CONTOH-0001",
		TechnicalPIC: "PIC TEKNIK",
	})
	require.Len(t, page.Claims, 1)

	kosong := list(t, s, inboxcloseclaim.Filter{
		PolicyNumber: "POL-CONTOH-0001",
		TechnicalPIC: "PIC TRAVEL",
	})
	require.Empty(t, kosong.Claims, "kedua penyaring berlaku bersamaan, bukan saling menggantikan")
}

// TestPaginationKeepsTotal menjaga total menghitung SELURUH yang cocok, bukan halamannya.
func TestPaginationKeepsTotal(t *testing.T) {
	s := store(t)

	page := list(t, s, inboxcloseclaim.Filter{Limit: 2, Offset: 0})
	require.Len(t, page.Claims, 2)
	require.Greater(t, page.Total, 2, "total menghitung seluruh baris yang cocok")

	lewat := list(t, s, inboxcloseclaim.Filter{Limit: 2, Offset: 9999})
	require.Empty(t, lewat.Claims)
	require.Equal(t, page.Total, lewat.Total, "total tidak berubah karena halaman")
}

// TestClosedClaimNumberRejectsRunningClaim menjaga gerbang kedua aksi.
func TestClosedClaimNumberRejectsRunningClaim(t *testing.T) {
	s := store(t)
	s.AddClaim(inboxcloseclaim.ClosedClaim{
		ClaimID:       "ASM-FW-GCNMFW-WORK PNCN.26.8888",
		ClaimNumber:   "PNCN.26.8888",
		ProcessStatus: "New",
	})

	_, err := s.ClosedClaimNumber(context.Background(), "ASM-FW-GCNMFW-WORK PNCN.26.8888")
	require.ErrorIs(t, err, inboxcloseclaim.ErrClaimNotFound,
		"klaim yang masih berjalan tidak boleh dapat diajukan dari layar ini")

	number, err := s.ClosedClaimNumber(context.Background(), "ASM-FW-GCNMFW-WORK PNCN.26.0001")
	require.NoError(t, err)
	require.Equal(t, "PNCN.26.0001", number)
}

// TestRecordRejectsSecondPendingRequest meniru constraint unik migrasi `0006`.
//
// Bila keduanya berbeda, uji di atas memori akan meloloskan keadaan yang Oracle tolak — atau
// sebaliknya, dan yang kedua jauh lebih berbahaya: dua permintaan salin yang lolos berarti
// DUA KLAIM BARU dari satu tombol yang ditekan dua kali.
func TestRecordRejectsSecondPendingRequest(t *testing.T) {
	s := store(t)
	ctx := context.Background()
	now := time.Now().UTC()

	first := inboxcloseclaim.ClaimRequest{
		ID:          "permintaan-1",
		Kind:        inboxcloseclaim.RequestReopen,
		ClaimID:     "ASM-FW-GCNMFW-WORK PNCN.26.0001",
		Status:      inboxcloseclaim.RequestPending,
		ActorLogin:  "BUDI",
		RequestedAt: now,
	}
	require.NoError(t, s.Record(ctx, first))

	second := first
	second.ID = "permintaan-2"
	require.ErrorIs(t, s.Record(ctx, second), inboxcloseclaim.ErrRequestPending)

	// Jenis yang BERBEDA atas klaim yang sama tetap boleh: yang dibatasi adalah satu
	// permintaan menunggu per (jenis, klaim).
	salin := first
	salin.ID = "permintaan-3"
	salin.Kind = inboxcloseclaim.RequestCopy
	require.NoError(t, s.Record(ctx, salin))
}

// TestPendingForOnlyReturnsWaitingRequests menjaga permintaan yang sudah dijalankan tidak
// lagi ditandai di layar.
func TestPendingForOnlyReturnsWaitingRequests(t *testing.T) {
	s := store(t)
	ctx := context.Background()

	require.NoError(t, s.Record(ctx, inboxcloseclaim.ClaimRequest{
		ID:          "permintaan-1",
		Kind:        inboxcloseclaim.RequestReopen,
		ClaimID:     "ASM-FW-GCNMFW-WORK PNCN.26.0001",
		Status:      inboxcloseclaim.RequestExecuted,
		ActorLogin:  "BUDI",
		RequestedAt: time.Now().UTC(),
	}))

	pending, err := s.PendingFor(ctx, []string{"ASM-FW-GCNMFW-WORK PNCN.26.0001"})
	require.NoError(t, err)
	require.Empty(t, pending, "permintaan yang sudah dijalankan tidak lagi tertunda")
}

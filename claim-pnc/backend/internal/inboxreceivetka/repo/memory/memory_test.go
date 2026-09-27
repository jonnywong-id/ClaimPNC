package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxreceivetka"
	"claim-pnc/internal/inboxreceivetka/repo/memory"
)

func completedAt() time.Time {
	return time.Date(2026, 9, 24, 0, 0, 0, 0, time.UTC)
}

// Daftar diurutkan menurut tanggal pendaftaran menaik — yang paling lama menunggu lebih
// dulu — dan yang tanggalnya kosong jatuh di akhir.
//
// Urutan itu padanan `ORDER BY w.REGISTERDATE_1, w.PYID` pada adapter SQL. Lihat catatan
// pada Repo.List.
func TestListOrdersByRegistrationDate(t *testing.T) {
	page, err := memory.NewSampleRepo().List(context.Background(), inboxreceivetka.Filter{})
	require.NoError(t, err)

	var order []string
	for _, one := range page.Tasks {
		order = append(order, one.ClaimNumber)
	}

	require.Equal(t, []string{
		"PNC-1546", // 2023-05-10
		"PNC-1729", // 2024-03-19
		"PNC-1811", // 2025-02-20
		"PNC-1902", // 2026-06-01
		"PNC-1977", // 2026-09-01
		"PNC-1955", // tanpa tanggal -> jatuh di akhir
	}, order)
}

func TestListFiltersOnFourTextColumns(t *testing.T) {
	cases := []struct {
		name    string
		keyword string
		want    int
	}{
		{name: "nomor klaim", keyword: "1546", want: 1},
		{name: "nomor polis", keyword: "12000000000002", want: 1},
		{name: "nama tertanggung", keyword: "bahari", want: 1},
		{name: "nama peserta", keyword: "Peserta Contoh Tiga", want: 1},
		{name: "tanpa memandang huruf", keyword: "pnc-19", want: 3},
		{name: "tidak cocok apa pun", keyword: "tidak ada", want: 0},
	}

	for _, one := range cases {
		t.Run(one.name, func(t *testing.T) {
			page, err := memory.NewSampleRepo().List(context.Background(),
				inboxreceivetka.Filter{Keyword: one.keyword})
			require.NoError(t, err)
			require.Len(t, page.Tasks, one.want)
		})
	}
}

// Pemotongan pada MaxRows ditiru persis, termasuk penandanya.
//
// Repo memori yang tidak pernah memotong akan membuat jalur "terpotong" tidak pernah tercoba
// sampai produksi — dan jalur itulah satu-satunya perbedaan yang disengaja terhadap
// `pyMaxRecords = 500` sistem lama.
func TestListTruncatesAndSaysSo(t *testing.T) {
	registeredOn := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	many := make([]inboxreceivetka.Task, 0, inboxreceivetka.MaxRows+5)
	for i := 0; i < inboxreceivetka.MaxRows+5; i++ {
		many = append(many, inboxreceivetka.Task{
			Reference:    "ASM-FW-GCNMFW-WORK PNC-9" + time.Duration(i).String(),
			ClaimKey:     "ASM-FW-GCNMFW-WORK PNC-9" + time.Duration(i).String(),
			ClaimNumber:  "PNC-9" + time.Duration(i).String(),
			RegisteredOn: &registeredOn,
		})
	}

	page, err := memory.NewRepo(memory.Options{Tasks: many}).
		List(context.Background(), inboxreceivetka.Filter{})
	require.NoError(t, err)

	require.Len(t, page.Tasks, inboxreceivetka.MaxRows)
	require.True(t, page.Truncated)
}

func TestListNotTruncatedWhenItFits(t *testing.T) {
	page, err := memory.NewSampleRepo().List(context.Background(), inboxreceivetka.Filter{})
	require.NoError(t, err)
	require.False(t, page.Truncated)
}

// Complete membuang barisnya dari daftar — padanan `TGL_DOC_LENGKAP` yang terisi lalu
// tersaring keluar oleh `IS NULL` pada adapter SQL.
//
// Ciri kedua Inbox pada `D-79` — baris hilang setelah dikerjakan — bertumpu pada ini.
func TestCompleteRemovesRowFromInbox(t *testing.T) {
	repo := memory.NewSampleRepo()

	saved, err := repo.Complete(context.Background(), inboxreceivetka.Completion{
		ClaimNumber: "PNC-1546",
		CompletedAt: completedAt(),
	})
	require.NoError(t, err)

	// Nilai yang dikembalikan dipakai menyusun surel, sehingga keenamnya harus benar.
	require.Equal(t, "PNC-1546", saved.ClaimNumber)
	require.Equal(t, "12000000000002", saved.PolicyNumber)
	require.Equal(t, "PT Contoh Sejahtera Abadi", saved.InsuredName)
	require.Equal(t, "Peserta Contoh Satu", saved.ParticipantName)
	require.NotNil(t, saved.DateOfLoss)
	require.NotEmpty(t, saved.ClaimKey)

	page, err := repo.List(context.Background(), inboxreceivetka.Filter{})
	require.NoError(t, err)
	for _, one := range page.Tasks {
		require.NotEqual(t, "PNC-1546", one.ClaimNumber, "baris yang sudah diisi harus hilang")
	}
}

// Pengisian kedua atas baris yang sama ditolak, bukan diterima diam-diam.
//
// Inilah yang membuat surel ganda tidak dapat terjadi ketika pengguna menekan Submit dua
// kali.
func TestCompleteTwiceIsRejected(t *testing.T) {
	repo := memory.NewSampleRepo()
	one := inboxreceivetka.Completion{ClaimNumber: "PNC-1546", CompletedAt: completedAt()}

	_, err := repo.Complete(context.Background(), one)
	require.NoError(t, err)

	_, err = repo.Complete(context.Background(), one)
	require.ErrorIs(t, err, inboxreceivetka.ErrTaskNotFound)
}

func TestCompleteUnknownClaimIsRejected(t *testing.T) {
	_, err := memory.NewSampleRepo().Complete(context.Background(),
		inboxreceivetka.Completion{ClaimNumber: "PNC-000000", CompletedAt: completedAt()})
	require.ErrorIs(t, err, inboxreceivetka.ErrTaskNotFound)
}

// Baris YATIM — ada di inbox, tetapi klaimnya tidak ditemukan — TAMPIL di daftar dan DITOLAK
// saat diisi.
//
// Keduanya diuji bersama karena keduanya adalah satu keputusan: gabungan ke `T_CLAIM_PNC`
// sengaja LEFT supaya pekerjaannya tidak hilang diam-diam, dan konsekuensinya penolakan itu
// baru terjadi saat Submit.
func TestOrphanRowIsListedButCannotBeCompleted(t *testing.T) {
	repo := memory.NewSampleRepo()

	page, err := repo.List(context.Background(), inboxreceivetka.Filter{})
	require.NoError(t, err)

	var orphan *inboxreceivetka.Task
	for i := range page.Tasks {
		if page.Tasks[i].ClaimNumber == "PNC-1977" {
			orphan = &page.Tasks[i]
		}
	}
	require.NotNil(t, orphan, "baris yatim harus TETAP tampil di daftar")
	require.Empty(t, orphan.ClaimKey, "baris yatim tidak punya pasangan di tabel klaim")

	_, err = repo.Complete(context.Background(), inboxreceivetka.Completion{
		ClaimNumber: "PNC-1977",
		CompletedAt: completedAt(),
	})
	require.ErrorIs(t, err, inboxreceivetka.ErrClaimMissing)
}

// Nomor klaim kembar menghentikan penulisan, bukan memilih salah satu barisnya.
func TestCompleteRejectsDuplicateClaimNumber(t *testing.T) {
	twin := inboxreceivetka.Task{
		Reference:   "ASM-FW-GCNMFW-WORK PNC-1546",
		ClaimKey:    "ASM-FW-GCNMFW-WORK PNC-1546",
		ClaimNumber: "PNC-1546",
	}
	repo := memory.NewRepo(memory.Options{Tasks: []inboxreceivetka.Task{twin, twin}})

	_, err := repo.Complete(context.Background(), inboxreceivetka.Completion{
		ClaimNumber: "PNC-1546",
		CompletedAt: completedAt(),
	})
	require.ErrorIs(t, err, inboxreceivetka.ErrClaimAmbiguous)
}

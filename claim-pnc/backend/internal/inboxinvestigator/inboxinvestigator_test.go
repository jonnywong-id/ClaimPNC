package inboxinvestigator_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxinvestigator"
)

func TestPenyaringDipangkasSpasinyaDiKeduaUjung(t *testing.T) {
	// Kata kunci datang dari parameter query yang mudah menyisakan spasi. Tanpa pemangkasan,
	// pola LIKE-nya menjadi `% PNC-1 %` dan tidak cocok dengan apa pun — hasilnya kosong
	// tanpa satu pun tanda bahwa sebabnya spasi.
	filter := inboxinvestigator.Filter{Keyword: "  PNC-100241  "}

	require.Equal(t, "PNC-100241", filter.Clean().Keyword)
}

func TestWorkbasketAdalahAntreanInvestigator(t *testing.T) {
	// Nilainya terbaca dari `Section/InputInvestigator_Section-Section.xml`, yang
	// mengirimkannya ke Report Definition sebagai `Operator = "InvestigatorPNC"`.
	//
	// Uji ini mengunci nilainya. Ia tampak sepele, dan justru itu sebabnya ia perlu: salah
	// ketik di sini menghasilkan inbox yang selalu kosong — bukan galat — dan pengguna akan
	// melaporkannya sebagai "tidak ada pekerjaan", bukan sebagai kerusakan.
	require.Equal(t, "InvestigatorPNC", inboxinvestigator.Workbasket)
}

func TestStatusSelesaiAdalahYangDisaringKeluar(t *testing.T) {
	// Report Definition menyaring `.pyStatusWork != "Resolved-Completed"`. Itulah yang
	// membuat baris HILANG dari inbox setelah dikerjakan — ciri kedua Inbox pada `D-79`.
	require.Equal(t, "Resolved-Completed", inboxinvestigator.ResolvedWorkStatus)
}

func TestBatasBarisMengikutiPyMaxRecordsSistemLama(t *testing.T) {
	// `pyMaxRecords = 500` pada `InboxRegisterCompliance_RD`. Angkanya diwarisi; yang
	// berubah adalah pemotongannya dinyatakan lewat Page.Truncated alih-alih senyap.
	require.Equal(t, 500, inboxinvestigator.MaxRows)
}

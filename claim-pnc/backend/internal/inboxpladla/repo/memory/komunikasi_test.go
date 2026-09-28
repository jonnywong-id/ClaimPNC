package memory_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxpladla"
	"claim-pnc/internal/inboxpladla/repo/memory"
)

// Uji di berkas ini menjaga KETIGA daftar komunikasi — `SetDataPLADLA` tipe 4, 5, dan 6.
//
// Ketiganya tidak pernah dibangun sampai 2026-09-28, dan ketiadaannya tidak terlihat dari
// layar: tampilan yang tidak punya tab tidak pernah diminta siapa pun.

// Tab "Komunikasi Masuk" berisi pesan yang DITUJUKAN kepada pemanggil.
func TestTheInboundListHoldsMessagesAddressedToTheCaller(t *testing.T) {
	store := memory.NewSampleStore()

	got := list(t, store, memory.SampleReinsurerLogin, "komunikasi-masuk", "")

	require.ElementsMatch(t, []string{"PNC-2001", "PNC-2009"}, got)
}

// Tab "Terkirim — Belum Dijawab" berisi pesan yang DIKIRIM pemanggil dan belum dijawab.
func TestTheOutboundListHoldsUnansweredMessagesSentByTheCaller(t *testing.T) {
	store := memory.NewSampleStore()

	got := list(t, store, memory.SampleReinsurerLogin, "komunikasi-terkirim", "")

	require.Equal(t, []string{"PNC-2002"}, got)
}

// Tab "Terkirim — Sudah Dijawab" berisi pesan yang dikirim pemanggil dan SUDAH dijawab.
//
// Klaim yang sama muncul di kedua tab terkirim, dan itu benar: ia punya DUA percakapan,
// satu sudah dijawab dan satu belum. Penyaringnya bekerja pada PERCAKAPAN, bukan pada
// klaim.
func TestTheAnsweredListHoldsMessagesThatWereReplied(t *testing.T) {
	store := memory.NewSampleStore()

	got := list(t, store, memory.SampleReinsurerLogin, "komunikasi-dijawab", "")

	require.Equal(t, []string{"PNC-2002"}, got)
}

// Percakapan milik mitra LAIN tidak memasukkan klaimnya ke daftar mana pun.
func TestAConversationBetweenOtherPartiesNeverEntersTheLists(t *testing.T) {
	store := memory.NewSampleStore()

	// KOM-04 ada pada PNC-2001 tetapi ditujukan ke login kedua. Login kedua TIDAK punya
	// pemberitahuan apa pun pada klaim itu, sehingga bila penyaringnya bocor, klaim itu
	// akan muncul di daftar login kedua.
	got := list(t, store, memory.SampleSecondLogin, "komunikasi-terkirim", "")
	require.Empty(t, got, "login kedua tidak mengirim satu pun pesan")
}

// Ketiga daftar komunikasi TIDAK mengecualikan Personal Accident maupun Travel.
//
// PNC-2009 lini `002`. Ia tidak muncul di satu pun daftar pemberitahuan — baik karena lini
// bisnisnya dikecualikan, maupun karena tidak ada PLA/DLA yang dikirimkan kepadanya —
// tetapi ia MUNCUL di daftar komunikasi.
//
// Perbedaan itu ada di kueri Pega: `BrowseCommunicationReas` tidak memuat satu pun syarat
// `grouppanel`. Menyeragamkannya akan menghilangkan klaim PA tanpa satu pun galat.
func TestTheCommunicationListsDoNotExcludePersonalAccident(t *testing.T) {
	store := memory.NewSampleStore()
	login := memory.SampleReinsurerLogin

	require.Contains(t,
		list(t, store, login, "komunikasi-masuk", ""), "PNC-2009",
		"daftar komunikasi seharusnya memuat klaim lini 002")

	for _, tab := range []string{"pla", "dla", "close"} {
		require.NotContains(t, list(t, store, login, tab, ""), "PNC-2009",
			"daftar %s mengecualikan lini 002", tab)
	}
}

// Daftar komunikasi TIDAK menuntut dokumen pemberitahuan terkirim.
//
// PNC-2009 tidak punya satu pun PLA maupun DLA. Bila penyaring dokumen ikut diberlakukan,
// ia akan hilang — dan hilangnya tidak akan terlihat sampai seseorang membandingkannya
// dengan Pega.
func TestTheCommunicationListsDoNotRequireASentAdvice(t *testing.T) {
	store := memory.NewSampleStore()

	tab, found := inboxpladla.FindTab("komunikasi-masuk")
	require.True(t, found)
	require.Empty(t, tab.AdviceKindSent)

	require.Contains(t,
		list(t, store, memory.SampleReinsurerLogin, "komunikasi-masuk", ""),
		"PNC-2009")
}

// Kolom "PLA No" BOLEH kosong pada daftar komunikasi, dan itu bukan kerusakan.
//
// Ia tetap diambil dengan rantai reasuradur yang sama — kueri Pega pun mengambilnya —
// tetapi klaim yang masuk daftar ini karena percakapan belum tentu punya PLA.
func TestTheAdviceNumberMayBeEmptyOnTheCommunicationLists(t *testing.T) {
	store := memory.NewSampleStore()

	row := rowOf(t, store, memory.SampleReinsurerLogin, "komunikasi-masuk", "PNC-2009")
	require.Empty(t, row.AdviceNo)

	// Pada klaim yang PUNYA PLA, kolomnya tetap terisi.
	row = rowOf(t, store, memory.SampleReinsurerLogin, "komunikasi-masuk", "PNC-2001")
	require.Equal(t, "PLA/2026/2001-R1", row.AdviceNo)
}

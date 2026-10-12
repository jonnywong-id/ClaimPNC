package memory_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxpladla"
	"claim-pnc/internal/inboxpladla/repo/memory"
)

// Uji di berkas ini menjaga KETIGA daftar komunikasi — `SetDataPLADLA` tipe 4, 5, dan 6.
//
// Ketiganya tidak pernah dibangun sampai 2026-09-28, dan ketiadaannya tidak terlihat dari
// layar: tampilan yang tidak punya tab tidak pernah diminta siapa pun.

// Tab "NOT ANSWERED" berisi pesan yang DITUJUKAN kepada pemanggil.
func TestTheInboundListHoldsMessagesAddressedToTheCaller(t *testing.T) {
	store := memory.NewSampleStore()

	got := list(t, store, memory.SampleReinsurerLogin, "not-answered", "")

	require.ElementsMatch(t, []string{"PNC-2001", "PNC-2009"}, got)
}

// Tab "NOT REPLIED FROM ASM" berisi pesan yang DIKIRIM pemanggil dan belum dijawab.
func TestTheOutboundListHoldsUnansweredMessagesSentByTheCaller(t *testing.T) {
	store := memory.NewSampleStore()

	got := list(t, store, memory.SampleReinsurerLogin, "not-replied-from-asm", "")

	require.Equal(t, []string{"PNC-2002"}, got)
}

// Tab "REPLIED FROM ASM" berisi pesan yang dikirim pemanggil dan SUDAH dijawab.
//
// Klaim yang sama muncul di kedua tab terkirim, dan itu benar: ia punya DUA percakapan,
// satu sudah dijawab dan satu belum. Penyaringnya bekerja pada PERCAKAPAN, bukan pada
// klaim.
func TestTheAnsweredListHoldsMessagesThatWereReplied(t *testing.T) {
	store := memory.NewSampleStore()

	got := list(t, store, memory.SampleReinsurerLogin, "replied-from-asm", "")

	require.Equal(t, []string{"PNC-2002"}, got)
}

// Percakapan milik mitra LAIN tidak memasukkan klaimnya ke daftar mana pun.
func TestAConversationBetweenOtherPartiesNeverEntersTheLists(t *testing.T) {
	store := memory.NewSampleStore()

	// KOM-04 ada pada PNC-2001 tetapi ditujukan ke login kedua. Login kedua TIDAK punya
	// pemberitahuan apa pun pada klaim itu, sehingga bila penyaringnya bocor, klaim itu
	// akan muncul di daftar login kedua.
	got := list(t, store, memory.SampleSecondLogin, "not-replied-from-asm", "")
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
		list(t, store, login, "not-answered", ""), "PNC-2009",
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

	tab, found := inboxpladla.FindTab("not-answered")
	require.True(t, found)
	require.Empty(t, tab.AdviceKindSent)

	require.Contains(t,
		list(t, store, memory.SampleReinsurerLogin, "not-answered", ""),
		"PNC-2009")
}

// Kolom "PLA No" BOLEH kosong pada daftar komunikasi, dan itu bukan kerusakan.
//
// Ia tetap diambil dengan rantai reasuradur yang sama — kueri Pega pun mengambilnya —
// tetapi klaim yang masuk daftar ini karena percakapan belum tentu punya PLA.
func TestTheAdviceNumberMayBeEmptyOnTheCommunicationLists(t *testing.T) {
	store := memory.NewSampleStore()

	row := rowOf(t, store, memory.SampleReinsurerLogin, "not-answered", "PNC-2009")
	require.Empty(t, row.AdviceNo)

	// Pada klaim yang PUNYA PLA, kolomnya tetap terisi.
	row = rowOf(t, store, memory.SampleReinsurerLogin, "not-answered", "PNC-2001")
	require.Equal(t, "PLA/2026/2001-R1", row.AdviceNo)
}

// KEENAM daftar terisi untuk login pengembangan yang ditentukan.
//
// # Kenapa uji ini ada, dan kenapa ia memeriksa KEENAMNYA sekaligus
//
// Permintaan Work Owner berbunyi "semuanya untuk akun JONNY bisa", dan "bisa" di sini
// berarti dua hal yang berbeda: tabnya tergambar, DAN ada isinya. Tab yang tergambar tetapi
// selalu kosong tidak dapat dibedakan dari penyaring yang rusak.
//
// Uji ini karena itu tidak memeriksa "tidak galat" melainkan "ada barisnya" — pada keenam
// daftar, dengan satu login. Ia yang akan gagal bila kelak seseorang memindahkan satu
// percakapan contoh atau satu pemberitahuan contoh ke login lain.
func TestEveryListHasRowsForTheDevelopmentLogin(t *testing.T) {
	const dev = "JONNY"

	store := memory.NewSampleStoreFor(dev)

	// Kode tabnya ditulis lengkap di sini, bukan diambil dari inboxpladla.Tabs(), supaya
	// uji ini ikut gagal bila sebuah tab DIHILANGKAN — bukan hanya bila isinya kosong.
	daftar := []string{
		"pla", "dla", "close",
		"not-answered", "not-replied-from-asm", "replied-from-asm",
	}

	for _, kode := range daftar {
		require.NotEmpty(t, list(t, store, dev, kode, ""),
			"daftar %q kosong untuk login pengembangan — tabnya tergambar tetapi "+
				"tidak dapat dipakai", kode)
	}
}

// Login yang TIDAK terdaftar tetap ditolak, apa pun daftarnya.
//
// Ini sisi lain dari uji di atasnya, dan ia yang menjaga agar penggantian login pada data
// contoh tidak berubah menjadi jalan pintas: yang berpindah adalah SIAPA mitranya, bukan
// longgarnya penyaring.
func TestAnUnregisteredLoginIsStillRefusedOnEveryList(t *testing.T) {
	store := memory.NewSampleStoreFor("JONNY")

	codes, err := store.ReinsurerCodes(context.Background(), "ORANGLAIN")
	require.NoError(t, err)
	require.Empty(t, codes,
		"login yang tidak terdaftar tidak boleh mendapat satu pun kode reasuradur")

	// Tanpa kode, permintaan daftarnya bahkan tidak dapat disusun — itulah yang
	// diterjemahkan lapisan aplikasi menjadi pesan "layar ini untuk mitra reasuransi".
	_, err = inboxpladla.NewQuery(
		inboxpladla.QueryInput{Tab: "pla"},
		inboxpladla.Caller{Login: "ORANGLAIN"},
		codes,
	)
	require.ErrorIs(t, err, inboxpladla.ErrCallerNotAReinsurer)
}

package inboxsalvage_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxsalvage"
)

// Uji di berkas ini menjaga TEKS YANG DIBACA PENGGUNA, bukan perilaku.
//
// # Kenapa ia perlu dijaga uji sama sekali
//
// Karena teks di modul ini ditulis di tempat yang sama dengan komentar Go — di dalam
// `tab.go`, berdampingan baris demi baris — dan kedua jenis tulisan itu punya pembaca yang
// berbeda dengan aturan yang berbeda. Komentar dibaca pengembang di dalam editor, tempat
// backtick berarti "ini nama kolom". Teks tab dibaca petugas di dalam `<p>`, tempat
// backtick berarti backtick.
//
// Keduanya pernah tertukar, dan ketahuan 2026-10-08 dari layar sungguhan: catatan daftar
// Outstanding tergambar berbunyi "klaim ber-`STSSALVAGE` 3 atau 5" — lengkap dengan
// backticknya.

// userFacingTexts mengumpulkan SELURUH teks yang berakhir di layar.
//
// Nama tab ikut, karena ia judul grid. Label pencarian ikut, karena ia label kotak cari.
// Yang TIDAK ikut hanyalah kode tab, yang memang tidak pernah digambar.
func userFacingTexts(t *testing.T) map[string]string {
	t.Helper()

	// SELURUH tiga belas, bukan hanya sembilan yang ditawarkan. Definisi keempat yang
	// tersembunyi tetap disimpan, dan yang disimpan akan tergambar begitu ia ditawarkan.
	texts := map[string]string{}
	for _, tab := range inboxsalvage.AllTabs() {
		texts[tab.Code+".nama"] = tab.Name
		texts[tab.Code+".keterangan"] = tab.Description
		texts[tab.Code+".label_pencarian"] = tab.SearchLabel

		if tab.Notice != "" {
			texts[tab.Code+".catatan"] = tab.Notice
		}
	}
	return texts
}

// Teks yang dibaca pengguna TIDAK boleh memuat penanda Markdown.
//
// Layar menggambarnya sebagai teks biasa — `{tab.keterangan}` di dalam `<p>` — sehingga
// backtick, bintang ganda, dan garis bawah ganda sampai ke mata pembaca apa adanya.
//
// Nama kolom basis data TETAP boleh disebut; yang dilarang hanyalah membungkusnya dengan
// penanda yang tidak pernah diterjemahkan siapa pun.
func TestUserFacingTextCarriesNoMarkdown(t *testing.T) {
	forbidden := map[string]string{
		"`":  "backtick",
		"**": "bintang ganda",
		"__": "garis bawah ganda",
	}

	for where, text := range userFacingTexts(t) {
		for marker, name := range forbidden {
			require.NotContains(t, text, marker,
				"teks %q memuat %s — ia tergambar apa adanya di layar, "+
					"bukan diterjemahkan", where, name)
		}
	}
}

// Teks yang dibaca pengguna TIDAK boleh menunjuk ke sesuatu yang tidak ada di layar.
//
// # Apa yang pernah terjadi
//
// Dua catatan daftar berbunyi "lihat keterangan selisih terencana". Panel itu DIHAPUS dari
// layar pada 2026-10-06, dan sejak itu `PlannedDifferences` tidak lagi dikirim ke layar
// sama sekali — `http/dto.go` tidak punya isian untuknya. Kalimatnya karena itu menyuruh
// pengguna melihat sesuatu yang tidak ada di mana pun, dan ia bertahan dua hari tanpa satu
// pun uji yang menangkapnya.
//
// # Kenapa penjaganya berupa daftar hitam, bukan pemeriksaan yang lebih pintar
//
// Karena yang dapat diperiksa dari sini hanyalah KATA-KATANYA. Apakah sebuah panel
// tergambar adalah urusan layar, dan modul ini tidak boleh tahu apa-apa tentang layar
// (`11-CROSSCUTTING.md` §1.2). Daftar hitam ini menangkap rujukan yang sudah diketahui
// mati; rujukan baru yang mati tetap menuntut mata manusia.
func TestUserFacingTextDoesNotPointAtWhatTheScreenNoLongerDraws(t *testing.T) {
	// Setiap baris: frasa yang dilarang, dan sebab ia dilarang.
	removed := map[string]string{
		"selisih terencana": "panel selisih terencana dihapus dari layar 2026-10-06, " +
			"dan PlannedDifferences tidak lagi dikirim ke layar",
		"bilah tab": "bilah tab dicabut 2026-10-03; tabel ringkas menjadi " +
			"satu-satunya navigasi",
	}

	for where, text := range userFacingTexts(t) {
		lower := strings.ToLower(text)
		for phrase, reason := range removed {
			require.NotContains(t, lower, phrase,
				"teks %q menunjuk ke %q — %s", where, phrase, reason)
		}
	}
}

// Catatan daftar harus BERDIRI SENDIRI: ia menjelaskan, bukan menunjuk ke tempat lain.
//
// Inilah aturan yang seharusnya mencegah cacat di atas sejak awal. Catatan yang menunjuk ke
// tempat lain hanya berguna selama tempat itu ada, dan tidak ada yang memberi tahu
// penulisnya ketika tempat itu dibongkar.
func TestEveryNoticeExplainsItselfInsteadOfPointingElsewhere(t *testing.T) {
	for _, tab := range inboxsalvage.AllTabs() {
		if tab.Notice == "" {
			continue
		}

		// "lihat" diperbolehkan hanya bila yang dirujuknya ada di dalam kalimat yang sama
		// — misalnya "lihat kolom Aging di sebelah kanan". Yang dilarang adalah menyuruh
		// pembaca mencari panel atau layar lain.
		lower := strings.ToLower(tab.Notice)
		for _, pointer := range []string{"lihat keterangan", "lihat panel", "lihat daftar di"} {
			require.NotContains(t, lower, pointer,
				"catatan daftar %q menunjuk ke tempat lain alih-alih menjelaskan "+
					"sendiri", tab.Code)
		}
	}
}

// Daftar Outstanding TIDAK lagi memikul catatan selisih, karena selisihnya sudah tidak ada.
//
// # Apa yang berubah
//
// Catatan itu dulu menerangkan mengapa angka pada tabel ringkas berbeda dengan jumlah baris
// daftarnya. Sejak 2026-10-08 pencacah dan daftarnya mencacah populasi yang SAMA, sehingga
// keterangannya berubah dari membantu menjadi menyesatkan.
//
// # Kenapa dijaga uji, bukan dihapus begitu saja
//
// Karena kalimat semacam itu paling mudah kembali: ia terdengar seperti pengetahuan domain
// yang berharga, dan pembacanya tidak punya cara mengetahui bahwa premisnya sudah dicabut.
// Uji ini menolaknya dari sisi KATA-KATANYA, dan `repo/memory` menolaknya dari sisi
// ANGKANYA — lihat TestOutstandingCounterMatchesItsOwnList.
func TestOutstandingCarriesNoStaleDifferenceNotice(t *testing.T) {
	for _, tab := range inboxsalvage.AllTabs() {
		if tab.Code != inboxsalvage.TabOutstanding {
			continue
		}

		lower := strings.ToLower(tab.Notice)
		for _, frasa := range []string{"tidak sama", "berbeda", "bukan kerusakan"} {
			require.NotContains(t, lower, frasa,
				"catatan daftar Outstanding masih mengaku angkanya berselisih, "+
					"padahal pencacah dan daftarnya kini mencacah populasi yang sama")
		}
	}
}

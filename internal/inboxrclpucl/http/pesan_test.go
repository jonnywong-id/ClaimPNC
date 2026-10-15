package inboxrclpuclhttp

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxrclpucl"
)

// TestKalimatKirimMenyebutKeMANAKlaimPindah menjaga kejujuran kalimat kedua tombol Kirim.
//
// # Riwayat penjaga ini, karena ia sudah DUA KALI berganti makna
//
// Bentuk pertama kalimatnya berbunyi "Klaim diteruskan ke Analyst." — tidak benar, dan Work
// Owner menemukannya dari keadaan nyata: klaimnya tidak bergerak sama sekali.
//
// Bentuk kedua menyatakan klaimnya MASIH di antrean dan BELUM bergerak. Itu benar pada
// saatnya, dan tidak lagi benar sejak tombolnya memindahkan tugas sendiri ke tahap
// Send To Analis.
//
// Yang dijaga sekarang bukan "jangan menjanjikan", melainkan **sebutkan ke mana** — karena
// perpindahannya kini terjadi, dan petugas perlu tahu klaimnya menjadi pekerjaan siapa.
func TestKalimatKirimMenyebutKeMANAKlaimPindah(t *testing.T) {
	for _, kind := range []inboxrclpucl.ClaimActionKind{
		inboxrclpucl.ActionSendToAnalyst,
		inboxrclpucl.ActionSendToPICTeknik,
	} {
		pesan := actionDoneMessage(kind)

		require.Containsf(t, pesan, "Send To Analis",
			"kalimat %s wajib menyebut tahap tujuannya", kind)
		require.Containsf(t, pesan, "PIC Teknik",
			"kalimat %s wajib menyebut klaimnya menjadi pekerjaan siapa", kind)

		// Kedua bentuk lama TIDAK boleh kembali: keduanya kini menyatakan hal yang salah.
		require.NotContainsf(t, pesan, "BELUM bergerak",
			"kalimat %s menyatakan klaim tidak bergerak, padahal ia pindah", kind)
		require.NotEqualf(t, "Klaim diteruskan ke Analyst.", pesan,
			"kalimat %s terlalu sedikit — ia tidak menyebut apa yang TIDAK terjadi", kind)

		// Baris antrean Pega sengaja dibiarkan, dan itu wajib disebut — petugas yang
		// membuka Pega akan melihatnya dan mengira perpindahannya gagal.
		require.Containsf(t, pesan, "Pega",
			"kalimat %s wajib menyebut baris antrean Pega yang masih tergambar", kind)
	}
}

// TestKalimatKirimTidakMemakaiKataBerhasil menjaga agar kalimatnya menyebut AKIBAT.
//
// "Berhasil" menjawab pertanyaan yang tidak ditanyakan petugas. Yang ia perlu tahu adalah ke
// mana klaimnya pergi, dan apa yang belum terjadi.
func TestKalimatKirimTidakMemakaiKataBerhasil(t *testing.T) {
	for _, kind := range []inboxrclpucl.ClaimActionKind{
		inboxrclpucl.ActionSendToAnalyst,
		inboxrclpucl.ActionSendToPICTeknik,
	} {
		require.NotContains(t, strings.ToLower(actionDoneMessage(kind)), "berhasil")
	}
}

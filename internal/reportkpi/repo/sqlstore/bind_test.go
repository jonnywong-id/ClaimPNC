package sqlstore

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

var penandaBind = regexp.MustCompile(`:(\d+)`)

// Tidak satu pun kueri MENGULANG penanda bind.
//
// # Kegagalan yang ini cegah
//
// Driver Oracle di sini mengikat menurut **urutan kemunculan** penanda, bukan menurut
// nomornya. `(:1 IS NULL OR k.TIPE = :1)` karena itu memakan DUA nilai, bukan satu.
//
// Pada 2026-10-09 kueri `summary` dan `detail` tab KPI Adjuster punya dua pasang penanda
// berulang. Keduanya mengirim empat nilai untuk enam dan delapan kemunculan, dan Oracle
// menolaknya dengan `ORA-01008: not all variables bound`. Yang sampai ke pengguna hanya
// "Terjadi kesalahan pada sistem" — seluruh tab tidak dapat dipakai.
//
// # Kenapa uji sqlmock tidak menangkapnya
//
// Tiruan itu mencocokkan jumlah argumen dengan yang DIHARAPKAN uji, bukan dengan yang
// dituntut teks SQL-nya. Kueri yang kekurangan bind tetap hijau di sana, dan baru gagal
// terhadap Oracle sungguhan.
//
// Uji ini memeriksa TEKS-nya, sehingga tidak bergantung pada driver maupun tiruan.
func TestTidakAdaPenandaBindBerulang(t *testing.T) {
	for name, text := range queries {
		hitung := map[string]int{}
		for _, m := range penandaBind.FindAllStringSubmatch(text, -1) {
			hitung[m[1]]++
		}

		for nomor, kali := range hitung {
			require.Equalf(t, 1, kali,
				"kueri %q memakai penanda :%s sebanyak %d kali.\n"+
					"Driver mengikat menurut urutan kemunculan, jadi tiap kemunculan "+
					"memakan satu nilai. Beri nomor sendiri pada tiap kemunculan, lalu "+
					"kirim nilainya berulang dari sisi Go.",
				name, nomor, kali)
		}
	}
}

// Penomoran penanda RAPAT dari :1 — tanpa lompatan.
//
// Nomor yang bolong menandakan satu penanda terhapus tanpa penomoran berikutnya ikut
// disesuaikan. Akibatnya sama dengan pengulangan: jumlah nilai yang dikirim tidak lagi
// cocok dengan jumlah kemunculan, dan galatnya baru muncul di tangan pengguna.
func TestPenomoranPenandaBindRapat(t *testing.T) {
	for name, text := range queries {
		ada := map[int]bool{}
		tertinggi := 0
		for _, m := range penandaBind.FindAllStringSubmatch(text, -1) {
			var n int
			_, err := fmt.Sscanf(m[1], "%d", &n)
			require.NoError(t, err)
			ada[n] = true
			if n > tertinggi {
				tertinggi = n
			}
		}
		for n := 1; n <= tertinggi; n++ {
			require.Truef(t, ada[n],
				"kueri %q memakai sampai :%d tetapi melewatkan :%d", name, tertinggi, n)
		}
	}
}

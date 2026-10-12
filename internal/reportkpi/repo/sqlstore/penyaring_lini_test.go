package sqlstore

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/reportkpi"
)

// Penyaring lini bisnis hanya boleh menempel pada kueri yang tabelnya MEMILIKI kolomnya.
//
// # Kenapa uji ini ada
//
// Ia mengunci cacat nyata yang ditemukan 2026-10-08: tombol Cari menjawab
// "Penilaian gagal diambil dari basis data" untuk setiap pencarian.
//
// Sebabnya, penanda /*LINE_FILTER*/ dipasang pada EMPAT kueri, padahal penyaringnya
// menyebut `a.GROUP_PANEL` dan `a.GROUPBISNISID` — dan alias `a` menunjuk tabel yang
// BERBEDA di tiap kueri:
//
//	progress_counts    a = POOLDATA.PEGA_DASHBOARDPNC   punya kedua kolom  ✓
//	analysis_spans     a = POOLDATA.T_CLAIM_PNC         TIDAK punya        ✗
//	acceptance_spans   a = POOLDATA.T_CLAIM_PNC         TIDAK punya        ✗
//	closure_spans      a = POOLDATA.T_CLAIM_PNC         TIDAK punya        ✗
//
// `T_CLAIM_PNC` memakai `GROUPPANEL` (tanpa garis bawah) dan `BUSINESSCODE`. Ketiga kueri
// itu karena itu selalu gagal dengan `ORA-00904`.
//
// Yang membuatnya lolos begitu lama: probe `-periksa` menjalankan kueri DASAR, sedangkan
// penyaringnya baru disisipkan saat melayani permintaan. Kolom yang hanya muncul di
// penyaring tidak pernah ikut diuji.
//
// Pega sendiri hanya memasang penyaring lini pada satu kueri —
// `{ASIS:TempBisnis.GROUP_PANEL}` di `RDB List/GetProgressForKPIPIC-SQL.xml`. Ketiga rule
// lainnya (`DataAnalisaPIC`, `DataAkseptasiPIC`, `GetDataClosePIC`) menyaring dengan
// `picteknik` dan tanggal saja. Jadi membuangnya bukan hanya memperbaiki galat — ia
// MENDEKATKAN hasilnya ke Pega (`P-5`).
func TestPenandaPenyaringLiniHanyaPadaKueriYangKolomnyaAda(t *testing.T) {
	// Daftar TERTUTUP, sengaja ditulis tangan. Menghitungnya otomatis dari berkas .sql
	// akan membuat uji ini ikut berubah setiap kali penandanya dipindahkan — yaitu persis
	// perubahan yang seharusnya ia tolak.
	boleh := map[string]bool{"progress_counts": true}

	for _, nama := range []string{
		"pic_list", "bands", "threshold_days",
		"progress_counts", "analysis_spans", "acceptance_spans", "closure_spans",
	} {
		punya := strings.Contains(query(nama), lineFilterMarker)
		require.Equalf(t, boleh[nama], punya,
			"kueri %q: penanda penyaring lini %v, seharusnya %v — penyaringnya menyebut "+
				"GROUP_PANEL dan GROUPBISNISID, yang hanya ada di PEGA_DASHBOARDPNC",
			nama, punya, boleh[nama])
	}
}

// Kueri bertabel T_CLAIM_PNC tidak boleh menyebut kolom milik PEGA_DASHBOARDPNC.
//
// Uji di atas menjaga PENANDANYA; uji ini menjaga ISINYA. Keduanya diperlukan karena kolom
// itu dapat masuk tanpa lewat penanda — misalnya bila seseorang menuliskannya langsung ke
// dalam kueri.
func TestKueriTClaimPNCTidakMenyebutKolomDashboard(t *testing.T) {
	for _, nama := range []string{"analysis_spans", "acceptance_spans", "closure_spans"} {
		teks := strings.ToUpper(query(nama))
		require.Containsf(t, teks, "T_CLAIM_PNC",
			"uji ini mengandaikan kueri %q membaca T_CLAIM_PNC", nama)
		require.NotContainsf(t, teks, "GROUP_PANEL",
			"kueri %q membaca T_CLAIM_PNC, yang kolomnya bernama GROUPPANEL — tanpa garis bawah", nama)
		require.NotContainsf(t, teks, "GROUPBISNISID",
			"kueri %q membaca T_CLAIM_PNC, yang tidak punya kolom itu; padanannya BUSINESSCODE", nama)
	}
}

// picQuery menolak kueri yang tidak punya penanda, alih-alih diam-diam tidak menyaring.
//
// Tanpa penolakan ini, memanggil picQuery pada kueri tanpa penanda akan mengembalikan
// kueri apa adanya — penyaringnya hilang tanpa satu pun tanda, dan laporannya memuat baris
// lini lain. Kegagalan yang diam seperti itu persis yang `D-49` berulang kali perbaiki.
func TestPicQueryMenolakKueriTanpaPenanda(t *testing.T) {
	require.Panics(t, func() {
		picQuery("closure_spans", reportkpi.LineNonMBU)
	}, "kueri tanpa penanda harus ditolak, bukan dikembalikan tanpa penyaring")

	require.NotPanics(t, func() {
		picQuery("progress_counts", reportkpi.LineNonMBU)
	})
}

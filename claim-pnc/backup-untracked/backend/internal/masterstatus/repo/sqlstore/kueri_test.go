package sqlstore

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Seluruh kueri yang dipanggil kode harus benar-benar ada di berkas .sql. Tanpa uji ini,
// salah ketik nama kueri baru ketahuan saat pengguna memanggil endpointnya.
func TestSeluruhKueriYangDipakaiAda(t *testing.T) {
	dipakai := []string{
		"status_klaim_daftar",
		"status_klaim_ambil",
		"status_klaim_situs",
		"status_klaim_urutan_berikutnya",
		"status_klaim_sisip",
		"status_klaim_perbarui",
		"status_klaim_periksa_tabel",
	}
	for _, nama := range dipakai {
		t.Run(nama, func(t *testing.T) {
			require.NotPanics(t, func() { _ = ambilKueri(nama) })
			require.NotEmpty(t, strings.TrimSpace(ambilKueri(nama)))
		})
	}
}

func TestKueriYangTidakAdaMenimbulkanPanik(t *testing.T) {
	require.Panics(t, func() { _ = ambilKueri("kueri_yang_tidak_pernah_ada") })
}

// Disiplin SQL portabel (D-20) hanya bertahan bila ditegakkan perkakas, bukan diingat
// orang. Uji ini adalah penegaknya sampai pemeriksaan pola SQL berjalan di CI.
func TestKueriMematuhiDisiplinSQLPortabel(t *testing.T) {
	terlarang := map[string]string{
		"SELECT *": "kolom harus disebut namanya; kolom baru tidak boleh diam-diam mengubah perilaku",
		"NVL(":     "pakai COALESCE",
		"SYSDATE":  "pakai CURRENT_TIMESTAMP",
		"DECODE(":  "pakai CASE WHEN",
		"ROWNUM":   "pakai OFFSET ... FETCH NEXT ... ROWS ONLY",
		"INSTR(":   "pakai POSITION",
		"LISTAGG(": "pakai STRING_AGG",
		"TO_CHAR(": "pemformatan tanggal dan angka dilakukan di Go",
		"LPAD(":    "pemformatan angka dilakukan di Go",
	}

	for nama, teks := range kueri {
		hurufBesar := strings.ToUpper(teks)
		for pola, alasan := range terlarang {
			require.NotContainsf(t, hurufBesar, pola,
				"kueri %q memakai %q — %s", nama, pola, alasan)
		}
		require.NotContainsf(t, hurufBesar, "(+)",
			"kueri %q memakai outer join gaya Oracle; pakai LEFT JOIN", nama)
	}
}

// FROM DUAL dilarang di mana pun KECUALI satu kueri.
//
// Pengecualian ini disengaja dan terbatas: NEXTVAL menuntutnya, dan memakai urutan yang
// sama dengan procedure lama adalah syarat agar kode yang diterbitkan aplikasi ini tidak
// pernah bertabrakan dengan kode yang pernah diterbitkan Pega. Perlakuannya sama dengan
// generator nomor klaim pada ADR-0005.
//
// Uji ini memagari pengecualian itu supaya ia tidak menyebar: kueri KEDUA yang memakai
// FROM DUAL akan membuat uji ini gagal.
func TestFromDualHanyaDiKueriUrutan(t *testing.T) {
	const dikecualikan = "status_klaim_urutan_berikutnya"

	for nama, teks := range kueri {
		if nama == dikecualikan {
			require.Contains(t, strings.ToUpper(teks), "FROM DUAL",
				"kueri urutan memang harus memakainya; bila tidak lagi, hapus pengecualiannya")
			continue
		}
		require.NotContainsf(t, strings.ToUpper(teks), "FROM DUAL",
			"kueri %q memakai FROM DUAL; hanya %q yang dibenarkan", nama, dikecualikan)
	}
}

// Nilai selalu lewat parameter binding. Kueri yang merangkai nilai ke dalam teks SQL
// adalah celah injeksi — pola yang diwarisi sistem lama lewat {ASIS:...}, 538 kemunculan.
func TestKueriMemakaiParameterBinding(t *testing.T) {
	berparameter := []string{
		"status_klaim_ambil",
		"status_klaim_sisip",
		"status_klaim_perbarui",
	}
	for _, nama := range berparameter {
		require.Containsf(t, ambilKueri(nama), ":1",
			"kueri %q harus memakai parameter binding", nama)
	}
}

// Modul ini menulis, dan yang ditulisnya adalah tabel milik sistem lama. Tidak ada satu
// pun jalur yang boleh MENGHAPUS baris master: layar Pega tidak punya tombol hapus, dan
// ADR-0012 melarang master dihapus permanen karena klaim lama merujuknya.
func TestTidakAdaKueriYangMenghapus(t *testing.T) {
	for nama, teks := range kueri {
		hurufBesar := strings.ToUpper(teks)
		require.NotContainsf(t, hurufBesar, "DELETE", "kueri %q menghapus baris", nama)
		require.NotContainsf(t, hurufBesar, "TRUNCATE", "kueri %q mengosongkan tabel", nama)
		require.NotContainsf(t, hurufBesar, "DROP ", "kueri %q membuang objek basis data", nama)
	}
}

// Kueri tulis hanya boleh menyentuh M_STS_CLAIM. Menyentuh V_STS_CLAIM akan menulis ke
// view yang dibaca 23 rule Pega, dan menyentuh tabel lain melanggar P-1.
func TestKueriTulisHanyaMenyentuhTabelDasar(t *testing.T) {
	for _, nama := range []string{"status_klaim_sisip", "status_klaim_perbarui"} {
		teks := strings.ToUpper(ambilKueri(nama))
		require.Contains(t, teks, "POOLDATA.M_STS_CLAIM",
			"kueri %q harus menulis ke tabel dasarnya", nama)
		require.NotContains(t, teks, "V_STS_CLAIM",
			"kueri %q tidak boleh menulis lewat view yang dibaca Pega", nama)
	}
}

// Kode dibentuk id_site || lpad(urutan, 3, '0'). Uji ini mengunci pembentukannya,
// termasuk perilaku yang SENGAJA tidak diperbaiki di atas 999.
func TestTigaDigitMeniruLpadOracle(t *testing.T) {
	require.Equal(t, "001", TigaDigit(1))
	require.Equal(t, "099", TigaDigit(99))
	require.Equal(t, "134", TigaDigit(134), "kode pertama pada master hari ini: 1 + 134 = 1134")
	require.Equal(t, "166", TigaDigit(166), "kode terakhir pada master hari ini")
	require.Equal(t, "999", TigaDigit(999))

	// LPAD Oracle tidak memotong. Kode ke-1000 memang menjadi lima karakter, dan itu
	// cacat skema warisan yang sengaja tidak ditutupi: memotongnya akan menghasilkan
	// kode ganda, yang jauh lebih buruk daripada kode yang kepanjangan.
	require.Equal(t, "1000", TigaDigit(1000),
		"perilaku ini diwarisi apa adanya; bila diubah, ubah juga catatannya di README")
}

// Nama indeks unik dipakai dua tempat: migrasi 0002 membuatnya, dan kode Go
// menerjemahkan galat bentrok dengan mencocokkan namanya. Bila keduanya berbeda, bentrok
// label akan muncul sebagai galat 500 di layar.
func TestNamaIndeksLabelSesuaiMigrasi(t *testing.T) {
	require.Equal(t, "UX_M_STS_CLAIM_LABEL", NamaIndeksLabel,
		"harus sama persis dengan nama indeks pada backend/migrations/0002")
}

// Nama kunci utama BUKAN tebakan. Ia dibaca dari ALL_CONSTRAINTS pada 2026-09-17, dan
// ternyata terbalik dari pola penamaan yang dipakai migrasi 0001 — `M_STS_CLAIM_PK`,
// bukan `PK_M_STS_CLAIM`. Salah nama membuat kode bentrok muncul sebagai galat 500.
func TestNamaKunciUtamaSesuaiBasisData(t *testing.T) {
	require.Equal(t, "M_STS_CLAIM_PK", NamaKunciUtama,
		"diverifikasi dari ALL_CONSTRAINTS; jangan diubah tanpa membaca ulang katalog")
}

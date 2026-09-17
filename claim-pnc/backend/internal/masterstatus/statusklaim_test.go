package masterstatus_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterstatus"
	"claim-pnc/internal/masterstatus/repo/memori"
)

// Acceptance criteria TKT-F4-005: "Master status memuat TEPAT 33 kode 1134–1166 beserta
// labelnya — dihitung dan dilaporkan angkanya."
func TestMasterMemuatTepat33Kode(t *testing.T) {
	daftar := memori.DaftarContoh()
	require.Len(t, daftar, 33, "V_STS_CLAIM memuat 33 kode; R-06 ditutup dengan angka ini")

	require.Equal(t, "1134", daftar[0].Kode, "kode terkecil")
	require.Equal(t, "1166", daftar[32].Kode, "kode terbesar")
}

// Kode pada SALINAN CSV berurutan tanpa lompatan, karena ia dibentuk
// id_site || lpad(urutan, 3, '0'). Lompatan berarti ada baris yang hilang dari salinan
// master di repo ini.
//
// Basis data yang berjalan justru PUNYA lompatan — lihat uji berikutnya.
func TestKodePadaSalinanCSVBerurutanTanpaLompatan(t *testing.T) {
	daftar := memori.DaftarContoh()

	for i, s := range daftar {
		diharapkan := 1134 + i
		require.Equal(t, itoa(diharapkan), s.Kode,
			"kode ke-%d seharusnya %d; deretnya tidak boleh berlubang", i+1, diharapkan)
	}
}

// Selisih antara CSV yang diserahkan Work Owner dan basis data yang berjalan.
//
// Pembacaan langsung POOLDATA.M_STS_CLAIM pada 2026-09-17 menemukan 32 baris, bukan 33:
// kode 1165 tidak ada di sana. Sebabnya belum dijelaskan.
//
// Uji ini tidak menguji kebenaran apa pun — ia MENGUNCI FAKTA supaya tidak hilang
// diam-diam. Bila kelak 1165 dihapus dari daftar contoh karena "produksi memang tidak
// punya", uji ini gagal dan memaksa pertanyaannya dibicarakan lebih dulu.
func TestSelisihDenganBasisDataProduksiTercatat(t *testing.T) {
	const kodeYangHanyaAdaDiCSV = "1165"

	ada := false
	for _, s := range memori.DaftarContoh() {
		if s.Kode == kodeYangHanyaAdaDiCSV {
			ada = true
			require.Equal(t, "Rejected Chasier", s.Label)
		}
	}
	require.True(t, ada,
		"kode %s ada di Database/v_sts_claim.csv tetapi TIDAK ada di POOLDATA.M_STS_CLAIM "+
			"per 2026-09-17; selisihnya belum dijelaskan Work Owner dan tidak boleh dihapus "+
			"dari daftar ini tanpa jawaban", kodeYangHanyaAdaDiCSV)
}

// Acceptance criteria TKT-F4-005: "Sebelas kode pertama (1134–1144) menyimpan penomoran
// lama 01–11 sebagai kolom terpisah, sehingga data historis tetap terbaca."
func TestSebelasKodePertamaMembawaPenomoranLama(t *testing.T) {
	daftar := memori.DaftarContoh()

	for i := 0; i < 11; i++ {
		require.NotEmpty(t, daftar[i].KodeLama,
			"kode %s seharusnya membawa penomoran lama", daftar[i].Kode)
		require.Equal(t, nolDiDepan(i+1), daftar[i].KodeLama,
			"penomoran lama kode %s", daftar[i].Kode)
	}
	for i := 11; i < len(daftar); i++ {
		require.Empty(t, daftar[i].KodeLama,
			"kode %s tidak pernah punya penomoran lama", daftar[i].Kode)
	}
}

// Arti kode TIDAK BOLEH disimpulkan dari pemakaiannya di rule. Tiga kesimpulan yang
// pernah diambil begitu terbukti salah seluruhnya (R-06); uji ini mengunci arti yang
// sebenarnya supaya kesalahan itu tidak dapat masuk kembali lewat perubahan data contoh.
func TestArtiTigaKodeYangPernahSalahDisimpulkan(t *testing.T) {
	arti := map[string]string{}
	for _, s := range memori.DaftarContoh() {
		arti[s.Kode] = s.Label
	}

	require.Equal(t, "Close Claim for this object", arti["1143"], "1143 BUKAN status awal")
	require.Equal(t, "LOD Report", arti["1150"], "1150 BUKAN penanda terdaftar")
	require.Equal(t, "Analyst", arti["1151"], "1151 BUKAN Investigator")
}

// Label kosong ditolak. Ini perbedaan yang DISENGAJA dari sistem lama, yang menandai
// isian ini pyRequired=false — dan label kosong akan tampil sebagai status kosong di 23
// rule Pega yang membaca V_STS_CLAIM.
func TestLabelKosongDitolak(t *testing.T) {
	for _, masukan := range []string{"", " ", "\t", "   \n  "} {
		pelanggaran := masterstatus.PeriksaLabel(masukan)
		require.Len(t, pelanggaran, 1, "masukan %q seharusnya melanggar tepat satu aturan", masukan)
		require.Equal(t, masterstatus.FieldLabel, pelanggaran[0].Field,
			"pelanggaran harus menunjuk kolomnya, supaya layar dapat menandainya")
	}
}

func TestLabelTerisiDiterima(t *testing.T) {
	require.Empty(t, masterstatus.PeriksaLabel("Paid"))
	require.Empty(t, masterstatus.PeriksaLabel("  Close Claim for this object  "),
		"spasi tepi dibuang sebelum diperiksa")
}

func TestLabelTerlaluPanjangDitolak(t *testing.T) {
	pas := strings.Repeat("a", masterstatus.PanjangLabelMaksimum)
	require.Empty(t, masterstatus.PeriksaLabel(pas), "tepat di batas masih diterima")

	lebih := strings.Repeat("a", masterstatus.PanjangLabelMaksimum+1)
	require.Len(t, masterstatus.PeriksaLabel(lebih), 1, "satu karakter di atas batas ditolak")
}

// Panjang dihitung dalam rune, bukan byte. Satu huruf beraksen memakan dua byte, dan
// menghitung byte akan membuat batasnya terasa berubah-ubah bagi pengguna.
func TestPanjangLabelDihitungDalamRuneBukanByte(t *testing.T) {
	// 100 rune, 200 byte.
	seratusRune := strings.Repeat("é", masterstatus.PanjangLabelMaksimum)
	require.Len(t, []byte(seratusRune), 2*masterstatus.PanjangLabelMaksimum, "prasyarat uji")

	require.Empty(t, masterstatus.PeriksaLabel(seratusRune),
		"100 huruf beraksen masih 100 karakter, bukan 200")
}

// Pelanggaran dikumpulkan SELURUHNYA, tidak berhenti pada yang pertama — meniru perilaku
// sistem lama yang menampilkan semua pesan validasi sekaligus.
func TestSeluruhPelanggaranDikembalikanSekaligus(t *testing.T) {
	// Satu masukan yang melanggar dua aturan tidak mungkin dibuat di sini: kosong dan
	// kepanjangan saling meniadakan. Yang diuji adalah bentuk galatnya membawa senarai,
	// bukan satu pesan.
	err := masterstatus.GalatValidasiBaru([]masterstatus.Pelanggaran{
		{Field: masterstatus.FieldLabel, Pesan: "satu"},
		{Field: masterstatus.FieldLabel, Pesan: "dua"},
	})
	require.Error(t, err)

	var validasi *masterstatus.GalatValidasi
	require.ErrorAs(t, err, &validasi)
	require.Len(t, validasi.Pelanggaran, 2)
}

func TestGalatValidasiKosongBenarBenarNil(t *testing.T) {
	err := masterstatus.GalatValidasiBaru(nil)
	require.NoError(t, err, "nil bertipe harus benar-benar nil supaya if err != nil terbaca apa adanya")
}

// "Paid" dan "PAID  " adalah status yang sama bagi pengguna. Ekspresi yang sama dipakai
// indeks unik di basis data (UPPER(TRIM(LSC_NOTE))), sehingga keduanya tidak dapat
// berbeda pendapat.
func TestKunciLabelMengabaikanBesarKecilHurufDanSpasi(t *testing.T) {
	require.Equal(t, masterstatus.KunciLabel("Paid"), masterstatus.KunciLabel("PAID"))
	require.Equal(t, masterstatus.KunciLabel("Paid"), masterstatus.KunciLabel("  paid  "))
	require.NotEqual(t, masterstatus.KunciLabel("Paid"), masterstatus.KunciLabel("Unpaid"))
}

// OLD_LSC_ID tersimpan sebagai CHAR berisi padding ("01  "). Spasi itu tidak pernah
// dimaksudkan sebagai bagian nilainya.
func TestBersihMembuangPaddingKolomCHAR(t *testing.T) {
	rapi := masterstatus.StatusKlaim{
		Kode:     " 1134 ",
		Label:    "  Abbreviated Report  ",
		KodeLama: "01  ",
	}.Bersih()

	require.Equal(t, "1134", rapi.Kode)
	require.Equal(t, "Abbreviated Report", rapi.Label)
	require.Equal(t, "01", rapi.KodeLama)
}

// Label pada master yang ada semuanya unik. Bila kelak tidak, indeks unik migrasi 0002
// akan gagal dibuat — dan itu harus diketahui dari sini, bukan dari DBA saat migrasi.
func TestSeluruhLabelPadaMasterUnik(t *testing.T) {
	terlihat := map[string]string{}
	for _, s := range memori.DaftarContoh() {
		kunci := masterstatus.KunciLabel(s.Label)
		sebelumnya, ada := terlihat[kunci]
		require.False(t, ada, "label %q dipakai dua kali: kode %s dan %s", s.Label, sebelumnya, s.Kode)
		terlihat[kunci] = s.Kode
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

func nolDiDepan(n int) string {
	s := itoa(n)
	if len(s) < 2 {
		return "0" + s
	}
	return s
}

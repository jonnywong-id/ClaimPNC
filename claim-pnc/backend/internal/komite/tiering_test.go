package komite_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/komite"
	"claim-pnc/internal/komite/repo/memory"
	"claim-pnc/internal/platform/money"
)

// rp memendekkan pembentukan nilai uang di berkas uji.
func rp(rupiah int64) money.Money { return money.FromRupiah(rupiah) }

// tentukan menjalankan mesin penjenjangan atas master yang berlaku hari ini.
func tentukan(t *testing.T, nilai money.Money, lini komite.BusinessLine) komite.Tiering {
	t.Helper()
	hasil, err := komite.Determine(nilai, lini, memory.SampleThresholds(), komite.DefaultPolicy())
	require.NoError(t, err)
	return hasil
}

// TestJenjangKumulatif adalah pengujian yang paling menentukan di modul ini.
//
// Ketujuh kasusnya diambil apa adanya dari tabel pada
// `docs/ticketing/B-7-Komite-Persetujuan-Klaim/spec.md`, dan dijalankan terhadap isi
// master yang sebenarnya (`Database/emailkomite.csv`). Bila salah satunya meleset, yang
// salah bukan angkanya melainkan aturannya — dan akibatnya adalah klaim disetujui oleh
// jumlah orang yang keliru.
func TestJenjangKumulatif(t *testing.T) {
	kasus := []struct {
		nama  string
		nilai money.Money
		lini  komite.BusinessLine
		mau   int
	}{
		{"PA Rp 5.000.000", rp(5_000_000), "PA", 1},
		{"PA Rp 75.000.000", rp(75_000_000), "PA", 3},
		{"PA Rp 150.000.000", rp(150_000_000), "PA", 4},
		{"Travel Rp 150.000.000", rp(150_000_000), "TRAVEL", 3},
		{"Non-MBU Rp 80.000.000", rp(80_000_000), "NONMBU", 2},
		{"Non-MBU Rp 750.000.000", rp(750_000_000), "NONMBU", 2},
		{"Non-MBU Rp 2.000.000.000", rp(2_000_000_000), "NONMBU", 3},
	}

	for _, k := range kasus {
		t.Run(k.nama, func(t *testing.T) {
			hasil := tentukan(t, k.nilai, k.lini)
			require.Equal(t, k.mau, hasil.TierCount(),
				"jumlah penyetuju untuk %s", k.nama)
		})
	}
}

// Jumlah penyetuju harus BERTAMBAH seiring besarnya klaim, tidak pernah berkurang.
// Inilah sifat yang membedakan aturan kumulatif dari pemilihan satu baris, dan ia
// diuji sebagai sifat — bukan hanya lewat contoh yang kebetulan cocok.
func TestJumlahPenyetujuTidakPernahBerkurangSeiringNilai(t *testing.T) {
	// Non-MBU sengaja dikecualikan: pita membuatnya BENAR-BENAR berkurang saat
	// menyeberangi Rp 100.000.000, dan itu perilaku yang disengaja — lihat
	// TestPitaNonMBUMenurunkanJumlahPenyetujuSaatMenyeberang.
	for _, lini := range []komite.BusinessLine{"PA", "TRAVEL"} {
		t.Run(string(lini), func(t *testing.T) {
			sebelumnya := 0
			for _, nilai := range []money.Money{
				rp(0), rp(10_000_000), rp(10_000_001), rp(50_000_000), rp(50_000_001),
				rp(100_000_000), rp(100_000_001), rp(200_000_000), rp(5_000_000_000),
			} {
				jumlah := tentukan(t, nilai, lini).TierCount()
				require.GreaterOrEqual(t, jumlah, sebelumnya,
					"jumlah penyetuju turun pada nilai %s", nilai)
				sebelumnya = jumlah
			}
		})
	}
}

// Ambang bawah diuji TEPAT DI BATASNYA. Di sinilah aturan paling sering salah, dan
// selisih satu rupiah di sini menambah atau menghapus satu jenjang persetujuan.
func TestBatasBawahDiujiTepatDiBatas(t *testing.T) {
	kasus := []struct {
		nama  string
		nilai money.Money
		lini  komite.BusinessLine
		mau   int
	}{
		{"PA tepat di Rp 10.000.000 belum menambah jenjang", rp(10_000_000), "PA", 1},
		{"PA satu rupiah di atasnya menambah jenjang", rp(10_000_001), "PA", 2},
		{"PA tepat di Rp 50.000.000", rp(50_000_000), "PA", 2},
		{"PA satu rupiah di atas Rp 50.000.000", rp(50_000_001), "PA", 3},
		{"Travel tepat di Rp 50.000.000", rp(50_000_000), "TRAVEL", 1},
		{"Travel satu rupiah di atasnya", rp(50_000_001), "TRAVEL", 2},
	}

	for _, k := range kasus {
		t.Run(k.nama, func(t *testing.T) {
			require.Equal(t, k.mau, tentukan(t, k.nilai, k.lini).TierCount())
		})
	}
}

// Pita hanya berlaku di Non-MBU, dan akumulasi tidak pernah menyeberang antarpita.
func TestPitaHanyaNonMBU(t *testing.T) {
	t.Run("Non-MBU memakai pita", func(t *testing.T) {
		bawah := tentukan(t, rp(80_000_000), "NONMBU")
		require.True(t, bawah.UsesBand)
		require.Equal(t, komite.BandLower, bawah.Band)
		require.Equal(t, 2, bawah.TierCount())

		atas := tentukan(t, rp(200_000_000), "NONMBU")
		require.True(t, atas.UsesBand)
		require.Equal(t, komite.BandUpper, atas.Band)
		// Hanya ID 2 yang cocok di pita atas pada nilai ini.
		require.Equal(t, 1, atas.TierCount())
	})

	// Inilah pengujian yang menjaga kesalahan paling merusak tidak terulang.
	// Memberlakukan pita ke PA dan Travel akan membuat kedua kasus di bawah menjadi
	// NOL penyetuju — klaimnya mandek.
	t.Run("PA dan Travel tanpa pita", func(t *testing.T) {
		pa := tentukan(t, rp(150_000_000), "PA")
		require.False(t, pa.UsesBand)
		require.Empty(t, pa.Band)
		require.Equal(t, 4, pa.TierCount(), "PA Rp 150 juta harus 4 penyetuju, bukan 2")

		kecil := tentukan(t, rp(5_000_000), "PA")
		require.Equal(t, 1, kecil.TierCount(), "PA Rp 5 juta harus 1 penyetuju, bukan 0")

		travel := tentukan(t, rp(150_000_000), "TRAVEL")
		require.False(t, travel.UsesBand)
		require.Equal(t, 3, travel.TierCount(), "Travel Rp 150 juta harus 3 penyetuju, bukan 0")
	})
}

// Menyeberangi batas pita MENURUNKAN jumlah penyetuju dari 2 menjadi 1, dan itu
// disengaja: komitenya berganti badan, bukan bertambah anggota (`D-52`).
//
// Diuji supaya perilaku ini tidak terbaca sebagai cacat saat uji kesetaraan kelak.
func TestPitaNonMBUMenurunkanJumlahPenyetujuSaatMenyeberang(t *testing.T) {
	tepatDiBatas := tentukan(t, rp(100_000_000), "NONMBU")
	require.Equal(t, komite.BandLower, tepatDiBatas.Band)
	require.Equal(t, 2, tepatDiBatas.TierCount())

	satuRupiahDiAtasnya := tentukan(t, rp(100_000_001), "NONMBU")
	require.Equal(t, komite.BandUpper, satuRupiahDiAtasnya.Band)
	require.Equal(t, 1, satuRupiahDiAtasnya.TierCount())
}

// LIMIT_TOP tidak boleh menyaring satu baris pun.
//
// Bila ia dipakai menyaring, setiap kasus di bawah akan berubah menjadi TEPAT SATU
// penyetuju — dan penjenjangan yang menjadi inti `D-14` hilang seluruhnya (`D-47`).
func TestLimitTopTidakMenyaring(t *testing.T) {
	// PA Rp 150 juta berada DI LUAR rentang tiga baris pertama (0–10jt, 10jt–50jt,
	// 50jt–100jt) namun tetap harus menyertakan ketiganya.
	pa := tentukan(t, rp(150_000_000), "PA")
	require.Equal(t, 4, pa.TierCount())

	// Nilai jauh di atas atap tangga tetap mendapat SELURUH penyetuju, bukan nol.
	// Ini sekaligus mengoreksi kalimat pada TKT-B07-001 yang menyebut klaim di atas
	// Rp 200.000.000 "tidak punya penyetuju sama sekali".
	jauhDiAtasAtap := tentukan(t, rp(5_000_000_000), "PA")
	require.Equal(t, 4, jauhDiAtasAtap.TierCount())

	travel := tentukan(t, rp(5_000_000_000), "TRAVEL")
	require.Equal(t, 3, travel.TierCount())
}

// Urutan penyetuju mengikuti DEGREE, dan Urutan selalu berurutan tanpa lompatan.
func TestUrutanMengikutiJenjang(t *testing.T) {
	hasil := tentukan(t, rp(150_000_000), "PA")
	require.Equal(t, 4, hasil.TierCount())

	mauJenjang := []int{1, 2, 3, 4}
	mauNama := []string{"Dr. Wahyu", "Dr. Rossa", "Dr. Rudy", "Dumasi"}

	for i, p := range hasil.Approvers {
		require.Equal(t, i+1, p.Order, "Urutan harus berurutan tanpa lompatan")
		require.Equal(t, mauJenjang[i], p.Tier)
		require.Equal(t, mauNama[i], p.Name)
	}

	// DEGREE pada PA seluruhnya berbeda, jadi tidak ada keraguan urutan.
	require.False(t, hasil.AmbiguousOrder)
}

// Non-MBU pita 1 punya DUA baris ber-DEGREE 1. Keadaan itu nyata ada di master, dan
// kueri lama yang hanya `ORDER BY DEGREE` akan mengurutkannya sesuka basis data.
//
// Modul ini mengurutkannya secara pasti — batas bawah lebih kecil lebih dulu — DAN
// melaporkan keraguannya, supaya perbedaan urutan terhadap Pega pada kasus seri tidak
// terbaca sebagai cacat saat uji kesetaraan.
func TestJenjangSeriDiurutkanPastiDanDilaporkan(t *testing.T) {
	hasil := tentukan(t, rp(80_000_000), "NONMBU")

	require.Equal(t, 2, hasil.TierCount())
	require.True(t, hasil.AmbiguousOrder, "dua baris ber-DEGREE 1 harus dilaporkan")

	// Yang berambang lebih rendah menyetujui lebih dulu.
	require.Equal(t, "ELLENSUPRIYATI", hasil.Approvers[0].Name)
	require.Equal(t, rp(0), hasil.Approvers[0].LowerBound)
	require.Equal(t, "INDRA", hasil.Approvers[1].Name)
	require.Equal(t, rp(50_000_001), hasil.Approvers[1].LowerBound)
}

// Hasil yang sama harus keluar setiap kali dihitung, berapa pun urutan baris master
// datang dari penyimpanan. Tanpa sifat ini, gerbang 1 tidak dapat dibandingkan.
func TestHasilTidakBergantungUrutanBarisMaster(t *testing.T) {
	asli := memory.SampleThresholds()

	terbalik := make([]komite.Threshold, 0, len(asli))
	for i := len(asli) - 1; i >= 0; i-- {
		terbalik = append(terbalik, asli[i])
	}

	dariAsli, err := komite.Determine(rp(80_000_000), "NONMBU", asli, komite.DefaultPolicy())
	require.NoError(t, err)
	dariTerbalik, err := komite.Determine(rp(80_000_000), "NONMBU", terbalik, komite.DefaultPolicy())
	require.NoError(t, err)

	require.Equal(t, dariAsli.Approvers, dariTerbalik.Approvers)
}

// Baris yang bukan jenjang persetujuan tidak pernah ikut terhitung.
//
// Dua jenisnya diuji sekaligus: baris tidak aktif, dan baris ber-STS_ADJ kosong yang
// sebenarnya penerima pemberitahuan registrasi — termasuk baris ber-DEGREE 0 yang
// menjadi pertanyaan terbuka pada TKT-B07-001.
func TestBarisBukanJenjangTidakIkutTerhitung(t *testing.T) {
	hasil := tentukan(t, rp(1_000_000), "NONMBU")

	for _, p := range hasil.Approvers {
		require.NotEqual(t, "9", p.ThresholdID, "baris DEGREE 0 ber-STS_ADJ kosong ikut terhitung")
		require.NotEqual(t, "12", p.ThresholdID, "baris tidak aktif ikut terhitung")
		require.NotZero(t, p.Tier, "tidak ada penyetuju yang boleh ber-jenjang 0")
	}

	// Non-MBU pita 1 pada Rp 1.000.000 hanya ID 7 yang cocok.
	require.Equal(t, 1, hasil.TierCount())
	require.Equal(t, "7", hasil.Approvers[0].ThresholdID)
}

func TestLiniTidakDikenalDibedakanDariTanpaPenyetuju(t *testing.T) {
	t.Run("lini tidak ada di master", func(t *testing.T) {
		_, err := komite.Determine(rp(1_000_000), "MARINE",
			memory.SampleThresholds(), komite.DefaultPolicy())
		require.ErrorIs(t, err, komite.ErrUnknownBusinessLine)
	})

	// Ada di master, punya baris, tetapi tidak satu pun jenjangnya cocok. Ini BUKAN
	// galat: ia keadaan data yang harus dilihat Work Owner, dan layar menampilkannya
	// sebagai peringatan yang mencolok.
	t.Run("ada tetapi tidak ada yang cocok", func(t *testing.T) {
		ambang := []komite.Threshold{{
			ID: "99", Name: "Uji", OperatorID: "UJI",
			BusinessLine: "PA", CommitteeType: "1",
			LowerBound: rp(1_000_000_000), UpperBound: rp(2_000_000_000), Tier: 1,
			Active: true, ForAdjustment: true,
		}}

		hasil, err := komite.Determine(rp(1_000), "PA", ambang, komite.DefaultPolicy())
		require.NoError(t, err)
		require.True(t, hasil.NoApprovers())
		require.Zero(t, hasil.TierCount())
	})
}

func TestMasukanCacatDikumpulkanSeluruhnya(t *testing.T) {
	_, err := komite.Determine(money.FromMinorUnits(-1), "  ",
		memory.SampleThresholds(), komite.DefaultPolicy())

	var validasi *komite.ValidationError
	require.ErrorAs(t, err, &validasi)
	require.Len(t, validasi.Violations, 2, "kedua pelanggaran harus dilaporkan sekaligus")

	field := []string{validasi.Violations[0].Field, validasi.Violations[1].Field}
	require.ElementsMatch(t, []string{komite.FieldValue, komite.FieldBusinessLine}, field)
}

// Penulisan lini yang berbeda-beda harus menghasilkan hasil yang sama. Sistem lama
// terbukti tidak konsisten soal kapitalisasi (`docs/Steering/11-SECURITY.md` §3.1).
func TestPenulisanLiniDinormalkan(t *testing.T) {
	for _, tulisan := range []komite.BusinessLine{"pa", "  PA  ", "Pa"} {
		t.Run(string(tulisan), func(t *testing.T) {
			require.Equal(t, 3, tentukan(t, rp(75_000_000), tulisan).TierCount())
		})
	}
}

func TestDaftarLiniDiambilDariDataBukanDaftarTetap(t *testing.T) {
	lini := komite.ListBusinessLines(memory.SampleThresholds())

	// Hanya lini yang punya jenjang persetujuan aktif yang muncul.
	require.Equal(t, []komite.BusinessLine{"BONDING", "NONMBU", "NONMBUAB", "NONMBUC", "PA", "TRAVEL"}, lini)

	// Lini baru yang ditambahkan ke master langsung ikut, tanpa perubahan kode.
	ditambah := append(memory.SampleThresholds(), komite.Threshold{
		ID: "900", Name: "Baru", OperatorID: "BARU", BusinessLine: "MARINE", CommitteeType: "1",
		LowerBound: rp(0), UpperBound: rp(1_000_000), Tier: 1,
		Active: true, ForAdjustment: true,
	})
	require.Contains(t, komite.ListBusinessLines(ditambah), komite.BusinessLine("MARINE"))
}

// Spasi dan baris baru pada data master tidak boleh sampai ke hasil.
//
// Ini bukan kerapian: pada `Database/emailkomite.csv` baris ID 4, isi OPERATOR_ID
// benar-benar diakhiri baris baru, dan membiarkannya membuat pencocokan identitas
// penyetuju gagal tanpa satu pun pesan galat.
func TestSpasiTepiDibuangDariDataMaster(t *testing.T) {
	ambang := []komite.Threshold{{
		ID: " 4 ", Name: " MARTEN ", OperatorID: "MARTENPETRUSLALAMENTIK_1\n",
		BusinessLine: " nonmbu ", CommitteeType: " 1 ",
		LowerBound: rp(0), UpperBound: rp(100_000_000), Tier: 1,
		Active: true, ForAdjustment: true,
	}}

	hasil, err := komite.Determine(rp(1_000_000), "NONMBU", ambang, komite.DefaultPolicy())
	require.NoError(t, err)
	require.Equal(t, 1, hasil.TierCount())
	require.Equal(t, "MARTENPETRUSLALAMENTIK_1", hasil.Approvers[0].OperatorID)
	require.Equal(t, "MARTEN", hasil.Approvers[0].Name)
	require.Equal(t, "4", hasil.Approvers[0].ThresholdID)
}

// Penanda ketidakhadiran dibawa apa adanya dan TIDAK menyaring siapa pun.
//
// Perilakunya belum dirumuskan di mana pun (`TKT-B07-002`), dan menebaknya berarti
// mengarang aturan yang menentukan siapa menyetujui uang.
func TestSedangAbsenDibawaTanpaMenyaring(t *testing.T) {
	ambang := []komite.Threshold{{
		ID: "1", Name: "Uji", OperatorID: "UJI", BusinessLine: "PA", CommitteeType: "1",
		LowerBound: rp(0), UpperBound: rp(100_000_000), Tier: 1,
		Active: true, ForAdjustment: true, Absent: true,
	}}

	hasil, err := komite.Determine(rp(1_000_000), "PA", ambang, komite.DefaultPolicy())
	require.NoError(t, err)
	require.Equal(t, 1, hasil.TierCount(), "penyetuju yang absen tetap terhitung")
	require.True(t, hasil.Approvers[0].Absent, "keadaan absennya tetap dilaporkan")
}

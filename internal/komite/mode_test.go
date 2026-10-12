package komite_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/komite"
	"claim-pnc/internal/komite/repo/memory"
	"claim-pnc/internal/platform/money"
)

// kolamSimasnet membangun keadaan yang digambarkan Work Owner: tiga orang berwenang pada
// tingkat yang sama untuk klaim di bawah Rp 3.000.000.
//
// Datanya SENGAJA dibuat, bukan diturunkan dari `Database/emailkomite.csv`, dan alasannya
// perlu dinyatakan dengan tepat.
//
// Work Owner menegaskan 2026-09-19: **setiap server punya POOLDATA-nya sendiri**, dan
// aplikasi membaca POOLDATA milik server tempat ia berjalan.
//
// Maka `Database/emailkomite.csv` bukan ekspor yang terpotong — ia LENGKAP untuk basis data
// yang diekspornya, yaitu milik server ASM. Baris ber-TYPE_BUSINESS `SIMASNET` dan
// `SIMASNETA` hidup di POOLDATA milik server Simasnet, dan memang tidak seharusnya ada di
// berkas ini.
//
// Itu sekaligus menjelaskan kenapa kueri Simasnet tetap menyaring `TYPE_BUSINESS`: di
// dalam basis datanya sendiri pun masih ada dua varian yang harus dibedakan.
//
// Akibatnya mode ini belum dapat diuji terhadap data sungguhan sampai ekspor dari basis
// data server Simasnet diterima — kuerinya ada di catatan pengembangan §15.5.
func kolamSimasnet() []komite.Threshold {
	baris := func(id, nama, operator string, jenjang int, bawah int64) komite.Threshold {
		return komite.Threshold{
			ID:            id,
			Name:          nama,
			OperatorID:    operator,
			BusinessLine:  "SIMASNET",
			CommitteeType: "1",
			LowerBound:    rp(bawah),
			UpperBound:    rp(3_000_000),
			Tier:          jenjang,
			Active:        true,
			ForAdjustment: true,
		}
	}

	return []komite.Threshold{
		baris("101", "Petugas A", "USERA", 1, 0),
		baris("102", "Petugas B", "USERB", 1, 0),
		baris("103", "Petugas C", "USERC", 1, 0),
		// Jenjang kedua, hanya untuk nilai yang lebih besar. Ia dibawa supaya terbukti
		// bahwa yang dipilih adalah jenjang TERENDAH, bukan sembarang yang cocok.
		baris("104", "Penyelia D", "USERD", 2, 3_000_001),
	}
}

func tentukanSimasnet(t *testing.T, nilai money.Money, opsi komite.Options) komite.Tiering {
	t.Helper()
	hasil, err := komite.DetermineWith(nilai, "SIMASNET", kolamSimasnet(),
		komite.SimasnetPolicy(), opsi)
	require.NoError(t, err)
	return hasil
}

// Inti mode Simasnet: dipilih TEPAT SATU penyetuju, bukan seluruh jenjang yang terlampaui.
//
// Bukti sumbernya satu kueri utuh — `ORDER BY degree, dbms_random.value` dibungkus
// `WHERE rownum = 1` (`RDB List/EmailKomiteBerjenjangSimasnet_sql-SQL.xml`).
func TestModeSimasnetMemilihTepatSatuPenyetuju(t *testing.T) {
	hasil := tentukanSimasnet(t, rp(1_000_000), komite.Options{})

	require.Equal(t, komite.ModeSingleApprover, hasil.Mode)
	require.Equal(t, 1, hasil.TierCount(), "mode ini memilih satu, bukan mengakumulasi")
	require.Len(t, hasil.Candidates, 3, "ketiga orang pada jenjang terendah adalah kandidatnya")
}

// Operator yang menginput DIKECUALIKAN dari kandidat.
//
// Inilah aturan yang paling berharga di mode ini: ia mencegah seseorang menyetujui
// pengajuannya sendiri. Sumbernya `Activity/SetEmailKomiteSimasnet-Act.xml`, yang
// menyusun potongan `"AND OPERATOR_ID!='" + OperatorID.pyUserIdentifier + "'"`.
func TestModeSimasnetMengecualikanPenginput(t *testing.T) {
	hasil := tentukanSimasnet(t, rp(1_000_000), komite.Options{Applicant: "USERA"})

	require.Len(t, hasil.Candidates, 2, "penginput tidak boleh menjadi kandidat")
	require.Equal(t, "USERA", hasil.ExcludedApplicant)

	for _, k := range hasil.Candidates {
		require.NotEqual(t, "USERA", k.OperatorID)
	}
	require.NotEqual(t, "USERA", hasil.Approvers[0].OperatorID,
		"penginput tidak boleh menyetujui pengajuannya sendiri")
}

// Pencocokan identitas harus tahan terhadap perbedaan penulisan.
//
// Bila ia gagal, penginput tetap menjadi kandidat dan dapat terpilih menyetujui
// pengajuannya sendiri — dan kegagalannya TIDAK terlihat, karena hasilnya tetap berupa
// nama yang masuk akal.
func TestPengecualianPenginputTahanTerhadapPenulisan(t *testing.T) {
	for _, tulisan := range []string{"usera", "  USERA  ", "UserA", "USERA\n"} {
		t.Run(tulisan, func(t *testing.T) {
			hasil := tentukanSimasnet(t, rp(1_000_000), komite.Options{Applicant: tulisan})
			require.Len(t, hasil.Candidates, 2)
			for _, k := range hasil.Candidates {
				require.NotEqual(t, "USERA", k.OperatorID)
			}
		})
	}
}

// Yang dipilih adalah jenjang TERENDAH yang memenuhi syarat, bukan sembarang yang cocok.
func TestModeSimasnetMemilihDariJenjangTerendah(t *testing.T) {
	// Rp 3.000.001 memenuhi kedua jenjang, tetapi hanya jenjang 1 yang menjadi kandidat.
	hasil := tentukanSimasnet(t, rp(3_000_001), komite.Options{})

	require.Len(t, hasil.Candidates, 3)
	for _, k := range hasil.Candidates {
		require.Equal(t, 1, k.Tier)
	}
	require.Equal(t, 1, hasil.Approvers[0].Tier)
}

// Pengacakan berada di balik seam, sehingga pengujian dapat menentukan siapa yang
// terpilih sementara produksi tetap mengacak.
//
// Tanpa seam ini, aturan Simasnet tidak dapat diuji sama sekali — hasil yang berbeda tiap
// kali dijalankan tidak dapat dibandingkan dengan apa pun.
func TestPengacakBeradaDiBalikSeam(t *testing.T) {
	for indeks, mau := range map[int]string{0: "USERA", 1: "USERB", 2: "USERC"} {
		hasil := tentukanSimasnet(t, rp(1_000_000),
			komite.Options{Randomizer: komite.FixedRandomizer(indeks)})

		require.Equal(t, mau, hasil.Approvers[0].OperatorID)
	}
}

// Tanpa randomizer yang dipasok, hasilnya TETAP — bukan diam-diam menjadi acak.
//
// Pilihan bawaan ini disengaja: sesuatu yang diam-diam menjadi acak jauh lebih berbahaya
// daripada sesuatu yang diam-diam menjadi tetap, karena yang pertama baru ketahuan saat
// dua orang membandingkan hasil dan menemukannya berbeda.
func TestTanpaPengacakHasilnyaTetap(t *testing.T) {
	pertama := tentukanSimasnet(t, rp(1_000_000), komite.Options{})
	kedua := tentukanSimasnet(t, rp(1_000_000), komite.Options{})

	require.Equal(t, pertama.Approvers, kedua.Approvers)
}

// Bila seluruh kandidat tersingkir karena penginputnya satu-satunya yang berwenang,
// hasilnya adalah "tanpa penyetuju" — bukan penginput yang menyetujui dirinya sendiri.
func TestSemuaKandidatTersingkirMenghasilkanTanpaPenyetuju(t *testing.T) {
	sendirian := []komite.Threshold{{
		ID: "201", Name: "Petugas Tunggal", OperatorID: "USERA",
		BusinessLine: "SIMASNET", CommitteeType: "1",
		LowerBound: rp(0), UpperBound: rp(3_000_000), Tier: 1,
		Active: true, ForAdjustment: true,
	}}

	hasil, err := komite.DetermineWith(rp(1_000_000), "SIMASNET", sendirian,
		komite.SimasnetPolicy(), komite.Options{Applicant: "USERA"})

	require.NoError(t, err)
	require.True(t, hasil.NoApprovers())
	require.Empty(t, hasil.Candidates)
}

// Mode Simasnet TIDAK mengenal pita — kuerinya memang tidak menyaring TYPE_KOMITE.
func TestModeSimasnetTidakMengenalPita(t *testing.T) {
	hasil := tentukanSimasnet(t, rp(1_000_000), komite.Options{})

	require.False(t, hasil.UsesBand)
	require.Empty(t, hasil.Band)
}

// Pengecualian penginput berlaku pada KEDUA mode — Work Owner, 2026-09-18.
//
// Di sistem lama ia hanya ada pada jalur Simasnet. Meluaskannya adalah perubahan perilaku
// yang disengaja, dan akibatnya diuji di sini terhadap master yang sebenarnya.
func TestModeKumulatifJugaMengecualikanPenginput(t *testing.T) {
	tanpaPenginput, err := komite.Determine(rp(80_000_000), "NONMBU",
		memory.SampleThresholds(), komite.DefaultPolicy())
	require.NoError(t, err)
	require.Equal(t, 2, tanpaPenginput.TierCount())

	// ELLENSUPRIYATI adalah jenjang pertama Non-MBU pita 1. Bila dialah yang mengajukan,
	// jumlah penyetujunya BERKURANG SATU.
	denganPenginput, err := komite.DetermineWith(rp(80_000_000), "NONMBU",
		memory.SampleThresholds(), komite.DefaultPolicy(),
		komite.Options{Applicant: "ELLENSUPRIYATI"})
	require.NoError(t, err)

	require.Equal(t, 1, denganPenginput.TierCount(),
		"penginput yang juga anggota komite mengurangi jumlah penyetuju")
	require.Equal(t, "INDRAGUNAWAN", denganPenginput.Approvers[0].OperatorID)
	require.Len(t, denganPenginput.Excluded, 1)
	require.Equal(t, "ELLENSUPRIYATI", denganPenginput.Excluded[0].OperatorID)
}

// Inilah akibat yang paling perlu diketahui sebelum aturan ini dinyalakan.
//
// Pada master yang berlaku, SETIAP lini punya jenjang terendah yang diisi SATU orang saja.
// Bila orang itu yang mengajukan, klaimnya berakhir TANPA PENYETUJU SAMA SEKALI — dan itu
// menghentikan klaim, bukan sekadar mengurangi satu tanda tangan.
//
// Ketujuh kasus di bawah dihitung dari `Database/emailkomite.csv`, bukan dikarang.
func TestPengecualianPenginputDapatMenghabiskanSeluruhPenyetuju(t *testing.T) {
	kasus := []struct {
		lini      komite.BusinessLine
		nilai     int64
		penginput string
	}{
		{"NONMBU", 20_000_000, "ELLENSUPRIYATI"},
		{"NONMBU", 200_000_000, "BAMBANGSETIADJIGUNAWAN"},
		{"NONMBUAB", 1_000_000, "ELLENSUPRIYATI"},
		{"NONMBUC", 1_000_000, "ELLENSUPRIYATI"},
		{"PA", 5_000_000, "WAHYUKRISTANTI"},
		{"TRAVEL", 10_000_000, "RATNAGUSNITASARI"},
		{"BONDING", 1_000_000, "RIZALGREATLIN"},
	}

	for _, k := range kasus {
		t.Run(string(k.lini)+"/"+k.penginput, func(t *testing.T) {
			hasil, err := komite.DetermineWith(rp(k.nilai), k.lini, memory.SampleThresholds(),
				komite.DefaultPolicy(), komite.Options{Applicant: k.penginput})
			require.NoError(t, err)

			require.True(t, hasil.NoApprovers(),
				"kasus ini justru harus kosong — itulah yang perlu diketahui sebelum dinyalakan")
			require.NotEmpty(t, hasil.Excluded,
				"sebabnya harus terlihat, bukan tampak seperti master yang berlubang")
		})
	}
}

// Penginput yang BUKAN anggota komite tidak mengubah apa pun, dan pengecualiannya tidak
// boleh disiratkan seolah-olah berakibat.
func TestPenginputBukanAnggotaKomiteTidakMengubahApaPun(t *testing.T) {
	hasil, err := komite.DetermineWith(rp(80_000_000), "NONMBU", memory.SampleThresholds(),
		komite.DefaultPolicy(), komite.Options{Applicant: "ADMINPNC"})

	require.NoError(t, err)
	require.Equal(t, 2, hasil.TierCount())
	require.Empty(t, hasil.Excluded, "tidak ada yang benar-benar tersingkir")
	require.Equal(t, "ADMINPNC", hasil.ExcludedApplicant,
		"yang diminta dikecualikan tetap dilaporkan, meski tidak berakibat")
}

// Mode bawaan tetap kumulatif bila kebijakan tidak menyebutkannya.
func TestModeBawaanAdalahKumulatif(t *testing.T) {
	require.Equal(t, komite.ModeCumulative, komite.Policy{}.EffectiveMode())
	require.Equal(t, komite.ModeCumulative, komite.DefaultPolicy().EffectiveMode())
	require.Equal(t, komite.ModeSingleApprover, komite.SimasnetPolicy().EffectiveMode())
}

// Baris ber-DEGREE nol TIDAK PERNAH ikut menyetujui — Work Owner, 2026-09-18.
//
// Penyaringnya diuji dengan baris yang SELURUH syarat lainnya terpenuhi, karena pada
// master yang berlaku baris seperti itu sudah tersaring lebih dulu oleh STS_ADJ. Tanpa
// uji ini, aturannya hanya berlaku secara kebetulan.
func TestJenjangNolTidakPernahMenyetujui(t *testing.T) {
	ambang := []komite.Threshold{
		{
			ID: "301", Name: "Sah", OperatorID: "SAH", BusinessLine: "PA", CommitteeType: "1",
			LowerBound: rp(0), UpperBound: rp(50_000_000), Tier: 1,
			Active: true, ForAdjustment: true,
		},
		{
			// Aktif, untuk adjustment, ambangnya terlampaui — hanya jenjangnya nol.
			ID: "302", Name: "Berjenjang Nol", OperatorID: "NOL", BusinessLine: "PA", CommitteeType: "1",
			LowerBound: rp(0), UpperBound: rp(50_000_000), Tier: 0,
			Active: true, ForAdjustment: true,
		},
	}

	hasil, err := komite.Determine(rp(1_000_000), "PA", ambang, komite.DefaultPolicy())
	require.NoError(t, err)

	require.Equal(t, 1, hasil.TierCount())
	require.Equal(t, "301", hasil.Approvers[0].ThresholdID)
}

// Batas pita berbeda antar entitas, dan keduanya ada di rule yang sama.
//
// `Activity/SetEmailKomite-Act.xml` memuat `@If(…>100000000,2,1)` untuk entitas rupiah
// dan `@If(…>7000,2,1)` untuk entitas SMI. Memakai satu angka untuk keduanya akan salah
// dengan selisih sekitar 14.000 kali lipat.
func TestBatasPitaDapatBerbedaAntarEntitas(t *testing.T) {
	smi := komite.Policy{
		Mode: komite.ModeCumulative,
		Bands: map[komite.BusinessLine]komite.BandPolicy{
			komite.BusinessLineNonMBU: {
				Boundary: money.FromRupiah(komite.NonMBUBandBoundarySMI),
				Lower:    komite.BandLower,
				Upper:    komite.BandUpper,
			},
		},
	}

	// Pada entitas SMI, USD 7.000 masih pita bawah dan USD 7.001 sudah pita atas.
	bawah, err := komite.Determine(money.FromRupiah(7_000), "NONMBU", memory.SampleThresholds(), smi)
	require.NoError(t, err)
	require.Equal(t, komite.BandLower, bawah.Band)

	atas, err := komite.Determine(money.FromRupiah(7_001), "NONMBU", memory.SampleThresholds(), smi)
	require.NoError(t, err)
	require.Equal(t, komite.BandUpper, atas.Band)

	// Sedangkan pada entitas rupiah, nilai yang sama masih jauh di pita bawah.
	rupiah, err := komite.Determine(money.FromRupiah(7_001), "NONMBU", memory.SampleThresholds(),
		komite.DefaultPolicy())
	require.NoError(t, err)
	require.Equal(t, komite.BandLower, rupiah.Band)
}

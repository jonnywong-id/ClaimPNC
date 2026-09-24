package inboxautoclaim_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxautoclaim"
)

// judulLengkap adalah baris judul berkas unggahan, memuat kolom wajib dan opsional.
//
// Nama kolomnya DARI `Activity/InsertKlaimToTable_Other-Act.xml`, bukan karangan. Tiga
// kolom yang dulu ada di sini — inisialid, prodke, currency — sudah dibuang: ketiganya
// hasil pencarian polis, bukan isian pengunggah.
const judulLengkap = "policyno,claimamount,dateofloss,reportdate," +
	"causeofloss,keyword,alasanklaim,objectname,flagtidakbayar,contractno"

func barisSah(polis string) string {
	return polis + ",12500000.00,03/01/2026,05/01/2026,12002,REF-1,catatan,OBJ-1,0,KTR-1"
}

func TestBerkasSahTerbacaMenjadiBarisUnggahan(t *testing.T) {
	berkas := judulLengkap + "\n" + barisSah("0100120260001") + "\n"

	baris, err := inboxautoclaim.ParseUpload(strings.NewReader(berkas))
	require.NoError(t, err)
	require.Len(t, baris, 1)

	require.Equal(t, "0100120260001", baris[0].PolicyNo)
	require.Equal(t, "12500000.00", baris[0].ClaimAmount)
	require.Equal(t, "03/01/2026", baris[0].DateOfLoss)
	require.Equal(t, "05/01/2026", baris[0].ReportDate)
	require.Equal(t, "12002", baris[0].CauseOfLoss)
	require.Equal(t, "REF-1", baris[0].Keyword)
	require.Equal(t, "catatan", baris[0].Reason)
	require.Equal(t, "OBJ-1", baris[0].ObjectName)
	require.Equal(t, "KTR-1", baris[0].ContractNo)

	// Nomor baris dihitung dari 1 dan memperhitungkan baris judul, karena yang
	// diperbaiki pengguna adalah berkasnya.
	require.Equal(t, 2, baris[0].LineNumber)
}

func TestTitikPadaNomorPolisDibuang(t *testing.T) {
	// `InputParam.PolicyNo = @replaceAll(.PolicyNo,".","")` pada
	// Activity/InsertKlaimToTable_Other-Act.xml. Tanpa ini, nomor polis yang diketik
	// dengan titik tidak akan pernah cocok dengan isi T_GENERAL — dan kegagalannya
	// terbaca sebagai "polis tidak ditemukan", bukan sebagai kesalahan penulisan.
	berkas := judulLengkap + "\n" + barisSah("01.001.2026.0001") + "\n"

	baris, err := inboxautoclaim.ParseUpload(strings.NewReader(berkas))
	require.NoError(t, err)
	require.Equal(t, "0100120260001", baris[0].PolicyNo)
}

func TestBerkasBerpemisahTitikKomaTerbaca(t *testing.T) {
	// Excel berlokal Indonesia menyimpan CSV dengan titik koma, dan berkas seperti itu
	// yang benar-benar sampai ke petugas. Memaksa koma membuat seluruh baris judul
	// terbaca sebagai satu kolom, dan pesan galatnya menyesatkan sepenuhnya.
	berkas := strings.ReplaceAll(judulLengkap, ",", ";") + "\n" +
		strings.ReplaceAll(barisSah("0100120260001"), ",", ";") + "\n"

	baris, err := inboxautoclaim.ParseUpload(strings.NewReader(berkas))
	require.NoError(t, err)
	require.Len(t, baris, 1)
	require.Equal(t, "0100120260001", baris[0].PolicyNo)
}

func TestBerkasBerBOMTerbaca(t *testing.T) {
	// Excel menulis BOM di awal berkas CSV. Tanpa penanganannya, judul kolom pertama
	// terbaca sebagai "\ufeffpolicyno" dan berkas yang benar ditolak.
	berkas := "\ufeff" + judulLengkap + "\n" + barisSah("0100120260001") + "\n"

	baris, err := inboxautoclaim.ParseUpload(strings.NewReader(berkas))
	require.NoError(t, err)
	require.Len(t, baris, 1)
	require.Equal(t, "0100120260001", baris[0].PolicyNo)
}

func TestBarisKosongDiAkhirBerkasDiabaikan(t *testing.T) {
	// Baris kosong di akhir adalah hal biasa pada berkas hasil Excel. Menolaknya berarti
	// menolak berkas yang sebenarnya benar.
	berkas := judulLengkap + "\n" + barisSah("0100120260001") + "\n\n\n"

	baris, err := inboxautoclaim.ParseUpload(strings.NewReader(berkas))
	require.NoError(t, err)
	require.Len(t, baris, 1)
}

func TestKolomWajibYangHilangDisebutSeluruhnya(t *testing.T) {
	berkas := "policyno\n0100120260001\n"

	_, err := inboxautoclaim.ParseUpload(strings.NewReader(berkas))
	require.Error(t, err)

	pesan := err.Error()
	for _, kolom := range []string{"claimamount", "dateofloss", "reportdate"} {
		require.Contains(t, pesan, kolom, "kolom %q yang hilang harus disebut", kolom)
	}
}

func TestBerkasTanpaBarisDataDitolak(t *testing.T) {
	_, err := inboxautoclaim.ParseUpload(strings.NewReader(judulLengkap + "\n"))
	require.ErrorIs(t, err, inboxautoclaim.ErrEmptyUpload)
}

func TestSeluruhKolomOpsionalBolehTidakAda(t *testing.T) {
	// Hanya empat kolom yang wajib. Enam sisanya boleh tidak ada sama sekali —
	// termasuk causeofloss, karena Activity/CreateCasePNC_AutoClaim-Act.xml menangani
	// penyebab kerugian kosong sebagai KEGAGALAN BARIS, bukan penolakan berkas.
	// Menolaknya di sini akan mengubah perilaku yang sudah ada — pelanggaran P-5.
	berkas := "policyno,claimamount,dateofloss,reportdate\n" +
		"0100120260001,12500000.00,03/01/2026,05/01/2026\n"

	baris, err := inboxautoclaim.ParseUpload(strings.NewReader(berkas))
	require.NoError(t, err)
	require.Equal(t, "", baris[0].CauseOfLoss)
	require.Equal(t, "", baris[0].Keyword)
	require.NoError(t, inboxautoclaim.CheckUploadShape(baris))
}

func TestJudulKolomDibacaTanpaMemandangSpasiDanBesarKecilHuruf(t *testing.T) {
	// "Policy No", "policy_no", dan "PolicyNo" adalah kolom yang sama bagi pengguna.
	// Menolak salah satunya hanya karena spasi tidak menolong siapa pun.
	berkas := "Policy No,Claim_Amount,Date-Of-Loss,REPORTDATE\n" +
		"0100120260001,12500000.00,03/01/2026,05/01/2026\n"

	baris, err := inboxautoclaim.ParseUpload(strings.NewReader(berkas))
	require.NoError(t, err)
	require.Equal(t, "0100120260001", baris[0].PolicyNo)
	require.Equal(t, "12500000.00", baris[0].ClaimAmount)
}

func TestSeluruhPelanggaranBentukDilaporkanSekaligus(t *testing.T) {
	// Kesetaraan perilaku, bukan selera (P-5). Pada berkas ratusan baris, satu galat per
	// percobaan berarti mengunggah ulang ratusan kali.
	berkas := judulLengkap + "\n" +
		",12500000.00,03/01/2026,05/01/2026,12002,,,,,\n" +
		"0100120260002,,03/01/2026,05/01/2026,12002,,,,,\n" +
		"0100120260003,12500000.00,,05/01/2026,12002,,,,,\n"

	baris, err := inboxautoclaim.ParseUpload(strings.NewReader(berkas))
	require.NoError(t, err)
	require.Len(t, baris, 3)

	var validasi *inboxautoclaim.ValidationError
	require.ErrorAs(t, inboxautoclaim.CheckUploadShape(baris), &validasi)
	require.Len(t, validasi.Violation, 3, "ketiga baris cacat harus dilaporkan sekaligus")

	pesan := validasi.Error()
	require.Contains(t, pesan, "baris 2 · policyno")
	require.Contains(t, pesan, "baris 3 · claimamount")
	require.Contains(t, pesan, "baris 4 · dateofloss")
}

func TestTanggalHarusBerformatHariBulanTahun(t *testing.T) {
	// Bentuknya diperiksa karena Activity/CreateCasePNC_AutoClaim-Act.xml memotongnya
	// dengan posisi karakter tetap. Teks berbentuk lain tidak menghasilkan galat — ia
	// menghasilkan TANGGAL LAIN, diam-diam.
	for _, salah := range []string{"2026-01-03", "3/1/2026", "03-01-2026", "03/01/26", "bukan tanggal"} {
		berkas := judulLengkap + "\n" +
			"0100120260001,12500000.00," + salah + ",05/01/2026,12002,,,,,\n"

		baris, err := inboxautoclaim.ParseUpload(strings.NewReader(berkas))
		require.NoError(t, err)

		err = inboxautoclaim.CheckUploadShape(baris)
		require.Error(t, err, "tanggal %q seharusnya ditolak", salah)
		require.Contains(t, err.Error(), "dd/mm/yyyy")
	}
}

func TestNilaiKlaimHarusAngkaTanpaPemisahRibuan(t *testing.T) {
	// Berkas dibungkus tanda kutip supaya nilai yang MEMUAT koma tetap terbaca sebagai
	// satu sel — tanpa itu, "12.500.000,00" terpecah menjadi dua kolom dan yang teruji
	// bukan aturan nilai uang melainkan aturan penguraian CSV.
	for _, salah := range []string{`"12.500.000,00"`, `"12,500,000"`, `"Rp 12500000"`, `12jt`, `1.2.3`} {
		berkas := judulLengkap + "\n" +
			"0100120260001," + salah + ",03/01/2026,05/01/2026,12002,,,,,\n"

		baris, err := inboxautoclaim.ParseUpload(strings.NewReader(berkas))
		require.NoError(t, err, "nilai %s", salah)
		require.Error(t, inboxautoclaim.CheckUploadShape(baris),
			"nilai %s seharusnya ditolak", salah)
	}
}

func TestNilaiKlaimBerbentukBenarDiterima(t *testing.T) {
	for _, benar := range []string{"0", "12500000", "12500000.00", "0.05"} {
		berkas := judulLengkap + "\n" +
			"0100120260001," + benar + ",03/01/2026,05/01/2026,12002,,,,,\n"

		baris, err := inboxautoclaim.ParseUpload(strings.NewReader(berkas))
		require.NoError(t, err)
		require.NoError(t, inboxautoclaim.CheckUploadShape(baris),
			"nilai %q seharusnya diterima", benar)
	}
}

func TestNilaiKlaimDesimalPenuhDipertahankanApaAdanya(t *testing.T) {
	// I-12: nilai uang disimpan presisi penuh. Teksnya tidak boleh diubah menjadi angka
	// lalu kembali menjadi teks, karena float64 tidak dapat mewakili setiap desimal.
	berkas := judulLengkap + "\n" +
		"0100120260001,12500000.005,03/01/2026,05/01/2026,12002,,,,,\n"

	baris, err := inboxautoclaim.ParseUpload(strings.NewReader(berkas))
	require.NoError(t, err)
	require.NoError(t, inboxautoclaim.CheckUploadShape(baris))
	require.Equal(t, "12500000.005", baris[0].ClaimAmount)
}

func TestBerkasMelebihiBatasBarisDitolak(t *testing.T) {
	var berkas strings.Builder
	berkas.WriteString(judulLengkap + "\n")
	for i := 0; i <= inboxautoclaim.MaxUploadRow; i++ {
		berkas.WriteString(fmt.Sprintf(
			"POL%06d,1000.00,03/01/2026,05/01/2026,12002,,,,,\n", i))
	}

	_, err := inboxautoclaim.ParseUpload(strings.NewReader(berkas.String()))
	require.Error(t, err)
	require.Contains(t, err.Error(), "Pecah berkasnya")
}

func TestJudulKolomGandaDitolak(t *testing.T) {
	berkas := judulLengkap + ",policyno\n" + barisSah("0100120260001") + ",0100120260002\n"

	_, err := inboxautoclaim.ParseUpload(strings.NewReader(berkas))
	require.Error(t, err)
	require.Contains(t, err.Error(), "dua kali")
}

// ---------------------------------------------------------------------------
// Urutan tanggal — kegagalan yang MENANDAI baris, bukan menolak berkas
// ---------------------------------------------------------------------------

func TestTanggalLaporSebelumTanggalKejadianMenandaiBaris(t *testing.T) {
	// Pesannya disalin harfiah dari InsertKlaimToTable_Other. Ia bukan pesan karangan:
	// teks inilah yang tersimpan di TMP_MESSAGE dan yang terbaca petugas di grid.
	pesan := inboxautoclaim.CheckDateOrder("05/01/2026", "03/01/2026")
	require.Equal(t, inboxautoclaim.MessageReportBeforeLoss, pesan)
}

func TestTanggalLaporSamaDenganTanggalKejadianLolos(t *testing.T) {
	// Syarat Pega-nya "harus setelah", tetapi perbandingannya `<`, bukan `<=`. Klaim yang
	// dilaporkan pada hari kejadian LOLOS — dan itu memang yang wajar.
	require.Equal(t, "", inboxautoclaim.CheckDateOrder("03/01/2026", "03/01/2026"))
}

func TestUrutanTanggalDibandingkanSebagaiTanggal_BukanSebagaiTeksApaAdanya(t *testing.T) {
	// Jebakan yang mudah terlewat: sebagai teks `dd/mm/yyyy`, "05/01/2026" LEBIH BESAR
	// daripada "28/12/2025" — sehingga pembandingan teks mentah akan meloloskan urutan
	// yang benar dan menolak yang salah secara acak menurut tanggalnya.
	//
	// Kejadian 28/12/2025, lapor 05/01/2026 adalah urutan yang BENAR.
	require.Equal(t, "", inboxautoclaim.CheckDateOrder("28/12/2025", "05/01/2026"))

	// Dan kebalikannya salah.
	require.Equal(t, inboxautoclaim.MessageReportBeforeLoss,
		inboxautoclaim.CheckDateOrder("05/01/2026", "28/12/2025"))
}

func TestUrutanTanggalTidakDinilaiBilaBentuknyaTidakTerbaca(t *testing.T) {
	// Bentuknya sudah dijamin CheckUploadShape. Bila sampai di sini tetap tidak terbaca,
	// membiarkannya lolos lebih aman daripada menandainya gagal atas dasar yang tidak
	// dapat dipastikan.
	require.Equal(t, "", inboxautoclaim.CheckDateOrder("bukan tanggal", "05/01/2026"))
}

// ---------------------------------------------------------------------------
// Pengelompokan
// ---------------------------------------------------------------------------

func TestBarisDikelompokkanMenurutPerusahaanMengikutiUrutanBerkas(t *testing.T) {
	// Satu berkas dapat menghasilkan beberapa batch — bukan karena pengunggah mengetik
	// kode perusahaan, melainkan karena polis-polis di dalamnya dapat berasal dari
	// perusahaan yang berbeda. Kunci pengelompokannya (inisialid, batch), mengikuti
	// GroupingAutoClaim2.
	line := []inboxautoclaim.UploadLine{
		{CompanyCode: "BPRC"},
		{CompanyCode: "MFIN"},
		{CompanyCode: "BPRC"},
	}

	urutan, kelompok := inboxautoclaim.GroupUploadLine(line)
	require.Equal(t, []string{"BPRC", "MFIN"}, urutan, "urutan mengikuti kemunculan di berkas")
	require.Len(t, kelompok["BPRC"], 2)
	require.Len(t, kelompok["MFIN"], 1)
}

func TestBarisTanpaPesanDinyatakanLolos(t *testing.T) {
	require.True(t, inboxautoclaim.UploadLine{}.Accepted())
	require.False(t, inboxautoclaim.UploadLine{
		Message: inboxautoclaim.MessagePolicyNotFound,
	}.Accepted())
}

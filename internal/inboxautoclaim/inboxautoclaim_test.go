package inboxautoclaim_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxautoclaim"
)

// Nama uji menyebutkan ATURANNYA, bukan nama fungsinya, supaya daftar uji terbaca sebagai
// daftar aturan yang berlaku (14-TESTING-STRATEGY.md §3.2).

func TestBarisBerhasilDitandaiDariPesanSuksesKlaim(t *testing.T) {
	// Ambang ini disalin harfiah dari Activity/REPORT_AUTO_CLAIM_ACT-Act.xml. Ia diuji
	// karena seluruh angka di grid bergantung padanya: satu huruf berbeda membuat setiap
	// baris berhasil terhitung gagal.
	berhasil := inboxautoclaim.Line{Message: inboxautoclaim.MessageSuccess}
	require.True(t, berhasil.Succeeded())
	require.True(t, berhasil.Processed())
}

func TestBarisGagalAdalahBarisBerpesanSelainSuksesKlaim(t *testing.T) {
	gagal := inboxautoclaim.Line{Message: "Penyebab kerugian tidak ditemukan"}
	require.False(t, gagal.Succeeded())
	require.True(t, gagal.Processed())
}

func TestBarisTanpaPesanBelumDiprosesDanBukanGagal(t *testing.T) {
	// Pembedaan ini yang paling mudah salah, dan kueri lamanya pun menegaskannya:
	// "AND TMP_MESSAGE!='Sukses Klaim' and TMP_MESSAGE is not null".
	belum := inboxautoclaim.Line{Message: ""}
	require.False(t, belum.Succeeded())
	require.False(t, belum.Processed())

	// Spasi saja tetap dihitung belum diproses: kolom CHAR berlebar tetap memadatkan
	// nilainya dengan spasi, dan baris yang belum diproses tidak boleh berubah menjadi
	// "gagal" hanya karena tipe kolomnya.
	spasi := inboxautoclaim.Line{Message: "   "}
	require.False(t, spasi.Processed())
}

func TestJumlahBelumProsesAdalahSelisihUploadDanProses(t *testing.T) {
	batch := inboxautoclaim.Batch{Uploaded: 10, Processed: 4}
	require.Equal(t, 6, batch.Pending())
}

func TestJumlahBelumProsesTidakPernahNegatif(t *testing.T) {
	// Tidak mungkin secara aturan, mungkin secara data. Angka negatif di layar hanya
	// memindahkan kebingungan ke pengguna.
	batch := inboxautoclaim.Batch{Uploaded: 3, Processed: 5}
	require.Equal(t, 0, batch.Pending())
}

func TestHalamanDiluarBatasDiperbaiki(t *testing.T) {
	for _, kasus := range []struct {
		nama   string
		minta  inboxautoclaim.PageRequest
		nomor  int
		ukuran int
	}{
		{"nol menjadi satu", inboxautoclaim.PageRequest{Number: 0, Size: 15}, 1, 15},
		{"negatif menjadi satu", inboxautoclaim.PageRequest{Number: -5, Size: 15}, 1, 15},
		{"ukuran nol menjadi bawaan", inboxautoclaim.PageRequest{Number: 2, Size: 0}, 2, inboxautoclaim.DefaultPageSize},
		{"ukuran melebihi batas dipangkas", inboxautoclaim.PageRequest{Number: 1, Size: 5000}, 1, inboxautoclaim.MaxPageSize},
		{"dalam batas dibiarkan", inboxautoclaim.PageRequest{Number: 3, Size: 20}, 3, 20},
	} {
		t.Run(kasus.nama, func(t *testing.T) {
			bersih := kasus.minta.Clean()
			require.Equal(t, kasus.nomor, bersih.Number)
			require.Equal(t, kasus.ukuran, bersih.Size)
		})
	}
}

func TestOffsetHalamanPertamaNol(t *testing.T) {
	require.Equal(t, 0, inboxautoclaim.PageRequest{Number: 1, Size: 15}.Offset())
	require.Equal(t, 15, inboxautoclaim.PageRequest{Number: 2, Size: 15}.Offset())
	require.Equal(t, 30, inboxautoclaim.PageRequest{Number: 3, Size: 15}.Offset())
}

func TestPenyaringHasilHanyaMenerimaTigaNilai(t *testing.T) {
	for masukan, harapan := range map[string]inboxautoclaim.Result{
		"":         inboxautoclaim.ResultAll,
		"berhasil": inboxautoclaim.ResultSucceeded,
		"BERHASIL": inboxautoclaim.ResultSucceeded,
		" gagal ":  inboxautoclaim.ResultFailed,
	} {
		hasil, err := inboxautoclaim.ParseResult(masukan)
		require.NoError(t, err, "masukan %q", masukan)
		require.Equal(t, harapan, hasil)
	}
}

func TestPenyaringHasilTidakDikenalDitolak(t *testing.T) {
	// Ditolak, BUKAN diperlakukan sebagai "semua": penyaring yang salah ketik lalu
	// diabaikan akan mengunduh berkas berisi seluruh baris padahal pengguna meminta yang
	// gagal saja.
	_, err := inboxautoclaim.ParseResult("sukses")
	require.ErrorIs(t, err, inboxautoclaim.ErrUnknownResult)
}

func TestNomorBatchBerikutnyaSatuBilaBelumAda(t *testing.T) {
	require.Equal(t, "1", inboxautoclaim.NextBatchNumber(nil))
}

func TestNomorBatchBerikutnyaMelanjutkanYangTerbesar(t *testing.T) {
	require.Equal(t, "4", inboxautoclaim.NextBatchNumber([]string{"1", "2", "3"}))
}

func TestNomorBatchDihitungSebagaiAngkaBukanTeks(t *testing.T) {
	// Inilah cacat yang dihindari: maksimum LEKSIKOGRAFIS dari {"9","10"} adalah "9",
	// sehingga MAX di SQL akan menerbitkan "10" untuk kedua kalinya.
	require.Equal(t, "11", inboxautoclaim.NextBatchNumber([]string{"9", "10"}))
}

func TestNomorBatchTidakMenabrakNomorBerbentukLain(t *testing.T) {
	// Baris lama dapat memuat apa saja. Nomor yang tidak dapat ditafsirkan sebagai angka
	// tidak ikut menentukan yang terbesar, tetapi tetap dihitung terpakai.
	require.Equal(t, "3", inboxautoclaim.NextBatchNumber([]string{"1", "2", "BATCH-A"}))
}

func TestBentukBerkasEksporBerhasilSesuaiPega(t *testing.T) {
	// Judul kolom disalin harfiah dari Activity/REPORT_AUTO_CLAIM_ACT-Act.xml. Uji ini
	// yang menjaganya tidak berubah diam-diam — berkasnya dibaca pihak di luar Sinarmas.
	spec, err := inboxautoclaim.ExportSpecFor(inboxautoclaim.ResultSucceeded)
	require.NoError(t, err)
	require.Equal(t, "Laporan Hasil Klaim", spec.FileName)
	require.Equal(t, []string{
		"Inisial", "No Polis", "No Klaim", "No Ref Bank", "No Aksep",
		"Currency", "Nilai Klaim", "No Objek", "Keterangan",
	}, spec.Header)
}

func TestBerkasEksporGagalSatuKolomLebihSedikit(t *testing.T) {
	// Perbedaan panjang ini NYATA di export dan bukan kekeliruan pembacaan: berkas GAGAL
	// tidak memuat "No Ref Bank".
	berhasil, err := inboxautoclaim.ExportSpecFor(inboxautoclaim.ResultSucceeded)
	require.NoError(t, err)
	gagal, err := inboxautoclaim.ExportSpecFor(inboxautoclaim.ResultFailed)
	require.NoError(t, err)

	require.Len(t, berhasil.Header, 9)
	require.Len(t, gagal.Header, 8)
	require.NotContains(t, gagal.Header, "No Ref Bank")
	require.Equal(t, "Laporan Hasil Gagal Klaim", gagal.FileName)
}

func TestJumlahKolomBerkasSamaDenganJumlahNilaiBarisnya(t *testing.T) {
	// Berkas CSV yang jumlah selnya tidak sama dengan judulnya akan terbaca bergeser di
	// Excel, dan pergeseran itu tidak menghasilkan galat apa pun.
	contoh := inboxautoclaim.Line{
		CompanyCode: "MFIN", PolicyNo: "0100120260001", ClaimID: "PNCN.26.101",
		Keyword: "REF-1", AcceptanceNo: "AKS-1", Currency: "1", CurrencyCode: "IDR",
		ClaimAmount: "1000.00", ProductSeq: "1", CauseOfLoss: "12002",
		Message: inboxautoclaim.MessageSuccess,
	}
	for _, hasil := range []inboxautoclaim.Result{inboxautoclaim.ResultSucceeded, inboxautoclaim.ResultFailed} {
		spec, err := inboxautoclaim.ExportSpecFor(hasil)
		require.NoError(t, err)
		require.Len(t, spec.Row(contoh), len(spec.Header), "hasil %q", hasil)
	}
}

func TestIsiTiapKolomBerkasEksporSesuaiKueriPega(t *testing.T) {
	// Uji ini ada karena DUA dari sembilan pemetaan kolom SALAH pada versi pertama modul
	// ini, dan keduanya baru terlihat setelah
	// InboxAutoClaim/BrowseReportClaimSPK_AutoClaim-SQL.xml diterima:
	//
	//	"No Objek"    dulu PRODKE   -> sebenarnya COL_ID
	//	"No Ref Bank" dulu KEYWORD  -> sebenarnya lookup lewat DB Link
	//
	// Nilai contohnya sengaja dibuat berbeda satu sama lain supaya kolom yang tertukar
	// langsung ketahuan — dengan nilai kembar, pemetaan yang salah tetap lulus.
	contoh := inboxautoclaim.Line{
		CompanyCode:  "MFIN",
		PolicyNo:     "0100120260001",
		ClaimID:      "ASM-FW-GCNMFW-WORK PNC-1865",
		AcceptanceNo: "AKS-1",
		Currency:     "1",
		CurrencyCode: "IDR",
		ClaimAmount:  "1000.00",
		CauseOfLoss:  "12002",
		ProductSeq:   "7",
		Keyword:      "REF-1",
		Message:      inboxautoclaim.MessageSuccess,
	}

	spec, err := inboxautoclaim.ExportSpecFor(inboxautoclaim.ResultSucceeded)
	require.NoError(t, err)

	require.Equal(t, []string{
		"MFIN",          // Inisial     <- INISIALID
		"0100120260001", // No Polis    <- NOPOLIS
		"PNC-1865",      // No Klaim    <- IDPEGA, prefix Pega dipangkas
		"",              // No Ref Bank <- DB Link, belum tersedia (R-03)
		"AKS-1",         // No Aksep    <- NOAKSEPTASI
		"IDR",           // Currency    <- KODE hasil lookup, bukan id "1"
		"1000.00",       // Nilai Klaim <- NILAIKLAIM
		"12002",         // No Objek    <- COL_ID, BUKAN PRODKE "7"
		"Sukses Klaim",  // Keterangan  <- TMP_MESSAGE
	}, spec.Row(contoh))
}

func TestAwalanKunciTeknisPegaDipangkasDariNomorKlaim(t *testing.T) {
	// `case when idpega like '%PNC%' then substr(IDPEGA,20,30) else idpega end`.
	// Angka 20 adalah panjang "ASM-FW-GCNMFW-WORK " ditambah satu — kebocoran kunci
	// teknis Pega ke data bisnis (utang teknis 4.1, dijawab D-22/D-71).
	require.Equal(t, "PNC-1865",
		inboxautoclaim.StripClaimPrefix("ASM-FW-GCNMFW-WORK PNC-1865"))
}

func TestNomorKlaimSistemBaruTidakIkutTerpangkas(t *testing.T) {
	// Ini SELISIH YANG DISENGAJA terhadap kueri Pega, dan sebabnya bernilai uang.
	//
	// Syarat aslinya `like '%PNC%'` juga cocok dengan "PNCN.26.8125" — nomor klaim
	// sistem baru menurut D-71. Karena nilainya lebih pendek dari 20 karakter,
	// `substr(...,20,30)` Oracle mengembalikan TEKS KOSONG: nomor klaimnya lenyap dari
	// berkas yang dikirim ke perusahaan rekanan, tanpa satu pun tanda.
	//
	// Pega sendiri tidak pernah menerbitkan nomor berbentuk itu, jadi selisihnya tidak
	// akan muncul pada gerbang 1 — ia muncul nanti, di produksi.
	require.Equal(t, "PNCN.26.8125", inboxautoclaim.StripClaimPrefix("PNCN.26.8125"))
}

func TestTeksGalatPadaKolomNomorKlaimTidakDipangkas(t *testing.T) {
	// Baris GAGAL menyimpan teks galat di kolom IDPEGA. Tidak satu pun memuat "PNC",
	// sehingga Pega membiarkannya utuh — dan begitu pula di sini.
	for _, pesan := range []string{
		inboxautoclaim.MessagePolicyNotFound,
		inboxautoclaim.MessageReceiverNotFound,
		inboxautoclaim.MessageReportBeforeLoss,
	} {
		require.Equal(t, pesan, inboxautoclaim.StripClaimPrefix(pesan))
	}
}

func TestEksporSeluruhBarisTidakTersedia(t *testing.T) {
	// Layar lama hanya punya EXPORT BERHASIL dan EXPORT GAGAL. Menambah "ekspor semua"
	// berarti menerbitkan berkas yang tidak pernah ada, dengan judul kolom yang harus
	// dikarang — kedua berkas yang ada pun berbeda panjangnya.
	_, err := inboxautoclaim.ExportSpecFor(inboxautoclaim.ResultAll)
	require.ErrorIs(t, err, inboxautoclaim.ErrUnknownResult)
}

func TestGalatValidasiMemuatSeluruhPelanggaran(t *testing.T) {
	err := &inboxautoclaim.ValidationError{Violation: []inboxautoclaim.Violation{
		{Field: "baris 2 · nopolis", Message: "Nomor polis wajib diisi."},
		{Field: "baris 3 · nilaiklaim", Message: "Nilai klaim wajib diisi."},
	}}

	var target *inboxautoclaim.ValidationError
	require.True(t, errors.As(error(err), &target))
	require.Len(t, target.Violation, 2)
	require.True(t, strings.Contains(err.Error(), "baris 2 · nopolis"))
	require.True(t, strings.Contains(err.Error(), "baris 3 · nilaiklaim"))
}

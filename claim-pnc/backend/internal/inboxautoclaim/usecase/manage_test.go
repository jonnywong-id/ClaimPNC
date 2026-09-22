package usecase_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxautoclaim"
	"claim-pnc/internal/inboxautoclaim/repo/memory"
	"claim-pnc/internal/inboxautoclaim/usecase"
)

// errPortalTidakSiap meniru galat pemilih portal, supaya penolakannya teruji tanpa
// mengimpor modul portal ke dalam uji modul ini.
var errPortalTidakSiap = errors.New("portal belum siap")

// layananContoh menyusun layanan di atas penyimpanan memori berisi data contoh.
func layananContoh(t *testing.T) (*usecase.Service, *memory.Repo) {
	t.Helper()

	repo := memory.NewSampleRepo()
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (inboxautoclaim.Repo, error) {
			if strings.ToUpper(alias) != "ASM" {
				return nil, errPortalTidakSiap
			}
			return repo, nil
		},
	})
	require.NoError(t, err)
	return service, repo
}

// barisUnggahan menyusun satu baris berkas yang bentuknya sudah benar.
//
// Perhatikan apa yang TIDAK ada di sini: kode perusahaan, prodke, dan mata uang.
// Ketiganya hasil pencarian polis, bukan isian pengunggah — dan itulah perubahan terbesar
// setelah Activity/InsertKlaimToTable_Other-Act.xml diterima.
func barisUnggahan(nomor int, polis string) inboxautoclaim.UploadRow {
	return inboxautoclaim.UploadRow{
		LineNumber:  nomor,
		PolicyNo:    polis,
		ClaimAmount: "100.00",
		DateOfLoss:  "01/04/2026",
		ReportDate:  "02/04/2026",
		CauseOfLoss: "12002",
	}
}

func TestLayananMenolakBahanTidakLengkap(t *testing.T) {
	// Penolakannya terjadi saat perakitan, bukan saat permintaan pertama datang: rakitan
	// setengah jadi harus gagal saat start, bukan saat pengguna sedang bekerja.
	_, err := usecase.NewService(usecase.Options{})
	require.Error(t, err)
}

func TestPortalLainDitolakBukanDilayaniPortalUtama(t *testing.T) {
	// R-20: jatuh ke koneksi default berarti membaca data satu badan hukum di basis data
	// badan hukum lain tanpa satu pun pesan galat.
	service, _ := layananContoh(t)

	_, err := service.ListBatch(context.Background(), "SIMASNET", inboxautoclaim.BatchFilter{Source: inboxautoclaim.SourceAneka})
	require.ErrorIs(t, err, errPortalTidakSiap)
}

func TestDaftarBatchDiringkasPerPerusahaanDanNomorBatch(t *testing.T) {
	service, _ := layananContoh(t)

	page, err := service.ListBatch(context.Background(), "ASM", inboxautoclaim.BatchFilter{Source: inboxautoclaim.SourceAneka})
	require.NoError(t, err)

	// Data contoh memuat empat batch: MFIN 2, MFIN 1, BPRC 1, ZZZZ 1.
	require.Equal(t, 4, page.Total)
	require.Len(t, page.Item, 4)
}

func TestBatchTerbaruTampilLebihDulu(t *testing.T) {
	// `ORDER BY BATCH DESC` pada BrowseClaimSPKAutoClaim. Versi pertama modul ini
	// mengurutkan menaik, sehingga batch yang baru saja diunggah petugas justru berada di
	// halaman TERAKHIR — dan pada perusahaan dengan puluhan batch, ia tidak akan
	// menemukannya tanpa membuka halaman satu per satu.
	service, _ := layananContoh(t)

	page, err := service.ListBatch(context.Background(), "ASM", inboxautoclaim.BatchFilter{Source: inboxautoclaim.SourceAneka,
		CompanyCode: "MFIN",
	})
	require.NoError(t, err)
	require.Len(t, page.Item, 2)
	require.Equal(t, "2", page.Item[0].BatchNumber, "batch terbesar lebih dulu")
	require.Equal(t, "1", page.Item[1].BatchNumber)
}

func TestHitunganBatchSesuaiIsiBarisnya(t *testing.T) {
	service, _ := layananContoh(t)

	page, err := service.ListBatch(context.Background(), "ASM", inboxautoclaim.BatchFilter{Source: inboxautoclaim.SourceAneka,
		CompanyCode: "MFIN",
	})
	require.NoError(t, err)
	require.Len(t, page.Item, 2)

	// MFIN batch 2: 4 baris, seluruhnya diproses, 2 berhasil dan 2 gagal.
	batchDua := page.Item[0]
	require.Equal(t, "2", batchDua.BatchNumber)
	require.Equal(t, 4, batchDua.Uploaded)
	require.Equal(t, 4, batchDua.Processed)
	require.Equal(t, 2, batchDua.Succeeded)
	require.Equal(t, 2, batchDua.Failed)
	require.Equal(t, 0, batchDua.Pending())

	// MFIN batch 1: 2 baris, belum satu pun diproses.
	batchSatu := page.Item[1]
	require.Equal(t, "1", batchSatu.BatchNumber)
	require.Equal(t, 2, batchSatu.Uploaded)
	require.Equal(t, 0, batchSatu.Processed)
	require.Equal(t, 2, batchSatu.Pending())
}

func TestTanggalProsesIkutDibawaKeBarisGrid(t *testing.T) {
	// TGLPROSES bagian KUNCI PENGELOMPOKAN, bukan sekadar kolom tampilan: satu nomor
	// batch yang diunggah pada dua tanggal berbeda tampil sebagai dua baris. Tanpa
	// kolomnya, kedua baris itu tampak kembar tanpa sebab.
	service, _ := layananContoh(t)

	page, err := service.ListBatch(context.Background(), "ASM", inboxautoclaim.BatchFilter{Source: inboxautoclaim.SourceAneka,
		CompanyCode: "BPRC",
	})
	require.NoError(t, err)
	require.Len(t, page.Item, 1)
	require.Equal(t, "20/01/2026", page.Item[0].ProcessedDate)
}

func TestBatchTanpaBarisMasterTetapTerlihat(t *testing.T) {
	// Inilah yang membuktikan LEFT JOIN berlaku. Kueri Pega aslinya memakai INNER JOIN
	// (`FROM ... a, M_AUTO_CLAIM_PNC b WHERE a.INISIALID = b.INISIALID`), sehingga baris
	// seperti ini HILANG dari layar tanpa satu pun tanda — padahal baris yang kode
	// perusahaannya tidak ada di master justru yang tidak akan pernah berhasil diproses.
	service, _ := layananContoh(t)

	page, err := service.ListBatch(context.Background(), "ASM", inboxautoclaim.BatchFilter{Source: inboxautoclaim.SourceAneka,
		CompanyCode: "ZZZZ",
	})
	require.NoError(t, err)
	require.Len(t, page.Item, 1)
	require.Equal(t, "ZZZZ", page.Item[0].CompanyCode)
	require.Equal(t, "", page.Item[0].CompanyName, "namanya kosong, barisnya tetap ada")
}

func TestPaginasiMemotongDaftarTanpaMengubahTotal(t *testing.T) {
	service, _ := layananContoh(t)

	halamanSatu, err := service.ListBatch(context.Background(), "ASM", inboxautoclaim.BatchFilter{Source: inboxautoclaim.SourceAneka,
		Page: inboxautoclaim.PageRequest{Number: 1, Size: 2},
	})
	require.NoError(t, err)
	require.Len(t, halamanSatu.Item, 2)
	require.Equal(t, 4, halamanSatu.Total, "total menyebut seluruh baris, bukan yang tampil")

	halamanDua, err := service.ListBatch(context.Background(), "ASM", inboxautoclaim.BatchFilter{Source: inboxautoclaim.SourceAneka,
		Page: inboxautoclaim.PageRequest{Number: 2, Size: 2},
	})
	require.NoError(t, err)
	require.Len(t, halamanDua.Item, 2)
	require.NotEqual(t, halamanSatu.Item[0].BatchNumber+halamanSatu.Item[0].CompanyCode,
		halamanDua.Item[0].BatchNumber+halamanDua.Item[0].CompanyCode)

	// Halaman di luar jangkauan menghasilkan daftar kosong, bukan galat: ia keadaan yang
	// wajar setelah baris berkurang di antara dua permintaan.
	halamanJauh, err := service.ListBatch(context.Background(), "ASM", inboxautoclaim.BatchFilter{Source: inboxautoclaim.SourceAneka,
		Page: inboxautoclaim.PageRequest{Number: 99, Size: 2},
	})
	require.NoError(t, err)
	require.Empty(t, halamanJauh.Item)
	require.Equal(t, 4, halamanJauh.Total)
}

func TestRincianBatchYangTidakAdaDitolakBukanDijawabKosong(t *testing.T) {
	// Keduanya terlihat sama di layar — tabel tanpa baris — tetapi menuntut tindakan yang
	// berbeda: yang pertama berarti tautannya salah.
	service, _ := layananContoh(t)

	_, err := service.ListLine(context.Background(), "ASM", inboxautoclaim.LineQuery{Source: inboxautoclaim.SourceAneka,
		CompanyCode: "MFIN", BatchNumber: "999",
	})
	require.ErrorIs(t, err, inboxautoclaim.ErrBatchNotFound)
}

func TestPenyaringHasilMemisahkanBerhasilGagalDanBelum(t *testing.T) {
	service, _ := layananContoh(t)
	ctx := context.Background()

	semua, err := service.ListLine(ctx, "ASM", inboxautoclaim.LineQuery{Source: inboxautoclaim.SourceAneka,
		CompanyCode: "BPRC", BatchNumber: "1",
	})
	require.NoError(t, err)
	require.Equal(t, 3, semua.Total)

	berhasil, err := service.ListLine(ctx, "ASM", inboxautoclaim.LineQuery{Source: inboxautoclaim.SourceAneka,
		CompanyCode: "BPRC", BatchNumber: "1", Result: inboxautoclaim.ResultSucceeded,
	})
	require.NoError(t, err)
	require.Equal(t, 1, berhasil.Total)

	// Dua baris BPRC yang belum diproses BUKAN baris gagal.
	gagal, err := service.ListLine(ctx, "ASM", inboxautoclaim.LineQuery{Source: inboxautoclaim.SourceAneka,
		CompanyCode: "BPRC", BatchNumber: "1", Result: inboxautoclaim.ResultFailed,
	})
	require.NoError(t, err)
	require.Equal(t, 0, gagal.Total)
}

func TestKodeMataUangDitampilkanBukanIdnya(t *testing.T) {
	// Kolom CURRENCY menyimpan ID; layar Pega menampilkan hasil lookup ke
	// POOLDATA.CURRENCY. Versi pertama modul ini menampilkan id-nya — pengguna melihat
	// "1", bukan "IDR".
	service, _ := layananContoh(t)

	page, err := service.ListLine(context.Background(), "ASM", inboxautoclaim.LineQuery{Source: inboxautoclaim.SourceAneka,
		CompanyCode: "BPRC", BatchNumber: "1",
	})
	require.NoError(t, err)

	kode := make([]string, 0, len(page.Item))
	for _, l := range page.Item {
		kode = append(kode, l.CurrencyCode)
	}
	require.Contains(t, kode, "IDR")
	require.Contains(t, kode, "USD")
}

func TestBerkasEksporBerhasilMemuatJudulDanBarisnya(t *testing.T) {
	service, _ := layananContoh(t)

	file, err := service.Export(context.Background(), "ASM", inboxautoclaim.LineQuery{Source: inboxautoclaim.SourceAneka,
		CompanyCode: "MFIN", BatchNumber: "2", Result: inboxautoclaim.ResultSucceeded,
	})
	require.NoError(t, err)
	require.Equal(t, 2, file.Rows)

	isi := string(file.Content)
	require.True(t, strings.HasPrefix(isi, "\ufeff"), "BOM ditulis supaya Excel membacanya sebagai UTF-8")
	require.Contains(t, isi, "Inisial,No Polis,No Klaim,No Ref Bank,No Aksep,Currency,Nilai Klaim,No Objek,Keterangan")
	require.Contains(t, isi, "PNCN.26.101")
	require.NotContains(t, isi, "Penyebab kerugian tidak ditemukan", "baris gagal tidak ikut")

	// Nama berkas menyebut perusahaan dan batch-nya, supaya dua unduhan berturut-turut
	// tidak saling menimpa di folder Unduhan.
	require.Equal(t, "Laporan Hasil Klaim - MFIN 2.csv", file.FileName)
}

func TestBerkasEksporGagalHanyaMemuatBarisGagal(t *testing.T) {
	service, _ := layananContoh(t)

	file, err := service.Export(context.Background(), "ASM", inboxautoclaim.LineQuery{Source: inboxautoclaim.SourceAneka,
		CompanyCode: "MFIN", BatchNumber: "2", Result: inboxautoclaim.ResultFailed,
	})
	require.NoError(t, err)
	require.Equal(t, 2, file.Rows)

	isi := string(file.Content)
	require.Contains(t, isi, "Penyebab kerugian tidak ditemukan")
	require.NotContains(t, isi, "PNCN.26.101", "baris berhasil tidak ikut")
	require.Equal(t, "Laporan Hasil Gagal Klaim - MFIN 2.csv", file.FileName)
}

func TestEksporBatchYangTidakAdaDitolak(t *testing.T) {
	// Berkas kosong karena batch-nya salah ketik tidak dapat dibedakan dari berkas kosong
	// karena batch-nya memang belum diproses.
	service, _ := layananContoh(t)

	_, err := service.Export(context.Background(), "ASM", inboxautoclaim.LineQuery{Source: inboxautoclaim.SourceAneka,
		CompanyCode: "MFIN", BatchNumber: "999", Result: inboxautoclaim.ResultSucceeded,
	})
	require.ErrorIs(t, err, inboxautoclaim.ErrBatchNotFound)
}

// ---------------------------------------------------------------------------
// Unggahan
// ---------------------------------------------------------------------------

func TestPerusahaanDiturunkanDariPolisBukanDariBerkas(t *testing.T) {
	// Inilah inti perubahan setelah InsertKlaimToTable_Other diterima. Berkasnya tidak
	// memuat kode perusahaan sama sekali; ia dicari dari `t_general.sourceofbusiness`
	// menurut nomor polisnya.
	//
	// Akibat yang mudah terlewat: satu berkas dapat menghasilkan BEBERAPA batch tanpa
	// pengunggah menyadarinya, karena polis-polis di dalamnya berasal dari perusahaan
	// yang berbeda.
	service, _ := layananContoh(t)

	hasil, err := service.Upload(context.Background(), "ASM", inboxautoclaim.SourceAneka, []inboxautoclaim.UploadRow{
		barisUnggahan(2, "0100120260500"), // MFIN
		barisUnggahan(3, "0200120260500"), // BPRC
	}, "ADMINPNC")
	require.NoError(t, err)
	require.Equal(t, 2, hasil.Rows)
	require.Len(t, hasil.Batch, 2)

	batch := map[string]string{}
	for _, b := range hasil.Batch {
		batch[b.CompanyCode] = b.BatchNumber
	}
	// MFIN sudah punya batch 1 dan 2, jadi yang baru adalah 3. BPRC baru punya 1.
	require.Equal(t, "3", batch["MFIN"])
	require.Equal(t, "2", batch["BPRC"])

	// Nama perusahaan dilengkapi supaya ringkasan menyebut nama, bukan kode.
	for _, b := range hasil.Batch {
		require.NotEmpty(t, b.CompanyName, "nama perusahaan %q kosong", b.CompanyCode)
	}
}

func TestBarisUnggahanYangLolosBelumDiproses(t *testing.T) {
	// Inilah yang membuat batch baru terambil pemrosesan: IDPEGA, NOAKSEPTASI, dan
	// TMP_MESSAGE harus kosong.
	service, _ := layananContoh(t)
	ctx := context.Background()

	hasil, err := service.Upload(ctx, "ASM", inboxautoclaim.SourceAneka, []inboxautoclaim.UploadRow{
		barisUnggahan(2, "0100120260500"),
	}, "ADMINPNC")
	require.NoError(t, err)
	require.Len(t, hasil.Batch, 1)
	require.Equal(t, 1, hasil.Batch[0].Succeeded)
	require.Equal(t, 0, hasil.Batch[0].Failed)

	page, err := service.ListLine(ctx, "ASM", inboxautoclaim.LineQuery{Source: inboxautoclaim.SourceAneka,
		CompanyCode: "MFIN", BatchNumber: hasil.Batch[0].BatchNumber,
	})
	require.NoError(t, err)
	require.Len(t, page.Item, 1)
	require.False(t, page.Item[0].Processed())
	require.Empty(t, page.Item[0].ClaimID)
	require.Empty(t, page.Item[0].AcceptanceNo)
	require.Equal(t, "ADMINPNC", page.Item[0].UploadedBy)

	// Prodke TIDAK berasal dari berkas — ia hasil pencarian polis.
	require.Equal(t, "1", page.Item[0].ProductSeq)
}

func TestBarisYangGagalPemeriksaanTETAPDisimpanBesertaPesannya(t *testing.T) {
	// Ini perilaku Pega yang paling mudah salah ditiru. InsertKlaimToTable_Other TIDAK
	// menolak baris yang gagal — ia MENYISIPKANNYA dengan pesan galat pada IDPEGA,
	// NOAKSEPTASI, dan TMP_MESSAGE sekaligus, sehingga barisnya langsung terhitung GAGAL
	// di grid dan ikut keluar di berkas ekspor GAGAL.
	//
	// Menolaknya di muka akan membuat pengguna kehilangan jejak baris yang bermasalah.
	service, _ := layananContoh(t)
	ctx := context.Background()

	// Polis ini ada di T_GENERAL tetapi tidak ada di JSON_POLIS.
	hasil, err := service.Upload(ctx, "ASM", inboxautoclaim.SourceAneka, []inboxautoclaim.UploadRow{
		barisUnggahan(2, "0100120260999"),
	}, "ADMINPNC")
	require.NoError(t, err)
	require.Len(t, hasil.Batch, 1)
	require.Equal(t, 0, hasil.Batch[0].Succeeded)
	require.Equal(t, 1, hasil.Batch[0].Failed)
	require.Empty(t, hasil.Rejected, "barisnya tersimpan, bukan ditolak")

	page, err := service.ListLine(ctx, "ASM", inboxautoclaim.LineQuery{Source: inboxautoclaim.SourceAneka,
		CompanyCode: "MFIN", BatchNumber: hasil.Batch[0].BatchNumber,
	})
	require.NoError(t, err)
	require.Len(t, page.Item, 1)

	baris := page.Item[0]
	require.Equal(t, inboxautoclaim.MessagePolicyNotFound, baris.Message)
	require.Equal(t, inboxautoclaim.MessagePolicyNotFound, baris.ClaimID,
		"ketiga kolom penanda membawa pesan yang sama")
	require.Equal(t, inboxautoclaim.MessagePolicyNotFound, baris.AcceptanceNo)
	require.True(t, baris.Processed())
	require.False(t, baris.Succeeded())
}

func TestTanggalLaporMendahuluiKejadianMenandaiBarisBukanMenolakBerkas(t *testing.T) {
	service, _ := layananContoh(t)
	ctx := context.Background()

	row := barisUnggahan(2, "0100120260500")
	row.DateOfLoss = "05/04/2026"
	row.ReportDate = "01/04/2026"

	hasil, err := service.Upload(ctx, "ASM", inboxautoclaim.SourceAneka, []inboxautoclaim.UploadRow{row}, "ADMINPNC")
	require.NoError(t, err, "berkasnya tidak ditolak")
	require.Equal(t, 1, hasil.Batch[0].Failed)

	page, err := service.ListLine(ctx, "ASM", inboxautoclaim.LineQuery{Source: inboxautoclaim.SourceAneka,
		CompanyCode: "MFIN", BatchNumber: hasil.Batch[0].BatchNumber,
	})
	require.NoError(t, err)
	require.Equal(t, inboxautoclaim.MessageReportBeforeLoss, page.Item[0].Message)
}

func TestBarisTanpaPerusahaanDitolakDanTidakDisimpan(t *testing.T) {
	// Satu-satunya kegagalan yang membuat baris TIDAK disisipkan. Tanpa kode perusahaan,
	// barisnya tidak punya tempat di grid mana pun — menyisipkannya berarti membuat baris
	// yang tidak akan pernah terlihat siapa pun.
	service, _ := layananContoh(t)
	ctx := context.Background()

	sebelum, err := service.ListBatch(ctx, "ASM", inboxautoclaim.BatchFilter{Source: inboxautoclaim.SourceAneka})
	require.NoError(t, err)

	hasil, err := service.Upload(ctx, "ASM", inboxautoclaim.SourceAneka, []inboxautoclaim.UploadRow{
		barisUnggahan(2, "0900120260500"), // polis tanpa perusahaan rekanan aktif
		barisUnggahan(3, "0999999999999"), // polis yang tidak terdaftar sama sekali
	}, "ADMINPNC")
	require.NoError(t, err, "berkasnya tidak ditolak, barisnya yang ditolak")
	require.Equal(t, 0, hasil.Rows)
	require.Empty(t, hasil.Batch)
	require.Len(t, hasil.Rejected, 2)

	// Nomor baris dan nomor polis ikut dilaporkan: tanpa keduanya, pengguna dengan berkas
	// ratusan baris hanya tahu "ada yang gagal".
	require.Equal(t, 2, hasil.Rejected[0].LineNumber)
	require.Equal(t, "0900120260500", hasil.Rejected[0].PolicyNo)
	require.Equal(t, inboxautoclaim.MessageReceiverNotFound, hasil.Rejected[0].Message)

	sesudah, err := service.ListBatch(ctx, "ASM", inboxautoclaim.BatchFilter{Source: inboxautoclaim.SourceAneka})
	require.NoError(t, err)
	require.Equal(t, sebelum.Total, sesudah.Total, "tidak satu batch pun boleh bertambah")
}

func TestBerkasBercacatBentukDitolakSeluruhnya(t *testing.T) {
	// Berbeda dari kegagalan pemeriksaan polis: berkas yang bentuknya salah belum
	// berbentuk berkas klaim sama sekali, sehingga ia ditolak utuh dan tidak satu baris
	// pun disimpan. Unggahan yang tersimpan setengah adalah keadaan yang tidak dapat
	// diperbaiki pengguna — ia tidak punya cara mengetahui baris mana yang sudah masuk.
	service, _ := layananContoh(t)
	ctx := context.Background()

	sebelum, err := service.ListBatch(ctx, "ASM", inboxautoclaim.BatchFilter{Source: inboxautoclaim.SourceAneka})
	require.NoError(t, err)

	rusak := barisUnggahan(3, "0200120260500")
	rusak.ClaimAmount = "seribu"

	_, err = service.Upload(ctx, "ASM", inboxautoclaim.SourceAneka, []inboxautoclaim.UploadRow{
		barisUnggahan(2, "0100120260500"),
		rusak,
	}, "ADMINPNC")
	require.Error(t, err)

	sesudah, err := service.ListBatch(ctx, "ASM", inboxautoclaim.BatchFilter{Source: inboxautoclaim.SourceAneka})
	require.NoError(t, err)
	require.Equal(t, sebelum.Total, sesudah.Total, "tidak satu batch pun boleh bertambah")
}

func TestDaftarPerusahaanDiambilDariMasterBukanDariBatch(t *testing.T) {
	// `BrowseCompanyClaimCredit-SQL.xml` membaca M_AUTO_CLAIM_PNC apa adanya, tanpa
	// menyentuh tabel batch.
	//
	// Versi pertama modul ini mengambilnya dari tabel batch dengan alasan "setiap pilihan
	// harus dijamin menghasilkan baris". Alasannya terdengar masuk akal dan tetap SALAH:
	// perusahaan yang baru didaftarkan di master dan belum punya batch satu pun tidak
	// akan muncul di penyaring, sehingga petugas tidak dapat memastikan batch-nya memang
	// belum ada — ia hanya melihat namanya hilang.
	//
	// Akibat sebaliknya juga nyata: ZZZZ punya batch tetapi TIDAK ada di master, sehingga
	// ia tidak muncul sebagai pilihan.
	service, _ := layananContoh(t)

	list, err := service.ListCompany(context.Background(), "ASM")
	require.NoError(t, err)

	kode := make([]string, 0, len(list))
	for _, c := range list {
		kode = append(kode, c.Code)
	}
	require.ElementsMatch(t, []string{"MFIN", "BPRC", "KRDU", "SRVY"}, kode)
	require.NotContains(t, kode, "ZZZZ", "punya batch tetapi tidak ada di master")
}

func TestKodePerusahaanDicocokkanApaAdanyaTanpaDiubahBesarKecilHurufnya(t *testing.T) {
	// Uji ini lahir dari cacat sungguhan: penyaring perusahaan SELALU menghasilkan tabel
	// kosong terhadap Oracle.
	//
	// Sebabnya bukan SQL-nya, melainkan perlakuan yang TIDAK SIMETRIS terhadap kodenya:
	//
	//	ListCompany  -> membaca INISIALID apa adanya, mengisi dropdown
	//	ListBatch    -> meng-UPPERCASE nilai yang dikirim balik dropdown itu
	//	kolomnya     -> tidak pernah diubah
	//
	// Selama seluruh kode kebetulan huruf besar, cacatnya tidak terlihat — dan data contoh
	// saya semuanya huruf besar, jadi tidak satu pun uji menangkapnya.
	//
	// Kueri Pega menjodohkan `a.INISIALID = b.INISIALID` tanpa mengubah keduanya, jadi
	// aturannya: nilai yang dibaca dari basis data dipakai APA ADANYA.
	repo := memory.NewRepo(
		map[string]string{"Mfin": "Mitra Finansial Nusantara"},
		map[string]memory.PolicyRow{},
		inboxautoclaim.Line{
			CompanyCode: "Mfin", BatchNumber: "1", ProcessedDate: "05/01/2026",
			PolicyNo: "0100120260001", UploadedBy: "ADMINPNC",
		},
	)
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (inboxautoclaim.Repo, error) { return repo, nil },
	})
	require.NoError(t, err)

	ctx := context.Background()

	// Kode yang dikirim layar adalah kode yang diberikan daftar perusahaan — apa adanya.
	list, err := service.ListCompany(ctx, "ASM")
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, "Mfin", list[0].Code, "daftar perusahaan tidak mengubah kodenya")

	page, err := service.ListBatch(ctx, "ASM", inboxautoclaim.BatchFilter{Source: inboxautoclaim.SourceAneka,
		CompanyCode: list[0].Code,
	})
	require.NoError(t, err)
	require.Len(t, page.Item, 1, "menyaring dengan kode dari daftar perusahaan harus menemukan batch-nya")

	// Dan kebalikannya: kode yang di-uppercase sendiri TIDAK cocok, sama seperti di Oracle.
	// Penyimpanan memori tidak boleh lebih pemaaf daripada basis datanya — fake yang lebih
	// longgar membuat uji hijau justru pada keadaan yang gagal di produksi.
	upper, err := service.ListBatch(ctx, "ASM", inboxautoclaim.BatchFilter{Source: inboxautoclaim.SourceAneka,
		CompanyCode: "MFIN",
	})
	require.NoError(t, err)
	require.Empty(t, upper.Item, "fake harus sama ketatnya dengan Oracle soal besar kecil huruf")
}

func TestRingkasanMenghitungBATCHDanCocokDenganTotalGrid(t *testing.T) {
	// Aturan yang mengikat ringkasan ini: satuan hitungnya SAMA dengan satu baris grid.
	//
	// Uji ini yang menjaganya. Bila kelak kuerinya diubah menjadi menghitung baris klaim,
	// angkanya akan terlihat "lebih ramai" dan langsung berselisih dengan paginasi grid —
	// pengguna melihat "27" lalu mengeklik dan mendapat 3 baris, tanpa penjelasan apa pun.
	service, _ := layananContoh(t)
	ctx := context.Background()

	ringkasan, err := service.SummarizeCompany(ctx, "ASM", inboxautoclaim.SourceAneka)
	require.NoError(t, err)

	// Total ringkasan = total paginasi grid tanpa penyaring.
	semua, err := service.ListBatch(ctx, "ASM", inboxautoclaim.BatchFilter{Source: inboxautoclaim.SourceAneka})
	require.NoError(t, err)
	require.Equal(t, semua.Total, ringkasan.Total,
		"baris All wajib sama dengan total grid tanpa penyaring")

	// Dan tiap irisan = total paginasi grid setelah disaring perusahaan itu.
	for _, c := range ringkasan.Company {
		page, err := service.ListBatch(ctx, "ASM", inboxautoclaim.BatchFilter{Source: inboxautoclaim.SourceAneka,
			CompanyCode: c.Code,
		})
		require.NoError(t, err)
		require.Equalf(t, page.Total, c.BatchCount,
			"jumlah pada ringkasan %q wajib sama dengan total grid setelah disaring", c.Code)
	}
}

func TestRingkasanDiurutkanMenurunMenurutJumlah(t *testing.T) {
	// Irisan terbesar lebih dulu, di grafik maupun di tabel.
	service, _ := layananContoh(t)

	ringkasan, err := service.SummarizeCompany(context.Background(), "ASM", inboxautoclaim.SourceAneka)
	require.NoError(t, err)
	require.NotEmpty(t, ringkasan.Company)

	for i := 1; i < len(ringkasan.Company); i++ {
		require.GreaterOrEqual(t, ringkasan.Company[i-1].BatchCount, ringkasan.Company[i].BatchCount,
			"ringkasan wajib urut menurun")
	}
}

func TestRingkasanTetapMemuatPerusahaanYangTidakAdaDiMaster(t *testing.T) {
	// Alasannya sama dengan LEFT JOIN pada grid: dengan INNER JOIN, batch seperti ini
	// hilang dari ringkasan tetapi TETAP tampil di grid — angka dan isi jadi tidak cocok,
	// dan pengguna tidak punya cara tahu sebabnya.
	service, _ := layananContoh(t)

	ringkasan, err := service.SummarizeCompany(context.Background(), "ASM", inboxautoclaim.SourceAneka)
	require.NoError(t, err)

	var ada bool
	for _, c := range ringkasan.Company {
		if c.Code == "ZZZZ" {
			ada = true
			require.Equal(t, "", c.Name, "namanya kosong, irisannya tetap ada")
		}
	}
	require.True(t, ada, "perusahaan tanpa baris master wajib ikut dihitung")
}
func TestRingkasanHanyaMemuatPerusahaanYangPunyaBatchDiTabIni(t *testing.T) {
	// Aturan ini berbalik DUA KALI, dan keduanya berasal dari laporan Work Owner.
	//
	//   semula   hanya perusahaan yang punya batch
	//   lalu     seluruh perusahaan master, termasuk yang berjumlah 0 — karena panel
	//            hanya memuat 2 dari 19 perusahaan dan sisanya tidak dapat dipilih
	//   sekarang kembali ke yang pertama, PER TAB
	//
	// Yang mengubahnya: begitu ketiga tab ada, menyemai master membuat ketiganya
	// menampilkan daftar yang SAMA PERSIS — masternya memang satu. Panel yang menjadi
	// satu-satunya penyaring jadi penuh baris yang bila diklik menghasilkan grid kosong,
	// dan itu dilaporkan sebagai "penyaringnya tidak berfungsi" (2026-09-20).
	//
	// Laporan kedua pun sebenarnya bukan tentang master: yang dicari Work Owner data
	// Asuransi Kredit, yang saat itu belum punya tabnya sendiri.
	service, _ := layananContoh(t)

	ringkasan, err := service.SummarizeCompany(context.Background(), "ASM", inboxautoclaim.SourceAneka)
	require.NoError(t, err)

	kode := map[string]int{}
	for _, c := range ringkasan.Company {
		kode[c.Code] = c.BatchCount
	}

	// SRVY dan KRDU ada di master tetapi tidak punya batch pada tab ANEKA — keduanya
	// TIDAK boleh muncul. SRVY justru punya batch pada tab Travel, dan itu yang membuat
	// kedua tab berbeda.
	require.NotContains(t, kode, "SRVY")
	require.NotContains(t, kode, "KRDU")

	// ZZZZ punya batch tetapi TIDAK ada di master — ia TETAP ikut, karena barisnya juga
	// tampil di grid dan justru baris itulah yang tidak akan pernah berhasil diproses.
	require.Contains(t, kode, "ZZZZ")
	require.Positive(t, kode["ZZZZ"])

	// Setiap baris yang tampil dapat dipakai menyaring dan menghasilkan baris, bukan
	// tabel kosong. Inilah yang dilaporkan rusak.
	for _, c := range ringkasan.Company {
		require.Positive(t, c.BatchCount, "baris berjumlah nol tidak boleh tampil: %s", c.Code)
	}

	// Total tetap menghitung BATCH, bukan jumlah perusahaan.
	semua, err := service.ListBatch(context.Background(), "ASM", inboxautoclaim.BatchFilter{Source: inboxautoclaim.SourceAneka})
	require.NoError(t, err)
	require.Equal(t, semua.Total, ringkasan.Total)
}

// Ketiga tab menghasilkan daftar perusahaan yang BERBEDA.
//
// Ini yang benar-benar dilaporkan Work Owner: "filter untuk ketiga tab seharusnya
// hasilnya berbeda". Selama ringkasan dihitung dari master, ketiganya identik dan
// laporan itu tidak dapat ditangkap uji mana pun.
func TestRingkasanTiapTabBerbeda(t *testing.T) {
	service, _ := layananContoh(t)

	kodePerTab := map[inboxautoclaim.Source][]string{}
	for _, source := range inboxautoclaim.AllSource() {
		ringkasan, err := service.SummarizeCompany(context.Background(), "ASM", source)
		require.NoError(t, err)

		var kode []string
		for _, c := range ringkasan.Company {
			kode = append(kode, c.Code)
		}
		kodePerTab[source] = kode
	}

	require.NotEqual(t, kodePerTab[inboxautoclaim.SourceAneka], kodePerTab[inboxautoclaim.SourceKredit])
	require.NotEqual(t, kodePerTab[inboxautoclaim.SourceAneka], kodePerTab[inboxautoclaim.SourceTravel])
	require.NotEqual(t, kodePerTab[inboxautoclaim.SourceKredit], kodePerTab[inboxautoclaim.SourceTravel])
}

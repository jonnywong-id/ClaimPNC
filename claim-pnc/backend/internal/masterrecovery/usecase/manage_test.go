package usecase_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterrecovery"
	"claim-pnc/internal/masterrecovery/repo/memory"
	"claim-pnc/internal/masterrecovery/usecase"
	"claim-pnc/internal/masterrecovery/virtualaccount"
)

const portalUji = "ASM"

// bangun merakit layanan di atas penyimpanan memori dan penerbit tiruan.
//
// Keduanya adalah adapter kedua di balik seam yang sama dengan yang dipakai produksi,
// sehingga yang diuji di sini adalah aturan modulnya — bukan tiruan aturan.
func bangun(t *testing.T) (*usecase.Service, *memory.Repo, *virtualaccount.Fake) {
	t.Helper()

	repo := memory.NewSampleRepo()
	issuer := virtualaccount.NewFake()

	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (masterrecovery.Repo, error) {
			if !strings.EqualFold(strings.TrimSpace(alias), portalUji) {
				// Portal yang tidak dikenal WAJIB gagal, tidak pernah jatuh ke portal utama
				// sebagai cadangan (`R-20`).
				return nil, errors.New("portal tidak tersedia")
			}
			return repo, nil
		},
		Issuer: issuer,
		// Jam dipatok supaya daftar tahun tidak berubah arti setiap pergantian tahun dan
		// membuat pengujian gagal tanpa ada yang menyentuh kode.
		Now: func() time.Time { return time.Date(2026, 9, 19, 0, 0, 0, 0, time.UTC) },
	})
	require.NoError(t, err)
	return service, repo, issuer
}

// TestSisaDihitungServerDanMengabaikanKirimanKlien mengunci keputusan bahwa rumus Sisa
// hidup di satu tempat.
//
// Layar tetap menghitungnya untuk diperlihatkan seketika; yang MENGIKAT adalah yang
// dihitung server. Tanpa ini, klien yang keliru — atau yang sengaja — dapat menyimpan sisa
// yang tidak sesuai dengan nilai klaim dan pembayarannya.
func TestSisaDihitungServerDanMengabaikanKirimanKlien(t *testing.T) {
	service, repo, _ := bangun(t)

	tersimpan, err := service.Save(context.Background(), portalUji, masterrecovery.Recovery{
		PrincipalName: "PT CONTOH",
		Year:          "2026",
		Remark:        "pengembalian sebagian",
		CasePosition:  "dalam proses",
		ClaimAmount:   160_000,
		Payment:       5_000,
		// Sengaja diisi angka yang salah. Ia harus DIABAIKAN.
		Remainder: 999_999,
	})
	require.NoError(t, err)
	require.Equal(t, masterrecovery.Amount(155_000), tersimpan.Remainder)

	// Yang benar-benar masuk penyimpanan pun harus angka server, bukan angka klien.
	disimpan := repo.Saved()
	require.Len(t, disimpan, 1)
	require.Equal(t, masterrecovery.Amount(155_000), disimpan[0].Remainder)
}

// TestNomorBatchDiterbitkanPenyimpananDanBertambah memastikan nomor tidak pernah datang
// dari pemanggil.
func TestNomorBatchDiterbitkanPenyimpananDanBertambah(t *testing.T) {
	service, _, _ := bangun(t)
	ctx := context.Background()

	isian := masterrecovery.Recovery{
		PrincipalName: "PT CONTOH",
		Year:          "2026",
		Remark:        "keterangan",
		CasePosition:  "posisi",
		// Nomor karangan dari klien. Ia harus diabaikan.
		Batch: 4321,
	}

	pertama, err := service.Save(ctx, portalUji, isian)
	require.NoError(t, err)
	require.Equal(t, int64(1), pertama.Batch)

	kedua, err := service.Save(ctx, portalUji, isian)
	require.NoError(t, err)
	require.Equal(t, int64(2), kedua.Batch)
}

// TestIdentitasPolisDicariServerBukanDipercayaDariKlien mengunci alasan keempat kolom
// identitas tidak diterima dari badan permintaan.
func TestIdentitasPolisDicariServerBukanDipercayaDariKlien(t *testing.T) {
	service, _, _ := bangun(t)

	tersimpan, err := service.Save(context.Background(), portalUji, masterrecovery.Recovery{
		PrincipalName: "PT CONTOH",
		Year:          "2026",
		Remark:        "keterangan",
		CasePosition:  "posisi",
		PolicyNo:      "CONTOH-POLIS-0001",
		// Nilai karangan dari klien. Seluruhnya harus ditimpa hasil pencarian.
		BusinessID:  "PALSU",
		BranchID:    "PALSU",
		AgentID:     "PALSU",
		MarketingID: "PALSU",
	})
	require.NoError(t, err)
	require.Equal(t, "01", tersimpan.BusinessID)
	require.Equal(t, "001", tersimpan.BranchID)
	require.Equal(t, "AG0001", tersimpan.AgentID)
	require.Equal(t, "MO0001", tersimpan.MarketingID)
}

// TestPolisTidakDitemukanTidakMembatalkanPenyimpanan mengunci kesetaraan perilaku dengan
// sistem lama.
//
// `Insert_mst_recoveryKlaimASM` tidak punya satu pun prasyarat yang menghentikan alur
// ketika pencarian polis kembali kosong. Menolak menyimpan di sini akan membuat petugas
// kehilangan seluruh isian karena basis data pihak lain sedang mati — sementara yang
// dicatat adalah uang yang sudah diterima.
func TestPolisTidakDitemukanTidakMembatalkanPenyimpanan(t *testing.T) {
	service, _, _ := bangun(t)

	tersimpan, err := service.Save(context.Background(), portalUji, masterrecovery.Recovery{
		PrincipalName: "PT CONTOH",
		Year:          "2026",
		Remark:        "keterangan",
		CasePosition:  "posisi",
		PolicyNo:      "POLIS-YANG-TIDAK-ADA",
		BusinessID:    "PALSU",
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), tersimpan.Batch, "batch tetap tersimpan")
	require.Empty(t, tersimpan.BusinessID, "identitas dikosongkan, bukan diisi kiriman klien")
	require.Empty(t, tersimpan.MarketingID)
}

// TestIsianTidakLengkapDitolakSebelumMenyentuhPenyimpanan memastikan validasi berjalan
// lebih dulu.
func TestIsianTidakLengkapDitolakSebelumMenyentuhPenyimpanan(t *testing.T) {
	service, repo, _ := bangun(t)

	_, err := service.Save(context.Background(), portalUji, masterrecovery.Recovery{})

	var validasi *masterrecovery.ValidationError
	require.ErrorAs(t, err, &validasi)
	require.NotEmpty(t, validasi.Violation)
	require.Empty(t, repo.Saved(), "tidak ada yang tersimpan saat isian ditolak")
}

// TestPortalTidakDikenalDitolak mengunci `R-20`: portal yang tidak dapat dilayani tidak
// pernah jatuh ke portal utama sebagai cadangan.
func TestPortalTidakDikenalDitolak(t *testing.T) {
	service, repo, _ := bangun(t)

	_, err := service.Save(context.Background(), "ENTITAS-LAIN", masterrecovery.Recovery{
		PrincipalName: "PT CONTOH",
		Year:          "2026",
		Remark:        "keterangan",
		CasePosition:  "posisi",
	})
	require.Error(t, err)
	require.Empty(t, repo.Saved())
}

// TestVADipakaiUlangUntukPrincipalYangSudahPunya mengunci langkah pertama
// `GeneratedVAClaimRecovery`.
//
// Menerbitkan VA kedua untuk principal yang sama berarti dana masuk ke rekening yang tidak
// diawasi siapa pun — itulah sebabnya pemeriksaan ini mendahului panggilan ke layanan luar.
func TestVADipakaiUlangUntukPrincipalYangSudahPunya(t *testing.T) {
	service, _, issuer := bangun(t)

	// Penerbit dibuat gagal: bila ia sampai ditembak, test ini akan gagal — dan itulah
	// tepatnya yang hendak dibuktikan.
	issuer.SetError(masterrecovery.ErrIssuerUnreachable)

	contoh := memory.SamplePrincipal()[0]
	hasil, err := service.IssueVirtualAccount(context.Background(), portalUji, masterrecovery.VirtualAccountRequest{
		ClientID:      contoh.ClientID,
		PrincipalName: contoh.Name,
		Email:         "petugas@example.invalid",
	})
	require.NoError(t, err)
	require.True(t, hasil.Reused, "nomor diambil dari master, bukan diterbitkan ulang")
	require.Equal(t, contoh.VirtualAccountNumber, hasil.Number)
}

// TestVAPencocokanPrincipalMengabaikanBesarKecilHuruf memastikan peniruan `upper(...)`
// benar-benar berlaku lewat jalur usecase, bukan hanya di fungsi pembanding.
func TestVAPencocokanPrincipalMengabaikanBesarKecilHuruf(t *testing.T) {
	service, _, issuer := bangun(t)
	issuer.SetError(masterrecovery.ErrIssuerUnreachable)

	contoh := memory.SamplePrincipal()[0]
	hasil, err := service.IssueVirtualAccount(context.Background(), portalUji, masterrecovery.VirtualAccountRequest{
		ClientID:      strings.ToLower(contoh.ClientID),
		PrincipalName: strings.ToLower(contoh.Name),
		Email:         "petugas@example.invalid",
	})
	require.NoError(t, err)
	require.True(t, hasil.Reused)
}

// TestVABaruDiterbitkanDanTercatatDiMaster mengunci langkah ketiga: penerbitan diikuti
// pencatatan, supaya penerbitan berikutnya menemukannya.
func TestVABaruDiterbitkanDanTercatatDiMaster(t *testing.T) {
	service, _, _ := bangun(t)
	ctx := context.Background()

	permintaan := masterrecovery.VirtualAccountRequest{
		ClientID:      "PRINCIPAL-BARU-001",
		PrincipalName: "PT PRINCIPAL BARU",
		Email:         "petugas@example.invalid",
	}

	pertama, err := service.IssueVirtualAccount(ctx, portalUji, permintaan)
	require.NoError(t, err)
	require.False(t, pertama.Reused)
	require.NotEmpty(t, pertama.Number)

	// Penerbitan kedua untuk principal yang sama harus memakai ulang, bukan menerbitkan
	// nomor baru — bukti bahwa yang pertama benar-benar tercatat.
	kedua, err := service.IssueVirtualAccount(ctx, portalUji, permintaan)
	require.NoError(t, err)
	require.True(t, kedua.Reused)
	require.Equal(t, pertama.Number, kedua.Number)
}

// TestVATidakTercatatKetikaPenerbitGagal memastikan kegagalan layanan luar tidak
// meninggalkan baris master bernomor kosong.
func TestVATidakTercatatKetikaPenerbitGagal(t *testing.T) {
	service, repo, issuer := bangun(t)
	issuer.SetError(masterrecovery.ErrIssuerRejected)

	sebelum, err := repo.ListPrincipal(context.Background())
	require.NoError(t, err)

	_, err = service.IssueVirtualAccount(context.Background(), portalUji, masterrecovery.VirtualAccountRequest{
		ClientID:      "PRINCIPAL-BARU-002",
		PrincipalName: "PT PRINCIPAL BARU DUA",
		Email:         "petugas@example.invalid",
	})
	require.ErrorIs(t, err, masterrecovery.ErrIssuerRejected)

	sesudah, err := repo.ListPrincipal(context.Background())
	require.NoError(t, err)
	require.Len(t, sesudah, len(sebelum), "tidak ada principal yang tercatat saat penerbitan gagal")
}

// TestBuktiBayarTersimpanDenganPengunggahnya mengunci pengisian kolom INPUTOPERATOR.
func TestBuktiBayarTersimpanDenganPengunggahnya(t *testing.T) {
	service, repo, _ := bangun(t)

	id, err := service.SaveDocument(context.Background(), portalUji, masterrecovery.Document{
		Name:       "bukti-transfer.pdf",
		MimeType:   "application/pdf",
		Content:    []byte("%PDF-1.4 contoh"),
		UploadedBy: "PETUGAS01",
	})
	require.NoError(t, err)
	require.NotEmpty(t, id)

	tersimpan, ada := repo.Document(id)
	require.True(t, ada)
	require.Equal(t, "PETUGAS01", tersimpan.UploadedBy)
}

// TestDaftarTahunTerbaruLebihDulu mengunci urutan pilihan Tahun.
//
// Batch yang dicatat hampir selalu tahun berjalan, dan menaruhnya di puncak menghemat satu
// gulir pada setiap pemakaian.
func TestDaftarTahunTerbaruLebihDulu(t *testing.T) {
	service, _, _ := bangun(t)

	tahun := service.Years(context.Background(), portalUji)
	require.NotEmpty(t, tahun)
	require.Equal(t, "2027", tahun[0], "satu tahun ke depan, untuk tahun buku yang sudah dibuka")
	require.Equal(t, "2016", tahun[len(tahun)-1], "sepuluh tahun ke belakang")

	// Setiap pilihan harus lolos aturan bentuk yang ditegakkan saat menyimpan. Tanpa ini,
	// dropdown dapat menawarkan nilai yang kemudian ditolak servernya sendiri.
	for _, t2 := range tahun {
		require.Empty(t, masterrecovery.CheckYear(t2), "tahun %q ditawarkan tetapi ditolak validasi", t2)
	}
}

// TestBarisKlaimDibacaTanpaMenyentuhPenyimpanan mengunci bahwa unggahan CSV hanyalah
// pembacaan — persis seperti sistem lama menyusunnya di klipboard sebelum menyimpan.
func TestBarisKlaimDibacaTanpaMenyentuhPenyimpanan(t *testing.T) {
	service, repo, _ := bangun(t)

	baris, ditolak, err := service.ReadClaimLine(portalUji, strings.NewReader("POL-1;1000\nPOL-2;2000\n"))
	require.NoError(t, err)
	require.Empty(t, ditolak)
	require.Len(t, baris, 2)
	require.Empty(t, repo.Saved(), "membaca berkas tidak menyimpan batch apa pun")
}

// TestBarisKlaimIkutTersimpanSebagaiBagianBatch memastikan daftar polis benar-benar
// terbawa sampai ke penyimpanan.
func TestBarisKlaimIkutTersimpanSebagaiBagianBatch(t *testing.T) {
	service, repo, _ := bangun(t)

	_, err := service.Save(context.Background(), portalUji, masterrecovery.Recovery{
		PrincipalName: "PT CONTOH",
		Year:          "2026",
		Remark:        "keterangan",
		CasePosition:  "posisi",
		ClaimLine: []masterrecovery.ClaimLine{
			{PolicyNo: "POL-1", ClaimAmount: 1_000},
			{PolicyNo: "POL-2", ClaimAmount: 2_000},
		},
	})
	require.NoError(t, err)

	disimpan := repo.Saved()
	require.Len(t, disimpan, 1)
	require.Len(t, disimpan[0].ClaimLine, 2)
	require.Equal(t, "POL-2", disimpan[0].ClaimLine[1].PolicyNo)
}

// TestLayananMenolakBahanYangTidakLengkap memastikan rakitan setengah jadi gagal saat
// start, bukan saat pengguna sedang bekerja.
func TestLayananMenolakBahanYangTidakLengkap(t *testing.T) {
	_, err := usecase.NewService(usecase.Options{Issuer: virtualaccount.NewFake()})
	require.Error(t, err)

	_, err = usecase.NewService(usecase.Options{
		RepoSelector: func(string) (masterrecovery.Repo, error) { return memory.NewSampleRepo(), nil },
	})
	require.Error(t, err)
}

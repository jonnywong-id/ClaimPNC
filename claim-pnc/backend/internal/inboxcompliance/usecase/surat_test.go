package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxcompliance"
	"claim-pnc/internal/inboxcompliance/repo/memory"
	"claim-pnc/internal/inboxcompliance/usecase"
)

// perenderPalsu merekam dokumen yang diminta digambar, bukan membentuk PDF sungguhan.
//
// Itulah gunanya seam RejectLetterRenderer: yang diuji di sini PEMETAANNYA — isian layar
// mana jatuh ke baris surat mana — bukan tata letak PDF-nya, yang sudah diuji paket
// rejectpdf.
type perenderPalsu struct {
	terakhir inboxcompliance.RejectLetterDocument
	jumlah   int
	isi      []byte
	galat    error
}

func (p *perenderPalsu) Render(
	d inboxcompliance.RejectLetterDocument,
) ([]byte, error) {
	p.terakhir = d
	p.jumlah++
	if p.galat != nil {
		return nil, p.galat
	}
	if p.isi == nil {
		return []byte("%PDF-palsu"), nil
	}
	return p.isi, nil
}

func serviceDenganPerender(
	t *testing.T,
	store *memory.Store,
	berkas inboxcompliance.DocumentStore,
	perender inboxcompliance.RejectLetterRenderer,
) *usecase.Service {
	t.Helper()

	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (inboxcompliance.Repo, error) {
			if alias != portal {
				return nil, errors.New("portal tidak dikenal: " + alias)
			}
			return store, nil
		},
		Clock: fixedClock{at: now},
		DocumentStoreSelector: func(alias string) (inboxcompliance.DocumentStore, error) {
			if alias != portal {
				return nil, errors.New("portal tidak dikenal: " + alias)
			}
			return berkas, nil
		},
		RejectLetterRenderer: perender,
	})
	require.NoError(t, err)
	return service
}

// suratContoh adalah isian form, seluruhnya KARANGAN (`D-69`).
func suratContoh() usecase.RejectLetterInput {
	return usecase.RejectLetterInput{
		Reference:     claimKey,
		Recipient:     "Bagian Umum",
		Position:      "Kepala Bagian",
		PatientName:   "PESERTA CONTOH / REG-0001",
		IncidentPlace: "RS CONTOH",
		IncidentDate:  "01 September 2026",
		DischargeDate: "05 September 2026",
		PaidAmount:    "Rp 1.500.000,-",
		PaymentDate:   "20 September 2026",
		Reasons:       []string{"Alasan contoh pertama", "", "Alasan contoh kedua"},
	}
}

// Surat terbit sebagai DOKUMEN KLAIM, bukan unduhan sekali pakai.
//
// Pega melampirkannya — `DownloadPDFReject` langkah 13-25. Yang diperiksa di sini: berkas
// terkirim ke penyimpanan, dan barisnya tercatat dengan nama serta kategori yang ditiru
// dari `local.Category`.
func TestSuratPenolakanTersimpanSebagaiDokumenKlaim(t *testing.T) {
	t.Parallel()

	store := paStore()
	store.SeedRejectPrefill(claimKey, inboxcompliance.RejectPrefill{
		PatientName: "OBJEK PERTANGGUNGAN CONTOH",
	})
	berkas := &storePalsu{imageID: "IMG-SURAT-1"}
	perender := &perenderPalsu{isi: []byte("%PDF-isi")}
	service := serviceDenganPerender(t, store, berkas, perender)

	hasil, err := service.GenerateRejectLetter(
		context.Background(), portal, usecaseCaller(), suratContoh())
	require.NoError(t, err)

	require.Len(t, berkas.unggah, 1)
	require.Equal(t, inboxcompliance.RejectLetterFileName, berkas.unggah[0].FileName)
	require.Equal(t, "pdf", berkas.unggah[0].MimeTypeHint)
	require.Equal(t, []byte("%PDF-isi"), berkas.unggah[0].Content)

	require.Equal(t, inboxcompliance.RejectLetterFileName, hasil.Document.Name)
	require.Equal(t, inboxcompliance.RejectLetterCategory, hasil.Document.Category)
	require.Equal(t, "IMG-SURAT-1", hasil.Document.StorageID)
	require.False(t, hasil.Replaced, "penekanan pertama tidak mengganti apa pun")
}

// storeSurat adalah antrean PA yang nama tertanggungnya TERISI.
//
// Dipisahkan dari paStore() yang dipakai bersama: helper itu sengaja tidak membawa
// InsuredName, dan menambahinya akan mengubah masukan belasan uji lain yang tidak ada
// urusannya dengan surat. Nama di bawah KARANGAN (`D-69`).
func storeSurat() *memory.Store {
	return memory.NewStore(memory.Row{
		Workbasket: inboxcompliance.WorkbasketCompliance,
		CreatedAt:  now,
		Item: inboxcompliance.WorkItem{
			CaseID:      "PNC-2114",
			Reference:   claimKey,
			GroupPanel:  inboxcompliance.GroupPanelPersonalAccident,
			InsuredName: "PT CONTOH SEJAHTERA",
		},
	})
}

// Enam isian surat datang dari klaim, pra-isi, dan jam — bukan dari form.
func TestSuratDirakitDariKlaimDanPraIsi(t *testing.T) {
	t.Parallel()

	store := storeSurat()
	store.SeedRejectPrefill(claimKey, inboxcompliance.RejectPrefill{
		PatientName: "OBJEK PERTANGGUNGAN CONTOH",
	})
	perender := &perenderPalsu{}
	service := serviceDenganPerender(t, store, &storePalsu{imageID: "IMG-1"}, perender)

	_, err := service.GenerateRejectLetter(
		context.Background(), portal, usecaseCaller(), suratContoh())
	require.NoError(t, err)

	surat := perender.terakhir

	// Dari pra-isi — baris "Perihal" dan "Lampiran".
	require.Equal(t, "OBJEK PERTANGGUNGAN CONTOH", surat.SubjectName)

	// Dari klaim. Ditaruh pada isian badan usaha karena klaim tidak membawa penanda
	// perorangan/badan usaha — lihat catatan di buildRejectLetter.
	require.Equal(t, "PT CONTOH SEJAHTERA", surat.RecipientCompany)
	require.Empty(t, surat.RecipientPerson)

	// Alamat tergambar kosong: ia ada di snapshot polis, bukan di tabel klaim.
	require.Empty(t, surat.RecipientAddress)

	// Dari jam.
	require.NotEmpty(t, surat.LetterDate)
	require.Contains(t, surat.LetterNumber, "/CL.AHID.ASM/")
}

/*
Isian form TIDAK terbawa ke surat, dan itu disengaja.

Templat `HTML/RejectRefundLetter-HTML.xml` tidak punya satu pun merge field untuk
kedelapan isian form maupun grid Alasan — baris a sampai e dan daftar alasan tergambar
kosong untuk diisi tangan. Work Owner menetapkan templat itu diikuti apa adanya
(2026-10-08).

Uji ini mengunci ketetapan tersebut dari arah yang benar: ia tidak memeriksa bahwa
isiannya "hilang", melainkan bahwa seluruh isian surat TIDAK MEMUAT teks yang diketik
petugas. Dengan begitu, siapa pun yang kelak menambahkan isian form ke surat akan
membuat uji ini merah — dan membaca alasannya di sini.
*/
func TestIsianFormTidakTerbawaKeSurat(t *testing.T) {
	t.Parallel()

	store := paStore()
	store.SeedRejectPrefill(claimKey, inboxcompliance.RejectPrefill{
		PatientName: "OBJEK PERTANGGUNGAN CONTOH",
	})
	perender := &perenderPalsu{}
	service := serviceDenganPerender(t, store, &storePalsu{imageID: "IMG-1"}, perender)

	// Isian yang mustahil muncul karena alasan lain — supaya pencarian di bawah tegas.
	isian := suratContoh()
	isian.Recipient = "KETIKAN-UP"
	isian.Position = "KETIKAN-JABATAN"
	isian.PatientName = "KETIKAN-PASIEN"
	isian.IncidentPlace = "KETIKAN-TEMPAT"
	isian.IncidentDate = "KETIKAN-TGL-KEJADIAN"
	isian.DischargeDate = "KETIKAN-TGL-KELUAR"
	isian.PaidAmount = "KETIKAN-NILAI"
	isian.PaymentDate = "KETIKAN-TGL-BAYAR"
	isian.Reasons = []string{"KETIKAN-ALASAN"}

	_, err := service.GenerateRejectLetter(
		context.Background(), portal, usecaseCaller(), isian)
	require.NoError(t, err)

	surat := perender.terakhir
	for _, isi := range []string{
		surat.LetterDate, surat.LetterNumber,
		surat.RecipientCompany, surat.RecipientPerson, surat.RecipientAddress,
		surat.SubjectName,
	} {
		require.NotContains(t, isi, "KETIKAN-",
			"isian form tidak boleh sampai ke surat")
	}
}

// Menekan tombol dua kali MENGGANTI suratnya, tidak menumpuk.
//
// Pega menghapus lampiran bernama sama lebih dulu. Tanpa itu, tiga penekanan meninggalkan
// tiga surat bernomor berbeda pada klaim yang sama dan tidak ada yang tahu mana yang
// berlaku — nomor suratnya berasal dari menit-detik, jadi ketiganya tampak sah.
func TestMenekanDuaKaliMenggantiSuratBukanMenumpuk(t *testing.T) {
	t.Parallel()

	store := paStore()
	store.SeedRejectPrefill(claimKey, inboxcompliance.RejectPrefill{
		PatientName: "OBJEK PERTANGGUNGAN CONTOH",
	})
	berkas := &storePalsu{imageID: "IMG-SURAT"}
	service := serviceDenganPerender(t, store, berkas, &perenderPalsu{})

	pertama, err := service.GenerateRejectLetter(
		context.Background(), portal, usecaseCaller(), suratContoh())
	require.NoError(t, err)
	require.False(t, pertama.Replaced)

	kedua, err := service.GenerateRejectLetter(
		context.Background(), portal, usecaseCaller(), suratContoh())
	require.NoError(t, err)
	require.True(t, kedua.Replaced, "surat lama seharusnya dihapus lebih dulu")

	// Berkas lama ikut dibuang dari penyimpanan, bukan ditinggalkan yatim.
	require.Len(t, berkas.hapus, 1)

	dokumen, err := store.FindDocuments(context.Background(), claimKey)
	require.NoError(t, err)

	var jumlahSurat int
	for _, d := range dokumen {
		if d.Name == inboxcompliance.RejectLetterFileName {
			jumlahSurat++
		}
	}
	require.Equal(t, 1, jumlahSurat, "hanya satu surat penolakan yang boleh tersisa")
}

// Perender gagal -> surat LAMA tetap utuh.
//
// Inilah sebab PDF dibentuk sebelum yang lama dihapus. Urutan sebaliknya meninggalkan
// klaim tanpa surat sama sekali karena kegagalan yang belum tentu permanen.
func TestPerenderGagalTidakMenghapusSuratLama(t *testing.T) {
	t.Parallel()

	store := paStore()
	store.SeedRejectPrefill(claimKey, inboxcompliance.RejectPrefill{
		PatientName: "OBJEK PERTANGGUNGAN CONTOH",
	})
	berkas := &storePalsu{imageID: "IMG-SURAT"}
	perender := &perenderPalsu{}
	service := serviceDenganPerender(t, store, berkas, perender)

	_, err := service.GenerateRejectLetter(
		context.Background(), portal, usecaseCaller(), suratContoh())
	require.NoError(t, err)

	perender.galat = errors.New("isian merusak tata letak")
	_, err = service.GenerateRejectLetter(
		context.Background(), portal, usecaseCaller(), suratContoh())
	require.Error(t, err)

	dokumen, err := store.FindDocuments(context.Background(), claimKey)
	require.NoError(t, err)

	var adaSurat bool
	for _, d := range dokumen {
		if d.Name == inboxcompliance.RejectLetterFileName {
			adaSurat = true
		}
	}
	require.True(t, adaSurat, "surat lama tidak boleh hilang karena pembentukan gagal")
	require.Empty(t, berkas.hapus, "tidak ada berkas yang boleh dihapus")
}

// Klaim yang tidak ada di antrean Compliance ditolak.
//
// Tanpa pemeriksaan ini, surat penolakan dapat ditempelkan ke klaim mana pun yang
// nomornya diketahui — termasuk yang sudah dibayar.
func TestSuratDitolakBilaKlaimTidakDiAntrean(t *testing.T) {
	t.Parallel()

	store := paStore()
	service := serviceDenganPerender(t, store, &storePalsu{}, &perenderPalsu{})

	isian := suratContoh()
	isian.Reference = "ASM-FW-GCNMFW-WORK PNC-9999"

	_, err := service.GenerateRejectLetter(
		context.Background(), portal, usecaseCaller(), isian)
	require.ErrorIs(t, err, inboxcompliance.ErrClaimNotInQueue)
}

// Tanpa perender terpasang, galatnya menyebut yang kurang — bukan galat internal.
func TestTanpaPerenderGalatnyaMenyebutYangKurang(t *testing.T) {
	t.Parallel()

	store := paStore()
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (inboxcompliance.Repo, error) { return store, nil },
		Clock:        fixedClock{at: now},
		DocumentStoreSelector: func(string) (inboxcompliance.DocumentStore, error) {
			return &storePalsu{}, nil
		},
	})
	require.NoError(t, err)

	_, err = service.GenerateRejectLetter(
		context.Background(), portal, usecaseCaller(), suratContoh())
	require.ErrorIs(t, err, inboxcompliance.ErrLetterRendererMissing)
}

// Tanpa layanan dokumen, surat tidak dapat disimpan — dan galatnya pun menyebutnya.
func TestTanpaLayananDokumenSuratTidakTerbit(t *testing.T) {
	t.Parallel()

	store := paStore()
	service, err := usecase.NewService(usecase.Options{
		RepoSelector:         func(string) (inboxcompliance.Repo, error) { return store, nil },
		Clock:                fixedClock{at: now},
		RejectLetterRenderer: &perenderPalsu{},
	})
	require.NoError(t, err)

	_, err = service.GenerateRejectLetter(
		context.Background(), portal, usecaseCaller(), suratContoh())
	require.ErrorIs(t, err, inboxcompliance.ErrDocumentServiceMissing)
}

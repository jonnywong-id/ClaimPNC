package usecase_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"claim-pnc/internal/inboxlaporanklaim"
	"claim-pnc/internal/inboxlaporanklaim/repo/memory"
	"claim-pnc/internal/inboxlaporanklaim/usecase"
	"claim-pnc/internal/platform/clock"
)

const portalAlias = "ASM"

// adminJakarta adalah petugas yang membuat sebagian besar berkas contoh; cabangnya 1001.
var adminJakarta = inboxlaporanklaim.Caller{
	Login:      "adminpnc",
	Name:       "Petugas Contoh",
	BranchCode: "1001",
}

func newService(t *testing.T) (*usecase.Service, *memory.Repo) {
	t.Helper()

	fixed := clock.FixedAt(time.Date(2026, 9, 19, 3, 0, 0, 0, time.UTC))
	repo := memory.NewRepo(memory.SampleOptions(fixed))

	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (inboxlaporanklaim.Repo, error) {
			if alias != portalAlias {
				return nil, errors.New("portal tidak tersedia")
			}
			return repo, nil
		},
		Clock: fixed,
	})
	if err != nil {
		t.Fatalf("membentuk layanan: %v", err)
	}
	return service, repo
}

func listOf(
	t *testing.T,
	service *usecase.Service,
	caller inboxlaporanklaim.Caller,
	query usecase.Query,
) usecase.ListResult {
	t.Helper()
	result, err := service.List(context.Background(), portalAlias, caller, query)
	if err != nil {
		t.Fatalf("mengambil daftar: %v", err)
	}
	return result
}

func TestServiceRefusesToStartWithoutClock(t *testing.T) {
	// Nilai bawaan jam sistem sengaja TIDAK disediakan: ia akan diam-diam benar di
	// produksi dan diam-diam salah di pengujian, sedangkan seluruh umur berkas di layar
	// ini dihitung darinya.
	_, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (inboxlaporanklaim.Repo, error) { return nil, nil },
	})
	if err == nil {
		t.Fatal("layanan terbentuk tanpa jam")
	}
}

func TestEachTabShowsOnlyItsOwnReports(t *testing.T) {
	service, _ := newService(t)

	// Cabang petugas dikosongkan supaya uji ini menguji ATURAN TAB, bukan batas cabang.
	// Keduanya diuji terpisah; menggabungkannya membuat kegagalan sulit dibaca.
	caller := inboxlaporanklaim.Caller{Login: "adminpnc"}

	suite := []struct {
		category inboxlaporanklaim.Category
		want     inboxlaporanklaim.Position
	}{
		{inboxlaporanklaim.CategoryOutstanding, inboxlaporanklaim.PositionOutstanding},
		{inboxlaporanklaim.CategoryUnregistered, inboxlaporanklaim.PositionNotRegistered},
		{inboxlaporanklaim.CategoryNotTransferred, inboxlaporanklaim.PositionNotTransferred},
	}

	for _, c := range suite {
		t.Run(string(c.category), func(t *testing.T) {
			result := listOf(t, service, caller, usecase.Query{Category: c.category})
			if result.Page.Total == 0 {
				t.Fatalf("tab %q kosong; berkas contoh seharusnya memuat setiap tab", c.category)
			}
			for _, report := range result.Page.Report {
				if report.Position != c.want {
					t.Fatalf("tab %q memuat berkas berposisi %q", c.category, report.Position)
				}
			}
		})
	}
}

func TestResolvedAndRejectedReportsAreHiddenFromEveryOpenTab(t *testing.T) {
	service, _ := newService(t)
	caller := inboxlaporanklaim.Caller{Login: "adminpnc"}

	// RCV-0011 dan RCV-0014 ditolak, RCV-0013 selesai. Kueri lama menyaringnya dengan
	// PYSTATUSWORK NOT IN ('Resolved-Completed','Resolved-Rejected') di setiap tab
	// terbuka; tab Data rejected justru mencarinya.
	open := listOf(t, service, caller, usecase.Query{
		Category:   inboxlaporanklaim.CategoryAll,
		Pagination: inboxlaporanklaim.Pagination{Size: inboxlaporanklaim.MaxPageSize},
	})
	for _, report := range open.Page.Report {
		switch report.ID {
		case "RCV-0011", "RCV-0013", "RCV-0014":
			t.Fatalf("berkas selesai/ditolak %q muncul di tab All data", report.ID)
		}
	}

	rejected := listOf(t, service, caller, usecase.Query{Category: inboxlaporanklaim.CategoryRejected})
	if rejected.Page.Total != 2 {
		t.Fatalf("tab Data rejected memuat %d berkas, ingin 2", rejected.Page.Total)
	}
}

func TestAcceptedTabShowsOnlyReportsWithAcceptanceNumber(t *testing.T) {
	service, _ := newService(t)

	result := listOf(t, service, inboxlaporanklaim.Caller{Login: "adminpnc"}, usecase.Query{
		Category: inboxlaporanklaim.CategoryAccepted,
	})

	// RCV-0004 dan RCV-0010 punya nomor akseptasi di berkas contoh.
	if result.Page.Total != 2 {
		t.Fatalf("tab akseptasi memuat %d berkas, ingin 2", result.Page.Total)
	}
	for _, report := range result.Page.Report {
		if !report.Registered() {
			t.Fatalf("berkas %q berakseptasi tetapi tidak bernomor klaim", report.ID)
		}
	}
}

func TestBranchOfCallerIsTheDefaultDataBoundary(t *testing.T) {
	service, _ := newService(t)

	result := listOf(t, service, adminJakarta, usecase.Query{
		Category:   inboxlaporanklaim.CategoryAll,
		Pagination: inboxlaporanklaim.Pagination{Size: inboxlaporanklaim.MaxPageSize},
	})

	if result.Page.Total == 0 {
		t.Fatal("daftar kosong untuk cabang petugas")
	}
	for _, report := range result.Page.Report {
		if report.BranchCode != adminJakarta.BranchCode {
			t.Fatalf("berkas cabang %q bocor ke petugas cabang %q",
				report.BranchCode, adminJakarta.BranchCode)
		}
	}
}

func TestChoosingRegionWidensBeyondOwnBranch(t *testing.T) {
	service, _ := newService(t)

	// Kanwil 01 memuat cabang 1001 dan 1002. Petugas bercabang 1001 yang memilihnya
	// karena itu ikut melihat berkas cabang 1002 — dan itulah satu-satunya alasan
	// dropdown Kanwil dibuat. Lihat catatan ANDAIAN pada usecase.buildFilter.
	result := listOf(t, service, adminJakarta, usecase.Query{
		Category:   inboxlaporanklaim.CategoryAll,
		RegionCode: "01",
		Pagination: inboxlaporanklaim.Pagination{Size: inboxlaporanklaim.MaxPageSize},
	})

	branch := map[string]bool{}
	for _, report := range result.Page.Report {
		branch[report.BranchCode] = true
	}
	if !branch["1002"] {
		t.Fatal("memilih kanwil tidak melebarkan daftar ke cabang lain di kanwil yang sama")
	}
	if branch["2001"] || branch["3001"] {
		t.Fatalf("cabang di luar kanwil 01 ikut tampil: %v", branch)
	}
}

func TestKeywordMatchesRegisterNumberExactlyNotPartially(t *testing.T) {
	service, _ := newService(t)
	caller := inboxlaporanklaim.Caller{Login: "adminpnc"}

	exact := listOf(t, service, caller, usecase.Query{
		Category: inboxlaporanklaim.CategoryAll,
		Keyword:  "RCV-0003",
	})
	if exact.Page.Total != 1 {
		t.Fatalf("pencarian persis menghasilkan %d baris, ingin 1", exact.Page.Total)
	}

	// Kueri lama memakai `a.pyid = '<cari>'`, bukan LIKE. Mengubahnya menjadi pencarian
	// sebagian membuat kueri berhenti memakai index pada tabel berpuluh juta baris.
	partial := listOf(t, service, caller, usecase.Query{
		Category: inboxlaporanklaim.CategoryAll,
		Keyword:  "RCV",
	})
	if partial.Page.Total != 0 {
		t.Fatalf("pencarian sebagian menghasilkan %d baris, ingin 0", partial.Page.Total)
	}
}

func TestBusinessLineFilterSeparatesSpecialGroupFromNonMBU(t *testing.T) {
	service, _ := newService(t)
	caller := inboxlaporanklaim.Caller{Login: "adminpnc"}

	special := listOf(t, service, caller, usecase.Query{
		Category:     inboxlaporanklaim.CategoryAll,
		BusinessLine: inboxlaporanklaim.BusinessLineSpecialGroup,
		Pagination:   inboxlaporanklaim.Pagination{Size: inboxlaporanklaim.MaxPageSize},
	})
	nonMBU := listOf(t, service, caller, usecase.Query{
		Category:     inboxlaporanklaim.CategoryAll,
		BusinessLine: inboxlaporanklaim.BusinessLineNonMBU,
		Pagination:   inboxlaporanklaim.Pagination{Size: inboxlaporanklaim.MaxPageSize},
	})

	if special.Page.Total == 0 || nonMBU.Page.Total == 0 {
		t.Fatalf("salah satu pilihan kosong: khusus=%d non-mbu=%d",
			special.Page.Total, nonMBU.Page.Total)
	}

	// Keduanya harus saling lepas: kelompok khusus justru DIKECUALIKAN dari Non-MBU.
	inSpecial := map[string]bool{}
	for _, report := range special.Page.Report {
		inSpecial[report.ID] = true
	}
	for _, report := range nonMBU.Page.Report {
		if inSpecial[report.ID] {
			t.Fatalf("berkas %q muncul di kelompok khusus DAN Non-MBU", report.ID)
		}
	}
}

func TestUnknownFilterValuesAreRejected(t *testing.T) {
	service, _ := newService(t)

	_, err := service.List(context.Background(), portalAlias, adminJakarta, usecase.Query{
		Category: "bukan-tab",
	})
	if !errors.Is(err, inboxlaporanklaim.ErrUnknownCategory) {
		t.Fatalf("galat tab tidak dikenal = %v", err)
	}

	_, err = service.List(context.Background(), portalAlias, adminJakarta, usecase.Query{
		Category:     inboxlaporanklaim.CategoryAll,
		BusinessLine: "bukan-lini",
	})
	if !errors.Is(err, inboxlaporanklaim.ErrUnknownBusinessLine) {
		t.Fatalf("galat lini bisnis tidak dikenal = %v", err)
	}
}

func TestPagingNeverRepeatsOrLosesARow(t *testing.T) {
	service, _ := newService(t)
	caller := inboxlaporanklaim.Caller{Login: "adminpnc"}

	seen := map[string]bool{}
	total := 0

	for page := 1; page <= 5; page++ {
		result := listOf(t, service, caller, usecase.Query{
			Category:   inboxlaporanklaim.CategoryAll,
			Pagination: inboxlaporanklaim.Pagination{Page: page, Size: 3},
		})
		total = result.Page.Total

		for _, report := range result.Page.Report {
			if seen[report.ID] {
				t.Fatalf("berkas %q muncul di lebih dari satu halaman", report.ID)
			}
			seen[report.ID] = true
		}
		if len(result.Page.Report) == 0 {
			break
		}
	}

	if len(seen) != total {
		t.Fatalf("terkumpul %d berkas dari %d yang dilaporkan total", len(seen), total)
	}
}

func TestPageBeyondTheLastIsEmptyNotAnError(t *testing.T) {
	service, _ := newService(t)

	result := listOf(t, service, inboxlaporanklaim.Caller{Login: "adminpnc"}, usecase.Query{
		Category:   inboxlaporanklaim.CategoryAll,
		Pagination: inboxlaporanklaim.Pagination{Page: 99, Size: 10},
	})
	if len(result.Page.Report) != 0 {
		t.Fatalf("halaman ke-99 memuat %d baris", len(result.Page.Report))
	}
	if result.Page.Total == 0 {
		t.Fatal("total hilang pada halaman di luar jangkauan")
	}
}

func TestBadgeCountsUseTheSameFilterAsTheTableBelowThem(t *testing.T) {
	service, _ := newService(t)

	// Pencacah yang mengabaikan penyaring akan menyebut jumlah seluruh cabang padahal
	// yang tampil satu cabang saja — dua angka berdampingan yang tidak berhubungan.
	all := listOf(t, service, inboxlaporanklaim.Caller{Login: "adminpnc"}, usecase.Query{
		Category: inboxlaporanklaim.CategoryAll,
	})
	scoped := listOf(t, service, adminJakarta, usecase.Query{
		Category: inboxlaporanklaim.CategoryAll,
	})

	if scoped.Summary.Total >= all.Summary.Total {
		t.Fatalf("pencacah bercabang %d tidak lebih kecil dari tanpa cabang %d",
			scoped.Summary.Total, all.Summary.Total)
	}
	if scoped.Summary.Total != scoped.Page.Total {
		t.Fatalf("lencana All data %d, isi tabel %d — keduanya harus sama",
			scoped.Summary.Total, scoped.Page.Total)
	}
}

func TestBadgePartsAddUpToTheTotal(t *testing.T) {
	service, _ := newService(t)

	result := listOf(t, service, inboxlaporanklaim.Caller{Login: "adminpnc"}, usecase.Query{
		Category: inboxlaporanklaim.CategoryAll,
	})
	s := result.Summary

	// Ketiga posisi saling lepas dan menutupi seluruh berkas terbuka. Bila jumlahnya
	// tidak sama dengan total, salah satu berkas berposisi yang tidak dikenali.
	if s.NotTransferred+s.Unregistered+s.Outstanding != s.Total {
		t.Fatalf("%d + %d + %d != %d",
			s.NotTransferred, s.Unregistered, s.Outstanding, s.Total)
	}
}

func TestCommunicationTabsShowOnlyTheCallersOwnReports(t *testing.T) {
	service, _ := newService(t)

	for _, category := range []inboxlaporanklaim.Category{
		inboxlaporanklaim.CategoryMessageUnanswered,
		inboxlaporanklaim.CategoryMessageWaiting,
		inboxlaporanklaim.CategoryMessageReplied,
	} {
		result := listOf(t, service, inboxlaporanklaim.Caller{Login: "adminpnc"}, usecase.Query{
			Category:   category,
			Pagination: inboxlaporanklaim.Pagination{Size: inboxlaporanklaim.MaxPageSize},
		})
		for _, report := range result.Page.Report {
			if report.CreatedBy != "adminpnc" {
				t.Fatalf("tab %q memuat berkas milik %q", category, report.CreatedBy)
			}
			if report.LastMessage == "" {
				t.Fatalf("tab %q memuat berkas tanpa pesan terakhir", category)
			}
		}
	}
}

func TestLastMessageIsEmptyOutsideCommunicationTabs(t *testing.T) {
	service, _ := newService(t)

	result := listOf(t, service, inboxlaporanklaim.Caller{Login: "adminpnc"}, usecase.Query{
		Category:   inboxlaporanklaim.CategoryAll,
		Pagination: inboxlaporanklaim.Pagination{Size: inboxlaporanklaim.MaxPageSize},
	})
	for _, report := range result.Page.Report {
		if report.LastMessage != "" {
			t.Fatalf("berkas %q membawa pesan terakhir di tab yang bukan komunikasi", report.ID)
		}
	}
}

func TestNewReportIsBornBlankInTheNotTransferredTab(t *testing.T) {
	service, _ := newService(t)

	saved, err := service.Create(context.Background(), portalAlias, adminJakarta)
	if err != nil {
		t.Fatalf("membuat berkas: %v", err)
	}

	// Tombol "Buat Baru" di sistem lama membuat berkas KOSONG lalu membukanya. Yang
	// terisi hanyalah yang diturunkan dari petugas penekannya.
	if saved.ClaimNumber != "" || saved.Transferred {
		t.Fatalf("berkas baru tidak lahir kosong: klaim=%q diserahkan=%v",
			saved.ClaimNumber, saved.Transferred)
	}
	if saved.Position != inboxlaporanklaim.PositionNotTransferred {
		t.Fatalf("posisi berkas baru = %q", saved.Position)
	}
	if saved.Origin != inboxlaporanklaim.OriginNew {
		t.Fatalf("asal berkas baru = %q, ingin %q", saved.Origin, inboxlaporanklaim.OriginNew)
	}
	if saved.BranchCode != adminJakarta.BranchCode {
		t.Fatalf("cabang berkas baru = %q, ingin %q", saved.BranchCode, adminJakarta.BranchCode)
	}
	if saved.ReporterName != adminJakarta.Name {
		t.Fatalf("nama pelapor = %q, ingin nama petugas %q", saved.ReporterName, adminJakarta.Name)
	}
	if !inboxlaporanklaim.IssuedHere(saved.ID) {
		t.Fatalf("nomor berkas baru %q tidak menandai asalnya", saved.ID)
	}

	found := listOf(t, service, adminJakarta, usecase.Query{
		Category:   inboxlaporanklaim.CategoryNotTransferred,
		Pagination: inboxlaporanklaim.Pagination{Size: inboxlaporanklaim.MaxPageSize},
	})
	for _, report := range found.Page.Report {
		if report.ID == saved.ID {
			return
		}
	}
	t.Fatalf("berkas %q tidak muncul di tab Data hasn't been transferred", saved.ID)
}

func TestTwoNewReportsNeverShareANumber(t *testing.T) {
	service, _ := newService(t)

	first, err := service.Create(context.Background(), portalAlias, adminJakarta)
	if err != nil {
		t.Fatalf("berkas pertama: %v", err)
	}
	second, err := service.Create(context.Background(), portalAlias, adminJakarta)
	if err != nil {
		t.Fatalf("berkas kedua: %v", err)
	}
	if first.ID == second.ID {
		t.Fatalf("dua berkas bernomor sama: %q", first.ID)
	}
}

func TestReportWithoutBranchIsStillCreatedJustLikePega(t *testing.T) {
	service, _ := newService(t)

	// `CreateNewCaseRCV` langkah 19 mengisi cabang dari hasil `GetIDCabang` dan TIDAK
	// memeriksa hasilnya: kueri yang tidak mengembalikan baris menghasilkan cabang
	// kosong, dan berkas tetap dibuat. Menolaknya adalah aturan BARU, dan `P-5`
	// menetapkan perilaku dipertahankan lebih dulu.
	saved, err := service.Create(context.Background(), portalAlias, inboxlaporanklaim.Caller{
		Login: "adminpnc",
		Name:  "Petugas Contoh",
	})
	if err != nil {
		t.Fatalf("berkas tanpa cabang ditolak: %v", err)
	}
	if saved.BranchCode != "" {
		t.Fatalf("cabang berkas = %q, ingin kosong", saved.BranchCode)
	}
}

func TestReportCannotBeCreatedWithoutIdentity(t *testing.T) {
	service, _ := newService(t)

	// Tanpa identitas, tidak ada yang dapat dicatat sebagai pembuatnya — dan itu satu
	// hal yang di sistem lama TIDAK mungkin terjadi: setiap tindakan di Pega punya
	// operator. Ia menandakan jembatan ke konteks pemanggil tidak terpasang.
	_, err := service.Create(context.Background(), portalAlias, inboxlaporanklaim.Caller{
		BranchCode: "1001",
	})
	if !errors.Is(err, inboxlaporanklaim.ErrCallerUnknown) {
		t.Fatalf("galat tanpa identitas = %v", err)
	}
}

func TestUnknownPortalIsRefusedInsteadOfServedByThePrimary(t *testing.T) {
	service, _ := newService(t)

	// Mengembalikan portal utama sebagai cadangan berarti membaca berkas satu badan
	// hukum dari basis data badan hukum lain tanpa satu pun pesan galat (R-20).
	_, err := service.List(context.Background(), "SIMAS", adminJakarta, usecase.Query{
		Category: inboxlaporanklaim.CategoryAll,
	})
	if err == nil {
		t.Fatal("portal yang tidak tersedia justru dilayani")
	}
}

func TestStorageFailureIsReportedNotSwallowed(t *testing.T) {
	service, repo := newService(t)
	repo.SetError(errors.New("basis data tidak dapat dihubungi"))

	_, err := service.List(context.Background(), portalAlias, adminJakarta, usecase.Query{
		Category: inboxlaporanklaim.CategoryAll,
	})
	if err == nil {
		t.Fatal("kegagalan penyimpanan dijawab seolah berhasil")
	}
}

func TestFormSavesOntoTheReportThatButtonCreated(t *testing.T) {
	service, _ := newService(t)

	created, err := service.Create(context.Background(), portalAlias, adminJakarta)
	if err != nil {
		t.Fatalf("membuat berkas: %v", err)
	}

	detail := inboxlaporanklaim.Detail{
		ReceivedDate:  time.Date(2026, 9, 18, 0, 0, 0, 0, time.UTC),
		DateOfLoss:    time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC),
		ReporterName:  "Pelapor Contoh",
		PolicyNumber:  "POL-CONTOH-1",
		InsuredName:   "Tertanggung Contoh",
		EstimateValue: inboxlaporanklaim.Rupiah(2_500_000),
		Chronology:    "Kronologis contoh.",
		DocumentCount: 3,
	}

	saved, err := service.Save(context.Background(), portalAlias, created.ID, adminJakarta, detail)
	if err != nil {
		t.Fatalf("menyimpan isian: %v", err)
	}

	if saved.ReporterName != "Pelapor Contoh" || saved.PolicyNumber != "POL-CONTOH-1" {
		t.Fatalf("isian tidak tersimpan: %+v", saved)
	}
	if saved.UpdatedBy != adminJakarta.Login {
		t.Fatalf("jejak penyimpan = %q, ingin %q", saved.UpdatedBy, adminJakarta.Login)
	}
	if saved.UpdatedAt.IsZero() {
		t.Fatal("waktu penyimpanan tidak dicatat")
	}

	// Yang tersimpan harus terbaca kembali — bukan hanya dikembalikan oleh pemanggilnya.
	reread, err := service.Get(context.Background(), portalAlias, created.ID)
	if err != nil {
		t.Fatalf("membaca ulang: %v", err)
	}
	if reread.Chronology != "Kronologis contoh." || reread.DocumentCount != 3 {
		t.Fatalf("isian tidak bertahan: %+v", reread)
	}
}

func TestSavingIsIdempotent(t *testing.T) {
	service, _ := newService(t)

	created, _ := service.Create(context.Background(), portalAlias, adminJakarta)
	detail := inboxlaporanklaim.Detail{ReporterName: "Pelapor Contoh", DocumentCount: 2}

	first, err := service.Save(context.Background(), portalAlias, created.ID, adminJakarta, detail)
	if err != nil {
		t.Fatalf("simpan pertama: %v", err)
	}
	second, err := service.Save(context.Background(), portalAlias, created.ID, adminJakarta, detail)
	if err != nil {
		t.Fatalf("simpan kedua: %v", err)
	}

	// Menekan Simpan dua kali menghasilkan keadaan yang sama persis — itulah yang membuat
	// PUT benar dan PATCH tidak.
	if inboxlaporanklaim.DetailOf(first) != inboxlaporanklaim.DetailOf(second) {
		t.Fatal("penyimpanan kedua menghasilkan isian yang berbeda")
	}
}

func TestLegacyReportIsRefusedBeforeAnyFieldIsChecked(t *testing.T) {
	service, _ := newService(t)

	// Isiannya sengaja dibuat CACAT. Yang harus dijawab adalah penolakan asal berkas,
	// bukan daftar galat isian: memberi pengguna daftar isian yang harus diperbaiki pada
	// form yang memang tidak dapat disimpan sama sekali adalah menyesatkan.
	broken := inboxlaporanklaim.Detail{
		ReporterName: strings.Repeat("a", inboxlaporanklaim.MaxNameLength+1),
	}

	_, err := service.Save(context.Background(), portalAlias, "RCV-0001", adminJakarta, broken)
	if !errors.Is(err, inboxlaporanklaim.ErrReadOnlyOrigin) {
		t.Fatalf("galat menyimpan berkas Pega = %v, ingin ErrReadOnlyOrigin", err)
	}
}

func TestSavingAReportThatDoesNotExistIsNotFound(t *testing.T) {
	service, _ := newService(t)

	_, err := service.Save(context.Background(), portalAlias, "RCVN.26.9999", adminJakarta,
		inboxlaporanklaim.Detail{})
	if !errors.Is(err, inboxlaporanklaim.ErrNotFound) {
		t.Fatalf("galat = %v, ingin ErrNotFound", err)
	}
}

func TestSavingRefusesInvalidFieldsWithEveryViolation(t *testing.T) {
	service, _ := newService(t)

	created, _ := service.Create(context.Background(), portalAlias, adminJakarta)
	broken := inboxlaporanklaim.Detail{
		ReporterName:  strings.Repeat("a", inboxlaporanklaim.MaxNameLength+1),
		EstimateValue: -1,
	}

	_, err := service.Save(context.Background(), portalAlias, created.ID, adminJakarta, broken)

	var failure *inboxlaporanklaim.ValidationError
	if !errors.As(err, &failure) {
		t.Fatalf("galat = %v, ingin ValidationError", err)
	}
	if len(failure.Violation) != 2 {
		t.Fatalf("pelanggaran = %d, ingin 2", len(failure.Violation))
	}
}

func TestSavingWithoutIdentityIsRefused(t *testing.T) {
	service, _ := newService(t)

	created, _ := service.Create(context.Background(), portalAlias, adminJakarta)
	_, err := service.Save(context.Background(), portalAlias, created.ID,
		inboxlaporanklaim.Caller{BranchCode: "1001"}, inboxlaporanklaim.Detail{})
	if !errors.Is(err, inboxlaporanklaim.ErrCallerUnknown) {
		t.Fatalf("galat = %v, ingin ErrCallerUnknown", err)
	}
}

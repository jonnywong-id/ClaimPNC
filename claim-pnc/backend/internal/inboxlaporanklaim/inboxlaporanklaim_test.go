package inboxlaporanklaim_test

import (
	"testing"
	"time"

	"claim-pnc/internal/inboxlaporanklaim"
)

func TestPositionDerivedFromTwoMarkersOnly(t *testing.T) {
	// Ketiga baris ini adalah CASE WHEN pada RDB List/ViewAllCase-SQL.xml, ditulis ulang
	// sebagai kasus uji. Bila aturannya bergeser, yang gagal adalah uji ini — bukan
	// laporan TAT tiga bulan kemudian.
	suite := []struct {
		name        string
		hasNumber   bool
		transferred bool
		want        inboxlaporanklaim.Position
	}{
		{"berkas berklaim dan sudah diserahkan", true, true, inboxlaporanklaim.PositionOutstanding},
		{"berkas sudah diserahkan tetapi belum berklaim", false, true, inboxlaporanklaim.PositionNotRegistered},
		{"berkas belum diserahkan", false, false, inboxlaporanklaim.PositionNotTransferred},
		// Kombinasi keempat — berklaim tetapi belum diserahkan — tidak punya cabang
		// sendiri di kueri lama; ia jatuh ke ELSE. Perilaku itu dipertahankan.
		{"berklaim tetapi belum diserahkan jatuh ke belum diserahkan", true, false, inboxlaporanklaim.PositionNotTransferred},
	}

	for _, c := range suite {
		t.Run(c.name, func(t *testing.T) {
			if got := inboxlaporanklaim.DerivePosition(c.hasNumber, c.transferred); got != c.want {
				t.Fatalf("posisi = %q, ingin %q", got, c.want)
			}
		})
	}
}

func TestPositionTextMatchesLegacyScreen(t *testing.T) {
	// D-13: teks yang dibaca petugas mengikuti layar Pega apa adanya. Uji ini menjaganya
	// tetap begitu — menerjemahkannya menjadi bahasa Indonesia akan lolos kompilasi
	// tanpa satu pun tanda bahwa layar berubah bagi penggunanya.
	suite := map[inboxlaporanklaim.Position]string{
		inboxlaporanklaim.PositionOutstanding:    "Outstanding",
		inboxlaporanklaim.PositionNotRegistered:  "Not Registered",
		inboxlaporanklaim.PositionNotTransferred: "Not Transferred",
	}
	for position, want := range suite {
		if string(position) != want {
			t.Fatalf("teks posisi = %q, ingin %q", position, want)
		}
	}
}

func TestAgingCountedInWholeCalendarDays(t *testing.T) {
	now := time.Date(2026, 9, 19, 8, 0, 0, 0, time.UTC)

	suite := []struct {
		name  string
		aging time.Time
		want  int
	}{
		{"hari ini", time.Date(2026, 9, 19, 23, 0, 0, 0, time.UTC), 0},
		// Berkas yang masuk kemarin sore berumur SATU hari, bukan nol karena belum genap
		// 24 jam. Umur berkas dibaca sebagai hari kalender, bukan sebagai lama waktu.
		{"kemarin sore", time.Date(2026, 9, 18, 17, 30, 0, 0, time.UTC), 1},
		{"sepuluh hari lalu", time.Date(2026, 9, 9, 8, 0, 0, 0, time.UTC), 10},
	}

	for _, c := range suite {
		t.Run(c.name, func(t *testing.T) {
			report := inboxlaporanklaim.ClaimReport{AgingAt: c.aging}
			if got := report.AgingDays(now); got != c.want {
				t.Fatalf("umur = %d hari, ingin %d", got, c.want)
			}
		})
	}
}

func TestAgingOfReportWithoutDateIsZeroNotEnormous(t *testing.T) {
	report := inboxlaporanklaim.ClaimReport{}
	if got := report.AgingDays(time.Date(2026, 9, 19, 0, 0, 0, 0, time.UTC)); got != 0 {
		// Tanpa penjagaan ini, berkas tanpa tanggal berumur lebih dari 700 ribu hari —
		// terhitung dari tahun nol — dan tampil sebagai berkas paling tertunggak di layar.
		t.Fatalf("umur berkas tanpa tanggal = %d, ingin 0", got)
	}
}

func TestLegacyAssignmentKeyOnlyForLegacyRows(t *testing.T) {
	legacy := inboxlaporanklaim.ClaimReport{
		AssignmentRef: "ASM-FW-GCNMFW-WORK RCV-0001",
		Origin:        inboxlaporanklaim.OriginLegacy,
	}
	want := "ASSIGN-WORKLIST ASM-FW-GCNMFW-WORK RCV-0001!ReceiveDocument_Flow"
	if got := legacy.LegacyAssignmentKey(); got != want {
		t.Fatalf("kunci penugasan = %q, ingin %q", got, want)
	}

	// Berkas yang diterbitkan aplikasi ini tidak punya penugasan di Pega. Menyusun kunci
	// separuh jadi akan membuka layar galat, bukan berkasnya.
	fresh := inboxlaporanklaim.ClaimReport{
		AssignmentRef: "apa pun",
		Origin:        inboxlaporanklaim.OriginNew,
	}
	if got := fresh.LegacyAssignmentKey(); got != "" {
		t.Fatalf("berkas baru punya kunci penugasan %q, ingin kosong", got)
	}

	notTransferred := inboxlaporanklaim.ClaimReport{Origin: inboxlaporanklaim.OriginLegacy}
	if got := notTransferred.LegacyAssignmentKey(); got != "" {
		t.Fatalf("berkas tanpa rujukan punya kunci %q, ingin kosong", got)
	}
}

func TestReportNumberIsZeroPaddedSoTextOrderMatchesIssueOrder(t *testing.T) {
	// D-71 butir 2 mencatat cacat ini pada nomor klaim: tanpa pemadatan, ".10" mendahului
	// ".9" saat diurutkan sebagai teks. Ia diketahui sebelum baris pertama terbit di sini,
	// sehingga tidak dibawa.
	ninth := inboxlaporanklaim.FormatReportNumber(2026, 9)
	tenth := inboxlaporanklaim.FormatReportNumber(2026, 10)

	if ninth != "RCVN.26.0009" {
		t.Fatalf("nomor kesembilan = %q, ingin RCVN.26.0009", ninth)
	}
	if tenth != "RCVN.26.0010" {
		t.Fatalf("nomor kesepuluh = %q, ingin RCVN.26.0010", tenth)
	}
	if !(ninth < tenth) {
		t.Fatalf("urutan teks %q >= %q — pengurutan tidak lagi sesuai urutan penerbitan", ninth, tenth)
	}
}

func TestReportNumberBeyondFourDigitsGrowsInsteadOfWrapping(t *testing.T) {
	// Memotongnya menjadi empat digit akan menerbitkan nomor ganda, dan nomor ganda jauh
	// lebih mahal daripada kolom yang melebar.
	if got := inboxlaporanklaim.FormatReportNumber(2026, 12345); got != "RCVN.26.12345" {
		t.Fatalf("nomor = %q, ingin RCVN.26.12345", got)
	}
}

func TestIssuedHereDistinguishesOriginFromNumberAlone(t *testing.T) {
	if !inboxlaporanklaim.IssuedHere("RCVN.26.0001") {
		t.Fatal("nomor terbitan sendiri tidak dikenali")
	}
	if inboxlaporanklaim.IssuedHere("RCV-0001") {
		t.Fatal("nomor warisan dikenali sebagai terbitan sendiri")
	}
	// Awalan tanpa titik bukan nomor modul ini; tanpa titiknya, nomor warisan yang
	// kebetulan berawalan sama akan ikut terbaca.
	if inboxlaporanklaim.IssuedHere("RCVN0001") {
		t.Fatal("nomor tanpa titik dikenali sebagai terbitan sendiri")
	}
}

func TestEveryScreenTabIsReachableByCode(t *testing.T) {
	list := inboxlaporanklaim.ListCategories()
	if len(list) != 9 {
		t.Fatalf("jumlah tab = %d, ingin 9", len(list))
	}

	for _, info := range list {
		found, known := inboxlaporanklaim.FindCategory(string(info.Category))
		if !known {
			t.Fatalf("tab %q tidak dapat ditemukan lewat kodenya", info.Category)
		}
		if found != info.Category {
			t.Fatalf("kode %q menghasilkan tab %q", info.Category, found)
		}
		if info.Title == "" {
			t.Fatalf("tab %q tidak punya judul", info.Category)
		}
	}
}

func TestUnknownTabIsRejectedInsteadOfFallingBackToAllData(t *testing.T) {
	// Di sistem lama, rantai @if menjadikan SETIAP nilai tak dikenal jatuh ke ViewAllCase,
	// sehingga salah ketik menghasilkan daftar yang tampak wajar tetapi bukan yang diminta.
	if _, known := inboxlaporanklaim.FindCategory("salah-ketik"); known {
		t.Fatal("kategori tidak dikenal justru diterima")
	}
}

func TestTabTitlesMatchLegacyScreenExactly(t *testing.T) {
	suite := map[inboxlaporanklaim.Category]string{
		inboxlaporanklaim.CategoryOutstanding:       "Outstanding Data",
		inboxlaporanklaim.CategoryUnregistered:      "Unregistered data",
		inboxlaporanklaim.CategoryNotTransferred:    "Data hasn't been transferred",
		inboxlaporanklaim.CategoryAccepted:          "Data has been accepted",
		inboxlaporanklaim.CategoryRejected:          "Data rejected",
		inboxlaporanklaim.CategoryMessageUnanswered: "Not answered communication",
		inboxlaporanklaim.CategoryMessageWaiting:    "Not replied from ASM",
		inboxlaporanklaim.CategoryMessageReplied:    "Replied from ASM",
		inboxlaporanklaim.CategoryAll:               "All data",
	}
	for category, want := range suite {
		if got := category.Title(); got != want {
			t.Fatalf("judul %q = %q, ingin %q", category, got, want)
		}
	}
}

func TestLegacyQueryNameKeptForEquivalenceTesting(t *testing.T) {
	// Perkakas uji kesetaraan S-8 menembak kueri Pega yang sama dengan tab yang sedang
	// dibandingkan. Tanpa pemetaan ini, pasangannya harus ditebak.
	suite := map[inboxlaporanklaim.Category]string{
		inboxlaporanklaim.CategoryOutstanding:    "ViewTableBrowseClaimRegister",
		inboxlaporanklaim.CategoryUnregistered:   "ViewTableBrowseClaimNotRegister",
		inboxlaporanklaim.CategoryNotTransferred: "ViewTableBrowseRCVInProcess",
		inboxlaporanklaim.CategoryAccepted:       "ViewTableBrowseRCVAcc",
		inboxlaporanklaim.CategoryRejected:       "ViewTableBrowseRCVReject",
		inboxlaporanklaim.CategoryAll:            "ViewAllCase",
	}
	for category, want := range suite {
		if got := category.LegacyQuery(); got != want {
			t.Fatalf("kueri lama %q = %q, ingin %q", category, got, want)
		}
	}
}

func TestOnlyThreeTabsReadConversations(t *testing.T) {
	message := 0
	for _, info := range inboxlaporanklaim.ListCategories() {
		if info.Message {
			message++
		}
		filter, isMessage := inboxlaporanklaim.MessageFilterOf(info.Category)
		if isMessage != info.Message {
			t.Fatalf("tab %q: Message=%v tetapi penyaring percakapan=%v", info.Category, info.Message, isMessage)
		}
		if isMessage && filter.Status == "" {
			t.Fatalf("tab komunikasi %q tidak punya status percakapan", info.Category)
		}
	}
	if message != 3 {
		t.Fatalf("jumlah tab komunikasi = %d, ingin 3", message)
	}
}

func TestRejectedTabHasNoBadgeBecauseLegacyNeverCountedIt(t *testing.T) {
	// Kueri pencacah lama menyaring PYSTATUSWORK NOT IN ('Resolved-Completed',
	// 'Resolved-Rejected'), sehingga berkas ditolak justru yang dikecualikan. Lencana
	// bertuliskan 0 akan menyatakan "tidak ada berkas ditolak", dan itu tidak benar.
	summary := inboxlaporanklaim.Summary{Total: 12}

	if _, counted := summary.CountOf(inboxlaporanklaim.CategoryRejected); counted {
		t.Fatal("tab Data rejected punya lencana, padahal sistem lama tidak menghitungnya")
	}
	if count, counted := summary.CountOf(inboxlaporanklaim.CategoryAll); !counted || count != 12 {
		t.Fatalf("lencana All data = %d (dihitung=%v), ingin 12 dan dihitung", count, counted)
	}
}

func TestBusinessLineCriteriaComeFromOnePlace(t *testing.T) {
	// Keempat kode kelompok khusus dipakai DUA arah: sebagai penyaring masuk pada
	// kelompok khusus, dan sebagai pengecualian pada Non-MBU. Bila keduanya menyimpang,
	// baris yang jatuh di antaranya hilang dari KEDUA tab tanpa satu pun tanda.
	_, included, _ := inboxlaporanklaim.BusinessLineSpecialGroup.Criteria()
	_, _, excluded := inboxlaporanklaim.BusinessLineNonMBU.Criteria()

	if len(included) == 0 {
		t.Fatal("kelompok khusus tidak menyaring apa pun")
	}
	if len(included) != len(excluded) {
		t.Fatalf("jumlah kode kelompok khusus %d, pengecualian Non-MBU %d", len(included), len(excluded))
	}
	for i := range included {
		if included[i] != excluded[i] {
			t.Fatalf("kode ke-%d: masuk %q, dikecualikan %q", i, included[i], excluded[i])
		}
	}
}

func TestBusinessLineAllFiltersNothing(t *testing.T) {
	panel, in, notIn := inboxlaporanklaim.BusinessLineAll.Criteria()
	if len(panel) != 0 || len(in) != 0 || len(notIn) != 0 {
		t.Fatalf("pilihan seluruh bisnis justru menyaring: panel=%v in=%v notIn=%v", panel, in, notIn)
	}
}

func TestUnknownBusinessLineIsRejected(t *testing.T) {
	if _, known := inboxlaporanklaim.FindBusinessLine("bukan-lini"); known {
		t.Fatal("lini bisnis tidak dikenal justru diterima")
	}
	if _, known := inboxlaporanklaim.FindBusinessLine(""); !known {
		t.Fatal("pilihan kosong seharusnya sah — ia berarti seluruh bisnis")
	}
}

func TestPaginationIsRepairedNotRejected(t *testing.T) {
	suite := []struct {
		name       string
		given      inboxlaporanklaim.Pagination
		wantPage   int
		wantSize   int
		wantOffset int
	}{
		{"kosong memakai bawaan", inboxlaporanklaim.Pagination{}, 1, inboxlaporanklaim.DefaultPageSize, 0},
		{"halaman nol menjadi satu", inboxlaporanklaim.Pagination{Page: 0, Size: 10}, 1, 10, 0},
		{"halaman negatif menjadi satu", inboxlaporanklaim.Pagination{Page: -5, Size: 10}, 1, 10, 0},
		{"halaman ketiga melewati dua puluh baris", inboxlaporanklaim.Pagination{Page: 3, Size: 10}, 3, 10, 20},
		{"ukuran melebihi batas dipotong", inboxlaporanklaim.Pagination{Page: 1, Size: 5000}, 1, inboxlaporanklaim.MaxPageSize, 0},
	}

	for _, c := range suite {
		t.Run(c.name, func(t *testing.T) {
			clean := c.given.Clean()
			if clean.Page != c.wantPage || clean.Size != c.wantSize {
				t.Fatalf("halaman = %d/%d, ingin %d/%d", clean.Page, clean.Size, c.wantPage, c.wantSize)
			}
			if got := c.given.Offset(); got != c.wantOffset {
				t.Fatalf("giliran = %d, ingin %d", got, c.wantOffset)
			}
		})
	}
}

func TestPageCountCoversRemainder(t *testing.T) {
	suite := []struct {
		total int
		size  int
		want  int
	}{
		{0, 10, 1},  // daftar kosong tetap satu halaman, bukan nol
		{10, 10, 1}, // pas
		{11, 10, 2}, // sisa satu baris tetap menuntut halaman kedua
		{57, 10, 6},
	}
	for _, c := range suite {
		page := inboxlaporanklaim.Page{
			Total:      c.total,
			Pagination: inboxlaporanklaim.Pagination{Page: 1, Size: c.size},
		}
		if got := page.TotalPages(); got != c.want {
			t.Fatalf("total %d ukuran %d -> %d halaman, ingin %d", c.total, c.size, got, c.want)
		}
	}
}

func TestSeamProvidesNoDeleteOperation(t *testing.T) {
	// ADR-0012 melarang penghapusan fisik data bernilai bisnis, dan layar ini memang
	// tidak menghapus apa pun. Operasi yang tidak ada di seam tidak dapat dipakai kode
	// yang ditulis kemudian tanpa keputusan sadar.
	var repo any = (*fakeRepo)(nil)
	if _, hasDelete := repo.(interface{ Delete(string) error }); hasDelete {
		t.Fatal("seam penyimpanan memiliki operasi hapus")
	}
}

type fakeRepo struct{}

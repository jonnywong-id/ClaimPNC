package inboxmanagerreceivepucl_test

import (
	"context"
	"errors"
	"testing"

	"claim-pnc/internal/inboxmanagerreceivepucl"
	"claim-pnc/internal/inboxmanagerreceivepucl/repo/memory"
)

// Uji di berkas ini menguji ATURAN modul, bukan fungsinya satu per satu: ia berjalan di atas
// penyimpanan memori yang meniru keempat penyaring SQL, sehingga yang dibuktikannya berlaku
// pula bagi yang berjalan di Oracle (`14-TESTING-STRATEGY.md` §3).

// penyelia adalah pemanggil yang dipakai seluruh uji di sini.
//
// Namanya tidak berpengaruh pada hasil: tidak satu pun tab menyaring menurut pemanggil.
// Justru itu yang diuji TestNoTabIsScopedToTheCaller.
var penyelia = inboxmanagerreceivepucl.Caller{Login: "PENYELIACONTOH"}

// newQuery menyusun permintaan yang sah, dan menggagalkan uji bila tidak dapat dibentuk.
func newQuery(t *testing.T, tab string) inboxmanagerreceivepucl.Query {
	t.Helper()

	q, err := inboxmanagerreceivepucl.NewQuery(
		inboxmanagerreceivepucl.QueryInput{Tab: tab},
		penyelia,
	)
	if err != nil {
		t.Fatalf("membentuk permintaan tab %q: %v", tab, err)
	}
	return q
}

// caseIDs mengambil nomor case satu halaman, supaya harapan uji terbaca sebagai daftar nomor
// alih-alih sebagai struktur bersarang.
func caseIDs(page inboxmanagerreceivepucl.Page) []string {
	result := make([]string, 0, len(page.Items))
	for _, item := range page.Items {
		result = append(result, item.CaseID)
	}
	return result
}

// listAll mengambil seluruh baris sebuah tab dalam satu halaman besar.
func listAll(t *testing.T, tab string) inboxmanagerreceivepucl.Page {
	t.Helper()

	store := memory.NewSampleStore()
	page, err := store.List(
		context.Background(),
		newQuery(t, tab),
		inboxmanagerreceivepucl.Pagination{Page: 1, Size: inboxmanagerreceivepucl.MaxPageSize},
	)
	if err != nil {
		t.Fatalf("mengambil isi tab %q: %v", tab, err)
	}
	return page
}

func TestReceivePATabHoldsOnlyPersonalAccidentDocuments(t *testing.T) {
	// Kedua tab Receive dipisahkan Group Panel, pengganti `.ReceiveDocument.TypeOfClaim`
	// yang tidak punya kolom basis data. Bila pemisahnya keliru, satu tab menampilkan isi
	// tab yang lain — dan kolom keduanya IDENTIK, sehingga tidak ada apa pun di layar yang
	// menandakannya.
	page := listAll(t, inboxmanagerreceivepucl.TabReceivePA)

	want := []string{"RCV-900001", "RCV-900002"}
	if got := caseIDs(page); !equal(got, want) {
		t.Fatalf("tab Receive PA berisi %v, seharusnya %v", got, want)
	}

	for _, item := range page.Items {
		if item.ClaimType != inboxmanagerreceivepucl.ClaimTypePA {
			t.Fatalf("baris %s berjenis %q, seharusnya %q",
				item.CaseID, item.ClaimType, inboxmanagerreceivepucl.ClaimTypePA)
		}
	}
}

func TestReceiveNonMBUTabHoldsEverythingElse(t *testing.T) {
	page := listAll(t, inboxmanagerreceivepucl.TabReceiveNonMBU)

	want := []string{"RCV-900003", "RCV-900004"}
	if got := caseIDs(page); !equal(got, want) {
		t.Fatalf("tab Receive NONMBU berisi %v, seharusnya %v", got, want)
	}

	for _, item := range page.Items {
		if item.ClaimType != inboxmanagerreceivepucl.ClaimTypeNonMBU {
			t.Fatalf("baris %s berjenis %q, seharusnya %q",
				item.CaseID, item.ClaimType, inboxmanagerreceivepucl.ClaimTypeNonMBU)
		}
	}
}

func TestDocumentWithoutGroupPanelAppearsInNoReceiveTab(t *testing.T) {
	// `GROUPPANEL_1 <> '002'` TIDAK menangkap NULL di Oracle maupun PostgreSQL, sehingga
	// berkas tanpa Group Panel tidak muncul di tab mana pun. Itu perilaku yang sama dengan
	// layar lama, tempat berkas tanpa `TypeOfClaim` tidak cocok dengan grid mana pun.
	//
	// Uji ini ada supaya keadaan itu menjadi keputusan yang tercatat, bukan kebetulan yang
	// kelak "diperbaiki" oleh orang yang mengira ia bug.
	for _, tab := range []string{
		inboxmanagerreceivepucl.TabReceivePA,
		inboxmanagerreceivepucl.TabReceiveNonMBU,
	} {
		for _, id := range caseIDs(listAll(t, tab)) {
			if id == "RCV-900005" {
				t.Fatalf("berkas tanpa Group Panel muncul di tab %s", tab)
			}
		}
	}
}

func TestClaimsNeverLeakIntoTheReceiveTabs(t *testing.T) {
	// Berkas penerimaan dokumen dan klaim hidup di SATU tabel, dibedakan hanya PXOBJCLASS.
	// Keduanya punya PYID, POLICYNO, dan QQNAME, sehingga pencampurannya tidak menghasilkan
	// satu pun galat — hanya baris yang tidak seharusnya ada.
	for _, tab := range []string{
		inboxmanagerreceivepucl.TabReceivePA,
		inboxmanagerreceivepucl.TabReceiveNonMBU,
	} {
		for _, id := range caseIDs(listAll(t, tab)) {
			if len(id) >= 4 && id[:4] == "PNC-" {
				t.Fatalf("klaim %s bocor ke tab %s", id, tab)
			}
		}
	}
}

func TestRCLPUCLTabHoldsOnlyOpenClaimsInItsOwnWorkbasket(t *testing.T) {
	// Tiga penyaring diuji sekaligus, dan ketiganya harus bekerja bersama:
	//
	//   kelas objek kerja   klaim, bukan berkas penerimaan dokumen
	//   antrean bersama     RCLPUCL, bukan antrean lain
	//   status kerja        belum selesai
	//
	// Penyaring antrean-lah yang paling penting di sini: ia TIDAK ADA di Report Definition
	// aslinya, dan tanpanya tab ini menampilkan seluruh klaim yang belum selesai.
	page := listAll(t, inboxmanagerreceivepucl.TabRCLPUCL)

	want := []string{"PNC-800002", "PNC-800003"}
	if got := caseIDs(page); !equal(got, want) {
		t.Fatalf("tab RCL/PUCL berisi %v, seharusnya %v", got, want)
	}
}

func TestRCLPUCLTabCarriesBothTracks(t *testing.T) {
	// `RCL_PUCL_1` menyimpan angka, dan yang digambar adalah teksnya. Bila penerjemahannya
	// hilang, kolomnya menampilkan "1" dan "2" — yang tidak berarti apa pun bagi pembaca.
	page := listAll(t, inboxmanagerreceivepucl.TabRCLPUCL)

	seen := map[string]bool{}
	for _, item := range page.Items {
		seen[item.Track] = true
	}

	for _, track := range []string{
		inboxmanagerreceivepucl.TrackRCL,
		inboxmanagerreceivepucl.TrackPUCL,
	} {
		if !seen[track] {
			t.Fatalf("tab RCL/PUCL tidak memuat satu pun baris berjalur %q", track)
		}
	}
}

func TestRCLPUCLTabHasNoClaimType(t *testing.T) {
	// Jenis Klaim diturunkan dari Group Panel, dan klaim di tab ini tidak membawanya —
	// kolomnya `CAST(NULL …)` di SQL. Mengisinya "NONMBU" akan menyatakan sesuatu yang
	// tidak pernah dibaca dari mana pun.
	for _, item := range listAll(t, inboxmanagerreceivepucl.TabRCLPUCL).Items {
		if item.ClaimType != "" {
			t.Fatalf("baris %s membawa Jenis Klaim %q; tab ini tidak menurunkannya",
				item.CaseID, item.ClaimType)
		}
	}
}

func TestNoTabIsScopedToTheCaller(t *testing.T) {
	// Layar ini pandangan PENYELIA: tidak satu pun tabnya menyaring menurut pemanggil.
	// Dua pemanggil yang berbeda karena itu melihat isi yang sama persis.
	//
	// Ini BUKAN kelalaian melainkan bentuk layarnya — dan justru karena itu ia diuji:
	// menambahkan penyaring kepemilikan kelak akan mengubah arti layar tanpa satu pun
	// keputusan yang mendasarinya.
	store := memory.NewSampleStore()

	for _, tab := range []string{
		inboxmanagerreceivepucl.TabReceivePA,
		inboxmanagerreceivepucl.TabReceiveNonMBU,
		inboxmanagerreceivepucl.TabRCLPUCL,
	} {
		page := inboxmanagerreceivepucl.Pagination{Page: 1, Size: 100}

		first, err := store.List(context.Background(), newQuery(t, tab), page)
		if err != nil {
			t.Fatalf("mengambil isi tab %q: %v", tab, err)
		}

		other, err := inboxmanagerreceivepucl.NewQuery(
			inboxmanagerreceivepucl.QueryInput{Tab: tab},
			inboxmanagerreceivepucl.Caller{Login: "PENYELIALAIN"},
		)
		if err != nil {
			t.Fatalf("membentuk permintaan tab %q: %v", tab, err)
		}

		second, err := store.List(context.Background(), other, page)
		if err != nil {
			t.Fatalf("mengambil isi tab %q: %v", tab, err)
		}

		if !equal(caseIDs(first), caseIDs(second)) {
			t.Fatalf("tab %s memberi isi berbeda kepada dua pemanggil: %v vs %v",
				tab, caseIDs(first), caseIDs(second))
		}
	}
}

func TestRowsAreOrderedNewestFirst(t *testing.T) {
	// Urutan ditetapkan tegas supaya paginasi di atasnya stabil. Ketiga Report Definition
	// di sistem lama tidak mengurutkan sama sekali, dan itu dapat dibiarkan selama seluruh
	// baris ditarik sekaligus — begitu halamannya dipotong, satu baris dapat muncul di dua
	// halaman sekaligus hilang dari halaman lain.
	got := caseIDs(listAll(t, inboxmanagerreceivepucl.TabRCLPUCL))
	want := []string{"PNC-800002", "PNC-800003"}

	if !equal(got, want) {
		t.Fatalf("urutan tab RCL/PUCL %v, seharusnya %v", got, want)
	}
}

func TestPaginationCutsWithoutLosingTheTotal(t *testing.T) {
	// Jumlah seluruh baris harus tetap benar meski halamannya dipotong: layar menggambar
	// "menampilkan 1–1 dari 2" dari angka itu, dan angka yang mengikuti ukuran halaman
	// akan selalu menyatakan "1 dari 1".
	store := memory.NewSampleStore()

	page, err := store.List(
		context.Background(),
		newQuery(t, inboxmanagerreceivepucl.TabRCLPUCL),
		inboxmanagerreceivepucl.Pagination{Page: 1, Size: 1},
	)
	if err != nil {
		t.Fatalf("mengambil halaman pertama: %v", err)
	}

	if len(page.Items) != 1 {
		t.Fatalf("halaman memuat %d baris, seharusnya 1", len(page.Items))
	}
	if page.Total != 2 {
		t.Fatalf("total %d, seharusnya 2", page.Total)
	}
	if page.TotalPages() != 2 {
		t.Fatalf("jumlah halaman %d, seharusnya 2", page.TotalPages())
	}
}

func TestEmptyTabCodeOpensTheDefaultTab(t *testing.T) {
	q, err := inboxmanagerreceivepucl.NewQuery(
		inboxmanagerreceivepucl.QueryInput{},
		penyelia,
	)
	if err != nil {
		t.Fatalf("membentuk permintaan tanpa kode tab: %v", err)
	}
	if q.Tab.Code != inboxmanagerreceivepucl.DefaultTab {
		t.Fatalf("tab bawaan %q, seharusnya %q",
			q.Tab.Code, inboxmanagerreceivepucl.DefaultTab)
	}
}

func TestUnknownTabIsRejectedAsValidation(t *testing.T) {
	// Ia galat VALIDASI, bukan galat internal: permintaannya berbentuk benar, isinya yang
	// salah. Lapisan transport memetakannya ke 422, dan layar menandai isiannya alih-alih
	// menampilkan "terjadi kesalahan pada sistem".
	_, err := inboxmanagerreceivepucl.NewQuery(
		inboxmanagerreceivepucl.QueryInput{Tab: "99"},
		penyelia,
	)

	var validation *inboxmanagerreceivepucl.ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("galat %v, seharusnya galat validasi", err)
	}
}

func TestQueryWithoutCallerIsRejected(t *testing.T) {
	// Tidak satu pun kueri menyaring menurut pemanggil, tetapi permintaannya tetap ditolak
	// tanpa identitas: seluruh baris layar ini milik pekerjaan orang lain, dan
	// pembukaannya wajib tercatat atas nama seseorang (`D-59`).
	_, err := inboxmanagerreceivepucl.NewQuery(
		inboxmanagerreceivepucl.QueryInput{},
		inboxmanagerreceivepucl.Caller{Login: "   "},
	)
	if !errors.Is(err, inboxmanagerreceivepucl.ErrCallerUnknown) {
		t.Fatalf("galat %v, seharusnya ErrCallerUnknown", err)
	}
}

func TestClaimTypeIsDerivedFromTheGroupPanel(t *testing.T) {
	// Penerjemah ini dipakai penyimpanan SQL DAN penyimpanan memori. Bila keduanya punya
	// penerjemah sendiri, uji yang lulus di atas memori tidak menyatakan apa pun tentang
	// yang berjalan di Oracle.
	cases := map[string]string{
		inboxmanagerreceivepucl.GroupPanelPA: inboxmanagerreceivepucl.ClaimTypePA,
		" 002 ":                              inboxmanagerreceivepucl.ClaimTypePA,
		"003":                                inboxmanagerreceivepucl.ClaimTypeNonMBU,
		"006":                                inboxmanagerreceivepucl.ClaimTypeNonMBU,
		"":                                   inboxmanagerreceivepucl.ClaimTypeNonMBU,
	}

	for panel, want := range cases {
		if got := inboxmanagerreceivepucl.ClaimTypeOf(panel); got != want {
			t.Fatalf("Group Panel %q menjadi %q, seharusnya %q", panel, got, want)
		}
	}
}

func TestEveryTabColumnHasATitleAndKey(t *testing.T) {
	// Kolom tanpa kunci tidak dapat digambar; kolom tanpa judul digambar sebagai kolom
	// tanpa kepala. Keduanya hanya terlihat setelah layar dibuka, dan uji ini yang
	// menangkapnya lebih dulu.
	for _, tab := range inboxmanagerreceivepucl.Tabs() {
		if tab.Blocked {
			continue
		}
		if len(tab.Columns) == 0 {
			t.Fatalf("tab %s (%s) tidak punya satu pun kolom", tab.Code, tab.Name)
		}
		for _, column := range tab.Columns {
			if column.Key == "" || column.Title == "" {
				t.Fatalf("tab %s memuat kolom tanpa kunci atau tanpa judul: %+v",
					tab.Code, column)
			}
		}
	}
}

func TestBothReceiveTabsShareTheSameColumns(t *testing.T) {
	// Keduanya digambar Report Definition yang SAMA, dijalankan dengan parameter berbeda.
	// Kolomnya karena itu wajib sama persis — perbedaan satu kolom saja berarti salah satu
	// tab tidak lagi mencerminkan grid aslinya.
	pa, found := inboxmanagerreceivepucl.FindTab(inboxmanagerreceivepucl.TabReceivePA)
	if !found {
		t.Fatal("tab Receive PA tidak ditemukan")
	}
	nonMBU, found := inboxmanagerreceivepucl.FindTab(
		inboxmanagerreceivepucl.TabReceiveNonMBU)
	if !found {
		t.Fatal("tab Receive NONMBU tidak ditemukan")
	}

	if len(pa.Columns) != len(nonMBU.Columns) {
		t.Fatalf("tab Receive PA punya %d kolom, NONMBU %d",
			len(pa.Columns), len(nonMBU.Columns))
	}
	for i := range pa.Columns {
		if pa.Columns[i] != nonMBU.Columns[i] {
			t.Fatalf("kolom ke-%d berbeda: %+v vs %+v",
				i, pa.Columns[i], nonMBU.Columns[i])
		}
	}
}

func TestPlannedDifferencesAreStated(t *testing.T) {
	// Selisih yang hanya tercatat di komentar akan dilaporkan berulang kali sebagai
	// kerusakan oleh orang yang membandingkan kedua layar berdampingan. Ia dikirim ke
	// layar sebagai data, dan uji ini memastikan daftarnya tidak pernah dikosongkan diam
	// diam.
	if len(inboxmanagerreceivepucl.PlannedDifferences) == 0 {
		t.Fatal("daftar selisih terencana kosong")
	}
	for i, line := range inboxmanagerreceivepucl.PlannedDifferences {
		if line == "" {
			t.Fatalf("selisih terencana ke-%d kosong", i)
		}
	}
}

// equal membandingkan dua senarai teks apa adanya, termasuk urutannya.
func equal(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

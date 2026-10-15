package inboxmanagerreceivepucl_test

import (
	"context"
	"errors"
	"sort"
	"strings"
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

func TestReceiveTabHoldsBothLinesOfBusinessTogether(t *testing.T) {
	// Kedua grid Pega digabung menjadi SATU tab pada 2026-09-30, dan himpunan barisnya wajib
	// tetap sama: gabungan tepat dari keduanya, tidak lebih dan tidak kurang.
	//
	// Keempat berkas di bawah adalah dua PA ditambah dua NONMBU — persis isi kedua grid lama
	// bila ditumpuk seperti di Pega.
	page := listAll(t, inboxmanagerreceivepucl.TabReceive)

	want := []string{"RCV-900001", "RCV-900002", "RCV-900003", "RCV-900004"}
	got := append([]string{}, caseIDs(page)...)
	sort.Strings(got)

	if !equal(got, want) {
		t.Fatalf("tab Receive berisi %v, seharusnya %v", got, want)
	}
}

func TestReceiveTabCarriesTheClaimTypeThatUsedToSplitIt(t *testing.T) {
	// Pembedaan PA versus NONMBU tidak hilang saat kedua grid digabung — ia pindah dari
	// "tabel yang mana" menjadi ISI kolom "Jenis Klaim".
	//
	// Bila penurunannya hilang, tabelnya tetap tampil utuh dan tidak ada satu pun galat;
	// yang terjadi hanyalah kolom itu kosong, dan pengguna kehilangan satu-satunya hal yang
	// membedakan kedua lini bisnis di layar ini.
	page := listAll(t, inboxmanagerreceivepucl.TabReceive)

	seen := map[string]bool{}
	for _, item := range page.Items {
		if item.ClaimType == "" {
			t.Fatalf("baris %s tidak membawa Jenis Klaim", item.CaseID)
		}
		seen[item.ClaimType] = true
	}

	for _, want := range []string{
		inboxmanagerreceivepucl.ClaimTypePA,
		inboxmanagerreceivepucl.ClaimTypeNonMBU,
	} {
		if !seen[want] {
			t.Fatalf("tab Receive tidak memuat satu pun baris berjenis %q", want)
		}
	}
}

func TestDocumentWithoutGroupPanelStillAppearsInNoTab(t *testing.T) {
	// Penyaring `GROUPPANEL_1 IS NOT NULL` adalah gabungan TEPAT dari kedua penyaring grid
	// Pega — bukan "tanpa penyaring". Berkas tanpa Group Panel karena itu tetap tidak
	// terlihat, persis seperti sebelum kedua grid digabung dan persis seperti di Pega.
	//
	// Uji ini ada supaya keadaan itu menjadi keputusan yang tercatat, bukan kebetulan yang
	// kelak "diperbaiki" oleh orang yang mengira ia bug.
	for _, id := range caseIDs(listAll(t, inboxmanagerreceivepucl.TabReceive)) {
		if id == "RCV-900005" {
			t.Fatal("berkas tanpa Group Panel muncul di tab Receive")
		}
	}
}

func TestClaimsNeverLeakIntoTheReceiveTab(t *testing.T) {
	// Berkas penerimaan dokumen dan klaim hidup di SATU tabel, dibedakan hanya PXOBJCLASS.
	// Keduanya punya PYID, POLICYNO, dan QQNAME, sehingga pencampurannya tidak menghasilkan
	// satu pun galat — hanya baris yang tidak seharusnya ada.
	for _, id := range caseIDs(listAll(t, inboxmanagerreceivepucl.TabReceive)) {
		if strings.HasPrefix(id, "PNC-") {
			t.Fatalf("klaim %s bocor ke tab Receive", id)
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
		inboxmanagerreceivepucl.TabReceive,
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

func TestEachTabOpensItsOwnWorkScreen(t *testing.T) {
	// Kedua grid di Pega punya tautan pada nomor case-nya, dan keduanya membuka layar yang
	// BERBEDA karena kelas objek kerjanya berbeda:
	//
	//	Receive    SetAssignmentInboxReceive_act   ASM-FW-GCNMFW-Work-ReceiveDocument
	//	RCL/PUCL   SetAssignmentInboxPUCL_act      ASM-FW-GCNMFW-Work-PNC
	//
	// Keduanya tidak boleh benar bersamaan pada satu tab. Bila penanda layar dokumen bocor
	// ke tab RCL/PUCL, nomor case di sana akan membuka layar PENERIMAAN DOKUMEN untuk sebuah
	// KLAIM — dan kuerinya menyaring kelas objek kerja, sehingga yang terjadi bukan layar
	// berisi data keliru melainkan "berkas tidak ditemukan" pada setiap baris.
	for _, tab := range inboxmanagerreceivepucl.Tabs() {
		wantDocument := tab.Code == inboxmanagerreceivepucl.TabReceive
		wantClaim := tab.Code == inboxmanagerreceivepucl.TabRCLPUCL

		if tab.OpensReceiveDocument != wantDocument {
			t.Fatalf("tab %s (%s) membuka layar dokumen = %v, seharusnya %v",
				tab.Code, tab.Name, tab.OpensReceiveDocument, wantDocument)
		}
		if tab.OpensClaimScreen != wantClaim {
			t.Fatalf("tab %s (%s) membuka layar klaim = %v, seharusnya %v",
				tab.Code, tab.Name, tab.OpensClaimScreen, wantClaim)
		}
		if tab.OpensReceiveDocument && tab.OpensClaimScreen {
			t.Fatalf("tab %s (%s) membuka DUA layar sekaligus", tab.Code, tab.Name)
		}
	}
}

func TestFieldGroupsUseThePegaHeadings(t *testing.T) {
	// `Section/InputReceiveDocument-Section.xml` punya enam `<pyTitle>`, dan modul ini
	// menggambar tiga di antaranya — yang tiga lagi tidak punya satu pun isian yang modul
	// ini baca.
	//
	// Uji ini menjaga kekeliruan 2026-10-10 tidak terulang: versi sebelumnya memakai lima
	// judul KARANGAN yang tidak satu pun ada di section, dengan alasan tertulis bahwa
	// section itu tidak punya judul panel. Judul karangan tidak dapat ditelusuri balik ke
	// bukti, dan `D-13` menuntut tampilan mengikuti Pega.
	groups := inboxmanagerreceivepucl.DocumentFieldGroupList()

	want := []string{"Data Pelaporan Klaim", "Data Tertanggung Klaim", "Alamat"}
	if len(groups) != len(want) {
		t.Fatalf("layar punya %d kelompok isian, seharusnya %d", len(groups), len(want))
	}
	for i, title := range want {
		if groups[i].Title != title {
			t.Fatalf("kelompok ke-%d berjudul %q, seharusnya %q", i+1, groups[i].Title, title)
		}
		if len(groups[i].Fields) == 0 {
			t.Fatalf("kelompok %q tidak punya satu pun isian", title)
		}
	}
}

func TestOnlyDocumentButtonsSitAboveTheFields(t *testing.T) {
	// Di section, tombol "Upload Form Klaim" (sel 179928) dan "View Form Klaim" (200348)
	// berada di dalam blok utama dekat puncaknya; `SendAttachmentToPNC` (1587331) dan
	// `CreateRegisterKlaimPNC` (1618486) berada SETELAH blok itu tertutup di 1564427, yaitu
	// di tempat Pega menggambar tombol flow action.
	//
	// Membalik keduanya membuat petugas yang membandingkan kedua layar berdampingan mencari
	// tombol di tempat yang salah.
	atas := map[string]bool{"unggah-form-klaim": true, "lihat-form-klaim": true}

	for _, action := range inboxmanagerreceivepucl.DocumentWriteActionList() {
		if action.AtTop != atas[action.Code] {
			t.Fatalf("tombol %q (%s) di atas = %v, seharusnya %v",
				action.Label, action.Code, action.AtTop, atas[action.Code])
		}
	}
}

func TestTheScreenHasExactlyTheTwoTabsPegaHas(t *testing.T) {
	// `Section/InboxManagerReceive_Section-Section.xml` memuat tepat dua `<pyTitle>`:
	// "Receive" dan "RCL/PUCL". Modul ini mengikutinya (`D-13`).
	//
	// Uji ini menjaga penggabungan 2026-09-30 tidak terurai kembali diam-diam: versi pertama
	// modul ini memecah Receive menjadi dua tab, dan pemecahan itu dicabut.
	tabs := inboxmanagerreceivepucl.Tabs()

	if len(tabs) != 2 {
		t.Fatalf("layar punya %d tab, seharusnya 2", len(tabs))
	}
	if tabs[0].Name != "Receive" {
		t.Fatalf("tab pertama bernama %q, seharusnya \"Receive\"", tabs[0].Name)
	}
	if tabs[1].Name != "RCL/PUCL" {
		t.Fatalf("tab kedua bernama %q, seharusnya \"RCL/PUCL\"", tabs[1].Name)
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

// ============================================================================
// LAYAR KERJA PENERIMAAN DOKUMEN — flow action `InputReceiveDocument`
// ============================================================================

// openDocument membuka layar kerja sebuah berkas, dan menggagalkan uji bila gagal.
func openDocument(t *testing.T, reference string) inboxmanagerreceivepucl.ReceiveDocument {
	t.Helper()

	doc, err := memory.NewSampleStore().Document(context.Background(), reference)
	if err != nil {
		t.Fatalf("membuka berkas %q: %v", reference, err)
	}
	return doc
}

func TestWorkScreenOpensTheDocumentBehindItsCaseNumber(t *testing.T) {
	// Kunci yang dipakai adalah `PZINSKEY` — nilai yang sama yang di Pega dikirim sebagai
	// parameter `kunci` ke `SetAssignmentInboxReceive_act` lalu dipakai Open Assignment.
	doc := openDocument(t, "ASM-FW-GCNMFW-WORK RCV-900001")

	if doc.CaseID != "RCV-900001" {
		t.Fatalf("berkas yang terbuka %q, seharusnya RCV-900001", doc.CaseID)
	}
	if doc.SenderName == "" || doc.Chronology == "" {
		t.Fatalf("isian dari tabel cermin kosong: %+v", doc)
	}
}

func TestWorkScreenStillOpensWhenTheMirrorTableHasNoMatch(t *testing.T) {
	// POOLDATA.T_CLAIM_RECIVEDCLAIM digabung `LEFT JOIN` karena tabel itu TIDAK PERNAH
	// DIBACA sistem lama — kelengkapan isinya belum terverifikasi.
	//
	// Berkas tanpa pasangan di sana harus tetap TERBUKA dengan isian kosong, bukan
	// dinyatakan tidak ada. Justru berkas seperti itulah yang paling perlu dilihat orang
	// yang memeriksa kelengkapan tabelnya.
	doc := openDocument(t, "ASM-FW-GCNMFW-WORK RCV-900004")

	if doc.CaseID != "RCV-900004" {
		t.Fatalf("berkas yang terbuka %q, seharusnya RCV-900004", doc.CaseID)
	}
	if doc.SenderName != "" {
		t.Fatalf("berkas tanpa pasangan membawa Nama Pengirim %q", doc.SenderName)
	}
	if doc.PolicyNumber == "" {
		t.Fatal("isian dari objek kerja ikut kosong; gabungannya bukan LEFT JOIN")
	}
}

func TestWorkScreenNeverOpensAClaim(t *testing.T) {
	// Berkas penerimaan dokumen dan klaim hidup di SATU tabel, dibedakan hanya PXOBJCLASS.
	// Tanpa penyaring itu, kunci milik sebuah klaim akan membuka layar kerja penerimaan
	// dokumen yang isinya klaim — tanpa satu pun tanda bahwa itu keliru.
	_, err := memory.NewSampleStore().Document(
		context.Background(), "ASM-FW-GCNMFW-WORK PNC-800002")

	if !errors.Is(err, inboxmanagerreceivepucl.ErrDocumentNotFound) {
		t.Fatalf("membuka kunci klaim menghasilkan %v, seharusnya ErrDocumentNotFound", err)
	}
}

func TestWorkScreenRejectsAnEmptyKeyDifferentlyFromAnUnknownOne(t *testing.T) {
	// Keduanya DIBEDAKAN karena tindak lanjutnya berbeda: yang pertama bug pemanggil, yang
	// kedua kunci yang keliru atau berkas milik portal lain. Menjawab keduanya sama akan
	// membuat yang pertama dicari di tempat yang salah.
	store := memory.NewSampleStore()

	_, err := store.Document(context.Background(), "   ")
	if !errors.Is(err, inboxmanagerreceivepucl.ErrReferenceRequired) {
		t.Fatalf("kunci kosong menghasilkan %v, seharusnya ErrReferenceRequired", err)
	}

	_, err = store.Document(context.Background(), "ASM-FW-GCNMFW-WORK RCV-TIDAK-ADA")
	if !errors.Is(err, inboxmanagerreceivepucl.ErrDocumentNotFound) {
		t.Fatalf("kunci tak dikenal menghasilkan %v, seharusnya ErrDocumentNotFound", err)
	}
}

func TestEveryValueBelongsToADrawnField(t *testing.T) {
	// Bentuk layar (DocumentFieldGroups) dan isinya (Values) hidup di dua tempat, dan
	// keduanya harus sepadan. Nilai yang tidak punya isian TIDAK akan pernah tergambar —
	// dan ketiadaannya tidak menghasilkan satu pun galat, hanya isian yang hilang diam-diam.
	drawn := map[string]inboxmanagerreceivepucl.Field{}
	for _, group := range inboxmanagerreceivepucl.DocumentFieldGroupList() {
		for _, field := range group.Fields {
			if _, clash := drawn[field.Key]; clash {
				t.Fatalf("isian %q digambar lebih dari sekali", field.Key)
			}
			drawn[field.Key] = field
		}
	}

	for key := range openDocument(t, "ASM-FW-GCNMFW-WORK RCV-900001").Values() {
		field, exists := drawn[key]
		if !exists {
			t.Fatalf("nilai %q tidak punya isian yang menggambarnya", key)
		}
		if field.Blocked {
			t.Fatalf("isian %q bertanda terhalang tetapi tetap membawa nilai", key)
		}
	}
}

func TestEveryUnblockedFieldHasAValue(t *testing.T) {
	// Kebalikan uji di atas, dan sama pentingnya: isian yang digambar TANPA tanda terhalang
	// tetapi tidak pernah punya nilai akan tergambar kosong selamanya — tidak dapat
	// dibedakan dari isian yang memang belum diisi petugas.
	values := openDocument(t, "ASM-FW-GCNMFW-WORK RCV-900001").Values()

	for _, group := range inboxmanagerreceivepucl.DocumentFieldGroupList() {
		for _, field := range group.Fields {
			if field.Blocked {
				continue
			}
			if _, exists := values[field.Key]; !exists {
				t.Fatalf("isian %q (%s) digambar tanpa tanda terhalang, "+
					"tetapi tidak pernah punya nilai", field.Key, field.Title)
			}
		}
	}
}

func TestEveryBlockedFieldNamesItsReasonAndItsOwner(t *testing.T) {
	// Penghalang tanpa alamat tidak pernah hilang (`D-36`). Isian yang digambar kosong tanpa
	// alasan akan dibaca sebagai data yang hilang, bukan sebagai isian yang belum terbawa.
	blocked := 0

	for _, group := range inboxmanagerreceivepucl.DocumentFieldGroupList() {
		if group.Title == "" {
			t.Fatal("ada kelompok isian tanpa judul")
		}
		for _, field := range group.Fields {
			if field.Key == "" || field.Title == "" {
				t.Fatalf("isian tanpa kunci atau tanpa judul: %+v", field)
			}
			if !field.Blocked {
				continue
			}
			blocked++
			if field.BlockedReason == "" {
				t.Fatalf("isian terhalang %q tidak menyebut alasannya", field.Key)
			}
			if field.BlockedOwner == "" {
				t.Fatalf("isian terhalang %q tidak menyebut pemiliknya", field.Key)
			}
		}
	}

	if blocked == 0 {
		t.Fatal("tidak ada satu pun isian terhalang; enam belas di antaranya memang belum " +
			"punya sumber, dan ketiadaannya wajib terlihat")
	}
}

func TestEveryWriteActionNamesItsActivityAndItsOwner(t *testing.T) {
	// Kedelapan tombol tetap digambar meski belum satu pun dapat dihidupkan. Yang membuatnya
	// berguna adalah penyebutan modul pemiliknya: "belum tersedia" tidak memberi tahu
	// siapa pun apa yang harus dikerjakan, sementara "`B-2` Registrasi Klaim" memberi tahu.
	actions := inboxmanagerreceivepucl.DocumentWriteActionList()

	if len(actions) == 0 {
		t.Fatal("tidak ada satu pun tombol; layar lama punya delapan")
	}

	seen := map[string]bool{}
	for _, action := range actions {
		if action.Code == "" || action.Label == "" {
			t.Fatalf("tombol tanpa kode atau tanpa label: %+v", action)
		}
		if seen[action.Code] {
			t.Fatalf("kode tombol ganda: %q", action.Code)
		}
		seen[action.Code] = true

		if action.Activity == "" {
			t.Fatalf("tombol %q tidak menyebut activity Pega-nya", action.Code)
		}
		if action.Owner == "" {
			t.Fatalf("tombol %q tidak menyebut modul pemiliknya", action.Code)
		}
	}
}

func TestFieldVisibilityFollowsThePegaConditions(t *testing.T) {
	// Layar lama TIDAK menggambar seluruh isian pada setiap berkas: 14 dari 24 isian punya
	// `<pyCondition>` pada selnya di `Section/InputReceiveDocument-Section.xml`. Empat di
	// antaranya sudah dapat diterjemahkan, dan uji ini menjaga keempatnya.
	//
	// Menggambarnya tanpa syarat membuat layar kita menampilkan isian yang di Pega tidak
	// ada — dilaporkan Work Owner 2026-10-10.
	punya := func(groups []inboxmanagerreceivepucl.FieldGroup, key string) bool {
		for _, group := range groups {
			for _, field := range group.Fields {
				if field.Key == key {
					return true
				}
			}
		}
		return false
	}

	for name, tc := range map[string]struct {
		detail inboxmanagerreceivepucl.ReceiveDocument
		want   map[string]bool
	}{
		// `.ReceiveDocument.PolicyNo != '' && GroupPanel != '002' && != '005'`
		"Fire berpolis": {
			detail: inboxmanagerreceivepucl.ReceiveDocument{GroupPanel: "006", PolicyNumber: "P-1"},
			want: map[string]bool{
				inboxmanagerreceivepucl.DocFieldPolicyLeader:    true,
				inboxmanagerreceivepucl.DocFieldBrokerReference: true,
				inboxmanagerreceivepucl.DocFieldInsuredEmail:    false,
				inboxmanagerreceivepucl.DocFieldDriverLicence:   false,
			},
		},
		// Personal Accident: Polis Leader dan No Ref Broker DIKECUALIKAN; dua lainnya muncul.
		"PA berpolis": {
			detail: inboxmanagerreceivepucl.ReceiveDocument{GroupPanel: "002", PolicyNumber: "P-2"},
			want: map[string]bool{
				inboxmanagerreceivepucl.DocFieldPolicyLeader:    false,
				inboxmanagerreceivepucl.DocFieldBrokerReference: false,
				inboxmanagerreceivepucl.DocFieldInsuredEmail:    true,
				inboxmanagerreceivepucl.DocFieldDriverLicence:   true,
			},
		},
		// Travel dikecualikan dari Polis Leader dan No Ref Broker, SAMA seperti PA — dan
		// itulah sebabnya Group Panel mentah dibawa berdampingan dengan Jenis Klaim, yang
		// tidak membedakan Travel dari lini lain.
		"Travel berpolis": {
			detail: inboxmanagerreceivepucl.ReceiveDocument{GroupPanel: "005", PolicyNumber: "P-3"},
			want: map[string]bool{
				inboxmanagerreceivepucl.DocFieldPolicyLeader:    false,
				inboxmanagerreceivepucl.DocFieldBrokerReference: false,
			},
		},
		// Tanpa nomor polis, dua syarat gugur — tetapi "SIM Pengendara" TIDAK menuntut
		// nomor polis, dan itu keadaan di section apa adanya.
		"PA tanpa polis": {
			detail: inboxmanagerreceivepucl.ReceiveDocument{GroupPanel: "002"},
			want: map[string]bool{
				inboxmanagerreceivepucl.DocFieldInsuredEmail:  false,
				inboxmanagerreceivepucl.DocFieldDriverLicence: true,
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			groups := inboxmanagerreceivepucl.DocumentFieldGroupsFor(tc.detail)
			for key, want := range tc.want {
				if punya(groups, key) != want {
					t.Fatalf("isian %q tergambar = %v, seharusnya %v", key, !want, want)
				}
			}
			// Isian tanpa syarat selalu ada, apa pun berkasnya.
			if !punya(groups, inboxmanagerreceivepucl.DocFieldInsuredName) {
				t.Fatal("isian tanpa syarat ikut tersaring")
			}
		})
	}
}

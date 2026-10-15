package inboxmanageradmin_test

import (
	"errors"
	"testing"
	"time"

	"claim-pnc/internal/inboxmanageradmin"
)

// Uji di berkas ini menguji ATURAN modul, bukan fungsi kecil apa pun: setiap namanya
// terbaca sebagai kalimat yang menyatakan aturannya (`14-TESTING-STRATEGY.md` §3.2).

func TestTigaTabMewakiliTigaUnitOrganisasi(t *testing.T) {
	tabs := inboxmanageradmin.Tabs()

	if len(tabs) != 3 {
		t.Fatalf("jumlah tab %d, seharusnya 3", len(tabs))
	}

	// Ejaan unit organisasi adalah KONTRAK: ia dibandingkan langsung dengan isi kolom
	// PXASSIGNEDORGUNIT. Merapikan "AdminTRAVEL" menjadi "AdminTravel" akan membuat tabnya
	// mengembalikan nol baris tanpa satu pun pesan galat, dan uji inilah yang menahannya.
	want := []string{"AdminPNC", "AdminPA", "AdminTRAVEL"}
	for i, tab := range tabs {
		if tab.OrgUnit != want[i] {
			t.Errorf("tab %q menyaring unit organisasi %q, seharusnya %q",
				tab.Name, tab.OrgUnit, want[i])
		}
	}
}

func TestKetigaTabBerbagiSusunanKolomYangSama(t *testing.T) {
	tabs := inboxmanageradmin.Tabs()

	first := tabs[0].Columns
	for _, tab := range tabs[1:] {
		if len(tab.Columns) != len(first) {
			t.Fatalf("tab %q punya %d kolom, tab pertama %d",
				tab.Name, len(tab.Columns), len(first))
		}
		for i, column := range tab.Columns {
			if column != first[i] {
				t.Errorf("tab %q kolom ke-%d adalah %+v, seharusnya %+v",
					tab.Name, i, column, first[i])
			}
		}
	}
}

func TestJudulKolomMengikutiCaptionSection(t *testing.T) {
	// Kedelapan judul di bawah disalin dari `<pyCaption …>` pada
	// `Section/InboxManagerAdmin_Section-Section.xml`. Yang paling mudah "dirapikan" tanpa
	// sengaja adalah yang pertama — "ID", yang di layar inbox lain berbunyi "Case ID".
	want := []string{
		"ID",
		"No Polis",
		"Nama Tertanggung",
		"Nama Bisnis",
		"Nama Sumber Bisnis",
		"Tanggal Pendaftaran",
		"Lama Waktu Klaim",
		"Admin PNC",
	}

	columns := inboxmanageradmin.Tabs()[0].Columns
	if len(columns) != len(want) {
		t.Fatalf("jumlah kolom %d, seharusnya %d", len(columns), len(want))
	}
	for i, title := range want {
		if columns[i].Title != title {
			t.Errorf("kolom ke-%d berjudul %q, seharusnya %q", i, columns[i].Title, title)
		}
	}
}

func TestLiniBisnisMembukaHanyaSatuTab(t *testing.T) {
	cases := []struct {
		line    string
		wantTab string
	}{
		{inboxmanageradmin.LineNonMBU, "Manajemen Admin - Non MBU"},
		{inboxmanageradmin.LinePA, "Manajemen Admin - PA"},
		{inboxmanageradmin.LineTravel, "Manajemen Admin - Travel"},
	}

	for _, c := range cases {
		caller := inboxmanageradmin.Caller{Login: "petugas", LineBusiness: c.line}
		visible := inboxmanageradmin.VisibleTabs(caller)

		if len(visible) != 1 {
			t.Fatalf("lini bisnis %q membuka %d tab, seharusnya 1", c.line, len(visible))
		}
		if visible[0].Name != c.wantTab {
			t.Errorf("lini bisnis %q membuka %q, seharusnya %q",
				c.line, visible[0].Name, c.wantTab)
		}
	}
}

func TestUnitOrganisasiDevelopmentMembukaKetigaTab(t *testing.T) {
	// Di sistem lama ketiga `pyContainerVisibleWhen` menyebut
	// `OperatorID.pyOrgUnit = 'Development'` sebagai alternatif kedua setelah lini bisnis.
	// Lini bisnisnya sengaja dikosongkan supaya yang diuji benar-benar cabang kedua.
	caller := inboxmanageradmin.Caller{
		Login:   "pengembang",
		OrgUnit: inboxmanageradmin.DevelopmentOrgUnit,
	}

	if got := len(inboxmanageradmin.VisibleTabs(caller)); got != 3 {
		t.Fatalf("unit organisasi Development membuka %d tab, seharusnya 3", got)
	}
}

func TestLiniBisnisDibandingkanTanpaMemandangHurufBesarKecil(t *testing.T) {
	// Pelonggaran yang disengaja terhadap Pega, dan satu-satunya. Ia hanya dapat MENAMBAH
	// tab yang terlihat, tidak pernah menghilangkan — kolom `M_LOGIN_PNC.LINE_BUSINESS`
	// bebas isi, dan ejaannya diketik manusia.
	for _, written := range []string{"pa", "Pa", " PA ", "pA"} {
		caller := inboxmanageradmin.Caller{Login: "petugas", LineBusiness: written}

		visible := inboxmanageradmin.VisibleTabs(caller)
		if len(visible) != 1 {
			t.Fatalf("lini bisnis %q membuka %d tab, seharusnya 1", written, len(visible))
		}
		if visible[0].Code != inboxmanageradmin.TabPA {
			t.Errorf("lini bisnis %q membuka tab %q, seharusnya PA", written, visible[0].Code)
		}
	}
}

func TestNilaiDiLuarKetiganyaTidakMembukaSatuTabPun(t *testing.T) {
	// Nilai di bawah adalah JABATAN KEPEGAWAIAN HCQ, dan itu disengaja: sampai 2026-09-27
	// nilai seperti inilah yang dibandingkan dengan ketiga tab, karena `Caller.LineBusiness`
	// diisi dari `EmpResponse.Placement.PositionName`. Akibatnya tidak seorang pun melihat
	// satu tab pun.
	//
	// Uji ini menahan nilai semacam itu kembali dianggap sah. Sejak koreksi, yang
	// dibandingkan adalah `M_LOGIN_PNC.LINE_BUSINESS` — dan bila kolom itu kebetulan diisi
	// jabatan, hasilnya tetap nol tab, bukan tab yang salah.
	caller := inboxmanageradmin.Caller{Login: "petugas", LineBusiness: "IT SPECIALIST"}

	if got := len(inboxmanageradmin.VisibleTabs(caller)); got != 0 {
		t.Fatalf("nilai di luar ketiganya membuka %d tab, seharusnya 0", got)
	}
	if got := inboxmanageradmin.DefaultTabFor(caller); got != "" {
		t.Errorf("tab bawaan %q, seharusnya kosong", got)
	}
}

func TestTabBawaanAdalahTabPertamaYangBolehDilihat(t *testing.T) {
	// Non-MBU adalah tab bawaan menyeluruh, tetapi petugas PA tidak boleh melihatnya.
	// Layar karena itu membuka tab pertama yang BOLEH ia lihat — bukan memaksa yang bawaan
	// lalu ditolak server.
	travel := inboxmanageradmin.Caller{
		Login:        "petugas",
		LineBusiness: inboxmanageradmin.LineTravel,
	}
	if got := inboxmanageradmin.DefaultTabFor(travel); got != inboxmanageradmin.TabTravel {
		t.Errorf("tab bawaan petugas Travel %q, seharusnya %q",
			got, inboxmanageradmin.TabTravel)
	}

	dev := inboxmanageradmin.Caller{
		Login:   "pengembang",
		OrgUnit: inboxmanageradmin.DevelopmentOrgUnit,
	}
	if got := inboxmanageradmin.DefaultTabFor(dev); got != inboxmanageradmin.DefaultTab {
		t.Errorf("tab bawaan pengembang %q, seharusnya %q", got, inboxmanageradmin.DefaultTab)
	}
}

func TestPermintaanTanpaIdentitasDitolak(t *testing.T) {
	_, err := inboxmanageradmin.NewQuery(
		inboxmanageradmin.QueryInput{},
		inboxmanageradmin.Caller{LineBusiness: inboxmanageradmin.LinePA},
	)

	if !errors.Is(err, inboxmanageradmin.ErrCallerUnknown) {
		t.Fatalf("galat %v, seharusnya ErrCallerUnknown", err)
	}
}

func TestPermintaanAtasTabYangBukanHaknyaDitolak(t *testing.T) {
	// Inilah yang membedakan modul ini dari sistem lama: di Pega tab yang tidak boleh
	// dilihat sekadar DISEMBUNYIKAN, sehingga tautan yang disusun tangan tetap dilayani.
	// Di sini server yang menolak.
	_, err := inboxmanageradmin.NewQuery(
		inboxmanageradmin.QueryInput{Tab: inboxmanageradmin.TabTravel},
		inboxmanageradmin.Caller{Login: "petugas", LineBusiness: inboxmanageradmin.LinePA},
	)

	if !errors.Is(err, inboxmanageradmin.ErrTabNotAllowed) {
		t.Fatalf("galat %v, seharusnya ErrTabNotAllowed", err)
	}
}

func TestPemanggilTanpaSatuTabPunDijawabGalatTersendiri(t *testing.T) {
	// Dibedakan dari ErrTabNotAllowed dengan sengaja: yang satu berarti pengguna salah
	// alamat, yang ini berarti layarnya memang tidak punya apa pun untuknya. Menjawab
	// keduanya sama akan membuat pengguna mencoba tab lain satu per satu.
	_, err := inboxmanageradmin.NewQuery(
		inboxmanageradmin.QueryInput{},
		inboxmanageradmin.Caller{Login: "petugas", LineBusiness: "IT SPECIALIST"},
	)

	if !errors.Is(err, inboxmanageradmin.ErrNoTabAllowed) {
		t.Fatalf("galat %v, seharusnya ErrNoTabAllowed", err)
	}
}

func TestTabTidakDikenalDijawabGalatValidasi(t *testing.T) {
	_, err := inboxmanageradmin.NewQuery(
		inboxmanageradmin.QueryInput{Tab: "99"},
		inboxmanageradmin.Caller{Login: "petugas", LineBusiness: inboxmanageradmin.LinePA},
	)

	var validation *inboxmanageradmin.ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("galat %v, seharusnya ValidationError", err)
	}
	if len(validation.Violations) != 1 ||
		validation.Violations[0].Field != inboxmanageradmin.FieldTab {
		t.Errorf("pelanggaran %+v, seharusnya menunjuk isian %q",
			validation.Violations, inboxmanageradmin.FieldTab)
	}
}

func TestPermintaanTanpaKodeTabMemakaiTabPertamaYangBolehDilihat(t *testing.T) {
	query, err := inboxmanageradmin.NewQuery(
		inboxmanageradmin.QueryInput{},
		inboxmanageradmin.Caller{Login: "petugas", LineBusiness: inboxmanageradmin.LinePA},
	)
	if err != nil {
		t.Fatalf("permintaan ditolak: %v", err)
	}

	if query.Tab.Code != inboxmanageradmin.TabPA {
		t.Errorf("tab %q, seharusnya %q", query.Tab.Code, inboxmanageradmin.TabPA)
	}
	if query.Tab.OrgUnit != inboxmanageradmin.OrgUnitPA {
		t.Errorf("unit organisasi %q, seharusnya %q",
			query.Tab.OrgUnit, inboxmanageradmin.OrgUnitPA)
	}
}

func TestHalamanDipotongSesuaiUkuranYangDibetulkan(t *testing.T) {
	rows := make([]inboxmanageradmin.WorkItem, 125)
	for i := range rows {
		rows[i] = inboxmanageradmin.WorkItem{CaseID: "PNC-" + itoa(i)}
	}

	// Ukuran bawaan 50 mengikuti `pyPageSize` ketiga grid Pega.
	page := inboxmanageradmin.Slice(rows, inboxmanageradmin.Pagination{})
	if got := len(page.Items); got != 50 {
		t.Errorf("halaman berisi %d baris, seharusnya 50", got)
	}
	if page.Total != 125 {
		t.Errorf("total %d, seharusnya 125", page.Total)
	}
	if got := page.TotalPages(); got != 3 {
		t.Errorf("total halaman %d, seharusnya 3", got)
	}

	// Halaman terakhir berisi sisanya, bukan halaman penuh.
	last := inboxmanageradmin.Slice(rows, inboxmanageradmin.Pagination{Page: 3, Size: 50})
	if got := len(last.Items); got != 25 {
		t.Errorf("halaman terakhir berisi %d baris, seharusnya 25", got)
	}
}

func TestHalamanDiLuarJangkauanMengembalikanSenaraiKosongBukanNil(t *testing.T) {
	// `[]` dan `null` ditangani berbeda oleh klien, dan yang kedua memaksa setiap layar
	// memeriksanya lebih dulu.
	page := inboxmanageradmin.Slice(
		[]inboxmanageradmin.WorkItem{{CaseID: "PNC-1"}},
		inboxmanageradmin.Pagination{Page: 9, Size: 50},
	)

	if page.Items == nil {
		t.Fatal("Items nil, seharusnya senarai kosong")
	}
	if len(page.Items) != 0 {
		t.Errorf("halaman di luar jangkauan berisi %d baris, seharusnya 0", len(page.Items))
	}
	if page.Total != 1 {
		t.Errorf("total %d, seharusnya tetap 1", page.Total)
	}
}

func TestHasilKosongTetapSatuHalaman(t *testing.T) {
	// Supaya layar tidak pernah menggambar "halaman 1 dari 0".
	page := inboxmanageradmin.Slice(nil, inboxmanageradmin.Pagination{})
	if got := page.TotalPages(); got != 1 {
		t.Errorf("total halaman %d, seharusnya 1", got)
	}
}

func TestUkuranHalamanDiBatasiSeratus(t *testing.T) {
	// `10-API-STRATEGY.md` §4: permintaan yang lebih besar DIBETULKAN ke batas, bukan
	// dipenuhi diam-diam.
	clean := inboxmanageradmin.Pagination{Page: 1, Size: 5000}.Normalize()
	if clean.Size != inboxmanageradmin.MaxPageSize {
		t.Errorf("ukuran %d, seharusnya %d", clean.Size, inboxmanageradmin.MaxPageSize)
	}
}

func TestLamaWaktuKlaimTerisiDariTanggalPendaftaran(t *testing.T) {
	registered := time.Date(2026, time.August, 26, 3, 0, 0, 0, time.UTC)
	now := time.Date(2026, time.September, 26, 3, 0, 0, 0, time.UTC)

	filled := inboxmanageradmin.WorkItem{RegisteredAt: &registered}.WithElapsed(now)
	if filled.ClaimElapsed != "1 month ago" {
		t.Errorf("lama waktu klaim %q, seharusnya %q", filled.ClaimElapsed, "1 month ago")
	}
}

func TestBarisTanpaTanggalPendaftaranTidakBerbunyiNolMenit(t *testing.T) {
	// `P-5` butir 13: nilai gagal yang tidak dapat dibedakan dari nol adalah cacat yang
	// justru sedang diperbaiki migrasi ini.
	filled := inboxmanageradmin.WorkItem{}.WithElapsed(time.Now())
	if filled.ClaimElapsed != "" {
		t.Errorf("lama waktu klaim %q, seharusnya kosong", filled.ClaimElapsed)
	}
}

// itoa menuliskan bilangan bulat kecil tanpa menarik strconv ke berkas uji.
func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	digits := ""
	for value > 0 {
		digits = string(rune('0'+value%10)) + digits
		value /= 10
	}
	return digits
}

func TestStatusKlaimDiturunkanDariStatusKerja(t *testing.T) {
	// Koreksi Work Owner 2026-09-27: kolom Status Klaim tidak lagi dicari ke `V_STS_CLAIM`
	// lewat `STATUSCLAIM_1`, melainkan diturunkan dari `PYSTATUSWORK`.
	//
	// Pemetaannya dari `RDB List/BrowseClaimALL-SQL.xml`, sama dengan modul Inbox
	// Outstanding.
	cases := []struct {
		workStatus string
		want       string
	}{
		{"New", "On Progress"},
		{"Resolved-Completed", "Close"},
		{"Resolved-Rejected", "Reject"},
		{" New ", "On Progress"},
		{"", ""},
	}

	for _, c := range cases {
		if got := inboxmanageradmin.DisplayStatusFor(c.workStatus); got != c.want {
			t.Errorf("status kerja %q menjadi %q, seharusnya %q", c.workStatus, got, c.want)
		}
	}
}

func TestStatusKerjaAsingTIDAKDipaksaMenjadiOnProgress(t *testing.T) {
	// Jebakan yang sudah pernah menggigit, tercatat di `inboxoutstanding.go`: cabang
	// default yang menghasilkan "On Progress" membuat klaim yang SUDAH DITUTUP berlabel
	// masih berjalan — tanpa satu pun galat.
	//
	// Nilai asing karena itu dikembalikan apa adanya. Status yang menyamar tidak pernah
	// ditanyakan siapa pun; status yang tampil apa adanya ditanyakan pada hari pertama.
	for _, asing := range []string{"Pending-Approval", "Open", "RESOLVED-COMPLETED"} {
		got := inboxmanageradmin.DisplayStatusFor(asing)

		if got == inboxmanageradmin.DisplayOnProgress {
			t.Errorf("status asing %q dipaksa menjadi %q", asing, inboxmanageradmin.DisplayOnProgress)
		}
		if got != asing {
			t.Errorf("status asing %q menjadi %q, seharusnya apa adanya", asing, got)
		}
	}
}

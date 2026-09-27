package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"claim-pnc/internal/inboxmanageradmin"
	"claim-pnc/internal/inboxmanageradmin/repo/memory"
	"claim-pnc/internal/inboxmanageradmin/usecase"
)

// tetapWaktu adalah seam Clock dengan jam yang tidak bergerak, supaya kolom Lama Waktu
// Klaim dapat diuji secara deterministik.
type tetapWaktu struct{ at time.Time }

func (t tetapWaktu) Now() time.Time { return t.at }

const portalUtama = "ASM"

// sekarang dipilih sama dengan waktu dasar data contoh, sehingga selisih hari pada uji di
// bawah dapat dihitung di kepala.
var sekarang = time.Date(2026, time.September, 26, 8, 0, 0, 0, time.UTC)

// loginPetugas adalah login yang dipakai seluruh uji berperan petugas biasa.
const loginPetugas = "petugas.contoh"

func layanan(t *testing.T) *usecase.Service {
	t.Helper()
	return layananDengan(t, memory.NewSampleStore())
}

// layananUntuk membentuk layanan yang lini bisnis loginPetugas-nya sudah ditetapkan.
//
// # Kenapa lini bisnisnya ditetapkan di PENYIMPANAN, bukan di Caller
//
// Karena sejak 2026-09-27 usecase membacanya sendiri dari `M_LOGIN_PNC` lewat seam, dan
// mengabaikan apa pun yang dibawa pemanggil. Uji yang menyetelnya di Caller akan lulus
// sambil menguji jalur yang tidak lagi dipakai — persis kekeliruan yang menyebabkan cacat
// ini tidak tertangkap lebih awal.
func layananUntuk(t *testing.T, line string) *usecase.Service {
	t.Helper()

	store := memory.NewSampleStore()
	store.SetLineBusiness(loginPetugas, line)
	return layananDengan(t, store)
}

func layananDengan(t *testing.T, store *memory.Store) *usecase.Service {
	t.Helper()

	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (inboxmanageradmin.Repo, error) {
			if alias != portalUtama {
				return nil, errors.New("portal tidak tersedia: " + alias)
			}
			return store, nil
		},
		LineBusinessSelector: func(alias string) (inboxmanageradmin.LineBusinessRepo, error) {
			// Portal yang salah ditolak DI SINI PULA, bukan hanya pada antreannya:
			// membaca kewenangan dari basis data entitas lain adalah `R-20`.
			if alias != portalUtama {
				return nil, errors.New("portal tidak tersedia: " + alias)
			}
			return store, nil
		},
		Clock: tetapWaktu{at: sekarang},
	})
	if err != nil {
		t.Fatalf("membentuk layanan gagal: %v", err)
	}
	return service
}

// petugas adalah identitas yang dibawa transport: login saja, TANPA lini bisnis.
//
// Bentuk ini menirukan keadaan sungguhan sesudah koreksi 2026-09-27 — lapisan transport
// tidak lagi mengisi lini bisnis, karena ia tidak tahu portal mana yang aktif saat identitas
// dibaca.
func petugas() inboxmanageradmin.Caller {
	return inboxmanageradmin.Caller{Login: loginPetugas}
}

func pengembang() inboxmanageradmin.Caller {
	return inboxmanageradmin.Caller{
		Login:   "pengembang.contoh",
		OrgUnit: inboxmanageradmin.DevelopmentOrgUnit,
	}
}

func TestKeteranganLayarHanyaMemuatTabYangBolehDilihat(t *testing.T) {
	service := layananUntuk(t, inboxmanageradmin.LinePA)

	meta, err := service.Metadata(context.Background(), portalUtama, petugas())
	if err != nil {
		t.Fatalf("keterangan layar gagal dimuat: %v", err)
	}
	if len(meta.Tabs) != 1 {
		t.Fatalf("petugas PA melihat %d tab, seharusnya 1", len(meta.Tabs))
	}
	if meta.Tabs[0].Code != inboxmanageradmin.TabPA {
		t.Errorf("tab yang terlihat %q, seharusnya %q", meta.Tabs[0].Code, inboxmanageradmin.TabPA)
	}

	// Ketiga tab tetap dikirim terpisah, supaya layar dapat menyebutkan apa saja yang ada
	// ketika pengguna tidak berhak atas satu pun.
	if len(meta.AllTabs) != 3 {
		t.Errorf("AllTabs memuat %d tab, seharusnya 3", len(meta.AllTabs))
	}
}

func TestKeteranganLayarMemantulkanLiniBisnisPemanggil(t *testing.T) {
	// Pengguna yang tidak melihat satu tab pun perlu tahu nilai APA yang terbaca sistem:
	// itulah satu-satunya petunjuk yang dapat ia sampaikan saat melapor, dan yang
	// membedakan "kolomnya belum diisi" dari "diisi dengan nilai yang tidak dikenal".
	//
	// Nilai di bawah sengaja jabatan kepegawaian HCQ. Sampai 2026-09-27 nilai SEPERTI INILAH
	// yang dipakai membandingkan tab, dan karena itu tidak seorang pun melihat satu tab pun.
	service := layananUntuk(t, "IT SPECIALIST")

	meta, err := service.Metadata(context.Background(), portalUtama, petugas())
	if err != nil {
		t.Fatalf("keterangan layar gagal dimuat: %v", err)
	}
	if len(meta.Tabs) != 0 {
		t.Fatalf("nilai di luar ketiganya melihat %d tab, seharusnya 0", len(meta.Tabs))
	}
	if meta.CallerLineBusiness != "IT SPECIALIST" {
		t.Errorf("lini bisnis yang dipantulkan %q, seharusnya %q",
			meta.CallerLineBusiness, "IT SPECIALIST")
	}
	if meta.DefaultTab != "" {
		t.Errorf("tab bawaan %q, seharusnya kosong", meta.DefaultTab)
	}
	if len(meta.ExpectedLineBusinesses) != 3 {
		t.Errorf("lini bisnis yang diharapkan %v, seharusnya tiga", meta.ExpectedLineBusinesses)
	}
}

func TestSetiapTabHanyaBerisiUnitOrganisasinyaSendiri(t *testing.T) {
	service := layanan(t)

	cases := []struct {
		tab      string
		wantCase []string
	}{
		// Urutannya menurun menurut tanggal pendaftaran. PNC-9002 tanggalnya KOSONG dan
		// karena itu berada paling bawah — bukan paling atas.
		{inboxmanageradmin.TabNonMBU, []string{"PNC-9001", "PNC-9002"}},
		{inboxmanageradmin.TabPA, []string{"PNC-9101", "PNC-9102"}},
		{inboxmanageradmin.TabTravel, []string{"PNC-9201"}},
	}

	for _, c := range cases {
		listed, err := service.List(
			context.Background(), portalUtama, pengembang(),
			inboxmanageradmin.QueryInput{Tab: c.tab}, inboxmanageradmin.Pagination{},
		)
		if err != nil {
			t.Fatalf("tab %s gagal dimuat: %v", c.tab, err)
		}

		got := make([]string, 0, len(listed.Page.Items))
		for _, item := range listed.Page.Items {
			got = append(got, item.CaseID)
		}

		if len(got) != len(c.wantCase) {
			t.Fatalf("tab %s berisi %v, seharusnya %v", c.tab, got, c.wantCase)
		}
		for i, want := range c.wantCase {
			if got[i] != want {
				t.Errorf("tab %s baris ke-%d adalah %q, seharusnya %q", c.tab, i, got[i], want)
			}
		}
	}
}

func TestKlaimSelesaiDitolakDanKlaimDitolakTidakMuncul(t *testing.T) {
	// Keduanya ada di data contoh pada unit organisasi yang tabnya diuji, sehingga bila
	// penyaring status kerja terlewat mereka akan muncul di sini.
	service := layanan(t)

	listed, err := service.List(
		context.Background(), portalUtama, pengembang(),
		inboxmanageradmin.QueryInput{Tab: inboxmanageradmin.TabNonMBU},
		inboxmanageradmin.Pagination{},
	)
	if err != nil {
		t.Fatalf("tab gagal dimuat: %v", err)
	}

	for _, item := range listed.Page.Items {
		if item.CaseID == "PNC-9003" {
			t.Error("klaim yang sudah selesai muncul di antrean")
		}
	}
}

func TestBerkasPenerimaanDokumenTidakBocorKeAntrean(t *testing.T) {
	// Tabel POOLDATA.T_CLAIMLIST_ADMIN memuat lebih dari satu jenis objek kerja — 142
	// baris RCV tercatat di README. Tanpa penyaring PXOBJCLASS, berkas RCV ikut tampil.
	service := layanan(t)

	listed, err := service.List(
		context.Background(), portalUtama, pengembang(),
		inboxmanageradmin.QueryInput{Tab: inboxmanageradmin.TabNonMBU},
		inboxmanageradmin.Pagination{},
	)
	if err != nil {
		t.Fatalf("tab gagal dimuat: %v", err)
	}

	for _, item := range listed.Page.Items {
		if item.CaseID == "RCV-7001" {
			t.Error("berkas penerimaan dokumen muncul di antrean klaim")
		}
	}
}

func TestPenugasanPadaUnitOrganisasiLainTidakMuncul(t *testing.T) {
	service := layanan(t)

	for _, tab := range []string{
		inboxmanageradmin.TabNonMBU,
		inboxmanageradmin.TabPA,
		inboxmanageradmin.TabTravel,
	} {
		listed, err := service.List(
			context.Background(), portalUtama, pengembang(),
			inboxmanageradmin.QueryInput{Tab: tab}, inboxmanageradmin.Pagination{},
		)
		if err != nil {
			t.Fatalf("tab %s gagal dimuat: %v", tab, err)
		}

		for _, item := range listed.Page.Items {
			if item.CaseID == "PNC-9301" {
				t.Errorf("penugasan unit organisasi lain muncul di tab %s", tab)
			}
		}
	}
}

func TestPermintaanAtasTabYangBukanHaknyaDitolakLayanan(t *testing.T) {
	// Penegakan di server, bukan hanya penyembunyian di layar. Tautan lama yang menyimpan
	// kode tab di riwayat peramban sampai ke sini.
	service := layananUntuk(t, inboxmanageradmin.LinePA)

	_, err := service.List(
		context.Background(), portalUtama, petugas(),
		inboxmanageradmin.QueryInput{Tab: inboxmanageradmin.TabTravel},
		inboxmanageradmin.Pagination{},
	)

	if !errors.Is(err, inboxmanageradmin.ErrTabNotAllowed) {
		t.Fatalf("galat %v, seharusnya ErrTabNotAllowed", err)
	}
}

func TestPortalLainDitolakBukanDialihkanKePortalUtama(t *testing.T) {
	// `R-20`: jatuh ke koneksi bawaan berarti menampilkan antrean satu badan hukum kepada
	// petugas badan hukum lain tanpa satu pun pesan galat.
	service := layanan(t)

	_, err := service.List(
		context.Background(), "SIMASNET", pengembang(),
		inboxmanageradmin.QueryInput{}, inboxmanageradmin.Pagination{},
	)

	if err == nil {
		t.Fatal("portal lain dilayani, seharusnya ditolak")
	}
}

func TestLamaWaktuKlaimTerisiSaatDaftarDimuat(t *testing.T) {
	service := layanan(t)

	listed, err := service.List(
		context.Background(), portalUtama, pengembang(),
		inboxmanageradmin.QueryInput{Tab: inboxmanageradmin.TabPA},
		inboxmanageradmin.Pagination{},
	)
	if err != nil {
		t.Fatalf("tab gagal dimuat: %v", err)
	}

	// PNC-9101 terdaftar sehari sebelum `sekarang`.
	if got := listed.Page.Items[0].ClaimElapsed; got != "1 day ago" {
		t.Errorf("lama waktu klaim %q, seharusnya %q", got, "1 day ago")
	}

	// PNC-9102 terdaftar 400 hari sebelumnya — melewati setahun, sehingga dua satuan
	// ditampilkan sekaligus.
	if got := listed.Page.Items[1].ClaimElapsed; got != "1 year 1 month ago" {
		t.Errorf("lama waktu klaim %q, seharusnya %q", got, "1 year 1 month ago")
	}
}

func TestBarisTanpaTanggalPendaftaranBerakhirDiBawahDanTanpaLamaWaktu(t *testing.T) {
	service := layanan(t)

	listed, err := service.List(
		context.Background(), portalUtama, pengembang(),
		inboxmanageradmin.QueryInput{Tab: inboxmanageradmin.TabNonMBU},
		inboxmanageradmin.Pagination{},
	)
	if err != nil {
		t.Fatalf("tab gagal dimuat: %v", err)
	}

	last := listed.Page.Items[len(listed.Page.Items)-1]
	if last.CaseID != "PNC-9002" {
		t.Fatalf("baris terakhir %q, seharusnya PNC-9002", last.CaseID)
	}
	if last.RegisteredAt != nil {
		t.Error("tanggal pendaftaran terisi, seharusnya kosong")
	}
	if last.ClaimElapsed != "" {
		t.Errorf("lama waktu klaim %q, seharusnya kosong", last.ClaimElapsed)
	}
}

func TestPermintaanMengembalikanTabYangBenarBenarDipakai(t *testing.T) {
	// Layar menggambar keadaan tabnya dari sini, bukan dari isian yang ia kirim:
	// permintaan tanpa kode tab dijawab dengan tab pertama yang boleh dilihat pemanggil.
	service := layananUntuk(t, inboxmanageradmin.LineTravel)

	listed, err := service.List(
		context.Background(), portalUtama, petugas(),
		inboxmanageradmin.QueryInput{}, inboxmanageradmin.Pagination{},
	)
	if err != nil {
		t.Fatalf("permintaan gagal: %v", err)
	}

	if listed.Query.Tab.Code != inboxmanageradmin.TabTravel {
		t.Errorf("tab yang dipakai %q, seharusnya %q",
			listed.Query.Tab.Code, inboxmanageradmin.TabTravel)
	}
}

func TestLayananMenolakDibentukTanpaSeamWajib(t *testing.T) {
	if _, err := usecase.NewService(usecase.Options{Clock: tetapWaktu{}}); err == nil {
		t.Error("layanan terbentuk tanpa RepoSelector")
	}

	if _, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (inboxmanageradmin.Repo, error) { return nil, nil },
	}); err == nil {
		t.Error("layanan terbentuk tanpa Clock")
	}
}

func TestLiniBisnisDibacaDariPenyimpananBukanDariPemanggil(t *testing.T) {
	// Uji ini menjaga cacat 2026-09-27 tidak kembali.
	//
	// Sampai tanggal itu lini bisnis dibawa lapisan transport dari sesi — dari jabatan
	// kepegawaian HCQ — sehingga tidak seorang pun melihat satu tab pun. Sekarang usecase
	// membacanya sendiri dari `M_LOGIN_PNC`, dan apa pun yang dibawa pemanggil DIABAIKAN.
	//
	// Caller di bawah sengaja membawa nilai yang SALAH dan penyimpanan membawa yang BENAR.
	// Bila suatu saat nilai pemanggil dipakai kembali, uji ini gagal dengan nol tab.
	service := layananUntuk(t, inboxmanageradmin.LineTravel)

	menyesatkan := inboxmanageradmin.Caller{
		Login:        loginPetugas,
		LineBusiness: inboxmanageradmin.LinePA,
	}

	meta, err := service.Metadata(context.Background(), portalUtama, menyesatkan)
	if err != nil {
		t.Fatalf("keterangan layar gagal dimuat: %v", err)
	}

	if len(meta.Tabs) != 1 {
		t.Fatalf("melihat %d tab, seharusnya 1", len(meta.Tabs))
	}
	if meta.Tabs[0].Code != inboxmanageradmin.TabTravel {
		t.Errorf("tab yang terlihat %q, seharusnya %q — nilai dari pemanggil ikut terpakai",
			meta.Tabs[0].Code, inboxmanageradmin.TabTravel)
	}
	if meta.CallerLineBusiness != inboxmanageradmin.LineTravel {
		t.Errorf("lini bisnis yang dipantulkan %q, seharusnya %q",
			meta.CallerLineBusiness, inboxmanageradmin.LineTravel)
	}
}

func TestPetugasTanpaLiniBisnisTidakMembukaTabDanBUKANGalat(t *testing.T) {
	// Keadaan yang PALING UMUM di produksi hari ini: `LINE_BUSINESS` baru terisi pada
	// sebagian petugas (`migrations/0004_DICABUT.md`).
	//
	// Ia harus menghasilkan layar yang MENJELASKAN DIRI, bukan galat. Di Pega pun
	// `pyPosition` yang tidak cocok satu pun sekadar tidak membuka kontainer mana pun — ia
	// tidak menggagalkan layarnya.
	//
	// NewStrictStore, BUKAN NewSampleStore: yang kedua memberi lini bisnis bawaan supaya
	// `PENYIMPANAN=memori` dapat dipakai mengembangkan, dan nilai bawaan itu justru
	// menyembunyikan keadaan yang diuji di sini.
	service := layananDengan(t, memory.NewStrictStore())

	meta, err := service.Metadata(context.Background(), portalUtama, petugas())
	if err != nil {
		t.Fatalf("petugas tanpa lini bisnis menghasilkan galat %v, seharusnya tidak", err)
	}

	if len(meta.Tabs) != 0 {
		t.Errorf("melihat %d tab, seharusnya 0", len(meta.Tabs))
	}
	if meta.CallerLineBusiness != "" {
		t.Errorf("lini bisnis %q, seharusnya kosong", meta.CallerLineBusiness)
	}

	// Ketiga tab tetap disebutkan, supaya layar dapat menerangkan apa yang ada dan nilai
	// apa yang membukanya — bukan menampilkan layar kosong tanpa sebab.
	if len(meta.AllTabs) != 3 || len(meta.ExpectedLineBusinesses) != 3 {
		t.Error("keterangan tab tidak lengkap saat pemanggil tidak berhak atas satu pun")
	}
}

func TestPortalLainDitolakSaatMembacaKewenangan(t *testing.T) {
	// `R-20`: `M_LOGIN_PNC` adalah tabel PER ENTITAS. Membacanya dari koneksi portal yang
	// salah berarti menilai kewenangan dengan data badan hukum lain — dan itu tidak
	// menghasilkan galat apa pun bila dibiarkan jatuh ke koneksi bawaan.
	service := layananUntuk(t, inboxmanageradmin.LinePA)

	if _, err := service.Metadata(context.Background(), "SIMASNET", petugas()); err == nil {
		t.Fatal("portal lain dilayani saat membaca kewenangan, seharusnya ditolak")
	}
}

func TestPenyimpananContohMemberiLiniBisnisBawaanBagiPengembangan(t *testing.T) {
	// Menjaga cacat 2026-09-27 (kedua) tidak kembali.
	//
	// Login datang dari HCQ sungguhan dan tidak dapat diketahui di muka, sehingga tanpa
	// nilai bawaan tidak ada satu pun kunci yang cocok — dan `PENYIMPANAN=memori` tidak
	// menampilkan satu tab pun kepada siapa pun. Itu bukan tiruan produksi melainkan
	// kelumpuhan: modul ini tidak dapat dikembangkan tanpa Oracle.
	service := layanan(t) // NewSampleStore — fixture pengembangan

	meta, err := service.Metadata(context.Background(), portalUtama, petugas())
	if err != nil {
		t.Fatalf("keterangan layar gagal dimuat: %v", err)
	}

	if len(meta.Tabs) == 0 {
		t.Fatal("penyimpanan contoh tidak membuka satu tab pun — mode memori lumpuh")
	}
	if meta.Tabs[0].Code != inboxmanageradmin.TabNonMBU {
		t.Errorf("tab bawaan %q, seharusnya Non-MBU", meta.Tabs[0].Code)
	}

	// Yang DITETAPKAN tetap menang atas nilai bawaan — kalau tidak, tidak ada uji yang
	// dapat menguji lini bisnis selain Non-MBU.
	tegas := layananUntuk(t, inboxmanageradmin.LineTravel)

	metaTegas, err := tegas.Metadata(context.Background(), portalUtama, petugas())
	if err != nil {
		t.Fatalf("keterangan layar gagal dimuat: %v", err)
	}
	if len(metaTegas.Tabs) != 1 || metaTegas.Tabs[0].Code != inboxmanageradmin.TabTravel {
		t.Errorf("lini bisnis yang ditetapkan kalah oleh nilai bawaan: %+v", metaTegas.Tabs)
	}
}

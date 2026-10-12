package sqlstore

import (
	"errors"
	"strings"
	"testing"
	"time"

	"claim-pnc/internal/inboxmanageradmin"
)

// Uji di berkas ini menjaga tiga hal yang HARUS sejalan tetapi hidup di tiga berkas
// berbeda: alias kolom di inboxmanageradmin.sql, senarai resultColumns di query.go, dan
// urutan pemindai scanWorkItem di inboxmanageradmin.go.
//
// Ketiganya tidak dapat diperiksa kompilator. Yang menahan penyimpangannya hanyalah uji ini.

func TestSetiapKueriYangDipakaiAda(t *testing.T) {
	for _, name := range []string{"list_by_org_unit", "check_table", "check_line_business"} {
		if _, exists := queries[name]; !exists {
			t.Errorf("kueri %q tidak ada di berkas .sql", name)
		}
	}
}

func TestKueriDaftarMengembalikanKesembilanAliasBerurutan(t *testing.T) {
	text := query("list_by_org_unit")

	position := 0
	for _, alias := range resultColumns {
		marker := "AS " + alias
		found := strings.Index(text[position:], marker)
		if found < 0 {
			t.Fatalf("alias %q tidak ada di kueri, atau urutannya salah", alias)
		}
		position += found + len(marker)
	}
}

func TestJumlahAliasSamaDenganJumlahYangDipindai(t *testing.T) {
	// Pemindai yang kelebihan atau kekurangan satu kolom menghasilkan galat runtime yang
	// pesannya hanya menyebut jumlah — bukan kolom mana. Uji ini yang menyebut kolomnya.
	counted := strings.Count(query("list_by_org_unit"), " AS ")
	if counted != len(resultColumns) {
		t.Errorf("kueri mengembalikan %d kolom beralias, resultColumns memuat %d",
			counted, len(resultColumns))
	}

	scanned := 0
	_, err := scanWorkItem(countingScanner{counted: &scanned})
	if err != nil {
		t.Fatalf("memindai baris gagal: %v", err)
	}
	if scanned != len(resultColumns) {
		t.Errorf("scanWorkItem memindai %d kolom, resultColumns memuat %d",
			scanned, len(resultColumns))
	}
}

func TestKueriDaftarMenyaringUnitOrganisasiStatusKerjaDanKelasObjek(t *testing.T) {
	text := query("list_by_org_unit")

	// Ketiganya adalah penyaring Report Definition `ManagementAdminView`. Hilangnya salah
	// satu tidak menghasilkan galat apa pun — yang terjadi adalah baris yang seharusnya
	// tersembunyi ikut tampil, dan itu tidak terlihat sampai ada yang membandingkan dengan
	// Pega.
	wajib := []string{
		"PXASSIGNEDORGUNIT = :1",
		"'Resolved-Completed'",
		"'Resolved-Rejected'",
		"PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'",
	}

	for _, fragment := range wajib {
		if !strings.Contains(text, fragment) {
			t.Errorf("kueri kehilangan penyaring %q", fragment)
		}
	}
}

func TestKueriDaftarMengurutkanSecaraTetap(t *testing.T) {
	// Report Definition aslinya tidak mengurutkan sama sekali. Itu dapat dibiarkan selama
	// seluruh baris ditarik sekaligus, tetapi halaman yang dipotong dari urutan yang tidak
	// ditetapkan membuat satu baris muncul di dua halaman sekaligus hilang dari halaman
	// lain.
	text := query("list_by_org_unit")

	if !strings.Contains(text, "ORDER BY") {
		t.Fatal("kueri tidak punya ORDER BY")
	}
	if !strings.Contains(text, "PYID") {
		t.Error("ORDER BY tidak punya pemutus seri PYID")
	}
}

func TestKueriTidakMemakaiSintaksKhasOracleYangDilarang(t *testing.T) {
	// `08-TECHNICAL-STRATEGY.md` §4.3 dan `09-DATABASE-STRATEGY.md` §4. Yang paling mudah
	// terselip di modul ini adalah `TO_CHAR` untuk tanggal — dan justru itu yang membuat
	// pengurutan tanggal menjadi pengurutan teks.
	terlarang := []string{"NVL(", "SYSDATE", "ROWNUM", "DECODE(", "TO_CHAR(", "SELECT *"}

	for name, text := range queries {
		upper := strings.ToUpper(text)
		for _, pola := range terlarang {
			if strings.Contains(upper, pola) {
				t.Errorf("kueri %q memakai %q yang dilarang", name, pola)
			}
		}
	}
}

func TestKueriTidakMerangkaiNilaiKeDalamTeksSQL(t *testing.T) {
	// Ketiga nilai unit organisasi adalah satu-satunya nilai yang berbeda antar tab. Di
	// Pega ia ditanam section sebagai literal; kalau cara itu ikut tersalin ke sini, bind
	// `:1` akan hilang dan nilainya kembali dirangkai.
	text := query("list_by_org_unit")

	for _, unit := range []string{
		inboxmanageradmin.OrgUnitNonMBU,
		inboxmanageradmin.OrgUnitPA,
		inboxmanageradmin.OrgUnitTravel,
	} {
		if strings.Contains(text, unit) {
			t.Errorf("kueri memuat literal %q — nilainya seharusnya lewat bind :1", unit)
		}
	}
}

func TestKueriHanyaMemakaiKolomYangTerbuktiAda(t *testing.T) {
	// Menjaga kekeliruan 2026-09-27 tidak terulang: kueri sempat memakai `STATUSCLAIM_1`,
	// kolom yang memang BELUM ada, dan seluruh layar gagal dengan ORA-00904.
	//
	// Ketiadaan daftar kolom `T_CLAIMLIST_ADMIN` (lihat kepala berkas .sql) membuat hal itu
	// tidak dapat dijaga mesin. Yang dapat dijaga: kolom yang SUDAH TERBUKTI tidak ada
	// tidak kembali dipakai diam-diam.
	text := strings.ToUpper(query("list_by_org_unit"))

	for _, belumAda := range []string{"STATUSCLAIM_1", "V_STS_CLAIM"} {
		if strings.Contains(text, belumAda) {
			t.Errorf("kueri memakai %q — kolom itu belum ada di T_CLAIMLIST_ADMIN", belumAda)
		}
	}

	// Ketiganya sempat disimpulkan tidak ada lalu terbukti ADA. Dipakai kembali dengan
	// sengaja, dan disebut di sini supaya penghapusannya tidak dianggap perbaikan.
	if !strings.Contains(text, "PXASSIGNEDORGUNIT") {
		t.Error("kueri kehilangan penyaring PXASSIGNEDORGUNIT")
	}
	if !strings.Contains(text, "PXCREATEOPNAME") {
		t.Error("kueri kehilangan kolom PXCREATEOPNAME")
	}
}

func TestKueriTidakLagiMenyentuhSkemaDATAPEGA(t *testing.T) {
	// Sumber modul ini berpindah ke POOLDATA.T_CLAIMLIST_ADMIN (Work Owner 2026-09-27),
	// dan kedua tabel DATAPEGA tidak lagi dibaca sama sekali. Uji ini yang menahan
	// keduanya kembali diam-diam lewat satu join yang ditambahkan "sementara" — join
	// itulah yang membawa dua cacat yang baru saja hilang: klaim tanpa assignment lenyap,
	// klaim dengan banyak assignment tampil berkali-kali.
	for name, text := range queries {
		upper := strings.ToUpper(text)
		for _, dilarang := range []string{
			"DATAPEGA", "PC_ASM_FW_GCNMFW_WORK", "PC_ASSIGN_WORKLIST",
		} {
			if strings.Contains(upper, dilarang) {
				t.Errorf("kueri %q masih menyentuh %q", name, dilarang)
			}
		}
	}
}

func TestKueriDaftarMembacaSatuTabelSaja(t *testing.T) {
	// Tabel datar menyimpan satu baris per klaim. Satu JOIN saja mengembalikan kemungkinan
	// satu klaim tampil berkali-kali — persis cacat yang ditinggalkan bersama join lamanya.
	text := strings.ToUpper(query("list_by_org_unit"))

	if strings.Contains(text, "JOIN") {
		t.Error("kueri daftar memakai JOIN — sumbernya seharusnya satu tabel datar")
	}
	if !strings.Contains(text, "POOLDATA.T_CLAIMLIST_ADMIN") {
		t.Error("kueri daftar tidak membaca POOLDATA.T_CLAIMLIST_ADMIN")
	}
}

func TestPemindaiMenerimaSeluruhKolomNULL(t *testing.T) {
	// Setiap kolom dapat NULL: STATUSCLAIM_1 yang tidak punya padanan di V_STS_CLAIM
	// menghasilkan CLAIM_STATUS NULL, dan kolom snapshot polis memang kosong pada klaim
	// yang polisnya belum terbaca.
	item, err := scanWorkItem(nullScanner{})
	if err != nil {
		t.Fatalf("memindai baris ber-NULL gagal: %v", err)
	}

	if item.RegisteredAt != nil {
		t.Error("RegisteredAt terisi, seharusnya nil untuk kolom NULL")
	}
	if item.CaseID != "" || item.ClaimStatus != "" {
		t.Errorf("isian teks %+v, seharusnya kosong untuk kolom NULL", item)
	}
}

func TestPemindaiMeneruskanGalatApaAdanya(t *testing.T) {
	sentinel := errors.New("koneksi terputus")

	if _, err := scanWorkItem(failingScanner{err: sentinel}); !errors.Is(err, sentinel) {
		t.Fatalf("galat %v, seharusnya diteruskan apa adanya", err)
	}
}

// countingScanner menghitung berapa kolom yang diminta scanWorkItem.
type countingScanner struct{ counted *int }

func (s countingScanner) Scan(dest ...any) error {
	*s.counted = len(dest)
	return nil
}

// nullScanner meniru baris yang seluruh kolomnya NULL — yakni tidak menulisi satu pun
// tujuan pemindaian.
type nullScanner struct{}

func (nullScanner) Scan(...any) error { return nil }

// failingScanner meniru kegagalan basis data.
type failingScanner struct{ err error }

func (s failingScanner) Scan(...any) error { return s.err }

// Memastikan tipe waktu yang dipakai pemindai memang dari pustaka standar, sehingga uji di
// atas tidak diam-diam bergantung pada driver.
var _ = time.Time{}

func TestGalatKolomHilangDikenaliDanDibedakan(t *testing.T) {
	// Ketiga kolom yang diminta belum ada sampai DBA menjalankan migrations/0005 tahap 1,
	// sehingga setiap pembukaan antrean gagal dengan ORA-00904. Keadaan itu berlangsung
	// lama dan punya tindakan yang jelas — ia layak dibedakan dari kegagalan tak terduga,
	// supaya layar dapat menyebut kolomnya alih-alih "Terjadi kesalahan pada sistem".
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "ORA-00904 kolom belum ada",
			err:  errors.New(`ORA-00904: "A"."PXASSIGNEDORGUNIT": invalid identifier`),
			want: true,
		},
		{
			name: "galat lain tidak ikut dikenali",
			err:  errors.New("ORA-12541: TNS:no listener"),
			want: false,
		},
		{
			name: "nil bukan galat",
			err:  nil,
			want: false,
		},
	}

	for _, c := range cases {
		if got := isMissingColumn(c.err); got != c.want {
			t.Errorf("%s: dikenali %v, seharusnya %v", c.name, got, c.want)
		}
	}
}

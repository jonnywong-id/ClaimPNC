package inboxclaimtreatynonprop_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"claim-pnc/internal/inboxclaimtreatynonprop"
	"claim-pnc/internal/inboxclaimtreatynonprop/repo/memory"
)

// Uji di berkas ini menguji ATURAN modul, bukan fungsinya satu per satu: ia berjalan di
// atas penyimpanan memori yang meniru keempat penyaring SQL, sehingga yang dibuktikannya
// berlaku pula bagi yang berjalan di Oracle (`14-TESTING-STRATEGY.md` §3).

// callerAdmin1 adalah pemanggil yang memiliki dua baris biasa dan satu baris TBA di data
// contoh.
var callerAdmin1 = inboxclaimtreatynonprop.Caller{Login: "ADMINNONPROP1"}

// newQuery menyusun permintaan yang sah, dan menggagalkan uji bila tidak dapat dibentuk.
func newQuery(
	t *testing.T,
	tab string,
	seeAll, tbaOnly bool,
	caller inboxclaimtreatynonprop.Caller,
) inboxclaimtreatynonprop.Query {
	t.Helper()

	q, err := inboxclaimtreatynonprop.NewQuery(
		inboxclaimtreatynonprop.QueryInput{Tab: tab, SeeAll: seeAll, TBAOnly: tbaOnly},
		caller,
	)
	if err != nil {
		t.Fatalf("membentuk permintaan tab %q: %v", tab, err)
	}
	return q
}

// claimIDs mengambil nomor klaim satu halaman, supaya harapan uji terbaca sebagai daftar
// nomor alih-alih sebagai jumlah baris.
func claimIDs(page inboxclaimtreatynonprop.Page) []string {
	result := make([]string, 0, len(page.Items))
	for _, item := range page.Items {
		result = append(result, item.ClaimID)
	}
	return result
}

func equal(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

// listAll mengambil SELURUH baris yang cocok dalam satu halaman besar.
func listAll(
	t *testing.T,
	store *memory.Store,
	q inboxclaimtreatynonprop.Query,
) inboxclaimtreatynonprop.Page {
	t.Helper()

	page, err := store.List(
		context.Background(), q,
		inboxclaimtreatynonprop.Pagination{Page: 1, Size: inboxclaimtreatynonprop.MaxPageSize},
	)
	if err != nil {
		t.Fatalf("mengambil isi tab %q: %v", q.Tab.Code, err)
	}
	return page
}

// Tab Admin tanpa checkbox apa pun hanya memberi pekerjaan MILIK pemanggil.
//
// Ia sekaligus membuktikan ketiga penyaring yang mudah terlewat: baris milik operator lain
// tidak ikut, baris di workbasket tidak ikut, dan baris ber-awalan selain `CLMNP-` tidak
// ikut.
func TestTabAdminHanyaMilikPemanggil(t *testing.T) {
	store := memory.NewSampleStore()
	q := newQuery(t, inboxclaimtreatynonprop.TabAdmin, false, false, callerAdmin1)

	got := claimIDs(listAll(t, store, q))
	want := []string{"CLMNP-1004", "CLMNP-1002", "CLMNP-1001"}

	if !equal(got, want) {
		t.Fatalf("tab Admin milik pemanggil = %v, ingin %v", got, want)
	}
}

// "See All Claim" melepas penyaring kepemilikan, dan HANYA itu.
//
// Baris milik operator lain ikut muncul; baris workbasket dan baris ber-awalan lain tetap
// tidak.
func TestSeeAllMelepasKepemilikanSaja(t *testing.T) {
	store := memory.NewSampleStore()
	q := newQuery(t, inboxclaimtreatynonprop.TabAdmin, true, false, callerAdmin1)

	got := claimIDs(listAll(t, store, q))
	want := []string{"CLMNP-1004", "CLMNP-1002", "CLMNP-1003", "CLMNP-1001"}

	if !equal(got, want) {
		t.Fatalf("tab Admin dengan See All = %v, ingin %v", got, want)
	}
}

// "See TBA Claim" SENDIRIAN menyaring klaim milik pemanggil yang polisnya belum terbit.
//
// Inilah selisih terencana `P-5` yang disetujui Work Owner 2026-09-22. Di sistem lama
// kombinasi ini tidak menjalankan kueri apa pun yang berbeda, sehingga hasilnya akan sama
// dengan TestTabAdminHanyaMilikPemanggil di atas.
func TestTBASendirianMenyaringPolisKosong(t *testing.T) {
	store := memory.NewSampleStore()
	q := newQuery(t, inboxclaimtreatynonprop.TabAdmin, false, true, callerAdmin1)

	got := claimIDs(listAll(t, store, q))
	want := []string{"CLMNP-1004"}

	if !equal(got, want) {
		t.Fatalf("tab Admin dengan TBA saja = %v, ingin %v", got, want)
	}
}

// Kedua checkbox bersama meniru `GetKlaimNonPropAdminTBA_SQL` apa adanya: seluruh petugas,
// hanya yang polisnya belum terbit.
func TestSeeAllDanTBABersama(t *testing.T) {
	store := memory.NewSampleStore()
	q := newQuery(t, inboxclaimtreatynonprop.TabAdmin, true, true, callerAdmin1)

	got := claimIDs(listAll(t, store, q))
	want := []string{"CLMNP-1004"}

	if !equal(got, want) {
		t.Fatalf("tab Admin dengan kedua checkbox = %v, ingin %v", got, want)
	}
}

// Tab Teknik membaca antrean bersama, dan mengurutkannya MENAIK — berlawanan arah dengan
// tab Admin, persis seperti `ORDER BY CARI21` pada kuerinya.
//
// Ia sekaligus membuktikan baris workbasket milik antrean LAIN tersaring.
func TestTabTeknikAntreanBersamaUrutMenaik(t *testing.T) {
	store := memory.NewSampleStore()
	q := newQuery(t, inboxclaimtreatynonprop.TabTechnical, false, false, callerAdmin1)

	got := claimIDs(listAll(t, store, q))
	want := []string{"CLMNP-2001", "CLMNP-2002"}

	if !equal(got, want) {
		t.Fatalf("tab Teknik = %v, ingin %v", got, want)
	}
}

// Penyaring awalan TIDAK ikut menangkap klaim treaty proporsional maupun objek kerja
// komite, sekalipun keduanya ditugaskan kepada pemanggil yang sama.
//
// Ini yang membedakan layar ini dari layar saudaranya, dan salah satu-satunya penyaring
// yang bila longgar akan mencampur dua lini bisnis tanpa satu pun galat.
func TestAwalanKlaimMenyaringLayarSaudara(t *testing.T) {
	store := memory.NewSampleStore()
	q := newQuery(t, inboxclaimtreatynonprop.TabAdmin, true, false, callerAdmin1)

	for _, id := range claimIDs(listAll(t, store, q)) {
		if id == "CLMP-1001" {
			t.Fatal("klaim treaty PROPORSIONAL ikut terbawa ke layar non-prop")
		}
		if id == "KMTNP-3001" {
			t.Fatal("objek kerja KOMITE ikut terbawa ke tab Admin")
		}
	}
}

// Status menyatakan ANTREAN, bukan status klaim: seluruh baris satu tab memakai teks yang
// sama, dan teksnya berbeda antar tab.
//
// Ia diuji karena kedua teks itu literal di dalam kueri, bukan data — dan hal yang tidak
// datang dari data adalah hal yang paling mudah tertinggal saat kuerinya disunting.
func TestStatusMengikutiAntrean(t *testing.T) {
	store := memory.NewSampleStore()

	admin := listAll(t, store, newQuery(
		t, inboxclaimtreatynonprop.TabAdmin, false, false, callerAdmin1))
	for _, item := range admin.Items {
		if item.Status != inboxclaimtreatynonprop.StatusEstimation {
			t.Fatalf("status baris %q di tab Admin = %q, ingin %q",
				item.ClaimID, item.Status, inboxclaimtreatynonprop.StatusEstimation)
		}
	}

	technical := listAll(t, store, newQuery(
		t, inboxclaimtreatynonprop.TabTechnical, false, false, callerAdmin1))
	for _, item := range technical.Items {
		if item.Status != inboxclaimtreatynonprop.StatusAcceptation {
			t.Fatalf("status baris %q di tab Teknik = %q, ingin %q",
				item.ClaimID, item.Status, inboxclaimtreatynonprop.StatusAcceptation)
		}
	}
}

// Aging dihitung dalam HARI KALENDER dari pembuatan objek kerja, bukan hari kerja.
//
// Waktunya dipatok supaya uji ini tidak berubah hasilnya esok hari.
func TestAgingHariKalender(t *testing.T) {
	// 2026-09-28 adalah hari Senin; 2026-09-18 sepuluh hari sebelumnya dan melewati dua
	// akhir pekan. Perhitungan hari KERJA akan menghasilkan 8, bukan 10.
	fixed := time.Date(2026, time.September, 28, 9, 0, 0, 0, time.UTC)
	store := memory.NewStoreAt(func() time.Time { return fixed }, memory.SampleRows()...)

	page := listAll(t, store, newQuery(
		t, inboxclaimtreatynonprop.TabAdmin, false, false, callerAdmin1))

	for _, item := range page.Items {
		if item.ClaimID != "CLMNP-1001" {
			continue
		}
		if item.AgingDays != 10 {
			t.Fatalf("aging CLMNP-1001 = %d hari, ingin 10 hari kalender", item.AgingDays)
		}
		return
	}
	t.Fatal("baris CLMNP-1001 tidak ditemukan di tab Admin")
}

// Tab komite DITOLAK dengan alasannya, bukan dijawab sebagai antrean kosong.
//
// Antrean kosong dan antrean yang belum dapat dibaca menuntut tindakan yang sama sekali
// berbeda dari pengguna, dan hanya yang kedua yang punya pemilik penghalang.
func TestTabKomiteDitolakDenganAlasan(t *testing.T) {
	_, err := inboxclaimtreatynonprop.NewQuery(
		inboxclaimtreatynonprop.QueryInput{Tab: inboxclaimtreatynonprop.TabCommittee},
		callerAdmin1,
	)

	var validation *inboxclaimtreatynonprop.ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("tab komite = %v, ingin galat validasi", err)
	}
	if len(validation.Violations) != 1 ||
		validation.Violations[0].Field != inboxclaimtreatynonprop.FieldTab {
		t.Fatalf("pelanggaran = %+v, ingin satu pelanggaran pada isian tab",
			validation.Violations)
	}
	if validation.Violations[0].Message == "" {
		t.Fatal("alasan terhalang kosong; pengguna tidak akan tahu kenapa")
	}
}

// Checkbox yang tidak dikenal sebuah tab DIABAIKAN, bukan dibawa diam-diam.
//
// Tanpa ini, dua permintaan yang hasilnya pasti sama akan tampak berbeda di log dan di
// kunci cache — dan layar akan menggambar centang yang menyala tanpa mengubah apa pun.
func TestCheckboxDiabaikanPadaTabYangTidakMengenalnya(t *testing.T) {
	q := newQuery(t, inboxclaimtreatynonprop.TabTechnical, true, true, callerAdmin1)

	if q.SeeAll {
		t.Error("See All terbawa ke tab Teknik yang tidak mengenalnya")
	}
	if q.TBAOnly {
		t.Error("See TBA terbawa ke tab Teknik yang tidak mengenalnya")
	}
}

// Pemanggil tanpa identitas DITOLAK sebelum menyentuh penyimpanan.
//
// Tab Admin menyaring menurut login; tanpa identitas, penyaring itu tidak dapat dibentuk —
// dan membiarkannya berarti menampilkan antrean seluruh petugas kepada sesi yang profilnya
// tidak lengkap.
func TestPemanggilTanpaIdentitasDitolak(t *testing.T) {
	_, err := inboxclaimtreatynonprop.NewQuery(
		inboxclaimtreatynonprop.QueryInput{}, inboxclaimtreatynonprop.Caller{Login: "   "},
	)
	if !errors.Is(err, inboxclaimtreatynonprop.ErrCallerUnknown) {
		t.Fatalf("pemanggil tanpa identitas = %v, ingin ErrCallerUnknown", err)
	}
}

// Paginasi memotong tanpa mengubah jumlah seluruhnya, dan tidak pernah menghasilkan
// "halaman 1 dari 0".
func TestPaginasiMemotongTanpaMengubahTotal(t *testing.T) {
	store := memory.NewSampleStore()
	q := newQuery(t, inboxclaimtreatynonprop.TabAdmin, true, false, callerAdmin1)

	page, err := store.List(
		context.Background(), q,
		inboxclaimtreatynonprop.Pagination{Page: 2, Size: 2},
	)
	if err != nil {
		t.Fatalf("mengambil halaman kedua: %v", err)
	}

	if page.Total != 4 {
		t.Fatalf("total = %d, ingin 4 — paginasi tidak boleh mengubah jumlah seluruhnya",
			page.Total)
	}
	if got := len(page.Items); got != 2 {
		t.Fatalf("halaman kedua berisi %d baris, ingin 2", got)
	}
	if page.TotalPages() != 2 {
		t.Fatalf("total halaman = %d, ingin 2", page.TotalPages())
	}

	kosong := inboxclaimtreatynonprop.Page{
		Pagination: inboxclaimtreatynonprop.Pagination{Page: 1, Size: 25},
	}
	if kosong.TotalPages() != 1 {
		t.Fatalf("halaman kosong = %d dari total, ingin minimal 1", kosong.TotalPages())
	}
}

// Setiap tab yang TIDAK terhalang wajib punya kolom, dan setiap kolomnya wajib punya judul.
//
// Tab tanpa kolom akan menggambar tabel kosong yang terbaca sebagai "tidak ada pekerjaan";
// kolom tanpa judul akan menggambar kepala tabel yang kosong.
func TestSetiapTabTerisiPunyaKolomBerjudul(t *testing.T) {
	for _, tab := range inboxclaimtreatynonprop.Tabs() {
		if tab.Blocked {
			if len(tab.Columns) != 0 {
				t.Errorf("tab terhalang %q punya kolom; ia tidak menggambar grid", tab.Name)
			}
			if tab.BlockedOwner == "" {
				t.Errorf("tab terhalang %q tidak menyebut pemilik penghalang", tab.Name)
			}
			continue
		}

		if len(tab.Columns) == 0 {
			t.Errorf("tab %q tidak punya satu pun kolom", tab.Name)
		}
		for _, column := range tab.Columns {
			if column.Key == "" || column.Title == "" {
				t.Errorf("tab %q punya kolom tanpa kunci atau tanpa judul: %+v",
					tab.Name, column)
			}
		}
	}
}

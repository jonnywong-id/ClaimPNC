package sqlstore

import (
	"strings"
	"testing"
	"time"

	"claim-pnc/internal/platform/clock"
)

// TestCommentQuotesBalancedPerLine menjaga jebakan go-ora yang sudah menggigit dua kali.
//
// Pengurai bind go-ora TIDAK melewati komentar. Kutip yang terbuka di satu baris komentar
// dan tertutup di baris lain membuat penanda bind di antaranya terbaca sebagai literal,
// dan pernyataan yang sebenarnya sah ditolak dengan ORA-00900.
//
// Gejalanya menyesatkan dan mahal: galatnya hanya muncul saat kueri dijalankan terhadap
// Oracle, pesannya tidak menyebut baris mana pun, dan membuang satu baris komentar justru
// membuatnya bertahan. Uji ini memindahkan penemuannya dari "saat petugas menekan tombol"
// menjadi "saat berkasnya disimpan".
//
// Yang diperiksa hanya baris KOMENTAR. Kutip di dalam SQL-nya sendiri memang berpasangan
// lintas baris pada beberapa kueri, dan itu sah.
func TestCommentQuotesBalancedPerLine(t *testing.T) {
	berkas, err := queryFiles.ReadDir(".")
	if err != nil {
		t.Fatalf("membaca direktori kueri: %v", err)
	}

	diperiksa := 0
	for _, f := range berkas {
		if !strings.HasSuffix(f.Name(), ".sql") {
			continue
		}
		diperiksa++

		isi, err := queryFiles.ReadFile(f.Name())
		if err != nil {
			t.Fatalf("membaca %s: %v", f.Name(), err)
		}

		for nomor, baris := range strings.Split(string(isi), "\n") {
			teks := strings.TrimSpace(strings.TrimRight(baris, "\r"))
			if !strings.HasPrefix(teks, "--") {
				continue
			}
			for _, kutip := range []string{`"`, `'`} {
				if strings.Count(teks, kutip)%2 != 0 {
					t.Errorf("%s:%d kutip %s tidak berpasangan di dalam komentar:\n  %s\n"+
						"Kutip yang membentang antar-baris komentar membuat go-ora "+
						"salah membaca penanda bind (ORA-00900). Tutup kutipnya di baris "+
						"yang sama, atau buang tanda kutipnya.",
						f.Name(), nomor+1, kutip, teks)
				}
			}
		}
	}

	if diperiksa == 0 {
		t.Fatal("tidak ada berkas .sql yang diperiksa — pemindaiannya tidak bekerja")
	}
	t.Logf("%d berkas kueri diperiksa", diperiksa)
}

// TestCalendarDatesBindAsWIB menjaga cacat R-12 yang terbukti terhadap Oracle.
//
// Kolom `DATEOFLOSS`, `REPORTDATE`, `RECEIVEDATE`, `POLIS_MULAI`, dan `POLIS_AKHIR`
// bertipe `DATE` — Oracle menyimpan JAM DINDING apa adanya, tanpa zona. Mengikat waktu UTC
// karena itu menuliskan jam dinding UTC, dan driver memasang zona sesi (WIB) saat
// membacanya kembali. Tanggal kejadian 15 Juli tersimpan `2026-07-14 17:00` lalu terbaca
// sebagai **14 Juli**.
//
// Sehari hilang tanpa galat, pada SETIAP tanggal kalender — karena semuanya bertengah
// malam WIB, dan tengah malam WIB selalu jatuh di hari sebelumnya menurut UTC.
//
// Uji ini merah bila pengikatnya dikembalikan ke `t.UTC()`, dengan pesan yang menyebut
// tanggal yang hilang — bukan sekadar "tidak sama".
func TestCalendarDatesBindAsWIB(t *testing.T) {
	// Tengah malam WIB: jam paling rentan, dan jam yang dipakai SELURUH tanggal kalender.
	tengahMalamWIB := time.Date(2026, time.July, 15, 0, 0, 0, 0, clock.ZoneWIB)

	terikat, ok := calendarDateOrNil(tengahMalamWIB).(time.Time)
	if !ok {
		t.Fatalf("calendarDateOrNil mengembalikan %T, bukan time.Time", calendarDateOrNil(tengahMalamWIB))
	}

	// Oracle DATE menyimpan jam dindingnya; itulah yang akan terbaca kembali sebagai WIB.
	tersimpan := terikat.Format("2006-01-02 15:04")
	const diharapkan = "2026-07-15 00:00"
	if tersimpan != diharapkan {
		t.Fatalf("kolom DATE akan menyimpan jam dinding %q, bukan %q;\n"+
			"tanggal bisnisnya %s akan terbaca kembali sebagai %s — sehari hilang (R-12)",
			tersimpan, diharapkan,
			clock.DateWIB(tengahMalamWIB).Format("2006-01-02"),
			terikat.Format("2006-01-02"))
	}

	// Instan yang sama, ditulis dalam zona lain, harus menghasilkan simpanan yang sama:
	// yang menentukan tanggal bisnis adalah WIB, bukan zona asal nilainya.
	samaTapiUTC := tengahMalamWIB.UTC()
	lain, _ := calendarDateOrNil(samaTapiUTC).(time.Time)
	if lain.Format("2006-01-02 15:04") != diharapkan {
		t.Fatalf("instan yang sama dalam UTC menghasilkan %q, bukan %q; "+
			"pengikat masih bergantung pada zona nilai masukannya",
			lain.Format("2006-01-02 15:04"), diharapkan)
	}

	if calendarDateOrNil(time.Time{}) != nil {
		t.Fatal("tanggal kosong harus menjadi NULL, bukan tahun 1")
	}
}

// TestInstantsStayUTC menjaga sisi lain pembedaannya.
//
// Kolom `TIMESTAMP(6)` — `DIBUAT_PADA`, `DIUBAH_PADA`, `DIHAPUS_PADA` — menyimpan INSTAN,
// bukan tanggal kalender, dan tetap UTC sesuai `F-5`. Menyeragamkan keduanya ke WIB akan
// menggeser seluruh jejak waktu tujuh jam.
func TestInstantsStayUTC(t *testing.T) {
	saat := time.Date(2026, time.July, 15, 0, 0, 0, 0, clock.ZoneWIB)

	terikat, _ := timeOrNil(saat).(time.Time)
	if _, offset := terikat.Zone(); offset != 0 {
		t.Fatalf("instan terikat dengan offset %d detik, bukan UTC; "+
			"`F-5` menetapkan seluruh waktu disimpan UTC", offset)
	}
}

// droppedFromClaimTable adalah kolom yang dibuang Work Owner dari POOLDATA.T_CLAIM_PNC
// pada 2026-09-26 15:31 (97 menjadi 82 kolom).
//
// Menyebut satu saja di kueri kepala klaim menggagalkan SETIAP pembuatan klaim dengan
// ORA-00904 — itulah yang terjadi sesudah perubahan itu, dan PNCN baru berhenti masuk ke
// tabelnya. Medan yang kolomnya hilang dipulihkan restoreDropped, bukan ditulis kembali.
var droppedFromClaimTable = []string{
	"TAHAP_KINI", "NILAI_ESTIMASI_SEN", "PELAPOR_EMAIL",
	"POLIS_MULAI", "POLIS_AKHIR", "POLIS_DEKLARASI", "POLIS_PENJAMIN_KREDIT",
	"FLAG_KLAIM", "STATUS_POSISI_PROGRES", "TRANSFER_COMPLIANCE", "MINTA_KEMBALI",
	"DIUBAH_OLEH", "DIUBAH_PADA",
}

func TestClaimHeaderQueriesUseOnlyExistingColumns(t *testing.T) {
	for _, name := range []string{"klaim_perbarui", "klaim_sisip", "klaim_ambil", "klaim_ambil_per_nomor"} {
		body := stripComments(loadQuery(name))
		for _, col := range droppedFromClaimTable {
			if strings.Contains(body, col) {
				t.Errorf("kueri %q menyebut %s, kolom yang sudah dibuang dari T_CLAIM_PNC", name, col)
			}
		}
		// DIHAPUS_PADA masih ada di tabel anak, jadi yang dilarang hanya pada kepala klaim.
		if strings.Contains(body, "DIHAPUS_PADA") {
			t.Errorf("kueri %q menyebut DIHAPUS_PADA; kepala klaim tidak lagi punya penanda hapus", name)
		}
	}
	if strings.Contains(stripComments(loadQuery("klaim_cari_ganda")), "k.DIHAPUS_PADA") {
		t.Error("pemeriksaan klaim ganda menyaring k.DIHAPUS_PADA, kolom yang sudah dibuang")
	}
}

func stripComments(sql string) string {
	var out []string
	for _, line := range strings.Split(sql, "\n") {
		if !strings.HasPrefix(strings.TrimSpace(line), "--") {
			out = append(out, line)
		}
	}
	return strings.Join(out, "\n")
}

// Persentase pada dokumen polis bertipe teks. Share koasuransi Sinar Mas dibaca darinya
// dan ditulis ke SHAREASM, kolom yang dibaca laporan lama — salah satu desimal di sini
// adalah salah bagian uang pada setiap klaim berkoasuransi.
func TestParsePercentReadsPolicyShares(t *testing.T) {
	cases := []struct {
		raw   string
		want  int64
		known bool
	}{
		{"100", 1_000_000, true},
		{"51", 510_000, true},
		{"2.88", 28_800, true},
		{"33.3333", 333_333, true},
		{" 60 ", 600_000, true},
		{"", 0, false},
		{"bukan angka", 0, false},
	}
	for _, c := range cases {
		got, known := parsePercent(c.raw)
		if int64(got) != c.want || known != c.known {
			t.Errorf("parsePercent(%q) = %d, %v; seharusnya %d, %v", c.raw, got, known, c.want, c.known)
		}
	}
}

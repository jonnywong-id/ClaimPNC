package inboxlaporanklaimhttp

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"claim-pnc/internal/inboxlaporanklaim"
)

// Tabel penyimpanan yang belum dibuat harus sampai ke layar sebagai penjelasan, bukan
// sebagai "terjadi kesalahan pada sistem".
//
// Tanpa pemetaan ini, galat Oracle jatuh ke penulis galat bersama yang menjawab 500
// dengan kalimat umum — dan kalimat itu mengirim orang mencari cacat di aplikasi,
// padahal yang kurang ada di basis data dan perbaikannya satu perintah DBA.
func TestTabelBelumDibuatDipetakanDenganPerbaikannya(t *testing.T) {
	galat := fmt.Errorf("menyisipkan: %w", inboxlaporanklaim.ErrStorageNotReady)

	status, body, dikenali := mapError(galat)

	if !dikenali {
		t.Fatal("galat tabel-belum-dibuat harus dikenali modul ini, bukan jatuh ke galat umum")
	}
	// 503, bukan 500: aplikasinya sehat dan keadaannya berakhir begitu migrasi dijalankan.
	if status != http.StatusServiceUnavailable {
		t.Errorf("status = %d, seharusnya %d", status, http.StatusServiceUnavailable)
	}
	if body.Code != CodePenyimpananBelumSiap {
		t.Errorf("kode = %q, seharusnya %q", body.Code, CodePenyimpananBelumSiap)
	}

	// Pesannya wajib menyebut LANGKAH PERBAIKANNYA. Pesan yang hanya menyatakan
	// kegagalan mengembalikan kita ke keadaan semula.
	for _, wajib := range []string{"migrasi", "CPNC_LAPORAN_KLAIM"} {
		if !strings.Contains(body.Message, wajib) {
			t.Errorf("pesan tidak menyebut %q; pesan: %s", wajib, body.Message)
		}
	}
}

// Ketiga keadaan yang menutup layar harus punya kode yang BERBEDA.
//
// Ketiganya berakhir sama bagi pengguna — layar tidak menampilkan apa yang diminta —
// tetapi perbaikannya ditangani tiga pihak yang berbeda: data pegawai, infrastruktur,
// dan DBA. Satu kode untuk lebih dari satu keadaan akan mengirim pekerjaan ke pihak
// yang salah.
func TestKeadaanYangMenutupLayarPunyaKodeBerbeda(t *testing.T) {
	terlihat := map[string]string{}
	for nama, galat := range map[string]error{
		"cabang tidak dikenali": inboxlaporanklaim.ErrBranchUnknown,
		"sumber cabang mati":    inboxlaporanklaim.ErrBranchUnreadable,
		"tabel belum dibuat":    inboxlaporanklaim.ErrStorageNotReady,
	} {
		_, body, dikenali := mapError(galat)
		if !dikenali {
			t.Fatalf("%s: seharusnya dikenali", nama)
		}
		if sebelumnya, ada := terlihat[body.Code]; ada {
			t.Errorf("%s memakai kode %q yang sudah dipakai %s", nama, body.Code, sebelumnya)
		}
		terlihat[body.Code] = nama
	}
}

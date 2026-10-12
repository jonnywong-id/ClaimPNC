// Package account memenuhi seam mastersurveyors.AccountRegistrar.
//
// # Kenapa pengisi seam di tahap ini MENCATAT, bukan membuat akun
//
// Sistem lama membuat instans `Data-Admin-Operator-ID` lewat `GCNMCreateOperator` — yaitu
// baris pada tabel operator milik ENGINE PEGA. Dua hal melarang modul master menulisnya:
//
//   - `P-1` (`D-21`): satu tabel hanya boleh ditulis satu sistem selama masa paralel, dan
//     tabel operator dimiliki Pega sampai `F-3` memindahkannya. Pelanggarannya tidak
//     menimbulkan pesan galat apa pun — ia baru terlihat sebagai data rusak.
//   - `F-3 Identitas & Akses` adalah pemilik identitas di sistem baru, dan ia berstatus
//     terhalang: kontrak HCC/HCQ nol jejak di export (`R-14`, `ADR-0024`).
//
// Karena itu modul master menyatakan KEHENDAKNYA — dengan kedelapan parameter Pega apa
// adanya — dan pengisi seam inilah yang memutuskan apa yang terjadi kemudian. Di tahap
// ini: permintaannya dicatat, tidak ada akun yang dibuat.
//
// Keputusan Work Owner 2026-09-19: perilaku Pega dibawa, tabel Pega tidak ditulis.
//
// # Apa yang HILANG karenanya, dinyatakan supaya tidak ditemukan sebagai kejutan
//
// Surveyor internal yang ditambahkan lewat layar baru BELUM dapat masuk ke aplikasi
// sampai `F-3` siap dan adapternya diganti. Nama loginnya tersimpan, kewajibannya
// ditegakkan, bentroknya diuji — hanya akunnya yang belum terbit.
//
// Lapisan Adapter — memenuhi interface yang dideklarasikan Domain.
package account

import (
	"context"
	"log/slog"
	"strings"
	"sync"

	"claim-pnc/internal/mastersurveyors"
)

// Recorder mencatat setiap permintaan pembuatan akun tanpa membuatnya.
//
// Ia aman dipakai beberapa goroutine sekaligus — daftar permintaannya dijaga mutex,
// karena satu instans dipakai bersama seluruh permintaan HTTP.
type Recorder struct {
	log *slog.Logger

	mu       sync.Mutex
	recorded []Recorded
}

// Recorded adalah satu permintaan yang tercatat.
type Recorded struct {
	PortalAlias string
	Request     mastersurveyors.AccountRequest
}

// NewRecorder membuat pengisi seam yang mencatat permintaan ke log.
//
// Logger boleh nil; bila nil dipakai slog.Default() supaya pemanggil tidak perlu
// menyiapkan apa pun hanya untuk menjalankan modul.
func NewRecorder(log *slog.Logger) *Recorder {
	if log == nil {
		log = slog.Default()
	}
	return &Recorder{log: log}
}

// Register mencatat permintaan pembuatan akun.
//
// # Yang SENGAJA tidak dicatat
//
// AccountRequest.Password TIDAK PERNAH ikut ke log. `docs/Steering/12-CROSSCUTTING.md`
// §2.4 melarang sandi masuk log tanpa perkecualian, dan sandi di sini dapat ditebak dari
// nama login — mencatatnya berarti menyebarkan sesuatu yang memang mudah ditebak ke
// tempat yang retensinya lebih longgar daripada basis data.
//
// Permintaannya sendiri tetap disimpan utuh di memori, termasuk sandinya, supaya adapter
// sungguhan kelak dapat memakai bahan yang sama tanpa membentuk ulang.
func (r *Recorder) Register(ctx context.Context, portalAlias string, req mastersurveyors.AccountRequest) error {
	r.mu.Lock()
	r.recorded = append(r.recorded, Recorded{PortalAlias: portalAlias, Request: req})
	r.mu.Unlock()

	r.log.InfoContext(ctx, "permintaan akun surveyor dicatat, akun BELUM dibuat",
		slog.String("portal", portalAlias),
		slog.String("login", req.UserID),
		slog.String("nama", req.UserName),
		slog.String("unit", req.Unit),
		slog.String("access_group", req.AccessGroup),
		slog.Bool("wajib_ganti_sandi", req.MustChangePassword),
		slog.String("alasan", "F-3 Identitas & Akses belum siap; P-1 melarang menulis tabel operator Pega"),
	)
	return nil
}

// Recorded mengembalikan salinan seluruh permintaan yang tercatat.
//
// Dipakai pengujian dan mode periksa. Ia salinan, bukan rujukan ke daftar aslinya —
// pemanggil tidak dapat mengubah keadaan Recorder lewat nilai yang dikembalikan.
func (r *Recorder) Recorded() []Recorded {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]Recorded, len(r.recorded))
	copy(out, r.recorded)
	return out
}

// Taken menyatakan nama login sudah tercatat pada permintaan sebelumnya.
//
// Pemeriksaan bentrok yang SEBENARNYA ada pada pengisi seam yang mengetahui seluruh
// identitas sistem — yaitu adapter `F-3` kelak. Yang ini hanya menjaga agar satu proses
// yang sedang berjalan tidak mencatat dua permintaan untuk login yang sama, dan tidak
// boleh dibaca sebagai jaminan keunikan.
func (r *Recorder) Taken(login string) bool {
	key := strings.ToUpper(strings.TrimSpace(login))
	if key == "" {
		return false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, item := range r.recorded {
		if strings.ToUpper(strings.TrimSpace(item.Request.UserID)) == key {
			return true
		}
	}
	return false
}

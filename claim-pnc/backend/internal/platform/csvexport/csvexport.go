// Package csvexport memuat kerangka unduhan CSV yang dipakai tombol ekspor di banyak layar:
// header unduhan, penulisan berpotong yang mengalirkan isi tanpa menumpuk memori, penanda
// berkas yang terpotong, dan pencatatan kegagalan setelah header terkirim.
//
// Yang tetap milik modul adalah ISI berkasnya — kolom, urutan, nama berkas, batas baris, dan
// kalimat penandanya. Yang tinggal di sini hanyalah mekanismenya, yang sebelumnya tersalin
// sama persis di tujuh paket.
package csvexport

import (
	"encoding/csv"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"claim-pnc/internal/platform/logging"
)

// BeginDownload memasang header unduhan.
//
// `no-store` bukan kehati-hatian berlebih: berkas ekspor memuat data nasabah dan nilai uang,
// dan ia tidak boleh mengendap di cache perantara mana pun.
func BeginDownload(w http.ResponseWriter, filename string) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	w.Header().Set("Cache-Control", "no-store")
}

// Flush mendorong isi yang sudah tertulis keluar, bukan menahannya sampai akhir. Itulah yang
// membuat unduhan besar mulai mengalir segera dan memori tidak menumpuk. Galat penulisan
// yang tertunda dikembalikan.
func Flush(w http.ResponseWriter, writer *csv.Writer) error {
	writer.Flush()
	if err := writer.Error(); err != nil {
		return err
	}
	if flusher, able := w.(http.Flusher); able {
		flusher.Flush()
	}
	return nil
}

// TruncationNotice menyusun baris penanda bahwa berkasnya tidak lengkap: limit baris dari
// total yang cocok. hint adalah anjuran tambahan modul, misalnya " Persempit pencariannya.",
// atau kosong.
func TruncationNotice(width, limit, total int, hint string) []string {
	notice := make([]string, width)
	if width == 0 {
		return notice
	}
	notice[0] = fmt.Sprintf("-- Terpotong pada %s baris dari %s yang cocok.%s --",
		strconv.Itoa(limit), strconv.Itoa(total), hint)
	return notice
}

// LogFailure mencatat kegagalan yang terjadi SETELAH header terkirim.
//
// Ia tidak dapat lagi dijawab sebagai galat HTTP — status sudah 200 dan sebagian berkas sudah
// sampai ke pengguna. Yang dapat dilakukan hanyalah menghentikan penulisan dan meninggalkan
// jejak, supaya unduhan yang terpotong punya pasangan keterangan di sisi peladen.
func LogFailure(r *http.Request, logger *slog.Logger, message string, err error) {
	logging.From(r.Context(), logger).Error(message,
		slog.String("jalur", r.URL.Path),
		slog.String("galat", err.Error()),
	)
}

// Paged menulis berkas CSV yang isinya dibaca berpotong.
type Paged[I any] struct {
	// Header adalah baris judul; lebarnya juga lebar baris penanda.
	Header []string
	// Limit adalah banyaknya baris maksimum pada satu berkas.
	Limit int
	// Notice menyusun baris penanda terpotong dari lebar dan total baris yang cocok.
	Notice func(width, total int) []string
	// Row menyusun satu baris berkas.
	Row func(item I) []string
	// Next membaca potongan berikutnya; page dimulai dari 2.
	Next func(page int) (items []I, total int, err error)
	// Fail mencatat kegagalan setelah header terkirim.
	Fail func(err error)
}

// Write menulis judul, lalu potongan pertama (items dari total yang cocok), lalu potongan
// berikutnya sampai seluruh baris tertulis, potongannya habis, atau Limit tercapai — yang
// terakhir ditandai satu baris penanda.
func (p Paged[I]) Write(w http.ResponseWriter, items []I, total int) {
	writer := csv.NewWriter(w)
	defer writer.Flush()

	if err := writer.Write(p.Header); err != nil {
		p.Fail(err)
		return
	}

	written := 0
	for page := 2; ; page++ {
		for _, item := range items {
			if written >= p.Limit {
				_ = writer.Write(p.Notice(len(p.Header), total))
				return
			}
			if err := writer.Write(p.Row(item)); err != nil {
				p.Fail(err)
				return
			}
			written++
		}

		if err := Flush(w, writer); err != nil {
			p.Fail(err)
			return
		}
		if written >= total || len(items) == 0 {
			return
		}

		next, nextTotal, err := p.Next(page)
		if err != nil {
			p.Fail(err)
			return
		}
		items, total = next, nextTotal
	}
}

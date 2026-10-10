// Package schedule menjalankan pekerjaan terjadwal harian di dalam binary aplikasi — padanan
// Agent dan Job Scheduler Pega (`S-6`).
//
// Sengaja kecil: satu goroutine per pekerjaan yang tidur sampai jam berikutnya, lalu
// menjalankan pekerjaannya. Tidak ada pustaka cron: jadwal Pega yang dibawa seluruhnya
// berbentuk "setiap hari pukul HH:MM".
//
// Package ini TIDAK menjamin pekerjaan hanya berjalan di satu instans. Jaminan itu menjadi
// tanggung jawab pekerjaannya sendiri — misalnya dengan mengunci baris yang ia proses —
// karena hanya pekerjaannya yang tahu apa artinya "sudah dikerjakan".
package schedule

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Clock adalah jam yang dipakai menentukan waktu tidur — diganti pada pengujian.
type Clock func() time.Time

// TimeOfDay adalah jam dan menit dalam sehari.
type TimeOfDay struct{ Hour, Minute int }

func (t TimeOfDay) String() string { return fmt.Sprintf("%02d:%02d", t.Hour, t.Minute) }

// ParseTimes membaca daftar jam "HH:MM" dipisah koma, mis. "08:15" atau "08:15,13:00".
func ParseTimes(text string) ([]TimeOfDay, error) {
	var out []TimeOfDay
	for _, part := range strings.Split(text, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		hh, mm, ok := strings.Cut(part, ":")
		h, errH := strconv.Atoi(hh)
		m, errM := strconv.Atoi(mm)
		if !ok || errH != nil || errM != nil || h < 0 || h > 23 || m < 0 || m > 59 {
			return nil, fmt.Errorf("jam %q bukan bentuk HH:MM", part)
		}
		out = append(out, TimeOfDay{Hour: h, Minute: m})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("tidak ada jam yang diisi")
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Hour*60+out[i].Minute < out[j].Hour*60+out[j].Minute
	})
	return out, nil
}

// Next mengembalikan saat jalan berikutnya yang LEBIH DARI now, pada zona loc.
func Next(now time.Time, times []TimeOfDay, loc *time.Location) time.Time {
	local := now.In(loc)
	for day := 0; day < 2; day++ {
		base := local.AddDate(0, 0, day)
		for _, t := range times {
			at := time.Date(base.Year(), base.Month(), base.Day(), t.Hour, t.Minute, 0, 0, loc)
			if at.After(local) {
				return at
			}
		}
	}
	return local // tidak tercapai selama times tidak kosong
}

// Daily menjalankan run setiap hari pada jam-jam itu sampai ctx dibatalkan. Ia memblokir;
// panggil di goroutine sendiri. Kegagalan satu putaran dicatat dan tidak menghentikan
// putaran berikutnya.
func Daily(ctx context.Context, logger *slog.Logger, name string, times []TimeOfDay, loc *time.Location,
	now Clock, run func(context.Context) error) {
	for {
		at := Next(now(), times, loc)
		logger.Info("pekerjaan terjadwal menunggu", slog.String("pekerjaan", name), slog.Time("berikutnya", at))
		timer := time.NewTimer(time.Until(at))
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
		started := now()
		if err := run(ctx); err != nil {
			logger.Error("pekerjaan terjadwal gagal", slog.String("pekerjaan", name), slog.String("galat", err.Error()))
			continue
		}
		logger.Info("pekerjaan terjadwal selesai", slog.String("pekerjaan", name),
			slog.Duration("lama", now().Sub(started)))
	}
}

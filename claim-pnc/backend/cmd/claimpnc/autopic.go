package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/platform/config"
	"claim-pnc/internal/platform/schedule"
	registrasiusecase "claim-pnc/internal/registrasi/usecase"
)

// startAutoPIC menyalakan agent PIC Teknik otomatis (AutoPICAgent) bila
// AGEN_PIC_OTOMATIS_AKTIF=true. Bawaannya MATI.
//
// Agent memproses klaim dengan mengunci tugasnya satu per satu, sehingga aman bila
// menyala di kedua instans; tetapi cukup dinyalakan di SATU instans supaya log tidak ganda.
// Jam yang tidak sah menggagalkan start: agent yang diniatkan menyala tetapi diam-diam
// tidak pernah berjalan lebih buruk daripada aplikasi yang menolak start.
func startAutoPIC(ctx context.Context, cfg config.AutoPIC, service *registrasiusecase.Service, logger *slog.Logger) error {
	if !cfg.Enabled {
		logger.Info("agent PIC Teknik otomatis mati — nyalakan dengan AGEN_PIC_OTOMATIS_AKTIF=true")
		return nil
	}
	times, err := schedule.ParseTimes(cfg.Times)
	if err != nil {
		return fmt.Errorf("AGEN_PIC_OTOMATIS_JAM: %w", err)
	}
	if service == nil {
		return fmt.Errorf("AGEN_PIC_OTOMATIS_AKTIF=true, tetapi modul registrasi tidak terpasang (basis data tidak tersedia)")
	}
	go schedule.Daily(ctx, logger, registrasiusecase.AutoPICAgent, times, clock.ZoneWIB, time.Now,
		func(ctx context.Context) error { return runAutoPIC(ctx, service, logger) })
	return nil
}

// runAutoPIC menjalankan satu putaran agent dan mencatat hasilnya.
func runAutoPIC(ctx context.Context, service *registrasiusecase.Service, logger *slog.Logger) error {
	result, err := service.AssignUnassignedTechnicalPIC(ctx)
	if err != nil {
		return err
	}
	for _, f := range result.Failures {
		logger.Error("agent PIC Teknik otomatis gagal pada satu klaim",
			slog.String("nomor_klaim", f.ClaimNumber), slog.String("galat", f.Err.Error()))
	}
	logger.Info("agent PIC Teknik otomatis",
		slog.Int("diperiksa", result.Examined), slog.Int("ber_pic", result.Assigned),
		slog.Int("tugas_dipindah", result.Moved), slog.Int("dilewati", result.Skipped),
		slog.Int("gagal", len(result.Failures)))
	return nil
}

package usecase

import (
	"context"
	"log/slog"
	"time"

	"claim-pnc/internal/inboxmanager"
)

// DetailExport mengalirkan berkas "Export Data Detail Klaim" ke emit.
//
// Validasi dan kewenangan diperiksa SEBELUM satu baris pun dibaca, sehingga penolakan selalu
// tiba sebagai galat — bukan sebagai berkas yang berhenti di tengah.
func (s *Service) DetailExport(
	ctx context.Context,
	portalAlias string,
	input inboxmanager.DetailExportInput,
	caller inboxmanager.Caller,
	emit func(inboxmanager.DetailExportRow) error,
) error {
	resolved, err := s.resolveCaller(ctx, portalAlias, caller)
	if err != nil {
		return err
	}

	query, err := inboxmanager.NewDetailExportQuery(input, resolved, portalAlias)
	if err != nil {
		return err
	}

	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return err
	}

	s.logDetailExport(query, portalAlias)
	return repo.DetailExport(ctx, query, emit)
}

// logDetailExport mencatat setiap ekspor detail klaim.
//
// Berkasnya memuat nama tertanggung dan nilai klaim seluruh petugas pada entitas itu. Sama
// dengan pembukaan tab (logOpen), jejaknya satu-satunya kontrol pengimbang selama pemeriksaan
// peran belum ada (`D-59`).
func (s *Service) logDetailExport(q inboxmanager.DetailExportQuery, portalAlias string) {
	if s.logger == nil {
		return
	}
	attrs := []any{
		slog.String("modul", "inbox-manager"),
		slog.String("portal", portalAlias),
		slog.String("login", q.Caller.Login),
		slog.String("lini_bisnis", q.LineBusiness),
		slog.Bool("seluruh_lini", q.AllLines),
	}
	if q.HasPeriod() {
		attrs = append(attrs,
			slog.String("dari", q.From.Format(time.DateOnly)),
			slog.String("sampai", q.Until.AddDate(0, 0, -1).Format(time.DateOnly)),
		)
	}
	s.logger.InfoContext(context.Background(), "export data detail klaim", attrs...)
}

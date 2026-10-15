// Package usecase mengorkestrasi modul Inbox Banding Harga Salvage.
//
// Tiga operasi, seluruhnya membaca:
//
//	Metadata  menyerahkan bentuk layar — tab, kolom, selisih terencana, keterbatasan
//	List      mengambil isi satu tab
//	Summary   menghitung tabel ringkas "Status Salvage / Jumlah"
//
// Tidak ada operasi yang menulis, dan ketiadaannya disengaja — alasannya ada di komentar seam
// inboxbandinghargasalvage.Repo: `Section/ButtonApproveRejectedRequest` tidak ada di export,
// sehingga kolom mana yang ditulis tombol Approve/Reject tidak diketahui.
package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"claim-pnc/internal/inboxbandinghargasalvage"
)

// Service melayani modul Inbox Banding Harga Salvage.
type Service struct {
	repoSelector   inboxbandinghargasalvage.RepoSelector
	writerSelector inboxbandinghargasalvage.WriterSelector
	docSelector    inboxbandinghargasalvage.DocumentReaderSelector
	logger         *slog.Logger
}

// Options adalah bahan pembentuk Service.
type Options struct {
	RepoSelector inboxbandinghargasalvage.RepoSelector

	// WriterSelector memilih penulis keputusan milik satu portal.
	//
	// Ia BOLEH nil, dan itu bukan kelalaian: perakit yang belum menyiapkan jalur tulis
	// mendapat layar yang tetap berjalan penuh untuk membaca, dan tombol keputusannya
	// menjawab dengan alasan alih-alih menggantung.
	WriterSelector inboxbandinghargasalvage.WriterSelector

	// DocumentSelector memilih pembaca dokumen banding milik satu portal.
	//
	// Ia BOLEH nil, dengan alasan yang sama seperti WriterSelector: layar tetap berjalan
	// penuh tanpanya, dan tombol "Lihat File" menjawab dengan alasan.
	DocumentSelector inboxbandinghargasalvage.DocumentReaderSelector

	// Logger dipakai mencatat pembukaan antrean yang diwakilkan — yakni saat seorang
	// petugas melihat antrean komite LAIN karena aturan yang ditiru dari Pega.
	//
	// Ia boleh nil; bila nil, catatannya tidak ditulis dan tidak ada yang gagal karenanya.
	Logger *slog.Logger
}

// NewService membentuk layanan modul Inbox Banding Harga Salvage.
func NewService(o Options) (*Service, error) {
	if o.RepoSelector == nil {
		return nil, errors.New("inboxbandinghargasalvage/usecase: RepoSelector wajib diisi")
	}
	return &Service{
		repoSelector:   o.RepoSelector,
		writerSelector: o.WriterSelector,
		docSelector:    o.DocumentSelector,
		logger:         o.Logger,
	}, nil
}

// Metadata adalah keterangan layar yang tidak bergantung isi antrean.
type Metadata struct {
	// Tabs adalah kedua tab beserta kolomnya.
	Tabs []inboxbandinghargasalvage.Tab

	// DefaultTab adalah tab yang terbuka pertama kali.
	DefaultTab string

	// SearchLabel dan SearchPlaceholder adalah teks kotak pencarian, diambil dari
	// `pyCaption` dan `pyActionPrompt` pada section aslinya.
	SearchLabel       string
	SearchPlaceholder string

	// DecisionColumns adalah kolom panel rincian pada grid History Cheker.
	//
	// Ia dikirim bersama bentuk layar, bukan bersama isinya, karena susunannya tetap —
	// sama seperti kolom kedua grid.
	DecisionColumns []inboxbandinghargasalvage.Column

	// PlannedDifferences menyatakan hal yang SENGAJA berbeda dari layar lama.
	PlannedDifferences []string

	// Limitations menyatakan hal yang belum berjalan penuh beserta alasannya.
	Limitations []string
}

// Metadata menyerahkan keterangan layar.
//
// Ia tidak menyentuh basis data sama sekali dan tidak bergantung portal: daftar tab dan
// kolomnya sama di seluruh entitas, karena ia bentuk layar — bukan data entitas.
func (s *Service) Metadata() Metadata {
	return Metadata{
		Tabs:               inboxbandinghargasalvage.Tabs(),
		DefaultTab:         inboxbandinghargasalvage.DefaultTab,
		SearchLabel:        inboxbandinghargasalvage.SearchLabel,
		SearchPlaceholder:  inboxbandinghargasalvage.SearchPlaceholder,
		DecisionColumns:    inboxbandinghargasalvage.DecisionColumns(),
		PlannedDifferences: inboxbandinghargasalvage.PlannedDifferences(),
		Limitations:        inboxbandinghargasalvage.Limitations(),
	}
}

// Listed adalah isi satu tab beserta permintaan yang benar-benar dipakai.
type Listed struct {
	// Page adalah satu halaman baris beserta jumlah seluruh baris yang cocok.
	Page inboxbandinghargasalvage.Page

	// Query adalah permintaan setelah divalidasi.
	//
	// Layar menggambar keadaan penyaringnya dari sini, bukan dari isian yang ia kirim —
	// sehingga kotak cari selalu memperlihatkan kata kunci yang BENAR-BENAR dipakai, dan
	// keterangan "Anda melihat antrean orang lain" terbaca dari Reviewer di dalamnya.
	Query inboxbandinghargasalvage.Query
}

// List mengambil isi satu tab.
func (s *Service) List(
	ctx context.Context,
	portalAlias string,
	caller inboxbandinghargasalvage.Caller,
	input inboxbandinghargasalvage.QueryInput,
	page inboxbandinghargasalvage.Pagination,
) (Listed, error) {
	query, err := inboxbandinghargasalvage.NewQuery(input, caller)
	if err != nil {
		return Listed{}, err
	}

	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return Listed{}, err
	}

	result, err := repo.List(ctx, query, page)
	if err != nil {
		return Listed{}, fmt.Errorf("mengambil isi tab %s: %w", query.Tab.Code, err)
	}

	s.logDelegation(portalAlias, caller, query)

	return Listed{Page: result, Query: query}, nil
}

// Summary menghitung tabel ringkas "Status Salvage / Jumlah".
//
// # Kenapa ia menghitung ULANG, bukan memakai angka dari List
//
// Karena tabel ringkas menampilkan KEDUA tab sekaligus, sementara List hanya mengambil tab
// yang sedang terbuka. Mengambilnya dari List berarti satu barisnya selalu kosong — atau
// membuat layar menjalankan daftar kedua tab hanya demi satu angka.
//
// Penyaring yang dipakainya sama persis dengan penyaring daftar masing-masing tab. Itulah
// yang membuat angkanya cocok dengan gridnya, dan itu selisih terencana nomor 4: di layar
// lama keduanya menghitung populasi yang berbeda.
func (s *Service) Summary(
	ctx context.Context,
	portalAlias string,
	caller inboxbandinghargasalvage.Caller,
) (inboxbandinghargasalvage.Summary, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return inboxbandinghargasalvage.Summary{}, err
	}

	tabs := inboxbandinghargasalvage.Tabs()
	rows := make([]inboxbandinghargasalvage.SummaryRow, 0, len(tabs))

	for _, tab := range tabs {
		query, err := inboxbandinghargasalvage.NewQuery(
			inboxbandinghargasalvage.QueryInput{Tab: tab.Code}, caller)
		if err != nil {
			return inboxbandinghargasalvage.Summary{}, err
		}

		count, err := repo.Count(ctx, query)
		if err != nil {
			return inboxbandinghargasalvage.Summary{},
				fmt.Errorf("menghitung tab %s: %w", tab.Code, err)
		}

		rows = append(rows, inboxbandinghargasalvage.SummaryRow{
			Label: tab.Name,
			Tab:   tab.Code,
			Count: count,
		})
	}

	return inboxbandinghargasalvage.Summary{Rows: rows}, nil
}

// logDelegation mencatat pembukaan antrean yang BUKAN milik pemanggil sendiri.
//
// # Kenapa ini layak dicatat, sementara pembukaan biasa tidak
//
// Karena ia satu-satunya keadaan di layar ini yang membuat seseorang melihat angka uang yang
// sedang diputuskan orang lain — dan aturannya tertanam sebagai nama orang di dalam kode,
// bukan berasal dari master peran (`D-15`, ditiru atas keputusan Work Owner).
//
// Sampai `F-4` menggantikannya, jejak di log inilah satu-satunya hal yang menyatakan siapa
// benar-benar memakainya, dan itu yang menjadi dasar memutuskan kapan aturan itu dicabut.
func (s *Service) logDelegation(
	portalAlias string,
	caller inboxbandinghargasalvage.Caller,
	query inboxbandinghargasalvage.Query,
) {
	if s.logger == nil || !query.Reviewer.Delegated {
		return
	}

	s.logger.Info(
		"antrean banding harga dibuka atas nama komite lain",
		slog.String("modul", "inbox-banding-harga-salvage"),
		slog.String("portal", portalAlias),
		slog.String("pemanggil", caller.Clean().Login),
		slog.String("antrean_milik", query.Reviewer.Name),
		slog.String("sebab",
			"aturan bernama orang di Activity/SetReqSalvage_Act langkah 8, "+
				"ditiru apa adanya atas keputusan Work Owner (D-15 menunggu F-4)"),
	)
}

// Decided adalah isi panel rincian satu klaim beserta permintaan yang dipakai.
type Decided struct {
	// Items adalah keputusan banding harga klaim itu, terbaru di atas.
	//
	// Kosong berarti klaim itu belum punya keputusan apa pun — keadaan yang sah, bukan
	// galat. Lihat seam Repo.ListDecisions.
	Items []inboxbandinghargasalvage.Decision

	// Query adalah permintaan setelah divalidasi. Layar membaca nomor klaim dan antreannya
	// dari sini, bukan dari isian yang ia kirim.
	Query inboxbandinghargasalvage.DecisionQuery
}

// Decisions mengambil isi panel rincian pada grid History Cheker.
//
// # Kenapa ia permintaan TERSENDIRI, bukan ikut di baris daftarnya
//
// Karena satu klaim dapat punya beberapa barang yang dibanding, sehingga panel ini adalah
// DAFTAR — bukan perluasan satu baris. Membawanya serta pada setiap baris grid berarti
// menjalankan kuerinya sebanyak baris yang tampil, untuk panel yang hampir selalu tertutup.
func (s *Service) Decisions(
	ctx context.Context,
	portalAlias string,
	caller inboxbandinghargasalvage.Caller,
	claimNo string,
) (Decided, error) {
	query, err := inboxbandinghargasalvage.NewDecisionQuery(claimNo, caller)
	if err != nil {
		return Decided{}, err
	}

	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return Decided{}, err
	}

	items, err := repo.ListDecisions(ctx, query)
	if err != nil {
		return Decided{}, fmt.Errorf("mengambil keputusan klaim %s: %w", query.ClaimNo, err)
	}

	return Decided{Items: items, Query: query}, nil
}

// Decide mencatat keputusan komite atas satu banding harga.
//
// # Kenapa ia mencatat ke log, sementara pembacaan tidak
//
// Karena ia satu-satunya operasi modul ini yang MENGUBAH nilai uang, dan `D-59` menjadikan
// jejak sebagai satu-satunya kontrol pengimbang justru untuk keadaan seperti ini: tidak ada
// pemisahan tugas, dan siapa pun yang punya menunya dapat memutuskan.
//
// Yang dicatat bukan sekadar "terjadi", melainkan langkah mana yang benar-benar berjalan —
// karena pada layar ini "disetujui" tidak selalu berarti "harganya berubah".
func (s *Service) Decide(
	ctx context.Context,
	portalAlias string,
	caller inboxbandinghargasalvage.Caller,
	input inboxbandinghargasalvage.DecisionInput,
) (inboxbandinghargasalvage.DecisionResult, error) {
	var empty inboxbandinghargasalvage.DecisionResult

	command, err := inboxbandinghargasalvage.NewDecisionCommand(input, caller)
	if err != nil {
		return empty, err
	}

	if s.writerSelector == nil {
		return empty, inboxbandinghargasalvage.ErrWriteNotAvailable
	}

	writer, err := s.writerSelector(portalAlias)
	if err != nil {
		return empty, err
	}

	result, err := writer.Decide(ctx, command)
	if err != nil {
		return empty, err
	}

	s.logDecision(portalAlias, caller, command, result)

	return result, nil
}

// logDecision mencatat keputusan yang benar-benar tersimpan.
func (s *Service) logDecision(
	portalAlias string,
	caller inboxbandinghargasalvage.Caller,
	command inboxbandinghargasalvage.DecisionCommand,
	result inboxbandinghargasalvage.DecisionResult,
) {
	if s.logger == nil {
		return
	}

	s.logger.Info(
		"keputusan banding harga salvage dicatat",
		slog.String("modul", "inbox-banding-harga-salvage"),
		slog.String("portal", portalAlias),
		slog.String("pemanggil", caller.Clean().Login),
		slog.String("atas_nama_komite", command.Reviewer.Name),
		slog.String("id_detail_salvage", command.DetailObject),
		slog.Bool("disetujui", command.Approved()),
		slog.Bool("harga_diterapkan", result.PriceApplied),
		slog.Bool("jenjang_berikutnya_ditutup", command.CascadeTo != ""),
		slog.String("catatan",
			"pengiriman ke balai lelang TIDAK dilakukan — keputusan Work Owner 2026-09-30"),
	)
}

// ============================================================================
// Dialog "Lihat File"
// ============================================================================

// Documents menyerahkan dokumen banding satu barang.
//
// # Kenapa ia permintaan TERSENDIRI, bukan ikut di baris grid
//
// Alasannya sama dengan panel rincian: dialog ini hampir selalu tertutup. Membawanya serta
// pada setiap baris berarti menjalankan kuerinya sebanyak baris yang tampil — dan kueri ini
// menyentuh tabel lampiran, yang barisnya membawa BLOB.
func (s *Service) Documents(
	ctx context.Context,
	portalAlias string,
	caller inboxbandinghargasalvage.Caller,
	detailObject, salvageID string,
) ([]inboxbandinghargasalvage.DocumentRow, error) {
	q, err := inboxbandinghargasalvage.NewDocumentQuery(detailObject, salvageID, caller)
	if err != nil {
		return nil, err
	}

	reader, err := s.documentReader(portalAlias)
	if err != nil {
		return nil, err
	}

	return reader.ListDocuments(ctx, q)
}

// DocumentContent menyerahkan isi satu dokumen banding.
//
// Kepemilikannya diperiksa di KUERI, bukan di sini — pemeriksaan yang dijalankan aplikasi
// setelah barisnya terambil berarti isi berkasnya sempat berada di memori aplikasi sebelum
// ditolak.
func (s *Service) DocumentContent(
	ctx context.Context,
	portalAlias string,
	caller inboxbandinghargasalvage.Caller,
	detailObject, salvageID, documentID string,
) (inboxbandinghargasalvage.DocumentContent, error) {
	var empty inboxbandinghargasalvage.DocumentContent

	q, err := inboxbandinghargasalvage.NewDocumentQuery(detailObject, salvageID, caller)
	if err != nil {
		return empty, err
	}

	if strings.TrimSpace(documentID) == "" {
		return empty, inboxbandinghargasalvage.ErrDocumentNotFound
	}

	reader, err := s.documentReader(portalAlias)
	if err != nil {
		return empty, err
	}

	return reader.DocumentContent(ctx, documentID, q)
}

// documentReader memilih pembaca dokumen milik satu portal.
//
// Pemasangan yang tidak menyediakannya dijawab ErrWriteNotAvailable — galat yang sama
// dipakai tombol keputusan, karena artinya pun sama: kemampuan ini tidak dipasang pada
// sambungan ini, bukan rusak.
func (s *Service) documentReader(
	portalAlias string,
) (inboxbandinghargasalvage.DocumentReader, error) {
	if s.docSelector == nil {
		return nil, inboxbandinghargasalvage.ErrWriteNotAvailable
	}
	return s.docSelector(portalAlias)
}

package usecase

import (
	"context"
	"log/slog"

	"claim-pnc/internal/inboxmanager"
)

// DecideInput adalah permintaan keputusan mentah dari layar.
type DecideInput struct {
	// Tab adalah kode antrean yang sedang diputuskan.
	Tab string

	// Verdict adalah keputusannya — "setujui" atau "tolak".
	Verdict inboxmanager.Verdict

	// Keys adalah kunci baris yang dipilih penyelia.
	Keys []string

	// Reason adalah alasan yang diketiknya.
	Reason string
}

// Decide menuliskan keputusan atas sejumlah baris antrean.
//
// # Ia satu-satunya operasi MENULIS di seluruh modul inbox
//
// Yang ditulisnya adalah kolom persetujuan pada tabel POOLDATA — tidak ada satu pun tabel
// DATAPEGA yang disentuh. Itu bukan kebetulan: selama masa paralel setiap tabel hanya boleh
// ditulis satu sistem (`P-1`), dan tabel objek kerja Pega tetap milik Pega.
//
// # Kenapa hasilnya membawa DUA angka
//
// Karena jumlah baris yang berubah boleh lebih kecil daripada jumlah yang dipilih: setiap
// pernyataan ikut menyaring status menunggu, sehingga baris yang sudah diputuskan orang lain
// tidak berubah. Selisihnya BUKAN kegagalan — ia keadaan yang wajib disampaikan, dan
// menyembunyikannya berarti melaporkan keberhasilan penuh atas pekerjaan yang tidak
// seluruhnya terjadi.
func (s *Service) Decide(
	ctx context.Context,
	portalAlias string,
	input DecideInput,
	caller inboxmanager.Caller,
) (inboxmanager.DecisionResult, error) {
	resolved, err := s.resolveCaller(ctx, portalAlias, caller)
	if err != nil {
		return inboxmanager.DecisionResult{}, err
	}

	decision, err := inboxmanager.NewDecision(
		input.Tab, input.Verdict, input.Keys, input.Reason, resolved)
	if err != nil {
		return inboxmanager.DecisionResult{}, err
	}

	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return inboxmanager.DecisionResult{}, err
	}

	changed, err := repo.Decide(ctx, decision)
	if err != nil {
		// Keputusan yang GAGAL pun dicatat. Tanpa itu, satu-satunya jejak percobaan
		// menulis adalah galat di log transport yang tidak menyebut siapa, antrean apa,
		// dan berapa baris.
		s.logDecision(portalAlias, decision, inboxmanager.DecisionResult{
			Requested: len(decision.Keys),
		}, err)
		return inboxmanager.DecisionResult{}, err
	}

	result := inboxmanager.DecisionResult{
		Requested: len(decision.Keys),
		Changed:   changed,
	}

	s.logDecision(portalAlias, decision, result, nil)
	return result, nil
}

// logDecision mencatat setiap keputusan yang dituliskan modul ini.
//
// # Kenapa ia WAJIB, bukan pelengkap
//
// `D-59` menetapkan satuan izin adalah menu dan tidak ada pemisahan tugas formal, sehingga
// jejak audit menjadi satu-satunya kontrol pengimbang yang tersisa. Di modul ini hal itu
// menggigit: penyelia yang berwenang atas menu ini menyetujui pengajuan yang boleh jadi
// diajukannya sendiri, dan tidak ada kontrol teknis yang mencegahnya.
//
// Yang dicatat karena itu lengkap: siapa, portal apa, antrean apa, keputusan apa, berapa
// baris diminta, berapa yang benar-benar berubah, dan KUNCI barisnya.
//
// Alasan penolakan TIDAK ikut dicatat di sini. Ia sudah tersimpan di kolom alasan barisnya,
// dan menyalinnya ke log berarti teks yang diketik pengguna berpindah ke tempat yang
// retensinya lebih longgar daripada basis data (`11-CROSSCUTTING.md` §2.4).
func (s *Service) logDecision(
	portalAlias string,
	d inboxmanager.Decision,
	result inboxmanager.DecisionResult,
	failure error,
) {
	if s.logger == nil {
		return
	}

	attrs := []any{
		slog.String("modul", "inbox-manager"),
		slog.String("portal", portalAlias),
		slog.String("login", d.Caller.Login),
		slog.String("tab", d.Tab.Code),
		slog.String("tab_nama", d.Tab.Name),
		slog.String("keputusan", string(d.Verdict)),
		slog.Int("diminta", result.Requested),
		slog.Int("berubah", result.Changed),
		slog.Any("kunci", d.Keys),
	}

	if failure != nil {
		s.logger.ErrorContext(context.Background(), "keputusan inbox manager GAGAL",
			append(attrs, slog.String("galat", failure.Error()))...)
		return
	}

	if stale := result.Stale(); stale > 0 {
		// Peringatan, bukan info: baris yang tidak berubah berarti dua orang mengerjakan
		// antrean yang sama, dan itu layak terlihat tanpa harus dicari.
		s.logger.WarnContext(context.Background(),
			"sebagian baris tidak berubah karena sudah diputuskan lebih dulu",
			append(attrs, slog.Int("tidak_berubah", stale))...)
		return
	}

	s.logger.InfoContext(context.Background(), "keputusan inbox manager ditulis", attrs...)
}

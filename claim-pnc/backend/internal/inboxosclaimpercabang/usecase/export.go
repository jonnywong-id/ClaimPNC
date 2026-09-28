package usecase

import (
	"context"
	"fmt"
	"strings"

	"claim-pnc/internal/inboxosclaimpercabang"
)

// ExportSession adalah satu kali ekspor yang sedang berjalan.
//
// # Kenapa ekspor punya sesi, sedangkan daftar tidak
//
// Karena dua hal pada ekspor hanya perlu dikerjakan SEKALI meski barisnya diambil
// berhalaman-halaman: menentukan cabang pemanggil, dan membaca faktor dominan seluruh klaim
// cabang itu. Mengulanginya pada setiap halaman berarti satu kueri tambahan per halaman untuk
// jawaban yang tidak berubah.
//
// Yang TIDAK disimpan di sini adalah barisnya. Sesi memegang cabang dan peta faktor; barisnya
// tetap diambil per halaman dan ditulis langsung ke jawaban, sehingga memori tetap datar
// berapa pun jumlah barisnya (`15-NFR-PERFORMANCE-SCALABILITY.md` §3.2 aturan 6).
type ExportSession struct {
	// Query adalah cabang yang barisnya diekspor, lengkap dengan namanya.
	//
	// Nama berkas disusun darinya, sehingga dua unduhan dari dua cabang tidak menghasilkan
	// dua berkas bernama sama di folder unduhan.
	Query inboxosclaimpercabang.Query

	repo    inboxosclaimpercabang.Repo
	factors map[string][]string
	clock   inboxosclaimpercabang.Clock
}

// BeginExport menyiapkan satu kali ekspor.
//
// Ia menempuh pemeriksaan yang SAMA PERSIS dengan List — identitas, lalu cabang — dan itu
// bukan pengulangan yang dapat dihemat: berkas ekspor memuat nama tertanggung, nomor polis,
// dan nilai uang, sehingga pemeriksaannya tidak boleh lebih longgar daripada layarnya.
func (s *Service) BeginExport(
	ctx context.Context,
	portalAlias string,
	caller inboxosclaimpercabang.Caller,
) (*ExportSession, error) {
	clean := caller.Clean()
	if clean.Login == "" {
		return nil, inboxosclaimpercabang.ErrCallerUnknown
	}

	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return nil, err
	}

	branch, resolved, err := repo.BranchOf(ctx, clean.DetailBranchCode)
	if err != nil {
		return nil, fmt.Errorf("menentukan cabang pemanggil: %w", err)
	}
	if !resolved {
		return nil, inboxosclaimpercabang.ErrBranchUnknown
	}

	query := inboxosclaimpercabang.Query{Branch: branch}

	// Faktor dominan dibaca SEKALI di sini, bukan per halaman dan bukan per baris.
	//
	// Ia terikat pada klaim outstanding satu cabang — penyaring dan gabungannya sama persis
	// dengan kueri daftarnya — sehingga besarnya tumbuh mengikuti pekerjaan satu cabang,
	// bukan mengikuti seluruh tabel.
	factors, err := repo.DominantFactors(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("membaca faktor dominan cabang %q: %w", branch.Code, err)
	}

	return &ExportSession{Query: query, repo: repo, factors: factors, clock: s.clock}, nil
}

// Page mengambil satu potong baris berkas, sudah lengkap dengan umur dan faktor dominannya.
func (e *ExportSession) Page(
	ctx context.Context,
	page inboxosclaimpercabang.Pagination,
) (inboxosclaimpercabang.ExportPage, error) {
	result, err := e.repo.ListForExport(ctx, e.Query, page)
	if err != nil {
		return inboxosclaimpercabang.ExportPage{},
			fmt.Errorf("mengambil baris ekspor cabang %q: %w", e.Query.Branch.Code, err)
	}

	now := e.clock.Now()
	for i := range result.Items {
		result.Items[i].AgingDays = inboxosclaimpercabang.AgingDaysSince(
			result.Items[i].RegisterDate, now)
		result.Items[i].DominantFactors = joinFactors(e.factors[result.Items[i].ClaimKey])
	}

	return result, nil
}

// joinFactors merangkai nama faktor dominan satu klaim.
//
// Bentuknya mengikuti kueri lama apa adanya: dipisah `", "`, dan `"-"` bila kosong.
//
//	NVL(LISTAGG(m.name, ', ') WITHIN GROUP (ORDER BY t.idx_dominanfactor), '-')
//
// Tanda `"-"` itu dipertahankan meski sel kosong akan sama jelasnya: ia yang selama ini
// terbaca di berkas, dan lembar kerja yang sudah menyaring nilai itu akan berhenti bekerja
// bila ia berubah menjadi kosong.
func joinFactors(names []string) string {
	if len(names) == 0 {
		return "-"
	}
	return strings.Join(names, ", ")
}

// Package usecase mengorkestrasi pencatatan Master Recovery: menerbitkan nomor batch,
// memilih principal, menerbitkan rekening virtual, menyimpan bukti bayar, membaca daftar
// klaim, dan menyimpan batch.
//
// Tugasnya tiga, dan tidak lebih: memilih Repo milik portal yang sedang aktif,
// menjalankan pemeriksaan isian, lalu memanggil Repo atau seam penerbit VA. Ia tidak tahu
// apa pun tentang HTTP maupun SQL.
//
// # Aksi yang ada, dan yang sengaja tidak ada
//
// Tidak ada Daftar, tidak ada Ubah, dan tidak ada Hapus. Itu bukan kelalaian, melainkan
// keputusan Work Owner 2026-09-19 untuk meniru sistem lama apa adanya — dan sistem lama
// memang tidak punya satu pun dari ketiganya:
//
//   - Tidak ada satu kueri pun di export yang MEMBACA
//     POOLDATA.MST_RECOVERY_ASM_PENJAMINAN; yang ada hanya penerbit nomor batch.
//   - `Database/INSERTMASTERRECOVERYKLAIM.prc` hanya mengenal INSERT.
//
// Rute yang tidak ada tidak dapat dipanggil kode yang ditulis kemudian tanpa keputusan
// sadar.
package usecase

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"claim-pnc/internal/masterrecovery"
)

// Service mencatat batch recovery di atas seam penyimpanan per portal.
type Service struct {
	repoSelector masterrecovery.RepoSelector
	issuer       masterrecovery.VirtualAccountIssuer
	now          func() time.Time
}

// Options adalah bahan pembentuk Service.
type Options struct {
	// RepoSelector memilih penyimpanan milik satu portal entitas. Wajib.
	RepoSelector masterrecovery.RepoSelector

	// Issuer menerbitkan rekening virtual. Wajib — pengisinya boleh adapter nyata maupun
	// fake, dan yang memilih adalah perakitan di cmd, bukan paket ini.
	Issuer masterrecovery.VirtualAccountIssuer

	// Now dapat diganti pada pengujian supaya daftar tahun tidak berubah arti setiap
	// pergantian tahun dan membuat pengujiannya gagal tanpa ada yang menyentuh kode.
	Now func() time.Time
}

// NewService membentuk Service dan menolak bahan yang tidak lengkap.
//
// Penolakannya terjadi saat perakitan di cmd, bukan saat permintaan pertama datang:
// rakitan yang setengah jadi harus gagal saat start, bukan saat pengguna sedang bekerja.
func NewService(o Options) (*Service, error) {
	if o.RepoSelector == nil {
		return nil, errors.New("masterrecovery/usecase: RepoSelector wajib diisi")
	}
	if o.Issuer == nil {
		return nil, errors.New("masterrecovery/usecase: Issuer wajib diisi")
	}

	now := o.Now
	if now == nil {
		now = time.Now
	}
	return &Service{repoSelector: o.RepoSelector, issuer: o.Issuer, now: now}, nil
}

// NextBatch mengembalikan nomor batch berikutnya untuk DITAMPILKAN di layar.
//
// Menggantikan `Activity/GetIDMasterRecoveryKlaim-Act.xml`, yang menjalankan
// `GetMasterRecoveryClaimSPK` lalu menyalin hasilnya ke `TempRecovery.TOTAL`.
//
// Angkanya PERKIRAAN, dan itu disebut terang-terangan di sini supaya tidak ada yang
// memperlakukannya sebagai jaminan: dua petugas yang membuka layar bersamaan akan melihat
// nomor yang sama. Yang mengikat adalah nomor yang diterbitkan Save di dalam transaksinya
// sendiri — sistem lama tidak punya pengaman itu sama sekali.
func (s *Service) NextBatch(ctx context.Context, portalAlias string) (int64, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return 0, err
	}

	batch, err := repo.NextBatch(ctx)
	if err != nil {
		return 0, fmt.Errorf("masterrecovery/usecase: membaca nomor batch berikutnya: %w", err)
	}
	return batch, nil
}

// Principals mengembalikan pilihan isian "Nama Principal".
//
// Menggantikan pra-aktivitas `getDataAllMSTVA` yang mengisi autocomplete pada layar lama.
// Tidak dipaginasi: master Virtual Account berisi dua baris pada portal ASM, dan ia
// daftar yang bertambah beberapa baris per tahun.
func (s *Service) Principals(ctx context.Context, portalAlias string) ([]masterrecovery.Principal, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return nil, err
	}

	list, err := repo.ListPrincipal(ctx)
	if err != nil {
		return nil, fmt.Errorf("masterrecovery/usecase: membaca daftar principal: %w", err)
	}
	return list, nil
}

// Years mengembalikan pilihan isian Tahun.
//
// # Kenapa dihitung, bukan dibaca
//
// Layar lama mengisinya dari Data Transform `GetListYear`, dan rule itu TIDAK ADA di
// export — satu dari ±242 rule yang hilang (`R-16`). Isinya tidak dapat dibaca dari mana
// pun, sehingga menyalinnya mustahil.
//
// Yang dikembalikan di sini adalah rentang yang dihitung dari tahun berjalan: sepuluh
// tahun ke belakang, satu ke depan. Sepuluh dipilih karena batch tertua yang benar-benar
// ada bertahun 2018, dan satu ke depan memberi ruang tahun buku yang sudah dibuka.
//
// Ini ASUMSI KERJA yang dicatat terbuka, bukan aturan yang dibaca. Begitu daftar
// sebenarnya tiba dari Tim Pega atau Work Owner, yang berubah hanyalah fungsi ini.
func (s *Service) Years(context.Context, string) []string {
	const (
		back    = 10
		forward = 1
	)

	current := s.now().Year()
	year := make([]string, 0, back+forward+1)
	// Terbaru lebih dulu: batch yang dicatat hampir selalu tahun berjalan, dan menaruhnya
	// di puncak menghemat satu gulir pada setiap pemakaian.
	for value := current + forward; value >= current-back; value-- {
		if value < masterrecovery.MinYear || value > masterrecovery.MaxYear {
			continue
		}
		year = append(year, fmt.Sprintf("%d", value))
	}
	return year
}

// LookupPolicy mencari identitas lini bisnis, cabang, agen, dan marketing dari nomor
// polis.
//
// Menggantikan `RDB List/GetRecoveryClaimData-SQL.xml`, yang di sistem lama dijalankan
// sebagai bagian dari aksi simpan. Di sini ia rute tersendiri supaya layar dapat
// menampilkan hasilnya SEBELUM petugas menyimpan — pada sistem lama, nomor polis yang
// salah ketik baru ketahuan setelah batch tersimpan dengan keempat identitas kosong.
func (s *Service) LookupPolicy(ctx context.Context, portalAlias, policyNo string) (masterrecovery.PolicyReference, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return masterrecovery.PolicyReference{}, err
	}

	policyNo = strings.TrimSpace(policyNo)
	if policyNo == "" {
		return masterrecovery.PolicyReference{}, masterrecovery.ErrPolicyNotFound
	}

	reference, err := repo.LookupPolicy(ctx, policyNo)
	if err != nil {
		if errors.Is(err, masterrecovery.ErrPolicyNotFound) {
			return masterrecovery.PolicyReference{}, err
		}
		return masterrecovery.PolicyReference{}, fmt.Errorf("masterrecovery/usecase: mencari polis %q: %w", policyNo, err)
	}
	return reference, nil
}

// IssueVirtualAccount menerbitkan rekening virtual untuk sebuah principal, atau
// mengembalikan yang sudah ada.
//
// # Urutannya meniru `GeneratedVAClaimRecovery`, termasuk pakai-ulangnya
//
//  1. Cari principal dengan Client ID dan nama yang sama di master VA.
//  2. Bila ADA → kembalikan nomornya, tandai Reused. Layanan luar tidak ditembak sama
//     sekali. Sistem lama pun demikian, dan alasannya kuat: menerbitkan VA kedua untuk
//     principal yang sama berarti dana masuk ke rekening yang tidak diawasi siapa pun.
//  3. Bila BELUM → terbitkan lewat seam, lalu catat ke master.
//
// # Dua hal yang sengaja TIDAK dibawa
//
// Pertama, `RDB List/GetDataVAbyValidasiVA-SQL.xml` menyusun klausa WHERE-nya dengan
// merangkai teks (`{ASIS:GeneratedVARecovery.NoteKomite}`) dari nama principal yang
// diketik pengguna — celah SQL injection yang nyata. Penggantinya memakai parameter
// binding tanpa perkecualian (`09-DATABASE-STRATEGY.md` §4).
//
// Kedua, `ADD_NEWMASTERVIRTUALACCOUNT.prc:18` mengembalikan NOMOR VA lewat parameter
// bernama `ErrMsg` ketika principalnya sudah ada — satu kolom yang membawa nilai berhasil
// sekaligus pesan galat. `D-68` menetapkan kontrak galat seperti itu tidak dibawa; di sini
// keduanya terpisah, dan "sudah ada" dinyatakan lewat penanda Reused yang dapat dibaca
// program.
func (s *Service) IssueVirtualAccount(
	ctx context.Context,
	portalAlias string,
	request masterrecovery.VirtualAccountRequest,
) (masterrecovery.VirtualAccount, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return masterrecovery.VirtualAccount{}, err
	}

	request.ClientID = strings.TrimSpace(request.ClientID)
	request.PrincipalName = strings.TrimSpace(request.PrincipalName)
	request.Email = strings.TrimSpace(request.Email)

	if err := masterrecovery.NewValidationError(masterrecovery.CheckVirtualAccountRequest(request)); err != nil {
		return masterrecovery.VirtualAccount{}, err
	}

	existing, err := repo.FindPrincipal(ctx, request.ClientID, request.PrincipalName)
	switch {
	case err == nil:
		return masterrecovery.VirtualAccount{
			Number:  existing.VirtualAccountNumber,
			Status:  existing.Status,
			Message: "Principal ini sudah memiliki virtual account.",
			Reused:  true,
		}, nil
	case !errors.Is(err, masterrecovery.ErrPrincipalNotFound):
		return masterrecovery.VirtualAccount{}, fmt.Errorf("masterrecovery/usecase: memeriksa principal: %w", err)
	}

	issued, err := s.issuer.Issue(ctx, portalAlias, request)
	if err != nil {
		// Ketiga galat penerbit diteruskan apa adanya: masing-masing menuntut tindak
		// lanjut yang berbeda, dan membungkusnya menjadi satu akan menghapus perbedaannya.
		if errors.Is(err, masterrecovery.ErrIssuerUnreachable) ||
			errors.Is(err, masterrecovery.ErrIssuerUnconfigured) ||
			errors.Is(err, masterrecovery.ErrIssuerRejected) {
			return masterrecovery.VirtualAccount{}, err
		}
		return masterrecovery.VirtualAccount{}, fmt.Errorf("masterrecovery/usecase: menerbitkan virtual account: %w", err)
	}
	if strings.TrimSpace(issued.Number) == "" {
		// Layanan menjawab tanpa nomor. Dicatat sebagai penolakan, bukan disimpan sebagai
		// baris master bernomor kosong yang kelak menerima dana entah ke mana.
		return masterrecovery.VirtualAccount{}, fmt.Errorf("%w: jawaban tidak memuat nomor virtual account", masterrecovery.ErrIssuerRejected)
	}

	if err := repo.SavePrincipal(ctx, masterrecovery.Principal{
		ClientID:             request.ClientID,
		Name:                 request.PrincipalName,
		VirtualAccountNumber: issued.Number,
		Email:                request.Email,
		Status:               issued.Status,
		Message:              issued.Message,
	}); err != nil {
		return masterrecovery.VirtualAccount{}, fmt.Errorf("masterrecovery/usecase: mencatat virtual account: %w", err)
	}

	issued.Reused = false
	return issued, nil
}

// SaveDocument menyimpan Bukti Bayar dan mengembalikan penandanya.
//
// Menggantikan `Call PNCSaveAttachmentToDB` pada activity lama, yang menyimpan metadata
// ke POOLDATA.DATA_ATTACHFILE lalu mengisi `TempRecovery.CoverageID` dengan DATAID
// jawabannya.
func (s *Service) SaveDocument(ctx context.Context, portalAlias string, document masterrecovery.Document) (string, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return "", err
	}

	if err := masterrecovery.NewValidationError(masterrecovery.CheckDocument(document)); err != nil {
		return "", err
	}

	id, err := repo.SaveDocument(ctx, document)
	if err != nil {
		return "", fmt.Errorf("%w: %v", masterrecovery.ErrDocumentNotSaved, err)
	}
	if strings.TrimSpace(id) == "" {
		// Procedure lama menjawab keadaan ini dengan DATAID kosong dan berhenti seolah
		// tidak terjadi apa-apa (`SET_ATTACHMENT_64BIT.prc:12`). Di sini ia menjadi galat
		// yang benar-benar galat — sebab batch yang menunjuk bukti bayar kosong tidak dapat
		// ditelusuri siapa pun kelak.
		return "", masterrecovery.ErrDocumentNotSaved
	}
	return id, nil
}

// ReadClaimLine membaca berkas CSV "Upload Data Klaim".
//
// Ia tidak menyentuh penyimpanan sama sekali: hasilnya dikembalikan ke layar untuk
// ditampilkan sebagai grid, lalu dikirim kembali bersama permintaan simpan. Itu meniru
// sistem lama, yang menyusun `TempRecovery.ObjectList` di klipboard sebelum menyimpannya
// sebagai satu dokumen JSON.
//
// Portal tetap diperiksa meski tidak ada basis data yang disentuh — permintaan tanpa
// portal ditolak di seluruh modul ini tanpa perkecualian (`TKT-F6-002`).
func (s *Service) ReadClaimLine(portalAlias string, source io.Reader) ([]masterrecovery.ClaimLine, []masterrecovery.Violation, error) {
	if _, err := s.repoSelector(portalAlias); err != nil {
		return nil, nil, err
	}
	return masterrecovery.ParseClaimLine(source)
}

// Save mencatat satu batch recovery.
//
// Menggantikan tombol **Transfer Recovery** beserta seluruh langkah
// `Activity/Insert_mst_recoveryKlaimASM-Act.xml`.
//
// # Urutannya, dan kenapa Sisa dihitung di sini
//
// Sisa TIDAK diterima dari badan permintaan, meski layar lama mengirimnya. Aturannya
// hidup di satu tempat — `masterrecovery.Remainder` — sehingga layar dan server tidak
// dapat berbeda pendapat tentang angka yang tersimpan. Layar tetap menghitungnya untuk
// diperlihatkan seketika; yang MENGIKAT adalah yang dihitung di sini.
//
// # Identitas polis dilengkapi server, dan kegagalannya tidak membatalkan apa pun
//
// Bila nomor polis disebut, keempat identitasnya dicari ulang di sini alih-alih dipercaya
// dari klien. Klien dapat mengirim apa saja; yang tersimpan harus benar-benar milik polis
// itu.
//
// Pencarian yang GAGAL — polisnya tidak ada, maupun DB Link ke MST_DET_SALES@ASMD sedang
// tidak dapat ditembak — tidak membatalkan penyimpanan. Sistem lama pun meneruskan dengan
// keempatnya kosong: `Insert_mst_recoveryKlaimASM` tidak punya satu pun prasyarat yang
// menghentikan alur ketika daftarnya kembali kosong.
//
// Itu keputusan yang diambil sadar, bukan kelonggaran: yang dicatat batch ini adalah
// NILAI UANG yang sudah diterima, dan menolak mencatatnya karena basis data pihak lain
// sedang mati akan membuat petugas kehilangan seluruh isian yang sudah diketik. Keempat
// kolom yang kosong tetap terlihat — layar membandingkannya dengan nomor polis yang
// diisi, dan mengatakannya kepada petugas.
func (s *Service) Save(ctx context.Context, portalAlias string, recovery masterrecovery.Recovery) (masterrecovery.Recovery, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return masterrecovery.Recovery{}, err
	}

	recovery = recovery.Clean()
	if err := masterrecovery.NewValidationError(masterrecovery.CheckRecovery(recovery)); err != nil {
		return masterrecovery.Recovery{}, err
	}

	recovery.Remainder = masterrecovery.Remainder(recovery.ClaimAmount, recovery.PreviousPayment, recovery.Payment)

	// Keempatnya dikosongkan LEBIH DULU, apa pun hasil pencarian di bawah. Membiarkan
	// nilai kiriman klien berarti menyimpan identitas yang tidak dapat dibuktikan milik
	// polis mana pun — dan klien tidak pernah menjadi sumber kebenaran untuk hal ini.
	recovery.BusinessID, recovery.BranchID, recovery.AgentID, recovery.MarketingID = "", "", "", ""

	if recovery.PolicyNo != "" {
		reference, err := repo.LookupPolicy(ctx, recovery.PolicyNo)
		if err == nil {
			recovery.BusinessID = reference.BusinessID
			recovery.BranchID = reference.BranchID
			recovery.AgentID = reference.AgentID
			recovery.MarketingID = reference.MarketingID
		}
		// Galat apa pun dibiarkan: keempatnya sudah kosong, dan alasannya ada di komentar
		// kepala fungsi ini. Yang gagal dicatat pemanggil lewat log, bukan dengan
		// membatalkan pencatatan uang yang sudah diterima.
	}

	saved, err := repo.Insert(ctx, recovery)
	if err != nil {
		if errors.Is(err, masterrecovery.ErrBatchTaken) {
			return masterrecovery.Recovery{}, err
		}
		return masterrecovery.Recovery{}, fmt.Errorf("masterrecovery/usecase: menyimpan batch recovery: %w", err)
	}
	return saved, nil
}

// EnsurePortalReady memeriksa portal dapat dilayani tanpa menyentuh satu baris pun.
//
// Dipakai transport untuk menolak lebih awal, sebelum badan permintaan dibaca.
func (s *Service) EnsurePortalReady(portalAlias string) error {
	if _, err := s.repoSelector(portalAlias); err != nil {
		return fmt.Errorf("masterrecovery/usecase: portal %q tidak dapat dilayani: %w", portalAlias, err)
	}
	return nil
}

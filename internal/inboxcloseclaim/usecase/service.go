// Package usecase mengorkestrasi modul Inbox Close Claim.
//
// Isinya dua hal yang justru karena itulah layak dipisahkan dari transport:
//
//  1. MENYATUKAN daftar klaim dengan permintaan yang masih menunggu atasnya. Tanpa itu,
//     layar tidak punya cara memberi tahu pengguna bahwa tombolnya sudah ditekan — dan
//     klaim TIDAK berubah seketika, karena yang tercatat baru permintaannya.
//  2. MEMAGARI kedua aksi tulis: memastikan klaimnya benar-benar ada dan benar-benar sudah
//     tutup sebelum permintaan dicatat.
//
// Menaruh keduanya di handler akan membuat setiap rute baru harus mengingat untuk
// melakukannya, dan yang lupa tidak menghasilkan galat apa pun.
package usecase

import (
	"context"
	"fmt"

	"claim-pnc/internal/inboxcloseclaim"
)

// Service melayani layar Inbox Close Claim.
type Service struct {
	claims     inboxcloseclaim.RepoSelector
	requests   inboxcloseclaim.RequestRepoSelector
	ids        inboxcloseclaim.IDGenerator
	clock      inboxcloseclaim.Clock
	canRequest func() bool
}

// Options adalah bahan pembentuk Service.
type Options struct {
	Claims   inboxcloseclaim.RepoSelector
	Requests inboxcloseclaim.RequestRepoSelector
	IDs      inboxcloseclaim.IDGenerator
	Clock    inboxcloseclaim.Clock

	// CanRequest menyatakan apakah pengajuan ReOpen dan Copy Klaim terbuka.
	//
	// Kosong berarti `inboxcloseclaim.CanRequestAction` — aturan yang sesungguhnya, yaitu
	// When rule `IsGCNMUser` yang isinya `1 = 2` dan karena itu SELALU SALAH.
	//
	// # Kenapa ia seam, padahal aturannya konstan
	//
	// Karena tanpa seam ini, SELURUH jalur pengajuan menjadi tidak dapat diuji: gerbangnya
	// menolak sebelum apa pun berjalan, sehingga pemeriksaan klaim, pencatatan niat,
	// penolakan permintaan ganda, dan pemetaan galatnya tidak pernah tersentuh satu uji pun.
	//
	// Akibatnya bukan sekadar cakupan uji yang turun. Hari ketika Work Owner membuka
	// gerbangnya, yang menyala adalah kode yang **tidak pernah sekali pun dijalankan** —
	// dan itu justru saat cacatnya paling mahal, karena kedua aksi menyentuh klaim yang
	// sudah tutup.
	//
	// Seam ini TIDAK melonggarkan aturannya: di `cmd/claimpnc` ia tidak diisi, sehingga yang
	// berlaku di aplikasi sungguhan tetap aturan yang sesungguhnya. Yang mengisinya hanya
	// uji, dan uji yang membuktikan aturan itu benar-benar menutup memakai jalur bawaannya —
	// lihat `TestRequestDitolakSaatGerbangTertutup`.
	CanRequest func() bool
}

// NewService membentuk service. Keempatnya wajib diisi.
//
// Klaim dan permintaan keduanya datang lewat SELECTOR, bukan repo tunggal: `ADR-0030`
// menetapkan satu database per entitas, dan tabel permintaan dibuat di setiap portal karena
// klaim yang dirujuknya pun ada di sana.
func NewService(o Options) (*Service, error) {
	if o.Claims == nil {
		return nil, fmt.Errorf("inboxcloseclaim/usecase: pemilih repo klaim wajib diisi")
	}
	if o.Requests == nil {
		return nil, fmt.Errorf("inboxcloseclaim/usecase: pemilih repo permintaan wajib diisi")
	}
	if o.IDs == nil {
		return nil, fmt.Errorf("inboxcloseclaim/usecase: pembangkit pengenal wajib diisi")
	}
	if o.Clock == nil {
		return nil, fmt.Errorf("inboxcloseclaim/usecase: jam wajib diisi")
	}

	// Gerbang kewenangan TIDAK wajib diisi, berbeda dari keempat seam di atas.
	//
	// Bawaannya adalah aturan yang sesungguhnya, sehingga perakit yang lupa mengisinya
	// mendapat perilaku yang BENAR — bukan perilaku yang terbuka. Seam yang bawaannya
	// meloloskan adalah seam yang cepat atau lambat akan meloloskan sesuatu.
	canRequest := o.CanRequest
	if canRequest == nil {
		canRequest = inboxcloseclaim.CanRequestAction
	}

	return &Service{
		claims:     o.Claims,
		requests:   o.Requests,
		ids:        o.IDs,
		clock:      o.Clock,
		canRequest: canRequest,
	}, nil
}

// CanRequest menyatakan apakah pengajuan terbuka.
//
// Transport memakainya untuk MEMBERI TAHU layar; Request memakainya untuk MENOLAK. Keduanya
// membaca gerbang yang sama, sehingga tombol yang tampil aktif dan endpoint yang menerima
// tidak dapat berselisih.
func (s *Service) CanRequest() bool { return s.canRequest() }

// ListQuery adalah permintaan isi layar.
type ListQuery struct {
	// PortalAlias adalah entitas yang sedang dibuka, diambil dari header `X-Portal` yang
	// sudah diperiksa middleware. Ia menentukan BASIS DATA mana yang dibaca.
	//
	// Permintaan tanpa portal DITOLAK, tidak pernah dilayani portal utama sebagai cadangan
	// (`R-20`).
	PortalAlias string

	Filter inboxcloseclaim.Filter
}

// ListResult adalah satu halaman hasil beserta permintaan yang menunggu atas baris-barisnya.
type ListResult struct {
	Page inboxcloseclaim.Page

	// Pending memetakan ClaimID ke permintaan yang masih menunggu.
	//
	// Ia ada supaya layar dapat MENYATAKANNYA. Tanpa itu, pengguna yang sudah menekan
	// ReOpen tidak melihat perubahan apa pun — barisnya tetap di sana, karena klaimnya
	// memang belum berubah — lalu menekannya lagi.
	Pending map[string][]inboxcloseclaim.ClaimRequest

	// PendingLookupError terisi bila permintaan tertunda GAGAL DIBACA.
	//
	// Ia sengaja BUKAN galat yang dikembalikan: tabel permintaan dibuat migrasi `0006` yang
	// belum dijalankan DBA di lingkungan mana pun, sehingga pembacaannya akan gagal di
	// setiap portal hari ini. Mengembalikannya sebagai galat berarti LAYAR INI MATI TOTAL
	// sampai perubahan skema selesai — padahal daftarnya sendiri sudah dapat dipakai.
	//
	// Yang dilakukan: daftar tetap tampil, penanda "permintaan terkirim" tidak muncul, dan
	// pemanggil WAJIB mencatat galat ini. Bila pemanggil mengabaikannya, kegagalan itu
	// menjadi tidak terlihat oleh siapa pun.
	PendingLookupError error
}

// List membaca satu halaman klaim tutup beserta permintaan yang menunggu atasnya.
func (s *Service) List(ctx context.Context, q ListQuery) (ListResult, error) {
	claims, err := s.claims(q.PortalAlias)
	if err != nil {
		return ListResult{}, err
	}

	page, err := claims.List(ctx, q.Filter)
	if err != nil {
		return ListResult{}, fmt.Errorf("inboxcloseclaim/usecase: membaca daftar klaim tutup: %w", err)
	}

	result := ListResult{Page: page, Pending: map[string][]inboxcloseclaim.ClaimRequest{}}

	requests, err := s.requests(q.PortalAlias)
	if err != nil {
		result.PendingLookupError = err
		return result, nil
	}

	ids := make([]string, 0, len(page.Claims))
	for _, claim := range page.Claims {
		ids = append(ids, claim.ClaimID)
	}

	pending, err := requests.PendingFor(ctx, ids)
	if err != nil {
		result.PendingLookupError = fmt.Errorf(
			"inboxcloseclaim/usecase: membaca permintaan tertunda: %w", err)
		return result, nil
	}
	result.Pending = pending
	return result, nil
}

// RequestCommand adalah permintaan ReOpen atau Copy Klaim dari layar.
type RequestCommand struct {
	PortalAlias string

	Kind    inboxcloseclaim.RequestKind
	ClaimID string
	Reason  string

	// ActorLogin dan ActorName diambil dari SESI, tidak pernah dari badan permintaan.
	//
	// Jejak yang pelakunya dikirim klien bukan jejak: siapa pun yang dapat memanggil
	// endpoint-nya dapat menuliskan nama orang lain di sana. Pada modul ini taruhannya
	// nyata — `D-59` menghapus pemisahan tugas, sehingga jejak audit adalah SATU-SATUNYA
	// kontrol pengimbang yang tersisa.
	ActorLogin string
	ActorName  string
}

// Request mencatat satu permintaan atas klaim yang sudah tutup.
//
// # Urutannya mengikat
//
// Klaim diperiksa LEBIH DULU, permintaan dicatat kemudian. Membaliknya berarti permintaan
// dapat tercatat atas kunci klaim yang tidak ada — atau atas klaim yang MASIH BERJALAN,
// yang justru tidak boleh dibuka kembali karena belum pernah tutup.
//
// # Yang TIDAK dilakukan di sini
//
// Klaimnya tidak disentuh. `P-1` menetapkan `DATAPEGA.PC_ASM_FW_GCNMFW_WORK` ditulis Pega
// selama masa paralel, dan Work Owner memutuskan 2026-09-23 aplikasi ini mencatat
// PERMINTAANNYA saja. Efek yang dikehendaki ikut disimpan pada barisnya supaya yang
// menjalankannya kelak tidak perlu menebak aturan mana yang berlaku.
func (s *Service) Request(
	ctx context.Context,
	cmd RequestCommand,
) (inboxcloseclaim.ClaimRequest, error) {
	// Kewenangan diperiksa PALING DULU — sebelum klaimnya dicari, dan sebelum apa pun
	// dibaca dari basis data.
	//
	// Urutannya mengikat. Memeriksanya belakangan berarti pemanggil yang tidak berwenang
	// tetap dapat menanyakan keberadaan sebuah klaim lewat perbedaan galat yang ia terima:
	// "tidak ditemukan" untuk kunci yang salah, "tidak berwenang" untuk kunci yang benar.
	// Perbedaan itu cukup untuk menebak nomor klaim satu per satu.
	if !s.canRequest() {
		return inboxcloseclaim.ClaimRequest{}, inboxcloseclaim.ErrRequestNotAllowed
	}

	claims, err := s.claims(cmd.PortalAlias)
	if err != nil {
		return inboxcloseclaim.ClaimRequest{}, err
	}

	number, err := claims.ClosedClaimNumber(ctx, cmd.ClaimID)
	if err != nil {
		// ErrClaimNotFound diteruskan APA ADANYA supaya transport dapat menjawab 404.
		// Membungkusnya akan membuat errors.Is di sana gagal, dan pengguna menerima 500
		// untuk nomor klaim yang salah ketik.
		return inboxcloseclaim.ClaimRequest{}, err
	}

	request := inboxcloseclaim.ClaimRequest{
		ID:          s.ids.New(),
		Kind:        cmd.Kind,
		ClaimID:     cmd.ClaimID,
		ClaimNumber: number,
		Reason:      cmd.Reason,
		Status:      inboxcloseclaim.RequestPending,
		ActorLogin:  cmd.ActorLogin,
		ActorName:   cmd.ActorName,
		RequestedAt: s.clock.Now(),
	}

	// Niat yang dicatat berbeda menurut jenisnya, dan keduanya ditetapkan Work Owner
	// 2026-09-23 — bukan dibaca dari export, yang memang tidak memuatnya.
	switch cmd.Kind {
	case inboxcloseclaim.RequestReopen:
		request.EffectWorkStatus = inboxcloseclaim.EfekStatusKerjaReopen
		request.EffectClaimStatus = inboxcloseclaim.EfekStatusKlaimReopen
	case inboxcloseclaim.RequestCopy:
		request.CopyScope = inboxcloseclaim.LingkupSalinBaku
	}

	request = request.Normalize()
	if err := request.Validate(); err != nil {
		return inboxcloseclaim.ClaimRequest{}, err
	}

	requests, err := s.requests(cmd.PortalAlias)
	if err != nil {
		return inboxcloseclaim.ClaimRequest{}, err
	}
	if err := requests.Record(ctx, request); err != nil {
		// ErrRequestPending diteruskan apa adanya, dengan alasan yang sama seperti
		// ErrClaimNotFound di atas.
		return inboxcloseclaim.ClaimRequest{}, err
	}
	return request, nil
}

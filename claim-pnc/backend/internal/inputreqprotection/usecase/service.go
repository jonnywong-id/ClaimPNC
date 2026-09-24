// Package usecase mengorkestrasi modul Input Req Protection.
//
// Isinya dua hal yang tidak boleh hidup di handler maupun di penyimpanan:
//
//  1. URUTAN pemeriksaan sebelum menyimpan — validasi bentuk, lalu penguncian, lalu
//     proteksi ganda. Urutannya menentukan pesan mana yang dilihat pengguna lebih dulu.
//  2. Pemilihan PENYIMPANAN menurut portal yang sedang dibuka.
//
// Menaruh keduanya di handler akan membuat setiap rute baru harus mengingat untuk
// melakukannya, dan yang lupa tidak menghasilkan galat apa pun — hanya proteksi yang
// tersimpan tanpa diperiksa, atau tersimpan di basis data entitas yang salah.
package usecase

import (
	"context"
	"fmt"
	"time"

	"claim-pnc/internal/inputreqprotection"
)

// Service melayani daftar dan penyuntingan permintaan proteksi.
type Service struct {
	protections inputreqprotection.RepoSelector
	now         func() time.Time
	location    *time.Location
}

// Options adalah bahan pembentuk Service.
type Options struct {
	// Protections wajib diisi.
	Protections inputreqprotection.RepoSelector

	// Now dapat diisi uji supaya waktu pembuatan dan aturan "hari ini" pada pemeriksaan
	// proteksi ganda dapat diperiksa secara deterministik.
	Now func() time.Time

	// Location adalah zona waktu yang dipakai menentukan "hari ini". Kosong berarti
	// Asia/Jakarta.
	//
	// Ia parameter, bukan konstanta, karena aturan proteksi ganda di sistem lama memakai
	// `@CurrentDate("dd MMM yyyy","WIB")` — tanggal kalender WIB, bukan 24 jam terakhir.
	// Lihat inputreqprotection.DuplicateKey.
	Location *time.Location
}

// NewService membentuk service.
func NewService(o Options) (*Service, error) {
	if o.Protections == nil {
		return nil, fmt.Errorf("inputreqprotection/usecase: pemilih repo proteksi wajib diisi")
	}

	now := o.Now
	if now == nil {
		now = time.Now
	}
	location := o.Location
	if location == nil {
		location = jakarta()
	}

	return &Service{protections: o.Protections, now: now, location: location}, nil
}

// jakarta mengembalikan zona WIB, jatuh ke offset tetap bila basis data zona waktu mesin
// tidak memuatnya.
//
// Cadangan itu bukan kemewahan: citra kontainer yang ramping kerap tidak menyertakan
// tzdata, dan tanpa cadangan seluruh pemeriksaan "hari ini" akan diam-diam memakai UTC —
// menggeser batas hari tujuh jam.
func jakarta() *time.Location {
	if loc, err := time.LoadLocation("Asia/Jakarta"); err == nil {
		return loc
	}
	return time.FixedZone("WIB", 7*60*60)
}

// ── Membaca ──────────────────────────────────────────────────────────────────────

// ListQuery adalah permintaan daftar dari layar.
type ListQuery struct {
	// PortalAlias adalah entitas yang sedang dibuka, dari header `X-Portal` yang sudah
	// diperiksa middleware. Ia menentukan BASIS DATA mana yang dibaca.
	//
	// Permintaan tanpa portal DITOLAK, tidak pernah dilayani portal utama sebagai cadangan
	// (`R-20`, `TKT-F6-002`).
	PortalAlias string

	Search string
	Limit  int
	Offset int
}

// List membaca satu halaman proteksi yang belum diakseptasi.
func (s *Service) List(ctx context.Context, q ListQuery) (inputreqprotection.Page, error) {
	stores, err := s.protections(q.PortalAlias)
	if err != nil {
		return inputreqprotection.Page{}, err
	}

	page, err := stores.Protections.List(ctx, inputreqprotection.Filter{
		Search: q.Search,
		Limit:  q.Limit,
		Offset: q.Offset,
	})
	if err != nil {
		return inputreqprotection.Page{}, fmt.Errorf("inputreqprotection/usecase: membaca daftar proteksi: %w", err)
	}
	return page, nil
}

// Get membaca satu proteksi menurut nomornya.
//
// Galat ErrNotFound diteruskan APA ADANYA, tidak dibungkus pesan lain, supaya transport
// dapat mengenalinya dengan errors.Is dan menjawab 404 alih-alih 500.
func (s *Service) Get(ctx context.Context, portalAlias, number string) (inputreqprotection.Protection, error) {
	stores, err := s.protections(portalAlias)
	if err != nil {
		return inputreqprotection.Protection{}, err
	}
	return stores.Protections.Get(ctx, number)
}

// ListTypes membaca master tipe proteksi milik portal yang sedang dibuka.
//
// # Kenapa lewat portal, padahal isinya master
//
// `ADR-0030` menetapkan satu basis data per entitas, dan master pun tinggal di dalamnya.
// Membacanya dari portal utama akan menampilkan tipe milik Asuransi Sinar Mas di layar
// Simas Insurtech — kesalahan yang tidak menghasilkan galat apa pun, hanya pilihan tipe
// yang keliru.
//
// # Kenapa tidak di-cache di sini
//
// Masternya kecil dan jarang berubah, sehingga cache akan menggoda. Tempatnya bukan di
// sini: cache per portal menuntut invalidasi yang belum ada pemiliknya, dan pilihan tipe
// yang basi setelah master disunting jauh lebih membingungkan daripada satu kueri
// tambahan. `14-NFR` §3.3 menetapkan cache ditambahkan setelah TERBUKTI perlu.
func (s *Service) ListTypes(ctx context.Context, portalAlias string) ([]inputreqprotection.ProtectionType, error) {
	stores, err := s.protections(portalAlias)
	if err != nil {
		return nil, err
	}

	daftar, err := stores.Types.ListTypes(ctx)
	if err != nil {
		return nil, fmt.Errorf("inputreqprotection/usecase: membaca master tipe proteksi: %w", err)
	}
	return daftar, nil
}

// FindClaim mencari klaim yang hendak ditaut, untuk mengisi field turunan pada form.
//
// # Kenapa layar memanggilnya, padahal server mencarinya lagi saat menyimpan
//
// Keduanya melayani hal berbeda. Pemanggilan dari layar membuat pengguna MELIHAT apa yang
// akan tersimpan sebelum ia menekan Simpan — termasuk DOL sebelum dan Cause of Loss
// sekarang, yang tidak dapat ia ketik.
//
// Pencarian saat menyimpan yang MENENTUKAN apa yang benar-benar tersimpan. Menghapus salah
// satunya menghilangkan hal yang berbeda: tanpa yang pertama pengguna menyimpan sesuatu yang
// belum pernah ia lihat; tanpa yang kedua nilai "sebelum" dapat dipalsukan.
//
// Galat `ErrClaimNotFound` diteruskan APA ADANYA supaya transport menjawab 404, bukan 500.
func (s *Service) FindClaim(
	ctx context.Context,
	portalAlias, number string,
) (inputreqprotection.Claim, error) {
	stores, err := s.protections(portalAlias)
	if err != nil {
		return inputreqprotection.Claim{}, err
	}
	return stores.Claims.FindClaim(ctx, number)
}

// ── Menulis ──────────────────────────────────────────────────────────────────────

// SaveCommand adalah permintaan menyimpan, baik membuat baru maupun menyunting.
type SaveCommand struct {
	PortalAlias string

	// Number diisi hanya saat menyunting. Kosong berarti membuat baru.
	Number string

	// Draft adalah isian form.
	Draft inputreqprotection.Draft

	// By adalah identitas pemanggil, diambil dari sesi — BUKAN dari badan permintaan.
	//
	// Kalau ia datang dari badan permintaan, siapa pun dapat menyimpan proteksi atas nama
	// orang lain, dan jejaknya akan menunjuk orang yang tidak melakukannya.
	By string
}

// Create menyimpan permintaan proteksi baru.
//
// # Urutan pemeriksaannya mengikat
//
//  1. bentuk isian (field wajib, detail perubahan sesuai tipe)
//  2. proteksi ganda pada polis dan tipe yang sama hari ini
//
// Dibalik, pengguna yang lupa mengisi Keterangan akan lebih dulu menerima pesan "sudah ada
// proteksi yang sama" — pesan yang benar tetapi menjawab pertanyaan yang tidak ia ajukan.
func (s *Service) Create(ctx context.Context, cmd SaveCommand) (inputreqprotection.Protection, error) {
	if cmd.By == "" {
		// Tidak boleh terjadi: rute dilindungi sesi. Bila terjadi, ia cacat pemrograman —
		// dan menyimpannya dengan pembuat kosong akan menghasilkan baris yang tidak dapat
		// ditelusuri.
		return inputreqprotection.Protection{}, fmt.Errorf("inputreqprotection/usecase: identitas pemanggil kosong")
	}

	draft := cmd.Draft.Normalize()
	if err := draft.Validate(); err != nil {
		return inputreqprotection.Protection{}, err
	}

	stores, err := s.protections(cmd.PortalAlias)
	if err != nil {
		return inputreqprotection.Protection{}, err
	}

	// Klaimnya dicari DI SINI, bukan dipercaya dari klien.
	//
	// Layar memang sudah mencarinya saat pengguna mengetik nomor klaim, dan mengisi field
	// turunannya dari sana. Tetapi pencarian itu terjadi di peramban, dan badan permintaan
	// yang dirakit tangan dapat menyebut nilai "sebelum" apa pun.
	//
	// Mengulangnya di sini bukan pemborosan: inilah satu-satunya pembacaan yang hasilnya
	// benar-benar tersimpan.
	claim, err := stores.Claims.FindClaim(ctx, draft.ClaimNumber)
	if err != nil {
		return inputreqprotection.Protection{}, err
	}

	at := s.now()
	if err := s.rejectDuplicate(ctx, stores.Protections, claim, draft, at, ""); err != nil {
		return inputreqprotection.Protection{}, err
	}

	saved, err := stores.Protections.Create(ctx, draft, claim, cmd.By, at)
	if err != nil {
		return inputreqprotection.Protection{}, fmt.Errorf("inputreqprotection/usecase: menyimpan proteksi: %w", err)
	}
	return saved, nil
}

// Update menyunting permintaan proteksi yang masih boleh disunting.
//
// # Kenapa keadaannya dibaca lebih dulu
//
// Penguncian oleh nomor klaim (`Protection.Editable`) adalah aturan bisnis, dan aturan
// bisnis diperiksa di sini supaya pesannya dapat dibedakan. Penyimpanan MEMERIKSANYA LAGI,
// dan itu bukan pengulangan yang sia-sia: pemeriksaan di sini menjaga pengguna dari
// kesalahan, pemeriksaan di sana menjaga data dari dua permintaan yang tiba bersamaan.
func (s *Service) Update(ctx context.Context, cmd SaveCommand) (inputreqprotection.Protection, error) {
	if cmd.By == "" {
		return inputreqprotection.Protection{}, fmt.Errorf("inputreqprotection/usecase: identitas pemanggil kosong")
	}

	stores, err := s.protections(cmd.PortalAlias)
	if err != nil {
		return inputreqprotection.Protection{}, err
	}

	existing, err := stores.Protections.Get(ctx, cmd.Number)
	if err != nil {
		return inputreqprotection.Protection{}, err
	}

	// Dua keadaan, dua pesan. Lihat inputreqprotection.ErrAccepted.
	if existing.Accepted() {
		return inputreqprotection.Protection{}, inputreqprotection.ErrAccepted
	}
	if !existing.Editable() {
		return inputreqprotection.Protection{}, inputreqprotection.ErrLocked
	}

	draft := cmd.Draft.Normalize()
	if err := draft.Validate(); err != nil {
		return inputreqprotection.Protection{}, err
	}

	// Dicari ulang juga saat menyunting: nomor klaimnya boleh berubah, dan nilai "sebelum"
	// harus mengikuti klaim yang BARU — bukan tertinggal pada klaim sebelumnya.
	claim, err := stores.Claims.FindClaim(ctx, draft.ClaimNumber)
	if err != nil {
		return inputreqprotection.Protection{}, err
	}

	at := s.now()
	if err := s.rejectDuplicate(ctx, stores.Protections, claim, draft, at, existing.Number); err != nil {
		return inputreqprotection.Protection{}, err
	}

	saved, err := stores.Protections.Update(ctx, cmd.Number, draft, claim, cmd.By, at)
	if err != nil {
		return inputreqprotection.Protection{}, fmt.Errorf("inputreqprotection/usecase: menyunting proteksi: %w", err)
	}
	return saved, nil
}

// rejectDuplicate menolak proteksi ganda pada polis dan tipe yang sama di hari yang sama.
//
// exceptNumber dikecualikan dari pemeriksaan supaya menyunting proteksi TANPA MENGUBAH
// polis maupun tipenya tidak ditolak oleh dirinya sendiri — cacat yang membuat setiap
// penyuntingan mustahil, dan yang hanya muncul saat menyunting, bukan saat membuat.
func (s *Service) rejectDuplicate(
	ctx context.Context,
	repo inputreqprotection.Repo,
	claim inputreqprotection.Claim,
	draft inputreqprotection.Draft,
	at time.Time,
	exceptNumber string,
) error {
	// Nomor polis datang dari KLAIM, bukan dari isian. Memakai isian akan membuat pemeriksaan
	// ganda berjalan atas polis yang belum tentu polis klaimnya — dan proteksi ganda yang
	// sebenarnya lolos.
	key := inputreqprotection.DuplicateKeyFor(claim.PolicyNumber, draft, at, s.location)

	duplicate, err := repo.HasDuplicate(ctx, key, exceptNumber)
	if err != nil {
		return fmt.Errorf("inputreqprotection/usecase: memeriksa proteksi ganda: %w", err)
	}
	if !duplicate {
		return nil
	}

	// Dikembalikan sebagai kesalahan validasi, bukan sebagai konflik teknis, supaya layar
	// menampilkannya di tempat yang sama dengan pesan validasi lain — menempel pada kolom
	// No Polis, yang memang harus diubah pengguna.
	v := &inputreqprotection.ValidationError{}
	v.Add(inputreqprotection.FieldPolicyNumber, inputreqprotection.DuplicateMessage)
	return v.OrNil()
}

package registrasi

import "time"

// Task adalah satu pekerjaan yang menunggu dikerjakan orang pada satu tahap klaim
// (`D-26`, `ADR-0019`).
//
// Sebuah klaim melewati banyak tugas berturut-turut. Satu tugas selalu berada di
// Worklist ATAU di Workbasket — tidak pernah di keduanya, dan itu ditegakkan oleh
// konstruktornya, bukan oleh kesepakatan.
type Task struct {
	ID          string
	ClaimID     string
	ClaimNumber string

	// Stage adalah pengenal tahap alur yang melahirkan tugas ini.
	Stage string

	Queue QueueKind

	// Workbasket terisi hanya untuk QueueWorkbasket.
	Workbasket string

	// Owner adalah orang yang mengerjakan tugas ini.
	//
	// Untuk Worklist ia terisi sejak tugas lahir. Untuk Workbasket ia KOSONG sampai
	// seseorang mengambilnya — itulah yang membedakan antrean bersama dari daftar
	// pribadi.
	Owner string

	CreatedAt   time.Time
	ClaimedAt   *time.Time
	CompletedAt *time.Time

	// CompletionReason menyimpan nama tindakan yang menutup tugas — nama Flow Action di
	// sistem lama. Ia dicatat supaya riwayat klaim dapat dibaca tanpa menebak.
	CompletionReason string
}

// NewTask membentuk tugas untuk sebuah tahap.
//
// Assignee ditentukan lebih dulu oleh seam Penugasan; fungsi ini hanya menyusunnya
// menjadi tugas yang konsisten dengan jenis antrean tahapnya.
func NewTask(id string, claim Claim, stage Stage, recipients Assignee, at time.Time) Task {
	t := Task{
		ID:          id,
		ClaimID:     claim.ID,
		ClaimNumber: claim.Number,
		Stage:       stage.ID,
		Queue:       stage.Queue,
		CreatedAt:   at.UTC(),
	}

	if stage.Queue == QueueWorkbasket {
		t.Workbasket = stage.Workbasket
		return t
	}

	t.Owner = recipients.Operator
	claimTime := at.UTC()
	t.ClaimedAt = &claimTime
	return t
}

// Open menyatakan tugas masih menunggu dikerjakan.
func (t Task) Open() bool { return t.CompletedAt == nil }

// Owned menyatakan tugas sudah punya pemilik.
func (t Task) Owned() bool { return t.Owner != "" }

// Get menjadikan seorang pengguna pemilik tugas dari Workbasket.
//
// Dua orang yang menekan tombol ambil pada tugas yang sama adalah kejadian biasa, bukan
// kasus tepi: sebuah Workbasket memang dilihat banyak orang sekaligus. Yang kedua
// menerima ErrTaskAlreadyClaimed — bukan diam-diam kehilangan pekerjaannya.
//
// Penjagaan terakhir terhadap balapan ada di penyimpanan (`TKT-B06-003`); pemeriksaan di
// sini menjawab kasus yang sudah terlihat tanpa menyentuh basis data.
func (t *Task) Get(by string, at time.Time) error {
	if !t.Open() {
		return ErrTaskAlreadyDone
	}
	if t.Owned() {
		if t.Owner == by {
			return nil
		}
		return ErrTaskAlreadyClaimed
	}
	t.Owner = by
	claimTime := at.UTC()
	t.ClaimedAt = &claimTime
	return nil
}

// Complete menutup tugas.
func (t *Task) Complete(by, action string, at time.Time) error {
	if !t.Open() {
		return ErrTaskAlreadyDone
	}
	if t.Owned() && t.Owner != by {
		return ErrNotTaskOwner
	}
	if !t.Owned() {
		// Tugas Workbasket yang belum diambil tidak dapat langsung diselesaikan:
		// tanpa langkah mengambil, tidak ada catatan siapa yang mengerjakannya.
		return ErrNotTaskOwner
	}
	done := at.UTC()
	t.CompletedAt = &done
	t.CompletionReason = action
	return nil
}

// Assignee adalah hasil aturan routing: satu orang, atau satu antrean bersama.
type Assignee struct {
	Operator   string
	Workbasket string
}

// DuplicateKey adalah satu kombinasi yang menandai klaim sebagai pendaftaran ulang atas
// kejadian yang sama.
//
// Field yang kosong TIDAK ikut diperiksa. Itu bukan kelalaian: sistem lama memang
// memeriksa lokasi pada sebagian lini saja.
type DuplicateKey struct {
	PolicyNumber  string
	InsuredItemID string
	Location      string
	CauseOfLoss   string
}

// DuplicateKeys menyusun seluruh kunci yang harus diperiksa untuk sebuah klaim.
//
// # Apa yang sebenarnya diperiksa sistem lama
//
// Dua pemeriksaan berbeda, bukan satu:
//
//   - Langkah 32 berjalan untuk SELURUH lini, satu kunci per objek: polis + objek.
//     Lokasi ikut menjadi bagian kunci HANYA bila lini bukan Personal Accident
//     (langkah 32.2 dilewati saat `IsPA`).
//   - Langkah 33 berjalan HANYA untuk Personal Accident, dan hanya untuk coverage yang
//     penyebab kerugiannya `12002`: polis + objek + lokasi + penyebab kerugian.
//
// Ini menjawab pertanyaan terbuka "Lokasi tidak diperiksa untuk PA dan Travel — sengaja?"
// pada `TKT-B02-003` hanya sebagian: untuk PA lokasi memang dikeluarkan dari kunci
// pertama, tetapi masuk lagi pada kunci kedua. Untuk Travel tidak ada perlakuan khusus
// sama sekali — ia mengikuti jalur umum, lokasi ikut diperiksa.
//
// Tanggal kejadian TIDAK menjadi bagian kunci mana pun. Akibatnya dua kejadian berbeda
// pada objek dan lokasi yang sama dianggap duplikat. Itu perilaku sistem lama, dan
// keputusan apakah ia dipertahankan masih milik Work Owner (`TKT-B02-003`).
func DuplicateKeys(k Claim) []DuplicateKey {
	var key []DuplicateKey
	for _, o := range k.InsuredItem {
		primary := DuplicateKey{PolicyNumber: k.Policy.Number, InsuredItemID: o.ID}
		if k.Policy.Line != LinePersonalAccident {
			primary.Location = k.Location
		}
		key = append(key, primary)

		if k.Policy.Line != LinePersonalAccident {
			continue
		}
		for _, c := range o.Coverage {
			if c.CauseOfLoss != CauseOfLossPA {
				continue
			}
			key = append(key, DuplicateKey{
				PolicyNumber:  k.Policy.Number,
				InsuredItemID: o.ID,
				Location:      k.Location,
				CauseOfLoss:   CauseOfLossPA,
			})
			break
		}
	}
	return key
}

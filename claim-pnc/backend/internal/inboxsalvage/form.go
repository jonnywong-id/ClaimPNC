package inboxsalvage

import (
	"strconv"
	"strings"
)

// FormMode menyatakan form Tambah sedang membuat baris baru atau mengubah yang ada.
//
// Sistem lama menyimpannya di `TempInsert.pyNote`, diisi `"Insert"` oleh
// `Data Transform/CNMShowInsertSalvage_dt-DT.xml`. Nilai itulah yang kelak menentukan
// cabang mana yang dijalankan `INSERT_SALVAGE`: ID salvage yang KOSONG berarti sisip, yang
// TERISI berarti perbarui (`Database/INSERT_SALVAGE.prc:18` dan `:31`).
type FormMode string

const (
	FormModeInsert FormMode = "insert"
	FormModeUpdate FormMode = "update"
)

// StatusOption adalah satu pilihan pada daftar "Status Salvage" di form Tambah.
//
// Isinya BUKAN karangan: `Activity/SetDataSalavage_act-Act.xml` langkah 5 menyusun halaman
// `PilihanSalvage` dengan tepat dua baris, masing-masing berpasangan label dan kode.
type StatusOption struct {
	Code  string
	Label string
}

// StatusOptions adalah isi daftar "Status Salvage", berurutan seperti di layar lama.
func StatusOptions() []StatusOption {
	return []StatusOption{
		{Code: "2", Label: "Salvage DiTerima"},
		{Code: "4", Label: "Waive Salvage"},
	}
}

// TransferStatusOnSubmit adalah nilai `STSTRANSFER` yang diterima sebuah pengajuan baru.
//
// Nilainya `3` — "ke checker" — dan ia terbaca dari komentar langkah 6
// `Activity/SetStsSalvagePNC_act-Act.xml` apa adanya:
//
//	Set Status Salvage // ASI (Ga Ke Checker = 1) // ASM (Ke Checker = 3)
//
// Artinya pengajuan yang disimpan di portal ASM LANGSUNG masuk daftar "Checker", sementara
// di portal Insurtech ia melompatinya. Modul ini baru membangun portal ASM, sehingga yang
// dipakai adalah `3`.
//
// Konsekuensi yang perlu disadari saat portal Insurtech dibangun kelak: nilainya BUKAN
// konstanta yang sama, dan menyalin berkas ini apa adanya akan mengirim pengajuan Insurtech
// ke antrean checker yang seharusnya dilewatinya.
const TransferStatusOnSubmit = "3"

// DetailItem adalah satu baris pada grid "Detail Item Salvage" di form Tambah.
//
// Keempat isiannya adalah keempat kolom yang benar-benar dibaca dari berkas CSV oleh
// `Activity/UploadDetailSalvage-Act.xml` langkah 3, tidak lebih:
//
//	.Item      -> DetailSalvage.Data(<APPEND>).Item
//	.Quantity  -> …(<LAST>).Quantity
//	.Satuan    -> …(<LAST>).Satuan
//	.REMARKS   -> …(<LAST>).REMARKS
type DetailItem struct {
	// Name — kolom **"Nama Item"** -> `DETAIL_PNC_SALVAGE.NAMABARANG`.
	Name string

	// Quantity — kolom **"Jumlah Item / Qty"** -> parameter `tTOTALHARGA`, yang procedure
	// tuliskan ke `DETAIL_PNC_SALVAGE.HARGAITEM`.
	//
	// Perhatikan nama parameternya menyebut HARGA sementara isiannya adalah JUMLAH. Itu
	// keadaan di `Database/INSERT_SALVAGE_DETAILS.prc:3`, bukan kekeliruan pembacaan;
	// nama di sini menyebut isinya (`D-19`).
	Quantity string

	// Unit — **"Satuan"** -> `DETAIL_PNC_SALVAGE.SATUAN`.
	Unit string

	// Remarks — **"Remark"** -> `DETAIL_PNC_SALVAGE.REMARK`.
	Remarks string
}

// IsEmpty menyatakan baris ini tidak memuat apa pun yang layak disimpan.
//
// Ia dipakai membuang baris kosong yang lahir dari berkas CSV ber-baris kosong di ujung —
// keadaan yang terjadi pada hampir setiap berkas yang disunting di Excel.
func (d DetailItem) IsEmpty() bool {
	return strings.TrimSpace(d.Name) == "" &&
		strings.TrimSpace(d.Quantity) == "" &&
		strings.TrimSpace(d.Unit) == "" &&
		strings.TrimSpace(d.Remarks) == ""
}

// FormInput adalah isian mentah form Tambah, belum divalidasi.
type FormInput struct {
	Mode FormMode

	// SalvageID diisi HANYA pada mode ubah. Ia yang menentukan cabang procedure.
	SalvageID string

	ClaimNo      string
	ObjectID     string
	ObjectName   string
	CoverageID   string
	CoverageName string

	InputDate     string
	SalvageType   string
	Status        string
	Location      string
	InJabodetabek bool
	Currency      string
	MinimumValue  string
	Quantity      string
	OfferValue    string
	InsuredShare  string
	Remark        string
	Email         string

	SurveyorName  string
	SurveyorPhone string
	SurveyorEmail string

	Items []DetailItem
}

// Form adalah pengajuan salvage yang sudah tervalidasi dan siap disimpan.
//
// # Kenapa nilai uang tetap berbentuk teks
//
// Karena `D-51` menetapkan nilai uang disimpan PRESISI PENUH dan hanya dibulatkan saat
// ditampilkan. Mengubahnya menjadi `float64` di sini berarti pembulatan biner terjadi
// sebelum nilainya sampai ke basis data — persis yang dilarang. Yang sudah dipastikan
// adalah BENTUKNYA: NewForm menolak isian yang bukan angka, sehingga penyimpanan menerima
// teks yang pasti dapat dibaca `NUMERIC`.
type Form struct {
	Mode      FormMode
	SalvageID string

	ClaimNo      string
	ObjectID     string
	ObjectName   string
	CoverageID   string
	CoverageName string

	InputDate     string
	SalvageType   string
	Status        string
	Location      string
	InJabodetabek bool
	Currency      string
	MinimumValue  string
	Quantity      string
	OfferValue    string
	InsuredShare  string
	Remark        string
	Email         string

	SurveyorName  string
	SurveyorPhone string
	SurveyorEmail string

	Items []DetailItem

	// TransferStatus adalah `STSTRANSFER` yang akan ditulis. Ia DITURUNKAN, bukan diketik
	// pengguna — lihat TransferStatusOnSubmit.
	TransferStatus string

	// Caller adalah identitas penyimpan. Ia menjadi `PNC_SALVAGE.PIC`.
	//
	// Sistem lama mengambilnya dari `pyWorkPage.ClaimData.UserTeknis` — PIC Teknik yang
	// tercatat pada klaimnya, bukan orang yang menekan tombol. Perbedaannya nyata pada
	// daftar "Request Balai Lelang", yang menyaring kolom yang sama.
	//
	// Di sini ia diisi PIC Teknik klaim bila penyimpanan dapat membacanya, dan jatuh ke
	// pemanggil bila tidak. Lihat catatan pada repo/sqlstore.
	Caller Caller
}

// NewForm membentuk pengajuan yang sah, atau menyatakan apa yang salah.
//
// # Yang WAJIB, dan dari mana daftarnya
//
// Sistem lama hanya memeriksa DUA isian sebelum menyimpan —
// `Activity/GCNMNewSalvage_act-Act.xml` langkah 4 dan 5, yang memasang pesan "Nama object
// harus diisi" dan "Nama coverage harus diisi". Tidak ada pemeriksaan lain di mana pun pada
// jalur itu.
//
// Ketiga pemeriksaan yang DITAMBAHKAN di sini — nomor klaim, jenis salvage, dan bentuk
// angka — menutup kegagalan yang di sistem lama sampai ke pengguna sebagai galat basis data
// mentah, bukan sebagai kalimat yang terbaca:
//
//   - Nomor klaim kosong menyimpan baris `PNC_SALVAGE` yang tidak tertaut ke klaim mana
//     pun, dan baris itu tidak akan pernah muncul di daftar mana pun karena seluruh kueri
//     menggabungkannya lewat nomor klaim.
//   - Nilai uang yang bukan angka gagal di `TO_NUMBER` procedure, dan pesannya adalah
//     `ORA-01722: invalid number`.
//
// Ia MEMPERBAIKI cara galat disampaikan, bukan mengubah hasil: isian yang sah menghasilkan
// baris yang sama persis.
func NewForm(input FormInput, caller Caller) (Form, error) {
	cleanCaller := caller.Clean()
	if cleanCaller.Login == "" {
		return Form{}, ErrCallerUnknown
	}

	mode := input.Mode
	if mode == "" {
		mode = FormModeInsert
	}

	form := Form{
		Mode:           mode,
		SalvageID:      strings.TrimSpace(input.SalvageID),
		ClaimNo:        strings.ToUpper(strings.TrimSpace(input.ClaimNo)),
		ObjectID:       strings.TrimSpace(input.ObjectID),
		ObjectName:     strings.TrimSpace(input.ObjectName),
		CoverageID:     strings.TrimSpace(input.CoverageID),
		CoverageName:   strings.TrimSpace(input.CoverageName),
		InputDate:      strings.TrimSpace(input.InputDate),
		SalvageType:    strings.TrimSpace(input.SalvageType),
		Status:         strings.TrimSpace(input.Status),
		Location:       strings.TrimSpace(input.Location),
		InJabodetabek:  input.InJabodetabek,
		Currency:       strings.TrimSpace(input.Currency),
		MinimumValue:   strings.TrimSpace(input.MinimumValue),
		Quantity:       strings.TrimSpace(input.Quantity),
		OfferValue:     strings.TrimSpace(input.OfferValue),
		InsuredShare:   strings.TrimSpace(input.InsuredShare),
		Remark:         strings.TrimSpace(input.Remark),
		Email:          strings.TrimSpace(input.Email),
		SurveyorName:   strings.TrimSpace(input.SurveyorName),
		SurveyorPhone:  strings.TrimSpace(input.SurveyorPhone),
		SurveyorEmail:  strings.TrimSpace(input.SurveyorEmail),
		TransferStatus: TransferStatusOnSubmit,
		Caller:         cleanCaller,
	}

	// Nomor klaim DIBESARKAN hurufnya, mengikuti `@toUpperCase(TempInsert.CaseID)` pada
	// langkah 2 `GCNMNewSalvage_act`. Tanpa itu, baris yang nomor klaimnya diketik huruf
	// kecil tidak akan pernah tergabung ke `T_CLAIM_PNC` — dan gabungannya berbentuk
	// perangkaian teks (`'ASM-FW-GCNMFW-WORK ' || a.noklaim`), yang peka huruf besar-kecil.
	violations := []Violation{}

	if form.ClaimNo == "" {
		violations = append(violations, Violation{
			Field:   FieldFormClaimNo,
			Message: "Nomor Klaim harus diisi.",
		})
	}

	// Kedua pesan ini ditulis PERSIS seperti di sistem lama (`D-13`).
	if form.ObjectName == "" {
		violations = append(violations, Violation{
			Field:   FieldFormObjectName,
			Message: "Nama object harus diisi",
		})
	}
	if form.CoverageName == "" {
		violations = append(violations, Violation{
			Field:   FieldFormCoverageName,
			Message: "Nama coverage harus diisi",
		})
	}

	if form.SalvageType == "" {
		violations = append(violations, Violation{
			Field:   FieldFormSalvageType,
			Message: "Jenis Salvage harus diisi.",
		})
	}

	if form.Status != "" && !knownStatus(form.Status) {
		violations = append(violations, Violation{
			Field:   FieldFormStatus,
			Message: "Status Salvage tidak dikenal.",
		})
	}

	violations = append(violations, numberViolations(form)...)

	// Mode ubah TANPA ID salvage ditolak di sini, bukan diteruskan.
	//
	// Diteruskan, ia akan jatuh ke cabang SISIP procedure dan menerbitkan baris BARU
	// alih-alih memperbarui yang ada — pengajuan ganda tanpa satu pun galat.
	if form.Mode == FormModeUpdate && form.SalvageID == "" {
		violations = append(violations, Violation{
			Field:   FieldFormClaimNo,
			Message: "Pengajuan yang diubah tidak dikenali. Muat ulang halaman lalu ulangi.",
		})
	}

	items, itemViolations := cleanItems(input.Items)
	violations = append(violations, itemViolations...)
	form.Items = items

	if len(violations) > 0 {
		return Form{}, NewValidationError(violations)
	}
	return form, nil
}

// knownStatus menyatakan kode status berasal dari daftar pilihan yang sah.
func knownStatus(code string) bool {
	for _, option := range StatusOptions() {
		if option.Code == code {
			return true
		}
	}
	return false
}

// numberViolations memeriksa bentuk keempat isian bernilai angka.
//
// Isian KOSONG diterima — sistem lama pun menerimanya, dan procedure menuliskannya sebagai
// `NULL`. Yang ditolak hanyalah isian terisi yang bukan angka.
func numberViolations(form Form) []Violation {
	checks := []struct {
		field string
		label string
		value string
	}{
		{FieldFormMinimum, "Minimum Salvage", form.MinimumValue},
		{FieldFormQuantity, "Quantity Salvage", form.Quantity},
		{"nilai_penawaran", "Nilai Penawaran", form.OfferValue},
		{"share_tertanggung", "Share Tertanggung", form.InsuredShare},
	}

	violations := []Violation{}
	for _, check := range checks {
		if check.value == "" {
			continue
		}
		if !isDecimal(check.value) {
			violations = append(violations, Violation{
				Field:   check.field,
				Message: check.label + " harus berupa angka.",
			})
		}
	}
	return violations
}

// isDecimal menyatakan teks dapat dibaca sebagai bilangan desimal.
//
// Ia memakai `strconv.ParseFloat` untuk MEMERIKSA saja; nilainya sendiri tidak pernah
// dipakai, dan yang diteruskan ke penyimpanan tetap teks aslinya (`D-51`).
func isDecimal(value string) bool {
	_, err := strconv.ParseFloat(strings.ReplaceAll(value, ",", "."), 64)
	return err == nil
}

// maxItems membatasi jumlah baris Detail Item Salvage dalam satu pengajuan.
//
// Sistem lama tidak punya batas sama sekali, dan berkas CSV yang diunggah dapat berisi
// berapa pun baris. Batas ini ADA supaya satu berkas keliru tidak menghasilkan ribuan
// pemanggilan procedure dalam satu permintaan. Angkanya dipilih jauh di atas pemakaian
// wajar; bila ia pernah tercapai dalam pemakaian nyata, ia harus dinaikkan dengan sadar,
// bukan dihapus.
const maxItems = 500

// cleanItems membuang baris kosong dan memeriksa sisanya.
func cleanItems(raw []DetailItem) ([]DetailItem, []Violation) {
	items := make([]DetailItem, 0, len(raw))
	violations := []Violation{}

	for index, item := range raw {
		if item.IsEmpty() {
			continue
		}

		clean := DetailItem{
			Name:     strings.TrimSpace(item.Name),
			Quantity: strings.TrimSpace(item.Quantity),
			Unit:     strings.TrimSpace(item.Unit),
			Remarks:  strings.TrimSpace(item.Remarks),
		}

		if clean.Name == "" {
			violations = append(violations, Violation{
				Field:   FieldFormItems,
				Message: "Nama Item pada baris " + strconv.Itoa(index+1) + " harus diisi.",
			})
		}
		if clean.Quantity != "" && !isDecimal(clean.Quantity) {
			violations = append(violations, Violation{
				Field: FieldFormItems,
				Message: "Jumlah Item pada baris " + strconv.Itoa(index+1) +
					" harus berupa angka.",
			})
		}

		items = append(items, clean)
	}

	if len(items) > maxItems {
		violations = append(violations, Violation{
			Field: FieldFormItems,
			Message: "Detail Item Salvage melebihi " + strconv.Itoa(maxItems) +
				" baris dalam satu pengajuan.",
		})
	}

	return items, violations
}

package komite

import "testing"

// Judul layar rincian komite adalah salinan tujuh sel label `Section/ShowTransfer`.
//
// Setiap kasus di bawah menyebut syarat Pega yang diwakilinya, supaya perubahan pada
// tabel ini tidak dapat dilakukan tanpa menyebut rule mana yang berubah.
func TestJudulMengikutiSyaratShowTransfer(t *testing.T) {
	t.Parallel()

	kasus := []struct {
		nama      string
		tipe      string
		bayar     string
		panel     string
		adaKomite bool
		mau       string
		alasan    string
	}{
		{
			nama: "tanpa baris komite judulnya telanjang",
			// Tanpa `TYPEKOMITE` tidak ada satu pun syarat yang dapat dinilai, dan
			// menebak akhirannya berarti mengarang. 218 dari 610 case ada di keadaan ini.
			adaKomite: false, mau: "CLAIM COMMITTEE",
			alasan: "syarat tidak dapat dinilai, jadi tidak ada akhiran",
		},
		{
			nama: "TransferType 1 memunculkan SURVEY",
			tipe: "1", adaKomite: true, panel: "006",
			mau:    "CLAIM COMMITTEE - SURVEY",
			alasan: ".TransferType =='1'",
		},
		{
			nama: "TransferType 2 memunculkan ADJUSTMENT",
			tipe: "2", bayar: "2", panel: "006", adaKomite: true,
			mau:    "CLAIM COMMITTEE - ADJUSTMENT",
			alasan: ".TransferType = 2 && !IsTravel && PaymentType != '6'",
		},
		{
			nama: "Travel MENYEMBUNYIKAN ADJUSTMENT",
			// Ini cabang yang paling mudah terlewat, dan ia ADA di data sungguhan:
			// KMT-3531 dan KMT-3689 ber-TYPEKOMITE 2, PAYMENTTYPE 2, GROUPPANEL_1 005.
			tipe: "2", bayar: "2", panel: "005", adaKomite: true,
			mau:    "CLAIM COMMITTEE",
			alasan: "!IsTravel gugur karena GroupPanel = 005",
		},
		{
			nama: "PaymentType 6 memunculkan TOLAK KLAIM dan membuang ADJUSTMENT",
			tipe: "2", bayar: "6", panel: "006", adaKomite: true,
			mau:    "CLAIM COMMITTEE - TOLAK KLAIM",
			alasan: "PaymentType != '6' gugur, syarat TOLAK KLAIM terpenuhi",
		},
		{
			nama: "TransferType 3 berbunyi REJECT, BUKAN Ex Gratia",
			// Kode yang sama diberi caption berbeda oleh dua rule. Uji ini yang menjaga
			// keduanya tidak diseragamkan diam-diam.
			tipe: "3", panel: "006", adaKomite: true,
			mau:    "CLAIM COMMITTEE - REJECT",
			alasan: ".TransferType =='3' pada ShowTransfer",
		},
		{
			nama: "TransferType 4 memunculkan LIABILITY",
			tipe: "4", panel: "006", adaKomite: true,
			mau:    "CLAIM COMMITTEE - LIABILITY",
			alasan: ".TransferType =='4'",
		},
		{
			nama: "TransferType 5 memunculkan FINAL",
			tipe: "5", panel: "006", adaKomite: true,
			mau:    "CLAIM COMMITTEE - FINAL",
			alasan: ".TransferType =='5'",
		},
		{
			nama: "dua syarat yang sama-sama benar tampil BERDUA",
			// Di Pega ketujuh label adalah sel berdampingan, bukan satu pilihan. Kode 3
			// dengan pembayaran 6 memenuhi dua syarat sekaligus, dan keduanya tampil.
			tipe: "3", bayar: "6", panel: "006", adaKomite: true,
			mau:    "CLAIM COMMITTEE - REJECT - TOLAK KLAIM",
			alasan: "label ShowTransfer digabung, bukan dipilih salah satu",
		},
		{
			nama: "kode berimbuh nol diperlakukan sama",
			tipe: "02", bayar: "06", panel: "006", adaKomite: true,
			mau:    "CLAIM COMMITTEE - TOLAK KLAIM",
			alasan: "kodeRingkas memangkas nol di depan, sama seperti paymentKind",
		},
		{
			nama: "kode yang tidak dikenal tidak memunculkan akhiran apa pun",
			tipe: "9", panel: "006", adaKomite: true,
			mau:    "CLAIM COMMITTEE",
			alasan: "tidak ada sel ShowTransfer yang menanganinya",
		},
	}

	for _, k := range kasus {
		t.Run(k.nama, func(t *testing.T) {
			t.Parallel()

			d := TransferDetail{
				HasCommitteeRecord: k.adaKomite,
				GroupPanel:         k.panel,
				Committee: CommitteeRecord{
					TransferTypeCode: k.tipe,
					PaymentTypeCode:  k.bayar,
				},
			}
			if got := d.Judul(); got != k.mau {
				t.Fatalf("judul = %q, mau %q (%s)", got, k.mau, k.alasan)
			}
		})
	}
}

// Kode mentah tidak boleh diturunkan balik dari "Tipe Komite".
//
// Kode 3 adalah buktinya: CommitteeKindOf menyebutnya "Ex Gratia" sementara judul layar
// menyebutnya "REJECT". Bila kode mentahnya dibuang, salah satu dari keduanya harus
// menebak dari hasil penurunan yang lain — dan tebakan itu akan salah.
func TestKodeMentahDanTipeKomiteBolehBerbeda(t *testing.T) {
	t.Parallel()

	d := TransferDetail{
		HasCommitteeRecord: true,
		GroupPanel:         "006",
		Committee: CommitteeRecord{
			TransferTypeCode: "3",
			Kind:             CommitteeKindOf("3", ""),
		},
	}

	if d.Committee.Kind != "Ex Gratia" {
		t.Fatalf("tipe komite = %q, mau %q", d.Committee.Kind, "Ex Gratia")
	}
	if judul := d.Judul(); judul != "CLAIM COMMITTEE - REJECT" {
		t.Fatalf("judul = %q, mau %q", judul, "CLAIM COMMITTEE - REJECT")
	}
}

package memory

import (
	"context"
	"strings"
	"time"

	"claim-pnc/internal/inboxservicecenter"
)

// FindDetail mencari satu rincian klaim.
//
// Penyaring PIC-nya ditiru persis seperti di SQL — termasuk akibatnya: klaim milik petugas
// lain dijawab ErrNotFound, sama dengan klaim yang memang tidak ada. Tanpa peniruan itu, uji
// yang lulus di sini tidak menyatakan apa pun tentang yang berjalan di Oracle.
func (s *Store) FindDetail(
	_ context.Context,
	q inboxservicecenter.DetailQuery,
) (inboxservicecenter.ClaimDetail, error) {
	for _, detail := range s.details {
		sameID := strings.EqualFold(strings.TrimSpace(detail.ID), strings.TrimSpace(q.ID))
		samePIC := strings.EqualFold(
			strings.TrimSpace(detail.TechnicalPIC),
			strings.TrimSpace(q.Caller.Login),
		)
		if sameID && samePIC {
			return detail, nil
		}
	}
	return inboxservicecenter.ClaimDetail{}, inboxservicecenter.ErrNotFound
}

// ListProgress mengambil riwayat catatan progres satu klaim.
//
// Dikunci REPAIRID, bukan ID — sama seperti kuerinya, dan itu pembedaan yang mudah tertukar
// karena keduanya sama-sama "nomor klaim" bagi mata yang membaca sekilas.
func (s *Store) ListProgress(
	_ context.Context,
	repairID string,
) ([]inboxservicecenter.ProgressNote, error) {
	clean := strings.TrimSpace(repairID)
	if clean == "" {
		return []inboxservicecenter.ProgressNote{}, nil
	}

	notes := []inboxservicecenter.ProgressNote{}
	for _, note := range s.progress {
		if strings.EqualFold(strings.TrimSpace(note.ClaimID), clean) {
			notes = append(notes, note)
		}
	}
	return notes, nil
}

// SampleDetails adalah rincian contoh, satu untuk setiap baris pada SampleClaims.
//
// Isinya karangan (`D-69`). Yang sengaja dibuat: baris pertama terisi RAPAT supaya ketujuh
// kelompok layar rincian punya isi yang terlihat, sedangkan sisanya terisi seadanya supaya
// layar yang menggambar isian kosong sebagai tanda pisah ikut teruji.
func SampleDetails() []inboxservicecenter.ClaimDetail {
	details := make([]inboxservicecenter.ClaimDetail, 0, len(SampleClaims()))

	for _, claim := range SampleClaims() {
		detail := inboxservicecenter.ClaimDetail{
			ID:                claim.ID,
			RepairID:          claim.RepairID,
			ClaimNumber:       claim.ClaimNumber,
			Type:              claim.Type,
			Owner:             claim.Owner,
			InputDate:         claim.InputDate,
			PolicyNumber:      claim.PolicyNumber,
			InsuredName:       claim.CustomerName,
			CustomerName:      claim.CustomerName,
			IMEI:              claim.IMEI,
			TechnicalPIC:      claim.TechnicalPIC,
			RepairStatus:      claim.RepairStatus,
			ApprovalStatus:    claim.ApprovalStatus,
			CommitteeApprover: claim.CommitteeApprover,
		}

		if claim.ID == "SC-000101" {
			enrich(&detail)
		}

		details = append(details, detail)
	}

	return details
}

// enrich mengisi rapat satu rincian contoh, supaya ketujuh kelompok layar punya isi.
func enrich(d *inboxservicecenter.ClaimDetail) {
	d.Insurance = "SIMAS INSURTECH"
	d.WarrantyStart = day(2026, time.January, 15)
	d.WarrantyEnd = day(2027, time.January, 14)
	d.CustomerPhone = "0800-CONTOH-01"
	d.IDCardNumber = "KTP-CONTOH-0001"
	d.PrincipalBill = "PB-CONTOH-0001"
	d.InsuranceBill = "IB-CONTOH-0001"
	d.QuotationNo = "QT-CONTOH-0001"
	d.QuotationValue = "2500000"

	d.ProductGroup = "HANDPHONE"
	d.ProductCategory = "SMARTPHONE"
	d.Brand = "Contoh Brand"
	d.Model = "Contoh Model X"
	d.Colour = "Hitam"
	d.Device = "Handphone"
	d.SerialNumber = "SN-CONTOH-0001"
	d.ItemWarranty = "Resmi"
	d.ObjectName = "Handphone Contoh"

	d.CollectPoint = "Gerai Contoh Pusat"
	d.RepairPoint = "Service Center Contoh"
	d.IsDelivery = "Y"
	d.SymptomCode = "LCD-01"
	d.SymptomDesc = "Layar tidak menyala setelah terjatuh"
	d.Analysis = "LCD perlu diganti; rangka tidak bengkok"
	d.Reason = "Kerusakan layar"
	d.CustomerReply = "Setuju diperbaiki"

	d.AcknowledgeDate = day(2026, time.September, 22)
	d.AssignedDate = day(2026, time.September, 23)
	d.EstimatedPickupAt = day(2026, time.September, 30)
	d.DownPaymentValue = "0"
	d.DownPaymentMethod = "TRANSFER"
	d.AccessoriesOther = "Kartu garansi"

	d.ServiceFee = "150000"
	d.SparepartFee = "1800000"
	d.SparepartValue = "1800000"
	d.TaxFee = "0"
	d.VAT = "214500"
	d.DeliveryFee = "50000"
	d.OtherFee = "0"
	d.Excess = "0"
	d.Deductible = "250000"
	d.TotalFee = "2214500"

	d.ChargerAdaptor = "Y"
	d.ChargerCable = "Y"
	d.CarKit = "N"
	d.RemovableAntenna = "N"
	d.Headset = "N"
	d.Battery = "Y"
	d.SimCard = "N"
	d.ExternalCover = "Y"
	d.BatteryCover = "Y"
	d.LCDText = "Retak"
	d.UnitCase = "Y"
	d.BoxUnit = "Y"
}

// SampleProgress adalah riwayat catatan progres contoh.
//
// Dikunci REPAIRID, bukan ID — persis seperti tabelnya. Hanya satu klaim yang punya riwayat,
// supaya layar yang menggambar riwayat KOSONG ikut teruji.
func SampleProgress() []inboxservicecenter.ProgressNote {
	return []inboxservicecenter.ProgressNote{
		{
			ClaimID:    "1000101",
			RecordedAt: day(2026, time.September, 22),
			Note:       "Unit diterima di gerai, kelengkapan dicek.",
			RecordedBy: SampleOwner,
		},
		{
			ClaimID:    "1000101",
			RecordedAt: day(2026, time.September, 23),
			Note:       "Pengecekan selesai, menunggu ketersediaan sparepart.",
			RecordedBy: SampleOwner,
		},
	}
}

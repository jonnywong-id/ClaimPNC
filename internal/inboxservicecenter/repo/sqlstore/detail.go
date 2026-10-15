package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"claim-pnc/internal/inboxservicecenter"
)

// detailColumns adalah ke-83 kolom yang dikembalikan find_detail.
//
// Urutannya WAJIB sama dengan urutan kolom di inboxservicecenter.sql dan dengan urutan
// pemindai scanDetail. Ia ditulis lengkap di sini pula supaya ketiga tempat itu dapat diuji
// kesesuaiannya di detail_test.go.
var detailColumns = []string{
	// General Information
	"ID", "REPAIRID", "CLAIMNO", "TYPE", "LOGIN", "INPUTDATE", "NOPOLIS", "INSURANCE",
	"STARTDATE", "ENDDATE", "CUST_NAME", "QQNAME", "NOHP", "NOKTP",
	"PRINCIPAL_BILL_NO", "INSURANCE_BILL_NO", "QUOTATIONNO", "QUOTATION_AMOUNT",
	// Informasi Unit
	"PROD_GROUP", "PROD_CATEGORY", "BRAND", "MODEL", "COLOUR", "DEVICE", "IMEI",
	"SERIALNO", "ITEMWARRANTY", "OBJECT",
	// Informasi Perbaikan
	"COLLECT_POINT", "REPAIR_POINT", "IS_DELIVERY", "SYMPTOM_CODE", "SYMPTOM_DESC",
	"ANALISA", "PIC", "STATUS", "REASON", "CANCELLED_REASON", "CUST_ARRIVAL",
	// Estimasi Date
	"ACKNOWLEDGE_DATE", "ASSIGNED_DATE", "COMPLETED_DATE", "RELEASE_DATE", "INVOICE_DATE",
	"ESTIMATED_PICKUP_DATE", "PICKUP_COURIER_DATE", "DOWNPAYMENT", "DP_NO", "DP_METHOD",
	"STS_APPROVAL", "REMARK", "KOMITEAPPROVE", "DETAILPART", "ACCESSORIESLAINYA",
	// Estimasi Biaya
	"SERVICE_FEE", "SPAREPART_FEE", "SUKUCADANG", "SUKUCADANGAPPROVE",
	"TAX_FEE", "TAX_FEEAPPROVE", "PPN", "PPNAPPROVE",
	"DELIVERY_FEE", "DELIVERY_FEEAPPROVE", "OTHER_FEE",
	"EXCESS", "EXCESSAPPROVE", "DEDUCTIBLE", "DEDUCAPPROVE",
	"TOTAL_FEE", "TOTAL_FEEAPPROVE",
	// Accessories Unit
	"DESKCHARGER", "CGARGERCABLE", "CARKIT", "REMOVABLEANTENNA", "HEADSET", "BATTERY",
	"SIMCARD", "EXTRACOVER", "BATTERYCOVER", "LCD_TEXT", "CASE", "BOXUNIT",
}

// FindDetail mengambil satu klaim beserta seluruh isian layar rinciannya.
//
// Klaim yang tidak ada dan klaim milik petugas lain dijawab SAMA, yaitu
// inboxservicecenter.ErrNotFound. Alasannya ada di komentar galat itu: membedakannya memberi
// tahu penanya bahwa sebuah ID nyata, dan ID di tabel ini berurutan.
func (r *Repo) FindDetail(
	ctx context.Context,
	q inboxservicecenter.DetailQuery,
) (inboxservicecenter.ClaimDetail, error) {
	row := r.db.QueryRowContext(
		ctx,
		query("find_detail"),
		strings.ToUpper(strings.TrimSpace(q.ID)),
		strings.ToUpper(strings.TrimSpace(q.Caller.Login)),
	)

	detail, err := scanDetail(row)
	if errors.Is(err, sql.ErrNoRows) {
		return inboxservicecenter.ClaimDetail{}, inboxservicecenter.ErrNotFound
	}
	if err != nil {
		return inboxservicecenter.ClaimDetail{}, fmt.Errorf("menjalankan kueri find_detail: %w", err)
	}
	return detail, nil
}

// ListProgress mengambil riwayat catatan progres satu klaim.
//
// Riwayat kosong bukan galat: klaim yang belum pernah dicatat progresnya memang tidak punya
// barisnya, dan itu keadaan biasa pada tab Registrasi SC.
func (r *Repo) ListProgress(
	ctx context.Context,
	repairID string,
) ([]inboxservicecenter.ProgressNote, error) {
	clean := strings.TrimSpace(repairID)
	if clean == "" {
		// Tanpa REPAIRID tidak ada yang dapat dicari. Mengirim kueri dengan kunci kosong
		// hanya menambah satu perjalanan ke basis data yang pasti nihil.
		return []inboxservicecenter.ProgressNote{}, nil
	}

	rows, err := r.db.QueryContext(ctx, query("list_progress"), clean)
	if err != nil {
		return nil, fmt.Errorf("menjalankan kueri list_progress: %w", err)
	}
	defer rows.Close()

	notes := []inboxservicecenter.ProgressNote{}
	for rows.Next() {
		var (
			claimID, note, recordedBy sql.NullString
			recordedAt                sql.NullTime
		)
		if err := rows.Scan(&claimID, &recordedAt, &note, &recordedBy); err != nil {
			return nil, fmt.Errorf("membaca baris kueri list_progress: %w", err)
		}
		notes = append(notes, inboxservicecenter.ProgressNote{
			ClaimID:    text(claimID),
			RecordedAt: timeOrNil(recordedAt),
			Note:       text(note),
			RecordedBy: text(recordedBy),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("menelusuri hasil kueri list_progress: %w", err)
	}

	return notes, nil
}

// scanDetail memindai satu baris menjadi ClaimDetail.
//
// Seluruh kolom teks dipindai lewat sql.NullString dan seluruh tanggal lewat sql.NullTime:
// tabel ini nyaris tanpa kolom NOT NULL, dan baris yang baru masuk dari portal mitra memang
// banyak kosongnya.
//
// Nilai uang dipindai sebagai TEKS, bukan angka. Alasannya ada di komentar text().
func scanDetail(row scanner) (inboxservicecenter.ClaimDetail, error) {
	var (
		// General Information
		id, repairID, claimNumber, claimType, owner      sql.NullString
		inputDate                                        sql.NullTime
		policyNumber, insurance                          sql.NullString
		warrantyStart, warrantyEnd                       sql.NullTime
		customerName, insuredName, customerPhone, idCard sql.NullString
		principalBill, insuranceBill                     sql.NullString
		quotationNo, quotationValue                      sql.NullString

		// Informasi Unit
		productGroup, productCategory, brand, model sql.NullString
		colour, device, imei, serialNumber          sql.NullString
		itemWarranty, objectName                    sql.NullString

		// Informasi Perbaikan
		collectPoint, repairPoint, isDelivery sql.NullString
		symptomCode, symptomDesc, analysis    sql.NullString
		technicalPIC, repairStatus            sql.NullString
		reason, cancelReason, customerReply   sql.NullString

		// Estimasi Date
		acknowledgeDate, assignedDate, completedDate sql.NullTime
		releaseDate, invoiceDate                     sql.NullTime
		estimatedPickupAt, courierPickupAt           sql.NullTime
		downPaymentValue, downPaymentNo              sql.NullString
		downPaymentMethod, approvalStatus            sql.NullString
		approvalRemark, committeeApprover            sql.NullString
		partDetailJSON, accessoriesOther             sql.NullString

		// Estimasi Biaya
		serviceFee, sparepartFee                   sql.NullString
		sparepartValue, sparepartValueAppr         sql.NullString
		taxFee, taxFeeAppr, vat, vatAppr           sql.NullString
		deliveryFee, deliveryFeeAppr, otherFee     sql.NullString
		excess, excessAppr, deductible, deductAppr sql.NullString
		totalFee, totalFeeAppr                     sql.NullString

		// Accessories Unit
		chargerAdaptor, chargerCable, carKit        sql.NullString
		removableAntenna, headset, battery, simCard sql.NullString
		externalCover, batteryCover, lcdText        sql.NullString
		unitCase, boxUnit                           sql.NullString
	)

	err := row.Scan(
		&id, &repairID, &claimNumber, &claimType, &owner, &inputDate,
		&policyNumber, &insurance, &warrantyStart, &warrantyEnd,
		&customerName, &insuredName, &customerPhone, &idCard,
		&principalBill, &insuranceBill, &quotationNo, &quotationValue,

		&productGroup, &productCategory, &brand, &model, &colour, &device,
		&imei, &serialNumber, &itemWarranty, &objectName,

		&collectPoint, &repairPoint, &isDelivery, &symptomCode, &symptomDesc,
		&analysis, &technicalPIC, &repairStatus, &reason, &cancelReason, &customerReply,

		&acknowledgeDate, &assignedDate, &completedDate, &releaseDate, &invoiceDate,
		&estimatedPickupAt, &courierPickupAt, &downPaymentValue, &downPaymentNo,
		&downPaymentMethod, &approvalStatus, &approvalRemark, &committeeApprover,
		&partDetailJSON, &accessoriesOther,

		&serviceFee, &sparepartFee, &sparepartValue, &sparepartValueAppr,
		&taxFee, &taxFeeAppr, &vat, &vatAppr,
		&deliveryFee, &deliveryFeeAppr, &otherFee,
		&excess, &excessAppr, &deductible, &deductAppr,
		&totalFee, &totalFeeAppr,

		&chargerAdaptor, &chargerCable, &carKit, &removableAntenna, &headset, &battery,
		&simCard, &externalCover, &batteryCover, &lcdText, &unitCase, &boxUnit,
	)
	if err != nil {
		return inboxservicecenter.ClaimDetail{}, err
	}

	return inboxservicecenter.ClaimDetail{
		ID:             text(id),
		RepairID:       text(repairID),
		ClaimNumber:    text(claimNumber),
		Type:           text(claimType),
		Owner:          text(owner),
		InputDate:      timeOrNil(inputDate),
		PolicyNumber:   text(policyNumber),
		Insurance:      text(insurance),
		WarrantyStart:  timeOrNil(warrantyStart),
		WarrantyEnd:    timeOrNil(warrantyEnd),
		CustomerName:   text(customerName),
		InsuredName:    text(insuredName),
		CustomerPhone:  text(customerPhone),
		IDCardNumber:   text(idCard),
		PrincipalBill:  text(principalBill),
		InsuranceBill:  text(insuranceBill),
		QuotationNo:    text(quotationNo),
		QuotationValue: text(quotationValue),

		ProductGroup:    text(productGroup),
		ProductCategory: text(productCategory),
		Brand:           text(brand),
		Model:           text(model),
		Colour:          text(colour),
		Device:          text(device),
		IMEI:            text(imei),
		SerialNumber:    text(serialNumber),
		ItemWarranty:    text(itemWarranty),
		ObjectName:      text(objectName),

		CollectPoint:  text(collectPoint),
		RepairPoint:   text(repairPoint),
		IsDelivery:    text(isDelivery),
		SymptomCode:   text(symptomCode),
		SymptomDesc:   text(symptomDesc),
		Analysis:      text(analysis),
		TechnicalPIC:  text(technicalPIC),
		RepairStatus:  text(repairStatus),
		Reason:        text(reason),
		CancelReason:  text(cancelReason),
		CustomerReply: text(customerReply),

		AcknowledgeDate:   timeOrNil(acknowledgeDate),
		AssignedDate:      timeOrNil(assignedDate),
		CompletedDate:     timeOrNil(completedDate),
		ReleaseDate:       timeOrNil(releaseDate),
		InvoiceDate:       timeOrNil(invoiceDate),
		EstimatedPickupAt: timeOrNil(estimatedPickupAt),
		CourierPickupAt:   timeOrNil(courierPickupAt),
		DownPaymentValue:  text(downPaymentValue),
		DownPaymentNo:     text(downPaymentNo),
		DownPaymentMethod: text(downPaymentMethod),
		ApprovalStatus:    text(approvalStatus),
		ApprovalRemark:    text(approvalRemark),
		CommitteeApprover: text(committeeApprover),
		PartDetailJSON:    text(partDetailJSON),
		AccessoriesOther:  text(accessoriesOther),

		ServiceFee:         text(serviceFee),
		SparepartFee:       text(sparepartFee),
		SparepartValue:     text(sparepartValue),
		SparepartValueAppr: text(sparepartValueAppr),
		TaxFee:             text(taxFee),
		TaxFeeAppr:         text(taxFeeAppr),
		VAT:                text(vat),
		VATAppr:            text(vatAppr),
		DeliveryFee:        text(deliveryFee),
		DeliveryFeeAppr:    text(deliveryFeeAppr),
		OtherFee:           text(otherFee),
		Excess:             text(excess),
		ExcessAppr:         text(excessAppr),
		Deductible:         text(deductible),
		DeductibleAppr:     text(deductAppr),
		TotalFee:           text(totalFee),
		TotalFeeAppr:       text(totalFeeAppr),

		ChargerAdaptor:   text(chargerAdaptor),
		ChargerCable:     text(chargerCable),
		CarKit:           text(carKit),
		RemovableAntenna: text(removableAntenna),
		Headset:          text(headset),
		Battery:          text(battery),
		SimCard:          text(simCard),
		ExternalCover:    text(externalCover),
		BatteryCover:     text(batteryCover),
		LCDText:          text(lcdText),
		UnitCase:         text(unitCase),
		BoxUnit:          text(boxUnit),
	}, nil
}

// text memangkas spasi satu kolom teks yang boleh NULL.
//
// # Kenapa nilai uang pun dibaca sebagai teks
//
// Karena DDL tabel ini belum ada (`R-08`), sehingga tipe kolomnya belum dapat dipastikan.
// Procedure-nya menerima ketujuh komponen biaya sebagai `NUMBER`, tetapi itu menyatakan tipe
// PARAMETER — bukan tipe kolomnya.
//
// Membacanya sebagai teks menampilkannya apa adanya seperti di layar lama, dan tidak
// menambahkan pembulatan yang tidak diminta siapa pun. Begitu DDL-nya tiba dan modul ini
// perlu MENGHITUNG — bukan sekadar menampilkan — nilainya dipindahkan ke tipe uang yang
// dipakai modul lain, bukan ke float (`I-12`).
func text(value sql.NullString) string {
	return strings.TrimSpace(value.String)
}

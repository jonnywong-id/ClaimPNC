package sqlstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/registrasi"
)

// Tipe PDF dan Tanggal Cetak LOD klaim Pega.
//
// Pega menyimpan `.PDFType` dan `.PrintDateLOD` pada adjustment di dokumen klaim
// (POOLDATA.JSON_KLAIM), bukan di kolomnya: `PEGA_CONVERT_JSONKLAIM_PNC.prc` tidak menyalin
// PDFType sama sekali (PDFTYPE kosong di seluruh baris Pega) dan memotong PrintDateLOD
// menjadi tanggal saja (`TO_DATE(SUBSTR(PrintDateLOD,1,8),'yyyymmdd')`, baris 1093). Tanpa
// dokumen itu, kolom Adjustment grid dan form AcceptationLOD klaim Pega selalu kosong
// padahal Pega menampilkannya. Terverifikasi 2026-09-30: klaim Pega yang dicetak hari itu
// membawa PDFType 15 dan PrintDateLOD lengkap di dokumennya.
//
// Kolom tetap didahulukan — ia yang ditulis aplikasi ini saat Print LOD. Dokumen hanya
// dipakai bila kolomnya kosong, atau bila tanggal cetaknya hanya tanggal (hasil potongan
// konversi) sementara dokumen punya waktunya.

// lodPending adalah satu baris adjustment yang Tipe PDF atau Tanggal Cetak LOD-nya belum
// lengkap dari kolom.
type lodPending struct {
	coverage                           *registrasi.Coverage
	index                              int
	objectID, coverageID, adjustmentID string
}

// lodDocument adalah bagian dokumen klaim yang dibaca: ObjectList → ObjectCoverageList →
// AdjustmentList, dengan kunci seperti procedure konversinya (ObjectID, CoverageID,
// pxListSubscript atau urutan + 1).
type lodDocument struct {
	ObjectList []struct {
		ObjectID           any `json:"ObjectID"`
		ObjectCoverageList []struct {
			CoverageID     any `json:"CoverageID"`
			AdjustmentList []struct {
				ListSubscript any `json:"pxListSubscript"`
				PDFType       any `json:"PDFType"`
				PrintDateLOD  any `json:"PrintDateLOD"`
			} `json:"AdjustmentList"`
		} `json:"ObjectCoverageList"`
	} `json:"ObjectList"`
}

type lodFact struct {
	lodType   string
	printedAt time.Time
}

// needsDocumentLOD menyatakan baris perlu dilengkapi dari dokumen klaim.
func needsDocumentLOD(line registrasi.SettlementLine) bool {
	return line.Acceptance.LODType == "" || line.Acceptance.Form.PrintDate.IsZero() ||
		dateOnlyWIB(line.Acceptance.Form.PrintDate)
}

// fillLODFromDocument melengkapi baris dari dokumen JSON_KLAIM terbaru klaim itu.
func fillLODFromDocument(ctx context.Context, exec executor, claimID string, pending []lodPending) error {
	if len(pending) == 0 {
		return nil
	}
	var blob []byte
	var text sql.NullString
	err := exec.QueryRowContext(ctx, loadQuery("adjustment_lod_dokumen"), claimID).Scan(&blob, &text)
	if errors.Is(err, sql.ErrNoRows) {
		return nil // klaim aplikasi ini: tidak punya dokumen Pega
	}
	if err != nil {
		return fmt.Errorf("registrasi/sqlstore: membaca dokumen klaim untuk Tipe PDF: %w", err)
	}
	body := blob
	if len(body) == 0 {
		body = []byte(text.String)
	}
	facts, ok := parseLODDocument(body)
	if !ok {
		return nil // dokumen tidak terbaca: kolomnya tetap dipakai apa adanya
	}
	for _, p := range pending {
		f, found := facts[lodKey(p.objectID, p.coverageID, p.adjustmentID)]
		if !found {
			continue
		}
		a := &p.coverage.Settlement[p.index].Acceptance
		if a.LODType == "" {
			a.LODType = f.lodType
		}
		if !f.printedAt.IsZero() && (a.Form.PrintDate.IsZero() || dateOnlyWIB(a.Form.PrintDate)) {
			a.Form.PrintDate = f.printedAt
		}
	}
	return nil
}

// parseLODDocument membaca PDFType dan PrintDateLOD setiap adjustment dokumen klaim.
func parseLODDocument(body []byte) (map[string]lodFact, bool) {
	var doc lodDocument
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil, false
	}
	out := map[string]lodFact{}
	for _, o := range doc.ObjectList {
		for _, c := range o.ObjectCoverageList {
			for n, a := range c.AdjustmentList {
				adjustmentID := docText(a.ListSubscript)
				if adjustmentID == "" {
					adjustmentID = strconv.Itoa(n + 1)
				}
				out[lodKey(docText(o.ObjectID), docText(c.CoverageID), adjustmentID)] = lodFact{
					lodType:   docText(a.PDFType),
					printedAt: parsePegaDateTime(docText(a.PrintDateLOD)),
				}
			}
		}
	}
	return out, true
}

func lodKey(objectID, coverageID, adjustmentID string) string {
	return strings.TrimSpace(objectID) + "/" + strings.TrimSpace(coverageID) + "/" + strings.TrimSpace(adjustmentID)
}

// docText membaca nilai dokumen Pega yang dapat berupa teks maupun angka.
func docText(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(t)
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	default:
		return strings.TrimSpace(fmt.Sprint(t))
	}
}

// parsePegaDateTime membaca DateTime Pega ("20260930T005959.123 GMT", GMT).
func parsePegaDateTime(s string) time.Time {
	s = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(s), "GMT"))
	for _, layout := range []string{"20060102T150405.000", "20060102T150405", "20060102"} {
		if t, err := time.ParseInLocation(layout, s, time.UTC); err == nil {
			return t
		}
	}
	return time.Time{}
}

// dateOnlyWIB menyatakan waktu tepat tengah malam WIB — tanggal tanpa jam dari kolom DATE.
func dateOnlyWIB(t time.Time) bool {
	if t.IsZero() {
		return false
	}
	w := t.In(clock.ZoneWIB)
	return w.Hour() == 0 && w.Minute() == 0 && w.Second() == 0
}

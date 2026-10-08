package sqlstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"claim-pnc/internal/registrasi"
)

// cifDocument adalah bagian Policy.CIFData yang dibaca blok Alamat · Telephone dan Email.
type cifDocument struct {
	CustomerC struct {
		IDCard string `json:"ASMIDCard"`
	} `json:"Customer_C"`
	CustomerP struct {
		IDCard string `json:"ASMIDCard"`
	} `json:"Customer_P"`
	AddressList []struct {
		Type         string `json:"ASMAddressType"`
		Address      string `json:"ASMAddress"`
		City         string `json:"ASMCity"`
		CityName     string `json:"CityName"`
		District     string `json:"ASMDistrict"`
		DistrictName string `json:"DistrictName"`
		RW           string `json:"ASMRW"`
		RWNote       string `json:"ASMRWNote"`
		ZipCode      string `json:"ASMZipCode"`
		Telfax       []struct {
			Type      string `json:"TelfaxType"`
			Code      string `json:"TelFaxCode"`
			Number    string `json:"TelfaxNumber"`
			Extension string `json:"TelFaxExt"`
		} `json:"ASMTelfax"`
	} `json:"AddressList"`
}

// InsuredProfile membaca CIF tertanggung dari dokumen polis. Polis tanpa dokumen menghasilkan
// profil kosong, bukan galat — blok itu sekadar tampil kosong.
func (r *PolicyRepo) InsuredProfile(ctx context.Context, policyNumber, prodKe string) (registrasi.InsuredProfile, error) {
	var raw sql.NullString
	err := r.db.QueryRowContext(ctx, loadQuery("polis_cif"), strings.TrimSpace(policyNumber), strings.TrimSpace(prodKe)).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && strings.TrimSpace(raw.String) == "") {
		return registrasi.InsuredProfile{}, nil
	}
	if err != nil {
		return registrasi.InsuredProfile{}, fmt.Errorf("registrasi/sqlstore: membaca CIF polis %q: %w", policyNumber, err)
	}
	return parseInsuredProfile([]byte(raw.String))
}

func parseInsuredProfile(body []byte) (registrasi.InsuredProfile, error) {
	var doc cifDocument
	if err := json.Unmarshal(body, &doc); err != nil {
		return registrasi.InsuredProfile{}, fmt.Errorf("registrasi/sqlstore: CIF polis tidak terbaca: %w", err)
	}
	p := registrasi.InsuredProfile{IDCard: strings.TrimSpace(doc.CustomerC.IDCard)}
	if p.IDCard == "" {
		p.IDCard = strings.TrimSpace(doc.CustomerP.IDCard)
	}
	for _, a := range doc.AddressList {
		addr := registrasi.InsuredAddress{
			Type: strings.TrimSpace(a.Type), TypeName: registrasi.AddressTypeName(a.Type),
			Address: strings.TrimSpace(a.Address),
			City:    strings.TrimSpace(a.City), CityName: strings.TrimSpace(a.CityName),
			District: strings.TrimSpace(a.District), DistrictName: strings.TrimSpace(a.DistrictName),
			RW: strings.TrimSpace(a.RW), RWName: strings.TrimSpace(a.RWNote),
			ZipCode: strings.TrimSpace(a.ZipCode),
		}
		for _, t := range a.Telfax {
			number := strings.TrimSpace(t.Number)
			addr.Phones = append(addr.Phones, registrasi.InsuredPhone{
				Type: strings.TrimSpace(t.Type), TypeName: registrasi.PhoneTypeName(t.Type, number),
				Code: strings.TrimSpace(t.Code), Number: number, Extension: strings.TrimSpace(t.Extension),
			})
		}
		p.Addresses = append(p.Addresses, addr)
	}
	return p, nil
}

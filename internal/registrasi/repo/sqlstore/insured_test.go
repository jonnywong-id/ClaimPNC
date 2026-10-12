package sqlstore

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/registrasi"
)

// CIF polis: KTP perusahaan didahulukan, alamat dan telepon diberi label. Data KARANGAN (D-69).
func TestPolicyRepoInsuredProfile(t *testing.T) {
	db, mock := be4DB(t)
	r := NewPolicyRepo(db)
	ctx := context.Background()

	doc := `{"Customer_C":{"ASMIDCard":""},"Customer_P":{"ASMIDCard":" 3171 "},
	  "AddressList":[{"ASMAddressType":"1","ASMAddress":" Jl. Contoh ","ASMCity":"C1","CityName":"KOTA A",
	  "ASMDistrict":"D1","DistrictName":"KEC A","ASMRW":"R1","ASMRWNote":"KEL A","ASMZipCode":"10000",
	  "ASMTelfax":[{"TelfaxType":"1","TelFaxCode":"","TelfaxNumber":"0811","TelFaxExt":""},
	               {"TelfaxType":"","TelfaxNumber":"x@contoh.example"}]}]}`
	mock.ExpectQuery(be4Q("polis_cif")).WithArgs("POL", "1").WillReturnRows(sqlmock.NewRows(be4Cols(1)).AddRow(doc))
	p, err := r.InsuredProfile(ctx, " POL ", " 1 ")
	require.NoError(t, err)
	require.Equal(t, "3171", p.IDCard)
	require.Len(t, p.Addresses, 1)
	a := p.Addresses[0]
	require.Equal(t, "Alamat Rumah", a.TypeName)
	require.Equal(t, "Jl. Contoh", a.Address)
	require.Equal(t, "KEL A", a.RWName)
	require.Equal(t, []registrasi.InsuredPhone{
		{Type: "1", TypeName: "Telepon Biasa", Number: "0811"},
		{TypeName: "Email", Number: "x@contoh.example"},
	}, a.Phones)

	// Tanpa dokumen, atau dokumen tanpa CIF: profil kosong.
	mock.ExpectQuery(be4Q("polis_cif")).WillReturnRows(sqlmock.NewRows(be4Cols(1)))
	p, err = r.InsuredProfile(ctx, "POL", "1")
	require.NoError(t, err)
	require.Equal(t, registrasi.InsuredProfile{}, p)
	mock.ExpectQuery(be4Q("polis_cif")).WillReturnRows(sqlmock.NewRows(be4Cols(1)).AddRow(nil))
	p, err = r.InsuredProfile(ctx, "POL", "1")
	require.NoError(t, err)
	require.Equal(t, registrasi.InsuredProfile{}, p)

	mock.ExpectQuery(be4Q("polis_cif")).WillReturnError(be4Boom)
	_, err = r.InsuredProfile(ctx, "POL", "1")
	require.ErrorIs(t, err, be4Boom)
	mock.ExpectQuery(be4Q("polis_cif")).WillReturnRows(sqlmock.NewRows(be4Cols(1)).AddRow("{rusak"))
	_, err = r.InsuredProfile(ctx, "POL", "1")
	require.ErrorContains(t, err, "CIF polis tidak terbaca")
	require.NoError(t, mock.ExpectationsWereMet())
}

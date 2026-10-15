package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	kc "claim-pnc/internal/konversicoverage"
)

type fakeSource map[string][]kc.ObjectRow // kunci: tabel objek + "|" + polis

func (f fakeSource) Objects(_ context.Context, line kc.Line, policyNo string) ([]kc.ObjectRow, error) {
	if policyNo == "RUSAK" {
		return nil, errors.New("ORA-12541")
	}
	return f[line.ObjectTable+"|"+policyNo], nil
}

type fakeTarget struct {
	batches []Batch
	commits []bool
}

func (f *fakeTarget) Apply(_ context.Context, batch Batch, commit bool) (Applied, error) {
	f.batches = append(f.batches, batch)
	f.commits = append(f.commits, commit)
	return Applied{
		ObjectsUpdated:   len(batch.Lines[0].Objects),
		CoverageInserted: len(batch.Lines[0].Coverage),
		SpreadInserted:   len(batch.Spreading),
	}, nil
}

func TestSetiapPolisDilaporkanTerpisah(t *testing.T) {
	source := fakeSource{
		"T_CARGOLIST|C1": {{IDPega: "X", PolicyNo: "C1", ProdKe: "0", IndexObject: "1",
			Document: []byte(`{"CoverageList":[{"Coverage":"ICC-A","SpreadingList":[{"TreatyType":"1"}]}]}`)}},
	}
	target := &fakeTarget{}
	report, err := New(source, target, func() time.Time { return time.Unix(0, 0) }).
		Run(context.Background(), []string{"C1", "TIDAKADA", "RUSAK"}, false)
	require.NoError(t, err)

	require.True(t, report.UjiCoba)
	require.Equal(t, []bool{false}, target.commits, "uji coba tidak meng-commit")
	require.Equal(t, 1, report.Berhasil)
	require.Equal(t, 1, report.TidakAda)
	require.Equal(t, 1, report.Gagal)
	require.Equal(t, []string{"T_CARGOLIST"}, report.Polis[0].Lini)
	require.Equal(t, 1, report.Polis[0].CoverageDitulis)
	require.Equal(t, 1, report.Polis[0].SpreadingDitulis)
	require.Contains(t, report.Polis[2].Galat, "ORA-12541")
}

func TestDaftarKosongDitolak(t *testing.T) {
	_, err := New(fakeSource{}, &fakeTarget{}, nil).Run(context.Background(), nil, true)
	require.ErrorIs(t, err, ErrEmptyList)
}

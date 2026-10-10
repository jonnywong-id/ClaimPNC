package inboxinvestigator_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxinvestigator"
)

func day(d int) time.Time { return time.Date(2026, 9, d, 0, 0, 0, 0, time.UTC) }

// Ketiga kendali ekspor diperiksa, dan setiap penolakan menyebut isian yang salah.
//
// Pesannya diuji karena ia yang sampai ke pengguna. Penyaring yang ditolak dengan kalimat
// yang tidak menyebut isian mana akan membuat orang mencoba berulang kali tanpa tahu apa
// yang harus diubah — persis keadaan yang membuat berkas kosong sistem lama membingungkan.
func TestExportFilterValidate(t *testing.T) {
	for name, probe := range map[string]struct {
		filter   inboxinvestigator.ExportFilter
		wantErr  bool
		mentions string
	}{
		"lengkap dan sah": {
			filter: inboxinvestigator.ExportFilter{
				From: day(1), To: day(30),
				Investigated: inboxinvestigator.InvestigatedYes,
			},
		},
		"rentang satu hari": {
			filter: inboxinvestigator.ExportFilter{
				From: day(5), To: day(5),
				Investigated: inboxinvestigator.InvestigatedNo,
			},
		},
		"tanpa tanggal Dari": {
			filter: inboxinvestigator.ExportFilter{
				To: day(30), Investigated: inboxinvestigator.InvestigatedYes,
			},
			wantErr: true, mentions: "Dari",
		},
		"tanpa tanggal Sampai": {
			filter: inboxinvestigator.ExportFilter{
				From: day(1), Investigated: inboxinvestigator.InvestigatedYes,
			},
			wantErr: true, mentions: "Sampai",
		},
		"rentang terbalik": {
			filter: inboxinvestigator.ExportFilter{
				From: day(30), To: day(1),
				Investigated: inboxinvestigator.InvestigatedYes,
			},
			wantErr: true, mentions: "lebih awal",
		},
		"tanpa pilihan investigasi": {
			filter:  inboxinvestigator.ExportFilter{From: day(1), To: day(30)},
			wantErr: true, mentions: "Pilih Investigation",
		},
		"pilihan investigasi tidak dikenal": {
			filter: inboxinvestigator.ExportFilter{
				From: day(1), To: day(30), Investigated: "2",
			},
			wantErr: true, mentions: "Pilih Investigation",
		},
	} {
		t.Run(name, func(t *testing.T) {
			err := probe.filter.Validate()
			if !probe.wantErr {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, inboxinvestigator.ErrExportFilterInvalid)
			require.Contains(t, err.Error(), probe.mentions)
		})
	}
}

// Jenis rumah sakit diterjemahkan persis seperti `@if` sistem lama.
//
// Yang paling penting di sini adalah baris KOSONG: `@if(SelectRS=="1", …, …)` menjawab
// "NON Rumah Sakit" untuk nilai apa pun selain "1", termasuk belum dijawab. Itu
// direplikasi apa adanya (`P-5`), dan uji ini yang mencegah seseorang "memperbaikinya"
// menjadi sel kosong di kemudian hari.
func TestHospitalKindLabel(t *testing.T) {
	for code, want := range map[string]string{
		"1":    "Rumah Sakit",
		"0":    "NON Rumah Sakit",
		"":     "NON Rumah Sakit",
		"ya":   "NON Rumah Sakit",
		" 1":   "NON Rumah Sakit",
		"TRUE": "NON Rumah Sakit",
	} {
		t.Run("kode "+code, func(t *testing.T) {
			row := inboxinvestigator.ExportRow{HospitalKindCode: code}
			require.Equal(t, want, row.HospitalKindLabel())
		})
	}
}

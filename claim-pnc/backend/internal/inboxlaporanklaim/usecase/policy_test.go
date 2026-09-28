package usecase_test

import (
	"context"
	"errors"
	"testing"

	"claim-pnc/internal/inboxlaporanklaim"
)

// Nomor ditulis bertitik dan berhuruf kecil; activity lama membuang titiknya dan
// membesarkan hurufnya sebelum mencari.
func TestPolicyLookupFillsTheFormFromTheLatestPolicy(t *testing.T) {
	service, _ := newService(t)

	got, err := service.LookupPolicy(context.Background(), portalAlias, " 1260.0000.0000.01 ")
	if err != nil {
		t.Fatalf("mencari polis: %v", err)
	}
	if got.Number != "12600000000001" || !got.Found {
		t.Fatalf("nomor/temuan = %q/%v", got.Number, got.Found)
	}
	if got.Policy.InsuredName != "PT CONTOH SEJAHTERA" || got.Policy.BusinessName != "ALL RISK" ||
		got.Policy.ReferenceNumber != "REF-0001" {
		t.Fatalf("isian polis = %+v", got.Policy)
	}
	if len(got.Notice) != 0 {
		t.Fatalf("polis PNC biasa tidak boleh berpesan: %+v", got.Notice)
	}
}

func TestPolicyLookupNotices(t *testing.T) {
	service, _ := newService(t)

	for _, tc := range []struct {
		name     string
		number   string
		code     string
		blocking bool
	}{
		{"tidak ditemukan", "99999999999999", inboxlaporanklaim.PolicyNotFound, false},
		{"syariah", "12600000000002", inboxlaporanklaim.PolicySyariah, true},
		{"bukan PNC", "12600000000003", inboxlaporanklaim.PolicyNotPNC, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := service.LookupPolicy(context.Background(), portalAlias, tc.number)
			if err != nil {
				t.Fatalf("mencari polis: %v", err)
			}
			if len(got.Notice) != 1 || got.Notice[0].Code != tc.code || got.Notice[0].Blocking != tc.blocking {
				t.Fatalf("pesan = %+v, ingin %s memblokir=%v", got.Notice, tc.code, tc.blocking)
			}
		})
	}
}

// Tombol yang dimatikan di layar tidak cukup: permintaan yang tidak lewat layar pun
// harus tertolak.
func TestSaveRejectsBlockedPolicies(t *testing.T) {
	service, _ := newService(t)
	created, err := service.Create(context.Background(), portalAlias, adminJakarta)
	if err != nil {
		t.Fatalf("membuat berkas: %v", err)
	}

	for _, number := range []string{"12600000000002", "12600000000003"} {
		_, err := service.Save(context.Background(), portalAlias, created.ID, adminJakarta,
			inboxlaporanklaim.Detail{PolicyNumber: number})
		var validation *inboxlaporanklaim.ValidationError
		if !errors.As(err, &validation) || validation.Violation[0].Field != "nomor_polis" {
			t.Fatalf("polis %s: galat = %v, ingin penolakan pada nomor_polis", number, err)
		}
	}

	// Polis yang belum terbit tetap boleh dilaporkan.
	if _, err := service.Save(context.Background(), portalAlias, created.ID, adminJakarta,
		inboxlaporanklaim.Detail{PolicyNumber: "99999999999999"}); err != nil {
		t.Fatalf("polis tidak ditemukan tidak boleh ditolak: %v", err)
	}
}

package db

import (
	"errors"
	"fmt"
	"testing"
)

func TestObjekYangBelumAdaDikenaliDariKodeOracle(t *testing.T) {
	for nama, galat := range map[string]error{
		"tabel":     errors.New("ORA-00942: table or view does not exist"),
		"sequence":  errors.New("ORA-02289: sequence does not exist"),
		"procedure": errors.New("ORA-04043: object POOLDATA.X does not exist"),
	} {
		if !IsMissingObject(galat) {
			t.Errorf("%s: seharusnya dikenali sebagai objek yang belum ada", nama)
		}
	}
}

// Galat yang dibungkus tetap dikenali. Adapter membungkus galatnya dengan konteks
// sebelum meneruskannya, sehingga pemeriksaan yang hanya mengenali galat telanjang
// akan berhenti bekerja justru pada bentuk yang sebenarnya dipakai.
func TestGalatYangDibungkusTetapDikenali(t *testing.T) {
	dibungkus := fmt.Errorf("menyisipkan %q: %w", "RCVN.26.1",
		errors.New("ORA-00942: table or view does not exist"))
	if !IsMissingObject(dibungkus) {
		t.Fatal("galat yang dibungkus seharusnya tetap dikenali")
	}
}

// Kegagalan lain TIDAK boleh ikut terbaca sebagai "migrasi belum jalan".
//
// Ini sisi yang lebih penting daripada sisi positifnya: salah mengenali akan menyuruh
// DBA menjalankan migrasi yang sudah jalan, lalu sebab yang sebenarnya — hak akses,
// koneksi putus, bind yang salah — dicari paling akhir.
func TestKegagalanLainTidakIkutTerbaca(t *testing.T) {
	for nama, galat := range map[string]error{
		"nil":                 nil,
		"hak akses kurang":    errors.New("ORA-01031: insufficient privileges"),
		"bind kurang":         errors.New("ORA-01008: not all variables bound"),
		"kolom tidak ada":     errors.New("ORA-00904: invalid identifier"),
		"koneksi putus":       errors.New("ORA-03113: end-of-file on communication channel"),
		"listener tidak tahu": errors.New("ORA-12514: listener does not currently know of service"),
	} {
		if IsMissingObject(galat) {
			t.Errorf("%s: tidak boleh terbaca sebagai objek yang belum ada", nama)
		}
	}
}

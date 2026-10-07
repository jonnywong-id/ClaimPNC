// Package tabletext mengurai tabel teks berkolom `|` menjadi irisan struct.
//
// Daftar kolom layar, isian rincian, dan tab inbox adalah DATA: puluhan baris berbentuk sama
// yang hanya berbeda isinya. Menuliskannya sebagai literal struct Go membuat setiap baris
// mengulang nama field yang sama; menuliskannya sebagai tabel membuat isinya terbaca
// sekaligus, dan catatan per baris tetap dapat ditulis sebagai baris berawalan `//` di dalam
// tabel itu sendiri.
//
// Bentuk tabelnya:
//
//	Key       | Title       | Blocked
//	id_master | Treaty ID   |
//	// catatan untuk baris di bawahnya
//	ri_type   | R/I Type    | true
//
// Baris pertama yang bukan catatan adalah judul: nama field struct, persis. Sel kosong berarti
// nilai nol field itu. Field yang didukung hanyalah teks (termasuk tipe bernama berdasar
// string), bool ("true" atau kosong), dan bilangan bulat — tabel yang tidak sesuai membuat
// Rows panik saat paket dimuat, bukan menghasilkan data yang diam-diam salah.
package tabletext

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// Rows mengurai table menjadi irisan T, berurutan sesuai baris tabel.
func Rows[T any](table string) []T {
	var header []int
	var names []string
	rows := []T{}
	typ := reflect.TypeOf((*T)(nil)).Elem()
	if typ.Kind() != reflect.Struct {
		panic(fmt.Sprintf("tabletext: %s bukan struct", typ))
	}
	for number, line := range strings.Split(table, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		cells := strings.Split(line, "|")
		for i := range cells {
			cells[i] = strings.TrimSpace(cells[i])
		}
		if header == nil {
			for _, name := range cells {
				field, ok := typ.FieldByName(name)
				if !ok || !field.IsExported() || len(field.Index) != 1 {
					panic(fmt.Sprintf("tabletext: %s tidak punya field %q", typ, name))
				}
				header = append(header, field.Index[0])
				names = append(names, name)
			}
			continue
		}
		if len(cells) != len(header) {
			panic(fmt.Sprintf("tabletext: baris %d berisi %d sel, judulnya %d", number+1, len(cells), len(header)))
		}
		var row T
		value := reflect.ValueOf(&row).Elem()
		for i, text := range cells {
			if err := set(value.Field(header[i]), text); err != nil {
				panic(fmt.Sprintf("tabletext: baris %d kolom %s: %v", number+1, names[i], err))
			}
		}
		rows = append(rows, row)
	}
	return rows
}

func set(field reflect.Value, text string) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(text)
	case reflect.Bool:
		switch text {
		case "true":
			field.SetBool(true)
		case "":
		default:
			return fmt.Errorf("nilai bool %q", text)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if text == "" {
			return nil
		}
		n, err := strconv.ParseInt(text, 10, 64)
		if err != nil {
			return err
		}
		field.SetInt(n)
	default:
		return fmt.Errorf("jenis field %s tidak didukung", field.Kind())
	}
	return nil
}

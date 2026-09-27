package main

import (
	"fmt"

	"claim-pnc/internal/inboxsalvage"
)

func main() {
	fmt.Println("== baris pencacah ==")
	for i, r := range inboxsalvage.CountRows() {
		fmt.Printf("%d. %s
", i+1, r.Label)
	}
	fmt.Println("== daftar ==")
	for i, t := range inboxsalvage.Tabs() {
		fmt.Printf("%d. %-24s %s
", i+1, t.Code, t.Name)
	}
}

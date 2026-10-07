package inboxautoclaimhttp

import (
	"context"

	"claim-pnc/internal/inboxautoclaim/usecase"
	"claim-pnc/internal/platform/httpkit"
)

// Caller menyebut pemanggil yang sudah terautentikasi.
//
// Ia dipakai mengisi kolom USERINPUT pada baris yang diunggah — kolom yang menjadi
// "User Upload" di grid. Bentuknya sempit dengan sengaja: modul ini hanya butuh tahu
// SIAPA yang mengunggah, bukan seluruh profil pengguna.
type Caller struct {
	// Login adalah nama pengguna yang diketik saat masuk, bukan NIK.
	//
	// Yang dipakai adalah Login karena itulah yang tertulis di kolom USERINPUT pada baris
	// lama: RDB List/GetHasilAutoClaim-SQL.xml menyaringnya dengan
	// {OperatorID.pyUserIdentifier}, yaitu identitas operator Pega — padanan terdekatnya
	// di sistem baru adalah login, bukan NIK.
	Login string
}

// Handler melayani permintaan Inbox Auto Claim.
type Handler struct {
	httpkit.Master[*usecase.Service, func(context.Context) (Caller, bool)]
}

// Options adalah bahan pembentuk Handler.
//
// Caller adalah jembatan SATU ARAH dari modul auth. Ia disuntikkan dari cmd, bukan
// diimpor di sini, supaya kedua modul tetap tidak saling mengimpor.
// WriteResponse dan WriteError disuntikkan dari cmd, bukan diimpor dari modul auth.
type Options = httpkit.MasterOptions[*usecase.Service, func(context.Context) (Caller, bool)]

// NewHandler membentuk handler modul Inbox Auto Claim.
func NewHandler(o Options) (*Handler, error) {
	base, err := httpkit.NewMaster("inboxautoclaim/http", o)
	if err != nil {
		return nil, err
	}
	return &Handler{Master: base}, nil
}

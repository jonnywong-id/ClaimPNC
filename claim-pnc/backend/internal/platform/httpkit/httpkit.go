// Package httpkit memuat kerangka handler HTTP yang dipakai bersama banyak modul: field yang
// dibawa setiap Handler, bentuk Options-nya, dan perakitannya.
//
// Setiap modul tetap memiliki tipe Handler dan Options-nya sendiri. Options modul adalah ALIAS
// ke salah satu bentuk di sini, sehingga literal `Options{Service: …, Logger: …}` di cmd dan
// di test tetap berlaku apa adanya; Handler modul MENANAM salah satu kerangka di sini, sehingga
// method modul tetap membaca h.Service, h.Logger, dan seterusnya.
package httpkit

import (
	"context"
	"errors"
	"log/slog"
	"reflect"
	"time"

	"claim-pnc/internal/platform/apierror"
	"claim-pnc/internal/platform/clock"
)

// Caller adalah pemanggil yang sudah terverifikasi sesinya, sebagaimana dibaca modul.
type Caller struct {
	Login string
}

// CallerReader mengambil identitas pemanggil dari konteks permintaan; false bila tidak ada.
// Ia jembatan SATU ARAH dari modul auth yang disuntikkan dari cmd, sehingga modul tidak
// mengimpor lapisan transport modul auth.
type CallerReader func(ctx context.Context) (Caller, bool)

// InboxOptions adalah bahan perakitan handler inbox.
type InboxOptions[S, C any] struct {
	Service   S
	GetCaller C
	Logger    *slog.Logger
	WriteJSON apierror.JSONWriter
	// FallbackErrorWriter menulis galat yang tidak dikenali pemetaan modul.
	FallbackErrorWriter apierror.ErrorWriter
}

// Inbox adalah field yang dibawa setiap handler inbox.
type Inbox[S, C any] struct {
	Service    S
	Caller     C
	Logger     *slog.Logger
	WriteJSON  apierror.JSONWriter
	WriteError apierror.ErrorWriter
}

// ErrorWriterFactory membangun penulis galat modul dari logger, penulis JSON, dan cadangannya
// — bentuk fungsi WriteError setiap modul.
type ErrorWriterFactory func(logger *slog.Logger, writeJSON apierror.JSONWriter, fallback apierror.ErrorWriter) apierror.ErrorWriter

// NewInbox merakit kerangka handler inbox; penulis galatnya dibangun lewat writeError modul.
func NewInbox[S, C any](o InboxOptions[S, C], writeError ErrorWriterFactory) Inbox[S, C] {
	return Inbox[S, C]{
		Service:    o.Service,
		Caller:     o.GetCaller,
		Logger:     o.Logger,
		WriteJSON:  o.WriteJSON,
		WriteError: writeError(o.Logger, o.WriteJSON, o.FallbackErrorWriter),
	}
}

// MasterOptions adalah bahan perakitan handler master yang mencatat pemanggilnya.
type MasterOptions[S, C any] struct {
	Service S
	Caller  C
	Logger  *slog.Logger

	// WriteResponse dan WriteError disuntikkan dari cmd, bukan diimpor dari modul auth.
	WriteResponse apierror.JSONWriter
	WriteError    apierror.ErrorWriter
}

// Master adalah field yang dibawa setiap handler master yang mencatat pemanggilnya.
type Master[S, C any] struct {
	Service       S
	Caller        C
	Logger        *slog.Logger
	WriteResponse apierror.JSONWriter
	WriteError    apierror.ErrorWriter
}

// NewMaster merakit kerangka handler master. Ia menolak perakitan yang tidak lengkap dengan
// pesan berawalan module, misalnya "masterbengkel/http".
func NewMaster[S, C any](module string, o MasterOptions[S, C]) (Master[S, C], error) {
	if isNil(o.Service) {
		return Master[S, C]{}, errors.New(module + ": Service wajib diisi")
	}
	if isNil(o.Caller) {
		return Master[S, C]{}, errors.New(module + ": Caller wajib diisi")
	}
	if o.WriteResponse == nil || o.WriteError == nil {
		return Master[S, C]{}, errors.New(module + ": WriteResponse dan WriteError wajib diisi")
	}
	return Master[S, C]{
		Service:       o.Service,
		Caller:        o.Caller,
		Logger:        o.Logger,
		WriteResponse: o.WriteResponse,
		WriteError:    o.WriteError,
	}, nil
}

// BasicOptions adalah bahan perakitan handler master yang tidak mencatat pemanggilnya.
type BasicOptions[S any] struct {
	Service S
	Logger  *slog.Logger

	// WriteResponse dan WriteError disuntikkan dari cmd, bukan diimpor dari modul auth.
	WriteResponse apierror.JSONWriter
	WriteError    apierror.ErrorWriter
}

// Basic adalah field yang dibawa handler master yang tidak mencatat pemanggilnya.
type Basic[S any] struct {
	Service       S
	Logger        *slog.Logger
	WriteResponse apierror.JSONWriter
	WriteError    apierror.ErrorWriter
}

// NewBasic merakit kerangka handler master tanpa pemanggil.
func NewBasic[S any](module string, o BasicOptions[S]) (Basic[S], error) {
	if isNil(o.Service) {
		return Basic[S]{}, errors.New(module + ": Service wajib diisi")
	}
	if o.WriteResponse == nil || o.WriteError == nil {
		return Basic[S]{}, errors.New(module + ": WriteResponse dan WriteError wajib diisi")
	}
	return Basic[S]{
		Service:       o.Service,
		Logger:        o.Logger,
		WriteResponse: o.WriteResponse,
		WriteError:    o.WriteError,
	}, nil
}

// isNil berlaku untuk nilai yang dapat nil (pointer, fungsi, antarmuka, peta, irisan, saluran):
// pointer bertipe yang nil pun terhitung nil.
func isNil(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Pointer, reflect.Func, reflect.Interface, reflect.Map, reflect.Slice, reflect.Chan:
		return rv.IsNil()
	}
	return false
}

// TimedOptions adalah bahan perakitan handler yang menampilkan atau menghitung tanggal.
type TimedOptions[S, C any] struct {
	Service   S
	GetCaller C
	Logger    *slog.Logger

	WriteJSON apierror.JSONWriter
	// FallbackErrorWriter menulis galat yang tidak dikenali pemetaan modul.
	FallbackErrorWriter apierror.ErrorWriter

	// Location adalah zona waktu tampilan; kosong berarti Asia/Jakarta. Ia parameter supaya
	// uji tidak bergantung pada basis data zona waktu mesin yang menjalankan.
	Location *time.Location

	// Now dapat diisi uji supaya waktu yang dipakai handler deterministik; kosong berarti
	// time.Now.
	Now func() time.Time
}

// Timed adalah field yang dibawa handler yang menampilkan atau menghitung tanggal.
type Timed[S, C any] struct {
	Service    S
	Caller     C
	Logger     *slog.Logger
	WriteJSON  apierror.JSONWriter
	WriteError apierror.ErrorWriter
	Location   *time.Location
	Now        func() time.Time
}

// NewTimed merakit kerangka handler bertanggal; penulis galatnya dibangun lewat writeError
// modul.
func NewTimed[S, C any](o TimedOptions[S, C], writeError ErrorWriterFactory) Timed[S, C] {
	location := o.Location
	if location == nil {
		location = clock.Jakarta()
	}
	now := o.Now
	if now == nil {
		now = time.Now
	}
	return Timed[S, C]{
		Service:    o.Service,
		Caller:     o.GetCaller,
		Logger:     o.Logger,
		WriteJSON:  o.WriteJSON,
		WriteError: writeError(o.Logger, o.WriteJSON, o.FallbackErrorWriter),
		Location:   location,
		Now:        now,
	}
}

// NewCrud merakit kerangka handler master yang MEMBAWA pemanggil tetapi tidak mewajibkannya —
// pemanggilnya hanya dipakai mengisi kolom pencatat perubahan bila tersedia. Yang diperiksa
// hanyalah Service dan kedua penulis jawaban.
func NewCrud[S, C any](module string, o MasterOptions[S, C]) (Master[S, C], error) {
	if isNil(o.Service) {
		return Master[S, C]{}, errors.New(module + ": Service wajib diisi")
	}
	if o.WriteResponse == nil || o.WriteError == nil {
		return Master[S, C]{}, errors.New(module + ": WriteResponse dan WriteError wajib diisi")
	}
	return Master[S, C]{
		Service:       o.Service,
		Caller:        o.Caller,
		Logger:        o.Logger,
		WriteResponse: o.WriteResponse,
		WriteError:    o.WriteError,
	}, nil
}

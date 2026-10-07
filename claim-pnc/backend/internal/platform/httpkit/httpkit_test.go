package httpkit

import (
	"context"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/platform/apierror"
)

type service struct{}

func writeJSON(http.ResponseWriter, *http.Request, int, any) {}
func writeErr(http.ResponseWriter, *http.Request, error)     {}

func TestNewInboxBuildsErrorWriter(t *testing.T) {
	var gotFallback apierror.ErrorWriter
	reader := CallerReader(func(context.Context) (Caller, bool) { return Caller{Login: "x"}, true })
	h := NewInbox(InboxOptions[*service, CallerReader]{
		Service: &service{}, GetCaller: reader, Logger: slog.Default(),
		WriteJSON: writeJSON, FallbackErrorWriter: writeErr,
	}, func(_ *slog.Logger, _ apierror.JSONWriter, fallback apierror.ErrorWriter) apierror.ErrorWriter {
		gotFallback = fallback
		return writeErr
	})
	require.NotNil(t, h.Service)
	require.NotNil(t, h.WriteError)
	require.NotNil(t, gotFallback)
	c, ok := h.Caller(context.Background())
	require.True(t, ok)
	require.Equal(t, "x", c.Login)
}

func TestNewMasterRejectsIncompleteAssembly(t *testing.T) {
	reader := CallerReader(func(context.Context) (Caller, bool) { return Caller{}, false })
	full := MasterOptions[*service, CallerReader]{Service: &service{}, Caller: reader, WriteResponse: writeJSON, WriteError: writeErr}

	_, err := NewMaster("m/http", full)
	require.NoError(t, err)

	noService := full
	noService.Service = nil
	_, err = NewMaster("m/http", noService)
	require.EqualError(t, err, "m/http: Service wajib diisi")

	noCaller := full
	noCaller.Caller = nil
	_, err = NewMaster("m/http", noCaller)
	require.EqualError(t, err, "m/http: Caller wajib diisi")

	noWriter := full
	noWriter.WriteError = nil
	_, err = NewMaster("m/http", noWriter)
	require.EqualError(t, err, "m/http: WriteResponse dan WriteError wajib diisi")
}

func TestNewBasicRejectsIncompleteAssembly(t *testing.T) {
	full := BasicOptions[*service]{Service: &service{}, WriteResponse: writeJSON, WriteError: writeErr}
	b, err := NewBasic("b/http", full)
	require.NoError(t, err)
	require.NotNil(t, b.Service)

	noService := full
	noService.Service = nil
	_, err = NewBasic("b/http", noService)
	require.EqualError(t, err, "b/http: Service wajib diisi")

	noWriter := full
	noWriter.WriteResponse = nil
	_, err = NewBasic("b/http", noWriter)
	require.EqualError(t, err, "b/http: WriteResponse dan WriteError wajib diisi")
}

func TestIsNil(t *testing.T) {
	var p *service
	var f func()
	require.True(t, isNil(nil))
	require.True(t, isNil(p))
	require.True(t, isNil(f))
	require.False(t, isNil(&service{}))
	require.False(t, isNil(3))
}

func TestNewTimedFillsDefaults(t *testing.T) {
	factory := func(*slog.Logger, apierror.JSONWriter, apierror.ErrorWriter) apierror.ErrorWriter { return writeErr }
	h := NewTimed(TimedOptions[*service, CallerReader]{Service: &service{}, WriteJSON: writeJSON}, factory)
	require.NotNil(t, h.Location)
	require.Equal(t, "Asia/Jakarta", h.Location.String())
	require.NotNil(t, h.Now)
	require.NotNil(t, h.WriteError)

	fixed := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	h = NewTimed(TimedOptions[*service, CallerReader]{Location: time.UTC, Now: func() time.Time { return fixed }}, factory)
	require.Equal(t, time.UTC, h.Location)
	require.Equal(t, fixed, h.Now())
}

func TestNewCrudAllowsMissingCaller(t *testing.T) {
	_, err := NewCrud("m/http", MasterOptions[*service, CallerReader]{WriteResponse: writeJSON, WriteError: writeErr})
	require.EqualError(t, err, "m/http: Service wajib diisi")
	_, err = NewCrud("m/http", MasterOptions[*service, CallerReader]{Service: &service{}})
	require.EqualError(t, err, "m/http: WriteResponse dan WriteError wajib diisi")
	h, err := NewCrud("m/http", MasterOptions[*service, CallerReader]{Service: &service{}, WriteResponse: writeJSON, WriteError: writeErr})
	require.NoError(t, err)
	require.Nil(t, h.Caller)
}

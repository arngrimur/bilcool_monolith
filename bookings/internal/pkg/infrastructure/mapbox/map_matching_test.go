package mapbox

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	extdomain "github.com/arngrimur/bilcool_monolith/bookings/pkg/domain"
)

func track(n int) []extdomain.TrackPoint {
	points := make([]extdomain.TrackPoint, n)
	for i := range points {
		points[i] = extdomain.TrackPoint{
			Lat:        59.0 + float64(i)*0.001,
			Lon:        18.0 + float64(i)*0.001,
			RecordedAt: time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC).Add(time.Duration(i) * 2 * time.Minute),
		}
	}
	return points
}

func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	c := NewClient("test-token")
	c.baseURL = srv.URL
	return c
}

const twoLineResponse = `{"code":"Ok","matchings":[
  {"geometry":{"coordinates":[[18.0,59.0],[18.0005,59.0005],[18.001,59.001]]}},
  {"geometry":{"coordinates":[[18.01,59.01],[18.011,59.011]]}}]}`

func TestMatchRoute_ReturnsMatchedGeometry(t *testing.T) {
	var gotQuery string
	var gotPath string
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(twoLineResponse))
	})

	lines, err := c.MatchRoute(context.Background(), track(3))

	require.NoError(t, err)
	require.Len(t, lines, 2, "one line per matching")
	require.Len(t, lines[0], 3)
	// Mapbox returns [lon, lat]; positions are lat/lon.
	require.InDelta(t, 59.0005, lines[0][1].Lat, 1e-9)
	require.InDelta(t, 18.0005, lines[0][1].Lon, 1e-9)

	require.Contains(t, gotQuery, "geometries=geojson")
	require.Contains(t, gotQuery, "overview=full")
	require.Contains(t, gotQuery, "radiuses=50;50;50")
	require.True(t, strings.HasPrefix(gotPath, "/18.000000,59.000000;"), "coordinates are lon,lat: %s", gotPath)
}

func TestMatchRoute_FewerThanTwoPoints(t *testing.T) {
	var calls atomic.Int32
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) { calls.Add(1) })

	lines, err := c.MatchRoute(context.Background(), track(1))

	require.NoError(t, err)
	require.Empty(t, lines)
	require.Zero(t, calls.Load())
}

func TestMatchRoute_NoAccessToken(t *testing.T) {
	var calls atomic.Int32
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) { calls.Add(1) })
	c.accessToken = ""

	_, err := c.MatchRoute(context.Background(), track(3))

	require.ErrorIs(t, err, ErrNoAccessToken)
	require.Zero(t, calls.Load())
}

func TestMatchRoute_FallsBackToRawPointsOnError(t *testing.T) {
	for name, handler := range map[string]http.HandlerFunc{
		"http error": func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusUnprocessableEntity) },
		"no match":   func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(`{"code":"NoMatch"}`)) },
		"bad json":   func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(`not json`)) },
		"no matchings": func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"code":"Ok","matchings":[]}`))
		},
	} {
		t.Run(name, func(t *testing.T) {
			c := newTestClient(t, handler)
			points := track(4)

			lines, err := c.MatchRoute(context.Background(), points)

			require.NoError(t, err)
			require.Len(t, lines, 1)
			require.Len(t, lines[0], 4)
			require.InDelta(t, points[2].Lat, lines[0][2].Lat, 1e-9)
			require.InDelta(t, points[2].Lon, lines[0][2].Lon, 1e-9)
		})
	}
}

func TestMatchRoute_SplitsLongTracksIntoOverlappingChunks(t *testing.T) {
	var calls atomic.Int32
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		require.LessOrEqual(t, strings.Count(r.URL.Path, ";")+1, maxWaypoints)
		_, _ = w.Write([]byte(`{"code":"Ok","matchings":[{"geometry":{"coordinates":[[18.0,59.0],[18.1,59.1]]}}]}`))
	})

	// 100 points fit in one request; 101 need a second one that overlaps by one point.
	lines, err := c.MatchRoute(context.Background(), track(100))
	require.NoError(t, err)
	require.Len(t, lines, 1)
	require.EqualValues(t, 1, calls.Load())

	calls.Store(0)
	lines, err = c.MatchRoute(context.Background(), track(101))
	require.NoError(t, err)
	require.Len(t, lines, 2)
	require.EqualValues(t, 2, calls.Load())
}

func TestMatchRoute_ErrorDoesNotLeakAccessToken(t *testing.T) {
	c := NewClient("super-secret")
	c.baseURL = "http://127.0.0.1:1" // nothing listens here

	_, err := c.matchGeometry(context.Background(), track(2))

	require.Error(t, err)
	require.NotContains(t, err.Error(), "super-secret")
}

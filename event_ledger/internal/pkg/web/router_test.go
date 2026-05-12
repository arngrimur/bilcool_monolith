package web_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/google/uuid"

	"github.com/arngrimur/bilcool_monolith/event_ledger/internal/pkg/domain"
	"github.com/arngrimur/bilcool_monolith/event_ledger/internal/pkg/web"
)

func TestWeb(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Web Suite")
}

type fakeQuerier struct {
	events []domain.EventItem
	err    error
}

func (f *fakeQuerier) QueryEvents(_ context.Context, _ domain.QueryParams) ([]domain.EventItem, error) {
	return f.events, f.err
}

func newRequest(method, target string) *http.Request {
	req, _ := http.NewRequest(method, target, nil)
	req.Header.Set("Correlation-Id", uuid.NewString())
	return req
}

var _ = Describe("HttpRouter", func() {
	var (
		router  *web.HttpRouter
		querier *fakeQuerier
	)

	BeforeEach(func() {
		querier = &fakeQuerier{}
		router = web.NewRouter(querier)
	})

	Describe("GET /ping", func() {
		It("returns 200 with ok status", func() {
			w := httptest.NewRecorder()
			router.ServeHTTP(w, newRequest(http.MethodGet, "/ping"))

			Expect(w.Code).To(Equal(http.StatusOK))
			var body map[string]string
			Expect(json.Unmarshal(w.Body.Bytes(), &body)).To(Succeed())
			Expect(body["status"]).To(Equal("ok"))
		})
	})

	Describe("GET /api/v1/events", func() {
		Context("when events exist", func() {
			BeforeEach(func() {
				querier.events = []domain.EventItem{
					{
						EventId:   "event-1",
						EventType: "booking_ended",
						Producer:  "bookings",
						EmittedAt: time.Now().UTC().Format(time.RFC3339Nano),
						Payload:   `{"key":"value"}`,
					},
				}
			})

			It("returns 200 with event list", func() {
				w := httptest.NewRecorder()
				router.ServeHTTP(w, newRequest(http.MethodGet, "/api/v1/events"))

				Expect(w.Code).To(Equal(http.StatusOK))
				var resp []domain.EventResponse
				Expect(json.Unmarshal(w.Body.Bytes(), &resp)).To(Succeed())
				Expect(resp).To(HaveLen(1))
				Expect(resp[0].EventId).To(Equal("event-1"))
			})
		})

		Context("when no events exist", func() {
			BeforeEach(func() {
				querier.events = []domain.EventItem{}
			})

			It("returns 200 with empty list", func() {
				w := httptest.NewRecorder()
				router.ServeHTTP(w, newRequest(http.MethodGet, "/api/v1/events"))

				Expect(w.Code).To(Equal(http.StatusOK))
				var resp []domain.EventResponse
				Expect(json.Unmarshal(w.Body.Bytes(), &resp)).To(Succeed())
				Expect(resp).To(BeEmpty())
			})
		})

		Context("with producer filter", func() {
			It("passes producer filter to querier", func() {
				querier.events = []domain.EventItem{}
				w := httptest.NewRecorder()
				router.ServeHTTP(w, newRequest(http.MethodGet, "/api/v1/events?producer=bookings"))

				Expect(w.Code).To(Equal(http.StatusOK))
			})
		})

		Context("with limit exceeding max", func() {
			It("caps limit at 50", func() {
				querier.events = []domain.EventItem{}
				w := httptest.NewRecorder()
				router.ServeHTTP(w, newRequest(http.MethodGet, "/api/v1/events?limit=999"))

				Expect(w.Code).To(Equal(http.StatusOK))
			})
		})

		Context("when querier returns an error", func() {
			BeforeEach(func() {
				querier.err = &queryError{}
			})

			It("returns 500", func() {
				w := httptest.NewRecorder()
				router.ServeHTTP(w, newRequest(http.MethodGet, "/api/v1/events"))

				Expect(w.Code).To(Equal(http.StatusInternalServerError))
			})
		})

		Context("when Correlation-Id header is missing", func() {
			It("returns 400", func() {
				w := httptest.NewRecorder()
				req, _ := http.NewRequest(http.MethodGet, "/api/v1/events", nil)
				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusBadRequest))
				var body map[string]string
				Expect(json.Unmarshal(w.Body.Bytes(), &body)).To(Succeed())
				Expect(body["error"]).To(Equal("Correlation-Id header is required"))
			})
		})
	})
})

type queryError struct{}

func (e *queryError) Error() string { return "query failed" }

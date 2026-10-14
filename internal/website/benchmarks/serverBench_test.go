package benchmarks

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func BenchmarkServer(b *testing.B) {
	s := SetupBenchmarkHelper(b)
	h := s.Handler()
	rec := httptest.NewRequest(http.MethodGet, "/", nil)
	assertBenchmarkResponse(b, h, rec, http.StatusOK, "Cully Solo")
	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, rec)
	}
}

func BenchmarkServerConcurrency(b *testing.B) {
	s := SetupBenchmarkHelper(b)
	h := s.Handler()
	assertBenchmarkResponse(b, h, httptest.NewRequest(http.MethodGet, "/", nil), http.StatusOK, "Cully Solo")
	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(p *testing.PB) {
		for p.Next() {
			rec := httptest.NewRequest(http.MethodGet, "/", nil)
			rr := httptest.NewRecorder()

			h.ServeHTTP(rr, rec)
		}
	})
}

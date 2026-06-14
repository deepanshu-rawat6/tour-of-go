package balancer_test

import (
	"testing"

	"tour_of_go/projects/from-scratch/05-load-balancer/internal/balancer"
)

func backends(t *testing.T, urls ...string) []*balancer.Backend {
	t.Helper()
	var bs []*balancer.Backend
	for _, u := range urls {
		b, err := balancer.NewBackend(u)
		if err != nil {
			t.Fatal(err)
		}
		bs = append(bs, b)
	}
	return bs
}

func TestRoundRobin_Distribution(t *testing.T) {
	bs := backends(t, "http://a:8001", "http://b:8002", "http://c:8003")
	rr := balancer.NewRoundRobin(bs)

	counts := map[string]int{}
	for i := 0; i < 9; i++ {
		b := rr.Next()
		if b == nil {
			t.Fatal("got nil backend")
		}
		counts[b.URL.Host]++
	}
	for _, c := range counts {
		if c != 3 {
			t.Fatalf("want 3 per backend, got %v", counts)
		}
	}
}

func TestRoundRobin_SkipsUnhealthy(t *testing.T) {
	bs := backends(t, "http://a:8001", "http://b:8002")
	bs[0].SetHealthy(false)
	rr := balancer.NewRoundRobin(bs)

	for i := 0; i < 4; i++ {
		b := rr.Next()
		if b == nil || b.URL.Host != "b:8002" {
			t.Fatalf("expected only b:8002, got %v", b)
		}
	}
}

func TestLeastConn_PicksLowest(t *testing.T) {
	bs := backends(t, "http://a:8001", "http://b:8002", "http://c:8003")
	bs[0].ActiveConns.Store(5)
	bs[1].ActiveConns.Store(2)
	bs[2].ActiveConns.Store(8)
	lc := balancer.NewLeastConn(bs)

	b := lc.Next()
	if b.URL.Host != "b:8002" {
		t.Fatalf("want b:8002 (least conns=2), got %s", b.URL.Host)
	}
}

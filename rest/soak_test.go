package rest_test

import (
	"bytes"
	"context"
	"flag"
	"io"
	"net"
	"net/http"
	"runtime"
	"runtime/debug"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/amatsagu/tempest/rest"
)

var soakDuration = flag.Duration("soak", 0, "duration for TestLongTerm15kRPSSoak (e.g. 5m)")

func raceDetectorEnabled() bool {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return false
	}

	for _, setting := range info.Settings {
		if setting.Key == "-race" {
			return setting.Value == "true"
		}
	}

	return false
}

func runSustainedLoadTest(t *testing.T, duration time.Duration, targetRPS int, requireMinRPS float64) {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer func() { _ = ln.Close() }()

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)

		hdr := w.Header()
		hdr["Content-Type"] = []string{rest.ContentTypeJSON}
		hdr["X-RateLimit-Bucket"] = []string{"load-bucket"}
		hdr["X-RateLimit-Limit"] = []string{"500"}
		hdr["X-RateLimit-Remaining"] = []string{"499"}
		hdr["X-RateLimit-Reset-After"] = []string{"1.0"}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"123456789012345678","channel_id":"1"}`))
	})

	srv := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		_ = srv.Serve(ln)
	}()

	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	client := rest.NewRestClient(rest.RestClientConfig{
		Token:               testToken("987654321012345678"),
		BaseURL:             "http://" + ln.Addr().String() + "/api/v10",
		MaxIdleConnsPerHost: 256,
		MaxRetries:          3,
		MaxWaitTime:         5 * time.Second,
	})
	defer client.HTTPClient.CloseIdleConnections()

	fileData := bytes.Repeat([]byte("P"), 4*1024)

	payload := struct {
		Content string `json:"content"`
		Flags   uint64 `json:"flags,omitzero"`
	}{
		Content: "sustained load event",
		Flags:   64,
	}

	const (
		workers   = 128
		serverCap = 1_500_000
	)
	stepNs := int64(time.Second) / int64(targetRPS)

	ctx, cancel := context.WithTimeout(t.Context(), duration)
	defer cancel()

	var (
		totalOK    atomic.Uint64
		totalFiles atomic.Uint64
		totalErr   atomic.Uint64
		seq        atomic.Uint64
		nextSlot   atomic.Int64
	)

	start := time.Now()
	nextSlot.Store(start.UnixNano())

	stopReport := make(chan struct{})
	var reportWG sync.WaitGroup

	if duration >= 10*time.Second {
		reportWG.Go(func() {
			ticker := time.NewTicker(15 * time.Second)
			defer ticker.Stop()

			var prevOK uint64
			prevTime := start

			for {
				select {
				case <-stopReport:
					return

				case now := <-ticker.C:
					curOK := totalOK.Load()
					curErr := totalErr.Load()
					curFiles := totalFiles.Load()

					dt := now.Sub(prevTime).Seconds()
					intervalRPS := float64(curOK-prevOK) / dt

					var ms runtime.MemStats
					runtime.ReadMemStats(&ms)

					buckets, routes := client.Limiter().Size()

					t.Logf("[soak %s] interval=%.0f req/s total_ok=%d files=%d err=%d heap_alloc=%.2fMiB heap_inuse=%.2fMiB gc=%d buckets=%d routes=%d",
						now.Sub(start).Round(time.Second), intervalRPS, curOK, curFiles, curErr,
						float64(ms.HeapAlloc)/(1024*1024), float64(ms.HeapInuse)/(1024*1024), ms.NumGC, buckets, routes)

					prevOK = curOK
					prevTime = now
				}
			}
		})
	}

	var wg sync.WaitGroup
	for range workers {
		wg.Go(func() {
			r := bytes.NewReader(nil)
			files := []rest.File{{Name: "event.png", Reader: r, ContentType: "image/png"}}

			var out struct {
				ID        string `json:"id"`
				ChannelID string `json:"channel_id"`
			}

			var pathBuf [64]byte
			copy(pathBuf[:], "/channels/")

			timer := time.NewTimer(time.Hour)
			timer.Stop()
			defer timer.Stop()

			for {
				slot := nextSlot.Add(stepNs) - stepNs
				now := time.Now().UnixNano()

				if slot > now {
					timer.Reset(time.Duration(slot - now))

					select {
					case <-ctx.Done():
						return
					case <-timer.C:
					}
				} else if ctx.Err() != nil {
					return
				}

				n := seq.Add(1)

				// Pseudo-random permutation across 1.5M simulated Discord servers.
				serverID := ((n * 2654435761) % serverCap) + 100_000_000_000_000_000
				p := strconv.AppendUint(pathBuf[:10], serverID, 10)
				p = append(p, "/messages"...)
				route := string(p)

				var callErr error
				if n%20 == 0 {
					r.Reset(fileData)
					callErr = client.RequestWithFiles(ctx, http.MethodPost, route, payload, files, &out)
					totalFiles.Add(1)
				} else {
					callErr = client.Request(ctx, http.MethodPost, route, payload, &out)
				}

				if callErr != nil {
					if ctx.Err() != nil {
						return
					}

					totalErr.Add(1)
					continue
				}

				totalOK.Add(1)
			}
		})
	}

	wg.Wait()
	close(stopReport)
	reportWG.Wait()

	elapsed := time.Since(start)
	okCount := totalOK.Load()
	errCount := totalErr.Load()
	fileCount := totalFiles.Load()
	totalCount := okCount + errCount

	if totalCount == 0 {
		t.Fatal("no requests completed during soak test")
	}

	actualRPS := float64(okCount) / elapsed.Seconds()
	errRate := float64(errCount) / float64(totalCount)
	buckets, routes := client.Limiter().Size()

	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	t.Logf("Summary: elapsed=%s ok=%d files=%d err=%d err_rate=%.4f%% achieved_rps=%.1f heap_alloc=%.2fMiB heap_inuse=%.2fMiB gc=%d buckets=%d routes=%d",
		elapsed.Round(time.Millisecond), okCount, fileCount, errCount, errRate*100, actualRPS,
		float64(ms.HeapAlloc)/(1024*1024), float64(ms.HeapInuse)/(1024*1024), ms.NumGC, buckets, routes)

	if errRate >= 0.005 {
		t.Fatalf("error rate %.4f%% exceeds 0.50%% budget (%d/%d errors)", errRate*100, errCount, totalCount)
	}

	if !raceDetectorEnabled() && actualRPS < requireMinRPS {
		t.Fatalf("achieved %.1f req/s, below required %.1f req/s", actualRPS, requireMinRPS)
	}

	const defaultMaxBuckets, defaultMaxRoutes = 65536, 65536
	if buckets > defaultMaxBuckets || routes > defaultMaxRoutes {
		t.Fatalf("rate limiter exceeded memory caps: buckets=%d routes=%d", buckets, routes)
	}
}

func TestThroughput15kRPSSmoke(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping 15k req/s smoke test in -short mode")
	}

	runSustainedLoadTest(t, 2*time.Second, 15_500, 15_000)
}

func TestLongTerm15kRPSSoak(t *testing.T) {
	if *soakDuration <= 0 {
		t.Skip("pass -timeout=0 -soak=5m to run the long-term 15k req/s stability soak test")
	}

	runSustainedLoadTest(t, *soakDuration, 15_500, 15_000)
}

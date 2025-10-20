package network

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"sync"
	"time"
)

const Version = "1.0.1"

// QualityResult holds the network quality test results
type QualityResult struct {
	UplinkCapacity   float64 // Mbps
	DownlinkCapacity float64 // Mbps
	IdleLatency      float64 // milliseconds
	Responsiveness   string  // Low, Medium, High
	ResponsivenessMs float64 // milliseconds
}

// TestConfig holds configuration for network tests
type TestConfig struct {
	TestDuration    time.Duration
	TestServers     []string
	UploadServers   []string
	UploadChunkSize int
	NumConnections  int
}

// DefaultConfig returns a default test configuration
func DefaultConfig() *TestConfig {
	return &TestConfig{
		TestDuration:   10 * time.Second,
		NumConnections: 4,
		TestServers: []string{
			"https://speed.cloudflare.com/__down?bytes=10000000", // Cloudflare speed test
			"https://www.google.com/generate_204",                // Google no-content
		},
		UploadServers: []string{
			"https://httpbin.org/post",
			"https://speed.cloudflare.com/__up?bytes=10000000",
		},
		UploadChunkSize: 512 * 1024, // 512KB
	}
}

// RunQualityTest performs a network quality test
func RunQualityTest(ctx context.Context, config *TestConfig) (*QualityResult, error) {
	if config == nil {
		config = DefaultConfig()
	}

	if config.TestDuration <= 0 {
		return nil, fmt.Errorf("test duration must be positive")
	}

	if len(config.TestServers) == 0 {
		return nil, fmt.Errorf("no download test servers configured")
	}

	downloadURL := config.TestServers[0]
	latencyURL := downloadURL
	if len(config.TestServers) > 1 {
		latencyURL = config.TestServers[1]
	}

	idleLatency, err := measureIdleLatency(ctx, latencyURL)
	if err != nil {
		return nil, fmt.Errorf("failed to measure idle latency: %w", err)
	}

	downloadMbps, loadedLatency, err := measureDownloadSpeed(ctx, config, downloadURL, latencyURL)
	if err != nil {
		return nil, fmt.Errorf("failed to measure download speed: %w", err)
	}

	uploadMbps, err := measureUploadSpeed(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to measure upload speed: %w", err)
	}

	result := &QualityResult{
		UplinkCapacity:   uploadMbps,
		DownlinkCapacity: downloadMbps,
		IdleLatency:      idleLatency,
		ResponsivenessMs: loadedLatency,
	}

	if loadedLatency < 200 {
		result.Responsiveness = "High"
	} else if loadedLatency < 1000 {
		result.Responsiveness = "Medium"
	} else {
		result.Responsiveness = "Low"
	}

	return result, nil
}

// measureIdleLatency measures network latency when idle
func measureIdleLatency(ctx context.Context, testURL string) (float64, error) {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	var totalLatency time.Duration
	successCount := 0
	numTests := 10

	for i := 0; i < numTests; i++ {
		start := time.Now()

		req, err := http.NewRequestWithContext(ctx, "GET", testURL, nil)
		if err != nil {
			continue
		}

		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		resp.Body.Close()

		latency := time.Since(start)
		totalLatency += latency
		successCount++

		time.Sleep(100 * time.Millisecond) // Small delay between tests
	}

	if successCount == 0 {
		return 0, fmt.Errorf("all latency tests failed")
	}

	avgLatency := totalLatency / time.Duration(successCount)
	return float64(avgLatency.Milliseconds()), nil
}

// measureDownloadSpeed measures download capacity and latency under load
func measureDownloadSpeed(ctx context.Context, config *TestConfig, downloadURL, latencyURL string) (float64, float64, error) {
	var totalBytes int64
	var mu sync.Mutex
	var wg sync.WaitGroup

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Start timer
	startTime := time.Now()
	deadline := startTime.Add(config.TestDuration)

	// Measure latency under load
	latencyChan := make(chan float64, 1)
	go func() {
		time.Sleep(2 * time.Second) // Wait for load to build up
		latency, _ := measureIdleLatency(ctx, latencyURL)
		latencyChan <- latency
	}()

	// Run parallel downloads
	for i := 0; i < config.NumConnections; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for time.Now().Before(deadline) {
				select {
				case <-ctx.Done():
					return
				default:
				}

				req, err := http.NewRequestWithContext(ctx, "GET", downloadURL, nil)
				if err != nil {
					continue
				}

				resp, err := client.Do(req)
				if err != nil {
					continue
				}

				bytes, _ := io.Copy(io.Discard, resp.Body)
				resp.Body.Close()

				mu.Lock()
				totalBytes += bytes
				mu.Unlock()
			}
		}()
	}

	wg.Wait()
	duration := time.Since(startTime).Seconds()

	// Get latency under load
	loadedLatency := <-latencyChan

	// Calculate Mbps
	mbps := (float64(totalBytes) * 8) / (duration * 1000000)

	return math.Round(mbps*1000) / 1000, loadedLatency, nil
}

// measureUploadSpeed measures upload capacity
func measureUploadSpeed(ctx context.Context, config *TestConfig) (float64, error) {
	if len(config.UploadServers) == 0 {
		return 0, fmt.Errorf("no upload servers configured")
	}

	chunkSize := config.UploadChunkSize
	if chunkSize <= 0 {
		chunkSize = 512 * 1024 // default to 512KB
	}

	payload := make([]byte, chunkSize)

	var totalBytes int64
	var mu sync.Mutex
	var wg sync.WaitGroup

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	startTime := time.Now()
	deadline := startTime.Add(config.TestDuration / 2)

	for i := 0; i < config.NumConnections; i++ {
		serverURL := config.UploadServers[i%len(config.UploadServers)]

		wg.Add(1)
		go func(target string) {
			defer wg.Done()

			for time.Now().Before(deadline) {
				select {
				case <-ctx.Done():
					return
				default:
				}

				reader := bytes.NewReader(payload)
				req, err := http.NewRequestWithContext(ctx, "POST", target, reader)
				if err != nil {
					continue
				}
				req.Header.Set("Content-Type", "application/octet-stream")
				req.ContentLength = int64(chunkSize)

				resp, err := client.Do(req)
				if err != nil {
					continue
				}

				io.Copy(io.Discard, resp.Body)
				statusOK := resp.StatusCode >= http.StatusOK && resp.StatusCode < 400
				resp.Body.Close()

				if !statusOK {
					continue
				}

				mu.Lock()
				totalBytes += int64(chunkSize)
				mu.Unlock()
			}
		}(serverURL)
	}

	wg.Wait()
	duration := time.Since(startTime).Seconds()
	if duration == 0 {
		return 0, fmt.Errorf("upload duration was zero")
	}

	mbps := (float64(totalBytes) * 8) / (duration * 1000000)

	return math.Round(mbps*1000) / 1000, nil
}

// FormatResult returns a formatted string of the test results
func (r *QualityResult) FormatResult() string {
	return fmt.Sprintf(`=========== SUMMARY ===========
Uplink capacity: %.3f Mbps
Downlink capacity: %.3f Mbps
Responsiveness: %s (%.3f milliseconds)
Idle Latency: %.3f milliseconds
`, r.UplinkCapacity, r.DownlinkCapacity, r.Responsiveness, r.ResponsivenessMs, r.IdleLatency)
}

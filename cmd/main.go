package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"networkquality/network"
)

func main() {
	// Command line flags
	duration := flag.Int("d", 10, "Test duration in seconds")
	connections := flag.Int("c", 4, "Number of parallel connections")
	verbose := flag.Bool("v", false, "Verbose output")
	quick := flag.Bool("q", false, "Quick test (5 seconds)")
	help := flag.Bool("h", false, "Show help")

	flag.Parse()

	if *help {
		printHelp()
		return
	}

	testDuration := time.Duration(*duration) * time.Second
	if *quick {
		testDuration = 5 * time.Second
	}

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\n\nTest interrupted by user")
		cancel()
		os.Exit(0)
	}()

	// Print header
	fmt.Println("networkquality")
	fmt.Println("==============")

	// Configure test
	config := network.DefaultConfig()
	config.TestDuration = testDuration
	config.NumConnections = *connections

	if *verbose {
		fmt.Printf("Configuration:\n")
		fmt.Printf("  Test duration: %v\n", config.TestDuration)
		fmt.Printf("  Connections: %d\n", config.NumConnections)
		fmt.Println()
	}

	// Run the test
	fmt.Println("Running network quality test...")
	fmt.Println()

	startTime := time.Now()
	result, err := network.RunQualityTest(ctx, config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	elapsed := time.Since(startTime)

	// Display results
	displayResults(result)

	if *verbose {
		fmt.Printf("\nTest completed in %.2f seconds\n", elapsed.Seconds())
	}
}

func displayResults(result *network.QualityResult) {
	// Display results in the same format as the screenshot
	fmt.Println("\n=========== SUMMARY ===========")
	fmt.Printf("Uplink capacity: %.3f Mbps\n", result.UplinkCapacity)
	fmt.Printf("Downlink capacity: %.3f Mbps\n", result.DownlinkCapacity)
	fmt.Printf("Responsiveness: %s (%.3f milliseconds)\n",
		result.Responsiveness, result.ResponsivenessMs)
	fmt.Printf("Idle Latency: %.3f milliseconds\n", result.IdleLatency)

	// Add visual indicator for quality
	fmt.Println("\n========== QUALITY ============")
	quality := calculateOverallQuality(result)
	fmt.Printf("Overall: %s\n", quality)

	// Performance bars
	fmt.Println("\n======== PERFORMANCE ==========")
	fmt.Printf("Download: %s\n", getPerformanceBar(result.DownlinkCapacity, 100))
	fmt.Printf("Upload:   %s\n", getPerformanceBar(result.UplinkCapacity, 50))
	fmt.Printf("Latency:  %s\n", getLatencyBar(result.IdleLatency))
}

func calculateOverallQuality(result *network.QualityResult) string {
	score := 0

	// Score based on download speed
	if result.DownlinkCapacity > 50 {
		score += 3
	} else if result.DownlinkCapacity > 25 {
		score += 2
	} else if result.DownlinkCapacity > 10 {
		score += 1
	}

	// Score based on upload speed
	if result.UplinkCapacity > 20 {
		score += 3
	} else if result.UplinkCapacity > 10 {
		score += 2
	} else if result.UplinkCapacity > 5 {
		score += 1
	}

	// Score based on latency
	if result.IdleLatency < 20 {
		score += 3
	} else if result.IdleLatency < 50 {
		score += 2
	} else if result.IdleLatency < 100 {
		score += 1
	}

	// Calculate overall quality
	switch {
	case score >= 8:
		return "⭐ Excellent"
	case score >= 6:
		return "✅ Good"
	case score >= 4:
		return "⚠️  Fair"
	default:
		return "❌ Poor"
	}
}

func getPerformanceBar(value float64, maxValue float64) string {
	barLength := 20
	filled := int((value / maxValue) * float64(barLength))
	if filled > barLength {
		filled = barLength
	}

	bar := "["
	for i := 0; i < barLength; i++ {
		if i < filled {
			bar += "█"
		} else {
			bar += "░"
		}
	}
	bar += fmt.Sprintf("] %.2f Mbps", value)

	return bar
}

func getLatencyBar(latency float64) string {
	barLength := 20
	// Inverse scale for latency (lower is better)
	maxLatency := 200.0
	filled := int((1 - (latency / maxLatency)) * float64(barLength))
	if filled < 0 {
		filled = 0
	}
	if filled > barLength {
		filled = barLength
	}

	bar := "["
	for i := 0; i < barLength; i++ {
		if i < filled {
			bar += "█"
		} else {
			bar += "░"
		}
	}
	bar += fmt.Sprintf("] %.2f ms", latency)

	return bar
}

func printHelp() {
	fmt.Println("networkquality - Test network quality and performance")
	fmt.Println("\nUsage: networkquality [options]")
	fmt.Println("\nOptions:")
	fmt.Println("  -d <seconds>  Test duration in seconds (default: 10)")
	fmt.Println("  -c <count>    Number of parallel connections (default: 4)")
	fmt.Println("  -q            Quick test (5 seconds)")
	fmt.Println("  -v            Verbose output")
	fmt.Println("  -h            Show this help message")
	fmt.Println("\nExamples:")
	fmt.Println("  networkquality           # Run standard test")
	fmt.Println("  networkquality -q        # Run quick test")
	fmt.Println("  networkquality -d 30     # Run 30-second test")
	fmt.Println("  networkquality -v        # Run with verbose output")
}

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	ct "github.com/daviddengcn/go-colortext"
	"github.com/P-0001/networkquality/network"
)

func main() {
	// Command line flags
	duration := flag.Int("d", 10, "Test duration in seconds")
	connections := flag.Int("c", 4, "Number of parallel connections")
	verbose := flag.Bool("v", false, "Verbose output")
	quick := flag.Bool("q", false, "Quick test (5 seconds)")
	help := flag.Bool("h", false, "Show help")
	version := flag.Bool("version", false, "Show version")

	flag.Parse()

	if *version {
		ct.Foreground(ct.Cyan, true)
		fmt.Print("networkquality ")
		ct.Foreground(ct.Green, true)
		fmt.Println("version", network.Version)
		ct.ResetColor()
		return
	}

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
		ct.Foreground(ct.Yellow, true)
		fmt.Println("\n\nTest interrupted by user")
		ct.ResetColor()
		cancel()
		os.Exit(0)
	}()

	// Print header
	ct.Foreground(ct.Cyan, true)
	fmt.Println("Networkquality")
	fmt.Println("==============")
	ct.ResetColor()

	// Configure test
	config := network.DefaultConfig()
	config.TestDuration = testDuration
	config.NumConnections = *connections

	if *verbose {
		ct.Foreground(ct.Magenta, false)
		fmt.Printf("Configuration:\n")
		ct.Foreground(ct.White, false)
		fmt.Printf("  Test duration: %v\n", config.TestDuration)
		fmt.Printf("  Connections: %d\n", config.NumConnections)
		ct.ResetColor()
		fmt.Println()
	}

	spinnerStop := make(chan struct{})
	spinnerDone := make(chan struct{})
	go func() {
		defer close(spinnerDone)
		frames := []rune{'|', '/', '-', '\\'}
		idx := 0
		ticker := time.NewTicker(120 * time.Millisecond)
		defer ticker.Stop()

		ct.Foreground(ct.Yellow, false)
		fmt.Print("Running network quality test... ")
		for {
			select {
			case <-spinnerStop:
				fmt.Print("\rRunning network quality test...    \r")
				ct.ResetColor()
				return
			case <-ticker.C:
				fmt.Printf("\rRunning network quality test... %c", frames[idx%len(frames)])
				idx++
			}
		}
	}()

	startTime := time.Now()
	result, err := network.RunQualityTest(ctx, config)
	close(spinnerStop)
	<-spinnerDone

	elapsed := time.Since(startTime)

	if err != nil {
		ct.Foreground(ct.Red, true)
		fmt.Printf("Running network quality test... failed\n\n")
		ct.ResetColor()
	} else {
		ct.Foreground(ct.Green, true)
		fmt.Printf("Running network quality test... done\n\n")
		ct.ResetColor()
	}

	if err != nil {
		ct.Foreground(ct.Red, true)
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		ct.ResetColor()
		os.Exit(1)
	}

	// Display results
	displayResults(result)

	if *verbose {
		ct.Foreground(ct.Magenta, false)
		fmt.Printf("\nTest completed in %.2f seconds\n", elapsed.Seconds())
		ct.ResetColor()
	}
}

func displayResults(result *network.QualityResult) {
	// Display results in the same format as the screenshot
	ct.Foreground(ct.Cyan, true)
	fmt.Println("\n=========== SUMMARY ===========")
	ct.ResetColor()
	
	ct.Foreground(ct.Green, false)
	fmt.Print("Uplink capacity: ")
	ct.Foreground(ct.White, true)
	fmt.Printf("%.3f Mbps\n", result.UplinkCapacity)
	ct.ResetColor()
	
	ct.Foreground(ct.Green, false)
	fmt.Print("Downlink capacity: ")
	ct.Foreground(ct.White, true)
	fmt.Printf("%.3f Mbps\n", result.DownlinkCapacity)
	ct.ResetColor()
	
	ct.Foreground(ct.Green, false)
	fmt.Print("Responsiveness: ")
	ct.Foreground(ct.White, true)
	fmt.Printf("%s (%.3f milliseconds)\n",
		result.Responsiveness, result.ResponsivenessMs)
	ct.ResetColor()
	
	ct.Foreground(ct.Green, false)
	fmt.Print("Idle Latency: ")
	ct.Foreground(ct.White, true)
	fmt.Printf("%.3f milliseconds\n", result.IdleLatency)
	ct.ResetColor()

	// Add visual indicator for quality
	ct.Foreground(ct.Cyan, true)
	fmt.Println("\n========== QUALITY ============")
	ct.ResetColor()
	quality, qualityColor := calculateOverallQuality(result)
	ct.Foreground(ct.White, false)
	fmt.Print("Overall: ")
	ct.Foreground(qualityColor, true)
	fmt.Printf("%s\n", quality)
	ct.ResetColor()

	// Performance bars
	ct.Foreground(ct.Cyan, true)
	fmt.Println("\n======== PERFORMANCE ==========")
	ct.ResetColor()
	
	ct.Foreground(ct.Blue, false)
	fmt.Print("Download: ")
	ct.ResetColor()
	fmt.Printf("%s\n", getPerformanceBar(result.DownlinkCapacity, 100))
	
	ct.Foreground(ct.Blue, false)
	fmt.Print("Upload:   ")
	ct.ResetColor()
	fmt.Printf("%s\n", getPerformanceBar(result.UplinkCapacity, 50))
	
	ct.Foreground(ct.Blue, false)
	fmt.Print("Latency:  ")
	ct.ResetColor()
	fmt.Printf("%s\n", getLatencyBar(result.IdleLatency))
}

func calculateOverallQuality(result *network.QualityResult) (string, ct.Color) {
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
		return "⭐ Excellent", ct.Green
	case score >= 6:
		return "✅ Good", ct.Cyan
	case score >= 4:
		return "⚠️  Fair", ct.Yellow
	default:
		return "❌ Poor", ct.Red
	}
}

func getPerformanceBar(value float64, maxValue float64) string {
	barLength := 20
	filled := int((value / maxValue) * float64(barLength))
	if filled > barLength {
		filled = barLength
	}

	ct.Foreground(ct.White, false)
	fmt.Print("[")
	ct.ResetColor()
	
	// Color the filled portion based on performance
	var barColor ct.Color
	percentage := (value / maxValue) * 100
	switch {
	case percentage >= 75:
		barColor = ct.Green
	case percentage >= 50:
		barColor = ct.Cyan
	case percentage >= 25:
		barColor = ct.Yellow
	default:
		barColor = ct.Red
	}
	
	ct.Foreground(barColor, true)
	for i := 0; i < filled; i++ {
		fmt.Print("█")
	}
	ct.ResetColor()
	
	ct.Foreground(ct.White, false)
	for i := filled; i < barLength; i++ {
		fmt.Print("░")
	}
	fmt.Print("]")
	ct.ResetColor()
	
	ct.Foreground(ct.White, true)
	fmt.Printf(" %.2f Mbps", value)
	ct.ResetColor()
	
	return ""
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

	ct.Foreground(ct.White, false)
	fmt.Print("[")
	ct.ResetColor()
	
	// Color based on latency (lower is better)
	var barColor ct.Color
	switch {
	case latency < 20:
		barColor = ct.Green
	case latency < 50:
		barColor = ct.Cyan
	case latency < 100:
		barColor = ct.Yellow
	default:
		barColor = ct.Red
	}
	
	ct.Foreground(barColor, true)
	for i := 0; i < filled; i++ {
		fmt.Print("█")
	}
	ct.ResetColor()
	
	ct.Foreground(ct.White, false)
	for i := filled; i < barLength; i++ {
		fmt.Print("░")
	}
	fmt.Print("]")
	ct.ResetColor()
	
	ct.Foreground(ct.White, true)
	fmt.Printf(" %.2f ms", latency)
	ct.ResetColor()
	
	return ""
}

func printHelp() {
	ct.Foreground(ct.Cyan, true)
	fmt.Println("networkquality")
	ct.Foreground(ct.White, false)
	fmt.Println(" - Test network quality and performance")
	ct.ResetColor()
	
	ct.Foreground(ct.Yellow, true)
	fmt.Println("\nUsage:")
	ct.ResetColor()
	ct.Foreground(ct.White, false)
	fmt.Println("  networkquality [options]")
	ct.ResetColor()
	
	ct.Foreground(ct.Yellow, true)
	fmt.Println("\nOptions:")
	ct.ResetColor()
	ct.Foreground(ct.Green, false)
	fmt.Print("  -d <seconds>  ")
	ct.Foreground(ct.White, false)
	fmt.Println("Test duration in seconds (default: 10)")
	ct.Foreground(ct.Green, false)
	fmt.Print("  -c <count>    ")
	ct.Foreground(ct.White, false)
	fmt.Println("Number of parallel connections (default: 4)")
	ct.Foreground(ct.Green, false)
	fmt.Print("  -q            ")
	ct.Foreground(ct.White, false)
	fmt.Println("Quick test (5 seconds)")
	ct.Foreground(ct.Green, false)
	fmt.Print("  -v            ")
	ct.Foreground(ct.White, false)
	fmt.Println("Verbose output")
	ct.Foreground(ct.Green, false)
	fmt.Print("  -h            ")
	ct.Foreground(ct.White, false)
	fmt.Println("Show this help message")
	ct.ResetColor()
	
	ct.Foreground(ct.Yellow, true)
	fmt.Println("\nExamples:")
	ct.ResetColor()
	ct.Foreground(ct.Cyan, false)
	fmt.Print("  networkquality           ")
	ct.Foreground(ct.White, false)
	fmt.Println("# Run standard test")
	ct.Foreground(ct.Cyan, false)
	fmt.Print("  networkquality -q        ")
	ct.Foreground(ct.White, false)
	fmt.Println("# Run quick test")
	ct.Foreground(ct.Cyan, false)
	fmt.Print("  networkquality -d 30     ")
	ct.Foreground(ct.White, false)
	fmt.Println("# Run 30-second test")
	ct.Foreground(ct.Cyan, false)
	fmt.Print("  networkquality -v        ")
	ct.Foreground(ct.White, false)
	fmt.Println("# Run with verbose output")
	ct.ResetColor()
}

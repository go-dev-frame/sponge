package common

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fatih/color"
)

// Bar represents a thread-safe progress bar.
type Bar struct {
	total     int64 // total items
	current   int64 // current progress
	startTime time.Time
	barWidth  int    // display width in terminal
	graph     string // symbol for completed portion
	arrow     string // symbol for current progress
	space     string // symbol for remaining portion

	lastDrawNano   atomic.Int64
	updateInterval time.Duration // refresh interval to avoid frequent I/O
}

// NewBar returns a new progress bar with the given total count.
func NewBar(total int64, t time.Time) *Bar {
	b := &Bar{
		total:          total,
		startTime:      t,
		barWidth:       50,
		graph:          "=",
		arrow:          ">",
		space:          " ",
		updateInterval: 500 * time.Millisecond,
	}

	b.lastDrawNano.Store(0)
	return b
}

// Increment advances the progress by 1 and redraws the bar if needed.
func (b *Bar) Increment() {
	atomic.AddInt64(&b.current, 1)
	b.draw(false) // Regular draw, respects interval
}

// Finish marks the bar as complete and prints the final state.
func (b *Bar) Finish() {
	atomic.StoreInt64(&b.current, b.total)
	b.draw(true) // Force final draw
	fmt.Println()
}

// Stop halts the bar at its current progress and prints the final state.
func (b *Bar) Stop() {
	b.draw(true) // Force a final draw at the current state
	fmt.Println()
}

// shouldDraw reports whether a redraw should occur using a CAS timestamp.
// Ensures only one goroutine wins the right to draw within the interval.
func (b *Bar) shouldDraw() bool {
	now := time.Now().UnixNano()
	intervalNano := b.updateInterval.Nanoseconds()

	lastNano := b.lastDrawNano.Load()
	if now-lastNano < intervalNano {
		// Too close to the last draw, skip
		return false
	}

	// Attempt to update the timestamp
	return b.lastDrawNano.CompareAndSwap(lastNano, now)
}

// draw renders the bar in the terminal.
// The 'force' parameter bypasses the update interval check.
func (b *Bar) draw(force bool) {
	current := atomic.LoadInt64(&b.current)

	// Redraw only when forced, reaching refresh interval, or on completion.
	if !force && current < b.total && !b.shouldDraw() {
		return
	}

	percent := float64(current) / float64(b.total)
	if percent > 1.0 {
		percent = 1.0
	}
	filledLength := int(float64(b.barWidth) * percent)

	// Build the visual bar
	var barBuilder strings.Builder
	barBuilder.Grow(b.barWidth + 2)
	barBuilder.WriteString("[")
	barBuilder.WriteString(strings.Repeat(b.graph, filledLength))

	// Show arrow only when not finished
	if current < b.total {
		if filledLength < b.barWidth {
			barBuilder.WriteString(b.arrow)
			barBuilder.WriteString(strings.Repeat(b.space, b.barWidth-filledLength-1))
		} else {
			barBuilder.WriteString(strings.Repeat(b.space, b.barWidth-filledLength))
		}
	} else {
		barBuilder.WriteString(strings.Repeat(b.space, b.barWidth-filledLength))
	}
	barBuilder.WriteString("]")

	elapsed := time.Since(b.startTime).Seconds()

	str := fmt.Sprintf("%8d / %-8d %s %6.2f%% %.2fs", current, b.total, barBuilder.String(), percent*100, elapsed)
	fmt.Printf("\r%s", color.HiBlackString(str))
}

// -----------------------------------------------------------------

// TimeBar represents a thread-safe time-based progress bar.
type TimeBar struct {
	totalDuration time.Duration // total duration
	startTime     time.Time     // start time
	barWidth      int           // display width in terminal
	graph         string        // symbol for completed portion
	arrow         string        // symbol for current progress
	space         string        // symbol for remaining portion

	// Background goroutine control
	wg   sync.WaitGroup
	done chan struct{}
}

// NewTimeBar returns a new time-based progress bar with the given duration.
func NewTimeBar(totalDuration time.Duration) *TimeBar {
	return &TimeBar{
		totalDuration: totalDuration,
		barWidth:      50,
		graph:         "=",
		arrow:         ">",
		space:         " ",
		done:          make(chan struct{}),
	}
}

// Start begins automatic updates in a background goroutine.
func (b *TimeBar) Start() {
	b.startTime = time.Now()
	b.wg.Go(func() { b.run() })
}

// stopped is an internal helper to handle shutting down the progress bar.
func (b *TimeBar) stopped(isFinal bool) {
	select {
	case <-b.done:
		// Already stopped, do nothing.
		return
	default:
		// Signal the run goroutine to stop.
		close(b.done)
	}
	b.wg.Wait()     // Wait for the goroutine to exit.
	b.draw(isFinal) // Perform one final draw.
	fmt.Println()   // Move to the next line.
}

// Finish stops the progress bar at 100%.
func (b *TimeBar) Finish() {
	b.stopped(true)
}

// Stop halts the progress bar at its current progress.
func (b *TimeBar) Stop() {
	b.stopped(false)
}

// run periodically refreshes the bar in the background.
func (b *TimeBar) run() {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-b.done:
			// Stop signal received, exit the loop.
			// The final draw is handled by the calling function (Finish/Stop).
			return
		case <-ticker.C:
			if time.Since(b.startTime) >= b.totalDuration {
				// Time has elapsed, exit.
				// The final draw will be handled by Finish().
				return
			}
			b.draw(false)
		}
	}
}

// draw renders the bar in the terminal.
// isFinal indicates whether this is the last draw (i.e., should show 100%).
func (b *TimeBar) draw(isFinal bool) {
	elapsed := time.Since(b.startTime)
	percent := elapsed.Seconds() / b.totalDuration.Seconds()

	// Handle final state and overflow
	if isFinal || percent >= 1.0 {
		percent = 1.0
		elapsed = b.totalDuration
	}

	filledLength := int(float64(b.barWidth) * percent)

	// Build the visual bar
	var barBuilder strings.Builder
	barBuilder.Grow(b.barWidth + 2)
	barBuilder.WriteString("[")
	barBuilder.WriteString(strings.Repeat(b.graph, filledLength))

	// Show arrow if not complete
	if percent < 1.0 {
		if filledLength < b.barWidth {
			barBuilder.WriteString(b.arrow)
		}
	}

	// Ensure the bar's total visible length is consistent.
	// Calculate remaining space inside the brackets `[]`.
	currentLen := len(strings.Repeat(b.graph, filledLength))
	if percent < 1.0 && filledLength < b.barWidth {
		currentLen += len(b.arrow)
	}
	remainingSpace := b.barWidth - currentLen
	if remainingSpace > 0 {
		barBuilder.WriteString(strings.Repeat(b.space, remainingSpace))
	}

	barBuilder.WriteString("]")

	// Print with carriage return for alignment.
	// Format times with one decimal place for consistency.
	str := fmt.Sprintf("%.1fs / %.1fs %s %.2f%%",
		elapsed.Seconds(),
		b.totalDuration.Seconds(),
		barBuilder.String(),
		percent*100,
	)
	fmt.Printf("\r%s", color.HiBlackString(str))
}

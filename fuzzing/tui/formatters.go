package tui

import (
	"fmt"
	"math/big"
	"strings"
	"time"
)

// formatDuration formats a duration as HH:MM:SS
func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

// formatNumber formats a big.Int as a human-readable string (1.2K, 3.4M, etc.)
func formatNumber(n *big.Int) string {
	if !n.IsUint64() {
		return n.String()
	}
	val := n.Uint64()
	if val < 1000 {
		return fmt.Sprintf("%d", val)
	}
	if val < 1000000 {
		return fmt.Sprintf("%.1fK", float64(val)/1000)
	}
	if val < 1000000000 {
		return fmt.Sprintf("%.1fM", float64(val)/1000000)
	}
	return fmt.Sprintf("%.1fB", float64(val)/1000000000)
}

// formatRate formats a rate with appropriate units
func formatRate(value uint64) string {
	if value < 1000 {
		return fmt.Sprintf("%d/sec", value)
	}
	if value < 1000000 {
		return fmt.Sprintf("%.1fK/sec", float64(value)/1000)
	}
	return fmt.Sprintf("%.1fM/sec", float64(value)/1000000)
}

// formatPercentage formats a percentage with one decimal place
func formatPercentage(numerator, denominator *big.Int) string {
	if denominator.Cmp(big.NewInt(0)) == 0 {
		return "0.0%"
	}
	// Calculate percentage: (numerator / denominator) * 100
	percent := new(big.Float).Quo(
		new(big.Float).SetInt(numerator),
		new(big.Float).SetInt(denominator),
	)
	percent.Mul(percent, big.NewFloat(100))

	percentFloat, _ := percent.Float64()
	return fmt.Sprintf("%.1f%%", percentFloat)
}

// renderProgressBar renders an ASCII progress bar
func renderProgressBar(progress float64, width int) string {
	if progress < 0 {
		progress = 0
	}
	if progress > 1 {
		progress = 1
	}

	filled := int(progress * float64(width))
	if filled > width {
		filled = width
	}
	empty := width - filled

	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)
	return fmt.Sprintf("[%s] %d%%", bar, int(progress*100))
}

// formatBytes formats bytes as human-readable string (KB, MB, GB)
func formatBytes(bytes uint64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	}
	if bytes < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
	}
	if bytes < 1024*1024*1024 {
		return fmt.Sprintf("%.1f MB", float64(bytes)/(1024*1024))
	}
	return fmt.Sprintf("%.1f GB", float64(bytes)/(1024*1024*1024))
}

// truncateString truncates a string to maxLen and adds "..." if needed
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

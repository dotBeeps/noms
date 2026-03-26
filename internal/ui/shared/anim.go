package shared

import "time"

// EaseOutQuad returns the ease-out quadratic value for t in [0,1].
// Fast start, smooth deceleration: t*(2-t).
func EaseOutQuad(t float64) float64 {
	if t >= 1 {
		return 1
	}
	if t <= 0 {
		return 0
	}
	return t * (2 - t)
}

// AnimProgress returns a clamped 0–1 progress value for a time-based animation.
func AnimProgress(start time.Time, duration time.Duration) float64 {
	if duration <= 0 {
		return 1
	}
	t := float64(time.Since(start)) / float64(duration)
	if t < 0 {
		return 0
	}
	if t > 1 {
		return 1
	}
	return t
}

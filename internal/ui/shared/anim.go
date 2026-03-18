package shared

import "time"

// Decay applies exponential decay: value *= factor. Returns the new value and
// whether the animation is still active (above threshold).
func Decay(value, factor, threshold float64) (float64, bool) {
	value *= factor
	if value < threshold {
		return 0, false
	}
	return value, true
}

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

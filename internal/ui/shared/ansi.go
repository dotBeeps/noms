package shared

import "regexp"

// sgrResetRe matches SGR sequences that contain a reset parameter (0 or 49)
// or have no parameters (implicit reset). This catches combined sequences
// like \x1b[0;38;5;240m that literal string replacement would miss.
var sgrResetRe = regexp.MustCompile(`\x1b\[([0-9;]*)m`)

// StabilizeBg re-injects a background color sequence after any SGR reset
// in the line. This prevents the panel background from being cleared when
// lipgloss or other ANSI producers emit reset sequences mid-line.
func StabilizeBg(line, bgSeq string) string {
	return sgrResetRe.ReplaceAllStringFunc(line, func(match string) string {
		// Extract params between \x1b[ and m
		params := match[2 : len(match)-1]

		// Empty params = implicit full reset (\x1b[m)
		if params == "" {
			return match + bgSeq
		}

		// Check if any semicolon-delimited parameter is 0/00 (full reset)
		// or 49 (default background reset).
		for len(params) > 0 {
			end := len(params)
			for i := 0; i < len(params); i++ {
				if params[i] == ';' {
					end = i
					break
				}
			}
			p := params[:end]
			if p == "0" || p == "00" || p == "49" {
				return match + bgSeq
			}
			if end < len(params) {
				params = params[end+1:]
			} else {
				break
			}
		}
		return match
	})
}

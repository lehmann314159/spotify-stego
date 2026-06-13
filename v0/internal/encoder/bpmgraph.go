package encoder

import (
	"fmt"
	"math"
	"strings"
)

// RenderBPMGraphSVG produces an SVG polyline graph of BPM values across the playlist.
func RenderBPMGraphSVG(tracks []Track) string {
	const (
		w, h       = 800, 200
		padL, padR = 50, 20
		padT, padB = 20, 40
	)
	if len(tracks) == 0 {
		return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 800 200"><text x="10" y="20">No tracks</text></svg>`
	}

	minBPM, maxBPM := math.MaxFloat64, -math.MaxFloat64
	for _, t := range tracks {
		if t.BPM > 0 {
			if t.BPM < minBPM {
				minBPM = t.BPM
			}
			if t.BPM > maxBPM {
				maxBPM = t.BPM
			}
		}
	}
	if minBPM == math.MaxFloat64 {
		minBPM, maxBPM = 60, 180
	}
	bpmRange := maxBPM - minBPM
	if bpmRange < 10 {
		bpmRange = 10
		minBPM -= 5
	}

	plotW := float64(w - padL - padR)
	plotH := float64(h - padT - padB)
	n := len(tracks)

	px := func(i int) float64 {
		if n == 1 {
			return float64(padL) + plotW/2
		}
		return float64(padL) + float64(i)*plotW/float64(n-1)
	}
	py := func(bpm float64) float64 {
		return float64(h-padB) - (bpm-minBPM)/bpmRange*plotH
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(
		`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" width="%d" height="%d">`,
		w, h, w, h,
	))
	sb.WriteString(`<style>
  .bpm-grid { stroke: #ddd; stroke-width:1; }
  .bpm-axis { font: 11px sans-serif; fill: #666; }
  .bpm-line { stroke: var(--color-bpm, #4a9eff); stroke-width:2; fill:none; }
  .bpm-dot  { fill: var(--color-bpm, #4a9eff); }
  .bpm-label { font: 9px sans-serif; fill: #333; text-anchor:middle; }
</style>`)

	// Grid lines at 20 BPM intervals
	for bpm := math.Ceil(minBPM/20) * 20; bpm <= maxBPM; bpm += 20 {
		y := py(bpm)
		sb.WriteString(fmt.Sprintf(
			`<line class="bpm-grid" x1="%d" y1="%.1f" x2="%d" y2="%.1f"/>`,
			padL, y, w-padR, y,
		))
		sb.WriteString(fmt.Sprintf(
			`<text class="bpm-axis" x="%d" y="%.1f" text-anchor="end">%.0f</text>`,
			padL-4, y+4, bpm,
		))
	}

	// Polyline
	var pts []string
	for i, t := range tracks {
		bpm := t.BPM
		if bpm == 0 {
			bpm = (minBPM + maxBPM) / 2
		}
		pts = append(pts, fmt.Sprintf("%.1f,%.1f", px(i), py(bpm)))
	}
	sb.WriteString(fmt.Sprintf(`<polyline class="bpm-line" points="%s"/>`, strings.Join(pts, " ")))

	// Dots and truncated labels
	for i, t := range tracks {
		bpm := t.BPM
		if bpm == 0 {
			bpm = (minBPM + maxBPM) / 2
		}
		x, y := px(i), py(bpm)
		sb.WriteString(fmt.Sprintf(`<circle class="bpm-dot" cx="%.1f" cy="%.1f" r="3"/>`, x, y))

		label := t.Title
		if len(label) > 12 {
			label = label[:12] + "…"
		}
		sb.WriteString(fmt.Sprintf(
			`<text class="bpm-label" x="%.1f" y="%.1f" transform="rotate(-45,%.1f,%.1f)">%s</text>`,
			x, float64(h-padB+12), x, float64(h-padB+12), label,
		))
	}

	sb.WriteString(`</svg>`)
	return sb.String()
}

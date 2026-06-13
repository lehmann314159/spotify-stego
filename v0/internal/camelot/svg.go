package camelot

import (
	"fmt"
	"math"
	"strings"
)

// Track is a minimal interface for SVG rendering — avoids import cycle.
type Track struct {
	CamelotCode string
}

// RenderWheelSVG produces a Camelot wheel SVG highlighting visited codes and
// drawing transition arrows between consecutive tracks.
func RenderWheelSVG(tracks []Track) string {
	const (
		cx, cy = 300.0, 300.0
		rOuter = 240.0
		rMid   = 170.0
		rInner = 100.0
		rLabel = 210.0
		rLabelInner = 135.0
	)

	visited := map[string]bool{}
	for _, t := range tracks {
		if t.CamelotCode != "" {
			visited[t.CamelotCode] = true
		}
	}

	var sb strings.Builder
	sb.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 600 600" width="600" height="600">`)
	sb.WriteString(`<style>
  .seg-a { fill: var(--color-seg-a, #e8f4f8); stroke: #999; stroke-width:1; }
  .seg-b { fill: var(--color-seg-b, #f8f0e8); stroke: #999; stroke-width:1; }
  .seg-visited { fill: var(--color-visited, #4a9eff); opacity:0.85; }
  .seg-label { font: bold 12px sans-serif; fill: #333; text-anchor: middle; dominant-baseline: middle; }
  .arrow-compat { stroke: #22aa44; stroke-width:2; fill:none; }
  .arrow-partial { stroke: #ffaa00; stroke-width:2; fill:none; }
  .arrow-none { stroke: #cc3333; stroke-width:2; fill:none; }
</style>`)

	// Draw 24 segments: positions 1-12, letters A (outer) and B (inner)
	for i := 1; i <= 12; i++ {
		for _, letter := range []string{"A", "B"} {
			code := fmt.Sprintf("%d%s", i, letter)
			angle1 := (float64(i-1)/12.0)*2*math.Pi - math.Pi/2 - math.Pi/12
			angle2 := angle1 + 2*math.Pi/12

			var rOut, rIn float64
			var baseClass string
			if letter == "A" {
				rOut, rIn = rOuter, rMid
				baseClass = "seg-a"
			} else {
				rOut, rIn = rMid, rInner
				baseClass = "seg-b"
			}

			x1o := cx + rOut*math.Cos(angle1)
			y1o := cy + rOut*math.Sin(angle1)
			x2o := cx + rOut*math.Cos(angle2)
			y2o := cy + rOut*math.Sin(angle2)
			x1i := cx + rIn*math.Cos(angle1)
			y1i := cy + rIn*math.Sin(angle1)
			x2i := cx + rIn*math.Cos(angle2)
			y2i := cy + rIn*math.Sin(angle2)

			cls := baseClass
			if visited[code] {
				cls += " seg-visited"
			}
			sb.WriteString(fmt.Sprintf(
				`<path class=%q d="M %.2f %.2f A %.2f %.2f 0 0 1 %.2f %.2f L %.2f %.2f A %.2f %.2f 0 0 0 %.2f %.2f Z"/>`,
				cls, x1o, y1o, rOut, rOut, x2o, y2o, x2i, y2i, rIn, rIn, x1i, y1i,
			))

			// Label
			midAngle := (angle1 + angle2) / 2
			var lr float64
			if letter == "A" {
				lr = rLabel
			} else {
				lr = rLabelInner
			}
			lx := cx + lr*math.Cos(midAngle)
			ly := cy + lr*math.Sin(midAngle)
			sb.WriteString(fmt.Sprintf(
				`<text class="seg-label" x="%.2f" y="%.2f">%s</text>`,
				lx, ly, code,
			))
		}
	}

	// Transition arrows
	for i := 1; i < len(tracks); i++ {
		a := tracks[i-1].CamelotCode
		b := tracks[i].CamelotCode
		if a == "" || b == "" {
			continue
		}
		score := Score(a, b)
		cls := "arrow-none"
		if score >= ScoreNeighbor {
			cls = "arrow-compat"
		} else if score >= ScoreDiagonal {
			cls = "arrow-partial"
		}

		ax, ay := segCenter(a, cx, cy, rMid, rOuter)
		bx, by := segCenter(b, cx, cy, rMid, rOuter)
		if ax == 0 && ay == 0 || bx == 0 && by == 0 {
			continue
		}
		// Draw arrow
		sb.WriteString(fmt.Sprintf(
			`<line class=%q x1="%.2f" y1="%.2f" x2="%.2f" y2="%.2f" marker-end="url(#arrowhead)"/>`,
			cls, ax, ay, bx, by,
		))
	}

	// Arrowhead marker
	sb.WriteString(`<defs>
  <marker id="arrowhead" markerWidth="8" markerHeight="6" refX="8" refY="3" orient="auto">
    <polygon points="0 0, 8 3, 0 6" fill="#555"/>
  </marker>
</defs>`)

	sb.WriteString(`</svg>`)
	return sb.String()
}

// segCenter returns the center point of a Camelot segment for arrow placement.
func segCenter(code string, cx, cy, rIn, rOut float64) (float64, float64) {
	num, letter, err := codeToNumber(code)
	if err != nil {
		return 0, 0
	}
	midAngle := (float64(num-1)/12.0)*2*math.Pi - math.Pi/2
	r := (rIn + rOut) / 2
	if letter == "B" {
		r = (rIn*0.6 + rOut*0.4)
	}
	return cx + r*math.Cos(midAngle), cy + r*math.Sin(midAngle)
}

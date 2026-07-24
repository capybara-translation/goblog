// Package healthchart renders daily-average health series as self-contained
// inline SVG line charts (no JavaScript). Pure functions only: given the
// same Chart, Render returns the same markup, which keeps it unit-testable.
//
// Dates are YYYY-MM-DD strings; they are parsed with time.Parse purely for
// day arithmetic (never scanned from the DB), so no timezone issues apply.
package healthchart

import (
	"errors"
	"fmt"
	"html/template"
	"math"
	"strconv"
	"strings"
	"time"
)

// ErrNoData is returned when no series has any points; the caller shows an
// empty-state message instead of an empty chart.
var ErrNoData = errors.New("healthchart: no data")

type Point struct {
	Date  string // YYYY-MM-DD
	Value float64
}

type Series struct {
	Label  string
	Points []Point
}

type Chart struct {
	Title  string
	Unit   string
	From   string // x-axis domain start (YYYY-MM-DD, inclusive)
	To     string // x-axis domain end
	Series []Series
}

// Fixed drawing geometry. The SVG scales responsively via viewBox + CSS.
const (
	chartW  = 720.0
	chartH  = 240.0
	padL    = 48.0 // room for y labels
	padR    = 12.0
	padT    = 12.0
	padB    = 28.0 // room for x labels
	dotR    = 3.0
	dateFmt = "2006-01-02"
)

// Series colors: index 0 red (blood pressure 最高 comes first), index 1 blue.
var seriesColors = [2]string{"#dc2626" /* red-600 */, "#2563eb" /* blue-600 */}

func parseDay(s string) (time.Time, error) { return time.Parse(dateFmt, s) }

// formatValue renders 72.5 as "72.5" and 71.0 as "71". Values are expected
// pre-rounded by the display service; this function does not cap precision.
func formatValue(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}

// niceTicks returns 3-6 evenly spaced round-number ticks covering [min, max].
func niceTicks(min, max float64) []float64 {
	if max <= min {
		max = min + 1
	}
	rawStep := (max - min) / 4
	mag := math.Pow(10, math.Floor(math.Log10(rawStep)))
	var step float64
	for _, m := range []float64{1, 2, 5, 10} {
		step = m * mag
		if step >= rawStep {
			break
		}
	}
	start := math.Floor(min/step) * step
	var ticks []float64
	for v := start; v < max+step; v += step {
		ticks = append(ticks, math.Round(v*1e6)/1e6)
	}
	return ticks
}

// Render returns the chart as inline SVG. Single-series charts get no
// legend; two-series charts (blood pressure) get one.
func Render(c Chart) (template.HTML, error) {
	total := 0
	for _, s := range c.Series {
		total += len(s.Points)
	}
	if total == 0 {
		return "", ErrNoData
	}

	from, err := parseDay(c.From)
	if err != nil {
		return "", fmt.Errorf("healthchart: bad From %q: %w", c.From, err)
	}
	to, err := parseDay(c.To)
	if err != nil {
		return "", fmt.Errorf("healthchart: bad To %q: %w", c.To, err)
	}
	if to.Before(from) {
		return "", fmt.Errorf("healthchart: From %q is after To %q", c.From, c.To)
	}
	daysSpan := to.Sub(from).Hours() / 24
	// For tick count: 0 days (same day) → 1 tick, 1 day diff → 2 ticks, etc.
	minTickCount := int(daysSpan) + 1
	totalDays := daysSpan
	if totalDays < 1 {
		totalDays = 1
	}

	// y domain: data min/max with 5% padding.
	yMin, yMax := math.Inf(1), math.Inf(-1)
	for _, s := range c.Series {
		for _, p := range s.Points {
			yMin = math.Min(yMin, p.Value)
			yMax = math.Max(yMax, p.Value)
		}
	}
	pad := (yMax - yMin) * 0.05
	if pad == 0 {
		pad = math.Abs(yMax) * 0.05
		if pad == 0 {
			pad = 1
		}
	}
	ticks := niceTicks(yMin-pad, yMax+pad)
	yLo, yHi := ticks[0], ticks[len(ticks)-1]

	plotW := chartW - padL - padR
	plotH := chartH - padT - padB
	xFor := func(date string) (float64, error) {
		d, err := parseDay(date)
		if err != nil {
			return 0, fmt.Errorf("healthchart: bad point date %q: %w", date, err)
		}
		frac := d.Sub(from).Hours() / 24 / totalDays
		return padL + frac*plotW, nil
	}
	yFor := func(v float64) float64 {
		return padT + (1-(v-yLo)/(yHi-yLo))*plotH
	}

	var b strings.Builder
	fmt.Fprintf(&b, `<svg viewBox="0 0 %g %g" role="img" aria-label="%s" xmlns="http://www.w3.org/2000/svg" style="width:100%%;height:auto">`,
		chartW, chartH, template.HTMLEscapeString(c.Title))

	// y grid + labels
	for _, tv := range ticks {
		y := yFor(tv)
		fmt.Fprintf(&b, `<line x1="%g" y1="%.1f" x2="%g" y2="%.1f" stroke="#e5e5e5" stroke-width="1"/>`, padL, y, chartW-padR, y)
		fmt.Fprintf(&b, `<text x="%g" y="%.1f" font-size="10" fill="#737373" text-anchor="end">%s</text>`, padL-6, y+3, formatValue(tv))
	}

	// x tick labels: ~6 evenly spaced dates; M/D for spans <= 120 days, Y/M beyond.
	// For short spans, reduce tickCount to avoid duplicate date labels.
	tickCount := 6
	if minTickCount < 6 {
		tickCount = minTickCount
	}
	longSpan := totalDays > 120
	for i := 0; i < tickCount; i++ {
		var d time.Time
		if tickCount == 1 {
			// Single tick: place at the left edge (From date).
			d = from
		} else {
			d = from.Add(time.Duration(float64(i) / float64(tickCount-1) * totalDays * 24 * float64(time.Hour)))
		}
		x := padL
		if tickCount > 1 {
			x = padL + float64(i)/float64(tickCount-1)*plotW
		}
		label := fmt.Sprintf("%d/%d", int(d.Month()), d.Day())
		if longSpan {
			label = fmt.Sprintf("%d/%d", d.Year(), int(d.Month()))
		}
		fmt.Fprintf(&b, `<text x="%.1f" y="%g" font-size="10" fill="#737373" text-anchor="middle">%s</text>`, x, chartH-8, label)
	}

	// series
	for si, s := range c.Series {
		if len(s.Points) == 0 {
			continue
		}
		color := seriesColors[si%len(seriesColors)]
		var pts []string
		for _, p := range s.Points {
			x, err := xFor(p.Date)
			if err != nil {
				return "", err
			}
			pts = append(pts, fmt.Sprintf("%.1f,%.1f", x, yFor(p.Value)))
		}
		fmt.Fprintf(&b, `<polyline points="%s" fill="none" stroke="%s" stroke-width="2"/>`, strings.Join(pts, " "), color)
		for _, p := range s.Points {
			x, _ := xFor(p.Date)
			fmt.Fprintf(&b, `<circle cx="%.1f" cy="%.1f" r="%g" fill="%s"><title>%s: %s%s</title></circle>`,
				x, yFor(p.Value), dotR, color, template.HTMLEscapeString(p.Date), formatValue(p.Value), template.HTMLEscapeString(c.Unit))
		}
	}

	// legend for two-series charts
	if len(c.Series) > 1 {
		lx := padL
		for si, s := range c.Series {
			color := seriesColors[si%len(seriesColors)]
			fmt.Fprintf(&b, `<rect x="%.1f" y="%g" width="10" height="10" fill="%s"/>`, lx, padT, color)
			fmt.Fprintf(&b, `<text x="%.1f" y="%g" font-size="11" fill="#404040">%s</text>`, lx+14, padT+9, template.HTMLEscapeString(s.Label))
			lx += 14 + float64(len([]rune(s.Label)))*11 + 16
		}
	}

	b.WriteString(`</svg>`)
	return template.HTML(b.String()), nil
}

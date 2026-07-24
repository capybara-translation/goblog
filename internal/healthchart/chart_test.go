package healthchart

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func weightChart() Chart {
	return Chart{
		Title: "体重", Unit: "kg", From: "2026-07-01", To: "2026-07-31",
		Series: []Series{{Label: "体重", Points: []Point{
			{Date: "2026-07-01", Value: 72.5},
			{Date: "2026-07-16", Value: 71.0},
			{Date: "2026-07-31", Value: 70.5},
		}}},
	}
}

func TestRender_BasicStructure(t *testing.T) {
	svg, err := Render(weightChart())
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	s := string(svg)
	if !strings.Contains(s, "<svg") || !strings.Contains(s, "viewBox=") {
		t.Error("missing svg/viewBox")
	}
	if got := strings.Count(s, "<circle"); got != 3 {
		t.Errorf("circles = %d, want 3 (one per point)", got)
	}
	if !strings.Contains(s, "<title>2026-07-16: 71kg</title>") {
		t.Errorf("point title missing: %s", s)
	}
	if strings.Count(s, "<polyline") != 1 {
		t.Error("want exactly 1 polyline for 1 series")
	}
}

// x 座標は暦日ベースの線形スケール: 7/1 と 7/16 の間隔は 7/16 と 7/31 の間隔と等しい
func TestRender_CalendarLinearXScale(t *testing.T) {
	svg, _ := Render(weightChart())
	re := regexp.MustCompile(`<circle cx="([0-9.]+)"`)
	ms := re.FindAllStringSubmatch(string(svg), -1)
	if len(ms) != 3 {
		t.Fatalf("found %d circles", len(ms))
	}
	xs := make([]float64, 3)
	for i, m := range ms {
		xs[i], _ = strconv.ParseFloat(m[1], 64)
	}
	gap1, gap2 := xs[1]-xs[0], xs[2]-xs[1]
	if diff := gap1 - gap2; diff > 0.5 || diff < -0.5 {
		t.Errorf("x gaps not equal: %v vs %v (15 days each)", gap1, gap2)
	}
	if xs[0] >= xs[1] || xs[1] >= xs[2] {
		t.Errorf("x not monotonic: %v", xs)
	}
}

func TestRender_TwoSeriesWithLegend(t *testing.T) {
	c := Chart{
		Title: "血圧", Unit: "mmHg", From: "2026-07-01", To: "2026-07-31",
		Series: []Series{
			{Label: "最高", Points: []Point{{Date: "2026-07-01", Value: 120}}},
			{Label: "最低", Points: []Point{{Date: "2026-07-01", Value: 80}}},
		},
	}
	svg, err := Render(c)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	s := string(svg)
	if !strings.Contains(s, "最高") || !strings.Contains(s, "最低") {
		t.Error("legend labels missing for two-series chart")
	}
	if strings.Count(s, "<polyline") != 2 {
		t.Error("want 2 polylines")
	}
}

func TestRender_NoData(t *testing.T) {
	c := Chart{Title: "体重", Unit: "kg", From: "2026-07-01", To: "2026-07-31",
		Series: []Series{{Label: "体重", Points: nil}}}
	if _, err := Render(c); !errors.Is(err, ErrNoData) {
		t.Fatalf("err = %v, want ErrNoData", err)
	}
}

// 手動 DB 挿入（過去データの遡及登録）経由で NaN/±Inf が health_records に
// 混入しても、その点だけ無視され、y 軸ドメインを汚染したり NaN/Inf 文字列が
// 出力に混じったりしないこと。
func TestRender_SkipsNonFiniteValues(t *testing.T) {
	c := Chart{Title: "体重", Unit: "kg", From: "2026-07-01", To: "2026-07-31",
		Series: []Series{{Label: "体重", Points: []Point{
			{Date: "2026-07-01", Value: math.Inf(1)},
			{Date: "2026-07-16", Value: 71.0},
		}}}}
	svg, err := Render(c)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	s := string(svg)
	if got := strings.Count(s, "<circle"); got != 1 {
		t.Errorf("circles = %d, want 1 (Inf point must be dropped)", got)
	}
	if strings.Contains(s, "NaN") || strings.Contains(s, "Inf") {
		t.Errorf("output contains non-finite literal: %s", s)
	}
}

// 全点が非有限値なら、データなし扱い（ErrNoData）で空状態表示に倒す。
func TestRender_AllNonFiniteReturnsErrNoData(t *testing.T) {
	c := Chart{Title: "体重", Unit: "kg", From: "2026-07-01", To: "2026-07-31",
		Series: []Series{{Label: "体重", Points: []Point{
			{Date: "2026-07-01", Value: math.NaN()},
			{Date: "2026-07-16", Value: math.Inf(-1)},
		}}}}
	if _, err := Render(c); !errors.Is(err, ErrNoData) {
		t.Fatalf("err = %v, want ErrNoData", err)
	}
}

// 値のフォーマット: 小数1桁の値は "72.5"、整数値は "72"（"72.0" にしない）
func TestFormatValue(t *testing.T) {
	for _, tc := range []struct {
		in   float64
		want string
	}{{72.5, "72.5"}, {71.0, "71"}, {120, "120"}} {
		if got := formatValue(tc.in); got != tc.want {
			t.Errorf("formatValue(%v) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// y 軸目盛り: データ範囲を覆う「切りのいい」値
func TestNiceTicks(t *testing.T) {
	ticks := niceTicks(70.2, 73.8)
	if len(ticks) < 3 || len(ticks) > 6 {
		t.Fatalf("tick count = %d, want 3-6: %v", len(ticks), ticks)
	}
	if ticks[0] > 70.2 || ticks[len(ticks)-1] < 73.8 {
		t.Errorf("ticks %v do not cover [70.2, 73.8]", ticks)
	}
	for i := 1; i < len(ticks); i++ {
		if fmt.Sprintf("%.6f", ticks[i]-ticks[i-1]) != fmt.Sprintf("%.6f", ticks[1]-ticks[0]) {
			t.Errorf("ticks not evenly spaced: %v", ticks)
		}
	}
}

// x 軸の重複ラベル: 短い span（1日）では、複数のティックが同じ日付をレンダリングしないこと
// また、x-軸ティック位置がすべて有限値で、スケール範囲内であることを検証する
func TestRender_XAxisNoDuplicatesOnShortSpan_SingleDay(t *testing.T) {
	c := Chart{
		Title: "体重", Unit: "kg", From: "2026-07-15", To: "2026-07-15",
		Series: []Series{{Label: "体重", Points: []Point{
			{Date: "2026-07-15", Value: 72.5},
		}}},
	}
	svg, err := Render(c)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	s := string(svg)

	// Extract all x-axis tick labels with their x position (e.g., x="48.5" y="232" ... "7/15")
	tickRe := regexp.MustCompile(`<text x="([^"]*)" y="232" font-size="10" fill="#737373" text-anchor="middle">([^<]+)</text>`)
	matches := tickRe.FindAllStringSubmatch(s, -1)
	if len(matches) == 0 {
		t.Error("no x-axis tick labels found")
		return
	}

	// Verify each tick position is finite and within bounds [padL, chartW-padR]
	const padL, chartW, padR = 48.0, 720.0, 12.0
	xMin, xMax := padL, chartW-padR

	var xs []float64
	labelCounts := make(map[string]int)
	for _, m := range matches {
		xStr, label := m[1], m[2]
		x, err := strconv.ParseFloat(xStr, 64)
		if err != nil {
			t.Errorf("failed to parse x=%q as float64: %v", xStr, err)
			continue
		}
		// Verify x is finite
		if math.IsNaN(x) || math.IsInf(x, 0) {
			t.Errorf("tick position x=%q is not finite (NaN/Inf), label=%q", xStr, label)
		}
		// Verify x is within bounds
		if x < xMin || x > xMax {
			t.Errorf("tick position x=%g is out of bounds [%g, %g], label=%q", x, xMin, xMax, label)
		}
		xs = append(xs, x)
		labelCounts[label]++
	}

	// For a single day, exactly one unique label should appear
	if len(labelCounts) != 1 {
		t.Errorf("x-axis labels have duplicates: %v", labelCounts)
	}

	// The label should appear exactly once
	for label, count := range labelCounts {
		if count != 1 {
			t.Errorf("label %q appears %d times, want 1", label, count)
		}
	}
}

// x 軸の重複ラベル: 3 日間の span では、複数のティックが同じ日付をレンダリングしないこと
func TestRender_XAxisNoDuplicatesOnShortSpan_ThreeDays(t *testing.T) {
	c := Chart{
		Title: "体重", Unit: "kg", From: "2026-07-13", To: "2026-07-15",
		Series: []Series{{Label: "体重", Points: []Point{
			{Date: "2026-07-13", Value: 72.5},
			{Date: "2026-07-14", Value: 72.0},
			{Date: "2026-07-15", Value: 71.5},
		}}},
	}
	svg, err := Render(c)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	s := string(svg)

	// Extract all x-axis tick labels
	tickRe := regexp.MustCompile(`<text x="[^"]*" y="232" font-size="10" fill="#737373" text-anchor="middle">([^<]+)</text>`)
	matches := tickRe.FindAllStringSubmatch(s, -1)
	if len(matches) == 0 {
		t.Error("no x-axis tick labels found")
		return
	}

	// Collect all label texts in order
	var labels []string
	for _, m := range matches {
		labels = append(labels, m[1])
	}

	// Check for duplicate labels
	seen := make(map[string]bool)
	for _, label := range labels {
		if seen[label] {
			t.Errorf("duplicate label %q found in: %v", label, labels)
		}
		seen[label] = true
	}
}

// From <= To validation: Render should error when From is after To
func TestRender_RejectReversedRange(t *testing.T) {
	c := Chart{
		Title: "体重", Unit: "kg", From: "2026-07-31", To: "2026-07-01",
		Series: []Series{{Label: "体重", Points: []Point{
			{Date: "2026-07-15", Value: 72.5},
		}}},
	}
	_, err := Render(c)
	if err == nil {
		t.Error("Render with From > To should error")
	}
	if !strings.Contains(err.Error(), "From") || !strings.Contains(err.Error(), "after To") {
		t.Errorf("error message should mention From/To/after: %v", err)
	}
}

// 有限だが極端に巨大な値: tick 計算 ((max-min)/4 等) が Inf/NaN にオーバーフローし、
// 空 ticks への index out of range でパニックしないこと。空データ状態に倒す。
func TestRender_HugeButFiniteValues_TwoExtremes(t *testing.T) {
	c := Chart{Title: "体重", Unit: "kg", From: "2026-07-01", To: "2026-07-02",
		Series: []Series{{Label: "体重", Points: []Point{
			{Date: "2026-07-01", Value: -1.7e308},
			{Date: "2026-07-02", Value: 1.7e308},
		}}}}
	if _, err := Render(c); !errors.Is(err, ErrNoData) {
		t.Fatalf("err = %v, want ErrNoData", err)
	}
}

// 有限だが極端に巨大な単一値: pad 加算が Inf にオーバーフローし NaN 座標の SVG
// を出力しないこと。空データ状態に倒す。
func TestRender_HugeButFiniteValue_Single(t *testing.T) {
	c := Chart{Title: "体重", Unit: "kg", From: "2026-07-01", To: "2026-07-01",
		Series: []Series{{Label: "体重", Points: []Point{
			{Date: "2026-07-01", Value: 1.7e308},
		}}}}
	svg, err := Render(c)
	if !errors.Is(err, ErrNoData) {
		t.Fatalf("err = %v, want ErrNoData; svg=%s", err, svg)
	}
}

// x 軸の重複ラベル: 長い span（Y/M ラベル、120日超）でも、6分割の等間隔ティックが
// 同じ年月に収まって重複ラベルをレンダリングしないこと（短い span の tickCount
// 削減とは別の重複パターン）。
func TestRender_XAxisNoDuplicatesOnLongSpan_130Days(t *testing.T) {
	c := Chart{
		Title: "体重", Unit: "kg", From: "2026-01-01", To: "2026-05-11",
		Series: []Series{{Label: "体重", Points: []Point{
			{Date: "2026-01-01", Value: 72.5},
			{Date: "2026-05-11", Value: 70.0},
		}}},
	}
	svg, err := Render(c)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	s := string(svg)

	tickRe := regexp.MustCompile(`<text x="[^"]*" y="232" font-size="10" fill="#737373" text-anchor="middle">([^<]+)</text>`)
	matches := tickRe.FindAllStringSubmatch(s, -1)
	if len(matches) == 0 {
		t.Fatal("no x-axis tick labels found")
	}
	seen := make(map[string]bool)
	for _, m := range matches {
		label := m[1]
		if seen[label] {
			t.Errorf("duplicate label %q found in x-axis ticks", label)
		}
		seen[label] = true
	}
}

// 200日 span でも重複なく、かつ 2 種類以上の年月ラベルが残ること（全部潰れて
// 1 ラベルだけになる過剰な dedupe になっていないかの確認）。
func TestRender_XAxisNoDuplicatesOnLongSpan_200Days(t *testing.T) {
	c := Chart{
		Title: "体重", Unit: "kg", From: "2026-01-01", To: "2026-07-19",
		Series: []Series{{Label: "体重", Points: []Point{
			{Date: "2026-01-01", Value: 72.5},
			{Date: "2026-07-19", Value: 70.0},
		}}},
	}
	svg, err := Render(c)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	s := string(svg)

	tickRe := regexp.MustCompile(`<text x="[^"]*" y="232" font-size="10" fill="#737373" text-anchor="middle">([^<]+)</text>`)
	matches := tickRe.FindAllStringSubmatch(s, -1)
	seen := make(map[string]bool)
	distinct := 0
	for _, m := range matches {
		label := m[1]
		if seen[label] {
			t.Errorf("duplicate label %q found in x-axis ticks", label)
			continue
		}
		seen[label] = true
		distinct++
	}
	if distinct < 2 {
		t.Errorf("distinct x-axis label count = %d, want >= 2", distinct)
	}
}

// 凡例は実際に描画された（有限点が1つ以上ある）系列のみを表示すること。
// 全点が非有限値の系列は polyline/circle が描かれないので、凡例にも出すべきでない。
func TestRender_LegendOnlyForDrawnSeries(t *testing.T) {
	c := Chart{
		Title: "血圧", Unit: "mmHg", From: "2026-07-01", To: "2026-07-01",
		Series: []Series{
			{Label: "最高", Points: []Point{{Date: "2026-07-01", Value: 120}}},
			{Label: "最低", Points: []Point{{Date: "2026-07-01", Value: math.Inf(1)}}},
		},
	}
	svg, err := Render(c)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	s := string(svg)
	if got := strings.Count(s, "最高"); got != 1 {
		t.Errorf("count of 最高 in output = %d, want 1 (legend only)", got)
	}
	if strings.Contains(s, "最低") {
		t.Errorf("legend should not include series with zero drawn points: %s", s)
	}
}

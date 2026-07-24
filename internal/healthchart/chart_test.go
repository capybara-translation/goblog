package healthchart

import (
	"errors"
	"fmt"
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

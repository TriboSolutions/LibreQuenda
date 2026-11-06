package analytics

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Charts: when a chart helps, the model picks its kind and gives the SQL of its data
// (show_chart), never the numbers themselves: LibreQuenda runs that SQL like any other
// and the web draws what it returned, so a chart shows exactly what the database says.

// The kinds the web draws (like the indicators' charts).
var chartKinds = []string{"bar", "stacked_bar", "horizontal_bar", "line", "pie", "heatmap"}

const (
	maxChartPoints = 120 // bars, points or slices… (a heatmap: cells)
	maxPieSlices   = 12
	maxHeatCells   = 400
	maxSeries      = 6
)

// ChartInput: what the model asks for.
type ChartInput struct {
	Kind   string `json:"kind"`
	Title  string `json:"title"`
	SQL    string `json:"sql"`
	X      string `json:"x"` // the column of the categories or the time axis (heatmap: the columns of the grid)
	Y      string `json:"y"` // heatmap only: the column of the rows of the grid
	Series []struct {
		Column string `json:"column"`
		Label  string `json:"label"`
	} `json:"series"` // numeric columns; pie and heatmap: exactly one
}

// Series: one numeric column of a chart.
type Series struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

// Chart: what the web draws. Rows: one object per point, x (and y) as text, the
// series as numbers (null when the database gave none).
type Chart struct {
	Kind   string           `json:"kind"`
	Title  string           `json:"title"`
	X      string           `json:"x"`
	Y      string           `json:"y,omitempty"`
	Series []Series         `json:"series"`
	Rows   []map[string]any `json:"rows"`
	SQL    string           `json:"sql"`
}

// chartSchema: show_chart's input, as the model reads it.
var chartSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"kind":  map[string]any{"type": "string", "enum": chartKinds, "description": "bar: compare categories (or a few periods); stacked_bar: parts of a whole per category or period; horizontal_bar: ranking or long category names; line: a trend over time; pie: shares of a total (at most 12 slices); heatmap: a value over two dimensions, e.g. weekday × hour"},
		"title": map[string]any{"type": "string", "description": "Short title in the language of the question, with the period it covers"},
		"sql":   map[string]any{"type": "string", "description": "One SELECT on the analytics views returning the chart's data, already aggregated and ordered as it should be drawn (by date for a line)"},
		"x":     map[string]any{"type": "string", "description": "Column with the categories or the time axis (heatmap: the grid's columns, e.g. hour)"},
		"y":     map[string]any{"type": "string", "description": "heatmap only: column with the grid's rows, e.g. weekday"},
		"series": map[string]any{
			"type":        "array",
			"description": "Numeric columns to draw, each with a label in the language of the question; pie and heatmap: exactly one",
			"items": map[string]any{
				"type":       "object",
				"properties": map[string]any{"column": map[string]any{"type": "string"}, "label": map[string]any{"type": "string"}},
				"required":   []string{"column", "label"},
			},
		},
	},
	"required": []string{"kind", "title", "sql", "x", "series"},
}

const chartDescription = "Shows the person a chart drawn from one read-only SELECT on the analytics views. Use it when they ask for a chart, graph or plot, or when a trend, ranking or share reads better drawn; pick the kind that fits. The SQL must return the data ready to draw: aggregated, one row per point, ordered."

// makeChart checks the model's request, runs its SQL and shapes the result for the web.
// The error says what to fix, for the model.
func makeChart(ctx context.Context, scope Scope, raw json.RawMessage) (*Chart, *Result, error) {
	var in ChartInput
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, nil, errors.New("show_chart's input is not valid JSON")
	}
	in.Title = strings.TrimSpace(in.Title)
	switch {
	case !slices.Contains(chartKinds, in.Kind):
		return nil, nil, fmt.Errorf("kind must be one of %s", strings.Join(chartKinds, ", "))
	case in.Title == "" || utf8.RuneCountInString(in.Title) > 120:
		return nil, nil, errors.New("give a title of up to 120 characters")
	case len(in.Series) == 0 || len(in.Series) > maxSeries:
		return nil, nil, fmt.Errorf("give 1 to %d series", maxSeries)
	case (in.Kind == "pie" || in.Kind == "heatmap") && len(in.Series) != 1:
		return nil, nil, errors.New("a pie or a heatmap draws exactly one series")
	case in.Kind == "heatmap" && in.Y == "":
		return nil, nil, errors.New("a heatmap needs y, the column of its rows")
	}
	var result, err = Run(ctx, scope, in.SQL)
	if err != nil {
		return nil, nil, err
	}
	var limit = maxChartPoints
	switch in.Kind {
	case "pie":
		limit = maxPieSlices
	case "heatmap":
		limit = maxHeatCells
	}
	if len(result.Rows) == 0 {
		return nil, result, errors.New("the SQL returned no rows: there is nothing to draw (say so instead)")
	}
	if len(result.Rows) > limit || result.Truncated {
		return nil, result, fmt.Errorf("%d rows is too many for a %s (at most %d): aggregate further or keep the top ones", len(result.Rows), in.Kind, limit)
	}
	var column = func(name string) (int, error) {
		if i := slices.Index(result.Columns, name); i >= 0 {
			return i, nil
		}
		return 0, fmt.Errorf("the SQL has no column %q (it has %s)", name, strings.Join(result.Columns, ", "))
	}
	var chart = &Chart{Kind: in.Kind, Title: in.Title, X: "x", SQL: in.SQL, Rows: make([]map[string]any, len(result.Rows))}
	var xi, yi int
	if xi, err = column(in.X); err != nil {
		return nil, result, err
	}
	if in.Kind == "heatmap" {
		if yi, err = column(in.Y); err != nil {
			return nil, result, err
		}
		chart.Y = "y"
	}
	var indexes = make([]int, len(in.Series))
	for s, series := range in.Series {
		if indexes[s], err = column(series.Column); err != nil {
			return nil, result, err
		}
		var label = strings.TrimSpace(series.Label)
		if label == "" || utf8.RuneCountInString(label) > 60 {
			label = series.Column
		}
		chart.Series = append(chart.Series, Series{Key: fmt.Sprintf("s%d", s), Label: label})
	}
	for r, row := range result.Rows {
		var point = map[string]any{"x": text(row[xi])}
		if chart.Y != "" {
			point["y"] = text(row[yi])
		}
		for s, i := range indexes {
			var value, ok = number(row[i])
			if !ok {
				return nil, result, fmt.Errorf("column %q is not a number (row %d: %v)", in.Series[s].Column, r+1, row[i])
			}
			point[chart.Series[s].Key] = value
		}
		chart.Rows[r] = point
	}
	return chart, result, nil
}

func text(value any) string {
	switch v := value.(type) {
	case nil:
		return "—"
	case string:
		return v
	default:
		return fmt.Sprint(v)
	}
}

// number: a value as a JSON number (numeric comes as text); nil stays nil.
func number(value any) (any, bool) {
	switch v := value.(type) {
	case nil:
		return nil, true
	case int64:
		return v, true
	case float64:
		return v, true
	case string:
		var f, err = strconv.ParseFloat(v, 64)
		return f, err == nil
	}
	return nil, false
}

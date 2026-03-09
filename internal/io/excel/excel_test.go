package excel

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/msjurset/golars/internal/dataframe"
	"github.com/msjurset/golars/internal/series"
)

func TestWriteAndReadRoundTrip(t *testing.T) {
	df, err := dataframe.New(
		series.NewString("name", []string{"Alice", "Bob", "Charlie"}),
		series.NewInt64("age", []int64{25, 30, 35}),
		series.NewFloat64("score", []float64{88.5, 92.3, 76.1}),
	)
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "test.xlsx")

	if err := WriteFile(path, df); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Verify file exists.
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("file not created: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("file is empty")
	}

	// Read it back.
	df2, err := ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	if df2.Height() != 3 {
		t.Errorf("expected 3 rows, got %d", df2.Height())
	}
	if df2.Width() != 3 {
		t.Errorf("expected 3 columns, got %d", df2.Width())
	}

	// Check column names.
	names := df2.Schema().Names()
	wantNames := []string{"name", "age", "score"}
	for i, want := range wantNames {
		if names[i] != want {
			t.Errorf("column %d: got %q, want %q", i, names[i], want)
		}
	}
}

func TestColLetterToIndex(t *testing.T) {
	tests := []struct {
		col  string
		want int
	}{
		{"A", 0},
		{"B", 1},
		{"Z", 25},
		{"AA", 26},
		{"AB", 27},
		{"AZ", 51},
		{"BA", 52},
	}
	for _, tt := range tests {
		got := colLetterToIndex(tt.col)
		if got != tt.want {
			t.Errorf("colLetterToIndex(%q) = %d, want %d", tt.col, got, tt.want)
		}
	}
}

func TestIndexToColLetter(t *testing.T) {
	tests := []struct {
		idx  int
		want string
	}{
		{0, "A"},
		{1, "B"},
		{25, "Z"},
		{26, "AA"},
		{27, "AB"},
		{51, "AZ"},
		{52, "BA"},
	}
	for _, tt := range tests {
		got := indexToColLetter(tt.idx)
		if got != tt.want {
			t.Errorf("indexToColLetter(%d) = %q, want %q", tt.idx, got, tt.want)
		}
	}
}

func TestCellCol(t *testing.T) {
	tests := []struct {
		ref  string
		want string
	}{
		{"A1", "A"},
		{"B12", "B"},
		{"AB12", "AB"},
		{"Z99", "Z"},
	}
	for _, tt := range tests {
		got := cellCol(tt.ref)
		if got != tt.want {
			t.Errorf("cellCol(%q) = %q, want %q", tt.ref, got, tt.want)
		}
	}
}

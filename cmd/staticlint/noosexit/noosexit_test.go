package noosexit

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	// analysistest.Run ищет файлы в директории testdata
	// Файлы с суффиксом .go, содержащие комментарии "// want", будут проверены
	analysistest.Run(t, "testdata_dir", Analyzer, "a")
}
// Package main представляет собой multichecker для статического анализа проекта.
//
// # Запуск
//
// Для запуска анализатора на всём проекте выполните:
//
//	go run ./cmd/staticlint ./...
//
// # Состав анализаторов
//
// 1. Стандартные анализаторы (golang.org/x/tools/go/analysis/passes):
//   - nilness: проверяет избыточные проверки на nil.
//   - httpresponse: проверяет, что тело HTTP-ответа закрывается.
//   - unusedresult: проверяет неиспользуемые результаты функций.
//   - shadow: обнаруживает затенение переменных.
//
// 2. Анализаторы staticcheck (honnef.co/go/tools/staticcheck):
//   - Класс SA (Static Analysis): SA1000 (регулярки), SA1019 (deprecated), SA4006 и др.
//   - Класс S (Simplify): S1000 (упрощение for-select).
//   - Класс U (Unused): U1000 (неиспользуемый код).
//
// 3. Публичные сторонние анализаторы:
//   - bodyclose (github.com/timakin/bodyclose): проверяет закрытие resp.Body.
//   - errcheck (github.com/kisielk/errcheck): проверяет обработку возвращаемых ошибок.
//
// 4. Собственный анализатор:
//   - noosexit: запрещает прямые вызовы os.Exit в функции main пакета main.
package main

import (
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/analysis/passes/httpresponse"
	"golang.org/x/tools/go/analysis/passes/nilness"
	"golang.org/x/tools/go/analysis/passes/shadow"
	"golang.org/x/tools/go/analysis/passes/unusedresult"

	"honnef.co/go/tools/staticcheck"

	"github.com/kisielk/errcheck/errcheck"
	"github.com/timakin/bodyclose/passes/bodyclose"

	"github.com/SilkovMax/go-musthave-metrics/cmd/staticlint/noosexit"
)

func main() {
	var analyzers []*analysis.Analyzer

	// 1. Стандартные анализаторы
	analyzers = append(analyzers,
		nilness.Analyzer,
		httpresponse.Analyzer,
		unusedresult.Analyzer,
		shadow.Analyzer,
	)

	// 2. Анализаторы staticcheck (SA, S1000, U1000)
	for _, a := range staticcheck.Analyzers {
		var analyzer *analysis.Analyzer

		if newA, ok := interface{}(a).(*analysis.Analyzer); ok {
			analyzer = newA
		} else if oldA, ok := interface{}(a).(interface{ GetAnalyzer() *analysis.Analyzer }); ok {
			analyzer = oldA.GetAnalyzer()
		} else {
			continue
		}

		if strings.HasPrefix(analyzer.Name, "SA") {
			analyzers = append(analyzers, analyzer)
		} else if analyzer.Name == "S1000" || analyzer.Name == "U1000" {
			analyzers = append(analyzers, analyzer)
		}
	}

	// 3. Публичные сторонние анализаторы
	analyzers = append(analyzers,
		bodyclose.Analyzer,
		errcheck.Analyzer, // <-- Заменили paralleltest на errcheck
	)

	// 4. Собственный анализатор
	analyzers = append(analyzers, noosexit.Analyzer)

	// Запуск multichecker
	multichecker.Main(analyzers...)
}
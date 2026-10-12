# internal/repository

Этот пакет содержит реализацию работы с базой данных, а также со внешними сервисами.

Важно, чтобы репозиторий не содержал бизнес-логику.

Репозиторий реализует паттерн Repository и служит абстракцией над различными источниками данных, такими как:
- базы данных (PostgreSQL, MySQL и др.)
- внешние API
- файловые системы
- кэши (Redis, Memcached)
- другие источники данных.


baseline
goos: windows
goarch: amd64
pkg: github.com/SilkovMax/go-musthave-metrics/internal/repository
cpu: AMD Ryzen 5 7500F 6-Core Processor
BenchmarkSetGauge-12                       12195            462498 ns/op            6199 B/op         38 allocs/op
BenchmarkIncrementCounter-12                1857           3153660 ns/op            6205 B/op         38 allocs/op
BenchmarkSetBatch-12                       13099            466253 ns/op            6061 B/op         37 allocs/op
BenchmarkGetAllMetrics-12                 213435             29132 ns/op          153984 B/op       1011 allocs/op
PASS
ok      github.com/SilkovMax/go-musthave-metrics/internal/repository    31.779s


pprof) top 20
Showing nodes accounting for 4490.77kB, 100% of 4490.77kB total
Showing top 20 nodes out of 53
      flat  flat%   sum%        cum   cum%
    2052kB 45.69% 45.69%     2052kB 45.69%  runtime.mallocgc
  902.59kB 20.10% 65.79%  1414.64kB 31.50%  compress/flate.NewWriter (inline)
  512.11kB 11.40% 77.20%   512.11kB 11.40%  net.newFD (inline)
  512.05kB 11.40% 88.60%   512.05kB 11.40%  compress/flate.newHuffmanBitWriter (inline)
  512.02kB 11.40%   100%   512.02kB 11.40%  internal/sync.newEntryNode[go.shape.interface {},go.shape.interface {}] (inline)
         0     0%   100%   512.05kB 11.40%  compress/flate.(*compressor).init
         0     0%   100%  1414.64kB 31.50%  compress/gzip.(*Writer).Write
         0     0%   100%   512.02kB 11.40%  crypto/internal/fips140.init
         0     0%   100%   512.02kB 11.40%  crypto/internal/fips140deps/godebug.Value (inline)
         0     0%   100%  1414.64kB 31.50%  github.com/SilkovMax/go-musthave-metrics/internal/handler.(*UpdateJSONHandler).ServeHTTP
         0     0%   100%  1414.64kB 31.50%  github.com/SilkovMax/go-musthave-metrics/internal/middleware.(*compressWriter).Write
         0     0%   100%  1414.64kB 31.50%  github.com/SilkovMax/go-musthave-metrics/internal/middleware.(*loggingResponseWriter).Write         0     0%   100%  1414.64kB 31.50%  github.com/SilkovMax/go-musthave-metrics/internal/middleware.GzipMiddleware.func1
         0     0%   100%  1414.64kB 31.50%  github.com/go-chi/chi/v5.(*Mux).ServeHTTP
         0     0%   100%  1414.64kB 31.50%  github.com/go-chi/chi/v5.(*Mux).routeHTTP
         0     0%   100%   512.02kB 11.40%  internal/godebug.(*Setting).Value
         0     0%   100%   512.02kB 11.40%  internal/godebug.(*Setting).Value.func1
         0     0%   100%   512.02kB 11.40%  internal/godebug.lookup
         0     0%   100%   512.02kB 11.40%  internal/sync.(*HashTrieMap[go.shape.interface {},go.shape.interface {}]).LoadOrStore
         0     0%   100%   512.11kB 11.40%  main.main








после изменений
$ go tool pprof -top -diff_base=profiles/base.pprof profiles/result.pprof
File: server.exe
Build ID: C:\Users\2BA0~1\AppData\Local\Temp\go-build352198691\b001\exe\server.exe2026-10-13 02:08:00.9925787 +0300 MSK
Type: inuse_space
Time: 2026-10-13 02:04:46 MSK
Showing nodes accounting for -1671.41kB, 37.22% of 4490.77kB total
Dropped 1 node (cum <= 22.45kB)
      flat  flat%   sum%        cum   cum%
 -902.59kB 20.10% 20.10% -1414.64kB 31.50%  compress/flate.NewWriter (inline)
  768.26kB 17.11%  2.99%   768.26kB 17.11%  go.uber.org/zap/zapcore.newCounters (inline)
    -513kB 11.42% 14.41%     -513kB 11.42%  runtime.mallocgc
 -512.05kB 11.40% 25.82%  -512.05kB 11.40%  compress/flate.newHuffmanBitWriter (inline)
 -512.02kB 11.40% 37.22%  -512.02kB 11.40%  internal/sync.newEntryNode[go.shape.interface {},go.shape.interface {}] (inline)
         0     0% 37.22%  -512.05kB 11.40%  compress/flate.(*compressor).init
         0     0% 37.22% -1414.64kB 31.50%  compress/gzip.(*Writer).Write
         0     0% 37.22%  -512.02kB 11.40%  crypto/internal/fips140.init
         0     0% 37.22%   512.02kB 11.40%  crypto/internal/fips140.init.0
         0     0% 37.22% -1414.64kB 31.50%  github.com/SilkovMax/go-musthave-metrics/internal/handler.(*UpdateJSONHandler).ServeHTTP
         0     0% 37.22% -1414.64kB 31.50%  github.com/SilkovMax/go-musthave-metrics/internal/middleware.(*compressWriter).Write
         0     0% 37.22% -1414.64kB 31.50%  github.com/SilkovMax/go-musthave-metrics/internal/middleware.(*loggingResponseWriter).Write         0     0% 37.22% -1414.64kB 31.50%  github.com/SilkovMax/go-musthave-metrics/internal/middleware.GzipMiddleware.func1
         0     0% 37.22% -1414.64kB 31.50%  github.com/go-chi/chi/v5.(*Mux).ServeHTTP
         0     0% 37.22% -1414.64kB 31.50%  github.com/go-chi/chi/v5.(*Mux).routeHTTP
         0     0% 37.22%   768.26kB 17.11%  go.uber.org/zap.(*Logger).WithOptions
         0     0% 37.22%   768.26kB 17.11%  go.uber.org/zap.Config.Build
         0     0% 37.22%   768.26kB 17.11%  go.uber.org/zap.Config.buildOptions.WrapCore.func5
         0     0% 37.22%   768.26kB 17.11%  go.uber.org/zap.Config.buildOptions.func1
         0     0% 37.22%   768.26kB 17.11%  go.uber.org/zap.New
         0     0% 37.22%   768.26kB 17.11%  go.uber.org/zap.optionFunc.apply
         0     0% 37.22%   768.26kB 17.11%  go.uber.org/zap/zapcore.NewSamplerWithOptions
         0     0% 37.22%  -512.02kB 11.40%  internal/sync.(*HashTrieMap[go.shape.interface {},go.shape.interface {}]).LoadOrStore
         0     0% 37.22%   768.26kB 17.11%  main.Initialize
         0     0% 37.22%   768.26kB 17.11%  main.main
         0     0% 37.22% -1414.64kB 31.50%  main.main.HashRequestMiddleware.func3.1
         0     0% 37.22% -1414.64kB 31.50%  main.main.HashResponseMiddleware.func4.1
         0     0% 37.22% -1414.64kB 31.50%  main.main.LoggingMiddleware.func5.1
         0     0% 37.22% -1414.64kB 31.50%  net/http.(*conn).serve
         0     0% 37.22% -1414.64kB 31.50%  net/http.HandlerFunc.ServeHTTP
         0     0% 37.22% -1414.64kB 31.50%  net/http.serverHandler.ServeHTTP
         0     0% 37.22%     -513kB 11.42%  runtime.allocm
         0     0% 37.22%      513kB 11.42%  runtime.findRunnable
         0     0% 37.22%      513kB 11.42%  runtime.injectglist
         0     0% 37.22%      513kB 11.42%  runtime.injectglist.func1
         0     0% 37.22%   768.26kB 17.11%  runtime.main
         0     0% 37.22%     -513kB 11.42%  runtime.mcall
         0     0% 37.22%     -513kB 11.42%  runtime.newm
         0     0% 37.22%     -513kB 11.42%  runtime.newobject
         0     0% 37.22%     -513kB 11.42%  runtime.park_m
         0     0% 37.22%    -1026kB 22.85%  runtime.resetspinning
         0     0% 37.22%     -513kB 11.42%  runtime.schedule
         0     0% 37.22%     -513kB 11.42%  runtime.startm
         0     0% 37.22%    -1026kB 22.85%  runtime.wakep
         0     0% 37.22%  -512.02kB 11.40%  sync.(*Map).LoadOrStore (inline)
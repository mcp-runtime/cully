# Local website benchmark results

Captured 2026-10-14 on macOS 26, Apple M4 Pro, Go 1.26.8, `darwin/arm64`.
These are local measurements, not portable performance guarantees. Each row is
one of three 100-iteration samples. Fixture checks and new server/data-directory
setup are outside the timed region for stateful requests; request creation,
handler execution, recorder creation and state cleanup are inside it.

Command:

```sh
go test ./internal/website/benchmarks -run '^$' -bench . -benchmem -benchtime=100x -count=3 -cpu=1
```

| Benchmark | ns/op (three samples) | B/op | allocs/op |
|---|---:|---:|---:|
| PublishedHandler | 57660, 35885, 34115 | 3956 | 30 |
| PhotoHandler404 | 642.1, 755.0, 629.6 | 1173, 1172, 1172 | 14 |
| PhotoHandler | 18345, 16532, 16591 | 7545 | 29 |
| InstillationHandler (`GET /install.sh`, 200) | 17149, 22757, 16773 | 23876 | 28 |
| SubmitHandler (successful 201) | 243185, 235585, 248285 | 22885, 22889, 22883 | 134 |
| ProfileImportHandler (successful 200) | 19081, 23351, 18178 | 14122, 14127, 14118 | 102 |
| Server | 39178, 41584, 39384 | 56712 | 32 |
| ServerConcurrency | 56386, 53805, 43759 | 61876 | 41 |

The benchmark preflights expected status and response content before timing.
The installer fixture is copied from the repository's root `install.sh`.

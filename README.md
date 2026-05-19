# bq-emulator-468-native-repro

Throwaway native-amd64 reproduction for
[goccy/bigquery-emulator#468](https://github.com/goccy/bigquery-emulator/issues/468)
("v0.7.0: COUNT(*) on empty table returns NULL instead of 0").

GitHub Actions `ubuntu-latest` runners are native `x86_64`, so the
[`verify-468`](.github/workflows/verify.yml) workflow runs the emulator
images without QEMU emulation and probes `COUNT`/`COUNTIF`/`SUM` on an empty
table against `:0.6.6`, `:0.7.0`, `:0.7.1`. See the workflow run logs for the
side-by-side result.

Safe to delete after the result is captured.

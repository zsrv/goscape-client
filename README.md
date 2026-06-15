# goscape-client — rev-244

The Go port of the **RuneScape 2** client, revision 244. This branch holds
the runnable client code.

For the project overview, requirements, full build & run instructions (including
the command-line flag reference and the browser / WebAssembly build), and the
project documentation, see the
**[`main` branch README](https://github.com/zsrv/goscape-client/blob/main/README.md)**.

## Quick start

```bash
go build ./...
go run ./cmd/client          # connect to a local server using default ports
```

See the [`main` README](https://github.com/zsrv/goscape-client/blob/main/README.md)
for all command-line flags and the browser (WebAssembly) build.

## License

Released under the [MIT License](LICENSE). This port builds on the
[Lost City](https://github.com/LostCityRS) project; see [`NOTICE`](NOTICE) for
third-party attribution.

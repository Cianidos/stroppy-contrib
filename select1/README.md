# select1

Requires Go 1.27 or newer.

```sh
go run .
go test ./...
stroppy build .
```

Published files are restored under workload/. Dependencies declared by those files
are resolved through Go modules. Custom workloads are trusted native code, not sandboxed.

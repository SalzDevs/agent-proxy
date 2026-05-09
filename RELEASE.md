# Release Checklist

Use this checklist before tagging a Groxy release.

## 1. Check working tree

```bash
git status
```

Make sure only intended changes are present.

## 2. Format code

```bash
gofmt -w .
```

## 3. Run tests

```bash
go test ./...
```

## 4. Run race tests

```bash
go test -race ./...
```

## 5. Run vet

```bash
go vet ./...
```

## 6. Run benchmarks

```bash
go test -bench=. -benchmem ./...
```

Benchmarks are mainly for spotting major regressions. Results depend on the
machine and environment.

## 7. Review docs

Check:

- `README.md`
- `examples/`
- exported Go doc comments
- `go doc ./...`

## 8. Choose release version

Groxy is currently pre-v1, so prefer tags like:

```text
v0.1.0
v0.2.0
v0.3.0
```

Use semantic versioning:

- patch: bug fixes / docs / small internal changes
- minor: new features or public API changes while pre-v1
- major: reserved for `v1.0.0` and beyond

## 9. Create and push tag

```bash
git tag v0.1.0
git push origin v0.1.0
```

## 10. Verify release

After pushing the tag, verify:

- GitHub Actions passed
- pkg.go.dev can load the module
- install works from another project

```bash
go get github.com/SalzDevs/groxy@v0.1.0
```

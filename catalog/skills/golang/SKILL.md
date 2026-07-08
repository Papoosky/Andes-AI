---
name: golang
description: Go best practices at andespath. Use when writing or reviewing Go code.
---

# Go — andespath

- Errors: wrap with context (`fmt.Errorf("...: %w", err)`), never ignore.
- Table-driven tests for logic with multiple cases.
- Small interfaces, defined on the consumer side.
- `gofmt` and `go vet` clean before committing.

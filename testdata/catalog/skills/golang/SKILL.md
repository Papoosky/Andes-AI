---
name: golang
description: Best practices de Go en andespath. Usar al escribir o revisar código Go.
---

# Go — andespath

- Errores: envolver con contexto (`fmt.Errorf("...: %w", err)`), nunca ignorar.
- Tests table-driven para lógica con múltiples casos.
- Interfaces chicas, definidas del lado del consumidor.
- `gofmt` y `go vet` limpios antes de commitear.

---
description: 'Instructions for writing Go code following idiomatic Go practices and community standards'
applyTo: '**/*.go,**/go.mod,**/go.sum'
---

- Use testify require for assertions in tests, e.g. `require.NoError(t, err)`, `require.Equal(t, expected, actual)`.

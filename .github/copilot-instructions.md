---
description: 'Instructions for writing Go code following idiomatic Go practices and community standards'
applyTo: '**/*.go,**/go.mod,**/go.sum'
---

- When rethrowing an error, wrap it with additional context using `fmt.Errorf` and the `%w` verb, e.g. `return fmt.Errorf("failed to do something: %w", err)`.
- Use testify require for assertions in tests, e.g. `require.NoError(t, err)`, `require.Equal(t, expected, actual)`.

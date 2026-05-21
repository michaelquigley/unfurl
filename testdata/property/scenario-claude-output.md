---
title: Claude Output
tags:
  - fixture
---

# Generated Note

This is a wrapped paragraph that looks like model output
because it keeps returning to the next physical line even
though the author intended a single rendered paragraph.

The document includes ordinary markdown constructs that
should remain structurally stable after unfurl processes the
paragraph content around them.

```go
package main

func main() {
	println("preserved")
}
```

```mermaid
graph TD
  A --> B
```

- this list item has a body that wraps across
  several continuation lines and should become
  one physical line after unfurl
- second item stays separate

> quoted paragraph text also wraps across
> multiple physical lines while keeping its
> blockquote marker.

| name | value |
| --- | ---: |
| alpha | 1 |
| beta | 2 |

[ref]: https://example.com

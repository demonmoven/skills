# Feishu Block Types Reference

## Table of Contents

- [Commonly Used Block Types](#commonly-used-block-types)
- [Read-only / Not Insertable via API](#read-only--not-insertable-via-api)
- [Code Block Language Enum (block_type 14)](#code-block-language-enum-block_type-14)
- [Diagram Board syntax_type (block_type 43)](#diagram-board-syntax_type-block_type-43)
- [Table Width](#table-width)
- [Descendant API Notes](#descendant-api-notes)

---

## Commonly Used Block Types

| block_type | Name       | Insertable via API | Notes |
|------------|------------|--------------------|-------|
| 2          | Text       | ✅ | Plain paragraph |
| 3–11       | Heading1–9 | ✅ | H1=3, H2=4, ... H9=11 |
| 12         | Bullet     | ✅ | Unordered list |
| 13         | Ordered    | ✅ | Ordered list |
| 14         | Code       | ✅ | See language enum below |
| 15         | Quote      | ✅ | Blockquote |
| 17         | Todo       | ✅ | Checkbox / task |
| 22         | Divider    | ✅ | Horizontal rule |
| 27         | Image      | ✅ | 3-step process required |
| 31         | Table      | ✅ | 3-step process; no post-create column width change |
| 43         | Board      | ✅ | Mermaid/PlantUML diagrams; needs extra permission |

## Read-only / Not Insertable via API

| block_type | Name          | Notes |
|------------|---------------|-------|
| 1          | Page          | Document root |
| 16         | Equation      | LaTeX, read-only via API |
| 18         | Bitable       | Multidimensional table embed |
| 19         | Callout       | Highlight block |
| 21         | Diagram       | Legacy diagram embed |
| 23         | File          | File attachment |
| 24/25      | Grid/GridCol  | Layout container |
| 28         | ISV           | Third-party widget |
| 30         | Sheet         | Spreadsheet embed |
| 32         | TableCell     | Auto-created by Table |
| 44         | Board embed   | Old board (43 is the insertable one) |
| 48         | SyncedBlock   | Synced block reference |

## Code Block Language Enum (block_type 14)

| Value | Language     |
|-------|--------------|
| 1     | PlainText    |
| 7     | Bash         |
| 8     | C#           |
| 9     | C++          |
| 10    | C            |
| 12    | CSS          |
| 22    | Go           |
| 24    | HTML         |
| 28    | JSON         |
| 29    | Java         |
| 30    | JavaScript   |
| 32    | Kotlin       |
| 39    | Markdown     |
| 43    | PHP          |
| 49    | Python       |
| 52    | Ruby         |
| 53    | Rust         |
| 56    | SQL          |
| 57    | Scala        |
| 60    | Shell        |
| 61    | Swift        |
| 63    | TypeScript   |
| 66    | XML          |
| 67    | YAML         |

### Full Language Enum (official)

| Value | Language | Value | Language | Value | Language |
|-------|----------|-------|----------|-------|----------|
| 1  | PlainText     | 26 | HTTP          | 51 | RPG           |
| 2  | ABAP          | 27 | Haskell       | 52 | Ruby          |
| 3  | Ada           | 28 | JSON          | 53 | Rust          |
| 4  | Apache        | 29 | Java          | 54 | SAS           |
| 5  | Apex          | 30 | JavaScript    | 55 | SCSS          |
| 6  | Assembly      | 31 | Julia         | 56 | SQL           |
| 7  | Bash          | 32 | Kotlin        | 57 | Scala         |
| 8  | C#            | 33 | LaTeX         | 58 | Scheme        |
| 9  | C++           | 34 | Lisp          | 59 | Scratch       |
| 10 | C             | 35 | Logo          | 60 | Shell         |
| 11 | COBOL         | 36 | Lua           | 61 | Swift         |
| 12 | CSS           | 37 | MATLAB        | 62 | Thrift        |
| 13 | CoffeeScript  | 38 | Makefile      | 63 | TypeScript    |
| 14 | D             | 39 | Markdown      | 64 | VBScript      |
| 15 | Dart          | 40 | Nginx         | 65 | Visual Basic  |
| 16 | Delphi        | 41 | Objective-C   | 66 | XML           |
| 17 | Django        | 42 | OpenEdgeABL   | 67 | YAML          |
| 18 | Dockerfile    | 43 | PHP           | 68 | CMake         |
| 19 | Erlang        | 44 | Perl          | 69 | Diff          |
| 20 | Fortran       | 45 | PostScript    | 70 | Gherkin       |
| 21 | FoxPro        | 46 | PowerShell    | 71 | GraphQL       |
| 22 | Go            | 47 | Prolog        | 72 | GLSL          |
| 23 | Groovy        | 48 | ProtoBuf      | 73 | Properties    |
| 24 | HTML          | 49 | Python        | 74 | Solidity      |
| 25 | HTMLBars      | 50 | R             | 75 | TOML          |

## Diagram Board syntax_type (block_type 43)

| Value | Syntax     | diagram_type guidance |
|-------|------------|-----------------------|
| 1     | PlantUML   | 0=auto, 1=mindmap, 2=sequence, 3=activity, 4=class |
| 2     | Mermaid    | Always use 0 (auto) |
| 3     | SVG        | Raw SVG markup |

Mermaid supported: flowchart, sequenceDiagram, mindmap, pie, classDiagram, erDiagram, stateDiagram-v2, gantt, quadrantChart, xychart-beta

## Table Width

Standard Feishu document table total width: **820px** (measured empirically).

Equal-width formula: `column_width = [820 // cols] * cols`

| Columns | Width each (px) |
|---------|----------------|
| 2 | 410 |
| 3 | 273 |
| 4 | 205 |
| 5 | 164 |
| 6 | 136 |

Minimum column width: 50px. Set via `table.property.column_width: int[]` on creation, or `update_table_property` after creation.

## Descendant API Notes

When using the descendant API (`POST /blocks/{parent}/descendant`):

1. **Every block must include `"children": []`** — even leaf blocks with no children. Omitting this field causes error 1770001.
2. **Divider blocks must include `"divider": {}`** — not just `{"block_type": 22}`.
3. **Quote blocks (type 15) are supported** — they work with the descendant API despite not being listed in some older docs.
4. **Code blocks (type 14) should NOT use descendant API** — they contain `\n` which causes 1770001. Use children API instead.
5. **`block_id` must be unique** per request — use UUID-based IDs.
6. **Max 1000 blocks per request** — use batch_size=200 for safety.
7. **Rate limit: 3 edits/sec per document** — add appropriate sleep between operations.

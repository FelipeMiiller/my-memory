# Canvas Export Reference

`mem export --canvas <note>` writes a subgraph as **JSON Canvas 1.0** — an open format that opens
directly in Obsidian Canvas, and in any other reader that implements the spec.

```bash
mem export --canvas docs/ARCHITECTURE.md --out graph.canvas
```

Use it when a topic is genuinely spatial — a decision map, a dependency cluster, a survey of
alternatives. A canvas complements notes; it does not replace them, because a canvas is not
indexed by MyMemory.

> [!warning]
> **A `.canvas` file is not indexed.** `include` patterns target `.md`. Anything that must be
> findable by search belongs in a note, even if a canvas visualises it.

---

## 1. File structure

```json
{
  "nodes": [],
  "edges": []
}
```

Both arrays are optional. IDs are 16-character lowercase hex strings.

---

## 2. Nodes

Required on every node: `id`, `type`, `x`, `y`, `width`, `height`.

| Type | Extra fields | Use for |
| --- | --- | --- |
| `text` | `text` (Markdown) | Labels, annotations, prose |
| `file` | `file`, optional `subpath` | A note or attachment in the vault |
| `link` | `url` | An external reference |
| `group` | `label`, optional `background`, `backgroundStyle` | A visual container |

```json
{
  "id": "6f0ad84f44ce9c17",
  "type": "file",
  "x": 0, "y": 0, "width": 400, "height": 300,
  "file": "docs/ARCHITECTURE.md",
  "color": "4"
}
```

`subpath` anchors to part of a file: `"subpath": "#Decision Log"`.

Array order is z-index — the first node is the bottom layer.

---

## 3. Edges

| Field | Required | Notes |
| --- | --- | --- |
| `id` | yes | Unique across nodes and edges |
| `fromNode` / `toNode` | yes | Must reference existing node IDs |
| `fromSide` / `toSide` | no | `top` · `right` · `bottom` · `left` |
| `fromEnd` | no | `none` (default) · `arrow` |
| `toEnd` | no | `arrow` (default) · `none` |
| `label` | no | Text on the edge |
| `color` | no | Preset `"1"`–`"6"` or hex |

```json
{
  "id": "0123456789abcdef",
  "fromNode": "6f0ad84f44ce9c17",
  "fromSide": "right",
  "toNode": "a1b2c3d4e5f67890",
  "toSide": "left",
  "label": "depends on"
}
```

---

## 4. Colour presets

`"1"` red · `"2"` orange · `"3"` yellow · `"4"` green · `"5"` cyan · `"6"` purple.
A hex string such as `"#FF0000"` is also accepted. Preset hues are intentionally unspecified —
each application uses its own brand colour.

---

## 5. Layout

- Coordinates may be negative; the canvas is unbounded.
- `x` grows right, `y` grows down; the pair is the top-left corner.
- Leave 50–100 px between nodes and 20–50 px padding inside a group.

| Node | Suggested width | Suggested height |
| --- | --- | --- |
| Small text | 200–300 | 80–150 |
| Medium text | 300–450 | 150–300 |
| Large text | 400–600 | 300–500 |
| File preview | 300–500 | 200–400 |
| Link preview | 250–400 | 100–200 |

---

## 6. Validation

Before shipping an edited canvas, confirm:

1. All `id` values are unique across **both** arrays
2. Every `fromNode` / `toNode` resolves to an existing node
3. Required fields present per type (`text`, `file`, `url`)
4. `type` is one of `text`, `file`, `link`, `group`
5. Sides are `top`/`right`/`bottom`/`left`; ends are `none`/`arrow`
6. Colour is `"1"`–`"6"` or valid hex
7. The file parses as JSON

> [!tip]
> The most common breakage is an unescaped newline inside a text node. In JSON a line break must
> be the two characters `\n`; writing `\\n` renders a literal backslash and `n`.

```bash
# quick structural check
python -c "import json,sys; d=json.load(open('graph.canvas')); ids={n['id'] for n in d['nodes']}; \
assert len(ids)==len(d['nodes']), 'duplicate node id'; \
assert all(e['fromNode'] in ids and e['toNode'] in ids for e in d['edges']), 'dangling edge'"
```

---

## 7. When to reach for a canvas

| Situation | Canvas? |
| --- | --- |
| Comparing three architectural options | yes — the branches are the insight |
| A subsystem's components and their edges | yes |
| A decision with reversals | no — a note with `## Decision Log` is more retrievable |
| Anything a future search must find | no — not indexed |

Rule of thumb: if the value is the **shape** of the information, draw it. If the value is the
**content**, write it down.

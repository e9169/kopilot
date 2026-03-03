---
name: Markdown instructions
description: Rules for writing Markdown files in this project to pass markdownlint checks.
applyTo: "**/*.md"
---

# Markdown Authoring Rules

Follow these rules when creating or updating any `.md` file in this project.

## MD025 — Single top-level heading

Each file must have **exactly one `#` H1 heading**. Jekyll front matter `title:` counts as an implicit H1, so do not add a `# Heading` body heading when a front matter `title` is present.

## MD028 — No blank lines inside blockquotes

Do **not** use a blank (empty) line between two adjacent blockquote groups. If two notices need visual separation, use a `>` separator line within one continuous blockquote, or place a paragraph/HR between separate blockquotes.

**Wrong:**
```markdown
> First notice

> Second notice
```

**Correct (merged with `>` separator):**
```markdown
> First notice
>
> Second notice
```

## MD031 — Blank lines around fenced code blocks

Every fenced code block (` ``` `) must have a blank line both **before and after** it, even inside list items or inside blockquotes.

**Wrong:**
```markdown
**Example:**
```bash
kopilot --version
```
Next paragraph.
```

**Correct:**
```markdown
**Example:**

```bash
kopilot --version
```

Next paragraph.
```

## MD032 — Blank lines around lists

Every bullet or numbered list **must be surrounded by blank lines**. This applies even when the line immediately before the list is a heading, bold label, blockquote continuation, or paragraph ending with `:`.

**Wrong:**
```markdown
**Example prompts:**
- "First item"
- "Second item"
```

**Correct:**
```markdown
**Example prompts:**

- "First item"
- "Second item"
```

## MD033 — No inline HTML

Avoid inline HTML in Markdown files. For Jekyll website files that intentionally use raw HTML for layout purposes, add this at the top of the file (after front matter):

```markdown
<!-- markdownlint-disable MD033 -->
```

Do **not** use `<details>/<summary>` for collapsible sections in the main README or docs — use plain headings and paragraphs instead.

## MD034 — No bare URLs

All URLs must be wrapped in angle brackets or linked text. Never paste a bare URL on its own line.

**Wrong:**
```markdown
For more details, see: https://example.com/docs
```

**Correct:**
```markdown
For more details, see the [documentation](https://example.com/docs).
```

## MD036 — No emphasis as headings

Do **not** use a standalone bold (`**text**`) or italic (`*text*`) line as a section heading. Use actual heading syntax (`###`, `####`, etc.) instead.

**Wrong:**
```markdown
**Quick install (recommended)**

Install with a single command...
```

**Correct:**
```markdown
### Quick install (recommended)

Install with a single command...
```

## MD040 — Fenced code blocks must declare a language

Every fenced code block must specify a language identifier.

**Wrong:**
```markdown
```
❯ /readonly
```
```

**Correct:**
```markdown
```text
❯ /readonly
```
```

Use `bash` for shell commands, `go` for Go code, `text` for terminal session output or prompts, `yaml` for YAML, etc.

## MD060 — Table column separator style

Table separator rows **must include a space on both sides of each dash sequence**, matching the style of the header and data rows.

**Wrong:**
```markdown
| Column A | Column B |
|----------|---------|
| value    | value   |
```

**Correct:**
```markdown
| Column A | Column B |
| -------- | ------- |
| value    | value   |
```

For centred alignment columns:

```markdown
| Header |
| :----: |
| value  |
```

## MD012 — No multiple consecutive blank lines

Use at most one blank line to separate content blocks. Never leave two or more consecutive blank lines.

## General rules

- Use a single `#` H1 at the top of each file — do not repeat it.
- Use fenced code blocks with a language identifier (` ```bash `, ` ```go `, etc.).
- End every file with a single trailing newline.

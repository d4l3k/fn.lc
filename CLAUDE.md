# fn.lc

Personal blog/portfolio of Tristan Rice (d4l3k), built with Hugo. Deployed to
GitHub Pages via `.github/workflows/hugo.yml` on push to `main`.

## Build

```sh
hugo                 # build site into public/
hugo server          # local dev server
make github          # refresh data/github.json + project star counts (needs go)
```

No theme — all templates are in `layouts/`. Styles are inlined in
`layouts/partials/header.html`.

## Layout

- `content/post/` — blog posts. Frontmatter: `title`, `date`.
  Posts synced from external sites (PyTorch blog/devlogs) start with an
  italic `_Originally published on ..._` attribution line and keep their
  images locally under `static/<post-slug>/`.
- `content/project/` — project pages, frontmatter only (`github`, `icon`,
  `stars`, `title`, `weight`, optional `site`). The body is usually empty;
  the description shown on the site comes from `data/github.json`, keyed by
  lowercase `owner/repo` matching the `github` field.
  `weight` = `stars + 1` and drives the "Top Projects" ordering.
  Icons are Font Awesome 4.6 names (used as `fa fa-<icon>`).
- `data/github.json` — snapshot of GitHub repo API objects. Refresh with
  `make github`. It pulls d4l3k's repos and then individually fetches any
  repo a project page references that the user repos endpoint doesn't
  return (pytorch/*, meta-pytorch/*, nwplus/*). Set `GITHUB_TOKEN` to avoid
  the 60 requests/hour unauthenticated limit.
- `static/` — static assets, one directory per post that has images.

## Conventions

- Styles are a single inline `<style>` block in `layouts/partials/header.html`.
  Images go through the `img` shortcode (`{{%/* img src="..." */%}}`), which
  wraps them in `<a class="img-link">` and renders an optional caption from
  the shortcode's inner text. `article img` deliberately bleeds wider than the
  720px text column; `article > p > img` keeps inline markdown images inside it.
- Embedded iframes use `class="embed"` plus an inline `aspect-ratio` so they
  scale with the column width.
- Raw HTML in markdown is enabled via `markup.goldmark.renderer.unsafe` in
  `config.yaml`.
- `relativeURLs: true` so the site works both at https://fn.lc/ and under
  the GitHub Pages project path.
- When adapting the author's existing writing into a blog post, keep the result
  as close to the source as possible. Preserve the original tone, wording, and
  section structure, making only the edits needed for readability and the blog
  format.
- Do not use em dashes in new prose.

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
  `util/fetch_github.go` (d4l3k's repos only) plus manual `gh api repos/...`
  fetches for org-owned repos (pytorch/*, meta-pytorch/*) referenced by
  project pages.
- `static/` — static assets, one directory per post that has images.

## Conventions

- The site still uses AMP-era markup (`<amp-img>` shortcodes, inline CSS) —
  match it rather than modernizing piecemeal.
- Raw HTML in markdown is enabled via `markup.goldmark.renderer.unsafe` in
  `config.yaml`.
- `relativeURLs: true` so the site works both at https://fn.lc/ and under
  the GitHub Pages project path.

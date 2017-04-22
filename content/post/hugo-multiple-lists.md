---
date: 2016-11-03T20:11:16Z
title: "Hugo: Multiple List Views and Grids"
image: "https://www.cs.ubc.ca/sites/cs/files/node_images/career-fair-59_small.jpg"
tags:
  - hugo
  - tcf
  - csss
  - ubc
---

<amp-img src="/images/career-fair-59_small.jpg" width="1024" height="681" layout="responsive"></amp-img>

Yesterday, I decided to take a shot at rewriting the [University of British
Columbia's Technical Career Fair](https://ubctcf.com/) (UBC TCF) website in Hugo. The TCF
is one of the many events
that the [UBC Computer Science Student Society](https://ubccsss.org) puts on every
year and there's been a day-of website for a number of years to allow companies to
find their booths and students to find out about the companies.

The old site was written in a combination of Django and Python and had a small
admin interface. Nothing about it really required to to be dynamic, other than
the interface for adding new companies. I figured it'd be way simpler to just
have every company be represented as a "post" in Hugo.

The porting process was fairly straight forward. The existing template system
mapped pretty much one-to-one onto Hugo and the Go template system, but with
a couple notable exceptions.

### Content Layout

After a bit of consideration I finally decided on the following content layout.

```
content
├── companies
│   ├── company-a.md
│   ├── company-b.md
│   └── tristan-rice.md
├── help.md
├── map.md
└── privacy.md
```

The site is fairly simple and only has a handful of pages. There's a page for
each company, a list of all the companies, a map of where the companies are and
a few other basic pages.

### Multiple List Views

In order to render both the list of companies and the map, I needed to be able
to have multiple list views. To my shock there wasn't an easy way to have
multiple lists of the same content nor any documentation. I finally stumbled upon
[spf13/hugo#148](https://github.com/spf13/hugo/issues/148) an issue that has
been open since December of 2013.

> [spf13 commented on Jan 15, 2014](https://github.com/spf13/hugo/issues/148#issuecomment-32371962)
>
> I believe we have this functionality today already with the top level pages.
> These pages have access to all of the site content and indexes. You can limit
> them with the 'first' function. The only thing missing is that the current
> location is the top level and it looks like you want to put them somewhere
> else.

I investigated a bit more but didn't find anything satisfactory.

My first attempt at implementing a top-level second page was simply putting a new file called `map.html` in `content/`. I had assumed that the Go HTML template engine would be present in all HTML files. This was false.

I ended up creating a basic `map.md` file with just a single param that would
tell the default `single.html` to render a different partial instead.

#### [`content/map.md`](https://github.com/ubccsss/tcf/blob/master/content/map.md)

```yaml
---
map: true
---
```

#### [`layouts/_default/single.html`](https://github.com/ubccsss/tcf/blob/master/layouts/_default/single.html)

```html
{{ if .Params.map }}
  {{ partial "map.html" . }}
{{ else }}
  <!-- Render Default Single Page -->
{{ end }}
```

This renders `map.html` instead of the default single page when it renders
`map.md`.

You can then just iterate over all the pages in the site and then filter by the
specific section you're looking for.

#### [`layouts/partials/map.html`](https://github.com/ubccsss/tcf/blob/master/layouts/partials/map.html)

```html
{{ range where .Site.Pages "Section" "companies" }}
  <!-- example -->
  <a href="{{.RelPermalink}}">.Title</a>
{{ end }}
```

Doing it this way with a flag and multiple `foo.md` files is kind of a pain,
but it allows some very flexible viewing methods. You could easily expand this
to sort by different parameters and orders.

### Grid Based Rendering

<amp-img src="/images/tcf-map.png" width="996" height="692" layout="responsive"></amp-img>

Creating this grid system probably took the most amount of time. The old website had some python code ordering the booths into a two dimensional array and then sending that to the template to render. Since the new format is all done statically using Go templates, that wasn't an option. I needed some way to convert the company markdown files into the desired format.

#### `content/companies/company-a.md`

```
---
title: Company A
website: ...
facebook: ...
twitter: ...
linkedin: ...
email: ...
booth: "14"
---

Some company.
```

My final solution involes using two nested `range` statements for `x` and `y` and then doing some math to convert those back into the correct booth number.

```html
{{ $pages := where .Site.Pages "Section" "companies" }}
{{ $booths := 52 }}
{{ $rows := 4 }}
{{ $columns := div $booths $rows }}
{{ range (seq 0 (sub $columns 1)) }}
  {{ $y := . }}
  {{ range (seq 0 (sub $rows 1)) }}
    {{ $x := . }}
    {{ $i := add (add (mul $x $columns) $y) 1 }}
    <!-- Do something with {{$i}} -->
  {{end}}
{{end}}
```

The final piece of the puzzle is getting the company info matched to the booth,
and if there is no company rendering it as disabled. To find the company, you
have to use `where` to scan all the pages to find the one with matching `booth`
parameter.

```html
{{ $is := printf "%d" $i }}
{{ $page := (index (where $pages "Params.booth" $is) 0)}}
<a id="{{$i}}" class="booth {{if not $page}}disabled{{end}}" href="{{$page.RelPermalink}}">
  <div class="booth-text">{{ $i }}</div>
  <div class="booth-title">{{ $page.Title }}</div>
</a>
```


#### Putting It All Together

```html
{{ $pages := where .Site.Pages "Section" "companies" }}
{{ $booths := 52 }}
{{ $rows := 4 }}
{{ $columns := div $booths $rows }}
{{ range (seq 0 (sub $columns 1)) }}
  {{ $y := . }}
  {{ range (seq 0 (sub $rows 1)) }}
    {{ $x := . }}
    {{ $i := add (add (mul $x $columns) $y) 1 }}
    {{ $is := printf "%d" $i }}
    {{ $page := (index (where $pages "Params.booth" $is) 0)}}
    <a id="{{$i}}" class="booth {{if not $page}}disabled{{end}}" href="{{$page.RelPermalink}}">
      <div class="booth-text">{{ $i }}</div>
      <div class="booth-title">{{ $page.Title }}</div>
    </a>
  {{end}}
{{end}}
```

The alignment of the rows was done by adding some CSS to set each booth to be a fixed `width: 20vmin` and the total row width  to `width: 100vmin`.

There's probably a much easier way of doing all of this.

### Final Code

```
content
├── companies
│   ├── company-a.md
│   ├── company-b.md
│   └── tristan-rice.md
├── help.md
├── map.md
└── privacy.md

layouts
├── companies
│   └── single.html
├── _default
│   ├── list.html
│   └── single.html
├── index.html
└── partials
    ├── footer.html
    ├── header.html
    └── map.html
```

All the source code for the above: https://github.com/ubccsss/tcf


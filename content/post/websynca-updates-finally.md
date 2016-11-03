{
    "slug": "websynca-updates-finally",
    "date": "2014-11-15T08:56:56.000Z",
    "tags": [],
    "title": "WebSyn.ca updates! Finally!",
    "publishdate": "2014-11-15T08:56:56.000Z"
}


I finally got some free time since my midterms are over, and I decided
to work on WebSyn.ca. I fixed a couple of bugs such as fixing file
export.

I also decided to update the overall visual style and update the format
of the file list page. The previous style was pretty terrible and just
an HTML table. The newer version is pretty much the same thing but looks
a bit more like an actual file manager.

![](http://67.media.tumblr.com/c55a6dac9f5c2a9c2bfe44315ea79dce/tumblr_inline_nf2o9diNZO1r3ivit.png)

This is a work in progress, but I think it is much better from a
usability standpoint. It also will allow me to more easily update it to
use JSON for directory navigation and add copy/paste functions.

I also started working on implementing chart support for tables. The
current implementation can only create non-persistent line charts, but
it should be straightforward to fix that.

![](http://67.media.tumblr.com/a307495652a0d37577cad827c8d3515f/tumblr_inline_nf2oetpzKN1r3ivit.png)

These charts are implemented using [Chart.](http://www.chartjs.org/)js.
Chart.js is a great library for simple HTML5 charts. The only draw back
is that it uses canvas, thus on non native zoom levels it can create
blurry charts. It also doesn’t respond well to being resized, but that
should be trivial to fix. One of the biggest benefits of Chart.JS is
that it provides a wide range of chart types with very similar input
formats. That should make it easy to expand beyond line charts. Another
great feature is that it provides interactivity straight out of the box.
This means there are helpful tooltips at each point that provide the
exact value of the data points.

The WebSyn.ca charts.js implementation only requires the tables.js
module. This means that it should work on every document type that
supports inline tables (which is all of them). A future feature would be
to have charts without a table, which would be useful in a presentation.
A work around would be to create the chart, save the canvas image and
delete the table.

Anyway, it felt really good to get some work on WebSyn.ca. I had been
neglecting it for too long due to a heavy freshman work load. Hopefully,
I’ll continue to have time to work on it but I have a bunch of lab exams
and finals in the next few weeks so no guarantees.


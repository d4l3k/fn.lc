{
    "slug": "messagediff",
    "date": "2015-08-11T21:15:38.000Z",
    "tags": [],
    "title": "messagediff",
    "publishdate": "2015-08-11T21:15:38.000Z"
}


A library for doing diffs of arbitrary Golang structs.

<https://github.com/d4l3k/messagediff>\

I put this together because I wanted an easy way to display diffs during
testing. It’s fairly similar to an internal library I used during my
internship this summer.

It’s pretty basic but I’m planning on adding LCS support if I ever get
around to it. It does have support for diffing non-exported fields using
go-spew’s unsafe reflect modifications.


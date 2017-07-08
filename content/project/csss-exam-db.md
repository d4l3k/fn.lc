---
date: 2017-03-09T05:37:09Z
github: ubccsss/exams
icon: database
image: /images/examdb/examdb-home.png
site: https://exams.ubccsss.org
stars: 0
tags:
- ubc
- csss
- go
title: CSSS Exam Database
weight: 1
---
As the webmaster and VP Communications for the [Computer Science Student
Society](https://ubccsss.org/) I spent a lot of time updating the site this past
year. I designed a web app from the ground up to replace the [old
"database"](https://ubccsss.org/services/exams). The old database was merely a
bunch of Drupal pages with manually uploaded files. During my time as webmaster,
I received exactly zero of these.

Check out the [new exam database](https://exams.ubccsss.org/)!
[Reddit Post](https://www.reddit.com/r/UBC/comments/64gjcv/the_csss_exam_database_has_been_redesigned_and/)

I built this using Go. All of the exams are found using a custom webscraper. It
can scrape all of the UBC CS websites, as well as Piazza and the files available
on the department undergrad servers.

On a side note: While scraping I found numerous unsecured files including
hundreds of completed student exams with names, student numbers and grades.
These were responsibly disclosed to the
[UBC CS department](https://www.cs.ubc.ca/).

Once I had the files, I built an interface for viewing files to classify them by
year, term, class, type, and sample. Once I had enough initial data, I fed this
into the Google Prediction API. I initially was working with using a custom
classifier, but it performed significantly (10%+) worse than the out of the box
solution. This was then used to classify the remaining files for display on the
website.

Since the new site was fully interactive, I was also able to add an upload form
for students to upload exams that aren't publicly available.


Here's some screenshots for your viewing.

### Homepage

{{< amp-img src="/images/examdb/examdb-home.png" />}}

### CPSC 121 Course Page

{{< amp-img src="/images/examdb/examdb-course.png" />}}

### Unclassified Files

{{< amp-img src="/images/examdb/examdb-unclassified.png" />}}

### Upload Form

{{< amp-img src="/images/examdb/examdb-upload.png" />}}g" />}}

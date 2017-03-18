---
date: 2017-03-17T19:19:15-07:00
title: nwHacks 2017 Tech Stack
image: /images/www.nwhacks.io.png
---

With [nwHacks 2017](https://www.nwhacks.io) coming up this weekend I figured it
would be a good time to do a writeup of the tech stack and all the different
components that are used to make the hackathon a success. This covers all of the
different components of the stack and what technologies were used.

<!--more-->

## [Main Website](https://www.nwhacks.io)

{{< amp-img src="/images/www.nwhacks.io.png" >}}

[Source Code](https://github.com/nwhacks/nwhacks2017_static)

The site is written in [Polymer](https://www.polymer-project.org/) and hosted on
GitHub with CloudFlare in front of it for secure connections. This makes it
super reliable since it's all static content that can be served. We're using
[polymer-cli](https://www.polymer-project.org/1.0/docs/tools/polymer-cli) to
generate bundled pages and dynamically load them using
[page.js](https://visionmedia.github.io/page.js/). Using webcomponents like this
makes it super easy to manage and highly efficient on load, since the user only
has to load what's needed.

Here's a stripped down version of our routing logic. As you can see `loadPage`
is super simple to understand and does all the dynamic page loading.

```javascript
Polymer({
  is: "main-app",

  /* ... more content here ... */

  ready: function() {
    page('/register', function() { self.route = 'register-closed'; });
    page('/sponsors', function() { self.route = 'sponsor-page'; });
    page('/rsvp/:id/:token/:status', function(e) {
      self.route = 'rsvp-page';
      self.params = e.params;
    });
    page('/', function() { self.route = 'index-page'; });
    page();
  },

  loadPage: function(route) {
    const pageURL = 'components/' + route + '/' + route + '.html';
    this.importHref(pageURL, null, this.handle404, true);
  }
});
```

### Schedule

{{< amp-img src="/images/nwhacks-schedule.png" >}}

[Source Code](https://github.com/nwhacks/nwhacks2017_static/blob/master/components/schedule-box/schedule-box.html)

Similar to last year, the schedule is a reusable element that fetches directly
from Google Calendar. This makes it very easy to add and change events. Hackers
can directly add this calendar to their own as well, making it super easy to
stay on top of what's happening at the hackathon.

### Registration

{{< amp-img src="/images/nwhacks-registration.png" >}}

[Source Code](https://github.com/nwhacks/nwhacks2017_static/tree/master/components/register-form)

This year, we decided to get rid of the Django app that was previously powering
the site. Having a static site greatly simplified deployment and a whole host of
other benefits. The one issue with that is we still need somewhere to store data
such as registration information. We decided to use
[Firebase](https://firebase.google.com/) as it fit our use case very nicely.

On submit, we upload the resume to Google Storage, and then save a new
registration object into Firebase.

### Selecting Hackers

{{< amp-img src="/images/nwhacks-select.png" >}}

[Source Code](https://github.com/nwhacks/nwhacks2017_static/tree/master/components/select-hackers)

Polymer and more specifically
[polymerfire](https://www.webcomponents.org/element/firebase/polymerfire) makes
it super easy to interact with Firebase. Here's a simple example of how we
render all of the registrations.

```html
<iron-list id="list" items="{{registrations}}" as="hacker">
  <template>
    <paper-card heading="[[title(hacker)]]">
      <!-- display the rest of the content -->
    </paper-card>
  </template>
</iron-list>

<firebase-query path="/registrations" data="{{registrations}}">
</firebase-query>
```

In practice, there's an intermediate step between displaying and fetching the
registrations since we need to filter and index it for search. We use
[lunr.js](http://lunrjs.com/) in numerous places for full text search. It's
super fast and we get access instantly to what we're looking for.

Any changes that happen on one screen are immediately viewable on others as
Firebase syncs the data between multiple browsers automatically.

### Statistics

{{< amp-img src="/images/nwhacks-stats.png" >}}

[Source Code](https://github.com/nwhacks/nwhacks2017_static/tree/master/components/stats-page)

We've got an integrated stats page that lets us view statistics about the
hackathon in realtime. It's a really satisfying feeling watching the
counters tick upwards after opening registration. This page also shows a bunch
of aggregate stats such as distributions of T-Shirt sizes and dietary
restrictions.

### Check-In

{{< amp-img src="/images/nwhacks-checkin.png" >}}

[Source Code](https://github.com/nwhacks/nwhacks2017_static/tree/master/components/checkin-page)

This year I built a highly optimized page for check-in. Previously we just used
the selection page above, but that was a lot more complicated than the current
system. You can now do everything through the keyboard and lookup someone in
seconds via full text search. It also syncs between browsers immediately so
there's never any confusion or page reloads required.

The old django system didn't sync between browsers leading to some confusion
during check-in last year.

The new system also makes it very explicit what needs to happen for each person.
A "standard" person has no highlights as seen above. Someone who is under 19, or
on the wait list has a bright orange bar indicating that some extra action needs
to be taken. Someone who was rejected but showed up anyways appears as red
indicating they shouldn't be allowed in.

## Email

{{< amp-img src="/images/nwhacks-email.png" >}}

This year we used [Mailgun](https://mailgun.com) for all of our email. They have
a Go library which made it simple to export data from Firebase and create new
mailing lists for every batch of emails that needed to be sent out. As you can
see above, we send ended up sending a lot of emails to the applicants and
accepted hackers.

They also gave us the benefit of being able to add unsubscribe links easily to
hopefully minimize the amount our email is listed as spam.

All our emails are written in Markdown before being compiled to HTML/text,
wrapped in our email template and being sent to Mailgun.

```md

---
subject: "WE CHOOSE YOU!"
---

Hey %recipient_fname%,

CONGRATULATIONS, YOU’RE IN! We loved your application, and we’d love even more for you to join us at UBC over the weekend of March 18-19!

...

## Please let us know whether you’re coming within the next 7 days [here](https://www.nwhacks.io/rsvp/%recipient.id%#begin).

...

Your friends,

The nwHacks Team
```

The mailing list variables and template language made it super simple to send
out things like RSVPs that required a custom URL for each hacker. We just added
their Firebase registration ID into the mailing list entry.

Unfortunately, the code for sending emails and managing the mailing lists is
closed source currently. In interests of time, I hard coded the credentials last
year and haven't gotten around to scrubbing the Git history.

## Hackathon Communication

### [Slack](https://nwhacks17.slack.com)

{{< amp-img src="/images/nwhacks-slack.png" >}}

We setup a Slack team for all attendees and sponsors to use day of for
communication and announcements. Super easy to setup by exporting data directly
from our admin interface and importing it into Slack.

### Social Media Cross Posting

We also setup a couple of links to automatically post from our Twitter page to
our Facebook page, as well as the Attendees Facebook group and Slack. For
Facebook, we used [Zapier](https://zapier.com) to cross-post the tweets. Zapier
only checks every 15 minutes which makes it a bit less than ideal, but it's very
quick to setup.

## Judging System

{{< amp-img src="/images/nwhacks-voting.png" >}}

[Source Code](https://github.com/nwhacks/gavel)

For judging we used a modified version of [gavel](https://github.com/anishathalye/gavel) themed to match the rest of our site. There's also customizations to display where hackers are, and which sponsor prizes they're applying for. This also has full text search powered by lunr.js.

{{< amp-img src="/images/nwhacks-voting-tables.png" >}}

We setup two instances of the judging application. One for the official judges,
and then one for a popular vote which anyone can use.

## Project Submissions

{{< amp-img src="/images/nwhacks-devpost.png" >}}

We used [Devpost](https://nwhacks2017.devpost.com/) once again despite my
reservations against it. I briefly considered creating my own submission tool,
but decided not to due to time constraints. They don't have any real API so last
year I wrote a scraper to fetch all of the projects off it and import them into
the judging system. This year, we're using the CSV dumps and copy pasting it.

## Final Thoughts

Overall, I'm super happy with how our tech stack worked out. A lot of the
decisions were made based of how reliable the service is and how easy it is to
maintain. Most of the stack is super reliable and doesn't cost us anything. The
day of systems like the judging app, I'm a bit less confident in (especially
considering I didn't write them). However, they only need to be up for about 24
hours and are running on a cheap VM hosted by [Vultr](https://www.vultr.com/).






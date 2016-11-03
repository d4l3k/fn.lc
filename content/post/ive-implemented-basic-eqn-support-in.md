{
    "slug": "ive-implemented-basic-eqn-support-in",
    "date": "2014-05-14T05:20:43.000Z",
    "tags": [],
    "image": "/images/tumblr_n5jtijixpI1r7h2fto2_1280.png",
    "title": "WebSyn.ca Equation Support",
    "publishdate": "2014-05-14T05:20:43.000Z"
}


<amp-img src="/images/tumblr_n5jtijixpI1r7h2fto1_1280.png" width="659" height="216" layout="responsive"></amp-img>



<amp-img src="/images/tumblr_n5jtijixpI1r7h2fto2_1280.png" width="664" height="208" layout="responsive"></amp-img>

I’ve implemented basic `=eqn()` support in WebSync. Right now it just executes
some javascript if the text in the cell starts with `=`. I’ve also added in one
helper function that returns the value of the cell in the format `c("A1")`.

We’ll see how this goes. I’m extremely hesitant to allow people to run
untrusted javascript code on people’s browsers. I might have to add in a
“This document uses untrusted javascript, are you willing to accept any
consequences?” on page load. A lot of the damage the untrusted script
might do is mitigated by the backend. WebSync by default refuses all XHR
requests except to specific endpoints. This should stop all attempts to
damage other documents if they’re using a browser that properly
implements XHR headers (which, as far as I know, is all of them).


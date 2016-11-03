{
    "slug": "hey-look-i-made-a-landing-page",
    "date": "2014-06-11T21:54:26.000Z",
    "tags": [],
    "title": "Hey look! I made a landing page! ",
    "publishdate": "2014-06-11T21:54:26.000Z"
}


<http://fn.lc>

I’m not terribly happy with it. It seems a bit bland and confusing. I’ll
probably add some explanatory text to the top.

Content aside, the setup is kind of neat. It uses erb, scss, and vim to
render the code into html.

“vim -f -n code.js +TOhtml +wq +q“

It’s interesting that you can use vim to modify/export files
programmatically.\
I also setup mina for deployment. It makes pushing a new version of the
site as easy as running “mina deploy”. Thank the FSM for deployment
systems.

I’ve been experimenting with them for deploying WebSyn.ca and it looks
like I’ll give docker another shot since 1.0 just came out. I’m not sure
yet what remote management I’ll use.


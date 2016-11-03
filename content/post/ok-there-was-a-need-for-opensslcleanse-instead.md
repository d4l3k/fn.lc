{
    "slug": "ok-there-was-a-need-for-opensslcleanse-instead",
    "date": "2014-04-18T03:12:21.000Z",
    "tags": [],
    "title": "Ok, there ...",
    "publishdate": "2014-04-18T03:12:21.000Z"
}


> Ok, there was a need for OPENSSL\_cleanse() instead of bzero() to
> prevent supposedly smart compilers from optimizing memory cleanups
> away. Understood.
>
> Ok, in case of an hypothetically super smart compiler,
> OPENSSL\_cleanse() had to be convoluted enough for the compiler not to
> recognize that this was actually bzero() in disguise. Understood.
>
> But then why there had been optimized assembler versions of
> OPENSSL\_cleanse() is beyond me. Did someone not trust the C
> obfuscation?

miod (via [opensslrampage](http://opensslrampage.org/){.tumblr_blog})


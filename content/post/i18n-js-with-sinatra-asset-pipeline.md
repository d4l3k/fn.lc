{
    "slug": "i18n-js-with-sinatra-asset-pipeline",
    "date": "2014-12-18T02:16:00.000Z",
    "tags": [],
    "title": "i18n-js with sinatra-asset-pipeline",
    "publishdate": "2014-12-18T02:16:00.000Z"
}


I just implemented [i18n-js](https://github.com/fnando/i18n-js) support
in WebSync. This came around after realizing my localization support for
the JavaScript front end was lacking.[\
](https://github.com/fnando/i18n-js "i18n-js")

The i18n-js library is super useful and integrates directly with
Sprockets and I18n making it as easy as doing:

```javascript
//= require i18n
//= require i18n/translations

// Some translation
I18n.t('translate-me')
```

However it’s designed for use with Rails and thus doesn’t play nicely
with Sinatra and sinatra-asset-pipeline. While it loaded just fine,
Sprockets couldn’t find the i18n javascript files.

```
Sprockets::FileNotFound - couldn't find file 'i18n'
```

I’m pretty sure this is because of the black magic Rails uses to find
files. My solution was to force i18n-js to load the middleware and add
the static files to the Sprockets files. It’s a huge hack, especially
with using “I18n:JS.method(:config).source\_location” to find their
location.

Here’s the full code:

```ruby
# i18n-js, this is a huge hack to get it to work with sinatra-asset-pipeline
require "i18n/js/middleware"

sprockets.register_preprocessor "application/javascript", :"i18n-js_dependencies" do |context, source|
  if context.logical_path == "i18n/filtered"
    ::I18n.load_path.each {|path| context.depend_on(File.expand_path(path))}
  end
  source
end
i18n_js_location = File.expand_path('../../../app/assets/javascripts',
        I18n::JS.method(:config).source_location[0])
sprockets.append_path i18n_js_location
```

If anyone knows a better way of doing this, I’d love to know.

As a side note, I wish I had designed WebSync as a RoR app from the
start. It would have saved a lot of time hacking Sinatra into being
almost identical to Rails. \*sighs\*


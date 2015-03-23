require 'nokogiri'
require 'pry'

task :default do
    system "sass scss/main.scss public/main.css"
    system "gvim -f -n code.js +TOhtml +wq +q"
    system "mv code.js.html public"
    system "erb index.erb > public/index.html"
end

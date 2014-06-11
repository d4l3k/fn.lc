require 'nokogiri'
require 'pry'

task :default do
    `sass scss/main.scss public/main.css`
    `gvim -f -n code.js +TOhtml +wq +q`
    `mv code.js.html public`
    `erb index.erb > public/index.html`
end
begin
  require 'vlad'
  Vlad.load
rescue LoadError
  # do nothing
end

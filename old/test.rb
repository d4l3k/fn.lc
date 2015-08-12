require 'sinatra'

get '/' do
    `rake`
    File.read("public/index.html")
end

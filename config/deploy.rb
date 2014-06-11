require 'bundler'
Bundler.require

set :application, "fn.lc"
set :domain, "direct.fn.lc"
set :deploy_to, "/srv/http/fn.lc"
set :repository, 'https://github.com/d4l3k/fn.lc.git'


task :deploy do
    deploy do
        invoke :'git:clone'
    end
end

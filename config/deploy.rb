require 'bundler'
Bundler.require

set :application, "fn.lc"
set :domain, "raven.fn.lc"
set :deploy_to, "/srv/http/fn.lc"
set :repository, 'https://github.com/d4l3k/fn.lc.git'


task :deploy do
    deploy do
        invoke :'git:clone'
        queue 'mkdir -p public/lib'
        queue 'ls'
        queue 'npm install --python=/usr/bin/python2'
        queue 'bower install'
        queue 'gulp'
    end
end

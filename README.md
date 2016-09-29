# gae-pastebin
Hi! Here's a pastebin that runs on Google App Engine!

## How to use

 * You'll need to download and extract [Google App Engine SDK](https://cloud.google.com/appengine/downloads) for Go
 * Clone the repository and initialize submodules with:
   - git submodule init
   - git submodule update
 * Now launch the App Engine Development Server and you're good to go!
   - goapp serve gae-pastebin
 * Deploy to your own Google account with:
   - goapp deploy -application [YOUR_PROJECT_ID] -version [YOUR_VERSION_ID]

_Oh, make sure to update static/js/base.js with your own GA user id!_

## ToDo

 * Loading app secrets from external sources
 * Scheduler for deleting old pastes

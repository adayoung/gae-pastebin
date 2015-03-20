# gae-pastebin
Hi! Here's a pastebin that runs on Google App Engine!

Right now it uses the HRD for storage (which kinda sucks) but I'll eventually make it use the Blobstore which is nicer.

## How to use

 * You'll need to download and extract [Google App Engine SDK](https://cloud.google.com/appengine/downloads) for Python
 * Clone the repository and initialize submodules with:
   - git submodule init
   - git submodule update
 * Make a file called local_settings.py in the root directory, put the following line in it:
   - ```SECRET_KEY = '<put a random string here>'```
 * Now launch the App Engine Development Server and you're good to go!

_Make sure to change the application id in app.yaml file before deploying to your own App Engine account!_

# -*- coding: utf-8 -*-

import fixsys

def webapp_add_wsgi_middleware(app):
	from google.appengine.ext.appstats import recording
	app = recording.appstats_wsgi_middleware(app)
	return app

# Timezone offset.  This is used to convert recorded times (which are
# all in UTC) to local time.  The default is US/Pacific winter time.

appstats_TZOFFSET = 0

# Fraction of requests to record.  Set this to a float between 0.0
# and 1.0 to record that fraction of all requests.

appstats_RECORD_FRACTION = 1.0


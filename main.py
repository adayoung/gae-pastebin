# -*- coding: utf-8 -*-

import fixsys
from os import getenv

from webapp2_extras.routes import RedirectRoute
import webapp2

from utils import BaseHandler, handle_404
import pastebin
import logging
import json

class MainHandler(BaseHandler):

	def get(self):
		self.redirect('/pastebin')

class CSPRHandler(BaseHandler):

	def post(self):

		cspr = self.request.body
		try:
			cspr = json.loads(self.request.body)
			logging.error(cspr)
		except:
			pass

app = webapp2.WSGIApplication([

	webapp2.Route(r'/', MainHandler, 'index'),
	webapp2.Route(r'/about', pastebin.About, 'about'),

	RedirectRoute(r'/pastebin', pastebin.PasteBin, name='pastebin', strict_slash=True),
	webapp2.Route(r'/pastebin/<pid:[a-z0-9\.]+>', pastebin.PasteBin, 'pastelink'),
	webapp2.Route(r'/search', pastebin.SearchTags, 'searchtags'),
	webapp2.Route(r'/clean', pastebin.Clean, 'cleanup'),
	webapp2.Route(r'/cspr', CSPRHandler, 'csp_report'),

], debug='Development' in getenv('SERVER_SOFTWARE'))

app.error_handlers[404] = handle_404

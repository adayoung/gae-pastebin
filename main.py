#!/usr/bin/env python
# -*- coding: utf-8 -*-

import fixsys # I have trust issues with utf-8 :-/
from os import getenv

import webapp2

class MainHandler(webapp2.RequestHandler):

	def get(self):
		self.response.write("""
			<!DOCTYPE html>
			<html>
			  <head>
			    <title>Hello, world!</title>
			  </head>
			<body>
			  <h1>Hello, world!</h1>
			</body>
			</html>
		""")

app = webapp2.WSGIApplication([

	webapp2.Route(r'/', MainHandler, 'index'),

], debug='Development' in getenv('SERVER_SOFTWARE'))

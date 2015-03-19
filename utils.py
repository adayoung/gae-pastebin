# -*- coding: utf-8 -*-

import fixsys
from os import getenv

from webapp2_extras.securecookie import SecureCookieSerializer
from google.appengine.api import users
from google.appengine.ext import ndb
from webapp2_extras import jinja2
import webapp2

import uuid

try:
	from local_settings.py import SECRET_KEY
except:
	SECRET_KEY = 'SUPER_SECRET_STRING_HERE' # !!!

DEBUG_MODE = 'Development' in getenv('SERVER_SOFTWARE')

if DEBUG_MODE:
	CSP = "Content-Security-Policy-Report-Only"
else:
	CSP = "Content-Security-Policy"

def jinja2_factory(app):
	j = jinja2.Jinja2(app, {
		'auto_reload': False,
		'trim_blocks': True,
		'lstrip_blocks': True,
		'optimized': True
	})
	j.environment.filters.update({
		# Set filters.
		# ...
	})
	j.environment.globals.update({
		# Set global variables.
		'uri_for': webapp2.uri_for,
		# ...
	})
	return j

def handle_404(request, response, exception):
	rv = jinja2.get_jinja2(factory=jinja2_factory).render_template('404.html')
	response.set_status(404)
	response.write(rv)

class BaseHandler(webapp2.RequestHandler):

	@webapp2.cached_property
	def jinja2(self):
		# Returns a Jinja2 renderer cached in the app registry.
		return jinja2.get_jinja2(factory=jinja2_factory)

	@webapp2.cached_property
	def sc(self):
		return SecureCookieSerializer('%s' % SECRET_KEY)

	def set_csrf(self):
		csrf_token = uuid.uuid4().get_hex()
		csrf_token = self.sc.serialize('csrf_token', csrf_token)
		self.response.set_cookie('csrf_token', csrf_token, httponly=True, overwrite=True, secure='Development' not in getenv('SERVER_SOFTWARE'))
		return csrf_token

	def render_response(self, _template, **context):
		# Renders a template and writes the result to the response.
		rv = self.jinja2.render_template(_template, **context)
		self.response.headers["Ada"] = "*skips about* Hi! <3 ^_^"
		self.response.headers["Strict-Transport-Security"] = "max-age=15552000"
		self.response.headers["X-Content-Type-Options"] = "nosniff"
		# https://developer.mozilla.org/en-US/docs/HTTP/X-Frame-Options
		self.response.headers["X-Frame-Options"] = "SAMEORIGIN"
		if 'msie' in self.request.headers.get('user-agent', '').lower():
			self.response.headers["X-UA-Compatible"] = "IE=edge,chrome=1"
			self.response.headers["X-XSS-Protection"] = "1; mode=block"

		self.response.write(rv)

def get_account():
	user = users.get_current_user()

	user_name = None
	user_id = -1
	if user:
		auth_url = users.create_logout_url('/pastebin')
		user_name = user.email()
		user_id = user.user_id()
	else:
		auth_url = users.create_login_url('/pastebin')

	account = {
		'auth_url': auth_url,
		'name': user_name,
		'user_id': user_id,
		'admin': users.is_current_user_admin(),
	}

	return account
#!/usr/bin/env python
# -*- coding: utf-8 -*-

import fixsys # I have trust issues with utf-8 :-/
import unittest
import webapp2

import main # main.py


class TestHandlers(unittest.TestCase):
	def test_hello(self):
		# Build a request object passing the URI path to be tested.
		# You can also pass headers, query arguments etc.
		request = webapp2.Request.blank('/')
		# Get a response for that request.
		response = request.get_response(main.app)

		# Let's check if the response is correct.
		self.assertEqual(response.status_int, 200)
		self.assertIn('Hello, world!', response.body)

if __name__ == '__main__':
	unittest.main()

#!/usr/bin/env python
# -*- coding: utf-8 -*-

from selenium.webdriver.common.keys import Keys
from selenium import webdriver
import unittest
import time
import os

class UITest(unittest.TestCase):

	def setUp(self):
		username = os.environ["SAUCE_USERNAME"]
		access_key = os.environ["SAUCE_ACCESS_KEY"]

		capabilities = {
			'browserName': 'chrome',
			'platform': 'Windows 10',
			'version': 'beta',
			'recordVideo': False,
			'recordScreenshots': False
		}

		capabilities["build"] = os.environ["TRAVIS_BUILD_NUMBER"]
		capabilities["tags"] = [os.environ["TRAVIS_PYTHON_VERSION"], "CI"]
		capabilities["tunnel-identifier"] = os.environ["TRAVIS_JOB_NUMBER"]

		hub_url = "%s:%s@localhost:4445" % (username, access_key)
		self.browser = webdriver.Remote(desired_capabilities=capabilities, \
			command_executor="http://%s/wd/hub" % hub_url)
		self.browser.implicitly_wait(10)

	def tearDown(self):
		self.browser.quit()

	def get_controls(self):
		browser = self.browser
		controls = {
			'paste_area': browser.find_element_by_id('content'),
			'paste_plain_button': browser.find_element_by_id('label_plain'),
			'paste_html_button': browser.find_element_by_id('label_html'),
			'paste_button': browser.find_element_by_id('paste_btn'),
			'document_body': browser.find_element_by_tag_name('body'),
		}
		return controls

	def test_teh_pastebin(self):
		self.browser.get('http://localhost:8080/')
		self.assertIn('Hello, world!', self.browser.title)

		# Lookie a story!
		"""
		# Ada made a pretty pastebin! It begins with a friendly looking
		# landing page with a large text box for pasting and cute little
		# buttons for posting it either plain-text or html

		self.browser.get('http://localhost:8080/')
		self.assertIn('Pastebin', self.browser.title)

		# Lookie all those pretty controls!
		controls = self.get_controls()

		# Lets try pasting something ^_^
		controls['paste_area'].send_keys('Hello, world!')
		controls['document_body'].send_keys(Keys.CONTROL + Keys.ENTER)

		pasted_text = self.browser.find_element_by_id('content')
		self.assertEqual('Hello, world!', pasted_text)

		# Lets try pasting something colourful!
		self.browser.get('http://localhost:8080/pastebin') # Wha? O.o

		controls = self.get_controls()
		paste_html = \
			'<div style="background:#fff"><h1>Hello, world!</h1></div>'
		controls['paste_area'].send_keys(paste_html)
		controls['document_body'].send_keys(Keys.ALT + Keys.ENTER)

		time.sleep(5)
		## Oops, how do we look at the iframe's content? O.o
		## Let's just open the iframe instead -hides-
		iframe = self.browser.find_element_by_id('content')
		iframe_src = iframe.get_attribute('src')
		self.browser.get(iframe_src)

		pasted_html = self.browser.page_source
		self.assertEqual(paste_html, pasted_html)
		"""

if __name__ == '__main__':
	unittest.main(warnings='ignore')

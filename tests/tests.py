from selenium import webdriver
import unittest
import os

class CheckEnv(unittest.TestCase):

	def setUp(self):
		username = os.environ["SAUCE_USERNAME"]
		access_key = os.environ["SAUCE_ACCESS_KEY"]

		capabilities = {
			'browserName': 'firefox',
			'platform': 'Linux',
			'version': 'beta',
		}

		capabilities["build"] = os.environ["TRAVIS_BUILD_NUMBER"]
		capabilities["tags"] = [os.environ["TRAVIS_PYTHON_VERSION"], "CI"]
		capabilities["tunnel-identifier"] = os.environ["TRAVIS_JOB_NUMBER"]

		hub_url = "%s:%s@localhost:4445" % (username, access_key)
		self.browser = webdriver.Remote(desired_capabilities=capabilities, command_executor="http://%s/wd/hub" % hub_url)
		self.browser.implicitly_wait(3)

	def tearDown(self):
		self.browser.quit()

	def test_selenium(self):
		self.browser.get('http://localhost:8080/')
		self.assertIn('Hello, world!', self.browser.title)

if __name__ == '__main__':
	unittest.main(warnings='ignore')

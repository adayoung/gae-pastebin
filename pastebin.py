# -*- coding: utf-8 -*-

import fixsys
from os import getenv
from traceback import format_exc as p_traceback

from google.appengine.datastore.datastore_query import Cursor
from google.appengine.api import memcache
from google.appengine.ext import deferred
from google.appengine.api import users
from google.appengine.ext import ndb
from markupsafe import escape
import webapp2

from humanize import naturaldelta
import general_counter

from utils import BaseHandler, CSP, get_account
from datetime import datetime, timedelta
# from lxml.html import clean
from hashlib import sha256
import logging
import json

class Paste(ndb.Model):
    """Models an individual paste entry."""
    user_id = ndb.StringProperty(indexed=True)
    title = ndb.StringProperty(indexed=False)
    content = ndb.BlobProperty(indexed=False)
    tags = ndb.StringProperty(repeated=True)
    format = ndb.StringProperty(indexed=False)
    ipaddr = ndb.StringProperty(indexed=False)
    zlib = ndb.BooleanProperty(indexed=False)
    date_published = ndb.DateTimeProperty(auto_now_add=True)
    expired = ndb.BooleanProperty(default=False)

    @classmethod
    def get_paste(self, id, decompress=False):
        paste = self.get_by_id(id)

        if paste is not None:
            paste.paste_id = paste.key.id()

            if paste.zlib and decompress:
                paste.content = paste.content.decode('zlib')

        return paste

class PasteBin(BaseHandler):

    def get(self, **kwargs):
        account = get_account()

        q = kwargs.get('pid')

        if q is not None:

            if q.endswith('.'): # People often end up clicking on links with a . in the end.
                self.redirect(self.request.path[:-1])
                return

            paste = Paste.get_paste(q, decompress=True)
            if paste:

                if paste.expired is True: # don't show expired pastes :o
                    if not users.is_current_user_admin():
                        webapp2.abort(404)

                (counter, last_seen) = (0, datetime.now())
                try:
                    (counter, last_seen) = general_counter.get_count(q)
                except:
                    pass # it's not there

                # Expire it if it's an ancient paste which escaped cleanup
                if len(paste.tags) > 0:
                    expiry = timedelta(days=180)
                else:
                    expiry = timedelta(days=90)

                if datetime.now() - last_seen > expiry:
                    paste.expired = True
                    paste.put()
                    if not users.is_current_user_admin():
                        webapp2.abort(404)

                # https://groups.google.com/d/msg/google-appengine-python/OCCOJNGTx34/TnJgtsE5lNsJ
                self.response.cache_control.no_cache = None
                self.response.cache_control.max_age = 15552000
                # https://groups.google.com/d/msg/google-appengine/ehP9nA6xAAE/6McnTHrgYbkJ
                self.response.headers['Cache-Control'] = 'private, max-age=15552000'

                if self.request.GET.get('download') == 'yes':
                    f_extension = 'txt'
                    f_type = "text/plain; charset=utf-8"
                    if paste.format == 'html':
                        f_extension = 'html'
                        f_type = "text/html; charset=utf-8"
                    self.response.headers[CSP] = "default-src 'none'"
                    self.response.headers["Content-Type"] = f_type
                    self.response.headers["X-Content-Type-Options"] = "nosniff"
                    self.response.headers["Content-Disposition"] = "attachment; filename=\"%s.%s\"" % (str(paste.title or paste.key.id()), f_extension)
                    self.response.write(paste.content)
                    return

                if self.request.GET.get('iframe') == 'yes':
                    self.response.headers[CSP] = "default-src 'none'; style-src 'self' 'unsafe-inline'; img-src *; report-uri /cspr"
                    self.response.headers["X-Content-Type-Options"] = "nosniff"
                    self.response.headers["X-Frame-Options"] = "SAMEORIGIN"
                    self.response.write(paste.content)
                    return

                use_srcdoc = True
                if len(paste.content) > 100 * 1024:
                    use_srcdoc = False

                if paste.tags == [u'']:
                    paste.tags = []

                if self.request.cookies.get('csrf_token'): # remove that cookie if it's not required
                    self.response.delete_cookie('csrf_token')

                csrf_token = None
                if users.is_current_user_admin() or paste.user_id == account.get('user_id'):
                    csrf_token = self.set_csrf()

                self.response.headers[CSP] = "default-src 'self'; font-src netdna.bootstrapcdn.com; script-src 'self' netdna.bootstrapcdn.com ajax.googleapis.com www.google-analytics.com; style-src 'self' netdna.bootstrapcdn.com 'unsafe-inline'; img-src *; object-src 'none'; media-src 'none'; report-uri /cspr"
                self.render_response('paste.html', **{
                    'csrf_token': csrf_token,
                    'paste': paste,
                    'use_srcdoc': use_srcdoc,
                    'counter': counter,
                    'account': account,
                })

                if not users.is_current_user_admin():
                    general_counter.increment(q)

                return
            else:
                webapp2.abort(404)

        csrf_token = self.set_csrf()
        self.response.headers[CSP] = "default-src 'self'; font-src netdna.bootstrapcdn.com; script-src 'self' 'unsafe-inline' netdna.bootstrapcdn.com ajax.googleapis.com www.google-analytics.com; style-src 'self' netdna.bootstrapcdn.com; img-src *; report-uri /cspr"
        self.render_response('pastebin.html', **{
            'paste': {
                    'title': 'Pastebin',
                    'csrf_token': csrf_token,
            },
            'account': account
        })

    def post(self, **kwargs):
        csrf_http = self.sc.deserialize('csrf_token', self.request.cookies.get('csrf_token'))

        if not csrf_http:
            logging.warning("Request aborted - missing token (cookie)")
            webapp2.abort(403)

        csrf_form = self.sc.deserialize('csrf_token', self.request.get('csrf_token'))

        if not csrf_form:
            logging.warning("Request aborted - missing token (form)")
            webapp2.abort(403)

        user = users.get_current_user()
        pid = kwargs.get('pid')

        # https://www.owasp.org/index.php/Cross-Site_Request_Forgery_%28CSRF%29_Prevention_Cheat_Sheet#Double_Submit_Cookies
        if csrf_http != csrf_form:
            logging.warning("Request aborted - token mismatch")
            webapp2.abort(403)

        if self.request.get('delete') == 'yes':
            if not user:
                logging.warning("Request aborted - missing user")
                webapp2.abort(403)

            q = Paste.get_paste(pid)
            if q is not None:
                if q.user_id == user.user_id() or users.is_current_user_admin():
                    logging.info("Deleting paste with paste_id [%s]" % pid)
                    q.expired = True
                    q.put()
            else:
                webapp2.abort(404)

            self.response.write("Paste [%s] was successfully deleted!" % pid)
            return

        content = self.request.get('content') or ''

        if len(content) == 0:
            if self.request.environ.get('HTTP_X_REQUESTED_WITH') == 'XMLHttpRequest':
                self.response.write(str('/pastebin'))
                return
            self.redirect('/pastebin')
            return

        p_zlib = False
        c_content = content
        if len(content) > 950 * 1024:
            c_content = content.encode('zlib')
            p_zlib = True

        paste_id = sha256("%s - %s" % (datetime.now().strftime("%Y-%m-%d %H:%M:%S.%s"), content)).hexdigest()
        paste = Paste.get_paste(paste_id[:8]) # [:8] # possible collision?
        if paste is not None:
            logging.info('Oops, we got a collision with id [%s]' % paste_id[:8])
            paste = Paste.get_paste(paste_id[:9])
            if paste is not None:
                logging.info('Oops, collision with id [%s]. We might end up overwriting paste with id [%s] :o' % (paste_id[:9], paste_id[:10]))
                paste = Paste(id=paste_id[:10]) # if you still get overwritten you're probably unlucky enough to die to a meteorite
                paste_id = paste_id[:10]
            else:
                paste = Paste(id=paste_id[:9]) # O_o
                paste_id = paste_id[:9]
        else:
            paste = Paste(id=paste_id[:8]) # woot!
            paste_id = paste_id[:8]

        paste.zlib = p_zlib
        paste.format = self.request.get('format')

        if user:
            paste.user_id = user.user_id()

        paste.title = self.request.get('title')
        if paste.title is None or not len(paste.title) > 0:
            if paste.format != 'html':
                paste.title = content[:50]
        else:
            paste.title = paste.title[:50]

        supplied_tags = self.request.get('tags') or ''
        supplied_tags = "".join(i for i in supplied_tags if i.isalnum() or i==' ').strip()
        paste.tags = list(set([i.strip().lower() for i in supplied_tags.split(' ') if len(i) < 15][:15]))

        paste.content = str(c_content) # because BadValueError: Expected str, got u''
        paste.ipaddr = self.request.remote_addr
        paste.expired = False

        try:
            logging.info('Creating new paste with paste_id [%s]' % paste_id)
            paste.put()
        except:
            logging.error('Oops, the \'put\' operation failed for this paste. - %s' % p_traceback())
            logging.info('Paste format: %s, length %s' % (paste.format, len(paste.content)))
            webapp2.abort(413, detail="Oops, there was an error with storing your input.")

        if self.request.environ.get('HTTP_X_REQUESTED_WITH') == 'XMLHttpRequest':
            self.response.write(str('/pastebin/%s' % paste_id))
            return

        # http://tools.ietf.org/html/rfc2616#section-10.3.4
        self.response.set_status(303)
        self.response.location = str('/pastebin/%s' % paste_id)

class SearchTags(BaseHandler):

    def get(self):
        account = get_account()

        tags = self.request.GET.get('tags') or ''
        if not len(tags) > 0:
            self.redirect('/pastebin')
            return

        supplied_tags = "".join(i for i in tags if i.isalnum() or i==' ').strip()
        tags = list(set([i.strip().lower() for i in supplied_tags.split(' ') if len(i) < 15][:15]))

        cursor = self.request.GET.get('c')
        if cursor:
            cursor = Cursor(urlsafe=cursor)

        if not len(tags) > 0:
            self.redirect('/pastebin')
            return
        elif len(tags) == 1 and tags[0] == u'':
            self.redirect('/pastebin')
            return

        tags.sort()
        results = []
        q = Paste.query(Paste.tags.IN(tags)).order(-Paste.date_published).order(Paste._key)
        (q_result, q_cursor, q_more) = q.fetch_page(10, start_cursor=cursor)

        for i in q_result:
            i.tags.sort()
            if set(tags).issubset(set(i.tags)) and not i.expired: # don't display expired pastes
                results.append({
                    'paste_id': i.key.id(),
                    'title': i.title or i.key.id(),
                    'tags' : i.tags,
                    'date' : "%s ago" % naturaldelta(datetime.now() - i.date_published),
                    'i_date' : i.date_published.strftime("%b %d, %Y"),
                    'format': i.format
                })

        if q_cursor:
            q_cursor = q_cursor.urlsafe()

        if self.request.cookies.get('csrf_token'): # remove that cookie if it's not required
            self.response.delete_cookie('csrf_token')

        self.response.headers[CSP] = "default-src 'self'; font-src netdna.bootstrapcdn.com; script-src 'self' 'unsafe-inline' netdna.bootstrapcdn.com ajax.googleapis.com www.google-analytics.com; style-src 'self' 'unsafe-inline' netdna.bootstrapcdn.com; img-src *; report-uri /cspr"

        context = {
            'paste': {
                'title': 'Search results for %s' % ', '.join(tags),
                'results': results,
                'tags': tags,
            },
            'account': account,
            'cursor': q_cursor,
            'q_more': q_more,
            'path_qs': self.request.path_qs,
        }

        if self.request.environ.get('HTTP_X_REQUESTED_WITH') == 'XMLHttpRequest':
            context.pop('account') # umm ...
            for i in range(len(context['paste']['results'])):
                context['paste']['results'][i]['title'] = escape(context['paste']['results'][i]['title'])
                for j in range(len(context['paste']['results'][i]['tags'])):
                    context['paste']['results'][i]['tags'][j] = escape(context['paste']['results'][i]['tags'][j])
            self.response.write(json.dumps(context))
            return

        self.render_response('search.html', **context)

class About(BaseHandler):

    def get(self):
        account = get_account()

        self.response.cache_control.no_cache = None
        self.response.cache_control.max_age = 15552000
        self.response.headers['Cache-Control'] = 'public, max-age=15552000'

        if self.request.cookies.get('csrf_token'): # remove that cookie if it's not required
            self.response.delete_cookie('csrf_token')

        self.response.headers[CSP] = "default-src 'self'; font-src netdna.bootstrapcdn.com; script-src 'self' netdna.bootstrapcdn.com ajax.googleapis.com www.google-analytics.com; style-src 'self' 'unsafe-inline' netdna.bootstrapcdn.com; img-src *; report-uri /cspr"
        self.render_response('about.html', **{
            'account': account
        })

class Clean(BaseHandler):

    @classmethod
    def clean(self, admin=False):

        sixmonthsago = datetime.now() - timedelta(days=180)
        o = general_counter.GeneralCounterShard.query(general_counter.GeneralCounterShard.last_viewed < sixmonthsago)
        o = o.fetch(100, keys_only=True)

        p = list(set([i.id().split('-')[1] for i in o])) # derive paste ids
        q = [] # the actual pastes ids to be deleted

        for i in p: # FIXME: This part is totally serial and needs to be _async'd
            num_shards = ndb.Key(general_counter.GeneralCounterShardConfig, i).get()

            if num_shards is not None:
                num_shards = num_shards.num_shards

                add_to_list = True

                counters = [ndb.Key(general_counter.GeneralCounterShard, 'shard-%s-%d' % (i, j)).get_async() for j in range(num_shards)]
                counters = [j.get_result() for j in counters]
                counters = [j for j in counters if j is not None]

                for counter in counters:
                    if counter.last_viewed > sixmonthsago:
                        add_to_list = False

                if add_to_list:
                    q.append(i)

        q = list(set(q)) # remove duplicates
        logging.info("About to delete the following pastes: %s" % str(q))

        if admin:
            return json.dumps(q)
        else:
            r = [ndb.Key(general_counter.GeneralCounterShardConfig, i) for i in q]
            s = [ndb.Key(Paste, i) for i in q]

            o = [i.delete_async() for i in o] # futures
            r = [i.delete_async() for i in r] # futures
            s = [i.delete_async() for i in s] # futures

            o = [i.get_result() for i in o] # results
            r = [i.get_result() for i in r] # results
            s = [i.get_result() for i in s] # results

    def get(self):
        isadmin = users.is_current_user_admin()
        if self.request.environ.get('HTTP_X_APPENGINE_CRON') == 'true' or 'Development' in getenv('SERVER_SOFTWARE') or isadmin:
            if isadmin:
                self.response.write(Clean.clean(admin=True))
            else:
                task = deferred.defer(Clean.clean)
        else:
            self.response.write('This route is available only to admin users.')


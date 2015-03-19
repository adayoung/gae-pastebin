# -*- coding: utf-8 -*-
# Source: https://raw.github.com/GoogleCloudPlatform/appengine-sharded-counters-python/c0641cf7f64288fb819ccf996533a1ab4aa53106/general_counter.py

# Copyright 2008 Google Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

# Some modifications by adayoung

"""A module implementing a general sharded counter."""


import random

from google.appengine.api import memcache
from google.appengine.ext import ndb
from datetime import datetime


SHARD_KEY_TEMPLATE = 'shard-{}-{:d}'


class GeneralCounterShardConfig(ndb.Model):
	"""Tracks the number of shards for each named counter."""
	num_shards = ndb.IntegerProperty(default=20)

	@classmethod
	def all_keys(cls, name):
		"""Returns all possible keys for the counter name given the config.

		Args:
			name: The name of the counter.

		Returns:
			The full list of ndb.Key values corresponding to all the possible
				counter shards that could exist.
		"""
		config = cls.get_or_insert(name)
		shard_key_strings = [SHARD_KEY_TEMPLATE.format(name, index)
							 for index in range(config.num_shards)]
		return [ndb.Key(GeneralCounterShard, shard_key_string)
				for shard_key_string in shard_key_strings]


class GeneralCounterShard(ndb.Model):
	"""Shards for each named counter."""
	count = ndb.IntegerProperty(default=0)
	last_viewed = ndb.DateTimeProperty(auto_now_add=True)


def get_count(name):
	"""Retrieve the count and last_viewed time for a given sharded counter.

	Args:
		name: The name of the counter.

	Returns:
		(Integer, datetime); the cumulative count of all sharded counters for the given
			counter name along with time.
	"""
	total = memcache.get(name)
	last = memcache.get("%s_time" % name) or datetime.now()
	if total is None:
		total = 0

		all_keys = GeneralCounterShardConfig.all_keys(name)
		all_shards = ndb.get_multi(all_keys)

		for counter in all_shards:
			if counter is not None:
				total += counter.count

		all_times = [i.last_viewed for i in all_shards if i is not None]
		if len(all_times) != 0:
			all_times.sort()
			last = all_times[-1]

		memcache.add(name, total, 60)
		memcache.add("%s_time" % name, last, 60)

	return (total, last)


def increment(name):
	"""Increment the value for a given sharded counter.

	Args:
		name: The name of the counter.
	"""
	try:
		config = GeneralCounterShardConfig.get_or_insert(name)
		_increment(name, config.num_shards)
	except:
		pass # don't bother if we've run out of quota or whatnot


@ndb.transactional
def _increment(name, num_shards):
	"""Transactional helper to increment the value for a given sharded counter.

	Also takes a number of shards to determine which shard will be used.

	Args:
		name: The name of the counter.
		num_shards: How many shards to use.
	"""
	index = random.randint(0, num_shards - 1)
	shard_key_string = SHARD_KEY_TEMPLATE.format(name, index)
	counter = GeneralCounterShard.get_by_id(shard_key_string)
	if counter is None:
		counter = GeneralCounterShard(id=shard_key_string)
	counter.count += 1
	try:
		counter.put()
	except:
		pass # don't bother if we've run out of quota or whatnot
	# Memcache increment does nothing if the name is not a key in memcache
	memcache.incr(name)


@ndb.transactional
def increase_shards(name, num_shards):
	"""Increase the number of shards for a given sharded counter.

	Will never decrease the number of shards.

	Args:
		name: The name of the counter.
		num_shards: How many shards to use.
	"""
	config = GeneralCounterShardConfig.get_or_insert(name)
	if config.num_shards < num_shards:
		config.num_shards = num_shards
		config.put()

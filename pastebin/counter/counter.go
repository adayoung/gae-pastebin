// Copyright 2011 Google Inc. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

// Some modifications for gae-pastebin by adayoung

package sharded_counter

import (
	"fmt"
	"math/rand"
	"time"

	"appengine"

	"appengine/datastore"
	"appengine/memcache"
)

type counterConfig struct {
	Shards int
}

type shard struct {
	Name  string
	Count int       `datastore:"count"`
	Last  time.Time `datastore:"last_viewed"`
}

const (
	defaultShards = 20
	configKind    = "GeneralCounterShardConfig"
	shardKind     = "GeneralCounterShard"
)

func memcacheKey(name string) string {
	return shardKind + ":" + name
}

// Count retrieves the value of the named counter.
func Count(ctx appengine.Context, name string) (int, error) {
	total := 0
	mkey := memcacheKey(name)
	if _, err := memcache.JSON.Get(ctx, mkey, &total); err == nil {
		return total, nil
	}
	q := datastore.NewQuery(shardKind).Filter("Name =", name)
	for t := q.Run(ctx); ; {
		var s shard
		_, err := t.Next(&s)
		if err == datastore.Done {
			break
		}
		if err != nil {
			return total, err
		}
		total += s.Count
	}
	memcache.JSON.Set(ctx, &memcache.Item{
		Key:        mkey,
		Object:     &total,
		Expiration: 60,
	})
	return total, nil
}

// Last retrieves the latest timestamp value across all counter shards for a given name
func Last(ctx appengine.Context, name string) (time.Time, error) {
	// I think we can do this with q.Order('-last_viewed').Limit(1) or something too
	last := time.Now().AddDate(0, 0, -180) // let's start from six months ago
	q := datastore.NewQuery(shardKind).Filter("Name =", name)
	for t := q.Run(ctx); ; {
		var s shard
		_, err := t.Next(&s)
		if err == datastore.Done {
			break
		}
		if err != nil {
			return last, err
		}
		if s.Last.After(last) {
			last = s.Last
		}
	}
	return last, nil
}

// Increment increments the named counter.
func Increment(ctx appengine.Context, name string) error {
	// Get counter config.
	var cfg counterConfig
	ckey := datastore.NewKey(ctx, configKind, name, 0, nil)
	err := datastore.RunInTransaction(ctx, func(ctx appengine.Context) error {
		err := datastore.Get(ctx, ckey, &cfg)
		if err == datastore.ErrNoSuchEntity {
			cfg.Shards = defaultShards
			_, err = datastore.Put(ctx, ckey, &cfg)
		}
		return err
	}, nil)
	if err != nil {
		return err
	}
	var s shard
	err = datastore.RunInTransaction(ctx, func(ctx appengine.Context) error {
		shardName := fmt.Sprintf("%s-shard%d", name, rand.Intn(cfg.Shards))
		key := datastore.NewKey(ctx, shardKind, shardName, 0, nil)
		err := datastore.Get(ctx, key, &s)
		// A missing entity and a present entity will both work.
		if err != nil && err != datastore.ErrNoSuchEntity {
			return err
		}
		s.Name = name
		s.Count++
		s.Last = time.Now() // ~adayoung was here -dances about-
		_, err = datastore.Put(ctx, key, &s)
		return err
	}, nil)
	if err != nil {
		return err
	}
	memcache.IncrementExisting(ctx, memcacheKey(name), 1)
	return nil
}

// IncreaseShards increases the number of shards for the named counter to n.
// It will never decrease the number of shards.
func IncreaseShards(ctx appengine.Context, name string, n int) error {
	ckey := datastore.NewKey(ctx, configKind, name, 0, nil)
	return datastore.RunInTransaction(ctx, func(ctx appengine.Context) error {
		var cfg counterConfig
		mod := false
		err := datastore.Get(ctx, ckey, &cfg)
		if err == datastore.ErrNoSuchEntity {
			cfg.Shards = defaultShards
			mod = true
		} else if err != nil {
			return err
		}
		if cfg.Shards < n {
			cfg.Shards = n
			mod = true
		}
		if mod {
			_, err = datastore.Put(ctx, ckey, &cfg)
		}
		return err
	}, nil)
}
package models

import (
	// Google Appengine Packages
	"appengine"
	"appengine/datastore"
)

type BatchCleanQ struct {
	BatchID string `datastore:"batch_id"`
}

func NewBatchCleanQ(c appengine.Context, batch_id string) error {
	batch_cq := new(BatchCleanQ)
	batch_cq.BatchID = batch_id

	key := datastore.NewKey(c, "BatchCleanQ", batch_id, 0, nil)
	_, err := datastore.Put(c, key, batch_cq)
	return err
}

func BatchCleaner(c appengine.Context) error {
	batch_cqr := datastore.NewQuery("BatchCleanQ").KeysOnly().Limit(1)
	batchcq_keys, err := batch_cqr.GetAll(c, nil)
	if err == nil {
		for _, batchcq_key := range batchcq_keys {
			batch_id := batchcq_key.StringID()
			pastes_404s, berr := datastore.NewQuery(PasteDSKind).
				Filter("batch_id =", batch_id).KeysOnly().Limit(100).GetAll(c, nil)
			if berr == nil {
				if len(pastes_404s) > 0 {
					if cerr := datastore.DeleteMulti(c, pastes_404s); cerr != nil {
						return cerr
					}
				} else {
					if cerr := datastore.DeleteMulti(c, batchcq_keys); cerr != nil {
						return cerr
					}
				}
			} else {
				return berr
			}
		}
	}
	return err
}

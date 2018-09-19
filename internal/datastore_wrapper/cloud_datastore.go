package datastore_wrapper

import (
	"context"

	"cloud.google.com/go/datastore"
)

type CloudDatastore struct {
	d *datastore.Client
}

func NewCloudDatastore(ctx context.Context, projectID string) (*CloudDatastore, error) {

	ds, err := datastore.NewClient(ctx, projectID)
	if err != nil {
		return &CloudDatastore{}, err
	}
	defer ds.Close()

	return &CloudDatastore{
		d: ds,
	}, nil
}

func (c *CloudDatastore) Get(ctx context.Context, k *datastore.Key, i interface{}) error {
	return c.d.Get(ctx, k, &i)
}

func (c *CloudDatastore) Put(ctx context.Context, k *datastore.Key, i interface{}) (*datastore.Key, error) {
	return c.d.Put(ctx, k, &i)
}

func (c *CloudDatastore) GetAll(ctx context.Context, q *datastore.Query, i interface{}) ([]*datastore.Key, error) {
	return c.d.GetAll(ctx, q, &i)
}

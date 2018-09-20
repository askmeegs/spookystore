// Copyright 2018 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Wraps Cloud Datastore in order to generate Mocks for unit tests
// following precedent set by: https://github.com/GoogleCloudPlatform/google-cloud-go/issues/644

package datastore_wrapper

import (
	"context"

	"cloud.google.com/go/datastore"
)

type CloudDatastore struct {
	D *datastore.Client
}

func NewCloudDatastore(projectID string) (*CloudDatastore, context.Context, error) {
	ctx := context.Background()
	ds, err := datastore.NewClient(ctx, projectID)
	if err != nil {
		return &CloudDatastore{}, ctx, err
	}
	return &CloudDatastore{
		D: ds,
	}, ctx, nil
}

func (c *CloudDatastore) Get(ctx context.Context, k *datastore.Key, i interface{}) error {
	return c.D.Get(ctx, k, i)
}

func (c *CloudDatastore) Put(ctx context.Context, k *datastore.Key, i interface{}) (*datastore.Key, error) {
	return c.D.Put(ctx, k, i)
}

func (c *CloudDatastore) GetAll(ctx context.Context, q *datastore.Query, i interface{}) ([]*datastore.Key, error) {
	return c.D.GetAll(ctx, q, i)
}

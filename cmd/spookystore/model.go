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

package main

import (
	"cloud.google.com/go/datastore"
	pb "github.com/m-okeefe/spookystore/internal/proto"
)

type Server struct {
	ds *datastore.Client
}

type User struct {
	K *datastore.Key `datastore:"__key__"`

	GoogleID             string            `datastore:"GoogleID"`
	ID                   string            `datastore:"ID"`
	DisplayName          string            `datastore:"DisplayName"`
	Picture              string            `datastore:"Picture"`
	Cart                 *pb.Cart          `datastore:"Cart"`
	Transactions         []*pb.Transaction `datastore:"Transactions"`
	Email                string            `datastore:"Email"`
	XXX_NoUnkeyedLiteral struct{}          `datastore:"XXX_NoUnkeyedLiteral"`
	XXX_unrecognized     []byte            `datastore:"XXX_unrecognized"`
	XXX_sizecache        int32             `datastore:"XXX_sizecache"`
}

type Product struct {
	K                    *datastore.Key `datastore:"__key__"`
	ID                   string         `datastore:"ID"`
	DisplayName          string         `datastore:"DisplayName"`
	PictureURL           string         `datastore:"PictureURL"`
	Cost                 float32        `datastore:"Cost"`
	Description          string         `datastore:"Description"`
	XXX_NoUnkeyedLiteral struct{}       `datastore:"XXX_NoUnkeyedLiteral"`
	XXX_unrecognized     []byte         `datastore:"XXX_unrecognized"`
	XXX_sizecache        int32          `datastore:"XXX_sizecache"`
}

type TransactionCounter struct {
	NumTransactions int32 `datastore:"NumTransactions"`
}

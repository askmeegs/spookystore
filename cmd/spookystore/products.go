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
	"flag"
	"fmt"
	"os"
	"strconv"

	"cloud.google.com/go/datastore"
	"golang.org/x/net/context"
	"google.golang.org/grpc/grpclog"

	"github.com/m-okeefe/spookystore/cmd/version"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var (
	projectID = flag.String("google-project-id", "", "google cloud project id")
	addr      = flag.String("addr", ":8003", "[host]:port to listen")

	log *logrus.Entry
)

type productsDirectory struct {
	ds         *datastore.Client
	ctx        context.Context
	productIds []int64
}

type Product struct {
	K           *datastore.Key `datastore:"__key__"`
	DisplayName string         `datastore:"DisplayName"`
	Description string         `datastore:"Description"`
	Cost        float64        `datastore:"Description"`
	Image       string         `datastore:"Image"`
}

func test() {
	flag.Parse()
	host, err := os.Hostname()
	if err != nil {
		log.Fatal(errors.Wrap(err, "cannot get hostname"))
	}
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.JSONFormatter{FieldMap: logrus.FieldMap{logrus.FieldKeyLevel: "severity"}})
	log = logrus.WithFields(logrus.Fields{
		"service": "userdirectory",
		"host":    host,
		"v":       version.Version(),
	})
	grpclog.SetLogger(log.WithField("facility", "grpc"))

	if env := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"); env == "" {
		log.Fatal("GOOGLE_APPLICATION_CREDENTIALS environment variable is not set")
	}

	if *projectID == "" {
		log.Fatal("google cloud project id is not set")
	}

	ctx := context.Background()

	ds, err := datastore.NewClient(ctx, *projectID)
	if err != nil {
		log.Fatal(err)
	}
	p := &productsDirectory{
		ds:  ds,
		ctx: ctx,
	}
	p.TestProduct()
}

func (p *productsDirectory) TestProduct() {

	log.Info("Trying to insert new product...")
	fmt.Printf("%#v", p)

	// Write product
	k, err := p.ds.Put(p.ctx, datastore.IncompleteKey("Product", nil), &Product{
		DisplayName: "scented candle",
		Description: "delightful candle. a mix of pumpkin spice, bonfire, and vanilla",
		Image:       "someurl.com",
		Cost:        9.99,
	})
	if err != nil {
		log.WithField("error", err).Fatal("failed to save to datastore")
	}
	id := fmt.Sprintf("%d", k.ID)
	log.WithField("id", id).Info("successfully created new product")

	// Read product
	var prod Product
	conv, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		log.Error(err)
	}
	err = p.ds.Get(p.ctx, datastore.IDKey("Product", conv, nil), &prod)
	if err == datastore.ErrNoSuchEntity {
		log.Debug("product not found")
		return
	} else if err != nil {
		log.WithField("error", err).Error("failed to query the datastore")
	}

	log.WithField("my product", prod).Info("successfully read product")

	// Add to products map (this is used to generate list of links for product pages)
	p.productIds = append(p.productIds, conv)

	// TODO - write endpoint to serve that list to generate the homepage (card for every product) + product page for every product

	// ie. when someone clicks on a productl link in the homepage, the resulting webpage should be a dynamically-rendered template
	/*
		- whose URL is the product ID
		- contents are the result of ds.Get(ID) -- image, etc.
	*/

	// TODO - figure out images + Cloud Storage.
}

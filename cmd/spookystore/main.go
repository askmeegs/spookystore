// Copyright 2017 Google Inc.
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
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strings"

	"cloud.google.com/go/datastore"
	"cloud.google.com/go/trace"
	"github.com/m-okeefe/spookystore/cmd/version"
	pb "github.com/m-okeefe/spookystore/internal/proto"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
)

var (
	projectID = flag.String("google-project-id", "", "google cloud project id")
	addr      = flag.String("addr", ":8001", "[host]:port to listen")

	log *logrus.Entry
)

type Server struct {
	ds          *datastore.Client
	productKeys []string // TODO - improve how I do this
}

func main() {
	flag.Parse()
	host, err := os.Hostname()
	if err != nil {
		log.Fatal(errors.Wrap(err, "cannot get hostname"))
	}
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.JSONFormatter{FieldMap: logrus.FieldMap{logrus.FieldKeyLevel: "severity"}})
	log = logrus.WithFields(logrus.Fields{
		"service": "spookystore",
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

	// Initialize server
	ctx := context.Background()
	ds, err := datastore.NewClient(ctx, *projectID)
	if err != nil {
		log.WithField("error", err).Fatal("failed to create datastore client")
	}
	defer ds.Close()

	tc, err := trace.NewClient(ctx, *projectID)
	if err != nil {
		log.Fatal(errors.Wrap(err, "failed to initialize tracing client"))
	}
	ts, err := trace.NewLimitedSampler(1.0, 10)
	if err != nil {
		log.Fatal(errors.Wrap(err, "failed to initialize sampling policy"))
	}
	tc.SetSamplingPolicy(ts)

	lis, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatal(err)
	}
	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(tc.GRPCServerInterceptor()))
	pb.RegisterSpookyStoreServer(grpcServer, &Server{ds, []string{}})

	// add products
	addProducts(ctx, ds)

	log.WithField("addr", *addr).Info("starting to listen on grpc")
	log.Fatal(grpcServer.Serve(lis))
}

// add products from JSON file to Cloud Datastore. return list of productk eys
func addProducts(ctx context.Context, ds *datastore.Client) ([]string, error) {
	pKeys := []string{}

	// add products only if not already present
	file, e := ioutil.ReadFile("./inventory/products.json")
	if e != nil {
		fmt.Println(e)
		return nil, e
	}
	var i map[string]Product
	json.Unmarshal(file, &i)
	for DispName, v := range i {
		q := datastore.NewQuery("Product").Filter("DisplayName =", DispName)
		var result []*Product
		k, err := ds.GetAll(ctx, q, &result)
		if err != nil {
			log.Errorf("Couldn't query: ", err)
		}
		if len(result) > 0 {
			log.Debug("Not adding item=%s, already exists", DispName)
			pKeys = append(pKeys, k[0].String())
			continue
		}
		key := datastore.IncompleteKey("Product", nil)
		p := &pb.Product{
			DisplayName: DispName,
			Cost:        v.Cost,
			PictureURL:  v.Image,
			Description: v.Description,
		}
		newK, err := ds.Put(ctx, key, p)
		if err != nil {
			return nil, err
		}
		spl := strings.Split(newK.String(), ",")
		if len(spl) < 2 {
			return nil, fmt.Errorf("Bad ID: %s", newK.String())
		}
		p.ID = spl[1]
		_, err = ds.Put(ctx, newK, p)
		if err != nil {
			return nil, err
		}
		pKeys = append(pKeys, newK.String())
	}
	return nil, nil
}

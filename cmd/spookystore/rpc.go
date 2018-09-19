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
	"fmt"
	"strconv"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/pkg/errors"

	dw "github.com/m-okeefe/spookystore/internal/datastore_wrapper"
	pb "github.com/m-okeefe/spookystore/internal/proto"

	"cloud.google.com/go/datastore"
	"cloud.google.com/go/trace"

	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

type Server struct {
	ds dw.DatastoreWrapper
}

// AuthorizeGoogle generates an OAuth2 client token for this Google user
func (s *Server) AuthorizeGoogle(ctx context.Context, goog *pb.User) (*pb.User, error) {
	span := trace.FromContext(ctx).NewChild("usersvc/AuthorizeGoogle")
	defer span.Finish()

	gid := goog.GetGoogleID()

	log := log.WithFields(logrus.Fields{
		"op":        "AuthorizeGoogle",
		"google.id": gid})
	log.Debug("received request")

	cs := span.NewChild("datastore/query/user/by_ID")

	q := datastore.NewQuery("User").Filter("GoogleID =", goog.GoogleID).Limit(1)
	var v []User

	if _, err := s.ds.GetAll(ctx, q, &v); err != nil {
		log.WithField("error", err).Error("failed to query the datastore")
		return nil, errors.Wrap(err, "failed to query")
	}
	cs.Finish()

	var id string
	if len(v) == 0 {
		cs = span.NewChild("datastore/put/user")

		u := &User{
			Email:       goog.Email,
			DisplayName: goog.DisplayName,
			GoogleID:    gid,
			Picture:     goog.Picture,
		}

		ik := datastore.IncompleteKey("User", nil)

		k, err := s.ds.Put(ctx, ik, u)
		if err != nil {
			log.WithField("error", err).Error("failed to save to datastore")
			return nil, errors.New("failed to save")
		}
		id = fmt.Sprintf("%d", k.ID)

		u.ID = id
		fmt.Println("SECOND PUT")
		fmt.Printf("REAL USER: %#v", u)
		_, err = s.ds.Put(ctx, datastore.IDKey("User", k.ID, nil), u)
		if err != nil {
			log.WithField("error", err).Error("failed to save with ID to datastore")
			return nil, errors.New("failed to save with ID")
		}

		log.WithField("id", id).Info("created new user")
		cs.Finish()
	} else {
		// return existing user
		id = fmt.Sprintf("%d", v[0].K.ID)
		log.WithField("id", id).Debug("user exists")
	}

	// retrieve user again from backend
	fmt.Printf("\n\nGET USER, ACTUAL ID: %s", id)

	user, err := s.GetUser(ctx, &pb.UserRequest{ID: id})
	if err != nil {
		return nil, errors.Wrap(err, "failed to retrieve user")
	} else if !user.GetFound() {
		return nil, errors.New("cannot find user that is just created")
	}
	return user.GetUser(), nil
}

// GetUser fetches this User from Cloud Datastore
func (s *Server) GetUser(ctx context.Context, req *pb.UserRequest) (*pb.UserResponse, error) {
	span := trace.FromContext(ctx).NewChild("usersvc/GetUser")
	defer span.Finish()

	log := log.WithFields(logrus.Fields{
		"op": "GetUser",
		"id": req.GetID()})
	start := time.Now()
	defer func() {
		log.WithField("elapsed", time.Since(start).String()).Debug("completed request")
	}()
	log.Debug("received request")

	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		return nil, errors.New("cannot parse ID")
	}

	cs := span.NewChild("datastore/query/user/by_id")
	defer cs.Finish()

	var v User
	err = s.ds.Get(ctx, datastore.IDKey("User", id, nil), &v)
	if err == datastore.ErrNoSuchEntity {
		log.Debug("user not found")
		return &pb.UserResponse{Found: false}, nil
	} else if err != nil {
		log.WithField("error", err).Error("failed to query the datastore")
		return nil, errors.Wrap(err, "failed to query")
	}

	return &pb.UserResponse{
		Found: true,
		User: &pb.User{
			ID:           req.ID,
			GoogleID:     v.GoogleID,
			DisplayName:  v.DisplayName,
			Picture:      v.Picture,
			Cart:         v.Cart,
			Transactions: v.Transactions,
		}}, nil
}

// GetNumTransactions fetches the Number of total SpookyStore transactions from Cloud datastore
func (s *Server) GetNumTransactions(ctx context.Context, req *pb.GetNumTransactionsRequest) (*pb.NumTransactionsResponse, error) {
	var t TransactionCounter
	k := datastore.NameKey("TransactionCounter", "AllPurchases", nil)
	err := s.ds.Get(ctx, k, &t)
	if err != nil {
		log.WithField("error", err).Error("failed to get num transactions")
		return nil, errors.Wrap(err, "failed to query")
	}
	return &pb.NumTransactionsResponse{
		NumTransactions: t.NumTransactions,
	}, nil
}

// GetAllProducts returns a list of all Products in the datastore
func (s *Server) GetAllProducts(ctx context.Context, req *pb.GetAllProductsRequest) (*pb.GetAllProductsResponse, error) {
	span := trace.FromContext(ctx).NewChild("spookystoresvc/GetAllProducts")
	defer span.Finish()

	log := log.WithFields(logrus.Fields{
		"op": "GetAllProducts"})
	start := time.Now()
	defer func() {
		log.WithField("elapsed", time.Since(start).String()).Debug("completed request")
	}()
	log.Debug("received request")

	cs := span.NewChild("datastore/query/products")
	defer cs.Finish()

	var result []*pb.Product
	_, err := s.ds.GetAll(ctx, datastore.NewQuery("Product"), &result)
	if err != nil {
		log.WithField("error", err).Error("failed to query the datastore")
		return nil, errors.Wrap(err, "failed to getAll")
	}
	return &pb.GetAllProductsResponse{ProductList: result}, nil
}

// GetProduct fetches a specific product from Datastore
func (s *Server) GetProduct(ctx context.Context, req *pb.GetProductRequest) (*pb.Product, error) {
	var v Product
	parsed, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		return &pb.Product{}, nil
	}

	err = s.ds.Get(ctx, datastore.IDKey("Product", parsed, nil), &v)
	if err == datastore.ErrNoSuchEntity {
		log.Debug("product not found")
		return &pb.Product{}, nil
	} else if err != nil {
		log.WithField("error", err).Error("failed to query the datastore")
		return nil, errors.Wrap(err, "failed to query")
	}
	log.Debug("found product")
	return &pb.Product{
		ID:          fmt.Sprintf("%d", v.K.ID),
		DisplayName: v.DisplayName,
		PictureURL:  v.PictureURL,
		Cost:        v.Cost,
		Description: v.Description,
	}, nil
}

func findProductInCart(items []*pb.CartItem, id string) int {
	for i, item := range items {
		if item.GetID() == id {
			return i
		}
	}
	return -1
}

// AddProductToCart adds one or more Quantity of this Product to a User's Cart
// Note - Cart works like a set, and only stores one CartItem per Product (but supports 1+ quantity of that product)
func (s *Server) AddProductToCart(ctx context.Context, req *pb.AddProductRequest) (*pb.AddProductResponse, error) {
	// get user
	userResp, err := s.GetUser(ctx, &pb.UserRequest{ID: req.UserID})
	if err != nil {
		return &pb.AddProductResponse{Success: false}, err
	}
	user := userResp.GetUser()
	if user.Cart == nil {
		user.Cart = &pb.Cart{
			Items:     []*pb.CartItem{},
			TotalCost: 0.0,
		}
	}

	items := user.Cart.Items
	var addToCost float32
	addToCost = 0.0

	// add to set
	if i := findProductInCart(items, req.ProductID); i > 0 {
		temp := items[i]
		temp.Quantity = temp.Quantity + req.Quantity
		addToCost = temp.Cost * float32(req.Quantity)
	} else {
		prod, err := s.GetProduct(ctx, &pb.GetProductRequest{ID: req.ProductID})
		if err != nil {
			return &pb.AddProductResponse{Success: false}, err
		}
		temp := &pb.CartItem{
			ID:          req.ProductID,
			DisplayName: prod.DisplayName,
			Cost:        prod.Cost,
			Quantity:    req.Quantity,
		}
		addToCost = prod.Cost * float32(req.Quantity)
		items = append(items, temp)
	}

	// update user with cart
	user.Cart.Items = items
	user.Cart.TotalCost += addToCost

	// put user
	parsed, err := strconv.ParseInt(req.UserID, 10, 64)
	if err != nil {
		return &pb.AddProductResponse{Success: false}, err
	}
	u := datastore.IDKey("User", parsed, nil)
	if _, err := s.ds.Put(ctx, u, user); err != nil {
		return &pb.AddProductResponse{Success: false}, err
	}
	return &pb.AddProductResponse{Success: true}, nil
}

// ClearCart zeroes out a User's Cart, and writes the empty Cart back to Datastore
func (s *Server) ClearCart(ctx context.Context, req *pb.UserRequest) (*pb.ClearCartResponse, error) {
	userResp, err := s.GetUser(ctx, &pb.UserRequest{ID: req.ID})
	if err != nil {
		return &pb.ClearCartResponse{Success: false}, err
	}
	user := userResp.GetUser()
	user.Cart = &pb.Cart{}

	// put user
	parsed, err := strconv.ParseInt(user.ID, 10, 64)
	u := datastore.IDKey("User", parsed, nil)
	if _, err := s.ds.Put(ctx, u, user); err != nil {
		return &pb.ClearCartResponse{Success: false}, err
	}
	return &pb.ClearCartResponse{Success: true}, nil
}

// CHeckout gets a user's Cart, clears it, then adds a new Transaction for that user with a Timestamp
func (s *Server) Checkout(ctx context.Context, req *pb.UserRequest) (*pb.CheckoutResponse, error) {
	userResp, err := s.GetUser(ctx, req)
	if err != nil {
		return &pb.CheckoutResponse{Success: false}, err
	}
	user := userResp.User
	t := &pb.Transaction{
		CompletedTime: ptypes.TimestampNow(),
		Items:         user.Cart,
	}
	if user.Transactions == nil {
		user.Transactions = []*pb.Transaction{}
	}
	user.Transactions = append(user.Transactions, t)

	// update user
	parsed, err := strconv.ParseInt(user.ID, 10, 64)
	u := datastore.IDKey("User", parsed, nil)
	if _, err := s.ds.Put(ctx, u, user); err != nil {
		return &pb.CheckoutResponse{Success: false}, err
	}

	// zero out their cart
	_, err = s.ClearCart(ctx, req)
	if err != nil {
		return &pb.CheckoutResponse{Success: false}, err
	}
	return &pb.CheckoutResponse{Success: true}, nil

}

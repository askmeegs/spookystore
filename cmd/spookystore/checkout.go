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
	"strconv"

	"cloud.google.com/go/datastore"

	"github.com/golang/protobuf/ptypes"
	pb "github.com/m-okeefe/spookystore/internal/proto"
	"golang.org/x/net/context"
)

func (s *Server) ClearCart(ctx context.Context, req *pb.UserRequest) (*pb.ClearCartResponse, error) {
	userResp, err := s.GetUser(ctx, &pb.UserRequest{ID: req.ID})
	if err != nil {
		return &pb.ClearCartResponse{Success: false}, err
	}
	user := userResp.GetUser()
	user.Cart = []string{} //empty cart

	// put user
	parsed, err := strconv.ParseInt(user.ID, 10, 64)
	u := datastore.IDKey("User", parsed, nil)
	if _, err := s.ds.Put(ctx, u, user); err != nil {
		return &pb.ClearCartResponse{Success: false}, err
	}
	return &pb.ClearCartResponse{Success: true}, nil
}

// Lists all products in this user's cart w/ the total cost
func (s *Server) GetCart(ctx context.Context, req *pb.UserRequest) (*pb.GetCartResponse, error) {
	userResp, err := s.GetUser(ctx, req)
	if err != nil {
		return &pb.GetCartResponse{}, err
	}

	cart := []*pb.Product{}
	var total float32

	for _, productID := range userResp.User.Cart {
		prod, err := s.GetProduct(ctx, &pb.GetProductRequest{ID: productID})
		if err != nil {
			return &pb.GetCartResponse{}, err
		}
		total += prod.Cost
		cart = append(cart, prod)
	}

	pl := &pb.ProductList{Items: cart}
	return &pb.GetCartResponse{Items: pl, TotalCost: total}, nil

}

// Transforms the Cart items into a Transaction
func (s *Server) Checkout(ctx context.Context, req *pb.UserRequest) (*pb.CheckoutResponse, error) {
	userResp, err := s.GetUser(ctx, req)
	if err != nil {
		return &pb.CheckoutResponse{Success: false}, err
	}
	user := userResp.User
	cart, err := s.GetCart(ctx, req)
	if err != nil {
		return &pb.CheckoutResponse{Success: false}, err
	}
	t := &pb.Transaction{
		CompletedTime: ptypes.TimestampNow(),
		Items:         cart.GetItems(),
		TotalCost:     cart.GetTotalCost(),
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

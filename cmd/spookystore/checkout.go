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
	pb "github.com/m-okeefe/spookystore/internal/proto"
	"golang.org/x/net/context"
)

// Lists all products in this user's cart w/ the total cost
func (s *Server) GetCart(ctx context.Context, req *pb.UserRequest) (*pb.GetCartResponse, error) {
	return &pb.GetCartResponse{}, nil
}

// Transforms the Cart items into a Transaction
func (s *Server) Checkout(ctx context.Context, req *pb.UserRequest) (*pb.CheckoutResponse, error) {
	return &pb.CheckoutResponse{}, nil
}

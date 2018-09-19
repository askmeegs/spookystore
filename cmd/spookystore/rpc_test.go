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
	"testing"

	"github.com/jonboulle/clockwork"

	"cloud.google.com/go/datastore"
	"github.com/golang/mock/gomock"
	dwmock "github.com/m-okeefe/spookystore/internal/datastore_wrapper/mock"
	pb "github.com/m-okeefe/spookystore/internal/proto"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
)

func TestAuthorizeGoogle(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	m := dwmock.NewMockDatastoreWrapper(ctrl)
	ts := &Server{m, clockwork.NewFakeClock()}
	ctx := context.Background()

	tests := []struct {
		u          *pb.User
		shouldPass bool
	}{
		{
			u: &pb.User{
				GoogleID: "12345",
			},
			shouldPass: true,
		},
	}

	for _, test := range tests {
		q := datastore.NewQuery("User").Filter("GoogleID =", test.u.GoogleID).Limit(1)
		var output []User

		m.EXPECT().GetAll(ctx, q, &output).Return([]*datastore.Key{}, nil)

		if test.shouldPass {
			modelUser := &User{
				Email:       test.u.Email,
				DisplayName: test.u.DisplayName,
				GoogleID:    test.u.GetGoogleID(),
				Picture:     test.u.Picture,
			}

			m.EXPECT().Put(ctx, datastore.IncompleteKey("User", nil), modelUser).Return(&datastore.Key{}, nil)
			temp := &User{
				ID:          "0",
				Email:       test.u.Email,
				DisplayName: test.u.DisplayName,
				GoogleID:    test.u.GetGoogleID(),
				Picture:     test.u.Picture,
			}

			m.EXPECT().Put(ctx, datastore.IDKey("User", 0, nil), temp).Return(&datastore.Key{}, nil)
			expectGetUser(m, ctx, "0", "")
		}

		_, err := ts.AuthorizeGoogle(ctx, test.u)
		if test.shouldPass {
			if err != nil {
				t.Error(err)
			}
		} else {
			if err == nil {
				t.Errorf("Expected to fail")
			}
		}
	}
}

func TestGetUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	m := dwmock.NewMockDatastoreWrapper(ctrl)
	ts := &Server{m, clockwork.NewFakeClock()}
	ctx := context.Background()

	tests := []struct {
		u          *pb.User
		shouldPass bool
	}{
		{
			u: &pb.User{
				ID:          "555",
				GoogleID:    "12345",
				Email:       "foo@gmail.com",
				DisplayName: "Foo Bar",
				Picture:     "bar.jpg",
			},
			shouldPass: true,
		},
		{
			u: &pb.User{
				ID: "NonNumericID123",
			},
			shouldPass: false,
		},
	}

	for _, test := range tests {
		if test.shouldPass {
			expectGetUser(m, ctx, test.u.ID, "")
		}
		_, err := ts.GetUser(ctx, &pb.UserRequest{ID: test.u.ID})
		if test.shouldPass {
			if err != nil {
				t.Error(err)
			}
		} else {
			if err == nil {
				t.Errorf("Test passed when expected to fail")
			}
		}
	}
}

func TestGetNumTransactions(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	m := dwmock.NewMockDatastoreWrapper(ctrl)
	ts := &Server{m, clockwork.NewFakeClock()}
	ctx := context.Background()

	var tc TransactionCounter
	k := datastore.NameKey("TransactionCounter", "AllPurchases", nil)
	m.EXPECT().Get(ctx, k, &tc).Return(nil)

	_, err := ts.GetNumTransactions(ctx, &pb.GetNumTransactionsRequest{})
	if err != nil {
		t.Error(err)
	}
}

func TestGetAllProducts(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	m := dwmock.NewMockDatastoreWrapper(ctrl)
	ts := &Server{m, clockwork.NewFakeClock()}
	ctx := context.Background()

	var result []*pb.Product
	m.EXPECT().GetAll(ctx, datastore.NewQuery("Product"), &result)

	_, err := ts.GetAllProducts(ctx, &pb.GetAllProductsRequest{})
	if err != nil {
		t.Error(err)
	}
}

func TestGetProduct(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	m := dwmock.NewMockDatastoreWrapper(ctrl)
	ts := &Server{m, clockwork.NewFakeClock()}
	ctx := context.Background()

	tests := []struct {
		p *pb.Product
	}{
		{
			p: &pb.Product{
				ID:          "601",
				DisplayName: "My Product",
				PictureURL:  "great.jpg",
				Cost:        29.50,
				Description: "An awesome product",
			},
		},
	}

	for _, test := range tests {
		parsed, _ := strconv.ParseInt(test.p.ID, 10, 64)
		var v Product
		m.EXPECT().Get(ctx, datastore.IDKey("Product", parsed, nil), &v)
		_, err := ts.GetProduct(ctx, &pb.GetProductRequest{ID: test.p.ID})
		if err != nil {
			t.Error(err)
		}
	}
}

func TestAddProductToCart(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	m := dwmock.NewMockDatastoreWrapper(ctrl)
	ts := &Server{m, clockwork.NewFakeClock()}
	ctx := context.Background()

	user := pb.User{
		ID:          "555",
		GoogleID:    "12345",
		Email:       "foo@gmail.com",
		DisplayName: "Foo Bar",
		Picture:     "bar.jpg",
	}

	expectGetUser(m, ctx, user.ID, "")
	var v Product
	m.EXPECT().Get(ctx, datastore.IDKey("Product", 123, nil), &v)

	parsed, _ := strconv.ParseInt(user.ID, 10, 64)
	u := datastore.IDKey("User", parsed, nil)

	finalUser := &pb.User{
		ID:   "555",
		Cart: &pb.Cart{Items: []*pb.CartItem{&pb.CartItem{ID: "123"}}},
	}
	m.EXPECT().Put(ctx, u, finalUser)

	_, err := ts.AddProductToCart(ctx, &pb.AddProductRequest{UserID: user.ID, ProductID: "123"})
	if err != nil {
		t.Error(err)
	}
}

func TestClearCart(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	m := dwmock.NewMockDatastoreWrapper(ctrl)
	ts := &Server{m, clockwork.NewFakeClock()}
	ctx := context.Background()

	user := pb.User{
		ID:          "555",
		GoogleID:    "12345",
		Email:       "foo@gmail.com",
		DisplayName: "Foo Bar",
		Picture:     "bar.jpg",
	}

	expectGetUser(m, ctx, user.ID, "")

	parsed, _ := strconv.ParseInt(user.ID, 10, 64)
	u := datastore.IDKey("User", parsed, nil)

	finalUser := &pb.User{
		ID:   "555",
		Cart: &pb.Cart{},
	}
	m.EXPECT().Put(ctx, u, finalUser)

	_, err := ts.ClearCart(ctx, &pb.UserRequest{ID: user.ID})
	if err != nil {
		t.Error(err)
	}
}

func TestCheckout(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	m := dwmock.NewMockDatastoreWrapper(ctrl)
	ts := &Server{m, clockwork.NewFakeClock()}
	ctx := context.Background()

	user := pb.User{
		ID:          "555",
		GoogleID:    "12345",
		Email:       "foo@gmail.com",
		DisplayName: "Foo Bar",
		Picture:     "bar.jpg",
	}

	expectGetUser(m, ctx, user.ID, "")

	parsed, _ := strconv.ParseInt(user.ID, 10, 64)
	u := datastore.IDKey("User", parsed, nil)

	putUser := &pb.User{
		ID:           "555",
		Transactions: []*pb.Transaction{&pb.Transaction{CompletedTime: ClockworkNow(ts)}},
	}
	m.EXPECT().Put(ctx, u, putUser)
	expectGetUser(m, ctx, user.ID, "")

	finalUser := &pb.User{
		ID:   "555",
		Cart: &pb.Cart{},
	}
	m.EXPECT().Put(ctx, u, finalUser)

	_, err := ts.Checkout(ctx, &pb.UserRequest{ID: user.ID})
	if err != nil {
		t.Error(err)
	}
}

func expectGetUser(m *dwmock.MockDatastoreWrapper, ctx context.Context, id string, errMsg string) {
	parsed, _ := strconv.ParseInt(id, 10, 64)

	if errMsg == "" {
		m.EXPECT().Get(ctx, datastore.IDKey("User", parsed, nil), &User{}).Return(nil)
	} else {
		m.EXPECT().Get(ctx, datastore.IDKey("User", parsed, nil), &User{}).Return(errors.New(errMsg))
	}
}

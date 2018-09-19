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
	ts := &Server{m}
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

func expectGetUser(m *dwmock.MockDatastoreWrapper, ctx context.Context, id string, errMsg string) {
	parsed, _ := strconv.ParseInt(id, 10, 64)

	if errMsg == "" {
		m.EXPECT().Get(ctx, datastore.IDKey("User", parsed, nil), &User{}).Return(nil)
	} else {
		m.EXPECT().Get(ctx, datastore.IDKey("User", parsed, nil), &User{}).Return(errors.New(errMsg))
	}
}

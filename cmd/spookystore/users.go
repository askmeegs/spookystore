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

	"github.com/pkg/errors"

	pb "github.com/m-okeefe/spookystore/internal/proto"

	"cloud.google.com/go/datastore"
	"cloud.google.com/go/trace"

	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

type user struct {
	K *datastore.Key `datastore:"__key__"`

	GoogleID             string            `datastore:"__key__"`
	ID                   string            `datastore:"ID"`
	DisplayName          string            `datastore:"DisplayName"`
	Picture              string            `datastore:"Picture"`
	Cart                 []string          `datastore:"Cart"`
	Transactions         []*pb.Transaction `datastore:"Transactions"`
	Email                string            `datastore:"Email"`
	XXX_NoUnkeyedLiteral struct{}          `datastore:"XXX_NoUnkeyedLiteral"`
	XXX_unrecognized     []byte            `datastore:"XXX_unrecognized"`
	XXX_sizecache        int32             `datastore:"XXX_sizecache"`
}

func (s *Server) AuthorizeGoogle(ctx context.Context, goog *pb.User) (*pb.User, error) {
	span := trace.FromContext(ctx).NewChild("usersvc/AuthorizeGoogle")
	defer span.Finish()

	log := log.WithFields(logrus.Fields{
		"op":        "AuthorizeGoogle",
		"google.id": goog.GetGoogleID()})
	log.Debug("received request")

	cs := span.NewChild("datastore/query/user/by_ID")
	q := datastore.NewQuery("User").Filter("GoogleID =", goog.GoogleID).Limit(1)
	var v []user
	if _, err := s.ds.GetAll(ctx, q, &v); err != nil {
		log.WithField("error", err).Error("failed to query the datastore")
		return nil, errors.Wrap(err, "failed to query")
	}
	cs.Finish()

	var id string
	if len(v) == 0 {
		cs = span.NewChild("datastore/put/user")

		u := &user{
			Email:       goog.Email,
			DisplayName: goog.DisplayName,
			GoogleID:    goog.GoogleID,
			Picture:     goog.Picture,
		}

		// create new user
		k, err := s.ds.Put(ctx, datastore.IncompleteKey("User", nil), u)
		if err != nil {
			log.WithField("error", err).Error("failed to save to datastore")
			return nil, errors.New("failed to save")
		}
		id = fmt.Sprintf("%d", k.ID)
		u.ID = id
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
	user, err := s.GetUser(ctx, &pb.UserRequest{ID: id})
	if err != nil {
		return nil, errors.Wrap(err, "failed to retrieve user")
	} else if !user.GetFound() {
		return nil, errors.New("cannot find user that is just created")
	}
	fmt.Println("AUTHORIZED GOOGLE. GOOGLEID is %s and REGULAR ID is %s\n", user.GetUser().GetGoogleID(), id)
	return user.GetUser(), nil
}

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

	var v user
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
			ID:           fmt.Sprintf("%d", v.K.ID),
			GoogleID:     v.GoogleID,
			DisplayName:  v.DisplayName,
			Picture:      v.Picture,
			Cart:         v.Cart,
			Transactions: v.Transactions,
		}}, nil
}

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

// Based on https://github.com/ahmetb/coffeelog/tree/master/cmd/web

package main

import (
	"context"
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"cloud.google.com/go/trace"
	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"
	"github.com/m-okeefe/spookystore/cmd/version"
	pb "github.com/m-okeefe/spookystore/internal/proto"
	"github.com/pkg/errors"
	logrus "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	plus "google.golang.org/api/plus/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
)

type server struct {
	cfg       *oauth2.Config
	spookySvc pb.SpookyStoreClient
	tc        *trace.Client
}

var (
	projectID          = flag.String("google-project-id", "", "google cloud project id")
	addr               = flag.String("addr", ":8000", "[host]:port to listen")
	oauthConfig        = flag.String("google-oauth2-config", "", "path to oauth2 config json")
	spookyStoreBackend = flag.String("spooky-store-addr", "", "address of spookystore backend")

	hashKey  = []byte("very-secret")      // TODO extract to env
	blockKey = []byte("a-lot-secret-key") // TODO extract to env
	sc       = securecookie.New(hashKey, blockKey)
)

var log *logrus.Entry

func main() {
	flag.Parse()
	host, err := os.Hostname()
	if err != nil {
		log.Fatal(errors.Wrap(err, "cannot get hostname"))
	}
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.JSONFormatter{FieldMap: logrus.FieldMap{logrus.FieldKeyLevel: "severity"}})
	log = logrus.WithFields(logrus.Fields{
		"service": "web",
		"host":    host,
		"v":       version.Version(),
	})
	grpclog.SetLogger(log.WithField("facility", "grpc"))
	sc.SetSerializer(securecookie.JSONEncoder{})

	if env := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"); env == "" {
		log.Fatal("GOOGLE_APPLICATION_CREDENTIALS environment variable is not set")
	}
	if *projectID == "" {
		log.Fatal("google cloud project id flag not specified")
	}
	if *spookyStoreBackend == "" {
		log.Fatal("spookystorebackend address flag not specified")
	}
	if *oauthConfig == "" {
		log.Fatal("google oauth2 config flag not specified")
	}

	b, err := ioutil.ReadFile(*oauthConfig)
	if err != nil {
		log.Fatal(errors.Wrap(err, "failed to parse config file"))
	}
	authConf, err := google.ConfigFromJSON(b)
	if err != nil {
		log.Fatal(errors.Wrap(err, "failed to parse config file"))
	}

	tc, err := trace.NewClient(context.Background(), *projectID)
	if err != nil {
		log.Fatal(errors.Wrap(err, "failed to initialize trace client"))
	}
	spookySvcConn, err := grpc.Dial(*spookyStoreBackend,
		grpc.WithInsecure(),
		grpc.WithUnaryInterceptor(tc.GRPCClientInterceptor()))
	if err != nil {
		log.Fatal(errors.Wrap(err, "cannot connect to backend spookystore service"))
	}
	defer func() {
		log.Info("closing connection to spookystore backend")
		spookySvcConn.Close()
	}()
	sp, err := trace.NewLimitedSampler(1.0, 5)
	if err != nil {
		log.Fatal(errors.Wrap(err, "failed to create sampling policy"))
	}
	tc.SetSamplingPolicy(sp)

	s := &server{
		tc:        tc,
		cfg:       authConf,
		spookySvc: pb.NewSpookyStoreClient(spookySvcConn),
	}

	// set up server
	r := mux.NewRouter()
	r.PathPrefix("/static/").HandlerFunc(http.StripPrefix("/static/", http.FileServer(http.Dir("static"))).ServeHTTP)
	r.Handle("/", s.traceHandler(logHandler(s.home))).Methods(http.MethodGet)
	r.Handle("/login", s.traceHandler(logHandler(s.login))).Methods(http.MethodGet)
	r.Handle("/logout", s.traceHandler(logHandler(s.logout))).Methods(http.MethodGet)
	r.Handle("/oauth2callback", s.traceHandler(logHandler(s.oauth2Callback))).Methods(http.MethodGet)
	r.Handle("/u/{id:[0-9]+}", s.traceHandler(logHandler(s.userProfile))).Methods(http.MethodGet)
	r.Handle("/cart/u/{id:[0-9]+}", s.traceHandler(logHandler(s.cart)))
	r.Handle("/checkout/u/{id:[0-9]+}", s.traceHandler(logHandler(s.checkout)))
	r.Handle("/addproduct/{id:[0-9]+}/{pid:[0-9]+}", s.traceHandler(logHandler(s.addProduct)))
	srv := http.Server{
		Addr:    *addr, // TODO make configurable
		Handler: r}
	log.WithFields(logrus.Fields{"addr": *addr,
		"spookyStore": *spookyStoreBackend}).Info("starting to listen on http")
	log.Fatal(errors.Wrap(srv.ListenAndServe(), "failed to listen/serve"))
}

type httpErrorWriter func(http.ResponseWriter, error)

func (s *server) getUser(ctx context.Context, id string) (*pb.UserResponse, error) {
	span := trace.FromContext(ctx).NewChild("get_user")
	defer span.Finish()
	span.SetLabel("user/id", id)

	cs := span.NewChild("rpc.Sent/GetUser")
	defer cs.Finish()
	userResp, err := s.spookySvc.GetUser(ctx, &pb.UserRequest{ID: id})
	return userResp, err
}

func (s *server) authUser(ctx context.Context, r *http.Request) (user *pb.User, errFunc httpErrorWriter, err error) {
	span := trace.FromContext(ctx).NewChild("authorize_user")
	defer span.Finish()

	c, err := r.Cookie("user")
	if err == http.ErrNoCookie {
		return nil, nil, nil
	}
	log.Debug("auth cookie found")
	var userID string
	if err := sc.Decode("user", c.Value, &userID); err != nil {
		return nil, badRequest, errors.Wrap(err, "failed to decode cookie")
	}

	userResp, err := s.getUser(ctx, userID)
	if err != nil {
		return nil, serverError, errors.Wrap(err, "failed to look up the user")
	} else if !userResp.GetFound() {
		return nil, badRequest, errors.New("unrecognized user")
	}
	return userResp.GetUser(), nil, nil
}

func (s *server) home(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user, errF, err := s.authUser(ctx, r)
	if err != nil {
		errF(w, err)
		return
	}

	if user == nil {
		serverError(w, errors.Wrap(err, "user is nil"))
	}

	resp, err := s.spookySvc.GetAllProducts(ctx, &pb.GetAllProductsRequest{})
	if err != nil {
		fmt.Println(err)
		serverError(w, errors.Wrap(err, "failed to get all products"))
	}

	log.WithField("logged_in", user != nil).Debug("serving home page")
	tmpl := template.Must(template.ParseFiles(
		filepath.Join("static", "template", "layout.html"),
		filepath.Join("static", "template", "home.html")))

	if err := tmpl.Execute(w, map[string]interface{}{
		"me":       user,
		"userID":   user.GetID(),
		"products": resp.ProductList.GetItems()}); err != nil {
		log.Fatal(err)
	}
}

func (s *server) login(w http.ResponseWriter, r *http.Request) {
	s.cfg.RedirectURL = "http://" + r.Host + "/oauth2callback" // TODO this is hacky
	s.cfg.Scopes = []string{"profile", "email"}
	url := s.cfg.AuthCodeURL("todo_rand_state",
		oauth2.SetAuthURLParam("access_type", "offline"))
	log.Debug("redirecting user to oauth2 consent page")
	w.Header().Set("Location", url)
	w.WriteHeader(http.StatusFound)
}

func (s *server) logout(w http.ResponseWriter, r *http.Request) {
	log.Debug("logout requested")
	for _, c := range r.Cookies() {
		c.Expires = time.Unix(1, 0)
		http.SetCookie(w, c)
		log.WithField("key", c.Name).Debug("cleared cookie")
	}
	w.Header().Set("Location", "/")
	w.WriteHeader(http.StatusFound)
}

func (s *server) oauth2Callback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	span := trace.FromContext(ctx)
	if state := r.URL.Query().Get("state"); state != "todo_rand_state" {
		badRequest(w, errors.New("wrong oauth2 state"))
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		badRequest(w, errors.New("missing oauth2 grant code"))
		return
	}

	cs := span.NewChild("oauth2/exchange_token")
	tok, err := s.cfg.Exchange(ctx, code)
	if err != nil {
		serverError(w, errors.Wrap(err, "oauth2 token exchange failed"))
		return
	}
	cs.Finish()

	cs = span.NewChild("gplus/get/me")
	svc, err := plus.New(oauth2.NewClient(ctx, s.cfg.TokenSource(ctx, tok)))
	if err != nil {
		serverError(w, errors.Wrap(err, "failed to construct g+ client"))
		return
	}
	me, err := plus.NewPeopleService(svc).Get("me").Do()
	if err != nil {
		serverError(w, errors.Wrap(err, "failed to query user g+ profile"))
		return
	}
	cs.Finish()
	log.WithField("google.id", me.Id).Debug("retrieved google user")

	cs = span.NewChild("authorize_google")
	user, err := s.spookySvc.AuthorizeGoogle(ctx,
		&pb.User{
			ID:          me.Id,
			Email:       me.Emails[0].Value,
			DisplayName: me.DisplayName,
			Picture:     me.Image.Url,
		})
	if err != nil {
		serverError(w, errors.Wrap(err, "failed to log in the user"))
		return
	}
	log.WithField("id", user.ID).Info("authenticated user with google")
	cs.Finish()

	// save the user id to cookies
	// TODO implement as sessions
	co, err := sc.Encode("user", user.ID)
	if err != nil {
		serverError(w, errors.Wrap(err, "failed to encode the token"))
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:  "user",
		Path:  "/",
		Value: co,
	})

	log.WithField("user.id", me.Id).Info("authenticated user")
	w.Header().Set("Location", "/")
	w.WriteHeader(http.StatusFound)
}

func (s *server) checkout(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id := mux.Vars(r)["id"]

	_, ef, err := s.authUser(ctx, r)
	if err != nil {
		ef(w, err)
		return
	}

	userResp, err := s.getUser(ctx, id)
	if err != nil {
		serverError(w, errors.Wrap(err, "checkout: failed to look up the user"))
		return
	} else if !userResp.GetFound() {
		errorCode(w, http.StatusNotFound, "not found", errors.New("user not found"))
		return
	}

	_, err = s.spookySvc.Checkout(ctx, &pb.UserRequest{ID: id})
	if err != nil {
		serverError(w, errors.Wrap(err, "checkout failed"))
		return
	}
	// take user to their transactions page
	w.Header().Set("Location", fmt.Sprintf("/u/%s", id))
	w.WriteHeader(http.StatusFound)
}

func (s *server) cart(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id := mux.Vars(r)["id"]

	me, ef, err := s.authUser(ctx, r)
	if err != nil {
		ef(w, err)
		return
	}

	userResp, err := s.getUser(ctx, id)
	if err != nil {
		serverError(w, errors.Wrap(err, "failed to look up the user"))
		return
	} else if !userResp.GetFound() {
		errorCode(w, http.StatusNotFound, "not found", errors.New("user not found"))
		return
	}

	cart, err := s.spookySvc.GetCart(ctx, &pb.UserRequest{ID: id})
	if err != nil {
		serverError(w, errors.Wrap(err, "failed to get cart"))
		return
	} else if !userResp.GetFound() {
		errorCode(w, http.StatusNotFound, "not found", errors.New("cart not found"))
		return
	}

	fmt.Println("\n\n\nGET CART, CART IS %#v", cart)

	tmpl := template.Must(template.ParseFiles(
		filepath.Join("static", "template", "layout.html"),
		filepath.Join("static", "template", "cart.html")))
	if err := tmpl.Execute(w, map[string]interface{}{
		"me":        me,
		"cart":      cart,
		"user":      userResp.GetUser(),
		"CartItems": cart.Items.GetItems(),
	}); err != nil {
		log.Fatal(err)
	}
	w.Header().Set("Location", "/") //take me home
	w.WriteHeader(http.StatusOK)
}

func (s *server) addProduct(w http.ResponseWriter, r *http.Request) {

	ctx := r.Context()
	span := trace.FromContext(ctx)

	userID := mux.Vars(r)["id"]
	productID := mux.Vars(r)["pid"]
	span.SetLabel("user/id", userID)

	_, ef, err := s.authUser(ctx, r)
	if err != nil {
		ef(w, err)
		return
	}

	_, err = s.spookySvc.AddProductToCart(ctx, &pb.AddProductRequest{UserID: userID, ProductID: productID})
	if err != nil {
		serverError(w, errors.Wrap(err, "failed to add product to cart"))
	}
	w.Header().Set("Location", "/")
	w.WriteHeader(http.StatusOK)
}

func (s *server) userProfile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	span := trace.FromContext(ctx)

	userID := mux.Vars(r)["id"]
	span.SetLabel("user/id", userID)

	me, ef, err := s.authUser(ctx, r)
	if err != nil {
		ef(w, err)
		return
	}

	userResp, err := s.getUser(ctx, userID)
	if err != nil {
		serverError(w, errors.Wrap(err, "failed to look up the user"))
		return
	} else if !userResp.GetFound() {
		errorCode(w, http.StatusNotFound, "not found", errors.New("user not found"))
		return
	}

	u := userResp.GetUser()

	tmpl := template.Must(template.ParseFiles(
		filepath.Join("static", "template", "layout.html"),
		filepath.Join("static", "template", "profile.html")))
	if err := tmpl.Execute(w, map[string]interface{}{
		"me":           me,
		"user":         u,
		"Transactions": u.GetTransactions(),
	}); err != nil {
		log.Fatal(err)
	}
	w.Header().Set("Location", "/") //take me home
	w.WriteHeader(http.StatusOK)
}

func errorCode(w http.ResponseWriter, code int, msg string, err error) {
	log.WithField("http.status", code).WithField("error", err).Warn(msg)
	w.WriteHeader(code)
	fmt.Fprint(w, errors.Wrap(err, msg))
}

func unauthorized(w http.ResponseWriter, err error) {
	errorCode(w, http.StatusUnauthorized, "unauthorized", err)
}

func badRequest(w http.ResponseWriter, err error) {
	errorCode(w, http.StatusBadRequest, "bad request", err)
}

func serverError(w http.ResponseWriter, err error) {
	errorCode(w, http.StatusInternalServerError, "server error", err)
}

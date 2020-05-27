// Copyright (c) 2019 The Jaeger Authors.
// Copyright (c) 2017 Uber Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package frontend

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"errors"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"go.uber.org/zap"

	"terazo_parcel_service/pkg/httperr"
	"terazo_parcel_service/pkg/log"
	"terazo_parcel_service/pkg/tracing"

	"github.com/DataDog/datadog-go/statsd"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"strconv"

	geoHash "github.com/TomiHiltunen/geohash-golang"
)

// Server implements jaeger-demo-frontend service
type Server struct {
	hostPort string
	tracer   opentracing.Tracer
	logger   log.Factory
	bestETA  *bestETA
	assetFS  http.FileSystem
	basepath string
}

// ConfigOptions used to make sure service clients
// can find correct server ports
type ConfigOptions struct {
	FrontendHostPort string
	DriverHostPort   string
	CustomerHostPort string
	RouteHostPort    string
	Basepath         string
}
type StatsdClient struct {
	logger log.Factory
}

// define requestor side metric to send to prometheus
var (
	requestProcessedCounterVec = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "processed_by_request",
			Help: "Total number of requests processed per user",
		},
		// this is where it takes in vars, can be anything we wanna track that is a string
		// in this case im looking at customer is and call type
		[]string{"customerName", "calltype", "location", "status"},
	)
)

//Then , its handler that wraps handlers
func (s StatsdClient) Then(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		statsd, err := statsd.New("127.0.0.1:8125")
		if err != nil {
			s.logger.Bg().Info("statsd failed fam")
		}
		method := r.Method
		path := r.URL.Path[1:]

		statsd.Incr(fmt.Sprintf("%s.%s", method, path), []string{"environment:dev"}, 1)
		next.ServeHTTP(w, r)
	})
}

// NewServer creates a new frontend.Server
func NewServer(options ConfigOptions, tracer opentracing.Tracer, logger log.Factory) *Server {
	assetFS := FS(false)
	return &Server{
		hostPort: options.FrontendHostPort,
		tracer:   tracer,
		logger:   logger,
		bestETA:  newBestETA(tracer, logger, options),
		assetFS:  assetFS,
		basepath: options.Basepath,
	}
}

// Run starts the frontend server
func (s *Server) Run() error {
	mux := s.createServeMux()
	s.logger.Bg().Info("Starting", zap.String("address", "http://"+path.Join(s.hostPort, s.basepath)))
	prometheus.MustRegister(requestProcessedCounterVec)
	return http.ListenAndServe(s.hostPort, mux)
}

func (s *Server) createServeMux() http.Handler {
	mux := tracing.NewServeMux(s.tracer)
	p := path.Join("/", s.basepath)
	mux.Handle(p, http.StripPrefix(p, http.FileServer(s.assetFS)))
	finalHandler := http.HandlerFunc(s.dispatch)
	// add the /metrics handler for  prometheus
	mux.Handle("/metrics", promhttp.Handler())

	mux.Handle(path.Join(p, "/dispatch"), StatsdClient.Then(StatsdClient{}, finalHandler))

	return mux
}

func (s *Server) dispatch(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	s.logger.For(ctx).Info("HTTP request received", zap.String("method", r.Method), zap.Stringer("url", r.URL))
	if err := r.ParseForm(); httperr.HandleError(w, err, http.StatusBadRequest) {
		s.logger.For(ctx).Error("bad request", zap.Error(err))
		return
	}

	// get the parameters from the request
	customerID := r.Form.Get("customer")
	fmt.Println(customerID)
	method := r.Method
	location := r.Header.Get("location")
	latitude := r.Header.Get("latitude")
	longitude := r.Header.Get("longitude")
	status := r.Header.Get("status")

	lat := 0.0
	long := 0.0

	// some logic to get the geohash from a lat long, we dont use this yet bc of grafana not populating the geohash...yet
	fmt.Println("**************************")
	fmt.Println(latitude, longitude)
	if s, err := strconv.ParseFloat(latitude, 64); err == nil {
		fmt.Println(s) // 3.1415927410125732
		lat = s
	}
	if s, err := strconv.ParseFloat(longitude, 64); err == nil {
		fmt.Println(s) // 3.1415927410125732
		long = s
	}
	geohash := geoHash.Encode(lat, long)

	fmt.Println("geohash: ")
	fmt.Println(geohash)

	if customerID == "" {
		http.Error(w, "Missing required 'customer' parameter", http.StatusBadRequest)
		return
	}

	// increment requestor processed_by_request with given parameters
	requestProcessedCounterVec.WithLabelValues(customerID, method, location, status).Inc()

	// if status is error on requestor side, throw BadRequest error and return
	if status == "error" {
		if span := opentracing.SpanFromContext(ctx); span != nil {
			ext.Error.Set(span, true)
		}
		err := errors.New("validation error occured from requestor")
		http.Error(w, "Bad request from requestor", http.StatusBadRequest)
		s.logger.For(ctx).Error("validation error occured from requestor", zap.String("customer_id", customerID), zap.Error(err))
		return
	}

	// TODO distinguish between user errors (such as invalid customer ID) and server failures
	response, err := s.bestETA.Get(ctx, customerID)
	if httperr.HandleError(w, err, http.StatusInternalServerError) {
		s.logger.For(ctx).Error("request failed", zap.Error(err))
		return
	}

	data, err := json.Marshal(response)
	if httperr.HandleError(w, err, http.StatusInternalServerError) {
		s.logger.For(ctx).Error("cannot marshal response", zap.Error(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

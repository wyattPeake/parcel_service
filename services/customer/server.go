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

package customer

import (
	"encoding/json"
	"net/http"

	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-lib/metrics"
	"go.uber.org/zap"

	"fmt"

	"github.com/DataDog/datadog-go/statsd"
	"terazo_parcel_service/pkg/httperr"
	"terazo_parcel_service/pkg/log"
	"terazo_parcel_service/pkg/tracing"

	"time"

	"io/ioutil"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Server implements Customer service
type Server struct {
	hostPort string
	tracer   opentracing.Tracer
	logger   log.Factory
	database *database
}
type StatsdClient struct {
	logger log.Factory
}

var (
	opsProcessed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "Test_Customer_service_uptime",
		Help: "How long the service has been up",
	})
)

// define customer side metric to send to prometheus
var (
	customerProcessedCounterVec = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "processed_by_customer",
			Help: "Total number of requests processed per customer",
		},
		// this is where it takes in vars, can be anything we wanna track that is a string
		// in this case im looking at customer is and call type
		[]string{"customerName", "calltype", "state", "region", "status"},
	)
)

func recordMetrics() {
	go func() {
		for {
			opsProcessed.Inc()
			time.Sleep(1 * time.Second)
		}
	}()
}

// Then ...
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

// NewServer creates a new customer.Server
func NewServer(hostPort string, tracer opentracing.Tracer, metricsFactory metrics.Factory, logger log.Factory) *Server {
	// call goroutine that records up time, for testing
	recordMetrics()
	return &Server{
		hostPort: hostPort,
		tracer:   tracer,
		logger:   logger,
		database: newDatabase(
			tracing.Init("mysql", metricsFactory, logger),
			logger.With(zap.String("component", "mysql")),
		),
	}

}

// Run starts the Customer server
func (s *Server) Run() error {
	mux := s.createServeMux()
	s.logger.Bg().Info("Starting", zap.String("address", "http://"+s.hostPort))
	recordMetrics()
	prometheus.MustRegister(customerProcessedCounterVec)
	return http.ListenAndServe(s.hostPort, mux)
}

func (s *Server) createServeMux() http.Handler {
	mux := tracing.NewServeMux(s.tracer)
	finalHandler := http.HandlerFunc(s.customer)
	mux.Handle("/metrics", promhttp.Handler())
	mux.Handle("/customer", StatsdClient.Then(StatsdClient{}, finalHandler))
	return mux

}

// find an available truck in that region
func GetTruck(r *http.Request, s *Server, region string) bool {
	ctx := r.Context()
	response, err := http.Get("http://thirdparty:8085/shipping/findtruck/" + region)
	if err != nil {
		s.logger.For(ctx).Error("GetTruck  error", zap.Error(err))
	}
	data, _ := ioutil.ReadAll(response.Body)

	status := response.Status
	fmt.Println(string(data))
	fmt.Println("$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$")
	fmt.Println(status)

	if status != "200 OK" {
		s.logger.For(ctx).Error("getTruck failed, no trucks avaiable in that region", zap.Error(err))
		return true
	} else {
		s.logger.For(ctx).Info("truck found in region")

	}

	return false
}

// request id/ client id amount of request
func (s *Server) customer(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	s.logger.For(ctx).Info("HTTP request received", zap.String("method", r.Method), zap.Stringer("url", r.URL))
	if err := r.ParseForm(); httperr.HandleError(w, err, http.StatusBadRequest) {
		s.logger.For(ctx).Error("bad request", zap.Error(err))
		return
	}

	customerID := r.Form.Get("customer")
	if customerID == "" {
		http.Error(w, "Missing required 'customer' parameter", http.StatusBadRequest)
		return
	}

	response, err := s.database.Get(ctx, customerID)
	if httperr.HandleError(w, err, http.StatusInternalServerError) {
		s.logger.For(ctx).Error("request failed", zap.Error(err))
		return
	}

	//status := r.Header.Get("status")

	data, err := json.Marshal(response)
	fmt.Println("**********************************")
	// im sorry
	location := "CA"
	region := "North West"

	if customerID == "123" {
		location = "WA"
		region = "north-west"
	} else if customerID == "567" {
		location = "OR"
		region = "north-west"
	} else if customerID == "392" {
		location = "CA"
		region = "south-west"
	} else {
		location = "AL"
		region = "south-east"
	}

	// if there is an error count it
	if GetTruck(r, s, region) {
		customerProcessedCounterVec.WithLabelValues(customerID, r.Method, location, region, "error").Inc()
	} else {
		customerProcessedCounterVec.WithLabelValues(customerID, r.Method, location, region, "processed").Inc()
	}

	if httperr.HandleError(w, err, http.StatusInternalServerError) {
		s.logger.For(ctx).Error("cannot marshal response", zap.Error(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

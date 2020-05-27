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
	"context"
	"errors"

	"github.com/opentracing/opentracing-go"
	tags "github.com/opentracing/opentracing-go/ext"
	"go.uber.org/zap"

	"terazo_parcel_service/pkg/delay"
	"terazo_parcel_service/pkg/log"
	"terazo_parcel_service/pkg/tracing"
	"terazo_parcel_service/services/config"
)

// database simulates Customer repository implemented on top of an SQL database
type database struct {
	tracer    opentracing.Tracer
	logger    log.Factory
	customers map[string]*Customer
	lock      *tracing.Mutex
}

func newDatabase(tracer opentracing.Tracer, logger log.Factory) *database {
	return &database{
		tracer: tracer,
		logger: logger,
		lock: &tracing.Mutex{
			SessionBaggageKey: "request",
		},
		customers: map[string]*Customer{
			"123": {
				ID:       "123",
				Name:     "Bailey Motorsport",
				Location: "WA",
			},
			"567": {
				ID:       "567",
				Name:     "Hu's Irish Whiskey Distillery",
				Location: "OR",
			},
			"392": {
				ID:       "392",
				Name:     "Worrels Esports Supplies",
				Location: "CA",
			},
			"731": {
				ID:       "731",
				Name:     "Wyatt’s Taxidermy, LLC",
				Location: "AL",
			},
		},
	}
}

func (d *database) Get(ctx context.Context, customerID string) (*Customer, error) {
	d.logger.For(ctx).Info("Loading customer", zap.String("customer_id", customerID))

	// simulate opentracing instrumentation of an SQL query
	if span := opentracing.SpanFromContext(ctx); span != nil {
		span := d.tracer.StartSpan("SQL SELECT", opentracing.ChildOf(span.Context()))
		tags.SpanKindRPCClient.Set(span)
		tags.PeerService.Set(span, "mysql")
		// #nosec
		span.SetTag("sql.query", "SELECT * FROM customer WHERE customer_id="+customerID)
		d.logger.Bg().Info("SELECT * FROM customer WHERE customer_id=", zap.String("customer id", customerID))
		defer span.Finish()
		ctx = opentracing.ContextWithSpan(ctx, span)
	}

	if !config.MySQLMutexDisabled {
		// simulate misconfigured connection pool that only gives one connection at a time
		d.lock.Lock(ctx)
		defer d.lock.Unlock()
	}

	// simulate RPC delay
	delay.Sleep(config.MySQLGetDelay, config.MySQLGetDelayStdDev)

	if customer, ok := d.customers[customerID]; ok {
		return customer, nil
	}
	return nil, errors.New("invalid customer ID")
}
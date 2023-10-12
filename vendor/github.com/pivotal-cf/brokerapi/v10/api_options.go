// Copyright (C) 2015-Present Pivotal Software, Inc. All rights reserved.

// This program and the accompanying materials are made available under
// the terms of the under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

// http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package brokerapi

import (
	"net/http"

	"code.cloudfoundry.org/lager/v3"
	"github.com/go-chi/chi/v5"
	"github.com/pivotal-cf/brokerapi/v10/auth"
	"github.com/pivotal-cf/brokerapi/v10/domain"
	"github.com/pivotal-cf/brokerapi/v10/middlewares"
)

type middlewareFunc func(http.Handler) http.Handler

func NewWithOptions(serviceBroker domain.ServiceBroker, logger lager.Logger, opts ...Option) http.Handler {
	cfg := newDefaultConfig(logger)
	WithOptions(append(opts, withDefaultMiddleware())...)(cfg)
	attachRoutes(cfg.router, serviceBroker, logger)

	return cfg.router
}

type Option func(*config)

func WithRouter(router chi.Router) Option {
	return func(c *config) {
		c.router = router
		c.customRouter = true
	}
}

func WithBrokerCredentials(brokerCredentials BrokerCredentials) Option {
	return func(c *config) {
		c.router.Use(auth.NewWrapper(brokerCredentials.Username, brokerCredentials.Password).Wrap)
	}
}

func WithCustomAuth(authMiddleware middlewareFunc) Option {
	return func(c *config) {
		c.router.Use(authMiddleware)
	}
}

// WithEncodedPath used to opt in to a gorilla/mux behaviour that would treat encoded
// slashes "/" as IDs. For example, it would change `PUT /v2/service_instances/foo%2Fbar`
// to treat `foo%2Fbar` as an instance ID, while the default behavior was to treat it
// as `foo/bar`. However, with moving to go-chi/chi, this is now the default behavior
// so this option no longer does anything.
//
// Deprecated: no longer has any effect
func WithEncodedPath() Option {
	return func(*config) {}
}

func withDefaultMiddleware() Option {
	return func(c *config) {
		if !c.customRouter {
			c.router.Use(middlewares.APIVersionMiddleware{LoggerFactory: c.logger}.ValidateAPIVersionHdr)
			c.router.Use(middlewares.AddCorrelationIDToContext)
			c.router.Use(middlewares.AddOriginatingIdentityToContext)
			c.router.Use(middlewares.AddInfoLocationToContext)
			c.router.Use(middlewares.AddRequestIdentityToContext)
		}
	}
}

func WithOptions(opts ...Option) Option {
	return func(c *config) {
		for _, o := range opts {
			o(c)
		}
	}
}

func newDefaultConfig(logger lager.Logger) *config {
	return &config{
		router:       chi.NewRouter(),
		customRouter: false,
		logger:       logger,
	}
}

type config struct {
	router       chi.Router
	customRouter bool
	logger       lager.Logger
}

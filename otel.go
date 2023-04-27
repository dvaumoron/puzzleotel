/*
 *
 * Copyright 2023 puzzletelemetry authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */
package puzzletelemetry

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

func NewResource(serviceName string, version string, envName string) *resource.Resource {
	rsc, _ := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(version),
			attribute.String("environment", envName),
		),
	)
	return rsc
}

func NewExporter() (*prometheus.Exporter, error) {
	return prometheus.New() // TODO
}

func NewTracerProvider(exp trace.SpanExporter, rsc *resource.Resource) *trace.TracerProvider {
	tp := trace.NewTracerProvider(trace.WithBatcher(exp), trace.WithResource(rsc))
	otel.SetTracerProvider(tp)
	return tp
}
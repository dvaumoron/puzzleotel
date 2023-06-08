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
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/uptrace/opentelemetry-go-extra/otelzap"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.uber.org/zap"
)

const telemetryKey = "puzzleTelemetry"

type waitingLog struct {
	Message string
	Error   error
}

func Init(serviceName string, version string) (*otelzap.Logger, *trace.TracerProvider) {
	waitingLogs := make([]waitingLog, 0, 2)
	if godotenv.Overload() == nil {
		waitingLogs = append(waitingLogs, waitingLog{Message: "Loaded .env file"})
	}

	execEnv := os.Getenv("EXEC_ENV")

	rsc, _ := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(version),
			attribute.String("environment", execEnv),
		),
	)

	opts := make([]trace.TracerProviderOption, 0, 3)
	opts = append(opts, trace.WithSampler(trace.AlwaysSample()), trace.WithResource(rsc))
	if execEnv != "" {
		exp, err := otlptracegrpc.New(context.Background())
		if err != nil {
			waitingLogs = append(waitingLogs, waitingLog{Message: "Failed to init exporter", Error: err})
			printWaitingAndExit(waitingLogs)
		}
		opts = append(opts, trace.WithBatcher(exp))
	}

	tp := trace.NewTracerProvider(opts...)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	return newLogger(tp, waitingLogs), tp
}

func newLogger(tp *trace.TracerProvider, waitingLogs []waitingLog) *otelzap.Logger {
	logConfigPath := os.Getenv("LOG_CONFIG_PATH")
	if logConfigPath == "" {
		return defaultLogConfig(tp, waitingLogs)
	}

	logConfig, err := os.ReadFile(logConfigPath)
	if err != nil {
		waitingLogs = append(waitingLogs, waitingLog{Message: "Failed to read logging config file", Error: err})
		return defaultLogConfig(tp, waitingLogs)
	}

	var cfg zap.Config
	if err = json.Unmarshal(logConfig, &cfg); err != nil {
		waitingLogs = append(waitingLogs, waitingLog{Message: "Failed to parse logging config file", Error: err})
		return defaultLogConfig(tp, waitingLogs)
	}

	logger, err := cfg.Build()
	if err != nil {
		waitingLogs = append(waitingLogs, waitingLog{Message: "Failed to init logger with config", Error: err})
		return defaultLogConfig(tp, waitingLogs)
	}
	return otelWrap(tp, logger, waitingLogs)
}

func defaultLogConfig(tp *trace.TracerProvider, waitingLogs []waitingLog) *otelzap.Logger {
	logger, err := zap.NewProduction()
	if err != nil {
		waitingLogs = append(waitingLogs, waitingLog{Message: "Failed to init logger with config", Error: err})
		printWaitingAndExit(waitingLogs)
	}
	return otelWrap(tp, logger, waitingLogs)
}

func printWaitingAndExit(waitingLogs []waitingLog) {
	for _, waitingLog := range waitingLogs {
		if err := waitingLog.Error; err == nil {
			fmt.Println(waitingLog.Message)
		} else {
			fmt.Println(waitingLog.Message, ":", err)
		}
	}
	os.Exit(1)
}

func otelWrap(tp *trace.TracerProvider, logger *zap.Logger, waitingLogs []waitingLog) *otelzap.Logger {
	otelLogger := otelzap.New(logger)

	ctx, span := tp.Tracer(telemetryKey).Start(context.Background(), "logger/initialization")
	defer span.End()

	for _, waitingLog := range waitingLogs {
		if err := waitingLog.Error; err == nil {
			otelLogger.InfoContext(ctx, waitingLog.Message)
		} else {
			otelLogger.WarnContext(ctx, waitingLog.Message, zap.Error(err))
		}
	}
	return otelLogger
}

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
package logger

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/uptrace/opentelemetry-go-extra/otelzap"
	"go.uber.org/zap"
)

type waitingLog struct {
	Message string
	Error   error
}

func New() *otelzap.Logger {
	waitingLogs := make([]waitingLog, 0, 2)
	if godotenv.Overload() == nil {
		waitingLogs = append(waitingLogs, waitingLog{Message: "Loaded .env file"})
	}

	logConfigPath := os.Getenv("LOG_CONFIG_PATH")
	if logConfigPath == "" {
		return defaultLogConfig(waitingLogs)
	}

	logConfig, err := os.ReadFile(logConfigPath)
	if err != nil {
		waitingLogs = append(waitingLogs, waitingLog{Message: "Failed to read logging config file", Error: err})
		return defaultLogConfig(waitingLogs)
	}

	var cfg zap.Config
	if err = json.Unmarshal(logConfig, &cfg); err != nil {
		waitingLogs = append(waitingLogs, waitingLog{Message: "Failed to parse logging config file", Error: err})
		return defaultLogConfig(waitingLogs)
	}

	logger, err := cfg.Build()
	if err != nil {
		waitingLogs = append(waitingLogs, waitingLog{Message: "Failed to init logger with config", Error: err})
		return defaultLogConfig(waitingLogs)
	}
	return otelWrap(logger, waitingLogs)
}

func defaultLogConfig(waitingLogs []waitingLog) *otelzap.Logger {
	logger, err := zap.NewProduction()
	if err != nil {
		for _, waitingLog := range waitingLogs {
			if err := waitingLog.Error; err == nil {
				fmt.Println(waitingLog.Message)
			} else {
				fmt.Println(waitingLog.Message+" :", err)
			}
		}
		fmt.Println("Failed to init logging with default config :", err)
		os.Exit(1)
	}
	return otelWrap(logger, waitingLogs)
}

func otelWrap(logger *zap.Logger, waitingLogs []waitingLog) *otelzap.Logger {
	otelLogger := otelzap.New(logger)
	for _, waitingLog := range waitingLogs {
		if err := waitingLog.Error; err == nil {
			otelLogger.Info(waitingLog.Message)
		} else {
			otelLogger.Warn(waitingLog.Message, zap.Error(err))
		}
	}
	return otelLogger
}

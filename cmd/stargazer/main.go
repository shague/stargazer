// Copyright (c) 2019 Red Hat and/or its affiliates.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// The main package for the Stargazer application.
package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/nimbess/stargazer/pkg/config"
	"github.com/nimbess/stargazer/pkg/controller"
	"github.com/nimbess/stargazer/pkg/controllers/node"
	log "github.com/sirupsen/logrus"
	"os"
	"strings"
)

// VERSION is updated during the build process using git
var VERSION = "0.0.1"
var version = false
var cfgName = "stargazer"
var cfgPath = "."
var verbose = false

func init() {
	log.SetReportCaller(true)
	flag.BoolVar(&version, "v", version, "Display version")
	flag.StringVar(&cfgPath, "config-path", cfgPath, "Stargazer config file path")
	flag.StringVar(&cfgName, "config-name", cfgName, "Stargazer config file name")
	flag.BoolVar(&verbose, "verbose", verbose, "Enable debug logging")
}

func main() {
	flag.Parse()
	if version {
		fmt.Println(VERSION)
		os.Exit(0)
	}

	//cfg := new(config.Config)
	cfg := config.NewConfig()
	if err := cfg.Parse(cfgPath, cfgName); err != nil {
		log.WithField("cfgName", cfgName).WithField("cfgPath", cfgPath).
			WithError(err).Warn("Failed to parse config, using defaults")
	}

	log.WithField("config", fmt.Sprintf("%+v", *cfg)).Info("Configuration loaded")

	logLevel, err := log.ParseLevel(cfg.LogLevel)
	if err != nil {
		logLevel = log.InfoLevel
	}
	log.SetLevel(logLevel)

	stop := make(chan struct{})
	defer close(stop)
	ctx := context.Background()
	// TODO: ensure connection to etcd is available

	controllerCtrl := &controllerControl{
		ctx:              ctx,
		controllerStates: make(map[string]*controllerState),
		config:           cfg,
		stop:             stop,
	}

	// Create and start each requested controller
	for _, controllerType := range strings.Split(cfg.Controllers, ",") {
		switch controllerType {
		case "node":
			nodeController := node.NewNodeController(ctx, cfg)
			controllerCtrl.controllerStates["Node"] = &controllerState{
				controller:  nodeController,
				threadiness: cfg.NodeWorkers,
			}
		default:
			log.WithField("controller", controllerType).Info("Invalid controller")
		}
	}

	controllerCtrl.RunControllers()
}

// Object for keeping track of controller states and statuses.
type controllerControl struct {
	ctx              context.Context
	controllerStates map[string]*controllerState
	config           *config.Config
	stop             chan struct{}
}

// Runs all the controllers and blocks indefinitely.
func (cc *controllerControl) RunControllers() {
	for controllerType, cs := range cc.controllerStates {
		log.WithField("ControllerType", controllerType).Info("Starting controller")
		go cs.controller.Run(cs.threadiness, cc.stop)
	}
	select {}
}

// Object for keeping track of Controller information.
type controllerState struct {
	controller  controller.Controller
	threadiness int
}

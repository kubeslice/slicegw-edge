/*  Copyright (c) 2023 Avesha, Inc. All rights reserved.
 *
 *  SPDX-License-Identifier: Apache-2.0
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *
 *  http://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 */

package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/kubeslice/slicegw-edge/pkg/edgeservice"
	"github.com/kubeslice/slicegw-edge/pkg/logger"

	sysctl "github.com/lorenzosaino/go-sysctl"
	"google.golang.org/grpc"
)

var (
	log *logger.Logger = logger.NewLogger()
)

func startGrpcServer(grpcPort string) error {
	address := fmt.Sprintf(":%s", grpcPort)
	log.Infof("Starting GRPC Server for SliceGw-Edge Pod at %v", address)

	ln, err := net.Listen("tcp", address)
	if err != nil {
		log.Errorf("Unable to listen on address: %v, err: %v", address, err.Error())
		return err
	}

	srv := grpc.NewServer()
	edgeservice.RegisterGwEdgeServiceServer(srv, &edgeservice.GwEdgeService{})

	err = srv.Serve(ln)
	if err != nil {
		log.Errorf("Failed to start GRPC server: %v", err.Error())
		return err
	}

	log.Infof("GRPC Server is exiting...")

	return nil
}

// shutdownHandler triggers application shutdown.
func shutdownHandler(wg *sync.WaitGroup) {
	// signChan channel is used to transmit signal notifications.
	signChan := make(chan os.Signal, 1)
	// Catch and relay certain signal(s) to signChan channel.
	signal.Notify(signChan, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// Blocking until a signal is sent over signChan channel. Progress to
	// next line after signal
	sig := <-signChan
	log.Infof("Teardown started with ", sig, "signal")

	wg.Done()
}

func main() {
	var grpcPort string = "5000"

	// Get value of a net.ipv4.ip_forward using sysctl
	val, err := sysctl.Get("net.ipv4.ip_forward")
	if err != nil {
		log.Fatalf("Retrive of ipv4.ip_forward errored %v", err)
	}
	if val != "1" {
		// Set value of a net.ipv4.ip_forward to 1 using sysctl
		err = sysctl.Set("net.ipv4.ip_forward", "1")
		if err != nil {
			log.Fatalf("Set of ipv4.ip_forward errored %v", err)
		}
	}
	wg := &sync.WaitGroup{}
	wg.Add(2)

	// Start the GRPC Server that the worker operator connects to.
	go startGrpcServer(grpcPort)

	go shutdownHandler(wg)

	wg.Wait()
	log.Infof("Slice Gateway Edge Server is exiting")
}

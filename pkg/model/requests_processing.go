/*
 * Copyright (C) 2019-Present Pivotal Software, Inc. All rights reserved.
 *
 * This program and the accompanying materials are made available under the terms
 * of the Apache License, Version 2.0 (the "License”); you may not use this file
 * except in compliance with the License. You may obtain a copy of the License at:
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software distributed
 * under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR
 * CONDITIONS OF ANY KIND, either express or implied. See the License for the
 * specific language governing permissions and limitations under the License.
 */

package model

import (
	"fmt"
	"log"
	"time"

	"skenario/pkg/simulator"
)

type RequestsProcessingStock interface {
	simulator.ThroughStock
	RequestCount() int32
}

type requestsProcessingStock struct {
	env                   simulator.Environment
	delegate              simulator.ThroughStock
	replicaNumber         int
	requestsExhausted     simulator.ThroughStock
	requestsComplete      simulator.SinkStock
	numRequestsSinceLast  int32
	replicaMaxRPSCapacity int64 // unused
}

func (rps *requestsProcessingStock) Name() simulator.StockName {
	name := fmt.Sprintf("%s [%d]", rps.delegate.Name(), rps.replicaNumber)
	return simulator.StockName(name)
}

func (rps *requestsProcessingStock) KindStocked() simulator.EntityKind {
	return rps.delegate.KindStocked()
}

func (rps *requestsProcessingStock) Count() uint64 {
	return rps.delegate.Count()
}

func (rps *requestsProcessingStock) EntitiesInStock() []*simulator.Entity {
	return rps.delegate.EntitiesInStock()
}

func (rps *requestsProcessingStock) Remove() simulator.Entity {
	return rps.delegate.Remove()
}

func (rps *requestsProcessingStock) Add(entity simulator.Entity) error {
	// TODO: this isn't correct anymore
	//rps.numRequestsSinceLast++

	log.Printf("processing: %v  exhausted: %v  completed: %v\n",
		len(rps.EntitiesInStock()),
		len(rps.requestsExhausted.EntitiesInStock()),
		len(rps.requestsComplete.EntitiesInStock()))

	req, ok := entity.(*requestEntity)
	if !ok {
		return fmt.Errorf("requests processing stock only supports request entities. got %T", entity)
	}
	cpuSecondsRemaining := req.cpuSecondsRequired - req.cpuSecondsConsumed

	// Request requires processing
	if cpuSecondsRemaining > 0 {
		interruptSeconds := cpuSecondsRemaining
		if interruptSeconds > 200*time.Millisecond {
			interruptSeconds = 200 * time.Millisecond
		}
		req.cpuSecondsConsumed += interruptSeconds
		rps.env.AddToSchedule(simulator.NewMovement(
			"interrupt_request",
			rps.env.CurrentMovementTime().Add(interruptSeconds),
			rps,
			rps,
		))
		return rps.delegate.Add(entity)
	}

	// Request is exhausted
	rps.env.AddToSchedule(simulator.NewMovement(
		"interrupt_request",
		rps.env.CurrentMovementTime().Add(time.Nanosecond),
		rps,
		rps,
	))
	rps.env.AddToSchedule(simulator.NewMovement(
		"complete_request",
		rps.env.CurrentMovementTime().Add(2*time.Nanosecond),
		rps.requestsExhausted,
		rps.requestsComplete,
	))
	return rps.requestsExhausted.Add(entity)
}

func (rps *requestsProcessingStock) RequestCount() int32 {
	rc := rps.numRequestsSinceLast
	rps.numRequestsSinceLast = 0
	return rc
}

func NewRequestsProcessingStock(env simulator.Environment, replicaNumber int, requestSink simulator.SinkStock, replicaMaxRPSCapacity int64) RequestsProcessingStock {
	return &requestsProcessingStock{
		env:                   env,
		delegate:              simulator.NewThroughStock("RequestsProcessing", "Request"),
		replicaNumber:         replicaNumber,
		requestsExhausted:     simulator.NewThroughStock("RequestsProcessing", "Request"),
		requestsComplete:      requestSink,
		replicaMaxRPSCapacity: replicaMaxRPSCapacity,
	}
}

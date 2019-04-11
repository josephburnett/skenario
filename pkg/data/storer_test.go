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

package data

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bvinc/go-sqlite-lite/sqlite3"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"skenario/pkg/model"
	"skenario/pkg/simulator"
)

func TestStorer(t *testing.T) {
	spec.Run(t, "Storer", testStorer, spec.Report(report.Terminal{}))
}

func testStorer(t *testing.T, describe spec.G, it spec.S) {
	var subject Storer
	var env simulator.Environment
	var startAt time.Time
	var runFor time.Duration
	var clusterConf model.ClusterConfig
	var kpaConf model.KnativeAutoscalerConfig

	it.Before(func() {
		startAt = time.Unix(0, 0)
		runFor = 10 * time.Minute
		env = simulator.NewEnvironment(context.Background(), startAt, runFor)

		clusterConf = model.ClusterConfig{
			LaunchDelay:      11 * time.Second,
			TerminateDelay:   22 * time.Second,
			NumberOfRequests: 33,
		}
		kpaConf = model.KnativeAutoscalerConfig{
			TickInterval:                11 * time.Second,
			StableWindow:                22 * time.Second,
			PanicWindow:                 33 * time.Second,
			ScaleToZeroGracePeriod:      44 * time.Second,
			TargetConcurrencyDefault:    5.5,
			TargetConcurrencyPercentage: 6.6,
			MaxScaleUpRate:              77,
		}
	})

	describe("Store()", func() {
		var completed []simulator.CompletedMovement
		var ignored []simulator.IgnoredMovement
		var conn *sqlite3.Conn
		var err error

		it.Before(func() {
			var dir string
			dir, err = os.Getwd()
			require.NoError(t, err)
			dbPath := filepath.Join(dir, "skenario_test.db")

			os.Remove(dbPath)

			subject = &storer{}

			completed, ignored, err = env.Run()
			assert.NoError(t, err)

			err = subject.Store(dbPath, completed, ignored, clusterConf, kpaConf)
			assert.NoError(t, err)

			conn, err = sqlite3.Open(dbPath)
			require.NoError(t, err)
		})

		it.After(func() {
			err = conn.Close()
			assert.NoError(t, err)
		})

		describe("scenario run metadata", func() {
			var recorded, origin, trafficPattern string
			var count int

			it.Before(func() {
				singleQuery(t, conn, `select recorded, origin, traffic_pattern from scenario_runs`, &recorded, &origin, &trafficPattern)
				singleQuery(t, conn, `select count(1) from scenario_runs`, &count)
			})

			it("inserts a record", func() {
				assert.Equal(t, 1, count)
			})

			it("records a timestamp", func() {
				assert.Contains(t, recorded, time.Now().Format(time.RFC3339))
			})

			it("sets the origin as 'skenario_cli'", func() {
				assert.Equal(t, "skenario_cli", origin)
			})

			it("sets the traffic pattern as 'golang_rand_uniform'", func() {
				assert.Equal(t, "golang_rand_uniform", trafficPattern)
			})
		})

		describe("scenario parameters", func() {
			var launchDelay, termDelay, numRequests int
			var tickInterval, stableWindow, panicWindow, scaleToZeroGrace int
			var concurrencyDefault, concurrencyPercent, maxScaleUp float64

			it.Before(func() {
				singleQuery(t, conn, `
					select cluster_launch_delay
						 , cluster_terminate_delay
						 , cluster_number_of_requests
						 , autoscaler_tick_interval
						 , autoscaler_stable_window
						 , autoscaler_panic_window
						 , autoscaler_scale_to_zero_grace_period
						 , autoscaler_target_concurrency_default
						 , autoscaler_target_concurrency_percentage
						 , autoscaler_max_scale_up_rate
					from scenario_runs `,
					&launchDelay, &termDelay, &numRequests, &tickInterval, &stableWindow, &panicWindow, &scaleToZeroGrace,
					&concurrencyDefault, &concurrencyPercent, &maxScaleUp,
				)
			})

			it("sets cluster configuration", func() {
				assert.Equal(t, 11000000000, launchDelay)
				assert.Equal(t, 22000000000, termDelay)
				assert.Equal(t, 33, numRequests)
			})

			it("sets autoscaler configuration", func() {
				assert.Equal(t, 11000000000, tickInterval)
				assert.Equal(t, 22000000000, stableWindow)
				assert.Equal(t, 33000000000, panicWindow)
				assert.Equal(t, 44000000000, scaleToZeroGrace)
				assert.Equal(t, 5.5, concurrencyDefault)
				assert.Equal(t, 6.6, concurrencyPercent)
				assert.Equal(t, 77.0, maxScaleUp)
			})
		})

		describe("entity records", func() {
			var entityCount int
			var name, kind string

			it.Before(func() {
				singleQuery(t, conn, `select count(1) from entities`, &entityCount)
				singleQuery(t, conn, `select name, kind from entities`, &name, &kind)
			})

			it("inserts a record", func() {
				assert.Equal(t, 1, entityCount)
			})

			it("inserts a name", func() {
				assert.Equal(t, "Scenario", name)
			})

			it("inserts a kind", func() {
				assert.Equal(t, "Scenario", kind)
			})
		})

		describe("stock records", func() {
			var stocksCount int
			var name, kind string
			var numStocksWithEmptySimulation = 3

			it.Before(func() {
				singleQuery(t, conn, `select count(1) from stocks`, &stocksCount)
				singleQuery(t, conn, `select name, kind_stocked from stocks`, &name, &kind)
			})

			it("inserts a record", func() {
				assert.Equal(t, numStocksWithEmptySimulation, stocksCount)
			})

			it("inserts a name", func() {
				assert.Equal(t, "BeforeScenario", name)
			})

			it("inserts a kind", func() {
				assert.Equal(t, "Scenario", kind)
			})
		})
	})
}

func singleQuery(t *testing.T, conn *sqlite3.Conn, sql string, scanDst ...interface{}) {
	selectStmt, err := conn.Prepare(sql)
	require.NoError(t, err)
	defer selectStmt.Close()

	hasResult, err := selectStmt.Step()
	require.True(t, hasResult)
	require.NoError(t, err)

	err = selectStmt.Scan(scanDst...)
	require.NoError(t, err)
}

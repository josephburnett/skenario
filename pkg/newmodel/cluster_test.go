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

package newmodel

import (
	"context"
	"testing"
	"time"

	"github.com/knative/serving/pkg/autoscaler"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"github.com/stretchr/testify/assert"

	"knative-simulator/pkg/newsimulator"
)

func TestCluster(t *testing.T) {
	spec.Run(t, "Cluster model", testCluster, spec.Report(report.Terminal{}))
}

func testCluster(t *testing.T, describe spec.G, it spec.S) {
	var subject ClusterModel
	var envFake = new(fakeEnvironment)

	it.Before(func() {
		subject = NewCluster(envFake)
		assert.NotNil(t, subject)
	})

	describe("NewCluster()", func() {
		it("sets an environment", func() {
			assert.Equal(t, envFake, subject.Env())
		})
	})

	describe("CurrentDesired()", func() {
		it("defaults to 0", func() {
			assert.Equal(t, int32(0), subject.CurrentDesired())
		})
	})

	describe("SetDesired()", func() {
		var rawSubject *clusterModel

		describe("there are launching replicas but no active replicas", func() {
			describe("new value > launching replicas", func() {
				it.Before(func() {
					subject = NewCluster(envFake)
					assert.NotNil(t, subject)

					rawSubject = subject.(*clusterModel)
					envFake.movements = make([]newsimulator.Movement, 0)

					err := rawSubject.replicasLaunching.Add(newsimulator.NewEntity("already launching", newsimulator.EntityKind("Replica")))
					assert.NoError(t, err)

					subject.SetDesired(9)
				})

				it("updates the number of desired replicas", func() {
					assert.Equal(t, int32(9), subject.CurrentDesired())
				})

				it("Adds replica entities to ReplicasLaunching to bring them up to desired", func() {
					assert.Equal(t, uint64(9), rawSubject.replicasLaunching.Count())
				})

				it("schedules movements of new entities from ReplicasLaunching to ReplicasActive", func() {
					assert.Len(t, envFake.movements, 8)
					assert.Equal(t, newsimulator.MovementKind("launching -> active"), envFake.movements[0].Kind())
				})
			})

			describe("new value < launching replicas", func() {
				it.Before(func() {
					subject = NewCluster(envFake)
					assert.NotNil(t, subject)

					rawSubject = subject.(*clusterModel)
					envFake.movements = make([]newsimulator.Movement, 0)

					err := rawSubject.replicasLaunching.Add(newsimulator.NewEntity("already launching", newsimulator.EntityKind("Replica")))
					assert.NoError(t, err)

					subject.SetDesired(0)
				})

				it("updates the number of desired replicas", func() {
					assert.Equal(t, int32(0), subject.CurrentDesired())
				})

				it("schedules movements from ReplicasLaunching to ReplicasTerminating", func() {
					assert.Len(t, envFake.movements, 1)
					assert.Equal(t, newsimulator.MovementKind("launching -> terminated"), envFake.movements[0].Kind())
				})
			})

			describe("new value == launching replicas", func() {
				it.Before(func() {
					subject = NewCluster(envFake)
					assert.NotNil(t, subject)

					rawSubject = subject.(*clusterModel)
					envFake.movements = make([]newsimulator.Movement, 0)

					err := rawSubject.replicasLaunching.Add(newsimulator.NewEntity("already launching 1", newsimulator.EntityKind("Replica")))
					assert.NoError(t, err)
					err = rawSubject.replicasLaunching.Add(newsimulator.NewEntity("already launching 2", newsimulator.EntityKind("Replica")))
					assert.NoError(t, err)

					subject.SetDesired(2)
					subject.SetDesired(2)
				})

				it("doesn't change anything", func() {
					assert.Equal(t, int32(2), subject.CurrentDesired())
					assert.Equal(t, uint64(2), rawSubject.replicasLaunching.Count())
				})
			})
		})

		describe("there are active replicas but no launching replicas", func() {
			it.Before(func() {
				subject = NewCluster(envFake)
				assert.NotNil(t, subject)

				rawSubject = subject.(*clusterModel)
				envFake.movements = make([]newsimulator.Movement, 0)

				err := rawSubject.replicasActive.Add(newsimulator.NewEntity("already active", newsimulator.EntityKind("Replica")))
				assert.NoError(t, err)
			})

			describe("new value > active replicas", func() {
				it.Before(func() {
					subject.SetDesired(2)
				})

				it("updates the number of desired replicas", func() {
					assert.Equal(t, int32(2), subject.CurrentDesired())
				})

				it("Adds one replica entity to ReplicasLaunching to close the gap between ReplicasActive and desired", func() {
					assert.Equal(t, uint64(1), rawSubject.replicasLaunching.Count())
				})

				it("schedules movements of new entities from ReplicasLaunching to ReplicasActive", func() {
					assert.Len(t, envFake.movements, 1)
					assert.Equal(t, newsimulator.MovementKind("launching -> active"), envFake.movements[0].Kind())
				})
			})

			describe("new value < active replicas", func() {
				it.Before(func() {
					subject.SetDesired(0)
				})

				it("updates the number of desired replicas", func() {
					assert.Equal(t, int32(0), subject.CurrentDesired())
				})

				it("schedules movements from ReplicasActive to ReplicasTerminating", func() {
					assert.Len(t, envFake.movements, 1)
					assert.Equal(t, newsimulator.MovementKind("active -> terminated"), envFake.movements[0].Kind())
				})
			})

			describe("new value == active replicas", func() {
				it.Before(func() {
					subject.SetDesired(1)
					subject.SetDesired(1)
				})

				it("doesn't change anything", func() {
					assert.Equal(t, int32(1), subject.CurrentDesired())
					assert.Equal(t, uint64(1), rawSubject.replicasActive.Count())
				})
			})
		})

		describe("there is a mix of active and launching replicas", func() {
			it.Before(func() {
				subject = NewCluster(envFake)
				assert.NotNil(t, subject)

				rawSubject = subject.(*clusterModel)
				envFake.movements = make([]newsimulator.Movement, 0)

				err := rawSubject.replicasActive.Add(newsimulator.NewEntity("already active", newsimulator.EntityKind("Replica")))
				assert.NoError(t, err)
				err = rawSubject.replicasLaunching.Add(newsimulator.NewEntity("already launching", newsimulator.EntityKind("Replica")))
				assert.NoError(t, err)
			})

			describe("new value > active replicas + launching replicas", func() {
				it.Before(func() {
					subject.SetDesired(3)
				})

				it("updates the number of desired replicas", func() {
					assert.Equal(t, int32(3), subject.CurrentDesired())
				})

				it("Adds another replica entity to ReplicasLaunching to close the gap between ReplicasLaunching + ReplicasActive and desired", func() {
					assert.Equal(t, uint64(2), rawSubject.replicasLaunching.Count())
				})

				it("adds another movement from ReplicasLaunching to ReplicasActive", func() {
					assert.Len(t, envFake.movements, 1)
					assert.Equal(t, newsimulator.MovementKind("launching -> active"), envFake.movements[0].Kind())
				})
			})

			describe("new value < active replicas + launching replicas", func() {
				it.Before(func() {
					subject.SetDesired(0)
				})

				it("updates the number of desired replicas", func() {
					assert.Equal(t, int32(0), subject.CurrentDesired())
				})

				it("schedules movements from ReplicasActive to ReplicasTerminating", func() {
					assert.Len(t, envFake.movements, 2)
					assert.Equal(t, "launching -> terminated", string(envFake.movements[0].Kind()))
					assert.Equal(t, "active -> terminated", string(envFake.movements[1].Kind()))
				})
			})

			describe("new value == active replicas + launching replicas", func() {
				it.Before(func() {
					subject.SetDesired(2)
					subject.SetDesired(2)
				})

				it("doesn't change anything", func() {
					assert.Equal(t, int32(2), subject.CurrentDesired())
					assert.Equal(t, uint64(1), rawSubject.replicasActive.Count())
					assert.Equal(t, uint64(1), rawSubject.replicasLaunching.Count())
				})
			})
		})

		describe("there are no active or launching replicas", func() {
			describe("new value > 0", func() {
				it.Before(func() {
					subject = NewCluster(envFake)
					assert.NotNil(t, subject)

					rawSubject = subject.(*clusterModel)
					envFake.movements = make([]newsimulator.Movement, 0)

					subject.SetDesired(1)
				})

				it("updates the number of desired replicas", func() {
					assert.Equal(t, int32(1), subject.CurrentDesired())
				})

				it("Adds replica entities to ReplicasLaunching to bring them up to desired", func() {
					assert.Equal(t, uint64(1), rawSubject.replicasLaunching.Count())
				})

				it("schedules movements of new entities from ReplicasLaunching to ReplicasActive", func() {
					assert.Len(t, envFake.movements, 1)
					assert.Equal(t, newsimulator.MovementKind("launching -> active"), envFake.movements[0].Kind())
				})

			})
		})
	})

	describe("CurrentLaunching()", func() {
		it.Before(func() {
			subject.SetDesired(7)
		})

		it("gives the .Count() of replicas launching", func() {
			assert.Equal(t, uint64(7), subject.CurrentLaunching())
		})
	})

	describe("CurrentActive()", func() {
		var rawSubject *clusterModel

		it.Before(func() {
			rawSubject = subject.(*clusterModel)
			rawSubject.replicasActive.Add(newsimulator.NewEntity("first entity", "Replica"))
			rawSubject.replicasActive.Add(newsimulator.NewEntity("second entity", "Replica"))
		})

		it("gives the .Count() of replicas active", func() {
			assert.Equal(t, uint64(2), subject.CurrentActive())
		})
	})

	describe("RecordToAutoscaler()", func() {
		var autoscalerFake *fakeAutoscaler
		var rawSubject *clusterModel
		var firstRecorded autoscaler.Stat
		var theTime = time.Now()
		var ctx = context.Background()

		it.Before(func() {
			rawSubject = subject.(*clusterModel)

			autoscalerFake = &fakeAutoscaler{
				recorded:   make([]autoscaler.Stat, 0),
				scaleTimes: make([]time.Time, 0),
			}

			rawSubject.replicasActive.Add(newsimulator.NewEntity("Test Replica 1", newsimulator.EntityKind("Replica")))
			rawSubject.replicasActive.Add(newsimulator.NewEntity("Test Replica 2", newsimulator.EntityKind("Replica")))
			rawSubject.replicasActive.Add(newsimulator.NewEntity("Test Replica 3", newsimulator.EntityKind("Replica")))

			subject.RecordToAutoscaler(autoscalerFake, &theTime, ctx)
			firstRecorded = autoscalerFake.recorded[0]
		})

		describe("Records added to the Autoscaler", func() {
			it("records once for each replica in ReplicasActive", func() {
				assert.Len(t, autoscalerFake.recorded, 3)
			})

			it("sets time to the movement OccursAt", func() {
				assert.Equal(t, &theTime, firstRecorded.Time)
			})

			it("sets the PodName to Replica name", func() {
				assert.Equal(t, "Test Replica 1", firstRecorded.PodName)
			})

			it("sets AverageConcurrentRequests to 1", func() {
				assert.Equal(t, float64(1.0), firstRecorded.AverageConcurrentRequests)
			})

			it("sets RequestCount to 1", func() {
				assert.Equal(t, int32(1), firstRecorded.RequestCount)
			})
		})
	})
}

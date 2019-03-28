package newsimulator

import (
	"testing"
	"time"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"github.com/stretchr/testify/assert"
)

func TestMovementPQ(t *testing.T) {
	spec.Run(t, "Movement priority queue", testMovementPQ, spec.Report(report.Terminal{}))
}

func testMovementPQ(t *testing.T, describe spec.G, it spec.S) {
	var subject MovementPriorityQueue
	var movement Movement
	var theTime time.Time

	describe("EnqueueMovement()", func() {
		it.Before(func() {
			theTime = time.Now()
			subject = NewMovementPriorityQueue()
			movement = NewMovement("test movement kind", theTime, nil, nil, "test movement")
		})

		it("adds Movements", func() {
			err := subject.EnqueueMovement(movement)
			assert.NoError(t, err)
		})

		it("returns an error if a movement tries to schedule for the same time as another movement", func() {
			err := subject.EnqueueMovement(movement)
			sameTimeMovement := NewMovement("another movement kind", theTime, nil, nil, "different movement, same time")

			err = subject.EnqueueMovement(sameTimeMovement)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "could not add Movement")
			assert.Contains(t, err.Error(), "there is already another movement scheduled at that time")
		})
	})

	describe("DequeueMovement()", func() {
		it.Before(func() {
			subject = NewMovementPriorityQueue()
			movement = NewMovement("test movement kind", time.Now(), nil, nil, "test movement")
		})

		it("returns Movements", func() {
			var dqmv Movement
			var err error
			err = subject.EnqueueMovement(movement)
			assert.NoError(t, err)

			dqmv, err, _ = subject.DequeueMovement()

			subject.Close()

			assert.NoError(t, err)
			assert.Equal(t, movement, dqmv)
		})

		it("returns a 'closed' flag to indicate whether the queue has closed", func() {
			var closed bool
			var err error

			subject.Close()
			mv, err, closed := subject.DequeueMovement()

			assert.Nil(t, mv)
			assert.NoError(t, err)
			assert.True(t, closed)

		})
	})

	describe("Close()", func() {
		it.Before(func() {
			subject = NewMovementPriorityQueue()
			movement = NewMovement("test movement kind", time.Now(), nil, nil, "test movement")
		})

		it("closes the heap", func() {
			subject.Close()
			assert.True(t, subject.IsClosed())
		})
	})

	describe("IsClosed()", func() {
		it.Before(func() {
			subject = NewMovementPriorityQueue()
			movement = NewMovement("test movement kind", time.Now(), nil, nil, "test movement")
		})

		it("starts false", func() {
			assert.False(t, subject.IsClosed())
		})
	})

	describe("helpers", func() {
		describe("occursAtToKey()", func() {
			it.Before(func() {
				movement = NewMovement("test movement kind", time.Unix(0, 111000111), nil, nil, "occurs at test movement")
			})

			it("returns the OccursAt() as a string", func() {
				key, err := occursAtToKey(movement)
				assert.NoError(t, err)
				assert.Equal(t, "111000111", key)
			})
		})

		describe("leftMovementIsEarlier()", func() {
			var earlier, later Movement

			it.Before(func() {
				earlier = NewMovement("test movement kind", time.Unix(111, 0), nil, nil, "earlier test movement")
				later = NewMovement("test movement kind", time.Unix(999, 0), nil, nil, "later test movement")
			})

			describe("when the first argument is earlier", func() {
				it("returns true", func() {
					assert.True(t, leftMovementIsEarlier(earlier, later))
				})
			})

			describe("when the second argument is earlier", func() {
				it("returns false", func() {
					assert.False(t, leftMovementIsEarlier(later, earlier))
				})
			})
		})
	})
}
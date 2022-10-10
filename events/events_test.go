package events

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

// TestEventPublishingAndSubscribing creates EventEmitter objects, subscribes EventHandler callbacks to them, and
// ensures that the events are received as intended.
func TestEventPublishingAndSubscribing(t *testing.T) {
	// Define some event types
	type TestEventA struct{}
	type TestEventB struct{}

	// Create event emitters for both events.
	eventAEmitter1 := EventEmitter[TestEventA]{}
	eventAEmitter2 := EventEmitter[TestEventA]{}
	eventBEmitter1 := EventEmitter[TestEventB]{}
	eventBEmitter2 := EventEmitter[TestEventB]{}

	// Create a dictionary to track event callback
	var eventAEmitter1PublishCount,
		eventAEmitter2PublishCount,
		eventBEmitter1PublishCount,
		eventBEmitter2PublishCount,
		eventAEmitterGlobalPublishCount,
		eventBEmitterGlobalPublishCount int

	// Create our callback methods for each event, where we update our count of published events.
	eventAEmitter1.Subscribe(func(event TestEventA) {
		eventAEmitter1PublishCount++
	})
	eventAEmitter2.Subscribe(func(event TestEventA) {
		eventAEmitter2PublishCount++
	})
	eventBEmitter1.Subscribe(func(event TestEventB) {
		eventBEmitter1PublishCount++
	})
	eventBEmitter2.Subscribe(func(event TestEventB) {
		eventBEmitter2PublishCount++
	})
	SubscribeAny(func(event TestEventA) {
		eventAEmitterGlobalPublishCount++
	})
	SubscribeAny(func(event TestEventB) {
		eventBEmitterGlobalPublishCount++
	})

	// Publish events a given amount of times.
	const (
		expectedEventAEmitter1PublishCount = 2
		expectedEventAEmitter2PublishCount = 5
		expectedEventBEmitter1PublishCount = 9
		expectedEventBEmitter2PublishCount = 13
	)
	for i := 0; i < expectedEventAEmitter1PublishCount; i++ {
		eventAEmitter1.Publish(TestEventA{})
	}
	for i := 0; i < expectedEventAEmitter2PublishCount; i++ {
		eventAEmitter2.Publish(TestEventA{})
	}
	for i := 0; i < expectedEventBEmitter1PublishCount; i++ {
		eventBEmitter1.Publish(TestEventB{})
	}
	for i := 0; i < expectedEventBEmitter2PublishCount; i++ {
		eventBEmitter2.Publish(TestEventB{})
	}

	// Assert we received the expected amount of callbacks.
	assert.EqualValues(t, expectedEventAEmitter1PublishCount, eventAEmitter1PublishCount)
	assert.EqualValues(t, expectedEventAEmitter2PublishCount, eventAEmitter2PublishCount)
	assert.EqualValues(t, expectedEventBEmitter1PublishCount, eventBEmitter1PublishCount)
	assert.EqualValues(t, expectedEventBEmitter2PublishCount, eventBEmitter2PublishCount)
	assert.EqualValues(t, expectedEventAEmitter1PublishCount+expectedEventAEmitter2PublishCount, eventAEmitterGlobalPublishCount)
	assert.EqualValues(t, expectedEventBEmitter1PublishCount+expectedEventBEmitter2PublishCount, eventBEmitterGlobalPublishCount)
}

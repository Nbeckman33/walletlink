// Copyright (c) 2018-2020 WalletLink.org <https://www.walletlink.org/>
// Copyright (c) 2018-2020 Coinbase, Inc. <https://www.coinbase.com/>
// Licensed under the Apache License, version 2.0

package rpc

import (
	"sync"

	"github.com/walletlink/walletlink/util"
)

// Subscriber - channel that takes in any type
type Subscriber chan<- interface{}

type subscriberSet map[Subscriber]struct{}

// PubSub - pub/sub interface for message senders
type PubSub struct {
	lock   sync.Mutex
	subMap map[string]subscriberSet      // Subscription ID -> Subscribers
	idMap  map[Subscriber]util.StringSet // Subscriber -> Subscription IDs
}

// NewPubSub - construct a PubSub
func NewPubSub() *PubSub {
	return &PubSub{
		subMap: map[string]subscriberSet{},
		idMap:  map[Subscriber]util.StringSet{},
	}
}

// Subscribe - subscribes a Subscriber to an id
func (cm *PubSub) Subscribe(id string, subscriber Subscriber) {
	if id == "" || subscriber == nil {
		return
	}
	cm.lock.Lock()
	defer cm.lock.Unlock()

	subscribers, ok := cm.subMap[id]
	if !ok {
		subscribers = subscriberSet{}
		cm.subMap[id] = subscribers
	}

	ids, ok := cm.idMap[subscriber]
	if !ok {
		ids = util.NewStringSet()
		cm.idMap[subscriber] = ids
	}

	subscribers[subscriber] = struct{}{}
	ids.Add(id)
}

// Unsubscribe - unsubscribes a Subscriber from an id
func (cm *PubSub) Unsubscribe(id string, subscriber Subscriber) {
	if id == "" || subscriber == nil {
		return
	}
	cm.lock.Lock()
	defer cm.lock.Unlock()

	cm.unsubscribeOne(id, subscriber)
}

// UnsubscribeAll - unsubscribes subscriber from every id and returns the number
// of unsubscriptions done
func (cm *PubSub) UnsubscribeAll(subscriber Subscriber) int {
	if subscriber == nil {
		return 0
	}
	cm.lock.Lock()
	defer cm.lock.Unlock()

	ids, ok := cm.idMap[subscriber]
	if !ok {
		return 0
	}

	idsLen := len(ids)

	for id := range ids {
		cm.unsubscribeOne(id, subscriber)
	}

	return idsLen
}

// Len - returns the number of subscribers for a given id
func (cm *PubSub) Len(id string) int {
	if id == "" {
		return 0
	}
	cm.lock.Lock()
	defer cm.lock.Unlock()

	subscribers, ok := cm.subMap[id]
	if !ok {
		return 0
	}

	return len(subscribers)
}

// Publish - publishes a message to all subscribers of an id and returns
// the number of subscribers messaged
func (cm *PubSub) Publish(id string, msg interface{}) int {
	if id == "" {
		return 0
	}
	cm.lock.Lock()
	defer cm.lock.Unlock()

	subscribers, ok := cm.subMap[id]
	if !ok {
		return 0
	}

	for subscriber := range subscribers {
		subscriber := subscriber
		go func() {
			subscriber <- msg
		}()
	}
	return len(subscribers)
}

func (cm *PubSub) unsubscribeOne(id string, subscriber Subscriber) {
	subSet, ok := cm.subMap[id]
	if !ok {
		return
	}

	delete(subSet, subscriber)

	if len(subSet) == 0 {
		delete(cm.subMap, id)
	}

	idSet := cm.idMap[subscriber]
	idSet.Remove(id)

	if len(idSet) == 0 {
		delete(cm.idMap, subscriber)
	}
}

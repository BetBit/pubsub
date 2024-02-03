package pubsub

import (
	"context"
	"fmt"
	pb "github.com/BetBit/pubsub"
	"sync"
)

type Sub struct {
	EventName string
	rw        sync.RWMutex
	callbacks map[string]struct {
		cb        func(ctx context.Context, payload []byte)
		errorChan chan error
	}
	accept bool
	cancel chan struct{}
}

func (this *Sub) subscribeAccept(msg *pb.Event) {
	this.rw.RLock()
	defer this.rw.RUnlock()
	this.cancel <- struct{}{}
	for _, v := range this.callbacks {
		v.errorChan <- nil
	}
}

func (this *Sub) subscribeForbidden(msg *pb.Event) {
	this.rw.RLock()
	defer this.rw.RUnlock()
	this.cancel <- struct{}{}
	for _, v := range this.callbacks {
		v.errorChan <- fmt.Errorf("subscribe forbidden")
	}
}

func (this *Sub) handleMessage(msg *pb.Event) {
	this.rw.RLock()
	defer this.rw.RUnlock()
	for _, v := range this.callbacks {
		ctx := context.Background()

		if msg.Error != "" {
			_err := errMsg{
				Message: msg.Error,
			}
			ctx = _err.withContext(ctx)
		}

		mt := Meta{
			ID:        msg.Id,
			EventName: msg.EventName,
			Project:   msg.AgentId,
			Timestamp: msg.Timestamp,
		}

		go v.cb(mt.withContext(ctx), msg.Payload)
	}
}

func (this *Sub) add(errorChan chan error, id string, cb func(ctx context.Context, payload []byte)) {
	this.rw.Lock()
	defer this.rw.Unlock()
	if this.callbacks == nil {
		this.callbacks = make(map[string]struct {
			cb        func(ctx context.Context, payload []byte)
			errorChan chan error
		})
	}
	this.callbacks[id] = struct {
		cb        func(ctx context.Context, payload []byte)
		errorChan chan error
	}{cb: cb, errorChan: errorChan}
}

func (this *Sub) delete(id string) bool {
	this.rw.Lock()
	defer this.rw.Unlock()
	if this.callbacks == nil {
		return false
	}
	delete(this.callbacks, id)
	return len(this.callbacks) == 0
}

type sub struct {
	rw   sync.RWMutex
	list map[string]*Sub
}

func (this *sub) SubscribeAccept(msg *pb.Event) {
	eventName := string(msg.Payload)
	sub, ok := this.Get(eventName)
	if !ok || sub.accept {
		return
	}
	sub.accept = true
	sub.subscribeAccept(msg)
}

func (this *sub) SubscribeForbidden(msg *pb.Event) {
	eventName := string(msg.Payload)
	sub, ok := this.Get(eventName)
	if !ok {
		return
	}
	sub.accept = false
	sub.subscribeForbidden(msg)
}

func (this *sub) HandleMessage(msg *pb.Event) {
	sub, ok := this.Get(msg.EventName)
	if !ok {
		return
	}

	if !sub.accept {
		return
	}

	sub.handleMessage(msg)
}

func (this *sub) Add(cancel chan struct{}, errorChan chan error, eventName, id string, cb func(ctx context.Context, payload []byte)) bool {
	sub, ok := this.Get(eventName)
	if !ok {
		sub = &Sub{
			EventName: eventName,
			cancel:    cancel,
		}

		this.rw.Lock()
		this.list[eventName] = sub
		this.rw.Unlock()
	}
	sub.add(errorChan, id, cb)

	if ok && sub.accept {
		errorChan <- nil
	}

	return ok
}

func (this *sub) Get(eventName string) (*Sub, bool) {
	this.rw.RLock()
	sub, ok := this.list[eventName]
	this.rw.RUnlock()
	return sub, ok
}

func (this *sub) Delete(eventName string, id string) {
	sub, ok := this.Get(eventName)
	if !ok {
		return
	}

	if ok = sub.delete(id); ok {
		this.rw.Lock()
		delete(this.list, eventName)
		this.rw.Unlock()
	}
}

func newSub() *sub {
	return &sub{
		list: make(map[string]*Sub),
	}
}

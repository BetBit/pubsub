package pubsub

import (
	"context"
	"fmt"
	pb "github.com/BetBit/pubsub/proto"
	"sync"
	"time"
)

type Pub struct {
	evt              *pb.Event
	errorChan        chan error
	deleteChan       chan struct{}
	subEventName     string
	subEventCallback func(ctx context.Context, payload []byte)
	sender           *sender
	timerStop        chan struct{}
}

func (this *Pub) handleMessage(msg *pb.Event) {
	if this.subEventCallback == nil {
		return
	}

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
		Project:   msg.ProjectId,
		Timestamp: msg.Timestamp,
	}

	go func() {
		this.subEventCallback(mt.withContext(ctx), msg.Payload)
		this.deleteChan <- struct{}{}
	}()
}

func (this *Pub) Sub(eventName string, cb func(ctx context.Context, payload []byte)) *Pub {
	this.subEventName = eventName
	this.subEventCallback = cb
	return this
}

func (this *Pub) handleDelete() {
	if this.subEventCallback != nil {
		return
	}

	this.deleteChan <- struct{}{}
}

func (this *Pub) subscribeForbidden() {
	this.errorChan <- fmt.Errorf("subscribe forbidden")
	this.subEventCallback = nil
}

func (this *Pub) Do() error {
	defer func() {
		this.handleDelete()
		if this.timerStop != nil {
			this.timerStop <- struct{}{}
		}
	}()
	this.timerStart()

	if this.subEventCallback != nil {
		this.sender.Send(&pb.Event{
			Id:        this.evt.Id,
			ProjectId: this.evt.ProjectId,
			EventName: "_.check.subscribe",
			Payload:   []byte(this.subEventName),
			Timestamp: time.Now().Unix(),
		})
	} else {
		this.sender.Send(this.evt)
	}

	err := <-this.errorChan
	if err != nil {
		return err
	}

	return nil
}

func (this *Pub) cancel() {
	this.errorChan <- nil
}

func (this *Pub) messageComplete() {
	this.errorChan <- nil
}

func (this *Pub) timerStart() {
	this.timerStop = make(chan struct{})

	go func() {
		timer := time.NewTimer(5 * time.Second)
		for {
			select {
			case <-timer.C:
				timer.Reset(5 * time.Second)

				go func() {
					err := this.sender.Send(this.evt)
					if err != nil {
						return
					}
				}()

			case <-this.timerStop:
				timer.Stop()

				return
			}
		}
	}()
}

func (this *Pub) eventForbidden() {
	this.errorChan <- fmt.Errorf("event forbidden")
	this.subEventCallback = nil
}

func (this *Pub) subscribeAccept() {
	this.sender.Send(this.evt)
}

type pub struct {
	rw   sync.RWMutex
	list map[string]*Pub
}

func (this *pub) SubscribeAccept(msg *pb.Event) {
	pub, ok := this.Get(msg.Id)
	if !ok {
		return
	}

	eventName := string(msg.Payload)
	if pub.subEventName != eventName {
		return
	}

	pub.subscribeAccept()
}

func (this *pub) SubscribeForbidden(msg *pb.Event) {
	pub, ok := this.Get(msg.Id)
	if !ok {
		return
	}

	eventName := string(msg.Payload)
	if pub.subEventName != eventName {
		return
	}

	pub.subscribeForbidden()
}

func (this *pub) EventForbidden(msg *pb.Event) {
	pub, ok := this.Get(msg.Id)
	if !ok {
		return
	}

	eventName := string(msg.Payload)
	if pub.evt.EventName != eventName {
		return
	}

	pub.eventForbidden()
}

func (this *pub) HandleMessage(msg *pb.Event) {
	pub, ok := this.Get(msg.Id)
	if !ok {
		return
	}

	if pub.subEventName != msg.EventName {
		return
	}

	pub.handleMessage(msg)
}

func (this *pub) Add(id string, pub *Pub) {
	this.rw.Lock()
	defer this.rw.Unlock()

	this.list[id] = pub
}

func (this *pub) Delete(id string) {
	this.rw.Lock()
	defer this.rw.Unlock()

	if _, ok := this.list[id]; ok {
		delete(this.list, id)
	}
}

func (this *pub) MessageComplete(id string) {
	pub, ok := this.Get(id)
	if !ok {
		return
	}
	pub.messageComplete()
}

func (this *pub) Get(id string) (*Pub, bool) {
	this.rw.RLock()
	defer this.rw.RUnlock()

	pub, ok := this.list[id]
	return pub, ok
}

func newPub() *pub {
	return &pub{
		list: make(map[string]*Pub),
	}
}

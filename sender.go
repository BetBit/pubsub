package pubsub

import (
	"fmt"
	pb "github.com/BetBit/pubsub/proto"
	"sync"
)

type sender struct {
	active bool
	rw     sync.RWMutex
	stream pb.PubSub_ChannelClient
}

func (this *sender) Send(evt *pb.Event) error {
	this.rw.RLock()
	defer this.rw.RUnlock()

	if !this.active {
		return fmt.Errorf("stream is not active")
	}

	return this.stream.Send(evt)
}

func (this *sender) Start(stream pb.PubSub_ChannelClient) {
	this.rw.Lock()
	this.stream = stream
	this.active = true
	this.rw.Unlock()

	go func() {
		<-stream.Context().Done()
		this.rw.Lock()
		this.active = false
		this.stream = nil
		this.rw.Unlock()
	}()
}

func newSender() *sender {
	return &sender{}
}

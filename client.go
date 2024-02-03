package pubsub

import (
	"context"
	"fmt"
	pb "github.com/BetBit/pubsub/proto"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"time"
)

type Client struct {
	clientId string
	agentId  string
	token    string
	attempts int
	conn     *grpc.ClientConn
	pub      *pub
	sub      *sub
	sender   *sender
}

// SetError - Передать ошибку с событием
func (this *Client) SetError(ctx context.Context, err error) context.Context {
	return errMsg{
		Message: err.Error(),
	}.withContext(ctx)
}

// CheckError - проеврить ошибку
func (this *Client) CheckError(ctx context.Context) error {
	return hasError(ctx)
}

// Meta - Мета-данные
func (this *Client) Meta(ctx context.Context) Meta {
	return fromContext(ctx)
}

// SetProject - Установить проект
func (this *Client) SetProject(ctx context.Context, project string) context.Context {
	mt := fromContext(ctx)
	mt.Project = project
	return mt.withContext(ctx)
}

// Pub - Публикация события
//
//	Имеет метод Sub для подписки на ответное событие
func (this *Client) Pub(ctx context.Context, eventName string, payload []byte) *Pub {
	mt := this.Meta(ctx)
	id := mt.ID
	if id == "" {
		id = uuid.New().String()
	}

	agentId := mt.Project
	if agentId == "" {
		agentId = this.agentId
	}

	errMsg := ""
	if err := hasError(ctx); err != nil {
		errMsg = err.Error()
	}

	evt := &pb.Event{
		Id:        id,
		AgentId:   agentId,
		EventName: eventName,
		Error:     errMsg,
		Payload:   payload,
		Timestamp: time.Now().Unix(),
	}

	errorChan := make(chan error, 2)
	deleteChan := make(chan struct{})
	pub := &Pub{
		evt:        evt,
		errorChan:  errorChan,
		deleteChan: deleteChan,
		sender:     this.sender,
	}

	ctx, cancel := context.WithCancel(ctx)
	this.pub.Add(id, pub)

	go func() {
		<-ctx.Done()
		pub.cancel()
	}()

	go func() {
		defer func() {
			close(deleteChan)
		}()

		<-deleteChan
		this.pub.Delete(id)
		cancel()
	}()

	return pub
}

// Sub - Подписка
func (this *Client) Sub(ctx context.Context, eventName string, cb func(ctx context.Context, payload []byte)) error {
	errorChan := make(chan error, 2)
	cancel := make(chan struct{})
	id := uuid.New().String()

	ok := this.sub.Add(cancel, errorChan, eventName, id, cb)
	if !ok {
		go func() {
			timer := time.NewTimer(time.Second * time.Duration(this.attempts))
			for {
				select {
				case <-timer.C:
					timer.Reset(time.Second * 5)
					this.sender.Send(&pb.Event{
						Id:        id,
						AgentId:   this.agentId,
						EventName: "_.check.subscribe",
						Payload:   []byte(eventName),
						Timestamp: time.Now().Unix(),
					})

				case <-cancel:
					timer.Stop()
				}
			}
		}()
	}

	go func() {
		<-ctx.Done()
		this.sub.Delete(eventName, id)
	}()

	if err := <-errorChan; err != nil {
		return err
	}

	return nil
}

func (this *Client) handleMessage(msg *pb.Event) {
	switch msg.EventName {
	// ToDo: ping/pong:
	//case "_.pong":

	case "_.hello":
		this.hello()
	case "_.message.complete":
		this.pub.MessageComplete(msg.Id)
	case "_.event.forbidden":
		this.pub.EventForbidden(msg)
	case "_.subscribe.accept":
		this.pub.SubscribeAccept(msg)
		this.sub.SubscribeAccept(msg)
	case "_.subscribe.forbidden":
		this.pub.SubscribeForbidden(msg)
		this.sub.SubscribeForbidden(msg)
	default:
		this.sub.HandleMessage(msg)
		this.pub.HandleMessage(msg)
	}
}

func (this *Client) handleError(err error) {
	switch status.Code(err) {
	case codes.Unavailable:
		this.reconnect()
	default:
		this.reconnect()
		fmt.Println("Error:", err)
	}
}

func (this *Client) observe(stream pb.PubSub_ChannelClient) {
	go func() {
		defer stream.CloseSend()
	LOOP:
		for {
			msg, err := stream.Recv()
			if msg != nil {
				this.handleMessage(msg)
			}

			if err != nil {
				this.handleError(err)
				break LOOP
			}
		}
	}()

	this.sender.Start(stream)
}

func (this *Client) connect() {
	c := pb.NewPubSubClient(this.conn)
	ctx := context.Background()

	fmt.Println("Connecting...")

	auth := &Auth{
		clientId: this.clientId,
		agentId:  this.agentId,
		token:    this.token,
	}

	stream, err := c.Channel(auth.WithContext(ctx))
	if err != nil {
		this.handleError(err)
		return
	}

	this.observe(stream)
}

func (this *Client) reconnect() {
	go func() {
		if this.attempts > 0 {
			fmt.Println("Reconnect.", this.attempts, "seconds")
		}
		time.Sleep(time.Second * time.Duration(this.attempts))
		if this.attempts == 0 {
			this.attempts = 1
		} else {
			this.attempts = this.attempts * 2
			if this.attempts > 8 {
				this.attempts = 8
			}
		}
		this.connect()
	}()
}

func (this *Client) hello() {
	this.attempts = 0
	fmt.Println("Connected")
}

type Options struct {
	ClientID    string
	Project     string
	Token       string
	Conn        *grpc.ClientConn
	ResendTimer time.Duration
}

func New(opt Options) *Client {
	m := &Client{
		clientId: opt.ClientID,
		agentId:  opt.Project,
		token:    opt.Token,
		conn:     opt.Conn,
		pub:      newPub(),
		sub:      newSub(),
		sender:   newSender(),
	}
	m.connect()
	return m
}

package main

import (
	message "examples/service"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"log"
)

type MessageSenderService struct {
	message.UnimplementedMessageSenderServer
}

func (MessageSenderService) Send(ctx context.Context, req *message.MessageRequest) (*message.MessageResponse, error) {
	log.Println("receive message:", req.GetSaySomething())
	resp := &message.MessageResponse{}
	resp.ResponseSomething = "roger that!"
	return resp, nil
}

func main() {
	srv := grpc.NewServer()
	messageSenderService := MessageSenderService{}
	//var m message.MessageSenderServer = message.MessageSenderServer.()
	message.RegisterMessageSenderServer(srv, &messageSenderService)
}

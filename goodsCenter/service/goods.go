package service

import (
	"errors"
	"time"
)

type GoodsService struct {
}

type Args struct{ Price1, Price2 int }

func (g GoodsService) Sum(args Args, reply *int) error {
	*reply = args.Price1 + args.Price2
	return nil
}

func (g GoodsService) Error(args Args, reply *int) error {
	return errors.New("rpcServer internal error")
}

func (g GoodsService) Sleep(args Args, reply *int) error {
	time.Sleep(time.Second * time.Duration(args.Price1))
	*reply = args.Price1 + args.Price2
	return nil
}

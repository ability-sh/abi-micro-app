package main

import (
	"log"
	"net"
	"os"

	srv "github.com/ability-sh/abi-micro-app/srv"
	"github.com/ability-sh/abi-micro/grpc"
	_ "github.com/ability-sh/abi-micro/logger"
	_ "github.com/ability-sh/abi-micro/lrucache"
	_ "github.com/ability-sh/abi-micro/mongodb"
	_ "github.com/ability-sh/abi-micro/oss"
	_ "github.com/ability-sh/abi-micro/redis"
	"github.com/ability-sh/abi-micro/runtime"
	"google.golang.org/grpc/reflection"
)

func main() {

	p, err := runtime.NewFilePayload("./config.yaml", runtime.NewPayload())

	if err != nil {
		log.Panicln(err)
	}

	addr := os.Getenv("ABI_MICRO_ADDR")

	if addr == "" {
		addr = ":8082"
	}

	lis, err := net.Listen("tcp", addr)

	if err != nil {
		log.Panicln(err)
	}

	s := grpc.NewServer(p)

	srv.Reg(s)

	reflection.Register(s)

	log.Println(addr)

	s.Serve(lis)
}

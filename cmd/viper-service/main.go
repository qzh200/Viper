// Command viper-service offers the service of the original software.
//
// The official PredatorSense application connects to a local port to read
// the state of the machine and to change it. Viper can offer the same
// service, so that the official application talks to Viper.
//
// The service is meant to run as a background program. It needs the rights
// of an administrator to reach the gaming provider, the settings that do
// not need them are reported without them.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"viper/internal/config"
	"viper/internal/hardware"
	"viper/internal/service"
)

func main() {
	address := flag.String("address", "", "address the service listens on, the default is the port of the original service")
	flag.Parse()

	store, err := config.Open()
	if err != nil {
		log.Println("viper-service:", err)
		os.Exit(1)
	}

	// The gaming provider needs the rights of an administrator. Without
	// them the service still runs and reports what it can.
	//
	// The provider is kept in an interface, not in a pointer: a pointer
	// that is nil inside an interface is not equal to nil, so the guards
	// of the service would let the call through and the call would crash
	// the service on the first request that needs the machine.
	var backend hardware.Gaming
	machine, err := hardware.NewBackend()
	if err != nil {
		log.Println("viper-service: 游戏硬件提供者不可用:", err)
	} else {
		backend = machine
		defer backend.Close()
	}

	provider := service.NewProvider(backend, store)

	server, err := service.New(service.Options{
		Address:  *address,
		Provider: provider,
	})
	if err != nil {
		log.Println("viper-service:", err)
		os.Exit(1)
	}
	defer server.Close()

	fmt.Println("viper-service is listening on", server.Address())

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-stop
		fmt.Println("viper-service is stopping")
		server.Close()
	}()

	if err := server.Serve(); err != nil {
		log.Println("viper-service:", err)
		os.Exit(1)
	}
}

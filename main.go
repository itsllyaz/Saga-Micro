package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/itsllyaz/go-microservices/application"
)

func main() {
	app := application.New()		

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)	
	defer cancel()

	err := app.Start(ctx)
	if err != nil{
		fmt.Println("failed to start app:", err)
	}
}


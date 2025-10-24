package main

import (
	"context"
	"fmt"
	"os"

	"github.com/leohubert/nginx-plex-auth-server/cmd"
)

func main() {
	cmdName := "api"
	if len(os.Args) >= 2 {
		cmdName = os.Args[1]
	}

	env, services, cleanup := cmd.Bootstrap(context.Background())
	defer cleanup()

	switch cmdName {
	case "api":
		cmd.ApiCmd(env, services)

	default:
		panic(fmt.Errorf("unknown command %s", cmdName))
	}
}

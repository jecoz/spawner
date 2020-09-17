package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/jecoz/spawner"
	"github.com/jecoz/spawner/ecs"
)

var (
	v  = flag.Bool("v", false, "Print software version")
	k  = flag.Bool("k", false, "Kill & purge a previously spawned world")
	ps = flag.Bool("ps", false, "Ignore any input and list running worlds")
)

var version = "N/A"

func exitErr(err error) {
	fmt.Fprintf(os.Stderr, err.Error())
	os.Exit(1)
}

func main() {
	flag.Parse()

	var s spawner.Spawner
	var err error
	s = new(ecs.Fargate)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	switch {
	case (*k && *ps) || (*k && *v) || (*ps && *v):
		flag.Usage()
		os.Exit(1)
	case *v:
		fmt.Println(version)
	case *k:
		var w spawner.World
		if err = spawner.DecodeWorld(&w, os.Stdin); err != nil {
			exitErr(err)
		}
		if err = s.Kill(ctx, w); err != nil {
			exitErr(err)
		}
	case *ps:
		var worlds []spawner.World
		if worlds, err = s.Ps(ctx); err != nil {
			exitErr(err)
		}
		if err = spawner.EncodeWorlds(os.Stdout, worlds...); err != nil {
			exitErr(err)
		}
	default:
		var w *spawner.World
		if w, err = s.Spawn(ctx, os.Stdin); err != nil {
			exitErr(err)
		}
		if err = spawner.EncodeWorlds(os.Stdout, *w); err != nil {
			exitErr(err)
		}
	}
}

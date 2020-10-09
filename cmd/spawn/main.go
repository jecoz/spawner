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
	ps = flag.Bool("ps", false, "Ignore any input and list running worlds. Requires g flag")
	g  = flag.String("g", "", "Specify galaxy domain")
)

var version = "v0.1.0"

func exitErr(err error) {
	fmt.Fprintf(os.Stderr, "%v\n", err.Error())
	os.Exit(1)
}

func main() {
	flag.Parse()

	var s spawner.Spawner
	s = ecs.NewFargate(version)

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
		if err := spawner.DecodeWorld(&w, os.Stdin); err != nil {
			exitErr(err)
		}
		if err := s.Kill(ctx, w); err != nil {
			exitErr(err)
		}
	case *ps:
		if *g == "" {
			exitErr(fmt.Errorf("ps requires g flag to be specified"))
		}
		var worlds []*spawner.World
		var err error
		if worlds, err = s.Ps(ctx, *g); err != nil {
			exitErr(err)
		}
		if err = spawner.EncodeWorlds(os.Stdout, worlds...); err != nil {
			exitErr(err)
		}
	default:
		var w *spawner.World
		var err error
		if w, err = s.Spawn(ctx, os.Stdin); err != nil {
			exitErr(err)
		}
		if err = spawner.EncodeWorlds(os.Stdout, w); err != nil {
			exitErr(err)
		}
	}
}

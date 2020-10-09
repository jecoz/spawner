package spawner

import (
	"context"
	"encoding/json"
	"io"
)

type World struct {
	Id      string `json:"id"`
	Galaxy  string `json:"galaxy"`
	Addr    string `json:"addr"`
	Spawner string `json:"spawner"`

	Details json.RawMessage `json:"details"`
}

type Spawner interface {
	// Name of this spawner.
	Name() string
	// Spawn starts a World. Once the function returns, World should
	// be ready to accept connections on World.Addr.
	Spawn(ctx context.Context, r io.Reader) (*World, error)
	// Kill stops & purges a previously spawned World.
	Kill(ctx context.Context, w World) error
	// Ps lists the currently running Worlds in the given Galaxy g.
	Ps(ctx context.Context, g string) ([]*World, error)
}

func DecodeWorld(w *World, r io.Reader) error         { return json.NewDecoder(r).Decode(w) }
func EncodeWorlds(w io.Writer, worlds ...*World) error { return json.NewEncoder(w).Encode(worlds) }

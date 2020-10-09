package main

import (
	"bufio"
	"encoding/json"
	"os"

	"github.com/jecoz/spawner/ecs"
)

func ptr(s string) *string { return &s }

func main() {
	d := ecs.TaskDefinition{
		Name:    ptr("video-encoder"),
		Cluster: ptr("keepinmind"),
		Subnets: []*string{
			ptr("subnet-1234"),
			ptr("subnet-5678"),
		},
		SecurityGroups: []*string{
			ptr("sg-1234"),
			ptr("sg-5678"),
		},
		Overrides: []*ecs.ContainerOverride{
			&ecs.ContainerOverride{
				Name: ptr("worker"),
				Command: []*string{
					ptr("ffmpeg"),
					ptr("-i"),
					ptr("this"),
					ptr("-o"),
					ptr("that"),
				},
			},
		},
	}
	b, err := json.MarshalIndent(&d, "", "\t")
	if err != nil {
		panic(err)
	}
	w := bufio.NewWriter(os.Stdout)
	w.Write(b)
	w.WriteString("\n")
	if err = w.Flush(); err != nil {
		panic(err)
	}
}

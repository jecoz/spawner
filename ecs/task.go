package ecs

type TaskDefinition struct {
	TaskDefinition string   `json:"task_definition"`
	Service        string   `json:"service"`
	Cluster        string   `json:"cluster"`
	Subnets        []string `json:"subnets"`
	SecurityGroups []string `json:"security_groups"`
}

// Task defines **what** should be executed, on **which** hardware.
type Task struct {
	ID         string          `json:"id"`
	ImageType  string          `json:"image_type"`
	Definition *TaskDefinition `json:"definition"`
}

type Container struct {
	Addr    string `json:"addr"`
	Arn     string `json:"name"`
	Cluster string `json:"cluster"`
}

package ecs

type ContainerOverride struct {
	// Name of the container. Check your task definition on the AWS console
	// to find it out.
	Name *string `json:"name"`
	// Override how the container is executed. In Docker terms, it changes
	// the RUN specification.
	Command []*string `json:"command"`

	// NOTE: we can change here container specific HW requirements, as well
	// as environment variables. Check out
	// https://github.com/aws/aws-sdk-go/blob/v1.35.5/service/ecs/api.go#L8286-L8327
}

type TaskDefinition struct {
	// The task definition name is the identification name, usually composed by
	// a user-defined task name and a revision number. Checkout the AWS console
	// to find it.
	Name *string `json:"name"`
	// Cluster specifies where the task is to be executed. The cluster must be
	// already present in the referred AWS ECS environment.
	Cluster        *string   `json:"cluster"`
	Subnets        []*string `json:"subnets"`
	SecurityGroups []*string `json:"security_groups"`
	// A task is composed by a variable number of containers. Here we can
	// override each one of them.
	// This is the place where we specify the HW requirements and, using the
	// "command override", how the tool is started.
	Overrides []*ContainerOverride

	// NOTE: it is also possible to override other global properties of the Task,
	// such as execution role, memory and CPU requirements. Check out
	// https://github.com/aws/aws-sdk-go/blob/v1.35.5/service/ecs/api.go#L18572-L18595
}

// Task contains the information required to contact or stop a running task.
// Spawner callers have to store this information if the want to be able to
// Kill the World.
type Task struct {
	Arn     *string `json:"arn"`
	Cluster *string `json:"cluster"`
	Addr    *string `json:"addr"`
}

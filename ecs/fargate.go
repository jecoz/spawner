package ecs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/jecoz/spawner"
)

const LastStatusPollInterval = time.Millisecond * time.Duration(500)

type Fargate struct {
	HostVersion string
	sess        *session.Session
	client      *ecs.ECS
}

func NewFargate(hostVersion string) *Fargate {
	sess := session.Must(session.NewSession())
	return &Fargate{
		HostVersion: hostVersion,
		sess:        sess,
		client:      ecs.New(sess),
	}
}

func (f *Fargate) Name() string { return "ecs.fargate-" + f.HostVersion }

func (f *Fargate) describeTask(ctx context.Context, cluster, arn *string) (*ecs.Task, error) {
	input := &ecs.DescribeTasksInput{
		Cluster: cluster,
		Tasks:   []*string{arn},
	}
	if err := input.Validate(); err != nil {
		return nil, err
	}
	resp, err := f.client.DescribeTasksWithContext(ctx, input)
	if err != nil {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	if len(resp.Tasks) == 0 {
		if len(resp.Failures) > 0 {
			return nil, fmt.Errorf("describe task: %v", resp.Failures[0].String())
		}
		return nil, fmt.Errorf("describe task: unable to fulfil request")
	}
	return resp.Tasks[0], nil
}

func (f *Fargate) waitRunningTask(ctx context.Context, cluster, arn *string) (task *ecs.Task, err error) {
	// Stop when the context is invalidated or the response is no longer
	// successfull. We're waiting a pending process to become running here,
	// not to resume from a lost connection.
	for {
		timer := time.NewTimer(LastStatusPollInterval)
		select {
		case <-timer.C:
			task, err = f.describeTask(ctx, cluster, arn)
			if err != nil {
				return
			}
			if *task.LastStatus == ecs.DesiredStatusRunning {
				return
			}
			// TODO: we could log each time we retry.
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			err = ctx.Err()
			return
		}

	}
}

func (f *Fargate) runTask(ctx context.Context, d TaskDefinition) (*ecs.Task, error) {
	overrides := []*ecs.ContainerOverride{}
	for _, v := range d.Overrides {
		overrides = append(overrides, &ecs.ContainerOverride{
			Name:    v.Name,
			Command: v.Command,
		})
	}
	lt := ecs.LaunchTypeFargate
	apip := ecs.AssignPublicIpEnabled
	input := &ecs.RunTaskInput{
		TaskDefinition: d.Name,
		Cluster:        d.Cluster,
		LaunchType:     &lt,
		NetworkConfiguration: &ecs.NetworkConfiguration{
			AwsvpcConfiguration: &ecs.AwsVpcConfiguration{
				AssignPublicIp: &apip,
				Subnets:        d.Subnets,
				SecurityGroups: d.SecurityGroups,
			},
		},
		Overrides: &ecs.TaskOverride{
			ContainerOverrides: overrides,
		},
	}
	if err := input.Validate(); err != nil {
		return nil, err
	}
	resp, err := f.client.RunTaskWithContext(ctx, input)
	if err != nil {
		return nil, err
	}
	if len(resp.Tasks) == 0 {
		if len(resp.Failures) > 0 {
			return nil, fmt.Errorf("run task: %v", resp.Failures[0].String())
		}
		return nil, fmt.Errorf("run task: unable to fulfil request")
	}
	return resp.Tasks[0], nil
}

func describeNetworkInterface(ctx context.Context, sess *session.Session, eni *string) (*ec2.NetworkInterface, error) {
	// NOTE: this function uses EC2. If more functions like this are needed,
	// extract them into a separte ec2 package.
	input := &ec2.DescribeNetworkInterfacesInput{
		NetworkInterfaceIds: []*string{eni},
	}
	if err := input.Validate(); err != nil {
		return nil, err
	}
	resp, err := ec2.New(sess).DescribeNetworkInterfacesWithContext(ctx, input)
	if err != nil {
		return nil, err
	}
	if len(resp.NetworkInterfaces) == 0 {
		return nil, fmt.Errorf("no interface found for %v", eni)
	}
	return resp.NetworkInterfaces[0], nil
}

func eniFromTask(task *ecs.Task) (*string, error) {
	if len(task.Attachments) == 0 {
		return nil, fmt.Errorf("missing task attachments")
	}
	var eniAttach *ecs.Attachment
	for i, v := range task.Attachments {
		if *v.Type == "ElasticNetworkInterface" {
			eniAttach = task.Attachments[i]
			break
		}
	}
	if eniAttach == nil {
		return nil, fmt.Errorf("missing ElasticNetworkInterface attachment")
	}
	var eni *string
	for _, v := range eniAttach.Details {
		if *v.Name == "networkInterfaceId" {
			eni = v.Value
			break
		}
	}
	if eni == nil || *eni == "" {
		return nil, fmt.Errorf("unable to find network interface id within eni attachment")
	}
	return eni, nil
}

func (f *Fargate) Spawn(ctx context.Context, r io.Reader) (*spawner.World, error) {
	var d TaskDefinition
	if err := json.NewDecoder(r).Decode(&d); err != nil {
		return nil, fmt.Errorf("decoding task: %w", err)
	}
	ecstask, err := f.runTask(ctx, d)
	if err != nil {
		return nil, err
	}

	// If an error occours from this point on, we need to
	// stop the task too.
	undo := true
	defer func() {
		if !undo {
			return
		}
		// Even though the original context was invalidated, we need to
		// ensure we're not leaking resources.
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()

		f.stopTask(ctx, d.Cluster, ecstask.TaskArn)
	}()

	if ecstask, err = f.waitRunningTask(ctx, d.Cluster, ecstask.TaskArn); err != nil {
		return nil, err
	}

	eni, err := eniFromTask(ecstask)
	if err != nil {
		return nil, err
	}
	ifi, err := describeNetworkInterface(ctx, f.sess, eni)
	if err != nil {
		return nil, err
	}

	task := Task{
		Arn:     ecstask.TaskArn,
		Addr:    ifi.Association.PublicIp,
		Cluster: d.Cluster,
	}

	var b bytes.Buffer
	if err := json.NewEncoder(&b).Encode(&task); err != nil {
		return nil, err
	}
	undo = false
	return &spawner.World{
		Id:      *task.Arn,
		Addr:    *task.Addr,
		Spawner: f.Name(),
		Details: json.RawMessage(b.Bytes()),
	}, nil
}

func (f *Fargate) stopTask(ctx context.Context, cluster, arn *string) error {
	input := &ecs.StopTaskInput{
		Cluster: cluster,
		Task:    arn,
	}
	if err := input.Validate(); err != nil {
		return err
	}
	_, err := f.client.StopTaskWithContext(ctx, input)
	return err
}

func (f *Fargate) Kill(ctx context.Context, w spawner.World) error {
	var t Task
	if err := json.Unmarshal(w.Details, &t); err != nil {
		return err
	}
	return f.stopTask(ctx, t.Cluster, t.Arn)
}

func (f *Fargate) Ps(ctx context.Context) ([]spawner.World, error) {
	return []spawner.World{}, fmt.Errorf("not implemented yet")
}

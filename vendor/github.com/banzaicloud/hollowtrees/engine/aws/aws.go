package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/banzaicloud/hollowtrees/conf"
	"github.com/banzaicloud/hollowtrees/engine"
	"github.com/sirupsen/logrus"
)

type AwsCloudEngine struct {
	session *session.Session
	region  string
}

var log *logrus.Entry

func New(region string) (engine.CloudEngine, error) {
	log = conf.Logger().WithField("package", "engine/aws")
	session, err := session.NewSession()
	if err != nil {
		log.Info("Error creating session ", err)
		return nil, err
	}
	return &AwsCloudEngine{
		session: session,
		region:  region,
	}, nil
}

func (engine *AwsCloudEngine) CreateHollowGroup(group *engine.HollowGroupRequest) (string, error) {
	asgSvc := autoscaling.New(engine.session, aws.NewConfig().WithRegion(engine.region))
	result, err := asgSvc.DescribeAutoScalingGroups(&autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: []*string{&group.AutoScalingGroupName},
	})
	if err != nil {
		return "", err
	}
	log.Info(result)

	// attach to asg-monitor (asg-scanner) (simply tag the ASG? or save the launch config to a db?)

	return group.AutoScalingGroupName, nil
}

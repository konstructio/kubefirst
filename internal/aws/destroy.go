package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing"
	"github.com/rs/zerolog/log"
)

// ElbTags describes a pair of tags assigned to an Elastic Load Balancer
type ElbTags struct {
	Key   string
	Value string
}

// ElbDeletionParameters describes an Elastic Load Balancer name and source
// security group to delete
type ElbDeletionParameters struct {
	ElbName                    string
	ElbSourceSecurityGroupName string
}

// GetLoadBalancerForDeletion gets all load balancers and returns details for
// a load balancer associated with the target EKS cluster
func (conf *AWSConfiguration) GetLoadBalancerForDeletion(eksClusterName string) (ElbDeletionParameters, error) {
	elbClient := elasticloadbalancing.NewFromConfig(conf.Config)

	// Get all elastic load balancers
	elasticLoadBalancers, err := elbClient.DescribeLoadBalancers(context.Background(), &elasticloadbalancing.DescribeLoadBalancersInput{})
	if err != nil {
		return ElbDeletionParameters{}, err
	}

	// Build list of Elastic Load Balancer names
	var elasticLoadBalancerNames []string
	for _, lb := range elasticLoadBalancers.LoadBalancerDescriptions {
		elasticLoadBalancerNames = append(elasticLoadBalancerNames, *lb.LoadBalancerName)
	}

	// Get tags for each Elastic Load Balancer and add to map
	elasticLoadBalancerTags := make(map[string][]ElbTags)
	for _, elb := range elasticLoadBalancerNames {
		// Describe tags per Elastic Load Balancer
		tags, err := elbClient.DescribeTags(context.Background(), &elasticloadbalancing.DescribeTagsInput{
			LoadBalancerNames: []string{elb},
		})
		if err != nil {
			return ElbDeletionParameters{}, err
		}

		// Compile tags
		tagsContainer := make([]ElbTags, 0)
		for _, tag := range tags.TagDescriptions {
			for _, desc := range tag.Tags {
				tagsContainer = append(tagsContainer, ElbTags{Key: *desc.Key, Value: *desc.Value})
			}
		}

		// Add to map
		elasticLoadBalancerTags[elb] = tagsContainer
	}

	// Return load balancer name based on associated tags
	var targetElb string
	for key, value := range elasticLoadBalancerTags {
		for _, tag := range value {
			fmt.Println(key, value)
			if tag.Key == fmt.Sprintf("kubernetes.io/cluster/%s", eksClusterName) && tag.Value == "owned" {
				targetElb = key
			}
		}
	}

	// Return error if not found
	var targetSecurityGroup *string
	if targetElb == "" {
		return ElbDeletionParameters{}, fmt.Errorf("no elastic load balancer found using name %s", eksClusterName)
	} else {
		elasticLoadBalancer, err := elbClient.DescribeLoadBalancers(context.Background(), &elasticloadbalancing.DescribeLoadBalancersInput{
			LoadBalancerNames: []string{targetElb},
		})
		if err != nil {
			return ElbDeletionParameters{}, err
		}
		targetSecurityGroup = elasticLoadBalancer.LoadBalancerDescriptions[0].SourceSecurityGroup.GroupName
	}

	// Format ElbDeletionParameters detailing name and source sg for ELB
	source := ElbDeletionParameters{
		ElbName:                    targetElb,
		ElbSourceSecurityGroupName: *targetSecurityGroup,
	}

	return source, nil
}

// DeleteSourceSecurityGroup deletes a source security group associated with an EKS cluster's
// Elastic Load Balancer
func (conf *AWSConfiguration) DeleteSourceSecurityGroup(elbdp ElbDeletionParameters) error {
	ec2Client := ec2.NewFromConfig(conf.Config)

	_, err := ec2Client.DeleteSecurityGroup(context.Background(), &ec2.DeleteSecurityGroupInput{
		GroupName: &elbdp.ElbSourceSecurityGroupName,
	})
	if err != nil {
		return err
	}

	log.Info().Msgf("deleted elastic load balancer source security group %s", elbdp.ElbSourceSecurityGroupName)

	return nil
}

// DeleteElasticLoadBalancer deletes an Elastic Load Balancer associated with an EKS cluster
func (conf *AWSConfiguration) DeleteElasticLoadBalancer(elbdp ElbDeletionParameters) error {
	elbClient := elasticloadbalancing.NewFromConfig(conf.Config)

	_, err := elbClient.DeleteLoadBalancer(context.Background(), &elasticloadbalancing.DeleteLoadBalancerInput{
		LoadBalancerName: &elbdp.ElbName,
	})
	if err != nil {
		return err
	}

	log.Info().Msgf("deleted elastic load balancer %s", elbdp.ElbName)

	return nil
}

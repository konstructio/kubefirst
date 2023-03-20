package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
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
	ElbName                 string
	ElbSourceSecurityGroups []string
}

// GetLoadBalancersForDeletion gets all load balancers and returns details for
// a load balancer associated with the target EKS cluster
func (conf *AWSConfiguration) GetLoadBalancersForDeletion(eksClusterName string) ([]ElbDeletionParameters, error) {
	elbClient := elasticloadbalancing.NewFromConfig(conf.Config)

	// Get all elastic load balancers
	elasticLoadBalancers, err := elbClient.DescribeLoadBalancers(context.Background(), &elasticloadbalancing.DescribeLoadBalancersInput{})
	if err != nil {
		return []ElbDeletionParameters{}, err
	}

	// Build list of Elastic Load Balancer names
	var elasticLoadBalancerNames []string
	for _, lb := range elasticLoadBalancers.LoadBalancerDescriptions {
		elasticLoadBalancerNames = append(elasticLoadBalancerNames, *lb.LoadBalancerName)
	}

	// Get tags for each Elastic Load Balancer
	elasticLoadBalancerTags := make(map[string][]ElbTags)
	for _, elb := range elasticLoadBalancerNames {
		// Describe tags per Elastic Load Balancer
		tags, err := elbClient.DescribeTags(context.Background(), &elasticloadbalancing.DescribeTagsInput{
			LoadBalancerNames: []string{elb},
		})
		if err != nil {
			return []ElbDeletionParameters{}, err
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

	// Return matched load balancers
	elasticLoadBalancersToDelete := []ElbDeletionParameters{}
	for key, value := range elasticLoadBalancerTags {
		for _, tag := range value {
			if tag.Key == fmt.Sprintf("kubernetes.io/cluster/%s", eksClusterName) && tag.Value == "owned" {
				elasticLoadBalancer, err := elbClient.DescribeLoadBalancers(context.Background(), &elasticloadbalancing.DescribeLoadBalancersInput{
					LoadBalancerNames: []string{key},
				})
				if err != nil {
					return []ElbDeletionParameters{}, err
				}
				targetSecurityGroups := elasticLoadBalancer.LoadBalancerDescriptions[0].SecurityGroups
				elasticLoadBalancersToDelete = append(elasticLoadBalancersToDelete, ElbDeletionParameters{
					ElbName:                 key,
					ElbSourceSecurityGroups: targetSecurityGroups,
				})
			}
		}
	}

	return elasticLoadBalancersToDelete, nil
}

// DeleteEKSSecurityGroups deletes security groups associated with an EKS cluster
func (conf *AWSConfiguration) DeleteEKSSecurityGroups(eksClusterName string) error {
	ec2Client := ec2.NewFromConfig(conf.Config, func(o *ec2.Options) {
		o.Region = RegionUsEast1
	})

	// Get dependent security groups
	filterName := "tag-key"
	maxResults := int32(1000)
	dependentSecurityGroups, err := ec2Client.DescribeSecurityGroups(context.Background(), &ec2.DescribeSecurityGroupsInput{
		MaxResults: &maxResults,
		Filters: []ec2Types.Filter{
			{
				Name:   &filterName,
				Values: []string{fmt.Sprintf("kubernetes.io/cluster/%s", eksClusterName)},
			},
		},
	})
	if err != nil {
		return err
	}

	// Delete matched security groups
	for _, sg := range dependentSecurityGroups.SecurityGroups {
		fmt.Printf("preparing to delete eks security group %s / %s", *sg.GroupName, *sg.GroupId)
		_, err = ec2Client.DeleteSecurityGroup(context.Background(), &ec2.DeleteSecurityGroupInput{
			GroupId: sg.GroupId,
		})
		if err != nil {
			return err
		}
		fmt.Printf("deleted security group %s / %s", *sg.GroupName, *sg.GroupId)
	}

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

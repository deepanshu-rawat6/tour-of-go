package scanner

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	cwtypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
)

// elbv2API is the subset of the ELBv2 client used by ALBScanner.
type elbv2API interface {
	DescribeLoadBalancers(ctx context.Context, in *elasticloadbalancingv2.DescribeLoadBalancersInput, opts ...func(*elasticloadbalancingv2.Options)) (*elasticloadbalancingv2.DescribeLoadBalancersOutput, error)
}

// cwAPI is the subset of the CloudWatch client used by ALBScanner.
type cwAPI interface {
	GetMetricStatistics(ctx context.Context, in *cloudwatch.GetMetricStatisticsInput, opts ...func(*cloudwatch.Options)) (*cloudwatch.GetMetricStatisticsOutput, error)
}

// ALBScanner finds ALBs with zero requests in the last 7 days.
type ALBScanner struct{}

func (ALBScanner) Scan(ctx context.Context, cfg aws.Config, region, accountID string) ([]Resource, error) {
	cfg.Region = region
	return scanALBs(ctx, elasticloadbalancingv2.NewFromConfig(cfg), cloudwatch.NewFromConfig(cfg), region, accountID)
}

func scanALBs(ctx context.Context, elbClient elbv2API, cwClient cwAPI, region, accountID string) ([]Resource, error) {
	var resources []Resource
	var marker *string
	for {
		out, err := elbClient.DescribeLoadBalancers(ctx, &elasticloadbalancingv2.DescribeLoadBalancersInput{
			Marker: marker,
		})
		if err != nil {
			return nil, err
		}
		for _, lb := range out.LoadBalancers {
			if lb.Type != "application" {
				continue
			}
			total, err := albRequestCount(ctx, cwClient, aws.ToString(lb.LoadBalancerArn))
			if err != nil {
				return nil, err
			}
			if total > 0 {
				continue
			}
			resources = append(resources, Resource{
				ID:        aws.ToString(lb.LoadBalancerArn),
				Type:      "alb",
				Region:    region,
				AccountID: accountID,
				Metadata:  map[string]string{"dns_name": aws.ToString(lb.DNSName)},
			})
		}
		if out.NextMarker == nil {
			break
		}
		marker = out.NextMarker
	}
	return resources, nil
}

// albRequestCount returns the total RequestCount for an ALB over the last 7 days.
func albRequestCount(ctx context.Context, cw cwAPI, lbARN string) (float64, error) {
	now := time.Now()
	out, err := cw.GetMetricStatistics(ctx, &cloudwatch.GetMetricStatisticsInput{
		Namespace:  aws.String("AWS/ApplicationELB"),
		MetricName: aws.String("RequestCount"),
		Dimensions: []cwtypes.Dimension{
			{Name: aws.String("LoadBalancer"), Value: aws.String(albDimension(lbARN))},
		},
		StartTime:  aws.Time(now.AddDate(0, 0, -7)),
		EndTime:    aws.Time(now),
		Period:     aws.Int32(604800), // 7 days in seconds
		Statistics: []cwtypes.Statistic{cwtypes.StatisticSum},
	})
	if err != nil {
		return 0, err
	}
	var total float64
	for _, dp := range out.Datapoints {
		if dp.Sum != nil {
			total += *dp.Sum
		}
	}
	return total, nil
}

// albDimension extracts the CloudWatch dimension value from an ALB ARN.
// ARN format: arn:aws:elasticloadbalancing:region:account:loadbalancer/app/name/id
// Dimension:  app/name/id
func albDimension(arn string) string {
	const prefix = "loadbalancer/"
	for i := 0; i < len(arn)-len(prefix); i++ {
		if arn[i:i+len(prefix)] == prefix {
			return arn[i+len(prefix):]
		}
	}
	return arn
}

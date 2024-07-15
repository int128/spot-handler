package spot

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	spothandlerv1 "github.com/int128/spot-handler/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const DetailTypeEC2SpotInstanceInterruptionWarning = "EC2 Spot Instance Interruption Warning"

type rawMessage struct {
	// https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/spot-instance-termination-notices.html
	Time       time.Time        `json:"time,omitempty"`
	Resources  []string         `json:"resources,omitempty"`
	DetailType string           `json:"detail-type,omitempty"`
	Detail     rawMessageDetail `json:"detail,omitempty"`
}

type rawMessageDetail struct {
	InstanceID     string `json:"instance-id,omitempty"`
	InstanceAction string `json:"instance-action,omitempty"`
}

func Parse(body string) (*spothandlerv1.EC2SpotInstanceInterruptionWarningSpec, error) {
	var msg rawMessage
	if err := json.NewDecoder(strings.NewReader(body)).Decode(&msg); err != nil {
		return nil, fmt.Errorf("invalid json: %w", err)
	}
	if msg.DetailType != DetailTypeEC2SpotInstanceInterruptionWarning {
		return nil, fmt.Errorf("detail-type %s is not supported", msg.DetailType)
	}
	if len(msg.Resources) != 1 {
		return nil, fmt.Errorf("len(resources) must be 1 but was %d", len(msg.Resources))
	}
	availabilityZone, err := parseAvailabilityZone(msg.Resources[0])
	if err != nil {
		return nil, fmt.Errorf("could not parse the availability zone: %w", err)
	}

	return &spothandlerv1.EC2SpotInstanceInterruptionWarningSpec{
		EventTime:        metav1.NewTime(msg.Time),
		InstanceID:       msg.Detail.InstanceID,
		AvailabilityZone: availabilityZone,
	}, nil
}

func parseAvailabilityZone(resource string) (string, error) {
	// The ARN format is `arn:aws:ec2:availability-zone:instance/instance-id`
	parts := strings.SplitN(resource, ":", 5)
	if len(parts) != 5 {
		return "", fmt.Errorf("invalid resource: %s", resource)
	}
	return parts[3], nil
}

package spot

import (
	"encoding/json"
	"fmt"
	"strings"
)

const DetailTypeEC2SpotInstanceInterruptionWarning = "EC2 Spot Instance Interruption Warning"

// Notice represents an EC2 Spot Instance interruption notice.
//
// See https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/spot-instance-termination-notices.html
type Notice struct {
	Resources  []string     `json:"resources,omitempty"`
	DetailType string       `json:"detail-type,omitempty"`
	Detail     NoticeDetail `json:"detail,omitempty"`

	// AvailabilityZone is parsed from Resources[0]
	AvailabilityZone string `json:"-"`
}

type NoticeDetail struct {
	InstanceID     string `json:"instance-id,omitempty"`
	InstanceAction string `json:"instance-action,omitempty"`
}

func Parse(body string) (Notice, error) {
	var notice Notice
	if err := json.NewDecoder(strings.NewReader(body)).Decode(&notice); err != nil {
		return notice, fmt.Errorf("invalid json: %w", err)
	}
	if len(notice.Resources) != 1 {
		return notice, fmt.Errorf("len(resources) must be 1 but was %d", len(notice.Resources))
	}
	availabilityZone, err := parseAvailabilityZone(notice.Resources[0])
	if err != nil {
		return notice, fmt.Errorf("could not parse the availability zone: %w", err)
	}
	notice.AvailabilityZone = availabilityZone
	return notice, nil
}

func parseAvailabilityZone(resource string) (string, error) {
	// The ARN format is `arn:aws:ec2:availability-zone:instance/instance-id`
	parts := strings.SplitN(resource, ":", 5)
	if len(parts) != 5 {
		return "", fmt.Errorf("invalid resource: %s", resource)
	}
	return parts[3], nil
}

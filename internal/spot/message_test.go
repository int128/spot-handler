package spot

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParse(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/spot-instance-termination-notices.html
		body := `{
    "version": "0",
    "id": "12345678-1234-1234-1234-123456789012",
    "detail-type": "EC2 Spot Instance Interruption Warning",
    "source": "aws.ec2",
    "account": "123456789012",
    "time": "yyyy-mm-ddThh:mm:ssZ",
    "region": "us-east-2",
    "resources": ["arn:aws:ec2:us-east-2a:instance/i-1234567890abcdef0"],
    "detail": {
        "instance-id": "i-1234567890abcdef0",
        "instance-action": "action"
    }
}`
		got, err := Parse(body)
		if err != nil {
			t.Fatalf("Parse() error: %s", err)
		}
		want := Notice{
			Resources:  []string{"arn:aws:ec2:us-east-2a:instance/i-1234567890abcdef0"},
			DetailType: DetailTypeEC2SpotInstanceInterruptionWarning,
			Detail: NoticeDetail{
				InstanceID:     "i-1234567890abcdef0",
				InstanceAction: "action",
			},
			AvailabilityZone: "us-east-2a",
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("Parse() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		_, err := Parse(`invalid`)
		if err == nil {
			t.Fatalf("Parse() wants error but was nil")
		}
	})
}

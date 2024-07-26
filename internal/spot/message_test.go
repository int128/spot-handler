package spot

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	spothandlerv1 "github.com/int128/spot-handler/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
    "time": "2021-02-03T14:05:06Z",
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
		want := &spothandlerv1.SpotInterruptionSpec{
			EventTimestamp:   metav1.Date(2021, 2, 3, 14, 5, 6, 0, time.UTC),
			InstanceID:       "i-1234567890abcdef0",
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

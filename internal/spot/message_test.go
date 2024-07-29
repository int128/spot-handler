package spot

import (
	_ "embed"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	spothandlerv1 "github.com/int128/spot-handler/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// An example message of spot interruption.
// https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/spot-instance-termination-notices.html
//
//go:embed testdata/message.json
var messageJSON string

func TestParse(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		got, err := Parse(messageJSON)
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

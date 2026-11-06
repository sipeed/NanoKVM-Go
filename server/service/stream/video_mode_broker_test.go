package stream

import "testing"

func TestVideoModeBrokerCoalescesAndUnsubscribes(t *testing.T) {
	t.Parallel()

	broker := NewVideoModeBroker()
	subscription := broker.Subscribe()

	broker.Publish()
	broker.Publish()

	select {
	case <-subscription.C:
	default:
		t.Fatal("expected a mode-change notification")
	}

	select {
	case <-subscription.C:
		t.Fatal("expected burst notifications to coalesce")
	default:
	}

	subscription.Close()
	broker.Publish()

	select {
	case _, ok := <-subscription.C:
		if ok {
			t.Fatal("received a notification after unsubscribe")
		}
	default:
		t.Fatal("expected unsubscribe to close the subscription")
	}
}

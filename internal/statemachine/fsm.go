package statemachine

import "fmt"

const (
	StatePendingPayment = "PENDING_PAYMENT"
	StatePaid           = "PAID"
	StateDelivered      = "DELIVERED"
	StateCompleted      = "COMPLETED"
	StateCancelled      = "CANCELLED"
	StateRefunding      = "REFUNDING"
	StateRefunded       = "REFUNDED"
)

const (
	EventPaymentCaptured  = "PAYMENT_CAPTURED"
	EventPaymentFailed    = "PAYMENT_FAILED"
	EventTimeoutClose     = "TIMEOUT_CLOSE"
	EventDigitalDelivered = "DIGITAL_DELIVERED"
	EventOrderCompleted   = "ORDER_COMPLETED"
	EventRefundApproved   = "REFUND_APPROVED"
	EventRefundCompleted  = "REFUND_COMPLETED"
)

var transitions = map[string]map[string]string{
	StatePendingPayment: {
		EventPaymentCaptured: StatePaid,
		EventPaymentFailed:   StateCancelled,
		EventTimeoutClose:    StateCancelled,
	},
	StatePaid: {
		EventDigitalDelivered: StateDelivered,
		EventRefundApproved:   StateRefunding,
	},
	StateDelivered: {
		EventOrderCompleted: StateCompleted,
		EventRefundApproved: StateRefunding,
	},
	StateCompleted: {
		EventRefundApproved: StateRefunding,
	},
	StateRefunding: {
		EventRefundCompleted: StateRefunded,
	},
}

func Transition(from, event string) (string, error) {
	events, ok := transitions[from]
	if !ok {
		return "", fmt.Errorf("unknown state: %s", from)
	}
	to, ok := events[event]
	if !ok {
		return "", fmt.Errorf("invalid transition: state=%s event=%s", from, event)
	}
	return to, nil
}

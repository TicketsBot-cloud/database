package database

type TicketSource int16

const (
	TicketSourceUnknown TicketSource = 0
	TicketSourcePanel   TicketSource = 1
	TicketSourceCommand TicketSource = 2
)

func (s TicketSource) String() string {
	switch s {
	case TicketSourcePanel:
		return "panel"
	case TicketSourceCommand:
		return "command"
	default:
		return "unknown"
	}
}

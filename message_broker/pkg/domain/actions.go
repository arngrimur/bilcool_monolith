package domain

import "context"

type ActionName int

const (
	ActionBegin ActionName = iota
	ActionCommit
	ActionInsert
	ActionUpdate
	ActionDelete
	ActionTruncate
	ActionTypeMessage
	ActionOriginMessage
	ActionLogicalDecodingMessage
	ActionStreamStartMessage
	ActionStreamStopMessage
	ActionStreamCommitMessage
)

//go:generate mockgen -source=actions.go -destination=actions_mock.go -package=domain github.com/arngrimur/bilcool_monolith/message_broker
type Action interface {
	// TODO: Add name to action for logging
	Execute(ctx context.Context, table Table) error
}

type Actions struct {
	actions map[ActionName][]Action
}

func NewActions() *Actions {
	return &Actions{
		actions: make(map[ActionName][]Action),
	}
}

func (a *Actions) RegisterAction(name ActionName, action Action) {
	a.actions[name] = append(a.actions[name], action)
}

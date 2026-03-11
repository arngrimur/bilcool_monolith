package domain

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

//go:generate mockgen -source=actions.go -destination=actions_mock.go -package=outbox github.com/arngrimur/bilcool_monolith/outbox Actions
type Action interface {
	Execute(table Table)
}

type Actions struct {
	actions map[ActionName][]Action
}

func NewActions() *Actions {
	return &Actions{
		actions: make(map[ActionName][]Action),
	}
}

func (a *Actions) Add(name ActionName, action Action) {
	a.actions[name] = append(a.actions[name], action)
}

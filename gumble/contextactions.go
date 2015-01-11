package gumble

// ContextActions is a map of ContextActions.
type ContextActions map[string]*ContextAction

func (ca ContextActions) create(action string) *ContextAction {
	contextAction := &ContextAction{
		name: action,
	}
	ca[action] = contextAction
	return contextAction
}

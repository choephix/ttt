package command

type Command struct {
	ID      string
	Title   string
	Handler func()
}

type Registry struct {
	commands map[string]Command
}

func NewRegistry() *Registry {
	return &Registry{commands: make(map[string]Command)}
}

func (r *Registry) Register(cmd Command) {
	r.commands[cmd.ID] = cmd
}

func (r *Registry) Execute(id string) bool {
	cmd, ok := r.commands[id]
	if !ok {
		return false
	}
	cmd.Handler()
	return true
}

func (r *Registry) Get(id string) (Command, bool) {
	cmd, ok := r.commands[id]
	return cmd, ok
}

func (r *Registry) List() []Command {
	cmds := make([]Command, 0, len(r.commands))
	for _, cmd := range r.commands {
		cmds = append(cmds, cmd)
	}
	return cmds
}

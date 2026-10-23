package cmd

import "sync"

var commands sync.Map

func Register(command Command) {
	commands.Store(command.name, command)
	for _, alias := range command.aliases {
		commands.Store(alias, command)
	}
}

func ByAlias(alias string) (Command, bool) {
	command, ok := commands.Load(alias)
	if !ok {
		return Command{}, false
	}
	return command.(Command), ok
}

func Commands() map[string]Command {
	cmd := make(map[string]Command)
	commands.Range(func(key, value any) bool {
		cmd[key.(string)] = value.(Command)
		return true
	})
	return cmd
}

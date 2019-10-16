package commands

type Command interface {
	Name() string
	Usage() string
	Handle(args []string) error
}
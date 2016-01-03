package main

import (
	"github.com/mitchellh/cli"
	"github.com/pims/pt/command"
	"os"
	"os/signal"
)

// Commands is the mapping of all the available pt commands.
var Commands map[string]cli.CommandFactory

var (
	UiColorBlack = cli.UiColor{37, false}
)

func init() {
	cui := cli.ColoredUi{
		InfoColor:  cli.UiColorGreen,
		ErrorColor: cli.UiColorRed,
		WarnColor:  cli.UiColorYellow,
		Ui: &cli.BasicUi{
			Writer: os.Stdout,
			Reader: os.Stdin,
		},
	}

	Commands = map[string]cli.CommandFactory{

		"search": func() (cli.Command, error) {
			return &command.SearchCommand{
				Ui: cui,
			}, nil
		},
	}
}

// makeShutdownCh returns a channel that can be used for shutdown
// notifications for commands. This channel will send a message for every
// interrupt received.
func makeShutdownCh() <-chan struct{} {
	resultCh := make(chan struct{})

	signalCh := make(chan os.Signal, 4)
	signal.Notify(signalCh, os.Interrupt)
	go func() {
		for {
			<-signalCh
			resultCh <- struct{}{}
		}
	}()

	return resultCh
}

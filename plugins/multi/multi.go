package multi

import (
	"github.com/turbinelabs/cli/command"
	tbnflag "github.com/turbinelabs/nonstdlib/flag"
	"github.com/turbinelabs/rotor"
	"github.com/turbinelabs/rotor/pkg/cluster_provider/multi"
	"github.com/turbinelabs/rotor/updater"
)


const defaultConfigFileLocation = "/etc/rotor-multi.json"

type multiRunner struct {
	configFileLocation string  // file location where multi runner config is present
	updaterFlags rotor.UpdaterFromFlags
}

func MultiCMD(updaterFlags rotor.UpdaterFromFlags) *command.Cmd {
	runner := &multiRunner{}
	cmd := &command.Cmd{
		Name:        "multi",
		Summary:     "Multi Collector from different sources",
		Usage:       "[OPTIONS]",
		Description: multiDescription,
		Runner:      runner,
	}

	flags := tbnflag.Wrap(&cmd.Flags)
	flags.StringVar(
		&runner.configFileLocation,
		"config-file",
		defaultConfigFileLocation,
		"config file location indicating where to read the config file for the multi mode in rotor",
	)

	runner.updaterFlags = updaterFlags
	return cmd
}

type multiComponent struct {
	Name string `json:"name"`
	Config map[string]string `json:"config"`  // change from map[string]string to struct for each using oneof
}

func (r multiRunner) Run(cmd *command.Cmd, args []string) command.CmdErr {
	if err := r.updaterFlags.Validate(); err != nil {
		return cmd.BadInput(err)
	}

	u, err := r.updaterFlags.Make()
	if err != nil {
		return cmd.Error(err)
	}

	m, err := multi.NewMultiClustersProvider(multi.ClustersProviderConfig{
		ConfigFileLocation:r.configFileLocation,
	})
	if err != nil {
		return cmd.Error(err)
	}
	updater.Loop(u, m.GetClusters)

	return command.NoError()
}


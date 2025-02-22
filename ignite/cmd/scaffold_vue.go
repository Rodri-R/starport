package ignitecmd

import (
	"github.com/spf13/cobra"

	"github.com/ignite/cli/ignite/chainconfig"
	"github.com/ignite/cli/ignite/pkg/cliui"
	"github.com/ignite/cli/ignite/pkg/cosmosgen"
)

// NewScaffoldVue scaffolds a Vue.js app for a chain.
func NewScaffoldVue() *cobra.Command {
	c := &cobra.Command{
		Use:     "vue",
		Short:   "Generate Vue 3 web app template",
		Args:    cobra.NoArgs,
		PreRunE: gitChangesConfirmPreRunHandler,
		RunE:    scaffoldVueHandler,
	}

	c.Flags().AddFlagSet(flagSetYes())
	c.Flags().StringP(flagPath, "p", "./"+chainconfig.DefaultVuePath, "path to scaffold content of the Vue.js app")

	return c
}

func scaffoldVueHandler(cmd *cobra.Command, args []string) error {
	session := cliui.New(cliui.StartSpinnerWithText(statusScaffolding))
	defer session.End()

	path := flagGetPath(cmd)
	if err := cosmosgen.Vue(path); err != nil {
		return err
	}

	return session.Printf("\n🎉 Scaffolded a Vue.js app in %s.\n\n", path)
}

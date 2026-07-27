package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/plexusone/omniavatar"
	"github.com/plexusone/omniavatar-core/render"
	_ "github.com/plexusone/omniavatar/providers/all"
)

var avatarListCmd = &cobra.Command{
	Use:   "list-avatars",
	Short: "List avatar IDs available from a provider",
	Long: `List avatar IDs usable as --avatar-id for 'vac avatar generate' and
'vac slides video --avatar-id'.

Provider avatar catalogs can be large, so use --search to filter by ID or
name (case-insensitive). API keys are read from the provider's env var
(e.g. HEYGEN_API_KEY) or --api-key.

Examples:
  vac avatar list-avatars --provider heygen --search abigail
  vac avatar list-avatars --provider heygen --limit 0 > avatars.txt`,
	RunE: runAvatarList,
}

var (
	alProvider string
	alAPIKey   string
	alSearch   string
	alLimit    int
)

func init() {
	avatarListCmd.Flags().StringVarP(&alProvider, "provider", "p", "heygen", "Avatar provider: heygen, tavus, or bithuman")
	avatarListCmd.Flags().StringVarP(&alAPIKey, "api-key", "k", "", "Provider API key (or use the provider's env var)")
	avatarListCmd.Flags().StringVar(&alSearch, "search", "", "Filter avatars by ID or name substring (case-insensitive)")
	avatarListCmd.Flags().IntVar(&alLimit, "limit", 50, "Maximum avatars to print (0 = all)")

	avatarCmd.AddCommand(avatarListCmd)
}

func runAvatarList(cmd *cobra.Command, args []string) error {
	ctx := newContext()

	envVar, ok := providerAPIKeyEnvs[alProvider]
	if !ok {
		return fmt.Errorf("unknown provider %q (available: heygen, tavus, bithuman, liveportrait-joyvasa)", alProvider)
	}
	apiKey := alAPIKey
	// Local providers don't require an API key
	if !localProviders[alProvider] {
		if apiKey == "" {
			apiKey = os.Getenv(envVar)
		}
		if apiKey == "" {
			return fmt.Errorf("%s API key required: use --api-key flag or %s env var", alProvider, envVar)
		}
	}

	provider, err := omniavatar.GetRenderProvider(alProvider, omniavatar.WithAPIKey(apiKey))
	if err != nil {
		return err
	}

	lister, ok := provider.(render.AvatarLister)
	if !ok {
		return fmt.Errorf("provider %q does not support listing avatars", alProvider)
	}

	fmt.Fprintf(os.Stderr, "Fetching avatars from %s (this can take a moment)...\n", alProvider)
	avatars, err := lister.ListAvatars(ctx, alSearch)
	if err != nil {
		return err
	}
	if len(avatars) == 0 {
		fmt.Fprintln(os.Stderr, "No avatars found.")
		return nil
	}

	shown := len(avatars)
	if alLimit > 0 && shown > alLimit {
		shown = alLimit
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	for _, a := range avatars[:shown] {
		name := a.Name
		if a.Gender != "" {
			name = fmt.Sprintf("%s (%s)", a.Name, a.Gender)
		}
		fmt.Fprintf(tw, "%s\t%s\n", a.ID, name)
	}
	if err := tw.Flush(); err != nil {
		return err
	}

	if len(avatars) > shown {
		fmt.Fprintf(os.Stderr, "\n... and %d more; narrow with --search or use --limit 0.\n", len(avatars)-shown)
	}
	return nil
}

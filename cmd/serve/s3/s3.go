package s3

import (
	"context"
	"fmt"
	"strings"

	"github.com/rclone/rclone/cmd"
	"github.com/rclone/rclone/cmd/serve/proxy/proxyflags"
	"github.com/rclone/rclone/fs/config/flags"
	"github.com/rclone/rclone/fs/hash"
	libhttp "github.com/rclone/rclone/lib/http"
	"github.com/rclone/rclone/vfs"
	"github.com/rclone/rclone/vfs/vfsflags"
	"github.com/spf13/cobra"
)

// DefaultOpt is the default values used for Options
var DefaultOpt = Options{
	Auth: libhttp.DefaultAuthCfg(),
	HTTP: libhttp.DefaultCfg(),

	pathBucketMode: true,
	hashName:       "MD5",
	hashType:       hash.MD5,

	noCleanup: false,
}

// Opt is options set by command line flags
var Opt = DefaultOpt

// flagPrefix is the prefix used to uniquely identify command line flags.
// It is intentionally empty for this package.
const flagPrefix = ""

func init() {
	flagSet := Command.Flags()
	libhttp.AddAuthFlagsPrefix(flagSet, flagPrefix, &Opt.Auth)
	libhttp.AddHTTPFlagsPrefix(flagSet, flagPrefix, &Opt.HTTP)

	vfsflags.AddFlags(flagSet)
	proxyflags.AddFlags(flagSet)

	flags.BoolVarP(flagSet, &Opt.pathBucketMode, "force-path-style", "", Opt.pathBucketMode, "If true use path style access if false use virtual hosted style (default true)")
	flags.StringVarP(flagSet, &Opt.hashName, "etag-hash", "", Opt.hashName, "Which hash to use for the ETag, or auto or blank for off")
	flags.StringArrayVarP(flagSet, &Opt.authPair, "s3-authkey", "", Opt.authPair, "Set key pair for v4 authorization, split by comma")
	flags.BoolVarP(flagSet, &Opt.noCleanup, "no-cleanup", "", Opt.noCleanup, "Not to cleanup empty folder after object is deleted")
}

// Command definition for cobra
var Command = &cobra.Command{
	Use:   "s3 remote:path",
	Short: `Serve remote:path over s3.`,
	Long:  strings.ReplaceAll(longHelp, "|", "`") + vfs.Help,
	RunE: func(command *cobra.Command, args []string) error {
		cmd.CheckArgs(1, 1, command, args)
		f := cmd.NewFsSrc(args)

		if Opt.hashName == "auto" {
			Opt.hashType = f.Hashes().GetOne()
		} else if Opt.hashName != "" {
			err := Opt.hashType.Set(Opt.hashName)
			if err != nil {
				return err
			}
		}
		cmd.Run(false, false, command, func() error {
			s := newServer(context.Background(), f, &Opt)

			server, err := libhttp.NewServer(context.Background(),
				libhttp.WithConfig(Opt.HTTP),
				libhttp.WithAuth(Opt.Auth),
			)

			if err != nil {
				return fmt.Errorf("failed to init server: %w", err)
			}

			router := server.Router()

			s.Bind(router)
			server.Serve()
			server.Wait()
			return nil
		})
		return nil
	},
}

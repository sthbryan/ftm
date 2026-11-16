package cli

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"slices"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/sthbryan/ftm/internal/core"
	"github.com/sthbryan/ftm/internal/i18n"
	"github.com/sthbryan/ftm/internal/version"
)

func newServeCmd(d *Deps) *cobra.Command {
	var openBrowser bool
	var port int
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Run ftm in the background with only the web view",
		Long: "Run the tunnel manager without the terminal view and serve the web view. When " +
			"ftm is already running, print where its web view is.",
		Example: "  ftm serve --open\n  ftm serve --port 40500",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			s, err := d.connect(true)
			if err != nil {
				return err
			}
			defer s.Close()

			if s.core == nil {
				if err := println(cmd, i18n.TF("already_running", s.client.URL())); err != nil {
					return err
				}
				if openBrowser {
					return d.OpenURL(s.client.URL())
				}
				return nil
			}

			url, err := s.core.ServeWeb(port)
			if err != nil {
				return err
			}
			_ = println(cmd, i18n.TF("version_output", version.Version))
			_ = println(cmd, i18n.TF("dashboard_url", url))
			_ = println(cmd, strings.TrimSpace(i18n.T("press_ctrl_c")))
			if openBrowser {
				if err := d.OpenURL(url); err != nil {
					printErr(cmd, i18n.TF("cli_open_failed", err))
				}
			}
			return wait(cmd.Context(), s.core)
		},
	}
	cmd.Flags().BoolVar(&openBrowser, "open", false, "Open the web view in the browser")
	cmd.Flags().IntVar(&port, "port", 0, "Port for the web view (picked for you when left out)")
	return cmd
}

func newSelfUpdateCmd(d *Deps) *cobra.Command {
	var check bool
	cmd := &cobra.Command{
		Use:     "self-update",
		Aliases: []string{"update"},
		Short:   "Update ftm to the latest release",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return Update(cmd.OutOrStdout(), check)
		},
	}
	cmd.Flags().BoolVar(&check, "check", false, "Only check whether a new version is out")
	return cmd
}

func newUninstallCmd(d *Deps) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Remove ftm from this computer",
		Long:  "Remove the ftm program. Your saved connections and settings stay.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !yes {
				ok, err := confirm(cmd, d, i18n.T("cli_confirm_uninstall"))
				if err != nil || !ok {
					return err
				}
			}
			return Uninstall(cmd.OutOrStdout())
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Remove without asking")
	return cmd
}

// Config keys `ftm config` reads and writes.
const (
	keyLanguage      = "language"
	keyNotifications = "notifications"
	keySound         = "notification_sound"
	keyAutostart     = "autostart"
)

var configKeys = []string{keyLanguage, keyNotifications, keySound, keyAutostart}

func newConfigCmd(d *Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Read and change ftm's settings",
		Long: "Read and change ftm's settings. Keys: " + strings.Join(configKeys, ", ") + ".\n" +
			"Without a subcommand, prints every setting.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error { return configGet(cmd, d, "") },
	}

	path := &cobra.Command{
		Use:   "path",
		Short: "Print where config.yaml is",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return println(cmd, d.Dirs.ConfigFile)
		},
	}
	get := &cobra.Command{
		Use:       "get [key]",
		Short:     "Print one setting, or all of them",
		Args:      cobra.MaximumNArgs(1),
		ValidArgs: configKeys,
		RunE: func(cmd *cobra.Command, args []string) error {
			key := ""
			if len(args) == 1 {
				key = args[0]
			}
			return configGet(cmd, d, key)
		},
	}
	set := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Change a setting",
		Example: "  ftm config set language es\n  ftm config set notification_sound off\n" +
			"  ftm config set autostart on",
		Args:      cobra.ExactArgs(2),
		ValidArgs: configKeys,
		RunE: func(cmd *cobra.Command, args []string) error {
			patch, err := settingsPatch(args[0], args[1])
			if err != nil {
				return err
			}
			s, err := d.connect(false)
			if err != nil {
				return err
			}
			defer s.Close()
			settings, err := s.UpdateSettings(cmd.Context(), patch)
			if err != nil {
				return err
			}
			return println(cmd, fmt.Sprintf("%s = %s", args[0], settingValue(settings, args[0])))
		},
	}
	edit := &cobra.Command{
		Use:   "edit",
		Short: "Open config.yaml in your editor",
		Long:  "Open config.yaml in $VISUAL or $EDITOR. ftm must be closed, or it would overwrite your edit.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if d.Attach != nil {
				if c, err := d.Attach(); err == nil && c != nil {
					return Tr("cli_config_edit_running")
				}
			}
			if _, err := os.Stat(d.Dirs.ConfigFile); errors.Is(err, os.ErrNotExist) {
				// Running the core once writes the default config.
				s, err := d.connect(false)
				if err != nil {
					return err
				}
				s.Close()
			}
			editor := editorCommand(os.Getenv("VISUAL"), os.Getenv("EDITOR"), runtime.GOOS)
			args := append(strings.Fields(editor), d.Dirs.ConfigFile)
			c := exec.CommandContext(cmd.Context(), args[0], args[1:]...)
			c.Stdin, c.Stdout, c.Stderr = os.Stdin, cmd.OutOrStdout(), cmd.ErrOrStderr()
			return c.Run()
		},
	}
	cmd.AddCommand(path, get, set, edit)
	return cmd
}

// editorCommand picks the editor for `ftm config edit`.
func editorCommand(visual, editor, goos string) string {
	switch {
	case strings.TrimSpace(visual) != "":
		return visual
	case strings.TrimSpace(editor) != "":
		return editor
	case goos == "windows":
		return "notepad"
	}
	return "vi"
}

func configGet(cmd *cobra.Command, d *Deps, key string) error {
	if key != "" && !slices.Contains(configKeys, key) {
		return errors.New(i18n.TF("cli_config_unknown_key", key, strings.Join(configKeys, ", ")))
	}
	s, err := d.connect(false)
	if err != nil {
		return err
	}
	defer s.Close()
	settings, err := s.Settings(cmd.Context())
	if err != nil {
		return err
	}
	if key != "" {
		return println(cmd, settingValue(settings, key))
	}
	for _, k := range configKeys {
		if err := println(cmd, fmt.Sprintf("%s = %s", k, settingValue(settings, k))); err != nil {
			return err
		}
	}
	return nil
}

func settingValue(s Settings, key string) string {
	switch key {
	case keyLanguage:
		return s.Language
	case keyNotifications:
		return s.NotificationsEnabled
	case keySound:
		return onOff(s.NotificationSound)
	case keyAutostart:
		if !s.AutostartSupported {
			return i18n.T("cli_autostart_desktop_only")
		}
		return onOff(s.AutostartEnabled)
	}
	return ""
}

func onOff(b bool) string {
	if b {
		return "on"
	}
	return "off"
}

func parseOnOff(raw string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "on", "true", "yes", "1":
		return true, nil
	case "off", "false", "no", "0":
		return false, nil
	}
	if b, err := strconv.ParseBool(raw); err == nil {
		return b, nil
	}
	return false, errors.New(i18n.TF("cli_config_bad_bool", raw))
}

// settingsPatch turns `config set key value` into a settings change.
func settingsPatch(key, value string) (SettingsPatch, error) {
	var p SettingsPatch
	switch key {
	case keyLanguage:
		lang := strings.ToLower(strings.TrimSpace(value))
		allowed := append([]string{core.LanguageAuto}, i18n.AvailableLanguages()...)
		if !slices.Contains(allowed, lang) {
			return p, errors.New(i18n.TF("cli_config_bad_value", key, strings.Join(allowed, ", ")))
		}
		p.Language = &lang
	case keyNotifications:
		status := strings.ToLower(strings.TrimSpace(value))
		allowed := []string{core.NotificationGranted, core.NotificationRejected, core.NotificationPending}
		if b, err := parseOnOff(status); err == nil {
			status = core.NotificationRejected
			if b {
				status = core.NotificationGranted
			}
		}
		if !slices.Contains(allowed, status) {
			return p, errors.New(i18n.TF("cli_config_bad_value", key, "on, off"))
		}
		p.NotificationsEnabled = &status
	case keySound:
		b, err := parseOnOff(value)
		if err != nil {
			return p, err
		}
		p.NotificationSound = &b
	case keyAutostart:
		b, err := parseOnOff(value)
		if err != nil {
			return p, err
		}
		p.Autostart = &b
	default:
		return p, errors.New(i18n.TF("cli_config_unknown_key", key, strings.Join(configKeys, ", ")))
	}
	return p, nil
}

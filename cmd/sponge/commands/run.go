package commands

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os/exec"
	"runtime"
	"strconv"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/go-dev-frame/sponge/cmd/sponge/server"
)

// OpenUICommand run the sponge ui service
func OpenUICommand() *cobra.Command {
	var (
		port       int
		spongeAddr string
		isLog      bool
	)

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run code generation engine service, generate code in UI interface",
		Long:  "Run code generation engine service, generate code in UI interface.",
		Example: color.HiBlackString(`  # Running ui service, local browser access only.
  sponge run

  # Running ui service, can be accessed from other host browsers.
  sponge run -a http://your-host-ip:24631`),
		SilenceErrors: true,
		SilenceUsage:  true,

		RunE: func(cmd *cobra.Command, args []string) error {
			if spongeAddr == "" {
				spongeAddr = fmt.Sprintf("http://localhost:%d", port)
			} else {
				if err := checkSpongeAddr(spongeAddr, port); err != nil {
					return err
				}
			}
			fmt.Printf("Code generation engine service running %s. Access %s in your browser.\n",
				getVersion(), color.HiCyanString(spongeAddr))
			go func() {
				_ = open(spongeAddr)
			}()
			server.RunHTTPServer(spongeAddr, port, isLog)
			return nil
		},
	}
	cmd.Flags().IntVarP(&port, "port", "p", 24631, "port on which the sponge service listens")
	cmd.Flags().StringVarP(&spongeAddr, "addr", "a", "", "address of the front-end page requesting the sponge service, e.g. http://192.168.1.10:24631 or https://your-domain.com")
	cmd.Flags().BoolVarP(&isLog, "log", "l", false, "enable service logging")
	return cmd
}

func open(visitURL string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
	}

	args = append(args, visitURL)
	return exec.Command(cmd, args...).Start()
}

func checkSpongeAddr(spongeAddr string, port int) error {
	paramErr := errors.New("the addr parameter is invalid,  e.g. sponge run --addr=http://192.168.1.10:24631")
	u, err := url.Parse(spongeAddr)
	if err != nil {
		return paramErr
	}

	if (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return paramErr
	}

	ip := net.ParseIP(u.Hostname())
	if ip != nil {
		if u.Port() != strconv.Itoa(port) {
			return errors.New("the port parameter is invalid, e.g. sponge run --port=8080 --addr=http://192.168.1.10:8080")
		}
	}

	return nil
}

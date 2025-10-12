package grpc

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/go-dev-frame/sponge/cmd/sponge/commands/perftest/common"
	"github.com/go-dev-frame/sponge/pkg/gobash"
	"github.com/go-dev-frame/sponge/pkg/gofile"
)

// PerfTestGRPCCMD creates a cobra command for gRPC performance test
func PerfTestGRPCCMD() *cobra.Command {
	var (
		proto []string
		dir   string
		out   string
	)

	cmd := &cobra.Command{
		Use:   "grpc",
		Short: "Run a performance test against gRPC service",
		Long: `For gRPC services created with Sponge, performance test code is included by default.
Simply fill in the parameters in the Test_service_xxx_benchmark function located in
internal/service/xxx_client_test.go to run the performance test.

For gRPC services created by other means, generate the gRPC code from the proto file first,
then fill in the parameters in the same Test_service_xxx_benchmark function under
internal/service/xxx_client_test.go to execute the performance test.
`,
		Example: color.HiBlackString(`  # Generate gRPC code from proto file
  %s grpc --proto=/path/to/proto

  # Generate gRPC code from proto directory
  %s grpc --dir=/path/to/proto-dir`,
			common.CommandPrefix, common.CommandPrefix),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(proto) == 0 && dir == "" {
				return cmd.Usage()
			}
			protoFiles := proto
			if dir != "" {
				files, err := gofile.ListFiles(dir, gofile.WithSuffix(".proto"))
				if err != nil {
					return err
				}
				protoFiles = append(protoFiles, files...)
			}
			if len(protoFiles) == 0 {
				return errors.New("no proto file found")
			}

			tempDir := "./protoFilesTemp"
			_ = os.MkdirAll(tempDir, 0755)
			for _, file := range protoFiles {
				data, err := os.ReadFile(file)
				if err != nil {
					return err
				}
				filename := gofile.GetFilename(file)
				_ = os.WriteFile(tempDir+"/"+filename, data, 0644)
			}
			defer os.RemoveAll(tempDir)

			mspName := "perftest_grpc"
			if out == "" {
				out = mspName
			}
			if gofile.IsExists(out) {
				out += "_" + time.Now().Format("0405")
			}
			cmdArgs := []string{
				"micro",
				"rpc-pb",
				"--module-name=" + mspName,
				"--server-name=" + mspName,
				"--project-name=" + mspName,
				"--protobuf-file=" + tempDir + "/*.proto",
				"--out=" + out,
			}

			result, err := gobash.Exec("sponge", cmdArgs...)
			if err != nil {
				return err
			}
			fmt.Println(string(result))

			fmt.Printf("generating gRPC test code...\n\n")

			cmdStr := "cd " + out + "&& make proto"
			result, err = gobash.Exec("bash", "-c", cmdStr)
			if err != nil {
				return err
			}
			fmt.Println(string(result))

			fmt.Printf("Update the host and port fields under grpcClient in config/xxx.yml, then fill in the parameters" +
				"in the Test_service_xxx_benchmark function in internal/service/xxx_client_test.go to run the performance test.\n\n")

			return nil
		},
	}

	cmd.Flags().StringArrayVarP(&proto, "proto", "f", nil, "path to proto file(s)")
	cmd.Flags().StringVarP(&dir, "dir", "d", "", "path to proto directory")
	cmd.Flags().StringVarP(&out, "out", "o", "", "output directory for generated code")

	return cmd
}

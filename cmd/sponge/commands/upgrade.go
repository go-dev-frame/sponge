package commands

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/go-dev-frame/sponge/pkg/gobash"
	"github.com/go-dev-frame/sponge/pkg/gofile"
	"github.com/go-dev-frame/sponge/pkg/utils"
)

// UpgradeCommand upgrade sponge binaries
func UpgradeCommand() *cobra.Command {
	var targetVersion string

	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade sponge version",
		Long:  "Upgrade sponge version.",
		Example: color.HiBlackString(`  # Upgrade to latest version
  sponge upgrade

  # Upgrade to specified version
  sponge upgrade --version=v1.5.6`),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if targetVersion == "" {
				targetVersion = latestVersion
			}
			ver, err := runUpgrade(targetVersion)
			if err != nil {
				return err
			}
			fmt.Printf("upgraded version to %s successfully.\n", ver)
			return nil
		},
	}

	cmd.Flags().StringVarP(&targetVersion, "version", "v", latestVersion, "upgrade sponge version")
	return cmd
}

func runUpgrade(targetVersion string) (string, error) {
	runningTip := "sponge binary upgrading "
	finishTip := "sponge binary upgraded " + installedSymbol
	failTip := "sponge binary upgrade failed " + lackSymbol
	p := utils.NewWaitPrinter(time.Millisecond * 500)
	p.LoopPrint(runningTip)
	err := runUpgradeCommand(targetVersion)
	if err != nil {
		p.StopPrint(failTip + "\nError: " + err.Error())
		return "", err
	}
	p.StopPrint(finishTip)

	runningTip = "template code upgrading "
	finishTip = "template code upgraded " + installedSymbol
	failTip = "template code upgrade failed " + lackSymbol
	p = utils.NewWaitPrinter(time.Millisecond * 500)
	p.LoopPrint(runningTip)
	ver, err := copyToTempDir(targetVersion)
	if err != nil {
		p.StopPrint(failTip + "\nError: " + err.Error())
		return "", err
	}
	p.StopPrint(finishTip)

	runningTip = "built-in plugins upgrading "
	finishTip = "built-in plugins upgraded " + installedSymbol
	failTip = "built-in plugins upgrade failed " + lackSymbol
	p = utils.NewWaitPrinter(time.Millisecond * 500)
	p.LoopPrint(runningTip)
	err = updateSpongeInternalPlugin(ver)
	if err != nil {
		p.StopPrint(failTip + "\nError: " + err.Error())
		return "", err
	}
	p.StopPrint(finishTip)
	return ver, nil
}

func runUpgradeCommand(targetVersion string) error {
	ctx, _ := context.WithTimeout(context.Background(), time.Minute*3) //nolint
	spongeVersion := "github.com/go-dev-frame/sponge/cmd/sponge@" + targetVersion
	if compareVersion(separatedVersion, targetVersion) {
		spongeVersion = strings.ReplaceAll(spongeVersion, "go-dev-frame", "zhufuyi")
	}
	result := gobash.Run(ctx, "go", "install", spongeVersion)
	for v := range result.StdOut {
		_ = v
	}
	if result.Err != nil {
		return result.Err
	}
	return nil
}

// copy the template files to a temporary directory
func copyToTempDir(targetVersion string) (string, error) {
	result, err := gobash.Exec("go", "env", "GOPATH")
	if err != nil {
		return "", fmt.Errorf("execute command failed, %v", err)
	}
	gopath := strings.ReplaceAll(string(result), "\n", "")
	if gopath == "" {
		return "", fmt.Errorf("$GOPATH is empty, you need set $GOPATH in your $PATH")
	}
	delimiter := ":"
	if gofile.IsWindows() {
		delimiter = ";"
	}
	if ss := strings.Split(gopath, delimiter); len(ss) > 1 {
		gopath = ss[0] // use the first $GOPATH
	}

	spongeDirName := ""
	if targetVersion == latestVersion {
		// find the new version of the sponge code directory
		arg := fmt.Sprintf("%s/pkg/mod/github.com/go-dev-frame", gopath)
		if compareVersion(separatedVersion, targetVersion) {
			arg = strings.ReplaceAll(arg, "go-dev-frame", "zhufuyi")
		}
		result, err = gobash.Exec("ls", adaptPathDelimiter(arg))
		if err != nil {
			return "", fmt.Errorf("execute command failed, %v", err)
		}

		spongeDirName = getLatestVersion(string(result))
		if spongeDirName == "" {
			return "", fmt.Errorf("not found sponge directory in '$GOPATH/pkg/mod/github.com/go-dev-frame'")
		}
	} else {
		spongeDirName = "sponge@" + targetVersion
	}

	srcDir := adaptPathDelimiter(fmt.Sprintf("%s/pkg/mod/github.com/go-dev-frame/%s", gopath, spongeDirName))
	if compareVersion(separatedVersion, targetVersion) {
		srcDir = strings.ReplaceAll(srcDir, "go-dev-frame", "zhufuyi")
	}
	destDir := adaptPathDelimiter(GetSpongeDir() + "/")
	targetDir := adaptPathDelimiter(destDir + ".sponge")

	err = executeCommand("rm", "-rf", targetDir)
	if err != nil {
		return "", err
	}
	err = executeCommand("cp", "-rf", srcDir, targetDir)
	if err != nil {
		return "", err
	}
	err = executeCommand("chmod", "-R", "744", targetDir)
	if err != nil {
		return "", err
	}
	_ = executeCommand("rm", "-rf", targetDir+"/cmd/sponge")
	_ = executeCommand("rm", "-rf", targetDir+"/cmd/protoc-gen-go-gin")
	_ = executeCommand("rm", "-rf", targetDir+"/cmd/protoc-gen-go-rpc-tmpl")
	_ = executeCommand("rm", "-rf", targetDir+"/cmd/protoc-gen-json-field")
	_ = executeCommand("rm", "-rf", targetDir+"/pkg")
	_ = executeCommand("rm", "-rf", targetDir+"/test")
	_ = executeCommand("rm", "-rf", targetDir+"/assets")

	versionNum := strings.Replace(spongeDirName, "sponge@", "", 1)
	err = os.WriteFile(versionFile, []byte(versionNum), 0644)
	if err != nil {
		return "", err
	}

	return versionNum, nil
}

func executeCommand(name string, args ...string) error {
	ctx, _ := context.WithTimeout(context.Background(), time.Second*30) //nolint
	result := gobash.Run(ctx, name, args...)
	for v := range result.StdOut {
		_ = v
	}
	if result.Err != nil {
		return fmt.Errorf("execute command failed, %v", result.Err)
	}
	return nil
}

func adaptPathDelimiter(filePath string) string {
	if gofile.IsWindows() {
		filePath = strings.ReplaceAll(filePath, "/", "\\")
	}
	return filePath
}

func getLatestVersion(s string) string {
	var dirNames = make(map[int]string)
	var nums []int

	dirs := strings.Split(s, "\n")
	for _, dirName := range dirs {
		if strings.Contains(dirName, "sponge@") {
			tmp := strings.ReplaceAll(dirName, "sponge@", "")
			ss := strings.Split(tmp, ".")
			if len(ss) != 3 {
				continue
			}
			if strings.Contains(ss[2], "v0.0.0") {
				continue
			}
			num := utils.StrToInt(ss[0])*10000 + utils.StrToInt(ss[1])*100 + utils.StrToInt(ss[2])
			nums = append(nums, num)
			dirNames[num] = dirName
		}
	}
	if len(nums) == 0 {
		return ""
	}

	sort.Ints(nums)
	return dirNames[nums[len(nums)-1]]
}

func updateSpongeInternalPlugin(targetVersion string) error {
	ctx, _ := context.WithTimeout(context.Background(), 3*time.Minute) //nolint
	genGinVersion := "github.com/go-dev-frame/sponge/cmd/protoc-gen-go-gin@" + targetVersion
	if compareVersion(separatedVersion, targetVersion) {
		genGinVersion = strings.ReplaceAll(genGinVersion, "go-dev-frame", "zhufuyi")
	}
	result := gobash.Run(ctx, "go", "install", genGinVersion)
	for v := range result.StdOut {
		_ = v
	}
	if result.Err != nil {
		return result.Err
	}

	ctx, _ = context.WithTimeout(context.Background(), 3*time.Minute) //nolint
	genRPCVersion := "github.com/go-dev-frame/sponge/cmd/protoc-gen-go-rpc-tmpl@" + targetVersion
	if compareVersion(separatedVersion, targetVersion) {
		genRPCVersion = strings.ReplaceAll(genRPCVersion, "go-dev-frame", "zhufuyi")
	}
	result = gobash.Run(ctx, "go", "install", genRPCVersion)
	for v := range result.StdOut {
		_ = v
	}
	if result.Err != nil {
		return result.Err
	}

	// v1.x.x version does not support protoc-gen-json-field
	if !strings.HasPrefix(targetVersion, "v1") {
		ctx, _ = context.WithTimeout(context.Background(), 3*time.Minute) //nolint
		genJSONVersion := "github.com/go-dev-frame/sponge/cmd/protoc-gen-json-field@" + targetVersion
		if compareVersion(separatedVersion, targetVersion) {
			genJSONVersion = strings.ReplaceAll(genJSONVersion, "go-dev-frame", "zhufuyi")
		}
		result = gobash.Run(ctx, "go", "install", genJSONVersion)
		for v := range result.StdOut {
			_ = v
		}
		if result.Err != nil {
			return result.Err
		}
	}

	return nil
}

// v1 >= v2 return true
// v1 < v2 return false
func compareVersion(v1, v2 string) bool {
	if v1 == "latest" {
		return true
	}
	if v2 == "latest" {
		return false
	}

	v1 = strings.ReplaceAll(v1, "v", "")
	v2 = strings.ReplaceAll(v2, "v", "")
	v1s := strings.Split(v1, ".")
	v2s := strings.Split(v2, ".")
	if len(v1s) < 3 || len(v2s) < 3 {
		return false
	}

	if v1s[0] != v2s[0] {
		return utils.StrToInt(v1s[0]) > utils.StrToInt(v2s[0])
	}

	if v1s[1] != v2s[1] {
		return utils.StrToInt(v1s[1]) > utils.StrToInt(v2s[1])
	}

	return utils.StrToInt(v1s[2]) > utils.StrToInt(v2s[2])
}

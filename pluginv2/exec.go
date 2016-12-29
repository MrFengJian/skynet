package pluginv2

import (
        "os/exec"
        "errors"
        "github.com/pborman/uuid"
        "os"
        "fmt"
)

func Exec(array []string) error {
        if len(array) < 1 {
                return errors.New("empty command to execute")
        }
        cmd := array[0]
        args := array[1:]
        command := exec.Command(cmd, args...)
        output, err := command.Output()
        if err != nil {
                fmt.Fprintf(os.Stderr, "Skynet error happens while exec %s %s\n", array, string(output))
        }
        return err
}

func NsExec(nsPath string, array []string) (err error) {
        nsDir := "/var/run/netns/"
        nsName := uuid.New()
        nsNormalPath := nsDir + nsName
        if err = os.MkdirAll(nsDir, 0755); err != nil {
                return err
        }
        os.Symlink(nsPath, nsNormalPath)
        defer os.Remove(nsNormalPath)
        all := []string{"ip", "netns", nsName, "exec"}
        all = append(all, array...)
        return Exec(all)
}

func changeTonsArgs(nsName string, array []string) []string {
        return append([]string{"ip", "netns", "exec", nsName}, array...)
}
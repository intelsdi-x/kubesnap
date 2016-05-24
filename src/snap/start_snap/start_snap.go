package main

import "os"
import "os/exec"
import "net/http"
import "sync"
import "time"

func main() {
        var wg sync.WaitGroup
        wg.Add(2)
        go exec.Command(os.Getenv("SNAPD_BIN"), "-t", "0", "-a", os.Getenv("PLUGINS_AUTOLOAD_DIR")).Run()
        go func() {
                defer wg.Done()
                for true {
                        resp, err := http.Get("http://localhost:8181/v1/plugins")
                        if err != nil {
                                time.Sleep(time.Second)
                                continue
                        }
                        if resp.StatusCode == 200 {
                                exec.Command(os.Getenv("SNAPCTL_BIN"), "task", "create", "-t", os.Getenv("TASK_AUTOLOAD_FILE")).Run()
                                return
                        }
                        break
                }
        }()
        wg.Wait()
}

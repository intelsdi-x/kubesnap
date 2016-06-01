package main

import "os"
import "os/exec"
import "net/http"
import "sync"
import "time"
import "io/ioutil"
import "encoding/json"
import "strconv"

type Plugins struct {
        Body Body `json:"body"`
}

type Body struct {
        LoadedPluigns []interface{} `json:"loaded_plugins"`
}

func main() {
        plugins := Plugins{}
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
                                body, err := ioutil.ReadAll(resp.Body)
                                defer resp.Body.Close()
                                if err != nil {
                                        return
                                }
                                json.Unmarshal(body, &plugins)
                                numPlugins, _ := strconv.Atoi(os.Getenv("PLUGINS_TO_LOAD"))
                                if len(plugins.Body.LoadedPluigns) < numPlugins {
                                        time.Sleep(time.Second)
                                        continue
                                }
                                exec.Command(os.Getenv("SNAPCTL_BIN"), "task", "create", "-t", os.Getenv("TASK_AUTOLOAD_FILE")).Run()
                                return
                        }
                        continue
                }
        }()
        wg.Wait()
}

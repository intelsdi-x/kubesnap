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
        LoadedPlugins []interface{} `json:"loaded_plugins"`
}

func getPlugins(path string) []string {
        var plugins []string
        files, _ := ioutil.ReadDir(path)
        for _, p := range files {
                plugins = append(plugins, path + "/" + p.Name())
        }
        return plugins
}

func main() {
        pluginsDir := os.Getenv("PLUGINS_AUTOLOAD_DIR")
        pluginsToLoad := os.Getenv("PLUGINS_TO_LOAD")
        snapd := os.Getenv("SNAPD_BIN")
        snapctl := os.Getenv("SNAPCTL_BIN")
        task := os.Getenv("TASK_AUTOLOAD_FILE")

        plugins := Plugins{}
        var wg sync.WaitGroup

        wg.Add(2)
        go exec.Command(snapd, "-t", "0", "-a", pluginsDir).Run()
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
                                numPlugins, _ := strconv.Atoi(pluginsToLoad)
                                if len(plugins.Body.LoadedPlugins) < numPlugins {
					// force load
                                        for _, p := range getPlugins(pluginsDir ) {
                                                exec.Command(snapctl, "plugin", "load", p).Run()
                                        }
                                        time.Sleep(time.Second)
                                        continue
                                }
                                exec.Command(snapctl, "task", "create", "-t", task).Run()
                                return
                        }
                        continue
                }
        }()
        wg.Wait()
}

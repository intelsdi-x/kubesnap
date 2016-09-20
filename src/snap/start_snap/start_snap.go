/*
http://www.apache.org/licenses/LICENSE-2.0.txt


Copyright 2016 Intel Corporation

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"time"

	"k8s.io/kubernetes/pkg/api"
	cl "k8s.io/kubernetes/pkg/client/unversioned"
)

const (
	minNodeCount = 2
	timeout      = 5
	interval     = 10
)

type Plugins struct {
	Body Body `json:"body"`
}

type Body struct {
	LoadedPlugins []interface{} `json:"loaded_plugins"`
}

type Tribe struct {
	Body Members `json:"body"`
}

type Members struct {
	Members []string `json:"members"`
}

func getPlugins(path string) []string {
	var plugins []string
	files, _ := ioutil.ReadDir(path)
	for _, p := range files {
		plugins = append(plugins, path+"/"+p.Name())
	}
	return plugins
}

type Client struct {
	namespace   string
	snapService string
	client      *cl.Client
}

func NewClient() (*Client, error) {
	c := new(Client)
	var err error
	c.namespace = os.Getenv("NAMESPACE")
	if c.namespace == "" {
		c.namespace = api.NamespaceSystem
	}
	c.client, err = cl.NewInCluster()
	if err != nil {
		return nil, fmt.Errorf("Cannot initialize client: %v", err)
	}
	c.snapService = os.Getenv("SNAP_SERVICE")
	if c.snapService == "" {
		c.snapService = "snap-tribe"
	}

	return c, nil

}
func GetTribeEndpoints(c *Client) (*api.Endpoints, error) {

	s, err := c.client.Services(c.namespace).Get(c.snapService)
	if err != nil {
		return nil, fmt.Errorf("Cannot get svc: %s :%v", s.GetName(), err)
	}
	snapEndpoints, err := c.client.Endpoints(c.namespace).Get(s.GetName())
	if err != nil {
		return nil, fmt.Errorf("Cannot get endpoints for %s: %v", s.GetName(), err)
	}

	return snapEndpoints, nil
}

func GetTribeSeed(c *Client) (string, error) {

	for t := time.Now(); time.Since(t) < timeout*time.Minute; time.Sleep(interval * time.Second) {
		endpoints, err := GetTribeEndpoints(c)
		if err != nil {
			return "", err
		}
		fmt.Printf("our endpoint list: %+v", endpoints.Subsets[0].Addresses)
		if len(endpoints.Subsets[0].Addresses) >= minNodeCount {
			return endpoints.Subsets[0].Addresses[0].IP, nil
		}
		fmt.Printf("Waiting for endpoints, current number:%v", len(endpoints.Subsets[0].Addresses))
	}

	return "", fmt.Errorf("Timeout waiting for snap tribe endpoints")
}

func GetTribeNodesCount(c *Client) (int, error) {
	endpoints, err := GetTribeEndpoints(c)
	if err != nil {
		return 0, err
	}
	return len(endpoints.Subsets), err
}

func main() {
	f, _ := os.Create("/tmp/start_snap.log")
	defer f.Close()
	w := bufio.NewWriter(f)

	pluginsDir := os.Getenv("PLUGINS_AUTOLOAD_DIR")
	pluginsToLoad := os.Getenv("PLUGINS_TO_LOAD")
	snapd := os.Getenv("SNAPD_BIN")
	snapctl := os.Getenv("SNAPCTL_BIN")
	task := os.Getenv("TASK_AUTOLOAD_FILE")
	client, err := NewClient()
	if err != nil {
		fmt.Printf("Error creating client: %v", err)
		os.Exit(1)
	}
	tribeSeed, err := GetTribeSeed(client)
	if err != nil {
		fmt.Printf("Error getting tribeSeed: %v", err)
		os.Exit(1)
	}
	//tribeSeed := os.Getenv("SNAP_SEED_IP")
	numTribeNodes, err := GetTribeNodesCount(client)
	if err != nil {
		fmt.Printf("Cannot get endpoint size: %v", err)
	}
	if numTribeNodes < minNodeCount {
		numTribeNodes = minNodeCount
	}

	myPodIP := os.Getenv("MY_POD_IP")
	agreement := "all-nodes"

	fmt.Fprintf(w, "tribe seed IP: %s, my POD IP: %s, expcted number of Tribe members: %s\n", tribeSeed, myPodIP, numTribeNodes)

	plugins := Plugins{}
	tribeNodes := Tribe{}
	var wg sync.WaitGroup

	wg.Add(2)
	if myPodIP != tribeSeed {
		fmt.Fprintf(w, "I'm NOT a tribe seed... \n")
		for true {
			w.Flush()
			resp, err := http.Get("http://" + tribeSeed + ":8181/v1/tribe/members")
			if err != nil {
				fmt.Fprintf(w, "Error listing tribe members - is seed ready?\n")
				time.Sleep(time.Second)
				continue
			}
			if resp.StatusCode == 200 {
				_, err := ioutil.ReadAll(resp.Body)
				defer resp.Body.Close()
				if err != nil {
					fmt.Fprintf(w, "Cannot parse response body for tribe members - exiting\n")
					return
				}
				fmt.Fprintf(w, "Response body for tribe members is valid - about to start snapd\n")
				break
			}
			fmt.Fprintf(w, "Listing tribe members not successful - waiting\n")
			time.Sleep(time.Second)
			continue
		}
		fmt.Fprintf(w, "Starintg snapd with tribe seed: %s\n", tribeSeed)
		w.Flush()
		go exec.Command(snapd, "-l", "1", "-o", "/tmp", "-t", "0", "--tribe", "--tribe-seed", tribeSeed, "--tribe-addr", myPodIP).Run()
		wg.Wait()
	}
	fmt.Fprintf(w, "I'm a tribe seed\n")
	go exec.Command(snapd, "-l", "1", "-o", "/tmp", "-t", "0", "--tribe", "--tribe-addr", myPodIP).Run()
	go func() {
		defer wg.Done()
		for true {
			w.Flush()
			resp, err := http.Get("http://localhost:8181/v1/tribe/members")
			if err != nil {
				fmt.Fprintf(w, "Error listing tribe members - is snapd ready?\n")
				time.Sleep(time.Second)
				continue
			}
			if resp.StatusCode == 200 {
				body, err := ioutil.ReadAll(resp.Body)
				defer resp.Body.Close()
				if err != nil {
					fmt.Fprintf(w, "Cannot parse response body for tribe members - exiting\n")
					return
				}
				json.Unmarshal(body, &tribeNodes)
				numNodes := numTribeNodes
				if len(tribeNodes.Body.Members) < numNodes {
					fmt.Fprintf(w, "Too few tribe members. Got: %v (%+v), Need: %v\n", len(tribeNodes.Body.Members), tribeNodes, numNodes)
					time.Sleep(time.Second)
					continue
				}
				fmt.Fprintf(w, "Got all tribe members (%+v) - creating agreement: %s\n", tribeNodes, agreement)
				exec.Command(snapctl, "agreement", "create", agreement).Run()
				fmt.Fprintf(w, "Attaching all nodes to agreeement... \n")
				for _, n := range tribeNodes.Body.Members {
					fmt.Fprintf(w, "Attaching node (%+v) to agreeement: %s\n", n, agreement)
					exec.Command(snapctl, "agreement", "join", agreement, n).Run()
					time.Sleep(time.Second)
					w.Flush()
				}
				break
			}
			fmt.Fprintf(w, "Listing tribe members not successful - waiting\n")
			w.Flush()
			continue
		}
		for true {
			w.Flush()
			fmt.Fprintf(w, "Loading plugins...\n")
			resp, err := http.Get("http://localhost:8181/v1/plugins")
			if err != nil {
				fmt.Fprintf(w, "Error listing plugins - is snapd ready?\n")
				time.Sleep(time.Second)
				continue
			}
			if resp.StatusCode == 200 {
				body, err := ioutil.ReadAll(resp.Body)
				defer resp.Body.Close()
				if err != nil {
					fmt.Fprintf(w, "Cannot parse response body for plugins list - exiting\n")
					return
				}
				json.Unmarshal(body, &plugins)
				numPlugins, _ := strconv.Atoi(pluginsToLoad)
				if len(plugins.Body.LoadedPlugins) < numPlugins {
					fmt.Fprintf(w, "Too few plugins loaded...\n")
					for _, p := range getPlugins(pluginsDir) {
						fmt.Fprintf(w, "Loading plugin: %+v\n", p)
						exec.Command(snapctl, "plugin", "load", p).Run()
					}
					time.Sleep(time.Second)
					continue
				}
				fmt.Fprintf(w, "All plugins loaded -  starting task\n")
				w.Flush()
				// account for plugins loaded on remote nodes
				// TODO improve this
				time.Sleep(3 * time.Second)
				exec.Command(snapctl, "task", "create", "-t", task).Run()
				return
			}
			fmt.Fprintf(w, "Listing plugins not successful - waiting\n")
			w.Flush()
			continue
		}
	}()
	wg.Wait()
}

## snap publisher with embedded REST API server

### Example configuration

Here is a sample task manifest utilizing the heapster publisher plugin:

```
  version: 1
  schedule:
    type: "simple"
    interval: "1s"
  workflow:
    collect:
      metrics:
        /intel/linux/docker/*/cpu_stats/cpu_usage/total_usage: {}
        /intel/linux/docker/*/memory_stats/usage/usage: {}
        /intel/linux/docker/*/memory_stats/stats/pgfault: {}
        /intel/linux/docker/*/memory_stats/stats/pgmajfault: {}
      process:
        -
          plugin_name: "passthru"
          process: null
          publish:
            -
              plugin_name: "heapster"
              config:
                stats_depth: 9
                server_port: 8777
```

Explanation:
* this will setup heapster publisher to expose REST server at port :8777
* publisher will keep a list of 9 most recent stats collected per each docker 

package publisher

const builtinMetricTemplate = `{
"id": "!!",
"name": "!!",
"aliases": [
],
"namespace": "??",
"labels": {
},
  "subcontainers": [],
"spec": {
 "creation_time": "??",
 "labels": {
 },
 "has_cpu": true,
 "cpu": {
 },
 "has_memory": true,
 "memory": {
 },
 "has_network": false,
 "has_filesystem": true,
 "has_diskio": true,
 "has_custom_metrics": false,
 "custom_metrics":[],
 "image": "??"
},
"stats": [
 {
  "timestamp": "!!",
  "custom_metrics": {
   "SNAP": [
   ]
  },
  "cpu": {
   "usage": {
    "total": "/cpu_stats/cpu_usage/total_usage"
   }
  },
  "memory": {
   "usage": "/memory_stats/usage/usage",
   "container_data": {
    "pgfault": "/memory_stats/stats/pgfault",
    "pgmajfault": "/memory_stats/stats/pgmajfault"
   }
  }
 }
]
}
`
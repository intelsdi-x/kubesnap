# Kubernetes stress test pod

To run stress pod, you must define what you want to stress. You must create new file based on template and modify **args: []** field with stress tool arguments. By default you will see:

    args: ["-c", "1"]

Just replace this with your arguments. You can see full stress tool argument list here:

| Argument          | Description                                           |
|-------------------|-------------------------------------------------------|
|-?, --help         |show this help statement                               |
| --version         |show version statement                                 |
|-v, --verbose      |be verbose                                             |
|-q, --quiet        |be quiet                                               |
|-n, --dry-run      |show what would have been done                         |
|-t, --timeout N    |timeout after N seconds                                |
| --backoff N       |wait factor of N microseconds before work starts       |
|-c, --cu N         |spawn N workers spinning on sqrt()                     |
|-i, --io N         |spawn N workers spinning on sync()                     |
|-m, --vm N         |spawn N workers spinning on malloc()/free()            |
| --vm-bytes B      |malloc B bytes per vm worker (default is 256MB)        |
| --vm-stride B     |touch a byte every B bytes (default is 4096)           |
| --vm-hang N       |sleep N secs before free (default none, 0 is inf)      |
| --vm-keep         |redirty memory instead of freeing and reallocating     |
|-d, --hdd N        |spawn N workers spinning on write()/unlink()           |
| --hdd-bytes B     |write B bytes per hdd worker (default is 1GB)          |

Note: Numbers may be suffixed with s,m,h,d,y (time) or B,K,M,G (size).

## Create pod or daemonset

If you don't know how to create or remove pod using kubectl tool, see [snap pods](/snap/pods/README.md#create-pod-or-daemonset).
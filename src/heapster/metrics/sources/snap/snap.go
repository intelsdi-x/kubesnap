package snap

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	cadvisor "github.com/google/cadvisor/info/v1"
	. "k8s.io/heapster/metrics/core"
	"k8s.io/heapster/metrics/sources/kubelet"

	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
	kube_api "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/client/cache"
	kube_client "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/fields"
)

var (
	snapRequestLatency = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace: "heapster",
			Subsystem: "snap",
			Name:      "request_duration_microseconds",
			Help:      "The snap request latencies in microseconds.",
		},
		[]string{"node"},
	)
)

const (
	infraContainerName = "POD"
	// TODO: following constants are copied from k8s, change to use them directly
	kubernetesPodNameLabel      = "io.kubernetes.pod.name"
	kubernetesPodNamespaceLabel = "io.kubernetes.pod.namespace"
	kubernetesPodUID            = "io.kubernetes.pod.uid"
	kubernetesContainerLabel    = "io.kubernetes.container.name"
)

func init() {
	prometheus.MustRegister(snapRequestLatency)
}

// snap-provided metrics for pod and system container.
type snapMetricsSource struct {
	host       kubelet.Host
	snapClient *SnapClient
	nodename   string
	hostname   string
	hostId     string
}

func NewSnapMetricsSource(host kubelet.Host, client *SnapClient, nodeName, hostName, hostId string) MetricsSource {
	return &snapMetricsSource{
		host:       host,
		snapClient: client,
		nodename:   nodeName,
		hostname:   hostName,
		hostId:     hostId,
	}
}

func (s *snapMetricsSource) Name() string {
	return s.String()
}

func (s *snapMetricsSource) String() string {
	return fmt.Sprintf("snap:%s:%d", s.host.IP, s.host.Port)
}

func (s *snapMetricsSource) handleSystemContainer(c *cadvisor.ContainerInfo, cMetrics *MetricSet) string {
	glog.V(8).Infof("SNAP Found system container %v with labels: %+v", c.Name, c.Spec.Labels)
	cName := c.Name
	if strings.HasPrefix(cName, "/") {
		cName = cName[1:]
	}
	cMetrics.Labels[LabelMetricSetType.Key] = MetricSetTypeSystemContainer
	cMetrics.Labels[LabelContainerName.Key] = cName
	return NodeContainerKey(s.nodename, cName)
}

func (s *snapMetricsSource) handleKubernetesContainer(cName, ns, podName string, c *cadvisor.ContainerInfo, cMetrics *MetricSet) string {
	var metricSetKey string
	if cName == infraContainerName {
		metricSetKey = PodKey(ns, podName)
		cMetrics.Labels[LabelMetricSetType.Key] = MetricSetTypePod
	} else {
		metricSetKey = PodContainerKey(ns, podName, cName)
		cMetrics.Labels[LabelMetricSetType.Key] = MetricSetTypePodContainer
		cMetrics.Labels[LabelContainerName.Key] = cName
		cMetrics.Labels[LabelContainerBaseImage.Key] = c.Spec.Image
	}
	cMetrics.Labels[LabelPodId.Key] = c.Spec.Labels[kubernetesPodUID]
	cMetrics.Labels[LabelPodName.Key] = podName
	cMetrics.Labels[LabelNamespaceName.Key] = ns
	// Needed for backward compatibility
	cMetrics.Labels[LabelPodNamespace.Key] = ns
	return metricSetKey
}

func isNode(c *cadvisor.ContainerInfo) bool {
	return c.Name == "/"
}

func (s *snapMetricsSource) decodeMetrics(c *cadvisor.ContainerInfo) (string, *MetricSet) {
	if len(c.Stats) == 0 {
		return "", nil
	}

	var metricSetKey string
	cMetrics := &MetricSet{
		CreateTime:   c.Spec.CreationTime,
		ScrapeTime:   c.Stats[0].Timestamp,
		MetricValues: map[string]MetricValue{},
		Labels: map[string]string{
			LabelNodename.Key: s.nodename,
			LabelHostname.Key: s.hostname,
			LabelHostID.Key:   s.hostId,
		},
		LabeledMetrics: []LabeledMetric{},
	}

	if isNode(c) {
		metricSetKey = NodeKey(s.nodename)
		cMetrics.Labels[LabelMetricSetType.Key] = MetricSetTypeNode
	} else {
		cName := c.Spec.Labels[kubernetesContainerLabel]
		ns := c.Spec.Labels[kubernetesPodNamespaceLabel]
		podName := c.Spec.Labels[kubernetesPodNameLabel]

		// Support for kubernetes 1.0.*
		if ns == "" && strings.Contains(podName, "/") {
			tokens := strings.SplitN(podName, "/", 2)
			if len(tokens) == 2 {
				ns = tokens[0]
				podName = tokens[1]
			}
		}
		if cName == "" {
			// Better this than nothing. This is a temporary hack for new heapster to work
			// with Kubernetes 1.0.*.
			// TODO: fix this with POD list.
			// Parsing name like:
			// k8s_kube-ui.7f9b83f6_kube-ui-v1-bxj1w_kube-system_9abfb0bd-811f-11e5-b548-42010af00002_e6841e8d
			pos := strings.Index(c.Name, ".")
			if pos >= 0 {
				// remove first 4 chars.
				cName = c.Name[len("k8s_"):pos]
			}
		}

		// No Kubernetes metadata so treat this as a system container.
		if cName == "" || ns == "" || podName == "" {
			metricSetKey = s.handleSystemContainer(c, cMetrics)
		} else {
			metricSetKey = s.handleKubernetesContainer(cName, ns, podName, c, cMetrics)
		}
	}

	for _, metric := range StandardMetrics {
		if metric.HasValue != nil && metric.HasValue(&c.Spec) {
			cMetrics.MetricValues[metric.Name] = metric.GetValue(&c.Spec, c.Stats[0])
		}
	}

	for _, metric := range LabeledMetrics {
		if metric.HasLabeledMetric != nil && metric.HasLabeledMetric(&c.Spec) {
			labeledMetrics := metric.GetLabeledMetric(&c.Spec, c.Stats[0])
			cMetrics.LabeledMetrics = append(cMetrics.LabeledMetrics, labeledMetrics...)
		}
	}

	if c.Spec.HasCustomMetrics {
	metricloop:
		for _, spec := range c.Spec.CustomMetrics {
			if cmValue, ok := c.Stats[0].CustomMetrics[spec.Name]; ok && cmValue != nil && len(cmValue) >= 1 {
				newest := cmValue[0]
				for _, metricVal := range cmValue {
					if newest.Timestamp.Before(metricVal.Timestamp) {
						newest = metricVal
					}
				}
				mv := MetricValue{}
				switch spec.Type {
				case cadvisor.MetricGauge:
					mv.MetricType = MetricGauge
				case cadvisor.MetricCumulative:
					mv.MetricType = MetricCumulative
				default:
					glog.V(4).Infof("Skipping %s: unknown custom metric type: %v", spec.Name, spec.Type)
					continue metricloop
				}

				switch spec.Format {
				case cadvisor.IntType:
					mv.ValueType = ValueInt64
					mv.IntValue = newest.IntValue
				case cadvisor.FloatType:
					mv.ValueType = ValueFloat
					mv.FloatValue = float32(newest.FloatValue)
				default:
					glog.V(4).Infof("Skipping %s: unknown custom metric format", spec.Name, spec.Format)
					continue metricloop
				}

				cMetrics.MetricValues[CustomMetricPrefix+spec.Name] = mv
			}
		}
	}

	return metricSetKey, cMetrics
}

func (s *snapMetricsSource) ScrapeMetrics(start, end time.Time) *DataBatch {
	containers, err := s.scrapeSnap(s.snapClient, s.host, start, end)
	if err != nil {
		glog.Errorf("error while getting containers from snap: %v", err)
	}
	glog.V(2).Infof("SNAP successfully obtained stats for %v containers", len(containers))
	glog.Infof("SNAP Scraping metrics start: %s, end: %s", start, end)

	result := &DataBatch{
		Timestamp:  end,
		MetricSets: map[string]*MetricSet{},
	}
	keys := make(map[string]bool)
	for _, c := range containers {
		name, metrics := s.decodeMetrics(&c)
		if name == "" || metrics == nil {
			continue
		}
		result.MetricSets[name] = metrics
		keys[name] = true
	}
	return result
}

func (s *snapMetricsSource) scrapeSnap(client *SnapClient, host kubelet.Host, start, end time.Time) ([]cadvisor.ContainerInfo, error) {
	startTime := time.Now()
	kc := &kubelet.KubeletClient{}
	defer snapRequestLatency.WithLabelValues(s.hostname).Observe(float64(time.Since(startTime)))
	return kc.GetAllRawContainers(host, start, end)
}

type snapProvider struct {
	nodeLister *cache.StoreToNodeLister
	reflector  *cache.Reflector
	snapClient *SnapClient
}

func (s *snapProvider) GetMetricsSources() []MetricsSource {
	sources := []MetricsSource{}
	nodes, err := s.nodeLister.List()
	if err != nil {
		glog.Errorf("error while listing nodes: %v", err)
		return sources
	}
	if len(nodes.Items) == 0 {
		glog.Error("No nodes received from APIserver.")
		return sources
	}

	nodeNames := make(map[string]bool)
	for _, node := range nodes.Items {
		nodeNames[node.Name] = true
		hostname, ip, err := getNodeHostnameAndIP(&node)
		if err != nil {
			glog.Errorf("%v", err)
			continue
		}
		sources = append(sources, NewSnapMetricsSource(
			kubelet.Host{IP: ip, Port: 8777},
			s.snapClient,
			node.Name,
			hostname,
			node.Spec.ExternalID,
		))
	}
	return sources
}

func getNodeHostnameAndIP(node *kube_api.Node) (string, string, error) {
	for _, c := range node.Status.Conditions {
		if c.Type == kube_api.NodeReady && c.Status != kube_api.ConditionTrue {
			return "", "", fmt.Errorf("SNAP Node %v is not ready", node.Name)
		}
	}
	hostname, ip := node.Name, ""
	for _, addr := range node.Status.Addresses {
		if addr.Type == kube_api.NodeHostName && addr.Address != "" {
			hostname = addr.Address
		}
		if addr.Type == kube_api.NodeInternalIP && addr.Address != "" {
			ip = addr.Address
		}
		if addr.Type == kube_api.NodeLegacyHostIP && addr.Address != "" && ip == "" {
			ip = addr.Address
		}
	}
	if ip != "" {
		return hostname, ip, nil
	}
	return "", "", fmt.Errorf("SNAP Node %v has no valid hostname and/or IP address: %v %v", node.Name, hostname, ip)
}

type SnapClient struct {
	client *http.Client
}

func newSnapClient() *SnapClient {
	c := &http.Client{
		Transport: http.DefaultTransport,
		Timeout:   time.Duration(1) * time.Second,
	}
	return &SnapClient{
		client: c,
	}
}

func NewSnapProvider(uri *url.URL) (MetricsSourceProvider, error) {
	kubeConfig, _, err := kubelet.GetKubeConfigs(uri)
	if err != nil {
		return nil, err
	}
	kubeClient := kube_client.NewOrDie(kubeConfig)
	snapClient := newSnapClient()

	// watch nodes
	lw := cache.NewListWatchFromClient(kubeClient, "nodes", kube_api.NamespaceAll, fields.Everything())
	nodeLister := &cache.StoreToNodeLister{Store: cache.NewStore(cache.MetaNamespaceKeyFunc)}
	reflector := cache.NewReflector(lw, &kube_api.Node{}, nodeLister.Store, time.Hour)
	reflector.Run()

	return &snapProvider{
		nodeLister: nodeLister,
		reflector:  reflector,
		snapClient: snapClient,
	}, nil
}

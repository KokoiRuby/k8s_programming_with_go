## Recap

kubelet as Worker agent is repsonsible for:

- **Pod LCM**: Reconcile PodSpec where src
  - API Server
  - File `--pod-manifest-path`
  - HTTP (server-push & endpoint pull)
- **Container HC**: LivenessProbe (eviction & restartPolicy) & ReadinessProbe
- **Container monitoring** via CAdvisor

## Mechanism

驱动同步循环的事件：pod 更新事件、pod 生命周期变化、kubelet 本身设置的执行周期、定时清理 housekeeping 事件等。

基于只可接收的 channel `<-chan`。

`*Manager`

![20200920120525.png](https://github.com/luozhiyun993/luozhiyun-SourceLearn/blob/master/%E6%B7%B1%E5%85%A5k8s/11.%E6%B7%B1%E5%85%A5k8s%EF%BC%9Akubelet%E5%B7%A5%E4%BD%9C%E5%8E%9F%E7%90%86%E5%8F%8A%E5%85%B6%E5%88%9D%E5%A7%8B%E5%8C%96%E6%BA%90%E7%A0%81%E5%88%86%E6%9E%90/20200920120525.png?raw=true)

kubelet → CRI gRPC → Container Runtime **底层容器运行时只需实现 gRPC 定义接口，以及启动 gRPC 服务**。

```go
// kubernetes/pkg/kubelet/apis/cri/runtime/v1alpha2/api.proto

// Pod & Container LCM
service RuntimeService {
    rpc Version(VersionRequest) returns (VersionResponse) {}
    rpc RunPodSandbox(RunPodSandboxRequest) returns (RunPodSandboxResponse) {}
    rpc StopPodSandbox(StopPodSandboxRequest) returns (StopPodSandboxResponse) {}
    rpc RemovePodSandbox(RemovePodSandboxRequest) returns (RemovePodSandboxResponse) {}
    rpc PodSandboxStatus(PodSandboxStatusRequest) returns (PodSandboxStatusResponse) {}
    rpc ListPodSandbox(ListPodSandboxRequest) returns (ListPodSandboxResponse) {}
    rpc CreateContainer(CreateContainerRequest) returns (CreateContainerResponse) {}
    rpc StartContainer(StartContainerRequest) returns (StartContainerResponse) {}
    rpc StopContainer(StopContainerRequest) returns (StopContainerResponse) {}
    rpc RemoveContainer(RemoveContainerRequest) returns (RemoveContainerResponse) {}
    rpc ListContainers(ListContainersRequest) returns (ListContainersResponse) {}
    rpc ContainerStatus(ContainerStatusRequest) returns (ContainerStatusResponse) {}
    rpc UpdateContainerResources(UpdateContainerResourcesRequest) returns (UpdateContainerResourcesResponse) {}
    rpc ReopenContainerLog(ReopenContainerLogRequest) returns (ReopenContainerLogResponse) {}
    rpc ExecSync(ExecSyncRequest) returns (ExecSyncResponse) {}
    rpc Exec(ExecRequest) returns (ExecResponse) {}
    rpc Attach(AttachRequest) returns (AttachResponse) {}
    rpc PortForward(PortForwardRequest) returns (PortForwardResponse) {}
    rpc ContainerStats(ContainerStatsRequest) returns (ContainerStatsResponse) {}
    rpc ListContainerStats(ListContainerStatsRequest) returns (ListContainerStatsResponse) {}
}

// Image LCM
service ImageService {
    rpc ListImages(ListImagesRequest) returns (ListImagesResponse) {}
    rpc ImageStatus(ImageStatusRequest) returns (ImageStatusResponse) {}
    rpc PullImage(PullImageRequest) returns (PullImageResponse) {}
    rpc RemoveImage(RemoveImageRequest) returns (RemoveImageResponse) {}
    rpc ImageFsInfo(ImageFsInfoRequest) returns (ImageFsInfoResponse) {}
}

```



![20200920120529.png](https://github.com/luozhiyun993/luozhiyun-SourceLearn/blob/master/%E6%B7%B1%E5%85%A5k8s/11.%E6%B7%B1%E5%85%A5k8s%EF%BC%9Akubelet%E5%B7%A5%E4%BD%9C%E5%8E%9F%E7%90%86%E5%8F%8A%E5%85%B6%E5%88%9D%E5%A7%8B%E5%8C%96%E6%BA%90%E7%A0%81%E5%88%86%E6%9E%90/20200920120529.png?raw=true)

## Run

1. Setup logServer & logClient
2. 启动 cloud provider sync manager
3. 初始化（不依赖 CR 的）模块
4. 启动 volume manager
5. 定时同步节点状态
6. 执行首次状态同步
7. 启动 node lease 类心跳上报 API Server
8. 定时更新 CR 状态
9. 设置 iptables rule 并执行定时同步
10. 启动 status manager
11. 启动 PLEG，周期性地向 container runtime 刷新当前所有容器的状态 = **API Server ← PLEG ← CR**
12. 12. 开始同步循环

```go
// kubernetes/pkg/kubelet/kubelet.go
func (kl *Kubelet) Run(updates <-chan kubetypes.PodUpdate) {
	ctx := context.Background()
	
    // 1. Setup logServer & logClient
    if kl.logServer == nil {
		file := http.FileServer(http.Dir(nodeLogDir))
		if utilfeature.DefaultFeatureGate.Enabled(features.NodeLogQuery) && kl.kubeletConfiguration.EnableSystemLogQuery {
			kl.logServer = http.StripPrefix("/logs/", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if nlq, errs := newNodeLogQuery(req.URL.Query()); len(errs) > 0 {
					http.Error(w, errs.ToAggregate().Error(), http.StatusBadRequest)
					return
				} else if nlq != nil {
					if req.URL.Path != "/" && req.URL.Path != "" {
						http.Error(w, "path not allowed in query mode", http.StatusNotAcceptable)
						return
					}
					if errs := nlq.validate(); len(errs) > 0 {
						http.Error(w, errs.ToAggregate().Error(), http.StatusNotAcceptable)
						return
					}
					if len(nlq.Services) > 0 {
						journal.ServeHTTP(w, req)
						return
					}
					if len(nlq.Files) == 1 {
						// Account for the \ being used on Windows clients
						req.URL.Path = filepath.ToSlash(nlq.Files[0])
					}
				}
				file.ServeHTTP(w, req)
			}))
		} else {
			kl.logServer = http.StripPrefix("/logs/", file)
		}
	}
	if kl.kubeClient == nil {
		klog.InfoS("No API server defined - no node status update will be sent")
	}

    
	// 2. 启动 cloud provider sync manager
    // https://kubernetes.feisky.xyz/extension/cloud-provider
	if kl.cloudResourceSyncManager != nil {
		go kl.cloudResourceSyncManager.Run(wait.NeverStop)
	}

	// 3. 初始化（不依赖 CR 的）模块
    if err := kl.initializeModules(); err != nil {
		kl.recorder.Eventf(kl.nodeRef, v1.EventTypeWarning, events.KubeletSetupFailed, err.Error())
		klog.ErrorS(err, "Failed to initialize internal modules")
		os.Exit(1)
	}

	kl.warnCgroupV1Usage()

	// 4. 启动 volume manager
	go kl.volumeManager.Run(kl.sourcesReady, wait.NeverStop)

	if kl.kubeClient != nil {
		go func() {
			kl.updateRuntimeUp()
			// 5. 定时同步节点状态
            wait.JitterUntil(kl.syncNodeStatus, kl.nodeStatusUpdateFrequency, 0.04, true, wait.NeverStop)
		}()
		
        // 6. 更新 CR 启动时间以及执行首次状态同步
		go kl.fastStatusUpdateOnce()

		// 7. 启动 node lease 类心跳上报 API Server
		go kl.nodeLeaseController.Run(context.Background())
	}
    
    // 8. 定时更新 CR 状态
	go wait.Until(kl.updateRuntimeUp, 5*time.Second, wait.NeverStop)

	// 9. 设置 iptables rule 并执行定时同步
	if kl.makeIPTablesUtilChains {
		kl.initNetworkUtil()
	}

	// 10. 启动 status manager
	kl.statusManager.Start()

	// 11. Start syncing RuntimeClasses if enabled.
	if kl.runtimeClassManager != nil {
		kl.runtimeClassManager.Start(wait.NeverStop)
	}

	// 11. 启动 PLEG - Pod Lifecycle Event Generator
    // 周期从 container runtime 获取当前所有容器的状态上报 API Server
	kl.pleg.Start()

	// 仅当 feature gate 开启启动 evented PLEG
	if utilfeature.DefaultFeatureGate.Enabled(features.EventedPLEG) {
		kl.eventedPleg.Start()
	}

    // 12. 开始同步循环
	kl.syncLoop(ctx, updates, kl)
}
```

### initializeModules

1. 设置 kubelet 数据目录
2. 创建 ContainerLogsDir
3. 启动 image manager → realImageGCManager
4. 启动 certificate manager if it was enabled.
5. 启动 OOM watcher.

```go
// kubernetes/pkg/kubelet/kubelet.go
func (kl *Kubelet) initializeModules() error {
	// Prometheus metrics.
	metrics.Register(
		collectors.NewVolumeStatsCollector(kl),
		collectors.NewLogMetricsCollector(kl.StatsProvider.ListPodStats),
	)
	metrics.SetNodeName(kl.nodeName)
	servermetrics.Register()

	// 1. 设置 kubelet 数据目录
	if err := kl.setupDataDirs(); err != nil {
		return err
	}

	// 2. 创建 ContainerLogsDir
	if _, err := os.Stat(ContainerLogsDir); err != nil {
		if err := kl.os.MkdirAll(ContainerLogsDir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %q: %v", ContainerLogsDir, err)
		}
	}

	// 3. 启动 image manager → realImageGCManager
	kl.imageManager.Start()

	// 4. 启动 certificate manager if it was enabled.
	if kl.serverCertificateManager != nil {
		kl.serverCertificateManager.Start()
	}

	// 5. 启动 OOM watcher.
	if kl.oomWatcher != nil {
		if err := kl.oomWatcher.Start(kl.nodeRef); err != nil {
			return fmt.Errorf("failed to start OOM watcher: %w", err)
		}
	}

	// 6. 启动资源分析器
	kl.resourceAnalyzer.Start()

	return nil
}
```

### realImageGCManager

1. goroutine 检测所有镜像，删除不再使用的 → CRI
2. goroutine 更新镜像缓存 → CRI

```go
// kubernetes/pkg/kubelet/images/image_gc_manager.go
func (im *realImageGCManager) Start() {
	ctx := context.Background()
	
    // 1. gorouine 检测所有镜像，删除不再使用的 → CRI
    go wait.Until(func() {
		_, err := im.detectImages(ctx, time.Now())
		if err != nil {
			klog.InfoS("Failed to monitor images", "err", err)
		}
	}, 5*time.Minute, wait.NeverStop)

	// 2. goroutine 更新镜像缓存 → CRI
	go wait.Until(func() {
		images, err := im.runtime.ListImages(ctx)
		if err != nil {
			klog.InfoS("Failed to update image list", "err", err)
		} else {
			im.imageCache.set(images)
		}
	}, 30*time.Second, wait.NeverStop)

}
```

### fastStatusUpdateOnce

1. 检查当前 kubelet 以及 CR 是否能让节点处于 Ready 状态，并将该状态同步给 API Server

```go
// kubernetes/pkg/kubelet/kubelet.go
func (kl *Kubelet) fastStatusUpdateOnce() {
	ctx := context.Background()
    
	start := kl.clock.Now()
	stopCh := make(chan struct{})

	wait.Until(func() {
		// 1. 检查当前 kubelet 以及 CR 是否能让节点处于 Ready 状态，并将该状态同步给 API Server
        if kl.fastNodeStatusUpdate(ctx, kl.clock.Since(start) >= nodeReadyGracePeriod) {
			close(stopCh)
		}
	}, 100*time.Millisecond, stopCh)
}
```

### updateRuntimeUp

1. 获取 CR 状态
2. 检查 network 是否处于 ready 状态
3. 检查 CR 是否处于 ready 状态
4. 启动依赖模块 - cadvisor、containerManager、evictionManager、containerLogManager、pluginManager
5. 设置 runtime 同步时间

```go
// kubernetes/pkg/kubelet/kubelet.go
func (kl *Kubelet) updateRuntimeUp() {
	kl.updateRuntimeMux.Lock()
	defer kl.updateRuntimeMux.Unlock()
	ctx := context.Background()

    // 1. 获取 CR 状态
	s, err := kl.containerRuntime.Status(ctx)
	if err != nil {
		klog.ErrorS(err, "Container runtime sanity check failed")
		return
	}
	if s == nil {
		klog.ErrorS(nil, "Container runtime status is nil")
		return
	}
    
    // 2. 检查 network 是否处于 ready 状态
	networkReady := s.GetRuntimeCondition(kubecontainer.NetworkReady)
	if networkReady == nil || !networkReady.Status {
		klogErrorS(nil, "Container runtime network not ready", "networkReady", networkReady)
		kl.runtimeState.setNetworkState(fmt.Errorf("container runtime network not ready: %v", networkReady))
	} else {
		// Set nil if the container runtime network is ready.
		kl.runtimeState.setNetworkState(nil)
	}
    
	// 3. 检查 CR 是否处于 ready 状态
	runtimeReady := s.GetRuntimeCondition(kubecontainer.RuntimeReady)
	if runtimeReady == nil || !runtimeReady.Status {
		klogErrorS(nil, "Container runtime not ready", "runtimeReady", runtimeReady)
		kl.runtimeState.setRuntimeState(fmt.Errorf("container runtime not ready: %v", runtimeReady))
		return
	}

	kl.runtimeState.setRuntimeState(nil)
	kl.runtimeState.setRuntimeHandlers(s.Handlers)
	kl.runtimeState.setRuntimeFeatures(s.Features)
    
    // 4. 启动依赖模块 - cadvisor、containerManager、evictionManager、containerLogManager、pluginManager
	kl.oneTimeInitializer.Do(kl.initializeRuntimeDependentModules)
	// 5. 设置 runtime 同步时间
    kl.runtimeState.setRuntimeSync(kl.clock.Now())
}
```

### syncLoop

1. 监听不同来源 (file/http/apiserver) 配置更新事件，handler 处理。
2. 监听 PLEG 事件，PLEG 启动会每秒进行一次 relist 根据最新 PodStatus 生成事件放入该管道中，然后消费。
3. 监听 Housekeeping 事件，每 2s 触发一次

```go
// kubernetes/pkg/kubelet/kubelet.go
func (kl *Kubelet) syncLoop(ctx context.Context, updates <-chan kubetypes.PodUpdate, handler SyncHandler) {
	klog.InfoS("Starting kubelet main sync loop")

	syncTicker := time.NewTicker(time.Second)
	defer syncTicker.Stop()
	housekeepingTicker := time.NewTicker(housekeepingPeriod)
	defer housekeepingTicker.Stop()
	plegCh := kl.pleg.Watch()
	const (
		base   = 100 * time.Millisecond
		max    = 5 * time.Second
		factor = 2
	)
	duration := base

	for {
		if err := kl.runtimeState.runtimeErrors(); err != nil {
			klog.ErrorS(err, "Skipping pod synchronization")
			// exponential backoff
			time.Sleep(duration)
			duration = time.Duration(math.Min(float64(max), factor*float64(duration)))
			continue
		}
		duration = base

		kl.syncLoopMonitor.Store(kl.clock.Now())
        // Entrypoint
		if !kl.syncLoopIteration(ctx, updates, handler, syncTicker.C, housekeepingTicker.C, plegCh) {
			break
		}
		kl.syncLoopMonitor.Store(kl.clock.Now())
	}
}
```

```go
// kubernetes/pkg/kubelet/kubelet.go
func (kl *Kubelet) syncLoopIteration(
    ctx            context.Context,               
    configCh       <-chan kubetypes.PodUpdate,    // a channel to read config events from
    handler        SyncHandler,                   // the SyncHandler to dispatch pods to
	syncCh         <-chan time.Time,              // a channel to read periodic sync events from
    housekeepingCh <-chan time.Time,              // a channel to read housekeeping events from
    plegCh         <-chan *pleg.PodLifecycleEvent // a channel to read PLEG updates from
) bool {
    // 监听事件循环
	select {
    // 1. 监听不同来源 (file/http/apiserver) 配置更新事件，handler 处理
	case u, open := <-configCh:
		if !open {
			klog.ErrorS(nil, "Update channel is closed, exiting the sync loop")
			return false
		}
        
		switch u.Op {
		case kubetypes.ADD:
			klog.V(2).InfoS("SyncLoop ADD", "source", u.Source, "pods", klog.KObjSlice(u.Pods))
			handler.HandlePodAdditions(u.Pods)
		case kubetypes.UPDATE:
			klog.V(2).InfoS("SyncLoop UPDATE", "source", u.Source, "pods", klog.KObjSlice(u.Pods))
			handler.HandlePodUpdates(u.Pods)
		case kubetypes.REMOVE:
			klog.V(2).InfoS("SyncLoop REMOVE", "source", u.Source, "pods", klog.KObjSlice(u.Pods))
			handler.HandlePodRemoves(u.Pods)
		case kubetypes.RECONCILE:
			klog.V(4).InfoS("SyncLoop RECONCILE", "source", u.Source, "pods", klog.KObjSlice(u.Pods))
			handler.HandlePodReconcile(u.Pods)
		case kubetypes.DELETE:
			klog.V(2).InfoS("SyncLoop DELETE", "source", u.Source, "pods", klog.KObjSlice(u.Pods))
			handler.HandlePodUpdates(u.Pods)
		case kubetypes.SET:
			klog.ErrorS(nil, "Kubelet does not support snapshot update")
		default:
			klog.ErrorS(nil, "Invalid operation type received", "operation", u.Op)
		}

		kl.sourcesReady.AddSource(u.Source)
```

```go
	// 2. 监听 PLEG 事件，PLEG 启动会每秒进行一次 relist 根据最新 PodStatus 生成事件放入该管道中
	case e := <-plegCh:
		if isSyncPodWorthy(e) {
			if pod, ok := kl.podManager.GetPodByUID(e.ID); ok {
				klog.V(2).InfoS("SyncLoop (PLEG): event for pod", "pod", klog.KObj(pod), "event", e)
				handler.HandlePodSyncs([]*v1.Pod{pod})
			} else {
				klog.V(4).InfoS("SyncLoop (PLEG): pod does not exist, ignore irrelevant event", "event", e)
			}
		}

		if e.Type == pleg.ContainerDied {
			if containerID, ok := e.Data.(string); ok {
				kl.cleanUpContainersInPod(e.ID, containerID)
			}
		}
```

```go
	// 3. 监听 Housekeeping 事件，每 2s 触发一次
	case <-housekeepingCh:
		if !kl.sourcesReady.AllReady() {
			// If the sources aren't ready or volume manager has not yet synced the states,
			// skip housekeeping, as we may accidentally delete pods from unready sources.
			klog.V(4).InfoS("SyncLoop (housekeeping, skipped): sources aren't ready yet")
		} else {
			start := time.Now()
			klog.V(4).InfoS("SyncLoop (housekeeping)")
			if err := handler.HandlePodCleanups(ctx); err != nil {
				klog.ErrorS(err, "Failed cleaning pods")
			}
			duration := time.Since(start)
			if duration > housekeepingWarningDuration {
				klog.ErrorS(fmt.Errorf("housekeeping took too long"), "Housekeeping took longer than expected", "expected", housekeepingWarningDuration, "actual", duration.Round(time.Millisecond))
			}
			klog.V(4).InfoS("SyncLoop (housekeeping) end", "duration", duration.Round(time.Millisecond))
		}
	}
```

```go
	// 4. 监听 pod 同步事件，需要同步的有：处于 ready 状态 + 内部 module 请求
	case <-syncCh:
		// Sync pods waiting for sync
		podsToSync := kl.getPodsToSync()
		if len(podsToSync) == 0 {
			break
		}
		klog.V(4).InfoS("SyncLoop (SYNC) pods", "total", len(podsToSync), "pods", klog.KObjSlice(podsToSync))
		handler.HandlePodSyncs(podsToSync)
	
	// 5. 监听 *Manager 更新事件 livenessManager/readinessManager/startupManager/containerManager
	case update := <-kl.livenessManager.Updates():
		if update.Result == proberesults.Failure {
			handleProbeSync(kl, update, handler, "liveness", "unhealthy")
		}
	case update := <-kl.readinessManager.Updates():
		ready := update.Result == proberesults.Success
		kl.statusManager.SetContainerReadiness(update.PodUID, update.ContainerID, ready)

		status := ""
		if ready {
			status = "ready"
		}
		handleProbeSync(kl, update, handler, "readiness", status)
	case update := <-kl.startupManager.Updates():
		started := update.Result == proberesults.Success
		kl.statusManager.SetContainerStartup(update.PodUID, update.ContainerID, started)

		status := "unhealthy"
		if started {
			status = "started"
		}
		handleProbeSync(kl, update, handler, "startup", status)
	case update := <-kl.containerManager.Updates():
		pods := []*v1.Pod{}
		for _, p := range update.PodUIDs {
			if pod, ok := kl.podManager.GetPodByUID(types.UID(p)); ok {
				klog.V(3).InfoS("SyncLoop (containermanager): event for pod", "pod", klog.KObj(pod), "event", update)
				pods = append(pods, pod)
			} else {
				// If the pod no longer exists, ignore the event.
				klog.V(4).InfoS("SyncLoop (containermanager): pod does not exist, ignore devices updates", "event", update)
			}
		}
		if len(pods) > 0 {
			handler.HandlePodSyncs(pods)
		}
```

#### HandlePodAdditions

`<-configCh` in syncLoop

1. 按 creation time 给 pod 排序
2. 添加 pod 到 pod manager；若 pod manager 中存在镜像 pod，更新 pod 的状态信息
3. 若 pod 没有被 terminated，过滤获取 active pod，验证 pod 是否在该节点上运行
4. 更新 pod 状态

```go
// kubernetes/pkg/kubelet/kubelet.go
func (kl *Kubelet) HandlePodAdditions(pods []*v1.Pod) {
	// 1. sort pod by creation time
    start := kl.clock.Now()
	sort.Sort(sliceutils.PodsByCreationTime(pods))
	if utilfeature.DefaultFeatureGate.Enabled(features.InPlacePodVerticalScaling) {
		kl.podResizeMutex.Lock()
		defer kl.podResizeMutex.Unlock()
	}
    
	for _, pod := range pods {
		existingPods := kl.podManager.GetPods()
		
        // 1. 添加 pod 到 pod manager
		kl.podManager.AddPod(pod)

        // 1. 若 pod manager 中存在镜像 pod，更新 pod 的状态信息
		pod, mirrorPod, wasMirror := kl.podManager.GetPodAndMirrorPod(pod)
		if wasMirror {
			if pod == nil {
				klog.V(2).InfoS("Unable to find pod for mirror pod, skipping", "mirrorPod", klog.KObj(mirrorPod), "mirrorPodUID", mirrorPod.UID)
				continue
			}
			kl.podWorkers.UpdatePod(UpdatePodOptions{
				Pod:        pod,
				MirrorPod:  mirrorPod,
				UpdateType: kubetypes.SyncPodUpdate,
				StartTime:  start,
			})
			continue
		}

        // 2. 若 pod 没有被 terminated
		if !kl.podWorkers.IsPodTerminationRequested(pod.UID) {
			// 2. 过滤获取 active pod
            activePods := kl.filterOutInactivePods(existingPods)
			if utilfeature.DefaultFeatureGate.Enabled(features.InPlacePodVerticalScaling) {
				podCopy := pod.DeepCopy()
				kl.updateContainerResourceAllocation(podCopy)

				if ok, reason, message := kl.canAdmitPod(activePods, podCopy); !ok {
					kl.rejectPod(pod, reason, message)
					continue
				}
				if err := kl.statusManager.SetPodAllocation(podCopy); err != nil {
					klog.ErrorS(err, "SetPodAllocation failed", "pod", klog.KObj(pod))
				}
			} else {
				// 2. 验证 pod 是否在该节点上运行
                if ok, reason, message := kl.canAdmitPod(activePods, pod); !ok {
					kl.rejectPod(pod, reason, message)
					continue
				}
			}
		}
        // 3. 更新 pod 状态
		kl.podWorkers.UpdatePod(UpdatePodOptions{
			Pod:        pod,
			MirrorPod:  mirrorPod,
			UpdateType: kubetypes.SyncPodCreate,
			StartTime:  start,
		})
	}
}
```

#### UpdatePod

```go
// kubernetes/pkg/kubelet/pod_worker.go
func (p *podWorkers) UpdatePod(options UpdatePodOptions) {
	...

	p.podLock.Lock()
	defer p.podLock.Unlock()

	...

    // 1. 根据 pod uid 获取 podUpdates 中的更新数据；若不存在，则创建一个 channel 启动 goroutine 尝试获取更新
	podUpdates, exists := p.podUpdates[uid]
	if !exists {

		podUpdates = make(chan struct{}, 1)
		p.podUpdates[uid] = podUpdates

		if kubetypes.IsStaticPod(pod) {
			p.waitingToStartStaticPodsByFullname[status.fullname] =
				append(p.waitingToStartStaticPodsByFullname[status.fullname], uid)
		}

		var outCh <-chan struct{}
		if p.workerChannelFn != nil {
			outCh = p.workerChannelFn(uid, podUpdates)
		} else {
			outCh = podUpdates
		}

		go func() {
			defer runtime.HandleCrash()
			defer klog.V(3).InfoS("Pod worker has stopped", "podUID", uid)
			p.podWorkerLoop(uid, outCh)
		}()
	}

	...

	// notify the pod worker there is a pending update
	status.pendingUpdate = &options
	status.working = true
	klog.V(4).InfoS("Notifying pod of pending update", "pod", klog.KRef(ns, name), "podUID", uid, "workType", status.WorkType())
	select {
	case podUpdates <- struct{}{}:
	default:
	}

	...
}
```

#### podWorkerLoop

遍历 podUpdates 调用 →？ 

```go
// kubernetes/pkg/kubelet/pod_worker.go
func (p *podWorkers) podWorkerLoop(podUID types.UID, podUpdates <-chan struct{}) {
	var lastSyncTime time.Time
    
    // iter chan
	for range podUpdates {
        ctx, update, canStart, canEverStart, ok := p.startPodSync(podUID)
		if !ok {
			continue
		}
		if !canEverStart {
			return
		}
		if !canStart {
			continue
		}

		podUID, podRef := podUIDAndRefForUpdate(update.Options)

		klog.V(4).InfoS("Processing pod event", "pod", podRef, "podUID", podUID, "updateType", update.WorkType)
		var isTerminal bool
        
        // 
		err := func() error {
			var status *kubecontainer.PodStatus
			var err error
			switch {
			case update.Options.RunningPod != nil:
			default:
                // 阻塞直到 cache 中有新数据，保证 worker 在 cache 更新前不会提前开始
				status, err = p.podCache.GetNewerThan(update.Options.Pod.UID, lastSyncTime)

				if err != nil {
					p.recorder.Eventf(update.Options.Pod, v1.EventTypeWarning, events.FailedSync, "error determining status: %v", err)
					return err
				}
			}

			...

			lastSyncTime = p.clock.Now()
			return err
		}()

		...
}

```

#### SyncPod

1. 生成 Pod Status API
2. 设置 pod IP & host IP
3. 设置 pod 状态
4. 检查网络插件是否 ready
5. 创建 pod cgroup
6. 为静态 pod 创建镜像
7. 创建 pod 数据目录
8. **等待 vol attach & mount**
9. 获取 image pull secret
10. 将 pod 纳入 probe manager 管理
11. **调用 CRI 创建 pod**

```go
// kubernetes/pkg/kubelet/kubelet.go
func (kl *Kubelet) SyncPod(
    ctx context.Context, 
    updateType kubetypes.SyncPodType, 
    pod, 
    mirrorPod *v1.Pod, 
    podStatus *kubecontainer.PodStatus
) (isTerminal bool, err error) {
	...

	// 1. 生成 Pod Status API
	apiPodStatus := kl.generateAPIPodStatus(pod, podStatus, false)
    
	// 2. 设置 pod IP & host IP
	podStatus.IPs = make([]string, 0, len(apiPodStatus.PodIPs))
	for _, ipInfo := range apiPodStatus.PodIPs {
		podStatus.IPs = append(podStatus.IPs, ipInfo.IP)
	}
	if len(podStatus.IPs) == 0 && len(apiPodStatus.PodIP) > 0 {
		podStatus.IPs = []string{apiPodStatus.PodIP}
	}

	...

 	// 3. 设置 pod 状态
	kl.statusManager.SetPodStatus(pod, apiPodStatus)

	// 4. 检查网络插件是否 ready
	if err := kl.runtimeState.networkErrors(); err != nil && !kubecontainer.IsHostNetworkPod(pod) {
		kl.recorder.Eventf(pod, v1.EventTypeWarning, events.NetworkNotReady, "%s: %v", NetworkNotReadyErrorMsg, err)
		return false, fmt.Errorf("%s: %v", NetworkNotReadyErrorMsg, err)
	}

	...

	// 5. 创建 pod cgroup
	pcm := kl.containerManager.NewPodContainerManager()
	if !kl.podWorkers.IsPodTerminationRequested(pod.UID) {
		firstSync := true
		// 判断该 pod 是否首次创建
        for _, containerStatus := range apiPodStatus.ContainerStatuses {
			if containerStatus.State.Running != nil {
				firstSync = false
				break
			}
		}
		// 若 cgrou 已存在 or pod 第一次运行，不要杀掉 pod 中的容器
		podKilled := false
		if !pcm.Exists(pod) && !firstSync {
			p := kubecontainer.ConvertPodStatusToRunningPod(kl.getRuntime().Type(), podStatus)
			if err := kl.killPod(ctx, pod, p, nil); err == nil {
				podKilled = true
			} else {
				if wait.Interrupted(err) {
					return false, nil
				}
				klog.ErrorS(err, "KillPod failed", "pod", klog.KObj(pod), "podStatus", podStatus)
			}
		}
		
        // 若 pod 未被删掉，且重启策略非 never
		if !(podKilled && pod.Spec.RestartPolicy == v1.RestartPolicyNever) {
			if !pcm.Exists(pod) {
				// 创建 cgroup
                if err := kl.containerManager.UpdateQOSCgroups(); err != nil {
					klog.V(2).InfoS("Failed to update QoS cgroups while syncing pod", "pod", klog.KObj(pod), "err", err)
				}
				if err := pcm.EnsureExists(pod); err != nil {
					kl.recorder.Eventf(pod, v1.EventTypeWarning, events.FailedToCreatePodContainer, "unable to ensure pod container exists: %v", err)
					return false, fmt.Errorf("failed to ensure that the pod: %v cgroups exist and are correctly applied: %v", pod.UID, err)
				}
			}
		}
	}

	// 6. 为静态 pod 创建镜像
	if kubetypes.IsStaticPod(pod) {
		deleted := false
		if mirrorPod != nil {
			if mirrorPod.DeletionTimestamp != nil || !kubepod.IsMirrorPodOf(mirrorPod, pod) {
				klog.InfoS("Trying to delete pod", "pod", klog.KObj(pod), "podUID", mirrorPod.ObjectMeta.UID)
				podFullName := kubecontainer.GetPodFullName(pod)
				var err error
				deleted, err = kl.mirrorPodClient.DeleteMirrorPod(podFullName, &mirrorPod.ObjectMeta.UID)
				if deleted {
					klog.InfoS("Deleted mirror pod because it is outdated", "pod", klog.KObj(mirrorPod))
				} else if err != nil {
					klog.ErrorS(err, "Failed deleting mirror pod", "pod", klog.KObj(mirrorPod))
				}
			}
		}
		if mirrorPod == nil || deleted {
			node, err := kl.GetNode()
			if err != nil {
				klog.V(4).ErrorS(err, "No need to create a mirror pod, since failed to get node info from the cluster", "node", klog.KRef("", string(kl.nodeName)))
			} else if node.DeletionTimestamp != nil {
				klog.V(4).InfoS("No need to create a mirror pod, since node has been removed from the cluster", "node", klog.KRef("", string(kl.nodeName)))
			} else {
				klog.V(4).InfoS("Creating a mirror pod for static pod", "pod", klog.KObj(pod))
				if err := kl.mirrorPodClient.CreateMirrorPod(pod); err != nil {
					klog.ErrorS(err, "Failed creating a mirror pod for", "pod", klog.KObj(pod))
				}
			}
		}
	}

	// 7. 创建 pod 数据目录
	if err := kl.makePodDataDirs(pod); err != nil {
		kl.recorder.Eventf(pod, v1.EventTypeWarning, events.FailedToMakePodDataDirectories, "error making pod data directories: %v", err)
		klog.ErrorS(err, "Unable to make pod data directories for pod", "pod", klog.KObj(pod))
		return false, err
	}

	// 8. 等待 vol attach & mount
	if err := kl.volumeManager.WaitForAttachAndMount(ctx, pod); err != nil {
		if !wait.Interrupted(err) {
			kl.recorder.Eventf(pod, v1.EventTypeWarning, events.FailedMountVolume, "Unable to attach or mount volumes: %v", err)
			klog.ErrorS(err, "Unable to attach or mount volumes for pod; skipping pod", "pod", klog.KObj(pod))
		}
		return false, err
	}

	// 9. 获取 image pull secret
	pullSecrets := kl.getPullSecretsForPod(pod)

	// 10. 将 pod 纳入 probe manager 管理
	kl.probeManager.AddPod(pod)

	...

	// 11. 调用 CRI 创建 pod
	sctx := context.WithoutCancel(ctx)
	result := kl.containerRuntime.SyncPod(sctx, pod, podStatus, pullSecrets, kl.backOff)
	kl.reasonCache.Update(pod.UID, result)
	if err := result.Error(); err != nil {

		for _, r := range result.SyncResults {
			if r.Error != kubecontainer.ErrCrashLoopBackOff && r.Error != images.ErrImagePullBackOff {
				return false, err
			}
		}

		return false, nil
	}

	...

	return false, nil
}
```

#### syncPod

1. 计算哪些 pod 中 container 需要创建/删除
2. 若 sandbox 变化则杀掉 pod
3. sandbox 无变化，遍历需要杀掉的容器列表
4. 清理同名 init
5. 创建 sandbox
6. 获取 sandbox conf 以便容器启动
7. 启动临时容器 prior to init 用于调试和诊断目的，即使 init 出现错误依旧可以运行
8. 依次启动 init 容器
9. 判断是否需要重新调整 container cpu/mem
10. 创建并启动正常容器

```go
// pkg/kubelet/kuberuntime/kuberuntime_manager.go

func (m *kubeGenericRuntimeManager) SyncPod(ctx context.Context, pod *v1.Pod, podStatus *kubecontainer.PodStatus, pullSecrets []v1.Secret, backOff *flowcontrol.Backoff) (result kubecontainer.PodSyncResult) {
	// 1. 计算哪些 pod 中 container 需要创建/删除
	podContainerChanges := m.computePodActions(ctx, pod, podStatus)
	... 

    // 2. 若 sandbox 变化则杀掉 pod
	if podContainerChanges.KillPod {
		if podContainerChanges.CreateSandbox {
			klog.V(4).InfoS("Stopping PodSandbox for pod, will start new one", "pod", klog.KObj(pod))
		} else {
			klog.V(4).InfoS("Stopping PodSandbox for pod, because all other containers are dead", "pod", klog.KObj(pod))
		}

		killResult := m.killPodWithSyncResult(ctx, pod, kubecontainer.ConvertPodStatusToRunningPod(m.runtimeName, podStatus), nil)
		result.AddPodSyncResult(killResult)
		if killResult.Error() != nil {
			klog.ErrorS(killResult.Error(), "killPodWithSyncResult failed")
			return
		}

		if podContainerChanges.CreateSandbox {
			m.purgeInitContainers(ctx, pod, podStatus)
		}
	} else {
        // 3. sandbox 无变化，遍历需要杀掉的容器列表
		for containerID, containerInfo := range podContainerChanges.ContainersToKill {
			klog.V(3).InfoS("Killing unwanted container for pod", "containerName", containerInfo.name, "containerID", containerID, "pod", klog.KObj(pod))
			killContainerResult := kubecontainer.NewSyncResult(kubecontainer.KillContainer, containerInfo.name)
			result.AddSyncResult(killContainerResult)
			if err := m.killContainer(ctx, pod, containerID, containerInfo.name, containerInfo.message, containerInfo.reason, nil, nil); err != nil {
				killContainerResult.Fail(kubecontainer.ErrKillContainer, err.Error())
				klog.ErrorS(err, "killContainer for pod failed", "containerName", containerInfo.name, "containerID", containerID, "pod", klog.KObj(pod))
				return
			}
		}
	}

	// 4. 清理同名 init
	m.pruneInitContainersBeforeStart(ctx, pod, podStatus)

	...

	// 5. 创建 sandbox
	podSandboxID := podContainerChanges.SandboxID
	if podContainerChanges.CreateSandbox {
		var msg string
		var err error

		klog.V(4).InfoS("Creating PodSandbox for pod", "pod", klog.KObj(pod))
		metrics.StartedPodsTotal.Inc()
		createSandboxResult := kubecontainer.NewSyncResult(kubecontainer.CreatePodSandbox, format.Pod(pod))
		result.AddSyncResult(createSandboxResult)

		sysctl.ConvertPodSysctlsVariableToDotsSeparator(pod.Spec.SecurityContext)

		...

		podSandboxID, msg, err = m.createPodSandbox(ctx, pod, podContainerChanges.Attempt)
		
        ...

		resp, err := m.runtimeService.PodSandboxStatus(ctx, podSandboxID, false)
		
        ...

	}

	...

    // 6. 获取 sandbox conf 以便容器启动
	configPodSandboxResult := kubecontainer.NewSyncResult(kubecontainer.ConfigPodSandbox, podSandboxID)
	result.AddSyncResult(configPodSandboxResult)
	podSandboxConfig, err := m.generatePodSandboxConfig(pod, podContainerChanges.Attempt)
	
    ...

	imageVolumePullResults, err := m.getImageVolumes(ctx, pod, podSandboxConfig, pullSecrets)
	
    ...

	}

	// 7. 启动临时容器 prior to init 用于调试和诊断目的，即使 init 出现错误依旧可以运行
	for _, idx := range podContainerChanges.EphemeralContainersToStart {
		start(ctx, "ephemeral container", metrics.EphemeralContainer, ephemeralContainerStartSpec(&pod.Spec.EphemeralContainers[idx]))
	}

	if !utilfeature.DefaultFeatureGate.Enabled(features.SidecarContainers) {
		// 8. 依次启动 init 容器
		if container := podContainerChanges.NextInitContainerToStart; container != nil {
			if err := start(ctx, "init container", metrics.InitContainer, containerStartSpec(container)); err != nil {
				return
			}

			klog.V(4).InfoS("Completed init container for pod", "containerName", container.Name, "pod", klog.KObj(pod))
		}
	} else {
		for _, idx := range podContainerChanges.InitContainersToStart {
			container := &pod.Spec.InitContainers[idx]
			if err := start(ctx, "init container", metrics.InitContainer, containerStartSpec(container)); err != nil {
				if types.IsRestartableInitContainer(container) {
					klog.V(4).InfoS("Failed to start the restartable init container for the pod, skipping", "initContainerName", container.Name, "pod", klog.KObj(pod))
					continue
				}
				klog.V(4).InfoS("Failed to initialize the pod, as the init container failed to start, aborting", "initContainerName", container.Name, "pod", klog.KObj(pod))
				return
			}

			klog.V(4).InfoS("Completed init container for pod", "containerName", container.Name, "pod", klog.KObj(pod))
		}
	}

	// 9. 判断是否需要重新调整 container cpu/mem
	if isInPlacePodVerticalScalingAllowed(pod) {
		if len(podContainerChanges.ContainersToUpdate) > 0 || podContainerChanges.UpdatePodResources {
			m.doPodResizeAction(pod, podStatus, podContainerChanges, result)
		}
	}

	// 10. 创建并启动正常容器
	for _, idx := range podContainerChanges.ContainersToStart {
		start(ctx, "container", metrics.Container, containerStartSpec(&pod.Spec.Containers[idx]))
	}

	return
}
```

#### startContainer

1. 拉取镜像
2. 获取容器状态
3. 生成容器配置
4. 创建容器获取 ID
5. 启动容器
6. 执行 postStart hook

```go
// pkg/kubelet/kuberuntime/kuberuntime_container.go

func (m *kubeGenericRuntimeManager) startContainer(ctx context.Context, podSandboxID string, podSandboxConfig *runtimeapi.PodSandboxConfig, spec *startSpec, pod *v1.Pod, podStatus *kubecontainer.PodStatus, pullSecrets []v1.Secret, podIP string, podIPs []string, imageVolumes kubecontainer.ImageVolumes) (string, error) {
	container := spec.container

	// 1. 拉取镜像
	podRuntimeHandler, err := m.getPodRuntimeHandler(pod)
    ...
	ref, err := kubecontainer.GenerateContainerRef(pod, container)
    ...
	imageRef, msg, err := m.imagePuller.EnsureImageExists(ctx, ref, pod, container.Image, pullSecrets, podSandboxConfig, podRuntimeHandler, container.ImagePullPolicy)
    ...


    // 2. 获取容器状态
	restartCount := 0
	containerStatus := podStatus.FindContainerStatusByName(container.Name)
	if containerStatus != nil {
		restartCount = containerStatus.RestartCount + 1
	} else {
		logDir := BuildContainerLogsDirectory(m.podLogsDirectory, pod.Namespace, pod.Name, pod.UID, container.Name)
		restartCount, err = calcRestartCountByLogDir(logDir)
		if err != nil {
			klog.InfoS("Cannot calculate restartCount from the log directory", "logDir", logDir, "err", err)
			restartCount = 0
		}
	}

	target, err := spec.getTargetID(podStatus)
	if err != nil {
		s, _ := grpcstatus.FromError(err)
		m.recordContainerEvent(pod, container, "", v1.EventTypeWarning, events.FailedToCreateContainer, "Error: %v", s.Message())
		return s.Message(), ErrCreateContainerConfig
	}

    // 3. 生成容器配置
	containerConfig, cleanupAction, err := m.generateContainerConfig(ctx, container, pod, restartCount, podIP, imageRef, podIPs, target, imageVolumes)
	if cleanupAction != nil {
		defer cleanupAction()
	}
	if err != nil {
		s, _ := grpcstatus.FromError(err)
		m.recordContainerEvent(pod, container, "", v1.EventTypeWarning, events.FailedToCreateContainer, "Error: %v", s.Message())
		return s.Message(), ErrCreateContainerConfig
	}

	err = m.internalLifecycle.PreCreateContainer(pod, container, containerConfig)
	if err != nil {
		s, _ := grpcstatus.FromError(err)
		m.recordContainerEvent(pod, container, "", v1.EventTypeWarning, events.FailedToCreateContainer, "Internal PreCreateContainer hook failed: %v", s.Message())
		return s.Message(), ErrPreCreateHook
	}

    // 4. 创建容器获取 ID
	containerID, err := m.runtimeService.CreateContainer(ctx, podSandboxID, containerConfig, podSandboxConfig)
	if err != nil {
		s, _ := grpcstatus.FromError(err)
		m.recordContainerEvent(pod, container, containerID, v1.EventTypeWarning, events.FailedToCreateContainer, "Error: %v", s.Message())
		return s.Message(), ErrCreateContainer
	}
	err = m.internalLifecycle.PreStartContainer(pod, container, containerID)
	if err != nil {
		s, _ := grpcstatus.FromError(err)
		m.recordContainerEvent(pod, container, containerID, v1.EventTypeWarning, events.FailedToStartContainer, "Internal PreStartContainer hook failed: %v", s.Message())
		return s.Message(), ErrPreStartHook
	}
	m.recordContainerEvent(pod, container, containerID, v1.EventTypeNormal, events.CreatedContainer, fmt.Sprintf("Created container %s", container.Name))

	// 5. 启动容器
	err = m.runtimeService.StartContainer(ctx, containerID)
	if err != nil {
		s, _ := grpcstatus.FromError(err)
		m.recordContainerEvent(pod, container, containerID, v1.EventTypeWarning, events.FailedToStartContainer, "Error: %v", s.Message())
		return s.Message(), kubecontainer.ErrRunContainer
	}
	m.recordContainerEvent(pod, container, containerID, v1.EventTypeNormal, events.StartedContainer, fmt.Sprintf("Started container %s", container.Name))

	...

	// 6. postStart hook
	if container.Lifecycle != nil && container.Lifecycle.PostStart != nil {
		kubeContainerID := kubecontainer.ContainerID{
			Type: m.runtimeName,
			ID:   containerID,
		}
		msg, handlerErr := m.runner.Run(ctx, kubeContainerID, pod, container, container.Lifecycle.PostStart)
		if handlerErr != nil {
			klog.ErrorS(handlerErr, "Failed to execute PostStartHook", "pod", klog.KObj(pod),
				"podUID", pod.UID, "containerName", container.Name, "containerID", kubeContainerID.String())
			// do not record the message in the event so that secrets won't leak from the server.
			m.recordContainerEvent(pod, container, kubeContainerID.ID, v1.EventTypeWarning, events.FailedPostStartHook, "PostStartHook failed")
			if err := m.killContainer(ctx, pod, kubeContainerID, container.Name, "FailedPostStartHook", reasonFailedPostStartHook, nil, nil); err != nil {
				klog.ErrorS(err, "Failed to kill container", "pod", klog.KObj(pod),
					"podUID", pod.UID, "containerName", container.Name, "containerID", kubeContainerID.String())
			}
			return msg, ErrPostStartHook
		}
	}

	return "", nil
}
```


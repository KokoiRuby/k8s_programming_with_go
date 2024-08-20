## Client-side Apply

`kubectl apply `是一种声明式 K8S 对象管理方式，也是最常用的部署/升级方式之一。

其意向表达是一种“期望” = “我”管理的字段应该和 apply 的配置文件一致，其他不在乎。

需要一种标识，追踪哪些字段是由 `kubectl apply` 管理的，该标识就以注解形式存在 `last-applied-configuration`。

全量记录本次 `kubectl apply` 的配置。

```yaml
metadata:
  annotations:
	kubectl.kubernetes.io/last-applied-configuration: |
        {
          "apiVersion": "v1",
          "kind": "Pod",
          "metadata": {
            "annotations": {},
            "creationTimestamp": null,
            "labels": {
              "run": "hello"
            },
            "name": "hello",
            "namespace": "default"
          },
          "spec": {
            "containers": [
              {
                "image": "nginx",
                "name": "hello",
                "resources": {}
              }
            ],
            "dnsPolicy": "ClusterFirst",
            "restartPolicy": "Always"
          },
          "status": {}
        }
```

CSA 原理：

1. 当对象不存在，则创建该对象。
2. 若对象已存在，根据以下**计算出 patch 发送增量 Delta 给 API Server**：
   - 对象最新状态（先 Get 一次）
   - 当前 kubectl apply 配置
   - 上次 kubectl apply 配置

基于 [strategic merge patch](https://kubernetes.io/docs/tasks/manage-kubernetes-objects/declarative-config/#how-apply-calculates-differences-and-merges-changes) 转为 K8s 资源更新的 JSON 合并策略。

## Server-side Apply

Since v1.22。

**SSA** 将对象合并的逻辑转移到了 APIServer，kubectl 只需提交即可 `--server-side`。

默认不显示，查看 `-o yaml --show-managed-fields`。

```yaml
metadata:
  managedFields:
  - apiVersion: v1
    fieldsType: FieldsV1
    fieldsV1:
      f:metadata:
        f:annotations:
          .: {}
          f:kubectl.kubernetes.io/last-applied-configuration: {}
        f:labels:
          .: {}
          f:run: {}
```

**字段管理 Field Management**：当一个字段值改变时，所有权从一个 Manager 变更都施加变更的 Manager 上。

**冲突**：如果尝试更新配置，发现字段拥有不同的值且由其他 Manager 管理，API Server 会进行告警。

**冲突解决**：

- 覆盖，更新修改字段的 Manager 为自身 --force
- 不覆盖，本次配置中，将冲突的字段删除，执行配置更新
- 不覆盖，成为共享 Manager，将冲突字段改成一致

SSA [优化](https://kubernetes.io/docs/reference/using-api/server-side-apply/#merge-strategy) strategic merge patch

```go
// 表明 service.spec.ports 这个数组由 ports.port 和 ports.protocol 组合值来确定唯一性
type ServiceSpec struct {
    // +listType=map
	// +listMapKey=port
	// +listMapKey=protocol
    Ports []ServicePort
}
```

```yaml
# client1
spec:
  ports:
  - name: 5678-8080
    port: 5678
    protocol: TCP
    targetPort: 8080
    
# client2, conflict
# 如果 force，那么 ports 会出现两条记录，分别属于不同的 Manager
spec:
  ports:
  - name: 5679-9999
    port: 5678
    protocol: TCP
    targetPort: 9999
```

:smile:

- 简化客户端逻辑
- 细粒度管理每个字段的 ownership
- 更精准的 dry-run
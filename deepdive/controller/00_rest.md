## What

REST **RE**presentational **S**tate **T**ransfer **表述状态转移**，一种设计 web 服务的架构手段/方式。

**资源表述**：客户端/服务端之间以固定格式传输 HTML/XML/JSON/JPEG/MPX

**Constraints**

1. Uniform Interface 统一接口，服务器将消息转换成统一格式。
   - 请求中标识资源
   - 客户端能获取足够资源信息以便随时能进行修改删除，服务器通过发送自描述 meta 让客户端知晓该如何操作
   - 服务器发送 hyperlink 让客户端知晓额外需要完成一项 task 所需的资源信息
2. Stateless 无状态，请求之间相互隔离。
3. Cache 缓存，提高响应速度 & 降低网络带宽。
4. Code on demand 按需代码，通过传输代码，服务器支持临时扩展自定义客户端的功能
5. Client-Server 客户端/服务端架构分离。

![img](https://media.licdn.com/dms/image/v2/C4E12AQGyRx8VLv2k7Q/article-cover_image-shrink_720_1280/article-cover_image-shrink_720_1280/0/1638073050249?e=1729123200&v=beta&t=G5TQd5JgssD9P9lIYoTvYUlmFemC-DctdQ3kttIIldA)

## REST over HTTP

REST 架构并不依赖于任何底层协议，目前主流基于 HTTP，使用标准 HTTP Verb 操作资源。

客户端获取 URL `example.com/orders` 信息便可操作资源

```bash
# create an order
curl -X POST example.com/orders -d '{"orderValue":9.9,"productId":2,"quantity":3}'

# update an order
curl -X PUT example.com/orders/1 -d '{"orderValue":100,"productId":1,"quantity":1}'
```

**Web API 成熟度**：目前只达到了 Level 2；Level 3 具有更强的自描述能力，只需对 URI 进行 GET 即可。

| Level | Desc                                                         |
| ----- | ------------------------------------------------------------ |
| 0     | Define one URI, and all operations are POST requests to this URI. |
| 1     | Create separate URIs for individual resources.               |
| 2     | Use HTTP methods to define operations on resources.          |
| 3     | Use hypermedia (HATEOAS)                                     |

```json
{
    "orderID":3,
    "productID":2,
    "quantity":4,
    "orderValue":16.60,
    "links": [
        {"rel":"product","href":"https://example.com/customers/3", "action":"GET" },
        {"rel":"product","href":"https://example.com/customers/3", "action":"PUT" }
    ]
}
```

## URI

Google Style

```bash
# https://<service>.googleapis.com/(<service>/)<version>/{resource/path}

# users/{userId}/profile
https://gmail.googleapis.com/gmail/v1/users/{userId}/profile

#users/{userId}/messages/{id}
https://gmail.googleapis.com/gmail/v1/users/{userId}/messages/{id}

# users/{userId}/labels/{id}
https://gmail.googleapis.com/gmail/v1/users/{userId}/labels/{id}

```

K8s API 沿根目录，可遍历出所有的资源

```bash
$ kubectl proxy
$ curl http://127.0.0.1:8001
$ curl http://127.0.0.1:8001/apis/autoscaling/v2
```

## Method

[RFC 9110](https://www.rfc-editor.org/rfc/rfc9110) & [RFC 5789](https://www.rfc-editor.org/rfc/rfc5789)

对于 POST/PUT/DELETE，如果处理时间过长需异步处理，可以返回 202 (Accepted)。

对于 PUT/PATCH/DELETE，如果响应 Body 为空，建议返回 204 (No Content)。

| Method  | Description                                                  | Success Status Code                                         | Is Idempotent |
| ------- | ------------------------------------------------------------ | ----------------------------------------------------------- | ------------- |
| GET     | Get the resource or List a resource collection               | 200-OK                                                      | True          |
| HEAD    | Return metadata of an object for a GET response.             | 200-OK                                                      | True          |
| OPTIONS | Get information about a request                              | 200-OK                                                      | True          |
| POST    | Create a new object based on the data provided, or submit a command | 201-Created with URL of created resource, 200-OK for Action | False         |
| PUT     | Replace an object, or create a named object, when applicable | 200-OK, 201-Created, 204-No Content                         | True          |
| PATCH   | Apply a partial update to an object                          | 200-OK, 204-No Content                                      | False         |
| DELETE  | Delete an object                                             | 200-OK, 204-No Content                                      | True          |

### Get

```bash
# get specific 
$ curl -X GET /resources/{name}?p1=v1&p2=v2&p3=v3
# get collection of 
$ curl -X GET /resources?p1=v1&p2=v2&p3=v3
```

```bash
# - to match all
$ curl -X GET https://library.googleapis.com/v1/shelves/-/books?filter=xxx
$ curl -X GET https://library.googleapis.com/v1/shelves/-/books/{id}
```

**Paginating**： 

--offset, ++page token，由服务端驱动分页

```bash
$ curl -X GET /calendars/primary/events?maxResults=10

# server returns token
{... "nextPageToken": "token" ...}

$ curl -X GET /calendars/primary/events?maxResults=10&pageToken=token
```

K8s limit + continue + remainingItemCount

```bash
$ kubectl get --raw '/api/v1/configmaps?limit=5'

# server returns continue
{... continue: "continue", remainingItemCount: X ...}

$ kubectl get --raw '/api/v1/configmaps?limit=5&continue=continue'
```

**Filtering**

| Operator   | Description           | Example                                               |
| ---------- | --------------------- | ----------------------------------------------------- |
| Comparison |                       |                                                       |
| eq         | Equal                 | city eq ‘Redmond’                                     |
| ne         | Not equal             | city ne ‘London’                                      |
| gt         | Greater than          | price gt 20                                           |
| ge         | Greater than or equal | price ge 10                                           |
| lt         | Less than             | price lt 20                                           |
| le         | Less than or equal    | price le 100                                          |
| Logical    |                       |                                                       |
| and        | Logical and           | price le 200 and price gt 3.5                         |
| or         | Logical or            | price le 3.5 or price gt 200                          |
| not        | Logical negation      | not price le 3.5                                      |
| Grouping   |                       |                                                       |
| ( )        | Precedence grouping   | (priority eq 1 or city eq ‘Redmond’) and price gt 100 |

```bash
$ curl -X GET https://api.contoso.com/v1.0/products?$filter=name eq 'Milk'
$ curl -X GET https://api.contoso.com/v1.0/products?$filter=name eq 'Milk' and price lt 2.55
$ curl -X GET https://api.contoso.com/v1.0/products?$filter=name eq 'Milk' or price lt 2.55
```

**Ordering**: orderBy

```bash
$ curl -X GET https://api.contoso.com/v1.0/people?$orderBy=name
$ curl -X GET https://api.contoso.com/v1.0/people?$orderBy=name desc
$ curl -X GET https://api.contoso.com/v1.0/people?$orderBy=name desc,hireDate
```

**Range**: for large obj like video; `HEAD` to get meta first.

```bash
$ curl -X HEAD https://adventure-works.com/products/10?fields=productImage HTTP/1.1

# server returns Content-Length: xxx

# range partial
$ curl -X -H Range: bytes=start-end \
	GET https://adventure-works.com/products/10?fields=productImage 
```

### Create

POST: `/resources` 倾向于表达不确定的创建，服务端收到请求后，可以自行生成资源 URI，自行添加字段。多次请求可能创建多个对象；返回 201 CREATED & Location Header。

PUT: `/resources/{name}` 倾向于表达确定的创建或者替换，客户端知道资源 URI，知道资源的全部表现内容，且多次请求结果幂等。返回 201 CREATED。

### Update

PUT 全局更新

- 如果创建了资源，返回 201 Created
- 如果更新成功且响应 Body 包含资源对象，返回 200 OK
- 如果更新成功但响应 Body 不包含资源对象，返回 204 No Content

PATCH 部分更新，适合大对象

- 更新成功后，可返回 200 (OK) 并在 Body 包含更新后的对象，或返回 204 (No Content) 置空 Body
- 异常返回
  - 400 Bad Request 请求参数非法
  - 404 Not Found 仅限 PATCH
  - 409 Conflict 请求字段不存在、并发冲突

### Delete

- 202 Accepted 异步删除
- 204 No Content 删除完成且 Response Body 为空
- 200 OK 删除完成且在 Response Body 包含删除状态描述

## Subresource

是否需要实现子资源 API 取决于：对象大小/业务复杂度。比如 K8s API obj **status**.

```bash
$ curl -X GET/PUT/PATCH /resources/resource/subresource
```

## Error handling

[RFC 9110](https://www.rfc-editor.org/rfc/rfc9110)

```json
{
  #... ignore apiVersion, kind, status fields
  "code": 409,
  "reason": "Conflict",
  "message": "Operation cannot be fulfilled on configmaps \"my-db-redis-scripts\": the object has been modified; please apply your changes to the latest version and try again",
  "details": {
    "name": "my-db-redis-scripts",
    "kind": "configmaps"
  }
}
---
{
  #... ignore apiVersion, kind, status fields
  "code": 409,
  "reason": "AlreadyExists",
  "message": "pods \"x\" already exists",
  "details": {
    "name": "x",
    "kind": "pods"
  }
}
```


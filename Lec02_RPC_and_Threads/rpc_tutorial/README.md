## 简介
远程过程调用（英语：Remote Procedure Call，缩写为 RPC）是一个计算机通信协议。该协议允许运行于一台计算机的程序调用另一台计算机的子程序，而程序员无需额外地为这个交互作用编程。

## Go语言使用RPC
### [标准库](https://studygolang.com/pkgdoc)的`net/rpc`包是怎么描述的：
`package rpc
import "net/rpc"`

`rpc`包提供了**通过网络或其他I/O连接对一个对象的导出方法的访问。服务端注册一个对象，使它作为一个服务被暴露，服务的名字是该对象的类型名。注册之后，对象的导出方法就可以被远程访问。服务端可以注册多个不同类型的对象（服务），但注册具有相同类型的多个对象是错误的。**

**只有满足如下标准的方法才能用于远程访问，其余方法会被忽略：**
**- 方法是导出的**
**- 方法有两个参数，都是导出类型或内建类型**
**- 方法的第二个参数是指针**
**- 方法只有一个error接口类型的返回值**


事实上，方法必须看起来像这样：
**`func (t *T) MethodName(argType T1, replyType *T2) error`**
其中T、T1和T2都能被`encoding/gob`包序列化。这些限制即使使用不同的编解码器也适用。（未来，对定制的编解码器可能会使用较宽松一点的限制）

方法的**第一个参数代表调用者提供的参数；第二个参数代表返回给调用者的参数**。方法的返回值，如果非nil，将被作为字符串回传，在客户端看来就和errors.New创建的一样。如果返回了错误，回复的参数将不会被发送给客户端。

服务端可能会单个连接上调用`ServeConn`管理请求。更典型地，它会创建一个网络监听器然后调用`Accept`；或者，对于HTTP监听器，调用`HandleHTTP`和`http.Serve`。

想要使用服务的客户端会创建一个连接，然后用该连接调用`NewClient`。

更方便的函数`Dial（DialHTTP）`会在一个原始的连接（或HTTP连接）上依次执行这两个步骤。

生成的Client类型值有两个方法，Call和Go，它们的参数为要调用的服务和方法、一个包含参数的指针、一个用于接收接个的指针。

Call方法会等待远端调用完成，而Go方法异步的发送调用请求并使用返回的Call结构体类型的Done通道字段传递完成信号。
除非设置了显式的编解码器，本包默认使用`encoding/gob`包来传输数据。


### 然后标准库也给出了一个例子：
`data.go`
```go
package server

import (
	"errors"
)

type Args struct {
	A, B int
}
type Quotient struct {
	Quo, Rem int
}

type Arith int

func (t *Arith) Multiply(args *Args, reply *int) error {
	*reply = args.A * args.B
	return nil
}
func (t *Arith) Divide(args *Args, quo *Quotient) error {
	if args.B == 0 {
		return errors.New("divide by zero")
	}
	quo.Quo = args.A / args.B
	quo.Rem = args.A % args.B
	return nil
}
```
分析：这是一个公共包，服务端和客户端都有用到里面的数据

`client.go`
```go
package main

import (
	"fmt"
	"log"
	"net/rpc"

	"learn-rpc/data"
)

func main() {
	client, err := rpc.DialHTTP("tcp", "127.0.0.1:1234")
	if err != nil {
		log.Fatal("dialing:", err)
	}
	// Synchronous call
	args := &data.Args{9, 9}
	var reply int
	err = client.Call("Arith.Multiply", args, &reply)
	if err != nil {
		log.Fatal("arith error:", err)
	}
	fmt.Printf("Arith: %d*%d=%d\n", args.A, args.B, reply)
}
```
分析：客户端程序，先获得一个`rpc.Client`的结构，然后调用`Call`方法


`server.go`
```go
package main

import (
	"log"
	"net"
	"net/http"
	"net/rpc"

	"learn-rpc/data"
)

func main() {
	arith := new(data.Arith)
	rpc.Register(arith)
	rpc.HandleHTTP()
	l, e := net.Listen("tcp", ":1234")
	if e != nil {
		log.Fatal("listen error:", e)
	}
	http.Serve(l, nil)
}
```
分析：先调用`Listen`方法获得`Listen`结构，接着调用将该结构传给`Serve`方法即可开始监听


**注意：在运行程序时，要注意包路径的问题，否则会编译错误**
先运行服务端，再运行客户端

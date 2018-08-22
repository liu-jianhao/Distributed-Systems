## 实验1介绍
在lab1中我们将建立一个MapReduce 库，学习如何使用 Go 建立一个容错的分布式系统。在 Part A 我们需要写一个简单的 MapReduce 程序。在 Part B我们需要实现一个 Master 为 MapReduce 的 Workers 安排任务，并处理 Workers 出现的错误。

## 代码执行流程
1. 应用提供一些的输入文件，一个map函数，一个reduce函数，以及reduce任务的数量(nReduce)
2. Master开启一个RPC服务，然后等待Workers来注册(使用maste.go/Register())。当有Task时，schedule.go/schedule()决定将这些Task指派给workers，以及如何处理Workers的错误
3. Master将每个输入文件当作Task，然后调用common_map.go/dMap()。这个过程既可以直接进行(串行模式)，也可以通过RPC来让Workers做。每次调用doMap()都会读取相应的文件，运行map函数，将key/value结果写入nReduce个中间文件之中。在map结束后总共会生成nMap * nReduce个文件。命令格式为：前缀-map编号-reduce编号，例如，如果有两个map tasks以及三个reduce tasks，map会生成2 * 3 = 6个中间文件。
```
 mrtmp.xxx-0-0
 mrtmp.xxx-0-1
 mrtmp.xxx-0-2
 mrtmp.xxx-1-0
 mrtmp.xxx-1-1
 mrtmp.xxx-1-2
```
每个Worker必须能够读取其他Worker写入的文件。真正的分布式系统会使用分布式存储来实现不同机器之间的共享，在这里我们将所有的Workers运行在一台电脑上，并使用本地文件系统
4. 此后Master会调用common_reduce.go/doReduce()，与doMap()一样，它也能直接完成或者通过工人完成。doReduce()将按照reduce task编号来汇总，生成nRduce个结果文件。例如上面的例子中按照如下分组进行汇总：
```
  // reduce task 0
  mrtmp.xxx-0-0
  mrtmp.xxx-1-0
  // reduce task 1
  mrtmp.xxx-0-1
  mrtmp.xxx-1-1
  // reduce task 2
  mrtmp.xxx-0-2
  mrtmp.xxx-1-2
```
5. 此后Master调用master_splitmerge.go/mr.merge()将所有生成的文件整合成一个输出。
6. Master发送Shutdown RPC给所有Workers，然后关闭自己的RPC服务

## Part I: Map/Reduce input and output
在我们实现 Map/Reduce 之前，首先要修复一个串行实现。给出的源码缺少两个关键部分：
1. 划分 Map 输出的函数 doMap()
2. 汇总 Reduce 输入的函数 doReduce()
首先建议阅读 common.go，这里定义了会用到的数据类型，文件命名方法等

### doMap() 函数
首先总结一下 Map 的过程，它对于每个 Map Task，都会进行以下操作：
1. 从某个数据文件 A.txt 中读取数据。
2. 自定义函数 mapF 对 A.txt 中的文件进行解读，形成一组 {Key, Val} 对。
3. 生成 nReduce 个子文件 A_1, A_2, ..., A_nReduce。
4. 利用 {Key, Val} 中的 Key 值做哈希，将得到的值对 nReduce 取模，以此为依据将其分配到子文件之中。

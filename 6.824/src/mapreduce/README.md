## 实验1介绍
在lab1中我们将建立一个MapReduce 库，学习如何使用 Go 建立一个容错的分布式系统。在 Part A 我们需要写一个简单的 MapReduce 程序。在 Part B我们需要实现一个 Master 为 MapReduce 的 Workers 安排任务，并处理 Workers 出现的错误。

## 代码执行流程
1. 应用提供一些的输入文件，一个map函数，一个reduce函数，以及reduce任务的数量(nReduce)
2. Master开启一个RPC服务，然后等待Workers来注册(使用maste.go/Register())。当有Task时，schedule.go/schedule()决定将这些Task指派给workers，以及如何处理Workers的错误
3. Master将每个输入文件当作Task，然后调用common_map.go/dMap()。这个过程既可以直接进行(串行模式)，也可以通过RPC来让Workers做。每次调用doMap()都会读取相应的文件，运行map函数，将key/value结果写入nReduce个中间文件之中。在map结束后总共会生成nMap * nReduce个文件。命令格式为：前缀-map编号-reduce编号(见common.go)，例如，如果有两个map tasks以及三个reduce tasks，map会生成2 * 3 = 6个中间文件。
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
```go
func doMap(
	jobName string, // the name of the MapReduce job
	mapTaskNumber int, // which map task this is
	inFile string,
	nReduce int, // the number of reduce task that will be run ("R" in the paper)
	mapF func(file string, contents string) []KeyValue,
) {
	// 你需要重写这个函数。你可以通过reduceName获取文件名,使用map任务的输入为reduce任务提供输出。
	// 下面给出的ihash函数应该被用于决定每个key属于的文件。
	//
	// map任务的中间输入以多文件的形式保存在文件系统上,它们的文件名说明是哪个map任务产生的,同时也说明哪个reduce任务会处理它们。
	// 想出如何存储键/值对在磁盘上的方案可能会非常棘手,特别地, 当我们考虑到key和value都包含新行(newlines),引用(quotes),或者其他
	// 你想到的字符。
	//
	// 有一种格式经常被用来序列化数据到字节流,然后可以通过字节流进行重建，这种格式是json。你没有被强制使用JSON,但是reduce任务的输出
	// 必须是JSON格式,熟悉JSON数据格式会对你有所帮助。你可以使用下面的代码将数据结构以JSON字符串的形式输出。对应的解码函数在common_reduce.go
	// 可以找到。
	//
	//   enc := json.NewEncoder(file)
	//   for _, kv := ... {
	//     err := enc.Encode(&kv)
	//
	//   记得关闭文件当你写完全部的数据之后。


	// 注：Map的大致流程如下(官方教材建议不上传代码，所以去除)
	// 　S1: 　打开输入文件，并且读取全部数据
	//  S2： 调用用户自定义的mapF函数,分检数据,在word count的案例中分割成单词
	//  S3： 将mapF返回的数据根据key分类,跟文件名对应(reduceName获取文件名)
	// 　S4: 　将分类好的数据分别写入不同文件

	// 打印参数
	fmt.Printf("jobName = %s, inFile = %s, mapTaskNumber = %d, nReduce = %d\n", jobName, inFile, mapTaskNumber, nReduce);

	// 读取文件
	bytes, err := ioutil.ReadFile(inFile);
	if err != nil {
		log.Fatal("Unable to read file: ", inFile);
	}

	// 解析输入文件为键值对数组
	kv_pairs := mapF(inFile, string(bytes))

	// 生成一组encoder用来对应文件
	encoders := make([]*json.Encoder, nReduce)
	for reduceTaskNumber := 0; reduceTaskNumber < nReduce; reduceTaskNumber++ {
		// reduceName(在comman.go)返回一个特定的文件名
		filename := reduceName(jobName, mapTaskNumber, reduceTaskNumber)
		file_ptr, err := os.Create(filename)
		if err != nil {
			log.Fatal("Unable to create file: ", filename)
		}

		defer file_ptr.Close()
		encoders[reduceTaskNumber] = json.NewEncoder(file_ptr)
	}

	// 利用encoder将键值对写入对应文件
	for _, key_val := range kv_pairs {
		key := key_val.Key
		reduce_idx := ihash(key) % uint32(nReduce)
		err := encoders[reduce_idx].Encode(key_val)
		if err != nil {
			log.Fatal("Unable to write to file")
		}
	}
}
```

### doReduce()函数
```go
func doReduce(
	jobName string, // the name of the whole MapReduce job
	reduceTaskNumber int, // which reduce task this is
	nMap int, // the number of map tasks that were run ("M" in the paper)
	reduceF func(key string, values []string) string,
) {
	// 你需要完成这个函数。你可与获取到来自map任务生产的中间数据，通过reduceName获取到文件名。
	//  记住你应该编码了值到中间文件,所以你需要解码它们。如果你选择了使用JSON,你通过创建decoder读取到多个
	// 解码之后的值，直接调用Decode直到返回错误。
	//
	// 你应该将reduce输出以JSON编码的方式保存到文件，文件名通过mergeName获取。我们建议你在这里使用JSON,

	// key是中间文件里面键值，value是字符串,这个map用于存储相同键值元素的合并

	// Reduce的过程如下：
	//   S1: 获取到Map产生的文件并打开(reduceName获取文件名)
	// 　S2：获取中间文件的数据(对多个map产生的文件更加值合并)
	// 　S3：打开文件（mergeName获取文件名），将用于存储Reduce任务的结果
	// 　S4：合并结果之后(S2)，进行reduceF操作, work count的操作将结果累加，也就是word出现在这个文件中出现的次数

	// 查看参数
	fmt.Printf("Reduce: job name = %s, reduce task id = %d, nMap = %d\n", jobName, reduceTaskNumber, nMap);

	// 建立哈希表，以slice形式存储同一key的所有value
	kv_map := make(map[string]([]string))

	// 读取同一个reduce task 下的所有文件，保存至哈希表
	for mapTaskNumber := 0; mapTaskNumber < nMap; mapTaskNumber++ {
		filename := reduceName(jobName, mapTaskNumber, reduceTaskNumber)
		f, err := os.Open(filename)
		if err != nil {
			log.Fatal("Unable to read from: ", filename)
		}
		defer f.Close()

		decoder := json.NewDecoder(f)
		var kv KeyValue
		for ; decoder.More(); {
			err := decoder.Decode(&kv)
			if err != nil {
				log.Fatal("json decode failed, ", err)
			}
			kv_map[kv.Key] = append(kv_map[kv.Key], kv.Value)
		}
	}
	
	// 对哈希表所有key进行升序排序
	keys := make([]string, 0, len(kv_map))
	for k, _ := range kv_map {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// 利用自定义的reduceF函数处理同一key下所有val
	// 按照key的顺序将结果以{{key, new_val}}形式输出
	outf, err := os.Create(mergeName(jobName, reduceTaskNumber))
	if err != nil {
		log.Fatal("Unable to create file: ", mergeName(jobName, reduceTaskNumber))
	}
	defer outf.Close()
	encoder := json.NewEncoder(outf)
	for _,k := range keys {
		encoder.Encode(KeyValue{k, reduceF(k, kv_map[k])})
	}
}
```
运行part1.sh或直接在命令行输入：
```
go test -run Sequential
```
即可看到运行结果：
```
ter: Starting Map/Reduce task test
jobName = test, inFile = 824-mrinput-0.txt, mapTaskNumber = 0, nReduce = 3
Reduce: job name = test, reduce task id = 0, nMap = 1
Reduce: job name = test, reduce task id = 1, nMap = 1
Reduce: job name = test, reduce task id = 2, nMap = 1
Merge: read mrtmp.test-res-0
Merge: read mrtmp.test-res-1
Merge: read mrtmp.test-res-2
master: Map/Reduce task completed
master: Starting Map/Reduce task test
jobName = test, inFile = 824-mrinput-0.txt, mapTaskNumber = 0, nReduce = 3
jobName = test, inFile = 824-mrinput-1.txt, mapTaskNumber = 1, nReduce = 3
jobName = test, inFile = 824-mrinput-2.txt, mapTaskNumber = 2, nReduce = 3
jobName = test, inFile = 824-mrinput-3.txt, mapTaskNumber = 3, nReduce = 3
jobName = test, inFile = 824-mrinput-4.txt, mapTaskNumber = 4, nReduce = 3
Reduce: job name = test, reduce task id = 0, nMap = 5
Reduce: job name = test, reduce task id = 1, nMap = 5
Reduce: job name = test, reduce task id = 2, nMap = 5
Merge: read mrtmp.test-res-0
Merge: read mrtmp.test-res-1
Merge: read mrtmp.test-res-2
master: Map/Reduce task completed
PASS
ok  	_/root/Distributed-Systems/6.824/src/mapreduce	3.831s
```
## Part II: Single-worker word count
既然map和reduce任务已经被连接，那么我们开始实现一些Map/Reduce操作。在这个实验中我们会实现单词计数器——一个简单经典的Map/Reduce例子。特别地,你们的任务是改写mapF和reduceF函数，这样wc.go就可以报告每个单词的数量了。一个单词是任何相邻的字母序列，正如unicode.IsLetter定义的那样。

这里有一些以pg-开头的输入文件位于~/6.824/src/main路径下面。尝试编译我们通过给你的软件，然后就使用我们提供的输入文件运行看看：
```	
    $ cd 6.824
    $ export "GOPATH=$PWD"
	$ cd "$GOPATH/src/main"
	$ go run wc.go master sequential pg-*.txt
	# command-line-arguments
	./wc.go:14: missing return at end of function
	./wc.go:21: missing return at end of function
```	
编译失败的原因是我们还没有编写完整的map函数(mapF)和reduce函数(reduceF)。在你开始阅读MapReduce论文的第二部分时，这里需要说明你的mapF函数和reduceF函数跟论文中有些区别。你的mapF函数会传递文件名和文件的内容；它将会分隔成单词，然后返回含有键值对的切片类型([]KeyValue),你的reduceF函数对于每个键都会调用一次，参数是mapF生成的切片数据，它只有一个返回值。

你可以通过下面的命令运行你的解决方案：

```	
	$ cd "$GOPATH/src/main"
	$ go run wc.go master sequential pg-*.txt
	master: Starting Map/Reduce task wcseq
	Merge: read mrtmp.wcseq-res-0
	Merge: read mrtmp.wcseq-res-1
	Merge: read mrtmp.wcseq-res-2
	master: Map/Reduce task completed
	14.59user 3.78system 0:14.81elapsed
```	

生产的结果会在文件mrtmp.wcseq中，通过下面的命令删除全部的中间数据：

```	
	$ rm mrtmp.*
```	

如果执行下面的命令出现如下内容，那么你的实现是正确的：

```	
	$ sort -n -k2 mrtmp.wcseq | tail -10
	he: 34077
	was: 37044
	that: 37495
	I: 44502
	in: 46092
	a: 60558
	to: 74357
	of: 79727
	and: 93990
	the: 154024
```	

为了让测试更加简单，你可以运行下面的脚本：

```	
	$ sh ./test-wc.sh
```	

脚本会告知你的解决方案是否正确。
## Part III: Distributing MapReduce tasks
在开始这部分实验之前要先阅读master.go的代码
目前为止我们都是串行地执行任务，Map/Reduce 最大的优势就是能够自动地并行执行普通的代码，不用开发者进行额外工作。在 Part III 我们会把任务分配给一组 worker thread，在多核上并行进行。虽然我们不在多机上进行，但是会用 RPC 来模拟分布式计算。

我们需要实现 `mapreduce/schedule.go` 的 `schedule()`，在一次任务中，Master 会调用两次 `schedule()`，一次用于 Map，一次用于 Reduce。`schedule()`将会把任务分配给 Workers，通常任务会比 Workers 数量多，因此 `schedule()` 会给每个 worker 一个 Task 序列，然后等待所有 Task 完成再返回。

`schedule()` 通过 `registerChan` 参数获取 Workers 信息，它会生成一个包含 Worker 的 RPC 地址的 string，有些 Worker 在调用 `schedule()` 之前就存在了，有的在调用的时候产生，他们都会出现在 `registerChan` 中。

`schedule()` 通过发送 `Worker.DoTask` RPC 调度 Worker 执行任务，可以用 `mapreduce/common_rpc.go` 中的 `call()` 函数发送。`call()` 的第一个参数是 Worker 的地址，可以从 `registerChan` 获取，第二个参数是 `"Worker.DoTask"` 字符串，第三个参数是 `DoTaskArgs` 结构体的指针，最后一个参数为 `nil`。

这是一个比较有含金量的练习。
`schedule()` 函数的调用发生在 `master.go` 中。首先，在`Distributed()` 中对 `schedule()` 做了一个再封装，使得其只需要一个参数判断是 Map 还是 Reduce。

## Part IV: Handling worker failures
在该部分中我们需要让 Master 能够处理 failed Worker。MapReduce 使这个处理相对简单，因为如果一个 Worker fails，Master 交给它的任何任务都会失败。此时 Master 需要把任务交给另一个 Worker。

一个 RPC 出错并不一定表示 Worker 没有执行任务，有可能只是 reply 丢失了，或是 Master 的 RPC 超时了。因此，有可能两个 Worker 都完成了同一个任务。同样的任务会生成同样的结果，所以这样并不会引发什么问题。并且，在该 lab 中，每个 Task 都是序列执行的，这就保证了结果的整体性。
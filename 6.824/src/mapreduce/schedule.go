package mapreduce

import (
	"fmt"
	"sync"
)

// jobPhase实际就是string，值为"Map"或者"Reduce"
// schedule starts and waits for all tasks in the given phase (Map or Reduce).
func (mr *Master) schedule(phase jobPhase) {
	var ntasks int
	var nios int // number of inputs (for reduce) or outputs (for map)
	switch phase {
	case mapPhase:
		ntasks = len(mr.files) // 输入文件个数
		nios = mr.nReduce      // 生成的中间文件的个数
	case reducePhase:
		ntasks = mr.nReduce
		nios = len(mr.files) // 获取Map生成的中间文件的个数
	}

	fmt.Printf("Schedule: %v %v tasks (%d I/Os)\n", ntasks, phase, nios)

	// All ntasks tasks have to be scheduled on workers, and only once all of
	// them have been completed successfully should the function return.
	// Remember that workers may fail, and that any given worker may finish
	// multiple tasks.
	//
	// TODO
	// 所有ntasks个任务都必须被workers调度，并且它们都成功完成如果函数返回，
	// 记住，workers可能会失败，并且有些worker可能会完成多个任务
	// 1. 从channel获取worker
	// 2. 通过worker进行rpc调用, `Worker.DoTask`,
	// 3. 若rpc调用执行失败, 则将任务重新塞入registerChannel执行
	// ps: 使用WaitGroup保证线程同步
	// 若不加Wait等待所有goroutine结束在返回, 则会导致一些结果文件并未生成, 测试挂掉
	fmt.Printf("Schedule: %v phase done\n", phase)

	var wg sync.WaitGroup
	for i := 0; i < ntasks; i++ {
		wg.Add(1) //增加wg的计数

		fmt.Printf("current taskNum: %v, nios: %v, phase: %v\n", i, nios, phase)
		// DoTaskArgs用来保存参数，用于worker分配工作
		var args DoTaskArgs
		args.JobName = mr.jobName
		args.Phase = phase
		args.TaskNumber = i
		args.NumOtherPhase = nios
		if phase == mapPhase {
			args.File = mr.files[i]
		}
		// 并发处理，速度更快
		go func() {
			defer wg.Done()
			// 在part3的基础上加一个无限循环，只要出错就换一个worker
			for {
				worker := <-mr.registerChannel
				fmt.Printf("current worker port: %v\n", worker)

				// call：本地的rpc调用，使用是unix套接字
				ok := call(worker, "Worker.DoTask", &args, nil)
				if ok {
					go func() { mr.registerChannel <- worker }()
					break
				}
			}
		}()
	}
	wg.Wait()
	fmt.Printf("Schedule: %v phase done\n", phase)
}

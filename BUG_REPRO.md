# Bug 复现说明

## Bug 是什么

MemoryQueue 在停止后仍允许提交路径向已经关闭的任务 channel 发送数据，导致提交任务时发生 panic；停止后重新启动还可能继续复用已关闭的 channel，队列生命周期状态与 worker 使用的 channel 不一致。

## 如何触发

先启动内存队列，停止队列，再提交一个任务，并重复执行停止、启动和提交的生命周期场景。该场景通过并发安全测试反复检查停止后的提交行为和重启后的队列可用性。

## 运行指令

```bash
go test ./backend/internal/queue -race -run '^TestMemoryQueueRejectsStoppedEnqueueAndRestarts$' -count=20
```

## 错误信息

停止后的提交触发 `send on closed channel`，目标测试因此失败。

## 错误堆栈

以下为上述指令实际产生的原始输出：

```text
Exit code 1
2026-08-20T10:00:42.893+0800	INFO	queue/memory.go:55	index queue started	{"workers": 2}
--- FAIL: TestMemoryQueueRejectsStoppedEnqueueAndRestarts (0.00s)
    memory_lifecycle_test.go:68: enqueue after stop panicked: send on closed channel
2026-08-20T10:00:42.894+0800	INFO	queue/memory.go:55	index queue started	{"workers": 2}
--- FAIL: TestMemoryQueueRejectsStoppedEnqueueAndRestarts (0.00s)
    memory_lifecycle_test.go:68: enqueue after stop panicked: send on closed channel
2026-08-20T10:00:42.895+0800	INFO	queue/memory.go:55	index queue started	{"workers": 2}
--- FAIL: TestMemoryQueueRejectsStoppedEnqueueAndRestarts (0.00s)
    memory_lifecycle_test.go:68: enqueue after stop panicked: send on closed channel
2026-08-20T10:00:42.896+0800	INFO	queue/memory.go:55	index queue started	{"workers": 2}
--- FAIL: TestMemoryQueueRejectsStoppedEnqueueAndRestarts (0.00s)
    memory_lifecycle_test.go:68: enqueue after stop panicked: send on closed channel
2026-08-20T10:00:42.896+0800	INFO	queue/memory.go:55	index queue started	{"workers": 2}
--- FAIL: TestMemoryQueueRejectsStoppedEnqueueAndRestarts (0.00s)
    memory_lifecycle_test.go:68: enqueue after stop panicked: send on closed channel
2026-08-20T10:00:42.897+0800	INFO	queue/memory.go:55	index queue started	{"workers": 2}
--- FAIL: TestMemoryQueueRejectsStoppedEnqueueAndRestarts (0.00s)
    memory_lifecycle_test.go:68: enqueue after stop panicked: send on closed channel
2026-08-20T10:00:42.897+0800	INFO	queue/memory.go:55	index queue started	{"workers": 2}
--- FAIL: TestMemoryQueueRejectsStoppedEnqueueAndRestarts (0.00s)
    memory_lifecycle_test.go:68: enqueue after stop panicked: send on closed channel
2026-08-20T10:00:42.897+0800	INFO	queue/memory.go:55	index queue started	{"workers": 2}
--- FAIL: TestMemoryQueueRejectsStoppedEnqueueAndRestarts (0.00s)
    memory_lifecycle_test.go:68: enqueue after stop panicked: send on closed channel
2026-08-20T10:00:42.898+0800	INFO	queue/memory.go:55	index queue started	{"workers": 2}
--- FAIL: TestMemoryQueueRejectsStoppedEnqueueAndRestarts (0.00s)
    memory_lifecycle_test.go:68: enqueue after stop panicked: send on closed channel
2026-08-20T10:00:42.898+0800	INFO	queue/memory.go:55	index queue started	{"workers": 2}
--- FAIL: TestMemoryQueueRejectsStoppedEnqueueAndRestarts (0.00s)
    memory_lifecycle_test.go:68: enqueue after stop panicked: send on closed channel
2026-08-20T10:00:42.898+0800	INFO	queue/memory.go:55	index queue started	{"workers": 2}
--- FAIL: TestMemoryQueueRejectsStoppedEnqueueAndRestarts (0.00s)
    memory_lifecycle_test.go:68: enqueue after stop panicked: send on closed channel
2026-08-20T10:00:42.899+0800	INFO	queue/memory.go:55	index queue started	{"workers": 2}
--- FAIL: TestMemoryQueueRejectsStoppedEnqueueAndRestarts (0.00s)
    memory_lifecycle_test.go:68: enqueue after stop panicked: send on closed channel
2026-08-20T10:00:42.899+0800	INFO	queue/memory.go:55	index queue started	{"workers": 2}
--- FAIL: TestMemoryQueueRejectsStoppedEnqueueAndRestarts (0.00s)
    memory_lifecycle_test.go:68: enqueue after stop panicked: send on closed channel
2026-08-20T10:00:42.899+0800	INFO	queue/memory.go:55	index queue started	{"workers": 2}
--- FAIL: TestMemoryQueueRejectsStoppedEnqueueAndRestarts (0.00s)
    memory_lifecycle_test.go:68: enqueue after stop panicked: send on closed channel
2026-08-20T10:00:42.900+0800	INFO	queue/memory.go:55	index queue started	{"workers": 2}
--- FAIL: TestMemoryQueueRejectsStoppedEnqueueAndRestarts (0.00s)
    memory_lifecycle_test.go:68: enqueue after stop panicked: send on closed channel
2026-08-20T10:00:42.900+0800	INFO	queue/memory.go:55	index queue started	{"workers": 2}
--- FAIL: TestMemoryQueueRejectsStoppedEnqueueAndRestarts (0.00s)
    memory_lifecycle_test.go:68: enqueue after stop panicked: send on closed channel
2026-08-20T10:00:42.900+0800	INFO	queue/memory.go:55	index queue started	{"workers": 2}
--- FAIL: TestMemoryQueueRejectsStoppedEnqueueAndRestarts (0.00s)
    memory_lifecycle_test.go:68: enqueue after stop panicked: send on closed channel
2026-08-20T10:00:42.900+0800	INFO	queue/memory.go:55	index queue started	{"workers": 2}
--- FAIL: TestMemoryQueueRejectsStoppedEnqueueAndRestarts (0.00s)
    memory_lifecycle_test.go:68: enqueue after stop panicked: send on closed channel
2026-08-20T10:00:42.901+0800	INFO	queue/memory.go:55	index queue started	{"workers": 2}
--- FAIL: TestMemoryQueueRejectsStoppedEnqueueAndRestarts (0.00s)
    memory_lifecycle_test.go:68: enqueue after stop panicked: send on closed channel
2026-08-20T10:00:42.901+0800	INFO	queue/memory.go:55	index queue started	{"workers": 2}
--- FAIL: TestMemoryQueueRejectsStoppedEnqueueAndRestarts (0.00s)
    memory_lifecycle_test.go:68: enqueue after stop panicked: send on closed channel
FAIL
FAIL	kn-system/backend/internal/queue	0.791s
FAIL
```

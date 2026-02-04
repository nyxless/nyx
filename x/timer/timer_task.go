package timer

import (
	"container/heap"
	"context"
	"fmt"
	"github.com/nyxless/nyx/x/log"
	"sync"
	"time"
)

type Task struct {
	ID       string        // 任务唯一标识
	expireAt time.Time     // 过期时间
	Period   time.Duration // 执行周期，0表示单次任务
	Callback func() error  // 回调函数
}

// 任务堆
type taskHeap []*Task

func (h taskHeap) Len() int           { return len(h) }
func (h taskHeap) Less(i, j int) bool { return h[i].expireAt.Before(h[j].expireAt) }
func (h taskHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *taskHeap) Push(x interface{}) {
	*h = append(*h, x.(*Task))
}

func (h *taskHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

type TimerTask struct {
	tasks       *taskHeap
	mu          sync.RWMutex
	wakeupChan  chan struct{}
	wg          sync.WaitGroup
	running     bool
	taskMap     map[string]*Task
	errorLogger func(error) // 错误日志处理器
	ctx         context.Context
	cancelFunc  context.CancelFunc
}

func NewTimerTask() *TimerTask { // {{{
	ctx, cancel := context.WithCancel(context.Background())
	h := &taskHeap{}
	heap.Init(h)

	tt := &TimerTask{
		tasks:      h,
		wakeupChan: make(chan struct{}, 1),
		taskMap:    make(map[string]*Task),
		running:    false,
		ctx:        ctx,
		cancelFunc: cancel,
		errorLogger: func(err error) {
			fmt.Println("timer task err:", err)
		},
	}

	return tt
} // }}}

func (tt *TimerTask) WithLogger(logger *log.Logger) {
	tt.errorLogger = func(err error) {
		logger.Error("timer task err:", err)
	}
}

// 启动
func (tt *TimerTask) Start() error { // {{{
	tt.mu.Lock()
	defer tt.mu.Unlock()

	if tt.running {
		return fmt.Errorf("timer task manager is already running")
	}

	tt.running = true
	tt.wg.Add(1)
	go tt.run()

	return nil
} // }}}

// 停止
func (tt *TimerTask) Stop() { // {{{
	tt.mu.Lock()
	if !tt.running {
		tt.mu.Unlock()
		return
	}

	tt.running = false
	tt.mu.Unlock()

	tt.cancelFunc()

	tt.wg.Wait()
} // }}}

// 添加定时任务, 按 period 周期执行，首次在 period 后执行
func (tt *TimerTask) AddTask(id string, period time.Duration, callback func() error) error {
	startTime := time.Now().Add(period)
	return tt.addTask(id, startTime, period, callback)
}

// 添加定时任务, 按 period 周期执行, 同时指定首次运行时间
func (tt *TimerTask) AddTaskWithStartTime(id string, startTime time.Time, period time.Duration, callback func() error) error {
	return tt.addTask(id, startTime, period, callback)
}

// 添加一次性任务, delay 后执行
func (tt *TimerTask) AddOnceTask(id string, delay time.Duration, callback func() error) error {
	startTime := time.Now().Add(delay)
	return tt.addTask(id, startTime, 0, callback)
}

// 添加一次性任务, 同时指定首次运行时间
func (tt *TimerTask) AddOnceTaskWithStartTime(id string, startTime time.Time, callback func() error) error {
	return tt.addTask(id, startTime, 0, callback)
}

func (tt *TimerTask) addTask(id string, startTime time.Time, period time.Duration, callback func() error) error { // {{{
	if period < 0 {
		return fmt.Errorf("period cannot be negative")
	}

	tt.mu.Lock()
	defer tt.mu.Unlock()

	// 检查任务ID是否已存在
	if _, exists := tt.taskMap[id]; exists {
		return fmt.Errorf("task with id '%s' already exists", id)
	}

	task := &Task{
		ID:       id,
		expireAt: startTime,
		Period:   period,
		Callback: callback,
	}

	heap.Push(tt.tasks, task)
	tt.taskMap[id] = task

	// 唤醒主循环处理新任务
	tt.wakeup()

	return nil
} // }}}

// 移除任务
func (tt *TimerTask) RemoveTask(id string) bool { // {{{
	tt.mu.Lock()
	defer tt.mu.Unlock()

	task, exists := tt.taskMap[id]
	if !exists {
		return false
	}

	// 标记任务为已移除（在堆实现中，无法直接删除中间元素）
	// 采用惰性删除策略：在map中删除，在堆执行时跳过
	delete(tt.taskMap, id)

	// 标记任务ID为空，executeExpiredTasks会跳过它
	task.ID = ""

	return true
} // }}}

// 获取任务数量
func (tt *TimerTask) GetTaskCount() int {
	tt.mu.RLock()
	defer tt.mu.RUnlock()

	return tt.getTaskCount()
}

// 获取下一个任务的执行时间
func (tt *TimerTask) GetNextTaskTime() (time.Time, bool) { // {{{
	tt.mu.RLock()
	defer tt.mu.RUnlock()

	if tt.tasks.Len() == 0 {
		return time.Time{}, false
	}

	// 需要遍历找到第一个有效任务
	for _, task := range *tt.tasks {
		if task.ID != "" { // 有效任务
			return task.expireAt, true
		}
	}

	return time.Time{}, false
} // }}}

// 添加唤醒机制
func (tt *TimerTask) wakeup() {
	select {
	case tt.wakeupChan <- struct{}{}:
	default:
		// 通道已满，说明已经有唤醒信号
	}
}

func (tt *TimerTask) run() { // {{{
	defer tt.wg.Done()

	const maxWaitDuration = 24 * time.Hour // 最大等待时间

	for {
		// 获取下一个任务的等待时间
		nextWakeup := tt.calculateNextWakeup()

		if nextWakeup <= 0 {
			// 立即执行或没有任务
			if tt.executeExpiredTasks() == 0 && tt.getTaskCount() == 0 {
				// 没有任务，等待唤醒或停止
				select {
				case <-tt.wakeupChan:
					continue
				case <-tt.ctx.Done():
					return
				}
			}
			continue
		}

		// 限制最大等待时间
		if nextWakeup > maxWaitDuration {
			nextWakeup = maxWaitDuration
		}

		// 等待时间到期、被唤醒或停止
		timer := time.NewTimer(nextWakeup)
		select {
		case <-timer.C:
			// 时间到期，执行任务
			tt.executeExpiredTasks()
		case <-tt.wakeupChan:
			// 被新任务唤醒
			timer.Stop()
			continue
		case <-tt.ctx.Done():
			timer.Stop()
			return
		}
	}
} // }}}

// 计算下一个任务的等待时间
func (tt *TimerTask) calculateNextWakeup() time.Duration { // {{{
	tt.mu.RLock()
	defer tt.mu.RUnlock()

	if tt.tasks.Len() == 0 {
		return 0
	}

	// 找到第一个有效任务
	for _, task := range *tt.tasks {
		if task.ID == "" {
			continue // 跳过已删除的任务
		}

		now := time.Now()
		if task.expireAt.After(now) {
			return task.expireAt.Sub(now)
		}
		return 0 // 立即执行
	}

	return 0 // 没有有效任务
} // }}}

// 执行到期任务，返回执行的任务数量
func (tt *TimerTask) executeExpiredTasks() int { // {{{
	tt.mu.Lock()
	defer tt.mu.Unlock()

	now := time.Now()
	count := 0
	executedTasks := make([]*Task, 0)

	for tt.tasks.Len() > 0 {
		task := (*tt.tasks)[0]

		// 跳过已删除的任务
		if task.ID == "" {
			heap.Pop(tt.tasks)
			continue
		}

		if task.expireAt.After(now) {
			break
		}

		heap.Pop(tt.tasks)
		count++

		// 如果是周期性任务，重新添加到堆中
		if task.Period > 0 {
			// 更新过期时间，保持精确的周期
			task.expireAt = task.expireAt.Add(task.Period)
			// 如果过期时间已经过去（比如回调执行时间过长），调整到下一个周期
			if task.expireAt.Before(now) || task.expireAt.Equal(now) {
				task.expireAt = now.Add(task.Period)
			}
			heap.Push(tt.tasks, task)
		} else {
			// 单次任务从映射中删除
			delete(tt.taskMap, task.ID)
		}

		// 记录待执行的任务
		executedTasks = append(executedTasks, task)
	}

	// 异步执行回调（在锁外执行）
	if len(executedTasks) > 0 {
		go tt.executeCallbacks(executedTasks)
	}

	return count
} // }}}

// 安全执行回调函数
func (tt *TimerTask) executeCallbacks(tasks []*Task) { // {{{
	for _, task := range tasks {
		func() {
			defer func() {
				if r := recover(); r != nil {
					err := fmt.Errorf("panic in timer task callback '%s': %v", task.ID, r)
					tt.errorLogger(err)
				}
			}()

			if task.Callback != nil {
				err := task.Callback()
				if err != nil {
					tt.errorLogger(fmt.Errorf("timer task err: %v", err))
				}
			}
		}()
	}
} // }}}

// 获取任务数量, 只统计有效任务
func (tt *TimerTask) getTaskCount() int {
	count := 0
	for _, task := range tt.taskMap {
		if task.ID != "" {
			count++
		}
	}
	return count
}

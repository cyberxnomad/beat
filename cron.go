package cron

import (
	"context"
	"regexp"
	"runtime"
	"sort"
	"sync"
	"time"
)

type JobFunc func(ctx context.Context, userdata any)

type job struct {
	Id       string  // 任务ID
	Func     JobFunc // 定时执行的任务
	Userdata any     // 用户数据

	Schedule Schedule  // 定时时间
	Next     time.Time // 下一次运行的时间
	Prev     time.Time // 前一次运行的时间
}

type Cron struct {
	jobs        []*job          // 任务集合
	jobWaiter   sync.WaitGroup  // 任务完成等待
	withRecover bool            // 是否启用recover
	lock        sync.Mutex      // 互斥锁
	running     bool            // 是否运行
	parser      ScheduleParser  // 解析器
	location    *time.Location  // 时区
	ctx         context.Context // 上下文
	log         Logger          // log

	operate chan any
}

type ScheduleParser interface {
	Parse(expr string) (Schedule, error)
}

type Schedule interface {
	// 根据给定时间，返回下一个可用的时间
	Next(time.Time) time.Time
}

// 排序需要用到的接口
type jobByTime []*job

func (s jobByTime) Len() int {
	return len(s)
}

func (s jobByTime) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s jobByTime) Less(i, j int) bool {
	if s[i].Next.IsZero() {
		return false
	}
	if s[j].Next.IsZero() {
		return true
	}

	return s[i].Next.Before(s[j].Next)
}

type (
	opAdd             *job
	opRemove          string
	opRemoveAll       struct{}
	opRemoveByPattern *regexp.Regexp
	opStop            struct{}
)

func emptyJobFunc(_ context.Context, _ any) {}

func New(opts ...Option) *Cron {
	c := &Cron{
		jobs:     []*job{},
		parser:   defaultParser,
		location: time.Local,
		ctx:      context.Background(),
		log:      defaultLogger,

		operate: make(chan any),
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

func (c *Cron) run() {
	c.log.Info("msg", "started")
	defer c.log.Info("msg", "stopped")

	now := c.now()

	// 获取一次所有任务的下一次有效时间
	for _, job := range c.jobs {
		job.Next = job.Schedule.Next(now)
		c.log.Info("job.action", "schedule", "job.id", job.Id, "job.next", job.Next.Format(time.RFC3339))
	}

	for {
		// 对任务的下一次执行时间进行排序，
		sort.Sort(jobByTime(c.jobs))

		var timer *time.Timer
		if len(c.jobs) == 0 || c.jobs[0].Next.IsZero() {
			// 没有任务或者时间太长，则休眠，依然可以处理添加或者停止请求
			timer = time.NewTimer(100000 * time.Hour)
		} else {
			// 获取最近执行时间的定时
			timer = time.NewTimer(c.jobs[0].Next.Sub(now))
		}

		for {
			select {
			case now = <-timer.C:
				now = now.In(c.location)
				c.log.Debug("job.action", "wake")

				// 执行所有已经到定时的任务
				for _, job := range c.jobs {
					if job.Next.After(now) || job.Next.IsZero() {
						break
					}
					c.log.Debug("job.action", "execute", "job.id", job.Id)
					// 是否使用带 recover 的执行方式
					if c.withRecover {
						c.executeJobWithRecover(job)
					} else {
						c.executeJob(job)
					}
					job.Prev = job.Next
					job.Next = job.Schedule.Next(now)
				}

			case op := <-c.operate:
				timer.Stop()
				now = c.now()

				switch arg := op.(type) {
				case opAdd:
					newJob := (*job)(arg)

					newJob.Next = newJob.Schedule.Next(now)
					c.addJob(newJob)

					c.log.Info("job.action", "add", "job.id", newJob.Id, "job.next", newJob.Next.Format(time.RFC3339))

				case opRemove:
					id := string(arg)

					c.removeJob(id)

					c.log.Info("job.action", "remove", "job.id", id)

				case opRemoveAll:
					c.removeAllJob()

					c.log.Info("job.action", "remove-all")

				case opRemoveByPattern:
					pattern := (*regexp.Regexp)(arg)

					c.removeJobByPattern(pattern)

					c.log.Info("job.action", "remove-by-pattern", "job.pattern", pattern.String())

				case opStop:
					return
				}
			}

			break
		}
	}
}

// 返回 c.location 的当前时间
func (c *Cron) now() time.Time {
	return time.Now().In(c.location)
}

// 开始执行任务，任务将在协程中执行
//
// ? 如果任务量大，可能会出现协程数量限制，后续考虑优化
func (c *Cron) executeJob(job *job) {
	c.jobWaiter.Add(1)
	go func() {
		defer c.jobWaiter.Done()
		job.Func(c.ctx, job.Userdata)
	}()
}

// 开始执行任务，任务将在协程中执行，如果出现 panic，将会恢复
//
// ? 如果任务量大，可能会出现协程数量限制，后续考虑优化
func (c *Cron) executeJobWithRecover(job *job) {
	c.jobWaiter.Add(1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				buf := make([]byte, 64<<10)
				n := runtime.Stack(buf, false)
				buf = buf[:n]
				c.log.Error("panic", r, "statck", buf)
			}
		}()

		defer c.jobWaiter.Done()
		job.Func(c.ctx, job.Userdata)
	}()
}

func (c *Cron) addJob(job *job) {
	found := c.find(job.Id)
	if found != nil {
		c.log.Warn("msg", "job already exists, overwrite the old one", "job.id", found.Id)
		c.removeJob(found.Id)
	}

	c.jobs = append(c.jobs, job)
}

// 移除任务
//
// 返回移除的任务对象，不存在则返回 nil
func (c *Cron) removeJob(id string) {
	jobs := make([]*job, 0)

	for _, job := range c.jobs {
		if job.Id != id {
			jobs = append(jobs, job)
		}
	}
	c.jobs = jobs
}

// 移除全部任务
func (c *Cron) removeAllJob() {
	c.jobs = make([]*job, 0)
}

// 通过ID前缀移除任务，所有任务ID含有指定前缀的任务都将移除
func (c *Cron) removeJobByPattern(pattern *regexp.Regexp) {
	jobs := make([]*job, 0)

	for _, job := range c.jobs {
		if !pattern.MatchString(job.Id) {
			jobs = append(jobs, job)
		}
	}

	c.jobs = jobs
}

// 通过 ID 查找任务
//
// 返回查找到的任务对象，不存在则返回 nil
func (c *Cron) find(id string) *job {
	for _, job := range c.jobs {
		if job.Id == id {
			return job
		}
	}

	return nil
}

// 添加任务
//
// 参数：
//
//	expr: 定时表达式
//	id: 任务ID，每个任务ID唯一
//	fn: 任务执行回调
//	userdata: 用于保存用户数据，回调时将传递该数据
func (c *Cron) Add(expr string, id string, fn JobFunc, userdata any) error {
	sched, err := c.parser.Parse(expr)
	if err != nil {
		return err
	}

	c.lock.Lock()
	defer c.lock.Unlock()

	job := &job{
		Id:       id,
		Schedule: sched,
		Func:     fn,
		Userdata: userdata,
	}
	if job.Func == nil {
		job.Func = emptyJobFunc
	}

	if !c.running {
		c.addJob(job)
	} else {
		c.operate <- opAdd(job)
	}

	return nil
}

// 移除任务
func (c *Cron) Remove(id string) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if !c.running {
		c.removeJob(id)
	} else {
		c.operate <- opRemove(id)
	}
}

// 清空任务
func (c *Cron) RemoveAll() {
	c.lock.Lock()
	defer c.lock.Unlock()

	if !c.running {
		c.removeAllJob()
	} else {
		c.operate <- opRemoveAll(struct{}{})
	}
}

// 通过正则表达式移除任务
func (c *Cron) RemoveByPattern(exp string) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	pattern, err := regexp.Compile(exp)
	if err != nil {
		return err
	}

	if !c.running {
		c.removeJobByPattern(pattern)
	} else {
		c.operate <- opRemoveByPattern(pattern)
	}

	return nil
}

// 停止运行
func (c *Cron) Stop() {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.running {
		c.operate <- opStop(struct{}{})
		c.running = false
	}
	c.jobWaiter.Wait()
}

// 开始运行，cron 将在协程中运行
func (c *Cron) Start() {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.running {
		return
	}

	c.running = true
	go c.run()
}

func (c *Cron) StartWithContext(ctx context.Context) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.running {
		return
	}

	c.ctx = ctx
	c.running = true
	go c.run()
}

// 开始运行，cron 将阻塞运行
func (c *Cron) Run() {
	c.lock.Lock()

	if c.running {
		c.lock.Unlock()
		return
	}

	c.running = true
	c.lock.Unlock()
	c.run()
}

// 开始运行，cron 将阻塞运行
func (c *Cron) RunWithContext(ctx context.Context) {
	c.lock.Lock()

	if c.running {
		c.lock.Unlock()
		return
	}

	c.ctx = ctx
	c.running = true
	c.lock.Unlock()
	c.run()
}

// 获取运行状态
func (c *Cron) IsRunning() bool {
	c.lock.Lock()
	defer c.lock.Unlock()

	return c.running
}

func (c *Cron) SetLogger(log Logger) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.log = log
}

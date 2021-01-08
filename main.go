package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type TaskType int64

const WorkerNum int = 5
const FIFOBuffNum int = 128

type Task0 struct{}

func (Task0) Task(*DeviceDriver) {
	// do nothing
}

type Task1 struct{
	Handles []Handle
}

func (task Task1) Task(dd *DeviceDriver) {
	handle := dd.OpenDevice(task.Handles)
	if handle == nil {
		// cannot get device // fault select device
		return
	}
	fmt.Println("Task1 is working")
	task.Handles = append(task.Handles, handle)
	time.Sleep(1 * time.Second)
	for _, handle := range task.Handles {
		dd.CloseDevice(handle)
	}
}

type Task2 struct{
	Handles []Handle
}

func (task Task2) Task(dd *DeviceDriver) {
	for i := 0; i < 3; i++ {
		handle := dd.OpenDevice(task.Handles)
		if handle == nil {
			// cannot get device // fault select device
			return
		}
		task.Handles = append(task.Handles, handle)
	}
	fmt.Println("Task2 is working")
	time.Sleep(1 * time.Second)
	for _, handle := range task.Handles {
		dd.CloseDevice(handle)
	}
}

type ITask interface {
	Task(*DeviceDriver)
}

func NewTask(taskType TaskType) ITask {
	switch taskType {
	case 1:
		return Task1{[]Handle{}}
	case 2:
		return Task2{[]Handle{}}
	default:
		return Task0{}
	}
}

type Handle *Device
type DeviceDriver struct {
	handle1 Handle
	handle2 Handle
	handle3 Handle
	handle4 Handle
	handle5 Handle
}

func NewDeviceDriver() *DeviceDriver {
	return &DeviceDriver{
		handle1: NewDevice(),
		handle2: NewDevice(),
		handle3: NewDevice(),
		handle4: NewDevice(),
		handle5: NewDevice(),
	}
}

func (dd *DeviceDriver) OpenDevice(handles []Handle) Handle {
	handle := dd.SelectDevice(handles)
	if handle == nil {
		return nil
	}
	handle.Open()
	return handle
}

func (*DeviceDriver) CloseDevice(handle Handle) {
	(*Device)(handle).Close()
}

func (dd *DeviceDriver) StartWorker(ctx context.Context, buff chan TaskType) {
	// num, _ := strconv.Atoi(os.Getenv("WORKER_COUNT"))
	num := WorkerNum
	wg := &sync.WaitGroup{}
	for i := 0; i < num; i++ {
		wg.Add(1)
		go func() {
			dd.Worker(ctx, buff)
			wg.Done()
		}()
	}
	wg.Wait()
}

func (dd *DeviceDriver) Worker(ctx context.Context, buff chan TaskType) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				dd.DoTask(buff)
			}
		}
	}()
}

func (dd *DeviceDriver) DoTask(buff chan TaskType) {
	taskType := <-buff
	task := NewTask(taskType)
	task.Task(dd)
}

func (dd *DeviceDriver) SelectDevice(handles []Handle) *Device {
	if dd.handle1.IsOpen {
		return dd.handle1
	} else if dd.handle2.IsOpen {
		return dd.handle2
	} else if dd.handle3.IsOpen {
		return dd.handle3
	} else if dd.handle4.IsOpen {
		return dd.handle4
	} else if dd.handle5.IsOpen {
		return dd.handle5
	} else {
		fmt.Println("Select Device Fault")
		dd.Fault(handles)
	}
	return nil
}

func (dd *DeviceDriver)Fault(handles []Handle) {
	fmt.Println("Fault Task")
	for _, handle := range handles {
		dd.CloseDevice(handle)
	}

}

type Device struct {
	IsOpen bool
}

func NewDevice() *Device {
	return &Device{IsOpen: true}
}

func (device *Device) Open() {
	device.IsOpen = false
}

func (device *Device) Close() {
	device.IsOpen = true
}

func main() {
	///////////////////前処理//////////////////
	buff := make(chan TaskType, FIFOBuffNum) // FIFOなBufferです。
	defer close(buff)

	ctx, cancel := context.WithCancel(context.Background())
	sig := make(chan os.Signal, 10)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	go func() {
		defer close(sig)
		<-sig
		cancel()
	}()
	dd := NewDeviceDriver()
	dd.StartWorker(ctx, buff)
	///////////////UserProgram////////////////
	wg := &sync.WaitGroup{}
	wg.Add(1)
	buff <- 1
	buff <- 2
	buff <- 2
	buff <- 1
	wg.Wait()
}

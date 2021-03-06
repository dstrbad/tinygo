package main

import (
	"runtime"
	"time"
)

// waitGroup is a small type reimplementing some of the behavior of sync.WaitGroup
type waitGroup uint

func (wg *waitGroup) wait() {
	n := 0
	for *wg != 0 {
		// pause and wait to be rescheduled
		runtime.Gosched()

		if n > 100 {
			// if something is using the sleep queue, this may be necessary
			time.Sleep(time.Millisecond)
		}

		n++
	}
}

func (wg *waitGroup) add(n uint) {
	*wg += waitGroup(n)
}

func (wg *waitGroup) done() {
	if *wg == 0 {
		panic("wait group underflow")
	}
	*wg--
}

var wg waitGroup

func main() {
	ch := make(chan int)
	println("len, cap of channel:", len(ch), cap(ch), ch == nil)

	wg.add(1)
	go sender(ch)

	n, ok := <-ch
	println("recv from open channel:", n, ok)

	for n := range ch {
		println("received num:", n)
	}

	wg.wait()
	n, ok = <-ch
	println("recv from closed channel:", n, ok)

	// Test bigger values
	ch2 := make(chan complex128)
	wg.add(1)
	go sendComplex(ch2)
	println("complex128:", <-ch2)
	wg.wait()

	// Test multi-sender.
	ch = make(chan int)
	wg.add(3)
	go fastsender(ch, 10)
	go fastsender(ch, 23)
	go fastsender(ch, 40)
	slowreceiver(ch)
	wg.wait()

	// Test multi-receiver.
	ch = make(chan int)
	wg.add(3)
	go fastreceiver(ch)
	go fastreceiver(ch)
	go fastreceiver(ch)
	slowsender(ch)
	wg.wait()

	// Test iterator style channel.
	ch = make(chan int)
	wg.add(1)
	go iterator(ch, 100)
	sum := 0
	for i := range ch {
		sum += i
	}
	wg.wait()
	println("sum(100):", sum)

	// Test simple selects.
	go selectDeadlock() // cannot use waitGroup here - never terminates
	wg.add(1)
	go selectNoOp()
	wg.wait()

	// Test select with a single send operation (transformed into chan send).
	ch = make(chan int)
	wg.add(1)
	go fastreceiver(ch)
	select {
	case ch <- 5:
	}
	close(ch)
	wg.wait()
	println("did send one")

	// Test select with a single recv operation (transformed into chan recv).
	select {
	case n := <-ch:
		println("select one n:", n)
	}

	// Test select recv with channel that has one entry.
	ch = make(chan int)
	wg.add(1)
	go func(ch chan int) {
		runtime.Gosched()
		ch <- 55
		wg.done()
	}(ch)
	select {
	case make(chan int) <- 3:
		println("unreachable")
	case n := <-ch:
		println("select n from chan:", n)
	case n := <-make(chan int):
		println("unreachable:", n)
	}
	wg.wait()

	// Test select recv with closed channel.
	close(ch)
	select {
	case make(chan int) <- 3:
		println("unreachable")
	case n := <-ch:
		println("select n from closed chan:", n)
	case n := <-make(chan int):
		println("unreachable:", n)
	}

	// Test select send.
	ch = make(chan int)
	wg.add(1)
	go fastreceiver(ch)
	select {
	case ch <- 235:
		println("select send")
	case n := <-make(chan int):
		println("unreachable:", n)
	}
	close(ch)
	wg.wait()

	// test non-concurrent buffered channels
	ch = make(chan int, 2)
	ch <- 1
	ch <- 2
	println("non-concurrent channel recieve:", <-ch)
	println("non-concurrent channel recieve:", <-ch)

	// test closing channels with buffered data
	ch <- 3
	ch <- 4
	close(ch)
	println("closed buffered channel recieve:", <-ch)
	println("closed buffered channel recieve:", <-ch)
	println("closed buffered channel recieve:", <-ch)

	// test using buffered channels as regular channels with special properties
	wg.add(6)
	ch = make(chan int, 2)
	go send(ch)
	go send(ch)
	go send(ch)
	go send(ch)
	go receive(ch)
	go receive(ch)
	wg.wait()
	close(ch)
	var count int
	for range ch {
		count++
	}
	println("hybrid buffered channel recieve:", count)

	// test blocking selects
	ch = make(chan int)
	sch1 := make(chan int)
	sch2 := make(chan int)
	sch3 := make(chan int)
	wg.add(3)
	go func() {
		defer wg.done()
		time.Sleep(time.Millisecond)
		sch1 <- 1
	}()
	go func() {
		defer wg.done()
		time.Sleep(time.Millisecond)
		sch2 <- 2
	}()
	go func() {
		defer wg.done()
		// merge sch2 and sch3 into ch
		for i := 0; i < 2; i++ {
			var v int
			select {
			case v = <-sch1:
			case v = <-sch2:
			}
			select {
			case sch3 <- v:
				panic("sent to unused channel")
			case ch <- v:
			}
		}
	}()
	sum = 0
	for i := 0; i < 2; i++ {
		select {
		case sch3 <- sum:
			panic("sent to unused channel")
		case v := <-ch:
			sum += v
		}
	}
	wg.wait()
	println("blocking select sum:", sum)
}

func send(ch chan<- int) {
	ch <- 1
	wg.done()
}

func receive(ch <-chan int) {
	<-ch
	wg.done()
}

func sender(ch chan int) {
	for i := 1; i <= 8; i++ {
		if i == 4 {
			time.Sleep(time.Microsecond)
			println("slept")
		}
		ch <- i
	}
	close(ch)
	wg.done()
}

func sendComplex(ch chan complex128) {
	ch <- 7 + 10.5i
	wg.done()
}

func fastsender(ch chan int, n int) {
	ch <- n
	ch <- n + 1
	wg.done()
}

func slowreceiver(ch chan int) {
	sum := 0
	for i := 0; i < 6; i++ {
		sum += <-ch
		time.Sleep(time.Microsecond)
	}
	println("sum of n:", sum)
}

func slowsender(ch chan int) {
	for n := 0; n < 6; n++ {
		time.Sleep(time.Microsecond)
		ch <- 12 + n
	}
}

func fastreceiver(ch chan int) {
	sum := 0
	for i := 0; i < 2; i++ {
		n := <-ch
		sum += n
	}
	println("sum:", sum)
	wg.done()
}

func iterator(ch chan int, top int) {
	for i := 0; i < top; i++ {
		ch <- i
	}
	close(ch)
	wg.done()
}

func selectDeadlock() {
	println("deadlocking")
	select {}
	println("unreachable")
}

func selectNoOp() {
	println("select no-op")
	select {
	default:
	}
	println("after no-op")
	wg.done()
}

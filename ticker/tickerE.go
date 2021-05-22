package ticker

import (
	"fmt"
	"time"
)

type tickerE struct {
	errInterval time.Duration
	errTimer    *time.Timer
	errorMsg    string
	okInterval  time.Duration
	okTimer     *time.Timer

	endChan chan string

	nextRun chan string
}

func NewTickerE(errInterval, okInterval time.Duration) *tickerE {
	ticker := &tickerE{
		errInterval: errInterval,
		errTimer:    time.NewTimer(errInterval),
		okInterval:  okInterval,
		okTimer:     time.NewTimer(okInterval),
		nextRun:     make(chan string),
		endChan:     make(chan string),
	}

	ticker.errTimer.Stop()
	go func() {
		for {
			select {
			case <-ticker.okTimer.C:
				ticker.nextRun <- fmt.Sprintf("update config succeed. next tick will in %v", okInterval)
				ticker.okTimer.Reset(okInterval)
			case <-ticker.errTimer.C:
				ticker.errTimer.Stop()
				ticker.okTimer.Reset(okInterval)
				ticker.nextRun <- fmt.Sprintf("update config go error : %v, will retry in %v", ticker.errorMsg, errInterval)
			case <-ticker.endChan:
				close(ticker.nextRun)
			}
		}
	}()
	return ticker
}

func (t *tickerE) runnerError(err error) {
	t.errorMsg = err.Error()
	t.errTimer.Reset(t.errInterval)
}

func (t *tickerE) Stop(msg string) {
	t.endChan <- msg
}

func (t *tickerE) Run(fn func(msg string) error) (ret chan int) {
	if err := fn(""); err != nil {
		t.runnerError(err)
	}
	ret = make(chan int)
	go func() {
		for msg := range t.nextRun {
			if err := fn(msg); err != nil {
				t.runnerError(err)
			}
		}
		ret <- 1
	}()
	return ret
}

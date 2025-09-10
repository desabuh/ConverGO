package cvrdt

const MaxValue = 1<<31 - 1

type LamportClock struct {
	counter int
}

func (l *LamportClock) Increment() int {
	if l.counter < MaxValue {
		l.counter += 1
	}
	return l.counter
}

func (l *LamportClock) Value() int {
	return l.counter
}

func (l *LamportClock) Update(received int) {
	if l.counter < MaxValue {
		l.counter = max(received, l.counter) + 1
		if l.counter > MaxValue {
			panic("clock value is out of bound")
		}
	}
}

package processors

func InterruptAll(processors []*Processor) {
	for _, processor := range processors {
		if processor != nil {
			processor.Interrupt()
		}
	}
}

func WaitAll(processors []*Processor) {
	for _, processor := range processors {
		if processor != nil {
			processor.Wait()
		}
	}
}

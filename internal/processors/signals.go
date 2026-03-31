package processors

func (p *Processor) Interrupt() {
	p.logger.Infof("Interrupted")

	p.cancel()
}

func (p *Processor) Done() <-chan struct{} {
	return p.done
}

func (p *Processor) Wait() {
	<-p.done
}

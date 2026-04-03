package processors

func (p *Processor) Interrupt() {
	p.logger.Infof("Interrupt requested")

	p.cancel()
}

func (p *Processor) Done() <-chan struct{} {
	return p.done
}

func (p *Processor) Wait() {
	<-p.done
}

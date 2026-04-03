package receiver

func (r *Receiver) Interrupt() {
	r.logger.Infof("Interrupt requested")
	r.cancel()
}

func (r *Receiver) Done() <-chan struct{} {
	return r.done
}

func (r *Receiver) Wait() {
	<-r.done
}

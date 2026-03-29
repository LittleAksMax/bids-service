package receiver

func (r *Receiver) Interrupt() {
	r.logger.Infof("Interrupted")
	if r.adsClient != nil {
		r.adsClient.CloseIdleConnections()
	}
	r.cancel()
}

func (r *Receiver) Done() <-chan struct{} {
	return r.done
}

func (r *Receiver) Wait() {
	<-r.done
}

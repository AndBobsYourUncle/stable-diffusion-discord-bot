package imagine_queue

type Queue interface {
	AddImagine(item *QueueItem) (int, error)
	StartPolling()
}

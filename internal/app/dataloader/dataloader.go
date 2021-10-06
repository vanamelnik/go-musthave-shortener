package dataloader

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"
)

// deletechanSize - размер канала с очередью на удаление
const deleteChanSize = 100

type (
	DataLoader struct {
		ctx context.Context
		// ticker тикает раз в определенный интервал времени, напоминая агрегатору, что нужно слить данные на удаление.
		ticker *time.Ticker
		// deleteFunc - функция BatchDelete из интерфейса storage.Storage.
		// Вызывается агрегатором для слива данных на удаление в базу.
		deleteFunc BatchDeleteFunc

		// deletechan - канал, по которому агрегатору отправляются задания на удаление.
		deleteCh chan taskDel
		// stopCh - канал для закрытия сервиса.
		stopCh chan struct{}

		// tasks - хранилище заданий на удаление по каждому пользователю.
		tasks map[uuid.UUID][]string
	}

	// taskDel - задание на удаление записей с ключами из массива keys, вызванное пользователем id.
	taskDel struct {
		id   uuid.UUID
		keys []string
	}

	// BatchDeleteFunc - функция интерфейса storage, вызываемая агрегатором для удаления
	// идентификаторов из хранилища.
	BatchDeleteFunc func(ctx context.Context, id uuid.UUID, keys []string) error
)

// DataLoader накапливает данные для пакетного удаления. Для отправки данных в очередь вызывается функция
// BatchDelete, которая по каналу отправляет данные агрегатору, накапливающему данные по разным пользователям.
// По истечении интервала времени <interval> агрегатором для каждого пользователя вызывается функция batchDeleteFunc,
// переданная конструктору.
//
// Вопрос: хороша ли такая реализация с точки зрения структуры, или лучше было бы встроить это всё прямо в storage?
func NewDataLoader(ctx context.Context, deleteFunc BatchDeleteFunc, interval time.Duration) DataLoader {
	dl := DataLoader{
		ctx:        ctx,
		ticker:     time.NewTicker(interval),
		deleteFunc: deleteFunc,
		deleteCh:   make(chan taskDel, deleteChanSize),
		stopCh:     make(chan struct{}),
		tasks:      make(map[uuid.UUID][]string),
	}
	go dl.aggregator()
	log.Println("DataLoader started")

	return dl
}

// BatchDelete отправляет по каналу данные агрегатору, накапливающему записи на удаление и сливающему их в базу по истечении заданного интервала.
func (dl DataLoader) BatchDelete(ctx context.Context, id uuid.UUID, keys []string) error {
	dl.deleteCh <- taskDel{
		id:   id,
		keys: keys,
	}
	return nil // error добавлен для совместимости со storage.BatchDelete
}

// Close закрывает сервис DataLoader
func (dl DataLoader) Close() {
	dl.ticker.Stop()
	if dl.stopCh != nil {
		close(dl.stopCh)
	}
	dl.stopCh = nil
	dl.flush()
	log.Println("DataLoader closed")
}

// aggregator накапливает данные на удаление по каждому пользователю и сливает их функции deleteFunc по истечении заданного интервала.
func (dl DataLoader) aggregator() {
	for {
		select {
		case task := <-dl.deleteCh: // пришли данные, надо их засунуть в накопитель
			dl.tasks[task.id] = append(dl.tasks[task.id], task.keys...)
			log.Printf("dataloader: got %d keys to delete from id %s", len(task.keys), task.id)
		case <-dl.ticker.C: // время удалять записи!
			dl.flush()
		case <-dl.stopCh: // пора и честь знать...
			log.Println("dataloader: aggregator stopped")
			return
		}
	}
}

// flush отправляет накопленные данные по всем пользователям на удаление
func (dl DataLoader) flush() {
	if len(dl.tasks) == 0 {
		return
	}

	log.Printf("dataloader: flush: we have %d tasks to delete", len(dl.tasks))
	for id, keys := range dl.tasks {
		log.Printf("dataloader: flush: deleting %d keys for id=%s", len(keys), id)
		if err := dl.deleteFunc(dl.ctx, id, keys); err != nil {
			log.Printf("dataloader: %v", err)
		}
		delete(dl.tasks, id)
	}
}

package inmem

import (
	"encoding/gob"
	"log"
	"os"
	"time"
)

// gobber - сервис, сохраняющий данные in-memory хранилища в файл в формате gob с заданной периодичностью.
// Сервис работает в своей горутине и завершается по сигналу из канала gobberStop.
func (db *DB) gobber() {
	log.Println("[INF] gobber started!")
	ticker := time.NewTicker(db.flushInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := db.flush(); err != nil {
				log.Printf("gobber: %v", err)
			}
		case <-db.gobberStop:
			log.Println("[INF] gobber stopped")

			return
		}
	}
}

// flush проверяет флаг isChanged и при необходимости сохраняет данные хранилища в файл.
func (db *DB) flush() error {
	db.Lock()
	defer db.Unlock()
	if !db.isChanged {

		return nil
	}
	file, err := os.OpenFile(db.fileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0777) // переписываем весь файл
	if err != nil {

		return err
	}
	defer file.Close()
	enc := gob.NewEncoder(file)
	if err = enc.Encode(&db.repo); err != nil {

		return err
	}
	db.isChanged = false
	log.Printf("[INF] gobber: saved changes to the file %s", db.fileName)

	return nil
}

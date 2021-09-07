package inmem

import (
	"encoding/gob"
	"log"
	"os"
	"time"
)

// readRepo считывает и декодирует данные хранилища из файла в формате gob.
func readRepo(fileName string) (map[string]string, error) {
	file, err := os.OpenFile(fileName, os.O_RDONLY|os.O_CREATE, 0777)
	if err != nil {

		return nil, err
	}
	dec := gob.NewDecoder(file)
	repo := make(map[string]string)
	if err = dec.Decode(&repo); err != nil {

		return nil, err
	}
	log.Printf("[INF] readRepo: successfully read repo from file %s", fileName)

	return repo, nil
}

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
			if err := db.flush(); err != nil {
				log.Printf("gobber: %v", err)
			}
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
		log.Println("[INF] gobber: no changes - no flush")

		return nil
	}
	file, err := os.OpenFile(db.fileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0777)
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

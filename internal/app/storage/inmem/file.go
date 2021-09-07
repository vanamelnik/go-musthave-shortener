package inmem

import (
	"encoding/gob"
	"fmt"
	"log"
	"os"
)

// initRepo считывает и декодирует данные хранилища из файла в формате gob.
// Если файл не найден - он создается функцией createRepoFile
func initRepo(fileName string) (map[string]string, error) {
	if _, err := os.Stat(fileName); err != nil {
		if os.IsNotExist(err) {

			return createRepoFile(fileName)
		}

		return nil, fmt.Errorf("initRepo: %v", err)
	}
	file, err := os.OpenFile(fileName, os.O_RDONLY, 0777)
	if err != nil {

		return nil, err
	}
	dec := gob.NewDecoder(file)
	repo := make(map[string]string)
	if err = dec.Decode(&repo); err != nil {

		return nil, fmt.Errorf("initRepo: %v", err)
	}
	log.Printf("[INF] readRepo: successfully read repo from file %s", fileName)

	return repo, nil
}

// createRepoFile создает файл и записывает в него сериализованную пустую map

func createRepoFile(fileName string) (map[string]string, error) {
	file, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0777)
	if err != nil {

		return nil, fmt.Errorf("createRepoFile: %v", err)
	}
	enc := gob.NewEncoder(file)
	repo := make(map[string]string)
	if err = enc.Encode(&repo); err != nil {

		return nil, fmt.Errorf("createRepoFile: %v", err)
	}
	log.Printf("[INF] createRepoFile: successfully created repo file %s", fileName)
	return repo, nil
}

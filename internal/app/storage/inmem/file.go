package inmem

import (
	"encoding/gob"
	"fmt"
	"log"
	"os"
)

// initRepo считывает и декодирует данные хранилища из файла в формате gob.
// Если файл не найден - он создается функцией createRepoFile.
func initRepo(fileName string) ([]row, error) {
	if _, err := os.Stat(fileName); err != nil {
		if os.IsNotExist(err) {
			return createRepoFile(fileName)
		}

		return nil, fmt.Errorf("initRepo: %v", err)
	}
	file, err := os.OpenFile(fileName, os.O_RDONLY, 0777)
	if err != nil {
		return nil, fmt.Errorf("initRepo: %v", err)
	}
	defer file.Close()

	dec := gob.NewDecoder(file)
	repo := make([]row, 0)
	if err = dec.Decode(&repo); err != nil {
		return nil, fmt.Errorf("initRepo: %v", err)
	}
	log.Printf("[INF] readRepo: successfully read repo from file %s", fileName)

	return repo, nil
}

// createRepoFile создает файл и записывает в него сериализованную пустую map (иначе автотест
// ругается на пустой файл).
func createRepoFile(fileName string) ([]row, error) {
	file, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0777)
	if err != nil {

		return nil, fmt.Errorf("createRepoFile: %v", err)
	}
	defer file.Close()

	enc := gob.NewEncoder(file)
	repo := make([]row, 0)
	if err = enc.Encode(&repo); err != nil {
		return nil, fmt.Errorf("createRepoFile: %v", err)
	}
	log.Printf("[INF] createRepoFile: successfully created repo file %s", fileName)

	return repo, nil
}

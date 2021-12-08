package dataloader_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vanamelnik/go-musthave-shortener/internal/app/dataloader"
	"github.com/vanamelnik/go-musthave-shortener/internal/app/storage/inmem"
)

func TestDataLoader(t *testing.T) {
	id1, id2, id3 := uuid.New(), uuid.New(), uuid.New()
	ctx := context.Background()
	// данные для сохранения
	toStore := []struct {
		id      uuid.UUID
		records map[string]string //[key]:url
	}{
		{
			id: id1,
			records: map[string]string{
				"id1_key1":  "id1_url1",
				"id1_key2":  "id1_url2",
				"id1_key3":  "id1_url3",
				"id1_key4":  "id1_url4",
				"id1_key5":  "id1_url5",
				"id1_key6":  "id1_url6",
				"id1_key7":  "id1_url7",
				"id1_key8":  "id1_url8",
				"id1_key9":  "id1_url9",
				"id1_key10": "id1_url10",
				"id1_key11": "id1_url11",
				"id1_key12": "id1_url12",
				"id1_key13": "id1_url13",
			},
		},
		{
			id: id2,
			records: map[string]string{
				"id2_key1":  "id2_url1",
				"id2_key2":  "id2_url2",
				"id2_key3":  "id2_url3",
				"id2_key4":  "id2_url4",
				"id2_key5":  "id2_url5",
				"id2_key6":  "id2_url6",
				"id2_key7":  "id2_url7",
				"id2_key8":  "id2_url8",
				"id2_key9":  "id2_url9",
				"id2_key10": "id2_url10",
			},
		},
	}

	toDelete := []struct {
		name string
		id   uuid.UUID
		keys []string
	}{
		{
			name: "Delete #1 - id1 is deleting his keys 1-3",
			id:   id1,
			keys: []string{
				"id1_key1",
				"id1_key2",
				"id1_key3",
			},
		},
		{
			name: "Delete #2 - id2 is deleting his keys 1-5",
			id:   id2,
			keys: []string{
				"id2_key1",
				"id2_key2",
				"id2_key3",
				"id2_key4",
				"id2_key5",
			},
		},
		{
			name: "Delete #3 - id1 is deleting his keys 10-13",
			id:   id1,
			keys: []string{
				"id1_key10",
				"id1_key11",
				"id1_key12",
				"id1_key13",
			},
		},
		{
			name: "Delete #4 - id2 is deleting his keys 9-13 (should log error wrong key for keys 11-13)",
			id:   id2,
			keys: []string{
				"id2_key9",
				"id2_key10",
				"id2_key11",
				"id2_key12",
				"id2_key13",
			},
		},
		{
			name: "Delete #5 - id1 is deleting his key 1 that already in the queue",
			id:   id1,
			keys: []string{"id1_key1"},
		},
		{
			name: "Delete #6 - id3 is trying to delete other user's data (id1 keys 4-5)",
			id:   id3,
			keys: []string{
				"id1_key4",
				"id1_key5",
			},
		},
	}

	tt := []struct {
		name         string
		id           uuid.UUID
		wantUserUrls map[string]string
	}{
		{
			name: "Test #1 - records of id1 which have not been deleted",
			id:   id1,
			wantUserUrls: map[string]string{
				"id1_key4": "id1_url4",
				"id1_key5": "id1_url5",
				"id1_key6": "id1_url6",
				"id1_key7": "id1_url7",
				"id1_key8": "id1_url8",
				"id1_key9": "id1_url9",
			},
		},
		{
			name: "Test #2 - records of id2 which have not been deleted",
			id:   id2,
			wantUserUrls: map[string]string{
				"id2_key6": "id2_url6",
				"id2_key7": "id2_url7",
				"id2_key8": "id2_url8",
			},
		},
	}

	const filename = "tmp.db"
	db, err := inmem.NewDB(filename, time.Hour)
	require.NoError(t, err)
	defer func() {
		db.Close()
		require.NoError(t, os.Remove(filename))
	}()

	t.Log("Storing data...")
	for _, taskStore := range toStore {
		for key, url := range taskStore.records {
			assert.NoError(t, db.Store(ctx, taskStore.id, key, url))
		}
	}

	dl := dataloader.NewDataLoader(ctx, db.BatchDelete, time.Millisecond)
	defer dl.Close()

	t.Log("Running delete tasks...")
	for _, delTask := range toDelete {
		delTask := delTask
		go func() {
			t.Log(delTask.name)
			//nolint:errcheck
			dl.BatchDelete(ctx, delTask.id, delTask.keys)
		}()
	}

	t.Log("Waiting...")
	time.Sleep(5 * time.Millisecond)

	t.Log("Checking keys which have not been deleted...")
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.wantUserUrls, db.GetAll(ctx, tc.id))
		})
	}
}

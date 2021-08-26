package inmem

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestGet тестирует функцию Get с использованием фейкового хранилища
func TestGet(t *testing.T) {
	db := DB{
		repo: map[string]string{
			"key1": "url1",
			"key2": "url2",
			"":     "url",
			"key":  "",
		},
	}
	tt := []struct {
		name    string
		key     string
		wantURL string
		wantErr bool
	}{
		{
			name:    "Normal case #1",
			key:     "key1",
			wantURL: "url1",
			wantErr: false,
		},
		{
			name:    "Normal case #2",
			key:     "key2",
			wantURL: "url2",
			wantErr: false,
		},
		{
			name:    "Empty key",
			key:     "",
			wantURL: "url",
			wantErr: false,
		},
		{
			name:    "Empty Url",
			key:     "key",
			wantURL: "",
			wantErr: false,
		},
		{
			name:    "Key not found",
			key:     "key999",
			wantURL: "",
			wantErr: true,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			url, err := db.Get(tc.key)
			if !tc.wantErr {
				assert.NoError(t, err)
			}
			assert.Equal(t, tc.wantURL, url)
		})
	}
}

// TestInmem тестирует связку Store - Get. В зависимости от поля "action" ("store",
// "get" и "both") выполняются тесты обоих методов.
func TestInmem(t *testing.T) {
	type args struct {
		key string
		url string
	}
	tt := []struct {
		name         string
		action       string
		args         args
		wantErrStore bool
		wantErrGet   bool
	}{
		{
			name:         "Normal case Store - Get",
			action:       "both",
			args:         args{key: "key1", url: "url1"},
			wantErrStore: false,
			wantErrGet:   false,
		},
		{
			name:         "Normal case Get",
			action:       "get",
			args:         args{key: "key1", url: "url1"},
			wantErrStore: false,
			wantErrGet:   false,
		},
		{
			name:         "Store non-unique key",
			action:       "store",
			args:         args{key: "key1", url: "url9999"},
			wantErrStore: true,
			wantErrGet:   false,
		},
		{
			name:         "Get wrong key",
			action:       "get",
			args:         args{key: "key2", url: "url11111"},
			wantErrStore: false,
			wantErrGet:   true,
		},
	}
	d := NewDB()
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			if tc.action == "store" || tc.action == "both" {
				if err := d.Store(tc.args.key, tc.args.url); (err != nil) != tc.wantErrStore {
					t.Errorf("DB.Store() error = %v, wantErr %v", err, tc.wantErrStore)
				}
			}
			if tc.action == "get" || tc.action == "both" {
				url, err := d.Get(tc.args.key)
				if (err != nil) != tc.wantErrGet {
					t.Fatalf("DB.Get() error = %v, wantErr %v", err, tc.wantErrGet)
				}
				if url != tc.args.url && err == nil {
					t.Errorf("DB.Get() Expected url = %v, got %v", tc.args.url, url)
				}
			}
		})
	}
}

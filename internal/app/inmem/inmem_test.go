package inmem

import (
	"testing"
)

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
		wantUrl string
		wantErr bool
	}{
		{
			name:    "Normal case #1",
			key:     "key1",
			wantUrl: "url1",
			wantErr: false,
		},
		{
			name:    "Normal case #2",
			key:     "key2",
			wantUrl: "url2",
			wantErr: false,
		},
		{
			name:    "Empty key",
			key:     "",
			wantUrl: "url",
			wantErr: false,
		},
		{
			name:    "Empty Url",
			key:     "key",
			wantUrl: "",
			wantErr: false,
		},
		{
			name:    "Key not found",
			key:     "key999",
			wantUrl: "",
			wantErr: true,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			url, err := db.Get(tc.key)
			if (err != nil) != tc.wantErr {
				if tc.wantErr {
					t.Error("Expected err, got no error")
				} else {
					t.Errorf("Expected no err, got error: %v", err)
				}
			}
			if url != tc.wantUrl {
				t.Errorf("Expected url = %v, got %v", tc.wantUrl, url)
			}
		})
	}
}

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
			} else if tc.action == "get" || tc.action == "both" {
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

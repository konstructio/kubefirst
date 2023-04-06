/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package pkg

import (
	"fmt"
	"os"
	"reflect"
	"testing"
)

func TestRemoveSubDomain(t *testing.T) {

	type args struct {
		domain string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name:    "single domain",
			args:    args{"https://example.com"},
			want:    "example.com",
			wantErr: false,
		},
		{
			name:    "subdomain.domain",
			args:    args{"https://hub.example.com"},
			want:    "example.com",
			wantErr: false,
		},
		{
			name:    "sub.subdomain.domain",
			args:    args{"https://hub.hub.example.com"},
			want:    "example.com",
			wantErr: false,
		},
		{
			name:    "another domain extension",
			args:    args{"https://x.xyz"},
			want:    "x.xyz",
			wantErr: false,
		},
		{
			name:    "invalid domain",
			args:    args{"https://xyz"},
			want:    "",
			wantErr: true,
		},
		{
			name:    "invalid domain",
			args:    args{"invalid-examplecom"},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RemoveSubDomain(tt.args.domain)
			if (err != nil) != tt.wantErr {
				t.Errorf("RemoveSubDomain() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RemoveSubDomain() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_isValidURL(t *testing.T) {
	type args struct {
		rawURL string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "valid url sample 1",
			args:    args{rawURL: "https://example.com"},
			wantErr: false,
		},
		{
			name:    "valid url sample 2",
			args:    args{rawURL: "https://hub.example.com"},
			wantErr: false,
		},
		{
			name:    "empty string",
			args:    args{rawURL: ""},
			wantErr: true,
		},
		{
			name:    "invalid url sample 1",
			args:    args{rawURL: "http//example.com"},
			wantErr: true,
		},
		{
			name:    "invalid url sample 2",
			args:    args{rawURL: "example.com"},
			wantErr: true,
		},
		{
			name:    "invalid url sample 3",
			args:    args{rawURL: "examplecom"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := IsValidURL(tt.args.rawURL); (err != nil) != tt.wantErr {
				t.Errorf("IsValidURL() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateK1Folder(t *testing.T) {
	emptyTempFolder, err := os.MkdirTemp("", "unit-test")
	if err != nil {
		t.Error(err)
	}

	populatedTempFolder, err := os.MkdirTemp("", "populated-unit-test")
	if err != nil {
		t.Error(err)
	}
	_, err = os.Create(fmt.Sprintf("%s/%s", populatedTempFolder, "argocd-init-values.yaml"))
	if err != nil {
		t.Error(err)
	}

	type args struct {
		folderPath string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "it has a folder, and folder is empty",
			args:    args{folderPath: emptyTempFolder},
			wantErr: false,
		},
		{
			name:    "it has a folder, and folder is not empty",
			args:    args{folderPath: populatedTempFolder},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateK1Folder(tt.args.folderPath); (err != nil) != tt.wantErr {
				t.Errorf("ValidateK1Folder() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetFileContent(t *testing.T) {

	file, err := os.CreateTemp("", "testing.txt")
	if err != nil {
		t.Error(err)
	}
	defer func(name string) {
		err := os.Remove(name)
		if err != nil {
			t.Error(err)
		}
	}(file.Name())

	fileWithContent, err := os.CreateTemp("", "testing-with-content")
	if err != nil {
		t.Error(err)
	}
	_, err = fileWithContent.Write([]byte("some-content"))
	if err != nil {
		t.Error(err)
	}
	err = fileWithContent.Close()
	if err != nil {
		t.Error(err)
	}

	defer func(name string) {
		err := os.Remove(name)
		if err != nil {
			t.Error(err)
		}
	}(fileWithContent.Name())

	tests := []struct {
		name     string
		filePath string
		want     []byte
		wantErr  bool
	}{
		{
			name:     "file doesn't exist",
			filePath: "non-existent-file.ext",
			want:     nil,
			wantErr:  true,
		},
		{
			name:     "file with no content, returns no content",
			filePath: file.Name(),
			want:     []byte(""),
			wantErr:  false,
		},
		{
			name:     "file with content, returns its content",
			filePath: fileWithContent.Name(),
			want:     []byte("some-content"),
			wantErr:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetFileContent(tt.filePath)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetFileContent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetFileContent() got = %v, want %v", got, tt.want)
			}
		})
	}
}

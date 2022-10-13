package pkg

import (
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
			want:    "https://example.com",
			wantErr: false,
		},
		{
			name:    "subdomain.domain",
			args:    args{"https://hub.example.com"},
			want:    "https://example.com",
			wantErr: false,
		},
		{
			name:    "sub.subdomain.domain",
			args:    args{"https://hub.hub.example.com"},
			want:    "https://example.com",
			wantErr: false,
		},
		{
			name:    "another domain extension",
			args:    args{"https://x.xyz"},
			want:    "https://x.xyz",
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

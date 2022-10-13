package pkg

import (
	"fmt"
	"log"
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
			args:    args{"example.com"},
			want:    "example.com",
			wantErr: false,
		},
		{
			name:    "subdomain.domain",
			args:    args{"hub.example.com"},
			want:    "example.com",
			wantErr: false,
		},
		{
			name:    "sub.subdomain.domain",
			args:    args{"hub.hub.example.com"},
			want:    "example.com",
			wantErr: false,
		},
		{
			name:    "another domain extension",
			args:    args{"x.xyz"},
			want:    "x.xyz",
			wantErr: false,
		},
		{
			name:    "invalid domain",
			args:    args{"xyz"},
			want:    "xyz",
			wantErr: false,
		},
		{
			name:    "invalid domain",
			args:    args{"invalid-examplecom"},
			want:    "example.com",
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
			log.Println("---debug---")
			fmt.Println(got)
			log.Println("---debug---")

		})
	}
}

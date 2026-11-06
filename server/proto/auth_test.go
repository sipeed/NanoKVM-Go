package proto

import "testing"

func TestValidateLoginReqShape(t *testing.T) {
	tests := []struct {
		name    string
		req     LoginReq
		wantErr bool
	}{
		{
			name: "password login",
			req:  LoginReq{Username: "admin", Password: "secret"},
		},
		{
			name: "challenge login",
			req:  LoginReq{Challenge: "opaque", Code: "123456"},
		},
		{
			name:    "empty",
			req:     LoginReq{},
			wantErr: true,
		},
		{
			name:    "missing password",
			req:     LoginReq{Username: "admin"},
			wantErr: true,
		},
		{
			name: "missing code still uses challenge flow",
			req:  LoginReq{Challenge: "opaque"},
		},
		{
			name: "missing challenge still uses challenge flow",
			req:  LoginReq{Code: "123456"},
		},
		{
			name:    "mixed shapes",
			req:     LoginReq{Username: "admin", Password: "secret", Challenge: "opaque", Code: "123456"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateLoginReqShape(tt.req)
			if tt.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}

package transport

import "testing"

func TestResolveAddress(t *testing.T) {
	tests := []struct {
		name string
		flag string
		port string
		want string
	}{
		{name: "default", want: "127.0.0.1:19081"},
		{name: "port environment", port: "19876", want: "127.0.0.1:19876"},
		{name: "flag wins", flag: "127.0.0.1:19991", port: "19876", want: "127.0.0.1:19991"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := ResolveAddress(test.flag, func(key string) string {
				if key == "PORT" {
					return test.port
				}
				return ""
			})
			if err != nil || got != test.want {
				t.Fatalf("ResolveAddress() = %q, %v; want %q", got, err, test.want)
			}
		})
	}
	if _, err := ResolveAddress("", func(string) string { return "not-a-port" }); err == nil {
		t.Fatal("非法 PORT 应返回错误")
	}
}

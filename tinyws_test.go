package tinyws

// 这个文件没有版权约束
import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var (
	globalTestData = []byte("hello world")
)

func testTcpSocket(quit chan bool, t *testing.T) (addr string) {
	addr = GetNoPortExists()

	addr = ":" + addr
	go func() {
		listener, err := net.Listen("tcp", addr)
		if err != nil {
			t.Errorf("%v\n", err)
			return
		}
		defer listener.Close()

		conn, err := listener.Accept()
		if err != nil {
			t.Errorf("%v\n", err)
			return
		}

		<-quit
		defer conn.Close()
	}()

	return addr

}

// 一个echo 测试服务
func newEchoWebSocketServer(t *testing.T, testData []byte) *httptest.Server {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := Upgrade(w, r)
		assert.NoError(t, err)
		if err != nil {
			return
		}
		// 读数据
		all, op, err := conn.ReadMessage()
		assert.NoError(t, err)
		// 和客户端发的比较一下, 是否相等
		assert.Equal(t, string(all), string(testData))
		// 再回写给客户端
		err = conn.WriteMessage(op, all)
		assert.NoError(t, err)
	}))
	s.URL = "ws" + strings.TrimPrefix(s.URL, "http")
	return s
}

// 一个echo tls测试服务
func newTLSEchoWebSocketServer(t *testing.T, testData []byte) *httptest.Server {
	s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := Upgrade(w, r)
		assert.NoError(t, err)
		if err != nil {
			return
		}
		// 读数据
		all, op, err := conn.ReadMessage()
		assert.NoError(t, err)
		// 和客户端发的比较一下, 是否相等
		assert.Equal(t, string(all), string(testData))
		// 再回写给客户端
		err = conn.WriteMessage(op, all)
		assert.NoError(t, err)
	}))
	s.URL = "wss" + strings.TrimPrefix(s.URL, "https")
	return s
	//return &tstServer{ser: s, URL: "wss" + strings.TrimPrefix(s.URL, "https")}
}

// 写和读
func sendAndRecv(c *Conn, t *testing.T, testData []byte) {
	// 写数据
	err := c.WriteMessage(Text, testData)
	assert.NoError(t, err)
	if err != nil {
		return
	}

	// 读数据
	all, op, err := c.ReadMessage()
	assert.NoError(t, err)
	if err != nil {
		return
	}
	assert.Equal(t, string(all), string(testData))
	assert.Equal(t, op, Text)

}

// 测试客户端连接
func TestDial(t *testing.T) {
	// 起个mock服务
	ts := newEchoWebSocketServer(t, globalTestData)
	// 发起连接
	c, err := Dial(ts.URL)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	defer c.Close()

	// 写和读
	sendAndRecv(c, t, globalTestData)
}

// 此测试代码来自gorilla/websocket
func rootCAs(t *testing.T, s *httptest.Server) *x509.CertPool {
	certs := x509.NewCertPool()
	for _, c := range s.TLS.Certificates {
		roots, err := x509.ParseCertificates(c.Certificate[len(c.Certificate)-1])
		if err != nil {
			t.Fatalf("error parsing server's root cert: %v", err)
		}
		for _, root := range roots {
			certs.AddCert(root)
		}
	}
	return certs
}

// 测试tls客户端连接
func TestDialTLS(t *testing.T) {
	// 起个mock服务
	ts := newTLSEchoWebSocketServer(t, globalTestData)
	// 发起连接
	c, err := Dial(ts.URL, WithTLSConfig(&tls.Config{RootCAs: rootCAs(t, ts)}))
	assert.NoError(t, err)
	if err != nil {
		return
	}
	defer c.Close()

	// 写和读
	sendAndRecv(c, t, globalTestData)
}

// 测试握手时的timeout
func TestDialTimeout(t *testing.T) {
	quit := make(chan bool)
	addr := testTcpSocket(quit, t)

	time.Sleep(time.Second / 100)
	_, err := Dial("ws://"+addr, WithDialTimeout(time.Second))
	fmt.Printf("%v\n", err)
	assert.Error(t, err)
	close(quit)
}

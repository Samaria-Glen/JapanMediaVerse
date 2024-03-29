package yquic

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yomorun/yomo/core/frame"
	"github.com/yomorun/yomo/pkg/frame-codec/y3codec"
	pkgtls "github.com/yomorun/yomo/pkg/tls"
)

const testHost = "localhost:9008"

const (
	handshakeName = "hello yomo"
	streamContent = "hello stream"
	CloseMessage  = "bye!"
)

func TestFrameConnection(t *testing.T) {
	go func() {
		if err := runListener(t); err != nil {
			panic(err)
		}
	}()

	fconn, err := DialAddr(context.TODO(), testHost,
		y3codec.Codec(), y3codec.PacketReadWriter(),
		pkgtls.MustCreateClientTLSConfig(), nil,
	)
	assert.NoError(t, err)

	err = fconn.WriteFrame(&frame.HandshakeAckFrame{})
	assert.NoError(t, err)

	for {
		f, err := fconn.ReadFrame()
		if err != nil {
			se := new(frame.ErrConnClosed)
			assert.True(t, errors.As(err, &se))
			assert.Equal(t, frame.NewErrConnClosed(true, CloseMessage), err)
			return
		}
		hf := f.(*frame.HandshakeFrame)
		assert.Equal(t, handshakeName, hf.Name)
	}
}

func runListener(t *testing.T) error {
	listener, err := ListenAddr(testHost, y3codec.Codec(), y3codec.PacketReadWriter(), pkgtls.MustCreateServerTLSConfig(testHost), nil)
	if err != nil {
		return err
	}

	fconn, err := listener.Accept(context.TODO())
	if err != nil {
		return err
	}

	f, err := fconn.ReadFrame()
	assert.NoError(t, err)
	assert.Equal(t, f.Type(), frame.TypeHandshakeAckFrame)

	if err := fconn.WriteFrame(&frame.HandshakeFrame{Name: handshakeName}); err != nil {
		return err
	}

	time.AfterFunc(time.Second, func() {
		err := fconn.CloseWithError(CloseMessage)
		assert.NoError(t, err)

		// close twice has no effect.
		err = fconn.CloseWithError(CloseMessage)
		assert.NoError(t, err)

		err = fconn.WriteFrame(&frame.DataFrame{Payload: []byte("aaaa")})
		assert.Equal(t, frame.NewErrConnClosed(false, CloseMessage), err)

		t.Log("close connection done")
	})

	return nil
}

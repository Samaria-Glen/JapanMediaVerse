package yomo

import (
	"context"
	"io"

	"github.com/yomorun/yomo/core"
	"github.com/yomorun/yomo/core/frame"
	"github.com/yomorun/yomo/core/metadata"
	"github.com/yomorun/yomo/pkg/id"
	"github.com/yomorun/yomo/pkg/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
	"golang.org/x/exp/slog"
)

// Source is responsible for sending data to yomo.
type Source interface {
	// Close will close the connection to YoMo-Zipper.
	Close() error
	// Connect to YoMo-Zipper.
	Connect() error
	// Write the data to directed downstream.
	Write(tag uint32, data []byte) error
	// SetErrorHandler set the error handler function when server error occurs
	SetErrorHandler(fn func(err error))
	// [Experimental] SetReceiveHandler set the observe handler function
	SetReceiveHandler(fn func(tag uint32, data []byte))
	// Pipe pipe the stream data to zipper.
	Pipe(tag uint32, reader io.Reader) error
}

// YoMo-Source
type yomoSource struct {
	name       string
	zipperAddr string
	client     *core.Client
	fn         func(uint32, []byte)
}

var _ Source = &yomoSource{}

// NewSource create a yomo-source
func NewSource(name, zipperAddr string, opts ...SourceOption) Source {
	clientOpts := make([]core.ClientOption, len(opts))
	for k, v := range opts {
		clientOpts[k] = core.ClientOption(v)
	}

	client := core.NewClient(name, core.ClientTypeSource, clientOpts...)

	return &yomoSource{
		name:       name,
		zipperAddr: zipperAddr,
		client:     client,
	}
}

// Close will close the connection to YoMo-Zipper.
func (s *yomoSource) Close() error {
	if err := s.client.Close(); err != nil {
		s.client.Logger().Error("failed to close the source", "err", err)
		return err
	}
	s.client.Logger().Debug("the source is closed")
	return nil
}

// Connect to YoMo-Zipper.
func (s *yomoSource) Connect() error {
	// set backflowframe handler
	s.client.SetBackflowFrameObserver(func(frm *frame.BackflowFrame) {
		if s.fn != nil {
			s.fn(frm.Tag, frm.Carriage)
		}
	})

	err := s.client.Connect(context.Background(), s.zipperAddr)
	return err
}

// Write writes data with specified tag.
func (s *yomoSource) Write(tag uint32, data []byte) error {
	md, deferFunc := TraceMetadata(s.client.ClientID(), s.name, false, s.client.TracerProvider(), s.client.Logger())
	defer deferFunc()

	mdBytes, err := md.Encode()
	// metadata
	if err != nil {
		return err
	}
	f := &frame.DataFrame{
		Tag:      tag,
		Metadata: mdBytes,
		Payload:  data,
	}
	s.client.Logger().Debug("source write", "tag", tag, "data", data)
	return s.client.WriteFrame(f)
}

// SetErrorHandler set the error handler function when server error occurs
func (s *yomoSource) SetErrorHandler(fn func(err error)) {
	s.client.SetErrorHandler(fn)
}

// [Experimental] SetReceiveHandler set the observe handler function
func (s *yomoSource) SetReceiveHandler(fn func(uint32, []byte)) {
	s.fn = fn
	s.client.Logger().Info("receive hander set for the source")
}

// Pipe pipe the stream data to zipper.
func (s *yomoSource) Pipe(tag uint32, reader io.Reader) error {
	md, deferFunc := TraceMetadata(s.client.ClientID(), s.name, true, s.client.TracerProvider(), s.client.Logger())
	defer deferFunc()
	// request server to create a stream
	dataStream, err := s.client.RequestStream()
	if err != nil {
		return err
	}
	// metadata
	mdBytes, err := md.Encode()
	// metadata
	if err != nil {
		return err
	}
	// stream frame encode
	streamFrame := &frame.StreamFrame{
		ClientID: s.client.ClientID(),
		StreamID: int64(dataStream.StreamID()),
		// TODO: make ChunkSize configurable or 0 for auto
		ChunkSize: 1024,
	}
	data, err := s.client.FrameStream().Codec().Encode(streamFrame)
	if err != nil {
		s.client.Logger().Error("client codec encode stream frame error", "err", err)
		return err
	}
	// write dataframe with streamID
	f := &frame.DataFrame{
		Tag:      tag,
		Metadata: mdBytes,
		Payload:  data,
	}
	err = s.client.WriteFrame(f)
	if err != nil {
		s.client.Logger().Error("source write frame error", "err", err)
		return err
	}
	s.client.Logger().Debug("source pipe stream", "tag", tag, "stream_id", dataStream.StreamID())
	// pipe stream
	buf := make([]byte, 1024)
	_, err = io.CopyBuffer(dataStream, reader, buf)
	if err != nil {
		if err == io.EOF {
			s.client.Logger().Info("source pipe stream done", "stream_id", dataStream.StreamID)
			return err
		}
		s.client.Logger().Error("source pipe stream error", "err", err)
		return err
	}
	return nil
}

// TraceMetadata generates source trace metadata.
func TraceMetadata(clientID, name string, streamed bool, tp oteltrace.TracerProvider, logger *slog.Logger) (metadata.M, func()) {
	deferFunc := func() {}
	var tid, sid string
	// trace
	traced := false
	if tp != nil {
		span, err := trace.NewSpan(tp, core.ClientTypeSource.String(), name, "", "")
		if err != nil {
			logger.Error("source trace error", "err", err)
		} else {
			deferFunc = func() { span.End() }
			tid = span.SpanContext().TraceID().String()
			sid = span.SpanContext().SpanID().String()
			traced = true
		}
	}
	if tid == "" {
		logger.Debug("source create new tid")
		tid = id.TID()
	}
	if sid == "" {
		logger.Debug("source create new sid")
		sid = id.SID()
	}
	logger.Debug("source metadata", "tid", tid, "sid", sid, "traced", traced, "streamed", streamed)

	md := core.NewDefaultMetadata(clientID, tid, sid, traced, streamed)

	return md, deferFunc
}

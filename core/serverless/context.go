// Package serverless provides the server serverless function context.
package serverless

import (
	"github.com/yomorun/yomo/core"
	"github.com/yomorun/yomo/core/frame"
	"github.com/yomorun/yomo/core/metadata"
)

// Context sfn handler context
type Context struct {
	writer frame.Writer
	tag    uint32
	md     metadata.M
	data   []byte
}

// NewContext creates a new serverless Context
func NewContext(writer frame.Writer, tag uint32, md metadata.M, data []byte) *Context {
	return &Context{
		writer: writer,
		tag:    tag,
		md:     md,
		data:   data,
	}
}

// Tag returns the tag of the data frame
func (c *Context) Tag() uint32 {
	return c.tag
}

// Data returns the data of the data frame
func (c *Context) Data() []byte {
	return c.data
}

func (c *Context) TID() string {
	tid, _ := c.md.Get(core.MetadataTIDKey)
	return tid
}

// Write writes the data
func (c *Context) Write(tag uint32, data []byte) error {
	if data == nil {
		return nil
	}

	mdBytes, err := c.md.Encode()
	if err != nil {
		return err
	}

	dataFrame := &frame.DataFrame{
		Tag:      tag,
		Metadata: mdBytes,
		Payload:  data,
	}

	return c.writer.WriteFrame(dataFrame)
}
